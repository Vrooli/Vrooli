package usageattribution

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/runsignal"

	"agent-manager/internal/domain"
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
		// R-floor and R-model-mix are informational: they claim no saving.
		if (row.Saved <= 0 && row.Rule != RuleFloor && row.Rule != RuleModelMix) || row.Confidence == "" || row.Action == "" || len(row.Evidence) == 0 || set != 1 {
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
	want := map[string]string{RuleContext: "research", RuleLifetime: "research", RuleModelMix: "research", RuleIdle: "heartbeat", RuleImage: "ui-review", RuleModel: "workflow_node:draft", RuleFriction: "failure:rg"}
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
	var quiet []Turn // one luna turn: no cheaper tier, no growth, nothing to flag
	for _, turn := range f.turns {
		if turn.Model == modelLuna {
			quiet = append(quiet, turn)
			break
		}
	}
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
	f.turns = append(f.turns, usageSpec{run: short, model: modelTerra, every: 5 * time.Minute, reports: 2, prompt: 50_000, growth: 1_000, output: 600, from: 10 * time.Hour, until: 30 * time.Hour}.turns()...)
	f.window = truthWindow(f.turns, map[string]float64{modelTerra: 0.4, modelLuna: 0.6}, steady(0.02), 6*24)
	result := Attribute(f.window, f.window.Start, f.turns, fixturePrices)
	report := Recommend(Inputs{Results: []Result{result}, Runs: f.runs, Episodes: f.episodes, Outcomes: f.outcomes, Prices: fixturePrices, Now: f.window.LastAt.Add(5 * time.Minute)})

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
		f.episodes[i].Pattern, f.episodes[i].RepeatedElement = "command-failure", ""
	}
	result := Attribute(f.window, f.window.Start, f.turns, fixturePrices)
	in := Inputs{Results: []Result{result}, Runs: f.runs, Episodes: f.episodes, Outcomes: f.outcomes, Prices: fixturePrices}
	report := Recommend(in)
	if len(report.Problems) != 1 {
		t.Fatalf("problems = %d, want the one recurring fingerprint", len(report.Problems))
	}
	p := report.Problems[0]
	if p.Description != "`search-hub query` failed." ||
		p.Owner != "search-hub query" || p.OwnerKind != OwnerCommand || p.Episodes != 5 || p.Runs != 1 || p.TypicalMS != 45_000 || p.WallClockMS != 5*45_000 || p.Points <= 0 {
		t.Fatalf("problem = %+v", p)
	}
	var recommended bool
	for _, row := range levers(report) {
		if row.Rule == RuleFriction {
			recommended = true
			if !strings.Contains(row.Action, strings.TrimSuffix(p.Description, ".")) || strings.Contains(row.Action, "Fix the recurring wait-misuse") {
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
	for _, rule := range dismissed.Rules {
		if rule.Rule == RuleFriction && !strings.Contains(rule.Reason, "dismissed") {
			t.Errorf("friction rule reason %q, want it to say the problems are dismissed", rule.Reason)
		}
	}
}

func TestDescribeStatesEveryPatternInOneSentenceWithoutInventingACause(t *testing.T) {
	patterns := []string{"stall", "read-then-reread", "wrong-primitive", "repeated-work", "command-failure", "help-recovery", "poll-loop", "oscillation", "edit-revert", "wait-misuse", "blocked-then-abandoned", "guidance-repair", "handoff-continuation", "fallback-after-capability", "capability-abandoned", "retry-after-failure", "flag-hunting", "abandoned-command-family", "time-to-first-success", "something-new"}
	for _, pattern := range patterns {
		got := Describe(pattern, "search-hub query", "rg", "rg", 0)
		if got == "" || strings.Contains(got, "fingerprint") || (pattern != "command-failure" && !strings.HasSuffix(got, ".")) || !strings.Contains(got, "`rg`") {
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
// episodes × the longest) and its typical is their median.
func TestProblemTotalsAreTheSumOfEpisodeCosts(t *testing.T) {
	f := newWeekFixture()
	costs := []int64{8_000, 8_000, 9_000, 7_000, 8_000, 8_000, 9_000, 8_000, 7_000, 8_000}
	f.episodes = nil
	for i, c := range costs {
		f.episodes = append(f.episodes, Episode{RunID: "orchestrator", EpisodeID: fmt.Sprintf("wait-%d", i), Pattern: "command-failure", Fingerprint: "wait-misuse", Turns: 2, Tokens: 1000, WallClockMS: c})
	}
	result := Attribute(f.window, f.window.Start, f.turns, fixturePrices)
	p := Recommend(Inputs{Results: []Result{result}, Runs: f.runs, Episodes: f.episodes, Prices: fixturePrices}).Problems[0]
	var sum, longest int64
	for _, c := range costs {
		sum += c
		longest = max(longest, c)
	}
	if p.WallClockMS != sum || p.WallClockMS > int64(p.Episodes)*longest || p.TypicalMS != 8_000 || p.TimeKind != runsignal.TimeRedundant || p.Tokens != 10_000 {
		t.Fatalf("total %d (sum %d), typical %d, kind %s, tokens %d", p.WallClockMS, sum, p.TypicalMS, p.TimeKind, p.Tokens)
	}
}

// M1: a saving is stated over the whole horizon, but only the time left before
// the reset can still be saved, and only by a target that keeps running.
func TestRecoverableSavingIsTheFutureShareOfRunningTargets(t *testing.T) {
	f := newWeekFixture()
	result := Attribute(f.window, f.window.Start, f.turns, fixturePrices)
	var now time.Time // the fixture's consumers have just made their newest attributed request
	for _, turn := range result.Turns {
		if turn.At.After(now) {
			now = turn.At
		}
	}
	report := Recommend(Inputs{Results: []Result{result}, Runs: f.runs, Episodes: f.episodes, Outcomes: f.outcomes, Prices: fixturePrices, Now: now})
	left := f.window.ResetAt.Sub(now).Hours()
	var running, stopped int
	for _, plan := range report.Plans {
		if math.Abs(plan.Use.RemainingHours-left) > 1e-9 {
			t.Fatalf("%s has %.2fh left, want %.2fh", plan.TargetID, plan.Use.RemainingHours, left)
		}
		want := 0.0
		if plan.Use.StillRunning {
			want = plan.Saved * (plan.Use.AtThisRate - plan.Use.ConsumerPoints) / plan.Use.AtThisRate
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
	}
	if running == 0 {
		t.Fatalf("the fixture needs running plans, got %d running and %d stopped", running, stopped)
	}
	if got := recoverable(5, Use{StillRunning: false, ConsumerPoints: 1, AtThisRate: 4}); got != 0 {
		t.Errorf("a target that stopped has nothing left to cut, got %.2f", got)
	}
	// Late in the window little of the horizon is left to extrapolate over, so
	// less of a running target's saving is still to be had than early on.
	early := Recommend(Inputs{Results: []Result{result}, Runs: f.runs, Episodes: f.episodes, Outcomes: f.outcomes, Prices: fixturePrices, Now: f.window.Start.Add(time.Hour)})
	for _, plan := range early.Plans {
		if plan.Use.StillRunning && plan.Recoverable > plan.Saved+1e-9 {
			t.Errorf("%s recovers more than it saves", plan.TargetID)
		}
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

func TestTimeKindsComeFromThePatternTable(t *testing.T) {
	for pattern, want := range map[string]string{"stall": runsignal.TimeIdle, "command-failure": runsignal.TimeRedundant, "wait-misuse": runsignal.TimeRedundant} {
		spec, ok := runsignal.PatternByID(pattern)
		if !ok || spec.TimeKind != want {
			t.Errorf("%s time kind = %q, want %s", pattern, spec.TimeKind, want)
		}
	}
}

// handResult builds a weekly-window attribution from turns whose points are
// given directly, so projection and lever tests do not depend on calibration.
func handResult(provider, runner, model string, used float64, turns []Turn, points float64) Result {
	r := Result{Window: Window{Provider: provider, Pool: "primary", Name: "10080m", Minutes: 10080, Start: fixtureStart, ResetAt: fixtureStart.Add(7 * 24 * time.Hour), UsedPercent: used}, From: fixtureStart, Observed: used}
	for _, turn := range turns {
		turn.Runner, turn.Model = runner, model
		if turn.Reports == 0 {
			turn.Reports = 1
		}
		r.Turns = append(r.Turns, turn)
		r.Points = append(r.Points, points)
		r.Weighted = append(r.Weighted, float64(turn.Prompt()))
		r.To = turn.At
	}
	return r
}

func steadyTurns(run string, from, to, every time.Duration, prompt int) []Turn {
	var out []Turn
	for at := from; at <= to; at += every {
		out = append(out, Turn{At: fixtureStart.Add(at), RunID: run, Tokens: domain.UsageCounters{CacheReadTokens: prompt - 2000, InputTokens: 2000}})
	}
	return out
}

// P3-F6: active is judged against now, never against another consumer's newest
// turn. A session idle for 3 hours is finished: it projects to its points, has
// nothing recoverable, and the active sessions' projected future fits in the
// window's headroom.
func TestProjectionJudgesActivityAgainstNowAndCapsAtHeadroom(t *testing.T) {
	idle := steadyTurns("idle", 0, 5*time.Hour, 10*time.Minute, 50_000)
	live := steadyTurns("live", 0, 20*time.Hour, 10*time.Minute, 50_000)
	result := handResult("openai", codex, modelTerra, 96, append(idle, live...), 0.1)
	runs := map[string]Run{"idle": fixtureRun("idle", "idle", false), "live": fixtureRun("live", "live", false)}
	now := fixtureStart.Add(20*time.Hour + 5*time.Minute)
	stats := sessionStatsFor(&result, runs)
	result.proj = newProjCtx(&result, now, stats)
	var future float64
	for _, s := range stats {
		p := s.project(&result)
		switch s.key {
		case "idle":
			if p.active || p.weekly != p.points {
				t.Errorf("idle session: active=%v weekly %.2f points %.2f, want finished and weekly == points", p.active, p.weekly, p.points)
			}
			if got := recoverable(3, p.use(&result, now)); got != 0 {
				t.Errorf("finished session recoverable = %.2f, want 0", got)
			}
		case "live":
			if !p.active || p.weekly <= p.points {
				t.Errorf("live session: active=%v weekly %.2f points %.2f, want a decaying future", p.active, p.weekly, p.points)
			}
		}
		future += p.future()
	}
	if headroom := 100 - result.Window.UsedPercent; future > headroom+1e-9 {
		t.Errorf("projected future %.2f exceeds the window's headroom %.2f", future, headroom)
	}
}

// P3-F7/F10: workers that start at the floor and grow are priced by restarting
// them net of the rebuild; workers that never grow have nothing to restart.
// There is no fan-out rule and no "short brief" advice.
func TestLifetimeRuleNetsRebuildAndFlatWorkersDoNotFire(t *testing.T) {
	for _, rule := range Rules {
		if rule == "R-fanout" {
			t.Fatal("R-fanout must be replaced by R-lifetime")
		}
	}
	grow := func(run string, flat bool) []Turn {
		turns := steadyTurns(run, 0, 100*time.Hour, 30*time.Minute, 50_000)
		for i := range turns {
			prompt := 50_000 + 1_000*i
			if flat {
				prompt = 90_000
			}
			turns[i].Tokens = domain.UsageCounters{CacheReadTokens: prompt - 2000, InputTokens: 2000}
		}
		return turns
	}
	for _, flat := range []bool{false, true} {
		var turns []Turn
		runs := map[string]Run{}
		for i := 0; i < 4; i++ {
			id := fmt.Sprintf("w%d", i)
			runs[id] = fixtureRun(id, "s", true)
			turns = append(turns, grow(id, flat)...)
		}
		result := handResult("openai", codex, modelTerra, 40, turns, 0.05)
		report := Recommend(Inputs{Results: []Result{result}, Runs: runs, Prices: fixturePrices})
		var lifetime *Recommendation
		for _, row := range levers(report) {
			if strings.Contains(row.Action+row.Rationale, "short brief") {
				t.Errorf("flat=%v: %s still advises a short brief: %s", flat, row.Rule, row.Action)
			}
			if row.Rule == RuleLifetime {
				lifetime = &row
			}
		}
		if flat != (lifetime == nil) {
			t.Fatalf("flat=%v: lifetime lever = %v, want it only for growing workers", flat, lifetime != nil)
		}
		if lifetime != nil && lifetime.Saved >= lifetime.Facts.Fanout.SubAgentPoints {
			t.Errorf("lifetime saves %.2f of the workers' %.2f pts; the rebuild must be netted", lifetime.Saved, lifetime.Facts.Fanout.SubAgentPoints)
		}
	}
}

// P3-F7: an interactive Opus session (no workload) gets a model-mix range
// that is unproven, not a promised saving.
func TestModelMixCoversInteractiveSessionsAsAnUnprovenRange(t *testing.T) {
	turns := steadyTurns("chat", 0, 10*time.Hour, 10*time.Minute, 80_000)
	result := handResult("anthropic", "claude-code", "claude-opus-4-7", 30, turns, 0.15)
	runs := map[string]Run{"chat": {ID: "chat", Source: SourceIndexed, Runner: "claude-code", Session: "chat"}}
	report := Recommend(Inputs{Results: []Result{result}, Runs: runs, Prices: fixturePrices})
	for _, row := range levers(report) {
		if row.Rule != RuleModelMix {
			continue
		}
		if row.Confidence != "unproven" || row.Saved != 0 || row.Facts.Model.ExpensiveWeeklyPoints*(1-row.Facts.Model.PriceRatio) <= 0 || !strings.Contains(row.Rationale, "prior") || !strings.Contains(row.Rationale, "0 to") {
			t.Fatalf("model-mix row = %+v, want an unproven range claiming no saving, labelled a prior", row)
		}
		for _, plan := range report.Plans {
			for _, lever := range plan.Levers {
				if lever.Rule == RuleModelMix {
					return
				}
			}
		}
		t.Fatal("the model-mix range was folded away; it claims no saving but must stay visible")
	}
	t.Fatal("no R-model-mix lever for an interactive Opus session")
}

// SPEC 11.2: usage friction projects like usage recommend: a session idle for
// 3h is finished, so its friction has nothing recoverable and the saving fits
// the window's headroom.
func TestFrictionProjectsAtNowAndCapsAtHeadroom(t *testing.T) {
	idle := steadyTurns("idle", 0, 5*time.Hour, 10*time.Minute, 50_000)
	result := handResult("openai", codex, modelTerra, 96, idle, 0.1)
	runs := map[string]Run{"idle": fixtureRun("idle", "idle", false)}
	var episodes []Episode
	for i := 0; i < 10; i++ {
		episodes = append(episodes, Episode{RunID: "idle", EpisodeID: fmt.Sprintf("e%d", i), Pattern: "command-failure", Fingerprint: "fp", Turns: 2, Tokens: 1000, WallClockMS: 8_000})
	}
	now := fixtureStart.Add(8*time.Hour + 5*time.Minute)
	in := Inputs{Results: []Result{result}, Runs: runs, Episodes: episodes, Prices: fixturePrices, Now: now}
	primary, _ := prepare(in)
	problems := frictionProblems(in, primary)
	if len(problems) != 1 {
		t.Fatalf("problems = %d, want 1", len(problems))
	}
	p := problems[0]
	if p.EstRecoverable != 0 {
		t.Errorf("idle session recoverable = %.2f, want 0", p.EstRecoverable)
	}
	if headroom := 100 - result.Window.UsedPercent; p.EstSaved > headroom+1e-9 {
		t.Errorf("saving %.2f exceeds window headroom %.2f", p.EstSaved, headroom)
	}
}

// R-lifetime credits a worker only with growth from its own first request: a
// worker that starts above the median floor and stays flat is not restarted.
func TestLifetimeDoesNotCreditAWorkersStartingExcess(t *testing.T) {
	var turns []Turn
	runs := map[string]Run{}
	for i := 0; i < 4; i++ {
		id := fmt.Sprintf("w%d", i)
		runs[id] = fixtureRun(id, "s", true)
		prompt := 50_000
		if i == 3 {
			prompt = 90_000 // starts far above the 50k floor and never grows
		}
		ts := steadyTurns(id, 0, 100*time.Hour, 30*time.Minute, prompt)
		turns = append(turns, ts...)
	}
	result := handResult("openai", codex, modelTerra, 40, turns, 0.05)
	report := Recommend(Inputs{Results: []Result{result}, Runs: runs, Prices: fixturePrices})
	for _, row := range levers(report) {
		if row.Rule == RuleLifetime {
			t.Fatalf("lifetime lever fired for flat workers: %+v", row.Facts.Fanout)
		}
	}
}

// The first request is what sets a run's floor: a many-request turn only gives
// an average, so an exact one-request turn wins even when it comes later.
func TestFirstRequestFloorPrefersAOneRequestTurn(t *testing.T) {
	turns := []Turn{
		{At: fixtureStart, RunID: "r", Reports: 5, Tokens: domain.UsageCounters{CacheReadTokens: 500_000}},
		{At: fixtureStart.Add(time.Minute), RunID: "r", Reports: 1, Tokens: domain.UsageCounters{CacheReadTokens: 20_000}},
	}
	result := handResult("openai", codex, modelTerra, 10, turns, 0.1)
	stats := sessionStatsFor(&result, map[string]Run{"r": fixtureRun("r", "s", false)})
	if got := stats[0].runs["r"].firstPrompt; got != 20_000 {
		t.Fatalf("first-request prompt = %v, want 20000 from the one-request turn, not the 100000 average", got)
	}
}

// A session that switches models is priced per turn's model, and each model's
// row counts only the runs that used that model.
func TestModelMixPricesEachTurnsModelAndCountsRunsPerModel(t *testing.T) {
	const opus, terra = "claude-opus-4-7", modelTerra
	var turns []Turn
	add := func(run, model string, n int) {
		for i := 0; i < n; i++ {
			turns = append(turns, Turn{At: fixtureStart.Add(time.Duration(len(turns)) * time.Minute), RunID: run, Model: model, Reports: 1, Tokens: domain.UsageCounters{CacheReadTokens: 50_000}})
		}
	}
	add("a", opus, 20) // a starts on opus and switches to terra
	add("a", terra, 20)
	add("b", terra, 20)
	add("c", opus, 20)
	result := Result{Window: Window{Provider: "anthropic", Pool: "primary", Name: "10080m", Minutes: 10080, Start: fixtureStart, ResetAt: fixtureStart.Add(7 * 24 * time.Hour), UsedPercent: 60}, From: fixtureStart, Observed: 60}
	for _, turn := range turns {
		turn.Runner = "claude-code"
		result.Turns = append(result.Turns, turn)
		result.Points = append(result.Points, 0.5)
		result.Weighted = append(result.Weighted, float64(turn.Prompt()))
		result.To = turn.At
	}
	runs := map[string]Run{"a": fixtureRun("a", "s", false), "b": fixtureRun("b", "s", false), "c": fixtureRun("c", "s", false)}
	report := Recommend(Inputs{Results: []Result{result}, Runs: runs, Prices: fixturePrices})
	got := map[string]*ModelFacts{}
	var use Use
	for _, row := range levers(report) {
		if row.Rule == RuleModelMix {
			got[row.Facts.Model.ExpensiveModel] = row.Facts.Model
			use = row.Use
		}
	}
	for model, want := range map[string]struct {
		runs   int
		points float64
	}{opus: {2, 20}, terra: {2, 20}} {
		f := got[model]
		if f == nil || f.ExpensiveRuns != want.runs || math.Abs(f.ExpensiveWeeklyPoints-min(want.points*use.AtThisRate/use.ConsumerPoints, use.AtThisRate)) > 1e-6 {
			t.Fatalf("%s facts = %+v, want %d runs and %v period points scaled to the week (only its own turns)", model, f, want.runs, want.points)
		}
	}
}

// The floor share is in the session's weighted units: a run's first request
// writes the floor to cache (creation weight), the rest re-read it.
func TestFloorShareIsInWeightedUnits(t *testing.T) {
	turns := steadyTurns("r", 0, 59*time.Minute, time.Minute, 50_000)
	result := handResult("openai", codex, modelTerra, 40, turns, 0.1)
	runs := map[string]Run{"r": fixtureRun("r", "s", false)}
	report := Recommend(Inputs{Results: []Result{result}, Runs: runs, Prices: fixturePrices})
	creation, _ := weightsOf(&result, codex, modelTerra)
	want := (creation + 59) / 60
	for _, row := range levers(report) {
		if row.Rule == RuleFloor {
			if math.Abs(row.Facts.Context.CachedShare-want) > 1e-9 {
				t.Fatalf("floor share = %v, want %v (creation weight %v on the first request)", row.Facts.Context.CachedShare, want, creation)
			}
			return
		}
	}
	t.Fatal("no R-floor lever")
}

// A model move priced from tier priors keeps its point estimate but is never
// more than low confidence and reads as a ceiling, not a measurement.
func TestModelRuleSaysWhenThePriceRatioIsAPrior(t *testing.T) {
	f := newWeekFixture()
	prior := func(runner, model string) Price {
		p := fixturePrices(runner, model)
		p.Prior = true
		return p
	}
	find := func(prices Prices) Recommendation {
		result := Attribute(f.window, f.window.Start, f.turns, prices)
		for _, row := range levers(Recommend(Inputs{Results: []Result{result}, Runs: f.runs, Episodes: f.episodes, Outcomes: f.outcomes, Prices: prices})) {
			if row.Rule == RuleModel {
				return row
			}
		}
		t.Fatal("R-model did not fire")
		return Recommendation{}
	}
	measured, guessed := find(fixturePrices), find(prior)
	if strings.Contains(measured.Rationale, "prior") || strings.Contains(measured.Action, "up to") {
		t.Errorf("measured prices must not be flagged: %q / %q", measured.Action, measured.Rationale)
	}
	if guessed.Confidence != "low" || !strings.Contains(guessed.Action, "up to") || !strings.Contains(guessed.Rationale, "price ratio is a prior, not measured") {
		t.Errorf("prior-priced move = %+v", guessed)
	}
	if guessed.Facts.Model.PriceRatio != measured.Facts.Model.PriceRatio {
		t.Errorf("point estimate changed: %v vs %v", guessed.Facts.Model.PriceRatio, measured.Facts.Model.PriceRatio)
	}
}

// Friction projects a run over the run's own span, like its session lever: a
// run that starts late in the window is not stretched back to the window start.
func TestFrictionProjectsFromTheRunsFirstTurn(t *testing.T) {
	turns := steadyTurns("late", 60*time.Hour, 70*time.Hour, 10*time.Minute, 50_000)
	result := handResult("openai", codex, modelTerra, 40, turns, 0.1)
	runs := map[string]Run{"late": fixtureRun("late", "late", false)}
	var episodes []Episode
	for i := 0; i < 10; i++ {
		episodes = append(episodes, Episode{RunID: "late", EpisodeID: fmt.Sprintf("e%d", i), Pattern: "command-failure", Fingerprint: "fp", Turns: 2, Tokens: 1000, WallClockMS: 8_000})
	}
	in := Inputs{Results: []Result{result}, Runs: runs, Episodes: episodes, Prices: fixturePrices, Now: fixtureStart.Add(70*time.Hour + time.Minute)}
	primary, _ := prepare(in)
	problems := frictionProblems(in, primary)
	if len(problems) != 1 {
		t.Fatalf("problems = %d, want 1", len(problems))
	}
	if got := problems[0].use.hours; math.Abs(got-10) > 0.01 {
		t.Fatalf("friction rate basis = %.2fh, want the run's own 10h span", got)
	}
}

// An active unproven model move states its saving in the same weekly terms as
// its neighbours, not the period's points.
func TestModelMixStoresAWeeklyFigure(t *testing.T) {
	turns := steadyTurns("chat", 0, 10*time.Hour, 10*time.Minute, 80_000)
	result := handResult("anthropic", "claude-code", "claude-opus-4-7", 30, turns, 0.15)
	runs := map[string]Run{"chat": {ID: "chat", Source: SourceIndexed, Runner: "claude-code", Session: "chat"}}
	report := Recommend(Inputs{Results: []Result{result}, Runs: runs, Prices: fixturePrices})
	for _, row := range levers(report) {
		if row.Rule != RuleModelMix {
			continue
		}
		if got := row.Facts.Model.ExpensiveWeeklyPoints; row.Use.AtThisRate <= row.Use.ConsumerPoints || math.Abs(got-row.Use.AtThisRate) > 1e-6 {
			t.Fatalf("weekly points = %.2f, want the weekly projection %.2f (period %.2f)", got, row.Use.AtThisRate, row.Use.ConsumerPoints)
		}
		return
	}
	t.Fatal("no R-model-mix lever")
}

// A floor-only plan stays visible when the floor is a material share of a
// target that used real points, even though it claims no saving.
func TestFloorOnlyPlanIsKeptWhenMaterial(t *testing.T) {
	turns := steadyTurns("r", 0, 59*time.Minute, time.Minute, 50_000)
	result := handResult("openai", codex, modelTerra, 40, turns, 0.1)
	runs := map[string]Run{"r": fixtureRun("r", "s", false)}
	report := Recommend(Inputs{Results: []Result{result}, Runs: runs, Prices: fixturePrices})
	for _, plan := range report.Plans {
		for _, lever := range plan.Levers {
			if lever.Rule == RuleFloor {
				return
			}
		}
	}
	t.Fatal("a material floor-only plan was folded away")
}
