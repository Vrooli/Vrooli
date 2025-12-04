package app

import (
	"database/sql"

	appconfig "scenario-dependency-analyzer/internal/config"
	"scenario-dependency-analyzer/internal/store"
	types "scenario-dependency-analyzer/internal/types"
)

// Package-level state for the analyzer process.
// Consolidated here for discoverability; prefer using Runtime methods where possible.
var (
	// defaultRuntime is the active runtime instance set by Run().
	defaultRuntime *Runtime

	// db is the fallback database connection used when no runtime is active.
	// Set by Run() for backward compatibility with code that accesses db directly.
	db *sql.DB

	// applyDiffsHook is an optional callback invoked during config sync operations.
	// Used primarily for testing to observe or modify apply behavior.
	applyDiffsHook func(string, *types.ServiceConfig)
)

// Runtime encapsulates shared state for the analyzer process.
type Runtime struct {
	cfg      appconfig.Config
	db       *sql.DB
	store    *store.Store
	analyzer *Analyzer
}

// NewRuntime constructs a runtime from configuration and database handle.
func NewRuntime(cfg appconfig.Config, dbConn *sql.DB) *Runtime {
	var backingStore *store.Store
	if dbConn != nil {
		backingStore = store.New(dbConn)
	}
	return &Runtime{
		cfg:      cfg,
		db:       dbConn,
		store:    backingStore,
		analyzer: NewAnalyzer(cfg, dbConn),
	}
}

// setDefaultRuntime stores the provided runtime as the global reference.
func setDefaultRuntime(rt *Runtime) {
	defaultRuntime = rt
}

// currentRuntime returns the active runtime instance (if any).
func currentRuntime() *Runtime {
	return defaultRuntime
}

// currentDB returns the database handle from the active runtime, or the global fallback.
// Prefer using currentStore() for new code, but this provides a bridge for legacy code.
func currentDB() *sql.DB {
	if rt := currentRuntime(); rt != nil && rt.db != nil {
		return rt.db
	}
	return db
}

// Analyzer exposes the runtime's analyzer instance.
func (rt *Runtime) Analyzer() *Analyzer { return rt.analyzer }

// Store exposes the runtime's backing store instance.
func (rt *Runtime) Store() *store.Store { return rt.store }

// DB exposes the runtime's database handle.
func (rt *Runtime) DB() *sql.DB { return rt.db }

// Config exposes the runtime configuration.
func (rt *Runtime) Config() appconfig.Config { return rt.cfg }
