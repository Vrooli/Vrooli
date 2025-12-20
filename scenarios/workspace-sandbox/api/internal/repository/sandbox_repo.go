// Package repository provides database operations for sandboxes.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"workspace-sandbox/internal/types"
)

func hydrateReservedFields(s *types.Sandbox, reservedPaths pq.StringArray) {
	// Prefer explicit reserved_paths; fall back to reserved_path; final fallback to scope_path.
	effective := make([]string, 0, maxInt(1, len(reservedPaths)))
	if len(reservedPaths) > 0 {
		for _, p := range reservedPaths {
			p = strings.TrimSpace(p)
			if p != "" {
				effective = append(effective, p)
			}
		}
	}
	if len(effective) == 0 {
		if strings.TrimSpace(s.ReservedPath) != "" {
			effective = append(effective, strings.TrimSpace(s.ReservedPath))
		} else if strings.TrimSpace(s.ScopePath) != "" {
			effective = append(effective, strings.TrimSpace(s.ScopePath))
		}
	}

	s.ReservedPaths = effective
	if len(effective) > 0 {
		s.ReservedPath = effective[0]
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Repository defines the interface for sandbox persistence operations.
// This interface enables testing with mock implementations and decouples
// the service layer from the specific database implementation.
//
// # Idempotency Support
//
// The repository provides FindByIdempotencyKey for checking if a sandbox
// was already created with a given idempotency key, enabling safe retries.
//
// # Optimistic Locking
//
// UpdateWithVersionCheck performs optimistic concurrency control by checking
// the version number before updating. This prevents lost updates when multiple
// callers modify the same sandbox concurrently.
type Repository interface {
	Create(ctx context.Context, s *types.Sandbox) error
	Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error)
	Update(ctx context.Context, s *types.Sandbox) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error)
	CheckScopeOverlap(ctx context.Context, scopePath, projectRoot string, excludeID *uuid.UUID) ([]types.PathConflict, error)
	GetActiveSandboxes(ctx context.Context, projectRoot string) ([]*types.Sandbox, error)
	LogAuditEvent(ctx context.Context, event *types.AuditEvent) error
	GetAuditLog(ctx context.Context, sandboxID *uuid.UUID, limit, offset int) ([]*types.AuditEvent, int, error)
	GetStats(ctx context.Context) (*types.SandboxStats, error)

	// --- Idempotency Support ---

	// FindByIdempotencyKey finds a sandbox by its idempotency key.
	// Returns nil, nil if no sandbox exists with that key.
	FindByIdempotencyKey(ctx context.Context, key string) (*types.Sandbox, error)

	// --- Optimistic Locking Support ---

	// UpdateWithVersionCheck updates a sandbox only if its version matches expected.
	// Returns ConcurrentModificationError if version mismatch occurs.
	// On success, increments the version and updates UpdatedAt.
	UpdateWithVersionCheck(ctx context.Context, s *types.Sandbox, expectedVersion int64) error

	// --- Transactional Support ---

	// BeginTx starts a new transaction and returns a Repository scoped to it.
	BeginTx(ctx context.Context) (TxRepository, error)

	// --- GC Support [OT-P1-003] ---

	// GetGCCandidates returns sandboxes eligible for garbage collection based on policy.
	GetGCCandidates(ctx context.Context, policy *types.GCPolicy, limit int) ([]*types.Sandbox, error)

	// --- Provenance Tracking Support ---

	// RecordAppliedChanges records file changes that were applied from a sandbox.
	RecordAppliedChanges(ctx context.Context, changes []*types.AppliedChange) error

	// GetPendingChanges returns pending (uncommitted) changes grouped by sandbox.
	GetPendingChanges(ctx context.Context, projectRoot string, limit, offset int) (*types.PendingChangesResult, error)

	// GetPendingChangeFiles returns all pending change records for the given project/sandboxes.
	GetPendingChangeFiles(ctx context.Context, projectRoot string, sandboxIDs []uuid.UUID) ([]*types.AppliedChange, error)

	// GetFileProvenance returns the history of changes for a specific file.
	GetFileProvenance(ctx context.Context, filePath, projectRoot string, limit int) ([]*types.AppliedChange, error)

	// MarkChangesCommitted updates applied_changes records with commit information.
	MarkChangesCommitted(ctx context.Context, ids []uuid.UUID, commitHash, commitMessage string) error
}

