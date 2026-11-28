package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

// Exercise the unchanged public graph through generic bounded Code, feedback
// loop, dynamic Wait, and synthetic Supabase semantics. Keep malformed loops
// and out-of-subset code fail-closed in the adjacent tests.
func TestPublic14745RunsBoundedScheduledAndManualLoopAgainstSyntheticSupabase(t *testing.T) {
	fixture := readFixture(t, "public-workflows/14745.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(fixture))); got != "79b6dbc7244701acc2d625e8146849087eb296e6477dd29ac51915ee75f0c147" {
		t.Fatalf("sanitized public fixture changed without provenance update: %s", got)
	}
	workflow, _, _, err := normalizeWorkflow(fixture)
	if err != nil {
		t.Fatalf("normalize unchanged sanitized public workflow: %v", err)
	}
	profile := readFixture(t, "public-workflow-profiles.json")
	simulator, err := NewDescriptionSimulator(profile, false)
	if err != nil {
		t.Fatalf("load synthetic service descriptions: %v", err)
	}
	var manualID, scheduleID string
	for _, node := range workflow.Nodes {
		switch node.Trigger {
		case TriggerManual:
			manualID = node.ID
		case TriggerScheduled:
			scheduleID = node.ID
		case "":
			if node.Type == "n8n-nodes-base.splitInBatches" && node.LoopBatchSize != 1 {
				t.Fatalf("public Loop Over Items defaults to single-item batches, got %d", node.LoopBatchSize)
			}
			if node.Type == "n8n-nodes-base.wait" && (node.WaitAmountPath != "seconds" || node.WaitUnit != "hours") {
				t.Fatalf("Wait must retain the public dynamic amount and pinned default hours unit, got path=%q unit=%q", node.WaitAmountPath, node.WaitUnit)
			}
		}
	}
	if manualID == "" || scheduleID == "" {
		t.Fatalf("public export must retain both manual and scheduled entry points: manual=%q schedule=%q", manualID, scheduleID)
	}
	for _, triggerID := range []string{manualID, scheduleID} {
		run, err := executeDetailedWithInput(context.Background(), workflow, triggerID, simulator, false, time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC), nil)
		if err != nil {
			t.Fatalf("run trigger %s: %v", triggerID, err)
		}
		loopCount, waitCount, createCount := 0, 0, 0
		for _, event := range run.Trace {
			switch {
			case strings.Contains(event.Operation, "/Loop Over Items"):
				loopCount++
			case strings.Contains(event.Operation, "/Wait / virtual resume"):
				waitCount++
				var seconds int
				if _, scanErr := fmt.Sscanf(event.ResponseSummary, "virtual resume after %d seconds", &seconds); scanErr != nil || seconds < 20*60*60 || seconds > 60*60*60 {
					t.Fatalf("dynamic Wait must use the public integer delay and pinned default hours unit: %q", event.ResponseSummary)
				}
			case event.RequestSummary == "Create a row" && strings.HasSuffix(event.Operation, "n8n-nodes-base.supabase:row.create"):
				createCount++
			}
		}
		if loopCount != 25 || waitCount != 25 || createCount != 25 {
			t.Fatalf("trigger %s should simulate 25 serial batches, waits and synthetic Supabase creates; got loop=%d wait=%d create=%d trace=%#v", triggerID, loopCount, waitCount, createCount, run.Trace)
		}
	}
}

func TestSplitInBatchesKeepsOwnedFeedbackAndBoundedLoopBody(t *testing.T) {
	workflow, _, _, err := normalizeWorkflow(readFixture(t, "public-workflows/14745.json"))
	if err != nil {
		t.Fatal(err)
	}
	var loop WorkflowNode
	var codeID, waitID string
	for _, node := range workflow.Nodes {
		if node.LoopBatchSize > 0 {
			loop = node
		}
		if node.Name == "Code in JavaScript1" {
			codeID = node.ID
		}
		if node.Name == "Wait" {
			waitID = node.ID
		}
	}
	if loop.ID == "" || codeID == "" || waitID == "" {
		t.Fatalf("fixture lacks expected generic loop nodes: loop=%q code=%q wait=%q", loop.ID, codeID, waitID)
	}
	if _, _, err := linearLoopBody(workflow, loop); err != nil {
		t.Fatalf("unchanged public loop body must be recognized: %v", err)
	}

	branched := workflow
	branched.Connections = append(append([]Connection(nil), workflow.Connections...), Connection{From: codeID, To: waitID, OutputPort: 0, InputPort: 0})
	if _, _, err := linearLoopBody(branched, loop); err == nil || !strings.Contains(err.Error(), "duplicate edge") {
		t.Fatalf("loop executor must reject a duplicate edge rather than execute it twice: %v", err)
	}

	foreignCycle := workflow
	foreignCycle.Connections = append(append([]Connection(nil), workflow.Connections...), Connection{From: waitID, To: codeID, OutputPort: 0, InputPort: 0})
	if err := validateGraph(foreignCycle); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("ordinary cycles must remain rejected even with one owned loop: %v", err)
	}

	missingFeedback := workflow
	missingFeedback.Connections = make([]Connection, 0, len(workflow.Connections)-1)
	feedbackEdges, err := loopFeedbackEdges(workflow)
	if err != nil || len(feedbackEdges) != 1 {
		t.Fatalf("expected one explicit feedback edge, got %v (%v)", feedbackEdges, err)
	}
	for _, edge := range workflow.Connections {
		if !feedbackEdges[edge] {
			missingFeedback.Connections = append(missingFeedback.Connections, edge)
		}
	}
	if err := validateGraph(missingFeedback); err == nil || !strings.Contains(err.Error(), "feedback") {
		t.Fatalf("loop node without its owned feedback edge must fail closed: %v", err)
	}

	wrongPort := workflow
	wrongPort.Connections = append([]Connection(nil), workflow.Connections...)
	for index, edge := range wrongPort.Connections {
		if edge.To == loop.ID && edge.From != loop.ID {
			wrongPort.Connections[index].InputPort = 1
		}
	}
	if err := validateGraph(wrongPort); err == nil || !strings.Contains(err.Error(), "port 0") {
		t.Fatalf("loop feedback on a non-main port must fail closed: %v", err)
	}
}
