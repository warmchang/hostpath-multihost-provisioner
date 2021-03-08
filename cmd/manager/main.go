package main

import (
	"flag"
	"net/http"
	"os"
	"syscall"

	"github.com/golang/glog"
)

// Start a server listening at port 8080 and route /.
func main() {
	syscall.Umask(0)

	// Add flags for glog.
	flag.Parse()
	flag.Set("logtostderr", "true")

	// Initialize HTTP server.
	const port = "8080"
	http.HandleFunc("/", router)

	glog.Infof("Starting port at %q\n", port)
	glog.Info(http.ListenAndServe(":"+port, nil))
}

// Represent a pair of path and HTTP method. It is useful to join pairs with handle functions.
type Route struct {
	path, method string
}

// Route the request to the different handle methods or to 404 not found if not route is
// defined for the request path and method.
func router(w http.ResponseWriter, r *http.Request) {
	// Create the route from the request path and method.
	path := r.URL.Path
	method := r.Method
	route := Route{path, method}

	// Map routes with handle methods.
	switch route {
	case Route{"/directories", http.MethodPost}:
		handleDirectoryPost(w, r)
	case Route{"/directories", http.MethodDelete}:
		handleDirectoryDelete(w, r)
	default:
		http.Error(w, "404 not found.", http.StatusNotFound)
	}
}

// Handle a POST request to create a directory. It extract the path value from the request body
// and create the directory in the given path.
func handleDirectoryPost(w http.ResponseWriter, r *http.Request) {
	// Try to parse the request body form.
	if err := r.ParseForm(); err != nil {
		glog.Errorf("ParseForm() err: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Extract the path to create from the body form.
	path := r.FormValue("path")

	// Create the directory at the given path. It returns an HTTP error if the creation fails.
	glog.Infof("Creating directory %q", path)
	if err := os.MkdirAll(path, 0777); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

// Handle a DELETE request to remove a directory. It extract the path value from the URL query param
// and removes the directory and all its content.
func handleDirectoryDelete(w http.ResponseWriter, r *http.Request) {
	// Check if the query param is provided and return an HTTP error otherwise, as it is mandatory.
	keys, ok := r.URL.Query()["path"]
	if !ok {
		glog.Error("Path query param not provided")
		http.Error(w, "Path query param not provided", http.StatusBadRequest)
		return
	}
	path := keys[0]

	// Remove the directory at the given path. It returns an HTTP error if deletion fails.
	glog.Infof("Removing directory %q", path)
	if err := os.RemoveAll(path); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}
