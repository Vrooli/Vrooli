package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"

	"github.com/vrooli/api-core/schedule"
)

// ArchiveRepository persists DiffArchive metadata and lists/queries it.
//
// The interface is intentionally separate from Repository (the sandbox
// repository) because archives have a different lifecycle and a much
// narrower query surface. The two repositories share the same SQLite
// database; the snapshot service rides one transaction across both
// (see Service.snapshotDiff in Phase 2) so the archive insert and the
// sandbox status flip become atomic.
//
// Insert is the only method that takes a *sql.Tx. All read/list/delete
// methods auto-commit on the shared *sql.DB — they are not part of any
// status-flip transaction and would only deadlock against one.
type ArchiveRepository interface {
	// Insert writes archive into the sandbox_diff_archives table.
	// Must be called inside a *sql.Tx so the row is committed
	// atomically with the sandbox status update. Pass tx=nil to
	// auto-commit (used only by tests; production always rides a tx).
	Insert(ctx context.Context, tx *sql.Tx, archive *types.DiffArchive) error

	// Get returns the archive for sandboxID, or (nil, nil) when no
	// archive exists.
	Get(ctx context.Context, sandboxID uuid.UUID) (*types.DiffArchive, error)

	// List returns archives matching filter and the total matching
	// count (for pagination), in the order the filter requests.
	List(ctx context.Context, filter types.ArchiveListFilter) ([]*types.DiffArchive, int, error)

	// Delete removes one archive row. Idempotent — returns nil if
	// the row is already absent. The caller is responsible for
	// removing the corresponding blobstore directory.
	Delete(ctx context.Context, sandboxID uuid.UUID) error

	// SumSizeBytes returns the sum of TotalBlobBytes across archives
	// matching projectRoot. Empty projectRoot sums all archives. Used
	// by retention to enforce a size budget.
	SumSizeBytes(ctx context.Context, projectRoot string) (int64, error)

	// OldestN returns up to n archives ordered by snapshot_at ASC
	// (oldest first). Used by retention to evict the oldest archives
	// when the size budget is exceeded.
	OldestN(ctx context.Context, n int) ([]*types.DiffArchive, error)

	// AllOrdered returns every archive ordered by snapshot_at ASC,
	// tie-broken by sandbox_id. Used by the archive-retention
	// reconciler which needs a global view to compute eviction sets
	// across age, size, and per-project caps in a single pass.
	//
	// The result is fully materialized in memory; archive metadata
	// is small (a few hundred bytes per row) so this is cheap up to
	// O(100k) rows. If the table grows past that, switch to a paged
	// or streaming variant — the algorithm in the reconciler is the
	// only caller.
	AllOrdered(ctx context.Context) ([]*types.DiffArchive, error)
}

// Verify compile-time conformance.
var _ ArchiveRepository = (*SandboxArchiveRepository)(nil)

// SandboxArchiveRepository is the production ArchiveRepository backed by
// the same *sql.DB the SandboxRepository uses.
type SandboxArchiveRepository struct {
	db    *sql.DB
	clock schedule.Clock
}

// NewArchiveRepository constructs the production archive repository.
// clk is required: SnapshotAt is derived from clk.Now when the inserted
// DiffArchive leaves SnapshotAt as the zero value.
func NewArchiveRepository(db *sql.DB, clk schedule.Clock) *SandboxArchiveRepository {
	if db == nil {
		panic("repository.NewArchiveRepository: db is required")
	}
	if clk == nil {
		panic("repository.NewArchiveRepository: clock is required")
	}
	return &SandboxArchiveRepository{db: db, clock: clk}
}

// archiveColumns is the canonical column projection for SELECTs. Kept
// in one place so scanArchive and List build the same shape.
const archiveColumns = `
	sandbox_id, snapshot_at, archive_state, files_json, stats_json,
	COALESCE(unified_diff_path, ''),
	total_blob_bytes, project_root,
	COALESCE(owner, ''),
	COALESCE(agent_manager_run_id, ''),
	sandbox_status`

