package detection

import (
	appconfig "scenario-dependency-analyzer/internal/config"
	types "scenario-dependency-analyzer/internal/types"
)

// detector.go - Main detector coordinator
//
// This file provides the public API for dependency detection, coordinating
// between catalog management, resource scanning, and scenario scanning.
// All heavy lifting is delegated to specialized components.

// Detector encapsulates resource/scenario scanning and catalog discovery logic.
// It coordinates between catalog management, resource detection, and scenario detection.
type Detector struct {
	cfg             appconfig.Config
	catalog         *catalogManager
	resourceScanner *resourceScanner
	scenarioScanner *scenarioScanner
}

// New creates a detector bound to the provided configuration.
func New(cfg appconfig.Config) *Detector {
	catalog := newCatalogManager(cfg)

	return &Detector{
		cfg:             cfg,
		catalog:         catalog,
		resourceScanner: newResourceScanner(catalog),
		scenarioScanner: newScenarioScanner(catalog),
	}
}

// Public API - Catalog queries

// RefreshCatalogs invalidates cached scenario/resource catalogs.
// Call this when scenarios or resources are added/removed.
func (d *Detector) RefreshCatalogs() {
	d.catalog.refresh()
}

// KnownScenario reports whether the provided name exists in the catalog.
func (d *Detector) KnownScenario(name string) bool {
	return d.catalog.isKnownScenario(name)
}

// KnownResource reports whether the provided resource is part of the catalog.
func (d *Detector) KnownResource(name string) bool {
	return d.catalog.isKnownResource(name)
}

// ScenarioCatalog returns a copy of the cached scenario name set.
func (d *Detector) ScenarioCatalog() map[string]struct{} {
	return d.catalog.getScenarioCatalog()
}

// Public API - Dependency scanning

// ScanResources performs resource detection for the scenario.
//
// It scans the scenario directory for:
//   - Explicit resource CLI commands (e.g., "resource-postgres")
//   - Resource usage patterns via heuristics (connection strings, env vars)
//   - Resources referenced in initialization files
//
// Returns a list of detected resource dependencies.
func (d *Detector) ScanResources(scenarioPath, scenarioName string, cfg *types.ServiceConfig) ([]types.ScenarioDependency, error) {
	return d.resourceScanner.scan(scenarioPath, scenarioName, cfg)
}

// ScanScenarioDependencies returns detected scenario-to-scenario edges.
//
// It scans the scenario directory for:
//   - "vrooli scenario" CLI commands
//   - Direct CLI script references
//   - Scenario port resolution calls
//
// Returns a list of detected scenario dependencies.
func (d *Detector) ScanScenarioDependencies(scenarioPath, scenarioName string) ([]types.ScenarioDependency, error) {
	return d.scenarioScanner.scanDependencies(scenarioPath, scenarioName)
}

// ScanSharedWorkflows detects shared workflow references.
//
// It scans the initialization directory for workflow files (n8n, huginn, windmill)
// that reference shared workflow templates.
//
// Returns a list of detected workflow dependencies.
func (d *Detector) ScanSharedWorkflows(scenarioPath, scenarioName string) ([]types.ScenarioDependency, error) {
	return d.scenarioScanner.scanWorkflows(scenarioPath, scenarioName)
}
