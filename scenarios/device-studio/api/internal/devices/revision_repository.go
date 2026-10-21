package devices

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"device-studio/internal/library"
)

type DeviceRevisionRepository interface {
	SaveCandidate(context.Context, DeviceRevision, uint32) (DeviceRevision, error)
	GetRevision(context.Context, string, uint32) (DeviceRevision, error)
	GetHead(context.Context, string) (DeviceRevision, error)
	ListRevisions(context.Context, string, uint32, int) ([]DeviceRevision, error)
	ListHeads(context.Context, string, int) ([]DeviceRevision, error)
}

type DeviceRevisionService struct {
	repo DeviceRevisionRepository
	now  func() time.Time
}

func NewDeviceRevisionService(repo DeviceRevisionRepository, now func() time.Time) (*DeviceRevisionService, error) {
	if repo == nil {
		return nil, errors.New("device revision repository is required")
	}
	if now == nil {
		now = time.Now
	}
	return &DeviceRevisionService{repo: repo, now: now}, nil
}

// SaveCandidate stores a new immutable revision or returns the existing
// revision for an exact canonical replay. expectedHead=0 creates the first.
func (s *DeviceRevisionService) SaveCandidate(ctx context.Context, input DeviceCandidate, expectedHead uint32) (DeviceRevision, error) {
	candidate, data, digest, err := CanonicalizeDeviceCandidate(input)
	if err != nil {
		return DeviceRevision{}, fmt.Errorf("%w: %v", ErrInvalidCandidate, err)
	}
	revision := DeviceRevision{
		DeviceID:  candidate.Definition.ID,
		SHA256:    digest,
		State:     DeviceRevisionCandidate,
		Candidate: candidate,
		CreatedAt: s.now().UTC(),
		canonical: data,
	}
	return s.repo.SaveCandidate(ctx, revision, expectedHead)
}

func (s *DeviceRevisionService) GetRevision(ctx context.Context, id string, revision uint32) (DeviceRevision, error) {
	return s.repo.GetRevision(ctx, id, revision)
}

func (s *DeviceRevisionService) GetHead(ctx context.Context, id string) (DeviceRevision, error) {
	return s.repo.GetHead(ctx, id)
}

func (s *DeviceRevisionService) ListRevisions(ctx context.Context, id string, after uint32, limit int) ([]DeviceRevision, error) {
	return s.repo.ListRevisions(ctx, id, after, limit)
}

func (s *DeviceRevisionService) ListHeads(ctx context.Context, afterID string, limit int) ([]DeviceRevision, error) {
	return s.repo.ListHeads(ctx, afterID, limit)
}

type deviceSQLDatabase interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type sqliteDeviceRevisionRepository struct{ db deviceSQLDatabase }

func NewSQLiteDeviceRevisionRepository(db deviceSQLDatabase) DeviceRevisionRepository {
	return &sqliteDeviceRevisionRepository{db: db}
}

