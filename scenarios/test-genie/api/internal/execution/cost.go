package execution

import (
	"context"
	"database/sql"
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
}

type CostSource interface {
	CostReport(context.Context, string, time.Time, time.Time) ([]CostSummary, error)
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
		if err := rows.Scan(&s.Scenario, &s.Phase, &s.Status, &s.WallClockMs, &cpuUser, &peak, &cpuRel, &memRel, &s.PredictedWallClockMs); err != nil {
			return nil, err
		}
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
