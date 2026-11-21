package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPublic6338SeededSheetNameDefectIsDetectedAndCorrected(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/6338.json")
	seededWorkflow := readFixture(t, "repairs/6338-seeded.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(seededWorkflow))); got != "9b13403cf55d1aa2988cb5c31bb631a7425941ea6e0c326352fa66ac12ea8e47" {
		t.Fatalf("seeded repair fixture hash changed: %s", got)
	}
	workflow, _, _, err := normalizeWorkflow(publicWorkflow)
	if err != nil {
		t.Fatalf("normalize public manual reminder: %v", err)
	}
	triggerID := ""
	for _, node := range workflow.Nodes {
		if node.Trigger == TriggerManual {
			triggerID = node.ID
		}
	}
	if triggerID == "" {
		t.Fatal("public manual trigger is missing")
	}
	profile := readFixture(t, "public-workflow-6338-profile.json")
	now := time.Date(2026, 11, 12, 8, 30, 0, 0, time.UTC)
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return now })
	failed, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Public reminder with seeded wrong sheet", WorkflowJSON: seededWorkflow, TriggerNodeID: triggerID, SimulatorProfile: "public-workflow-6338-profile", SimulatorSourceJSON: profile})
	if err != nil {
		t.Fatalf("run seeded sheet-name defect: %v", err)
	}
	if failed.State != StateFailed || !strings.Contains(failed.FaultSummary, "synthetic HTTP 404") || traceHasOperation(failed.Trace, "gmail:message.send") {
		t.Fatalf("wrong sheet name must fail before mail delivery: %#v", failed)
	}
	repaired, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Corrected public reminder sheet name", WorkflowJSON: publicWorkflow, TriggerNodeID: triggerID, SimulatorProfile: "public-workflow-6338-profile", SimulatorSourceJSON: profile, RepairOfID: failed.ID})
	if err != nil {
		t.Fatalf("run corrected workflow revision: %v", err)
	}
	if repaired.State != StateNeedsReview || !traceHasOperation(repaired.Trace, "googleSheets:sheet.read") || !traceHasOperation(repaired.Trace, "gmail:message.send") || !traceHasOperation(repaired.Trace, "repair lineage") {
		t.Fatalf("corrected public reminder lacks successful synthetic handoff and lineage: %#v", repaired)
	}
}
