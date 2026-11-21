package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPublic16904SeededDocumentSelectionDefectIsDetectedAndCorrected(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/16904.json")
	seededWorkflow := readFixture(t, "repairs/16904-seeded.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(seededWorkflow))); got != "3421fe2a7f0c3ca4aba165b277c95218e67453b8b124e99223daea9cfe0eb549" {
		t.Fatalf("seeded repair fixture hash changed: %s", got)
	}
	workflow, _, _, err := normalizeWorkflow(publicWorkflow)
	if err != nil {
		t.Fatalf("normalize public webhook attendance workflow: %v", err)
	}
	triggerID := ""
	for _, node := range workflow.Nodes {
		if node.Trigger == TriggerWebhook {
			triggerID = node.ID
		}
	}
	if triggerID == "" {
		t.Fatal("public webhook trigger is missing")
	}
	profile := readFixture(t, "public-workflow-16904-profile.json")
	input := map[string]any{"body": map[string]any{
		"name": "Synthetic Attendee", "email": "new-attendee@synthetic.invalid", "role": "Participant", "eventId": "synthetic-event-1", "day": "Day 1",
	}}
	now := time.Date(2026, 11, 12, 15, 4, 5, 0, time.UTC)
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return now })
	failed, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Public attendance with seeded wrong document", WorkflowJSON: seededWorkflow, TriggerNodeID: triggerID, TriggerInput: input, SimulatorProfile: "public-workflow-16904-profile", SimulatorSourceJSON: profile})
	if err != nil {
		t.Fatalf("run seeded document-selection defect: %v", err)
	}
	if failed.State != StateFailed || !strings.Contains(failed.FaultSummary, "synthetic HTTP 404") || traceHasOperation(failed.Trace, "googleSheets:sheet.append") || traceHasOperation(failed.Trace, "gmail:message.send") {
		t.Fatalf("wrong spreadsheet selection must fail before writes or mail: %#v", failed)
	}
	repaired, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Corrected public attendance document", WorkflowJSON: publicWorkflow, TriggerNodeID: triggerID, TriggerInput: input, SimulatorProfile: "public-workflow-16904-profile", SimulatorSourceJSON: profile, RepairOfID: failed.ID})
	if err != nil {
		t.Fatalf("run corrected workflow revision: %v", err)
	}
	if repaired.State != StateNeedsReview || !traceHasOperation(repaired.Trace, "googleSheets:sheet.read") || !traceHasOperation(repaired.Trace, "googleSheets:sheet.append") || !traceHasOperation(repaired.Trace, "gmail:message.send") || !traceHasOperation(repaired.Trace, "repair lineage") {
		t.Fatalf("corrected public attendance lacks successful synthetic handoff and lineage: %#v", repaired)
	}
}
