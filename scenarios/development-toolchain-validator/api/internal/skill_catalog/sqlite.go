package skill_catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"development-toolchain-validator/internal/clock"
)

const skillTimeFormat = time.RFC3339Nano

const (
	upsertSkillSQL = `
INSERT INTO skill_catalog (id, version, content_hash, synced_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  version      = excluded.version,
  content_hash = excluded.content_hash,
  synced_at    = excluded.synced_at
`

	selectSkillByIDSQL = `
SELECT id, version, content_hash, synced_at
FROM skill_catalog
WHERE id = ?
`

	listSkillsSQL = `
SELECT id, version, content_hash, synced_at
FROM skill_catalog
ORDER BY id ASC
`
)

type sqliteRepository struct {
	db    *sql.DB
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db *sql.DB, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

func (s *sqliteRepository) Upsert(ctx context.Context, sk Skill) (bool, bool, error) {
	if sk.SyncedAt.IsZero() {
		sk.SyncedAt = s.clock.Now().UTC()
	}
	// Detect insert vs update by first reading the existing row. SQLite's
	// ON CONFLICT DO UPDATE doesn't tell us which branch fired, and we
	// want to report added/updated counts to callers.
	existing, err := s.Get(ctx, sk.ID)
	existed := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		var nf ErrSkillNotFound
		if !errors.As(err, &nf) {
			return false, false, fmt.Errorf("upsert skill %q: probe: %w", sk.ID, err)
		}
		// not found is fine, existed stays false.
	}

	_, err = s.db.ExecContext(ctx, upsertSkillSQL,
		sk.ID, sk.Version, sk.ContentHash,
		sk.SyncedAt.Format(skillTimeFormat),
	)
	if err != nil {
		return false, false, fmt.Errorf("upsert skill %q: %w", sk.ID, err)
	}

	if !existed {
		return true, true, nil
	}
	changed := existing.Version != sk.Version || existing.ContentHash != sk.ContentHash
	return false, changed, nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Skill, error) {
	row := s.db.QueryRowContext(ctx, selectSkillByIDSQL, id)
	sk, err := scanSkill(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Skill{}, ErrSkillNotFound{ID: id}
	}
	if err != nil {
		return Skill{}, fmt.Errorf("get skill %q: %w", id, err)
	}
	return sk, nil
}

func (s *sqliteRepository) List(ctx context.Context) ([]Skill, error) {
	rows, err := s.db.QueryContext(ctx, listSkillsSQL)
	if err != nil {
		return nil, fmt.Errorf("list skill_catalog: %w", err)
	}
	defer rows.Close()

	var out []Skill
	for rows.Next() {
		sk, err := scanSkill(rows)
		if err != nil {
			return nil, fmt.Errorf("scan skill: %w", err)
		}
		out = append(out, sk)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate skills: %w", err)
	}
	return out, nil
}

func (s *sqliteRepository) DeleteMissing(ctx context.Context, keep []string) (int, error) {
	if len(keep) == 0 {
		res, err := s.db.ExecContext(ctx, `DELETE FROM skill_catalog`)
		if err != nil {
			return 0, fmt.Errorf("delete-missing (full): %w", err)
		}
		n, _ := res.RowsAffected()
		return int(n), nil
	}
	placeholders := make([]string, len(keep))
	args := make([]any, len(keep))
	for i, id := range keep {
		placeholders[i] = "?"
		args[i] = id
	}
	q := fmt.Sprintf(`DELETE FROM skill_catalog WHERE id NOT IN (%s)`, strings.Join(placeholders, ","))
	res, err := s.db.ExecContext(ctx, q, args...)
	if err != nil {
		return 0, fmt.Errorf("delete-missing: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSkill(s rowScanner) (Skill, error) {
	var (
		sk      Skill
		syncRaw string
	)
	if err := s.Scan(&sk.ID, &sk.Version, &sk.ContentHash, &syncRaw); err != nil {
		return Skill{}, err
	}
	t, err := time.Parse(skillTimeFormat, syncRaw)
	if err != nil {
		return Skill{}, fmt.Errorf("parse synced_at %q: %w", syncRaw, err)
	}
	sk.SyncedAt = t
	return sk, nil
}