// TxRepository is a Repository bound to a transaction.
// It includes Commit and Rollback methods for transaction control.
type TxRepository interface {
	Repository
	Commit() error
	Rollback() error
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

	// Set initial version for new sandboxes
	s.Version = 1

	query := `
		INSERT INTO sandboxes (
			id, scope_path, reserved_path, reserved_paths, project_root, owner, owner_type, status,
			driver, driver_version, tags, metadata, idempotency_key, version, base_commit_hash
		) VALUES ($1, $2, NULLIF($3, ''), NULLIF($4::text[], ARRAY[]::text[]), $5, $6, $7, $8, $9, $10, $11, $12, NULLIF($13, ''), $14, NULLIF($15, ''))
		RETURNING created_at, last_used_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		s.ID, s.ScopePath, s.ReservedPath, pq.Array(s.ReservedPaths), s.ProjectRoot, s.Owner, s.OwnerType, s.Status,
		s.Driver, s.DriverVersion, pq.Array(s.Tags), metadataJSON, s.IdempotencyKey, s.Version, s.BaseCommitHash,
	).Scan(&s.CreatedAt, &s.LastUsedAt, &s.UpdatedAt)
}

// Get retrieves a sandbox by ID.
func (r *SandboxRepository) Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	query := `
		SELECT id, scope_path, COALESCE(reserved_path, scope_path), reserved_paths, project_root, owner, owner_type, status, error_message,
			created_at, last_used_at, stopped_at, approved_at, deleted_at,
			driver, driver_version, lower_dir, upper_dir, work_dir, merged_dir,
			size_bytes, file_count, active_pids, session_count, tags, metadata,
			COALESCE(idempotency_key, ''), updated_at, version, COALESCE(base_commit_hash, '')
		FROM sandboxes
		WHERE id = $1`

	s := &types.Sandbox{}
	var metadataJSON []byte
	var tags pq.StringArray
	var activePIDs pq.Int64Array
	var reservedPaths pq.StringArray

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.ScopePath, &s.ReservedPath, &reservedPaths, &s.ProjectRoot, &s.Owner, &s.OwnerType, &s.Status, &s.ErrorMsg,
		&s.CreatedAt, &s.LastUsedAt, &s.StoppedAt, &s.ApprovedAt, &s.DeletedAt,
		&s.Driver, &s.DriverVersion, &s.LowerDir, &s.UpperDir, &s.WorkDir, &s.MergedDir,
		&s.SizeBytes, &s.FileCount, &activePIDs, &s.SessionCount, &tags, &metadataJSON,
		&s.IdempotencyKey, &s.UpdatedAt, &s.Version, &s.BaseCommitHash,
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
	hydrateReservedFields(s, reservedPaths)

	return s, nil
}

// Update updates a sandbox record.
// This method increments the version automatically for optimistic locking support.
// For explicit version checking, use UpdateWithVersionCheck instead.
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
			tags = $15, metadata = $16,
			version = version + 1, updated_at = NOW()
		WHERE id = $1
		RETURNING version, updated_at`

	err = r.db.QueryRowContext(ctx, query,
		s.ID, s.Status, s.ErrorMsg,
		s.StoppedAt, s.ApprovedAt, s.DeletedAt,
		s.LowerDir, s.UpperDir, s.WorkDir, s.MergedDir,
		s.SizeBytes, s.FileCount, activePIDs, s.SessionCount,
		pq.Array(s.Tags), metadataJSON,
	).Scan(&s.Version, &s.UpdatedAt)
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

// queryBuilder helps construct parameterized SQL queries safely.
// All user-provided values are added as parameterized arguments, never interpolated.
type queryBuilder struct {
	conditions []string
	args       []interface{}
	argNum     int
}

func newQueryBuilder() *queryBuilder {
	return &queryBuilder{argNum: 1}
}

func (qb *queryBuilder) addCondition(column string, op string, value interface{}) {
	qb.conditions = append(qb.conditions, column+" "+op+" $"+strconv.Itoa(qb.argNum))
	qb.args = append(qb.args, value)
	qb.argNum++
}

func (qb *queryBuilder) addInCondition(column string, values []types.Status) {
	if len(values) == 0 {
		return
	}
	placeholders := make([]string, len(values))
	for i, v := range values {
		placeholders[i] = "$" + strconv.Itoa(qb.argNum)
		qb.args = append(qb.args, v)
		qb.argNum++
	}
	qb.conditions = append(qb.conditions, column+" IN ("+strings.Join(placeholders, ",")+")")
}

func (qb *queryBuilder) whereClause() string {
	if len(qb.conditions) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(qb.conditions, " AND ")
}

func (qb *queryBuilder) nextArgNum() int {
	return qb.argNum
}

// List retrieves sandboxes matching the filter.
// Uses queryBuilder to safely construct parameterized queries.
func (r *SandboxRepository) List(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error) {
	qb := newQueryBuilder()

	// Build WHERE conditions - all values are parameterized
	if len(filter.Status) > 0 {
		qb.addInCondition("status", filter.Status)
	}
	if filter.Owner != "" {
		qb.addCondition("owner", "=", filter.Owner)
	}
	if filter.ProjectRoot != "" {
		qb.addCondition("project_root", "=", filter.ProjectRoot)
	}
	if filter.ScopePath != "" {
		qb.addCondition("scope_path", "=", filter.ScopePath)
	}
	if !filter.CreatedFrom.IsZero() {
		qb.addCondition("created_at", ">=", filter.CreatedFrom)
	}
	if !filter.CreatedTo.IsZero() {
		qb.addCondition("created_at", "<=", filter.CreatedTo)
	}

	whereClause := qb.whereClause()
	args := qb.args

	// Get total count using parameterized query
	// The whereClause contains only column names and $N placeholders, never user data
	countQuery := "SELECT COUNT(*) FROM sandboxes " + whereClause
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count sandboxes: %w", err)
	}

	// Apply pagination with reasonable bounds
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

	// Build main query with parameterized LIMIT/OFFSET
	limitArg := qb.nextArgNum()
	offsetArg := limitArg + 1
	query := `
		SELECT id, scope_path, COALESCE(reserved_path, scope_path), reserved_paths, project_root, owner, owner_type, status, error_message,
			created_at, last_used_at, stopped_at, approved_at, deleted_at,
			driver, driver_version, lower_dir, upper_dir, work_dir, merged_dir,
			size_bytes, file_count, active_pids, session_count, tags, metadata,
			COALESCE(idempotency_key, ''), updated_at, version
		FROM sandboxes
		` + whereClause + `
		ORDER BY created_at DESC
		LIMIT $` + strconv.Itoa(limitArg) + ` OFFSET $` + strconv.Itoa(offsetArg)

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
		var reservedPaths pq.StringArray

		err := rows.Scan(
			&s.ID, &s.ScopePath, &s.ReservedPath, &reservedPaths, &s.ProjectRoot, &s.Owner, &s.OwnerType, &s.Status, &s.ErrorMsg,
			&s.CreatedAt, &s.LastUsedAt, &s.StoppedAt, &s.ApprovedAt, &s.DeletedAt,
			&s.Driver, &s.DriverVersion, &s.LowerDir, &s.UpperDir, &s.WorkDir, &s.MergedDir,
			&s.SizeBytes, &s.FileCount, &activePIDs, &s.SessionCount, &tags, &metadataJSON,
			&s.IdempotencyKey, &s.UpdatedAt, &s.Version,
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
		hydrateReservedFields(s, reservedPaths)

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

// GetAuditLog retrieves audit events, optionally filtered by sandbox ID.
// [OT-P1-004] Audit Trail Metadata
// Returns events in reverse chronological order (most recent first).
func (r *SandboxRepository) GetAuditLog(ctx context.Context, sandboxID *uuid.UUID, limit, offset int) ([]*types.AuditEvent, int, error) {
	// Set reasonable defaults
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	// Build query with optional sandbox filter
	var whereClause string
	var args []interface{}

	if sandboxID != nil {
		whereClause = "WHERE sandbox_id = $1"
		args = append(args, *sandboxID)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM sandbox_audit_log " + whereClause
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count audit events: %w", err)
	}

	// Build main query
	limitArgNum := len(args) + 1
	offsetArgNum := limitArgNum + 1

	query := `
		SELECT id, sandbox_id, event_type, event_time, actor, actor_type, details, sandbox_state
		FROM sandbox_audit_log
		` + whereClause + `
		ORDER BY event_time DESC
		LIMIT $` + strconv.Itoa(limitArgNum) + ` OFFSET $` + strconv.Itoa(offsetArgNum)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit log: %w", err)
	}
	defer rows.Close()

	var events []*types.AuditEvent
	for rows.Next() {
		event := &types.AuditEvent{}
		var detailsJSON, stateJSON []byte
		var sandboxID sql.NullString

		err := rows.Scan(
			&event.ID,
			&sandboxID,
			&event.EventType,
			&event.EventTime,
			&event.Actor,
			&event.ActorType,
			&detailsJSON,
			&stateJSON,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit event: %w", err)
		}

		// Parse sandbox ID
		if sandboxID.Valid {
			id, _ := uuid.Parse(sandboxID.String)
			event.SandboxID = &id
		}

		// Parse JSON fields
		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &event.Details)
		}
		if len(stateJSON) > 0 {
			json.Unmarshal(stateJSON, &event.SandboxState)
		}

		events = append(events, event)
	}

	return events, totalCount, nil
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

