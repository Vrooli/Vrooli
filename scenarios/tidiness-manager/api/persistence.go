package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

// StoreLintTypeIssues persists parsed lint/type issues from light scans.
// It uses ON CONFLICT with the idx_issues_dedup partial unique index to avoid duplicates.
func (ts *TidinessStore) StoreLintTypeIssues(ctx context.Context, scenario string, issues []Issue) (int, error) {
	if len(issues) == 0 {
		return 0, nil
	}

	// Use a transaction for batch insert
	tx, err := ts.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Use ON CONFLICT with the partial unique index (idx_issues_dedup)
	// The index uses COALESCE to handle NULL line/column numbers
	query := `
		INSERT INTO issues (
			scenario, file_path, category, severity, title, description,
			line_number, column_number, resource_used, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'open')
		ON CONFLICT (scenario, file_path, category, COALESCE(line_number, 0), COALESCE(column_number, 0))
		WHERE status = 'open'
		DO UPDATE SET
			severity = EXCLUDED.severity,
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			updated_at = CURRENT_TIMESTAMP
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, issue := range issues {
		// Build title from rule or message
		title := issue.Message
		if len(title) > 100 {
			title = title[:97] + "..."
		}

		// Resource is the tool that produced the issue
		resource := "make " + issue.Category // e.g., "make lint" or "make type"

		// Convert 0 to nil for proper COALESCE handling
		var lineNum, colNum *int
		if issue.Line > 0 {
			lineNum = &issue.Line
		}
		if issue.Column > 0 {
			colNum = &issue.Column
		}

		_, err := stmt.ExecContext(ctx,
			scenario,
			issue.File,
			issue.Category,
			issue.Severity,
			title,
			issue.Message,
			lineNum,
			colNum,
			resource,
		)
		if err != nil {
			// Log but continue - don't fail entire batch on one issue
			continue
		}
		inserted++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return inserted, nil
}

// GetDetailedFileMetrics retrieves all file metrics for a scenario from the database
func (ts *TidinessStore) GetDetailedFileMetrics(ctx context.Context, scenario string) ([]DetailedFileMetrics, error) {
	query := `
		SELECT
			file_path, language, file_extension, line_count,
			todo_count, fixme_count, hack_count,
			import_count, function_count, code_lines, comment_lines,
			comment_to_code_ratio, has_test_file,
			complexity_avg, complexity_max, duplication_pct
		FROM file_metrics
		WHERE scenario = $1
	`

	rows, err := ts.db.QueryContext(ctx, query, scenario)
	if err != nil {
		return nil, fmt.Errorf("failed to query file metrics: %w", err)
	}
	defer rows.Close()

	var metrics []DetailedFileMetrics
	for rows.Next() {
		var m DetailedFileMetrics
		var lang, ext sql.NullString
		var complexityAvg sql.NullFloat64
		var complexityMax sql.NullInt64
		var duplicationPct sql.NullFloat64

		err := rows.Scan(
			&m.FilePath, &lang, &ext, &m.LineCount,
			&m.TodoCount, &m.FixmeCount, &m.HackCount,
			&m.ImportCount, &m.FunctionCount, &m.CodeLines, &m.CommentLines,
			&m.CommentRatio, &m.HasTestFile,
			&complexityAvg, &complexityMax, &duplicationPct,
		)
		if err != nil {
			continue // Skip rows with scan errors
		}

		if lang.Valid {
			m.Language = lang.String
		}
		if ext.Valid {
			m.FileExtension = ext.String
		}
		if complexityAvg.Valid {
			val := complexityAvg.Float64
			m.ComplexityAvg = &val
		}
		if complexityMax.Valid {
			val := int(complexityMax.Int64)
			m.ComplexityMax = &val
		}
		if duplicationPct.Valid {
			val := duplicationPct.Float64
			m.DuplicationPct = &val
		}

		metrics = append(metrics, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating file metrics: %w", err)
	}

	return metrics, nil
}

// StalenessInfo contains metadata about scan freshness
type StalenessInfo struct {
	LastScanAt       *string `json:"last_scan_at,omitempty"`
	IsStale          bool    `json:"is_stale"`
	ModifiedFiles    int     `json:"modified_files,omitempty"`
	StaleReason      string  `json:"stale_reason,omitempty"`
	RescanCommand    string  `json:"rescan_command,omitempty"`
}

// GetStalenessInfo checks if issues might be out-of-date for a scenario
// by comparing the last scan time against the scenario's file modification times
func (ts *TidinessStore) GetStalenessInfo(ctx context.Context, scenario, scenarioPath string) (*StalenessInfo, error) {
	info := &StalenessInfo{
		RescanCommand: fmt.Sprintf("tidiness-manager scan %s", scenario),
	}

	// Get last scan time from scan_history first
	var lastScan sql.NullTime
	err := ts.db.QueryRowContext(ctx, `
		SELECT MAX(created_at) FROM scan_history WHERE scenario = $1
	`, scenario).Scan(&lastScan)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get last scan time: %w", err)
	}

	// If scan_history has no records, try file_metrics.updated_at as fallback
	// (light scans write to file_metrics but not always to scan_history)
	if !lastScan.Valid {
		err = ts.db.QueryRowContext(ctx, `
			SELECT MAX(updated_at) FROM file_metrics WHERE scenario = $1
		`, scenario).Scan(&lastScan)
		if err != nil && err != sql.ErrNoRows {
			// Non-fatal
			lastScan.Valid = false
		}
	}

	if !lastScan.Valid {
		// No scans ever run (neither scan_history nor file_metrics have data)
		info.IsStale = true
		info.StaleReason = "no scans have been run yet"
		return info, nil
	}

	scanTime := lastScan.Time.Format("2006-01-02T15:04:05Z")
	info.LastScanAt = &scanTime

	// Check if any files have been modified since the last scan
	if scenarioPath != "" {
		modifiedCount := countModifiedFilesSince(scenarioPath, lastScan.Time)
		if modifiedCount > 0 {
			info.IsStale = true
			info.ModifiedFiles = modifiedCount
			info.StaleReason = fmt.Sprintf("%d file(s) modified since last scan", modifiedCount)
		}
	}

	return info, nil
}

// countModifiedFilesSince walks the scenario directory and counts source files
// that have been modified after the given time
func countModifiedFilesSince(scenarioPath string, since time.Time) int {
	count := 0
	extensions := map[string]bool{
		".go":  true,
		".ts":  true,
		".tsx": true,
		".js":  true,
		".jsx": true,
		".py":  true,
	}

	filepath.Walk(scenarioPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip directories we don't care about
		if info.IsDir() {
			name := info.Name()
			if name == "node_modules" || name == ".git" || name == "vendor" ||
				name == "dist" || name == "build" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Only check source files
		ext := filepath.Ext(path)
		if !extensions[ext] {
			return nil
		}

		// Check if modified after scan
		if info.ModTime().After(since) {
			count++
		}
		return nil
	})

	return count
}
