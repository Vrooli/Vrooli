package usageattribution

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/interventions"
	"agent-manager/internal/opportunities"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/tokenaccounting"
	"agent-manager/internal/toolfriction"
)

func weekReport(t *testing.T, now time.Time) (Report, Result) {
	t.Helper()
	f := newWeekFixture()
	result := Attribute(f.window, f.window.Start, f.turns, fixturePrices)
	return Recommend(Inputs{Results: []Result{result}, Runs: f.runs, Outcomes: f.outcomes, Prices: fixturePrices, Now: now}), result
}

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

// growingTurns is one request a minute from start whose prompt grows by step
// from 50k and, every compactEvery requests (0: never), drops back to 60k.
func growingTurns(run string, start time.Duration, n, step, compactEvery int) []Turn {
	var out []Turn
	prompt := 50_000
	for i := 0; i < n; i++ {
		if compactEvery > 0 && i > 0 && i%compactEvery == 0 {
			prompt = 60_000
		}
		out = append(out, Turn{At: fixtureStart.Add(start + time.Duration(i)*time.Minute), RunID: run, Reports: 1, Tokens: domain.UsageCounters{CacheReadTokens: prompt - 2000, InputTokens: 2000}})
		prompt += step
	}
	return out
}

func byKind(report Report, kind string) []Action {
	var out []Action
	for _, a := range report.Actions {
		if a.Kind == kind {
			out = append(out, a)
		}
	}
	return out
}

// P4-A6: one list, ranked by score, every item owner-level and actionable:
// owner, short title, plain action, evidence, effect in its own unit, a score,
// a confidence and a verifying measure; nothing under the materiality floor.
func TestActionsAreOwnerLevelRankedAndCheckable(t *testing.T) {
	report, _ := weekReport(t, time.Time{})
	if len(report.Actions) == 0 || report.Below == 0 {
		t.Fatalf("actions = %d below = %d, want both listed and folded actions", len(report.Actions), report.Below)
	}
	for i, a := range report.Actions {
		if a.Rank != i+1 || (i > 0 && a.Score > report.Actions[i-1].Score && a.OwnerKind != OwnerSession) {
			t.Errorf("rank %d out of order: %+v", i+1, a)
		}
		if a.Owner == "" || a.OwnerKind == "" || a.Title == "" || len([]rune(a.Title)) > titleRunes || a.Text == "" || a.Verify == "" || a.Confidence == "" || a.Effect.Unit == "" || a.Effect.Basis == "" {
			t.Errorf("incomplete action %+v", a)
		}
		if len(a.Evidence)+len(a.Drilldown) == 0 {
			t.Errorf("%s has no evidence", a.Title)
		}
		if a.Source == SourceLever && a.Score < toolfriction.MaterialScore && a.Effect.Basis != BasisRange || a.ScoreBasis == "" {
			t.Errorf("%s listed under the materiality floor: %+v", a.Title, a.Effect)
		}
	}
	for _, kind := range []string{RuleContext, RuleModelMix} {
		if len(byKind(report, kind)) == 0 {
			t.Errorf("%s did not fire on the fixture week", kind)
		}
	}
	if len(report.Rules) != len(Rules) {
		t.Fatalf("rule statuses = %d, want one per family", len(report.Rules))
	}
}

// P4-F5/P4-A6: R-context is an aggregate per runner with sessions as
// drill-down. A finished session is never "projected": it says what it would
// have saved. A session that compacts gets no compact or restart advice of its
// own and its saving is priced on its regrowth after each compaction. A
// still-running session that never compacted is a projected part of the
// aggregate, never a second action for the same quota.
func TestContextLeverCountsCompactionsAndNeverProjectsFinishedSessions(t *testing.T) {
	var turns []Turn
	turns = append(turns, growingTurns("done", 0, 300, 1_200, 0)...)                  // finished hours before now
	turns = append(turns, growingTurns("compactor", 10*time.Hour, 300, 1_200, 60)...) // compacts every 60 requests
	turns = append(turns, growingTurns("grower", 10*time.Hour, 300, 1_200, 0)...)     // still running, never compacts
	result := handResult("anthropic", "claude-code", "claude-opus-4-7", 30, turns, 0.02)
	runs := map[string]Run{}
	for _, id := range []string{"done", "compactor", "grower"} {
		runs[id] = Run{ID: id, Runner: "claude-code", Session: id, SessionLabel: id + " session"}
	}
	now := fixtureStart.Add(10*time.Hour + 300*time.Minute)
	report := Recommend(Inputs{Results: []Result{result}, Runs: runs, Prices: fixturePrices, Now: now})

	var aggregate *Action
	sessions := map[string]Action{}
	for _, a := range byKind(report, RuleContext) {
		switch a.OwnerKind {
		case OwnerHarness:
			aggregate = &a
		case OwnerSession:
			sessions[a.Owner] = a
		}
	}
	if aggregate == nil || aggregate.Owner != "claude-code session context" || aggregate.Effect.Basis != BasisProjected {
		t.Fatalf("aggregate = %+v, want one projected claude-code harness action", aggregate)
	}
	if _, ok := sessions["compactor"]; ok {
		t.Error("a session that compacts was told to compact or restart")
	}
	if _, ok := sessions["done"]; ok {
		t.Error("a finished session got a session action")
	}
	drill := map[string]Drill{}
	for _, d := range aggregate.Drilldown {
		drill[d.ID] = d
	}
	// The still-running grower is priced inside the aggregate: a second
	// restart-now action for it would count the same quota twice.
	if _, ok := sessions["grower"]; ok {
		t.Errorf("grower is in the aggregate's drill-down and also has its own action: %+v", sessions["grower"])
	}
	if d := drill["grower"]; !d.StillRunning || !strings.Contains(d.Detail, "projected to save") {
		t.Fatalf("grower drill = %+v, want a projected, still-running part of the aggregate", d)
	}
	if d := drill["done"]; d.StillRunning || !strings.Contains(d.Detail, "would have saved") || strings.Contains(d.Detail, "projected") {
		t.Errorf("finished session drill = %+v, want 'would have saved'", d)
	}
	if d := drill["compactor"]; !strings.Contains(d.Detail, "compacted 4×") || d.Value >= drill["grower"].Value {
		t.Errorf("compactor drill = %+v (grower %.2f), want its compactions counted and a smaller regrowth saving", d, drill["grower"].Value)
	}
	for _, a := range report.Actions {
		if strings.Contains(a.Text+a.Title, "compactor") {
			t.Errorf("an action names the compacting session: %+v", a)
		}
	}

	finished := Recommend(Inputs{Results: []Result{result}, Runs: runs, Prices: fixturePrices, Now: now.Add(24 * time.Hour)})
	for _, a := range finished.Actions {
		if a.Effect.Basis == BasisProjected || a.OwnerKind == OwnerSession || a.Recoverable != 0 {
			t.Errorf("a day after every session stopped, %q is %s with %.2f recoverable", a.Title, a.Effect.Basis, a.Recoverable)
		}
	}
	if len(byKind(finished, RuleContext)) == 0 || byKind(finished, RuleContext)[0].Effect.Basis != BasisWouldHaveSaved {
		t.Errorf("finished R-context = %+v, want a 'would have saved' aggregate", byKind(finished, RuleContext))
	}
}

