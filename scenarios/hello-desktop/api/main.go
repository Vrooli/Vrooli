package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "23100"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// The desktop runtime's /health contract accepts "healthy" or "degraded".
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})
	mux.HandleFunc("/api/greet", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "Desktop User"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Hello, " + name + "!"})
	})
	log.Printf("hello-desktop API listening on 127.0.0.1:%s", port)
	log.Fatal(http.ListenAndServe("127.0.0.1:"+port, mux))
}
