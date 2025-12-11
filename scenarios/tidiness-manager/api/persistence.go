package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrDuplicateIssue indicates an insert attempted to create a duplicate issue.
	ErrDuplicateIssue = errors.New("duplicate issue")
)

// IssueCounts aggregates open issue counts by category for a scenario.
type IssueCounts struct {
	Total     int
	Lint      int
	Type      int
	LongFiles int
}

// IssueStatusUpdate captures the fields returned when updating an issue status.
type IssueStatusUpdate struct {
	ID        int
	Status    string
	UpdatedAt string
}

// TidinessStore centralizes persistence concerns so handlers/orchestration stay focused on flow control.
type TidinessStore struct {
	db *sql.DB
}

// NewTidinessStore constructs a store backed by the provided database connection.
func NewTidinessStore(db *sql.DB) *TidinessStore {
	return &TidinessStore{db: db}
}

func (ts *TidinessStore) PersistDetailedFileMetrics(ctx context.Context, scenario string, metrics []DetailedFileMetrics) error {
	if len(metrics) == 0 {
		return nil
	}

	query := `
		INSERT INTO file_metrics (
			scenario, file_path, language, file_extension,
			line_count, todo_count, fixme_count, hack_count,
			import_count, function_count, code_lines, comment_lines,
			comment_to_code_ratio, has_test_file,
			complexity_avg, complexity_max, duplication_pct,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, CURRENT_TIMESTAMP)
		ON CONFLICT (scenario, file_path)
		DO UPDATE SET
			language = EXCLUDED.language,
			file_extension = EXCLUDED.file_extension,
			line_count = EXCLUDED.line_count,
			todo_count = EXCLUDED.todo_count,
			fixme_count = EXCLUDED.fixme_count,
			hack_count = EXCLUDED.hack_count,
			import_count = EXCLUDED.import_count,
			function_count = EXCLUDED.function_count,
			code_lines = EXCLUDED.code_lines,
			comment_lines = EXCLUDED.comment_lines,
			comment_to_code_ratio = EXCLUDED.comment_to_code_ratio,
			has_test_file = EXCLUDED.has_test_file,
			complexity_avg = EXCLUDED.complexity_avg,
			complexity_max = EXCLUDED.complexity_max,
			duplication_pct = EXCLUDED.duplication_pct,
			updated_at = CURRENT_TIMESTAMP
	`

	for _, metric := range metrics {
		_, err := ts.db.ExecContext(ctx, query,
			scenario,
			metric.FilePath,
			metric.Language,
			metric.FileExtension,
			metric.LineCount,
			metric.TodoCount,
			metric.FixmeCount,
			metric.HackCount,
			metric.ImportCount,
			metric.FunctionCount,
			metric.CodeLines,
			metric.CommentLines,
			metric.CommentRatio,
			metric.HasTestFile,
			metric.ComplexityAvg,
			metric.ComplexityMax,
			metric.DuplicationPct,
		)
		if err != nil {
			return fmt.Errorf("failed to persist detailed metrics for %s: %w", metric.FilePath, err)
		}
	}

	return nil
}

func (ts *TidinessStore) PersistFileMetrics(ctx context.Context, scenario string, metrics []FileMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	query := `
		INSERT INTO file_metrics (scenario, file_path, line_count, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (scenario, file_path)
		DO UPDATE SET
			line_count = EXCLUDED.line_count,
			updated_at = CURRENT_TIMESTAMP
	`

	for _, metric := range metrics {
		_, err := ts.db.ExecContext(ctx, query,
			scenario,
			metric.Path,
			metric.Lines,
		)
		if err != nil {
			return fmt.Errorf("failed to persist file metric for %s: %w", metric.Path, err)
		}
	}

	return nil
}

