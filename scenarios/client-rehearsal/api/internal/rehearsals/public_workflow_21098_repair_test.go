package rehearsals

import (
	"strings"
	"testing"
	"time"
)

func TestPublic21098SeededWrongSheetDefectFailsBeforeEffectsAndHandoffRepairs(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/21098.json")
	seededWorkflow := readFixture(t, "repairs/21098-seeded.json")
	requireRepairFixtureHash(t, "sanitized public", publicWorkflow, "ef2efa5c177289695c072f0880b93ee7816f565214922cab821f001d95d52f88")
	requireRepairFixtureHash(t, "seeded defect", seededWorkflow, "add9a346da66b1c89431ba3bda8821bbcae2647ae953a3242989665f6dac0052")
	publicGraph, publicRevision, _, err := normalizeWorkflow(publicWorkflow)
	if err != nil {
		t.Fatal(err)
	}
	seededGraph, seededRevision, _, err := normalizeWorkflow(seededWorkflow)
	if err != nil {
		t.Fatal(err)
	}
	if publicRevision == seededRevision || seededGraph.Nodes[3].Request.Input["sheetName"] == publicGraph.Nodes[3].Request.Input["sheetName"] {
		t.Fatal("seeded wrong-sheet mutation must create a distinct, selector-visible workflow revision")
	}
	triggerID := requireWorkflowTriggerID(t, publicGraph, TriggerWebhook, "")
	profile := readFixture(t, "public-workflow-21098-catalog.json")
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return time.Date(2026, 11, 16, 7, 0, 0, 0, time.UTC) })
	input := map[string]any{"body": map[string]any{"name": "Synthetic Lead", "email": "lead@example.invalid", "message": "Made-up repair input", "source": "synthetic form"}}
	failed, corrected := runPublicWorkflowRepairPair(t, service, CreateInput{Title: "Public lead intake with seeded wrong-sheet selector", WorkflowJSON: seededWorkflow, TriggerNodeID: triggerID, TriggerInput: input, SimulatorProfile: "public-workflow-21098-catalog", SimulatorSourceJSON: profile}, publicWorkflow, "Corrected public lead intake sheet selector")
	if !strings.Contains(failed.FaultSummary, "synthetic HTTP 404") || traceHasOperation(failed.Trace, "slack:message.post") || traceHasOperation(failed.Trace, "gmail:message.send") {
		t.Fatalf("wrong Sheets selector must fail before Slack/Gmail effects: %#v", failed)
	}
	if !traceHasOperation(corrected.Trace, "googleSheets:sheet.append") || !traceHasOperation(corrected.Trace, "slack:message.post") || !traceHasOperation(corrected.Trace, "gmail:message.send") || !strings.Contains(corrected.HandoffMarkdown, "Human review is required") {
		t.Fatalf("corrected workflow lacks synthetic service results and repair handoff: %#v", corrected)
	}
}
