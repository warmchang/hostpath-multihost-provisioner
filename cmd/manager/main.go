package main

import (
    "fmt"
    "net/http"
	"os"
)

func main() {
    const port = "8083"
    http.HandleFunc("/", router)

    fmt.Printf("Starting port at %q\n", port)
    fmt.Println(http.ListenAndServe(":" + port, nil))
}

type Route struct {
    path, method interface{}
}

func router(w http.ResponseWriter, r *http.Request) {
    path := r.URL.Path
    method := r.Method
    route := Route{ path, method }

    switch route {
    case Route{"/directories", http.MethodPost}:
        handleDirectoryPost(w, r)
    case Route{"/directories", http.MethodDelete}:
        handleDirectoryDelete(w, r)
    default:
        http.Error(w, "404 not found.", http.StatusNotFound)
    }
}

func handleDirectoryPost(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        fmt.Fprintf(w, "ParseForm() err: %v", err)
        return
    }

    path := r.FormValue("path")
    fmt.Printf("Creating directory %q", path)
	if err := os.MkdirAll(path, 0777); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func handleDirectoryDelete(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        fmt.Fprintf(w, "ParseForm() err: %v", err)
        return
    }

    path := r.FormValue("path")
    fmt.Printf("Removing directory %q", path)
	if err := os.RemoveAll(path); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}