func (r *sqliteDeviceRevisionRepository) SaveCandidate(ctx context.Context, proposed DeviceRevision, expected uint32) (DeviceRevision, error) {
	if !library.IsValidStableID(proposed.DeviceID) || proposed.SHA256 == "" || len(proposed.canonical) == 0 || proposed.CreatedAt.IsZero() {
		return DeviceRevision{}, errors.New("device candidate revision is incomplete")
	}
	if expected == ^uint32(0) {
		return DeviceRevision{}, errors.New("device revision number is exhausted")
	}
	targetRevision := expected + 1
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return DeviceRevision{}, fmt.Errorf("begin device revision: %w", err)
	}
	finished := false
	defer func() {
		if !finished {
			_ = tx.Rollback()
		}
	}()
	if prior, getErr := getAtRevision(ctx, tx, proposed.DeviceID, targetRevision); getErr == nil && prior.SHA256 == proposed.SHA256 {
		_ = tx.Rollback()
		finished = true
		return prior, nil
	} else if getErr == nil {
		_ = tx.Rollback()
		finished = true
		return DeviceRevision{}, r.conflict(ctx, proposed.DeviceID, expected)
	} else if !errors.Is(getErr, ErrRevisionNotFound) {
		return DeviceRevision{}, fmt.Errorf("check device revision replay: %w", getErr)
	}
	var current int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(revision),0) FROM device_revisions WHERE device_id=?`, proposed.DeviceID).Scan(&current); err != nil {
		return DeviceRevision{}, fmt.Errorf("read device revision head: %w", err)
	}
	if current < 0 || current > int64(^uint32(0)) {
		return DeviceRevision{}, errors.New("stored device revision head is invalid")
	}
	if uint32(current) != expected {
		_ = tx.Rollback()
		finished = true
		return DeviceRevision{}, r.conflict(ctx, proposed.DeviceID, expected)
	}
	proposed.Revision = targetRevision
	_, err = tx.ExecContext(ctx, `INSERT INTO device_revisions(device_id,revision,sha256,candidate_json,created_at) VALUES(?,?,?,?,?)`,
		proposed.DeviceID, proposed.Revision, proposed.SHA256, string(proposed.canonical), proposed.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		_ = tx.Rollback()
		finished = true
		return r.resolveWriteRace(ctx, proposed, expected, fmt.Errorf("insert device revision: %w", err))
	}
	if err := tx.Commit(); err != nil {
		finished = true
		return r.resolveWriteRace(ctx, proposed, expected, fmt.Errorf("commit device revision: %w", err))
	}
	finished = true
	return proposed, nil
}

func (r *sqliteDeviceRevisionRepository) GetRevision(ctx context.Context, id string, revision uint32) (DeviceRevision, error) {
	if !library.IsValidStableID(id) || revision == 0 {
		return DeviceRevision{}, ErrRevisionNotFound
	}
	var row DeviceRevision
	var data, created string
	if err := r.db.QueryRowContext(ctx, `SELECT revision,sha256,candidate_json,created_at FROM device_revisions WHERE device_id=? AND revision=?`, id, revision).Scan(&row.Revision, &row.SHA256, &data, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DeviceRevision{}, fmt.Errorf("%w: %s revision %d", ErrRevisionNotFound, id, revision)
		}
		return DeviceRevision{}, fmt.Errorf("read device revision: %w", err)
	}
	return decodeDeviceRow(id, row.Revision, row.SHA256, []byte(data), created)
}

func (r *sqliteDeviceRevisionRepository) GetHead(ctx context.Context, id string) (DeviceRevision, error) {
	if !library.IsValidStableID(id) {
		return DeviceRevision{}, ErrRevisionNotFound
	}
	var revision uint32
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(revision),0) FROM device_revisions WHERE device_id=?`, id).Scan(&revision); err != nil {
		return DeviceRevision{}, fmt.Errorf("read device revision head: %w", err)
	}
	if revision == 0 {
		return DeviceRevision{}, fmt.Errorf("%w: %s", ErrRevisionNotFound, id)
	}
	return r.GetRevision(ctx, id, revision)
}

func (r *sqliteDeviceRevisionRepository) ListRevisions(ctx context.Context, id string, after uint32, limit int) ([]DeviceRevision, error) {
	if !library.IsValidStableID(id) || limit < 1 || limit > 100 {
		return nil, errors.New("device revision list requires a stable id and limit between 1 and 100")
	}
	rows, err := r.db.QueryContext(ctx, `SELECT revision,sha256,candidate_json,created_at FROM device_revisions WHERE device_id=? AND revision>? ORDER BY revision LIMIT ?`, id, after, limit)
	if err != nil {
		return nil, fmt.Errorf("list device revisions: %w", err)
	}
	defer rows.Close()
	result := make([]DeviceRevision, 0)
	for rows.Next() {
		var revision uint32
		var digest, data, created string
		if err := rows.Scan(&revision, &digest, &data, &created); err != nil {
			return nil, fmt.Errorf("scan device revision: %w", err)
		}
		item, err := decodeDeviceRow(id, revision, digest, []byte(data), created)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read device revisions: %w", err)
	}
	return result, nil
}