// --- Idempotency Support ---

// FindByIdempotencyKey finds a sandbox by its idempotency key.
// Returns nil, nil if no sandbox exists with that key.
// This enables idempotent create operations: if a sandbox was already created
// with the same key, return it instead of creating a duplicate.
func (r *SandboxRepository) FindByIdempotencyKey(ctx context.Context, key string) (*types.Sandbox, error) {
	if key == "" {
		return nil, nil // No idempotency key means no lookup
	}

	query := `
		SELECT id, scope_path, COALESCE(reserved_path, scope_path), reserved_paths, project_root, owner, owner_type, status, error_message,
			created_at, last_used_at, stopped_at, approved_at, deleted_at,
			driver, driver_version, lower_dir, upper_dir, work_dir, merged_dir,
			size_bytes, file_count, active_pids, session_count, tags, metadata,
			COALESCE(idempotency_key, ''), updated_at, version, COALESCE(base_commit_hash, '')
		FROM sandboxes
		WHERE idempotency_key = $1`

	s := &types.Sandbox{}
	var metadataJSON []byte
	var tags pq.StringArray
	var activePIDs pq.Int64Array
	var reservedPaths pq.StringArray

	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&s.ID, &s.ScopePath, &s.ReservedPath, &reservedPaths, &s.ProjectRoot, &s.Owner, &s.OwnerType, &s.Status, &s.ErrorMsg,
		&s.CreatedAt, &s.LastUsedAt, &s.StoppedAt, &s.ApprovedAt, &s.DeletedAt,
		&s.Driver, &s.DriverVersion, &s.LowerDir, &s.UpperDir, &s.WorkDir, &s.MergedDir,
		&s.SizeBytes, &s.FileCount, &activePIDs, &s.SessionCount, &tags, &metadataJSON,
		&s.IdempotencyKey, &s.UpdatedAt, &s.Version, &s.BaseCommitHash,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find sandbox by idempotency key: %w", err)
	}

	s.Tags = tags
	s.ActivePIDs = make([]int, len(activePIDs))
	for i, pid := range activePIDs {
		s.ActivePIDs[i] = int(pid)
	}
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &s.Metadata)
	}
	hydrateReservedFields(s, reservedPaths)

	return s, nil
}

// --- Optimistic Locking Support ---