// The growth model: a segment growing g a request for n requests, capped at
// k, saves g·n(n−k)/2 less (n/k − 1) rebuilds; a compacting run regrows from
// each compaction, so the same growth split in two segments saves less.
func TestGrowthModelSavesGrowthBeyondTheCapNetOfRebuilds(t *testing.T) {
	k, saved := capSaving([]segment{{requests: 100, first: 0, last: 100_000}}, 0)
	if k != minCap || saved != 1000*100*80/2 {
		t.Fatalf("cap %d saves %.0f, want cap %d saving %d", k, saved, minCap, 1000*100*80/2)
	}
	_, withRebuild := capSaving([]segment{{requests: 100, first: 0, last: 100_000}}, 500_000)
	if withRebuild >= saved || withRebuild <= 0 {
		t.Errorf("with rebuilds %.0f, want below %.0f and positive", withRebuild, saved)
	}
	if _, split := capSaving([]segment{{50, 0, 50_000}, {50, 0, 50_000}}, 0); split >= saved {
		t.Errorf("a compacting run saves %.0f, want less than the uncompacted %.0f", split, saved)
	}
	if _, flat := capSaving([]segment{{requests: 100, first: 90_000, last: 90_000}}, 1); flat != 0 {
		t.Errorf("a flat run saves %.0f, want 0", flat)
	}
}

// R-lifetime prices sub-agents by their own growth: growing workers fire,
// netting the rebuild; workers that start high and stay flat do not.
func TestLifetimeFiresOnlyForGrowingSubAgents(t *testing.T) {
	for _, flat := range []bool{false, true} {
		var turns []Turn
		runs := map[string]Run{}
		for i := 0; i < 4; i++ {
			id := fmt.Sprintf("w%d", i)
			runs[id] = fixtureRun(id, "s", true)
			step := 2_000
			if flat {
				step = 0
			}
			turns = append(turns, growingTurns(id, time.Duration(i)*time.Hour, 200, step, 0)...)
		}
		result := handResult("openai", codex, modelTerra, 40, turns, 0.05)
		primary, sessions := prepare(Inputs{Results: []Result{result}, Runs: runs, Now: fixtureStart.Add(48 * time.Hour)})
		rows, status := ruleGrowth(Inputs{}, primary, sessions, true)
		if flat != (len(rows) == 0) {
			t.Fatalf("flat=%v: rows %+v, want lifetime only for growing workers", flat, rows)
		}
		if flat && status.Reason == "" {
			t.Error("a silent rule must say why")
		}
		if !flat && (rows[0].Effect.Value >= 4*200*0.05 || rows[0].Owner != codex+" sub-agent context" || rows[0].Effect.Basis != BasisWouldHaveSaved) {
			t.Errorf("lifetime = %+v, want a finished harness lever saving less than the workers used", rows[0])
		}
	}
}

// P3-F6: active is judged against now. A session idle for 3 hours is
// finished: it projects to its points with nothing recoverable, and the active
// sessions' projected future fits in the window's headroom.
func TestProjectionJudgesActivityAgainstNowAndCapsAtHeadroom(t *testing.T) {
	idle := steadyTurns("idle", 0, 5*time.Hour, 10*time.Minute, 50_000)
	live := steadyTurns("live", 0, 20*time.Hour, 10*time.Minute, 50_000)
	result := handResult("openai", codex, modelTerra, 96, append(idle, live...), 0.1)
	runs := map[string]Run{"idle": fixtureRun("idle", "idle", false), "live": fixtureRun("live", "live", false)}
	stats := sessionStatsFor(&result, runs)
	result.proj = newProjCtx(&result, fixtureStart.Add(20*time.Hour+5*time.Minute), stats)
	var future float64
	for _, s := range stats {
		p := s.project(&result)
		switch s.key {
		case "idle":
			if p.active || p.weekly != p.points || p.recoverable(3) != 0 {
				t.Errorf("idle session: %+v, want finished, weekly == points, nothing recoverable", p)
			}
		case "live":
			if !p.active || p.weekly <= p.points {
				t.Errorf("live session: %+v, want a decaying future", p)
			}
		}
		future += p.future()
	}
	if headroom := 100 - result.Window.UsedPercent; future > headroom+1e-9 {
		t.Errorf("projected future %.2f exceeds the window's headroom %.2f", future, headroom)
	}
}

// P3-F7: Opus work outside a workload is an unproven range (0 to the price
// gap) labelled a prior; it scores 0 so it never outranks a point estimate.
func TestModelMixIsAnUnprovenRangeAggregatedPerRole(t *testing.T) {
	var turns []Turn
	runs := map[string]Run{}
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("chat-%d", i)
		runs[id] = Run{ID: id, Runner: "claude-code", Session: id}
		turns = append(turns, steadyTurns(id, 0, 10*time.Hour, 10*time.Minute, 80_000)...)
	}
	result := handResult("anthropic", "claude-code", "claude-opus-4-7", 30, turns, 0.15)
	report := Recommend(Inputs{Results: []Result{result}, Runs: runs, Prices: fixturePrices, Now: fixtureStart.Add(30 * time.Hour)})
	rows := byKind(report, RuleModelMix)
	if len(rows) != 1 {
		t.Fatalf("model-mix rows = %d, want one aggregate for claude-code sessions", len(rows))
	}
	a := rows[0]
	if a.Confidence != "unproven" || a.Effect.Basis != BasisRange || a.Effect.Value != 0 || a.Effect.Upper <= 0 || a.Score != 0 || !strings.Contains(a.Rationale, "prior") || len(a.Drilldown) != 3 {
		t.Fatalf("model-mix = %+v, want an unproven range over three sessions", a)
	}
}

