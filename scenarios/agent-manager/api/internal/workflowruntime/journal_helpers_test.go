package workflowruntime

import (
	"encoding/json"
	"testing"

	"agent-manager/internal/domain"
)

// projectOld reproduces the pre-enrichment journal projection (payload only) so a
// test can prove the enriched ProjectJournal does not perturb how the original
// expanded conditions evaluate.
func projectOld(journal []*domain.WorkflowJournalEntry) []any {
	values := make([]any, 0, len(journal))
	for _, entry := range journal {
		var value any
		_ = json.Unmarshal(entry.Payload, &value)
		values = append(values, value)
	}
	return values
}

func je(seq int64, kind domain.WorkflowJournalKind, nodeID string, payload map[string]any) *domain.WorkflowJournalEntry {
	b, _ := json.Marshal(payload)
	return &domain.WorkflowJournalEntry{Sequence: seq, Kind: kind, NodeID: nodeID, Payload: b}
}

func sliceAttempt(seq int64) *domain.WorkflowJournalEntry {
	return je(seq, domain.WorkflowJournalAttempt, "slice", map[string]any{"nodeId": "slice", "ordinal": 0, "strategy": "fresh_run"})
}

func structured(seq int64, node string, value map[string]any) *domain.WorkflowJournalEntry {
	return je(seq, domain.WorkflowJournalStructured, node, map[string]any{"status": "success", "value": value})
}

func reviewOutput(seq int64, accepted bool) *domain.WorkflowJournalEntry {
	return je(seq, domain.WorkflowJournalChild, "review", map[string]any{"childExecutionId": "child", "status": "succeeded", "output": map[string]any{"review": map[string]any{"accepted": accepted, "note": "n"}}})
}

const (
	oldLatestValue     = "journal.filter(j, has(j.value) && has(j.status) && j.status == 'success')[journal.filter(j, has(j.value) && has(j.status) && j.status == 'success').size() - 1].value"
	oldSliceCount      = "journal.filter(j, has(j.nodeId) && j.nodeId == 'slice' && has(j.ordinal)).size()"
	oldReviewNotAccept = "(journal.filter(j, has(j.output) && has(j.output.review) && has(j.output.review.accepted)).size() > 0 && !journal.filter(j, has(j.output) && has(j.output.review) && has(j.output.review.accepted))[journal.filter(j, has(j.output) && has(j.output.review) && has(j.output.review.accepted)).size() - 1].output.review.accepted)"
	newReviewNotAccept = "(has(latest(journal, 'review').output) && !latest(journal, 'review').output.review.accepted)"
)

// TestJournalHelpersMatchExpandedForms proves the phase-3 CEL helpers evaluate
// identically to the verbose journal.filter incantations they replace, across a
// range of journal states, AND that the enriched projection leaves the original
// expanded forms evaluating exactly as before.
func TestJournalHelpersMatchExpandedForms(t *testing.T) {
	eval, err := NewExpressionEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	input := map[string]any{"constraints": map[string]any{"maxSlices": 2}}

	fixtures := map[string][]*domain.WorkflowJournalEntry{
		"continue_correction_required": {
			sliceAttempt(2),
			structured(3, "slice", map[string]any{"outcome": "continue", "handoff": "h", "completedSlice": "s", "correctionRequired": true, "approvalRequired": false}),
		},
		"continue_review_rejected": {
			sliceAttempt(2),
			structured(3, "slice", map[string]any{"outcome": "continue", "handoff": "h", "completedSlice": "s", "correctionRequired": false, "approvalRequired": false}),
			reviewOutput(4, false),
		},
		"continue_review_accepted": {
			sliceAttempt(2),
			structured(3, "slice", map[string]any{"outcome": "continue", "handoff": "h", "completedSlice": "s", "correctionRequired": false, "approvalRequired": true}),
			reviewOutput(4, true),
		},
		"correction_complete_supersedes_slice": {
			sliceAttempt(2),
			structured(3, "slice", map[string]any{"outcome": "continue", "handoff": "h", "completedSlice": "s", "correctionRequired": true, "approvalRequired": false}),
			reviewOutput(4, false),
			je(5, domain.WorkflowJournalAttempt, "correction", map[string]any{"nodeId": "correction", "ordinal": 0}),
			structured(6, "correction", map[string]any{"outcome": "complete", "handoff": "c", "completedSlice": "s corrected"}),
		},
		"two_slices_at_limit": {
			sliceAttempt(2),
			structured(3, "slice", map[string]any{"outcome": "continue", "handoff": "h", "completedSlice": "s", "correctionRequired": false, "approvalRequired": false}),
			sliceAttempt(4),
			structured(5, "slice", map[string]any{"outcome": "continue", "handoff": "h2", "completedSlice": "s2", "correctionRequired": false, "approvalRequired": false}),
		},
	}

	pairs := []struct {
		name string
		old  string
		new  string
	}{
		{"outcome_complete", oldLatestValue + ".outcome == 'complete'", "latest(journal).value.outcome == 'complete'"},
		{"outcome_blocked", oldLatestValue + ".outcome == 'blocked'", "latest(journal).value.outcome == 'blocked'"},
		{"outcome_abstained", oldLatestValue + ".outcome == 'abstained'", "latest(journal).value.outcome == 'abstained'"},
		{
			"route_to_correction",
			"(" + oldLatestValue + ".outcome == 'continue' && has(" + oldLatestValue + ".correctionRequired) && " + oldLatestValue + ".correctionRequired) || " + oldReviewNotAccept,
			"(latest(journal).value.outcome == 'continue' && latest(journal).value.correctionRequired) || " + newReviewNotAccept,
		},
		{
			"decision_to_approval",
			oldLatestValue + ".outcome == 'continue' && " + oldLatestValue + ".approvalRequired && " + oldSliceCount + " < input.constraints.maxSlices",
			"latest(journal).value.outcome == 'continue' && latest(journal).value.approvalRequired && count(journal, 'slice') < input.constraints.maxSlices",
		},
		{
			"decision_to_slice_limit",
			oldLatestValue + ".outcome == 'continue' && " + oldSliceCount + " >= input.constraints.maxSlices",
			"latest(journal).value.outcome == 'continue' && count(journal, 'slice') >= input.constraints.maxSlices",
		},
	}

	type outcome struct {
		v   bool
		err bool
	}
	run := func(cond string, journal []any) outcome {
		v, err := eval.Evaluate(cond, ExpressionContext{Input: input, Journal: journal})
		return outcome{v: v, err: err != nil}
	}

	for fname, entries := range fixtures {
		oldProj := projectOld(entries)
		newProj := ProjectJournal(entries)
		for _, p := range pairs {
			oldOnOld := run(p.old, oldProj)
			oldOnNew := run(p.old, newProj)
			newOnNew := run(p.new, newProj)
			if oldOnOld != oldOnNew {
				t.Errorf("[%s/%s] enrichment perturbed the old form: old@old=%+v old@new=%+v", fname, p.name, oldOnOld, oldOnNew)
			}
			if oldOnNew != newOnNew {
				t.Errorf("[%s/%s] helper form diverged: old=%+v new=%+v", fname, p.name, oldOnNew, newOnNew)
			}
		}
	}
}
