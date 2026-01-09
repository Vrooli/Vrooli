// Package seams provides interfaces for external dependencies and variation points.
//
// These seams enable testability by allowing behavior substitution at key boundaries:
// - Clock: Time operations (time.Now)
// - IDGenerator: Unique identifier generation (uuid.New)
// - FileSystem: Filesystem operations (os.ReadDir, os.Stat, os.ReadFile, os.WriteFile)
// - DeploymentReporter: Deployment analysis operations
//
// Usage:
//
//	// Production code uses default implementations
//	deps := seams.NewDefaults()
//
//	// Test code can substitute behavior
//	deps := &seams.Dependencies{
//	    Clock: &mockClock{fixedTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
//	    IDs:   &mockIDGenerator{nextID: "test-uuid-1"},
//	}
package seams

import (
	"io/fs"
	"time"
)

// Clock abstracts time operations to enable deterministic testing.
type Clock interface {
	Now() time.Time
}

// IDGenerator abstracts unique identifier generation.
type IDGenerator interface {
	NewID() string
}

// FileSystem abstracts filesystem operations for testability.
type FileSystem interface {
	ReadDir(name string) ([]fs.DirEntry, error)
	Stat(name string) (fs.FileInfo, error)
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
}

// DeploymentReporter abstracts deployment analysis operations.
// This seam isolates the deployment package from consumers.
type DeploymentReporter interface {
	BuildReport(scenarioName, scenarioPath, scenariosDir string, cfg interface{}) interface{}
	LoadReport(scenarioPath string) (interface{}, error)
	PersistReport(scenarioPath string, report interface{}) error
}

// DependencyStore abstracts persistence operations for dependency data.
// This seam enables testing without a real database connection.
type DependencyStore interface {
	// StoreDependencies persists the declared and detected dependencies for a scenario.
	StoreDependencies(analysis interface{}, extras interface{}) error
	// LoadStoredDependencies returns all persisted dependencies for a scenario, grouped by type.
	LoadStoredDependencies(scenario string) (map[string]interface{}, error)
	// LoadAllDependencies returns all scenario dependencies ordered for graph generation.
	LoadAllDependencies() (interface{}, error)
	// UpdateScenarioMetadata upserts cached metadata about a scenario's service.json.
	UpdateScenarioMetadata(name string, cfg interface{}, scenarioPath string) error
	// LoadScenarioMetadataMap fetches cached summaries for each scenario.
	LoadScenarioMetadataMap() (map[string]interface{}, error)
	// LoadOptimizationRecommendations fetches stored optimization recommendations.
	LoadOptimizationRecommendations(scenario string) (interface{}, error)
	// PersistOptimizationRecommendations replaces prior recommendations for a scenario.
	PersistOptimizationRecommendations(scenario string, recs interface{}) error
	// CollectAnalysisMetrics aggregates summary stats for health endpoints.
	CollectAnalysisMetrics() (map[string]interface{}, error)
	// CleanupInvalidScenarioDependencies removes scenario dependency rows for names
	// that are no longer known to the catalog.
	CleanupInvalidScenarioDependencies(known map[string]struct{}) error
}

// DependencyDetector abstracts dependency detection operations.
// This seam enables testing with controlled detection results.
type DependencyDetector interface {
	// RefreshCatalogs invalidates cached scenario/resource catalogs.
	RefreshCatalogs()
	// KnownScenario reports whether the provided name exists in the catalog.
	KnownScenario(name string) bool
	// KnownResource reports whether the provided resource is part of the catalog.
	KnownResource(name string) bool
	// ScenarioCatalog returns a copy of the cached scenario name set.
	ScenarioCatalog() map[string]struct{}
	// ScanResources performs resource detection for the scenario.
	ScanResources(scenarioPath, scenarioName string, cfg interface{}) (interface{}, error)
	// ScanScenarioDependencies returns detected scenario-to-scenario edges.
	ScanScenarioDependencies(scenarioPath, scenarioName string) (interface{}, error)
	// ScanSharedWorkflows detects shared workflow references.
	ScanSharedWorkflows(scenarioPath, scenarioName string) (interface{}, error)
}

// Dependencies aggregates all seam interfaces for convenient injection.
type Dependencies struct {
	Clock      Clock
	IDs        IDGenerator
	FS         FileSystem
	Deployment DeploymentReporter
	Store      DependencyStore
	Detector   DependencyDetector
}
