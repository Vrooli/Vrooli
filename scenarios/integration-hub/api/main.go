package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/health"
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
	mux.HandleFunc("/health", healthHandler())
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

// healthHandler serves /health through api-core/health so the response carries
// the lifecycle build and start identity the control plane verifies on start.
func healthHandler() http.HandlerFunc { return health.New("integration-hub").Handler() }
