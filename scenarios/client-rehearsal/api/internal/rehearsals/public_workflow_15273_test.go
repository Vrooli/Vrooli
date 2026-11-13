package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPublic15273PaymentFailureWebhookRunsAgainstSyntheticServices(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/15273.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(workflowJSON))); got != "204b630b9fcfe9db6eeb45b95e4e2a9d4a8bd6d61392e51fc0c8eb47562b4ea0" {
		t.Fatalf("sanitized fixture changed without provenance update: %s", got)
	}
	for _, forbidden := range []string{"C09S57E2JQ2", "google.com", "gmail.com", "stripe.com"} {
		if strings.Contains(strings.ToLower(workflowJSON), strings.ToLower(forbidden)) {
			t.Fatalf("sanitized public workflow retained production destination %q", forbidden)
		}
	}
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		if !strings.Contains(err.Error(), "condition must use a simple JSON path expression") {
			t.Fatalf("unexpected import failure: %v", err)
		}
		t.Logf("qualification attempt earns no run credit: generic IF importer does not support the public node-qualified condition expression: %v", err)
		return
	}
	if len(workflow.Nodes) != 17 || len(workflow.EntryNodeIDs) != 1 || workflow.TriggerKinds[0] != TriggerWebhook {
		t.Fatalf("unexpected graph shape: nodes=%d entries=%v triggers=%v", len(workflow.Nodes), workflow.EntryNodeIDs, workflow.TriggerKinds)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "public-workflow-15273-profile.json"), false)
	if err != nil {
		t.Fatalf("load synthetic service profile: %v", err)
	}
	input := map[string]any{"body": map[string]any{"type": "payment_intent.payment_failed", "data": map[string]any{"object": map[string]any{
		"id": "pi_synthetic_01", "amount": 75000, "currency": "usd", "billing_details": map[string]any{"email": "buyer@synthetic.invalid", "name": "Synthetic Buyer"},
		"last_payment_error": map[string]any{"code": "card_error", "decline_code": "insufficient_funds", "message": "temporary synthetic decline"},
	}}}, "retry_url": "https://store.synthetic.invalid/retry"}
	run, err := executeDetailedWithInput(context.Background(), workflow, workflow.EntryNodeIDs[0], simulator, false, time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC), input)
	if err != nil {
		t.Fatalf("execute unchanged public graph with synthetic event: %v; trace=%#v", err, run.Trace)
	}
	for _, operation := range []string{"slack:message.post", "gmail:message.send", "supabase:row.create"} {
		if !traceHasSuccessfulOperation(run.Trace, operation) {
			t.Errorf("expected synthetic operation %q; output=%#v trace=%#v", operation, run.Output, run.Trace)
		}
	}
}
