package main

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type bulkPlanTestTarget struct {
	Scenario string
	IDs      []string
}

func bulkPlanTestAdapter() bulkFixPlanAdapter[bulkPlanTestTarget] {
	return bulkFixPlanAdapter[bulkPlanTestTarget]{
		prepare: func([]claudeFixTarget) ([]bulkPlanTestTarget, []string, error) {
			return []bulkPlanTestTarget{{Scenario: "alpha", IDs: []string{"a"}}, {Scenario: "bravo", IDs: []string{"b"}}}, nil, nil
		},
		count: func(targets []bulkPlanTestTarget) int {
			total := 0
			for _, target := range targets {
				total += len(target.IDs)
			}
			return total
		},
		scenarios: func(targets []bulkPlanTestTarget) []string {
			var names []string
			for _, target := range targets {
				names = append(names, target.Scenario)
			}
			return names
		},
		split: func(targets []bulkPlanTestTarget, _ int) [][]bulkPlanTestTarget {
			return [][]bulkPlanTestTarget{{targets[0]}, {targets[1]}}
		},
		prompt: func(targets []bulkPlanTestTarget, extra string) (string, string, map[string]string, error) {
			return "prompt:" + strings.Join(targets[0].IDs, ",") + extra, "fix", nil, nil
		},
		issueIDs: func(targets []bulkPlanTestTarget) []string {
			var ids []string
			for _, target := range targets {
				ids = append(ids, target.IDs...)
			}
			return ids
		},
		action: "test-fix",
		noun:   "test findings",
	}
}

func TestBuildBulkFixPlanBuildsSingleAgentPlan(t *testing.T) {
	plan := &fixPlan{FixType: "test"}
	got, err := buildBulkFixPlan(&claudeFixRequest{}, plan, "!", "test-model", bulkPlanTestAdapter())
	if err != nil {
		t.Fatal(err)
	}
	if got.IssueCount != 2 || !reflect.DeepEqual(got.Scenarios, []string{"alpha", "bravo"}) {
		t.Fatalf("plan summary = %+v", got)
	}
	if len(got.AgentEntries) != 1 {
		t.Fatalf("agent entries = %d, want 1", len(got.AgentEntries))
	}
	entry := got.AgentEntries[0]
	if entry.Config.Action != "test-fix" || entry.Config.Name != "fix (multi)" || entry.Config.Prompt != "prompt:a!" {
		t.Fatalf("single agent config = %+v", entry.Config)
	}
	if entry.Config.Metadata["issue_count"] != "2" || entry.Config.Metadata["model"] != "test-model" {
		t.Fatalf("single agent metadata = %v", entry.Config.Metadata)
	}
	if got.Message != "Started fix across 2 scenario(s)" {
		t.Fatalf("message = %q", got.Message)
	}
}

func TestBuildBulkFixPlanBuildsBatchedAgents(t *testing.T) {
	plan := &fixPlan{FixType: "test"}
	req := &claudeFixRequest{AgentCount: 2}
	got, err := buildBulkFixPlan(req, plan, "", "test-model", bulkPlanTestAdapter())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.AgentEntries) != 2 {
		t.Fatalf("agent entries = %d, want 2", len(got.AgentEntries))
	}
	for i, entry := range got.AgentEntries {
		want := fmt.Sprintf("fix (batch %d/2)", i+1)
		if entry.Config.Name != want || entry.Config.Metadata["batch_index"] != fmt.Sprintf("%d/2", i+1) {
			t.Fatalf("batch %d = %+v", i+1, entry.Config)
		}
	}
	if got.Message != "Started 2 agents to address 2 test findings across 2 scenario(s)" {
		t.Fatalf("message = %q", got.Message)
	}
}

func TestBuildBulkFixPlanRejectsEmptyFindings(t *testing.T) {
	adapter := bulkPlanTestAdapter()
	adapter.prepare = func([]claudeFixTarget) ([]bulkPlanTestTarget, []string, error) { return nil, nil, nil }
	adapter.count = func([]bulkPlanTestTarget) int { return 0 }
	_, err := buildBulkFixPlan(&claudeFixRequest{}, &fixPlan{FixType: "test"}, "", "", adapter)
	planErr, ok := err.(*fixPlanError)
	if !ok || planErr.status != http.StatusBadRequest || planErr.message != "No matching test findings found to fix" {
		t.Fatalf("empty plan error = %#v", err)
	}
}
