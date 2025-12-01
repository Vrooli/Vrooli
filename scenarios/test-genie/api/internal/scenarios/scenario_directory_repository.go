package scenarios

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	pq "github.com/lib/pq"

	"test-genie/internal/suite"
)

// ScenarioSummary aggregates queue + execution telemetry for a single scenario.
type ScenarioSummary struct {
	ScenarioName              string                 `json:"scenarioName"`
	ScenarioDescription       string                 `json:"scenarioDescription,omitempty"`
	ScenarioStatus            string                 `json:"scenarioStatus,omitempty"`
	ScenarioTags              []string               `json:"scenarioTags,omitempty"`
	PendingRequests           int                    `json:"pendingRequests"`
	TotalRequests             int                    `json:"totalRequests"`
	LastRequestAt             *time.Time             `json:"lastRequestAt,omitempty"`
	LastRequestPriority       string                 `json:"lastRequestPriority,omitempty"`
	LastRequestStatus         string                 `json:"lastRequestStatus,omitempty"`
	LastRequestNotes          string                 `json:"lastRequestNotes,omitempty"`
	LastRequestCoverageTarget *int                   `json:"lastRequestCoverageTarget,omitempty"`
	LastRequestTypes          []string               `json:"lastRequestTypes,omitempty"`
	TotalExecutions           int                    `json:"totalExecutions"`
	LastExecutionAt           *time.Time             `json:"lastExecutionAt,omitempty"`
	LastExecutionID           *uuid.UUID             `json:"lastExecutionId,omitempty"`
	LastExecutionPreset       string                 `json:"lastExecutionPreset,omitempty"`
	LastExecutionSuccess      *bool                  `json:"lastExecutionSuccess,omitempty"`
	LastExecutionPhases       []suite.PhaseExecutionResult `json:"lastExecutionPhases,omitempty"`
	LastExecutionPhaseSummary *suite.PhaseSummary          `json:"lastExecutionPhaseSummary,omitempty"`
	LastFailureAt             *time.Time             `json:"lastFailureAt,omitempty"`
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

// ScenarioDirectoryRepository loads scenario summaries from Postgres.
type ScenarioDirectoryRepository struct {
	db *sql.DB
}

func NewScenarioDirectoryRepository(db *sql.DB) *ScenarioDirectoryRepository {
	return &ScenarioDirectoryRepository{db: db}
}

func (r *ScenarioDirectoryRepository) List(ctx context.Context) ([]ScenarioSummary, error) {
	rows, err := r.db.QueryContext(ctx, scenarioSummaryQuery(false))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []ScenarioSummary
	for rows.Next() {
		summary, err := scanScenarioSummary(rows)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return summaries, nil
}

func (r *ScenarioDirectoryRepository) Get(ctx context.Context, scenario string) (*ScenarioSummary, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, fmt.Errorf("scenario is required")
	}
	row := r.db.QueryRowContext(ctx, scenarioSummaryQuery(true), scenario)
	summary, err := scanScenarioSummary(row)
	if err != nil {
		return nil, err
	}
	return &summary, nil
}

func scenarioSummaryQuery(filter bool) string {
	builder := strings.Builder{}
	builder.WriteString(`
WITH scenario_names AS (
	SELECT DISTINCT scenario_name FROM suite_requests
	UNION
	SELECT DISTINCT scenario_name FROM suite_executions
),
queue_stats AS (
	SELECT
		scenario_name,
		COUNT(*) AS total_requests,
		COUNT(*) FILTER (WHERE status IN ('queued', 'delegated')) AS pending_requests,
		MAX(updated_at) AS last_request_at
	FROM suite_requests
	GROUP BY scenario_name
),
queue_last AS (
	SELECT DISTINCT ON (scenario_name)
		scenario_name,
		priority,
		status,
		notes,
		coverage_target,
		requested_types,
		updated_at
	FROM suite_requests
	ORDER BY scenario_name, updated_at DESC
),
execution_stats AS (
	SELECT
		scenario_name,
		COUNT(*) AS total_executions,
		MAX(completed_at) AS last_execution_at
	FROM suite_executions
	GROUP BY scenario_name
),
execution_last AS (
	SELECT DISTINCT ON (scenario_name)
		scenario_name,
		id,
		preset_used,
		success,
		completed_at,
		phases
	FROM suite_executions
	ORDER BY scenario_name, completed_at DESC
),
failure_last AS (
	SELECT DISTINCT ON (scenario_name)
		scenario_name,
		completed_at
	FROM suite_executions
	WHERE success = false
	ORDER BY scenario_name, completed_at DESC
)
SELECT
	names.scenario_name,
	COALESCE(queue_stats.pending_requests, 0) AS pending_requests,
	COALESCE(queue_stats.total_requests, 0) AS total_requests,
	queue_stats.last_request_at,
	queue_last.priority,
	queue_last.status,
	queue_last.notes,
	queue_last.coverage_target,
	queue_last.requested_types,
	COALESCE(execution_stats.total_executions, 0) AS total_executions,
	execution_stats.last_execution_at,
	execution_last.id,
	execution_last.preset_used,
	execution_last.success,
	execution_last.phases,
	failure_last.completed_at
FROM scenario_names names
LEFT JOIN queue_stats ON queue_stats.scenario_name = names.scenario_name
LEFT JOIN queue_last ON queue_last.scenario_name = names.scenario_name
LEFT JOIN execution_stats ON execution_stats.scenario_name = names.scenario_name
LEFT JOIN execution_last ON execution_last.scenario_name = names.scenario_name
LEFT JOIN failure_last ON failure_last.scenario_name = names.scenario_name
`)
	if filter {
		builder.WriteString("WHERE names.scenario_name = $1\n")
	}
	builder.WriteString(`ORDER BY COALESCE(execution_stats.last_execution_at, queue_stats.last_request_at) DESC NULLS LAST, names.scenario_name ASC
`)
	return builder.String()
}

func scanScenarioSummary(scanner rowScanner) (ScenarioSummary, error) {
	var summary ScenarioSummary
	var lastRequestAt sql.NullTime
	var lastRequestPriority sql.NullString
	var lastRequestStatus sql.NullString
	var lastRequestNotes sql.NullString
	var coverageTarget sql.NullInt64
	var requestedTypes pq.StringArray
	var lastExecutionAt sql.NullTime
	var lastExecutionID sql.NullString
	var lastExecutionSuccess sql.NullBool
	var lastExecutionPreset sql.NullString
	var lastExecutionPhases []byte
	var lastFailureAt sql.NullTime

	if err := scanner.Scan(
		&summary.ScenarioName,
		&summary.PendingRequests,
		&summary.TotalRequests,
		&lastRequestAt,
		&lastRequestPriority,
		&lastRequestStatus,
		&lastRequestNotes,
		&coverageTarget,
		&requestedTypes,
		&summary.TotalExecutions,
		&lastExecutionAt,
		&lastExecutionID,
		&lastExecutionPreset,
		&lastExecutionSuccess,
		&lastExecutionPhases,
		&lastFailureAt,
	); err != nil {
		return summary, err
	}

	if lastRequestAt.Valid {
		t := lastRequestAt.Time.UTC()
		summary.LastRequestAt = &t
	}
	if lastRequestPriority.Valid {
		summary.LastRequestPriority = lastRequestPriority.String
	}
	if lastRequestStatus.Valid {
		summary.LastRequestStatus = lastRequestStatus.String
	}
	if lastRequestNotes.Valid {
		summary.LastRequestNotes = lastRequestNotes.String
	}
	if coverageTarget.Valid {
		val := int(coverageTarget.Int64)
		summary.LastRequestCoverageTarget = &val
	}
	if len(requestedTypes) > 0 {
		summary.LastRequestTypes = append(summary.LastRequestTypes, requestedTypes...)
	}
	if lastExecutionAt.Valid {
		t := lastExecutionAt.Time.UTC()
		summary.LastExecutionAt = &t
	}
	if lastExecutionID.Valid && lastExecutionID.String != "" {
		if id, err := uuid.Parse(lastExecutionID.String); err == nil {
			summary.LastExecutionID = &id
		}
	}
	if lastExecutionPreset.Valid {
		summary.LastExecutionPreset = lastExecutionPreset.String
	}
	if lastExecutionSuccess.Valid {
		val := lastExecutionSuccess.Bool
		summary.LastExecutionSuccess = &val
	}
	if len(lastExecutionPhases) > 0 {
		var phases []suite.PhaseExecutionResult
		if err := json.Unmarshal(lastExecutionPhases, &phases); err == nil {
			summary.LastExecutionPhases = phases
			if len(phases) > 0 {
				summaryValue := suite.SummarizePhases(phases)
				summary.LastExecutionPhaseSummary = &summaryValue
			}
		}
	}
	if lastFailureAt.Valid {
		t := lastFailureAt.Time.UTC()
		summary.LastFailureAt = &t
	}

	return summary, nil
}
