package execution

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"test-genie/internal/shared/stats"
)

type CostSample struct {
	Scenario             string
	Phase                string
	Status               string
	WallClockMs          int64
	CPUUserMs            int64
	PeakRSSBytes         int64
	CPUReliability       string
	MemoryReliability    string
	MeasurementScope     string
	PredictedWallClockMs sql.NullInt64
	CacheHit             bool
	CacheAudit           bool
	CacheAuditMismatch   bool
	CacheNoSaving        bool
	// QueueLatencyMs is the wait between the run being requested and it getting
	// a concurrency slot. -1 means unknown: rows recorded before requested_at
	// existed genuinely cannot say, and reporting 0 for them would understate
	// fleet queue latency by claiming measured immediacy.
	QueueLatencyMs int64
}

type CostSummary struct {
	Scenario                           string
	Phase                              string
	SampleCount                        int
	PassingSampleCount                 int
	FailingSampleCount                 int
	ReliableSampleCount                int
	ExcludedSampleCount                int
	TotalWallClockMs                   int64
	MedianWallClockMs                  int64
	P90WallClockMs                     int64
	PassingMedianWallClockMs           int64
	PassingP90WallClockMs              int64
	FailingMedianWallClockMs           int64
	FailingP90WallClockMs              int64
	TotalCPUUserMs                     int64
	MaxPeakRSSBytes                    int64
	ChangeWallClockMs                  int64
	ChangePercent                      float64
	PredictionSampleCount              int
	PredictionErrorTotalMs             int64
	PredictionMeanAbsoluteErrorMs      int64
	PredictionMeanAbsoluteErrorPercent float64
	CacheHitCount                      int
	ExecutedSampleCount                int
	CacheHitRatePercent                float64
	CacheAuditCount                    int
	CacheAuditMismatchCount            int
	CacheNoSavingCount                 int
	CacheAuditWallClockMs              int64
	EstimatedGrossSavedWallClockMs     int64
	EstimatedNetSavedWallClockMs       int64
	// ProviderScenario names the scenario whose provider owns this phase, so a
	// cost row points at something ownable rather than at a phase name.
	ProviderScenario string
	// Queue latency percentiles over the samples that know their request time.
	// -1 when no sample in the bucket does.
	QueueLatencyMedianMs int64
	QueueLatencyP90Ms    int64
	// RepeatFailureWallClockMs is wall-clock spent re-deriving a failure this
	// phase had already produced. See CostReport for how it is attributed.
	RepeatFailureWallClockMs int64
	RepeatFailureSampleCount int
	// MeasurementScopes counts collector generations in the auditable history.
	MeasurementScopes map[string]int `json:"measurementScopes,omitempty"`
}

type CostSource interface {
	CostReport(context.Context, string, time.Time, time.Time) ([]CostSummary, error)
}

// SuiteEnvelopeEstimate is the measured host reservation for one suite. RAM
// and CPU are the p90 of each historical run's maximum simultaneously active
// phase claims; a run's phases are never summed when they did not overlap.
type SuiteEnvelopeEstimate struct {
	Scenario string `json:"scenario"`
	Preset   string `json:"preset"`
	RAMBytes int64  `json:"ramBytes"`
	CPUMilli int64  `json:"cpuMilli"`
	Runs     int    `json:"runs"`
	Reliable bool   `json:"reliable"`
}

const suiteEnvelopeMinRuns = 5

type envelopeInterval struct {
	start, end time.Time
	ram, cpu   int64
}

