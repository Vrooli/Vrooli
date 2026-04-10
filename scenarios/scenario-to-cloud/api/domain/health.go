// Package domain defines the core domain types for the scenario-to-cloud scenario.
package domain

// HealthLevel represents the overall health of a deployment.
type HealthLevel string

const (
	HealthHealthy   HealthLevel = "healthy"
	HealthDegraded  HealthLevel = "degraded"
	HealthUnhealthy HealthLevel = "unhealthy"
	HealthStopped   HealthLevel = "stopped"
	HealthFailed    HealthLevel = "failed"
	HealthPending   HealthLevel = "pending"
	HealthStarting  HealthLevel = "starting"
	HealthUnknown   HealthLevel = "unknown"
)

// HealthCheckStatus represents the result of a single health check.
type HealthCheckStatus string

const (
	HealthCheckPass  HealthCheckStatus = "pass"
	HealthCheckWarn  HealthCheckStatus = "warn"
	HealthCheckFail  HealthCheckStatus = "fail"
	HealthCheckSkip  HealthCheckStatus = "skip"
	HealthCheckError HealthCheckStatus = "error"
)

// HealthCheck represents a single check within a health section.
type HealthCheck struct {
	ID      string            `json:"id"`
	Title   string            `json:"title"`
	Status  HealthCheckStatus `json:"status"`
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// HealthSection groups related health checks by category.
type HealthSection struct {
	Category   string            `json:"category"`
	Title      string            `json:"title"`
	Status     HealthCheckStatus `json:"status"`
	Checks     []HealthCheck     `json:"checks"`
	PassCount  int               `json:"pass_count"`
	WarnCount  int               `json:"warn_count"`
	FailCount  int               `json:"fail_count"`
	ErrorCount int               `json:"error_count"`
}

// Recommendation is an actionable suggestion to fix a health issue.
type Recommendation struct {
	Priority int    `json:"priority"` // 1=critical, 2=important, 3=suggestion
	Category string `json:"category"`
	Summary  string `json:"summary"`
	Command  string `json:"command,omitempty"`
}

// FreshnessState represents whether deployed code matches local scenario state.
type FreshnessState string

const (
	FreshnessCurrent  FreshnessState = "current"
	FreshnessOutdated FreshnessState = "outdated"
	FreshnessUnknown  FreshnessState = "unknown"
)

// FreshnessStatus summarizes version/fingerprint parity between local and deployed state.
type FreshnessStatus struct {
	Status               FreshnessState `json:"status"`
	Summary              string         `json:"summary"`
	VersionStatus        FreshnessState `json:"version_status"`
	FingerprintStatus    FreshnessState `json:"fingerprint_status"`
	LocalVersion         string         `json:"local_version,omitempty"`
	DeployedVersion      string         `json:"deployed_version,omitempty"`
	VersionSource        string         `json:"version_source,omitempty"`
	LocalBundleSHA256    string         `json:"local_bundle_sha256,omitempty"`
	DeployedBundleSHA256 string         `json:"deployed_bundle_sha256,omitempty"`
	Notes                []string       `json:"notes,omitempty"`
}

// HealthResponse is the top-level response from the health endpoint.
type HealthResponse struct {
	OK              bool             `json:"ok"`
	Health          HealthLevel      `json:"health"`
	DeploymentID    string           `json:"deployment_id"`
	DeploymentName  string           `json:"deployment_name"`
	ScenarioID      string           `json:"scenario_id"`
	Domain          string           `json:"domain,omitempty"`
	Host            string           `json:"host,omitempty"`
	Summary         string           `json:"summary"`
	Sections        []HealthSection  `json:"sections"`
	Freshness       *FreshnessStatus `json:"freshness,omitempty"`
	Recommendations []Recommendation `json:"recommendations,omitempty"`
	DurationMs      int64            `json:"duration_ms"`
	Timestamp       string           `json:"timestamp"`
}
