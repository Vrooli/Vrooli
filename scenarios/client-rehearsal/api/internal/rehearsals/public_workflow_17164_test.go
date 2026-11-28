package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestPublic17164ImportsRSSPlaceholderAsRequiredSyntheticInput(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/17164.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(workflowJSON))); got != "2788a548183fb6a9cf96f9c16a590ab7da4e87ae7850fac7c0dd014e6194f715" {
		t.Fatalf("sanitized fixture hash changed without provenance update: %s", got)
	}
	if strings.Contains(workflowJSON, `"credentials"`) || strings.Contains(workflowJSON, "googleapis.com") || strings.Contains(workflowJSON, "slack.com") {
		t.Fatal("sanitized graph retained credentials or live Google/Slack service hostnames")
	}
	for _, match := range regexp.MustCompile(`https?://([^/\s"'<>]+)`).FindAllStringSubmatch(workflowJSON, -1) {
		if len(match) < 2 || !strings.HasSuffix(match[1], ".synthetic.invalid") {
			t.Fatalf("sanitized graph retained a non-synthetic URL host: %q", match)
		}
	}
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("normalize sanitized template with a required RSS input: %v", err)
	}
	if len(workflow.Nodes) != 9 {
		t.Fatalf("expected all nine executable nodes, got %d", len(workflow.Nodes))
	}
	var rss *WorkflowNode
	for i := range workflow.Nodes {
		if workflow.Nodes[i].RSSURLPath != "" {
			rss = &workflow.Nodes[i]
			break
		}
	}
	if rss == nil || rss.RSSURLPath != ".feedUrl" {
		t.Fatalf("expected RSS placeholder to become required feedUrl input, got %#v", rss)
	}
	if rss.Request == nil || rss.Request.Host != "rss.synthetic.invalid" {
		t.Fatalf("RSS input must be routed only to the synthetic adapter: %#v", rss.Request)
	}
	t.Log("import retained all executable nodes; RSS placeholder is a required synthetic-only feedUrl input")
	simulator, err := NewDescriptionSimulator(readFixture(t, "public-workflow-17164-profile.json"), false)
	if err != nil {
		t.Fatalf("load synthetic RSS/Gemini/Sheets/Slack profile: %v", err)
	}
	run, err := executeDetailedWithInput(context.Background(), workflow, workflow.EntryNodeIDs[0], simulator, false,
		time.Date(2026, 11, 13, 7, 0, 0, 0, time.UTC), map[string]any{"feedUrl": "https://feeds.synthetic.invalid/one"})
	if err != nil {
		t.Fatalf("execute source workflow against synthetic profile: %v", err)
	}
	for _, node := range []string{"Read Blog Feed", "Generate Social Posts with Gemini", "Save Drafts to Sheet", "Share Drafts for Review"} {
		found := false
		for _, event := range run.Trace {
			if strings.Contains(event.RequestSummary, node) && event.Status == "succeeded" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected successful source node %q, trace=%#v", node, run.Trace)
		}
	}
	if got := countRequests(run.Trace, "googleSheets:sheet.append"); got != 2 {
		t.Errorf("expected both feed items to generate separate sheet drafts, got %d", got)
	}
	if got := countRequests(run.Trace, "slack:message.post"); got != 2 {
		t.Errorf("expected both feed items to generate separate review posts, got %d", got)
	}
}
