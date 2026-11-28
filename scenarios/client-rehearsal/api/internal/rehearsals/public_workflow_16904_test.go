package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPublic16904WebhookAttendanceUsesGenericSheetsAndMail(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/16904.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(workflowJSON))); got != "dbef2a8dac9eab999cdddd0e5be413c2de5d7b454329b3803160fb906b55e704" {
		t.Fatalf("sanitized fixture hash changed without provenance update: %s", got)
	}
	for _, forbidden := range []string{"1fzBymGE1OozTAxpor_KeAo8COsFZ1lEHDo3C69rOnCk", "gmail.com", "google.com"} {
		if strings.Contains(workflowJSON, forbidden) {
			t.Fatalf("sanitized public graph retained source destination %q", forbidden)
		}
	}
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("import sanitized public attendance workflow: %v", err)
	}
	if len(workflow.Nodes) != 11 || len(workflow.EntryNodeIDs) != 1 || workflow.TriggerKinds[0] != TriggerWebhook {
		t.Fatalf("unexpected public attendance graph shape: nodes=%d entries=%v triggers=%v", len(workflow.Nodes), workflow.EntryNodeIDs, workflow.TriggerKinds)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "public-workflow-16904-profile.json"), false)
	if err != nil {
		t.Fatalf("load synthetic Sheets and Gmail profile: %v", err)
	}
	input := map[string]any{"body": map[string]any{
		"name": "Synthetic Attendee", "email": "new-attendee@synthetic.invalid", "role": "Participant", "eventId": "synthetic-event-1", "day": "Day 1",
	}}
	now := time.Date(2026, 11, 12, 15, 4, 5, 0, time.UTC)
	run, err := executeDetailedWithInput(context.Background(), workflow, workflow.EntryNodeIDs[0], simulator, false, now, input)
	if err != nil {
		t.Fatalf("run public attendance graph with synthetic input: %v trace=%#v", err, run.Trace)
	}
	for _, operation := range []string{"googleSheets:sheet.read", "googleSheets:sheet.append", "gmail:message.send"} {
		if !traceHasSuccessfulOperation(run.Trace, operation) {
			t.Errorf("expected synthetic operation %q in trace: %#v", operation, run.Trace)
		}
	}
	if strings.Contains(strings.Join(traceSummaries(run.Trace), " "), "gmail.com") {
		t.Fatal("trace retained a non-synthetic mail destination")
	}
}
