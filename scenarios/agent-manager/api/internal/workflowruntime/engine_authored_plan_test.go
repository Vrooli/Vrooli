// Responsibility: retain engine test declarations within their original package.
package workflowruntime

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/workflowcatalog"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthoredPhasedPlanUsesReviewCorrectionAndCorrectedOutput(t *testing.T) { // [REQ:REQ-P2-001]
	definition := loadAuthoredPhasedPlanDefinition(t)
	engine, _, children := testEngine(t, definition)
	reviews := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
	engine.Subworkflows = reviews
	execution, err := engine.Start(context.Background(), revision(definition), phasedPlanInput(2), "authored-correction")
	if err != nil {
		t.Fatal(err)
	}

	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeStructured(children, children.requests[0].runID, map[string]any{"outcome": "continue", "handoff": "original", "correctionRequired": false, "approvalRequired": false})
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeLatestReview(t, reviews, false, "replace the handoff")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	if len(children.requests) != 2 || children.requests[1].source == nil || *children.requests[1].source != children.requests[0].runID {
		t.Fatalf("review rejection did not start named correction: %+v", children.requests)
	}
	if !strings.Contains(children.requests[1].prompt, "replace the handoff") {
		t.Fatalf("correction prompt omitted review note: %q", children.requests[1].prompt)
	}
	completeStructured(children, children.requests[1].runID, map[string]any{"outcome": "complete", "handoff": "corrected"})
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	// A final complete claim is still only a claim until independent review.
	pending := mustAdvance(t, engine, execution.ID)
	if pending.Status.Terminal() || pending.CurrentNodeID != "review" {
		t.Fatalf("final complete slice bypassed independent review: %+v", pending)
	}
	completeLatestReview(t, reviews, true, "accepted")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	final := mustAdvance(t, engine, execution.ID)
	if final.Status != domain.WorkflowExecutionSucceeded || !strings.Contains(string(final.Output), `"handoff":"corrected"`) || strings.Contains(string(final.Output), `"handoff":"original"`) {
		t.Fatalf("terminal output did not select correction: status=%s reason=%+v output=%s", final.Status, final.TerminalReason, final.Output)
	}
}

func TestAuthoredPhasedPlanEnforcesConsumerSliceLimit(t *testing.T) { // [REQ:REQ-P2-001]
	definition := loadAuthoredPhasedPlanDefinition(t)
	engine, store, children := testEngine(t, definition)
	reviews := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
	engine.Subworkflows = reviews
	execution, err := engine.Start(context.Background(), revision(definition), phasedPlanInput(1), "authored-max-slices")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeStructured(children, children.requests[0].runID, map[string]any{"outcome": "continue", "handoff": "one", "correctionRequired": false, "approvalRequired": false})
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeLatestReview(t, reviews, true, "accepted")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	final := mustAdvance(t, engine, execution.ID)
	if final.Status != domain.WorkflowExecutionBudgetExhausted || final.TerminalReason == nil || final.TerminalReason.BudgetName != "authored_limit" {
		t.Fatalf("slice limit terminal=%+v", final)
	}
	attempts, err := store.ListAttempts(context.Background(), execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	slices := 0
	for _, attempt := range attempts {
		if attempt.NodeID == "slice" {
			slices++
		}
	}
	if slices != 1 || len(children.requests) != 1 {
		t.Fatalf("maxSlices=1 created %d slice attempts and %d Run requests", slices, len(children.requests))
	}
}

func TestAuthoredPhasedPlanPreservesBlockedAndAbstainedTerminals(t *testing.T) { // [REQ:REQ-P2-001]
	for _, tc := range []struct {
		name  string
		value map[string]any
		want  domain.WorkflowExecutionStatus
	}{
		{name: "blocked", value: map[string]any{"outcome": "blocked", "handoff": "paused", "blocker": map[string]any{"code": "operator_input", "summary": "operator input required", "retryable": true}}, want: domain.WorkflowExecutionBlocked},
		{name: "abstained", value: map[string]any{"outcome": "abstained", "reason": "insufficient evidence"}, want: domain.WorkflowExecutionAbstained},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition := loadAuthoredPhasedPlanDefinition(t)
			engine, _, children := testEngine(t, definition)
			execution, err := engine.Start(context.Background(), revision(definition), phasedPlanInput(2), "authored-terminal-"+tc.name)
			if err != nil {
				t.Fatal(err)
			}
			mustAdvance(t, engine, execution.ID)
			mustAdvance(t, engine, execution.ID)
			completeStructured(children, children.requests[0].runID, tc.value)
			mustAdvance(t, engine, execution.ID)
			mustAdvance(t, engine, execution.ID)
			final := mustAdvance(t, engine, execution.ID)
			if final.Status != tc.want {
				t.Fatalf("terminal=%+v", final)
			}
		})
	}
}