// R-model compares a workload's finished runs; a ratio from tier priors keeps
// its estimate but is low confidence and says so.
func TestModelRuleMovesAWorkloadAndLabelsPriors(t *testing.T) {
	f := newWeekFixture()
	find := func(prices Prices) Action {
		result := Attribute(f.window, f.window.Start, f.turns, prices)
		primary, sessions := prepare(Inputs{Results: []Result{result}, Runs: f.runs})
		rows, _ := ruleModel(Inputs{Outcomes: f.outcomes, Prices: prices}, primary, sessions)
		if len(rows) == 0 {
			t.Fatal("R-model did not fire")
		}
		return rows[0]
	}
	measured := find(fixturePrices)
	if measured.OwnerKind != OwnerWorkload || measured.Owner != "workflow_node:draft" || !strings.Contains(measured.Text, modelTerra+" to "+modelLuna) || strings.Contains(measured.Rationale, "prior") {
		t.Errorf("measured move = %+v", measured)
	}
	guessed := find(func(runner, model string) Price { p := fixturePrices(runner, model); p.Prior = true; return p })
	if guessed.Confidence != "low" || !strings.Contains(guessed.Rationale, "price ratio is a prior") {
		t.Errorf("prior-priced move = %+v", guessed)
	}
}

// A lever family that cannot fire says why from the data.
func TestRulesExplainWhyTheyDidNotFire(t *testing.T) {
	f := newWeekFixture()
	var quiet []Turn
	for _, turn := range f.turns {
		if turn.Model == modelLuna {
			quiet = append(quiet, turn)
			break
		}
	}
	result := Attribute(f.window, f.window.Start, quiet, fixturePrices)
	report := Recommend(Inputs{Results: []Result{result}, Runs: f.runs, Prices: fixturePrices})
	for _, status := range report.Rules {
		if status.Fired || status.Reason == "" {
			t.Errorf("status %+v: want not fired with a data reason", status)
		}
	}
}

// P4-F5: labels are short titles, never a raw first prompt.
func TestSessionLabelsAreShortTitles(t *testing.T) {
	prompt := "Work until this verified end state is true: `agent-manager usage recommendations` is one owner-level list\nRead first: WORKER.md"
	if got := shortLabel(Run{SessionLabel: "1e5904ed " + prompt, Runner: "codex"}, "web-console:1e5904ed-aaaa"); got != "codex session 1e5904ed" {
		t.Errorf("label = %q, want the runner and short id instead of a raw prompt", got)
	}
	if got := shortLabel(Run{SessionLabel: "d7be7470 Agent-manager usage intelligence"}, "s"); got != "d7be7470 Agent-manager usage intelligence" {
		t.Errorf("title = %q, want it kept", got)
	}
	if got := shortLabel(Run{Workload: "workflow_node:draft", SessionLabel: prompt}, "s"); got != "Workflow draft" {
		t.Errorf("workload label = %q", got)
	}
	if got := shortLabel(Run{}, "web-console:0d5f3c2e-1111"); got != "session 0d5f3c2e" {
		t.Errorf("unlabelled = %q", got)
	}
}

// mergeFixture is one week with a quota window and every non-quota source:
// a failing CLI, a dismissed finding, a bypass, a skill loaded in three
// sessions, a repeated sequence a program covers, and a gated intervention.
func mergeFixture() Inputs {
	turns := steadyTurns("main", 0, 20*time.Hour, 5*time.Minute, 80_000)
	result := handResult("anthropic", "claude-code", "claude-sonnet-4-6", 40, turns, 0.1) // 0.1 pt a request
	in := Inputs{Results: []Result{result}, Runs: map[string]Run{"main": {ID: "main", Runner: "claude-code", Session: "main"}}, Prices: fixturePrices, Now: fixtureStart.Add(40 * time.Hour)}
	at := fixtureStart
	call := func(run, owner, path string, f toolfriction.Features) toolfriction.Call {
		at = at.Add(time.Minute)
		return toolfriction.Call{RunID: run, At: at, Owner: owner, CommandPath: path, Outcome: runsignal.OutcomeSuccess, ResultTokens: 1_000, Residency: 50, Features: f}
	}
	for i := 0; i < 40; i++ {
		c := call(fmt.Sprintf("r%d", i%5), "test-genie", "test-genie runs wait", toolfriction.Features{Signature: "park unavailable", ErrorLine: "park unavailable", Command: "test-genie runs wait x"})
		c.Outcome = runsignal.OutcomeFailure
		in.Calls = append(in.Calls, c)
	}
	for i := 0; i < 6; i++ {
		c := call(fmt.Sprintf("r%d", i%3), "agent-manager", "agent-manager task", toolfriction.Features{Signature: "dismissed", ErrorLine: "dismissed", Command: "agent-manager task"})
		c.Outcome = runsignal.OutcomeFailure
		in.Calls = append(in.Calls, c)
	}
	for i := 0; i < 3; i++ {
		run := fmt.Sprintf("s%d", i)
		for j := 0; j < 3; j++ {
			in.Calls = append(in.Calls, call(run, "", "sqlite3", toolfriction.Features{Command: "sqlite3 am.db 'select * from runs'", BypassOwner: "agent-manager", BypassMethod: toolfriction.BypassDatabase}))
		}
		for j := 0; j < 3; j++ {
			for _, path := range []string{"vrooli-memory journal note", "search-hub query", "vrooli scenario logs"} {
				in.Calls = append(in.Calls, call(run, strings.Fields(path)[0], path, toolfriction.Features{Command: path}))
			}
			in.Calls = append(in.Calls, call(run, "", "rg", toolfriction.Features{Command: "rg x"}))
		}
		skill := call(fmt.Sprintf("k%d", i), "prompt-manager", "prompt-manager skill read", toolfriction.Features{Command: "prompt-manager skill read big"})
		skill.SkillID, skill.ResultTokens, skill.Residency = "big", 20_000, 200
		in.Calls = append(in.Calls, skill)
	}
	dismissed := toolfriction.Analyze(toolfriction.Window{Calls: in.Calls})
	in.Statuses = map[string]ProblemStatus{}
	for _, f := range dismissed.Findings {
		if f.Owner == "agent-manager" && f.Kind == toolfriction.KindFailure {
			in.Statuses[f.Fingerprint] = ProblemStatus{Fingerprint: f.Fingerprint, Status: StatusDismissed}
		}
	}
	in.Library = &opportunities.Library{Programs: []opportunities.Program{{Name: "search-hub.orient", Scenario: "search-hub", Bindings: []string{"vrooli-memory/journal/note", "search-hub/query"}}}}
	in.Interventions = &interventions.Summary{Classified: 10, Categories: []interventions.CategorySummary{{Category: interventions.QualityMiss, Shown: 4, Work: interventions.Work{Tokens: 4_000_000}}}}
	return in
}

