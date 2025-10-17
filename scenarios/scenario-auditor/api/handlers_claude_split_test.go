package main

import "testing"

func TestClampAgentCount(t *testing.T) {
	tests := []struct {
		name      string
		requested int
		total     int
		want      int
	}{
		{"zero total", 3, 0, 0},
		{"below minimum", 0, 5, 1},
		{"above total", 10, 4, 4},
		{"within limits", 3, 12, 3},
		{"needs more agents", 1, 120, 3},
		{"above max cap", 10, 200, maxBulkFixAgents},
		{"caps when total exceeds capacity", 0, maxBulkFixAgents*maxIssuesPerAgent + 200, maxBulkFixAgents},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampAgentCount(tt.requested, tt.total)
			if got != tt.want {
				t.Fatalf("clampAgentCount(%d, %d) = %d, want %d", tt.requested, tt.total, got, tt.want)
			}
		})
	}
}

func TestSplitStandardsTargets(t *testing.T) {
	targets := []standardsFixMultiScenario{
		{
			Scenario:     "alpha",
			ScenarioPath: "/scenarios/alpha",
			Violations: []StandardsViolation{
				{ID: "a1"},
				{ID: "a2"},
			},
		},
		{
			Scenario:     "bravo",
			ScenarioPath: "/scenarios/bravo",
			Violations: []StandardsViolation{
				{ID: "b1"},
				{ID: "b2"},
				{ID: "b3"},
			},
		},
	}

	splits := splitStandardsTargets(targets, 3)

	total := 0
	for _, group := range splits {
		if len(group) == 0 {
			t.Fatalf("expected non-empty group")
		}
		for _, scenario := range group {
			if scenario.Scenario == "" {
				t.Fatalf("scenario name should be preserved")
			}
			total += len(scenario.Violations)
			if len(scenario.Violations) > maxIssuesPerAgent {
				t.Fatalf("expected each agent chunk to be <= %d", maxIssuesPerAgent)
			}
		}
	}

	if total != countStandardsViolations(targets) {
		t.Fatalf("expected total %d, got %d", countStandardsViolations(targets), total)
	}

	if len(splits) > 3 {
		t.Fatalf("expected at most 3 groups, got %d", len(splits))
	}
}

func TestSplitStandardsTargetsSingleAgent(t *testing.T) {
	targets := []standardsFixMultiScenario{
		{
			Scenario:     "alpha",
			ScenarioPath: "/scenarios/alpha",
			Violations:   []StandardsViolation{{ID: "a1"}},
		},
	}

	splits := splitStandardsTargets(targets, 1)
	if len(splits) != 1 {
		t.Fatalf("expected single group, got %d", len(splits))
	}
	if len(splits[0]) != 1 || len(splits[0][0].Violations) != 1 {
		t.Fatalf("expected original violations to be preserved")
	}
}

func TestSplitVulnerabilityTargets(t *testing.T) {
	targets := []vulnerabilityFixMultiScenario{
		{
			Scenario:     "alpha",
			ScenarioPath: "/scenarios/alpha",
			Findings:     []StoredVulnerability{{ID: "a1"}, {ID: "a2"}, {ID: "a3"}},
		},
		{
			Scenario:     "bravo",
			ScenarioPath: "/scenarios/bravo",
			Findings:     []StoredVulnerability{{ID: "b1"}, {ID: "b2"}},
		},
	}

	splits := splitVulnerabilityTargets(targets, 2)
	if len(splits) == 0 {
		t.Fatalf("expected at least one group")
	}

	total := 0
	for _, group := range splits {
		for _, scenario := range group {
			total += len(scenario.Findings)
			if len(scenario.Findings) > maxIssuesPerAgent {
				t.Fatalf("expected each agent chunk to be <= %d", maxIssuesPerAgent)
			}
		}
	}

	if total != countVulnerabilityFindings(targets) {
		t.Fatalf("expected total %d, got %d", countVulnerabilityFindings(targets), total)
	}
}
