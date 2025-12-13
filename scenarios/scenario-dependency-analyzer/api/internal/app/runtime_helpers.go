package app

import (
	"database/sql"
	"sync"

	appconfig "scenario-dependency-analyzer/internal/config"
)

var (
	configOnce   sync.Once
	cachedConfig appconfig.Config
)

func loadConfig() appconfig.Config {
	configOnce.Do(func() {
		cachedConfig = appconfig.Load()
	})
	return cachedConfig
}

// ensureRuntime makes sure a runtime exists (used when Run hasn't set one yet).
func ensureRuntime(cfg appconfig.Config, dbConn *sql.DB) *Runtime {
	if rt := currentRuntime(); rt != nil {
		return rt
	}
	rt := NewRuntime(cfg, dbConn)
	setDefaultRuntime(rt)
	return rt
}

// analyzerInstance returns the active Analyzer, constructing via runtime if needed.
func analyzerInstance() *Analyzer {
	if rt := currentRuntime(); rt != nil && rt.Analyzer() != nil {
		return rt.Analyzer()
	}
	rt := ensureRuntime(loadConfig(), db)
	if rt != nil {
		return rt.Analyzer()
	}
	return nil
}
