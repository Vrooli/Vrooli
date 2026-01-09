// Package domain provides core domain types for scenario completeness scoring.
// These types represent the fundamental concepts of the scoring domain:
// requirements, validations, and operational targets.
//
// This package exists to provide a single source of truth for domain types,
// avoiding duplicate definitions across collectors, validators, and other packages.
//
// # Domain Vocabulary
//
// The scenario-completeness-scoring domain uses a minimal set of core concepts:
//
// ## Core Entities
//
//   - Scenario: A deployable application being scored (identified by directory name)
//   - Requirement: A unit of work from requirements/*.json linking PRD to tests
//   - OperationalTarget: A high-level PRD capability grouping related requirements
//   - Validation: A reference proving a requirement works (unit/API/E2E tests)
//
// ## Scoring Concepts (defined in scoring package)
//
//   - Score: The completeness score (0-100) with four dimensions
//   - Dimension: Quality (50%), Coverage (15%), Quantity (10%), UI (25%)
//   - Metrics: Raw collected data from a scenario (counts, UI analysis)
//   - MetricCounts: Pass/Total counts for scoring (canonical type in scoring pkg)
//   - Penalty: Points deducted for validation quality issues
//
// ## Configuration Concepts (defined in config package)
//
//   - ScoringConfig: Top-level configuration with components and penalties
//   - ComponentConfig: Enable/disable toggles for each dimension
//   - PenaltyConfig: Enable/disable toggles for each penalty type
//
// ## Health Concepts (defined in health/circuitbreaker packages)
//
//   - CollectorStatus: OK | Degraded | Failed for each data collector
//   - CircuitBreaker: Resilience pattern that disables failing collectors
//
// ## History Concepts (defined in history package)
//
//   - Snapshot: A saved score at a point in time
//   - TrendDirection: Improving | Declining | Stalled | Stable
//
// # Type Ownership
//
// Domain types are organized by responsibility:
//   - domain: Data structures from external sources (requirements, sync metadata)
//   - scoring: Calculation types (Metrics, MetricCounts, ScoreBreakdown)
//   - config: Configuration types (ScoringConfig, presets)
//   - validators: Analysis types (ValidationQualityAnalysis, PenaltyParameters)
package domain

// Requirement represents a requirement from the requirements JSON.
// Requirements are the fundamental unit of work tracking in a scenario,
// linking PRD targets to test validations.
type Requirement struct {
	ID                  string       `json:"id"`
	Title               string       `json:"title,omitempty"`
	Status              string       `json:"status,omitempty"`
	Priority            string       `json:"priority,omitempty"`
	Category            string       `json:"category,omitempty"`
	PRDRef              string       `json:"prd_ref,omitempty"`
	OperationalTargetID string       `json:"operational_target_id,omitempty"`
	Children            []string     `json:"children,omitempty"`
	Validation          []Validation `json:"validation,omitempty"`
}

// Validation represents a validation reference that proves a requirement works.
// Validations can be automated (unit, API, E2E tests) or manual.
type Validation struct {
	Type       string `json:"type"`
	Ref        string `json:"ref,omitempty"`
	WorkflowID string `json:"workflow_id,omitempty"`
	Phase      string `json:"phase,omitempty"`
	Status     string `json:"status,omitempty"`
}

// OperationalTarget represents a high-level business capability from the PRD.
// Operational targets group related requirements under cohesive objectives.
type OperationalTarget struct {
	ID           string   `json:"id"`
	Title        string   `json:"title,omitempty"`
	Priority     string   `json:"priority,omitempty"`
	Requirements []string `json:"requirements,omitempty"`
}

// RequirementsIndex holds the requirements index/module data loaded from JSON.
type RequirementsIndex struct {
	Requirements []Requirement `json:"requirements"`
	Imports      []string      `json:"imports,omitempty"`
}

// SyncMetadata holds requirement sync metadata from the sync process.
type SyncMetadata struct {
	LastSynced         string                    `json:"last_synced"`
	Requirements       map[string]SyncedReq      `json:"requirements"`
	OperationalTargets []SyncedOperationalTarget `json:"operational_targets,omitempty"`
}

// SyncedReq holds synced requirement status.
type SyncedReq struct {
	Status string `json:"status"`
}

// SyncedOperationalTarget holds operational target from sync metadata.
type SyncedOperationalTarget struct {
	Key         string        `json:"key"`
	TargetID    string        `json:"target_id"`
	FolderHint  string        `json:"folder_hint,omitempty"`
	Status      string        `json:"status"`
	Criticality string        `json:"criticality,omitempty"`
	Counts      *TargetCounts `json:"counts,omitempty"`
}

// TargetCounts holds counts for an operational target.
type TargetCounts struct {
	Total      int `json:"total"`
	Complete   int `json:"complete"`
	InProgress int `json:"in_progress"`
	Pending    int `json:"pending"`
}

// NOTE: MetricCounts is defined in scoring/types.go as the canonical type.
// The scoring package owns this type since it represents passing/total counts
// used exclusively for score calculation. See scoring.MetricCounts.

// ServiceConfig holds minimal service configuration from .vrooli/service.json.
type ServiceConfig struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Version  string `json:"version"`
}

// TestResults holds test execution results from coverage data.
// This is used internally by collectors to load test data from files.
// For scoring, only Total and Passing are used (converted to scoring.MetricCounts).
// NOTE: Failing is intentionally omitted - it can be derived as (Total - Passing)
// and the scoring engine only needs pass/total counts.
type TestResults struct {
	Total   int    // Total number of tests executed
	Passing int    // Number of tests that passed
	LastRun string // RFC3339 timestamp of when tests last ran
}
