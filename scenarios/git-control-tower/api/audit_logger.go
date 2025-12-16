package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// AuditLogger abstracts audit logging operations to enable testing.
// This is a seam for isolating database side effects.
//
// Production code uses PostgresAuditLogger which writes to the database.
// Test code can use FakeAuditLogger to verify logging without database access.
//
// SEAM BOUNDARY: All audit logging must flow through this interface.
// [REQ:GCT-OT-P0-007] PostgreSQL audit logging
type AuditLogger interface {
	// Log records an audit entry.
	// Returns error only for unexpected failures (not for graceful degradation).
	Log(ctx context.Context, entry AuditEntry) error

	// Query retrieves audit entries matching the request.
	Query(ctx context.Context, req AuditQueryRequest) (*AuditQueryResponse, error)

	// IsConfigured returns true if audit logging is available.
	IsConfigured() bool
}

// PostgresAuditLogger implements AuditLogger using PostgreSQL.
type PostgresAuditLogger struct {
	db *sql.DB
}

// NewPostgresAuditLogger creates a new PostgreSQL audit logger.
// Returns nil if db is nil (graceful degradation).
func NewPostgresAuditLogger(db *sql.DB) *PostgresAuditLogger {
	if db == nil {
		return nil
	}
	return &PostgresAuditLogger{db: db}
}

func (l *PostgresAuditLogger) IsConfigured() bool {
	return l != nil && l.db != nil
}

func (l *PostgresAuditLogger) Log(ctx context.Context, entry AuditEntry) error {
	if !l.IsConfigured() {
		return nil // Graceful degradation
	}

	// Ensure timestamp is set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	// Serialize metadata to JSON
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
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := l.db.ExecContext(ctx, query,
		entry.Operation,
		entry.RepoDir,
		entry.Branch,
		pq.Array(entry.Paths),
		entry.CommitHash,
		entry.CommitMessage,
		entry.Success,
		entry.Error,
		entry.Timestamp,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to insert audit entry: %w", err)
	}

	return nil
}

func (l *PostgresAuditLogger) Query(ctx context.Context, req AuditQueryRequest) (*AuditQueryResponse, error) {
	if !l.IsConfigured() {
		return &AuditQueryResponse{
			Entries:   []AuditEntry{},
			Timestamp: time.Now().UTC(),
		}, nil
	}

	// Build query with filters
	query := `
		SELECT id, operation, repo_dir, branch, paths,
		       commit_hash, commit_message, success, error_message,
		       created_at, metadata
		FROM git_audit_log
		WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if req.Operation != "" {
		query += fmt.Sprintf(" AND operation = $%d", argNum)
		args = append(args, req.Operation)
		argNum++
	}
	if req.Branch != "" {
		query += fmt.Sprintf(" AND branch = $%d", argNum)
		args = append(args, req.Branch)
		argNum++
	}
	if !req.Since.IsZero() {
		query += fmt.Sprintf(" AND created_at >= $%d", argNum)
		args = append(args, req.Since)
		argNum++
	}
	if !req.Until.IsZero() {
		query += fmt.Sprintf(" AND created_at <= $%d", argNum)
		args = append(args, req.Until)
		argNum++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS filtered"
	var total int
	if err := l.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count audit entries: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC"
	if req.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, req.Limit)
		argNum++
	}
	if req.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argNum)
		args = append(args, req.Offset)
	}

	rows, err := l.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit entries: %w", err)
	}
	defer rows.Close()

	entries := []AuditEntry{}
	for rows.Next() {
		var entry AuditEntry
		var paths pq.StringArray
		var metadataJSON []byte

		err := rows.Scan(
			&entry.ID,
			&entry.Operation,
			&entry.RepoDir,
			&entry.Branch,
			&paths,
			&entry.CommitHash,
			&entry.CommitMessage,
			&entry.Success,
			&entry.Error,
			&entry.Timestamp,
			&metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit entry: %w", err)
		}

		entry.Paths = paths
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &entry.Metadata); err != nil {
				// Non-fatal: continue with nil metadata
				entry.Metadata = nil
			}
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
