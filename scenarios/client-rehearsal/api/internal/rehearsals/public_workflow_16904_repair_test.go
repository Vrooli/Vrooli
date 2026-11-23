package rehearsals

import (
	"strings"
	"testing"
	"time"
)

func TestPublic16904SeededDocumentSelectionDefectIsDetectedAndCorrected(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/16904.json")
	seededWorkflow := readFixture(t, "repairs/16904-seeded.json")
	requireRepairFixtureHash(t, "seeded repair", seededWorkflow, "3421fe2a7f0c3ca4aba165b277c95218e67453b8b124e99223daea9cfe0eb549")
	workflow, _, _, err := normalizeWorkflow(publicWorkflow)
	if err != nil {
		t.Fatalf("normalize public webhook attendance workflow: %v", err)
	}
	triggerID := requireWorkflowTriggerID(t, workflow, TriggerWebhook, "")
	profile := readFixture(t, "public-workflow-16904-profile.json")
	input := map[string]any{"body": map[string]any{
		"name": "Synthetic Attendee", "email": "new-attendee@synthetic.invalid", "role": "Participant", "eventId": "synthetic-event-1", "day": "Day 1",
	}}
	now := time.Date(2026, 11, 12, 15, 4, 5, 0, time.UTC)
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return now })
	failed, repaired := runPublicWorkflowRepairPair(t, service, CreateInput{Title: "Public attendance with seeded wrong document", WorkflowJSON: seededWorkflow, TriggerNodeID: triggerID, TriggerInput: input, SimulatorProfile: "public-workflow-16904-profile", SimulatorSourceJSON: profile}, publicWorkflow, "Corrected public attendance document")
	if !strings.Contains(failed.FaultSummary, "synthetic HTTP 404") || traceHasOperation(failed.Trace, "googleSheets:sheet.append") || traceHasOperation(failed.Trace, "gmail:message.send") {
		t.Fatalf("wrong spreadsheet selection must fail before writes or mail: %#v", failed)
	}
	if !traceHasOperation(repaired.Trace, "googleSheets:sheet.read") || !traceHasOperation(repaired.Trace, "googleSheets:sheet.append") || !traceHasOperation(repaired.Trace, "gmail:message.send") {
		t.Fatalf("corrected public attendance lacks successful synthetic handoff and lineage: %#v", repaired)
	}
}
