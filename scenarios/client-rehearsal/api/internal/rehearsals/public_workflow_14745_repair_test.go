package rehearsals

import (
	"strings"
	"testing"
	"time"
)

func TestPublic14745SeededWaitSelectorDefectIsDetectedAndCorrected(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/14745.json")
	seededWorkflow := readFixture(t, "repairs/14745-seeded.json")
	requireRepairFixtureHash(t, "public", publicWorkflow, "79b6dbc7244701acc2d625e8146849087eb296e6477dd29ac51915ee75f0c147")
	requireRepairFixtureHash(t, "seeded", seededWorkflow, "7816288c4d9aee7199c4b7cd297830cfc00a6cf62269441c01dd7578cbedbe7f")
	workflow, _, _, err := normalizeWorkflow(publicWorkflow)
	if err != nil {
		t.Fatalf("normalize public workflow: %v", err)
	}
	triggerID := requireWorkflowTriggerID(t, workflow, TriggerScheduled, "")
	profile := readFixture(t, "public-workflow-profiles.json")
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return time.Date(2026, 11, 11, 0, 30, 0, 0, time.UTC) })
	failed, corrected := runPublicWorkflowRepairPair(t, service, CreateInput{Title: "Public Supabase keepalive with seeded Wait selector defect", WorkflowJSON: seededWorkflow, TriggerNodeID: triggerID, SimulatorProfile: "public-workflow-profiles", SimulatorSourceJSON: profile}, publicWorkflow, "Corrected public Supabase scheduled keepalive")
	if !strings.Contains(failed.FaultSummary, "amount path missingSeconds is missing") || traceHasOperation(failed.Trace, "row.create") {
		t.Fatalf("seeded virtual-Wait selector defect was not detected before service effects: %#v", failed)
	}
	if !traceHasNode(corrected.Trace, "Wait") || !traceHasOperation(corrected.Trace, "row.create") || !strings.Contains(corrected.HandoffMarkdown, "Human review is required") {
		t.Fatalf("corrected workflow did not produce a repair handoff: %#v", corrected)
	}
}
