package orchestration

import (
	"encoding/json"
	"testing"

	"agent-manager/internal/domain"
)

func TestWorkflowVerdict_ExtractsDeclaredJSONPointer(t *testing.T) {
	verdict, ok := workflowVerdict(json.RawMessage(`{"evaluation":{"verdict":"pass"}}`), "/evaluation/verdict")
	if !ok || verdict != "pass" {
		t.Fatalf("verdict=%q ok=%v", verdict, ok)
	}
	if _, ok := workflowVerdict(json.RawMessage(`{"evaluation":{"verdict":true}}`), "/evaluation/verdict"); ok {
		t.Fatal("non-string verdict must be incomplete")
	}
	if _, ok := workflowVerdict(json.RawMessage(`{"evaluation":{}}`), "/evaluation/verdict"); ok {
		t.Fatal("missing verdict must be incomplete")
	}
}

func TestOutcomeStatus_RequiresSuccessfulStructuredVerdict(t *testing.T) {
	if got := outcomeStatus(&domain.StructuredResult{Status: domain.StructuredResultSuccess}, "pass"); got != "complete" {
		t.Fatalf("status=%q", got)
	}
	if got := outcomeStatus(&domain.StructuredResult{Status: domain.StructuredResultInvalid}, "pass"); got != "incomplete" {
		t.Fatalf("status=%q", got)
	}
}
