package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type sqliteRepository struct{ db SQLDatabase }

func NewSQLiteRepository(db SQLDatabase) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) CreateRevision(ctx context.Context, revision BundleRevision) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin library revision: %w", err)
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO library_bundle_revisions(bundle_id,revision,manifest_sha256,manifest_json,review_sha256,review_json,published_at) VALUES(?,?,?,?,?,?,?)`,
		revision.Manifest.BundleID, revision.Manifest.Revision, revision.ManifestSHA256, string(revision.manifestJSON), revision.ReviewSHA256, string(revision.reviewJSON), revision.PublishedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert library bundle revision: %w", err)
	}
	for _, artifact := range revision.Manifest.Artifacts {
		key := revision.artifactKeys[artifact.ID]
		if key == "" {
			return errors.New("library artifact storage reference is missing")
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO library_bundle_artifacts(bundle_id,revision,artifact_id,blob_key) VALUES(?,?,?,?)`, revision.Manifest.BundleID, revision.Manifest.Revision, artifact.ID, key); err != nil {
			return fmt.Errorf("insert library artifact reference %q: %w", artifact.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit library bundle revision: %w", err)
	}
	return nil
}

func (r *sqliteRepository) GetRevision(ctx context.Context, bundleID string, revision uint32) (BundleRevision, error) {
	stored := storedRevision{artifactKeys: make(map[string]string)}
	var published string
	err := r.db.QueryRowContext(ctx, `SELECT manifest_json,manifest_sha256,review_json,review_sha256,published_at FROM library_bundle_revisions WHERE bundle_id=? AND revision=?`, bundleID, revision).
		Scan(&stored.manifestJSON, &stored.manifestSHA, &stored.reviewJSON, &stored.reviewSHA, &published)
	if errors.Is(err, sql.ErrNoRows) {
		return BundleRevision{}, fmt.Errorf("%w: %s revision %d", ErrRevisionNotFound, bundleID, revision)
	}
	if err != nil {
		return BundleRevision{}, fmt.Errorf("get library bundle revision: %w", err)
	}
	stored.publishedAt, err = time.Parse(time.RFC3339Nano, published)
	if err != nil {
		return BundleRevision{}, fmt.Errorf("parse library publication time: %w", err)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT artifact_id,blob_key FROM library_bundle_artifacts WHERE bundle_id=? AND revision=? ORDER BY artifact_id`, bundleID, revision)
	if err != nil {
		return BundleRevision{}, fmt.Errorf("list library artifact references: %w", err)
	}
	for rows.Next() {
		var id, key string
		if err := rows.Scan(&id, &key); err != nil {
			_ = rows.Close()
			return BundleRevision{}, fmt.Errorf("scan library artifact reference: %w", err)
		}
		if id == "" || key == "" || stored.artifactKeys[id] != "" {
			_ = rows.Close()
			return BundleRevision{}, errors.New("library artifact references are malformed")
		}
		stored.artifactKeys[id] = key
	}
	rowsErr := rows.Err()
	closeErr := rows.Close()
	if err := errors.Join(rowsErr, closeErr); err != nil {
		return BundleRevision{}, fmt.Errorf("read library artifact references: %w", err)
	}
	return BundleRevision{
		ManifestSHA256: stored.manifestSHA, ReviewSHA256: stored.reviewSHA,
		PublishedAt: stored.publishedAt, artifactKeys: stored.artifactKeys,
		manifestJSON: stored.manifestJSON, reviewJSON: stored.reviewJSON,
	}, nil
}

func (r *sqliteRepository) ListRevisions(ctx context.Context, afterBundleID string, afterRevision uint32, limit int) ([]BundleRevision, error) {
	if limit < 1 || limit > 100 || (afterBundleID == "" && afterRevision != 0) {
		return nil, errors.New("library revision list requires a valid cursor and page size between 1 and 100")
	}
	rows, err := r.db.QueryContext(ctx, `SELECT bundle_id,revision FROM library_bundle_revisions WHERE bundle_id>? OR (bundle_id=? AND revision>?) ORDER BY bundle_id,revision LIMIT ?`, afterBundleID, afterBundleID, afterRevision, limit)
	if err != nil {
		return nil, fmt.Errorf("list library revisions: %w", err)
	}
	type key struct {
		bundle   string
		revision uint32
	}
	keys := make([]key, 0, limit)
	for rows.Next() {
		var item key
		if err := rows.Scan(&item.bundle, &item.revision); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan library revision cursor: %w", err)
		}
		keys = append(keys, item)
	}
	rowsErr := rows.Err()
	closeErr := rows.Close()
	if err := errors.Join(rowsErr, closeErr); err != nil {
		return nil, fmt.Errorf("read library revision cursor: %w", err)
	}
	result := make([]BundleRevision, 0, len(keys))
	for _, item := range keys {
		revision, err := r.GetRevision(ctx, item.bundle, item.revision)
		if err != nil {
			return nil, err
		}
		result = append(result, revision)
	}
	return result, nil
}
