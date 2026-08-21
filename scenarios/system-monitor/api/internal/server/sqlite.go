package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"

	sqliterepo "github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/sqlite"
)

func connectSQLite() (*sqliterepo.Repository, *filerouting.RoutedRoots, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{})
	if err != nil {
		return nil, nil, fmt.Errorf("create storage resolver: %w", err)
	}

	primaryPaths, err := resolver.Resolve(storage.Options{ScenarioID: "system-monitor"})
	if err != nil {
		return nil, nil, fmt.Errorf("resolve storage roots: %w", err)
	}
	roots := filerouting.New(primaryPaths)
	ctx := context.Background()
	// RoutedRoots.Pick selects the class root through the same seam used by
	// request-scoped file stores; the database itself is routed by RoutedDB.
	dataRoot, err := roots.Pick(ctx, storage.ClassData)
	if err != nil {
		return nil, nil, fmt.Errorf("pick data root: %w", err)
	}
	dbPath := filepath.Join(dataRoot, "system-monitor.db")

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, nil, fmt.Errorf("create db directory: %w", err)
	}

	repo, err := sqliterepo.NewRepository(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open sqlite: %w", err)
	}

	log.Printf("SQLite database opened at %s", dbPath)
	return repo, roots, nil
}
