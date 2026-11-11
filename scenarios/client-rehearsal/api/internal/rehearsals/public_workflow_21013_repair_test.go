package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"
)

func TestPublic21013SeededValidationOutputDefectIsDetectedAndCorrected(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/21013.json")
	seededWorkflow := readFixture(t, "repairs/21013-seeded.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(publicWorkflow))); got != "f863ec05b5a2011622b1451b56f492c2282ef2c21a80523715fa12954104245c" {
		t.Fatalf("public fixture hash changed: %s", got)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(seededWorkflow))); got != "164a24007bdddc9e7a4c93454246bc7387d8432c1188488c76428c20f61be9c8" {
		t.Fatalf("seeded defect fixture hash changed: %s", got)
	}
	workflow, _, _, err := normalizeWorkflow(publicWorkflow)
	if err != nil {
		t.Fatalf("normalize public graph: %v", err)
	}
	triggerID := ""
	for _, node := range workflow.Nodes {
		if node.Trigger == TriggerWebhook {
			triggerID = node.ID
		}
	}
	if triggerID == "" {
		t.Fatal("public workflow webhook trigger missing")
	}
	profile := readFixture(t, "native_service_catalog.json")
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return time.Date(2026, 11, 10, 23, 20, 0, 0, time.UTC) })
	failed, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Public form intake with seeded validation-output defect", WorkflowJSON: seededWorkflow, TriggerNodeID: triggerID, SimulatorProfile: "native-service-catalog", SimulatorSourceJSON: profile, RequiredOutputPaths: []string{"valid"}})
	if err != nil {
		t.Fatalf("run seeded validation defect: %v", err)
	}
	if failed.State != StateFailed || failed.FaultSummary != "Required output paths missing: valid" {
		t.Fatalf("seeded output omission was not detected: %#v", failed)
	}
	corrected, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Corrected public form validation output", WorkflowJSON: publicWorkflow, TriggerNodeID: triggerID, SimulatorProfile: "native-service-catalog", SimulatorSourceJSON: profile, RequiredOutputPaths: []string{"valid"}, RepairOfID: failed.ID})
	if err != nil {
		t.Fatalf("run corrected public workflow: %v", err)
	}
	if corrected.State != StateNeedsReview || !traceHasNode(corrected.Trace, "Respond Error") || traceHasOperation(corrected.Trace, "googleSheets:sheet.append") || traceHasOperation(corrected.Trace, "gmail:message.send") || !traceHasOperation(corrected.Trace, "repair lineage") {
		t.Fatalf("corrected invalid-form path did not produce safe repair handoff: %#v", corrected)
	}
}
