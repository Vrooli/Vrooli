// Package repository provides database operations for sandboxes.
//
// Storage backend: SQLite via modernc.org/sqlite. The schema lives in
// schema.sql (embedded) and is applied on startup. See
// docs/internal/STORAGE_AUDIT.md for the architectural rationale and
// docs/plans/sqlite-cutover-implementation-plan.md for the design notes.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/types"
)

// hydrateReservedFields normalizes ReservedPath/ReservedPaths/ScopePath into
// the canonical representation expected by the rest of the codebase. Kept
// in sync with the contract documented in types.Sandbox.
func hydrateReservedFields(s *types.Sandbox, reservedPaths []string) {
	effective := make([]string, 0, max(1, len(reservedPaths)))
	for _, p := range reservedPaths {
		p = strings.TrimSpace(p)
		if p != "" {
			effective = append(effective, p)
		}
	}
	if len(effective) == 0 && !s.NoLock {
		if strings.TrimSpace(s.ReservedPath) != "" {
			effective = append(effective, strings.TrimSpace(s.ReservedPath))
		} else if strings.TrimSpace(s.ScopePath) != "" {
			effective = append(effective, strings.TrimSpace(s.ScopePath))
		}
	}

	s.ReservedPaths = effective
	if len(effective) > 0 {
		s.ReservedPath = effective[0]
	} else if s.NoLock {
		s.ReservedPath = ""
	}
}

// Repository defines the interface for sandbox persistence operations.
//
// # Idempotency Support
//
// FindByIdempotencyKey lets callers check whether a sandbox was already
// created for a given key, enabling safe retries.
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

	FindByIdempotencyKey(ctx context.Context, key string) (*types.Sandbox, error)
	UpdateWithVersionCheck(ctx context.Context, s *types.Sandbox, expectedVersion int64) error

	BeginTx(ctx context.Context) (TxRepository, error)

	GetGCCandidates(ctx context.Context, policy *types.GCPolicy, limit int) ([]*types.Sandbox, error)

	RecordAppliedChanges(ctx context.Context, changes []*types.AppliedChange) error
	GetPendingChanges(ctx context.Context, projectRoot string, limit, offset int) (*types.PendingChangesResult, error)
	GetPendingChangeFiles(ctx context.Context, projectRoot string, sandboxIDs []uuid.UUID) ([]*types.AppliedChange, error)
	GetFileProvenance(ctx context.Context, filePath, projectRoot string, limit int) ([]*types.AppliedChange, error)
	MarkChangesCommitted(ctx context.Context, ids []uuid.UUID, commitHash, commitMessage string) error
	MarkChangesCommittedByPath(ctx context.Context, projectRoot string, filePaths []string, commitHash, commitMessage string) (int, int, error)
	GetPendingChangesByRun(ctx context.Context, projectRoot string) ([]types.ProvenanceRunGroup, error)

	// Heal-state durability (Round 3 Phase 6). The auto-heal loop's
	// failure history previously lived only in memory and was lost on
	// restart, so a permanently broken sandbox would silently retry
	// forever after every reboot.
	GetHealState(ctx context.Context, sandboxID uuid.UUID) (*HealStateRow, error)
	UpsertHealState(ctx context.Context, row HealStateRow) error
	ClearHealState(ctx context.Context, sandboxID uuid.UUID) error
	ListHealState(ctx context.Context) ([]HealStateRow, error)
}

// HealStateRow is the durable representation of a sandbox's auto-heal
// failure history. Mirrors the heal_state SQLite table.
type HealStateRow struct {
	SandboxID           uuid.UUID
	ConsecutiveFailures int
	LastAttempt         time.Time
	LastError           string
}

// TxRepository is a Repository bound to a transaction.
type TxRepository interface {
	Repository
	Commit() error
	Rollback() error

	// Tx exposes the underlying *sql.Tx so multi-table operations can ride
	// the same transaction as a sandbox status flip (notably the diff
	// archive insert in service_archive.go). Mocks may return nil — the
	// archive repository falls back to its own *sql.DB when tx is nil.
	Tx() *sql.Tx
}

// dbExec abstracts *sql.DB and *sql.Tx so the same query helpers can serve
// both the non-transactional and transactional repositories.
type dbExec interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Verify SandboxRepository implements Repository interface.
var (
	_ Repository   = (*SandboxRepository)(nil)
	_ TxRepository = (*TxSandboxRepository)(nil)
)

// SandboxRepository provides database operations for sandboxes against a
// shared *sql.DB. The injected clock supplies every UTC timestamp the
// repository writes (CreatedAt/UpdatedAt/DeletedAt/EventTime/AppliedAt),
// so tests can drive timestamp-sensitive paths (GC candidate selection,
// audit ordering, idempotency-window expiry) deterministically through
// FakeClock.
type SandboxRepository struct {
	db    *sql.DB
	clock clock.Clock
}

// NewSandboxRepository creates a new repository. clk is required and may
// not be nil — production wires clock.System{}, tests wire FakeClock.
func NewSandboxRepository(db *sql.DB, clk clock.Clock) *SandboxRepository {
	if clk == nil {
		panic("repository.NewSandboxRepository: clock is required")
	}
	return &SandboxRepository{db: db, clock: clk}
}

// TxSandboxRepository is bound to a single transaction. The DSN should set
// _txlock=immediate so BeginTx acquires the SQLite reserved lock up front,
// giving Create + CheckScopeOverlap mutual exclusion against concurrent
// callers racing to claim overlapping reserved paths. Inherits the parent
// repository's clock at BeginTx so the transaction's writes share the same
// time source as the surrounding session.
type TxSandboxRepository struct {
	tx    *sql.Tx
	clock clock.Clock
}

func (r *TxSandboxRepository) Commit() error   { return r.tx.Commit() }
func (r *TxSandboxRepository) Rollback() error { return r.tx.Rollback() }
func (r *TxSandboxRepository) Tx() *sql.Tx     { return r.tx }

func (r *SandboxRepository) BeginTx(ctx context.Context) (TxRepository, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	return &TxSandboxRepository{tx: tx, clock: r.clock}, nil
}

