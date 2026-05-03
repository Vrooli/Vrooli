package settings

import (
	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/execution"
)

// agentAdapter bridges Store to agentmanager.SettingsReader.
type agentAdapter struct {
	store *Store
}

// NewAgentAdapter creates a SettingsReader backed by the given Store.
func NewAgentAdapter(store *Store) *agentAdapter {
	return &agentAdapter{store: store}
}

func (a *agentAdapter) LoadAgentSettings() (maxTurns, timeoutSeconds int32, err error) {
	s, err := a.store.Load()
	if err != nil {
		return 0, 0, err
	}
	return int32(s.AgentMaxTurns), int32(s.AgentTimeoutSeconds), nil
}

// policyAdapter bridges Store to execution.PolicyProvider.
type policyAdapter struct {
	store *Store
}

// NewPolicyAdapter creates a PolicyProvider backed by the given Store.
func NewPolicyAdapter(store *Store) *policyAdapter {
	return &policyAdapter{store: store}
}

func (a *policyAdapter) LoadPolicy() (execution.Policy, error) {
	s, err := a.store.Load()
	if err != nil {
		return execution.Policy{}, err
	}
	return execution.Policy{
		DefaultMode:        execution.Mode(s.DefaultMode),
		AutoFixup:          s.AutoFixup,
		MaxFixupAttempts:   s.MaxFixupAttempts,
		ReviewAgentEnabled: s.ReviewAgentEnabled,
	}, nil
}

// governanceAdapter bridges Store to execution.GovernanceProvider.
type governanceAdapter struct {
	store *Store
}

// NewGovernanceAdapter creates a GovernanceProvider backed by the given Store.
func NewGovernanceAdapter(store *Store) *governanceAdapter {
	return &governanceAdapter{store: store}
}

func (a *governanceAdapter) LoadGovernance() (execution.GovernanceSettings, error) {
	s, err := a.store.Load()
	if err != nil {
		return execution.GovernanceSettings{}, err
	}
	// Copy the lane map so callers cannot mutate the settings store's
	// in-memory snapshot.
	laneLimits := make(map[string]int, len(s.LaneConcurrencyLimits))
	for k, v := range s.LaneConcurrencyLimits {
		laneLimits[k] = v
	}
	return execution.GovernanceSettings{
		LaneLimits:                    laneLimits,
		MaxQueueDepth:                 s.MaxQueueDepth,
		CircuitBreakerThreshold:       s.CircuitBreakerThreshold,
		CircuitBreakerCooldownMinutes: s.CircuitBreakerCooldownMinutes,
		ExecutionCostCapPerRun:        s.ExecutionCostCapPerRun,
		CostPerTurnEstimate:           s.CostPerTurnEstimate,
		AgentMaxTurns:                 s.AgentMaxTurns,
	}, nil
}

// lanePolicyAdapter bridges Store to agentactivity.LanePolicy. Lookups
// load the latest settings on each call so a Settings update is picked up
// without restarting the service. Failures fall back to the lane defaults
// in DefaultSettings — saturation is preferable to spawn errors during a
// transient store outage.
type lanePolicyAdapter struct {
	store *Store
}

// NewLanePolicyAdapter creates a LanePolicy backed by the given Store.
func NewLanePolicyAdapter(store *Store) *lanePolicyAdapter {
	return &lanePolicyAdapter{store: store}
}

func (a *lanePolicyAdapter) LimitFor(lane agentactivity.Lane) int {
	s, err := a.store.Load()
	if err != nil {
		// Defaults rather than zero so a load error does not silently
		// uncap every lane. defaultLaneConcurrencyLimits is the same map
		// DefaultSettings would have produced.
		return defaultLaneConcurrencyLimits()[string(lane)]
	}
	if val, ok := s.LaneConcurrencyLimits[string(lane)]; ok && val > 0 {
		return val
	}
	return defaultLaneConcurrencyLimits()[string(lane)]
}

// reviewThresholdsAdapter bridges Store to execution.ReviewThresholdsProvider.
type reviewThresholdsAdapter struct {
	store *Store
}

// NewReviewThresholdsAdapter creates a ReviewThresholdsProvider backed by the given Store.
func NewReviewThresholdsAdapter(store *Store) *reviewThresholdsAdapter {
	return &reviewThresholdsAdapter{store: store}
}

func (a *reviewThresholdsAdapter) LoadReviewThresholds() (*execution.ReviewThresholds, error) {
	s, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	return &execution.ReviewThresholds{
		CodeQualityMinScore:   s.ReviewCodeQualityMinScore,
		TestMinPassRate:       s.ReviewTestMinPassRate,
		MaxBlockingViolations: s.ReviewMaxBlockingViolations,
		MaxWarnings:           s.ReviewMaxWarnings,
		RequireScreenshots:    s.ReviewRequireScreenshots,
		RequireTests:          s.ReviewRequireTests,
	}, nil
}