func (ts *TidinessStore) StoreAIIssue(ctx context.Context, scenario string, issue AIIssue, sessionID string, campaignID *int) error {
	query := `
		INSERT INTO issues (
			scenario, file_path, category, severity, title, description,
			line_number, column_number, agent_notes, remediation_steps,
			session_id, campaign_id, resource_used, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (scenario, file_path, category, line_number, column_number, created_at)
		DO UPDATE SET
			description = EXCLUDED.description,
			remediation_steps = EXCLUDED.remediation_steps,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := ts.db.ExecContext(ctx, query,
		scenario,
		issue.FilePath,
		issue.Category,
		issue.Severity,
		issue.Title,
		issue.Description,
		issue.LineNumber,
		issue.ColumnNumber,
		issue.AgentNotes,
		issue.RemediationSteps,
		sessionID,
		campaignID,
		"resource-claude-code",
		"open",
	)

	return err
}

func (ts *TidinessStore) RecordScanHistory(ctx context.Context, scenario, scanType string, result *SmartScanResult, campaignID *int) error {
	query := `
		INSERT INTO scan_history (
			scenario, scan_type, resource_used, issues_found,
			duration_seconds, campaign_id, session_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := ts.db.ExecContext(ctx, query,
		scenario,
		scanType,
		"resource-claude-code",
		result.IssuesFound,
		result.Duration.Seconds(),
		campaignID,
		result.SessionID,
	)

	return err
}

func (ts *TidinessStore) ListAgentIssues(ctx context.Context, req AgentIssuesRequest) ([]AgentIssue, error) {
	query := buildIssuesQuery(req)
	args := buildIssuesArgs(req)

	rows, err := ts.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	issues := []AgentIssue{}
	for rows.Next() {
		var issue AgentIssue
		var lineNum, colNum sql.NullInt64
		var agentNotes, remediation sql.NullString

		err := rows.Scan(
			&issue.ID,
			&issue.Scenario,
			&issue.FilePath,
			&issue.Category,
			&issue.Severity,
			&issue.Title,
			&issue.Description,
			&lineNum,
			&colNum,
			&agentNotes,
			&remediation,
			&issue.Status,
			&issue.CreatedAt,
		)
		if err != nil {
			continue
		}

		issue.LineNumber = assignNullInt(lineNum)
		issue.ColumnNumber = assignNullInt(colNum)
		issue.AgentNotes = assignNullString(agentNotes)
		issue.RemediationSteps = assignNullString(remediation)

		issues = append(issues, issue)
	}

	return issues, nil
}

type AgentIssuePayload struct {
	Scenario         string
	FilePath         string
	Category         string
	Severity         string
	Title            string
	Description      string
	LineNumber       *int
	ColumnNumber     *int
	AgentNotes       string
	RemediationSteps string
	CampaignID       *int
	SessionID        string
	ResourceUsed     string
}

func (ts *TidinessStore) InsertAgentIssue(ctx context.Context, payload AgentIssuePayload) (int, string, error) {
	query := `
		INSERT INTO issues (
			scenario, file_path, category, severity, title, description,
			line_number, column_number, agent_notes, remediation_steps,
			campaign_id, session_id, resource_used
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at
	`

	var id int
	var createdAt string
	err := ts.db.QueryRowContext(
		ctx,
		query,
		payload.Scenario, payload.FilePath, payload.Category, payload.Severity,
		payload.Title, payload.Description, payload.LineNumber, payload.ColumnNumber,
		payload.AgentNotes, payload.RemediationSteps, payload.CampaignID,
		payload.SessionID, payload.ResourceUsed,
	).Scan(&id, &createdAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return 0, "", ErrDuplicateIssue
		}
		return 0, "", err
	}

	return id, createdAt, nil
}

func (ts *TidinessStore) UpdateIssueStatus(ctx context.Context, id int, status, resolutionNotes string) (IssueStatusUpdate, error) {
	query := `
		UPDATE issues
		SET status = $1, resolution_notes = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
		RETURNING id, status, updated_at
	`

	var result IssueStatusUpdate
	err := ts.db.QueryRowContext(ctx, query, status, resolutionNotes, id).
		Scan(&result.ID, &result.Status, &result.UpdatedAt)

	return result, err
}

func (ts *TidinessStore) FetchIssueCounts(ctx context.Context) (map[string]IssueCounts, error) {
	query := `
		SELECT
			scenario,
			COUNT(*) as total_issues,
			COUNT(CASE WHEN category = 'lint' THEN 1 END) as lint_issues,
			COUNT(CASE WHEN category = 'type' THEN 1 END) as type_issues,
			COUNT(CASE WHEN category = 'length' THEN 1 END) as long_files
		FROM issues
		WHERE status = 'open'
		GROUP BY scenario
	`

	rows, err := ts.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	issueCounts := make(map[string]IssueCounts)
	for rows.Next() {
		var scenario string
		var counts IssueCounts
		if err := rows.Scan(&scenario, &counts.Total, &counts.Lint, &counts.Type, &counts.LongFiles); err != nil {
			continue
		}
		issueCounts[scenario] = counts
	}

	return issueCounts, nil
}
