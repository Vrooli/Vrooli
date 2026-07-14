package adoptions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"react-component-library/internal/clock"

	"github.com/google/uuid"
)

// sqliteRepository is the production Repository impl. Unexported so
// callers depend on the Repository interface — tests substitute the
// fake without reaching inside the struct.
type sqliteRepository struct {
	db    *sql.DB
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db *sql.DB, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

const timeFormat = time.RFC3339Nano

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
	if _, err := s.db.ExecContext(ctx, `
INSERT INTO adoption_records
  (id, component_id, library_id, scenario, adopted_path, adopted_version, source_sha256, adopted_snapshot_sha256, library_version_status, local_status, status_detail, created_at, refreshed_at, applied_at, drift_backlog_ref)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?, '', ?, '')
`, id, in.ComponentID, in.LibraryID, in.Scenario, in.AdoptedPath, in.AdoptedVersion, in.SourceSHA256, in.AdoptedSnapshotSHA256,
		string(LibraryVersionStatusCurrent), string(LocalStatusClean), now.Format(timeFormat), now.Format(timeFormat)); err != nil {
		return Adoption{}, fmt.Errorf("insert adoption: %w", err)
	}
	for _, file := range in.Files {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO adoption_files (adoption_id, library_path, adopted_path, source_sha256, adopted_snapshot_sha256) VALUES (?, ?, ?, ?, ?)`, id, file.LibraryPath, file.AdoptedPath, file.SourceSHA256, file.AdoptedSnapshotSHA256); err != nil {
			return Adoption{}, fmt.Errorf("insert adoption file %q: %w", file.AdoptedPath, err)
		}
	}
	return s.Get(ctx, id)
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
		if _, err := s.db.ExecContext(ctx, `INSERT INTO adoption_files (adoption_id, library_path, adopted_path, source_sha256, adopted_snapshot_sha256) VALUES (?, ?, ?, ?, ?)`, in.ID, file.LibraryPath, file.AdoptedPath, file.SourceSHA256, file.AdoptedSnapshotSHA256); err != nil {
			return Adoption{}, fmt.Errorf("insert reapplied adoption file: %w", err)
		}
	}
	updated.Files, err = s.listFiles(ctx, in.ID)
	return updated, err
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
SELECT id, component_id, library_id, scenario, adopted_path, adopted_version, source_sha256, adopted_snapshot_sha256, library_version_status, local_status, status_detail, created_at, refreshed_at, applied_at, drift_backlog_ref
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

func (s *sqliteRepository) listFiles(ctx context.Context, adoptionID string) ([]AdoptionFile, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT library_path, adopted_path, source_sha256, adopted_snapshot_sha256 FROM adoption_files WHERE adoption_id = ? ORDER BY library_path`, adoptionID)
	if err != nil {
		return nil, fmt.Errorf("list adoption files: %w", err)
	}
	defer rows.Close()
	var files []AdoptionFile
	for rows.Next() {
		var file AdoptionFile
		if err := rows.Scan(&file.LibraryPath, &file.AdoptedPath, &file.SourceSHA256, &file.AdoptedSnapshotSHA256); err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

func (s *sqliteRepository) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM adoption_records WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete adoption %q: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrAdoptionNotFound{ID: id}
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
SELECT id, component_id, library_id, scenario, adopted_path, adopted_version, source_sha256, adopted_snapshot_sha256, library_version_status, local_status, status_detail, created_at, refreshed_at, applied_at, drift_backlog_ref
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
	)
	if err := s.Scan(&a.ID, &a.ComponentID, &a.LibraryID, &a.Scenario, &a.AdoptedPath, &a.AdoptedVersion,
		&a.SourceSHA256, &a.AdoptedSnapshotSHA256, &libraryStatusRaw, &localStatusRaw, &a.StatusDetail, &createdRaw, &refreshedRaw, &appliedRaw, &a.DriftBacklogRef); err != nil {
		return Adoption{}, err
	}
	a.LibraryVersionStatus = LibraryVersionStatus(libraryStatusRaw)
	a.LocalStatus = LocalStatus(localStatusRaw)
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
