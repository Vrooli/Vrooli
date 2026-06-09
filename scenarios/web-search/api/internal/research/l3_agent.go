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
	b.WriteString("1. GATHER: run `web-search findings search \"<query>\"` to load the NEARBY existing findings (bounded — do not scan the whole store).\n")
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
