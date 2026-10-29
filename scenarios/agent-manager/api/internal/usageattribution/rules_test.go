package usageattribution

import (
	"strings"
	"testing"
	"time"
)

func weekReport(t *testing.T) (Report, Result) {
	t.Helper()
	f := newWeekFixture()
	result := Attribute(f.window, f.window.Start, f.turns, fixturePrices)
	return Recommend(Inputs{Results: []Result{result}, Runs: f.runs, Episodes: f.episodes, Outcomes: f.outcomes, Prices: fixturePrices}), result
}

// A6: every rule family fires on the fixture week, and each row carries an
// estimate, a confidence, evidence and an action.
func TestEveryRuleFamilyFiresWithEstimateConfidenceEvidenceAndAction(t *testing.T) {
	report, _ := weekReport(t)
	fired := map[string]int{}
	for _, row := range report.Recommendations {
		fired[row.Rule]++
		if row.WeeklySaved <= 0 || row.Confidence == "" || row.Action == "" || len(row.Evidence) == 0 || len(row.Metrics) == 0 {
			t.Errorf("%s row incomplete: %+v", row.Rule, row)
		}
		if row.Provider != "openai" || row.Window != "10080m" {
			t.Errorf("%s row is not stated in the weekly window: %s %s", row.Rule, row.Provider, row.Window)
		}
	}
	for _, rule := range Rules {
		if fired[rule] == 0 {
			t.Errorf("rule %s did not fire on the fixture week", rule)
		}
	}
	for _, status := range report.Rules {
		if !status.Fired || status.Rows != fired[status.Rule] {
			t.Errorf("status %+v does not match %d fired rows", status, fired[status.Rule])
		}
	}
}

// A6: the ranking puts the dominant consumer's lever first, and ranks are
// in order of weekly saving.
func TestRecommendationsRankTheDominantConsumerFirst(t *testing.T) {
	report, _ := weekReport(t)
	if report.Dominant != "research" {
		t.Fatalf("dominant consumer = %q, want the research orchestrator session", report.Dominant)
	}
	top := report.Recommendations[0]
	if top.TargetID != report.Dominant || top.Rank != 1 {
		t.Fatalf("top recommendation = %s on %s, want the dominant consumer %s", top.Rule, top.TargetID, report.Dominant)
	}
	for i := 1; i < len(report.Recommendations); i++ {
		if report.Recommendations[i].WeeklySaved > report.Recommendations[i-1].WeeklySaved || report.Recommendations[i].Rank != i+1 {
			t.Fatalf("rank %d out of order: %+v", i+1, report.Recommendations[i])
		}
	}
}

// Each rule's finding names the lever the fixture planted.
func TestRulesTargetThePlantedLevers(t *testing.T) {
	report, result := weekReport(t)
	byRule := map[string]Recommendation{}
	for _, row := range report.Recommendations {
		if _, seen := byRule[row.Rule]; !seen {
			byRule[row.Rule] = row
		}
	}
	want := map[string]string{RuleContext: "research", RuleFanout: "research", RuleIdle: "heartbeat", RuleImage: "ui-review", RuleModel: "workflow_node:draft", RuleFriction: "reread:docs/SPEC.md"}
	for rule, target := range want {
		if got := byRule[rule].TargetID; got != target {
			t.Errorf("%s targets %q, want %q", rule, got, target)
		}
	}
	if model := byRule[RuleModel]; !strings.Contains(model.Action, modelTerra+" to "+modelLuna) {
		t.Errorf("R-model action = %q, want a move from terra to luna", model.Action)
	}
	// A friction finding never costs more than its runs consumed.
	var orchestrator float64
	for i, turn := range result.Turns {
		if turn.RunID == "orchestrator" {
			orchestrator += result.Points[i]
		}
	}
	if friction := byRule[RuleFriction]; friction.ConsumerPoints <= 0 || friction.ConsumerPoints > orchestrator {
		t.Errorf("friction priced at %.2f pts, want within the run's %.2f pts", friction.ConsumerPoints, orchestrator)
	}
}

