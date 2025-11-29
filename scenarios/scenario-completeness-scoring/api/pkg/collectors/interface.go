// Package collectors provides data collection for scenario completeness scoring.
// It gathers metrics from scenario directories including requirements, tests, and UI analysis.
// [REQ:SCS-CORE-001] Metric collection interface
package collectors

import (
	"scenario-completeness-scoring/pkg/domain"
	"scenario-completeness-scoring/pkg/scoring"
)

// Collector gathers metrics from a scenario directory
type Collector interface {
	// Collect gathers all metrics from the scenario
	Collect(scenarioRoot string) (*scoring.Metrics, error)
}

// MetricsCollector is the main implementation that orchestrates all collectors
type MetricsCollector struct {
	VrooliRoot string
}

// NewMetricsCollector creates a new collector with the given Vrooli root
func NewMetricsCollector(vrooliRoot string) *MetricsCollector {
	return &MetricsCollector{
		VrooliRoot: vrooliRoot,
	}
}

// Collect gathers all metrics for a scenario
// [REQ:SCS-CORE-001] Main collection orchestration
func (c *MetricsCollector) Collect(scenarioName string) (*scoring.Metrics, error) {
	scenarioRoot := c.VrooliRoot + "/scenarios/" + scenarioName

	// Load service config for category
	serviceConfig := loadServiceConfig(scenarioRoot)

	// Load and analyze requirements
	requirements := loadRequirements(scenarioRoot)
	syncData := loadSyncMetadata(scenarioRoot)
	reqPass := calculateRequirementPass(requirements, syncData)
	targetPass := calculateTargetPass(requirements, syncData)

	// Load test results
	testResults := loadTestResults(scenarioRoot)

	// Collect UI metrics
	uiMetrics := collectUIMetrics(scenarioRoot)

	// Build requirement trees for depth calculation
	reqTrees := buildRequirementTrees(requirements)

	return &scoring.Metrics{
		Scenario: scenarioName,
		Category: serviceConfig.Category,
		Requirements: scoring.MetricCounts{
			Total:   reqPass.Total,
			Passing: reqPass.Passing,
		},
		Targets: scoring.MetricCounts{
			Total:   targetPass.Total,
			Passing: targetPass.Passing,
		},
		Tests: scoring.MetricCounts{
			Total:   testResults.Total,
			Passing: testResults.Passing,
		},
		UI:            uiMetrics,
		LastTestRun:   testResults.LastRun,
		Requirements_: reqTrees,
	}, nil
}

// GetScenarioRoot returns the full path to a scenario directory
func (c *MetricsCollector) GetScenarioRoot(scenarioName string) string {
	return c.VrooliRoot + "/scenarios/" + scenarioName
}

// LoadRequirements loads all requirements with validation data for a scenario.
// This is used by the validation quality analyzer.
func (c *MetricsCollector) LoadRequirements(scenarioName string) []domain.Requirement {
	scenarioRoot := c.GetScenarioRoot(scenarioName)
	return loadRequirements(scenarioRoot)
}

// PassMetrics holds pass/fail counts for scoring calculations
type PassMetrics struct {
	Total   int
	Passing int
}

// Type aliases for backward compatibility - these delegate to domain types
// to maintain a clean transition while avoiding breaking changes.
type (
	// Requirement is an alias to the domain.Requirement type
	Requirement = domain.Requirement
	// Validation is an alias to the domain.Validation type
	Validation = domain.Validation
	// RequirementsIndex is an alias to the domain.RequirementsIndex type
	RequirementsIndex = domain.RequirementsIndex
	// SyncMetadata is an alias to the domain.SyncMetadata type
	SyncMetadata = domain.SyncMetadata
	// SyncedReq is an alias to the domain.SyncedReq type
	SyncedReq = domain.SyncedReq
	// SyncedOperationalTarget is an alias to the domain.SyncedOperationalTarget type
	SyncedOperationalTarget = domain.SyncedOperationalTarget
	// TargetCounts is an alias to the domain.TargetCounts type
	TargetCounts = domain.TargetCounts
	// ServiceConfig is an alias to the domain.ServiceConfig type
	ServiceConfig = domain.ServiceConfig
	// TestResults is an alias to the domain.TestResults type
	TestResults = domain.TestResults
)

// RequirementsData is kept for JSON unmarshaling compatibility
// It maps directly to domain.RequirementsIndex
type RequirementsData = domain.RequirementsIndex
