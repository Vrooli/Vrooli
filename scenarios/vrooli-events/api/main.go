package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/server"
	"github.com/vrooli/vrooli/packages/proto/descriptorimage"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/api/internal/modules"
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

	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		log.Fatalf("create db dir: %v", err)
	}

	db, err := database.Open(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          cfg.DBPath,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	if err := database.EnsureSchemas(ctx, db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	db.SetTestPoolInitializer(func(ctx context.Context, testDB *sql.DB) error {
		return database.EnsureSchemas(ctx, testDB, modules.AllSchemas()...)
	})

	eventStore, err := store.NewSQLiteStoreWithDB(ctx, db, store.SQLiteConfig{
		MaxAge:        cfg.MaxAge,
		MaxSizeBytes:  cfg.MaxSizeBytes,
		QueryLimit:    cfg.QueryLimit,
		QueryLimitMax: cfg.QueryLimitMax,
	})
	if err != nil {
		log.Fatalf("open event store: %v", err)
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

	polStore, err := policy.NewSQLiteStore(db)
	if err != nil {
		log.Fatalf("open policy store: %v", err)
	}

	subStore, err := subscription.NewSQLiteStore(db)
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
	descriptorSource, err := descriptorimage.NewForRepo(captureValidationRepoRoot())
	if err != nil {
		log.Fatalf("descriptor source: %v", err)
	}
	if _, err := descriptorSource.LoadWithRetry(5, 100*time.Millisecond); err != nil {
		log.Fatalf("descriptor initial load: %v", err)
	}
	srv.descriptorSource = descriptorSource
	// A generation is ordered across ordinary restarts as well as in-process
	// mutations. The value is opaque to clients; only monotonic replacement
	// matters. Wall-clock nanoseconds provide a durable ordering floor without
	// making policy reads depend on another persistence layer.
	srv.policyVersion.Store(time.Now().UTC().UnixNano())
	go srv.runSubscriptionDispatcher(ctx)

	// These baseline headers apply to every API response, including rejected
	// provenance and malformed-ingest requests. They are safe on localhost and
	// prevent accidental weakening when Events is later exposed behind TLS.
	routes := srv.routes()
	validationPath, validationHandler := scenariovalidationv1connect.NewScenarioValidationServiceHandler(newCaptureValidationHandler(captureValidationRepoRoot(), polStore, eventStore))
	routes.Handle(validationPath, validationHandler)
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)
	aggregateStore := store.ReceiptAggregateStore(eventStore)
	receiptMeasures, err := receiptMeasuresHandler(aggregateStore, schedule.System())
	if err != nil {
		log.Fatalf("receipt measures: %v", err)
	}
	rootMux.Handle("/measures/", http.StripPrefix("/measures", receiptMeasures))
	rootMux.Handle("/", securityHeaders(provenance.Middleware(provenance.CLIUtilVerifier{})(routes)))

	if err := server.Run(server.Config{
		Handler:      apihttp.TestModeMiddleware(rootMux),
		WriteTimeout: 0, // SSE requires no write timeout
		Cleanup: func(_ context.Context) error {
			cancel()
			eventBroker.Close()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "0")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
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
	descriptorSource  *descriptorimage.Source
	policyVersion     atomic.Int64
}
