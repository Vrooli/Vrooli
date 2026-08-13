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
	"strings"
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

// SecurityFriction measures only the observable security-phase loop. A
// failure followed by a later passed observation for the same scenario is a
// green transition; repeated failures before that transition are recurrence.
// The timestamps come from completed server-owned runs, so a missing run is
// not treated as a zero-duration success.
type SecurityFriction struct {
	FailedAttempts     int           `json:"failedAttempts,omitempty"`
	GreenTransitions   int           `json:"greenTransitions,omitempty"`
	RecurringFailures  int           `json:"recurringFailures,omitempty"`
	UnknownFindings    int           `json:"unknownFindings,omitempty"`
	BlockedActions     int           `json:"blockedActions,omitempty"`
	RepairAttempts     int           `json:"repairAttempts,omitempty"`
	RepairSuccesses    int           `json:"repairSuccesses,omitempty"`
	ProviderOutages    int           `json:"providerOutages,omitempty"`
	TimeToGreenSamples int           `json:"timeToGreenSamples,omitempty"`
	TimeToGreen        DurationStats `json:"timeToGreen,omitempty"`
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
	Phase               string                `json:"phase"`
	Provider            string                `json:"provider,omitempty"`
	FindingSource       string                `json:"findingSource,omitempty"`
	TotalObservations   int                   `json:"totalObservations"`
	Passed              int                   `json:"passed"`
	Failed              int                   `json:"failed"`
	Skipped             int                   `json:"skipped"`
	Degraded            int                   `json:"degraded"`
	Availability        float64               `json:"availability"`
	FailureRate         float64               `json:"failureRate"`
	MetricsAdopted      int                   `json:"metricsAdopted"`
	SkipReasons         []LabeledCount        `json:"skipReasons,omitempty"`
	Classifications     []LabeledCount        `json:"classifications,omitempty"`
	Duration            DurationStats         `json:"duration"`
	WorstScenarios      []ScenarioFailureRate `json:"worstScenarios,omitempty"`
	SecurityFriction    SecurityFriction      `json:"securityFriction,omitempty"`
	ConsecutiveFailures int                   `json:"consecutiveFailures,omitempty"`
	FailureStreakSince  time.Time             `json:"failureStreakSince,omitempty"`
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
// normalized (lower-case) phase name to its catalog metadata; it is built
// authoritatively from the catalog, so its key set IS the live-phase set.
// Observations whose phase is absent from the map (legacy pseudo-phases like
// `coverage`/`lint` from historical runs) are excluded so the rollup stays ⊆
// catalog.
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
	phase                string
	provider             string
	findingSource        string
	total                int
	passed               int
	failed               int
	skipped              int
	degraded             int
	metricsAdopted       int
	skipReasons          map[string]int
	classifications      map[string]int
	durations            []int
	perScenario          map[string]*scenarioAcc
	securityObservations []execution.PhaseObservation
	statuses             []execution.PhaseObservation
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
		meta, known := phaseMeta[obs.PhaseName]
		if !known {
			// Non-catalog (legacy) phase — historical runs persist pseudo-phases
			// like `coverage`/`lint` that are no longer catalog phases. Drop them
			// so the rollup is ⊆ catalog and consumers (UI, meta-opt) see only
			// live phases. phaseMeta is built authoritatively from the catalog, so
			// membership is the catalog-scope test.
			continue
		}
		acc := accs[obs.PhaseName]
		if acc == nil {
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
	a.statuses = append(a.statuses, obs)
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
	if strings.EqualFold(a.findingSource, "security") &&
		(obs.Status == "passed" || obs.Status == "failed") {
		a.securityObservations = append(a.securityObservations, obs)
	}
}

func (a *phaseAccumulator) finalize() PhaseReliability {
	executed := a.passed + a.failed
	return PhaseReliability{
		Phase:               a.phase,
		Provider:            a.provider,
		FindingSource:       a.findingSource,
		TotalObservations:   a.total,
		Passed:              a.passed,
		Failed:              a.failed,
		Skipped:             a.skipped,
		Degraded:            a.degraded,
		Availability:        ratio(executed, a.total),
		FailureRate:         ratio(a.failed, executed),
		MetricsAdopted:      a.metricsAdopted,
		SkipReasons:         histogram(a.skipReasons),
		Classifications:     histogram(a.classifications),
		Duration:            durationStats(a.durations),
		WorstScenarios:      worstScenarios(a.perScenario),
		SecurityFriction:    securityFriction(a.phase, a.findingSource, a.securityObservations),
		ConsecutiveFailures: consecutiveFailureCount(a.statuses),
		FailureStreakSince:  failureStreakSince(a.statuses),
	}
}

func consecutiveFailureCount(observations []execution.PhaseObservation) int {
	if len(observations) == 0 {
		return 0
	}
	sort.SliceStable(observations, func(i, j int) bool { return observations[i].CompletedAt.After(observations[j].CompletedAt) })
	count := 0
	for _, obs := range observations {
		if obs.Status != "failed" {
			break
		}
		count++
	}
	return count
}

// failureStreakSince returns the oldest completed observation in the current
// contiguous failure streak. A duration-bearing timestamp lets consumers apply
// a real sustained-degradation window instead of equating sample count with
// elapsed time.
func failureStreakSince(observations []execution.PhaseObservation) time.Time {
	if len(observations) == 0 {
		return time.Time{}
	}
	ordered := append([]execution.PhaseObservation(nil), observations...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].CompletedAt.After(ordered[j].CompletedAt) })
	var oldest time.Time
	for _, obs := range ordered {
		if obs.Status != "failed" {
			break
		}
		if oldest.IsZero() || obs.CompletedAt.Before(oldest) {
			oldest = obs.CompletedAt
		}
	}
	return oldest
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

