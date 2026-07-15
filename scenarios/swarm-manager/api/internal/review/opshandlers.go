// Declarative-operation completion handlers for the review flow.
//
// In the PULL->PUSH model, a review round is an operation the runner starts; the
// agent's evidence + classification arrive as the operation's validated result on
// completion. These closed-vocabulary handlers materialize that result into the
// review round the reroute opened — writing evidence, assessment, classification,
// and the normalized verdict — instead of the old GET-time poll of the agent
// run's state.
//
// CRITICAL (recommendation-not-mutation): an agent verdict is a RECOMMENDATION.
// commit-review-round records the review artifacts + recommended verdict and
// opens the operator review gate (flips the item out of in_review to
// review_pending via the SAME onRoundTerminal callback the poller fired), but
// performs NO terminal item-status mutation. The operator's accept/fail/followup
// decision (backlog review_decide) stays the sole authority for the item's
// terminal status, with its own immutable decision record.
package review

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/opsrunner"
)

// RegisterOpsHandlers binds the review completion handlers onto the runner's
// action registry. commit-review-round finalizes a review-round operation;
// request-evidence appends a targeted evidence-request sub-round's result into
// the open round's request thread. Both are backlog-item scoped (the initiative
// review round commits through commit-initiative-review in package
// initiativereview). Registration overrides the registry's pre-registered
// no-ops.
func (s *Service) RegisterOpsHandlers(reg *opsrunner.ActionRegistry) {
	reg.Register(agentops.ActionCommitReviewRound, s.commitReviewRound)
	reg.Register(agentops.ActionRequestEvidence, s.requestEvidence)
}

// reviewHandoff is the enriched review declared output carried under the
// operation result's `handoff` object (the mode's validated round output). The
// completion handler materializes the review Round from it deterministically.
type reviewHandoff struct {
	Verdict                string                  `json:"verdict"`
	AgentAssessment        string                  `json:"agent_assessment"`
	Evidence               []EvidenceItem          `json:"evidence"`
	ImprovementSuggestions []ImprovementSuggestion `json:"improvement_suggestions"`
	RegressionIntroduced   bool                    `json:"regression_introduced"`
	Notes                  []string                `json:"notes"`
	Summary                string                  `json:"summary"`
}

// reviewResultEnvelope is the reviewResult contract payload the bridge delivers:
// the normalized verdict plus the review handoff.
type reviewResultEnvelope struct {
	Verdict string          `json:"verdict"`
	Handoff json.RawMessage `json:"handoff"`
}

// commitReviewRound materializes the completing review-round operation's result
// into the open review round and opens the operator review gate. It correlates
// the round by the live run id the reroute stamped on it at start. Idempotent: a
// re-delivered round whose correlated round is already terminal is a no-op.
func (s *Service) commitReviewRound(ctx context.Context, ac opsrunner.ActionContext) error {
	kind, name, err := splitReviewItemRef(ac.Target.ID)
	if err != nil {
		return err
	}
	itemDir := s.resolveItemDir(kind, name)
	runID := RunIDForExecution(ac.Workflow, ac.ExecutionID)

	round, err := FindGatheringRoundByRunID(itemDir, runID)
	if err != nil {
		return fmt.Errorf("commit-review-round: load rounds for %s: %w", ac.Target.ID, err)
	}
	if round == nil {
		// No round correlates to this run — nothing to write (a legacy round or a
		// round the reroute never linked). Benign no-op so the transition commits.
		return nil
	}
	if round.Status == RoundStatusComplete || round.Status == RoundStatusFailed {
		return nil // idempotent replay: already finalized
	}

	FinalizeRoundFromResult(round, ac.Result, ac.Outcome)

	if err := SaveRound(itemDir, *round); err != nil {
		return fmt.Errorf("commit-review-round: save round for %s: %w", ac.Target.ID, err)
	}

	// Open the operator review gate (in_review -> review_pending). This is NOT a
	// terminal item mutation; the operator still decides the terminal status.
	if s.onRoundTerminal != nil {
		s.onRoundTerminal(ctx, kind, name, *round)
	}
	slog.Info("review ops: review round committed", "kind", kind, "name", name,
		"round", round.RoundNum, "status", round.Status, "outcome", ac.Outcome, "execution", ac.ExecutionID)
	return nil
}