// SuiteEnvelopeEstimate returns a scenario- and preset-scoped envelope. The
// history schema records phase timestamps so overlapping work can be summed at
// each sweep point. Missing timestamps or unreliable resource readings exclude
// that phase; fewer than five usable runs is an honest unknown.
func (r *SuiteExecutionRepository) SuiteEnvelopeEstimate(ctx context.Context, scenario, preset string) (SuiteEnvelopeEstimate, error) {
	scenario, preset = strings.TrimSpace(scenario), strings.TrimSpace(preset)
	q := `SELECT e.id, p.started_at, p.completed_at, p.wall_clock_ms, p.cpu_user_ms, p.peak_rss_bytes
		FROM suite_execution_phases p JOIN suite_executions e ON e.id = p.execution_id
		WHERE e.scenario_name = ? AND COALESCE(NULLIF(e.preset_used, ''), NULLIF(e.requested_preset, ''), '') = ?
		AND e.completed_at IS NOT NULL AND p.started_at IS NOT NULL AND p.completed_at IS NOT NULL
		AND p.cache_hit = 0
		AND p.cpu_reliability = 'RELIABILITY_RELIABLE' AND p.memory_reliability = 'RELIABILITY_RELIABLE'
		ORDER BY e.completed_at DESC`
	rows, err := r.db.QueryContext(ctx, q, scenario, preset)
	if err != nil {
		return SuiteEnvelopeEstimate{}, err
	}
	defer rows.Close()
	byRun := map[string][]envelopeInterval{}
	for rows.Next() {
		var runID, started, completed string
		var wall, cpu, ram sql.NullInt64
		if err := rows.Scan(&runID, &started, &completed, &wall, &cpu, &ram); err != nil {
			return SuiteEnvelopeEstimate{}, err
		}
		start, startErr := time.Parse(time.RFC3339Nano, started)
		end, endErr := time.Parse(time.RFC3339Nano, completed)
		if startErr != nil || endErr != nil || !end.After(start) || !wall.Valid || wall.Int64 <= 0 || !cpu.Valid || !ram.Valid || ram.Int64 <= 0 {
			continue
		}
		cpuMilli := cpu.Int64 * 1000 / wall.Int64
		if cpuMilli < 100 {
			cpuMilli = 100
		}
		byRun[runID] = append(byRun[runID], envelopeInterval{start: start, end: end, ram: ram.Int64, cpu: cpuMilli})
	}
	if err := rows.Err(); err != nil {
		return SuiteEnvelopeEstimate{}, err
	}
	rams, cpus := make([]int64, 0, len(byRun)), make([]int64, 0, len(byRun))
	for _, intervals := range byRun {
		if valueRAM, valueCPU, ok := maxConcurrentEnvelope(intervals); ok {
			rams, cpus = append(rams, valueRAM), append(cpus, valueCPU)
		}
	}
	estimate := SuiteEnvelopeEstimate{Scenario: scenario, Preset: preset, Runs: len(rams)}
	if len(rams) < suiteEnvelopeMinRuns {
		return estimate, nil
	}
	sort.Slice(rams, func(i, j int) bool { return rams[i] < rams[j] })
	sort.Slice(cpus, func(i, j int) bool { return cpus[i] < cpus[j] })
	estimate.RAMBytes = rams[stats.NearestRankIndex(len(rams), .9)]
	estimate.CPUMilli = cpus[stats.NearestRankIndex(len(cpus), .9)]
	estimate.Reliable = true
	return estimate, nil
}

