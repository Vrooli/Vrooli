// Package revisionstore contains the narrowly shared append-only SQLite
// mechanics used by typed physical-component revision owners. It does not
// validate or interpret domain candidates.
package revisionstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotFound = errors.New("revision not found")
	ErrConflict = errors.New("revision head conflict")
)

type Domain uint8

const (
	SceneRevisions Domain = iota + 1
	MotionRevisions
	PhysicalShotPresetRevisions
	LibraryIntakeRevisions
)

func (d Domain) table() (string, string, bool) {
	switch d {
	case SceneRevisions:
		return "scene_revisions", "scene_id", true
	case MotionRevisions:
		return "motion_revisions", "motion_id", true
	case PhysicalShotPresetRevisions:
		return "physical_shot_preset_revisions", "preset_id", true
	case LibraryIntakeRevisions:
		return "library_intake_candidate_revisions", "candidate_id", true
	default:
		return "", "", false
	}
}

type Record[T any] struct {
	ID        string
	Revision  uint32
	SHA256    string
	Value     T
	Canonical []byte
	CreatedAt time.Time
}

type Codec[T any] struct {
	ID     func(T) string
	Decode func([]byte) (T, string, error)
}

type rowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Database is the narrow context-aware SQL surface shared by database/sql and
// api-core's routed pool. Methods must receive the request context on every
// operation so test-mode routing remains active.
type Database interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// SQLite stores canonical snapshots in a domain-owned table. Table and ID
// column are compile-time owner constants, checked before they enter SQL.
type SQLite[T any] struct {
	db       Database
	table    string
	idColumn string
	codec    Codec[T]
}

func NewSQLite[T any](db Database, domain Domain, codec Codec[T]) (*SQLite[T], error) {
	table, idColumn, knownDomain := domain.table()
	if db == nil || !knownDomain || codec.ID == nil || codec.Decode == nil {
		return nil, errors.New("revision store requires a database, fixed owner identifiers, and a complete codec")
	}
	return &SQLite[T]{db: db, table: table, idColumn: idColumn, codec: codec}, nil
}

func (r *SQLite[T]) Save(ctx context.Context, proposed Record[T], expected uint32) (Record[T], error) {
	if proposed.ID == "" || proposed.ID != r.codec.ID(proposed.Value) || proposed.SHA256 == "" || len(proposed.Canonical) == 0 || proposed.CreatedAt.IsZero() {
		return Record[T]{}, errors.New("candidate revision is incomplete")
	}
	decoded, digest, err := r.codec.Decode(proposed.Canonical)
	if err != nil || digest != proposed.SHA256 || r.codec.ID(decoded) != proposed.ID {
		return Record[T]{}, errors.New("candidate revision canonical bytes do not match its typed identity and digest")
	}
	if expected == ^uint32(0) {
		return Record[T]{}, errors.New("revision number is exhausted")
	}
	target := expected + 1
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Record[T]{}, fmt.Errorf("begin revision write: %w", err)
	}
	finished := false
	defer func() {
		if !finished {
			_ = tx.Rollback()
		}
	}()
	if prior, getErr := r.getAt(ctx, tx, proposed.ID, target); getErr == nil {
		_ = tx.Rollback()
		finished = true
		if prior.SHA256 == proposed.SHA256 {
			return prior, nil
		}
		return Record[T]{}, r.conflict(ctx, proposed.ID, expected)
	} else if !errors.Is(getErr, ErrNotFound) {
		return Record[T]{}, fmt.Errorf("check target revision replay: %w", getErr)
	}
	var current int64
	query := fmt.Sprintf("SELECT COALESCE(MAX(revision),0) FROM %s WHERE %s=?", r.table, r.idColumn)
	if err := tx.QueryRowContext(ctx, query, proposed.ID).Scan(&current); err != nil {
		return Record[T]{}, fmt.Errorf("read revision head: %w", err)
	}
	if current < 0 || current > int64(^uint32(0)) {
		return Record[T]{}, errors.New("stored revision head is invalid")
	}
	if uint32(current) != expected {
		_ = tx.Rollback()
		finished = true
		return Record[T]{}, r.conflict(ctx, proposed.ID, expected)
	}
	proposed.Revision = target
	insert := fmt.Sprintf("INSERT INTO %s(%s,revision,sha256,candidate_json,created_at) VALUES(?,?,?,?,?)", r.table, r.idColumn)
	if _, err := tx.ExecContext(ctx, insert, proposed.ID, target, proposed.SHA256, string(proposed.Canonical), proposed.CreatedAt.UTC().Format(time.RFC3339Nano)); err != nil {
		_ = tx.Rollback()
		finished = true
		return r.resolveWriteRace(ctx, proposed, expected, fmt.Errorf("insert candidate revision: %w", err))
	}
	if err := tx.Commit(); err != nil {
		finished = true
		return r.resolveWriteRace(ctx, proposed, expected, fmt.Errorf("commit candidate revision: %w", err))
	}
	finished = true
	return proposed, nil
}

func (r *SQLite[T]) Get(ctx context.Context, id string, revision uint32) (Record[T], error) {
	if id == "" || revision == 0 {
		return Record[T]{}, ErrNotFound
	}
	query := fmt.Sprintf("SELECT revision,sha256,candidate_json,created_at FROM %s WHERE %s=? AND revision=?", r.table, r.idColumn)
	var row Record[T]
	var data, created string
	if err := r.db.QueryRowContext(ctx, query, id, revision).Scan(&row.Revision, &row.SHA256, &data, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record[T]{}, ErrNotFound
		}
		return Record[T]{}, fmt.Errorf("read candidate revision: %w", err)
	}
	return r.decode(id, row.Revision, row.SHA256, []byte(data), created)
}

