package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/server"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/broker"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		home, _ := os.UserHomeDir()
		dbPath = filepath.Join(home, ".vrooli", "vrooli-events", "events.db")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatalf("create db dir: %v", err)
	}

	eventStore, err := store.NewSQLiteStore(ctx, store.SQLiteConfig{DBPath: dbPath})
	if err != nil {
		log.Fatalf("open store: %v", err)
	}

	eventBroker := broker.NewBroker(eventStore)

	// Start background pruner
	go store.StartPruner(ctx, store.PrunerConfig{Store: eventStore})

	srv := &Server{
		store:  eventStore,
		broker: eventBroker,
	}

	mux := srv.routes()

	if err := server.Run(server.Config{
		Handler:      mux,
		WriteTimeout: 0, // SSE requires no write timeout
		Cleanup: func(_ context.Context) error {
			cancel()
			eventBroker.Close()
			return eventStore.Close()
		},
	}); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// Server holds dependencies for HTTP handlers.
type Server struct {
	store  store.Store
	broker *broker.Broker
}