// SuiteEnvelopeEstimates returns every scenario/preset envelope with enough
// history for the operator admission surface. Unknown combinations are
// intentionally omitted: the absence is the honest "no evidence" state.
func (r *SuiteExecutionRepository) SuiteEnvelopeEstimates(ctx context.Context) ([]SuiteEnvelopeEstimate, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT e.scenario_name, COALESCE(NULLIF(e.preset_used, ''), NULLIF(e.requested_preset, ''), '') FROM suite_executions e WHERE e.completed_at IS NOT NULL ORDER BY e.scenario_name, 2`)
	if err != nil {
		return nil, err
	}
	var keys [][2]string
	for rows.Next() {
		var scenario, preset string
		if err := rows.Scan(&scenario, &preset); err != nil {
			_ = rows.Close()
			return nil, err
		}
		keys = append(keys, [2]string{scenario, preset})
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var out []SuiteEnvelopeEstimate
	for _, key := range keys {
		scenario, preset := key[0], key[1]
		estimate, err := r.SuiteEnvelopeEstimate(ctx, scenario, preset)
		if err != nil {
			return nil, err
		}
		if estimate.Reliable {
			out = append(out, estimate)
		}
	}
	return out, nil
}

func maxConcurrentEnvelope(intervals []envelopeInterval) (int64, int64, bool) {
	type event struct {
		at       time.Time
		ram, cpu int64
		delta    int
	}
	events := make([]event, 0, len(intervals)*2)
	for _, in := range intervals {
		events = append(events, event{at: in.start, ram: in.ram, cpu: in.cpu, delta: 1}, event{at: in.end, ram: in.ram, cpu: in.cpu, delta: -1})
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].at.Equal(events[j].at) {
			return events[i].delta < events[j].delta
		}
		return events[i].at.Before(events[j].at)
	})
	var ram, cpu, maxRAM, maxCPU int64
	for _, e := range events {
		if e.delta < 0 {
			ram -= e.ram
			cpu -= e.cpu
			continue
		}
		ram += e.ram
		cpu += e.cpu
		if ram > maxRAM {
			maxRAM = ram
		}
		if cpu > maxCPU {
			maxCPU = cpu
		}
	}
	return maxRAM, maxCPU, len(events) > 0
}

const calibrationInterval = 7 * 24 * time.Hour

// phaseDurationRiskPercentile is the observed tail used by the scheduler's
// contention guard. A single pathological run must not serialize a phase for
// the whole history window. It matches the cost report's p90 convention and
// should be revisited if timeout escapes or false serializations show
// miscalibration.
const phaseDurationRiskPercentile = 0.90

// deadlineHistoryWindow bounds the history PhaseDurationEstimate reads. It is
// shorter than the 30-day sizing window because the scheduler uses the
// observed p90 and the window bounds how long stale tail behavior can affect
// admission.
const deadlineHistoryWindow = 14 * 24 * time.Hour

// deadlineMinSamples is the minimum observed duration history used by the
// deadline guard. Inclusive nearest-rank returns the maximum for p90 when
// fewer than ten samples exist; that conservative tail is appropriate for a
// capacity claim, but a maximum-as-p90 would incorrectly exclude a phase from
// every batch. Capacity claims deliberately keep their own conservative tail.
const deadlineMinSamples = 10

// minUnmeasurableEvidence is how many measurable-but-never-RELIABLE samples a
// phase must accumulate inside the calibration window before it is treated as
// unmeasurable rather than uncalibrated.
//
// Three is chosen because the distinction it draws is "has not produced a
// reliable sample yet" versus "cannot produce one here", and a single miss
// cannot separate them. Set it lower and a phase that simply has not been run
// uncontended yet is written off; set it higher and a permanently BEST_EFFORT
// phase keeps vetoing concurrency for days while it accumulates evidence.
const minUnmeasurableEvidence = 3

// CalibrationDecision keeps at least one uncontended sample for every planned
// phase that can produce one. It is intentionally separate from the 30-day
// eligibility window in PhaseCostEstimate: this is a sampling cadence, not a
// cache TTL.
//
// The "that can produce one" qualifier is load-bearing. A provider may report
// its resources UNAVAILABLE — workflow-health does, emitting a wall clock with
// no CPU or RSS — and `metricColumns` then stores no reliability at all. Such a
// phase can never satisfy a reliable-sample requirement, so demanding one from
// every planned phase made this function return "calibrate" on literally every
// run, permanently. Measured 2026-08-08: 100% of runs across every scenario
// reported `reliable sample for workflow is older than calibration interval`.
//
// An unmeasurable phase costs nothing to exclude: PhaseCostEstimate already
// returns unknown for it, and an unknown-size phase runs serially by its own
// rule. Excluding it here removes a veto over *other* phases' concurrency,
// which is not something its own unmeasurability entitles it to.
//
// UNAVAILABLE is not the only permanent case, and treating it as the only one
// left a second false positive of the same shape. `Collector.cpuMemReliability`
// in packages/api-core/metrics degrades CPU/RSS to BEST_EFFORT whenever more
// than one collector is active *inside the provider process*. That is a
// property of the provider's own concurrency, not of Test Genie's phase
// scheduling: serializing this run's phases does not quiet a provider that is
// also serving another run, and the global run cap is two. So a phase that
// keeps reporting BEST_EFFORT cannot be calibrated by the remedy the veto
// prescribes, and demanding it forever pins the whole run serial. Measured
// 2026-08-08: the `experience` phase reported BEST_EFFORT and never RELIABLE
// for source-ledger, vrooli-memory, and scenario-to-desktop, and those runs
// executed all twenty phases one at a time (sum of phase durations 102.7 s
// against 103.4 s of wall-clock).
func (r *SuiteExecutionRepository) CalibrationDecision(ctx context.Context, scenario string, phases []string, descriptorDigest string) (bool, string) {
	cutoff := time.Now().UTC().Add(-calibrationInterval).Format(time.RFC3339Nano)
	for _, phase := range phases {
		var observed int
		var measurable int
		var latestReliable sql.NullString
		err := r.db.QueryRowContext(ctx, `
SELECT
  COUNT(*),
  SUM(CASE WHEN p.cpu_reliability IS NOT NULL AND p.cpu_reliability <> '' THEN 1 ELSE 0 END),
  MAX(CASE WHEN p.cpu_reliability = 'RELIABILITY_RELIABLE'
            AND p.memory_reliability = 'RELIABILITY_RELIABLE'
           THEN e.completed_at END)
FROM suite_execution_phases p
JOIN suite_executions e ON e.id = p.execution_id
WHERE e.scenario_name = ? AND p.phase_name = ?
  AND e.completed_at IS NOT NULL
  AND e.completed_at >= ?`, scenario, phase, cutoff).Scan(&observed, &measurable, &latestReliable)
		if err != nil {
			return true, fmt.Sprintf("calibration history for %s could not be read", phase)
		}
		switch {
		case latestReliable.Valid && latestReliable.String >= cutoff:
			// A recent uncontended sample exists. Nothing to calibrate.
		case observed == 0:
			// No history at all inside the window. A phase nobody has measured
			// is measured serially before it is scheduled against anything.
			return true, fmt.Sprintf("no sample for %s inside the calibration interval (%s)", phase, calibrationInterval)
		case measurable == 0:
			// Observed repeatedly, never with a reliability. The provider
			// reports its resources unavailable, so no serial run will ever
			// change this. Reporting it as a calibration need would be a
			// permanent false positive.
		case measurable >= minUnmeasurableEvidence:
			// Measured repeatedly, never reliably. Reaching this branch means
			// zero RELIABLE samples in the window despite the provider stamping
			// a reliability every time, which is the BEST_EFFORT case described
			// above: caused inside the provider process and unreachable from
			// here. Excluded for the same reason as UNAVAILABLE.
		default:
			return true, fmt.Sprintf("reliable sample for %s is older than calibration interval (%s)", phase, calibrationInterval)
		}
	}
	if strings.TrimSpace(descriptorDigest) != "" {
		var recentDescriptor sql.NullString
		if err := r.db.QueryRowContext(ctx, `
SELECT descriptor_snapshot_digest
FROM suite_executions
WHERE scenario_name = ? AND completed_at IS NOT NULL
ORDER BY completed_at DESC LIMIT 1`, scenario).Scan(&recentDescriptor); err == nil && recentDescriptor.Valid && recentDescriptor.String != descriptorDigest {
			return true, "descriptor snapshot changed; first run must be measured serially"
		}
	}
	return false, ""
}

// PhaseCostEstimate adapts durable measured history to scheduler claim units.
// A phase without a reliable sample returns unknown measurements; admission
// uses the orchestrator's named fallback reservation so that one unmeasured
// phase does not serialize unrelated work.
func (r *SuiteExecutionRepository) PhaseCostEstimate(ctx context.Context, scenario, phase string) (ramBytes, cpuMilli int64, reliable bool) {
	type sample struct{ wall, cpu, peak int64 }
	load := func(scope string) ([]sample, error) {
		q := `SELECT COALESCE(p.wall_clock_ms, p.duration_ms), p.cpu_user_ms, p.peak_rss_bytes
			FROM suite_execution_phases p JOIN suite_executions e ON e.id = p.execution_id
			WHERE p.phase_name = ? AND e.completed_at >= ? AND e.completed_at < ?
			AND p.cpu_reliability = 'RELIABILITY_RELIABLE' AND p.memory_reliability = 'RELIABILITY_RELIABLE'`
		q += " AND p.measurement_scope = 'MEASUREMENT_SCOPE_OPERATION'"
		args := []any{strings.TrimSpace(phase), time.Now().UTC().Add(-30 * 24 * time.Hour).Format(time.RFC3339Nano), time.Now().UTC().Add(time.Second).Format(time.RFC3339Nano)}
		if strings.TrimSpace(scope) != "" {
			q += " AND e.scenario_name = ?"
			args = append(args, strings.TrimSpace(scope))
		}
		rows, err := r.db.QueryContext(ctx, q, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []sample
		for rows.Next() {
			var s sample
			var cpu, peak sql.NullInt64
			if err := rows.Scan(&s.wall, &cpu, &peak); err != nil {
				return nil, err
			}
			if cpu.Valid && peak.Valid && s.wall > 0 && peak.Int64 > 0 {
				s.cpu, s.peak = cpu.Int64, peak.Int64
				out = append(out, s)
			}
		}
		return out, rows.Err()
	}
	target, err := load(scenario)
	if err != nil {
		return 0, 0, false
	}
	samples := target
	if len(samples) < 5 {
		all, allErr := load("")
		if allErr != nil {
			return 0, 0, false
		}
		if len(all) > 0 {
			samples = all
		}
	}
	if len(samples) == 0 {
		return 0, 0, false
	}
	ramValues := make([]int64, 0, len(samples))
	cpuValues := make([]int64, 0, len(samples))
	for _, s := range samples {
		ramValues = append(ramValues, s.peak)
		value := s.cpu * 1000 / s.wall
		if value < 100 {
			value = 100
		}
		cpuValues = append(cpuValues, value)
	}
	sort.Slice(ramValues, func(i, j int) bool { return ramValues[i] < ramValues[j] })
	sort.Slice(cpuValues, func(i, j int) bool { return cpuValues[i] < cpuValues[j] })
	return ramValues[stats.NearestRankIndex(len(ramValues), 0.9)], cpuValues[stats.NearestRankIndex(len(cpuValues), 0.9)], true
}

// PhaseDurationEstimate returns the observed p90 wall-clock duration for the
// scheduler's deadline check.
//
// The p90 rather than the median is intentional: the guard protects the
// ordinary slow tail against contention, while the percentile prevents one
// pathological run from pinning a phase to serial forever. A phase that runs
// in 10 seconds nineteen times and 140 seconds once should not be excluded
// from every otherwise-safe batch solely because of that outlier.
//
// It is deliberately not the planner's estimate. The planner's number arrives
// through the request as `EstimatedDurationSeconds`, is rounded to whole
// seconds, and is biased upward by a sample pool the budget-planner work has
// not yet corrected — measured 2026-08-08, it predicted 56 s for a `contracts`
// phase that ran in 23.9 s. Sizing a *deadline* guard from an estimate that
// overshoots by 2.3x serializes phases that are nowhere near their timeout,
// which is the opposite of what the guard is for.
//
// Unlike PhaseCostEstimate this deliberately does not filter on reliability.
// Reliability qualifies the CPU and RSS readings, which are sampled from
// process-wide rusage; the duration is timed by Test Genie around the call and
// is exact regardless. Failing phases are included because they are the slow
// ones — 2.2x the passing average — and a guard that only saw passing runs
// would understate the risk it exists to catch.
func (r *SuiteExecutionRepository) PhaseDurationEstimate(ctx context.Context, scenario, phase string) (int64, bool) {
	load := func(scope string) ([]int64, error) {
		q := `SELECT COALESCE(p.wall_clock_ms, p.duration_ms)
			FROM suite_execution_phases p JOIN suite_executions e ON e.id = p.execution_id
			WHERE p.phase_name = ? AND e.completed_at >= ? AND e.completed_at < ?
			AND p.cache_hit = 0 AND COALESCE(p.wall_clock_ms, p.duration_ms) > 0`
		args := []any{
			strings.TrimSpace(phase),
			time.Now().UTC().Add(-deadlineHistoryWindow).Format(time.RFC3339Nano),
			time.Now().UTC().Add(time.Second).Format(time.RFC3339Nano),
		}
		if strings.TrimSpace(scope) != "" {
			q += " AND e.scenario_name = ?"
			args = append(args, strings.TrimSpace(scope))
		}
		rows, err := r.db.QueryContext(ctx, q, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []int64
		for rows.Next() {
			var ms int64
			if err := rows.Scan(&ms); err != nil {
				return nil, err
			}
			out = append(out, ms)
		}
		return out, rows.Err()
	}
	// Scenario-scoped history first: the same phase costs very different
	// amounts on different scenarios (`security` measured a 54 s median on
	// prompt-manager against 137 s on browser-automation-studio), so a fleet
	// blend would both over- and under-state the deadline risk. The fleet pool
	// is a fallback for a phase this scenario has barely run.
	samples, err := load(scenario)
	if err != nil {
		return 0, false
	}
	if len(samples) < 5 {
		all, allErr := load("")
		if allErr != nil {
			return 0, false
		}
		if len(all) > 0 {
			samples = all
		}
	}
	if len(samples) == 0 {
		return 0, false
	}
	if len(samples) < deadlineMinSamples {
		return 0, false
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	// Use the same nearest-rank ceiling as PhaseCostEstimate so the scheduler
	// and cost report agree about what p90 means, especially for small samples.
	index := stats.NearestRankIndex(len(samples), phaseDurationRiskPercentile)
	return samples[index], true
}

// HasPersistedMetrics reports whether a terminal run has a metrics rollup for
// the phase. It is intentionally separate from CostReport: conformance needs a
// durable adoption signal even when resource measurements are unavailable.
func (r *SuiteExecutionRepository) HasPersistedMetrics(ctx context.Context, phase string) (bool, error) {
	const q = `
SELECT EXISTS(
  SELECT 1
  FROM suite_execution_phases p
  JOIN suite_executions e ON e.id = p.execution_id
  WHERE p.phase_name = ? AND p.metrics_present = 1 AND e.terminal_outcome IS NOT NULL
)`
	var present bool
	if err := r.db.QueryRowContext(ctx, q, strings.TrimSpace(phase)).Scan(&present); err != nil {
		return false, err
	}
	return present, nil
}

func (r *SuiteExecutionRepository) CostReport(ctx context.Context, scenario string, since, until time.Time) ([]CostSummary, error) {
	q := `
		SELECT e.scenario_name, p.phase_name, p.status,
		       COALESCE(p.wall_clock_ms, p.duration_ms),
		       p.cpu_user_ms, p.peak_rss_bytes, p.cpu_reliability, p.memory_reliability,
		       p.measurement_scope,
		       p.cache_hit, p.cache_audit, p.cache_audit_mismatch, p.cache_no_saving,
		       p.predicted_duration_ms,
		       e.requested_at, e.started_at
FROM suite_execution_phases p
JOIN suite_executions e ON e.id = p.execution_id
WHERE e.completed_at >= ? AND e.completed_at < ?`
	args := []any{since.UTC().Format(time.RFC3339Nano), until.UTC().Format(time.RFC3339Nano)}
	if strings.TrimSpace(scenario) != "" {
		q += " AND e.scenario_name = ?"
		args = append(args, strings.TrimSpace(scenario))
	}
	q += " ORDER BY e.scenario_name, p.phase_name, e.completed_at"
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type bucket struct {
		key string
		CostSummary
		reliableWall   []int64
		passingWall    []int64
		failingWall    []int64
		queueLatencies []int64
		// sawFirstFailure marks that one honest derivation of this phase's
		// failure has already been counted, so subsequent ones are repeats.
		sawFirstFailure bool
	}
	buckets := map[string]*bucket{}
	for rows.Next() {
		var s CostSample
		var cpuUser, peak sql.NullInt64
		var cpuRel, memRel sql.NullString
		var measurementScope sql.NullString
		var cacheHit, cacheAudit, cacheAuditMismatch, cacheNoSaving int
		var requestedAt, startedAt sql.NullString
		if err := rows.Scan(&s.Scenario, &s.Phase, &s.Status, &s.WallClockMs, &cpuUser, &peak, &cpuRel, &memRel, &measurementScope, &cacheHit, &cacheAudit, &cacheAuditMismatch, &cacheNoSaving, &s.PredictedWallClockMs, &requestedAt, &startedAt); err != nil {
			return nil, err
		}
		s.QueueLatencyMs = queueLatencyMs(requestedAt, startedAt)
		s.CacheHit = cacheHit != 0
		s.CacheAudit = cacheAudit != 0
		s.CacheAuditMismatch = cacheAuditMismatch != 0
		s.CacheNoSaving = cacheNoSaving != 0
		if cpuUser.Valid {
			s.CPUUserMs = cpuUser.Int64
		}
		if peak.Valid {
			s.PeakRSSBytes = peak.Int64
		}
		if cpuRel.Valid {
			s.CPUReliability = cpuRel.String
		}
		if memRel.Valid {
			s.MemoryReliability = memRel.String
		}
		if measurementScope.Valid {
			s.MeasurementScope = measurementScope.String
		}
		key := s.Scenario + "\x00" + s.Phase
		b := buckets[key]
		if b == nil {
			b = &bucket{key: key, CostSummary: CostSummary{Scenario: s.Scenario, Phase: s.Phase, MeasurementScopes: map[string]int{}}}
			buckets[key] = b
		}
		b.MeasurementScopes[s.MeasurementScope]++
		b.SampleCount++
		if s.CacheHit {
			b.CacheHitCount++
		} else {
			b.ExecutedSampleCount++
		}
		if s.CacheAudit {
			b.CacheAuditCount++
		}
		if s.CacheAuditMismatch {
			b.CacheAuditMismatchCount++
		}
		if s.CacheNoSaving {
			b.CacheNoSavingCount++
		}
		if s.CacheAudit {
			b.CacheAuditWallClockMs += maxInt64(0, s.WallClockMs)
		}
		if isPassingPhaseStatus(s.Status) {
			b.PassingSampleCount++
			b.passingWall = append(b.passingWall, maxInt64(0, s.WallClockMs))
		} else if isFailingPhaseStatus(s.Status) {
			b.FailingSampleCount++
			b.failingWall = append(b.failingWall, maxInt64(0, s.WallClockMs))
			// Repeat-failure cost: wall-clock spent EXECUTING a failure this
			// phase had already produced. The first execution in the window is
			// the honest derivation and is not counted; every later one repeats
			// work whose answer was already known. Cache hits are excluded
			// because they cost nothing to serve — they are the saving, not the
			// cost. Rows are read in completed_at order, so "first" is the
			// earliest in the window.
			if !s.CacheHit {
				if b.sawFirstFailure {
					b.RepeatFailureWallClockMs += maxInt64(0, s.WallClockMs)
					b.RepeatFailureSampleCount++
				}
				b.sawFirstFailure = true
			}
		}
		if s.QueueLatencyMs >= 0 {
			b.queueLatencies = append(b.queueLatencies, s.QueueLatencyMs)
		}
		// Prediction accuracy is a wall-clock comparison and does not depend on
		// optional CPU/memory metrics being reliable. Keep it visible even when
		// the resource-cost sample is excluded from resource totals.
		if s.PredictedWallClockMs.Valid && s.PredictedWallClockMs.Int64 >= 0 {
			actual := maxInt64(0, s.WallClockMs)
			errorMs := actual - s.PredictedWallClockMs.Int64
			b.PredictionSampleCount++
			b.PredictionErrorTotalMs += errorMs
			b.PredictionMeanAbsoluteErrorMs += maxInt64(errorMs, -errorMs)
		}
		if s.CPUReliability != "RELIABILITY_RELIABLE" || s.MemoryReliability != "RELIABILITY_RELIABLE" {
			b.ExcludedSampleCount++
			continue
		}
		b.ReliableSampleCount++
		b.TotalWallClockMs += maxInt64(0, s.WallClockMs)
		b.TotalCPUUserMs += maxInt64(0, s.CPUUserMs)
		if s.PeakRSSBytes > b.MaxPeakRSSBytes {
			b.MaxPeakRSSBytes = s.PeakRSSBytes
		}
		b.reliableWall = append(b.reliableWall, maxInt64(0, s.WallClockMs))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]CostSummary, 0, len(buckets))
	for _, b := range buckets {
		sort.Slice(b.reliableWall, func(i, j int) bool { return b.reliableWall[i] < b.reliableWall[j] })
		if n := len(b.reliableWall); n > 0 {
			b.MedianWallClockMs = b.reliableWall[(n-1)/2]
			b.P90WallClockMs = b.reliableWall[stats.NearestRankIndex(n, 0.9)]
		}
		if n := len(b.passingWall); n > 0 {
			sort.Slice(b.passingWall, func(i, j int) bool { return b.passingWall[i] < b.passingWall[j] })
			b.PassingMedianWallClockMs = b.passingWall[(n-1)/2]
			b.PassingP90WallClockMs = b.passingWall[stats.NearestRankIndex(n, 0.9)]
		}
		if n := len(b.failingWall); n > 0 {
			sort.Slice(b.failingWall, func(i, j int) bool { return b.failingWall[i] < b.failingWall[j] })
			b.FailingMedianWallClockMs = b.failingWall[(n-1)/2]
			b.FailingP90WallClockMs = b.failingWall[stats.NearestRankIndex(n, 0.9)]
		}
		if b.PredictionSampleCount > 0 {
			b.PredictionMeanAbsoluteErrorMs /= int64(b.PredictionSampleCount)
			predictedTotal := b.TotalWallClockMs - b.PredictionErrorTotalMs
			if predictedTotal > 0 {
				b.PredictionMeanAbsoluteErrorPercent = float64(b.PredictionMeanAbsoluteErrorMs) / float64(predictedTotal) * 100
			}
		}
		if b.SampleCount > 0 {
			b.CacheHitRatePercent = float64(b.CacheHitCount) / float64(b.SampleCount) * 100
		}
		// A hit has no measured provider duration. Estimate the avoided work from
		// the median executed sample for this phase, then subtract the measured
		// cost of audit executions. This is intentionally conservative: the
		// report never presents gross avoided work as net savings.
		if len(b.reliableWall) > 0 {
			b.EstimatedGrossSavedWallClockMs = int64(b.CacheHitCount) * b.MedianWallClockMs
		}
		b.EstimatedNetSavedWallClockMs = b.EstimatedGrossSavedWallClockMs - b.CacheAuditWallClockMs
		b.QueueLatencyMedianMs, b.QueueLatencyP90Ms = -1, -1
		if n := len(b.queueLatencies); n > 0 {
			sort.Slice(b.queueLatencies, func(i, j int) bool { return b.queueLatencies[i] < b.queueLatencies[j] })
			b.QueueLatencyMedianMs = b.queueLatencies[(n-1)/2]
			b.QueueLatencyP90Ms = b.queueLatencies[stats.NearestRankIndex(n, 0.9)]
		}
		b.reliableWall = nil
		b.passingWall = nil
		b.failingWall = nil
		b.queueLatencies = nil
		result = append(result, b.CostSummary)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Scenario == result[j].Scenario {
			return result[i].Phase < result[j].Phase
		}
		return result[i].Scenario < result[j].Scenario
	})
	return result, nil
}

func isPassingPhaseStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "passed", "pass", "success", "succeeded":
		return true
	default:
		return false
	}
}

func isFailingPhaseStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "failure", "errored", "error", "timeout", "timed_out", "aborted":
		return true
	default:
		return false
	}
}

// queueLatencyMs computes the wait between request and start, in milliseconds.
//
// It returns -1 when either timestamp is missing or unparseable, and when the
// arithmetic comes out negative. -1 means UNKNOWN and is propagated as such:
// substituting 0 would report measured immediacy for a run whose wait was never
// recorded, which is exactly the kind of confident-wrong number this plan set
// out to remove.
func queueLatencyMs(requestedAt, startedAt sql.NullString) int64 {
	if !requestedAt.Valid || !startedAt.Valid {
		return -1
	}
	requested, ok := parseCostTimestamp(requestedAt.String)
	if !ok {
		return -1
	}
	started, ok := parseCostTimestamp(startedAt.String)
	if !ok {
		return -1
	}
	delta := started.Sub(requested).Milliseconds()
	if delta < 0 {
		return -1
	}
	return delta
}

// costTimestampLayouts are the formats timestamps have been written in over the
// life of this table. Accepting all of them keeps historical rows readable
// instead of silently dropping them from the projection.
var costTimestampLayouts = []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.999999999Z07:00", "2006-01-02 15:04:05"}

func parseCostTimestamp(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	for _, layout := range costTimestampLayouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}

// FoldFleet merges per-scenario rows into one row per PHASE.
//
// A fleet-wide answer to "where does suite time go" is a per-phase question.
// Reported per scenario it is several hundred rows, which is why every such
// question was previously answered with hand-written SQL instead of a command.
//
// Counts and totals sum. Percentiles do NOT: a median of medians is not a
// median, so they are recomputed as a sample-count-weighted mean of the inputs
// and named as such. Peak RSS takes the maximum, because the fleet's peak is
// the largest single observation, not their sum.
func FoldFleet(rows []CostSummary) []CostSummary {
	byPhase := map[string]*CostSummary{}
	weights := map[string]int{}
	queueWeights := map[string]int{}
	order := []string{}

	for _, row := range rows {
		agg := byPhase[row.Phase]
		if agg == nil {
			agg = &CostSummary{Phase: row.Phase, Scenario: "*", ProviderScenario: row.ProviderScenario, QueueLatencyMedianMs: -1, QueueLatencyP90Ms: -1}
			byPhase[row.Phase] = agg
			order = append(order, row.Phase)
		}
		if agg.ProviderScenario == "" {
			agg.ProviderScenario = row.ProviderScenario
		}
		agg.SampleCount += row.SampleCount
		agg.PassingSampleCount += row.PassingSampleCount
		agg.FailingSampleCount += row.FailingSampleCount
		agg.ReliableSampleCount += row.ReliableSampleCount
		agg.ExcludedSampleCount += row.ExcludedSampleCount
		agg.ExecutedSampleCount += row.ExecutedSampleCount
		agg.TotalWallClockMs += row.TotalWallClockMs
		agg.TotalCPUUserMs += row.TotalCPUUserMs
		agg.ChangeWallClockMs += row.ChangeWallClockMs
		agg.CacheHitCount += row.CacheHitCount
		agg.CacheAuditCount += row.CacheAuditCount
		agg.CacheAuditMismatchCount += row.CacheAuditMismatchCount
		agg.CacheNoSavingCount += row.CacheNoSavingCount
		agg.CacheAuditWallClockMs += row.CacheAuditWallClockMs
		agg.EstimatedGrossSavedWallClockMs += row.EstimatedGrossSavedWallClockMs
		agg.EstimatedNetSavedWallClockMs += row.EstimatedNetSavedWallClockMs
		agg.RepeatFailureWallClockMs += row.RepeatFailureWallClockMs
		agg.RepeatFailureSampleCount += row.RepeatFailureSampleCount
		if row.MaxPeakRSSBytes > agg.MaxPeakRSSBytes {
			agg.MaxPeakRSSBytes = row.MaxPeakRSSBytes
		}
		if row.ReliableSampleCount > 0 {
			agg.MedianWallClockMs += row.MedianWallClockMs * int64(row.ReliableSampleCount)
			agg.P90WallClockMs += row.P90WallClockMs * int64(row.ReliableSampleCount)
			weights[row.Phase] += row.ReliableSampleCount
		}
		if row.QueueLatencyMedianMs >= 0 && row.SampleCount > 0 {
			if agg.QueueLatencyMedianMs < 0 {
				agg.QueueLatencyMedianMs, agg.QueueLatencyP90Ms = 0, 0
			}
			agg.QueueLatencyMedianMs += row.QueueLatencyMedianMs * int64(row.SampleCount)
			agg.QueueLatencyP90Ms += row.QueueLatencyP90Ms * int64(row.SampleCount)
			queueWeights[row.Phase] += row.SampleCount
		}
	}

	out := make([]CostSummary, 0, len(order))
	for _, phase := range order {
		agg := byPhase[phase]
		if w := weights[phase]; w > 0 {
			agg.MedianWallClockMs /= int64(w)
			agg.P90WallClockMs /= int64(w)
		}
		if w := queueWeights[phase]; w > 0 {
			agg.QueueLatencyMedianMs /= int64(w)
			agg.QueueLatencyP90Ms /= int64(w)
		}
		if agg.SampleCount > 0 {
			agg.CacheHitRatePercent = float64(agg.CacheHitCount) / float64(agg.SampleCount) * 100
		}
		out = append(out, *agg)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TotalWallClockMs > out[j].TotalWallClockMs })
	return out
}
