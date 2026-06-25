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
)

// resolveEventDBPath returns the SQLite DSN for the event log database.
func resolveEventDBPath() (string, error) {
	if p := os.Getenv("SWARM_MANAGER_SQLITE_PATH"); p != "" {
		return "file:" + p + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=synchronous(NORMAL)", nil
	}
	dbPath, err := runtimepaths.DataPath("events.db")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o750); err != nil {
		return "", err
	}
	return "file:" + dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=synchronous(NORMAL)", nil
}

// initEventLog initializes the event log database, emitter, and stats engine.
func (s *Server) initEventLog() {
	dsn, err := resolveEventDBPath()
	if err != nil {
		slog.Warn("failed to resolve event database path", "error", err)
		return
	}
	eventDB, err := database.Connect(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		slog.Warn("failed to open event database, stats will be unavailable", "error", err)
		return
	}
	repo := eventlog.NewSQLiteRepository(eventDB)
	if err := repo.InitSchema(context.Background()); err != nil {
		slog.Error("event log schema init error", "error", err)
		s.eventDB = eventDB
		return
	}
	s.eventDB = eventDB
	s.eventRepo = repo
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
	if s.recordsService != nil {
		s.recordsService.SetEventLogger(s.emitter)
	}
}
