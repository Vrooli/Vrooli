package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/storage"

	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/stats"
)

// resolveEventDBPath returns the SQLite DSN for the event log database.
func resolveEventDBPath() string {
	if p := os.Getenv("SWARM_MANAGER_SQLITE_PATH"); p != "" {
		return "file:" + p + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=synchronous(NORMAL)"
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		slog.Warn("storage resolver error, using fallback", "error", err)
		home, _ := os.UserHomeDir()
		p := filepath.Join(home, ".vrooli", "data", "swarm-manager", "events.db")
		return "file:" + p + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=synchronous(NORMAL)"
	}
	dbPath, err := resolver.Path(
		storage.Options{ScenarioID: "swarm-manager"},
		storage.ClassData,
		"events.db",
	)
	if err != nil {
		slog.Warn("path resolution error, using fallback", "error", err)
		home, _ := os.UserHomeDir()
		dbPath = filepath.Join(home, ".vrooli", "data", "swarm-manager", "events.db")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		slog.Warn("mkdir error for event log", "error", err)
	}
	return "file:" + dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=synchronous(NORMAL)"
}

// initEventLog initializes the event log database, emitter, and stats engine.
func (s *Server) initEventLog() {
	dsn := resolveEventDBPath()
	eventDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		slog.Warn("failed to open event database, stats will be unavailable", "error", err)
		return
	}
	eventDB.SetMaxOpenConns(1)
	eventDB.SetMaxIdleConns(1)
	repo := eventlog.NewSQLiteRepository(eventDB)
	if err := repo.InitSchema(context.Background()); err != nil {
		slog.Error("event log schema init error", "error", err)
		s.eventDB = eventDB
		return
	}
	s.eventDB = eventDB
	s.emitter = eventlog.NewEmitter(repo)
	s.statsEngine = stats.NewEngine(repo)
	if err := s.statsEngine.Rebuild(context.Background()); err != nil {
		slog.Error("stats rebuild error", "error", err)
	} else {
		slog.Info("stats engine initialized, replayed events")
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
	if s.initiativeService != nil {
		s.initiativeService.SetEventLogger(s.emitter)
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
}
