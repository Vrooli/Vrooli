package research

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"web-search/internal/findings"
	"web-search/internal/research/agentmanager"
)

// HighConfidenceThreshold is the confidence-gate constant for the L3 reconcile
// path. A distilled claim at or above this confidence may ACT (supersede an
// outdated finding, or be written as a high-trust finding); below it the
// reconcile FLAGS the contested finding into DISPUTED rather than silently
// overwriting a contested claim. Named here so the gate is a single SSOT.
const HighConfidenceThreshold = 0.75

var ErrInvalidInput = errors.New("invalid research input")

const (
	// DefaultGatherFindings is the bounded GATHER size used when a caller omits a
	// positive max.
	DefaultGatherFindings = 20
	// MaxGatherFindings is the HARD cap on the bounded GATHER sweep. OT-P1-003
	// requires the gather to read findings "semantically near the query", never a
	// whole-store scan; the cap is enforced server-side so a caller cannot widen
	// the sweep.
	MaxGatherFindings = 20
	// DefaultMaxResearchLoops is the iteration budget written into the L3 task
	// contract: the agent must converge (answer with what it has) within this
	// many search→read→gap→re-search loops rather than iterate indefinitely.
	// It is a prompt-level directive — the hard run lifecycle bound (timeout /
	// cancellation) is owned by agent-manager per the L3 design decision.
	DefaultMaxResearchLoops = 10
)

// GatheredFinding is one finding semantically near a gather query, projected to
// the fields the reconcile step reasons over.
type GatheredFinding struct {
	FindingID  string
	Claim      string
	Confidence float64
	// Status is the lifecycle state ("active" | "disputed").
	Status string
	// Score is the semantic relevance of this finding to the query.
	Score float64
}

// clampGatherLimit enforces the bounded-sweep contract: an omitted/non-positive
// max defaults to DefaultGatherFindings (itself bounded by the cap); any larger
// request is clamped to the service's gather cap (MaxGatherFindings unless
// overridden via Deps.GatherCap).
func (s *Service) clampGatherLimit(max int) int {
	def := DefaultGatherFindings
	if def > s.gatherCap {
		def = s.gatherCap
	}
	switch {
	case max <= 0:
		return def
	case max > s.gatherCap:
		return s.gatherCap
	default:
		return max
	}
}

