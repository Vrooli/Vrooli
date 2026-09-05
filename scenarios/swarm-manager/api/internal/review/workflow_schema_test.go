package review

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

func TestReviewWorkflowSchemasEnforceEvidenceGaps(t *testing.T) {
	for _, tc := range []struct {
		name  string
		path  string
		valid map[string]any
		bad   map[string]any
	}{
		{
			name:  "independent review",
			path:  filepath.Join("..", "..", "..", ".vrooli", "agent-manager", "independent-review.json"),
			valid: independentReviewResult("capture action unavailable"),
			bad:   independentReviewResult(""),
		},
		{
			name:  "milestone review",
			path:  filepath.Join("..", "..", "..", ".vrooli", "agent-manager", "milestone-review.json"),
			valid: milestoneReviewResult("capture action unavailable"),
			bad:   milestoneReviewResult(""),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			schema := compileWorkflowOutputSchema(t, tc.path)
			if err := schema.Validate(tc.valid); err != nil {
				t.Fatalf("valid output rejected: %v", err)
			}
			if err := schema.Validate(tc.bad); err == nil {
				t.Fatal("unavailable evidence without attempted_producer was accepted")
			}
		})
	}
}

func compileWorkflowOutputSchema(t *testing.T, path string) *jsonschema.Schema {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		OutputSchema json.RawMessage `json:"outputSchema"`
	}
	if err := json.Unmarshal(raw, &workflow); err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("memory://workflow-output.json", bytes.NewReader(workflow.OutputSchema)); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("memory://workflow-output.json")
	if err != nil {
		t.Fatal(err)
	}
	return schema
}

func unavailableEvidence(reason string) map[string]any {
	return map[string]any{
		"id": "evidence-1", "criterion_id": "criterion-1", "type": "cli_output", "title": "capture unavailable", "description": "The registered producer could not run.", "producer": "swarm-review", "trust": "reported", "settlement": "unavailable", "unavailable_reason": "producer unavailable", "attempted_producer": reason,
	}
}

func independentReviewResult(attemptedProducer string) map[string]any {
	return map[string]any{"result": map[string]any{"outcome": "accepted", "handoff": map[string]any{
		"verdict": "ready", "agent_assessment": "Evidence is complete.", "evidence": []any{unavailableEvidence(attemptedProducer)}, "criterion_verdicts": []any{map[string]any{"criterion_id": "criterion-1", "settlement": "unavailable", "evidence_ids": []any{"evidence-1"}}}, "improvement_suggestions": []any{}, "regression_introduced": false, "notes": []any{}, "summary": "Review complete.", "disposition": map[string]any{"kind": "attention", "rationale": "Capture is unavailable.", "confidence": "low"},
	}}}
}

func milestoneReviewResult(attemptedProducer string) map[string]any {
	return map[string]any{"result": map[string]any{
		"verdict": "partial", "assessment": "Capture is unavailable.", "evidence": []any{unavailableEvidence(attemptedProducer)}, "criterion_verdicts": []any{map[string]any{"criterion": "criterion-1", "verdict": "unavailable", "evidence": []any{"evidence-1"}}}, "proposals": []any{},
	}}
}