func (r *sqliteDeviceRevisionRepository) ListHeads(ctx context.Context, afterID string, limit int) ([]DeviceRevision, error) {
	if limit < 1 || limit > 100 {
		return nil, errors.New("device candidate page size must be between 1 and 100")
	}
	rows, err := r.db.QueryContext(ctx, `SELECT r.device_id,r.revision,r.sha256,r.candidate_json,r.created_at FROM device_revisions r JOIN (SELECT device_id,MAX(revision) AS revision FROM device_revisions GROUP BY device_id) h ON r.device_id=h.device_id AND r.revision=h.revision WHERE r.device_id>? ORDER BY r.device_id LIMIT ?`, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("list device candidate heads: %w", err)
	}
	result := make([]DeviceRevision, 0, limit)
	for rows.Next() {
		var id, digest, data, created string
		var revision uint32
		if err := rows.Scan(&id, &revision, &digest, &data, &created); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan device candidate head: %w", err)
		}
		item, err := decodeDeviceRow(id, revision, digest, []byte(data), created)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		result = append(result, item)
	}
	rowsErr, closeErr := rows.Err(), rows.Close()
	if err := errors.Join(rowsErr, closeErr); err != nil {
		return nil, fmt.Errorf("read device candidate heads: %w", err)
	}
	return result, nil
}

func (r *sqliteDeviceRevisionRepository) conflict(ctx context.Context, id string, expected uint32) error {
	current, err := r.GetHead(ctx, id)
	if errors.Is(err, ErrRevisionNotFound) {
		return fmt.Errorf("%w: device %s expected head %d, current head 0", ErrRevisionConflict, id, expected)
	}
	if err != nil {
		return fmt.Errorf("%w: unable to read current head: %v", ErrRevisionConflict, err)
	}
	return fmt.Errorf("%w: device %s expected head %d, current head %d", ErrRevisionConflict, id, expected, current.Revision)
}

func (r *sqliteDeviceRevisionRepository) resolveWriteRace(ctx context.Context, proposed DeviceRevision, expected uint32, cause error) (DeviceRevision, error) {
	target, err := r.GetRevision(ctx, proposed.DeviceID, proposed.Revision)
	if err == nil && target.SHA256 == proposed.SHA256 {
		return target, nil
	}
	if err == nil || errors.Is(err, ErrRevisionNotFound) {
		return DeviceRevision{}, r.conflict(ctx, proposed.DeviceID, expected)
	}
	return DeviceRevision{}, errors.Join(cause, fmt.Errorf("resolve device revision target %d outcome: %w", proposed.Revision, err))
}

type deviceRowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func getAtRevision(ctx context.Context, q deviceRowQuerier, id string, revision uint32) (DeviceRevision, error) {
	var storedDigest, data, created string
	if err := q.QueryRowContext(ctx, `SELECT sha256,candidate_json,created_at FROM device_revisions WHERE device_id=? AND revision=?`, id, revision).Scan(&storedDigest, &data, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DeviceRevision{}, ErrRevisionNotFound
		}
		return DeviceRevision{}, err
	}
	return decodeDeviceRow(id, revision, storedDigest, []byte(data), created)
}

func decodeDeviceRow(id string, revision uint32, storedDigest string, data []byte, created string) (DeviceRevision, error) {
	candidate, digest, err := decodeCanonicalCandidate(data)
	if err != nil {
		return DeviceRevision{}, fmt.Errorf("corrupt device revision %s/%d: %w", id, revision, err)
	}
	if candidate.Definition.ID != id || digest != storedDigest {
		return DeviceRevision{}, fmt.Errorf("corrupt device revision %s/%d: identity or digest mismatch", id, revision)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, created)
	if err != nil || createdAt.IsZero() {
		return DeviceRevision{}, fmt.Errorf("corrupt device revision %s/%d: invalid creation time", id, revision)
	}
	return DeviceRevision{DeviceID: id, Revision: revision, SHA256: digest, State: DeviceRevisionCandidate, Candidate: candidate, CreatedAt: createdAt.UTC(), canonical: append([]byte(nil), data...)}, nil
}

var _ DeviceRevisionRepository = (*sqliteDeviceRevisionRepository)(nil)
