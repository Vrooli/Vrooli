package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// CheckpointRepository Implementation
// ============================================================================

type checkpointRepository struct {
	db  *DB
	log *logrus.Logger
}

var _ repository.CheckpointRepository = (*checkpointRepository)(nil)

// checkpointRow is the database row representation for run_checkpoints.
type checkpointRow struct {
	RunID             uuid.UUID    `db:"run_id"`
	Phase             string       `db:"phase"`
	StepWithinPhase   int          `db:"step_within_phase"`
	SandboxID         NullableUUID `db:"sandbox_id"`
	WorkDir           string       `db:"work_dir"`
	LockID            NullableUUID `db:"lock_id"`
	LastEventSequence int64        `db:"last_event_sequence"`
	LastHeartbeat     SQLiteTime   `db:"last_heartbeat"`
	RetryCount        int          `db:"retry_count"`
	SavedAt           SQLiteTime   `db:"saved_at"`
	Metadata          MetadataMap  `db:"metadata"`
}

func (c *checkpointRow) toDomain() *domain.RunCheckpoint {
	return &domain.RunCheckpoint{
		RunID:             c.RunID,
		Phase:             domain.RunPhase(c.Phase),
		StepWithinPhase:   c.StepWithinPhase,
		SandboxID:         c.SandboxID.ToPtr(),
		WorkDir:           c.WorkDir,
		LockID:            c.LockID.ToPtr(),
		LastEventSequence: c.LastEventSequence,
		LastHeartbeat:     c.LastHeartbeat.Time(),
		RetryCount:        c.RetryCount,
		SavedAt:           c.SavedAt.Time(),
		Metadata:          c.Metadata,
	}
}

func checkpointFromDomain(c *domain.RunCheckpoint) *checkpointRow {
	return &checkpointRow{
		RunID:             c.RunID,
		Phase:             string(c.Phase),
		StepWithinPhase:   c.StepWithinPhase,
		SandboxID:         NewNullableUUID(c.SandboxID),
		WorkDir:           c.WorkDir,
		LockID:            NewNullableUUID(c.LockID),
		LastEventSequence: c.LastEventSequence,
		LastHeartbeat:     SQLiteTime(c.LastHeartbeat),
		RetryCount:        c.RetryCount,
		SavedAt:           SQLiteTime(c.SavedAt),
		Metadata:          c.Metadata,
	}
}

const checkpointColumns = `run_id, phase, step_within_phase, sandbox_id, work_dir,
	lock_id, last_event_sequence, last_heartbeat, retry_count, saved_at, metadata`

func (r *checkpointRepository) Save(ctx context.Context, checkpoint *domain.RunCheckpoint) error {
	checkpoint.SavedAt = time.Now()
	row := checkpointFromDomain(checkpoint)

	// Upsert: insert or update on conflict
	query := `INSERT INTO run_checkpoints (run_id, phase, step_within_phase, sandbox_id, work_dir,
		lock_id, last_event_sequence, last_heartbeat, retry_count, saved_at, metadata)
		VALUES (:run_id, :phase, :step_within_phase, :sandbox_id, :work_dir,
		:lock_id, :last_event_sequence, :last_heartbeat, :retry_count, :saved_at, :metadata)
		ON CONFLICT (run_id) DO UPDATE SET
		phase = EXCLUDED.phase, step_within_phase = EXCLUDED.step_within_phase,
		sandbox_id = EXCLUDED.sandbox_id, work_dir = EXCLUDED.work_dir,
		lock_id = EXCLUDED.lock_id, last_event_sequence = EXCLUDED.last_event_sequence,
		last_heartbeat = EXCLUDED.last_heartbeat, retry_count = EXCLUDED.retry_count,
		saved_at = EXCLUDED.saved_at, metadata = EXCLUDED.metadata`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}
	return nil
}

func (r *checkpointRepository) Get(ctx context.Context, runID uuid.UUID) (*domain.RunCheckpoint, error) {
	query := r.db.Rebind(fmt.Sprintf("SELECT %s FROM run_checkpoints WHERE run_id = ?", checkpointColumns))
	var row checkpointRow
	if err := r.db.GetContext(ctx, &row, query, runID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get checkpoint: %w", err)
	}
	return row.toDomain(), nil
}

func (r *checkpointRepository) Delete(ctx context.Context, runID uuid.UUID) error {
	query := r.db.Rebind(`DELETE FROM run_checkpoints WHERE run_id = ?`)
	_, err := r.db.ExecContext(ctx, query, runID)
	if err != nil {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}
	return nil
}

