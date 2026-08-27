package adoptions

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// sqliteRepository is the production Repository impl. Unexported so
// callers depend on the Repository interface — tests substitute the
// fake without reaching inside the struct.
type sqliteRepository struct {
	db    *sql.DB
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db *sql.DB, clk schedule.Clock) Repository {
	// Existing scenario databases predate adoption modes. SQLite does not
	// provide a portable ADD COLUMN IF NOT EXISTS, so the one-time migration
	// is intentionally idempotent by accepting the duplicate-column result.
	_, _ = db.Exec(`ALTER TABLE adoption_records ADD COLUMN mode TEXT NOT NULL DEFAULT 'copied'`)
	// adoption_files is a soft-owned child table. Older Delete implementations
	// removed only the parent row, leaving provenance ghosts that could keep an
	// otherwise unused library version permanently materialized. Reconcile that
	// historical state at construction while the transactional Delete below
	// prevents it from returning.
	_, _ = db.Exec(`DELETE FROM adoption_files WHERE NOT EXISTS (SELECT 1 FROM adoption_records WHERE adoption_records.id = adoption_files.adoption_id)`)
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

const timeFormat = time.RFC3339Nano

func forkStatus(reason string) ForkStatus {
	if strings.TrimSpace(reason) == "" {
		return ForkStatusNone
	}
	return ForkStatusDeclared
}

func (s *sqliteRepository) Create(ctx context.Context, in CreateInput) (Adoption, error) {
	if strings.TrimSpace(in.ComponentID) == "" {
		return Adoption{}, ErrInvalidAdoption{Field: "component_id", Reason: "required"}
	}
	if strings.TrimSpace(in.Scenario) == "" {
		return Adoption{}, ErrInvalidAdoption{Field: "scenario", Reason: "required"}
	}
	if strings.TrimSpace(in.AdoptedPath) == "" {
		return Adoption{}, ErrInvalidAdoption{Field: "adopted_path", Reason: "required"}
	}
	now := s.clock.Now().UTC()
	id := strings.TrimSpace(in.ID)
	if id == "" {
		id = uuid.NewString()
	}
	suggestions, err := json.Marshal(in.IncludeSuggestions)
	if err != nil {
		return Adoption{}, fmt.Errorf("encode adoption suggestions: %w", err)
	}
	extensions, err := json.Marshal(in.ExtensionPoints)
	if err != nil {
		return Adoption{}, fmt.Errorf("encode adoption extension points: %w", err)
	}
	mode := in.Mode
	if mode == "" {
		mode = AdoptionModeCopied
	}
	if !mode.Valid() {
		return Adoption{}, ErrInvalidAdoption{Field: "mode", Reason: "must be copied, linked, or ejected"}
	}
	if _, err := s.db.ExecContext(ctx, `
INSERT INTO adoption_records
  (id, component_id, library_id, scenario, adopted_path, adopted_version, source_sha256, adopted_snapshot_sha256, library_version_status, local_status, status_detail, created_at, refreshed_at, applied_at, drift_backlog_ref, suggested_dependencies, fork_status, fork_reason, extension_points, mode)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?, '', ?, '', ?, ?, ?, ?, ?)
`, id, in.ComponentID, in.LibraryID, in.Scenario, in.AdoptedPath, in.AdoptedVersion, in.SourceSHA256, in.AdoptedSnapshotSHA256,
		string(LibraryVersionStatusCurrent), string(LocalStatusClean), now.Format(timeFormat), now.Format(timeFormat), string(suggestions), forkStatus(in.ForkReason), strings.TrimSpace(in.ForkReason), string(extensions), string(mode)); err != nil {
		return Adoption{}, fmt.Errorf("insert adoption: %w", err)
	}
	for _, file := range in.Files {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO adoption_files (adoption_id, library_path, adopted_path, source_sha256, adopted_snapshot_sha256, source_asset_id, source_library_id, source_version) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, id, file.LibraryPath, file.AdoptedPath, file.SourceSHA256, file.AdoptedSnapshotSHA256, file.SourceAssetID, file.SourceLibraryID, file.SourceVersion); err != nil {
			return Adoption{}, fmt.Errorf("insert adoption file %q: %w", file.AdoptedPath, err)
		}
	}
	return s.Get(ctx, id)
}

