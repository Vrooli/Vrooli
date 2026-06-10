package research

import (
	"context"
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

const (
	// DefaultGatherFindings is the bounded GATHER size used when a caller omits a
	// positive max.
	DefaultGatherFindings = 20
	// MaxGatherFindings is the HARD cap on the bounded GATHER sweep. OT-P1-003
	// requires the gather to read findings "semantically near the query", never a
	// whole-store scan; the cap is enforced server-side so a caller cannot widen
	// the sweep.
	MaxGatherFindings = 20
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
// max defaults to DefaultGatherFindings; any larger request is clamped to the
// MaxGatherFindings hard cap.
func clampGatherLimit(max int) int {
	switch {
	case max <= 0:
		return DefaultGatherFindings
	case max > MaxGatherFindings:
		return MaxGatherFindings
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
	limit := clampGatherLimit(max)
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
	gathered, _, err := s.GatherRelatedFindings(ctx, query, MaxGatherFindings)
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
	if s.agentManager == nil {
		return agentmanager.RunResult{}, fmt.Errorf("research: L3 unavailable: agent-manager not configured")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return agentmanager.RunResult{}, fmt.Errorf("research: L3 query is required")
	}
	return s.agentManager.Spawn(ctx, agentmanager.SpawnRequest{
		Query:  query,
		Title:  "L3 research: " + query,
		Prompt: buildL3Prompt(query),
	})
}

// GetResearchStatus polls an L3 run by id.
func (s *Service) GetResearchStatus(ctx context.Context, runID string) (agentmanager.RunState, error) {
	if s.agentManager == nil {
		return agentmanager.RunState{}, fmt.Errorf("research: L3 unavailable: agent-manager not configured")
	}
	return s.agentManager.GetRunState(ctx, runID)
}

// buildL3Prompt renders the L3 research-and-reconcile task prompt. It encodes
// the GATHER -> RESEARCH -> RECONCILE loop and the confidence-gated curation
// policy the bounded post-step must follow.
func buildL3Prompt(query string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Research question: %s\n\n", query)
	b.WriteString("You are an L3 research agent for the web-search scenario. Follow this loop:\n\n")
	fmt.Fprintf(&b, "1. GATHER: run `web-search research gather \"<query>\"` to load the NEARBY existing findings. This is a BOUNDED sweep (the server caps it at %d findings semantically near the query) — use it instead of a free-form `findings search`; never scan the whole store.\n", MaxGatherFindings)
	b.WriteString("2. RESEARCH the gap: use `web-search research l2 \"<focused sub-query>\"` (full-page fetch + cited synthesis) for what the existing findings do not already cover. Use `web-search search` for fresh candidate URLs.\n")
	b.WriteString("3. RECONCILE (bounded post-step, answer first): distill what you learned into citation-backed claims and curate the store:\n")
	fmt.Fprintf(&b, "   - When a new claim is well-supported and clearly contradicts an existing finding (confidence >= %.2f), SUPERSEDE the outdated finding via `web-search findings supersede <old-id> --replacement <new-id> --reason \"<why>\"`.\n", HighConfidenceThreshold)
	b.WriteString("   - When sources conflict and you are NOT confident, FLAG the contested finding via `web-search findings flag <id> --reason \"<contradiction>\"` (it moves to DISPUTED). NEVER silently overwrite a contested claim.\n")
	b.WriteString("   - Write new citation-backed claims via `web-search findings add --claim \"...\" --confidence <0..1> --source l3 --citations \"url|title,...\"`.\n\n")
	b.WriteString("Ground every claim in a citation. Abstain rather than fabricate. Keep curation bounded — answer the question first, then reconcile.\n")
	return b.String()
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
		case it.Confidence >= HighConfidenceThreshold:
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