// The merged list: tool friction, skill weight, opportunities and gated
// interventions rank beside the quota levers on one score, the share of their
// own kind of harm, stated with its total; a dismissed finding is gone and a
// bypass is listed once (by the opportunity that names what agents reached).
func TestEverySourceMergesIntoOneRankedList(t *testing.T) {
	in := mergeFixture()
	report := Recommend(in)
	totals := toolfriction.Analyze(toolfriction.Window{Calls: in.Calls}).Totals
	if !strings.Contains(strings.Join(Scoring, " "), "assumption: kinds of harm weigh equally") {
		t.Fatalf("scoring = %q", Scoring)
	}
	seen := map[string]Action{}
	bypasses := 0
	for _, a := range report.Actions {
		seen[a.Source+"/"+a.Kind+"/"+a.Owner] = a
		if a.Kind == toolfriction.KindBypass {
			bypasses++
		}
		if a.Kind == toolfriction.KindOutputWeight && a.Owner == "prompt-manager skill read" {
			t.Errorf("skill loads counted twice, as output weight and R-skill: %+v", a)
		}
		if a.Owner == "agent-manager task" {
			t.Errorf("dismissed finding listed: %+v", a)
		}
	}
	failure, ok := seen["tool/failure/test-genie runs wait"]
	runsInWindow := float64(totals.Runs)
	wantFailure := math.Sqrt(failure.Effect.Value / float64(totals.VrooliFailed) * float64(failure.runs) / runsInWindow)
	if !ok || failure.OwnerKind != OwnerCommand || failure.Effect.Unit != UnitCalls || math.Abs(failure.Score-wantFailure) > 1e-9 || !strings.Contains(failure.ScoreBasis, fmt.Sprintf("%d Vrooli CLI failed calls", totals.VrooliFailed)) || failure.runs != 5 || len(failure.Fingerprints) != 1 {
		t.Errorf("failure = %+v, want its share of the window's Vrooli-owned failed calls over 5 runs: score √(share × breadth) = %.4f", failure, wantFailure)
	}
	skill, ok := seen["tool/"+RuleSkill+"/big"]
	if !ok || skill.OwnerKind != OwnerSkill || skill.Effect.Unit != UnitTokens || math.Abs(skill.Score-math.Sqrt(float64(3*20_000*200)/float64(totals.ContextTax)*3/runsInWindow)) > 1e-9 {
		t.Errorf("skill = %+v, want its share of the window's context tax", skill)
	}
	if bypasses != 1 || seen["opportunity/bypass/agent-manager"].Source == "" {
		t.Errorf("bypasses listed %d times, want once from opportunities: %v", bypasses, seen)
	}
	sequence, ok := seen["opportunity/sequence/search-hub.orient"]
	if !ok || sequence.OwnerKind != OwnerProgram || !strings.HasPrefix(sequence.Title, "Adopt program") {
		t.Errorf("sequence = %+v, want an adopt action owned by the program", sequence)
	}
	if !strings.Contains(sequence.Rationale, "program-library verdict: adopt `search-hub.orient`") {
		t.Errorf("sequence rationale %q lacks the program-library verdict", sequence.Rationale)
	}
	quality, ok := seen["intervention/quality_miss/"+interventions.QualityMiss.Owner()]
	if !ok || quality.Effect.Unit != UnitInterventions || quality.Effect.Value != 4 || math.Abs(quality.Score-0.4) > 1e-9 {
		t.Errorf("intervention = %+v, want 4 of 10 operator messages", quality)
	}
	for i := 1; i < len(report.Actions); i++ {
		if report.Actions[i].Score > report.Actions[i-1].Score && report.Actions[i].OwnerKind != OwnerSession {
			t.Fatalf("not ranked by score at %d", i)
		}
	}
}

// Silence needs proof: an ungated intervention source and an empty tool source
// say what they could not see.
func TestEmptySourcesStateCoverage(t *testing.T) {
	in := mergeFixture()
	in.Calls = []toolfriction.Call{}
	in.Interventions = &interventions.Summary{Classified: 120, Unclassified: 30}
	report := Recommend(in)
	text := strings.Join(report.Coverage, "\n")
	for _, want := range []string{"cannot tell whether tools caused friction", "no category has passed the accuracy gate", "cannot tell, not none", "anthropic 10080m window"} {
		if !strings.Contains(text, want) {
			t.Errorf("coverage %q lacks %q", text, want)
		}
	}
	for _, a := range report.Actions {
		if a.Source == SourceIntervention {
			t.Errorf("ungated intervention listed: %+v", a)
		}
	}
	none := Recommend(Inputs{Sources: mergeFixture().Sources})
	if len(none.Actions) == 0 || none.Actions[0].Score <= 0 || !strings.Contains(strings.Join(none.Coverage, ""), "quota levers cannot be evaluated") {
		t.Errorf("without windows the tool actions are still scored on their own harm: %+v %q", none.Actions, none.Coverage)
	}
}

// The cap the operator is told is one number, so the saving reported is what
// that cap saves: every member re-priced at it (a member that nets nothing at
// it drops out), never the sum of each member's own best cap.
func TestLeverSavingIsPricedAtTheStatedCapNotEachRunsBest(t *testing.T) {
	var turns []Turn
	runs := map[string]Run{}
	for i, spec := range []struct{ n, step int }{{400, 900}, {400, 1_100}, {150, 2_500}, {90, 3_000}, {60, 40}} {
		id := fmt.Sprintf("w%d", i)
		runs[id] = fixtureRun(id, "s"+id, true)
		turns = append(turns, growingTurns(id, time.Duration(i)*10*time.Hour, spec.n, spec.step, 0)...)
	}
	result := handResult("openai", codex, modelTerra, 40, turns, 0.05)
	in := Inputs{Results: []Result{result}, Runs: runs, Now: fixtureStart.Add(80 * time.Hour)}
	primary, sessions := prepare(in)
	rows, _ := ruleGrowth(in, primary, sessions, true)
	if len(rows) != 1 {
		t.Fatalf("rows = %+v, want one sub-agent aggregate", rows)
	}
	var k int
	if _, err := fmt.Sscanf(rows[0].Title, "Restart codex sub-agents after ~%d requests", &k); err != nil {
		t.Fatal(err)
	}
	floors := measuredFloors(sessions[primary[0]])
	var atStated, atOwnBest float64
	caps := map[int]bool{}
	for _, s := range sessions[primary[0]] {
		use := s.project(primary[0])
		for _, rs := range s.sortedRuns() {
			segs := rs.segments()
			rebuild := rebuildCost(floors[floorKey{rs.runner, rs.model, rs.child}], creationWeight(primary[0], rs.runner, rs.model), float64(rs.prompt)/float64(rs.reports))
			best, tokens := capSaving(segs, rebuild)
			if tokens <= 0 {
				continue
			}
			caps[best] = true
			atOwnBest += use.saving(min(tokens*rs.pointsPerToken(), rs.points))
			if net := netAtCap(segs, rebuild, k); net > 0 {
				atStated += use.saving(min(net*rs.pointsPerToken(), rs.points))
			}
		}
	}
	if len(caps) < 2 {
		t.Fatalf("fixture has one best cap %v: it cannot tell a median cap from per-run caps", caps)
	}
	if math.Abs(rows[0].Effect.Value-atStated) > 1e-9 || rows[0].Effect.Value >= atOwnBest {
		t.Errorf("saving %.4f, want the %.4f every member nets at the stated cap %d, below the %.4f of summing each run's own best", rows[0].Effect.Value, atStated, k, atOwnBest)
	}
}

