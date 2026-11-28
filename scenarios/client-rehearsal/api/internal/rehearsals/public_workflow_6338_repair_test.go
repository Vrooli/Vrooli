package rehearsals

import (
	"strings"
	"testing"
	"time"
)

func TestPublic6338SeededSheetNameDefectIsDetectedAndCorrected(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/6338.json")
	seededWorkflow := readFixture(t, "repairs/6338-seeded.json")
	requireRepairFixtureHash(t, "seeded repair", seededWorkflow, "9b13403cf55d1aa2988cb5c31bb631a7425941ea6e0c326352fa66ac12ea8e47")
	workflow, _, _, err := normalizeWorkflow(publicWorkflow)
	if err != nil {
		t.Fatalf("normalize public manual reminder: %v", err)
	}
	triggerID := requireWorkflowTriggerID(t, workflow, TriggerManual, "")
	profile := readFixture(t, "public-workflow-6338-profile.json")
	now := time.Date(2026, 11, 12, 8, 30, 0, 0, time.UTC)
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return now })
	failed, repaired := runPublicWorkflowRepairPair(t, service, CreateInput{Title: "Public reminder with seeded wrong sheet", WorkflowJSON: seededWorkflow, TriggerNodeID: triggerID, SimulatorProfile: "public-workflow-6338-profile", SimulatorSourceJSON: profile}, publicWorkflow, "Corrected public reminder sheet name")
	if !strings.Contains(failed.FaultSummary, "synthetic HTTP 404") || traceHasOperation(failed.Trace, "gmail:message.send") {
		t.Fatalf("wrong sheet name must fail before mail delivery: %#v", failed)
	}
	if !traceHasOperation(repaired.Trace, "googleSheets:sheet.read") || !traceHasOperation(repaired.Trace, "gmail:message.send") {
		t.Fatalf("corrected public reminder lacks successful synthetic handoff and lineage: %#v", repaired)
	}
}
