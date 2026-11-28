package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPublic1110WebhookToSyntheticNotionRunsUnmodifiedSanitizedGraph(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/1110.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(workflowJSON))); got != "fd1bcb27c567df7402299bb5687f1275a4d2fe7f8d3dababcc25aa8d0fd8e20d" {
		t.Fatalf("sanitized fixture changed without provenance update: %s", got)
	}
	for _, forbidden := range []string{"discord.com", "notion.so", "google.com", "gmail.com", "C09"} {
		if strings.Contains(strings.ToLower(workflowJSON), strings.ToLower(forbidden)) {
			t.Fatalf("sanitized public workflow retained production destination %q", forbidden)
		}
	}
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("import unchanged sanitized public graph: %v", err)
	}
	if len(workflow.Nodes) != 7 || len(workflow.EntryNodeIDs) != 1 || workflow.TriggerKinds[0] != TriggerWebhook {
		t.Fatalf("unexpected public graph shape: nodes=%d entries=%v triggers=%v", len(workflow.Nodes), workflow.EntryNodeIDs, workflow.TriggerKinds)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "public-workflow-1110-profile.json"), false)
	if err != nil {
		t.Fatalf("load bounded synthetic service profile: %v", err)
	}
	input := map[string]any{"body": map[string]any{
		"type": 2,
		"data": map[string]any{"options": []any{map[string]any{"value": "article.synthetic.invalid/story"}}},
	}}
	run, err := executeDetailedWithInput(context.Background(), workflow, workflow.EntryNodeIDs[0], simulator, false, time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC), input)
	if err != nil {
		t.Fatalf("execute unmodified public graph with bounded synthetic webhook event: %v; trace=%#v", err, run.Trace)
	}
	for _, node := range []string{"Webhook", "Check type", "HTTP Request", "Extract Title", "Add Link to Notion", "Reply on Discord"} {
		if !traceHasNode(run.Trace, node) {
			t.Errorf("expected full selected path to include %q: %#v", node, run.Trace)
		}
	}
	if !traceHasSuccessfulOperation(run.Trace, "GET /story") || !traceHasSuccessfulOperation(run.Trace, "notion:databasePage.create") {
		t.Fatalf("expected synthetic article read and Notion create: %#v", run.Trace)
	}
	for _, sourceValue := range []string{"Source title must not persist", "Source body must not persist"} {
		if strings.Contains(fmt.Sprint(run.Output), sourceValue) || strings.Contains(fmt.Sprint(run.Trace), sourceValue) {
			t.Fatalf("source HTML value survived synthetic response redaction: %q", sourceValue)
		}
	}
}