func TestEqualLengthWindowsOfTwoPoolsBothSurvive(t *testing.T) {
	mk := func(pool string) Result {
		r := handResult("openai", codex, modelTerra, 10, nil, 0)
		r.Window.Pool = pool
		return r
	}
	got := selectResults([]Result{mk("primary"), mk("secondary")}, WindowKey{})
	if len(got) != 2 || got[0].Window.Pool != "primary" || got[1].Window.Pool != "secondary" {
		t.Fatalf("selected %d windows, want both pools' equal-length windows", len(got))
	}
}

// Owner-impact ranking: a defect that touches many runs outranks a bigger
// share of harm made by one run's loop; quota levers (a share of a window's
// points) and friction actions (√(share × breadth)) are ranked in their own
// groups, never interleaved, each score naming its scale and absolute cost;
// a live session's own advice ranks after every aggregate.
func TestRankingWeighsBreadthAndKeepsLiveSessionsBelowAggregates(t *testing.T) {
	const windowRuns = 200
	loop := Action{Kind: "polling", Source: SourceTool, Title: "one run's jq loop", Effect: Effect{Value: 300, Unit: UnitCalls, Basis: BasisObserved}}.
		against(1000, "polled calls").affecting(1, windowRuns)
	blocker := Action{Kind: "failure", Source: SourceTool, Title: "control plane blocks 16 runs", Effect: Effect{Value: 50, Unit: UnitCalls, Basis: BasisObserved}}.
		against(1000, "failed calls").affecting(16, windowRuns)
	session := Action{Kind: RuleContext, Source: SourceLever, OwnerKind: OwnerSession, Title: "restart this session", Effect: Effect{Value: 20, Unit: UnitPoints, Basis: BasisProjected}}.
		against(40, "points")
	lever := Action{Kind: RuleContext, Source: SourceLever, OwnerKind: OwnerHarness, Title: "compact", Effect: Effect{Value: 1, Unit: UnitPoints, Basis: BasisProjected}}.
		against(40, "points")
	ranked, below := rank([]Action{loop, session, lever, blocker}, nil)
	var order []string
	for _, a := range ranked {
		order = append(order, a.Title)
	}
	want := []string{"compact", "control plane blocks 16 runs", "one run's jq loop", "restart this session"}
	if below != 0 || strings.Join(order, "|") != strings.Join(want, "|") {
		t.Fatalf("order %v (below %d), want %v", order, below, want)
	}
	if got := ranked[1].Score; math.Abs(got-math.Sqrt(0.05*16/windowRuns)) > 1e-9 || !strings.HasPrefix(ranked[1].ScoreBasis, "friction scale: 6.3% = √(share 5.0% (50 calls) of failed calls") || !strings.Contains(ranked[1].ScoreBasis, "16 of 200 runs") {
		t.Errorf("blocker score %.4f basis %q, want √(share × breadth) with its scale, inputs and absolute cost shown", got, ranked[1].ScoreBasis)
	}
	if ranked[0].Score != 0.025 || ranked[0].ScoreBasis != "quota scale: 2.5% (1.0 pts) of points" {
		t.Errorf("a quota lever scores its quota share on its own scale, got %.3f %q", ranked[0].Score, ranked[0].ScoreBasis)
	}
}

// A bypass of one scenario's HTTP and a finding that the same scenario's
// command is guessed at, about the same target, are one action listing both.
func TestSignalsAboutOneTargetOfOneOwnerMerge(t *testing.T) {
	bypass := Action{
		Kind: "bypass", Source: SourceOpportunity, Title: "Add CLI/RPC for test-genie via HTTP", Text: "add an artifact RPC", Rationale: "204 calls", Verify: "v1",
		Effect: Effect{Value: 204, Unit: UnitCalls, Basis: BasisObserved}, Evidence: []Ref{{Kind: "run", ID: "r1"}}, scenario: "test-genie", words: targetWords("/runs/{id}/artifacts", "test-genie"),
	}.
		against(2741, "bypass calls").affecting(28, 281)
	guess := Action{
		Kind: "shape_guess", Source: SourceTool, Title: "Make `test-genie runs artifacts` easier to call", Text: "name the flag", Rationale: "28 guesses", Verify: "v2",
		Effect: Effect{Value: 28, Unit: UnitCalls, Basis: BasisObserved}, Evidence: []Ref{{Kind: "run", ID: "r2"}}, scenario: "test-genie", words: targetWords("test-genie runs artifacts", "test-genie"),
	}.
		against(374, "shape guesses").affecting(5, 281)
	other := Action{Kind: "failure", Source: SourceTool, Title: "unrelated", Effect: Effect{Value: 40, Unit: UnitCalls, Basis: BasisObserved}, scenario: "test-genie", words: targetWords("test-genie validation", "test-genie")}.
		against(2000, "failed calls").affecting(10, 281)
	guess.Fingerprints = []string{"guess-fp"}
	other.Fingerprints = []string{"other-fp"}
	rules := map[string]*RuleStatus{"bypass": {Rule: "bypass"}, "shape_guess": {Rule: "shape_guess"}, "failure": {Rule: "failure"}}
	ranked, _ := rank([]Action{bypass, guess, other}, rules)
	if len(ranked) != 2 {
		t.Fatalf("actions %d, want the two artifact signals merged and the unrelated one kept", len(ranked))
	}
	// Rows count listed actions: the folded-in guess is not a row of its own.
	if rules["bypass"].Rows != 1 || rules["shape_guess"].Rows != 0 || rules["failure"].Rows != 1 {
		t.Errorf("rows bypass %d, shape_guess %d, failure %d; want 1, 0, 1 after the merge", rules["bypass"].Rows, rules["shape_guess"].Rows, rules["failure"].Rows)
	}
	var merged Action
	for _, a := range ranked {
		if strings.Contains(a.Title, "artifact") {
			merged = a
		}
	}
	if !strings.Contains(merged.Title, "test-genie") || !strings.Contains(merged.Rationale, "28 guesses") || !strings.Contains(merged.Rationale, "204 calls") || len(merged.Evidence) != 2 || !strings.Contains(merged.Verify, "v2") {
		t.Errorf("merged = %+v, want both signals, evidence and checks", merged)
	}
	if !slices.Equal(merged.Fingerprints, []string{"guess-fp"}) {
		t.Errorf("merged fingerprints = %v, want the folded tool finding's, so it can still be reported or dismissed", merged.Fingerprints)
	}
}

