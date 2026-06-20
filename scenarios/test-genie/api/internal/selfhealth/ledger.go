// Package selfhealth builds Test Genie's compute-on-read reliability and
// performance ledger over a recent window of persisted runs. It holds NO raw
// SQL: it composes over the engine-neutral execution.SuiteExecutionRepository
// aggregation methods (storage-steer §8), so the ledger is engine-portable.
//
// The ledger is snapshot-only — no persisted rollups, no trend history (those
// are deliberately deferred). Every metric is advisory; nothing here gates a
// build.
package selfhealth

import (
	"context"
	"sort"
	"time"

	"test-genie/internal/execution"
)

// DefaultWindow is the default look-back window for the ledger. It is wider than
// the 50-row UI history cap because reliability statistics need more samples
// than the most-recent-runs list shows.
const DefaultWindow = 30 * 24 * time.Hour

// maxWorstScenariosPerPhase bounds the worst-offender ranking per phase.
const maxWorstScenariosPerPhase = 5

// minRunsForWorstRanking requires a scenario to have at least this many executed
// observations of a phase before it can appear in the worst-offenders ranking,
// so a single unlucky run does not dominate.
const minRunsForWorstRanking = 2

// observationSource is the read seam the ledger composes over — satisfied by
// *execution.SuiteExecutionRepository. Defined here (consumer-side) so the
// ledger depends only on the data it needs.
type observationSource interface {
	AggregatePhaseObservations(ctx context.Context, since time.Time, limit int) ([]execution.PhaseObservation, error)
	CountRunOutcomes(ctx context.Context, since time.Time, limit int) ([]execution.RunOutcomeCount, error)
}

// PhaseMeta is catalog metadata for one phase, supplied by the caller (built
// from the phase catalog) so the ledger can attribute phases to providers
// without importing the catalog package.
type PhaseMeta struct {
	Provider      string // delegated provider scenario; "" for native phases
	FindingSource string // lower-case finding-source token; "" when none
	Delegated     bool
}

// LabeledCount is one bucket of a histogram (skip reason, classification, …).
type LabeledCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// DurationStats summarizes execution durations (seconds) for executed phases.
type DurationStats struct {
	Samples int `json:"samples"`
	P50     int `json:"p50"`
	P95     int `json:"p95"`
	Min     int `json:"min"`
	Max     int `json:"max"`
	Avg     int `json:"avg"`
}

// OutcomeCount is one bucket of the run-level terminal-outcome histogram.
type OutcomeCount struct {
	Outcome string `json:"outcome"`
	Count   int    `json:"count"`
}

// ScenarioFailureRate ranks a scenario's failure rate for one phase.
type ScenarioFailureRate struct {
	Scenario    string  `json:"scenario"`
	Executed    int     `json:"executed"`
	Failures    int     `json:"failures"`
	FailureRate float64 `json:"failureRate"`
}

// PhaseReliability is the per-phase reliability + performance rollup.
type PhaseReliability struct {
	Phase             string                `json:"phase"`
	Provider          string                `json:"provider,omitempty"`
	FindingSource     string                `json:"findingSource,omitempty"`
	TotalObservations int                   `json:"totalObservations"`
	Passed            int                   `json:"passed"`
	Failed            int                   `json:"failed"`
	Skipped           int                   `json:"skipped"`
	Degraded          int                   `json:"degraded"`
	Availability      float64               `json:"availability"`
	FailureRate       float64               `json:"failureRate"`
	MetricsAdopted    int                   `json:"metricsAdopted"`
	SkipReasons       []LabeledCount        `json:"skipReasons,omitempty"`
	Classifications   []LabeledCount        `json:"classifications,omitempty"`
	Duration          DurationStats         `json:"duration"`
	WorstScenarios    []ScenarioFailureRate `json:"worstScenarios,omitempty"`
}

// ProviderReliability is the per-provider rollup across its phase(s).
type ProviderReliability struct {
	Provider          string        `json:"provider"`
	Phases            []string      `json:"phases"`
	TotalObservations int           `json:"totalObservations"`
	Passed            int           `json:"passed"`
	Failed            int           `json:"failed"`
	Skipped           int           `json:"skipped"`
	Availability      float64       `json:"availability"`
	FailureRate       float64       `json:"failureRate"`
	MetricsAdopted    int           `json:"metricsAdopted"`
	Duration          DurationStats `json:"duration"`
}

// Ledger is the assembled reliability + performance snapshot.
type Ledger struct {
	WindowDays   int                   `json:"windowDays"`
	RunCount     int                   `json:"runCount"`
	Availability float64               `json:"availability"`
	RunOutcomes  []OutcomeCount        `json:"runOutcomes"`
	Phases       []PhaseReliability    `json:"phases"`
	Providers    []ProviderReliability `json:"providers"`
}

// Builder assembles ledgers from an observation source.
type Builder struct {
	source observationSource
	window time.Duration
	now    func() time.Time
}

