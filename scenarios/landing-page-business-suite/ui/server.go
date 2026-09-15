package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	port := os.Getenv("UI_PORT")
	if port == "" {
		port = "3000"
	}
	root := filepath.Join(filepath.Dir(os.Args[0]), "dist")
	if _, err := os.Stat(root); err != nil {
		root = "dist"
	}
	files := http.FileServer(http.Dir(root))
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		}
		files.ServeHTTP(w, r)
	})
	log.Printf("landing-page-business-suite UI listening on %s", port)
	log.Fatal(http.ListenAndServe(":"+port, h))
}