// A provider's windows see the same sessions: the growth levers state one cap
// per runner, fitted in the provider's longest window, never a second,
// contradictory cap from a shorter window.
func TestGrowthLeversStateOneCapPerRunner(t *testing.T) {
	var turns []Turn
	runs := map[string]Run{}
	for i := 0; i < 4; i++ {
		id := fmt.Sprintf("w%d", i)
		runs[id] = fixtureRun(id, "s", true)
		turns = append(turns, growingTurns(id, time.Duration(i)*time.Hour, 200, 2_000, 0)...)
	}
	weekly := handResult("anthropic", "claude-code", "claude-opus-4-7", 40, turns, 0.05)
	short := handResult("anthropic", "claude-code", "claude-opus-4-7", 60, turns, 0.05)
	short.Window.Pool, short.Window.Name, short.Window.Minutes = "session", "300m", 300
	report := Recommend(Inputs{Results: []Result{short, weekly}, Runs: runs, Prices: fixturePrices, Now: fixtureStart.Add(48 * time.Hour)})
	lifetime := byKind(report, RuleLifetime)
	if len(lifetime) != 1 || lifetime[0].Window != "10080m" {
		t.Fatalf("lifetime levers = %+v, want one, in the weekly window", lifetime)
	}
}

// R-skill states what it could not see: loads whose result tokens are unknown
// count no context tax, so a silent or small R-skill says so instead of
// reading as "under the materiality floor".
func TestSkillWeightStatesUnknownTokenLoads(t *testing.T) {
	var calls []toolfriction.Call
	for i := 0; i < 4; i++ {
		load := toolfriction.Call{RunID: fmt.Sprintf("r%d", i), CommandPath: "prompt-manager skill read", Owner: "prompt-manager", SkillID: "big", Residency: 100, TokenBasis: tokenaccounting.BasisUnknown}
		calls = append(calls, load)
	}
	known := toolfriction.Call{RunID: "r0", CommandPath: "prompt-manager skill read", Owner: "prompt-manager", SkillID: "big", Residency: 100, ResultTokens: 4_000, TokenBasis: tokenaccounting.BasisEstimated}
	calls = append(calls, known)
	report := Recommend(Inputs{Sources: Sources{Calls: calls}})
	var status RuleStatus
	for _, rule := range report.Rules {
		if rule.Rule == RuleSkill {
			status = rule
		}
	}
	if !strings.Contains(status.Reason, "token counts unknown for 4 loads") {
		t.Errorf("R-skill reason = %q, want the unknown loads named", status.Reason)
	}
	for _, a := range byKind(report, RuleSkill) {
		if !strings.Contains(a.Rationale, "≈4000 tokens a load") || !strings.Contains(a.Rationale, "Token counts unknown for 4 loads") {
			t.Errorf("R-skill rationale = %q, want the known loads' size and the unknown count, never 0 tokens", a.Rationale)
		}
	}
}