// UpdateWithVersionCheck updates a sandbox only if its version matches expected.
// Returns ConcurrentModificationError if version mismatch occurs.
// On success, increments the version and updates UpdatedAt.
//
// This implements optimistic concurrency control: multiple callers can read
// a sandbox and attempt to update it, but only one succeeds. Others get
// an error and can retry after re-fetching.
func (r *SandboxRepository) UpdateWithVersionCheck(ctx context.Context, s *types.Sandbox, expectedVersion int64) error {
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
			tags = $15, metadata = $16,
			version = version + 1, updated_at = NOW()
		WHERE id = $1 AND version = $17
		RETURNING version, updated_at`

	err = r.db.QueryRowContext(ctx, query,
		s.ID, s.Status, s.ErrorMsg,
		s.StoppedAt, s.ApprovedAt, s.DeletedAt,
		s.LowerDir, s.UpperDir, s.WorkDir, s.MergedDir,
		s.SizeBytes, s.FileCount, activePIDs, s.SessionCount,
		pq.Array(s.Tags), metadataJSON, expectedVersion,
	).Scan(&s.Version, &s.UpdatedAt)

	if err == sql.ErrNoRows {
		// Version mismatch - fetch current version for error message
		var currentVersion int64
		verQuery := `SELECT version FROM sandboxes WHERE id = $1`
		if verErr := r.db.QueryRowContext(ctx, verQuery, s.ID).Scan(&currentVersion); verErr == nil {
			return types.NewConcurrentModificationError(s.ID.String(), expectedVersion, currentVersion)
		}
		return types.NewNotFoundError(s.ID.String())
	}

	return err
}

// --- Transactional Support ---

// BeginTx starts a new transaction and returns a Repository scoped to it.
// The returned TxRepository must be committed or rolled back by the caller.
func (r *SandboxRepository) BeginTx(ctx context.Context) (TxRepository, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &TxSandboxRepository{tx: tx}, nil
}

// TxSandboxRepository is a SandboxRepository bound to a transaction.
type TxSandboxRepository struct {
	tx *sql.Tx
}

// Verify TxSandboxRepository implements TxRepository.
var _ TxRepository = (*TxSandboxRepository)(nil)

// Commit commits the transaction.
func (r *TxSandboxRepository) Commit() error {
	return r.tx.Commit()
}

// Rollback aborts the transaction.
func (r *TxSandboxRepository) Rollback() error {
	return r.tx.Rollback()
}

// Create inserts a new sandbox record within the transaction.
func (r *TxSandboxRepository) Create(ctx context.Context, s *types.Sandbox) error {
	metadataJSON, err := json.Marshal(s.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	s.Version = 1

	query := `
		INSERT INTO sandboxes (
			id, scope_path, reserved_path, reserved_paths, project_root, owner, owner_type, status,
			driver, driver_version, tags, metadata, idempotency_key, version
		) VALUES ($1, $2, NULLIF($3, ''), NULLIF($4::text[], ARRAY[]::text[]), $5, $6, $7, $8, $9, $10, $11, $12, NULLIF($13, ''), $14)
		RETURNING created_at, last_used_at, updated_at`

	return r.tx.QueryRowContext(ctx, query,
		s.ID, s.ScopePath, s.ReservedPath, pq.Array(s.ReservedPaths), s.ProjectRoot, s.Owner, s.OwnerType, s.Status,
		s.Driver, s.DriverVersion, pq.Array(s.Tags), metadataJSON, s.IdempotencyKey, s.Version,
	).Scan(&s.CreatedAt, &s.LastUsedAt, &s.UpdatedAt)
}

// Get retrieves a sandbox by ID within the transaction.
func (r *TxSandboxRepository) Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	query := `
		SELECT id, scope_path, COALESCE(reserved_path, scope_path), reserved_paths, project_root, owner, owner_type, status, error_message,
			created_at, last_used_at, stopped_at, approved_at, deleted_at,
			driver, driver_version, lower_dir, upper_dir, work_dir, merged_dir,
			size_bytes, file_count, active_pids, session_count, tags, metadata,
			COALESCE(idempotency_key, ''), updated_at, version, COALESCE(base_commit_hash, '')
		FROM sandboxes
		WHERE id = $1`

	s := &types.Sandbox{}
	var metadataJSON []byte
	var tags pq.StringArray
	var activePIDs pq.Int64Array
	var reservedPaths pq.StringArray

	err := r.tx.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.ScopePath, &s.ReservedPath, &reservedPaths, &s.ProjectRoot, &s.Owner, &s.OwnerType, &s.Status, &s.ErrorMsg,
		&s.CreatedAt, &s.LastUsedAt, &s.StoppedAt, &s.ApprovedAt, &s.DeletedAt,
		&s.Driver, &s.DriverVersion, &s.LowerDir, &s.UpperDir, &s.WorkDir, &s.MergedDir,
		&s.SizeBytes, &s.FileCount, &activePIDs, &s.SessionCount, &tags, &metadataJSON,
		&s.IdempotencyKey, &s.UpdatedAt, &s.Version, &s.BaseCommitHash,
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
	hydrateReservedFields(s, reservedPaths)

	return s, nil
}

// Update updates a sandbox record within the transaction.
func (r *TxSandboxRepository) Update(ctx context.Context, s *types.Sandbox) error {
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
			tags = $15, metadata = $16,
			version = version + 1, updated_at = NOW()
		WHERE id = $1
		RETURNING version, updated_at`

	return r.tx.QueryRowContext(ctx, query,
		s.ID, s.Status, s.ErrorMsg,
		s.StoppedAt, s.ApprovedAt, s.DeletedAt,
		s.LowerDir, s.UpperDir, s.WorkDir, s.MergedDir,
		s.SizeBytes, s.FileCount, activePIDs, s.SessionCount,
		pq.Array(s.Tags), metadataJSON,
	).Scan(&s.Version, &s.UpdatedAt)
}

