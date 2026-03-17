// Package config — orchestration settings define the control surface for
// run execution, safety isolation, health detection, and process termination.
// All numeric fields use int (not time.Duration) for clean JSON round-tripping.

package config

import (
	"fmt"

	"agent-manager/internal/domain"
)

// =============================================================================
// ORCHESTRATION SETTINGS
// =============================================================================

// OrchestrationSettings groups all tunable parameters that govern how runs
// are executed, isolated, monitored, and terminated.
type OrchestrationSettings struct {
	RunExecution       RunExecutionSettings       `json:"runExecution"`
	SafetyIsolation    SafetyIsolationSettings    `json:"safetyIsolation"`
	HealthDetection    HealthDetectionSettings    `json:"healthDetection"`
	ProcessTermination ProcessTerminationSettings `json:"processTermination"`
}

// RunExecutionSettings control how agent runs execute.
type RunExecutionSettings struct {
	// RunTimeoutMinutes is the maximum execution time for a single run.
	// Higher = more time for complex tasks, but longer resource usage.
	// Range: 1–9999. Default: 30.
	RunTimeoutMinutes int `json:"runTimeoutMinutes"`

	// MaxConcurrentRuns limits total simultaneous runs.
	// Higher = more parallelism, but more resource usage.
	// Range: 1–9999. Default: 10.
	MaxConcurrentRuns int `json:"maxConcurrentRuns"`

	// MaxTurns limits conversation turns per run.
	// Higher = more agent autonomy, but potential for runaway loops.
	// Range: 1–9999. Default: 100.
	MaxTurns int `json:"maxTurns"`
}

// SafetyIsolationSettings control accident prevention and isolation.
type SafetyIsolationSettings struct {
	// RequireSandbox makes all runs use sandbox isolation.
	// true = safer but slower due to sandbox overhead. Default: true.
	RequireSandbox bool `json:"requireSandbox"`

	// RequireApproval makes all runs need review before applying changes.
	// false = auto-apply (use with caution). Default: true.
	RequireApproval bool `json:"requireApproval"`

	// NetworkAccess controls network access for sandboxed runs.
	// "none" = no network, "localhost" = local only, "full" = unrestricted.
	// Default: "localhost".
	NetworkAccess string `json:"networkAccess"`
}

// HealthDetectionSettings control how the system monitors run health.
type HealthDetectionSettings struct {
	// HeartbeatIntervalSeconds is how often runs send heartbeat signals.
	// Lower = faster detection of stale runs, but more overhead.
	// Range: 1–9999. Default: 15.
	HeartbeatIntervalSeconds int `json:"heartbeatIntervalSeconds"`

	// StaleThresholdSeconds is how long without a heartbeat before a run is stale.
	// Must be greater than HeartbeatIntervalSeconds.
	// Range: 10–9999. Default: 300.
	StaleThresholdSeconds int `json:"staleThresholdSeconds"`

	// MaxRecoveryAgeSeconds is the maximum age of a stale run eligible for recovery.
	// Must be greater than StaleThresholdSeconds.
	// Range: 30–9999. Default: 600.
	MaxRecoveryAgeSeconds int `json:"maxRecoveryAgeSeconds"`

	// ReconcilerIntervalSeconds is how often the reconciler checks for stale runs.
	// Range: 5–9999. Default: 30.
	ReconcilerIntervalSeconds int `json:"reconcilerIntervalSeconds"`
}

// ProcessTerminationSettings control how processes are shut down.
type ProcessTerminationSettings struct {
	// GracePeriodSeconds is how long to wait after SIGTERM before SIGKILL.
	// Range: 1–9999. Default: 5.
	GracePeriodSeconds int `json:"gracePeriodSeconds"`

	// KillProcessGroup sends signals to the entire process group.
	// Default: true.
	KillProcessGroup bool `json:"killProcessGroup"`

	// KillOrphans terminates orphaned child processes after run completion.
	// Default: true.
	KillOrphans bool `json:"killOrphans"`

	// OrphanGracePeriodSeconds is how long orphan processes may live before kill.
	// Range: 30–9999. Default: 600.
	OrphanGracePeriodSeconds int `json:"orphanGracePeriodSeconds"`

	// TerminationMaxRetries is how many times to retry termination.
	// Range: 1–99. Default: 3.
	TerminationMaxRetries int `json:"terminationMaxRetries"`
}

// =============================================================================
// DEFAULTS
// =============================================================================

// DefaultOrchestrationSettings returns production-ready defaults.
func DefaultOrchestrationSettings() OrchestrationSettings {
	return OrchestrationSettings{
		RunExecution: RunExecutionSettings{
			RunTimeoutMinutes: 30,
			MaxConcurrentRuns: 10,
			MaxTurns:          100,
		},
		SafetyIsolation: SafetyIsolationSettings{
			RequireSandbox:  true,
			RequireApproval: true,
			NetworkAccess:   "localhost",
		},
		HealthDetection: HealthDetectionSettings{
			HeartbeatIntervalSeconds:  15,
			StaleThresholdSeconds:     300,
			MaxRecoveryAgeSeconds:     600,
			ReconcilerIntervalSeconds: 30,
		},
		ProcessTermination: ProcessTerminationSettings{
			GracePeriodSeconds:       5,
			KillProcessGroup:         true,
			KillOrphans:              true,
			OrphanGracePeriodSeconds: 600,
			TerminationMaxRetries:    3,
		},
	}
}

// =============================================================================
// VALIDATION
// =============================================================================

// validNetworkAccess is the set of allowed NetworkAccess values.
var validNetworkAccess = map[string]bool{
	"none":      true,
	"localhost": true,
	"full":      true,
}

