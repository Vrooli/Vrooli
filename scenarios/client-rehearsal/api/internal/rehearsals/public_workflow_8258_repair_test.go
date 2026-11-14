package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPublic8258SeededTableSelectorDefectIsDetectedAndCorrected(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/8258.json")
	seededWorkflow := readFixture(t, "repairs/8258-seeded.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(publicWorkflow))); got != "debf15d6e30439f3b75574523e436cccfc97cd5fa71f139604d00b8517e55f7c" {
		t.Fatalf("public fixture hash changed: %s", got)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(seededWorkflow))); got != "4f0170197217331231198459766f06fa11bfa8a254811239ea3f4b383166b1b7" {
		t.Fatalf("seeded fixture hash changed: %s", got)
	}
	workflow, _, _, err := normalizeWorkflow(publicWorkflow)
	if err != nil {
		t.Fatalf("normalize public graph: %v", err)
	}
	triggerID := ""
	for _, node := range workflow.Nodes {
		if node.Name == "Webhook" && node.Trigger == TriggerWebhook {
			triggerID = node.ID
		}
	}
	if triggerID == "" {
		t.Fatal("public read-row webhook trigger missing")
	}
	profile := readFixture(t, "public-workflow-profiles.json")
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return time.Date(2026, 11, 11, 0, 40, 0, 0, time.UTC) })
	failed, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Public Supabase read with seeded table selector defect", WorkflowJSON: seededWorkflow, TriggerNodeID: triggerID, SimulatorProfile: "public-workflow-supabase-catalog", SimulatorSourceJSON: profile})
	if err != nil {
		t.Fatalf("run seeded table selector defect: %v", err)
	}
	if failed.State != StateFailed || !strings.Contains(failed.FaultSummary, "synthetic HTTP 404") {
		t.Fatalf("unmatched synthetic table must fail closed: %#v", failed)
	}
	corrected, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Corrected public Supabase read-row workflow", WorkflowJSON: publicWorkflow, TriggerNodeID: triggerID, SimulatorProfile: "public-workflow-supabase-catalog", SimulatorSourceJSON: profile, RepairOfID: failed.ID})
	if err != nil {
		t.Fatalf("run corrected public graph: %v", err)
	}
	if corrected.State != StateNeedsReview || !traceHasOperation(corrected.Trace, "supabase:row.get") || !traceHasOperation(corrected.Trace, "repair lineage") || !strings.Contains(corrected.HandoffMarkdown, "Human review is required") {
		t.Fatalf("corrected public read did not produce a repair handoff: %#v", corrected)
	}
}