// Delete marks a sandbox as deleted within the transaction.
func (r *TxSandboxRepository) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE sandboxes
		SET status = 'deleted', deleted_at = $2, version = version + 1, updated_at = NOW()
		WHERE id = $1 AND status != 'deleted'`

	result, err := r.tx.ExecContext(ctx, query, id, now)
	if err != nil {
		return fmt.Errorf("failed to delete sandbox: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("sandbox not found or already deleted")
	}

	return nil
}

// List retrieves sandboxes matching the filter within the transaction.
// For brevity, this delegates to a shared implementation.
func (r *TxSandboxRepository) List(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error) {
	// For transactions, we use the same query logic but with tx instead of db
	// This is a simplified version - full implementation would need to be extracted
	return nil, fmt.Errorf("List not implemented for transactions - use non-transactional repository")
}

// CheckScopeOverlap checks for scope conflicts within the transaction.
// Uses FOR UPDATE to lock conflicting rows and prevent race conditions.
func (r *TxSandboxRepository) CheckScopeOverlap(ctx context.Context, scopePath, projectRoot string, excludeID *uuid.UUID) ([]types.PathConflict, error) {
	// Use FOR UPDATE to lock the rows and prevent concurrent inserts with overlapping scopes
	query := `
		SELECT id, scope_path, reserved_path, reserved_paths, status
		FROM sandboxes
		WHERE project_root = $1
		  AND status IN ('creating', 'active', 'stopped')
		  AND ($2::uuid IS NULL OR id != $2)
		FOR UPDATE`

	var excludeIDArg interface{}
	if excludeID != nil {
		excludeIDArg = *excludeID
	}

	rows, err := r.tx.QueryContext(ctx, query, projectRoot, excludeIDArg)
	if err != nil {
		return nil, fmt.Errorf("failed to check scope overlap: %w", err)
	}
	defer rows.Close()

	var conflicts []types.PathConflict
	for rows.Next() {
		var id uuid.UUID
		var existingScope string
		var existingReservedPath sql.NullString
		var existingReservedPaths pq.StringArray
		var status types.Status

		if err := rows.Scan(&id, &existingScope, &existingReservedPath, &existingReservedPaths, &status); err != nil {
			return nil, fmt.Errorf("failed to scan conflict: %w", err)
		}

		effective := make([]string, 0, maxInt(1, len(existingReservedPaths)))
		if len(existingReservedPaths) > 0 {
			for _, p := range existingReservedPaths {
				p = strings.TrimSpace(p)
				if p != "" {
					effective = append(effective, p)
				}
			}
		}
		if len(effective) == 0 {
			if existingReservedPath.Valid && strings.TrimSpace(existingReservedPath.String) != "" {
				effective = append(effective, strings.TrimSpace(existingReservedPath.String))
			} else if strings.TrimSpace(existingScope) != "" {
				effective = append(effective, strings.TrimSpace(existingScope))
			}
		}

		for _, existingPrefix := range effective {
			conflictType := types.CheckPathOverlap(existingPrefix, scopePath)
			if conflictType == "" {
				continue
			}
			conflicts = append(conflicts, types.PathConflict{
				ExistingID:    id.String(),
				ExistingScope: existingPrefix,
				NewScope:      scopePath,
				ConflictType:  conflictType,
			})
		}
	}

	return conflicts, nil
}

// GetActiveSandboxes returns active sandboxes within the transaction.
func (r *TxSandboxRepository) GetActiveSandboxes(ctx context.Context, projectRoot string) ([]*types.Sandbox, error) {
	return nil, fmt.Errorf("GetActiveSandboxes not implemented for transactions")
}

// LogAuditEvent records an audit event within the transaction.
func (r *TxSandboxRepository) LogAuditEvent(ctx context.Context, event *types.AuditEvent) error {
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

	_, err = r.tx.ExecContext(ctx, query,
		event.SandboxID, event.EventType, event.Actor, event.ActorType,
		detailsJSON, stateJSON,
	)
	return err
}

// GetStats is not supported within transactions.
func (r *TxSandboxRepository) GetStats(ctx context.Context) (*types.SandboxStats, error) {
	return nil, fmt.Errorf("GetStats not implemented for transactions")
}

// GetAuditLog is not supported within transactions.
func (r *TxSandboxRepository) GetAuditLog(ctx context.Context, sandboxID *uuid.UUID, limit, offset int) ([]*types.AuditEvent, int, error) {
	return nil, 0, fmt.Errorf("GetAuditLog not implemented for transactions")
}

// FindByIdempotencyKey finds a sandbox by idempotency key within the transaction.
func (r *TxSandboxRepository) FindByIdempotencyKey(ctx context.Context, key string) (*types.Sandbox, error) {
	if key == "" {
		return nil, nil
	}

	query := `
		SELECT id, scope_path, COALESCE(reserved_path, scope_path), reserved_paths, project_root, owner, owner_type, status, error_message,
			created_at, last_used_at, stopped_at, approved_at, deleted_at,
			driver, driver_version, lower_dir, upper_dir, work_dir, merged_dir,
			size_bytes, file_count, active_pids, session_count, tags, metadata,
			COALESCE(idempotency_key, ''), updated_at, version, COALESCE(base_commit_hash, '')
		FROM sandboxes
		WHERE idempotency_key = $1
		FOR UPDATE`

	s := &types.Sandbox{}
	var metadataJSON []byte
	var tags pq.StringArray
	var activePIDs pq.Int64Array
	var reservedPaths pq.StringArray

	err := r.tx.QueryRowContext(ctx, query, key).Scan(
		&s.ID, &s.ScopePath, &s.ReservedPath, &reservedPaths, &s.ProjectRoot, &s.Owner, &s.OwnerType, &s.Status, &s.ErrorMsg,
		&s.CreatedAt, &s.LastUsedAt, &s.StoppedAt, &s.ApprovedAt, &s.DeletedAt,
		&s.Driver, &s.DriverVersion, &s.LowerDir, &s.UpperDir, &s.WorkDir, &s.MergedDir,
		&s.SizeBytes, &s.FileCount, &activePIDs, &s.SessionCount, &tags, &metadataJSON,
		&s.IdempotencyKey, &s.UpdatedAt, &s.Version, &s.BaseCommitHash,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find sandbox by idempotency key: %w", err)
	}

	s.Tags = tags
	s.ActivePIDs = make([]int, len(activePIDs))
	for i, pid := range activePIDs {
		s.ActivePIDs[i] = int(pid)
	}
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &s.Metadata)
	}
	hydrateReservedFields(s, reservedPaths)

	return s, nil
}

// UpdateWithVersionCheck updates with version check within the transaction.
func (r *TxSandboxRepository) UpdateWithVersionCheck(ctx context.Context, s *types.Sandbox, expectedVersion int64) error {
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
			tags = $15, metadata = $16,
			version = version + 1, updated_at = NOW()
		WHERE id = $1 AND version = $17
		RETURNING version, updated_at`

	err = r.tx.QueryRowContext(ctx, query,
		s.ID, s.Status, s.ErrorMsg,
		s.StoppedAt, s.ApprovedAt, s.DeletedAt,
		s.LowerDir, s.UpperDir, s.WorkDir, s.MergedDir,
		s.SizeBytes, s.FileCount, activePIDs, s.SessionCount,
		pq.Array(s.Tags), metadataJSON, expectedVersion,
	).Scan(&s.Version, &s.UpdatedAt)

	if err == sql.ErrNoRows {
		var currentVersion int64
		verQuery := `SELECT version FROM sandboxes WHERE id = $1`
		if verErr := r.tx.QueryRowContext(ctx, verQuery, s.ID).Scan(&currentVersion); verErr == nil {
			return types.NewConcurrentModificationError(s.ID.String(), expectedVersion, currentVersion)
		}
		return types.NewNotFoundError(s.ID.String())
	}

	return err
}

