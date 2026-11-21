package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPublic21098SeededWrongSheetDefectFailsBeforeEffectsAndHandoffRepairs(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/21098.json")
	seededWorkflow := readFixture(t, "repairs/21098-seeded.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(publicWorkflow))); got != "ef2efa5c177289695c072f0880b93ee7816f565214922cab821f001d95d52f88" {
		t.Fatalf("sanitized public fixture hash changed: %s", got)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(seededWorkflow))); got != "add9a346da66b1c89431ba3bda8821bbcae2647ae953a3242989665f6dac0052" {
		t.Fatalf("seeded defect fixture hash changed: %s", got)
	}
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
	triggerID := ""
	for _, node := range publicGraph.Nodes {
		if node.Trigger == TriggerWebhook {
			triggerID = node.ID
		}
	}
	if triggerID == "" {
		t.Fatal("sanitized public lead webhook trigger missing")
	}
	profile := readFixture(t, "public-workflow-21098-catalog.json")
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return time.Date(2026, 11, 16, 7, 0, 0, 0, time.UTC) })
	input := map[string]any{"body": map[string]any{"name": "Synthetic Lead", "email": "lead@example.invalid", "message": "Made-up repair input", "source": "synthetic form"}}
	failed, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Public lead intake with seeded wrong-sheet selector", WorkflowJSON: seededWorkflow, TriggerNodeID: triggerID, TriggerInput: input, SimulatorProfile: "public-workflow-21098-catalog", SimulatorSourceJSON: profile})
	if err != nil {
		t.Fatalf("run seeded selector defect: %v", err)
	}
	if failed.State != StateFailed || !strings.Contains(failed.FaultSummary, "synthetic HTTP 404") || traceHasOperation(failed.Trace, "slack:message.post") || traceHasOperation(failed.Trace, "gmail:message.send") {
		t.Fatalf("wrong Sheets selector must fail before Slack/Gmail effects: %#v", failed)
	}
	corrected, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Corrected public lead intake sheet selector", WorkflowJSON: publicWorkflow, TriggerNodeID: triggerID, TriggerInput: input, SimulatorProfile: "public-workflow-21098-catalog", SimulatorSourceJSON: profile, RepairOfID: failed.ID})
	if err != nil {
		t.Fatalf("run repaired public workflow: %v", err)
	}
	if corrected.State != StateNeedsReview || !traceHasOperation(corrected.Trace, "googleSheets:sheet.append") || !traceHasOperation(corrected.Trace, "slack:message.post") || !traceHasOperation(corrected.Trace, "gmail:message.send") || !traceHasOperation(corrected.Trace, "repair lineage") || !strings.Contains(corrected.HandoffMarkdown, "Human review is required") {
		t.Fatalf("corrected workflow lacks synthetic service results and repair handoff: %#v", corrected)
	}
}
