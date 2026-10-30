package usageattribution

import (
	"fmt"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/transcriptindex"
	"agent-manager/internal/usageforecast"
)

// Shared fixtures: a synthetic week whose quota truth is known, so tests can
// assert what the attribution and the rules must conclude.

var fixtureStart = time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)

// Fixture models priced like the live price book: cached input is a tenth
// of input; terra costs 5× luna per token.
const (
	modelTerra = "gpt-6.1-terra"
	modelLuna  = "gpt-6-luna"
	codex      = string(domain.RunnerTypeCodex)
)

func fixturePrices(_, model string) Price {
	switch model {
	case modelTerra:
		return Price{Input: 1e-6, CacheRead: 5e-8, Output: 5e-6, Known: true}
	case modelLuna:
		return Price{Input: 2e-7, CacheRead: 1e-8, Output: 1e-6, Known: true}
	}
	return Price{}
}

// usageSpec is one session's steady usage in the fixture week.
type usageSpec struct {
	run     Run
	model   string
	every   time.Duration
	reports int // reports per turn
	prompt  int // prompt tokens per report (all but 2k cached), at the first turn
	growth  int // prompt tokens added per turn: a worker's context grows
	output  int
	facts   transcriptindex.TurnFacts
	from    time.Duration
	until   time.Duration
}

func (u usageSpec) turns() []Turn {
	var out []Turn
	end := u.until
	if end == 0 {
		end = 7 * 24 * time.Hour
	}
	for i, at := 0, u.from+u.every; at <= end; i, at = i+1, at+u.every {
		prompt := u.prompt + u.growth*i
		tokens := domain.UsageCounters{InputTokens: 2000 * u.reports, CacheReadTokens: (prompt - 2000) * u.reports, OutputTokens: u.output * u.reports}
		out = append(out, Turn{At: fixtureStart.Add(at), RunID: u.run.ID, Runner: codex, Model: u.model, Tokens: tokens, Reports: u.reports, Facts: u.facts})
	}
	return out
}

// truthWindow builds the integer used-percent series a provider would report
// for turns whose true cost is pointsPerUSD per unit of API price, plus
// off-disk burn at offDisk(hour) points per hour.
func truthWindow(turns []Turn, pointsPerUSD map[string]float64, offDisk func(hour int) float64, hours int) Window {
	w := Window{Provider: "openai", Pool: "primary", Name: "10080m", Minutes: 10080, Start: fixtureStart, ResetAt: fixtureStart.Add(7 * 24 * time.Hour)}
	w.Series = []usageforecast.Point{{At: fixtureStart}}
	exact, reported := 0.0, 0.0
	byMinute := map[time.Time]float64{}
	for _, turn := range turns {
		byMinute[turn.At] += pointsPerUSD[turn.Model] * fixturePrices(turn.Runner, turn.Model).cost(turn.Tokens)
	}
	for minute := 1; minute <= hours*60; minute++ {
		at := fixtureStart.Add(time.Duration(minute) * time.Minute)
		exact += byMinute[at] + offDisk((minute-1)/60)/60
		if floor := float64(int(exact)); floor > reported {
			reported = floor
			w.Series = append(w.Series, usageforecast.Point{At: at, Percent: reported})
		}
		w.LastAt = at
	}
	w.UsedPercent = reported
	return w
}

// draftRunTokens is one draft run's total: six turns of two 30k reports.
var draftRunTokens = domain.UsageCounters{InputTokens: 24_000, CacheReadTokens: 336_000, OutputTokens: 18_000}

func fixtureRun(id, session string, child bool) Run {
	return Run{ID: id, Label: "run " + id, Source: SourceAgentManager, Runner: codex, Session: session, SessionLabel: "session " + session, Child: child}
}