func TestAuthoredPhasedPlanCreatesFreshRunForNextSlice(t *testing.T) { // [REQ:REQ-P2-001]
	definition := loadAuthoredPhasedPlanDefinition(t)
	engine, _, children := testEngine(t, definition)
	reviews := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
	engine.Subworkflows = reviews
	execution, err := engine.Start(context.Background(), revision(definition), phasedPlanInput(2), "authored-fresh-slices")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeStructured(children, children.requests[0].runID, map[string]any{"outcome": "continue", "handoff": "slice-one-handoff", "correctionRequired": false, "approvalRequired": false})
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeLatestReview(t, reviews, true, "accepted")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	if len(children.requests) != 2 || children.requests[0].runID == children.requests[1].runID {
		t.Fatalf("slice loop did not create distinct Runs: %+v", children.requests)
	}
	if children.requests[1].source != nil || !strings.Contains(children.requests[1].prompt, "slice-one-handoff") {
		t.Fatalf("next slice was not fresh with bounded handoff: %+v", children.requests[1])
	}
}

func TestAuthoredPhasedPlanAutoModeSkipsApprovalWait(t *testing.T) { // [REQ:SWM-AUTONOMY-001]
	definition := loadAuthoredPhasedPlanDefinition(t)
	engine, _, children := testEngine(t, definition)
	reviews := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
	engine.Subworkflows = reviews
	execution, err := engine.Start(context.Background(), revision(definition), phasedPlanInput(2, "auto"), "authored-auto-mode")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeStructured(children, children.requests[0].runID, map[string]any{"outcome": "continue", "handoff": "auto-slice", "correctionRequired": false, "approvalRequired": true})
	for i := 0; i < 4; i++ {
		mustAdvance(t, engine, execution.ID)
	}
	completeLatestReview(t, reviews, true, "accepted")
	for i := 0; i < 5; i++ {
		mustAdvance(t, engine, execution.ID)
	}
	if len(children.requests) != 2 {
		t.Fatalf("automatic gate still waited for approval: got %d slice runs", len(children.requests))
	}
}

