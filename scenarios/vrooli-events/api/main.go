package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/broker"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/config"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
)

// DOC: docs/concepts/ARCHITECTURE.md
// DOC: docs/reference/configuration.md
func main() {
	if preflight.Run(preflight.Config{
		ScenarioName: "vrooli-events",
	}) {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Load()

	if err := migrateLegacyDBPath(cfg.DBPath); err != nil {
		log.Fatalf("migrate legacy db: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		log.Fatalf("create db dir: %v", err)
	}

	eventStore, err := store.NewSQLiteStore(ctx, store.SQLiteConfig{
		DBPath:        cfg.DBPath,
		MaxAge:        cfg.MaxAge,
		MaxSizeBytes:  cfg.MaxSizeBytes,
		QueryLimit:    cfg.QueryLimit,
		QueryLimitMax: cfg.QueryLimitMax,
	})
	if err != nil {
		log.Fatalf("open store: %v", err)
	}

	eventBroker := broker.NewBroker(eventStore, broker.BrokerConfig{
		SubscriberBufSize: cfg.SubscriberBufSize,
		HeartbeatInterval: cfg.HeartbeatInterval,
	})

	// Start background pruner
	go store.StartPruner(ctx, store.PrunerConfig{
		Store:    eventStore,
		Interval: cfg.PruneInterval,
	})

	polStore, err := policy.NewSQLiteStore(eventStore.DB())
	if err != nil {
		log.Fatalf("open policy store: %v", err)
	}

	subStore, err := subscription.NewSQLiteStore(eventStore.DB())
	if err != nil {
		log.Fatalf("open subscription store: %v", err)
	}

	policyBroadcaster := policy.NewPolicyBroadcaster()
	webhookDeliverer := subscription.NewWebhookDeliverer()

	srv := &Server{
		store:             eventStore,
		broker:            eventBroker,
		config:            cfg,
		policyStore:       polStore,
		policyEval:        policy.NewEvaluator(polStore),
		subStore:          subStore,
		policyBroadcaster: policyBroadcaster,
		webhookDeliverer:  webhookDeliverer,
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

func migrateLegacyDBPath(dst string) error {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	src := filepath.Join(home, ".vrooli", "vrooli-events", "events.db")
	if src == dst {
		return nil
	}
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if _, err := os.Stat(dst); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

// Server holds dependencies for HTTP handlers.
// Fields are interfaces, enabling unit tests to substitute lightweight
// mocks without requiring real SQLite or goroutine-backed brokers.
type Server struct {
	store             store.Store
	broker            broker.EventBroker
	config            config.Config
	policyStore       policy.Store
	policyEval        *policy.Evaluator
	subStore          subscription.Store
	policyBroadcaster *policy.PolicyBroadcaster
	webhookDeliverer  *subscription.WebhookDeliverer
}
