// Package handlers provides HTTP handlers organized by domain concept.
// This package follows "screaming architecture" - the structure reflects
// the domain model (scores, config, health, analysis) rather than
// technical concerns (controllers, middlewares).
package handlers

import (
	"scenario-completeness-scoring/pkg/analysis"
	"scenario-completeness-scoring/pkg/circuitbreaker"
	"scenario-completeness-scoring/pkg/collectors"
	"scenario-completeness-scoring/pkg/config"
	"scenario-completeness-scoring/pkg/health"
	"scenario-completeness-scoring/pkg/history"
)

// Context holds shared dependencies for all handlers.
// This replaces the Server struct pattern where handlers were methods
// with access to all server state.
type Context struct {
	VrooliRoot     string
	Collector      *collectors.MetricsCollector
	CBRegistry     *circuitbreaker.Registry
	HealthTracker  *health.Tracker
	ConfigLoader   *config.Loader
	HistoryDB      *history.DB
	HistoryRepo    *history.Repository
	TrendAnalyzer  *history.TrendAnalyzer
	WhatIfAnalyzer *analysis.WhatIfAnalyzer
	BulkRefresher  *analysis.BulkRefresher
}

// NewContext creates a handler context with all required dependencies.
func NewContext(
	vrooliRoot string,
	collector *collectors.MetricsCollector,
	cbRegistry *circuitbreaker.Registry,
	healthTracker *health.Tracker,
	configLoader *config.Loader,
	historyDB *history.DB,
	historyRepo *history.Repository,
	trendAnalyzer *history.TrendAnalyzer,
	whatIfAnalyzer *analysis.WhatIfAnalyzer,
	bulkRefresher *analysis.BulkRefresher,
) *Context {
	return &Context{
		VrooliRoot:     vrooliRoot,
		Collector:      collector,
		CBRegistry:     cbRegistry,
		HealthTracker:  healthTracker,
		ConfigLoader:   configLoader,
		HistoryDB:      historyDB,
		HistoryRepo:    historyRepo,
		TrendAnalyzer:  trendAnalyzer,
		WhatIfAnalyzer: whatIfAnalyzer,
		BulkRefresher:  bulkRefresher,
	}
}
