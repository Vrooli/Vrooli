package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"ui-health/internal/aisearch"
	"ui-health/internal/clock"
	"ui-health/internal/modules"
	"ui-health/internal/server"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	_ "modernc.org/sqlite"

	healthH "ui-health/handlers/health"
	reindexH "ui-health/handlers/reindex"
	searchH "ui-health/handlers/search"
	validationH "ui-health/handlers/validation"
)

func sqliteDSN() (string, error) {
	if path := strings.TrimSpace(os.Getenv("SQLITE_PATH")); path != "" {
		return sqliteFileDSN(path)
	}
	if path := strings.TrimSpace(os.Getenv("SQLITE_DB")); path != "" {
		return sqliteFileDSN(path)
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: "ui-health"},
		storage.ClassData,
		"ui-health.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve ui-health db path: %w", err)
	}
	return sqliteFileDSN(path)
}

func sqliteFileDSN(path string) (string, error) {
	if strings.HasPrefix(path, "file:") {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		path,
	), nil
}

func main() {
	if preflight.Run(preflight.Config{ScenarioName: "ui-health"}) {
		return
	}

	dsn, err := sqliteDSN()
	if err != nil {
		log.Fatalf("sqlite configuration failed: %v", err)
	}

	db, err := database.Connect(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db, modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	logger := log.Default()
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		log.Fatalf("resolve repo root: %v", err)
	}

	// AI search wiring: embedder + vector store + discovery + reconciler.
	searchCfg := aisearch.LoadConfigFromEnv()
	embedder := aisearch.NewEmbedder(searchCfg.EmbedModel)
	vectorStore := aisearch.NewVectorStore(searchCfg.QdrantURL, searchCfg.QdrantAPIKey, aisearch.DefaultCollection, aisearch.DefaultVectorSize)
	discovery := aisearch.NewFilesystemDiscoverySource(repoRoot)
	aiService := aisearch.NewService(aisearch.Options{
		Embedder:    embedder,
		VectorStore: vectorStore,
		Discovery:   discovery,
		Parallelism: searchCfg.ReconcileParallelism,
		Threshold:   searchCfg.SearchThreshold,
	})

	if err := aiService.EnsureCollection(context.Background()); err != nil {
		logger.Printf("[ui-health] qdrant collection ensure failed (continuing with degraded search): %v", err)
	}

	syncCtx, cancelSync := context.WithCancel(context.Background())
	syncLoop := aisearch.NewSyncLoop(aiService.Reconciler())
	go syncLoop.Start(syncCtx)

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: logger},
		healthH.Module(db, "ui-health-api", "1.0.0"),
		validationH.Module(logger, repoRoot),
		searchH.Module(logger, aiService),
		reindexH.Module(logger, reindexH.ServiceAdapter{Service: aiService}),
	)

	if err := apiserver.Run(apiserver.Config{
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error {
			cancelSync()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
