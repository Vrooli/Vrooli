package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPublic14745SeededWaitSelectorDefectIsDetectedAndCorrected(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/14745.json")
	seededWorkflow := readFixture(t, "repairs/14745-seeded.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(publicWorkflow))); got != "79b6dbc7244701acc2d625e8146849087eb296e6477dd29ac51915ee75f0c147" {
		t.Fatalf("public fixture hash changed: %s", got)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(seededWorkflow))); got != "7816288c4d9aee7199c4b7cd297830cfc00a6cf62269441c01dd7578cbedbe7f" {
		t.Fatalf("seeded fixture hash changed: %s", got)
	}
	workflow, _, _, err := normalizeWorkflow(publicWorkflow)
	if err != nil {
		t.Fatalf("normalize public workflow: %v", err)
	}
	triggerID := ""
	for _, node := range workflow.Nodes {
		if node.Trigger == TriggerScheduled {
			triggerID = node.ID
		}
	}
	if triggerID == "" {
		t.Fatal("public scheduled trigger is missing")
	}
	profile := readFixture(t, "public-workflow-profiles.json")
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return time.Date(2026, 11, 11, 0, 30, 0, 0, time.UTC) })
	failed, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Public Supabase keepalive with seeded Wait selector defect", WorkflowJSON: seededWorkflow, TriggerNodeID: triggerID, SimulatorProfile: "public-workflow-profiles", SimulatorSourceJSON: profile})
	if err != nil {
		t.Fatalf("run seeded Wait defect: %v", err)
	}
	if failed.State != StateFailed || !strings.Contains(failed.FaultSummary, "amount path missingSeconds is missing") || traceHasOperation(failed.Trace, "row.create") {
		t.Fatalf("seeded virtual-Wait selector defect was not detected before service effects: %#v", failed)
	}
	corrected, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Corrected public Supabase scheduled keepalive", WorkflowJSON: publicWorkflow, TriggerNodeID: triggerID, SimulatorProfile: "public-workflow-profiles", SimulatorSourceJSON: profile, RepairOfID: failed.ID})
	if err != nil {
		t.Fatalf("run corrected public workflow: %v", err)
	}
	if corrected.State != StateNeedsReview || !traceHasOperation(corrected.Trace, "repair lineage") || !traceHasNode(corrected.Trace, "Wait") || !traceHasOperation(corrected.Trace, "row.create") || !strings.Contains(corrected.HandoffMarkdown, "Human review is required") {
		t.Fatalf("corrected workflow did not produce a repair handoff: %#v", corrected)
	}
}