// GatherRelatedFindings runs the bounded GATHER step of the
// research-and-reconcile loop: it returns the findings semantically NEAR the
// query, capped at MaxGatherFindings. The cap is enforced here (not left to the
// caller or the agent's free-form search), so the L3 agent calls this endpoint
// instead of an unbounded `findings search`. The returned slice is additionally
// truncated to the cap defensively, so a misbehaving seam cannot widen the
// sweep. Requires a wired Gatherer seam.
func (s *Service) GatherRelatedFindings(ctx context.Context, query string, max int) ([]GatheredFinding, int, error) {
	if s.gatherer == nil {
		return nil, 0, fmt.Errorf("research: gather unavailable: findings index not configured")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, 0, fmt.Errorf("research: gather query is required")
	}
	limit := s.clampGatherLimit(max)
	out, err := s.gatherer.Gather(ctx, query, limit)
	if err != nil {
		return nil, limit, err
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, limit, nil
}

// CycleResult is the outcome of one bounded research-and-reconcile cycle.
type CycleResult struct {
	// Gathered is the bounded set of nearby findings the cycle read first.
	Gathered []GatheredFinding
	// Brief is the answer produced before any curation.
	Brief Brief
	// Reconciled records the gate's decision per proposed reconcile item.
	Reconciled []ReconcileResult
	// ReconcileSkipped is true when the bounded reconcile post-step was
	// deliberately skipped because the answer step errored or produced no
	// summary — the store is never curated off a failed or empty run.
	ReconcileSkipped bool
}

// Answerer is the answer-first step of a research cycle: given the gathered
// findings it produces the brief and the reconcile items the post-step will
// apply. The cycle runs reconcile ONLY if this returns no error and a non-empty
// summary.
type Answerer func(ctx context.Context, gathered []GatheredFinding) (Brief, []ReconcileItem, error)

// RunResearchCycle encodes the OT-P1-003 budget order deterministically so it is
// unit-testable independent of the live agent loop: GATHER (bounded) -> answer
// (produce the brief) -> RECONCILE (bounded post-step). The reconcile step runs
// strictly AFTER a non-empty answer is produced; if the answer step errors or
// yields no summary, reconcile is skipped and the store is left untouched. This
// is the executable contract the L3 prompt mirrors.
func (s *Service) RunResearchCycle(ctx context.Context, query string, answer Answerer) (CycleResult, error) {
	if answer == nil {
		return CycleResult{}, fmt.Errorf("research: cycle answerer is required")
	}
	gathered, _, err := s.GatherRelatedFindings(ctx, query, s.gatherCap)
	if err != nil {
		return CycleResult{}, err
	}
	brief, items, err := answer(ctx, gathered)
	if err != nil {
		// Answer-first: a failed answer means we never curate the store.
		return CycleResult{Gathered: gathered, ReconcileSkipped: true}, err
	}
	if strings.TrimSpace(brief.Summary) == "" {
		// Nothing was answered -> skip the bounded reconcile post-step.
		return CycleResult{Gathered: gathered, Brief: brief, ReconcileSkipped: true}, nil
	}
	results, err := s.Reconcile(ctx, items)
	if err != nil {
		return CycleResult{Gathered: gathered, Brief: brief}, err
	}
	return CycleResult{Gathered: gathered, Brief: brief, Reconciled: results}, nil
}

// RunL3 starts an agent-manager run that performs the iterative
// research-and-reconcile loop and returns the run handle. The loop semantics
// (GATHER nearby findings -> research the gap with L2 tools -> RECONCILE:
// distill, supersede outdated, flag low-confidence contradictions) are encoded
// in the task prompt; the budget order is answer-first, curate as a bounded
// post-step.
func (s *Service) RunL3(ctx context.Context, query string) (agentmanager.RunResult, error) {
	return s.StartResearch(ctx, query, "")
}

func (s *Service) StartResearch(ctx context.Context, query, key string) (agentmanager.RunResult, error) {
	query = strings.TrimSpace(query)
	if query == "" || len(query) > 4096 {
		return agentmanager.RunResult{}, fmt.Errorf("%w: query must contain 1..4096 bytes", ErrInvalidInput)
	}
	if s.agentManager == nil {
		return agentmanager.RunResult{}, fmt.Errorf("%w: agent-manager not configured", agentmanager.ErrNotAvailable)
	}
	return s.agentManager.Spawn(ctx, agentmanager.SpawnRequest{
		Query:          query,
		Title:          "L3 research: " + query,
		IdempotencyKey: key, GatherCap: s.gatherCap, ConfidenceGate: s.confidenceGate, MaxLoops: s.maxLoops,
	})
}

// GetResearchStatus reads a declared L3 execution by id.
func (s *Service) GetResearchStatus(ctx context.Context, runID string) (agentmanager.RunState, error) {
	if s.agentManager == nil {
		return agentmanager.RunState{}, fmt.Errorf("%w: agent-manager not configured", agentmanager.ErrNotAvailable)
	}
	return s.agentManager.GetRunState(ctx, runID)
}

// ReconcileItem is one distilled claim the L3 reconcile post-step proposes to
// apply against an existing finding. The Service.Reconcile helper encodes the
// confidence gate deterministically so it is unit-testable: ACT when confident,
// FLAG (dispute) otherwise.
type ReconcileItem struct {
	// ExistingID is the contested existing finding the distilled claim bears on.
	ExistingID string
	// Confidence is the distilled claim's confidence in [0,1].
	Confidence float64
	// Contradicts is true when the distilled claim conflicts with the existing
	// finding (vs. merely reinforcing it).
	Contradicts bool
	// ReplacementID, when set on a high-confidence contradiction, is the new
	// finding that supersedes ExistingID.
	ReplacementID string
	// Reason explains the supersede / flag for the audit row.
	Reason string
}

// ReconcileAction names the action the gate chose for a ReconcileItem.
type ReconcileAction string

const (
	// ActionNone means the item did not contradict — nothing to curate.
	ActionNone ReconcileAction = "none"
	// ActionSupersede means a high-confidence contradiction retired the existing
	// finding.
	ActionSupersede ReconcileAction = "supersede"
	// ActionFlag means a low-confidence contradiction flagged the existing
	// finding as DISPUTED rather than overwriting it.
	ActionFlag ReconcileAction = "flag"
)

// ReconcileResult records the gate's decision for one item.
type ReconcileResult struct {
	ExistingID string
	Action     ReconcileAction
}

// Reconcile applies the confidence-gated curation policy for the L3 reconcile
// post-step over a set of distilled items. It is the deterministic core the L3
// loop's RECONCILE step performs against the findings store: a high-confidence
// contradiction SUPERSEDES the outdated finding; a low-confidence contradiction
// FLAGS it into DISPUTED. A non-contradiction is left untouched. The gate never
// silently overwrites a contested claim. Requires a wired Findings seam.
func (s *Service) Reconcile(ctx context.Context, items []ReconcileItem) ([]ReconcileResult, error) {
	if s.findings == nil {
		return nil, fmt.Errorf("research: reconcile unavailable: findings store not configured")
	}
	results := make([]ReconcileResult, 0, len(items))
	for _, it := range items {
		id := strings.TrimSpace(it.ExistingID)
		if id == "" {
			continue
		}
		switch {
		case !it.Contradicts:
			results = append(results, ReconcileResult{ExistingID: id, Action: ActionNone})
		case it.Confidence >= s.confidenceGate:
			reason := it.Reason
			if strings.TrimSpace(reason) == "" {
				reason = "superseded by higher-confidence L3 finding"
			}
			if _, err := s.findings.Supersede(ctx, id, strings.TrimSpace(it.ReplacementID), reason); err != nil {
				return results, fmt.Errorf("research: reconcile supersede %q: %w", id, err)
			}
			results = append(results, ReconcileResult{ExistingID: id, Action: ActionSupersede})
		default:
			reason := it.Reason
			if strings.TrimSpace(reason) == "" {
				reason = "low-confidence contradiction surfaced by L3 research"
			}
			if _, err := s.findings.Flag(ctx, id, reason); err != nil {
				return results, fmt.Errorf("research: reconcile flag %q: %w", id, err)
			}
			results = append(results, ReconcileResult{ExistingID: id, Action: ActionFlag})
		}
	}
	return results, nil
}

// compile-time guard: internalfindings.Service satisfies FindingsService.
var _ FindingsService = (findings.Service)(nil)

func (s *Service) WaitResearch(ctx context.Context, id string, seconds int) (agentmanager.RunState, error) {
	w, ok := s.agentManager.(agentmanager.Waiter)
	if !ok {
		return agentmanager.RunState{}, fmt.Errorf("%w: owner wait is not configured", agentmanager.ErrNotAvailable)
	}
	return w.Wait(ctx, id, seconds)
}