func (r *TxSandboxRepository) BeginTx(ctx context.Context) (TxRepository, error) {
	return nil, errors.New("nested transactions not supported")
}

// ---------------------------------------------------------------------------
// Sandbox CRUD
// ---------------------------------------------------------------------------

const sandboxColumns = `
	id, COALESCE(name, ''), scope_path, COALESCE(reserved_path, ''), reserved_paths, no_lock, project_root,
	COALESCE(owner, ''), owner_type, status, COALESCE(error_message, ''),
	created_at, last_used_at, stopped_at, approved_at, deleted_at,
	driver_id, driver_version, COALESCE(lower_dir, ''), COALESCE(upper_dir, ''),
	COALESCE(work_dir, ''), COALESCE(merged_dir, ''),
	size_bytes, file_count, active_pids, session_count, tags, metadata, behavior,
	COALESCE(idempotency_key, ''), updated_at, version, COALESCE(base_commit_hash, ''),
	home_overlay_state`

func scanSandbox(row interface {
	Scan(...any) error
},
) (*types.Sandbox, error) {
	var (
		s             types.Sandbox
		idStr         string
		createdAt     string
		lastUsedAt    string
		stoppedAt     sql.NullString
		approvedAt    sql.NullString
		deletedAt     sql.NullString
		updatedAt     string
		reservedPaths string
		activePIDsStr string
		tagsStr       string
		metadataJSON  string
		behaviorJSON  string
		noLock        int
	)

	var homeOverlayState string
	if err := row.Scan(
		&idStr, &s.Name, &s.ScopePath, &s.ReservedPath, &reservedPaths, &noLock, &s.ProjectRoot,
		&s.Owner, &s.OwnerType, &s.Status, &s.ErrorMsg,
		&createdAt, &lastUsedAt, &stoppedAt, &approvedAt, &deletedAt,
		&s.DriverID, &s.DriverVersion, &s.LowerDir, &s.UpperDir,
		&s.WorkDir, &s.MergedDir,
		&s.SizeBytes, &s.FileCount, &activePIDsStr, &s.SessionCount, &tagsStr, &metadataJSON, &behaviorJSON,
		&s.IdempotencyKey, &updatedAt, &s.Version, &s.BaseCommitHash,
		&homeOverlayState,
	); err != nil {
		return nil, err
	}
	s.HomeOverlayState = types.HomeOverlayState(homeOverlayState)

	id, err := parseUUID(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse sandbox id: %w", err)
	}
	s.ID = id
	s.NoLock = noLock != 0

	if s.CreatedAt, err = parseTime(createdAt); err != nil {
		return nil, err
	}
	if s.LastUsedAt, err = parseTime(lastUsedAt); err != nil {
		return nil, err
	}
	if s.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return nil, err
	}
	if s.StoppedAt, err = parseTimePtr(stoppedAt); err != nil {
		return nil, err
	}
	if s.ApprovedAt, err = parseTimePtr(approvedAt); err != nil {
		return nil, err
	}
	if s.DeletedAt, err = parseTimePtr(deletedAt); err != nil {
		return nil, err
	}

	tags, err := parseStrings(tagsStr)
	if err != nil {
		return nil, err
	}
	s.Tags = tags

	pids, err := parseInts(activePIDsStr)
	if err != nil {
		return nil, err
	}
	s.ActivePIDs = pids

	if err := parseJSONObject(metadataJSON, &s.Metadata); err != nil {
		return nil, fmt.Errorf("decode metadata: %w", err)
	}
	if err := parseJSONObject(behaviorJSON, &s.Behavior); err != nil {
		return nil, fmt.Errorf("decode behavior: %w", err)
	}

	rps, err := parseStrings(reservedPaths)
	if err != nil {
		return nil, err
	}
	hydrateReservedFields(&s, rps)

	return &s, nil
}

