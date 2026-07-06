package execution

import (
	"context"

	"swarm-manager/internal/agentactivity"
)

// countActiveExecutions counts records that are starting or running (under
// the mutex). Execution.Records only ever represent backlog item processing,
// which lives in the Execute lane — see countActiveByLane for the multi-
// lane equivalent backed by agentactivity records.
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

// laneCapacity returns the Execute-lane cap from governance settings,
// defaulting to the value in DefaultGovernanceSettings if no entry exists.
// Backlog item processing is the sole consumer of execution.Records, and
// every such run lives in the Execute lane — so a single lookup is enough
// without the (purpose, phaseKind) → lane derivation done in agentactivity.
func laneCapacity(gov GovernanceSettings, lane agentactivity.Lane) int {
	if val, ok := gov.LaneLimits[string(lane)]; ok && val > 0 {
		return val
	}
	return DefaultGovernanceSettings().LaneLimits[string(lane)]
}

// ExecuteLaneCapacity returns the configured Execute-lane concurrency cap — the
// number of backlog executions that can run at once. The ETA engine divides a
// goal's total remaining work by this to model throughput. Falls back to the
// built-in default when governance cannot be loaded.
func (s *Service) ExecuteLaneCapacity() int {
	gov, err := s.governanceProvider.LoadGovernance()
	if err != nil {
		gov = DefaultGovernanceSettings()
	}
	return laneCapacity(gov, agentactivity.LaneExecute)
}

// ResetCircuitBreaker clears the circuit breaker for a specific item.
func (s *Service) ResetCircuitBreaker(itemKey string) error {
	return s.circuitBreaker.Reset(itemKey)
}

// GovernanceStatus returns the current governance state for the overview
// endpoint, including per-lane utilization. Execute-lane active and queue
// counts come from execution.Records (the canonical source for backlog
// item processing); Investigate / Review / Reconcile come from
// agentactivity.Records (the universal tracked-spawn ledger).
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

	executeActive := countActiveExecutions(records)
	queued := countQueuedExecutions(records)

	// Investigate / Review / Reconcile are populated by agentactivity. If
	// the activity reader is not wired (legacy NewService callers), the
	// lanes still appear in the response with active=0 — operators see the
	// shape, just not live numbers for those lanes. Once
	// SetActivityLaneReader is called by the wiring layer the counts go live.
	activityCounts := map[agentactivity.Lane]int{
		agentactivity.LaneInvestigate: 0,
		agentactivity.LaneExecute:     0,
		agentactivity.LaneReview:      0,
		agentactivity.LaneReconcile:   0,
	}
	if s.activityLaneReader != nil {
		if counts, lerr := s.activityLaneReader.LaneActiveCounts(); lerr == nil {
			for k, v := range counts {
				activityCounts[k] = v
			}
		}
	}
	// Execute lane utilization is the union of execution starts (always
	// live) and any agentactivity entries that landed in the lane (e.g.
	// follow-ups). Take the max — both views see the same canonical run
	// once it is started, but agentactivity may carry extra records.
	if activityCounts[agentactivity.LaneExecute] < executeActive {
		activityCounts[agentactivity.LaneExecute] = executeActive
	}

	lanes := make([]LaneStatus, 0, 4)
	for _, lane := range agentactivity.Lanes() {
		laneQueue := 0
		if lane == agentactivity.LaneExecute {
			laneQueue = queued
		}
		lanes = append(lanes, LaneStatus{
			Lane:     string(lane),
			Active:   activityCounts[lane],
			Capacity: laneCapacity(gov, lane),
			Queue:    laneQueue,
		})
	}

	totalActive := 0
	for _, l := range lanes {
		totalActive += l.Active
	}

	agentMaxTurns := gov.AgentMaxTurns
	if agentMaxTurns <= 0 {
		agentMaxTurns = 600
	}
	estimatedQueuedCost := float64(queued) * gov.CostPerTurnEstimate * float64(agentMaxTurns)

	return &GovernanceStatusResponse{
		Lanes:               lanes,
		ActiveExecutions:    totalActive,
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
