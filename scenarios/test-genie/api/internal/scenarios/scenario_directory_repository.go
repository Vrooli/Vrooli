package scenarios

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"test-genie/internal/dbexec"
	"test-genie/internal/orchestrator"
	"test-genie/internal/storage/sqliteutil"

	"github.com/google/uuid"
)

// ScenarioSummary aggregates execution telemetry for a single scenario.
type ScenarioSummary struct {
	ScenarioName              string                              `json:"scenarioName"`
	ScenarioDescription       string                              `json:"scenarioDescription,omitempty"`
	ScenarioStatus            string                              `json:"scenarioStatus,omitempty"`
	ScenarioTags              []string                            `json:"scenarioTags,omitempty"`
	TotalExecutions           int                                 `json:"totalExecutions"`
	LastExecutionAt           *time.Time                          `json:"lastExecutionAt,omitempty"`
	LastExecutionID           *uuid.UUID                          `json:"lastExecutionId,omitempty"`
	LastExecutionPreset       string                              `json:"lastExecutionPreset,omitempty"`
	LastExecutionSuccess      *bool                               `json:"lastExecutionSuccess,omitempty"`
	LastExecutionPhases       []orchestrator.PhaseExecutionResult `json:"lastExecutionPhases,omitempty"`
	LastExecutionPhaseSummary *orchestrator.PhaseSummary          `json:"lastExecutionPhaseSummary,omitempty"`
	LastFailureAt             *time.Time                          `json:"lastFailureAt,omitempty"`
	Testing                   *TestingCapabilities                `json:"testing,omitempty"`
}

type executionSummary struct {
	TotalExecutions           int
	LastExecutionAt           *time.Time
	LastExecutionID           *uuid.UUID
	LastExecutionPreset       string
	LastExecutionSuccess      *bool
	LastExecutionPhases       []orchestrator.PhaseExecutionResult
	LastExecutionPhaseSummary *orchestrator.PhaseSummary
	LastFailureAt             *time.Time
}

// ScenarioDirectoryRepository loads scenario summaries from Test Genie's
// embedded SQLite database.
type ScenarioDirectoryRepository struct {
	db    dbexec.Executor
	clock func() time.Time
}

func NewScenarioDirectoryRepository(db dbexec.Executor) *ScenarioDirectoryRepository {
	return &ScenarioDirectoryRepository{
		db:    db,
		clock: time.Now,
	}
}

func (r *ScenarioDirectoryRepository) List(ctx context.Context) ([]ScenarioSummary, error) {
	names, err := r.loadScenarioNames(ctx)
	if err != nil {
		return nil, err
	}
	return r.buildSummaries(ctx, names)
}

func (r *ScenarioDirectoryRepository) Get(ctx context.Context, scenario string) (*ScenarioSummary, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, fmt.Errorf("scenario is required")
	}

	summaries, err := r.buildSummaries(ctx, []string{scenario})
	if err != nil {
		return nil, err
	}
	if len(summaries) == 0 {
		return nil, sql.ErrNoRows
	}
	return &summaries[0], nil
}

func (r *ScenarioDirectoryRepository) buildSummaries(ctx context.Context, names []string) ([]ScenarioSummary, error) {
	executionData, err := r.loadExecutionSummaries(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]ScenarioSummary, 0, len(names))
	for _, name := range names {
		executionSummary := executionData[name]
		if executionSummary == nil {
			continue
		}

		summary := ScenarioSummary{ScenarioName: name}
		if executionSummary != nil {
			summary.TotalExecutions = executionSummary.TotalExecutions
			summary.LastExecutionAt = executionSummary.LastExecutionAt
			summary.LastExecutionID = executionSummary.LastExecutionID
			summary.LastExecutionPreset = executionSummary.LastExecutionPreset
			summary.LastExecutionSuccess = executionSummary.LastExecutionSuccess
			summary.LastExecutionPhases = append([]orchestrator.PhaseExecutionResult(nil), executionSummary.LastExecutionPhases...)
			summary.LastExecutionPhaseSummary = executionSummary.LastExecutionPhaseSummary
			summary.LastFailureAt = executionSummary.LastFailureAt
		}
		summaries = append(summaries, summary)
	}

	sort.SliceStable(summaries, func(i, j int) bool {
		left := latestActivity(summaries[i])
		right := latestActivity(summaries[j])
		switch {
		case left == nil && right == nil:
			return summaries[i].ScenarioName < summaries[j].ScenarioName
		case left == nil:
			return false
		case right == nil:
			return true
		case left.Equal(*right):
			return summaries[i].ScenarioName < summaries[j].ScenarioName
		default:
			return left.After(*right)
		}
	})

	return summaries, nil
}

func (r *ScenarioDirectoryRepository) loadScenarioNames(ctx context.Context) ([]string, error) {
	const q = `
SELECT scenario_name FROM suite_executions
ORDER BY scenario_name ASC
`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return names, nil
}

func (r *ScenarioDirectoryRepository) loadExecutionSummaries(ctx context.Context) (map[string]*executionSummary, error) {
	const q = `
SELECT
	scenario_name,
	id,
	preset_used,
	success,
	phases,
	completed_at
FROM suite_executions
ORDER BY scenario_name ASC, completed_at DESC
`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*executionSummary)
	for rows.Next() {
		var (
			scenarioName string
			rawID        string
			preset       sql.NullString
			success      int
			phasesValue  any
			completedAt  any
		)
		if err := rows.Scan(
			&scenarioName,
			&rawID,
			&preset,
			&success,
			&phasesValue,
			&completedAt,
		); err != nil {
			return nil, err
		}

		entry := result[scenarioName]
		if entry == nil {
			entry = &executionSummary{}
			result[scenarioName] = entry
		}
		entry.TotalExecutions++

		completed, err := sqliteutil.ParseTimestamp(completedAt)
		if err != nil {
			return nil, err
		}
		succeeded := success == 1
		if !succeeded && entry.LastFailureAt == nil {
			entry.LastFailureAt = &completed
		}
		if entry.LastExecutionAt != nil {
			continue
		}

		entry.LastExecutionAt = &completed
		if parsedID, err := uuid.Parse(rawID); err == nil {
			entry.LastExecutionID = &parsedID
		}
		entry.LastExecutionPreset = preset.String
		entry.LastExecutionSuccess = &succeeded

		var phases []orchestrator.PhaseExecutionResult
		if err := sqliteutil.UnmarshalJSON(phasesValue, &phases); err != nil {
			return nil, err
		}
		entry.LastExecutionPhases = phases
		if len(phases) > 0 {
			summaryValue := orchestrator.SummarizePhases(phases)
			entry.LastExecutionPhaseSummary = &summaryValue
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func latestActivity(summary ScenarioSummary) *time.Time {
	return summary.LastExecutionAt
}
