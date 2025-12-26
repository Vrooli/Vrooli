package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
)

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "audio-tools",
	}) {
		return // Process was re-exec'd after rebuild
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "healthy", "service": "audio-tools", "timestamp": "%d"}`, time.Now().Unix())
	})

	log.Printf("Starting audio-tools API server")
	if err := server.Run(server.Config{
		Cleanup: func(ctx context.Context) error {
			return nil
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