// Insert writes a row, validating shape and serializing JSON fields.
// Idempotent on (sandbox_id) by design — a re-insert for the same
// sandbox replaces the prior row. In practice the snapshot service
// only ever calls this once per sandbox (terminal transition is
// one-way per the state machine), but the upsert shape keeps tests
// honest and tolerates retries against transient SQL failures.
func (r *SandboxArchiveRepository) Insert(ctx context.Context, tx *sql.Tx, archive *types.DiffArchive) error {
	if archive == nil {
		return errors.New("archive_repo: nil archive")
	}
	if archive.SandboxID == uuid.Nil {
		return errors.New("archive_repo: sandbox_id required")
	}
	if !archive.ArchiveState.IsValid() {
		return fmt.Errorf("archive_repo: invalid archive_state %q", archive.ArchiveState)
	}
	// archive-bearing terminal states: Approved, Rejected, Deleted.
	// Error is terminal in the state-machine sense but never archive-
	// bearing — the not-captured row is written under StatusDeleted
	// after Error→Deleted, not under StatusError.
	switch archive.SandboxStatus {
	case types.StatusApproved, types.StatusRejected, types.StatusDeleted:
		// ok
	case types.StatusError:
		return fmt.Errorf("archive_repo: sandbox_status %q is not an archive-bearing terminal state", archive.SandboxStatus)
	default:
		return fmt.Errorf("archive_repo: sandbox_status must be terminal (got %q)", archive.SandboxStatus)
	}

	// not_captured rows must have no blob references and no diff blob.
	if archive.ArchiveState == types.ArchiveStateNotCaptured {
		if len(archive.Files) > 0 {
			return errors.New("archive_repo: not_captured archive has files (must be empty)")
		}
		if archive.UnifiedDiffSHA256 != "" {
			return errors.New("archive_repo: not_captured archive has unified diff blob (must be empty)")
		}
		if archive.TotalBlobBytes != 0 {
			return errors.New("archive_repo: not_captured archive has non-zero total_blob_bytes")
		}
	}

	if archive.SnapshotAt.IsZero() {
		archive.SnapshotAt = r.clock.Now().UTC()
	}

	filesJSON, err := json.Marshal(archive.Files)
	if err != nil {
		return fmt.Errorf("archive_repo: marshal files: %w", err)
	}
	if archive.Files == nil {
		// Stable shape: avoid storing "null" for unset Files.
		filesJSON = []byte("[]")
	}
	statsJSON, err := json.Marshal(archive.Stats)
	if err != nil {
		return fmt.Errorf("archive_repo: marshal stats: %w", err)
	}

	// INSERT OR REPLACE makes the call idempotent on sandbox_id.
	const query = `
		INSERT OR REPLACE INTO sandbox_diff_archives (
			sandbox_id, snapshot_at, archive_state, files_json, stats_json,
			unified_diff_path, total_blob_bytes, project_root, owner,
			agent_manager_run_id, sandbox_status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	args := []any{
		uuidText(archive.SandboxID),
		formatTime(archive.SnapshotAt),
		string(archive.ArchiveState),
		string(filesJSON),
		string(statsJSON),
		nullableString(archive.UnifiedDiffSHA256),
		archive.TotalBlobBytes,
		archive.ProjectRoot,
		nullableString(archive.Owner),
		nullableString(archive.AgentManagerRunID),
		string(archive.SandboxStatus),
	}

	if tx != nil {
		_, err = tx.ExecContext(ctx, query, args...)
	} else {
		_, err = r.db.ExecContext(ctx, query, args...)
	}
	if err != nil {
		return fmt.Errorf("archive_repo: insert: %w", err)
	}
	return nil
}

// Get returns the archive for sandboxID, or (nil, nil) when missing.
func (r *SandboxArchiveRepository) Get(ctx context.Context, sandboxID uuid.UUID) (*types.DiffArchive, error) {
	if sandboxID == uuid.Nil {
		return nil, errors.New("archive_repo: sandbox_id required")
	}
	row := r.db.QueryRowContext(ctx,
		"SELECT "+archiveColumns+" FROM sandbox_diff_archives WHERE sandbox_id = ?",
		uuidText(sandboxID),
	)
	a, err := scanArchive(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("archive_repo: get: %w", err)
	}
	return a, nil
}

// List runs the filtered/sorted/paginated listing query and returns
// the matching slice plus the total matching count (pre-pagination).
func (r *SandboxArchiveRepository) List(ctx context.Context, filter types.ArchiveListFilter) ([]*types.DiffArchive, int, error) {
	whereSQL, whereArgs, err := buildArchiveWhere(filter)
	if err != nil {
		return nil, 0, err
	}

	orderSQL, err := buildArchiveOrder(filter)
	if err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = defaultArchiveListLimit
	}
	if limit > maxArchiveListLimit {
		limit = maxArchiveListLimit
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	// Total count (without limit/offset) for pagination metadata.
	var total int
	countQuery := "SELECT COUNT(*) FROM sandbox_diff_archives"
	if whereSQL != "" {
		countQuery += " WHERE " + whereSQL
	}
	if err := r.db.QueryRowContext(ctx, countQuery, whereArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("archive_repo: count: %w", err)
	}

	dataQuery := "SELECT " + archiveColumns + " FROM sandbox_diff_archives"
	if whereSQL != "" {
		dataQuery += " WHERE " + whereSQL
	}
	dataQuery += " " + orderSQL + " LIMIT ? OFFSET ?"

	args := append([]any{}, whereArgs...)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("archive_repo: list: %w", err)
	}
	defer rows.Close()

	out := []*types.DiffArchive{}
	for rows.Next() {
		a, err := scanArchive(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("archive_repo: list scan: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("archive_repo: list iterate: %w", err)
	}
	return out, total, nil
}

// Delete removes one archive row by primary key. Returns nil whether
// the row existed or not — the caller's intent is "ensure absent."
func (r *SandboxArchiveRepository) Delete(ctx context.Context, sandboxID uuid.UUID) error {
	if sandboxID == uuid.Nil {
		return errors.New("archive_repo: sandbox_id required")
	}
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM sandbox_diff_archives WHERE sandbox_id = ?",
		uuidText(sandboxID),
	)
	if err != nil {
		return fmt.Errorf("archive_repo: delete: %w", err)
	}
	return nil
}

// SumSizeBytes sums TotalBlobBytes across archives matching projectRoot.
// Empty projectRoot sums all rows.
func (r *SandboxArchiveRepository) SumSizeBytes(ctx context.Context, projectRoot string) (int64, error) {
	var total sql.NullInt64
	var row *sql.Row
	if projectRoot == "" {
		row = r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(total_blob_bytes), 0) FROM sandbox_diff_archives`)
	} else {
		row = r.db.QueryRowContext(ctx,
			`SELECT COALESCE(SUM(total_blob_bytes), 0) FROM sandbox_diff_archives WHERE project_root = ?`,
			projectRoot,
		)
	}
	if err := row.Scan(&total); err != nil {
		return 0, fmt.Errorf("archive_repo: sum size: %w", err)
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Int64, nil
}

