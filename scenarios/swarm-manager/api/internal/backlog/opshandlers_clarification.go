// Clarification thread-commit action handlers.
//
// In the PULL→PUSH model, a clarification is an operation the runner starts; the
// agent's answer arrives as the operation's validated result on completion. These
// closed-vocabulary handlers materialize that result into the clarification
// thread the operator opened — appending the assistant turn and its parsed impact
// — instead of the old GET-time poll of the agent run's state. Thread identity,
// the operator's user messages, and their attachments are written by the HTTP
// entrypoint BEFORE the operation starts (commit-before-async), so they survive a
// start failure or retry; these handlers only append the assistant side.
package backlog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/opsrunner"
	"swarm-manager/internal/workshop"
)

// clarificationFallbackAnswer is written when a clarification round abstains or
// delivers an unparseable result, so the operator always sees a turn rather than
// a silently stalled thread.
const clarificationFallbackAnswer = "The clarification agent could not produce an answer. Please try again."

// commitClarificationTurn appends the completing clarification operation's
// assistant turn to its thread. It correlates the operation to the thread by the
// live run id the entrypoint stamped on the thread at start, appends the answer
// derived from the validated result, parses any impact, and resolves the thread
// on a terminal completed outcome. It is idempotent at the coordination layer:
// the dispatcher consumes the transition's idempotency key, so a re-delivered
// round never appends a second turn for the same execution.
func (d OpsHandlerDeps) commitClarificationTurn(_ context.Context, ac opsrunner.ActionContext) error {
	kind, name, err := splitItemRef(ac.Target.ID)
	if err != nil {
		return err
	}
	itemDir := d.Store.ItemDir(kind, name)
	runID := runIDForExecution(ac.Workflow, ac.ExecutionID)

	thread, err := findClarificationThread(itemDir, runID)
	if err != nil {
		return fmt.Errorf("commit-clarification: load threads for %s: %w", ac.Target.ID, err)
	}
	if thread == nil {
		// No thread correlates to this run — nothing to write (a legacy thread or a
		// clarification the entrypoint never linked). Treat as a benign no-op so the
		// transition still commits.
		return nil
	}

	content := clarificationAnswerFromResult(ac.Result)
	now := d.now().UTC().Format(time.RFC3339)
	thread.Messages = append(thread.Messages, workshop.ClarificationMessage{
		Role:      "assistant",
		Content:   content,
		CreatedAt: now,
	})
	if impact := workshop.ParseImpactXML(content); impact != nil {
		thread.LatestImpact = impact
	}
	// A terminal completed clarification resolves the thread; a continue keeps it
	// active for further follow-up. blocked/needs-attention leave it active so the
	// operator can retry.
	if ac.Outcome == "completed" {
		thread.Status = "resolved"
	}
	thread.UpdatedAt = now
	if err := workshop.SaveClarification(itemDir, thread); err != nil {
		return fmt.Errorf("commit-clarification: save thread for %s: %w", ac.Target.ID, err)
	}
	d.log().Info("backlog ops: clarification turn committed", "kind", kind, "name", name, "thread", thread.ID, "outcome", ac.Outcome)
	return nil
}

// runIDForExecution returns the live run id recorded on the workflow's operation
// record for an execution, empty if none (e.g. the synchronous test path).
func runIDForExecution(w agentops.WorkflowInstance, executionID string) string {
	for _, op := range w.Operations {
		if op.ExecutionID == executionID {
			return op.RunID
		}
	}
	return ""
}

// findClarificationThread returns the active thread correlated to a run id. When
// the run id is empty (synchronous path) it falls back to the most recently
// updated active thread, so tests and the sim path still land the turn.
func findClarificationThread(itemDir, runID string) (*workshop.ClarificationThread, error) {
	threads, err := workshop.LoadAllClarifications(itemDir)
	if err != nil {
		return nil, err
	}
	var fallback *workshop.ClarificationThread
	for i := range threads {
		t := threads[i]
		if runID != "" && t.RunID == runID {
			return &t, nil
		}
		if t.Status == "active" && (fallback == nil || t.UpdatedAt > fallback.UpdatedAt) {
			ft := t
			fallback = &ft
		}
	}
	if runID != "" {
		// A run id was expected but matched no thread: do not guess.
		return nil, nil
	}
	return fallback, nil
}

// clarificationAnswerFromResult extracts the human-readable answer from a
// clarification operation's validated handoff result. It prefers an explicit
// answer/summary field, falls back to the raw handoff, and to a generic message
// when the result is absent (an abstaining round).
func clarificationAnswerFromResult(raw json.RawMessage) string {
	if len(raw) == 0 {
		return clarificationFallbackAnswer
	}
	var envelope struct {
		Handoff json.RawMessage `json:"handoff"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || len(envelope.Handoff) == 0 {
		return clarificationFallbackAnswer
	}
	var h struct {
		Answer  string `json:"answer"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(envelope.Handoff, &h); err == nil {
		if strings.TrimSpace(h.Answer) != "" {
			return h.Answer
		}
		if strings.TrimSpace(h.Summary) != "" {
			return h.Summary
		}
	}
	return string(envelope.Handoff)
}
