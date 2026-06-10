package server

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/storage"

	sqliterepo "github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/sqlite"
)

func connectSQLite() (*sqliterepo.Repository, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{})
	if err != nil {
		return nil, fmt.Errorf("create storage resolver: %w", err)
	}

	dbPath, err := resolver.Path(
		storage.Options{ScenarioID: "system-monitor"},
		storage.ClassData,
		"system-monitor.db",
	)
	if err != nil {
		return nil, fmt.Errorf("resolve db path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	repo, err := sqliterepo.NewRepository(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	log.Printf("SQLite database opened at %s", dbPath)
	return repo, nil
}
