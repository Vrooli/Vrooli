package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	commonv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
)

func main() {
	stateDir := os.Getenv("INTEGRATION_HUB_DATA_DIR")
	if stateDir == "" {
		stateDir = os.Getenv("SCENARIO_DATA_DIR")
	}
	if stateDir == "" {
		stateDir = filepath.Join("data")
	}
	store, err := NewStore(filepath.Join(stateDir, "connections.json"))
	if err != nil {
		log.Fatal(err)
	}
	hub := NewHub(store, cliCredentialStore{})
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "healthy", "service": "integration-hub", "timestamp": time.Now().UTC().Format(time.RFC3339), "readiness": true})
	})
	path, handler := commonv1connect.NewConnectionServiceHandler(hub)
	mux.Handle(path, handler)
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "15000"
	}
	server := &http.Server{Addr: ":" + port, Handler: mux}
	log.Printf("integration-hub listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
