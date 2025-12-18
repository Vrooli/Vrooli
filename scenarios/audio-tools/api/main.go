package main

import (
	"github.com/vrooli/api-core/preflight"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "audio-tools",
	}) {
		return // Process was re-exec'd after rebuild
	}

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "healthy", "service": "audio-tools", "timestamp": "%d"}`, time.Now().Unix())
	})

	log.Printf("Starting API server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
