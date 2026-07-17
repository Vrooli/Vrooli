package execution

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"test-genie/internal/orchestrator/phases"

	"github.com/google/uuid"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

const insertPhaseHistorySQL = `
INSERT INTO suite_execution_phases (
	execution_id, ordinal, phase_name, status, duration_seconds, error_text,
	classification, remediation, runnability_verdict, runnability_reason,
	finding_source, metrics_present, findings_blockers, findings_errors,
	findings_warnings, findings_infos, findings_total
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

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
		if _, err := tx.ExecContext(ctx, insertPhaseHistorySQL,
			executionID.String(), ordinal, name, strings.ToLower(strings.TrimSpace(result.Status)), maxInt(0, result.DurationSeconds),
			result.Error, result.Classification, result.Remediation, result.RunnabilityVerdict,
			result.RunnabilityReason, result.FindingSource, boolToInt(result.Metrics != nil),
			summary.GetBlockers(), summary.GetErrors(), summary.GetWarnings(), summary.GetInfos(), summary.GetTotal(),
		); err != nil {
			return fmt.Errorf("insert compact phase history: %w", err)
		}
	}
	return nil
}

func (r *SuiteExecutionRepository) loadPhaseHistory(ctx context.Context, executionID uuid.UUID) ([]phases.ExecutionResult, error) {
	const q = `
SELECT phase_name, status, duration_seconds, error_text, classification,
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
		var summary runspb.PhaseFindingsSummary
		if err := rows.Scan(&result.Name, &result.Status, &result.DurationSeconds, &result.Error,
			&result.Classification, &result.Remediation, &result.RunnabilityVerdict,
			&result.RunnabilityReason, &result.FindingSource, &metricsPresent,
			&summary.Blockers, &summary.Errors, &summary.Warnings, &summary.Infos, &summary.Total); err != nil {
			return nil, err
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
