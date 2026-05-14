package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"react-component-library/internal/clock"
	"react-component-library/internal/modules"
	"react-component-library/internal/server"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	adoptionsH "react-component-library/handlers/adoptions"
	componentsH "react-component-library/handlers/components"
	healthH "react-component-library/handlers/health"
	previewH "react-component-library/handlers/preview"
	versionsH "react-component-library/handlers/versions"

	adoptionsInternal "react-component-library/internal/adoptions"
	componentsInternal "react-component-library/internal/components"
)

// sqliteDSN resolves the SQLite database file path and wraps it in a DSN
// with the canonical pragma string. Resolution order:
//
//  1. SQLITE_PATH env — the canonical override.
//  2. SQLITE_DB env — alias accepted for symmetry with other scenarios.
//  3. storage.NewResolver(ProfileAuto) — the storage-steer-mandated
//     filesystem-safe-by-default location.
//
// The pragmas mirror agent-inbox; tweak in lockstep with
// internal/testutil/db.NewSQLite so production and tests open files the
// same way.
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
		storage.Options{ScenarioID: "react-component-library"},
		storage.ClassData,
		"react-component-library.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve react-component-library db path: %w", err)
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
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "react-component-library"}) {
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

	sourceRoot, err := componentsH.DefaultSourceRoot()
	if err != nil {
		log.Fatalf("components source root: %v", err)
	}
	componentsSvc, componentsRepo := componentsH.BuildService(db, clock.System{}, sourceRoot)

	scenariosRoot, err := adoptionsH.DefaultScenariosRoot()
	if err != nil {
		log.Fatalf("adoptions scenarios root: %v", err)
	}
	adoptionsSvc, scenariosReader := adoptionsH.BuildService(db, clock.System{}, adoptionsH.LibraryFromComponents(componentsSvc), scenariosRoot)

	// Install the swarm-manager drift reporter so Refresh files a
	// `fix` backlog item when an adoption first transitions to
	// behind/modified. CLI-only per [feedback_skills_use_cli_never_api.md];
	// disable by setting RCL_DRIFT_REPORTER=off.
	if strings.TrimSpace(os.Getenv("RCL_DRIFT_REPORTER")) != "off" {
		reporter := adoptionsInternal.NewSwarmManagerCLIReporter(nil)
		if path := strings.TrimSpace(os.Getenv("RCL_SWARM_MANAGER_BIN")); path != "" {
			reporter.BinaryPath = path
		}
		adoptionsInternal.SetDriftReporter(adoptionsSvc, reporter, log.Default())
	}

	versionsResolver := versionsH.AdoptionResolverFromService(adoptionsSvc, scenariosReader)
	versionsSvc := versionsH.BuildService(db, clock.System{}, versionsResolver)

	// Wire post-save versions recording. Listener errors are logged
	// inside the adapter; UpdateContent does not fail when recording
	// fails (the file is already on disk).
	componentsInternal.SetContentChangeListener(componentsSvc, &versionsH.ListenerAdapter{
		Service: versionsSvc,
		Logger:  log.Default(),
	})

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.Default()},
		adoptionsH.ModuleFromService(adoptionsSvc, log.Default()),
		componentsH.ModuleFromService(componentsSvc, componentsRepo, sourceRoot, log.Default()),
		healthH.Module(db, "react-component-library-api", "1.0.0"),
		previewH.Module(componentsSvc, log.Default()),
		versionsH.Module(db, clock.System{}, versionsResolver, log.Default()),
	)

	if err := apiserver.Run(apiserver.Config{
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
