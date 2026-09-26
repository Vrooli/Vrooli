package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/runtimepaths"
	"swarm-manager/internal/stats"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
)

// resolveEventDBPath returns the SQLite DSN for the event log database.
//
// SWARM_MANAGER_SQLITE_PATH stays readable because it is scenario-scoped: its
// name carries its owner, so it cannot capture a sibling's database the way the
// generic variables did. The DSN itself now comes from the one owned seam.
func resolveEventDBPath() (string, error) {
	if p := os.Getenv("SWARM_MANAGER_SQLITE_PATH"); p != "" {
		return storage.SQLiteDSNAt(p, storage.SQLiteTuning{})
	}
	dbPath, err := runtimepaths.DataPath("events.db")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o750); err != nil {
		return "", err
	}
	return storage.SQLiteDSNAt(dbPath, storage.SQLiteTuning{})
}

// initEventLog initializes the event log database, emitter, and incremental
// Stats projection. The projection is a derived read model: event history
// remains authoritative and can always rebuild it.
func (s *Server) initEventLog() {
	dsn, err := resolveEventDBPath()
	if err != nil {
		slog.Warn("failed to resolve event database path", "error", err)
		return
	}
	// Open through database.Open so every connection is routed via
	// *database.RoutedDB. Combined with devrouting.Register + TestModeMiddleware
	// in main(), this lets test-genie install an isolated test pool at runtime so
	// destructive e2e playbooks never touch the live event log / evidence ledger.
	eventDB, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		slog.Warn("failed to open event database, event history and stats will be unavailable", "error", err)
		return
	}
	// Apply the per-domain schemas to whichever pool is active (live at boot;
	// the test pool once installed) via the shared EnsureSchemas seam. This is
	// the "primary schema application" seam storage-manager requires.
	if err := database.EnsureSchemas(context.Background(), eventDB.Primary(),
		database.SchemaProviderFunc(eventlog.Schema),
	); err != nil {
		slog.Error("event log schema init error", "error", err)
		s.eventDB = eventDB
		return
	}
	repo := eventlog.NewSQLiteRepository(eventDB)
	s.eventDB = eventDB
	s.eventRepo = repo
	s.emitter = eventlog.NewEmitter(repo)
	s.statsEngine = stats.NewEngine(repo)
	if err := s.statsEngine.Rebuild(context.Background()); err != nil {
		slog.Error("stats rebuild error", "error", err)
	} else {
		slog.Info("stats projection initialized from event history")
	}
}

// wireEventLoggers connects the event emitter to all mutating services.
func (s *Server) wireEventLoggers() {
	if s.emitter == nil {
		return
	}
	if s.backlogHandler != nil {
		s.backlogHandler.SetEventLogger(s.emitter)
	}
	if s.executionSvc != nil {
		s.executionSvc.SetEventLogger(s.emitter)
	}
	if s.goalService != nil {
		s.goalService.SetEventLogger(s.emitter)
	}
	if s.queueHandler != nil {
		s.queueHandler.SetEventLogger(s.emitter)
	}
	if s.capturesHandler != nil {
		s.capturesHandler.SetEventLogger(s.emitter)
	}
	if s.reviewSvc != nil {
		s.reviewSvc.SetEventLogger(s.emitter)
	}
	if s.recordsService != nil {
		s.recordsService.SetEventLogger(s.emitter)
	}
}
