// Package main provides analyzer type definitions for the secrets-manager API.
//
// These types represent the response structures received from the
// scenario-dependency-analyzer service, which provides dependency analysis
// and fitness scoring for deployment scenarios across different tiers.
package main

import "time"

// analyzerDeploymentReport represents the complete deployment analysis
// report returned by the scenario-dependency-analyzer service.
type analyzerDeploymentReport struct {
	// Scenario is the name of the analyzed scenario.
	Scenario string `json:"scenario"`

	// ReportVersion indicates the schema version of this report.
	ReportVersion int `json:"report_version"`

	// GeneratedAt is when the analyzer generated this report.
	GeneratedAt time.Time `json:"generated_at"`

	// Dependencies lists all identified dependencies (resources, scenarios, etc.).
	Dependencies []analyzerDependency `json:"dependencies"`

	// Aggregates contains per-tier fitness summaries.
	Aggregates map[string]analyzerAggregate `json:"aggregates"`
}

// analyzerDependency represents a single dependency identified by the analyzer.
type analyzerDependency struct {
	// Name is the dependency identifier (e.g., "postgres", "redis").
	Name string `json:"name"`

	// Type categorizes the dependency (e.g., "resource", "scenario").
	Type string `json:"type"`

	// ResourceType further classifies resources (e.g., "database", "cache").
	ResourceType string `json:"resource_type"`

	// Requirements specifies the resource demands of this dependency.
	Requirements analyzerRequirement `json:"requirements"`

	// TierSupport maps deployment tiers to their support status.
	TierSupport map[string]analyzerTierSupport `json:"tier_support"`

	// Alternatives lists potential substitutes for this dependency.
	Alternatives []string `json:"alternatives"`

	// Source indicates where this dependency was discovered (e.g., "service.json").
	Source string `json:"source"`
}

// analyzerRequirement specifies the resource demands of a dependency.
type analyzerRequirement struct {
	// RAMMB is the estimated RAM requirement in megabytes.
	RAMMB int `json:"ram_mb"`

	// DiskMB is the estimated disk space requirement in megabytes.
	DiskMB int `json:"disk_mb"`

	// CPUCores is the estimated CPU core requirement.
	CPUCores int `json:"cpu_cores"`

	// Network describes network requirements (e.g., "internet", "local").
	Network string `json:"network"`

	// Source indicates how these requirements were determined.
	Source string `json:"source"`

	// Confidence indicates reliability of the requirement estimates.
	Confidence string `json:"confidence"`
}

// analyzerTierSupport describes whether a dependency is supported for a tier.
type analyzerTierSupport struct {
	// Supported indicates if the dependency works on this tier.
	Supported bool `json:"supported"`

	// FitnessScore is the compatibility score (0.0-1.0) for this tier.
	FitnessScore float64 `json:"fitness_score"`

	// Notes provides additional context about tier compatibility.
	Notes string `json:"notes"`

	// Reason explains why the dependency is or isn't supported.
	Reason string `json:"reason"`

	// Alternatives lists recommended substitutes for unsupported tiers.
	Alternatives []string `json:"alternatives"`
}

// analyzerAggregate contains aggregate fitness metrics for a deployment tier.
type analyzerAggregate struct {
	// FitnessScore is the overall tier compatibility score (0.0-1.0).
	FitnessScore float64 `json:"fitness_score"`

	// DependencyCount is the total number of dependencies for this tier.
	DependencyCount int `json:"dependency_count"`

	// BlockingDependencies lists dependencies that prevent deployment.
	BlockingDependencies []string `json:"blocking_dependencies"`

	// EstimatedRequirements aggregates resource demands across all dependencies.
	EstimatedRequirements analyzerRequirementTotals `json:"estimated_requirements"`
}

// analyzerRequirementTotals aggregates resource requirements across dependencies.
type analyzerRequirementTotals struct {
	// RAMMB is the total estimated RAM requirement in megabytes.
	RAMMB int `json:"ram_mb"`

	// DiskMB is the total estimated disk space requirement in megabytes.
	DiskMB int `json:"disk_mb"`

	// CPUCores is the total estimated CPU core requirement.
	CPUCores int `json:"cpu_cores"`
}
