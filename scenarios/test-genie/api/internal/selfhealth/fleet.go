package selfhealth

import (
	"context"
	"sort"
	"strings"
	"time"

	"test-genie/internal/execution"
)

// maxTopFindingSources / maxFleetScenariosRanked bound the fleet ledger's
// ranked lists so the surface stays digestible; the full per-scenario list is
// still returned (callers paginate/render as needed).
const maxTopFindingSources = 8

// fleetObservationSource is the read seam the fleet ledger composes over —
// satisfied by *execution.SuiteExecutionRepository. It adds the per-scenario
// run rollup to the phase observations the per-phase ledger already consumes.
type fleetObservationSource interface {
	AggregatePhaseObservations(ctx context.Context, since time.Time, limit int) ([]execution.PhaseObservation, error)
	AggregateScenarioRuns(ctx context.Context, since time.Time, limit int) ([]execution.ScenarioRunRollup, error)
}

// FleetScenarioHealth is one scenario's health rollup over the window. Every
// field is computed from STORED runs — this is an aggregation over what already
// executed, never a live fleet run. AgeDays/LastRunAt make staleness explicit.
type FleetScenarioHealth struct {
	Scenario     string    `json:"scenario"`
	Runs         int       `json:"runs"`
	PassedRuns   int       `json:"passedRuns"`
	FailedRuns   int       `json:"failedRuns"`
	Availability float64   `json:"availability"`
	FailureRate  float64   `json:"failureRate"`
	Issues       int       `json:"issues"` // failed phase observations in the window
	LastRunAt    time.Time `json:"lastRunAt"`
	LastOutcome  string    `json:"lastOutcome"`
	AgeDays      float64   `json:"ageDays"`
}

// FleetFindingSource is one finding-source's issue count across the fleet — the
// "where do failures cluster" signal for the meta-optimization lens.
type FleetFindingSource struct {
	Source string `json:"source"`
	Issues int    `json:"issues"`
}

// FleetLedger is the assembled fleet-wide health snapshot, computed-on-read over
// stored runs. CapturedAt is the as-of stamp for the WHOLE rollup; each scenario
// additionally carries its own LastRunAt so a stale datum can never read as
// fresh. NeverTestedInWindow is honest about coverage gaps when a fleet roster
// is supplied.
type FleetLedger struct {
	WindowDays          int                   `json:"windowDays"`
	CapturedAt          time.Time             `json:"capturedAt"`
	ScenariosTested     int                   `json:"scenariosTested"`
	ScenariosTotal      int                   `json:"scenariosTotal"`
	TotalRuns           int                   `json:"totalRuns"`
	TotalIssues         int                   `json:"totalIssues"`
	Scenarios           []FleetScenarioHealth `json:"scenarios"`
	TopFindingSources   []FleetFindingSource  `json:"topFindingSources"`
	NeverTestedInWindow []string              `json:"neverTestedInWindow"`
}

// FleetBuilder assembles fleet ledgers from a fleet observation source.
type FleetBuilder struct {
	source fleetObservationSource
	window time.Duration
	now    func() time.Time
}

// NewFleetBuilder returns a fleet ledger builder. A zero window uses
// DefaultWindow (shared with the per-scenario ledger).
func NewFleetBuilder(source fleetObservationSource, window time.Duration) *FleetBuilder {
	if window <= 0 {
		window = DefaultWindow
	}
	return &FleetBuilder{source: source, window: window, now: time.Now}
}