// BeginTx is not supported within transactions (nested transactions not supported).
func (r *TxSandboxRepository) BeginTx(ctx context.Context) (TxRepository, error) {
	return nil, fmt.Errorf("nested transactions not supported")
}

// GetGCCandidates is not supported within transactions.
func (r *TxSandboxRepository) GetGCCandidates(ctx context.Context, policy *types.GCPolicy, limit int) ([]*types.Sandbox, error) {
	return nil, fmt.Errorf("GetGCCandidates not implemented for transactions")
}

// --- GC Support [OT-P1-003] ---

// GetGCCandidates returns sandboxes eligible for garbage collection based on policy.
// It builds a query that finds sandboxes matching any of the policy criteria.
// Results are ordered by created_at (oldest first) to prioritize old sandboxes.
func (r *SandboxRepository) GetGCCandidates(ctx context.Context, policy *types.GCPolicy, limit int) ([]*types.Sandbox, error) {
	if policy == nil {
		defaultPolicy := types.DefaultGCPolicy()
		policy = &defaultPolicy
	}

	qb := newQueryBuilder()
	now := time.Now()

	// Build status filter - never touch active or creating sandboxes
	statuses := policy.Statuses
	if len(statuses) == 0 {
		// Default: only collect stopped, error, and optionally terminal states
		statuses = []types.Status{types.StatusStopped, types.StatusError}
		if policy.IncludeTerminal {
			statuses = append(statuses, types.StatusApproved, types.StatusRejected)
		}
	}
	// Ensure we never accidentally collect active or creating sandboxes
	safeStatuses := make([]types.Status, 0, len(statuses))
	for _, s := range statuses {
		if s != types.StatusActive && s != types.StatusCreating {
			safeStatuses = append(safeStatuses, s)
		}
	}
	if len(safeStatuses) == 0 {
		// No valid statuses - return empty result
		return []*types.Sandbox{}, nil
	}
	qb.addInCondition("status", safeStatuses)

	// Build OR conditions for policy criteria
	var orConditions []string

	// MaxAge: sandboxes older than this threshold
	if policy.MaxAge > 0 {
		cutoff := now.Add(-policy.MaxAge)
		orConditions = append(orConditions, fmt.Sprintf("created_at < $%d", qb.nextArgNum()))
		qb.args = append(qb.args, cutoff)
		qb.argNum++
	}

	// IdleTimeout: sandboxes not used recently
	if policy.IdleTimeout > 0 {
		idleCutoff := now.Add(-policy.IdleTimeout)
		orConditions = append(orConditions, fmt.Sprintf("last_used_at < $%d", qb.nextArgNum()))
		qb.args = append(qb.args, idleCutoff)
		qb.argNum++
	}

	// TerminalDelay: approved/rejected sandboxes after delay
	if policy.IncludeTerminal && policy.TerminalDelay > 0 {
		terminalCutoff := now.Add(-policy.TerminalDelay)
		orConditions = append(orConditions,
			fmt.Sprintf("(status IN ('approved', 'rejected') AND (approved_at < $%d OR stopped_at < $%d))", qb.nextArgNum(), qb.nextArgNum()+1))
		qb.args = append(qb.args, terminalCutoff, terminalCutoff)
		qb.argNum += 2
	}

	// If no policy criteria specified, don't return any candidates
	if len(orConditions) == 0 {
		return []*types.Sandbox{}, nil
	}

	// Build the final query
	whereClause := qb.whereClause()
	if whereClause != "" {
		whereClause += " AND (" + strings.Join(orConditions, " OR ") + ")"
	} else {
		whereClause = "WHERE (" + strings.Join(orConditions, " OR ") + ")"
	}

	// Apply limit
	if limit <= 0 {
		limit = 1000 // Default limit
	}
	limitArg := qb.nextArgNum()

	query := `
		SELECT id, scope_path, COALESCE(reserved_path, scope_path), reserved_paths, project_root, owner, owner_type, status, error_message,
			created_at, last_used_at, stopped_at, approved_at, deleted_at,
			driver, driver_version, lower_dir, upper_dir, work_dir, merged_dir,
			size_bytes, file_count, active_pids, session_count, tags, metadata,
			COALESCE(idempotency_key, ''), updated_at, version
		FROM sandboxes
		` + whereClause + `
		ORDER BY created_at ASC
		LIMIT $` + strconv.Itoa(limitArg)

	qb.args = append(qb.args, limit)

	rows, err := r.db.QueryContext(ctx, query, qb.args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query GC candidates: %w", err)
	}
	defer rows.Close()

	var sandboxes []*types.Sandbox
	for rows.Next() {
		s := &types.Sandbox{}
		var metadataJSON []byte
		var tags pq.StringArray
		var activePIDs pq.Int64Array
		var reservedPaths pq.StringArray

		err := rows.Scan(
			&s.ID, &s.ScopePath, &s.ReservedPath, &reservedPaths, &s.ProjectRoot, &s.Owner, &s.OwnerType, &s.Status, &s.ErrorMsg,
			&s.CreatedAt, &s.LastUsedAt, &s.StoppedAt, &s.ApprovedAt, &s.DeletedAt,
			&s.Driver, &s.DriverVersion, &s.LowerDir, &s.UpperDir, &s.WorkDir, &s.MergedDir,
			&s.SizeBytes, &s.FileCount, &activePIDs, &s.SessionCount, &tags, &metadataJSON,
			&s.IdempotencyKey, &s.UpdatedAt, &s.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan GC candidate: %w", err)
		}

		s.Tags = tags
		s.ActivePIDs = make([]int, len(activePIDs))
		for i, pid := range activePIDs {
			s.ActivePIDs[i] = int(pid)
		}
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &s.Metadata)
		}
		hydrateReservedFields(s, reservedPaths)

		sandboxes = append(sandboxes, s)
	}

	return sandboxes, nil
}

