package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestLaunchAutomatedFixGroups(t *testing.T) {
	job := &automatedFixJob{scenario: "demo", automationRunID: "run-7"}
	metadata := map[string]string{"trigger": "automation_loop", "scope": "base"}
	var configs []AgentStartConfig
	var records []struct {
		scenario, kind, severity, agentID string
		count                             int
		runID                             string
	}
	startAgent := func(config AgentStartConfig) (*AgentInfo, error) {
		configs = append(configs, config)
		return &AgentInfo{ID: fmt.Sprintf("agent-%d", len(configs))}, nil
	}
	recordStart := func(scenario, kind, severity, agentID string, count int, runID string) AutomatedFixRecord {
		records = append(records, struct {
			scenario, kind, severity, agentID string
			count                             int
			runID                             string
		}{scenario, kind, severity, agentID, count, runID})
		return AutomatedFixRecord{}
	}

	agents, err := launchAutomatedFixGroups(
		job,
		[][]string{{"issue-1"}, {"issue-2"}},
		"security", "vulnerabilities", agentActionVulnerabilityFix,
		"test-model", metadata,
		map[string]string{"issue-1": "high", "issue-2": "medium"},
		func(ids []string, _ string) (string, string, map[string]string, error) {
			return "prompt for " + ids[0], "Fix " + ids[0], map[string]string{"scope": "prompt"}, nil
		},
		func(ids []string) []string { return ids },
		startAgent,
		recordStart,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 2 || len(configs) != 2 || len(records) != 2 {
		t.Fatalf("agents/configs/records = %d/%d/%d, want 2/2/2", len(agents), len(configs), len(records))
	}
	if metadata["scope"] != "base" {
		t.Fatalf("input metadata mutated: %#v", metadata)
	}
	for index, config := range configs {
		wantID := fmt.Sprintf("issue-%d", index+1)
		if config.Label != "Fix "+wantID || config.Name != "Fix "+wantID+" (automated)" || config.Action != agentActionVulnerabilityFix {
			t.Errorf("config %d identity = %+v", index, config)
		}
		if config.Scenario != "demo" || config.Prompt != "prompt for "+wantID || config.Model != "test-model" || !reflect.DeepEqual(config.IssueIDs, []string{wantID}) {
			t.Errorf("config %d payload = %+v", index, config)
		}
		if config.Metadata["scope"] != "prompt" || config.Metadata["issue_kind"] != "vulnerabilities" || config.Metadata["batch_index"] != fmt.Sprintf("%d/2", index+1) {
			t.Errorf("config %d metadata = %#v", index, config.Metadata)
		}
	}
	if records[0].scenario != "demo" || records[0].kind != "security" || records[0].severity != "high" || records[0].agentID != "agent-1" || records[0].count != 1 || records[0].runID != "run-7" {
		t.Errorf("first start record = %+v", records[0])
	}
	if records[1].severity != "medium" || records[1].agentID != "agent-2" {
		t.Errorf("second start record = %+v", records[1])
	}
}
