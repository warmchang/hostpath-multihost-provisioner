/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
    "context"
	"errors"
	"flag"
	"os"
	"path"
    "net"
    "net/http"
    "net/url"
    "fmt"

	"github.com/golang/glog"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/v6/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"syscall"
)

const (
	provisionerName           = "kubeboost.github.com/hostpath-multihost-provider"
    storageManagerServiceName = "hostpath-multihost-manager"
    storageManagerServicePort = "8080"
)

type hostPathProvisioner struct {
	// The directory to create PV-backing directories in
	pvDir string

	// Identity of this hostPathProvisioner, set to node's name. Used to identify
	// "this" provisioner's PVs.
	identity string

	// Override the default reclaim-policy of dynamicly provisioned volumes
	// (which is remove).
	reclaimPolicy string
}

// NewHostPathProvisioner creates a new hostpath provisioner
func NewHostPathProvisioner() controller.Provisioner {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		glog.Fatal("env variable NODE_NAME must be set so that this provisioner can identify itself")
	}

	pvDir := os.Getenv("PV_DIR")
	if pvDir == "" {
		glog.Fatal("env variable PV_DIR must be set so that this provisioner knows where to place its data")
	}

	reclaimPolicy := os.Getenv("PV_RECLAIM_POLICY")
	return &hostPathProvisioner{
		pvDir:         pvDir,
		identity:      nodeName,
		reclaimPolicy: reclaimPolicy,
	}
}

var _ controller.Provisioner = &hostPathProvisioner{}

// Provision creates a storage asset and returns a PV object representing it.
func (p *hostPathProvisioner) Provision(_ context.Context, options controller.ProvisionOptions) (*v1.PersistentVolume, controller.ProvisioningState, error) {
	path := path.Join(p.pvDir, options.PVC.Namespace+"-"+options.PVC.Name+"-"+options.PVName)
	glog.Infof("creating backing directory: %v", path)
    sendRequestToManager(path, createDir)

	reclaimPolicy := *options.StorageClass.ReclaimPolicy
	if p.reclaimPolicy != "" {
		reclaimPolicy = v1.PersistentVolumeReclaimPolicy(p.reclaimPolicy)
	}

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: options.PVName,
			Annotations: map[string]string{
				"hostPathMultiHostProvisionerIdentity": p.identity,
			},
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: reclaimPolicy,
			AccessModes:                   options.PVC.Spec.AccessModes,
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)],
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: path,
				},
			},
		},
	}

	return pv, controller.ProvisioningFinished, nil
}


type HttpStatusError struct {
    status int
}

func (e HttpStatusError) Error() string {
    return fmt.Sprintf("HTTP Status Error with status code: %v", e.status)
}

func sendRequestToManager(path string, requestFunc func(chan error, string, string)) error {
    glog.Infof("Looking for service %q.", storageManagerServiceName)
    ips, err := net.LookupHost(storageManagerServiceName)
    if err != nil {
        glog.Fatalf("Error looking for service: %q", err.Error())
        return err
    }

    glog.Infof("Start sending requests.")
    results := make(chan error)
    for _, ip := range ips {
        go requestFunc(results, ip, path)
    }

    for range ips {
        err := <- results    
        if err != nil {
            return err
        }
    }

    return nil
}

func createDir(results chan error, ip string, path string) {
    glog.Infof("Sending to ip: %q to path %q", ip, path)
    targetUrl := fmt.Sprintf("http://%v:%v/directories", ip, storageManagerServicePort)
    glog.Infof("Request sent to %q.", targetUrl)

    resp, err := http.PostForm(targetUrl, url.Values{"path": {path}})
    if err != nil {
        results <- err
        return
    }

    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        results <- HttpStatusError{resp.StatusCode}
        return
    }

    results <- nil
}

func deleteDir(results chan error, ip string, path string) {
    glog.Infof("Sending to ip: %q and path %q", ip, path)
    targetUrl := fmt.Sprintf("http://%v:%v/directories?path=%v", ip, storageManagerServicePort, path)
    glog.Infof("Request sent to %q.", targetUrl)

    req, err := http.NewRequest(http.MethodDelete, targetUrl, nil)
    if err != nil {
        results <- err
        return
    }

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        results <- err
        return
    }

    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        results <- HttpStatusError{resp.StatusCode}
        return
    }

    results <- nil
}



// Delete removes the storage asset that was created by Provision represented
// by the given PV.
func (p *hostPathProvisioner) Delete(_ context.Context, volume *v1.PersistentVolume) error {
	ann, ok := volume.Annotations["hostPathMultiHostProvisionerIdentity"]
	if !ok {
		return errors.New("identity annotation not found on PV")
	}
	if ann != p.identity {
		return &controller.IgnoredError{Reason: "identity annotation on PV does not match ours"}
	}

    onDelete := volume.Spec.PersistentVolumeReclaimPolicy
    if onDelete == "Retain" {
        return nil
    }

	path := volume.Spec.PersistentVolumeSource.HostPath.Path
	glog.Info("removing backing directory: %v", path)
    sendRequestToManager(path, deleteDir)

	return nil
}

func main() {
	syscall.Umask(0)

	flag.Parse()
	flag.Set("logtostderr", "true")

	// Create an InClusterConfig and use it to create a client for the controller
	// to use to communicate with Kubernetes
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("Failed to create config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	// The controller needs to know what the server version is because out-of-tree
	// provisioners aren't officially supported until 1.5
	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		glog.Fatalf("Error getting server version: %v", err)
	}

	// Create the provisioner: it implements the Provisioner interface expected by
	// the controller
	hostPathProvisioner := NewHostPathProvisioner()

	// Start the provision controller which will dynamically provision hostPath
	// PVs
	pc := controller.NewProvisionController(clientset, 
        provisionerName, 
        hostPathProvisioner,
        serverVersion.GitVersion,
        controller.LeaderElection(true),
    )

	pc.Run(context.Background())
}