func insertSandbox(ctx context.Context, exec dbExec, clk clock.Clock, s *types.Sandbox) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.Version == 0 {
		s.Version = 1
	}
	now := clk.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	if s.LastUsedAt.IsZero() {
		s.LastUsedAt = now
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = now
	}

	metadataJSON, err := jsonObject(s.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	behaviorJSON, err := jsonObject(s.Behavior)
	if err != nil {
		return fmt.Errorf("marshal behavior: %w", err)
	}

	const query = `
		INSERT INTO sandboxes (
			id, name, scope_path, reserved_path, reserved_paths, no_lock, project_root,
			owner, owner_type, status,
			created_at, last_used_at, updated_at,
			driver_id, driver_version, tags, metadata, behavior,
			idempotency_key, version, base_commit_hash, active_pids,
			home_overlay_state
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	homeOverlayState := s.HomeOverlayState
	if homeOverlayState == "" {
		homeOverlayState = types.HomeOverlayAbsent
	}
	_, err = exec.ExecContext(ctx, query,
		uuidText(s.ID),
		nullableString(s.Name),
		s.ScopePath,
		nullableString(s.ReservedPath),
		jsonStrings(s.ReservedPaths),
		boolInt(s.NoLock),
		s.ProjectRoot,
		nullableString(s.Owner),
		string(s.OwnerType),
		string(s.Status),
		formatTime(s.CreatedAt),
		formatTime(s.LastUsedAt),
		formatTime(s.UpdatedAt),
		s.DriverID,
		s.DriverVersion,
		jsonStrings(s.Tags),
		metadataJSON,
		behaviorJSON,
		nullableString(s.IdempotencyKey),
		s.Version,
		nullableString(s.BaseCommitHash),
		jsonInts(s.ActivePIDs),
		string(homeOverlayState),
	)
	return err
}

// Create inserts a new sandbox record.
func (r *SandboxRepository) Create(ctx context.Context, s *types.Sandbox) error {
	return insertSandbox(ctx, r.db, r.clock, s)
}

func (r *TxSandboxRepository) Create(ctx context.Context, s *types.Sandbox) error {
	return insertSandbox(ctx, r.tx, r.clock, s)
}

func getSandbox(ctx context.Context, exec dbExec, id uuid.UUID) (*types.Sandbox, error) {
	query := "SELECT " + sandboxColumns + " FROM sandboxes WHERE id = ?"
	row := exec.QueryRowContext(ctx, query, uuidText(id))
	s, err := scanSandbox(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get sandbox: %w", err)
	}
	return s, nil
}

func (r *SandboxRepository) Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	return getSandbox(ctx, r.db, id)
}

func (r *TxSandboxRepository) Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	return getSandbox(ctx, r.tx, id)
}

func updateSandbox(ctx context.Context, exec dbExec, clk clock.Clock, s *types.Sandbox) error {
	metadataJSON, err := jsonObject(s.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	behaviorJSON, err := jsonObject(s.Behavior)
	if err != nil {
		return fmt.Errorf("marshal behavior: %w", err)
	}

	now := clk.Now().UTC()
	const query = `
		UPDATE sandboxes SET
			status = ?, error_message = ?,
			stopped_at = ?, approved_at = ?, deleted_at = ?,
			lower_dir = ?, upper_dir = ?, work_dir = ?, merged_dir = ?,
			size_bytes = ?, file_count = ?, active_pids = ?, session_count = ?,
			tags = ?, metadata = ?, behavior = ?,
			home_overlay_state = ?,
			version = version + 1,
			updated_at = ?,
			last_used_at = ?
		WHERE id = ?`

	homeOverlayState := s.HomeOverlayState
	if homeOverlayState == "" {
		homeOverlayState = types.HomeOverlayAbsent
	}
	if _, err := exec.ExecContext(ctx, query,
		string(s.Status), nullableString(s.ErrorMsg),
		formatTimePtr(s.StoppedAt), formatTimePtr(s.ApprovedAt), formatTimePtr(s.DeletedAt),
		nullableString(s.LowerDir), nullableString(s.UpperDir), nullableString(s.WorkDir), nullableString(s.MergedDir),
		s.SizeBytes, s.FileCount, jsonInts(s.ActivePIDs), s.SessionCount,
		jsonStrings(s.Tags), metadataJSON, behaviorJSON,
		string(homeOverlayState),
		formatTime(now),
		formatTime(now),
		uuidText(s.ID),
	); err != nil {
		return err
	}

	// Re-read version + timestamps for the caller.
	row := exec.QueryRowContext(ctx, "SELECT version, updated_at FROM sandboxes WHERE id = ?", uuidText(s.ID))
	var version int64
	var updatedAtStr string
	if err := row.Scan(&version, &updatedAtStr); err != nil {
		return err
	}
	s.Version = version
	if s.UpdatedAt, err = parseTime(updatedAtStr); err != nil {
		return err
	}
	s.LastUsedAt = now
	return nil
}

func (r *SandboxRepository) Update(ctx context.Context, s *types.Sandbox) error {
	return updateSandbox(ctx, r.db, r.clock, s)
}

func (r *TxSandboxRepository) Update(ctx context.Context, s *types.Sandbox) error {
	return updateSandbox(ctx, r.tx, r.clock, s)
}

func deleteSandbox(ctx context.Context, exec dbExec, clk clock.Clock, id uuid.UUID) error {
	now := clk.Now().UTC()
	const query = `
		UPDATE sandboxes
		SET status = 'deleted', deleted_at = ?, version = version + 1, updated_at = ?
		WHERE id = ? AND status != 'deleted'`
	res, err := exec.ExecContext(ctx, query, formatTime(now), formatTime(now), uuidText(id))
	if err != nil {
		return fmt.Errorf("delete sandbox: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("sandbox not found or already deleted")
	}
	return nil
}

func (r *SandboxRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return deleteSandbox(ctx, r.db, r.clock, id)
}

func (r *TxSandboxRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return deleteSandbox(ctx, r.tx, r.clock, id)
}

// ---------------------------------------------------------------------------
// queryBuilder: small helper for assembling parameterized WHERE clauses.
// ---------------------------------------------------------------------------

type queryBuilder struct {
	conditions []string
	args       []any
}

func newQueryBuilder() *queryBuilder { return &queryBuilder{} }

func (qb *queryBuilder) addCondition(column, op string, value any) {
	qb.conditions = append(qb.conditions, column+" "+op+" ?")
	qb.args = append(qb.args, value)
}

func (qb *queryBuilder) addInCondition(column string, values []types.Status) {
	if len(values) == 0 {
		return
	}
	placeholders := strings.Repeat("?,", len(values))
	placeholders = placeholders[:len(placeholders)-1]
	qb.conditions = append(qb.conditions, column+" IN ("+placeholders+")")
	for _, v := range values {
		qb.args = append(qb.args, string(v))
	}
}

func (qb *queryBuilder) whereClause() string {
	if len(qb.conditions) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(qb.conditions, " AND ")
}

// ---------------------------------------------------------------------------
// List / GetActiveSandboxes / GetGCCandidates
// ---------------------------------------------------------------------------

// List retrieves sandboxes matching the filter.
func (r *SandboxRepository) List(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error) {
	qb := newQueryBuilder()
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
	if filter.Name != "" {
		qb.addCondition("name", "LIKE", "%"+filter.Name+"%")
	}
	if !filter.CreatedFrom.IsZero() {
		qb.addCondition("created_at", ">=", formatTime(filter.CreatedFrom))
	}
	if !filter.CreatedTo.IsZero() {
		qb.addCondition("created_at", "<=", formatTime(filter.CreatedTo))
	}

	whereClause := qb.whereClause()

	var totalCount int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sandboxes "+whereClause, qb.args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("count sandboxes: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 10000 {
		limit = 10000
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := "SELECT " + sandboxColumns + " FROM sandboxes " + whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args := append(append([]any{}, qb.args...), limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list sandboxes: %w", err)
	}
	defer rows.Close()

	var sandboxes []*types.Sandbox
	for rows.Next() {
		s, err := scanSandbox(rows)
		if err != nil {
			return nil, err
		}
		sandboxes = append(sandboxes, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &types.ListResult{
		Sandboxes:  sandboxes,
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     offset,
	}, nil
}

func (r *TxSandboxRepository) List(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error) {
	return nil, errors.New("List not implemented for transactions - use non-transactional repository")
}

// GetActiveSandboxes returns sandboxes in non-terminal states for a project.
func (r *SandboxRepository) GetActiveSandboxes(ctx context.Context, projectRoot string) ([]*types.Sandbox, error) {
	res, err := r.List(ctx, &types.ListFilter{
		Status:      []types.Status{types.StatusCreating, types.StatusActive},
		ProjectRoot: projectRoot,
		Limit:       10000,
	})
	if err != nil {
		return nil, err
	}
	return res.Sandboxes, nil
}

func (r *TxSandboxRepository) GetActiveSandboxes(ctx context.Context, projectRoot string) ([]*types.Sandbox, error) {
	return nil, errors.New("GetActiveSandboxes not implemented for transactions")
}

// ---------------------------------------------------------------------------
// CheckScopeOverlap: detects reserved-path conflicts between an incoming
// scope and existing sandboxes within the same project root. The
// transactional variant (TxSandboxRepository) relies on the
// _txlock=immediate DSN parameter to serialize concurrent writers.
// ---------------------------------------------------------------------------

func checkScopeOverlap(ctx context.Context, exec dbExec, scopePath, projectRoot string, excludeID *uuid.UUID) ([]types.PathConflict, error) {
	const query = `
		SELECT id, scope_path, COALESCE(reserved_path, ''), reserved_paths, no_lock, status
		FROM sandboxes
		WHERE project_root = ?
		  AND status IN ('creating', 'active', 'stopped')
		  AND no_lock = 0
		  AND (? = '' OR id != ?)`

	exclude := ""
	if excludeID != nil {
		exclude = excludeID.String()
	}

	rows, err := exec.QueryContext(ctx, query, projectRoot, exclude, exclude)
	if err != nil {
		return nil, fmt.Errorf("check scope overlap: %w", err)
	}
	defer rows.Close()

	var conflicts []types.PathConflict
	for rows.Next() {
		var (
			idStr             string
			existingScope     string
			existingReserved  string
			existingReserveds string
			noLock            int
			status            string
		)
		if err := rows.Scan(&idStr, &existingScope, &existingReserved, &existingReserveds, &noLock, &status); err != nil {
			return nil, fmt.Errorf("scan conflict: %w", err)
		}
		if noLock != 0 {
			continue
		}

		reservedList, err := parseStrings(existingReserveds)
		if err != nil {
			return nil, err
		}

		effective := make([]string, 0, max(1, len(reservedList)))
		for _, p := range reservedList {
			p = strings.TrimSpace(p)
			if p != "" {
				effective = append(effective, p)
			}
		}
		if len(effective) == 0 {
			if strings.TrimSpace(existingReserved) != "" {
				effective = append(effective, strings.TrimSpace(existingReserved))
			} else if strings.TrimSpace(existingScope) != "" {
				effective = append(effective, strings.TrimSpace(existingScope))
			}
		}

		for _, prefix := range effective {
			conflictType := types.CheckPathOverlap(prefix, scopePath)
			if conflictType == "" {
				continue
			}
			conflicts = append(conflicts, types.PathConflict{
				ExistingID:    idStr,
				ExistingScope: prefix,
				NewScope:      scopePath,
				ConflictType:  conflictType,
			})
		}
	}
	return conflicts, rows.Err()
}

func (r *SandboxRepository) CheckScopeOverlap(ctx context.Context, scopePath, projectRoot string, excludeID *uuid.UUID) ([]types.PathConflict, error) {
	return checkScopeOverlap(ctx, r.db, scopePath, projectRoot, excludeID)
}

func (r *TxSandboxRepository) CheckScopeOverlap(ctx context.Context, scopePath, projectRoot string, excludeID *uuid.UUID) ([]types.PathConflict, error) {
	return checkScopeOverlap(ctx, r.tx, scopePath, projectRoot, excludeID)
}

// ---------------------------------------------------------------------------
// Audit log
// ---------------------------------------------------------------------------

func logAuditEvent(ctx context.Context, exec dbExec, clk clock.Clock, event *types.AuditEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.EventTime.IsZero() {
		event.EventTime = clk.Now().UTC()
	}
	if event.ActorType == "" {
		event.ActorType = "system"
	}

	detailsJSON, _ := jsonObject(event.Details)
	if detailsJSON == "" {
		detailsJSON = "{}"
	}
	stateJSON := ""
	if event.SandboxState != nil {
		s, err := jsonObject(event.SandboxState)
		if err == nil {
			stateJSON = s
		}
	}

	var sandboxIDArg sql.NullString
	if event.SandboxID != nil {
		sandboxIDArg = sql.NullString{String: event.SandboxID.String(), Valid: true}
	}

	const query = `
		INSERT INTO sandbox_audit_log (id, sandbox_id, event_type, event_time, actor, actor_type, details, sandbox_state)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := exec.ExecContext(ctx, query,
		uuidText(event.ID),
		sandboxIDArg,
		event.EventType,
		formatTime(event.EventTime),
		nullableString(event.Actor),
		event.ActorType,
		detailsJSON,
		nullableString(stateJSON),
	)
	return err
}