// NewBuilder returns a ledger Builder over the given source. A zero window uses
// DefaultWindow.
func NewBuilder(source observationSource, window time.Duration) *Builder {
	if window <= 0 {
		window = DefaultWindow
	}
	return &Builder{source: source, window: window, now: time.Now}
}

// Build computes the ledger over the configured window. phaseMeta maps a
// normalized (lower-case) phase name to its catalog metadata; phases absent from
// the map are still reported (with empty provider).
func (b *Builder) Build(ctx context.Context, phaseMeta map[string]PhaseMeta) (*Ledger, error) {
	since := b.now().UTC().Add(-b.window)

	observations, err := b.source.AggregatePhaseObservations(ctx, since, 0)
	if err != nil {
		return nil, err
	}
	outcomeCounts, err := b.source.CountRunOutcomes(ctx, since, 0)
	if err != nil {
		return nil, err
	}

	ledger := &Ledger{
		WindowDays:  int(b.window.Hours() / 24),
		RunOutcomes: runOutcomes(outcomeCounts),
	}
	ledger.RunCount, ledger.Availability = runAvailability(outcomeCounts)
	ledger.Phases = buildPhaseReliability(observations, phaseMeta)
	ledger.Providers = buildProviderReliability(ledger.Phases, observations, phaseMeta)
	return ledger, nil
}

func runOutcomes(counts []execution.RunOutcomeCount) []OutcomeCount {
	out := make([]OutcomeCount, 0, len(counts))
	for _, c := range counts {
		label := c.TerminalOutcome
		if label == "" {
			label = "unknown"
		}
		out = append(out, OutcomeCount{Outcome: label, Count: c.Count})
	}
	return out
}

// runAvailability returns total runs and the suite availability: the fraction of
// runs that produced a result (passed or failed) out of all terminal runs. This
// is the metric Phase B4a corrects — catastrophic runs now count in the
// denominator.
func runAvailability(counts []execution.RunOutcomeCount) (int, float64) {
	total, produced := 0, 0
	for _, c := range counts {
		total += c.Count
		switch execution.TerminalOutcome(c.TerminalOutcome) {
		case execution.TerminalOutcomePassed, execution.TerminalOutcomeFailed:
			produced += c.Count
		}
	}
	if total == 0 {
		return 0, 0
	}
	return total, ratio(produced, total)
}

// phaseAccumulator gathers per-phase signal during a single pass.
type phaseAccumulator struct {
	phase           string
	provider        string
	findingSource   string
	total           int
	passed          int
	failed          int
	skipped         int
	degraded        int
	metricsAdopted  int
	skipReasons     map[string]int
	classifications map[string]int
	durations       []int
	perScenario     map[string]*scenarioAcc
}

type scenarioAcc struct {
	executed int
	failures int
}

func buildPhaseReliability(observations []execution.PhaseObservation, phaseMeta map[string]PhaseMeta) []PhaseReliability {
	accs := make(map[string]*phaseAccumulator)
	for _, obs := range observations {
		if obs.PhaseName == "" {
			continue
		}
		acc := accs[obs.PhaseName]
		if acc == nil {
			meta := phaseMeta[obs.PhaseName]
			acc = &phaseAccumulator{
				phase:           obs.PhaseName,
				provider:        meta.Provider,
				findingSource:   obs.FindingSource,
				skipReasons:     map[string]int{},
				classifications: map[string]int{},
				perScenario:     map[string]*scenarioAcc{},
			}
			if acc.findingSource == "" {
				acc.findingSource = meta.FindingSource
			}
			accs[obs.PhaseName] = acc
		}
		acc.observe(obs)
	}

	phases := make([]PhaseReliability, 0, len(accs))
	for _, acc := range accs {
		phases = append(phases, acc.finalize())
	}
	sort.Slice(phases, func(i, j int) bool { return phases[i].Phase < phases[j].Phase })
	return phases
}

func (a *phaseAccumulator) observe(obs execution.PhaseObservation) {
	a.total++
	switch obs.Status {
	case "passed":
		a.passed++
	case "failed":
		a.failed++
	case "skipped":
		a.skipped++
	}
	if obs.RunnabilityVerdict == "run_degraded" {
		a.degraded++
	}
	if obs.MetricsPresent {
		a.metricsAdopted++
	}
	executed := obs.Status == "passed" || obs.Status == "failed"
	if executed && obs.DurationSeconds > 0 {
		a.durations = append(a.durations, obs.DurationSeconds)
	}
	if obs.Status == "skipped" {
		reason := obs.RunnabilityReason
		if reason == "" {
			reason = "unspecified"
		}
		a.skipReasons[reason]++
	}
	if obs.Classification != "" {
		a.classifications[obs.Classification]++
	}
	if obs.ScenarioName != "" {
		sc := a.perScenario[obs.ScenarioName]
		if sc == nil {
			sc = &scenarioAcc{}
			a.perScenario[obs.ScenarioName] = sc
		}
		if executed {
			sc.executed++
			if obs.Status == "failed" {
				sc.failures++
			}
		}
	}
}

