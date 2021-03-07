# Dynamic Provisioning of Kubernetes HostPath Volumes

## Installation
```bash
# install dynamic hostpath provisioner
kubectl apply -f https://raw.githubusercontent.com/kubeboost/hostpath-provisioner/master/manifests/rbac.yaml
kubectl apply -f https://raw.githubusercontent.com/kubeboost/hostpath-provisioner/master/manifests/deployment.yaml
kubectl apply -f https://raw.githubusercontent.com/kubeboost/hostpath-provisioner/master/manifests/daemonset.yaml
kubectl apply -f https://raw.githubusercontent.com/kubeboost/hostpath-provisioner/master/manifests/service.yaml
kubectl apply -f https://raw.githubusercontent.com/kubeboost/hostpath-provisioner/master/manifests/storageclass.yaml

# create a test-pvc and a pod writing to it
kubectl apply -f https://raw.githubusercontent.com/kubeboost/hostpath-provisioner/master/manifests/test-claim.yaml
kubectl apply -f https://raw.githubusercontent.com/kubeboost/hostpath-provisioner/master/manifests/test-pod.yaml

# expect a file to exist on your host
$ ls -la /var/kubernetes/default-hostpath-multihost-test-claim-pvc-*/

kubectl delete pod hostpath-multihost-test-pod
kubectl delete pvc hostpath-multihost-test-claim

# expect the file and folder to be removed from your host
$ ls -la /var/kubernetes/default-hostpath-multihost-test-claim-pvc-*/
```

# Dynamic Provisioning

In order for dynamic provision, the process of allocating and binding a suitable Volume to a PersistentVolumeClaim, to happen, a Workload (usually a single Pod) needs to watch the Kubernetes API for new Claims, create Volumes for them and Bind the Volume to the Claim. Similar the same Workload is responsible to remove unneeded Volumes when the Claim goes away and the RetainPolicy does not tell otherwise.

For GoogleComputeEngine, Amazon AWS and even for Minikube there are such Provisioners that know how to handle the creation of GCE-Disks, AWS-Disks or HostPaths for Minikube.

# The Dynamic HostPath MultiHost Provisioner

This is a hard fork of the HostPath Provisioner given in the [hostpath-provisioner](https://github.com/MaZderMind/hostpath-provisioner)-Project. It adds the functionality to create a hostpath volume that is available in every node in the cluster, which is useful for DaemonSets.

## How does it works?

The Hostpath Multihost Provisioner is composed by the Provisioner, a Deployment with just one replica that listens for the creation and deletion of PersistentVolumeClaims; and the Manager, a DaemonSet which creates and deletes the volume directories from every node.

The Provisioner creates PersistentVolumes for the PersistentVolumeClaims attached to its StorageClasses. When a new PersistentVolumeClaim is created for the given Provisioner, it sends a request to every Manager pod, to ensure that the volume directory is created on every node. When a PersistentVolumeClaim managed by this Provisioner is deleted there are to choices, if the PersistentVolumeReclaimPolicy is set to Retain, it sets the PersistentVolume as Released, to be managed by the system administrator; if the PersistentVolumeReclaimPolicy is set to Delete, then it deletes the PersistentVolume resource and send a delete request to the manager, to ensure that the volume directories are deleted from every node.

The created PersistentVolumes are not attached to any node, then, Pods running on different nodes will be able to attach the same PersistentVolumeClaim, and as consequence, every Pod will write to the node where the Pod is running.

## Why to use it?

This provider is really useful to be used along with DaemonSets. DaemonSets currently do not support PersistentVolumeClaimsTemplates, as StatefulSets do, so most of the times you have to work with `volumes.hostPath` instead of PersistentVolumeClaims. The `volumes.hostPath` field has a drawback, which is that it breaks encapsulation, as different applications would be able to access the same resource even if is not intendeed.

With HostPath MultiHost Provisioner, you can ensure to provide an isolated path for your DaemonSets where to write data, and also allow the Pods in different nodes to attatch to the same PersistentVolumeClaim.  Furthermore, this Provisioner, clean up the data on every node, when a PersistentVolumeClaim with ReclaimPolicy Delete is removed, so this Provisioner is prefered over using directly `volumes.hostPath` field, just for the clean up of the volume when the PersistentVolumeClaim is not needed anymore.

This Provider is useful as long as DaemonSets do not implement a PersistentVolumeClaimTemplate, as StatefulSet, as it is at the moment of writing this doc.
## What can I customize?

The goal of this Provisioner is not to be highly customizable, but easy and simple to use. That is the reason why some of the customization included in [hostpath-provisioner](https://github.com/MaZderMind/hostpath-provisioner)-Project are stripped out, as the `PV_RECLAIM_POLICY` environment variable, which allowed to overwrite the Reclaim Policy set on the StorageClass. Then, if you want to set a ReclaimPolicy for your PersistentVolumes, use the ReclaimPolicy field in your StorageClass instead of relying on the Provider.

The only customization left to the final user, as is the only one which is meaningful and which can impact them, it is the path where to create the volumes on every node. To change the path just open the `manifests/daemonset.yaml` and change the variable `spec.template.spec.volumes.hostPath.path` to the path where you desire that the Manager creates the volumes on your nodes.