// requestEvidence appends a completing evidence-request operation's gathered
// evidence + assistant turn into the open round's request thread, correlated by
// the run id stamped on the thread at start. A completed round resolves the
// thread; a continue keeps it open. Idempotent: a re-delivered round whose thread
// is already resolved is a no-op.
func (s *Service) requestEvidence(_ context.Context, ac opsrunner.ActionContext) error {
	kind, name, err := splitReviewItemRef(ac.Target.ID)
	if err != nil {
		return err
	}
	itemDir := s.resolveItemDir(kind, name)
	runID := RunIDForExecution(ac.Workflow, ac.ExecutionID)

	rounds, err := LoadRounds(itemDir)
	if err != nil {
		return fmt.Errorf("request-evidence: load rounds for %s: %w", ac.Target.ID, err)
	}
	roundIdx, threadIdx := findThreadByRunID(rounds, runID)
	if roundIdx < 0 {
		return nil // no thread correlates to this run: benign no-op
	}
	round := &rounds[roundIdx]
	thread := &round.RequestThreads[threadIdx]
	if thread.Status == "fulfilled" || thread.Status == "dismissed" {
		return nil // idempotent replay
	}

	handoff := parseReviewHandoff(ac.Result)
	round.Evidence = append(round.Evidence, handoff.Evidence...)
	content := firstNonEmptyString(handoff.Summary, handoff.AgentAssessment, "Evidence gathered.")
	addedIDs := make([]string, 0, len(handoff.Evidence))
	for _, ev := range handoff.Evidence {
		if ev.ID != "" {
			addedIDs = append(addedIDs, ev.ID)
		}
	}
	thread.Messages = append(thread.Messages, RequestMessage{
		Role:             "assistant",
		Content:          content,
		Timestamp:        s.now().Format(time.RFC3339),
		AddedEvidenceIDs: addedIDs,
	})
	if isEvidenceCompleteOutcome(ac.Outcome) {
		thread.Status = "fulfilled"
	}

	if err := SaveRound(itemDir, *round); err != nil {
		return fmt.Errorf("request-evidence: save round for %s: %w", ac.Target.ID, err)
	}
	slog.Info("review ops: evidence request committed", "kind", kind, "name", name,
		"round", round.RoundNum, "thread", thread.ID, "outcome", ac.Outcome, "added_evidence", len(addedIDs))
	return nil
}

// FinalizeRoundFromResult materializes a completing review operation's validated
// result into a gathering round and sets its terminal status. It is shared by the
// backlog review-round handler and the initiative-review handler so both
// reproduce the review Round identically. The agent verdict is a RECOMMENDATION:
// a successful review (accepted / changes-requested) finalizes the round complete
// (downgraded to failed if it lacks a valid assessment/classification); an
// abstain or genuine failure finalizes it failed with its gathered artifacts
// intact. It never mutates the round's identity (number, run id, generated-at,
// execution id).
func FinalizeRoundFromResult(round *Round, result json.RawMessage, outcome string) {
	applyReviewHandoff(round, parseReviewHandoff(result))
	if isReviewSuccessOutcome(outcome) {
		round.Status = RoundStatusComplete
		*round = normalizeRound(*round) // downgrades to failed if assessment/classification missing
	} else {
		round.Status = RoundStatusFailed
		if strings.TrimSpace(round.FailureReason) == "" {
			round.FailureReason = reviewAbstainReason(outcome)
		}
	}
	round.CurrentRunStatus = ""
}

// applyReviewHandoff copies the enriched handoff fields onto the round, preserving
// the round's identity (number, run id, generated-at, execution id).
func applyReviewHandoff(round *Round, h reviewHandoff) {
	round.Classification = strings.TrimSpace(h.Verdict)
	round.AgentAssessment = strings.TrimSpace(h.AgentAssessment)
	round.RegressionIntroduced = h.RegressionIntroduced
	if len(h.Evidence) > 0 {
		round.Evidence = h.Evidence
	}
	if round.Evidence == nil {
		round.Evidence = []EvidenceItem{}
	}
	if len(h.ImprovementSuggestions) > 0 {
		round.ImprovementSuggestions = h.ImprovementSuggestions
	}
	if len(h.Notes) > 0 {
		round.Notes = h.Notes
	}
}

