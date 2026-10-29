package usageattribution

import (
	"fmt"
	"math"
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

// levers lists every lever of every plan.
func levers(report Report) []Recommendation {
	var out []Recommendation
	for _, plan := range append(append([]Plan(nil), report.Plans...), report.Folded...) {
		out = append(out, plan.Levers...)
	}
	return out
}

// A6: every rule family fires on the fixture week, and each lever carries an
// estimate, a confidence, evidence, typed facts and an action.
func TestEveryRuleFamilyFiresWithEstimateConfidenceEvidenceAndAction(t *testing.T) {
	report, _ := weekReport(t)
	fired := map[string]int{}
	for _, row := range levers(report) {
		fired[row.Rule]++
		facts := row.Facts
		set := 0
		for _, present := range []bool{facts.Context != nil, facts.Fanout != nil, facts.Idle != nil, facts.Image != nil, facts.Friction != nil, facts.Model != nil} {
			if present {
				set++
			}
		}
		if row.Saved <= 0 || row.Confidence == "" || row.Action == "" || len(row.Evidence) == 0 || set != 1 {
			t.Errorf("%s lever incomplete: %+v", row.Rule, row)
		}
		if row.Provider != "openai" || row.Window != "10080m" {
			t.Errorf("%s lever is not stated in the weekly window: %s %s", row.Rule, row.Provider, row.Window)
		}
	}
	for _, rule := range Rules {
		if fired[rule] == 0 {
			t.Errorf("rule %s did not fire on the fixture week", rule)
		}
	}
	for _, status := range report.Rules {
		if !status.Fired || status.Rows != fired[status.Rule] {
			t.Errorf("status %+v does not match %d fired levers", status, fired[status.Rule])
		}
	}
}

// A6: the ranking puts the dominant consumer's plan first, and plans are in
// order of combined saving.
func TestPlansRankTheDominantConsumerFirst(t *testing.T) {
	report, _ := weekReport(t)
	if report.Dominant != "research" {
		t.Fatalf("dominant consumer = %q, want the research orchestrator session", report.Dominant)
	}
	top := report.Plans[0]
	if top.TargetID != report.Dominant || top.Rank != 1 {
		t.Fatalf("top plan = %s, want the dominant consumer %s", top.TargetID, report.Dominant)
	}
	for i := 1; i < len(report.Plans); i++ {
		if report.Plans[i].Saved > report.Plans[i-1].Saved || report.Plans[i].Rank != i+1 {
			t.Fatalf("rank %d out of order: %+v", i+1, report.Plans[i])
		}
	}
}

// P2-A4: levers of one target form one plan whose combined saving applies each
// lever to what the larger ones leave, so it is below their sum and at least
// the largest; the marginals add up to it.
func TestPlanCombinesOverlappingLeversWithoutDoubleCounting(t *testing.T) {
	report, _ := weekReport(t)
	plan := report.Plans[0]
	if len(plan.Levers) < 2 {
		t.Fatalf("the dominant session has %d levers, want context and fan-out together", len(plan.Levers))
	}
	var sum, marginals float64
	for i, lever := range plan.Levers {
		sum += lever.Saved
		marginals += lever.Marginal
		if lever.Marginal > lever.Saved+1e-9 {
			t.Errorf("%s adds %.3f after larger levers, more than its own %.3f", lever.Rule, lever.Marginal, lever.Saved)
		}
		if i > 0 && lever.Saved > plan.Levers[i-1].Saved {
			t.Errorf("levers out of order at %d", i)
		}
	}
	if plan.Saved >= sum-1e-9 || plan.Saved < plan.Levers[0].Saved-1e-9 || plan.Saved > plan.Use.AtThisRate+1e-9 {
		t.Errorf("combined %.3f, want within [largest %.3f, sum %.3f) and under the projected use %.3f", plan.Saved, plan.Levers[0].Saved, sum, plan.Use.AtThisRate)
	}
	if math.Abs(marginals-plan.Saved) > 1e-9 {
		t.Errorf("marginals add to %.6f, want the combined %.6f", marginals, plan.Saved)
	}
	// Two levers saving fractions a and b of the use combine to a+b-ab.
	a, b := plan.Levers[0].Saved/plan.Use.AtThisRate, plan.Levers[1].Saved/plan.Use.AtThisRate
	if len(plan.Levers) == 2 && math.Abs(plan.Saved-plan.Use.AtThisRate*(a+b-a*b)) > 1e-9 {
		t.Errorf("combined %.6f, want projected × (a+b−ab) = %.6f", plan.Saved, plan.Use.AtThisRate*(a+b-a*b))
	}
}

// Every plan states its use: points so far, the projection, the share of the
// window's burn and whether the target is still running.
func TestPlansCarryUseSoFarProjectionShareAndRunningState(t *testing.T) {
	report, result := weekReport(t)
	for _, plan := range report.Plans {
		if plan.Use.ConsumerPoints <= 0 || plan.Use.AtThisRate < plan.Use.ConsumerPoints-1e-9 || plan.Use.Share <= 0 || plan.Use.Share > 1 {
			t.Errorf("%s use incomplete: %+v", plan.TargetID, plan.Use)
		}
		if plan.TargetKind == "session" && math.Abs(plan.Use.Share-plan.Use.ConsumerPoints/result.Observed) > 1e-9 {
			t.Errorf("%s share %.4f, want points / observed %.4f", plan.TargetID, plan.Use.Share, plan.Use.ConsumerPoints/result.Observed)
		}
		for _, lever := range plan.Levers {
			if lever.Use != plan.Use {
				t.Errorf("%s lever %s states a different use %+v than its plan %+v", plan.TargetID, lever.Rule, lever.Use, plan.Use)
			}
		}
	}
	if !report.Plans[0].Use.StillRunning {
		t.Error("the orchestrator ran to the end of the week and must be still running")
	}
}

// Plans saving under 0.1 points are counted, not listed.
func TestPlansSavingLessThanATenthOfAPointAreFolded(t *testing.T) {
	big := Recommendation{Rule: RuleContext, TargetKind: "session", TargetID: "a", Provider: "p", Pool: "w", Window: "1", Saved: 4, Use: Use{ConsumerPoints: 10, AtThisRate: 20}}
	small := Recommendation{Rule: RuleIdle, TargetKind: "session", TargetID: "b", Provider: "p", Pool: "w", Window: "1", Saved: 0.08, Use: Use{ConsumerPoints: 1, AtThisRate: 1}}
	plans, folded := buildPlans([]Recommendation{small, big})
	if len(plans) != 1 || plans[0].TargetID != "a" || plans[0].Rank != 1 || len(folded) != 1 || math.Abs(folded[0].Saved-0.08) > 1e-12 {
		t.Fatalf("plans=%+v folded=%+v", plans, folded)
	}
}

// A window filter evaluates the rules for that window alone and states
// savings over its length.
func TestRecommendationsForOneWindowAreStatedOverThatWindow(t *testing.T) {
	f := newWeekFixture()
	weekly := Attribute(f.window, f.window.Start, f.turns, fixturePrices)
	hours := Window{Provider: "openai", Pool: "primary", Name: "300m", Minutes: 300, Start: fixtureStart.Add(6*24*time.Hour - 3*time.Hour), ResetAt: fixtureStart.Add(6*24*time.Hour + 2*time.Hour), Series: weekly.Window.Series}
	hours.Series = nil
	for _, p := range f.window.Series {
		if p.At.After(hours.Start) {
			hours.Series = append(hours.Series, p)
		}
	}
	hours.LastAt, hours.UsedPercent = f.window.LastAt, f.window.UsedPercent
	short := Attribute(hours, hours.Start, f.turns, fixturePrices)
	in := Inputs{Results: []Result{weekly, short}, Runs: f.runs, Episodes: f.episodes, Outcomes: f.outcomes, Prices: fixturePrices}

	if all := Recommend(in); len(all.Plans) == 0 || all.Plans[0].Window != "10080m" {
		t.Fatalf("with no filter plans are stated in the weekly window, got %+v", all.Plans)
	}
	in.Only = WindowKey{Provider: "openai", Pool: "primary", Window: "300m"}
	filtered := Recommend(in)
	for _, plan := range filtered.Plans {
		if plan.Window != "300m" {
			t.Errorf("plan %s is stated in %s, want only 300m", plan.TargetID, plan.Window)
		}
		if plan.Use.AtThisRate > plan.Use.ConsumerPoints*(300.0/60.0/(300.0/60.0/7))+1e-9 {
			t.Errorf("plan %s projects %.2f from %.2f beyond the window's length", plan.TargetID, plan.Use.AtThisRate, plan.Use.ConsumerPoints)
		}
	}
	if top, ok := filtered.Top(in.Only); len(filtered.Plans) > 0 && (!ok || top.Rank != 1) {
		t.Errorf("Top(%v) = %+v, %v", in.Only, top, ok)
	}
	for _, status := range filtered.Rules {
		if status.Rule == RuleModel && status.Fired {
			t.Error("R-model states weekly outcomes and must not fire for a 5-hour window")
		}
	}
}

// Each rule's finding names the lever the fixture planted.
func TestRulesTargetThePlantedLevers(t *testing.T) {
	report, result := weekReport(t)
	byRule := map[string]Recommendation{}
	for _, row := range levers(report) {
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
	if model := byRule[RuleModel]; !strings.Contains(model.Action, modelTerra+" to "+modelLuna) || model.Facts.Model.CheaperModel != modelLuna {
		t.Errorf("R-model action = %q facts %+v, want a move from terra to luna", model.Action, model.Facts.Model)
	}
	// A friction problem never costs more than its runs consumed.
	var orchestrator float64
	for i, turn := range result.Turns {
		if turn.RunID == "orchestrator" {
			orchestrator += result.Points[i]
		}
	}
	if friction := byRule[RuleFriction]; friction.Use.ConsumerPoints <= 0 || friction.Use.ConsumerPoints > orchestrator {
		t.Errorf("friction priced at %.2f pts, want within the run's %.2f pts", friction.Use.ConsumerPoints, orchestrator)
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
// extrapolated to a week; its saving stays within what it used. A session
// still running is projected at its rate, and no saving exceeds the
// consumer's projected use.
func TestSavingsAreBoundedByTheConsumersProjectedUse(t *testing.T) {
	f := newWeekFixture()
	short := fixtureRun("vega-e1", "vega-e1", false)
	f.runs[short.ID] = short
	f.turns = append(f.turns, usageSpec{run: short, model: modelTerra, every: 5 * time.Minute, reports: 2, prompt: 248_000, output: 600, from: 10 * time.Hour, until: 30 * time.Hour}.turns()...)
	f.window = truthWindow(f.turns, map[string]float64{modelTerra: 0.4, modelLuna: 0.6}, steady(0.02), 6*24)
	result := Attribute(f.window, f.window.Start, f.turns, fixturePrices)
	report := Recommend(Inputs{Results: []Result{result}, Runs: f.runs, Episodes: f.episodes, Outcomes: f.outcomes, Prices: fixturePrices})

	var sawShort, sawActive bool
	for _, plan := range report.Plans {
		for _, row := range levers(Report{Plans: []Plan{plan}}) {
			if row.Rule == RuleModel {
				continue // stated from the 14-day outcome history, already weekly
			}
			if row.Saved > row.Use.AtThisRate+1e-9 {
				t.Errorf("%s %s saves %.2f pts, more than the consumer's projected %.2f", row.Rule, row.TargetID, row.Saved, row.Use.AtThisRate)
			}
			switch {
			case row.Rule == RuleContext && row.TargetID == short.ID:
				sawShort = true
				if row.Use.StillRunning || row.Saved > row.Use.ConsumerPoints || row.Use.AtThisRate != row.Use.ConsumerPoints {
					t.Errorf("stopped session extrapolated: saves %.2f, projected %.2f, used %.2f, running=%v", row.Saved, row.Use.AtThisRate, row.Use.ConsumerPoints, row.Use.StillRunning)
				}
			case row.Rule == RuleContext && row.TargetID == "research":
				sawActive = true
				if !row.Use.StillRunning || row.Use.AtThisRate <= row.Use.ConsumerPoints {
					t.Errorf("running session not projected: projected %.2f, used %.2f", row.Use.AtThisRate, row.Use.ConsumerPoints)
				}
			}
		}
	}
	if !sawShort || !sawActive {
		t.Fatalf("fixture rows missing: short=%v active=%v", sawShort, sawActive)
	}
}

// W8b: R-context counts provider requests, not turns; the action and the
// facts say so (a Codex turn makes many requests, each re-reading the context).
func TestContextRuleNamesItsCountRequests(t *testing.T) {
	report, _ := weekReport(t)
	for _, row := range levers(report) {
		if row.Rule != RuleContext {
			continue
		}
		if !strings.Contains(row.Action, " requests)") || strings.Contains(row.Action, "turns") || row.Facts.Context.Requests == 0 {
			t.Fatalf("action = %q facts %+v, want a request count", row.Action, row.Facts.Context)
		}
		return
	}
	t.Fatal("no R-context lever")
}

// P2-A4: a recurring problem is described in one plain sentence with its owner
// and a typical duration, never the generic "review the fingerprint" text; a
// dismissed problem is listed but not recommended.
func TestFrictionProblemsAreDescribedAndDismissedOnesAreNotRecommended(t *testing.T) {
	f := newWeekFixture()
	for i := range f.episodes {
		f.episodes[i].OwnerScenario, f.episodes[i].OwnerCommand, f.episodes[i].WallClockMS = "search-hub", "search-hub query", 45_000
		f.episodes[i].Pattern, f.episodes[i].RepeatedElement = "fallback-after-capability", ""
	}
	result := Attribute(f.window, f.window.Start, f.turns, fixturePrices)
	in := Inputs{Results: []Result{result}, Runs: f.runs, Episodes: f.episodes, Outcomes: f.outcomes, Prices: fixturePrices}
	report := Recommend(in)
	if len(report.Problems) != 1 {
		t.Fatalf("problems = %d, want the one recurring fingerprint", len(report.Problems))
	}
	p := report.Problems[0]
	if p.Description != "`search-hub query` works, then the agent falls back to a tool outside Vrooli for the same job." ||
		p.Owner != "search-hub query" || p.OwnerKind != OwnerCommand || p.Episodes != 5 || p.Runs != 1 || p.TypicalMS != 45_000 || p.WallClockMS != 5*45_000 || p.Points <= 0 {
		t.Fatalf("problem = %+v", p)
	}
	var recommended bool
	for _, row := range levers(report) {
		if row.Rule == RuleFriction {
			recommended = true
			if !strings.Contains(row.Action, p.Description[:40]) || strings.Contains(row.Action, "Fix the recurring read-then-reread") {
				t.Errorf("friction action %q does not reuse the problem description", row.Action)
			}
		}
	}
	if !recommended {
		t.Fatal("an open problem must be recommended")
	}
	in.Dismissed = map[string]bool{p.Fingerprint: true}
	dismissed := Recommend(in)
	if len(dismissed.Problems) != 1 {
		t.Error("a dismissed problem stays listed")
	}
	for _, row := range levers(dismissed) {
		if row.Rule == RuleFriction {
			t.Errorf("dismissed problem recommended: %+v", row)
		}
	}
}

func TestDescribeStatesEveryPatternInOneSentenceWithoutInventingACause(t *testing.T) {
	patterns := []string{"stall", "read-then-reread", "wrong-primitive", "repeated-work", "command-failure", "help-recovery", "poll-loop", "oscillation", "edit-revert", "wait-misuse", "blocked-then-abandoned", "guidance-repair", "handoff-continuation", "fallback-after-capability", "capability-abandoned", "retry-after-failure", "flag-hunting", "abandoned-command-family", "time-to-first-success", "something-new"}
	for _, pattern := range patterns {
		got := Describe(pattern, "search-hub query", "rg", "rg", 0)
		if got == "" || strings.Contains(got, "fingerprint") || !strings.HasSuffix(got, ".") || !strings.Contains(got, "`rg`") {
			t.Errorf("Describe(%s) = %q, want plain sentences that name the command", pattern, got)
		}
	}
	if got := Describe("wrong-primitive", "search-hub query", "", "rg", 0); got != "Agents run `rg` repeatedly instead of the preferred `search-hub query`." {
		t.Errorf("wrong-primitive = %q", got)
	}
	// Two problems of one pattern that differ only by command read differently.
	a, b := Describe("handoff-continuation", "", "", "", 0), Describe("handoff-continuation", "", "search-hub", "search-hub query", 0)
	if a == b || b != "The agent declares it is done, then keeps working with no new instruction. Involves `search-hub query`." {
		t.Errorf("handoff descriptions = %q / %q", a, b)
	}
	// A description states what happens; durations are separate fields.
	for _, pattern := range []string{"stall", "read-then-reread", "time-to-first-success"} {
		if got := Describe(pattern, "", "", "", 0); strings.ContainsAny(got, "0123456789") {
			t.Errorf("Describe(%s) = %q must not embed a duration", pattern, got)
		}
	}
	if got := Describe("wait-misuse", "", "", "", 4); got != "An agent repeats a short wait 4 times instead of one blocking wait." {
		t.Errorf("wait-misuse = %q", got)
	}
	for ms, want := range map[int64]string{450: "450 ms", 6075: "6 s", 185_000: "3m 05s", 11_478_000: "3h 11m"} {
		if got := FormatDuration(ms); got != want {
			t.Errorf("FormatDuration(%d) = %q, want %q", ms, got, want)
		}
	}
	_ = fmt.Sprint
}

// A problem's total is the sum of its episodes' own cost (never more than
// episodes × the longest), its typical is their median, and a handoff problem
// reports work, not lost time.
func TestProblemTotalsAreTheSumOfEpisodeCostsAndHandoffsReportWorkNotTime(t *testing.T) {
	f := newWeekFixture()
	costs := []int64{8_000, 8_000, 9_000, 7_000, 8_000, 8_000, 9_000, 8_000, 7_000, 8_000}
	build := func(pattern string, ms []int64) Report {
		f.episodes = nil
		for i, c := range ms {
			f.episodes = append(f.episodes, Episode{RunID: "orchestrator", EpisodeID: fmt.Sprintf("%s-%d", pattern, i), Pattern: pattern, Fingerprint: pattern, Turns: 2, Tokens: 1000, WallClockMS: c})
		}
		result := Attribute(f.window, f.window.Start, f.turns, fixturePrices)
		return Recommend(Inputs{Results: []Result{result}, Runs: f.runs, Episodes: f.episodes, Prices: fixturePrices})
	}
	p := build("read-then-reread", costs).Problems[0]
	var sum, longest int64
	for _, c := range costs {
		sum += c
		longest = max(longest, c)
	}
	if p.WallClockMS != sum || p.WallClockMS > int64(p.Episodes)*longest || p.TypicalMS != 8_000 || !p.TimeLost || p.Tokens != 10_000 {
		t.Fatalf("total %d (sum %d), typical %d, lost %v, tokens %d", p.WallClockMS, sum, p.TypicalMS, p.TimeLost, p.Tokens)
	}
	h := build("handoff-continuation", []int64{0, 0, 0}).Problems[0]
	if h.TimeLost || h.WallClockMS != 0 || h.EpisodeTurns != 6 || h.Tokens != 3000 || strings.Contains(h.Description, " ms") {
		t.Fatalf("handoff problem = %+v, want work reported and no lost time", h)
	}
}

// M1: a saving is stated over the whole horizon, but only the time left before
// the reset can still be saved, and only by a target that keeps running.
func TestRecoverableSavingIsThePartOfTheHorizonLeftForRunningTargets(t *testing.T) {
	f := newWeekFixture()
	result := Attribute(f.window, f.window.Start, f.turns, fixturePrices)
	now := f.window.ResetAt.Add(-24 * time.Hour) // 6 of 7 days elapsed: 14.3% of the window is left
	report := Recommend(Inputs{Results: []Result{result}, Runs: f.runs, Episodes: f.episodes, Outcomes: f.outcomes, Prices: fixturePrices, Now: now})
	share := 24.0 / 168
	var running, stopped int
	for _, plan := range report.Plans {
		if math.Abs(plan.Use.RemainingHours-24) > 1e-9 || math.Abs(plan.Use.RemainingShare-share) > 1e-9 {
			t.Fatalf("%s has %.2fh (%.3f) left, want 24h", plan.TargetID, plan.Use.RemainingHours, plan.Use.RemainingShare)
		}
		want := 0.0
		if plan.Use.StillRunning {
			want = plan.Saved * share
			running++
		} else {
			stopped++
		}
		if math.Abs(plan.Recoverable-want) > 1e-9 {
			t.Errorf("%s recoverable %.4f, want %.4f (saved %.3f, running %v)", plan.TargetID, plan.Recoverable, want, plan.Saved, plan.Use.StillRunning)
		}
		if plan.Recoverable > plan.Saved+1e-9 {
			t.Errorf("%s recovers more than it saves", plan.TargetID)
		}
		for _, lever := range plan.Levers {
			if lever.Use.StillRunning && math.Abs(lever.Recoverable-lever.Saved*share) > 1e-9 {
				t.Errorf("%s %s lever recoverable %.4f, want %.4f", plan.TargetID, lever.Rule, lever.Recoverable, lever.Saved*share)
			}
		}
	}
	if running == 0 {
		t.Fatalf("the fixture needs running plans, got %d running and %d stopped", running, stopped)
	}
	if got := recoverable(5, Use{StillRunning: false, RemainingShare: 0.5}); got != 0 {
		t.Errorf("a target that stopped has nothing left to cut, got %.2f", got)
	}
	// With the window just started nearly all of the saving is recoverable.
	early := Recommend(Inputs{Results: []Result{result}, Runs: f.runs, Episodes: f.episodes, Outcomes: f.outcomes, Prices: fixturePrices, Now: f.window.Start.Add(time.Hour)})
	top := early.Plans[0]
	if top.Recoverable < 0.99*top.Saved {
		t.Errorf("an hour in, %.3f of %.3f is recoverable", top.Recoverable, top.Saved)
	}
}

// m2: friction plans are a cross-cut view and say so; session plans do not.
func TestFrictionPlansAreMarkedCrossCut(t *testing.T) {
	report, _ := weekReport(t)
	var friction, other int
	for _, plan := range append(append([]Plan(nil), report.Plans...), report.Folded...) {
		if plan.CrossCut != (plan.TargetKind == "friction") {
			t.Errorf("%s/%s cross-cut = %v", plan.TargetKind, plan.TargetID, plan.CrossCut)
		}
		if plan.CrossCut {
			friction++
		} else {
			other++
		}
	}
	if friction == 0 || other == 0 {
		t.Fatalf("friction %d other %d plans", friction, other)
	}
}

func TestTimeKindsSayWhatTheCostMeasures(t *testing.T) {
	for pattern, want := range map[string]string{
		"stall": TimeIdle, "handoff-continuation": TimeNone, "blocked-then-abandoned": TimeNone, "time-to-first-success": TimeElapsed,
		"poll-loop": TimeElapsed, "read-then-reread": TimeWasted, "retry-after-failure": TimeWasted, "fallback-after-capability": TimeWasted,
	} {
		if got := timeKind(pattern); got != want {
			t.Errorf("%s time kind = %s, want %s", pattern, got, want)
		}
	}
}