// Build computes the fleet ledger over the configured window. roster, when
// non-nil, is the full set of fleet scenario names; the difference between it
// and the tested set is surfaced as NeverTestedInWindow (an explicit coverage
// gap, never a silent zero). When roster is nil, ScenariosTotal == tested and
// the never-tested list is empty (unknown, not asserted).
func (b *FleetBuilder) Build(ctx context.Context, roster []string) (*FleetLedger, error) {
	now := b.now().UTC()
	since := now.Add(-b.window)

	runRollups, err := b.source.AggregateScenarioRuns(ctx, since, 0)
	if err != nil {
		return nil, err
	}
	observations, err := b.source.AggregatePhaseObservations(ctx, since, 0)
	if err != nil {
		return nil, err
	}

	// Per-scenario issue counts (failed phase observations) + fleet-wide
	// finding-source clustering, in one pass over the phase observations.
	issuesByScenario := map[string]int{}
	issuesBySource := map[string]int{}
	for _, obs := range observations {
		if obs.Status != "failed" {
			continue
		}
		if name := strings.TrimSpace(obs.ScenarioName); name != "" {
			issuesByScenario[name]++
		}
		source := strings.TrimSpace(obs.FindingSource)
		if source == "" {
			source = "unattributed"
		}
		issuesBySource[source]++
	}

	ledger := &FleetLedger{
		WindowDays: int(b.window.Hours() / 24),
		CapturedAt: now,
	}

	tested := make(map[string]struct{}, len(runRollups))
	for _, roll := range runRollups {
		name := strings.TrimSpace(roll.ScenarioName)
		if name == "" {
			continue
		}
		tested[name] = struct{}{}
		failed := roll.Runs - roll.Passed
		if failed < 0 {
			failed = 0
		}
		health := FleetScenarioHealth{
			Scenario:     name,
			Runs:         roll.Runs,
			PassedRuns:   roll.Passed,
			FailedRuns:   failed,
			Availability: ratio(roll.Passed, roll.Runs),
			FailureRate:  ratio(failed, roll.Runs),
			Issues:       issuesByScenario[name],
			LastRunAt:    roll.LastCompletedAt,
			LastOutcome:  roll.LastOutcome,
		}
		if !roll.LastCompletedAt.IsZero() {
			health.AgeDays = now.Sub(roll.LastCompletedAt).Hours() / 24
		}
		ledger.Scenarios = append(ledger.Scenarios, health)
		ledger.TotalRuns += roll.Runs
		ledger.TotalIssues += health.Issues
	}

	// Rank most-errored first: failure rate, then absolute failures, then issue
	// count, then name (deterministic). Healthy + fresh scenarios sort last.
	sort.SliceStable(ledger.Scenarios, func(i, j int) bool {
		a, c := ledger.Scenarios[i], ledger.Scenarios[j]
		if a.FailureRate != c.FailureRate {
			return a.FailureRate > c.FailureRate
		}
		if a.FailedRuns != c.FailedRuns {
			return a.FailedRuns > c.FailedRuns
		}
		if a.Issues != c.Issues {
			return a.Issues > c.Issues
		}
		return a.Scenario < c.Scenario
	})

	ledger.ScenariosTested = len(tested)
	ledger.TopFindingSources = topFindingSources(issuesBySource)

	if roster != nil {
		seen := map[string]struct{}{}
		for _, name := range roster {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, ok := tested[name]; ok {
				continue
			}
			if _, dup := seen[name]; dup {
				continue
			}
			seen[name] = struct{}{}
			ledger.NeverTestedInWindow = append(ledger.NeverTestedInWindow, name)
		}
		sort.Strings(ledger.NeverTestedInWindow)
		ledger.ScenariosTotal = ledger.ScenariosTested + len(ledger.NeverTestedInWindow)
	} else {
		ledger.ScenariosTotal = ledger.ScenariosTested
	}

	return ledger, nil
}

func topFindingSources(bySource map[string]int) []FleetFindingSource {
	out := make([]FleetFindingSource, 0, len(bySource))
	for source, n := range bySource {
		out = append(out, FleetFindingSource{Source: source, Issues: n})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Issues != out[j].Issues {
			return out[i].Issues > out[j].Issues
		}
		return out[i].Source < out[j].Source
	})
	if len(out) > maxTopFindingSources {
		out = out[:maxTopFindingSources]
	}
	return out
}
