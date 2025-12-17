// Package repository provides database operations for sandboxes.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"workspace-sandbox/internal/types"
)

// Repository defines the interface for sandbox persistence operations.
// This interface enables testing with mock implementations and decouples
// the service layer from the specific database implementation.
type Repository interface {
	Create(ctx context.Context, s *types.Sandbox) error
	Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error)
	Update(ctx context.Context, s *types.Sandbox) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error)
	CheckScopeOverlap(ctx context.Context, scopePath, projectRoot string, excludeID *uuid.UUID) ([]types.PathConflict, error)
	GetActiveSandboxes(ctx context.Context, projectRoot string) ([]*types.Sandbox, error)
	LogAuditEvent(ctx context.Context, event *types.AuditEvent) error
	GetStats(ctx context.Context) (*types.SandboxStats, error)
}

// Verify SandboxRepository implements Repository interface.
var _ Repository = (*SandboxRepository)(nil)

// SandboxRepository provides database operations for sandboxes.
type SandboxRepository struct {
	db *sql.DB
}

// NewSandboxRepository creates a new repository.
func NewSandboxRepository(db *sql.DB) *SandboxRepository {
	return &SandboxRepository{db: db}
}

// Create inserts a new sandbox record.
func (r *SandboxRepository) Create(ctx context.Context, s *types.Sandbox) error {
	metadataJSON, err := json.Marshal(s.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO sandboxes (
			id, scope_path, project_root, owner, owner_type, status,
			driver, driver_version, tags, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, last_used_at`

	return r.db.QueryRowContext(ctx, query,
		s.ID, s.ScopePath, s.ProjectRoot, s.Owner, s.OwnerType, s.Status,
		s.Driver, s.DriverVersion, pq.Array(s.Tags), metadataJSON,
	).Scan(&s.CreatedAt, &s.LastUsedAt)
}

// Get retrieves a sandbox by ID.
func (r *SandboxRepository) Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	query := `
		SELECT id, scope_path, project_root, owner, owner_type, status, error_message,
			created_at, last_used_at, stopped_at, approved_at, deleted_at,
			driver, driver_version, lower_dir, upper_dir, work_dir, merged_dir,
			size_bytes, file_count, active_pids, session_count, tags, metadata
		FROM sandboxes
		WHERE id = $1`

	s := &types.Sandbox{}
	var metadataJSON []byte
	var tags pq.StringArray
	var activePIDs pq.Int64Array

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.ScopePath, &s.ProjectRoot, &s.Owner, &s.OwnerType, &s.Status, &s.ErrorMsg,
		&s.CreatedAt, &s.LastUsedAt, &s.StoppedAt, &s.ApprovedAt, &s.DeletedAt,
		&s.Driver, &s.DriverVersion, &s.LowerDir, &s.UpperDir, &s.WorkDir, &s.MergedDir,
		&s.SizeBytes, &s.FileCount, &activePIDs, &s.SessionCount, &tags, &metadataJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sandbox: %w", err)
	}

	s.Tags = tags
	s.ActivePIDs = make([]int, len(activePIDs))
	for i, pid := range activePIDs {
		s.ActivePIDs[i] = int(pid)
	}
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &s.Metadata)
	}

	return s, nil
}

// Update updates a sandbox record.
func (r *SandboxRepository) Update(ctx context.Context, s *types.Sandbox) error {
	metadataJSON, err := json.Marshal(s.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	activePIDs := make(pq.Int64Array, len(s.ActivePIDs))
	for i, pid := range s.ActivePIDs {
		activePIDs[i] = int64(pid)
	}

	query := `
		UPDATE sandboxes SET
			status = $2, error_message = $3,
			stopped_at = $4, approved_at = $5, deleted_at = $6,
			lower_dir = $7, upper_dir = $8, work_dir = $9, merged_dir = $10,
			size_bytes = $11, file_count = $12, active_pids = $13, session_count = $14,
			tags = $15, metadata = $16
		WHERE id = $1`

	_, err = r.db.ExecContext(ctx, query,
		s.ID, s.Status, s.ErrorMsg,
		s.StoppedAt, s.ApprovedAt, s.DeletedAt,
		s.LowerDir, s.UpperDir, s.WorkDir, s.MergedDir,
		s.SizeBytes, s.FileCount, activePIDs, s.SessionCount,
		pq.Array(s.Tags), metadataJSON,
	)
	return err
}

// Delete marks a sandbox as deleted.
func (r *SandboxRepository) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE sandboxes
		SET status = 'deleted', deleted_at = $2
		WHERE id = $1 AND status != 'deleted'`

	result, err := r.db.ExecContext(ctx, query, id, now)
	if err != nil {
		return fmt.Errorf("failed to delete sandbox: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("sandbox not found or already deleted")
	}

	return nil
}

// List retrieves sandboxes matching the filter.
func (r *SandboxRepository) List(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	// Build WHERE conditions
	if len(filter.Status) > 0 {
		placeholders := make([]string, len(filter.Status))
		for i, s := range filter.Status {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, s)
			argNum++
		}
		conditions = append(conditions, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ",")))
	}

	if filter.Owner != "" {
		conditions = append(conditions, fmt.Sprintf("owner = $%d", argNum))
		args = append(args, filter.Owner)
		argNum++
	}

	if filter.ProjectRoot != "" {
		conditions = append(conditions, fmt.Sprintf("project_root = $%d", argNum))
		args = append(args, filter.ProjectRoot)
		argNum++
	}

	if filter.ScopePath != "" {
		conditions = append(conditions, fmt.Sprintf("scope_path = $%d", argNum))
		args = append(args, filter.ScopePath)
		argNum++
	}

	if !filter.CreatedFrom.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argNum))
		args = append(args, filter.CreatedFrom)
		argNum++
	}

	if !filter.CreatedTo.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argNum))
		args = append(args, filter.CreatedTo)
		argNum++
	}

	// Build query
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM sandboxes %s", whereClause)
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count sandboxes: %w", err)
	}

	// Apply pagination
	// Default limit is applied at the handler/service layer from config.
	// Here we just ensure reasonable bounds.
	limit := filter.Limit
	if limit <= 0 {
		limit = 100 // Fallback default; prefer config.Limits.DefaultListLimit
	}
	if limit > 10000 {
		limit = 10000 // Hard cap for safety
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT id, scope_path, project_root, owner, owner_type, status, error_message,
			created_at, last_used_at, stopped_at, approved_at, deleted_at,
			driver, driver_version, lower_dir, upper_dir, work_dir, merged_dir,
			size_bytes, file_count, active_pids, session_count, tags, metadata
		FROM sandboxes
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argNum, argNum+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list sandboxes: %w", err)
	}
	defer rows.Close()

	var sandboxes []*types.Sandbox
	for rows.Next() {
		s := &types.Sandbox{}
		var metadataJSON []byte
		var tags pq.StringArray
		var activePIDs pq.Int64Array

		err := rows.Scan(
			&s.ID, &s.ScopePath, &s.ProjectRoot, &s.Owner, &s.OwnerType, &s.Status, &s.ErrorMsg,
			&s.CreatedAt, &s.LastUsedAt, &s.StoppedAt, &s.ApprovedAt, &s.DeletedAt,
			&s.Driver, &s.DriverVersion, &s.LowerDir, &s.UpperDir, &s.WorkDir, &s.MergedDir,
			&s.SizeBytes, &s.FileCount, &activePIDs, &s.SessionCount, &tags, &metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sandbox: %w", err)
		}

		s.Tags = tags
		s.ActivePIDs = make([]int, len(activePIDs))
		for i, pid := range activePIDs {
			s.ActivePIDs[i] = int(pid)
		}
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &s.Metadata)
		}

		sandboxes = append(sandboxes, s)
	}

	return &types.ListResult{
		Sandboxes:  sandboxes,
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     offset,
	}, nil
}

