package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// AuditLogger abstracts audit logging operations to enable testing.
// This is a seam for isolating database side effects.
//
// Production code uses SQLiteAuditLogger which writes to the database.
// Test code can use FakeAuditLogger to verify logging without database access.
//
// SEAM BOUNDARY: All audit logging must flow through this interface.
// [REQ:GCT-OT-P0-007] SQLite audit logging
type AuditLogger interface {
	// Log records an audit entry.
	// Returns error only for unexpected failures (not for graceful degradation).
	Log(ctx context.Context, entry AuditEntry) error

	// Query retrieves audit entries matching the request.
	Query(ctx context.Context, req AuditQueryRequest) (*AuditQueryResponse, error)

	// IsConfigured returns true if audit logging is available.
	IsConfigured() bool
}

// SQLiteAuditLogger implements AuditLogger using SQLite.
type SQLiteAuditLogger struct {
	db *sql.DB
}

// NewSQLiteAuditLogger creates a new SQLite audit logger.
// Returns nil if db is nil (graceful degradation).
func NewSQLiteAuditLogger(db *sql.DB) *SQLiteAuditLogger {
	if db == nil {
		return nil
	}
	return &SQLiteAuditLogger{db: db}
}

func (l *SQLiteAuditLogger) IsConfigured() bool {
	return l != nil && l.db != nil
}

func (l *SQLiteAuditLogger) Log(ctx context.Context, entry AuditEntry) error {
	if !l.IsConfigured() {
		return nil // Graceful degradation
	}

	// Ensure timestamp is set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	// Serialize paths and metadata to JSON
	var pathsJSON []byte
	if len(entry.Paths) > 0 {
		var err error
		pathsJSON, err = json.Marshal(entry.Paths)
		if err != nil {
			return fmt.Errorf("failed to marshal paths: %w", err)
		}
	}

	var metadataJSON []byte
	if entry.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(entry.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO git_audit_log (
			operation, repo_dir, branch, paths,
			commit_hash, commit_message, success, error_message,
			created_at, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := l.db.ExecContext(ctx, query,
		entry.Operation,
		entry.RepoDir,
		entry.Branch,
		nullableJSON(pathsJSON),
		entry.CommitHash,
		entry.CommitMessage,
		entry.Success,
		entry.Error,
		formatTimestamp(entry.Timestamp),
		nullableJSON(metadataJSON),
	)
	if err != nil {
		return fmt.Errorf("failed to insert audit entry: %w", err)
	}

	return nil
}

// buildAuditFilterQuery builds the WHERE clause and args for audit log queries.
func buildAuditFilterQuery(req AuditQueryRequest) (string, []interface{}) {
	query := `
		SELECT id, operation, repo_dir, branch, paths,
		       commit_hash, commit_message, success, error_message,
		       created_at, metadata
		FROM git_audit_log
		WHERE 1=1
	`
	var args []interface{}

	if req.Operation != "" {
		query += " AND operation = ?"
		args = append(args, req.Operation)
	}
	if req.Branch != "" {
		query += " AND branch = ?"
		args = append(args, req.Branch)
	}
	if !req.Since.IsZero() {
		query += " AND created_at >= ?"
		args = append(args, formatTimestamp(req.Since))
	}
	if !req.Until.IsZero() {
		query += " AND created_at <= ?"
		args = append(args, formatTimestamp(req.Until))
	}
	return query, args
}

// scanAuditEntry scans a single row into an AuditEntry.
func scanAuditEntry(rows *sql.Rows) (AuditEntry, error) {
	var entry AuditEntry
	var pathsJSON sql.NullString
	var metadataJSON sql.NullString
	var errorMessage sql.NullString
	var createdAt string

	err := rows.Scan(
		&entry.ID,
		&entry.Operation,
		&entry.RepoDir,
		&entry.Branch,
		&pathsJSON,
		&entry.CommitHash,
		&entry.CommitMessage,
		&entry.Success,
		&errorMessage,
		&createdAt,
		&metadataJSON,
	)
	if err != nil {
		return entry, fmt.Errorf("failed to scan audit entry: %w", err)
	}

	entry.Error = errorMessage.String
	entry.Timestamp = parseTimestamp(createdAt)

	if pathsJSON.Valid && strings.TrimSpace(pathsJSON.String) != "" {
		if err := json.Unmarshal([]byte(pathsJSON.String), &entry.Paths); err != nil {
			entry.Paths = nil
		}
	}

	if metadataJSON.Valid && strings.TrimSpace(metadataJSON.String) != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &entry.Metadata); err != nil {
			entry.Metadata = nil
		}
	}

	return entry, nil
}

func (l *SQLiteAuditLogger) Query(ctx context.Context, req AuditQueryRequest) (*AuditQueryResponse, error) {
	if !l.IsConfigured() {
		return &AuditQueryResponse{
			Entries:   []AuditEntry{},
			Timestamp: time.Now().UTC(),
		}, nil
	}

	query, args := buildAuditFilterQuery(req)

	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS filtered"
	var total int
	if err := l.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count audit entries: %w", err)
	}

	query += " ORDER BY created_at DESC"
	if req.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, req.Limit)
	}
	if req.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, req.Offset)
	}

	rows, err := l.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit entries: %w", err)
	}
	defer rows.Close()

	entries := []AuditEntry{}
	for rows.Next() {
		entry, err := scanAuditEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit entries: %w", err)
	}

	return &AuditQueryResponse{
		Entries:   entries,
		Total:     total,
		Timestamp: time.Now().UTC(),
	}, nil
}

// NoOpAuditLogger is an audit logger that does nothing.
// Used when audit logging is disabled or database is unavailable.
type NoOpAuditLogger struct{}

func (l *NoOpAuditLogger) IsConfigured() bool {
	return false
}

func (l *NoOpAuditLogger) Log(_ context.Context, _ AuditEntry) error {
	return nil
}

func (l *NoOpAuditLogger) Query(_ context.Context, _ AuditQueryRequest) (*AuditQueryResponse, error) {
	return &AuditQueryResponse{
		Entries:   []AuditEntry{},
		Timestamp: time.Now().UTC(),
	}, nil
}

func formatTimestamp(ts time.Time) string {
	return ts.UTC().Format(time.RFC3339Nano)
}

func parseTimestamp(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}

	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}

	return parsed
}

func nullableJSON(payload []byte) interface{} {
	if len(payload) == 0 {
		return nil
	}
	return string(payload)
}