func (r *SandboxRepository) LogAuditEvent(ctx context.Context, event *types.AuditEvent) error {
	return logAuditEvent(ctx, r.db, r.clock, event)
}

func (r *TxSandboxRepository) LogAuditEvent(ctx context.Context, event *types.AuditEvent) error {
	return logAuditEvent(ctx, r.tx, r.clock, event)
}

// GetAuditLog retrieves audit events ordered by event_time DESC.
func (r *SandboxRepository) GetAuditLog(ctx context.Context, sandboxID *uuid.UUID, limit, offset int) ([]*types.AuditEvent, int, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	whereClause := ""
	args := []any{}
	if sandboxID != nil {
		whereClause = "WHERE sandbox_id = ?"
		args = append(args, sandboxID.String())
	}

	var totalCount int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sandbox_audit_log "+whereClause, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("count audit events: %w", err)
	}

	query := `
		SELECT id, sandbox_id, event_type, event_time, COALESCE(actor, ''), actor_type, details, COALESCE(sandbox_state, '')
		FROM sandbox_audit_log
		` + whereClause + `
		ORDER BY event_time DESC
		LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit log: %w", err)
	}
	defer rows.Close()

	var events []*types.AuditEvent
	for rows.Next() {
		var (
			ev           types.AuditEvent
			idStr        string
			sandboxIDStr sql.NullString
			eventTimeStr string
			detailsJSON  string
			stateJSON    string
		)
		if err := rows.Scan(&idStr, &sandboxIDStr, &ev.EventType, &eventTimeStr, &ev.Actor, &ev.ActorType, &detailsJSON, &stateJSON); err != nil {
			return nil, 0, fmt.Errorf("scan audit event: %w", err)
		}
		id, err := parseUUID(idStr)
		if err != nil {
			return nil, 0, fmt.Errorf("parse audit id: %w", err)
		}
		ev.ID = id
		if sandboxIDStr.Valid && sandboxIDStr.String != "" {
			sid, err := parseUUID(sandboxIDStr.String)
			if err != nil {
				return nil, 0, fmt.Errorf("parse audit sandbox id: %w", err)
			}
			ev.SandboxID = &sid
		}
		if ev.EventTime, err = parseTime(eventTimeStr); err != nil {
			return nil, 0, err
		}
		if err := parseJSONObject(detailsJSON, &ev.Details); err != nil {
			return nil, 0, fmt.Errorf("decode audit details: %w", err)
		}
		if err := parseJSONObject(stateJSON, &ev.SandboxState); err != nil {
			return nil, 0, fmt.Errorf("decode audit state: %w", err)
		}
		events = append(events, &ev)
	}
	return events, totalCount, rows.Err()
}

func (r *TxSandboxRepository) GetAuditLog(ctx context.Context, sandboxID *uuid.UUID, limit, offset int) ([]*types.AuditEvent, int, error) {
	return nil, 0, errors.New("GetAuditLog not implemented for transactions")
}

// ---------------------------------------------------------------------------
// Stats: aggregate counts per status and total size.
// ---------------------------------------------------------------------------

func (r *SandboxRepository) GetStats(ctx context.Context) (*types.SandboxStats, error) {
	const query = `
		SELECT
			COUNT(*) AS total,
			SUM(CASE WHEN status = 'active'   THEN 1 ELSE 0 END) AS active,
			SUM(CASE WHEN status = 'stopped'  THEN 1 ELSE 0 END) AS stopped,
			SUM(CASE WHEN status = 'error'    THEN 1 ELSE 0 END) AS errored,
			SUM(CASE WHEN status = 'approved' THEN 1 ELSE 0 END) AS approved,
			SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END) AS rejected,
			SUM(CASE WHEN status = 'deleted'  THEN 1 ELSE 0 END) AS deleted,
			COALESCE(SUM(size_bytes), 0)                          AS total_size,
			COALESCE(AVG(size_bytes), 0)                          AS avg_size
		FROM sandboxes`

	stats := &types.SandboxStats{}
	if err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalCount,
		&stats.ActiveCount,
		&stats.StoppedCount,
		&stats.ErrorCount,
		&stats.ApprovedCount,
		&stats.RejectedCount,
		&stats.DeletedCount,
		&stats.TotalSizeBytes,
		&stats.AvgSizeBytes,
	); err != nil {
		return nil, fmt.Errorf("get sandbox stats: %w", err)
	}
	return stats, nil
}

func (r *TxSandboxRepository) GetStats(ctx context.Context) (*types.SandboxStats, error) {
	return nil, errors.New("GetStats not implemented for transactions")
}

// ---------------------------------------------------------------------------
// Idempotency lookup
// ---------------------------------------------------------------------------

func findByIdempotencyKey(ctx context.Context, exec dbExec, key string) (*types.Sandbox, error) {
	if key == "" {
		return nil, nil
	}
	query := "SELECT " + sandboxColumns + " FROM sandboxes WHERE idempotency_key = ?"
	row := exec.QueryRowContext(ctx, query, key)
	s, err := scanSandbox(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find sandbox by idempotency key: %w", err)
	}
	return s, nil
}

func (r *SandboxRepository) FindByIdempotencyKey(ctx context.Context, key string) (*types.Sandbox, error) {
	return findByIdempotencyKey(ctx, r.db, key)
}

func (r *TxSandboxRepository) FindByIdempotencyKey(ctx context.Context, key string) (*types.Sandbox, error) {
	return findByIdempotencyKey(ctx, r.tx, key)
}

// ---------------------------------------------------------------------------
// Optimistic locking
// ---------------------------------------------------------------------------

func updateWithVersionCheck(ctx context.Context, exec dbExec, clk clock.Clock, s *types.Sandbox, expectedVersion int64) error {
	metadataJSON, err := jsonObject(s.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	behaviorJSON, err := jsonObject(s.Behavior)
	if err != nil {
		return fmt.Errorf("marshal behavior: %w", err)
	}

	now := clk.Now().UTC()
	const query = `
		UPDATE sandboxes SET
			status = ?, error_message = ?,
			stopped_at = ?, approved_at = ?, deleted_at = ?,
			lower_dir = ?, upper_dir = ?, work_dir = ?, merged_dir = ?,
			size_bytes = ?, file_count = ?, active_pids = ?, session_count = ?,
			tags = ?, metadata = ?, behavior = ?,
			home_overlay_state = ?,
			version = version + 1,
			updated_at = ?,
			last_used_at = ?
		WHERE id = ? AND version = ?`

	homeOverlayState := s.HomeOverlayState
	if homeOverlayState == "" {
		homeOverlayState = types.HomeOverlayAbsent
	}
	res, err := exec.ExecContext(ctx, query,
		string(s.Status), nullableString(s.ErrorMsg),
		formatTimePtr(s.StoppedAt), formatTimePtr(s.ApprovedAt), formatTimePtr(s.DeletedAt),
		nullableString(s.LowerDir), nullableString(s.UpperDir), nullableString(s.WorkDir), nullableString(s.MergedDir),
		s.SizeBytes, s.FileCount, jsonInts(s.ActivePIDs), s.SessionCount,
		jsonStrings(s.Tags), metadataJSON, behaviorJSON,
		string(homeOverlayState),
		formatTime(now),
		formatTime(now),
		uuidText(s.ID), expectedVersion,
	)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		var currentVersion int64
		if err := exec.QueryRowContext(ctx, "SELECT version FROM sandboxes WHERE id = ?", uuidText(s.ID)).Scan(&currentVersion); err == nil {
			return types.NewConcurrentModificationError(s.ID.String(), expectedVersion, currentVersion)
		}
		return types.NewNotFoundError(s.ID.String())
	}

	row := exec.QueryRowContext(ctx, "SELECT version, updated_at FROM sandboxes WHERE id = ?", uuidText(s.ID))
	var version int64
	var updatedAtStr string
	if err := row.Scan(&version, &updatedAtStr); err != nil {
		return err
	}
	s.Version = version
	if s.UpdatedAt, err = parseTime(updatedAtStr); err != nil {
		return err
	}
	s.LastUsedAt = now
	return nil
}

func (r *SandboxRepository) UpdateWithVersionCheck(ctx context.Context, s *types.Sandbox, expectedVersion int64) error {
	return updateWithVersionCheck(ctx, r.db, r.clock, s, expectedVersion)
}

func (r *TxSandboxRepository) UpdateWithVersionCheck(ctx context.Context, s *types.Sandbox, expectedVersion int64) error {
	return updateWithVersionCheck(ctx, r.tx, r.clock, s, expectedVersion)
}

// ---------------------------------------------------------------------------
// GC candidates
// ---------------------------------------------------------------------------

func (r *SandboxRepository) GetGCCandidates(ctx context.Context, policy *types.GCPolicy, limit int) ([]*types.Sandbox, error) {
	if policy == nil {
		defaultPolicy := types.DefaultGCPolicy()
		policy = &defaultPolicy
	}

	statuses := policy.Statuses
	if len(statuses) == 0 {
		statuses = []types.Status{types.StatusStopped, types.StatusError}
		if policy.IncludeTerminal {
			statuses = append(statuses, types.StatusApproved, types.StatusRejected)
		}
	}
	safeStatuses := make([]types.Status, 0, len(statuses))
	for _, s := range statuses {
		if s != types.StatusActive && s != types.StatusCreating {
			safeStatuses = append(safeStatuses, s)
		}
	}
	if len(safeStatuses) == 0 {
		return []*types.Sandbox{}, nil
	}

	qb := newQueryBuilder()
	qb.addInCondition("status", safeStatuses)

	now := r.clock.Now().UTC()
	var orConditions []string

	if policy.MaxAge > 0 {
		orConditions = append(orConditions, "created_at < ?")
		qb.args = append(qb.args, formatTime(now.Add(-policy.MaxAge)))
	}
	if policy.IdleTimeout > 0 {
		orConditions = append(orConditions, "last_used_at < ?")
		qb.args = append(qb.args, formatTime(now.Add(-policy.IdleTimeout)))
	}
	if policy.IncludeTerminal && policy.TerminalDelay > 0 {
		orConditions = append(orConditions, "(status IN ('approved', 'rejected') AND (approved_at < ? OR stopped_at < ?))")
		cutoff := formatTime(now.Add(-policy.TerminalDelay))
		qb.args = append(qb.args, cutoff, cutoff)
	}

	if len(orConditions) == 0 {
		return []*types.Sandbox{}, nil
	}

	whereClause := qb.whereClause()
	if whereClause != "" {
		whereClause += " AND (" + strings.Join(orConditions, " OR ") + ")"
	} else {
		whereClause = "WHERE (" + strings.Join(orConditions, " OR ") + ")"
	}

	if limit <= 0 {
		limit = 1000
	}

	query := "SELECT " + sandboxColumns + " FROM sandboxes " + whereClause + " ORDER BY created_at ASC LIMIT ?"
	args := append(append([]any{}, qb.args...), limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query GC candidates: %w", err)
	}
	defer rows.Close()

	var sandboxes []*types.Sandbox
	for rows.Next() {
		s, err := scanSandbox(rows)
		if err != nil {
			return nil, err
		}
		sandboxes = append(sandboxes, s)
	}
	return sandboxes, rows.Err()
}

func (r *TxSandboxRepository) GetGCCandidates(ctx context.Context, policy *types.GCPolicy, limit int) ([]*types.Sandbox, error) {
	return nil, errors.New("GetGCCandidates not implemented for transactions")
}

// ---------------------------------------------------------------------------
// Provenance / applied_changes
// ---------------------------------------------------------------------------

func recordAppliedChanges(ctx context.Context, exec dbExec, clk clock.Clock, changes []*types.AppliedChange) error {
	if len(changes) == 0 {
		return nil
	}
	const query = `
		INSERT INTO applied_changes (
			id, sandbox_id, sandbox_owner, sandbox_owner_type,
			file_path, project_root, change_type, file_size, applied_at, agent_manager_run_id,
			run_outcome, provenance_state, conversation_id, cost_usd
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	for _, c := range changes {
		if c.ID == uuid.Nil {
			c.ID = uuid.New()
		}
		if c.AppliedAt.IsZero() {
			c.AppliedAt = clk.Now().UTC()
		}
		var costArg any
		if c.CostUSD != 0 {
			costArg = c.CostUSD
		}
		_, err := exec.ExecContext(ctx, query,
			uuidText(c.ID),
			uuidText(c.SandboxID),
			nullableString(c.SandboxOwner),
			nullableString(c.SandboxOwnerType),
			c.FilePath,
			c.ProjectRoot,
			c.ChangeType,
			c.FileSize,
			formatTime(c.AppliedAt),
			nullableString(c.AgentManagerRunID),
			nullableString(c.RunOutcome),
			nullableString(c.ProvenanceState),
			nullableString(c.ConversationID),
			costArg,
		)
		if err != nil {
			return fmt.Errorf("record applied change for %s: %w", c.FilePath, err)
		}
	}
	return nil
}

