package settings

import (
	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/execution"
)

// policyAdapter bridges Store to execution.PolicyProvider.
type policyAdapter struct {
	store *Store
}

// NewPolicyAdapter creates a PolicyProvider backed by the given Store.
func NewPolicyAdapter(store *Store) *policyAdapter {
	return &policyAdapter{store: store}
}

// LoadPolicy derives the legacy execution.Policy view from the canonical
// PolicyControls projection so the two seams can never disagree.
func (a *policyAdapter) LoadPolicy() (execution.Policy, error) {
	s, err := a.store.Load()
	if err != nil {
		return execution.Policy{}, err
	}
	controls := ProjectPolicyControls(s)
	return execution.Policy{
		DefaultMode:        execution.Mode(controls.Execution.DefaultMode),
		AutoFixup:          controls.Retry.AutoFixup,
		MaxFixupAttempts:   controls.Retry.MaxFixupAttempts,
		ReviewAgentEnabled: controls.Review.AgentEnabled,
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

// LoadAutoFiler returns the normalized auto-filer policy block. The auto-filer
// re-reads this on every cycle so operators can enable, brake, or retarget it
// without restarting swarm-manager.
func (a *governanceAdapter) LoadAutoFiler() (AutoFilerSettings, error) {
	s, err := a.store.Load()
	if err != nil {
		return AutoFilerSettings{}, err
	}
	return s.AutoFiler, nil
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
		// Turn budget comes from the PolicyControls projection (same
		// persisted field) so the governance cost estimate and the policy
		// surface always agree.
		AgentMaxTurns:     ProjectPolicyControls(s).Budgets.MaxTurns,
		FixBeforeFeature:  s.FixBeforeFeature,
		AutoFilerEnabled:  s.AutoFiler.Enabled,
		AutoFilerStrategy: s.AutoFiler.Strategy,
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
	// Derived from the canonical PolicyControls projection so the review
	// thresholds handed to the reviewer always match the policy-controls
	// surface.
	controls := ProjectPolicyControls(s)
	return &execution.ReviewThresholds{
		CodeQualityMinScore:   controls.Review.CodeQualityMinScore,
		TestMinPassRate:       controls.Review.TestMinPassRate,
		MaxBlockingViolations: controls.Review.MaxBlockingViolations,
		MaxWarnings:           controls.Review.MaxWarnings,
		RequireScreenshots:    controls.Review.RequireScreenshots,
		RequireTests:          controls.Review.RequireTests,
	}, nil
}
