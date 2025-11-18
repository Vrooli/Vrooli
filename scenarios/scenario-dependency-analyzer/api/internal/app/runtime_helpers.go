package app

import (
	"database/sql"
	"sync"

	appconfig "scenario-dependency-analyzer/internal/config"
)

var (
	fallbackAnalyzer     *Analyzer
	fallbackAnalyzerOnce sync.Once
)

// ensureRuntime makes sure a runtime exists (used when Run hasn't set one yet).
func ensureRuntime(cfg appconfig.Config, dbConn *sql.DB) *Runtime {
	if rt := currentRuntime(); rt != nil {
		return rt
	}
	rt := NewRuntime(cfg, dbConn)
	setDefaultRuntime(rt)
	return rt
}

// analyzerInstance returns the active Analyzer, constructing a fallback if necessary.
func analyzerInstance() *Analyzer {
	if rt := currentRuntime(); rt != nil && rt.Analyzer() != nil {
		return rt.Analyzer()
	}
	fallbackAnalyzerOnce.Do(func() {
		fallbackAnalyzer = NewAnalyzer(appconfig.Load(), db)
	})
	return fallbackAnalyzer
}
