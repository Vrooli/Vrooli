package rehearsals

import (
	"testing"
	"time"
)

func TestPublic1804SeededFunctionOutputShapeFailureIsCorrectedWithLineage(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/1804.json")
	seededWorkflow := readFixture(t, "repairs/1804-seeded.json")
	requireRepairFixtureHash(t, "public", publicWorkflow, "f07ea4384b83db37b3f999bdfce143c83e477ebd3defff2e8f0f02a8eb377ce8")
	requireRepairFixtureHash(t, "seeded", seededWorkflow, "57d0f481de8cfe5f14cb77cf9401da4d653389400ceca62d32f30638c10e7ced")
	workflow, _, _, err := normalizeWorkflow(publicWorkflow)
	if err != nil {
		t.Fatalf("normalize public workflow: %v", err)
	}
	trigger := requireWorkflowTriggerID(t, workflow, TriggerPolling, "")
	profile := readFixture(t, "public_workflow_1804_catalog.json")
	now := time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC)
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return now })
	failed, corrected := runPublicWorkflowRepairPair(t, service, CreateInput{Title: "Public issue sync with seeded Function return-shape defect", WorkflowJSON: seededWorkflow, TriggerNodeID: trigger, SimulatorProfile: "public-workflow-1804-catalog", SimulatorSourceJSON: profile}, publicWorkflow, "Corrected public GitHub issue to Notion sync")
	if traceHasOperation(failed.Trace, "notion:databasePage.getAll") {
		t.Fatalf("wrong output shape reached Notion operations: trace=%#v", failed.Trace)
	}
	if !traceHasOperation(corrected.Trace, "notion:databasePage.getAll") || !traceHasOperation(corrected.Trace, "notion:databasePage.update") {
		t.Fatalf("corrected public revision lacks Notion update handoff and lineage: %#v", corrected)
	}
}
