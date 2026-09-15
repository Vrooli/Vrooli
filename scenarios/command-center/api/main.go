package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"command-center/internal/trends"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

const scenarioName = "command-center"

func main() {
	if preflight.Run(preflight.Config{
		ScenarioName: "command-center",
	}) {
		return
	}

	registryPath := resolveRegistryPath()
	reg, err := LoadRegistry(registryPath)
	if err != nil {
		log.Fatalf("failed to load outcome registry at %s: %v", registryPath, err)
	}
	slog.Info("loaded outcome registry", "path", registryPath, "rooms", len(reg.Rooms))

	dsn, err := storage.SQLiteDSN(storage.SQLiteConfig{Scenario: scenarioName})
	if err != nil {
		log.Fatalf("failed to resolve trend storage: %v", err)
	}
	db, err := database.Connect(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: dsn, MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		log.Fatalf("failed to open trend storage: %v", err)
	}
	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(trends.Schema)); err != nil {
		log.Fatalf("failed to initialize trend storage: %v", err)
	}
	srv := NewServerWithTrendStore(reg, trends.NewSQLiteStore(db))

	if err := server.Run(server.Config{
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// resolveRegistryPath returns the path to the versioned outcome registry.
// Honors COMMAND_CENTER_REGISTRY_PATH for tests; otherwise resolves
// ../config/outcome-registry.json
// relative to the API binary's working directory.
func resolveRegistryPath() string {
	if p := os.Getenv("COMMAND_CENTER_REGISTRY_PATH"); p != "" {
		return p
	}
	return filepath.Join("..", "config", "outcome-registry.json")
}
