
package app

import (
	"database/sql"

	appconfig "scenario-dependency-analyzer/internal/config"
	"scenario-dependency-analyzer/internal/store"
)

// Runtime encapsulates shared state for the analyzer process.
type Runtime struct {
	cfg      appconfig.Config
	db       *sql.DB
	store    *store.Store
	analyzer *Analyzer
}

var defaultRuntime *Runtime

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

// Analyzer exposes the runtime's analyzer instance.
func (rt *Runtime) Analyzer() *Analyzer { return rt.analyzer }

// Store exposes the runtime's backing store instance.
func (rt *Runtime) Store() *store.Store { return rt.store }

// DB exposes the runtime's database handle.
func (rt *Runtime) DB() *sql.DB { return rt.db }

// Config exposes the runtime configuration.
func (rt *Runtime) Config() appconfig.Config { return rt.cfg }
