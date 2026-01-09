// Package handlers provides HTTP handlers organized by domain concept.
// This package follows "screaming architecture" - the structure reflects
// the domain model (scores, config, health, analysis) rather than
// technical concerns (controllers, middlewares).
package handlers

import (
	"regexp"
	"strings"

	"scenario-completeness-scoring/pkg/analysis"
	"scenario-completeness-scoring/pkg/circuitbreaker"
	"scenario-completeness-scoring/pkg/collectors"
	"scenario-completeness-scoring/pkg/config"
	"scenario-completeness-scoring/pkg/health"
	"scenario-completeness-scoring/pkg/history"
)

// scenarioNamePattern validates scenario names
// ASSUMPTION: Scenario names follow standard Vrooli naming conventions
// HARDENED: Explicit validation prevents path traversal and injection attacks
var scenarioNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

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

// ValidateScenarioName validates a scenario name for safety and correctness
// Returns an error message if invalid, empty string if valid
// ASSUMPTION: Scenario names are alphanumeric with hyphens/underscores
// HARDENED: Prevents path traversal attacks (../) and empty names
func ValidateScenarioName(name string) string {
	if name == "" {
		return "scenario name cannot be empty"
	}
	if len(name) > 64 {
		return "scenario name too long (max 64 characters)"
	}
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "scenario name contains invalid characters"
	}
	if !scenarioNamePattern.MatchString(name) {
		return "scenario name must start with alphanumeric and contain only letters, numbers, hyphens, and underscores"
	}
	return ""
}
