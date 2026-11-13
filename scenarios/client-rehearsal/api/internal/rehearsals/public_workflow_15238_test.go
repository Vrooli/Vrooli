package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPublic15238PromoExpiryRunsWithSyntheticHandoffs(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/15238.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(workflowJSON))); got != "3cfd4622b00018055edc6b2617ac95600b99eeac5b531e47488311e82d7d15d8" {
		t.Fatalf("sanitized fixture hash changed without provenance update: %s", got)
	}
	for _, forbidden := range []string{"C09S57E2JQ2", "gmail.com", "google.com", "1KiQqP-eWsyCKWZa1X07rV0eGhk93TXC30uS9POyNGZQ"} {
		if strings.Contains(workflowJSON, forbidden) {
			t.Fatalf("sanitized public graph retained source destination %q", forbidden)
		}
	}
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("import sanitized public promo workflow: %v", err)
	}
	if len(workflow.Nodes) != 10 || len(workflow.EntryNodeIDs) != 1 || workflow.TriggerKinds[0] != TriggerWebhook {
		t.Fatalf("unexpected public promo graph shape: nodes=%d entries=%v triggers=%v", len(workflow.Nodes), workflow.EntryNodeIDs, workflow.TriggerKinds)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "public-workflow-15238-profile.json"), false)
	if err != nil {
		t.Fatalf("load synthetic Sheets, Gmail, and Slack profile: %v", err)
	}
	input := map[string]any{"body": map[string]any{
		"code": "SYNTHETIC-20", "discount_type": "percent", "value": 20,
		"expiry": "2026-11-14T00:00:00Z", "usage_limit": 50,
	}}
	now := time.Date(2026, 11, 12, 15, 4, 5, 0, time.UTC)
	run, err := executeDetailedWithInput(context.Background(), workflow, workflow.EntryNodeIDs[0], simulator, false, now, input)
	if err != nil {
		t.Fatalf("run public promo graph with synthetic input: %v trace=%#v", err, run.Trace)
	}
	for _, operation := range []string{"googleSheets:sheet.append", "gmail:message.send", "slack:message.post"} {
		if !traceHasSuccessfulOperation(run.Trace, operation) {
			t.Errorf("expected synthetic operation %q in trace: %#v", operation, run.Trace)
		}
	}
	if strings.Contains(strings.Join(traceSummaries(run.Trace), " "), "C09S57E2JQ2") {
		t.Fatal("trace retained source Slack channel id")
	}
}
