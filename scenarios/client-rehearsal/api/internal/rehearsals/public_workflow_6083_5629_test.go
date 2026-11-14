package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPublic6083LeadFollowupImportGapIsRecorded(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/6083.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(workflowJSON))); got != "81c4cbb7a1ed9039bb808405e863970be3471cc0541757da38714a6b64554d5e" {
		t.Fatalf("sanitized fixture hash changed without provenance update: %s", got)
	}
	for _, forbidden := range []string{"gmail.com", "google.com", "621412350", "C09S57E2JQ2"} {
		if strings.Contains(workflowJSON, forbidden) {
			t.Fatalf("sanitized public graph retained source destination %q", forbidden)
		}
	}
	_, _, _, err := normalizeWorkflow(workflowJSON)
	if err == nil || !strings.Contains(err.Error(), "first graph adapter supports exactly one condition per If node") {
		t.Fatalf("expected unsupported multi-condition If import to remain unqualified until generic support is coordinated, got %v", err)
	}
}

func TestPublic5629ErrorHandlerRetainsGenericCodeOutputBoundary(t *testing.T) {
	childJSON := readFixture(t, "public-workflows/5629.json")
	parentBundle := fmt.Sprintf(`{"entryWorkflowId":"synthetic-primary","workflows":[{"id":"synthetic-primary","name":"Synthetic primary flow","settings":{"errorWorkflow":"public-template-5629"},"nodes":[{"id":"start","name":"Manual","type":"n8n-nodes-base.manualTrigger","parameters":{}},{"id":"operation","name":"Synthetic service operation","type":"n8n-nodes-base.httpRequest","parameters":{"method":"GET","url":"https://unmapped.synthetic.invalid/failure"}}],"connections":{"Manual":{"main":[[{"node":"Synthetic service operation","type":"main","index":0}]]}}},%s]}`, childJSON)
	workflow, _, _, err := normalizeWorkflow(parentBundle)
	if err != nil {
		t.Fatalf("normalize synthetic parent and unchanged public error workflow: %v", err)
	}
	if workflow.ErrorWorkflowID != "public-template-5629" || workflow.Subworkflows[workflow.ErrorWorkflowID] == nil {
		t.Fatalf("expected public graph attached as the configured error workflow: %#v", workflow)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "public-workflow-5852-profile.json"), false)
	if err != nil {
		t.Fatalf("load bounded synthetic notification profile: %v", err)
	}
	handler := workflow.Subworkflows[workflow.ErrorWorkflowID]
	if handler.ErrorNodeID == "" {
		t.Fatal("expected retained public graph to expose its generic Error Trigger entry")
	}
	input := map[string]any{
		"workflow": map[string]any{"name": "Synthetic account sync"},
		"execution": map[string]any{
			"id": "synthetic-execution-5629", "url": "https://n8n.synthetic.invalid/executions/5629",
			"error": map[string]any{
				"node":    map[string]any{"name": "Synthetic CRM upsert"},
				"message": "synthetic service returned 503", "description": "retry after service recovery",
				"timestamp": "2026-11-12T12:00:00Z", "level": "error",
			},
		},
	}
	trace := []TraceEvent{}
	_, err = executeGraph(context.Background(), *handler, handler.ErrorNodeID, input, simulator, false, time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC), &trace, map[string]bool{}, 0)
	if err == nil || !strings.Contains(err.Error(), "single Code output must contain one bounded json object") {
		t.Fatalf("expected unchanged Code node's legacy object return to fail closed without adaptation, got %v", err)
	}
	if !traceHasNode(trace, "Error Trigger") || traceHasOperation(trace, "telegram:message.sendMessage") || traceHasOperation(trace, "gmail:message.send") {
		t.Fatalf("failed Code node must not reach a notification effect: %#v", trace)
	}
}

func TestPublic5629ImportsAfterSkippingDisabledNotificationNodes(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/5629.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(workflowJSON))); got != "277315752d93f302ac06e98ca2215f2a6fea0762abaff1f957ca234f27da2c1d" {
		t.Fatalf("sanitized fixture hash changed without provenance update: %s", got)
	}
	for _, forbidden := range []string{"621412350", "gmail.com"} {
		if strings.Contains(workflowJSON, forbidden) {
			t.Fatalf("sanitized public error graph retained source destination %q", forbidden)
		}
	}
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("normalize unchanged workflow with disabled unsupported node: %v", err)
	}
	for _, node := range workflow.Nodes {
		if node.Name == "Notify Whatsapp" {
			t.Fatal("disabled WhatsApp node must be omitted before node-type adapter lookup")
		}
	}
}