// weekFixture is the shared week: a dominant long-context orchestrator with
// a sub-agent burst, an idle poller, a screenshot-heavy session and a
// workload that ran on both terra and luna.
type weekFixture struct {
	turns    []Turn
	runs     map[string]Run
	episodes []Episode
	outcomes []Outcome
	window   Window
}

func newWeekFixture() weekFixture {
	f := weekFixture{runs: map[string]Run{}}
	add := func(spec usageSpec) {
		f.runs[spec.run.ID] = spec.run
		f.turns = append(f.turns, spec.turns()...)
	}
	// Dominant consumer: starts at a 50k floor and grows to ≈290k over the
	// week, a request burst every 10 minutes.
	add(usageSpec{run: fixtureRun("orchestrator", "research", false), model: modelTerra, every: 10 * time.Minute, reports: 3, prompt: 60_000, growth: 400, output: 800})
	// Workers start at the 50k floor and grow to ≈250k over 48 requests.
	for i := 0; i < 6; i++ {
		add(usageSpec{run: fixtureRun(fmt.Sprintf("subagent-%d", i), "research", true), model: modelTerra, every: 30 * time.Minute, reports: 2, prompt: 50_000, growth: 8_500, output: 500, from: 24 * time.Hour, until: 36 * time.Hour})
	}
	// Idle poller: waits with tiny output.
	poller := fixtureRun("poller", "heartbeat", false)
	add(usageSpec{run: poller, model: modelLuna, every: 20 * time.Minute, reports: 1, prompt: 40_000, output: 50, facts: transcriptindex.TurnFacts{ToolCalls: 1, ToolKinds: map[string]int{"wait": 1}}})
	// Screenshot-heavy browser session.
	add(usageSpec{run: fixtureRun("browser", "ui-review", false), model: modelLuna, every: 2 * time.Hour, reports: 2, prompt: 30_000, output: 400, facts: transcriptindex.TurnFacts{ImagesIn: 4, ImagesInHighDetail: 3, ImageBytesIn: 4 << 20}})
	// A workload run on both models: terra runs consume this window.
	for i := 0; i < 4; i++ {
		run := fixtureRun(fmt.Sprintf("draft-terra-%d", i), fmt.Sprintf("draft-terra-%d", i), false)
		run.Workload = "workflow_node:draft"
		add(usageSpec{run: run, model: modelTerra, every: time.Hour, reports: 2, prompt: 30_000, output: 1500, from: time.Duration(i) * 30 * time.Hour, until: time.Duration(i)*30*time.Hour + 6*time.Hour})
		f.outcomes = append(f.outcomes, Outcome{RunID: run.ID, Runner: codex, Model: modelTerra, Workload: run.Workload, Succeeded: i != 3, Tokens: draftRunTokens})
	}
	for i := 0; i < 6; i++ {
		f.outcomes = append(f.outcomes, Outcome{RunID: fmt.Sprintf("draft-luna-%d", i), Runner: codex, Model: modelLuna, Workload: "workflow_node:draft", Succeeded: true, Tokens: draftRunTokens})
	}
	for i := 0; i < 5; i++ {
		f.episodes = append(f.episodes, Episode{RunID: "orchestrator", EpisodeID: fmt.Sprintf("ep-%d", i), Pattern: "command-failure", Fingerprint: "failure:rg", Turns: 30, Tokens: 400_000})
	}
	// True quota: terra 0.4 points per USD, luna 0.6 (luna is a little dearer
	// per USD than its list price), plus 0.02 pts/h of off-disk use.
	f.window = truthWindow(f.turns, map[string]float64{modelTerra: 0.4, modelLuna: 0.6}, steady(0.02), 6*24)
	return f
}

func steady(perHour float64) func(int) float64 { return func(int) float64 { return perHour } }

// burst is off-disk use of perHour during [from, to) hours only: another
// machine or a provider web app working for a while.
func burst(from, to int, perHour float64) func(int) float64 {
	return func(hour int) float64 {
		if hour >= from && hour < to {
			return perHour
		}
		return 0
	}
}
