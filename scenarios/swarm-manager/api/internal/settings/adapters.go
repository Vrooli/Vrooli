package settings

import (
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

func (a *agentAdapter) LoadAgentSettings() (maxTurns, timeoutSeconds int32, requiresApproval bool, err error) {
	s, err := a.store.Load()
	if err != nil {
		return 0, 0, false, err
	}
	return int32(s.AgentMaxTurns), int32(s.AgentTimeoutSeconds), s.AgentRequiresApproval, nil
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
		DefaultMode:      execution.Mode(s.DefaultMode),
		AutoFixup:        s.AutoFixup,
		MaxFixupAttempts: s.MaxFixupAttempts,
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
	return execution.GovernanceSettings{
		MaxConcurrentExecutions:       s.MaxConcurrentExecutions,
		MaxQueueDepth:                 s.MaxQueueDepth,
		CircuitBreakerThreshold:       s.CircuitBreakerThreshold,
		CircuitBreakerCooldownMinutes: s.CircuitBreakerCooldownMinutes,
		ExecutionCostCapPerRun:        s.ExecutionCostCapPerRun,
		CostPerTurnEstimate:           s.CostPerTurnEstimate,
		AgentMaxTurns:                 s.AgentMaxTurns,
	}, nil
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
