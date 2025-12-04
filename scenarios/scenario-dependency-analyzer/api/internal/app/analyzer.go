package app

import (
	"database/sql"

	"scenario-dependency-analyzer/internal/app/services"
	appconfig "scenario-dependency-analyzer/internal/config"
	"scenario-dependency-analyzer/internal/detection"
	"scenario-dependency-analyzer/internal/seams"
	"scenario-dependency-analyzer/internal/store"
	types "scenario-dependency-analyzer/internal/types"
)

// Analyzer coordinates scenario analysis capabilities and shared state.
type Analyzer struct {
	cfg      appconfig.Config
	db       *sql.DB
	store    *store.Store
	detector *detection.Detector
	services services.Registry
	seams    *seams.Dependencies // Optional seams for testability
}

// AnalyzerOption configures an Analyzer during construction.
type AnalyzerOption func(*Analyzer)

// WithSeams sets custom seams (for testing or dependency injection).
func WithSeams(s *seams.Dependencies) AnalyzerOption {
	return func(a *Analyzer) {
		a.seams = s
	}
}

// NewAnalyzer constructs an Analyzer bound to the provided configuration and database handle.
func NewAnalyzer(cfg appconfig.Config, db *sql.DB, opts ...AnalyzerOption) *Analyzer {
	var backingStore *store.Store
	if db != nil {
		backingStore = store.New(db)
	}
	analyzer := &Analyzer{
		cfg:      cfg,
		db:       db,
		store:    backingStore,
		detector: detection.New(cfg),
		seams:    seams.Default, // Use package defaults
	}
	// Apply options
	for _, opt := range opts {
		opt(analyzer)
	}
	analyzer.services = newServices(analyzer)
	return analyzer
}

// Seams returns the analyzer's seam dependencies.
func (a *Analyzer) Seams() *seams.Dependencies {
	if a.seams == nil {
		return seams.Default
	}
	return a.seams
}

// Store exposes the backing persistence layer (primarily for tests).
func (a *Analyzer) Store() *store.Store {
	return a.store
}

// DB exposes the analyzer's database connection (primarily for health checks).
func (a *Analyzer) DB() *sql.DB {
	return a.db
}

// Detector exposes the backing detection engine.
func (a *Analyzer) Detector() *detection.Detector {
	return a.detector
}

// Services exposes the registered service implementations.
func (a *Analyzer) Services() services.Registry {
	return a.services
}

// generateDependencyGraph proxies to the legacy graph generator
// while the logic is migrated into Analyzer-owned methods.
func (a *Analyzer) generateDependencyGraph(graphType string) (*types.DependencyGraph, error) {
	return generateDependencyGraph(graphType)
}