func securityFriction(_ string, findingSource string, observations []execution.PhaseObservation) SecurityFriction {
	if !strings.EqualFold(findingSource, "security") {
		return SecurityFriction{}
	}
	byScenario := make(map[string][]execution.PhaseObservation)
	for _, obs := range observations {
		if strings.TrimSpace(obs.ScenarioName) == "" {
			continue
		}
		byScenario[obs.ScenarioName] = append(byScenario[obs.ScenarioName], obs)
	}
	var timeToGreen []int
	var friction SecurityFriction
	for _, entries := range byScenario {
		sort.SliceStable(entries, func(i, j int) bool { return entries[i].CompletedAt.Before(entries[j].CompletedAt) })
		failedSinceGreen := false
		var failedAt time.Time
		for _, obs := range entries {
			classification := strings.ToLower(strings.TrimSpace(obs.Classification + " " + obs.RunnabilityReason))
			if strings.Contains(classification, "unknown") {
				friction.UnknownFindings++
			}
			if strings.Contains(classification, "block") || strings.Contains(classification, "deny") {
				friction.BlockedActions++
			}
			if strings.Contains(classification, "repair") || strings.Contains(classification, "remediat") {
				friction.RepairAttempts++
				if obs.Status == "passed" {
					friction.RepairSuccesses++
				}
			}
			if strings.Contains(classification, "provider") && (strings.Contains(classification, "unavailable") || strings.Contains(classification, "outage") || strings.Contains(classification, "stale")) {
				friction.ProviderOutages++
			}
			switch obs.Status {
			case "failed":
				friction.FailedAttempts++
				if failedSinceGreen {
					friction.RecurringFailures++
				}
				failedSinceGreen = true
				if failedAt.IsZero() {
					failedAt = obs.CompletedAt
				}
			case "passed":
				if !failedSinceGreen {
					continue
				}
				friction.GreenTransitions++
				if !failedAt.IsZero() && !obs.CompletedAt.IsZero() && obs.CompletedAt.After(failedAt) {
					timeToGreen = append(timeToGreen, int(obs.CompletedAt.Sub(failedAt).Round(time.Second)/time.Second))
				}
				failedSinceGreen = false
				failedAt = time.Time{}
			}
		}
	}
	friction.TimeToGreenSamples = len(timeToGreen)
	friction.TimeToGreen = durationStats(timeToGreen)
	return friction
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