// Tool-friction semantics reach the action list: each action is its finding's
// largest share over that kind's base (toolfriction.Totals.Base: failures over
// Vrooli-owned failures, degraded notices and slow time over Vrooli CLI calls
// and time), a root-cause roll-up is one action that states each kind's share
// over its own base, a slow command is stated in excess wall-clock, guesses
// name the alias to add, help lookups alone say the usage is not memorable,
// polling of a command whose blocking sibling exists steers agents to it, and
// a concentrated finding says so beside its breadth.
func TestToolActionsCarryFrictionSemantics(t *testing.T) {
	totals := toolfriction.Totals{Calls: 184_000, FailedCalls: 4000, VrooliCalls: 800, VrooliFailed: 50, DurationMS: 6_000_000, VrooliDurationMS: 500_000, ShapeGuesses: 400, VrooliShapeGuesses: 40, Polled: 3000, VrooliPolled: 300, ContextTax: 1_000_000}
	shares := func(costs ...any) []toolfriction.Share {
		var out []toolfriction.Share
		for i := 0; i < len(costs); i += 2 {
			kind, cost := costs[i].(string), int64(costs[i+1].(int))
			total, base := totals.Base(kind)
			out = append(out, toolfriction.Share{Kind: kind, Unit: toolfriction.UnitOf(kind), Cost: cost, Share: float64(cost) / float64(total), Base: base})
		}
		return out
	}
	subs := []toolfriction.SubSignature{{Kind: "failure", Signature: "identity broker did not answer", Calls: 30}, {Kind: "degraded", Signature: "identity_refresh_unavailable", Calls: 10}}
	report := toolfriction.Report{Totals: totals, Findings: []toolfriction.Finding{
		{Fingerprint: "f1", Kind: toolfriction.KindFailure, Owner: "vrooli", Calls: 40, Runs: 20, HarmRuns: 20, Unit: toolfriction.UnitCalls, Cost: 40, ErrorLine: "identity broker did not answer", RootCause: "identity", Subs: subs, Shares: shares(toolfriction.KindFailure, 30, toolfriction.KindDegraded, 10)},
		{Fingerprint: "f2", Kind: toolfriction.KindDegraded, Owner: "test-genie", CommandPath: "test-genie runs", Calls: 30, Runs: 10, HarmRuns: 10, Unit: toolfriction.UnitCalls, Cost: 30, ErrorLine: "park unavailable", Shares: shares(toolfriction.KindDegraded, 30)},
		{Fingerprint: "f3", Kind: toolfriction.KindSlow, Owner: "agent-manager", CommandPath: "agent-manager run status", Calls: 20, Runs: 8, HarmRuns: 8, Unit: toolfriction.UnitMS, Cost: 60_000, Shares: shares(toolfriction.KindSlow, 60_000)},
		{
			Fingerprint: "f4", Kind: toolfriction.KindShapeGuess, Owner: "agent-manager", CommandPath: "agent-manager run", Calls: 12, Runs: 6, HarmRuns: 6, Unit: toolfriction.UnitCalls, Cost: 12, WrongShape: 12, Shares: shares(toolfriction.KindShapeGuess, 12),
			Guesses: []toolfriction.Guess{{Token: "runs", Suggestion: "run", Calls: 8}, {Token: "--tail", Calls: 4}},
		},
		{Fingerprint: "f5", Kind: toolfriction.KindOutputWeight, Owner: "vrooli", CommandPath: "vrooli codex", Calls: 5, Runs: 3, HarmRuns: 3, Unit: toolfriction.UnitTokens, Cost: 100_000, PrintedTokens: 262_144, ReadTokens: 10_000, Shares: shares(toolfriction.KindOutputWeight, 100_000)},
		{Fingerprint: "f6", Kind: toolfriction.KindPolling, Owner: "vrooli", CommandPath: "vrooli scenario status", Calls: 90, Runs: 30, HarmRuns: 20, Unit: toolfriction.UnitCalls, Cost: 90, TopRunShare: 0.62, Blocking: []string{"vrooli scenario wait"}, Steered: 30, SteeredRuns: 12, Shares: shares(toolfriction.KindPolling, 90)},
		{
			Fingerprint: "f7", Kind: toolfriction.KindShapeGuess, Owner: "vrooli-memory", CommandPath: "vrooli-memory journal note", Calls: 40, Runs: 9, HarmRuns: 9, Unit: toolfriction.UnitCalls, Cost: 12, WrongShape: 1, HelpLookups: 11, Shares: shares(toolfriction.KindShapeGuess, 12),
			Guesses: []toolfriction.Guess{{Token: "--title", Calls: 1}},
		},
		{Fingerprint: "f8", Kind: toolfriction.KindPolling, Owner: "device-control", CommandPath: "device-control session list", Calls: 30, Runs: 9, HarmRuns: 4, Unit: toolfriction.UnitCalls, Cost: 30, Shares: shares(toolfriction.KindPolling, 30)},
	}}
	got := map[string]Action{}
	for _, a := range toolActions(report, nil, nil, 100) {
		got[a.Fingerprints[0]] = a
	}
	rollUp, _ := rank([]Action{got["f1"]}, nil)
	if a := got["f1"]; a.harm != 50 || a.Effect.Value != 30 || !strings.Contains(a.harmName, "50 Vrooli CLI failed calls") || a.RootCause != "identity" || len(a.Subs) != 2 || !strings.Contains(a.Title, "identity root cause") ||
		!strings.Contains(a.Text, "identity broker did not answer") || !strings.Contains(a.Text, "identity_refresh_unavailable") || len(rollUp) != 1 ||
		!strings.Contains(rollUp[0].ScoreBasis, "share 60.0% (30 calls) of the window's 50 Vrooli CLI failed calls") || !strings.Contains(rollUp[0].ScoreBasis, "also degraded 10 calls = 1.2% of the window's 800 Vrooli CLI calls") {
		t.Errorf("roll-up = %+v (%q), want the 30 failures over the 50 Vrooli failures and the 10 notices over the 800 Vrooli CLI calls, each base named", a, rollUp[0].ScoreBasis)
	}
	if a := got["f2"]; a.harm != 800 || a.Effect.Unit != UnitCalls || !strings.Contains(a.harmName, "800 Vrooli CLI calls") {
		t.Errorf("degraded = %+v, want calls against the window's 800 Vrooli CLI calls, not every command", a)
	}
	if a := got["f3"]; a.Kind != toolfriction.KindSlow || a.Effect.Unit != UnitMS || a.harm != 500_000 || a.OwnerKind != OwnerCommand || a.Owner != "agent-manager run status" ||
		!strings.Contains(a.Text, "faster or give it a cheaper status/summary") || !strings.Contains(a.harmName, "Vrooli CLI time") {
		t.Errorf("slow = %+v, want excess wall-clock against Vrooli CLI time owned by the path", a)
	}
	if a := got["f4"]; !strings.Contains(a.Text, "add alias `runs`→`run`") || !strings.Contains(a.Text, "fix help for `--tail`") || len(a.Guesses) != 2 {
		t.Errorf("guesses = %+v, want concrete alias and help fixes", a)
	}
	if a := got["f5"]; !strings.Contains(a.Text, "printed about 262.1k") && !strings.Contains(a.Text, "printed about 262") {
		t.Errorf("output weight = %q, want printed against read tokens", a.Text)
	}
	// The wait stands in for 30 of the 90 polls (those after a start): the
	// action claims those, and says the rest have no blocking sibling.
	if a := got["f6"]; a.OwnerKind != OwnerSkill || !strings.Contains(a.Title, "to `vrooli scenario wait`") || a.Effect.Value != 30 || !strings.Contains(a.Text, "Agents poll `vrooli scenario status` where `vrooli scenario wait` blocks") || !strings.Contains(a.Text, "(30 of 90 polled calls); no blocking sibling is shown to apply to the other 60") {
		t.Errorf("polling with a blocking sibling = %+v, want the skills/docs and help told to steer agents to the wait for the polls it covers", a)
	}
	if a := got["f8"]; a.OwnerKind != OwnerCommand || !strings.Contains(a.Text, "Give `device-control session list` a blocking wait") {
		t.Errorf("polling without a sibling = %+v, want a blocking wait added", a)
	}
	if a := got["f7"]; !strings.Contains(a.Title, "usage memorable") || !strings.Contains(a.Text, "usage is not memorable at the call site (AGENTS.md/help): agents looked up its help 11 times and called it with a wrong shape 1 times") || !strings.Contains(a.Text, "also fix help for `--title` (x1)") {
		t.Errorf("mostly help lookups = %q, want the usage made memorable first, the one guess after", a.Text)
	}
	ranked, _ := rank([]Action{got["f6"]}, nil)
	if len(ranked) != 1 || ranked[0].TopRunShare != 0.62 || !strings.Contains(ranked[0].ScoreBasis, "62% of it in one run") {
		t.Errorf("concentrated = %+v, want the top-run share shown beside breadth", ranked)
	}
	// Breadth is the runs that show the harm (4), not every run that called
	// the command (9).
	if plain, _ := rank([]Action{got["f8"]}, nil); !strings.Contains(plain[0].ScoreBasis, "4 of 100 runs") {
		t.Errorf("breadth = %q, want the 4 runs that show the harm", plain[0].ScoreBasis)
	}
	// Breadth is the runs the steer covers (12), not the runs that polled (20)
	// or every run that called it (30).
	if !strings.Contains(ranked[0].ScoreBasis, "breadth 12.0%: 12 of 100 runs") {
		t.Errorf("breadth = %q, want the 12 runs the action covers", ranked[0].ScoreBasis)
	}
}

// Levers on one window's quota combine without double counting: a run two
// levers act on counts once at the larger lever, a session gives at most its
// projected future points, and another window's levers and non-lever actions
// add nothing.
func TestCombinedRecoverableCountsEachRunOnce(t *testing.T) {
	key := WindowKey{Provider: "openai", Pool: "codex", Window: "weekly"}
	lever := func(window string, recovers ...recovery) Action {
		a := Action{Source: SourceLever, Provider: key.Provider, Pool: key.Pool, Window: window, recovers: recovers}
		for _, r := range recovers {
			a.Recoverable += r.points
		}
		return a
	}
	actions := []Action{
		lever(key.Window, recovery{run: "a", session: "s1", points: 4, future: 20}, recovery{run: "b", session: "s2", points: 3, future: 4}),
		lever(key.Window, recovery{run: "a", session: "s1", points: 6, future: 20}, recovery{run: "c", session: "s2", points: 2, future: 4}),
		lever("5h", recovery{run: "d", session: "s3", points: 50, future: 50}),
		{Source: SourceTool, Recoverable: 9},
	}
	// Run a: max(4, 6) = 6. Session s2: b 3 + c 2 = 5, capped at its 4 future points.
	if points, levers := CombinedRecoverable(actions, key); points != 10 || levers != 2 {
		t.Fatalf("combined = %.1f from %d levers, want 10 from 2", points, levers)
	}
}