// --- Provenance Tracking Support ---

// RecordAppliedChanges records file changes that were applied from a sandbox.
// This enables provenance tracking - knowing which sandbox modified which files.
func (r *SandboxRepository) RecordAppliedChanges(ctx context.Context, changes []*types.AppliedChange) error {
	if len(changes) == 0 {
		return nil
	}

	query := `
		INSERT INTO applied_changes (
			id, sandbox_id, sandbox_owner, sandbox_owner_type,
			file_path, project_root, change_type, file_size
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	for _, change := range changes {
		_, err := r.db.ExecContext(ctx, query,
			change.ID, change.SandboxID, change.SandboxOwner, change.SandboxOwnerType,
			change.FilePath, change.ProjectRoot, change.ChangeType, change.FileSize,
		)
		if err != nil {
			return fmt.Errorf("failed to record applied change for %s: %w", change.FilePath, err)
		}
	}

	return nil
}

// GetPendingChanges returns pending (uncommitted) changes grouped by sandbox.
func (r *SandboxRepository) GetPendingChanges(ctx context.Context, projectRoot string, limit, offset int) (*types.PendingChangesResult, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Build query based on whether projectRoot is provided
	var args []interface{}
	whereClause := "WHERE committed_at IS NULL"
	if projectRoot != "" {
		whereClause += " AND project_root = $1"
		args = append(args, projectRoot)
	}

	// Get total count of pending files
	countQuery := "SELECT COUNT(*) FROM applied_changes " + whereClause
	var totalFiles int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalFiles); err != nil {
		return nil, fmt.Errorf("failed to count pending changes: %w", err)
	}

	// Get grouped summaries
	argNum := len(args) + 1
	query := `
		SELECT sandbox_id, sandbox_owner, COUNT(*) as file_count, MAX(applied_at) as latest_applied
		FROM applied_changes
		` + whereClause + `
		GROUP BY sandbox_id, sandbox_owner
		ORDER BY latest_applied DESC
		LIMIT $` + strconv.Itoa(argNum) + ` OFFSET $` + strconv.Itoa(argNum+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending changes: %w", err)
	}
	defer rows.Close()

	var summaries []types.PendingChangesSummary
	for rows.Next() {
		var summary types.PendingChangesSummary
		err := rows.Scan(&summary.SandboxID, &summary.SandboxOwner, &summary.FileCount, &summary.LatestApplied)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pending change summary: %w", err)
		}
		summaries = append(summaries, summary)
	}

	return &types.PendingChangesResult{
		Summaries:  summaries,
		TotalFiles: totalFiles,
	}, nil
}

// GetPendingChangeFiles returns all pending change records for the given project/sandboxes.
func (r *SandboxRepository) GetPendingChangeFiles(ctx context.Context, projectRoot string, sandboxIDs []uuid.UUID) ([]*types.AppliedChange, error) {
	var args []interface{}
	whereClause := "WHERE committed_at IS NULL"
	argNum := 1

	if projectRoot != "" {
		whereClause += fmt.Sprintf(" AND project_root = $%d", argNum)
		args = append(args, projectRoot)
		argNum++
	}

	if len(sandboxIDs) > 0 {
		placeholders := make([]string, len(sandboxIDs))
		for i, id := range sandboxIDs {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, id)
			argNum++
		}
		whereClause += " AND sandbox_id IN (" + strings.Join(placeholders, ",") + ")"
	}

	query := `
		SELECT id, sandbox_id, sandbox_owner, sandbox_owner_type,
			   file_path, project_root, change_type, file_size, applied_at
		FROM applied_changes
		` + whereClause + `
		ORDER BY applied_at ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending change files: %w", err)
	}
	defer rows.Close()

	var changes []*types.AppliedChange
	for rows.Next() {
		change := &types.AppliedChange{}
		err := rows.Scan(
			&change.ID, &change.SandboxID, &change.SandboxOwner, &change.SandboxOwnerType,
			&change.FilePath, &change.ProjectRoot, &change.ChangeType, &change.FileSize, &change.AppliedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pending change file: %w", err)
		}
		changes = append(changes, change)
	}

	return changes, nil
}

