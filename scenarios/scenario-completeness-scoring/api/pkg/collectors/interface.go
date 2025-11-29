// Package collectors provides data collection for scenario completeness scoring.
// It gathers metrics from scenario directories including requirements, tests, and UI analysis.
// [REQ:SCS-CORE-001] Metric collection interface
// [REQ:SCS-CORE-003] Graceful degradation with partial results
// [REQ:SCS-CORE-004] Return partial scores when some collectors fail
package collectors

import (
	"fmt"
	"log"
	"os"

	"scenario-completeness-scoring/pkg/circuitbreaker"
	"scenario-completeness-scoring/pkg/domain"
	apierrors "scenario-completeness-scoring/pkg/errors"
	"scenario-completeness-scoring/pkg/scoring"
)

// Collector gathers metrics from a scenario directory
type Collector interface {
	// Collect gathers all metrics from the scenario
	Collect(scenarioRoot string) (*scoring.Metrics, error)
}

// CollectorName constants for circuit breaker and health tracking
const (
	CollectorRequirements = "requirements"
	CollectorTests        = "tests"
	CollectorUI           = "ui"
	CollectorService      = "service"
)

// MetricsCollector is the main implementation that orchestrates all collectors
type MetricsCollector struct {
	VrooliRoot string
	cbRegistry *circuitbreaker.Registry // Optional circuit breaker integration
}

// NewMetricsCollector creates a new collector with the given Vrooli root
func NewMetricsCollector(vrooliRoot string) *MetricsCollector {
	return &MetricsCollector{
		VrooliRoot: vrooliRoot,
	}
}

// NewMetricsCollectorWithCircuitBreaker creates a collector with circuit breaker support
// [REQ:SCS-CB-001] Circuit breaker integration for collector resilience
func NewMetricsCollectorWithCircuitBreaker(vrooliRoot string, registry *circuitbreaker.Registry) *MetricsCollector {
	return &MetricsCollector{
		VrooliRoot: vrooliRoot,
		cbRegistry: registry,
	}
}

// SetCircuitBreakerRegistry sets the circuit breaker registry for graceful degradation
func (c *MetricsCollector) SetCircuitBreakerRegistry(registry *circuitbreaker.Registry) {
	c.cbRegistry = registry
}

// CollectionResult holds metrics along with partial result information
// [REQ:SCS-CORE-004] Return partial scores when some collectors fail
type CollectionResult struct {
	Metrics       *scoring.Metrics
	PartialResult *apierrors.PartialResult
}

// Collect gathers all metrics for a scenario
// [REQ:SCS-CORE-001] Main collection orchestration
// [REQ:SCS-CORE-003] Graceful degradation - returns partial results when possible
func (c *MetricsCollector) Collect(scenarioName string) (*scoring.Metrics, error) {
	result, err := c.CollectWithPartialResults(scenarioName)
	if err != nil {
		return nil, err
	}
	return result.Metrics, nil
}