func (a *phaseAccumulator) finalize() PhaseReliability {
	executed := a.passed + a.failed
	return PhaseReliability{
		Phase:             a.phase,
		Provider:          a.provider,
		FindingSource:     a.findingSource,
		TotalObservations: a.total,
		Passed:            a.passed,
		Failed:            a.failed,
		Skipped:           a.skipped,
		Degraded:          a.degraded,
		Availability:      ratio(executed, a.total),
		FailureRate:       ratio(a.failed, executed),
		MetricsAdopted:    a.metricsAdopted,
		SkipReasons:       histogram(a.skipReasons),
		Classifications:   histogram(a.classifications),
		Duration:          durationStats(a.durations),
		WorstScenarios:    worstScenarios(a.perScenario),
	}
}

func buildProviderReliability(phases []PhaseReliability, observations []execution.PhaseObservation, phaseMeta map[string]PhaseMeta) []ProviderReliability {
	type provAcc struct {
		phases         map[string]struct{}
		total          int
		passed         int
		failed         int
		skipped        int
		metricsAdopted int
		durations      []int
	}
	accs := map[string]*provAcc{}

	get := func(provider string) *provAcc {
		acc := accs[provider]
		if acc == nil {
			acc = &provAcc{phases: map[string]struct{}{}}
			accs[provider] = acc
		}
		return acc
	}

	for _, obs := range observations {
		meta := phaseMeta[obs.PhaseName]
		if meta.Provider == "" {
			continue // native phase: no provider attribution
		}
		acc := get(meta.Provider)
		acc.phases[obs.PhaseName] = struct{}{}
		acc.total++
		switch obs.Status {
		case "passed":
			acc.passed++
		case "failed":
			acc.failed++
		case "skipped":
			acc.skipped++
		}
		if obs.MetricsPresent {
			acc.metricsAdopted++
		}
		if (obs.Status == "passed" || obs.Status == "failed") && obs.DurationSeconds > 0 {
			acc.durations = append(acc.durations, obs.DurationSeconds)
		}
	}

	providers := make([]ProviderReliability, 0, len(accs))
	for name, acc := range accs {
		executed := acc.passed + acc.failed
		phaseList := make([]string, 0, len(acc.phases))
		for p := range acc.phases {
			phaseList = append(phaseList, p)
		}
		sort.Strings(phaseList)
		providers = append(providers, ProviderReliability{
			Provider:          name,
			Phases:            phaseList,
			TotalObservations: acc.total,
			Passed:            acc.passed,
			Failed:            acc.failed,
			Skipped:           acc.skipped,
			Availability:      ratio(executed, acc.total),
			FailureRate:       ratio(acc.failed, executed),
			MetricsAdopted:    acc.metricsAdopted,
			Duration:          durationStats(acc.durations),
		})
	}
	sort.Slice(providers, func(i, j int) bool { return providers[i].Provider < providers[j].Provider })
	return providers
}

func worstScenarios(perScenario map[string]*scenarioAcc) []ScenarioFailureRate {
	ranked := make([]ScenarioFailureRate, 0, len(perScenario))
	for name, sc := range perScenario {
		if sc.executed < minRunsForWorstRanking || sc.failures == 0 {
			continue
		}
		ranked = append(ranked, ScenarioFailureRate{
			Scenario:    name,
			Executed:    sc.executed,
			Failures:    sc.failures,
			FailureRate: ratio(sc.failures, sc.executed),
		})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].FailureRate != ranked[j].FailureRate {
			return ranked[i].FailureRate > ranked[j].FailureRate
		}
		if ranked[i].Failures != ranked[j].Failures {
			return ranked[i].Failures > ranked[j].Failures
		}
		return ranked[i].Scenario < ranked[j].Scenario
	})
	if len(ranked) > maxWorstScenariosPerPhase {
		ranked = ranked[:maxWorstScenariosPerPhase]
	}
	return ranked
}

func histogram(counts map[string]int) []LabeledCount {
	if len(counts) == 0 {
		return nil
	}
	out := make([]LabeledCount, 0, len(counts))
	for label, count := range counts {
		out = append(out, LabeledCount{Label: label, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Label < out[j].Label
	})
	return out
}

func durationStats(values []int) DurationStats {
	if len(values) == 0 {
		return DurationStats{}
	}
	sorted := append([]int(nil), values...)
	sort.Ints(sorted)
	sum := 0
	for _, v := range sorted {
		sum += v
	}
	return DurationStats{
		Samples: len(sorted),
		P50:     percentile(sorted, 0.5),
		P95:     percentile(sorted, 0.95),
		Min:     sorted[0],
		Max:     sorted[len(sorted)-1],
		Avg:     int(float64(sum)/float64(len(sorted)) + 0.5),
	}
}

// percentile mirrors execution.percentileSeconds: nearest-rank over a presorted
// ascending slice.
func percentile(sorted []int, p float64) int {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 || p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	idx := int(p*float64(len(sorted)-1) + 0.5)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func ratio(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}
