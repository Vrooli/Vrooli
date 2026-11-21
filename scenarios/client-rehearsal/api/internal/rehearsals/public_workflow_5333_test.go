package rehearsals

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestPublic5333RSSDeduplicationRunsAgainstSyntheticFeedAndSheetsProfiles(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/5333.json")
	for _, forbidden := range []string{"C09281Y80AH", "1HIldK5vDx2luvmYEstU2PQzSS68xBrpp7lJ-8rQb2No", "cachedResultUrl", "api.n8n.io"} {
		if strings.Contains(workflowJSON, forbidden) {
			t.Fatalf("sanitized public workflow retained source destination %q", forbidden)
		}
	}
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("normalize unchanged sanitized public workflow: %v", err)
	}
	if len(workflow.Nodes) != 7 || len(workflow.Connections) != 6 || len(workflow.EntryNodeIDs) != 1 || workflow.TriggerKinds[0] != TriggerScheduled {
		t.Fatalf("expected seven executable nodes and one Cron entry, got nodes=%d edges=%d entries=%v triggers=%v", len(workflow.Nodes), len(workflow.Connections), workflow.EntryNodeIDs, workflow.TriggerKinds)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "public-workflow-5333-profile.json"), false)
	if err != nil {
		t.Fatalf("load deterministic synthetic feed, Sheets and Slack profiles: %v", err)
	}
	run, err := executeDetailedWithInput(context.Background(), workflow, workflow.EntryNodeIDs[0], simulator, false, time.Date(2026, 11, 11, 7, 0, 0, 0, time.UTC), nil)
	if err != nil {
		t.Fatalf("execute sanitized public workflow: %v", err)
	}
	for _, operation := range []string{"googleSheets:sheet.read", "n8n-nodes-base.rssFeedRead:feed.read", "googleSheets:sheet.append", "slack:message.post"} {
		if !traceHasOperation(run.Trace, operation) {
			t.Errorf("expected operation %q in trace: %#v", operation, run.Trace)
		}
	}
	if got := countRequests(run.Trace, "googleSheets:sheet.read"); got != 2 {
		t.Fatalf("expected two generic Sheets reads, got %d", got)
	}
	foundCode := false
	for _, event := range run.Trace {
		if event.RequestSummary == "Filter Unpublished Articles" && event.ResponseSummary == "safe Code interpreter emitted 2 synthetic items" {
			foundCode = true
		}
	}
	if !foundCode {
		t.Fatalf("expected the public duplicate-filter Code to retain only two new articles: %#v", run.Trace)
	}
	if got := countRequestSummaries(run.Trace, "GET rss.synthetic.invalid/v1/feed"); got != 2 {
		t.Fatalf("expected two bounded RSS simulator calls, got %d", got)
	}
	if got := countRequests(run.Trace, "googleSheets:sheet.append"); got != 2 {
		t.Fatalf("expected the two not-yet-posted synthetic stories to be appended, got %d", got)
	}
	if got := countRequests(run.Trace, "slack:message.post"); got != 2 {
		t.Fatalf("expected the two not-yet-posted synthetic stories to be posted, got %d", got)
	}
	if strings.Contains(strings.ToLower(strings.Join(traceSummaries(run.Trace), " ")), "google.com") || strings.Contains(strings.Join(traceSummaries(run.Trace), " "), "C09281Y80AH") {
		t.Fatal("trace retained a real destination from the public export")
	}
}

func countRequestSummaries(trace []TraceEvent, summary string) int {
	count := 0
	for _, event := range trace {
		if event.RequestSummary == summary {
			count++
		}
	}
	return count
}

func traceSummaries(trace []TraceEvent) []string {
	values := make([]string, 0, len(trace)*2)
	for _, event := range trace {
		values = append(values, event.RequestSummary, event.ResponseSummary)
	}
	return values
}