func TestAuthoredPhasedPlanManualModeRestoresApprovalWait(t *testing.T) { // [REQ:SWM-AUTONOMY-002]
	definition := loadAuthoredPhasedPlanDefinition(t)
	engine, _, children := testEngine(t, definition)
	reviews := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
	engine.Subworkflows = reviews
	execution, err := engine.Start(context.Background(), revision(definition), phasedPlanInput(2, "manual"), "authored-manual-mode")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeStructured(children, children.requests[0].runID, map[string]any{"outcome": "continue", "handoff": "manual-slice", "correctionRequired": false, "approvalRequired": true})
	for i := 0; i < 4; i++ {
		mustAdvance(t, engine, execution.ID)
	}
	completeLatestReview(t, reviews, true, "accepted")
	for i := 0; i < 5; i++ {
		mustAdvance(t, engine, execution.ID)
	}
	if len(children.requests) != 1 {
		t.Fatalf("manual gate bypassed approval: got %d slice runs", len(children.requests))
	}
	state, err := engine.Store.Get(context.Background(), execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if state.CurrentNodeID != "approval" {
		t.Fatalf("manual mode current node = %q, want approval", state.CurrentNodeID)
	}
}

func TestAuthoredAdaptivePlanApprovalReasons(t *testing.T) { // [REQ:SWM-P0-015]
	unattended := func(v bool) *bool { return &v }
	for _, tc := range []struct {
		name, mode, reason string
		unattended         *bool
		wantFresh          bool
		falseApprovalFlag  bool
	}{
		{"adaptive phase", "manual", "phase-boundary", unattended(true), true, false},
		{"ordinary phase", "manual", "phase-boundary", unattended(false), false, false},
		{"unattended absent defaults to manual", "manual", "phase-boundary", unattended(false), false, false},
		{"adaptive missing reason", "manual", "", unattended(true), false, false},
		{"adaptive operator", "manual", "operator-decision", unattended(true), false, false},
		{"adaptive operator with auto phases", "auto", "operator-decision", unattended(true), false, false},
		{"ordinary operator with auto phases", "auto", "operator-decision", unattended(false), false, false},
		{"operator reason overrides false flag", "auto", "operator-decision", unattended(true), false, true},
		{"ordinary auto phase", "auto", "phase-boundary", unattended(false), true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition := loadAuthoredPhasedPlanDefinition(t)
			engine, _, children := testEngine(t, definition)
			reviews := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
			engine.Subworkflows = reviews
			var input map[string]any
			if err := json.Unmarshal(phasedPlanInput(2, tc.mode), &input); err != nil {
				t.Fatal(err)
			}
			if tc.unattended != nil {
				input["constraints"].(map[string]any)["unattended"] = *tc.unattended
			}
			raw, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			execution, err := engine.Start(t.Context(), revision(definition), raw, "approval-reason-"+tc.name)
			if err != nil {
				t.Fatal(err)
			}
			mustAdvance(t, engine, execution.ID)
			mustAdvance(t, engine, execution.ID)
			result := map[string]any{"outcome": "continue", "handoff": "verified first repair", "correctionRequired": false, "approvalRequired": !tc.falseApprovalFlag}
			if tc.reason != "" {
				result["approvalReason"] = tc.reason
			}
			completeStructured(children, children.requests[0].runID, result)
			for i := 0; i < 4; i++ {
				mustAdvance(t, engine, execution.ID)
			}
			if len(children.requests) != 1 {
				t.Fatal("next repair started before independent review completed")
			}
			completeLatestReview(t, reviews, true, "independent checks passed")
			for i := 0; i < 5; i++ {
				mustAdvance(t, engine, execution.ID)
			}
			state, err := engine.Store.Get(t.Context(), execution.ID)
			if err != nil {
				t.Fatal(err)
			}
			if !tc.wantFresh {
				if len(children.requests) != 1 || state.CurrentNodeID != "approval" {
					t.Fatalf("operator boundary was bypassed: children=%d node=%s", len(children.requests), state.CurrentNodeID)
				}
				return
			}
			if len(children.requests) != 2 || children.requests[1].source != nil || children.requests[0].runID == children.requests[1].runID {
				t.Fatalf("routine adaptive repair did not continue with fresh context: children=%d node=%s", len(children.requests), state.CurrentNodeID)
			}
			if !strings.Contains(children.requests[1].prompt, "verified first repair") {
				t.Fatal("fresh context omitted the preceding verified repair handoff")
			}
		})
	}
}

func TestAuthoredPhasedPlanFeedsCorrectedHandoffToNextSlice(t *testing.T) { // [REQ:REQ-P2-001]
	definition := loadAuthoredPhasedPlanDefinition(t)
	engine, _, children := testEngine(t, definition)
	reviews := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
	engine.Subworkflows = reviews
	execution, err := engine.Start(context.Background(), revision(definition), phasedPlanInput(2), "authored-corrected-handoff")
	if err != nil {
		t.Fatal(err)
	}

	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeStructured(children, children.requests[0].runID, map[string]any{"outcome": "continue", "handoff": "superseded-handoff", "correctionRequired": false, "approvalRequired": false})
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeLatestReview(t, reviews, false, "replace the handoff")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeStructured(children, children.requests[1].runID, map[string]any{"outcome": "continue", "handoff": "corrected-handoff", "correctionRequired": false, "approvalRequired": false})
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeLatestReview(t, reviews, true, "accepted")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	if len(children.requests) != 3 {
		t.Fatalf("requests=%d, want original, correction, and next slice", len(children.requests))
	}
	nextSlice := children.requests[2]
	if nextSlice.source != nil || !strings.Contains(nextSlice.prompt, "corrected-handoff") {
		t.Fatalf("next slice did not receive corrected handoff: %+v", nextSlice)
	}
}

