package execution

import (
	"context"
)

// countActiveExecutions counts records that are starting or running (under the mutex).
func countActiveExecutions(records []Record) int {
	count := 0
	for _, r := range records {
		if r.Status == StatusStarting || r.Status == StatusRunning {
			count++
		}
	}
	return count
}

// countQueuedExecutions counts records that are pending or scheduled.
func countQueuedExecutions(records []Record) int {
	count := 0
	for _, r := range records {
		if r.Status == StatusPending {
			count++
		}
	}
	return count
}

// ResetCircuitBreaker clears the circuit breaker for a specific item.
func (s *Service) ResetCircuitBreaker(itemKey string) error {
	return s.circuitBreaker.Reset(itemKey)
}

// GovernanceStatus returns the current governance state for the overview endpoint.
func (s *Service) GovernanceStatus() (*GovernanceStatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	gov, err := s.governanceProvider.LoadGovernance()
	if err != nil {
		return nil, err
	}

	records, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	brokenItems, _ := s.circuitBreaker.BrokenItems(gov.CircuitBreakerCooldownMinutes)
	if brokenItems == nil {
		brokenItems = []string{}
	}

	active := countActiveExecutions(records)
	queued := countQueuedExecutions(records)

	agentMaxTurns := gov.AgentMaxTurns
	if agentMaxTurns <= 0 {
		agentMaxTurns = 600
	}
	estimatedQueuedCost := float64(queued) * gov.CostPerTurnEstimate * float64(agentMaxTurns)

	return &GovernanceStatusResponse{
		ActiveExecutions:    active,
		MaxConcurrent:       gov.MaxConcurrentExecutions,
		QueueDepth:          queued,
		MaxQueueDepth:       gov.MaxQueueDepth,
		CircuitBrokenItems:  brokenItems,
		EstimatedQueuedCost: estimatedQueuedCost,
	}, nil
}

// Policy returns current execution policy from the unified settings store.
func (s *Service) Policy(_ context.Context) (Policy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.policyProvider.LoadPolicy()
}
