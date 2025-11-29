// Package domain provides core domain types for scenario completeness scoring.
// These types represent the fundamental concepts of the scoring domain:
// requirements, validations, and operational targets.
//
// This package exists to provide a single source of truth for domain types,
// avoiding duplicate definitions across collectors, validators, and other packages.
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

// MetricCounts represents basic metric counts for scoring and validation.
type MetricCounts struct {
	Total   int `json:"total"`
	Passing int `json:"passing"`
}

// ServiceConfig holds minimal service configuration from .vrooli/service.json.
type ServiceConfig struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Version  string `json:"version"`
}

// TestResults holds test execution results from coverage data.
type TestResults struct {
	Total   int
	Passing int
	Failing int
	LastRun string
}
