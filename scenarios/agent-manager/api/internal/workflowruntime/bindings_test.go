package workflowruntime

import (
	"encoding/json"
	"strings"
	"testing"

	"agent-manager/internal/domain"
)

func TestEvaluateBindingsJsonPrettyReturnsValidPromptJSON(t *testing.T) {
	values, err := EvaluateBindings([]domain.WorkflowInputBinding{{
		Name: "constraints", Source: domain.WorkflowBindingInput, Selector: "$.constraints",
		Limit: 1, MaxBytes: 4096, RenderAs: "json_pretty", MissingPolicy: "error",
	}}, BindingContext{Input: json.RawMessage(`{"constraints":{"executionStrategy":"adaptive-improvement","maxSlices":4,"writeScope":["scenarios/swarm-manager/**"]}}`)})
	if err != nil {
		t.Fatalf("EvaluateBindings() error = %v", err)
	}
	rendered, ok := values["constraints"].(string)
	if !ok {
		t.Fatalf("json_pretty binding type = %T, want string", values["constraints"])
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(rendered), &decoded); err != nil {
		t.Fatalf("json_pretty output is not valid JSON: %v (%q)", err, rendered)
	}
	if decoded["executionStrategy"] != "adaptive-improvement" {
		t.Fatalf("decoded constraints = %#v", decoded)
	}
}

func TestEvaluateBindingsJsonKeepsStructuredValueForEndNodeAssembly(t *testing.T) {
	values, err := EvaluateBindings([]domain.WorkflowInputBinding{{
		Name: "result", Source: domain.WorkflowBindingInput, Selector: "$.result",
		Limit: 1, MaxBytes: 4096, RenderAs: "json", MissingPolicy: "error",
	}}, BindingContext{Input: json.RawMessage(`{"result":{"outcome":"complete"}}`)})
	if err != nil {
		t.Fatalf("EvaluateBindings() error = %v", err)
	}
	if _, ok := values["result"].(map[string]any); !ok {
		t.Fatalf("json binding type = %T, want structured map", values["result"])
	}
}

func TestRenderPromptRendersStructuredEvaluatorBindingInsideSkillEnvelope(t *testing.T) {
	const source = `<skills count="1">
  <skill id="evaluator"><![CDATA[
Assess the treatment result:
{{.treatment}}
]]></skill>
</skills>`

	got, err := RenderPrompt(source, map[string]any{"treatment": "{\n  \"quality\": \"good\",\n  \"token\": \"alpha\"\n}"})
	if err != nil {
		t.Fatalf("RenderPrompt() error = %v", err)
	}
	if strings.Contains(got, "{{.treatment}}") {
		t.Fatalf("RenderPrompt() left evaluator binding unresolved: %q", got)
	}
	if !strings.Contains(got, `"token": "alpha"`) || !strings.Contains(got, `"quality": "good"`) {
		t.Fatalf("RenderPrompt() did not embed structured treatment result: %q", got)
	}
}

func TestRenderPromptPreservesFullPlanAndRejectsOversizedInput(t *testing.T) {
	payload := strings.Repeat("contract and evidence\n", 8192) + "FINAL_REQUIRED_OUTCOME"
	got, err := RenderPrompt("Review the complete snapshot:\n{{.snapshot}}", map[string]any{"snapshot": payload})
	if err != nil || !strings.HasSuffix(got, "FINAL_REQUIRED_OUTCOME") || strings.Count(got, payload) != 1 {
		t.Fatalf("a bounded full-plan review must retain its final requirement exactly once: bytes=%d err=%v", len(got), err)
	}
	if _, err := RenderPrompt("{{.snapshot}}", map[string]any{"snapshot": strings.Repeat("x", (2<<20)+1)}); err == nil {
		t.Fatal("review input above the 2 MiB host ceiling must be rejected")
	}
}

func TestIndependentReviewSkillEmbedsSnapshotExactlyOnce(t *testing.T) {
	source := reconciledSkillTemplate(t, "swarm-manager-workflow-independent-review")
	payload := "BOUND_CANONICAL_REVIEW_EVIDENCE"
	got, err := RenderPrompt(source, map[string]any{"snapshot": payload, "entity": "bound execution"})
	if err != nil || strings.Count(got, payload) != 1 {
		t.Fatalf("the review must receive one copy of its immutable snapshot, got count=%d err=%v", strings.Count(got, payload), err)
	}
}