// CreateBatch persists all adoption rows and their per-file provenance in one
// SQLite transaction. Scenario files are written by the service before this
// boundary and are rolled back there if this transaction fails.
func (s *sqliteRepository) CreateBatch(ctx context.Context, inputs []CreateInput) ([]Adoption, error) {
	if len(inputs) == 0 {
		return nil, ErrInvalidAdoption{Field: "inputs", Reason: "at least one adoption is required"}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin batch adoption transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now := s.clock.Now().UTC().Format(timeFormat)
	ids := make([]string, 0, len(inputs))
	for _, in := range inputs {
		if strings.TrimSpace(in.ComponentID) == "" {
			return nil, ErrInvalidAdoption{Field: "component_id", Reason: "required"}
		}
		if strings.TrimSpace(in.Scenario) == "" {
			return nil, ErrInvalidAdoption{Field: "scenario", Reason: "required"}
		}
		if strings.TrimSpace(in.AdoptedPath) == "" {
			return nil, ErrInvalidAdoption{Field: "adopted_path", Reason: "required"}
		}
		id := strings.TrimSpace(in.ID)
		if id == "" {
			id = uuid.NewString()
		}
		suggestions, err := json.Marshal(in.IncludeSuggestions)
		if err != nil {
			return nil, fmt.Errorf("encode batch adoption suggestions: %w", err)
		}
		extensions, err := json.Marshal(in.ExtensionPoints)
		if err != nil {
			return nil, fmt.Errorf("encode batch adoption extension points: %w", err)
		}
		mode := in.Mode
		if mode == "" {
			mode = AdoptionModeCopied
		}
		if !mode.Valid() {
			return nil, ErrInvalidAdoption{Field: "mode", Reason: "must be copied, linked, or ejected"}
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO adoption_records
  (id, component_id, library_id, scenario, adopted_path, adopted_version, source_sha256, adopted_snapshot_sha256, library_version_status, local_status, status_detail, created_at, refreshed_at, applied_at, drift_backlog_ref, suggested_dependencies, fork_status, fork_reason, extension_points, mode)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?, '', ?, '', ?, ?, ?, ?, ?)
`, id, in.ComponentID, in.LibraryID, in.Scenario, in.AdoptedPath, in.AdoptedVersion, in.SourceSHA256, in.AdoptedSnapshotSHA256,
			string(LibraryVersionStatusCurrent), string(LocalStatusClean), now, now, string(suggestions), forkStatus(in.ForkReason), strings.TrimSpace(in.ForkReason), string(extensions), string(mode)); err != nil {
			return nil, fmt.Errorf("insert batch adoption %q: %w", id, err)
		}
		for _, file := range in.Files {
			if _, err := tx.ExecContext(ctx, `INSERT INTO adoption_files (adoption_id, library_path, adopted_path, source_sha256, adopted_snapshot_sha256, source_asset_id, source_library_id, source_version) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, id, file.LibraryPath, file.AdoptedPath, file.SourceSHA256, file.AdoptedSnapshotSHA256, file.SourceAssetID, file.SourceLibraryID, file.SourceVersion); err != nil {
				return nil, fmt.Errorf("insert batch adoption file %q: %w", file.AdoptedPath, err)
			}
		}
		ids = append(ids, id)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit batch adoptions: %w", err)
	}
	created := make([]Adoption, 0, len(ids))
	for _, id := range ids {
		adoption, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		created = append(created, adoption)
	}
	return created, nil
}