// GetFileProvenance returns the history of changes for a specific file.
func (r *SandboxRepository) GetFileProvenance(ctx context.Context, filePath, projectRoot string, limit int) ([]*types.AppliedChange, error) {
	if limit <= 0 {
		limit = 50
	}

	var args []interface{}
	args = append(args, filePath)
	whereClause := "WHERE file_path = $1"
	argNum := 2

	if projectRoot != "" {
		whereClause += fmt.Sprintf(" AND project_root = $%d", argNum)
		args = append(args, projectRoot)
		argNum++
	}

	query := `
		SELECT id, sandbox_id, sandbox_owner, sandbox_owner_type,
			   file_path, project_root, change_type, file_size, applied_at,
			   committed_at, COALESCE(commit_hash, ''), COALESCE(commit_message, '')
		FROM applied_changes
		` + whereClause + `
		ORDER BY applied_at DESC
		LIMIT $` + strconv.Itoa(argNum)

	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query file provenance: %w", err)
	}
	defer rows.Close()

	var changes []*types.AppliedChange
	for rows.Next() {
		change := &types.AppliedChange{}
		err := rows.Scan(
			&change.ID, &change.SandboxID, &change.SandboxOwner, &change.SandboxOwnerType,
			&change.FilePath, &change.ProjectRoot, &change.ChangeType, &change.FileSize, &change.AppliedAt,
			&change.CommittedAt, &change.CommitHash, &change.CommitMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file provenance: %w", err)
		}
		changes = append(changes, change)
	}

	return changes, nil
}

// MarkChangesCommitted updates applied_changes records with commit information.
func (r *SandboxRepository) MarkChangesCommitted(ctx context.Context, ids []uuid.UUID, commitHash, commitMessage string) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, 0, len(ids)+3)
	args = append(args, time.Now(), commitHash, commitMessage)

	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+4)
		args = append(args, id)
	}

	query := `
		UPDATE applied_changes
		SET committed_at = $1, commit_hash = $2, commit_message = $3
		WHERE id IN (` + strings.Join(placeholders, ",") + `)`

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to mark changes committed: %w", err)
	}

	return nil
}

// --- Provenance Tracking for TxSandboxRepository (stubs) ---

// RecordAppliedChanges is not implemented for transactions.
func (r *TxSandboxRepository) RecordAppliedChanges(ctx context.Context, changes []*types.AppliedChange) error {
	return fmt.Errorf("RecordAppliedChanges not implemented for transactions")
}

// GetPendingChanges is not implemented for transactions.
func (r *TxSandboxRepository) GetPendingChanges(ctx context.Context, projectRoot string, limit, offset int) (*types.PendingChangesResult, error) {
	return nil, fmt.Errorf("GetPendingChanges not implemented for transactions")
}

// GetPendingChangeFiles is not implemented for transactions.
func (r *TxSandboxRepository) GetPendingChangeFiles(ctx context.Context, projectRoot string, sandboxIDs []uuid.UUID) ([]*types.AppliedChange, error) {
	return nil, fmt.Errorf("GetPendingChangeFiles not implemented for transactions")
}

// GetFileProvenance is not implemented for transactions.
func (r *TxSandboxRepository) GetFileProvenance(ctx context.Context, filePath, projectRoot string, limit int) ([]*types.AppliedChange, error) {
	return nil, fmt.Errorf("GetFileProvenance not implemented for transactions")
}

// MarkChangesCommitted is not implemented for transactions.
func (r *TxSandboxRepository) MarkChangesCommitted(ctx context.Context, ids []uuid.UUID, commitHash, commitMessage string) error {
	return fmt.Errorf("MarkChangesCommitted not implemented for transactions")
}
