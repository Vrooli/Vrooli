package rehearsals

import (
	"strings"
	"testing"
	"time"
)

func TestPublic8258SeededTableSelectorDefectIsDetectedAndCorrected(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/8258.json")
	seededWorkflow := readFixture(t, "repairs/8258-seeded.json")
	requireRepairFixtureHash(t, "public", publicWorkflow, "debf15d6e30439f3b75574523e436cccfc97cd5fa71f139604d00b8517e55f7c")
	requireRepairFixtureHash(t, "seeded", seededWorkflow, "4f0170197217331231198459766f06fa11bfa8a254811239ea3f4b383166b1b7")
	workflow, _, _, err := normalizeWorkflow(publicWorkflow)
	if err != nil {
		t.Fatalf("normalize public graph: %v", err)
	}
	triggerID := requireWorkflowTriggerID(t, workflow, TriggerWebhook, "Webhook")
	profile := readFixture(t, "public-workflow-profiles.json")
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return time.Date(2026, 11, 11, 0, 40, 0, 0, time.UTC) })
	failed, corrected := runPublicWorkflowRepairPair(t, service, CreateInput{Title: "Public Supabase read with seeded table selector defect", WorkflowJSON: seededWorkflow, TriggerNodeID: triggerID, SimulatorProfile: "public-workflow-supabase-catalog", SimulatorSourceJSON: profile}, publicWorkflow, "Corrected public Supabase read-row workflow")
	if !strings.Contains(failed.FaultSummary, "synthetic HTTP 404") {
		t.Fatalf("unmatched synthetic table must fail closed: %#v", failed)
	}
	if !traceHasOperation(corrected.Trace, "supabase:row.get") || !strings.Contains(corrected.HandoffMarkdown, "Human review is required") {
		t.Fatalf("corrected public read did not produce a repair handoff: %#v", corrected)
	}
}