func (r *SQLite[T]) Head(ctx context.Context, id string) (Record[T], error) {
	if id == "" {
		return Record[T]{}, ErrNotFound
	}
	query := fmt.Sprintf("SELECT COALESCE(MAX(revision),0) FROM %s WHERE %s=?", r.table, r.idColumn)
	var revision uint32
	if err := r.db.QueryRowContext(ctx, query, id).Scan(&revision); err != nil {
		return Record[T]{}, fmt.Errorf("read candidate head: %w", err)
	}
	if revision == 0 {
		return Record[T]{}, ErrNotFound
	}
	return r.Get(ctx, id, revision)
}

func (r *SQLite[T]) List(ctx context.Context, id string, after uint32, limit int) ([]Record[T], error) {
	if id == "" || limit < 1 || limit > 100 {
		return nil, errors.New("revision list requires an ID and page size between 1 and 100")
	}
	query := fmt.Sprintf("SELECT revision,sha256,candidate_json,created_at FROM %s WHERE %s=? AND revision>? ORDER BY revision LIMIT ?", r.table, r.idColumn)
	rows, err := r.db.QueryContext(ctx, query, id, after, limit)
	if err != nil {
		return nil, fmt.Errorf("list candidate revisions: %w", err)
	}
	defer rows.Close()
	result := make([]Record[T], 0)
	for rows.Next() {
		var revision uint32
		var digest, data, created string
		if err := rows.Scan(&revision, &digest, &data, &created); err != nil {
			return nil, fmt.Errorf("scan candidate revision: %w", err)
		}
		item, err := r.decode(id, revision, digest, []byte(data), created)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read candidate revisions: %w", err)
	}
	return result, nil
}

// ListHeads returns one latest immutable revision per candidate in stable ID
// order. It is a projection over the revision owner, not a second catalog.
func (r *SQLite[T]) ListHeads(ctx context.Context, afterID string, limit int) ([]Record[T], error) {
	if limit < 1 || limit > 100 {
		return nil, errors.New("candidate head list page size must be between 1 and 100")
	}
	query := fmt.Sprintf(`SELECT r.%s,r.revision,r.sha256,r.candidate_json,r.created_at FROM %s r JOIN (SELECT %s,MAX(revision) AS revision FROM %s GROUP BY %s) h ON r.%s=h.%s AND r.revision=h.revision WHERE r.%s>? ORDER BY r.%s LIMIT ?`,
		r.idColumn, r.table, r.idColumn, r.table, r.idColumn, r.idColumn, r.idColumn, r.idColumn, r.idColumn)
	rows, err := r.db.QueryContext(ctx, query, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("list candidate heads: %w", err)
	}
	result := make([]Record[T], 0, limit)
	for rows.Next() {
		var id, digest, data, created string
		var revision uint32
		if err := rows.Scan(&id, &revision, &digest, &data, &created); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan candidate head: %w", err)
		}
		item, err := r.decode(id, revision, digest, []byte(data), created)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		result = append(result, item)
	}
	rowsErr := rows.Err()
	closeErr := rows.Close()
	if err := errors.Join(rowsErr, closeErr); err != nil {
		return nil, fmt.Errorf("read candidate heads: %w", err)
	}
	return result, nil
}

func (r *SQLite[T]) getAt(ctx context.Context, q rowQuerier, id string, revision uint32) (Record[T], error) {
	query := fmt.Sprintf("SELECT sha256,candidate_json,created_at FROM %s WHERE %s=? AND revision=?", r.table, r.idColumn)
	var digest, data, created string
	if err := q.QueryRowContext(ctx, query, id, revision).Scan(&digest, &data, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record[T]{}, ErrNotFound
		}
		return Record[T]{}, err
	}
	return r.decode(id, revision, digest, []byte(data), created)
}

func (r *SQLite[T]) decode(id string, revision uint32, storedDigest string, data []byte, created string) (Record[T], error) {
	value, digest, err := r.codec.Decode(data)
	if err != nil {
		return Record[T]{}, fmt.Errorf("corrupt revision %s/%d: %w", id, revision, err)
	}
	if r.codec.ID(value) != id || digest != storedDigest {
		return Record[T]{}, fmt.Errorf("corrupt revision %s/%d: identity or digest mismatch", id, revision)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, created)
	if err != nil || createdAt.IsZero() {
		return Record[T]{}, fmt.Errorf("corrupt revision %s/%d: invalid creation time", id, revision)
	}
	return Record[T]{ID: id, Revision: revision, SHA256: digest, Value: value, Canonical: append([]byte(nil), data...), CreatedAt: createdAt.UTC()}, nil
}

func (r *SQLite[T]) conflict(ctx context.Context, id string, expected uint32) error {
	head, err := r.Head(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return fmt.Errorf("%w: expected head %d, current head 0", ErrConflict, expected)
	}
	if err != nil {
		return fmt.Errorf("%w: unable to read current head: %v", ErrConflict, err)
	}
	return fmt.Errorf("%w: expected head %d, current head %d", ErrConflict, expected, head.Revision)
}

func (r *SQLite[T]) resolveWriteRace(ctx context.Context, proposed Record[T], expected uint32, cause error) (Record[T], error) {
	target, err := r.Get(ctx, proposed.ID, proposed.Revision)
	if err == nil && target.SHA256 == proposed.SHA256 {
		return target, nil
	}
	if err == nil || errors.Is(err, ErrNotFound) {
		return Record[T]{}, r.conflict(ctx, proposed.ID, expected)
	}
	return Record[T]{}, errors.Join(cause, fmt.Errorf("resolve target revision outcome: %w", err))
}
