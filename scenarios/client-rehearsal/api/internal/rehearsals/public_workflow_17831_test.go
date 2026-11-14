package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPublic17831ProjectIntakeRunsManualAndWebhookThroughSyntheticServices(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/17831.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(workflowJSON))); got != "8184b1c054300842c0dd144eaa137a237b9c017badb537ebc249614bb1eff56a" {
		t.Fatalf("sanitized fixture hash changed without provenance update: %s", got)
	}
	for _, forbidden := range []string{"gmail.com", "google.com", "+1234567890", "Entreprise Démo"} {
		if strings.Contains(workflowJSON, forbidden) {
			t.Fatalf("sanitized public graph retained source data %q", forbidden)
		}
	}
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("import sanitized public project-intake graph: %v", err)
	}
	if len(workflow.Nodes) != 9 || len(workflow.EntryNodeIDs) != 2 || len(workflow.TriggerKinds) != 2 {
		t.Fatalf("unexpected public project-intake graph shape: nodes=%d entries=%v triggers=%v", len(workflow.Nodes), workflow.EntryNodeIDs, workflow.TriggerKinds)
	}
	triggerIDs := map[TriggerKind]string{}
	for index, kind := range workflow.TriggerKinds {
		triggerIDs[kind] = workflow.EntryNodeIDs[index]
	}
	if triggerIDs[TriggerManual] == "" || triggerIDs[TriggerWebhook] == "" {
		t.Fatalf("expected public manual and webhook entries, got %v", workflow.TriggerKinds)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "public-workflow-17831-profile.json"), false)
	if err != nil {
		t.Fatalf("load generic synthetic Sheets and Gmail profile: %v", err)
	}
	now := time.Date(2026, 11, 12, 15, 4, 5, 0, time.UTC)
	manual, err := executeDetailedWithInput(context.Background(), workflow, triggerIDs[TriggerManual], simulator, false, now, nil)
	if err != nil {
		t.Fatalf("run public manual project-intake entry: %v trace=%#v", err, manual.Trace)
	}
	webhookInput := map[string]any{"body": map[string]any{
		"client_name": "Synthetic Client", "email": "client@synthetic.invalid", "project_name": "Synthetic Workflow Project",
		"project_type": "Automation", "phone": "+10000000000", "deadline": "2026-11-30", "message": "Synthetic project intake.",
	}, "source": "synthetic_webhook"}
	webhook, err := executeDetailedWithInput(context.Background(), workflow, triggerIDs[TriggerWebhook], simulator, false, now, webhookInput)
	if err != nil {
		t.Fatalf("run public webhook project-intake entry: %v trace=%#v", err, webhook.Trace)
	}
	for trigger, run := range map[string]ExecutionResult{"manual": manual, "webhook": webhook} {
		for _, operation := range []string{"googleSheets:sheet.append", "gmail:message.send"} {
			if !traceHasSuccessfulOperation(run.Trace, operation) {
				t.Errorf("%s entry did not produce synthetic %s: %#v", trigger, operation, run.Trace)
			}
		}
		if strings.Contains(strings.Join(traceSummaries(run.Trace), " "), "google.com") || strings.Contains(strings.Join(traceSummaries(run.Trace), " "), "gmail.com") {
			t.Errorf("%s trace retained a real service destination", trigger)
		}
	}
}
