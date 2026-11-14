package rehearsals

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestPublic16364DailyNewsletterRunsAgainstSyntheticSheetsAndGmail(t *testing.T) {
	workflow, _, _, err := normalizeWorkflow(readFixture(t, "public-workflows/16364.json"))
	if err != nil {
		t.Fatalf("normalize unchanged sanitized public workflow: %v", err)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "public_workflow_16364_catalog.json"), false)
	if err != nil {
		t.Fatalf("load synthetic service descriptions: %v", err)
	}
	trigger := ""
	for _, node := range workflow.Nodes {
		if node.Trigger == TriggerScheduled {
			trigger = node.ID
		}
	}
	if trigger == "" {
		t.Fatal("public schedule trigger missing")
	}
	run, err := executeDetailedWithInput(context.Background(), workflow, trigger, simulator, false, time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC), nil)
	if err != nil {
		t.Fatalf("execute public scheduled workflow: %v", err)
	}
	for _, node := range []string{"Read Products", "Filter by Today", "Products Found?", "Build HTML Email", "Read Recipients", "Prepare per Recipient", "Send Newsletter"} {
		if !traceHasNode(run.Trace, node) {
			t.Errorf("public graph did not execute %q", node)
		}
	}
	if !traceHasOperation(run.Trace, "googleSheets:sheet.read") || !traceHasOperation(run.Trace, "gmail:message.send") {
		t.Fatalf("expected synthetic Sheets reads and Gmail sends; trace=%#v", run.Trace)
	}
	if sends := countRequests(run.Trace, "gmail:message.send"); sends != 2 {
		t.Fatalf("expected one simulated send per synthetic recipient, got %d; trace=%#v", sends, run.Trace)
	}
	for _, event := range run.Trace {
		if strings.Contains(event.RequestSummary, "@example.com") || strings.Contains(event.RequestSummary, "@gmail.com") {
			t.Fatalf("trace leaked non-synthetic email: %#v", event)
		}
	}
}

func countRequests(trace []TraceEvent, suffix string) int {
	count := 0
	for _, event := range trace {
		if strings.HasSuffix(event.Operation, suffix) && strings.TrimSpace(event.RequestSummary) == "" {
			count++
		}
	}
	return count
}
