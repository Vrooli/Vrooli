// Package collectors provides data collection for scenario completeness scoring
// [REQ:SCS-CORE-001] Metric collection interface
package collectors

import (
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

// LoadRequirements loads all requirements with validation data for a scenario
// This is used by the validation quality analyzer
func (c *MetricsCollector) LoadRequirements(scenarioName string) []Requirement {
	scenarioRoot := c.GetScenarioRoot(scenarioName)
	return loadRequirements(scenarioRoot)
}

// PassMetrics holds pass/fail counts
type PassMetrics struct {
	Total   int
	Passing int
}

// ServiceConfig holds minimal service configuration
type ServiceConfig struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Version  string `json:"version"`
}

// Requirement represents a requirement from the requirements JSON
type Requirement struct {
	ID                  string          `json:"id"`
	Title               string          `json:"title"`
	Status              string          `json:"status"`
	Priority            string          `json:"priority,omitempty"`
	Category            string          `json:"category,omitempty"`
	PRDRef              string          `json:"prd_ref,omitempty"`
	Children            []string        `json:"children,omitempty"`
	OperationalTargetID string          `json:"operational_target_id,omitempty"`
	Validation          []ValidationRef `json:"validation,omitempty"`
}

// ValidationRef represents a validation reference
type ValidationRef struct {
	Type   string `json:"type"`
	Ref    string `json:"ref"`
	Phase  string `json:"phase,omitempty"`
	Status string `json:"status,omitempty"`
}

// RequirementsData holds the requirements index/module data
type RequirementsData struct {
	Requirements []Requirement `json:"requirements"`
	Imports      []string      `json:"imports,omitempty"`
}

// TestResults holds test execution results
type TestResults struct {
	Total   int
	Passing int
	Failing int
	LastRun string
}

// SyncMetadata holds requirement sync metadata
type SyncMetadata struct {
	LastSynced   string                 `json:"last_synced"`
	Requirements map[string]SyncedReq   `json:"requirements"`
}

// SyncedReq holds synced requirement status
type SyncedReq struct {
	Status string `json:"status"`
}