func (s *sqliteRepository) UpdateAppliedSnapshot(ctx context.Context, in AppliedSnapshotUpdate) (Adoption, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return Adoption{}, ErrInvalidAdoption{Field: "id", Reason: "required"}
	}
	appliedAt := in.AppliedAt.UTC()
	if appliedAt.IsZero() {
		appliedAt = s.clock.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `
UPDATE adoption_records
SET adopted_version = ?,
    source_sha256 = ?,
    adopted_snapshot_sha256 = ?,
    library_version_status = ?,
    local_status = ?,
    status_detail = '',
    refreshed_at = ?,
    applied_at = ?,
    drift_backlog_ref = ''
WHERE id = ?
`, in.AdoptedVersion, in.SourceSHA256, in.AdoptedSnapshotSHA256, string(LibraryVersionStatusCurrent), string(LocalStatusClean),
		appliedAt.Format(timeFormat), appliedAt.Format(timeFormat), id)
	if err != nil {
		return Adoption{}, fmt.Errorf("update applied snapshot %q: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return Adoption{}, ErrAdoptionNotFound{ID: id}
	}
	return s.Get(ctx, id)
}

func (s *sqliteRepository) UpdateAppliedUnit(ctx context.Context, in AppliedUnitUpdate) (Adoption, error) {
	updated, err := s.UpdateAppliedSnapshot(ctx, in.AppliedSnapshotUpdate)
	if err != nil {
		return Adoption{}, err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM adoption_files WHERE adoption_id = ?`, in.ID); err != nil {
		return Adoption{}, fmt.Errorf("clear adoption files: %w", err)
	}
	for _, file := range in.Files {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO adoption_files (adoption_id, library_path, adopted_path, source_sha256, adopted_snapshot_sha256, source_asset_id, source_library_id, source_version) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, in.ID, file.LibraryPath, file.AdoptedPath, file.SourceSHA256, file.AdoptedSnapshotSHA256, file.SourceAssetID, file.SourceLibraryID, file.SourceVersion); err != nil {
			return Adoption{}, fmt.Errorf("insert reapplied adoption file: %w", err)
		}
	}
	updated.Files, err = s.listFiles(ctx, in.ID)
	return updated, err
}

func (s *sqliteRepository) Rebaseline(ctx context.Context, in RebaselineInput) (Adoption, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return Adoption{}, ErrInvalidAdoption{Field: "id", Reason: "required"}
	}
	res, err := s.db.ExecContext(ctx, `UPDATE adoption_records SET adopted_snapshot_sha256 = ? WHERE id = ?`, in.AdoptedSnapshotSHA256, id)
	if err != nil {
		return Adoption{}, fmt.Errorf("rebaseline snapshot %q: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return Adoption{}, ErrAdoptionNotFound{ID: id}
	}
	if in.Files != nil {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM adoption_files WHERE adoption_id = ?`, id); err != nil {
			return Adoption{}, fmt.Errorf("clear adoption files: %w", err)
		}
		for _, file := range in.Files {
			if _, err := s.db.ExecContext(ctx, `INSERT INTO adoption_files (adoption_id, library_path, adopted_path, source_sha256, adopted_snapshot_sha256, source_asset_id, source_library_id, source_version) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, id, file.LibraryPath, file.AdoptedPath, file.SourceSHA256, file.AdoptedSnapshotSHA256, file.SourceAssetID, file.SourceLibraryID, file.SourceVersion); err != nil {
				return Adoption{}, fmt.Errorf("insert rebaselined adoption file: %w", err)
			}
		}
	}
	return s.Get(ctx, id)
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Adoption, error) {
	row := s.db.QueryRowContext(ctx, selectAdoptionByIDSQL, id)
	a, err := scanAdoption(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Adoption{}, ErrAdoptionNotFound{ID: id}
	}
	if err != nil {
		return Adoption{}, fmt.Errorf("get adoption %q: %w", id, err)
	}
	a.Files, err = s.listFiles(ctx, a.ID)
	if err != nil {
		return Adoption{}, err
	}
	return a, nil
}

func (s *sqliteRepository) List(ctx context.Context, q ListQuery) ([]Adoption, error) {
	limit := q.Limit
	if limit <= 0 {
		return nil, nil
	}
	var (
		clauses []string
		args    []any
	)
	if cid := strings.TrimSpace(q.ComponentID); cid != "" {
		clauses = append(clauses, "component_id = ?")
		args = append(args, cid)
	}
	if sc := strings.TrimSpace(q.Scenario); sc != "" {
		clauses = append(clauses, "scenario = ?")
		args = append(args, sc)
	}
	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
SELECT id, component_id, library_id, scenario, adopted_path, adopted_version, source_sha256, adopted_snapshot_sha256, library_version_status, local_status, status_detail, created_at, refreshed_at, applied_at, drift_backlog_ref, suggested_dependencies, fork_status, fork_reason, extension_points, mode
FROM adoption_records
%s
ORDER BY created_at DESC, id ASC
LIMIT ?
`, where), args...)
	if err != nil {
		return nil, fmt.Errorf("list adoptions: %w", err)
	}
	defer rows.Close()
	var out []Adoption
	for rows.Next() {
		a, err := scanAdoption(rows)
		if err != nil {
			return nil, fmt.Errorf("scan adoption: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate adoptions: %w", err)
	}
	// Close the parent cursor before child lookups. SQLite's single-connection
	// in-memory test database cannot service a nested query while rows is open.
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close adoptions rows: %w", err)
	}
	for i := range out {
		files, err := s.listFiles(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Files = files
	}
	return out, nil
}

func (s *sqliteRepository) ListEffective(ctx context.Context, componentID string, limit int) ([]EffectiveAdoption, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT a.id, a.component_id, a.library_id, a.scenario, a.adopted_path, a.adopted_version, a.source_sha256, a.adopted_snapshot_sha256, a.library_version_status, a.local_status, a.status_detail, a.created_at, a.refreshed_at, a.applied_at, a.drift_backlog_ref,
       f.source_asset_id, f.source_library_id, f.source_version
FROM adoption_files f JOIN adoption_records a ON a.id = f.adoption_id
WHERE f.source_asset_id = ?
ORDER BY a.created_at DESC, a.id ASC, f.adopted_path ASC
LIMIT ?`, componentID, limit)
	if err != nil {
		return nil, fmt.Errorf("list effective adoptions: %w", err)
	}
	defer rows.Close()
	out := make([]EffectiveAdoption, 0)
	for rows.Next() {
		var effective EffectiveAdoption
		if err := rows.Scan(&effective.ParentAdoption.ID, &effective.ParentAdoption.ComponentID, &effective.ParentAdoption.LibraryID, &effective.ParentAdoption.Scenario, &effective.ParentAdoption.AdoptedPath, &effective.ParentAdoption.AdoptedVersion, &effective.ParentAdoption.SourceSHA256, &effective.ParentAdoption.AdoptedSnapshotSHA256, new(string), new(string), &effective.ParentAdoption.StatusDetail, new(string), new(string), new(string), &effective.ParentAdoption.DriftBacklogRef, &effective.SourceAssetID, &effective.SourceLibraryID, &effective.SourceVersion); err != nil {
			return nil, fmt.Errorf("scan effective adoption: %w", err)
		}
		effective.Mediated = effective.ParentAdoption.ComponentID != effective.SourceAssetID
		out = append(out, effective)
	}
	return out, rows.Err()
}

func (s *sqliteRepository) listFiles(ctx context.Context, adoptionID string) ([]AdoptionFile, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT library_path, adopted_path, source_sha256, adopted_snapshot_sha256, source_asset_id, source_library_id, source_version FROM adoption_files WHERE adoption_id = ? ORDER BY library_path`, adoptionID)
	if err != nil {
		return nil, fmt.Errorf("list adoption files: %w", err)
	}
	defer rows.Close()
	var files []AdoptionFile
	for rows.Next() {
		var file AdoptionFile
		if err := rows.Scan(&file.LibraryPath, &file.AdoptedPath, &file.SourceSHA256, &file.AdoptedSnapshotSHA256, &file.SourceAssetID, &file.SourceLibraryID, &file.SourceVersion); err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

func (s *sqliteRepository) Delete(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin adoption delete %q: %w", id, err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM adoption_files WHERE adoption_id = ?`, id); err != nil {
		return fmt.Errorf("delete adoption files %q: %w", id, err)
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM adoption_records WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete adoption %q: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrAdoptionNotFound{ID: id}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit adoption delete %q: %w", id, err)
	}
	return nil
}

func (s *sqliteRepository) ApplyRefresh(ctx context.Context, updates []RefreshUpdate) (int, error) {
	if len(updates) == 0 {
		return 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	touched := 0
	for _, u := range updates {
		res, err := tx.ExecContext(ctx, `
UPDATE adoption_records SET library_version_status = ?, local_status = ?, status_detail = ?, refreshed_at = ?
WHERE id = ?
`, string(u.LibraryVersionStatus), string(u.LocalStatus), u.StatusDetail, u.RefreshedAt.UTC().Format(timeFormat), u.ID)
		if err != nil {
			return touched, fmt.Errorf("apply refresh for %q: %w", u.ID, err)
		}
		n, _ := res.RowsAffected()
		touched += int(n)
		if u.ForkStatus != ForkStatusNone {
			if _, err := tx.ExecContext(ctx, `UPDATE adoption_records SET fork_status = ? WHERE id = ?`, string(u.ForkStatus), u.ID); err != nil {
				return touched, fmt.Errorf("set fork status for %q: %w", u.ID, err)
			}
		}
		// drift_backlog_ref updates ride alongside the row update so we
		// stay in a single transaction. ClearDriftBacklogRef wins over a
		// non-empty DriftBacklogRef so callers can be explicit either way.
		switch {
		case u.ClearDriftBacklogRef:
			if _, err := tx.ExecContext(ctx, `UPDATE adoption_records SET drift_backlog_ref = '' WHERE id = ?`, u.ID); err != nil {
				return touched, fmt.Errorf("clear drift ref for %q: %w", u.ID, err)
			}
		case u.DriftBacklogRef != "":
			if _, err := tx.ExecContext(ctx, `UPDATE adoption_records SET drift_backlog_ref = ? WHERE id = ?`, u.DriftBacklogRef, u.ID); err != nil {
				return touched, fmt.Errorf("set drift ref for %q: %w", u.ID, err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return touched, fmt.Errorf("commit refresh: %w", err)
	}
	return touched, nil
}

const selectAdoptionByIDSQL = `
SELECT id, component_id, library_id, scenario, adopted_path, adopted_version, source_sha256, adopted_snapshot_sha256, library_version_status, local_status, status_detail, created_at, refreshed_at, applied_at, drift_backlog_ref, suggested_dependencies, fork_status, fork_reason, extension_points, mode
FROM adoption_records WHERE id = ?
`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAdoption(s rowScanner) (Adoption, error) {
	var (
		a                Adoption
		libraryStatusRaw string
		localStatusRaw   string
		createdRaw       string
		refreshedRaw     string
		appliedRaw       string
		suggestedRaw     string
		forkStatusRaw    string
		extensionsRaw    string
		modeRaw          string
	)
	if err := s.Scan(&a.ID, &a.ComponentID, &a.LibraryID, &a.Scenario, &a.AdoptedPath, &a.AdoptedVersion,
		&a.SourceSHA256, &a.AdoptedSnapshotSHA256, &libraryStatusRaw, &localStatusRaw, &a.StatusDetail, &createdRaw, &refreshedRaw, &appliedRaw, &a.DriftBacklogRef, &suggestedRaw, &forkStatusRaw, &a.ForkReason, &extensionsRaw, &modeRaw); err != nil {
		return Adoption{}, err
	}
	if suggestedRaw != "" {
		if err := json.Unmarshal([]byte(suggestedRaw), &a.IncludeSuggestions); err != nil {
			return Adoption{}, fmt.Errorf("parse suggested dependencies: %w", err)
		}
	}
	if extensionsRaw != "" {
		if err := json.Unmarshal([]byte(extensionsRaw), &a.ExtensionPoints); err != nil {
			return Adoption{}, fmt.Errorf("parse extension points: %w", err)
		}
	}
	a.LibraryVersionStatus = LibraryVersionStatus(libraryStatusRaw)
	a.LocalStatus = LocalStatus(localStatusRaw)
	a.ForkStatus = ForkStatus(forkStatusRaw)
	a.Mode = AdoptionMode(modeRaw)
	if a.Mode == "" {
		a.Mode = AdoptionModeCopied
	}
	created, err := time.Parse(timeFormat, createdRaw)
	if err != nil {
		return Adoption{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	a.CreatedAt = created
	if refreshedRaw != "" {
		ref, err := time.Parse(timeFormat, refreshedRaw)
		if err != nil {
			return Adoption{}, fmt.Errorf("parse refreshed_at %q: %w", refreshedRaw, err)
		}
		a.RefreshedAt = ref
	}
	if appliedRaw != "" {
		applied, err := time.Parse(timeFormat, appliedRaw)
		if err != nil {
			return Adoption{}, fmt.Errorf("parse applied_at %q: %w", appliedRaw, err)
		}
		a.AppliedAt = applied
	}
	return a, nil
}

func (s *sqliteRepository) UpdateMode(ctx context.Context, id string, mode AdoptionMode, reason string) (Adoption, error) {
	if !mode.Valid() {
		return Adoption{}, ErrInvalidAdoption{Field: "mode", Reason: "must be copied, linked, or ejected"}
	}
	if mode == AdoptionModeEjected && strings.TrimSpace(reason) == "" {
		return Adoption{}, ErrInvalidAdoption{Field: "reason", Reason: "required for ejected adoption"}
	}
	res, err := s.db.ExecContext(ctx, `UPDATE adoption_records SET mode = ?, fork_reason = ?, fork_status = ? WHERE id = ?`, string(mode), strings.TrimSpace(reason), forkStatus(reason), id)
	if err != nil {
		return Adoption{}, fmt.Errorf("update adoption mode %q: %w", id, err)
	}
	count, _ := res.RowsAffected()
	if count == 0 {
		return Adoption{}, ErrAdoptionNotFound{ID: id}
	}
	return s.Get(ctx, id)
}

func (s *sqliteRepository) UpdateLinked(ctx context.Context, id, adoptedPath, adoptedVersion, sourceSHA256 string) (Adoption, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE adoption_records SET adopted_path = ?, adopted_version = ?, source_sha256 = ?, adopted_snapshot_sha256 = '', local_status = ?, library_version_status = ?, mode = ?, fork_status = '', fork_reason = '' WHERE id = ?`, adoptedPath, adoptedVersion, sourceSHA256, string(LocalStatusClean), string(LibraryVersionStatusCurrent), string(AdoptionModeLinked), id)
	if err != nil {
		return Adoption{}, fmt.Errorf("update linked adoption %q: %w", id, err)
	}
	count, _ := res.RowsAffected()
	if count == 0 {
		return Adoption{}, ErrAdoptionNotFound{ID: id}
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM adoption_files WHERE adoption_id = ?`, id); err != nil {
		return Adoption{}, fmt.Errorf("clear copied files for linked adoption %q: %w", id, err)
	}
	return s.Get(ctx, id)
}
