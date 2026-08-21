package catalog

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrGenerationNotFound = errors.New("catalog generation not found")
	ErrNoActiveGeneration = errors.New("catalog has no active generation")
)

type Migration struct {
	Version int
	SQL     string
}

func Migrate(ctx context.Context, db *sql.DB) error {
	return ApplyMigrations(ctx, db, []Migration{
		{Version: 1, SQL: Schema()},
		{Version: 2, SQL: SearchSchema()},
		{Version: 3, SQL: IndexControlSchema()},
		{Version: 4, SQL: MetricsSchema()},
	})
}

// ApplyMigrations applies each migration and its ledger row in one transaction.
// A failed statement rolls back the schema and version marker together.
func ApplyMigrations(ctx context.Context, db *sql.DB, migrations []Migration) error {
	if db == nil {
		return fmt.Errorf("catalog migrations require database")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin catalog migration: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS code_facts_catalog_migrations (version INTEGER PRIMARY KEY, applied_at_unix INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("create catalog migration ledger: %w", err)
	}
	for _, migration := range migrations {
		if migration.Version <= 0 || strings.TrimSpace(migration.SQL) == "" {
			return fmt.Errorf("invalid catalog migration version %d", migration.Version)
		}
		var applied int
		err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM code_facts_catalog_migrations WHERE version = ?`, migration.Version).Scan(&applied)
		if err != nil {
			return fmt.Errorf("read catalog migration %d: %w", migration.Version, err)
		}
		if applied > 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
			return fmt.Errorf("apply catalog migration %d: %w", migration.Version, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO code_facts_catalog_migrations(version, applied_at_unix) VALUES(?, unixepoch())`, migration.Version); err != nil {
			return fmt.Errorf("record catalog migration %d: %w", migration.Version, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit catalog migrations: %w", err)
	}
	return nil
}

type SQLiteRepository struct {
	db    *sql.DB
	clock Clock
}

func NewSQLiteRepository(db *sql.DB, clock Clock) *SQLiteRepository {
	return &SQLiteRepository{db: db, clock: clock}
}

func (r *SQLiteRepository) BeginGeneration(ctx context.Context, generation Generation) error {
	if err := r.ready(); err != nil {
		return err
	}
	if strings.TrimSpace(generation.ID) == "" || strings.TrimSpace(generation.Policy) == "" {
		return fmt.Errorf("catalog generation requires id and policy")
	}
	state := generation.State
	if state == "" {
		state = GenerationShadow
	}
	if state != GenerationShadow {
		return fmt.Errorf("new catalog generation state must be %q", GenerationShadow)
	}
	created := generation.CreatedAt.UTC()
	if created.IsZero() {
		created = r.clock.Now().UTC()
	}
	updated := generation.UpdatedAt.UTC()
	if updated.IsZero() {
		updated = created
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO code_facts_generations(
  id, state, policy, source_digest, descriptor_digest, created_at_unix, updated_at_unix, failure
) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`, generation.ID, state, generation.Policy, generation.SourceDigest, generation.DescriptorDigest, created.Unix(), updated.Unix(), generation.Failure)
	if err != nil {
		return fmt.Errorf("begin catalog generation %q: %w", generation.ID, err)
	}
	return nil
}

func (r *SQLiteRepository) UpsertFiles(ctx context.Context, generationID string, files []SourceFile) error {
	if err := r.ready(); err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin catalog file upsert: %w", err)
	}
	defer tx.Rollback()
	if err := requireShadowGeneration(ctx, tx, generationID); err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO code_facts_source_files(
  generation_id, id, path, language, role, scope, authority, owner,
  content_hash, size_bytes, mod_time_unix_nano, searchable
) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(generation_id, id) DO UPDATE SET
  path=excluded.path, language=excluded.language, role=excluded.role,
  scope=excluded.scope, authority=excluded.authority, owner=excluded.owner,
  content_hash=excluded.content_hash, size_bytes=excluded.size_bytes,
  mod_time_unix_nano=excluded.mod_time_unix_nano, searchable=excluded.searchable`)
	if err != nil {
		return fmt.Errorf("prepare catalog file upsert: %w", err)
	}
	defer stmt.Close()
	for _, file := range files {
		if err := validateSourceFile(generationID, file); err != nil {
			return err
		}
		searchable := 0
		if file.Searchable {
			searchable = 1
		}
		if _, err := stmt.ExecContext(ctx,
			generationID, file.ID, file.Path, file.Language, file.Role, file.Scope,
			file.Authority, file.Owner, file.Hash, file.Size, file.ModTime.UnixNano(), searchable,
		); err != nil {
			return fmt.Errorf("upsert catalog file %q: %w", file.Path, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE code_facts_generations SET updated_at_unix = ? WHERE id = ?`, r.clock.Now().UTC().Unix(), generationID); err != nil {
		return fmt.Errorf("touch catalog generation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit catalog file upsert: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) DeleteFiles(ctx context.Context, generationID string, ids []string) error {
	if err := r.ready(); err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin catalog delete: %w", err)
	}
	defer tx.Rollback()
	if err := requireShadowGeneration(ctx, tx, generationID); err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `DELETE FROM code_facts_source_files WHERE generation_id = ? AND id = ?`)
	if err != nil {
		return fmt.Errorf("prepare catalog delete: %w", err)
	}
	defer stmt.Close()
	for _, id := range ids {
		if _, err := stmt.ExecContext(ctx, generationID, id); err != nil {
			return fmt.Errorf("delete catalog file %q: %w", id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit catalog delete: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) CompleteGeneration(ctx context.Context, generationID, sourceDigest, descriptorDigest string) error {
	if err := r.ready(); err != nil {
		return err
	}
	if strings.TrimSpace(sourceDigest) == "" {
		return fmt.Errorf("catalog generation source digest is required")
	}
	result, err := r.db.ExecContext(ctx, `
UPDATE code_facts_generations
SET source_digest = ?, descriptor_digest = ?, updated_at_unix = ?
WHERE id = ? AND state = 'shadow'`, sourceDigest, descriptorDigest, r.clock.Now().UTC().Unix(), generationID)
	if err != nil {
		return fmt.Errorf("complete catalog generation %q: %w", generationID, err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrGenerationNotFound
	}
	return nil
}

func (r *SQLiteRepository) FailGeneration(ctx context.Context, generationID, failure string) error {
	if err := r.ready(); err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `
UPDATE code_facts_generations
SET state = 'failed', failure = ?, updated_at_unix = ?
WHERE id = ? AND state = 'shadow'`, failure, r.clock.Now().UTC().Unix(), generationID)
	if err != nil {
		return fmt.Errorf("fail catalog generation %q: %w", generationID, err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrGenerationNotFound
	}
	return nil
}

func (r *SQLiteRepository) PageFiles(ctx context.Context, generationID, token string, limit int) (Page, error) {
	if err := r.ready(); err != nil {
		return Page{}, err
	}
	if limit <= 0 {
		limit = 256
	}
	if limit > 4096 {
		limit = 4096
	}
	after, err := decodePageToken(token)
	if err != nil {
		return Page{}, err
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, path, language, role, scope, authority, owner, content_hash,
       size_bytes, mod_time_unix_nano, searchable
FROM code_facts_source_files
WHERE generation_id = ? AND path > ?
ORDER BY path
LIMIT ?`, generationID, after, limit+1)
	if err != nil {
		return Page{}, fmt.Errorf("page catalog files: %w", err)
	}
	defer rows.Close()
	files := make([]SourceFile, 0, limit+1)
	for rows.Next() {
		var file SourceFile
		var role string
		var modTime int64
		var searchable int
		if err := rows.Scan(&file.ID, &file.Path, &file.Language, &role, &file.Scope, &file.Authority, &file.Owner, &file.Hash, &file.Size, &modTime, &searchable); err != nil {
			return Page{}, fmt.Errorf("scan catalog file: %w", err)
		}
		file.Generation = generationID
		file.Role = Role(role)
		file.ModTime = time.Unix(0, modTime).UTC()
		file.Searchable = searchable == 1
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return Page{}, fmt.Errorf("iterate catalog files: %w", err)
	}
	page := Page{Files: files}
	if len(files) > limit {
		page.Files = files[:limit]
		page.NextToken = encodePageToken(page.Files[len(page.Files)-1].Path)
	}
	return page, nil
}

func (r *SQLiteRepository) Activate(ctx context.Context, generationID string) error {
	if err := r.ready(); err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin catalog activation: %w", err)
	}
	defer tx.Rollback()
	if err := requireShadowGeneration(ctx, tx, generationID); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM code_facts_source_files WHERE generation_id = ?`, generationID).Scan(&count); err != nil {
		return fmt.Errorf("count catalog generation files: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("catalog generation %q has no source files", generationID)
	}
	now := r.clock.Now().UTC().Unix()
	if _, err := tx.ExecContext(ctx, `UPDATE code_facts_generations SET state = 'retired', updated_at_unix = ? WHERE state = 'active'`, now); err != nil {
		return fmt.Errorf("retire active catalog generation: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE code_facts_generations SET state = 'active', updated_at_unix = ? WHERE id = ? AND state = 'shadow'`, now, generationID)
	if err != nil {
		return fmt.Errorf("activate catalog generation: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrGenerationNotFound
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit catalog activation: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) Active(ctx context.Context) (Generation, error) {
	if err := r.ready(); err != nil {
		return Generation{}, err
	}
	var generation Generation
	var created, updated int64
	err := r.db.QueryRowContext(ctx, `
SELECT id, state, policy, source_digest, descriptor_digest, failure,
       created_at_unix, updated_at_unix
FROM code_facts_generations WHERE state = 'active'`).Scan(
		&generation.ID, &generation.State, &generation.Policy, &generation.SourceDigest,
		&generation.DescriptorDigest, &generation.Failure, &created, &updated,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Generation{}, ErrNoActiveGeneration
	}
	if err != nil {
		return Generation{}, fmt.Errorf("read active catalog generation: %w", err)
	}
	generation.CreatedAt = time.Unix(created, 0).UTC()
	generation.UpdatedAt = time.Unix(updated, 0).UTC()
	return generation, nil
}

func (r *SQLiteRepository) Rollback(ctx context.Context, generationID string) error {
	if err := r.ready(); err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin catalog rollback: %w", err)
	}
	defer tx.Rollback()
	var state string
	if err := tx.QueryRowContext(ctx, `SELECT state FROM code_facts_generations WHERE id=?`, generationID).Scan(&state); err != nil {
		return fmt.Errorf("read rollback generation: %w", err)
	}
	if state != GenerationRetired {
		return fmt.Errorf("rollback generation %q is %s, want retired", generationID, state)
	}
	now := r.clock.Now().UTC().Unix()
	if _, err := tx.ExecContext(ctx, `UPDATE code_facts_generations SET state='retired',updated_at_unix=? WHERE state='active'`, now); err != nil {
		return fmt.Errorf("retire current catalog generation: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE code_facts_generations SET state='active',updated_at_unix=? WHERE id=? AND state='retired'`, now, generationID)
	if err != nil {
		return fmt.Errorf("restore catalog generation: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrGenerationNotFound
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit catalog rollback: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) ready() error {
	if r == nil || r.db == nil || r.clock == nil {
		return fmt.Errorf("catalog repository requires database and clock")
	}
	return nil
}

func requireShadowGeneration(ctx context.Context, tx *sql.Tx, id string) error {
	var state string
	err := tx.QueryRowContext(ctx, `SELECT state FROM code_facts_generations WHERE id = ?`, id).Scan(&state)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrGenerationNotFound
	}
	if err != nil {
		return fmt.Errorf("read catalog generation %q: %w", id, err)
	}
	if state != GenerationShadow {
		return fmt.Errorf("catalog generation %q is %s, want shadow", id, state)
	}
	return nil
}

func validateSourceFile(generationID string, file SourceFile) error {
	if file.Generation != "" && file.Generation != generationID {
		return fmt.Errorf("catalog file %q generation %q does not match %q", file.Path, file.Generation, generationID)
	}
	if strings.TrimSpace(file.ID) == "" || strings.TrimSpace(file.Path) == "" || strings.TrimSpace(file.Hash) == "" {
		return fmt.Errorf("catalog file requires id, path, and content hash")
	}
	if file.Size < 0 {
		return fmt.Errorf("catalog file %q has negative size", file.Path)
	}
	return nil
}

func encodePageToken(path string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(path))
}

func decodePageToken(token string) (string, error) {
	if token == "" {
		return "", nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", fmt.Errorf("invalid catalog page token %q: %w", strconv.Quote(token), err)
	}
	return string(raw), nil
}

var _ Repository = (*SQLiteRepository)(nil)
