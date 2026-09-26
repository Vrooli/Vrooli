package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-manager/internal/sqlcompat"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/filerouting"
	corestorage "github.com/vrooli/api-core/storage"
)

// SQLiteCollector stores generated content under the routed data root and
// durable metadata in SQLite. Every method resolves the root from ctx so a
// test-mode request never writes artifacts into production storage.
type SQLiteCollector struct {
	db    sqlcompat.DB
	roots *filerouting.RoutedRoots
}

func NewSQLiteCollector(db sqlcompat.DB, roots *filerouting.RoutedRoots) *SQLiteCollector {
	return &SQLiteCollector{db: db, roots: roots}
}

func (c *SQLiteCollector) root(ctx context.Context) (string, error) {
	if c == nil || c.db == nil || c.roots == nil {
		return "", fmt.Errorf("artifact collector is not configured")
	}
	root, err := c.roots.Pick(ctx, corestorage.ClassData)
	if err != nil {
		return "", fmt.Errorf("resolve artifact data root: %w", err)
	}
	return filepath.Join(root, "artifacts"), nil
}

func (c *SQLiteCollector) Store(ctx context.Context, req StoreRequest) (*Artifact, error) {
	if req.RunID == uuid.Nil || req.Content == nil || strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("artifact requires run id, name, and content")
	}
	root, err := c.root(ctx)
	if err != nil {
		return nil, err
	}
	id := uuid.New()
	rel := filepath.Join(req.RunID.String(), id.String())
	absolute := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
		return nil, fmt.Errorf("create artifact directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(absolute), ".artifact-*")
	if err != nil {
		return nil, fmt.Errorf("create artifact: %w", err)
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(tmp, hash), req.Content)
	closeErr := tmp.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(tmp.Name())
		if copyErr != nil {
			return nil, fmt.Errorf("write artifact: %w", copyErr)
		}
		return nil, fmt.Errorf("close artifact: %w", closeErr)
	}
	if err := os.Rename(tmp.Name(), absolute); err != nil {
		_ = os.Remove(tmp.Name())
		return nil, fmt.Errorf("publish artifact: %w", err)
	}
	now := time.Now().UTC()
	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		_ = os.Remove(absolute)
		return nil, fmt.Errorf("encode artifact metadata: %w", err)
	}
	artifact := &Artifact{ID: id, RunID: req.RunID, Type: req.Type, Name: req.Name, StoragePath: rel, ContentSize: written, ContentType: req.ContentType, Checksum: hex.EncodeToString(hash.Sum(nil)), Metadata: req.Metadata, CreatedAt: now}
	_, err = c.db.ExecContext(ctx, `INSERT INTO artifacts (id, run_id, artifact_type, name, storage_path, content_size, content_type, checksum, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, artifact.ID, artifact.RunID, artifact.Type, artifact.Name, artifact.StoragePath, artifact.ContentSize, artifact.ContentType, artifact.Checksum, metadata, artifact.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		_ = os.Remove(absolute)
		return nil, fmt.Errorf("persist artifact metadata: %w", err)
	}
	c.roots.RecordWrite(ctx)
	return artifact, nil
}

func (c *SQLiteCollector) Get(ctx context.Context, id uuid.UUID) (*Artifact, error) {
	return c.get(ctx, `WHERE id = ?`, id)
}

func (c *SQLiteCollector) get(ctx context.Context, clause string, args ...any) (*Artifact, error) {
	var row struct {
		ID          string `db:"id"`
		RunID       string `db:"run_id"`
		Type        string `db:"type"`
		Name        string `db:"name"`
		StoragePath string `db:"storage_path"`
		ContentType string `db:"content_type"`
		Checksum    string `db:"checksum"`
		CreatedAt   string `db:"created_at"`
		ContentSize int64  `db:"content_size"`
		Metadata    []byte `db:"metadata"`
	}
	err := c.db.GetContext(ctx, &row, `SELECT id, run_id, artifact_type AS type, name, storage_path, content_size, content_type, checksum, metadata, created_at FROM artifacts `+clause, args...)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(row.ID)
	if err != nil {
		return nil, err
	}
	runID, err := uuid.Parse(row.RunID)
	if err != nil {
		return nil, err
	}
	created, err := time.Parse(time.RFC3339Nano, row.CreatedAt)
	if err != nil {
		return nil, err
	}
	metadata := map[string]string{}
	if len(row.Metadata) > 0 {
		if err := json.Unmarshal(row.Metadata, &metadata); err != nil {
			return nil, err
		}
	}
	return &Artifact{ID: id, RunID: runID, Type: ArtifactType(row.Type), Name: row.Name, StoragePath: row.StoragePath, ContentSize: row.ContentSize, ContentType: row.ContentType, Checksum: row.Checksum, Metadata: metadata, CreatedAt: created}, nil
}

func (c *SQLiteCollector) Read(ctx context.Context, id uuid.UUID) (io.ReadCloser, error) {
	artifact, err := c.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	root, err := c.root(ctx)
	if err != nil {
		return nil, err
	}
	return os.Open(filepath.Join(root, artifact.StoragePath))
}

func (c *SQLiteCollector) List(ctx context.Context, runID uuid.UUID, opts ListOptions) ([]*Artifact, error) {
	query := `SELECT id FROM artifacts WHERE run_id = ?`
	args := []any{runID}
	if opts.Type != nil {
		query += ` AND artifact_type = ?`
		args = append(args, *opts.Type)
	}
	query += ` ORDER BY created_at DESC`
	if opts.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, opts.Limit)
	}
	if opts.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, opts.Offset)
	}
	rows, err := c.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var artifacts []*Artifact
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		parsed, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		artifact, err := c.Get(ctx, parsed)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, rows.Err()
}

func (c *SQLiteCollector) Delete(ctx context.Context, id uuid.UUID) error {
	artifact, err := c.Get(ctx, id)
	if err != nil {
		return err
	}
	root, err := c.root(ctx)
	if err != nil {
		return err
	}
	if _, err := c.db.ExecContext(ctx, `DELETE FROM artifacts WHERE id = ?`, id); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(root, artifact.StoragePath)); err != nil && !os.IsNotExist(err) {
		return err
	}
	c.roots.RecordWrite(ctx)
	return nil
}

func (c *SQLiteCollector) DeleteByRun(ctx context.Context, runID uuid.UUID) error {
	artifacts, err := c.List(ctx, runID, ListOptions{})
	if err != nil {
		return err
	}
	for _, artifact := range artifacts {
		if err := c.Delete(ctx, artifact.ID); err != nil {
			return err
		}
	}
	return nil
}

func (c *SQLiteCollector) DeleteBefore(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	if limit <= 0 {
		return 0, fmt.Errorf("artifact retention batch limit must be positive")
	}
	rows, err := c.db.QueryxContext(ctx, `SELECT id FROM artifacts WHERE created_at < ? ORDER BY created_at ASC LIMIT ?`, cutoff.UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return 0, err
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, id := range ids {
		if err := c.Delete(ctx, id); err != nil {
			return 0, err
		}
	}
	return len(ids), nil
}

var (
	_ Collector          = (*SQLiteCollector)(nil)
	_ RetentionCollector = (*SQLiteCollector)(nil)
)