// A rule that cannot fire says why from the data.
func TestRulesExplainWhyTheyDidNotFire(t *testing.T) {
	f := newWeekFixture()
	quiet := []Turn{f.turns[0]}
	result := Attribute(f.window, f.window.Start, quiet, fixturePrices)
	report := Recommend(Inputs{Results: []Result{result}, Runs: f.runs, Prices: fixturePrices})
	if len(report.Rules) != len(Rules) {
		t.Fatalf("rule statuses = %d, want one per family", len(report.Rules))
	}
	for _, status := range report.Rules {
		if status.Fired || status.Reason == "" {
			t.Errorf("status %+v: want not fired with a data reason", status)
		}
	}
}

// F-ORCH-8: a session that ran for a few hours and stopped is not
// extrapolated to a week; its estimate stays within what it used. A session
// still running is projected at its rate, and no estimate exceeds the
// consumer's projected weekly use. Every row states its basis.
func TestRecommendationSavingsAreBoundedByTheConsumersProjectedWeeklyUse(t *testing.T) {
	f := newWeekFixture()
	short := fixtureRun("vega-e1", "vega-e1", false)
	f.runs[short.ID] = short
	f.turns = append(f.turns, usageSpec{run: short, model: modelTerra, every: 5 * time.Minute, reports: 2, prompt: 248_000, output: 600, from: 10 * time.Hour, until: 30 * time.Hour}.turns()...)
	f.window = truthWindow(f.turns, map[string]float64{modelTerra: 0.4, modelLuna: 0.6}, steady(0.02), 6*24)
	result := Attribute(f.window, f.window.Start, f.turns, fixturePrices)
	report := Recommend(Inputs{Results: []Result{result}, Runs: f.runs, Episodes: f.episodes, Outcomes: f.outcomes, Prices: fixturePrices})

	metric := func(row Recommendation, name string) (float64, bool) {
		for _, m := range row.Metrics {
			if m.Name == name {
				return m.Value, true
			}
		}
		return 0, false
	}
	var sawShort, sawActive bool
	for _, row := range report.Recommendations {
		if row.Rule == RuleModel {
			continue // stated from the 14-day outcome history, already weekly
		}
		projected, ok := metric(row, "projected_weekly_points")
		if !ok || !strings.Contains(row.Rationale, "Basis: ") {
			t.Fatalf("%s %s states no basis: %q %+v", row.Rule, row.TargetID, row.Rationale, row.Metrics)
		}
		if row.WeeklySaved > projected+1e-9 {
			t.Errorf("%s %s saves %.2f pts/wk, more than the consumer's projected %.2f", row.Rule, row.TargetID, row.WeeklySaved, projected)
		}
		running, _ := metric(row, "still_running")
		switch {
		case row.Rule == RuleContext && row.TargetID == short.ID:
			sawShort = true
			if running != 0 || row.WeeklySaved > row.ConsumerPoints || projected != row.ConsumerPoints {
				t.Errorf("stopped session extrapolated: saves %.2f, projected %.2f, used %.2f, running=%v", row.WeeklySaved, projected, row.ConsumerPoints, running)
			}
		case row.Rule == RuleContext && row.TargetID == "research":
			sawActive = true
			if running != 1 || projected <= row.ConsumerPoints {
				t.Errorf("running session not projected: projected %.2f, used %.2f", projected, row.ConsumerPoints)
			}
		}
	}
	if !sawShort || !sawActive {
		t.Fatalf("fixture rows missing: short=%v active=%v", sawShort, sawActive)
	}
}

// W8b: R-context counts provider requests, not turns; the action and metric
// say so (a Codex turn makes many requests, each re-reading the context).
func TestContextRuleNamesItsCountRequests(t *testing.T) {
	report, _ := weekReport(t)
	for _, row := range report.Recommendations {
		if row.Rule != RuleContext {
			continue
		}
		if !strings.Contains(row.Action, " requests)") || strings.Contains(row.Action, "turns") {
			t.Fatalf("action = %q, want a request count", row.Action)
		}
		for _, m := range row.Metrics {
			if m.Name == "turns" {
				t.Fatalf("metric %q mislabels provider requests", m.Name)
			}
		}
		return
	}
	t.Fatal("no R-context row")
}