func (r *SandboxRepository) RecordAppliedChanges(ctx context.Context, changes []*types.AppliedChange) error {
	return recordAppliedChanges(ctx, r.db, r.clock, changes)
}

func (r *TxSandboxRepository) RecordAppliedChanges(ctx context.Context, changes []*types.AppliedChange) error {
	return recordAppliedChanges(ctx, r.tx, r.clock, changes)
}

// GetPendingChanges returns counts grouped by sandbox.
func (r *SandboxRepository) GetPendingChanges(ctx context.Context, projectRoot string, limit, offset int) (*types.PendingChangesResult, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	whereClause := "WHERE committed_at IS NULL"
	args := []any{}
	if projectRoot != "" {
		whereClause += " AND project_root = ?"
		args = append(args, projectRoot)
	}

	var totalFiles int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM applied_changes "+whereClause, args...).Scan(&totalFiles); err != nil {
		return nil, fmt.Errorf("count pending changes: %w", err)
	}

	query := `
		SELECT sandbox_id, COALESCE(sandbox_owner, '') AS sandbox_owner,
		       COUNT(*) AS file_count, MAX(applied_at) AS latest_applied
		FROM applied_changes
		` + whereClause + `
		GROUP BY sandbox_id, sandbox_owner
		ORDER BY latest_applied DESC
		LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query pending changes: %w", err)
	}
	defer rows.Close()

	var summaries []types.PendingChangesSummary
	for rows.Next() {
		var (
			sandboxIDStr string
			summary      types.PendingChangesSummary
			latest       string
		)
		if err := rows.Scan(&sandboxIDStr, &summary.SandboxOwner, &summary.FileCount, &latest); err != nil {
			return nil, fmt.Errorf("scan pending change summary: %w", err)
		}
		id, err := parseUUID(sandboxIDStr)
		if err != nil {
			return nil, err
		}
		summary.SandboxID = id
		if summary.LatestApplied, err = parseTime(latest); err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	return &types.PendingChangesResult{
		Summaries:  summaries,
		TotalFiles: totalFiles,
	}, rows.Err()
}

func (r *TxSandboxRepository) GetPendingChanges(ctx context.Context, projectRoot string, limit, offset int) (*types.PendingChangesResult, error) {
	return nil, errors.New("GetPendingChanges not implemented for transactions")
}

func (r *SandboxRepository) GetPendingChangeFiles(ctx context.Context, projectRoot string, sandboxIDs []uuid.UUID) ([]*types.AppliedChange, error) {
	whereClause := "WHERE committed_at IS NULL"
	args := []any{}
	if projectRoot != "" {
		whereClause += " AND project_root = ?"
		args = append(args, projectRoot)
	}
	if len(sandboxIDs) > 0 {
		placeholders := strings.Repeat("?,", len(sandboxIDs))
		placeholders = placeholders[:len(placeholders)-1]
		whereClause += " AND sandbox_id IN (" + placeholders + ")"
		for _, id := range sandboxIDs {
			args = append(args, id.String())
		}
	}

	query := `
		SELECT id, sandbox_id, COALESCE(sandbox_owner, ''), COALESCE(sandbox_owner_type, ''),
			   file_path, project_root, change_type, file_size, applied_at,
			   COALESCE(agent_manager_run_id, ''),
			   COALESCE(run_outcome, ''),
			   COALESCE(provenance_state, ''), COALESCE(conversation_id, ''),
			   COALESCE(cost_usd, 0)
		FROM applied_changes
		` + whereClause + `
		ORDER BY applied_at ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query pending change files: %w", err)
	}
	defer rows.Close()

	var changes []*types.AppliedChange
	for rows.Next() {
		change, err := scanAppliedChangePending(rows)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, rows.Err()
}