// Every score states its absolute cost in the unit of its harm.
func TestScoreBasisStatesTheAbsoluteCostInItsUnit(t *testing.T) {
	for unit, want := range map[string]string{UnitCalls: "321 calls", UnitTokens: "96.8M tokens", UnitMS: "8m 31s", UnitPoints: "13.0 pts", UnitInterventions: "4 interventions"} {
		value := map[string]float64{UnitCalls: 321, UnitTokens: 96_811_354, UnitMS: 511_078, UnitPoints: 13, UnitInterventions: 4}[unit]
		ranked, _ := rank([]Action{Action{Source: SourceTool, Kind: "k", Title: unit, Effect: Effect{Value: value, Unit: unit, Basis: BasisObserved}}.against(value*2, "its base")}, nil)
		if len(ranked) != 1 || !strings.Contains(ranked[0].ScoreBasis, "50.0% ("+want+") of its base") {
			t.Errorf("%s: %+v", unit, ranked)
		}
	}
}

// Quota levers of one window that price the same runs are not additive: the
// lower-ranked lever names the higher-ranked ones it shares runs with and how
// many. Each lever's verify command carries every evidence run it names (and
// how many runs it prices) against the runs of the same filter since the
// change, never a window that holds the left side's runs.
func TestOverlappingLeversSayTheyAreNotAdditiveAndVerifyAgainstLaterRuns(t *testing.T) {
	var turns []Turn
	runs := map[string]Run{}
	for i := 0; i < 12; i++ {
		id := fmt.Sprintf("w%02d", i)
		run := fixtureRun(id, "s", true)
		run.Runner = "claude-code"
		runs[id] = run
		turns = append(turns, growingTurns(id, time.Duration(i)*time.Hour, 200, 2_000, 0)...)
	}
	result := handResult("anthropic", "claude-code", "claude-opus-4-7", 60, turns, 0.02)
	report := Recommend(Inputs{Results: []Result{result}, Runs: runs, Prices: fixturePrices, Now: fixtureStart.Add(48 * time.Hour)})
	lifetime, mix := byKind(report, RuleLifetime), byKind(report, RuleModelMix)
	if len(lifetime) != 1 || len(mix) != 1 {
		t.Fatalf("lifetime %d, model mix %d, want one each over the same sub-agents", len(lifetime), len(mix))
	}
	first, second := lifetime[0], mix[0]
	if first.Rank > second.Rank {
		first, second = second, first
	}
	if strings.Contains(report.Actions[0].Rationale, "Not additive") || !strings.Contains(second.Rationale, "Not additive: it shares runs with ") || !strings.Contains(second.Rationale, fmt.Sprintf("#%d on 12", first.Rank)) || !strings.Contains(second.Rationale, "of its 12 runs, whose quota those levers price too") {
		t.Errorf("overlap: #%d %q / #%d %q", first.Rank, first.Rationale, second.Rank, second.Rationale)
	}
	// A second window of the same provider prices the same runs, but its quota
	// is another: overlaps are named within one window only.
	short := handResult("anthropic", "claude-code", "claude-opus-4-7", 40, turns, 0.02)
	short.Window.Pool, short.Window.Name, short.Window.Minutes, short.Window.ResetAt = "five_hour", "300m", 300, fixtureStart.Add(50*time.Hour)
	both := Recommend(Inputs{Results: []Result{result, short}, Runs: runs, Prices: fixturePrices, Now: fixtureStart.Add(48 * time.Hour)})
	named := regexp.MustCompile(`#(\d+) on \d+`)
	windows := map[string]bool{}
	for _, a := range both.Actions {
		windows[a.Window] = true
		for _, m := range named.FindAllStringSubmatch(a.Rationale, -1) {
			rank, _ := strconv.Atoi(m[1])
			if other := both.Actions[rank-1]; other.Window != a.Window {
				t.Errorf("#%d (%s window) names #%d of the %s window as overlapping", a.Rank, a.Window, rank, other.Window)
			}
		}
	}
	if len(windows) != 2 {
		t.Fatalf("windows = %v, want levers in both", windows)
	}
	if !slices.ContainsFunc(Scoring, func(line string) bool { return strings.HasPrefix(line, "quota levers are not additive") }) {
		t.Error("the scoring notes must say quota levers are not additive")
	}
	for _, a := range []Action{first, second} {
		var ids []string
		for _, ref := range a.Evidence {
			ids = append(ids, ref.ID)
		}
		if len(ids) != maxDrill || !strings.Contains(a.Verify, "--left-run-ids "+strings.Join(ids, ",")+" ") || !strings.Contains(a.Verify, fmt.Sprintf("its %d evidence runs (of the 12 runs it prices) against the runs since the change", maxDrill)) {
			t.Errorf("%s verify = %q, want all %d evidence runs and the cap stated", a.Kind, a.Verify, len(ids))
		}
		if strings.Contains(a.Verify, "--right-window") || !strings.Contains(a.Verify, `--right-filter-json '{"runner_type":"claude-code","window":{"custom":{"from":"<time of the change>","to":"<now>"}}}'`) {
			t.Errorf("%s right side = %q, want the same runner's runs since the change", a.Kind, a.Verify)
		}
	}
}

// A recommendation from an opportunity states what its number counts, as the
// opportunity names it: a bypass replaces calls, it does not save them.
func TestOpportunityActionsStateWhatTheirNumberCounts(t *testing.T) {
	report := opportunities.Report{Opportunities: []opportunities.Opportunity{{Kind: opportunities.KindBypass, Owner: "test-genie", Subject: "test-genie via HTTP", Sessions: 7, Occurrences: 37, CallsSaved: 22, SavedLabel: "calls replaced", SavedBasis: "direct reads; 22 of 37 counted"}}}
	rows := opportunityActions(report, toolfriction.Totals{BypassCalls: 100}, 50)
	if len(rows) != 1 || !strings.Contains(rows[0].Rationale, "37 occurrences in 7 sessions; calls replaced = direct reads; 22 of 37 counted") {
		t.Fatalf("rows = %+v", rows)
	}
}
