package shots

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/proto"
)

type sqlDatabase interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteRepository struct{ db sqlDatabase }

func NewSQLiteRepository(db sqlDatabase) Repository { return &sqliteRepository{db: db} }

const timestampFormat = "2006-01-02T15:04:05.000000000Z07:00"

func (r *sqliteRepository) Create(ctx context.Context, shot Shot, revision Revision) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin shot create: %w", err)
	}
	defer tx.Rollback()
	stamp := shot.CreatedAt.UTC().Format(timestampFormat)
	if _, err := tx.ExecContext(ctx, `INSERT INTO shots(id,project_id,name,created_at,updated_at,head_revision) VALUES(?,?,?,?,?,1)`, shot.ID, shot.ProjectID, shot.Name, stamp, stamp); err != nil {
		return fmt.Errorf("insert shot: %w", err)
	}
	if err := insertRevision(ctx, tx, revision); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit shot create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) Save(ctx context.Context, id string, expected int64, revision Revision) (Shot, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Shot{}, fmt.Errorf("begin shot save: %w", err)
	}
	defer tx.Rollback()
	stamp := revision.CreatedAt.UTC().Format(timestampFormat)
	shot, err := scanShot(tx.QueryRowContext(ctx, `UPDATE shots SET name=?,updated_at=?,head_revision=head_revision+1 WHERE id=? AND head_revision=? RETURNING id,project_id,name,created_at,updated_at,head_revision`, revision.Name, stamp, id, expected))
	if errors.Is(err, sql.ErrNoRows) {
		var current int64
		lookupErr := tx.QueryRowContext(ctx, `SELECT head_revision FROM shots WHERE id=?`, id).Scan(&current)
		if errors.Is(lookupErr, sql.ErrNoRows) {
			return Shot{}, ErrShotNotFound{ID: id}
		}
		if lookupErr != nil {
			return Shot{}, fmt.Errorf("read shot revision after conflict: %w", lookupErr)
		}
		return Shot{}, ErrRevisionConflict{ShotID: id, Expected: expected, Current: current}
	}
	if err != nil {
		return Shot{}, fmt.Errorf("update shot head: %w", err)
	}
	if err := insertRevision(ctx, tx, revision); err != nil {
		return Shot{}, err
	}
	if err := tx.Commit(); err != nil {
		return Shot{}, fmt.Errorf("commit shot save: %w", err)
	}
	return shot, nil
}