func (r *TxSandboxRepository) GetPendingChangeFiles(ctx context.Context, projectRoot string, sandboxIDs []uuid.UUID) ([]*types.AppliedChange, error) {
	return nil, errors.New("GetPendingChangeFiles not implemented for transactions")
}

func scanAppliedChangePending(rows *sql.Rows) (*types.AppliedChange, error) {
	var (
		c            types.AppliedChange
		idStr        string
		sandboxIDStr string
		appliedAt    string
	)
	if err := rows.Scan(
		&idStr, &sandboxIDStr, &c.SandboxOwner, &c.SandboxOwnerType,
		&c.FilePath, &c.ProjectRoot, &c.ChangeType, &c.FileSize, &appliedAt,
		&c.AgentManagerRunID,
		&c.RunOutcome,
		&c.ProvenanceState, &c.ConversationID,
		&c.CostUSD,
	); err != nil {
		return nil, fmt.Errorf("scan applied change: %w", err)
	}
	id, err := parseUUID(idStr)
	if err != nil {
		return nil, err
	}
	c.ID = id
	sid, err := parseUUID(sandboxIDStr)
	if err != nil {
		return nil, err
	}
	c.SandboxID = sid
	if c.AppliedAt, err = parseTime(appliedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *SandboxRepository) GetFileProvenance(ctx context.Context, filePath, projectRoot string, limit int) ([]*types.AppliedChange, error) {
	if limit <= 0 {
		limit = 50
	}

	args := []any{filePath}
	whereClause := "WHERE file_path = ?"
	if projectRoot != "" {
		whereClause += " AND project_root = ?"
		args = append(args, projectRoot)
	}

	query := `
		SELECT id, sandbox_id, COALESCE(sandbox_owner, ''), COALESCE(sandbox_owner_type, ''),
			   file_path, project_root, change_type, file_size, applied_at,
			   committed_at, COALESCE(commit_hash, ''), COALESCE(commit_message, ''),
			   COALESCE(agent_manager_run_id, ''),
			   COALESCE(run_outcome, ''),
			   COALESCE(provenance_state, ''), COALESCE(conversation_id, ''),
			   COALESCE(cost_usd, 0)
		FROM applied_changes
		` + whereClause + `
		ORDER BY applied_at DESC
		LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query file provenance: %w", err)
	}
	defer rows.Close()

	var changes []*types.AppliedChange
	for rows.Next() {
		var (
			c            types.AppliedChange
			idStr        string
			sandboxIDStr string
			appliedAt    string
			committedAt  sql.NullString
		)
		if err := rows.Scan(
			&idStr, &sandboxIDStr, &c.SandboxOwner, &c.SandboxOwnerType,
			&c.FilePath, &c.ProjectRoot, &c.ChangeType, &c.FileSize, &appliedAt,
			&committedAt, &c.CommitHash, &c.CommitMessage,
			&c.AgentManagerRunID,
			&c.RunOutcome,
			&c.ProvenanceState, &c.ConversationID,
			&c.CostUSD,
		); err != nil {
			return nil, fmt.Errorf("scan file provenance: %w", err)
		}
		id, err := parseUUID(idStr)
		if err != nil {
			return nil, err
		}
		c.ID = id
		sid, err := parseUUID(sandboxIDStr)
		if err != nil {
			return nil, err
		}
		c.SandboxID = sid
		if c.AppliedAt, err = parseTime(appliedAt); err != nil {
			return nil, err
		}
		if c.CommittedAt, err = parseTimePtr(committedAt); err != nil {
			return nil, err
		}
		changes = append(changes, &c)
	}
	return changes, rows.Err()
}

func (r *TxSandboxRepository) GetFileProvenance(ctx context.Context, filePath, projectRoot string, limit int) ([]*types.AppliedChange, error) {
	return nil, errors.New("GetFileProvenance not implemented for transactions")
}

func (r *SandboxRepository) MarkChangesCommitted(ctx context.Context, ids []uuid.UUID, commitHash, commitMessage string) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]

	args := make([]any, 0, len(ids)+3)
	args = append(args, formatTime(r.clock.Now().UTC()), commitHash, commitMessage)
	for _, id := range ids {
		args = append(args, id.String())
	}

	query := "UPDATE applied_changes SET committed_at = ?, commit_hash = ?, commit_message = ? WHERE id IN (" + placeholders + ")"
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("mark changes committed: %w", err)
	}
	return nil
}

func (r *TxSandboxRepository) MarkChangesCommitted(ctx context.Context, ids []uuid.UUID, commitHash, commitMessage string) error {
	return errors.New("MarkChangesCommitted not implemented for transactions")
}

func (r *SandboxRepository) MarkChangesCommittedByPath(ctx context.Context, projectRoot string, filePaths []string, commitHash, commitMessage string) (int, int, error) {
	if len(filePaths) == 0 {
		return 0, 0, nil
	}

	placeholders := strings.Repeat("?,", len(filePaths))
	placeholders = placeholders[:len(placeholders)-1]

	args := make([]any, 0, len(filePaths)+4)
	args = append(args, formatTime(r.clock.Now().UTC()), commitHash, commitMessage, projectRoot)
	for _, fp := range filePaths {
		if !strings.HasPrefix(fp, "/") {
			fp = filepath.Join(projectRoot, fp)
		}
		args = append(args, fp)
	}

	query := `
		UPDATE applied_changes
		SET committed_at = ?, commit_hash = ?, commit_message = ?
		WHERE project_root = ?
		  AND file_path IN (` + placeholders + `)
		  AND committed_at IS NULL`

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, 0, fmt.Errorf("mark changes committed by path: %w", err)
	}
	marked, _ := res.RowsAffected()
	notFound := len(filePaths) - int(marked)
	if notFound < 0 {
		notFound = 0
	}
	return int(marked), notFound, nil
}

func (r *TxSandboxRepository) MarkChangesCommittedByPath(ctx context.Context, projectRoot string, filePaths []string, commitHash, commitMessage string) (int, int, error) {
	return 0, 0, errors.New("MarkChangesCommittedByPath not implemented for transactions")
}

func (r *SandboxRepository) GetPendingChangesByRun(ctx context.Context, projectRoot string) ([]types.ProvenanceRunGroup, error) {
	args := []any{}
	whereClause := "WHERE committed_at IS NULL"
	if projectRoot != "" {
		whereClause += " AND project_root = ?"
		args = append(args, projectRoot)
	}

	query := `
		SELECT COALESCE(agent_manager_run_id, ''), sandbox_id, COALESCE(sandbox_owner, ''),
			   file_path, change_type, applied_at,
			   COALESCE(run_outcome, ''), COALESCE(conversation_id, ''),
			   COALESCE(cost_usd, 0), COALESCE(provenance_state, '')
		FROM applied_changes
		` + whereClause + `
		ORDER BY COALESCE(agent_manager_run_id, ''), applied_at ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query pending changes by run: %w", err)
	}
	defer rows.Close()

	groupMap := make(map[string]*types.ProvenanceRunGroup)
	var groupOrder []string

	for rows.Next() {
		var (
			runID, sandboxID, owner, filePath, changeType, runOutcome, convID, provState string
			appliedAtStr                                                                 string
			costUSD                                                                      float64
		)
		if err := rows.Scan(&runID, &sandboxID, &owner, &filePath, &changeType, &appliedAtStr,
			&runOutcome, &convID, &costUSD, &provState); err != nil {
			return nil, fmt.Errorf("scan pending change by run: %w", err)
		}
		appliedAt, err := parseTime(appliedAtStr)
		if err != nil {
			return nil, err
		}

		group, exists := groupMap[runID]
		if !exists {
			group = &types.ProvenanceRunGroup{
				RunID:          runID,
				SandboxID:      sandboxID,
				SandboxOwner:   owner,
				RunOutcome:     runOutcome,
				ConversationID: convID,
				CostUSD:        costUSD,
			}
			groupMap[runID] = group
			groupOrder = append(groupOrder, runID)
		}

		relPath := filePath
		if projectRoot != "" {
			relPath = strings.TrimPrefix(filePath, projectRoot+"/")
		}

		group.Files = append(group.Files, types.ProvenanceFile{
			FilePath:     filePath,
			RelativePath: relPath,
			ChangeType:   changeType,
			AppliedAt:    appliedAt,
			State:        types.ProvenanceFileState(provState),
		})
		if appliedAt.After(group.LatestAppliedAt) {
			group.LatestAppliedAt = appliedAt
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]types.ProvenanceRunGroup, 0, len(groupOrder))
	for _, key := range groupOrder {
		result = append(result, *groupMap[key])
	}
	return result, nil
}

func (r *TxSandboxRepository) GetPendingChangesByRun(ctx context.Context, projectRoot string) ([]types.ProvenanceRunGroup, error) {
	return nil, errors.New("GetPendingChangesByRun not implemented for transactions")
}
