package app

import (
	"database/sql"
	"sync"

	appconfig "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/config"
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
		if dbConn != nil && (rt.DB() == nil || rt.DB() != dbConn) {
			rt = NewRuntime(cfg, dbConn)
			setDefaultRuntime(rt)
		}
		return rt
	}
	rt := NewRuntime(cfg, dbConn)
	setDefaultRuntime(rt)
	return rt
}

// analyzerInstance returns the active Analyzer, constructing via runtime if needed.
func analyzerInstance() *Analyzer {
	if rt := currentRuntime(); rt != nil && rt.Analyzer() != nil {
		if rt.Store() == nil && db != nil {
			rt = ensureRuntime(loadConfig(), db)
		}
		return rt.Analyzer()
	}
	rt := ensureRuntime(loadConfig(), db)
	if rt != nil {
		return rt.Analyzer()
	}
	return nil
}
