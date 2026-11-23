package rehearsals

import (
	"testing"
	"time"
)

func TestPublic16364SeededSheetSelectionFailureIsCorrectedWithLineage(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/16364.json")
	seededWorkflow := readFixture(t, "repairs/16364-seeded.json")
	requireRepairFixtureHash(t, "public", publicWorkflow, "19217657f3142431844d90963efe9354ecac7d4eace685f4c7827ac5164cc295")
	requireRepairFixtureHash(t, "seeded", seededWorkflow, "3216c935bc503b4f8b23aeef152ae619a2f644a4b2ed45af8e3670e403f2d3c0")
	workflow, _, _, err := normalizeWorkflow(publicWorkflow)
	if err != nil {
		t.Fatalf("normalize public workflow: %v", err)
	}
	trigger := requireWorkflowTriggerID(t, workflow, TriggerScheduled, "")
	_, publicHash, _, err := normalizeWorkflow(publicWorkflow)
	if err != nil {
		t.Fatal(err)
	}
	_, seededHash, _, err := normalizeWorkflow(seededWorkflow)
	if err != nil {
		t.Fatal(err)
	}
	if publicHash == seededHash {
		t.Fatal("Sheets selector mutation did not change workflow revision fingerprint")
	}
	profile := readFixture(t, "public_workflow_16364_catalog.json")
	now := time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC)
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return now })
	failed, corrected := runPublicWorkflowRepairPair(t, service, CreateInput{Title: "Daily newsletter seeded missing-sheet selection", WorkflowJSON: seededWorkflow, TriggerNodeID: trigger, SimulatorProfile: "public-workflow-16364-catalog", SimulatorSourceJSON: profile}, publicWorkflow, "Corrected daily newsletter sheet selection")
	if traceHasOperation(failed.Trace, "gmail:message.send") {
		t.Fatalf("failed sheet selection reached Gmail: %#v", failed.Trace)
	}
	if !traceHasOperation(corrected.Trace, "googleSheets:sheet.read") || !traceHasOperation(corrected.Trace, "gmail:message.send") {
		t.Fatalf("corrected public revision lacks safe repair handoff and lineage: %#v", corrected)
	}
}
