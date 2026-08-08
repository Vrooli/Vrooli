package execution

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"test-genie/internal/orchestrator/phases"

	"github.com/google/uuid"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

const insertPhaseHistorySQL = `
INSERT INTO suite_execution_phases (
	execution_id, ordinal, phase_name, status, duration_ms, predicted_duration_ms, duration_seconds, error_text,
	classification, remediation, runnability_verdict, runnability_reason,
	finding_source, metrics_present, findings_blockers, findings_errors,
	findings_warnings, findings_infos, findings_total, wall_clock_ms, cpu_user_ms,
	cpu_sys_ms, peak_rss_bytes, cpu_reliability, memory_reliability, gpu_reliability
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

// insertPhaseHistory persists the compact, normalized history projection. It
// intentionally excludes rich findings, metrics, logs, and presentation data:
// those fields have one durable owner in immutable run evidence.
func insertPhaseHistory(ctx context.Context, tx *sql.Tx, executionID uuid.UUID, results []phases.ExecutionResult) error {
	for ordinal, result := range results {
		name := strings.ToLower(strings.TrimSpace(result.Name))
		if name == "" {
			return fmt.Errorf("phase history %d has no name", ordinal)
		}
		summary := result.FindingsSummary
		if summary == nil {
			summary = &runspb.PhaseFindingsSummary{}
		}
		wallClock, cpuUser, cpuSys, peakRSS, cpuReliability, memoryReliability, gpuReliability := metricColumns(result.Metrics)
		durationMs := result.DurationMilliseconds
		if durationMs < 0 {
			durationMs = 0
		}
		if durationMs == 0 && result.DurationSeconds > 0 {
			durationMs = int64(result.DurationSeconds) * 1000
		}
		durationSeconds := int((durationMs + 999) / 1000)
		predictedMs := result.PredictedDurationMilliseconds
		if predictedMs < 0 {
			predictedMs = 0
		}
		var predicted any
		if predictedMs > 0 {
			predicted = predictedMs
		}
		if _, err := tx.ExecContext(ctx, insertPhaseHistorySQL,
			executionID.String(), ordinal, name, strings.ToLower(strings.TrimSpace(result.Status)), durationMs, predicted, durationSeconds,
			result.Error, result.Classification, result.Remediation, result.RunnabilityVerdict,
			result.RunnabilityReason, result.FindingSource, boolToInt(result.Metrics != nil), summary.GetBlockers(), summary.GetErrors(),
			summary.GetWarnings(), summary.GetInfos(), summary.GetTotal(), wallClock, cpuUser, cpuSys,
			peakRSS, cpuReliability, memoryReliability, gpuReliability,
		); err != nil {
			return fmt.Errorf("insert compact phase history: %w", err)
		}
	}
	return nil
}

func metricColumns(metrics *commonv1.ExecutionMetrics) (any, any, any, any, any, any, any) {
	if metrics == nil {
		return nil, nil, nil, nil, nil, nil, nil
	}
	resources := metrics.GetResources()
	if resources == nil {
		return metrics.GetWallClockMs(), nil, nil, nil, nil, nil, nil
	}
	var cpuUser, cpuSys, peak any
	if resources.GetCpu() != commonv1.Reliability_RELIABILITY_UNAVAILABLE {
		cpuUser, cpuSys = resources.GetCpuUserMs(), resources.GetCpuSysMs()
	}
	if resources.GetMemory() != commonv1.Reliability_RELIABILITY_UNAVAILABLE {
		peak = resources.GetPeakRssBytes()
	}
	return metrics.GetWallClockMs(), cpuUser, cpuSys, peak, resources.GetCpu().String(), resources.GetMemory().String(), resources.GetGpu().String()
}

func (r *SuiteExecutionRepository) loadPhaseHistory(ctx context.Context, executionID uuid.UUID) ([]phases.ExecutionResult, error) {
	const q = `
	SELECT phase_name, status, duration_ms, predicted_duration_ms, duration_seconds, error_text, classification,
       remediation, runnability_verdict, runnability_reason, finding_source,
       metrics_present, findings_blockers, findings_errors, findings_warnings,
       findings_infos, findings_total
FROM suite_execution_phases
WHERE execution_id = ?
ORDER BY ordinal ASC`
	rows, err := r.db.QueryContext(ctx, q, executionID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := make([]phases.ExecutionResult, 0)
	for rows.Next() {
		var result phases.ExecutionResult
		var metricsPresent int
		var predictedDuration sql.NullInt64
		var summary runspb.PhaseFindingsSummary
		if err := rows.Scan(&result.Name, &result.Status, &result.DurationMilliseconds, &predictedDuration, &result.DurationSeconds, &result.Error,
			&result.Classification, &result.Remediation, &result.RunnabilityVerdict,
			&result.RunnabilityReason, &result.FindingSource, &metricsPresent,
			&summary.Blockers, &summary.Errors, &summary.Warnings, &summary.Infos, &summary.Total); err != nil {
			return nil, err
		}
		if predictedDuration.Valid {
			result.PredictedDurationMilliseconds = predictedDuration.Int64
		}
		// History intentionally records only metrics presence, never the rich
		// provider metrics payload. The read model does not recreate a marker:
		// callers that need the payload use immutable run evidence.
		_ = metricsPresent
		if summary.GetTotal() != 0 || summary.GetBlockers() != 0 || summary.GetErrors() != 0 || summary.GetWarnings() != 0 || summary.GetInfos() != 0 {
			result.FindingsSummary = &summary
		}
		results = append(results, result)
	}
	return results, rows.Err()
}
