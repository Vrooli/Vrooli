package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"
)

func TestPublic1804GitHubIssueEditRunsAgainstSyntheticNotion(t *testing.T) {
	fixture := readFixture(t, "public-workflows/1804.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(fixture))); got != "f07ea4384b83db37b3f999bdfce143c83e477ebd3defff2e8f0f02a8eb377ce8" {
		t.Fatalf("sanitized public fixture changed: %s", got)
	}
	workflow, _, _, err := normalizeWorkflow(fixture)
	if err != nil {
		t.Fatalf("normalize unchanged public workflow: %v", err)
	}
	var trigger string
	for _, node := range workflow.Nodes {
		if node.Trigger == TriggerPolling {
			trigger = node.ID
		}
	}
	if trigger == "" {
		t.Fatal("GitHub polling trigger missing")
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "public_workflow_1804_catalog.json"), false)
	if err != nil {
		t.Fatalf("load synthetic Notion catalog: %v", err)
	}
	cases := []struct {
		action    string
		branch    string
		operation string
		lookup    bool
	}{
		{action: "opened", branch: "Create database page", operation: "notion:databasePage.create"},
		{action: "edited", branch: "Edit issue", operation: "notion:databasePage.update", lookup: true},
		{action: "deleted", branch: "Delete issue", operation: "notion:page.archive", lookup: true},
		{action: "closed", branch: "Close issue", operation: "notion:databasePage.update", lookup: true},
		{action: "reopened", branch: "Reopen issue", operation: "notion:databasePage.update", lookup: true},
	}
	for _, test := range cases {
		input := syntheticTriggerInput(TriggerPolling)
		input["body"].(map[string]any)["action"] = test.action
		run, runErr := executeDetailedWithInput(context.Background(), workflow, trigger, simulator, false, time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC), input)
		if runErr != nil {
			t.Fatalf("run public GitHub action %q: %v", test.action, runErr)
		}
		if !traceHasNode(run.Trace, test.branch) || !traceHasOperation(run.Trace, test.operation) {
			t.Errorf("action %q failed to select %s / %s: %#v", test.action, test.branch, test.operation, run.Trace)
		}
		if test.lookup != traceHasOperation(run.Trace, "notion:databasePage.getAll") {
			t.Errorf("action %q lookup=%v; trace=%#v", test.action, traceHasOperation(run.Trace, "notion:databasePage.getAll"), run.Trace)
		}
		if !traceHasNode(run.Trace, "Trigger on issues") {
			t.Errorf("action %q did not enter through GitHub polling trigger", test.action)
		}
	}
}