// Validate checks all settings are within safe bounds.
func (s *OrchestrationSettings) Validate() error {
	if err := s.RunExecution.validate(); err != nil {
		return wrapConfigSection("runExecution", err)
	}
	if err := s.SafetyIsolation.validate(); err != nil {
		return wrapConfigSection("safetyIsolation", err)
	}
	if err := s.HealthDetection.validate(); err != nil {
		return wrapConfigSection("healthDetection", err)
	}
	if err := s.ProcessTermination.validate(); err != nil {
		return wrapConfigSection("processTermination", err)
	}

	// Cross-field: heartbeat must fire before stale detection kicks in.
	if s.HealthDetection.HeartbeatIntervalSeconds >= s.HealthDetection.StaleThresholdSeconds {
		return wrapConfigSection("healthDetection", domain.NewConfigInvalidError(
			"heartbeatIntervalSeconds",
			fmt.Sprintf("must be less than staleThresholdSeconds (%d), got %d",
				s.HealthDetection.StaleThresholdSeconds, s.HealthDetection.HeartbeatIntervalSeconds),
			nil,
		))
	}

	// Cross-field: stale threshold must be less than recovery age.
	if s.HealthDetection.StaleThresholdSeconds >= s.HealthDetection.MaxRecoveryAgeSeconds {
		return wrapConfigSection("healthDetection", domain.NewConfigInvalidError(
			"staleThresholdSeconds",
			fmt.Sprintf("must be less than maxRecoveryAgeSeconds (%d), got %d",
				s.HealthDetection.MaxRecoveryAgeSeconds, s.HealthDetection.StaleThresholdSeconds),
			nil,
		))
	}

	// Cross-field: total termination time must fit within stale detection window.
	totalTermination := s.ProcessTermination.GracePeriodSeconds * s.ProcessTermination.TerminationMaxRetries
	if totalTermination >= s.HealthDetection.StaleThresholdSeconds {
		return wrapConfigSection("processTermination", domain.NewConfigInvalidError(
			"gracePeriodSeconds",
			fmt.Sprintf("gracePeriodSeconds (%d) * terminationMaxRetries (%d) = %d must be less than staleThresholdSeconds (%d)",
				s.ProcessTermination.GracePeriodSeconds, s.ProcessTermination.TerminationMaxRetries,
				totalTermination, s.HealthDetection.StaleThresholdSeconds),
			nil,
		))
	}

	return nil
}

func (r *RunExecutionSettings) validate() error {
	if r.RunTimeoutMinutes < 1 || r.RunTimeoutMinutes > 9999 {
		return domain.NewConfigInvalidError("runTimeoutMinutes", fmt.Sprintf("must be between 1 and 9999, got %d", r.RunTimeoutMinutes), nil)
	}
	if r.MaxConcurrentRuns < 1 || r.MaxConcurrentRuns > 9999 {
		return domain.NewConfigInvalidError("maxConcurrentRuns", fmt.Sprintf("must be between 1 and 9999, got %d", r.MaxConcurrentRuns), nil)
	}
	if r.MaxTurns < 1 || r.MaxTurns > 9999 {
		return domain.NewConfigInvalidError("maxTurns", fmt.Sprintf("must be between 1 and 9999, got %d", r.MaxTurns), nil)
	}
	return nil
}

func (s *SafetyIsolationSettings) validate() error {
	if !validNetworkAccess[s.NetworkAccess] {
		return domain.NewConfigInvalidError("networkAccess", fmt.Sprintf("must be one of none, localhost, full; got %q", s.NetworkAccess), nil)
	}
	return nil
}

func (h *HealthDetectionSettings) validate() error {
	if h.HeartbeatIntervalSeconds < 1 || h.HeartbeatIntervalSeconds > 9999 {
		return domain.NewConfigInvalidError("heartbeatIntervalSeconds", fmt.Sprintf("must be between 1 and 9999, got %d", h.HeartbeatIntervalSeconds), nil)
	}
	if h.StaleThresholdSeconds < 10 || h.StaleThresholdSeconds > 9999 {
		return domain.NewConfigInvalidError("staleThresholdSeconds", fmt.Sprintf("must be between 10 and 9999, got %d", h.StaleThresholdSeconds), nil)
	}
	if h.MaxRecoveryAgeSeconds < 30 || h.MaxRecoveryAgeSeconds > 9999 {
		return domain.NewConfigInvalidError("maxRecoveryAgeSeconds", fmt.Sprintf("must be between 30 and 9999, got %d", h.MaxRecoveryAgeSeconds), nil)
	}
	if h.ReconcilerIntervalSeconds < 5 || h.ReconcilerIntervalSeconds > 9999 {
		return domain.NewConfigInvalidError("reconcilerIntervalSeconds", fmt.Sprintf("must be between 5 and 9999, got %d", h.ReconcilerIntervalSeconds), nil)
	}
	return nil
}

func (p *ProcessTerminationSettings) validate() error {
	if p.GracePeriodSeconds < 1 || p.GracePeriodSeconds > 9999 {
		return domain.NewConfigInvalidError("gracePeriodSeconds", fmt.Sprintf("must be between 1 and 9999, got %d", p.GracePeriodSeconds), nil)
	}
	if p.OrphanGracePeriodSeconds < 30 || p.OrphanGracePeriodSeconds > 9999 {
		return domain.NewConfigInvalidError("orphanGracePeriodSeconds", fmt.Sprintf("must be between 30 and 9999, got %d", p.OrphanGracePeriodSeconds), nil)
	}
	if p.TerminationMaxRetries < 1 || p.TerminationMaxRetries > 99 {
		return domain.NewConfigInvalidError("terminationMaxRetries", fmt.Sprintf("must be between 1 and 99, got %d", p.TerminationMaxRetries), nil)
	}
	return nil
}
