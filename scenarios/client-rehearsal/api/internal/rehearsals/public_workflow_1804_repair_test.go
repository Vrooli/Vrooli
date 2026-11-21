package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"
)

func TestPublic1804SeededFunctionOutputShapeFailureIsCorrectedWithLineage(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/1804.json")
	seededWorkflow := readFixture(t, "repairs/1804-seeded.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(publicWorkflow))); got != "f07ea4384b83db37b3f999bdfce143c83e477ebd3defff2e8f0f02a8eb377ce8" {
		t.Fatalf("public fixture hash changed: %s", got)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(seededWorkflow))); got != "57d0f481de8cfe5f14cb77cf9401da4d653389400ceca62d32f30638c10e7ced" {
		t.Fatalf("seeded fixture hash changed: %s", got)
	}
	workflow, _, _, err := normalizeWorkflow(publicWorkflow)
	if err != nil {
		t.Fatalf("normalize public workflow: %v", err)
	}
	trigger := ""
	for _, node := range workflow.Nodes {
		if node.Trigger == TriggerPolling {
			trigger = node.ID
		}
	}
	if trigger == "" {
		t.Fatal("GitHub polling trigger missing")
	}
	profile := readFixture(t, "public_workflow_1804_catalog.json")
	now := time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC)
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return now })
	failed, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Public issue sync with seeded Function return-shape defect", WorkflowJSON: seededWorkflow, TriggerNodeID: trigger, SimulatorProfile: "public-workflow-1804-catalog", SimulatorSourceJSON: profile})
	if err != nil {
		t.Fatalf("run seeded defect: %v", err)
	}
	if failed.State != StateFailed || traceHasOperation(failed.Trace, "notion:databasePage.getAll") {
		t.Fatalf("wrong output shape must fail before Notion operations: state=%s trace=%#v", failed.State, failed.Trace)
	}
	corrected, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Corrected public GitHub issue to Notion sync", WorkflowJSON: publicWorkflow, TriggerNodeID: trigger, SimulatorProfile: "public-workflow-1804-catalog", SimulatorSourceJSON: profile, RepairOfID: failed.ID})
	if err != nil {
		t.Fatalf("run corrected public workflow: %v", err)
	}
	if corrected.State != StateNeedsReview || !traceHasOperation(corrected.Trace, "notion:databasePage.getAll") || !traceHasOperation(corrected.Trace, "notion:databasePage.update") || !traceHasOperation(corrected.Trace, "repair lineage") {
		t.Fatalf("corrected public revision lacks Notion update handoff and lineage: %#v", corrected)
	}
}
