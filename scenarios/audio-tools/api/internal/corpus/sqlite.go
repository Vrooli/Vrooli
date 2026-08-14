package corpus

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface the sqliteRepository depends
// on. Both *sql.DB (repository unit tests) and *database.RoutedDB
// (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production corpus Repository.
func NewSQLiteRepository(db SQLExecutor, clk schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

// clipTimeFormat matches RFC3339Nano so created_at sorts lexicographically
// in time order for a fixed zone.
const clipTimeFormat = time.RFC3339Nano

const (
	insertClipSQL = `
INSERT INTO corpus_clips (id, reference_text, tags, duration_ms, sample_rate_hz, format, blob_key, source, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	selectClipColumns = `id, reference_text, tags, duration_ms, sample_rate_hz, format, blob_key, source, created_at`
	selectClipByIDSQL = `SELECT ` + selectClipColumns + ` FROM corpus_clips WHERE id = ?`
	deleteClipSQL     = `DELETE FROM corpus_clips WHERE id = ?`
)

func (s *sqliteRepository) Create(ctx context.Context, c Clip) (Clip, error) {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = s.clock.Now().UTC()
	}
	c.Source = c.Source.Normalize()
	if c.BlobKey == "" {
		return Clip{}, fmt.Errorf("corpus: clip %q requires a blob_key", c.ID)
	}
	tags, err := marshalTags(c.Tags)
	if err != nil {
		return Clip{}, err
	}
	if _, err := s.db.ExecContext(ctx, insertClipSQL,
		c.ID, c.ReferenceText, tags, c.DurationMs, c.SampleRateHz, c.Format, c.BlobKey, string(c.Source),
		c.CreatedAt.Format(clipTimeFormat),
	); err != nil {
		return Clip{}, fmt.Errorf("insert clip %q: %w", c.ID, err)
	}
	return c, nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Clip, error) {
	row := s.db.QueryRowContext(ctx, selectClipByIDSQL, id)
	c, err := scanClip(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Clip{}, ErrClipNotFound{ID: id}
	}
	if err != nil {
		return Clip{}, fmt.Errorf("get clip %q: %w", id, err)
	}
	return c, nil
}

func (s *sqliteRepository) List(ctx context.Context, filter ListFilter) ([]Clip, error) {
	query := `SELECT ` + selectClipColumns + ` FROM corpus_clips`
	var args []any
	if filter.TagContains != "" {
		query += ` WHERE tags LIKE ?`
		args = append(args, "%"+filter.TagContains+"%")
	}
	query += ` ORDER BY created_at DESC, id DESC`
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list clips: %w", err)
	}
	defer rows.Close()

	var clips []Clip
	for rows.Next() {
		c, err := scanClip(rows)
		if err != nil {
			return nil, fmt.Errorf("scan clip: %w", err)
		}
		clips = append(clips, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate clips: %w", err)
	}
	return clips, nil
}

func (s *sqliteRepository) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, deleteClipSQL, id)
	if err != nil {
		return fmt.Errorf("delete clip %q: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete clip %q rows: %w", id, err)
	}
	if n == 0 {
		return ErrClipNotFound{ID: id}
	}
	return nil
}

// marshalTags serializes the JSON tags column; a nil slice encodes as [].
func marshalTags(tags []string) (string, error) {
	if tags == nil {
		tags = []string{}
	}
	b, err := json.Marshal(tags)
	if err != nil {
		return "", fmt.Errorf("marshal tags: %w", err)
	}
	return string(b), nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanClip(sc rowScanner) (Clip, error) {
	var (
		c          Clip
		tagsRaw    string
		sourceRaw  string
		createdRaw string
	)
	if err := sc.Scan(&c.ID, &c.ReferenceText, &tagsRaw, &c.DurationMs, &c.SampleRateHz,
		&c.Format, &c.BlobKey, &sourceRaw, &createdRaw,
	); err != nil {
		return Clip{}, err
	}
	if tagsRaw != "" && tagsRaw != "null" {
		if err := json.Unmarshal([]byte(tagsRaw), &c.Tags); err != nil {
			return Clip{}, fmt.Errorf("parse tags: %w", err)
		}
	}
	c.Source = Source(sourceRaw).Normalize()
	created, err := time.Parse(clipTimeFormat, createdRaw)
	if err != nil {
		return Clip{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	c.CreatedAt = created
	return c, nil
}
