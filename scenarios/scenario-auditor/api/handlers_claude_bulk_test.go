package main

import "testing"

type bulkTestFinding struct{ ID string }
type bulkTestGroup struct {
	Scenario string
	Path     string
	Findings []bulkTestFinding
}

func TestPrepareBulkTargetsGroupsScenarioAndDeduplicatesIDs(t *testing.T) {
	targets := []claudeFixTarget{
		{Scenario: "scenario-auditor", IssueIDs: []string{"z", "a"}},
		{Scenario: "scenario-auditor", IssueIDs: []string{"a"}},
	}
	load := func(string) []bulkTestFinding { return []bulkTestFinding{{ID: "z"}, {ID: "a"}} }
	selectFindings := func(all []bulkTestFinding, ids []string) ([]bulkTestFinding, []string) {
		return selectStoredFindings(all, ids, func(item bulkTestFinding) string { return item.ID })
	}
	groups, ids, err := prepareBulkTargets(
		targets,
		load,
		selectFindings,
		func(scenario, path string, findings []bulkTestFinding) bulkTestGroup {
			return bulkTestGroup{Scenario: scenario, Path: path, Findings: findings}
		},
		func(group *bulkTestGroup, findings []bulkTestFinding) {
			group.Findings = append(group.Findings, findings...)
		},
		"test findings",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].Scenario != "scenario-auditor" || groups[0].Path == "" {
		t.Fatalf("groups = %+v", groups)
	}
	if len(groups[0].Findings) != 3 {
		t.Fatalf("grouped findings = %+v, want three appended selections", groups[0].Findings)
	}
	if len(ids) != 2 || ids[0] != "a" || ids[1] != "z" {
		t.Fatalf("collected IDs = %v, want [a z]", ids)
	}
}

func TestPrepareBulkTargetsRejectsNoMatches(t *testing.T) {
	_, _, err := prepareBulkTargets[bulkTestFinding, bulkTestGroup](
		nil,
		func(string) []bulkTestFinding { return nil },
		func([]bulkTestFinding, []string) ([]bulkTestFinding, []string) { return nil, nil },
		func(string, string, []bulkTestFinding) bulkTestGroup { return bulkTestGroup{} },
		func(*bulkTestGroup, []bulkTestFinding) {},
		"test findings",
	)
	if err == nil || err.Error() != "no matching test findings found for provided targets" {
		t.Fatalf("empty result error = %v", err)
	}
}