// CheckScopeOverlap checks if a scope path overlaps with any active sandbox.
func (r *SandboxRepository) CheckScopeOverlap(ctx context.Context, scopePath, projectRoot string, excludeID *uuid.UUID) ([]types.PathConflict, error) {
	query := `SELECT id, scope_path, status FROM check_scope_overlap($1, $2, $3)`

	var excludeIDArg interface{}
	if excludeID != nil {
		excludeIDArg = *excludeID
	}

	rows, err := r.db.QueryContext(ctx, query, scopePath, projectRoot, excludeIDArg)
	if err != nil {
		return nil, fmt.Errorf("failed to check scope overlap: %w", err)
	}
	defer rows.Close()

	var conflicts []types.PathConflict
	for rows.Next() {
		var id uuid.UUID
		var existingScope string
		var status types.Status

		if err := rows.Scan(&id, &existingScope, &status); err != nil {
			return nil, fmt.Errorf("failed to scan conflict: %w", err)
		}

		conflictType := types.CheckPathOverlap(existingScope, scopePath)
		conflicts = append(conflicts, types.PathConflict{
			ExistingID:    id.String(),
			ExistingScope: existingScope,
			NewScope:      scopePath,
			ConflictType:  conflictType,
		})
	}

	return conflicts, nil
}

// GetActiveSandboxes returns all sandboxes in active states.
func (r *SandboxRepository) GetActiveSandboxes(ctx context.Context, projectRoot string) ([]*types.Sandbox, error) {
	filter := &types.ListFilter{
		Status:      []types.Status{types.StatusCreating, types.StatusActive},
		ProjectRoot: projectRoot,
		Limit:       10000,
	}

	result, err := r.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return result.Sandboxes, nil
}

// LogAuditEvent records a sandbox operation in the audit log.
func (r *SandboxRepository) LogAuditEvent(ctx context.Context, event *types.AuditEvent) error {
	detailsJSON, err := json.Marshal(event.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	stateJSON, err := json.Marshal(event.SandboxState)
	if err != nil {
		stateJSON = []byte("{}")
	}

	query := `
		INSERT INTO sandbox_audit_log (sandbox_id, event_type, actor, actor_type, details, sandbox_state)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err = r.db.ExecContext(ctx, query,
		event.SandboxID, event.EventType, event.Actor, event.ActorType,
		detailsJSON, stateJSON,
	)
	return err
}

// GetStats retrieves aggregate sandbox statistics using the database function.
func (r *SandboxRepository) GetStats(ctx context.Context) (*types.SandboxStats, error) {
	query := `SELECT * FROM get_sandbox_stats()`

	stats := &types.SandboxStats{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalCount,
		&stats.ActiveCount,
		&stats.StoppedCount,
		&stats.ErrorCount,
		&stats.ApprovedCount,
		&stats.RejectedCount,
		&stats.DeletedCount,
		&stats.TotalSizeBytes,
		&stats.AvgSizeBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get sandbox stats: %w", err)
	}

	return stats, nil
}
