package rehearsals

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPublic6920RunsAsGenericErrorWorkflowHandler(t *testing.T) {
	var handler map[string]any
	if err := json.Unmarshal([]byte(readFixture(t, "public-workflows/6920.json")), &handler); err != nil {
		t.Fatal(err)
	}
	parent := map[string]any{
		"id":       "synthetic-primary-flow",
		"name":     "Synthetic primary flow",
		"settings": map[string]any{"errorWorkflow": "public-template-6920"},
		"nodes": []any{
			map[string]any{"id": "start", "name": "Manual", "type": "n8n-nodes-base.manualTrigger", "parameters": map[string]any{}},
			map[string]any{"id": "operation", "name": "Synthetic primary operation", "type": "n8n-nodes-base.httpRequest", "parameters": map[string]any{"method": "GET", "url": "https://unmapped.synthetic.invalid/failure"}},
		},
		"connections": map[string]any{"Manual": map[string]any{"main": []any{[]any{map[string]any{"node": "Synthetic primary operation", "type": "main", "index": 0}}}}},
	}
	bundle, err := json.Marshal(map[string]any{
		"entryWorkflowId": "synthetic-primary-flow",
		"workflows":       []any{parent, handler},
	})
	if err != nil {
		t.Fatal(err)
	}
	workflow, _, _, err := normalizeWorkflow(string(bundle))
	if err != nil {
		t.Fatalf("import sanitized public error handler: %v", err)
	}
	if workflow.ErrorWorkflowID != "public-template-6920" {
		t.Fatalf("error handler reference not bound: %#v", workflow)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "native_service_catalog.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	trace, runErr := (SyntheticEngine{}).Execute(context.Background(), workflow, workflow.EntryNodeIDs[0], simulator, false, time.Date(2026, 11, 10, 13, 0, 0, 0, time.UTC))
	if runErr == nil || !strings.Contains(runErr.Error(), "no matching synthetic API operation") {
		t.Fatalf("expected seeded primary failure to remain visible, got %v trace=%#v", runErr, trace)
	}
	if !traceHasSuccessfulOperation(trace, "public-template-6920/Send a message: n8n-nodes-base.gmail:message.send") {
		t.Fatalf("public error handler did not execute through generic synthetic Gmail binding: %#v", trace)
	}
}