func (r *checkpointRepository) ListStale(ctx context.Context, olderThan time.Duration) ([]*domain.RunCheckpoint, error) {
	cutoff := time.Now().Add(-olderThan)
	query := r.db.Rebind(fmt.Sprintf("SELECT %s FROM run_checkpoints WHERE last_heartbeat < ?", checkpointColumns))

	var rows []checkpointRow
	if err := r.db.SelectContext(ctx, &rows, query, cutoff); err != nil {
		return nil, fmt.Errorf("failed to list stale checkpoints: %w", err)
	}

	result := make([]*domain.RunCheckpoint, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

func (r *checkpointRepository) Heartbeat(ctx context.Context, runID uuid.UUID) error {
	query := r.db.Rebind(`UPDATE run_checkpoints SET last_heartbeat = ? WHERE run_id = ?`)
	_, err := r.db.ExecContext(ctx, query, time.Now(), runID)
	if err != nil {
		return fmt.Errorf("failed to update checkpoint heartbeat: %w", err)
	}
	return nil
}

// ============================================================================
// IdempotencyRepository Implementation
// ============================================================================

type idempotencyRepository struct {
	db  *DB
	log *logrus.Logger
}

var _ repository.IdempotencyRepository = (*idempotencyRepository)(nil)

// idempotencyRow is the database row representation for idempotency_records.
type idempotencyRow struct {
	Key        string         `db:"key"`
	Status     string         `db:"status"`
	EntityID   NullableUUID   `db:"entity_id"`
	EntityType sql.NullString `db:"entity_type"`
	CreatedAt  SQLiteTime     `db:"created_at"`
	ExpiresAt  SQLiteTime     `db:"expires_at"`
	Response   []byte         `db:"response"`
}

func (i *idempotencyRow) toDomain() *domain.IdempotencyRecord {
	entityType := ""
	if i.EntityType.Valid {
		entityType = i.EntityType.String
	}
	return &domain.IdempotencyRecord{
		Key:        i.Key,
		Status:     domain.IdempotencyStatus(i.Status),
		EntityID:   i.EntityID.ToPtr(),
		EntityType: entityType,
		CreatedAt:  i.CreatedAt.Time(),
		ExpiresAt:  i.ExpiresAt.Time(),
		Response:   i.Response,
	}
}

const idempotencyColumns = `key, status, entity_id, entity_type, created_at, expires_at, response`

func (r *idempotencyRepository) Check(ctx context.Context, key string) (*domain.IdempotencyRecord, error) {
	query := r.db.Rebind(fmt.Sprintf("SELECT %s FROM idempotency_records WHERE key = ? AND expires_at > ?", idempotencyColumns))
	var row idempotencyRow
	if err := r.db.GetContext(ctx, &row, query, key, time.Now()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to check idempotency: %w", err)
	}
	return row.toDomain(), nil
}

func (r *idempotencyRepository) Reserve(ctx context.Context, key string, ttl time.Duration) (*domain.IdempotencyRecord, error) {
	now := time.Now()
	record := &domain.IdempotencyRecord{
		Key:       key,
		Status:    domain.IdempotencyStatusPending,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}

	query := `INSERT INTO idempotency_records (key, status, created_at, expires_at)
		VALUES (:key, :status, :created_at, :expires_at)`

	row := struct {
		Key       string     `db:"key"`
		Status    string     `db:"status"`
		CreatedAt SQLiteTime `db:"created_at"`
		ExpiresAt SQLiteTime `db:"expires_at"`
	}{
		Key:       record.Key,
		Status:    string(record.Status),
		CreatedAt: SQLiteTime(record.CreatedAt),
		ExpiresAt: SQLiteTime(record.ExpiresAt),
	}

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		return nil, fmt.Errorf("failed to reserve idempotency key: %w", err)
	}
	return record, nil
}

func (r *idempotencyRepository) Complete(ctx context.Context, key string, entityID uuid.UUID, entityType string, response []byte) error {
	query := r.db.Rebind(`UPDATE idempotency_records SET status = ?, entity_id = ?, entity_type = ?, response = ? WHERE key = ?`)
	_, err := r.db.ExecContext(ctx, query, string(domain.IdempotencyStatusComplete), entityID, entityType, response, key)
	if err != nil {
		return fmt.Errorf("failed to complete idempotency: %w", err)
	}
	return nil
}

func (r *idempotencyRepository) Fail(ctx context.Context, key string) error {
	query := r.db.Rebind(`UPDATE idempotency_records SET status = ? WHERE key = ?`)
	_, err := r.db.ExecContext(ctx, query, string(domain.IdempotencyStatusFailed), key)
	if err != nil {
		return fmt.Errorf("failed to fail idempotency: %w", err)
	}
	return nil
}

func (r *idempotencyRepository) CleanupExpired(ctx context.Context) (int, error) {
	query := r.db.Rebind(`DELETE FROM idempotency_records WHERE expires_at < ?`)
	result, err := r.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired idempotency records: %w", err)
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

// ============================================================================
// PolicyRepository Implementation
// ============================================================================

type policyRepository struct {
	db  *DB
	log *logrus.Logger
}

var _ repository.PolicyRepository = (*policyRepository)(nil)

// policyRow is the database row representation for policies.
type policyRow struct {
	ID           uuid.UUID       `db:"id"`
	Name         string          `db:"name"`
	Description  string          `db:"description"`
	Priority     int             `db:"priority"`
	ScopePattern string          `db:"scope_pattern"`
	Rules        PolicyRulesJSON `db:"rules"`
	Enabled      bool            `db:"enabled"`
	CreatedBy    string          `db:"created_by"`
	CreatedAt    SQLiteTime      `db:"created_at"`
	UpdatedAt    SQLiteTime      `db:"updated_at"`
}

func (p *policyRow) toDomain() *domain.Policy {
	return &domain.Policy{
		ID:           p.ID,
		Name:         p.Name,
		Description:  p.Description,
		Priority:     p.Priority,
		ScopePattern: p.ScopePattern,
		Rules:        p.Rules.V,
		Enabled:      p.Enabled,
		CreatedBy:    p.CreatedBy,
		CreatedAt:    p.CreatedAt.Time(),
		UpdatedAt:    p.UpdatedAt.Time(),
	}
}

func policyFromDomain(p *domain.Policy) *policyRow {
	return &policyRow{
		ID:           p.ID,
		Name:         p.Name,
		Description:  p.Description,
		Priority:     p.Priority,
		ScopePattern: p.ScopePattern,
		Rules:        PolicyRulesJSON{V: p.Rules},
		Enabled:      p.Enabled,
		CreatedBy:    p.CreatedBy,
		CreatedAt:    SQLiteTime(p.CreatedAt),
		UpdatedAt:    SQLiteTime(p.UpdatedAt),
	}
}

const policyColumns = `id, name, description, priority, scope_pattern, rules, enabled, created_by, created_at, updated_at`

func (r *policyRepository) Create(ctx context.Context, policy *domain.Policy) error {
	if policy.ID == uuid.Nil {
		policy.ID = uuid.New()
	}
	now := time.Now()
	policy.CreatedAt = now
	policy.UpdatedAt = now

	row := policyFromDomain(policy)
	query := `INSERT INTO policies (id, name, description, priority, scope_pattern, rules, enabled, created_by, created_at, updated_at)
		VALUES (:id, :name, :description, :priority, :scope_pattern, :rules, :enabled, :created_by, :created_at, :updated_at)`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}
	return nil
}

func (r *policyRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Policy, error) {
	query := r.db.Rebind(fmt.Sprintf("SELECT %s FROM policies WHERE id = ?", policyColumns))
	var row policyRow
	if err := r.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	return row.toDomain(), nil
}

func (r *policyRepository) List(ctx context.Context, filter repository.ListFilter) ([]*domain.Policy, error) {
	base := fmt.Sprintf("SELECT %s FROM policies ORDER BY priority DESC, updated_at DESC", policyColumns)
	queryWithPaging, args := appendLimitOffset(base, filter.Limit, filter.Offset)
	query := r.db.Rebind(queryWithPaging)

	var rows []policyRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}

	result := make([]*domain.Policy, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

func (r *policyRepository) ListEnabled(ctx context.Context) ([]*domain.Policy, error) {
	query := r.db.Rebind(fmt.Sprintf("SELECT %s FROM policies WHERE enabled = true ORDER BY priority DESC", policyColumns))

	var rows []policyRow
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("failed to list enabled policies: %w", err)
	}

	result := make([]*domain.Policy, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

func (r *policyRepository) Update(ctx context.Context, policy *domain.Policy) error {
	policy.UpdatedAt = time.Now()
	row := policyFromDomain(policy)

	query := `UPDATE policies SET name = :name, description = :description,
		priority = :priority, scope_pattern = :scope_pattern, rules = :rules,
		enabled = :enabled, created_by = :created_by, updated_at = :updated_at
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}
	return nil
}

func (r *policyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := r.db.Rebind(`DELETE FROM policies WHERE id = ?`)
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}
	return nil
}

func (r *policyRepository) FindByScope(ctx context.Context, scopePath string) ([]*domain.Policy, error) {
	// Simple LIKE-based pattern matching for now
	// In production, you might want more sophisticated glob matching
	query := r.db.Rebind(fmt.Sprintf("SELECT %s FROM policies WHERE enabled = true AND (scope_pattern = '' OR ? LIKE REPLACE(REPLACE(scope_pattern, '*', '%%'), '?', '_')) ORDER BY priority DESC", policyColumns))

	var rows []policyRow
	if err := r.db.SelectContext(ctx, &rows, query, scopePath); err != nil {
		return nil, fmt.Errorf("failed to find policies by scope: %w", err)
	}

	result := make([]*domain.Policy, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

// ============================================================================
// LockRepository Implementation
// ============================================================================

type lockRepository struct {
	db  *DB
	log *logrus.Logger
}

var _ repository.LockRepository = (*lockRepository)(nil)

// lockRow is the database row representation for scope_locks.
type lockRow struct {
	ID          uuid.UUID  `db:"id"`
	RunID       uuid.UUID  `db:"run_id"`
	ScopePath   string     `db:"scope_path"`
	ProjectRoot string     `db:"project_root"`
	AcquiredAt  SQLiteTime `db:"acquired_at"`
	ExpiresAt   SQLiteTime `db:"expires_at"`
}

func (l *lockRow) toDomain() *domain.ScopeLock {
	return &domain.ScopeLock{
		ID:          l.ID,
		RunID:       l.RunID,
		ScopePath:   l.ScopePath,
		ProjectRoot: l.ProjectRoot,
		AcquiredAt:  l.AcquiredAt.Time(),
		ExpiresAt:   l.ExpiresAt.Time(),
	}
}

const lockColumns = `id, run_id, scope_path, project_root, acquired_at, expires_at`

func (r *lockRepository) Acquire(ctx context.Context, lock *domain.ScopeLock) error {
	if lock.ID == uuid.Nil {
		lock.ID = uuid.New()
	}
	lock.AcquiredAt = time.Now()

	query := `INSERT INTO scope_locks (id, run_id, scope_path, project_root, acquired_at, expires_at)
		VALUES (:id, :run_id, :scope_path, :project_root, :acquired_at, :expires_at)`

	row := struct {
		ID          uuid.UUID  `db:"id"`
		RunID       uuid.UUID  `db:"run_id"`
		ScopePath   string     `db:"scope_path"`
		ProjectRoot string     `db:"project_root"`
		AcquiredAt  SQLiteTime `db:"acquired_at"`
		ExpiresAt   SQLiteTime `db:"expires_at"`
	}{
		ID:          lock.ID,
		RunID:       lock.RunID,
		ScopePath:   lock.ScopePath,
		ProjectRoot: lock.ProjectRoot,
		AcquiredAt:  SQLiteTime(lock.AcquiredAt),
		ExpiresAt:   SQLiteTime(lock.ExpiresAt),
	}

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	return nil
}

func (r *lockRepository) Release(ctx context.Context, id uuid.UUID) error {
	query := r.db.Rebind(`DELETE FROM scope_locks WHERE id = ?`)
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	return nil
}

func (r *lockRepository) ReleaseByRun(ctx context.Context, runID uuid.UUID) error {
	query := r.db.Rebind(`DELETE FROM scope_locks WHERE run_id = ?`)
	_, err := r.db.ExecContext(ctx, query, runID)
	if err != nil {
		return fmt.Errorf("failed to release locks by run: %w", err)
	}
	return nil
}

func (r *lockRepository) Check(ctx context.Context, scopePath, projectRoot string) ([]*domain.ScopeLock, error) {
	// Find overlapping locks (same scope_path and project_root, not expired)
	query := r.db.Rebind(fmt.Sprintf("SELECT %s FROM scope_locks WHERE scope_path = ? AND project_root = ? AND expires_at > ?", lockColumns))

	var rows []lockRow
	if err := r.db.SelectContext(ctx, &rows, query, scopePath, projectRoot, time.Now()); err != nil {
		return nil, fmt.Errorf("failed to check locks: %w", err)
	}

	result := make([]*domain.ScopeLock, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

func (r *lockRepository) Refresh(ctx context.Context, id uuid.UUID, newExpiry int64) error {
	expiresAt := time.Unix(newExpiry, 0)
	query := r.db.Rebind(`UPDATE scope_locks SET expires_at = ? WHERE id = ?`)
	_, err := r.db.ExecContext(ctx, query, expiresAt, id)
	if err != nil {
		return fmt.Errorf("failed to refresh lock: %w", err)
	}
	return nil
}

func (r *lockRepository) CleanupExpired(ctx context.Context) (int, error) {
	query := r.db.Rebind(`DELETE FROM scope_locks WHERE expires_at < ?`)
	result, err := r.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired locks: %w", err)
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}