// CollectWithPartialResults gathers metrics with detailed partial result info
// [REQ:SCS-CORE-003] Graceful degradation
// [REQ:SCS-CORE-004] Detailed partial results
func (c *MetricsCollector) CollectWithPartialResults(scenarioName string) (*CollectionResult, error) {
	scenarioRoot := c.VrooliRoot + "/scenarios/" + scenarioName
	partial := apierrors.NewPartialResult()

	// Verify scenario exists
	if _, err := os.Stat(scenarioRoot); os.IsNotExist(err) {
		return nil, apierrors.NewScoringError(
			scenarioName,
			fmt.Sprintf("scenario directory not found: %s", scenarioRoot),
			apierrors.CategoryFileSystem,
			err,
		)
	}

	// Initialize metrics with defaults
	metrics := &scoring.Metrics{
		Scenario: scenarioName,
		Category: "utility", // Default category
		Requirements: scoring.MetricCounts{
			Total:   0,
			Passing: 0,
		},
		Targets: scoring.MetricCounts{
			Total:   0,
			Passing: 0,
		},
		Tests: scoring.MetricCounts{
			Total:   0,
			Passing: 0,
		},
	}

	// Collect service config (needed for category)
	if c.shouldCollect(CollectorService) {
		serviceConfig := loadServiceConfig(scenarioRoot)
		if serviceConfig.Category != "" {
			metrics.Category = serviceConfig.Category
		}
		partial.MarkAvailable(CollectorService)
		c.recordSuccess(CollectorService)
	} else {
		partial.MarkFailed(CollectorService, apierrors.NewCollectorWarning(
			CollectorService, "circuit breaker open - using default category",
		))
	}

	// Collect requirements
	if c.shouldCollect(CollectorRequirements) {
		requirements := loadRequirements(scenarioRoot)
		syncData := loadSyncMetadata(scenarioRoot)
		reqPass := calculateRequirementPass(requirements, syncData)
		targetPass := calculateTargetPass(requirements, syncData)
		reqTrees := buildRequirementTrees(requirements)

		metrics.Requirements = scoring.MetricCounts{
			Total:   reqPass.Total,
			Passing: reqPass.Passing,
		}
		metrics.Targets = scoring.MetricCounts{
			Total:   targetPass.Total,
			Passing: targetPass.Passing,
		}
		metrics.Requirements_ = reqTrees

		partial.MarkAvailable(CollectorRequirements)
		c.recordSuccess(CollectorRequirements)
	} else {
		partial.MarkFailed(CollectorRequirements, apierrors.NewCollectorWarning(
			CollectorRequirements, "circuit breaker open - requirements data unavailable",
		))
	}

	// Collect test results
	if c.shouldCollect(CollectorTests) {
		testResults := loadTestResults(scenarioRoot)
		metrics.Tests = scoring.MetricCounts{
			Total:   testResults.Total,
			Passing: testResults.Passing,
		}
		metrics.LastTestRun = testResults.LastRun
		partial.MarkAvailable(CollectorTests)
		c.recordSuccess(CollectorTests)
	} else {
		partial.MarkFailed(CollectorTests, apierrors.NewCollectorWarning(
			CollectorTests, "circuit breaker open - test results unavailable",
		))
	}

	// Collect UI metrics
	if c.shouldCollect(CollectorUI) {
		uiMetrics := collectUIMetrics(scenarioRoot)
		metrics.UI = uiMetrics
		partial.MarkAvailable(CollectorUI)
		c.recordSuccess(CollectorUI)
	} else {
		partial.MarkFailed(CollectorUI, apierrors.NewCollectorWarning(
			CollectorUI, "circuit breaker open - UI metrics unavailable",
		))
	}

	// Log partial results if not complete
	if !partial.IsComplete {
		log.Printf("partial_collection | scenario=%s missing=%v confidence=%.2f",
			scenarioName, partial.Missing, partial.Confidence)
	}

	return &CollectionResult{
		Metrics:       metrics,
		PartialResult: partial,
	}, nil
}

// shouldCollect checks if a collector should run (circuit breaker check)
func (c *MetricsCollector) shouldCollect(collector string) bool {
	if c.cbRegistry == nil {
		return true // No circuit breaker, always collect
	}

	cb := c.cbRegistry.GetIfExists(collector)
	if cb == nil {
		return true // No breaker for this collector
	}

	return cb.Allow()
}

// recordSuccess records a successful collection to the circuit breaker
func (c *MetricsCollector) recordSuccess(collector string) {
	if c.cbRegistry == nil {
		return
	}
	cb := c.cbRegistry.GetIfExists(collector)
	if cb != nil {
		cb.RecordSuccess()
	}
}

// recordFailure records a failed collection to the circuit breaker
func (c *MetricsCollector) recordFailure(collector string) {
	if c.cbRegistry == nil {
		return
	}
	cb := c.cbRegistry.Get(collector) // Get or create
	cb.RecordFailure()
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
