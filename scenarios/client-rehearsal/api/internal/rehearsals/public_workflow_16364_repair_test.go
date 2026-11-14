package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"
)

func TestPublic16364SeededSheetSelectionFailureIsCorrectedWithLineage(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/16364.json")
	seededWorkflow := readFixture(t, "repairs/16364-seeded.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(publicWorkflow))); got != "19217657f3142431844d90963efe9354ecac7d4eace685f4c7827ac5164cc295" {
		t.Fatalf("public fixture hash changed: %s", got)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(seededWorkflow))); got != "3216c935bc503b4f8b23aeef152ae619a2f644a4b2ed45af8e3670e403f2d3c0" {
		t.Fatalf("seeded fixture hash changed: %s", got)
	}
	workflow, _, _, err := normalizeWorkflow(publicWorkflow)
	if err != nil {
		t.Fatalf("normalize public workflow: %v", err)
	}
	trigger := ""
	for _, node := range workflow.Nodes {
		if node.Trigger == TriggerScheduled {
			trigger = node.ID
		}
	}
	if trigger == "" {
		t.Fatal("scheduled trigger missing")
	}
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
	failed, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Daily newsletter seeded missing-sheet selection", WorkflowJSON: seededWorkflow, TriggerNodeID: trigger, SimulatorProfile: "public-workflow-16364-catalog", SimulatorSourceJSON: profile})
	if err != nil {
		t.Fatalf("run seeded defect: %v", err)
	}
	if failed.State != StateFailed {
		t.Fatalf("expected seeded sheet selection to fail: state=%s fault=%q", failed.State, failed.FaultSummary)
	}
	if traceHasOperation(failed.Trace, "gmail:message.send") {
		t.Fatalf("failed sheet selection reached Gmail: %#v", failed.Trace)
	}
	corrected, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Corrected daily newsletter sheet selection", WorkflowJSON: publicWorkflow, TriggerNodeID: trigger, SimulatorProfile: "public-workflow-16364-catalog", SimulatorSourceJSON: profile, RepairOfID: failed.ID})
	if err != nil {
		t.Fatalf("run corrected public revision: %v", err)
	}
	if corrected.State != StateNeedsReview || !traceHasOperation(corrected.Trace, "googleSheets:sheet.read") || !traceHasOperation(corrected.Trace, "gmail:message.send") || !traceHasOperation(corrected.Trace, "repair lineage") {
		t.Fatalf("corrected public revision lacks safe repair handoff and lineage: %#v", corrected)
	}
}