func loadAuthoredPhasedPlanDefinition(t *testing.T) domain.WorkflowDefinition {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "swarm-manager", ".vrooli", "agent-manager", "phased-plan-drain.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read authored phased plan: %v", err)
	}
	parsed, err := workflowcatalog.Parse(raw, nil)
	if err != nil {
		t.Fatalf("parse authored phased plan: %v", err)
	}
	if len(parsed.Diagnostics) != 0 {
		t.Fatalf("authored phased plan diagnostics: %+v", parsed.Diagnostics)
	}
	definition := parsed.Definition
	// Runtime receives reconciled definitions, where prompt-manager content is
	// materialized into promptTemplate. These authored-workflow tests exercise
	// that runtime contract rather than accidentally testing raw source JSON.
	for i := range definition.Nodes {
		node := &definition.Nodes[i]
		if node.Run != nil && node.Run.PromptRef != nil && node.Run.PromptTemplate == "" {
			node.Run.PromptTemplate = reconciledSkillTemplate(t, node.Run.PromptRef.SkillID)
		}
		if node.Continue != nil && node.Continue.PromptRef != nil && node.Continue.PromptTemplate == "" {
			node.Continue.PromptTemplate = reconciledSkillTemplate(t, node.Continue.PromptRef.SkillID)
		}
	}
	return definition
}

func reconciledSkillTemplate(t *testing.T, skillID string) string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "prompt-manager", "store", "skills", "packs", "core", skillID, "SKILL.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read reconciled prompt skill %q: %v", skillID, err)
	}
	if strings.TrimSpace(string(content)) == "" {
		t.Fatalf("reconciled prompt skill %q was empty", skillID)
	}
	return string(content)
}

func phasedPlanInput(maxSlices int, modes ...string) json.RawMessage {
	mode := "manual"
	if len(modes) > 0 {
		mode = modes[0]
	}
	value, _ := json.Marshal(map[string]any{
		"projectRoot":     "/repo",
		"plan":            map[string]any{"reference": "plan-1", "frontierDigest": "sha256:frontier"},
		"planExecutionId": "plan-execution-1",
		"consumer":        map[string]any{"executionId": "execution-1", "entityKind": "execute", "entityName": "plan", "entityVersion": "sha256:entity"},
		"constraints":     map[string]any{"maxSlices": maxSlices, "writeScope": []string{"scenarios/example/**"}, "sliceApprovalMode": mode},
	})
	return value
}

func completeStructured(children *fakeChildren, runID uuid.UUID, value map[string]any) {
	raw, _ := json.Marshal(value)
	state := children.states[runID]
	state.Terminal = true
	state.Result = &domain.RunResult{FinalOutput: string(raw), Structured: &domain.StructuredResult{Status: domain.StructuredResultSuccess, Value: raw}}
	children.states[runID] = state
}

func completeLatestReview(t *testing.T, reviews *fakeSubworkflows, accepted bool, note string) {
	t.Helper()
	if len(reviews.starts) == 0 {
		t.Fatal("review workflow was not started")
	}
	start := reviews.starts[len(reviews.starts)-1]
	id := reviews.byKey[start.IdempotencyKey]
	output, _ := json.Marshal(map[string]any{"result": map[string]any{"accepted": accepted, "note": note}})
	reviews.states[id] = SubworkflowState{ExecutionID: id, Terminal: true, Status: domain.WorkflowExecutionSucceeded, Output: output}
}