func insertRevision(ctx context.Context, tx *sql.Tx, revision Revision) error {
	snapshot, err := proto.MarshalOptions{Deterministic: true}.Marshal(revision.Snapshot)
	if err != nil {
		return fmt.Errorf("encode shot snapshot: %w", err)
	}
	issues, err := json.Marshal(revision.Issues)
	if err != nil {
		return fmt.Errorf("encode shot issues: %w", err)
	}
	plate, err := marshalMedia(revision.Plate)
	if err != nil {
		return err
	}
	active, err := marshalMedia(revision.ActiveMask)
	if err != nil {
		return err
	}
	hardware, err := marshalMedia(revision.HardwareMask)
	if err != nil {
		return err
	}
	screen, err := marshalMedia(revision.Screen)
	if err != nil {
		return err
	}
	eligible := 0
	if revision.RenderEligible {
		eligible = 1
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO shot_revisions(shot_id,revision,created_at,name,snapshot,recipe_sha256,scene_sha256,presentation_sha256,scene_version,presentation_version,compositor_profile,plate,active_mask,hardware_mask,screen,render_eligible,issues) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		revision.ShotID, revision.Revision, revision.CreatedAt.UTC().Format(timestampFormat), revision.Name, snapshot,
		revision.RecipeSHA256, revision.SceneSHA256, revision.PresentationSHA256, revision.SceneVersion, revision.PresentationVersion,
		revision.CompositorProfile, nullableBytes(plate), nullableBytes(active), nullableBytes(hardware), nullableBytes(screen), eligible, issues)
	if err != nil {
		return fmt.Errorf("insert immutable shot revision: %w", err)
	}
	return nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Shot, error) {
	shot, err := scanShot(r.db.QueryRowContext(ctx, `SELECT id,project_id,name,created_at,updated_at,head_revision FROM shots WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Shot{}, ErrShotNotFound{ID: id}
	}
	if err != nil {
		return Shot{}, fmt.Errorf("get shot: %w", err)
	}
	return shot, nil
}

func (r *sqliteRepository) List(ctx context.Context, projectID string, limit int, after *Shot) ([]Shot, error) {
	query := `SELECT id,project_id,name,created_at,updated_at,head_revision FROM shots WHERE project_id=? ORDER BY created_at DESC,id DESC LIMIT ?`
	args := []any{projectID, limit}
	if after != nil {
		query = `SELECT id,project_id,name,created_at,updated_at,head_revision FROM shots WHERE project_id=? AND (created_at<? OR (created_at=? AND id<?)) ORDER BY created_at DESC,id DESC LIMIT ?`
		cursor := after.CreatedAt.UTC().Format(timestampFormat)
		args = []any{projectID, cursor, cursor, after.ID, limit}
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list shots: %w", err)
	}
	defer rows.Close()
	result := make([]Shot, 0)
	for rows.Next() {
		shot, scanErr := scanShot(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, shot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate shots: %w", err)
	}
	return result, nil
}

func (r *sqliteRepository) GetRevision(ctx context.Context, id string, revision int64) (Revision, error) {
	value, err := scanRevision(r.db.QueryRowContext(ctx, `SELECT shot_id,revision,created_at,name,snapshot,recipe_sha256,scene_sha256,presentation_sha256,scene_version,presentation_version,compositor_profile,plate,active_mask,hardware_mask,screen,render_eligible,issues FROM shot_revisions WHERE shot_id=? AND revision=?`, id, revision))
	if errors.Is(err, sql.ErrNoRows) {
		return Revision{}, ErrRevisionNotFound{ShotID: id, Revision: revision}
	}
	if err != nil {
		return Revision{}, fmt.Errorf("get shot revision: %w", err)
	}
	return value, nil
}

func (r *sqliteRepository) ListRevisions(ctx context.Context, id string, limit int, after int64) ([]Revision, error) {
	query := `SELECT shot_id,revision,created_at,name,snapshot,recipe_sha256,scene_sha256,presentation_sha256,scene_version,presentation_version,compositor_profile,plate,active_mask,hardware_mask,screen,render_eligible,issues FROM shot_revisions WHERE shot_id=? ORDER BY revision DESC LIMIT ?`
	args := []any{id, limit}
	if after > 0 {
		query = `SELECT shot_id,revision,created_at,name,snapshot,recipe_sha256,scene_sha256,presentation_sha256,scene_version,presentation_version,compositor_profile,plate,active_mask,hardware_mask,screen,render_eligible,issues FROM shot_revisions WHERE shot_id=? AND revision<? ORDER BY revision DESC LIMIT ?`
		args = []any{id, after, limit}
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list shot revisions: %w", err)
	}
	defer rows.Close()
	result := make([]Revision, 0)
	for rows.Next() {
		value, scanErr := scanRevision(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate shot revisions: %w", err)
	}
	return result, nil
}

type scanner interface{ Scan(...any) error }

func scanShot(row scanner) (Shot, error) {
	var shot Shot
	var created, updated string
	err := row.Scan(&shot.ID, &shot.ProjectID, &shot.Name, &created, &updated, &shot.HeadRevision)
	if err != nil {
		return Shot{}, err
	}
	shot.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return Shot{}, fmt.Errorf("parse shot created_at: %w", err)
	}
	shot.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		return Shot{}, fmt.Errorf("parse shot updated_at: %w", err)
	}
	return shot, nil
}

func scanRevision(row scanner) (Revision, error) {
	var revision Revision
	var created string
	var snapshot, plate, active, hardware, screen, issues []byte
	var eligible int
	err := row.Scan(&revision.ShotID, &revision.Revision, &created, &revision.Name, &snapshot, &revision.RecipeSHA256,
		&revision.SceneSHA256, &revision.PresentationSHA256, &revision.SceneVersion, &revision.PresentationVersion,
		&revision.CompositorProfile, &plate, &active, &hardware, &screen, &eligible, &issues)
	if err != nil {
		return Revision{}, err
	}
	revision.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return Revision{}, fmt.Errorf("parse shot revision created_at: %w", err)
	}
	revision.Snapshot = &shotsv1.ShotSnapshot{}
	if err := proto.Unmarshal(snapshot, revision.Snapshot); err != nil {
		return Revision{}, fmt.Errorf("decode shot revision snapshot: %w", err)
	}
	if err := json.Unmarshal(issues, &revision.Issues); err != nil {
		return Revision{}, fmt.Errorf("decode shot revision issues: %w", err)
	}
	revision.Plate, err = unmarshalMedia(plate)
	if err != nil {
		return Revision{}, err
	}
	revision.ActiveMask, err = unmarshalMedia(active)
	if err != nil {
		return Revision{}, err
	}
	revision.HardwareMask, err = unmarshalMedia(hardware)
	if err != nil {
		return Revision{}, err
	}
	revision.Screen, err = unmarshalMedia(screen)
	if err != nil {
		return Revision{}, err
	}
	revision.RenderEligible = eligible != 0
	return revision, nil
}

func marshalMedia(media *shotsv1.ResolvedSourceMedia) ([]byte, error) {
	if media == nil {
		return nil, nil
	}
	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(media)
	if err != nil {
		return nil, fmt.Errorf("encode resolved source metadata: %w", err)
	}
	return data, nil
}

func unmarshalMedia(data []byte) (*shotsv1.ResolvedSourceMedia, error) {
	if len(data) == 0 {
		return nil, nil
	}
	media := &shotsv1.ResolvedSourceMedia{}
	if err := proto.Unmarshal(data, media); err != nil {
		return nil, fmt.Errorf("decode resolved source metadata: %w", err)
	}
	return media, nil
}

func nullableBytes(data []byte) any {
	if data == nil {
		return nil
	}
	return data
}
