package execution

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
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
	PredictedWallClockMs sql.NullInt64
	CacheHit             bool
	CacheAudit           bool
	CacheAuditMismatch   bool
	CacheNoSaving        bool
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
}

type CostSource interface {
	CostReport(context.Context, string, time.Time, time.Time) ([]CostSummary, error)
}

const calibrationInterval = 7 * 24 * time.Hour

// deadlineHistoryWindow bounds the history PhaseDurationEstimate reads. It is
// shorter than the 30-day sizing window because that estimate takes the slowest
// observation rather than a percentile, and the window is what keeps one
// pathological run from serializing a phase indefinitely.
const deadlineHistoryWindow = 14 * 24 * time.Hour

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
// A phase without a reliable sample is intentionally unknown and must run
// serially until history is available.
func (r *SuiteExecutionRepository) PhaseCostEstimate(ctx context.Context, scenario, phase string) (ramBytes, cpuMilli int64, reliable bool) {
	type sample struct{ wall, cpu, peak int64 }
	load := func(scope string) ([]sample, error) {
		q := `SELECT COALESCE(p.wall_clock_ms, p.duration_ms), p.cpu_user_ms, p.peak_rss_bytes
			FROM suite_execution_phases p JOIN suite_executions e ON e.id = p.execution_id
			WHERE p.phase_name = ? AND e.completed_at >= ? AND e.completed_at < ?
			AND p.cpu_reliability = 'RELIABILITY_RELIABLE' AND p.memory_reliability = 'RELIABILITY_RELIABLE'`
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
	index := func(n int) int { return (n*9+9)/10 - 1 }
	return ramValues[index(len(ramValues))], cpuValues[index(len(cpuValues))], true
}

// PhaseDurationEstimate returns the slowest wall-clock the phase has been
// observed to take recently, for the scheduler's deadline check.
//
// The slowest rather than the p90 PhaseCostEstimate uses for sizing, because
// the two answer different questions with different costs of being wrong.
// Sizing asks how much of the host to reserve, and over-reserving costs
// throughput, so the middle of the distribution is the right input. This guard
// asks whether contention could push a phase past its own timeout, and the two
// errors are not symmetric: erring high costs one phase its concurrency, while
// erring low costs a passing run a false timeout — a fabricated failure in the
// surface whose entire job is to be trusted. A phase that runs in 10 s
// nineteen times and 140 s on the twentieth is exactly what a timeout catches,
// and every percentile short of the maximum waves it into a batch.
//
// The remedy for the maximum's usual weakness — one pathological run pinning a
// phase to serial forever — is the window rather than the statistic. It is
// deliberately shorter than the 30-day pool PhaseCostEstimate draws on, so a
// fixed pathology ages out in a fortnight instead of a month.
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
	slowest := samples[0]
	for _, ms := range samples[1:] {
		if ms > slowest {
			slowest = ms
		}
	}
	return slowest, true
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
		       p.cache_hit, p.cache_audit, p.cache_audit_mismatch, p.cache_no_saving,
		       p.predicted_duration_ms
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
		reliableWall []int64
		passingWall  []int64
		failingWall  []int64
	}
	buckets := map[string]*bucket{}
	for rows.Next() {
		var s CostSample
		var cpuUser, peak sql.NullInt64
		var cpuRel, memRel sql.NullString
		var cacheHit, cacheAudit, cacheAuditMismatch, cacheNoSaving int
		if err := rows.Scan(&s.Scenario, &s.Phase, &s.Status, &s.WallClockMs, &cpuUser, &peak, &cpuRel, &memRel, &cacheHit, &cacheAudit, &cacheAuditMismatch, &cacheNoSaving, &s.PredictedWallClockMs); err != nil {
			return nil, err
		}
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
		key := s.Scenario + "\x00" + s.Phase
		b := buckets[key]
		if b == nil {
			b = &bucket{key: key, CostSummary: CostSummary{Scenario: s.Scenario, Phase: s.Phase}}
			buckets[key] = b
		}
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
			b.P90WallClockMs = b.reliableWall[(n*9+9)/10-1]
		}
		if n := len(b.passingWall); n > 0 {
			sort.Slice(b.passingWall, func(i, j int) bool { return b.passingWall[i] < b.passingWall[j] })
			b.PassingMedianWallClockMs = b.passingWall[(n-1)/2]
			b.PassingP90WallClockMs = b.passingWall[(n*9+9)/10-1]
		}
		if n := len(b.failingWall); n > 0 {
			sort.Slice(b.failingWall, func(i, j int) bool { return b.failingWall[i] < b.failingWall[j] })
			b.FailingMedianWallClockMs = b.failingWall[(n-1)/2]
			b.FailingP90WallClockMs = b.failingWall[(n*9+9)/10-1]
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
		b.reliableWall = nil
		b.passingWall = nil
		b.failingWall = nil
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