// AllOrdered returns every archive in oldest-first order.
func (r *SandboxArchiveRepository) AllOrdered(ctx context.Context) ([]*types.DiffArchive, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+archiveColumns+" FROM sandbox_diff_archives ORDER BY snapshot_at ASC, sandbox_id ASC",
	)
	if err != nil {
		return nil, fmt.Errorf("archive_repo: all ordered: %w", err)
	}
	defer rows.Close()

	out := []*types.DiffArchive{}
	for rows.Next() {
		a, err := scanArchive(rows)
		if err != nil {
			return nil, fmt.Errorf("archive_repo: all ordered scan: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("archive_repo: all ordered iterate: %w", err)
	}
	return out, nil
}

// OldestN returns up to n archives ordered by snapshot_at ascending.
// Returns the empty slice when there are no archives.
func (r *SandboxArchiveRepository) OldestN(ctx context.Context, n int) ([]*types.DiffArchive, error) {
	if n <= 0 {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+archiveColumns+" FROM sandbox_diff_archives ORDER BY snapshot_at ASC LIMIT ?",
		n,
	)
	if err != nil {
		return nil, fmt.Errorf("archive_repo: oldest: %w", err)
	}
	defer rows.Close()

	out := []*types.DiffArchive{}
	for rows.Next() {
		a, err := scanArchive(rows)
		if err != nil {
			return nil, fmt.Errorf("archive_repo: oldest scan: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("archive_repo: oldest iterate: %w", err)
	}
	return out, nil
}

const (
	defaultArchiveListLimit = 100
	maxArchiveListLimit     = 1000
)

// buildArchiveWhere assembles the WHERE clause for List from filter.
// Returns SQL fragment (without "WHERE" prefix) plus the args slice.
func buildArchiveWhere(f types.ArchiveListFilter) (string, []any, error) {
	parts := []string{}
	args := []any{}

	if len(f.Statuses) > 0 {
		// Validate every status is one of the three terminal states.
		placeholders := make([]string, 0, len(f.Statuses))
		for _, s := range f.Statuses {
			switch s {
			case types.StatusApproved, types.StatusRejected, types.StatusDeleted:
				// ok
			default:
				return "", nil, fmt.Errorf("archive_repo: invalid status filter %q (allowed: approved, rejected, deleted)", s)
			}
			placeholders = append(placeholders, "?")
			args = append(args, string(s))
		}
		parts = append(parts, "sandbox_status IN ("+strings.Join(placeholders, ",")+")")
	}

	if f.ProjectRoot != "" {
		parts = append(parts, "project_root = ?")
		args = append(args, f.ProjectRoot)
	}
	if f.Owner != "" {
		parts = append(parts, "owner = ?")
		args = append(args, f.Owner)
	}
	if f.AgentManagerRunID != "" {
		parts = append(parts, "agent_manager_run_id = ?")
		args = append(args, f.AgentManagerRunID)
	}
	if f.Search != "" {
		// LIKE escape: literal _, %, and \ are escaped with \. We
		// build the SQL with ESCAPE '\' so the pattern is unambiguous.
		needle := "%" + escapeLike(f.Search) + "%"
		parts = append(parts,
			`(COALESCE(owner, '') LIKE ? ESCAPE '\' OR COALESCE(agent_manager_run_id, '') LIKE ? ESCAPE '\' OR sandbox_id LIKE ? ESCAPE '\')`,
		)
		args = append(args, needle, needle, needle)
	}
	if !f.SnapshotAtFrom.IsZero() {
		parts = append(parts, "snapshot_at >= ?")
		args = append(args, formatTime(f.SnapshotAtFrom))
	}
	if !f.SnapshotAtTo.IsZero() {
		parts = append(parts, "snapshot_at <= ?")
		args = append(args, formatTime(f.SnapshotAtTo))
	}

	return strings.Join(parts, " AND "), args, nil
}

// buildArchiveOrder validates SortBy and constructs the ORDER BY clause.
func buildArchiveOrder(f types.ArchiveListFilter) (string, error) {
	var (
		col  string
		desc bool
	)
	switch f.SortBy {
	case "":
		// Default: newest first by snapshot_at, ignoring SortDesc.
		col = "snapshot_at"
		desc = true
	case "snapshot_at":
		col = "snapshot_at"
		desc = f.SortDesc
	case "total_blob_bytes":
		col = "total_blob_bytes"
		desc = f.SortDesc
	default:
		return "", fmt.Errorf("archive_repo: invalid sort_by %q (allowed: snapshot_at, total_blob_bytes)", f.SortBy)
	}
	dir := "DESC"
	if !desc {
		dir = "ASC"
	}
	// Tie-break by sandbox_id for deterministic order across runs.
	return fmt.Sprintf("ORDER BY %s %s, sandbox_id ASC", col, dir), nil
}

// escapeLike escapes the SQLite LIKE wildcards (_, %) and the escape
// character (\) itself. The query uses ESCAPE '\' to make this work.
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// scanArchive reads one row into a *types.DiffArchive.
func scanArchive(row interface {
	Scan(...any) error
},
) (*types.DiffArchive, error) {
	var (
		a              types.DiffArchive
		idStr          string
		snapshotAt     string
		archiveState   string
		filesJSON      string
		statsJSON      string
		unifiedDiff    string
		totalBlobBytes int64
		projectRoot    string
		owner          string
		runID          string
		sandboxStatus  string
	)
	if err := row.Scan(
		&idStr, &snapshotAt, &archiveState, &filesJSON, &statsJSON,
		&unifiedDiff, &totalBlobBytes, &projectRoot, &owner, &runID, &sandboxStatus,
	); err != nil {
		return nil, err
	}

	id, err := parseUUID(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse archive id: %w", err)
	}
	a.SandboxID = id

	t, err := parseTime(snapshotAt)
	if err != nil {
		return nil, fmt.Errorf("parse snapshot_at: %w", err)
	}
	a.SnapshotAt = t

	a.ArchiveState = types.ArchiveState(archiveState)
	a.SandboxStatus = types.Status(sandboxStatus)
	a.UnifiedDiffSHA256 = unifiedDiff
	a.TotalBlobBytes = totalBlobBytes
	a.ProjectRoot = projectRoot
	a.Owner = owner
	a.AgentManagerRunID = runID

	if err := json.Unmarshal([]byte(filesJSON), &a.Files); err != nil {
		return nil, fmt.Errorf("parse files_json: %w", err)
	}
	if err := json.Unmarshal([]byte(statsJSON), &a.Stats); err != nil {
		return nil, fmt.Errorf("parse stats_json: %w", err)
	}
	return &a, nil
}

// Time helper kept package-local to avoid leaking sqlite_codec.go
// internals. The format matches what the rest of the repository writes
// for RFC3339Nano timestamps; if a future migration changes that, both
// helpers should change together.
var _ = time.RFC3339Nano // documents the format contract
