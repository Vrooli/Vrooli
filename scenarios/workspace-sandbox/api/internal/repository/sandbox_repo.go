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
			id, scope_path, project_root, owner, owner_type, status,
			driver, driver_version, tags, metadata, idempotency_key, version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NULLIF($11, ''), $12)
		RETURNING created_at, last_used_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		s.ID, s.ScopePath, s.ProjectRoot, s.Owner, s.OwnerType, s.Status,
		s.Driver, s.DriverVersion, pq.Array(s.Tags), metadataJSON, s.IdempotencyKey, s.Version,
	).Scan(&s.CreatedAt, &s.LastUsedAt, &s.UpdatedAt)
}

// Get retrieves a sandbox by ID.
func (r *SandboxRepository) Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	query := `
		SELECT id, scope_path, project_root, owner, owner_type, status, error_message,
			created_at, last_used_at, stopped_at, approved_at, deleted_at,
			driver, driver_version, lower_dir, upper_dir, work_dir, merged_dir,
			size_bytes, file_count, active_pids, session_count, tags, metadata,
			COALESCE(idempotency_key, ''), updated_at, version
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
		&s.IdempotencyKey, &s.UpdatedAt, &s.Version,
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
		SELECT id, scope_path, project_root, owner, owner_type, status, error_message,
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

		err := rows.Scan(
			&s.ID, &s.ScopePath, &s.ProjectRoot, &s.Owner, &s.OwnerType, &s.Status, &s.ErrorMsg,
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
		SELECT id, scope_path, project_root, owner, owner_type, status, error_message,
			created_at, last_used_at, stopped_at, approved_at, deleted_at,
			driver, driver_version, lower_dir, upper_dir, work_dir, merged_dir,
			size_bytes, file_count, active_pids, session_count, tags, metadata,
			COALESCE(idempotency_key, ''), updated_at, version
		FROM sandboxes
		WHERE idempotency_key = $1`

	s := &types.Sandbox{}
	var metadataJSON []byte
	var tags pq.StringArray
	var activePIDs pq.Int64Array

	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&s.ID, &s.ScopePath, &s.ProjectRoot, &s.Owner, &s.OwnerType, &s.Status, &s.ErrorMsg,
		&s.CreatedAt, &s.LastUsedAt, &s.StoppedAt, &s.ApprovedAt, &s.DeletedAt,
		&s.Driver, &s.DriverVersion, &s.LowerDir, &s.UpperDir, &s.WorkDir, &s.MergedDir,
		&s.SizeBytes, &s.FileCount, &activePIDs, &s.SessionCount, &tags, &metadataJSON,
		&s.IdempotencyKey, &s.UpdatedAt, &s.Version,
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
			id, scope_path, project_root, owner, owner_type, status,
			driver, driver_version, tags, metadata, idempotency_key, version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NULLIF($11, ''), $12)
		RETURNING created_at, last_used_at, updated_at`

	return r.tx.QueryRowContext(ctx, query,
		s.ID, s.ScopePath, s.ProjectRoot, s.Owner, s.OwnerType, s.Status,
		s.Driver, s.DriverVersion, pq.Array(s.Tags), metadataJSON, s.IdempotencyKey, s.Version,
	).Scan(&s.CreatedAt, &s.LastUsedAt, &s.UpdatedAt)
}

// Get retrieves a sandbox by ID within the transaction.
func (r *TxSandboxRepository) Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	query := `
		SELECT id, scope_path, project_root, owner, owner_type, status, error_message,
			created_at, last_used_at, stopped_at, approved_at, deleted_at,
			driver, driver_version, lower_dir, upper_dir, work_dir, merged_dir,
			size_bytes, file_count, active_pids, session_count, tags, metadata,
			COALESCE(idempotency_key, ''), updated_at, version
		FROM sandboxes
		WHERE id = $1`

	s := &types.Sandbox{}
	var metadataJSON []byte
	var tags pq.StringArray
	var activePIDs pq.Int64Array

	err := r.tx.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.ScopePath, &s.ProjectRoot, &s.Owner, &s.OwnerType, &s.Status, &s.ErrorMsg,
		&s.CreatedAt, &s.LastUsedAt, &s.StoppedAt, &s.ApprovedAt, &s.DeletedAt,
		&s.Driver, &s.DriverVersion, &s.LowerDir, &s.UpperDir, &s.WorkDir, &s.MergedDir,
		&s.SizeBytes, &s.FileCount, &activePIDs, &s.SessionCount, &tags, &metadataJSON,
		&s.IdempotencyKey, &s.UpdatedAt, &s.Version,
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
		SELECT id, scope_path, status
		FROM sandboxes
		WHERE project_root = $1
		  AND status IN ('creating', 'active', 'stopped')
		  AND ($3::uuid IS NULL OR id != $3)
		FOR UPDATE`

	var excludeIDArg interface{}
	if excludeID != nil {
		excludeIDArg = *excludeID
	}

	rows, err := r.tx.QueryContext(ctx, query, projectRoot, scopePath, excludeIDArg)
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
		if conflictType != "" {
			conflicts = append(conflicts, types.PathConflict{
				ExistingID:    id.String(),
				ExistingScope: existingScope,
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

// FindByIdempotencyKey finds a sandbox by idempotency key within the transaction.
func (r *TxSandboxRepository) FindByIdempotencyKey(ctx context.Context, key string) (*types.Sandbox, error) {
	if key == "" {
		return nil, nil
	}

	query := `
		SELECT id, scope_path, project_root, owner, owner_type, status, error_message,
			created_at, last_used_at, stopped_at, approved_at, deleted_at,
			driver, driver_version, lower_dir, upper_dir, work_dir, merged_dir,
			size_bytes, file_count, active_pids, session_count, tags, metadata,
			COALESCE(idempotency_key, ''), updated_at, version
		FROM sandboxes
		WHERE idempotency_key = $1
		FOR UPDATE`

	s := &types.Sandbox{}
	var metadataJSON []byte
	var tags pq.StringArray
	var activePIDs pq.Int64Array

	err := r.tx.QueryRowContext(ctx, query, key).Scan(
		&s.ID, &s.ScopePath, &s.ProjectRoot, &s.Owner, &s.OwnerType, &s.Status, &s.ErrorMsg,
		&s.CreatedAt, &s.LastUsedAt, &s.StoppedAt, &s.ApprovedAt, &s.DeletedAt,
		&s.Driver, &s.DriverVersion, &s.LowerDir, &s.UpperDir, &s.WorkDir, &s.MergedDir,
		&s.SizeBytes, &s.FileCount, &activePIDs, &s.SessionCount, &tags, &metadataJSON,
		&s.IdempotencyKey, &s.UpdatedAt, &s.Version,
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
