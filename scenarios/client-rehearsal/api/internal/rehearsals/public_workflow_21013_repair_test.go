package rehearsals

import (
	"testing"
	"time"
)

func TestPublic21013SeededValidationOutputDefectIsDetectedAndCorrected(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/21013.json")
	seededWorkflow := readFixture(t, "repairs/21013-seeded.json")
	requireRepairFixtureHash(t, "public", publicWorkflow, "6086f7a9c84f37121399be67f4e61ef28213ff7728d8c5e9714efa5399b10fe9")
	requireRepairFixtureHash(t, "seeded defect", seededWorkflow, "0c2836016ff0092367a667a62d5a45302f13e85bcc5fa94e69f1ba48e357fc43")
	workflow, _, _, err := normalizeWorkflow(publicWorkflow)
	if err != nil {
		t.Fatalf("normalize public graph: %v", err)
	}
	triggerID := requireWorkflowTriggerID(t, workflow, TriggerWebhook, "")
	profile := readFixture(t, "native_service_catalog.json")
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return time.Date(2026, 11, 10, 23, 20, 0, 0, time.UTC) })
	failed, corrected := runPublicWorkflowRepairPair(t, service, CreateInput{Title: "Public form intake with seeded validation-output defect", WorkflowJSON: seededWorkflow, TriggerNodeID: triggerID, SimulatorProfile: "native-service-catalog", SimulatorSourceJSON: profile, RequiredOutputPaths: []string{"valid"}}, publicWorkflow, "Corrected public form validation output")
	if failed.FaultSummary != "Required output paths missing: valid" {
		t.Fatalf("seeded output omission was not detected: %#v", failed)
	}
	if !traceHasNode(corrected.Trace, "Respond Error") || traceHasOperation(corrected.Trace, "googleSheets:sheet.append") || traceHasOperation(corrected.Trace, "gmail:message.send") {
		t.Fatalf("corrected invalid-form path did not produce safe repair handoff: %#v", corrected)
	}
}