// parseReviewHandoff extracts the enriched review handoff from a review operation
// result. It reads the reviewResult envelope's `handoff` object first (review-
// round), and falls back to treating the whole result as the handoff (an
// evidence-request handoff-style result). An absent/garbled result yields a zero
// handoff, which the callers treat as an abstaining round.
func parseReviewHandoff(raw json.RawMessage) reviewHandoff {
	if len(raw) == 0 {
		return reviewHandoff{}
	}
	var envelope reviewResultEnvelope
	if err := json.Unmarshal(raw, &envelope); err == nil && len(envelope.Handoff) > 0 {
		var h reviewHandoff
		if json.Unmarshal(envelope.Handoff, &h) == nil {
			if strings.TrimSpace(h.Verdict) == "" {
				h.Verdict = strings.TrimSpace(envelope.Verdict)
			}
			return h
		}
	}
	var direct struct {
		Handoff json.RawMessage `json:"handoff"`
	}
	if err := json.Unmarshal(raw, &direct); err == nil && len(direct.Handoff) > 0 {
		var h reviewHandoff
		if json.Unmarshal(direct.Handoff, &h) == nil {
			return h
		}
	}
	var h reviewHandoff
	_ = json.Unmarshal(raw, &h)
	return h
}

// isReviewSuccessOutcome reports whether a review-round outcome finalizes the
// round as a successfully-completed review (a recommendation is available). Both
// accepted and changes-requested are successful reviews; only abstain/failed are
// not.
func isReviewSuccessOutcome(outcome string) bool {
	switch outcome {
	case "accepted", "changes-requested":
		return true
	default:
		return false
	}
}

func isEvidenceCompleteOutcome(outcome string) bool {
	return outcome == "completed"
}

func reviewAbstainReason(outcome string) string {
	if outcome == "failed" {
		return "review round failed"
	}
	return "review agent could not derive an honest verdict; round abstained to operator attention"
}

// RunIDForExecution returns the live run id recorded on the workflow's operation
// record for an execution, empty if none (e.g. the synchronous test path).
func RunIDForExecution(w agentops.WorkflowInstance, executionID string) string {
	for _, op := range w.Operations {
		if op.ExecutionID == executionID {
			return op.RunID
		}
	}
	return ""
}

// FindGatheringRoundByRunID returns the review round correlated to a run id. When
// the run id is empty (synchronous path) it falls back to the most recent
// gathering round, so tests and the sim path still land the result. Shared by the
// backlog and initiative review completion handlers.
func FindGatheringRoundByRunID(itemDir, runID string) (*Round, error) {
	rounds, err := LoadRounds(itemDir)
	if err != nil {
		return nil, err
	}
	var fallback *Round
	for i := range rounds {
		r := rounds[i]
		if runID != "" && r.RunID == runID {
			rc := r
			return &rc, nil
		}
		if r.Status == RoundStatusGathering && (fallback == nil || r.RoundNum > fallback.RoundNum) {
			rc := r
			fallback = &rc
		}
	}
	if runID != "" {
		return nil, nil // a run id was expected but matched no round: do not guess
	}
	return fallback, nil
}

// findThreadByRunID returns the indices of the round + request thread correlated
// to a run id, or (-1, -1) if none. An empty run id falls back to the most recent
// pending thread.
func findThreadByRunID(rounds []Round, runID string) (roundIdx, threadIdx int) {
	fallbackRound, fallbackThread := -1, -1
	for ri := range rounds {
		for ti := range rounds[ri].RequestThreads {
			t := rounds[ri].RequestThreads[ti]
			if runID != "" && t.RunID == runID {
				return ri, ti
			}
			if runID == "" && t.Status == "pending" {
				fallbackRound, fallbackThread = ri, ti
			}
		}
	}
	if runID != "" {
		return -1, -1
	}
	return fallbackRound, fallbackThread
}

// splitReviewItemRef parses a target ref ("kind/name") into its kind and name.
func splitReviewItemRef(id string) (kind, name string, err error) {
	parts := strings.SplitN(strings.TrimSpace(id), "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("review ops: target ref %q is not a kind/name ref", id)
	}
	return parts[0], parts[1], nil
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
