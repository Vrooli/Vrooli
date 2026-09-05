// Package trials is the empirical local-model gate: it generates a task suite
// from the Guide space, dispatches each task through agent-manager's real
// sandboxed-agent primitive (profile reconcile-scenario → task create → run
// create --run-mode sandboxed → poll run get → run diff), then EVALUATES the
// produced evidence against a fixture oracle it owns
// — deterministic checks where possible, an agent-judge fallback otherwise — and
// records success-rate + tokens + wall-time as a historical trend. Readiness is
// ultimately PROVEN here, not declared from coverage. It also reports the
// recursive Guide-gate-coverage metric (% of Guide tasks with a live gate).
//
// The split of responsibility is deliberate: agent-manager is ONLY the
// sandboxed-agent spawner — it returns evidence (a sandbox diff + token/time
// metrics). Deciding whether the SWE task was actually solved is MoM's job and
// lives in the Evaluator. The Runner never returns a PASS/FAIL; it returns
// evidence (or an honest VerdictError if the spawn/collection itself failed).
//
// Layering mirrors the canonical domain pattern:
//
//	handler → Service → {TaskGenerator, FixtureResolver, Runner, Evaluator, Repository}
//	             ↑            ↑              ↑              ↑          ↑           ↑
//	        (proto edge)  Guide space   fixture corpus  agent-mgr  oracle/judge  history+gates
//	                                                  (all faked in tests)
//
// Trials are EXPLICIT-INVOCATION ONLY — RunTrials never runs on a hot path, is
// always sandboxed, and CI never dispatches a live model (the Runner + Evaluator
// seams are faked). The proto wire types live one floor up and never import this
// package (api-steer §7). See docs/concepts/DOMAINS.md (trials) and
// docs/concepts/FLOWS.md (trial lifecycle).
package trials

import "time"

// Verdict is the outcome of a single run.
type Verdict int

const (
	VerdictUnspecified Verdict = iota
	VerdictPass                // deterministic check passed, or judge pass
	VerdictFail                // ran but produced a wrong / insufficient result
	VerdictError               // the run itself errored (sandbox / agent failure)
)

// Suite names — the SWE-task families generated from the Guide space, plus the
// negative/honesty family.
const (
	SuiteAddFeature = "add-feature"
	SuiteResearch   = "research"
	SuiteComprehend = "comprehend"
	SuiteBugfix     = "bugfix"
	SuiteNegative   = "negative"
)

// TrialTask is one empirical task in the suite, generated from the Guide space.
type TrialTask struct {
	ID          string
	Suite       string
	GuideTaskID string // the Guide-space row this exercises
	Description string
	Negative    bool // a negative / honesty case
}

// TrialRun is one local-model run, dispatched via agent-manager's sandboxed
// primitive and judged by the Evaluator. GuideTaskID is carried for
// gate-coverage accounting and FixtureRev for idempotency (neither is on the
// proto wire form — both are owned-state columns).
type TrialRun struct {
	ID             string
	TaskID         string
	Suite          string
	Model          string
	GuideTaskID    string
	FixtureRev     string // content revision of the fixture this run exercised
	Verdict        Verdict
	Tokens         int64
	DurationMs     int64
	SandboxDiffRef string
	At             time.Time
}

// HistoryPoint is an aggregated metric point in the trend.
type HistoryPoint struct {
	At               time.Time
	SuccessRate      float64
	MedianTokens     int64
	MedianDurationMs int64
	RunCount         int
}

// History is the trend plus the most recent runs.
type History struct {
	Points     []HistoryPoint
	RecentRuns []TrialRun
}

// GateCoverage is the recursive Guide-gate-coverage metric.
type GateCoverage struct {
	GuideTasksTotal    int
	GuideTasksWithGate int
	Ratio              float64
}
