package graph

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"architecture-cartographer/internal/clock"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db                     SQLExecutor
	clock                  clock.Clock
	sourceFingerprintReady atomic.Bool
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

const snapshotTimeFormat = time.RFC3339Nano

const (
	insertSnapshotSQL = `
INSERT INTO graph_snapshots (id, scenario, content_hash, source_fingerprint, payload, payload_codec, extracted_at, extraction_ms)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(scenario, content_hash) DO UPDATE SET
  source_fingerprint = excluded.source_fingerprint,
  payload = excluded.payload,
  payload_codec = excluded.payload_codec,
  extracted_at = excluded.extracted_at,
  extraction_ms = excluded.extraction_ms`

	selectSnapshotByIDSQL = `
SELECT id, scenario, content_hash, source_fingerprint, payload, payload_codec, extracted_at, extraction_ms
FROM graph_snapshots WHERE id = ?`

	selectSnapshotByHashSQL = `
SELECT id, scenario, content_hash, source_fingerprint, payload, payload_codec, extracted_at, extraction_ms
FROM graph_snapshots WHERE scenario = ? AND content_hash = ?`

	selectSnapshotBySourceFingerprintSQL = `
SELECT id, scenario, content_hash, source_fingerprint, payload, payload_codec, extracted_at, extraction_ms
FROM graph_snapshots
WHERE scenario = ? AND source_fingerprint = ?
ORDER BY extracted_at DESC, id DESC
LIMIT 1`

	selectLatestSnapshotMetaSQL = `
SELECT id, scenario, content_hash, source_fingerprint, extracted_at, extraction_ms, length(payload)
FROM graph_snapshots
WHERE scenario = ?
ORDER BY extracted_at DESC, id DESC
LIMIT 1`

	listSnapshotsSQL = `
SELECT id, scenario, content_hash, source_fingerprint, payload, payload_codec, extracted_at, extraction_ms
FROM graph_snapshots
WHERE (? = '' OR scenario = ?)
ORDER BY extracted_at DESC, id DESC
LIMIT ?`

	clearSnapshotsSQL = `DELETE FROM graph_snapshots WHERE scenario = ?`

	addSourceFingerprintColumnSQL = `
ALTER TABLE graph_snapshots ADD COLUMN source_fingerprint TEXT NOT NULL DEFAULT ''`

	// payload_codec records how each row's payload is encoded. An empty
	// string means the legacy raw-JSON encoding, so existing rows stay
	// readable without being rewritten: a blocking migration over
	// multi-hundred-megabyte payloads is exactly the wrong thing to run on a
	// database that just caused a disk-full incident.
	addPayloadCodecColumnSQL = `
ALTER TABLE graph_snapshots ADD COLUMN payload_codec TEXT NOT NULL DEFAULT ''`

	createSourceFingerprintIndexSQL = `
CREATE INDEX IF NOT EXISTS idx_graph_snapshots_source_fingerprint
  ON graph_snapshots(scenario, source_fingerprint, extracted_at DESC, id DESC)
  WHERE source_fingerprint <> ''`
)

func (r *sqliteRepository) SaveSnapshot(ctx context.Context, s GraphSnapshot) (GraphSnapshot, error) {
	if err := r.ensureSourceFingerprintColumn(ctx); err != nil {
		return GraphSnapshot{}, err
	}
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if s.ExtractedAt.IsZero() {
		s.ExtractedAt = r.clock.Now().UTC()
	}
	payload, err := json.Marshal(snapshotPayload{
		Languages: s.Languages,
		Files:     s.Files,
		Packages:  s.Packages,
		Symbols:   s.Symbols,
		Imports:   s.Imports,
		Skipped:   s.SkippedAdapters,
	})
	if err != nil {
		return GraphSnapshot{}, fmt.Errorf("encode snapshot %q: %w", s.ID, err)
	}
	stored, codec, err := encodePayload(payload)
	if err != nil {
		return GraphSnapshot{}, fmt.Errorf("compress snapshot %q: %w", s.ID, err)
	}
	_, err = r.db.ExecContext(ctx, insertSnapshotSQL,
		s.ID, s.Scenario, s.ContentHash, s.SourceFingerprint, stored, codec,
		s.ExtractedAt.Format(snapshotTimeFormat), s.ExtractionMS,
	)
	if err != nil {
		return GraphSnapshot{}, fmt.Errorf("insert snapshot %q: %w", s.ID, err)
	}
	return s, nil
}

func (r *sqliteRepository) GetSnapshot(ctx context.Context, id string) (GraphSnapshot, error) {
	if err := r.ensureSourceFingerprintColumn(ctx); err != nil {
		return GraphSnapshot{}, err
	}
	row := r.db.QueryRowContext(ctx, selectSnapshotByIDSQL, id)
	s, err := scanSnapshot(row)
	if errors.Is(err, sql.ErrNoRows) {
		return GraphSnapshot{}, ErrSnapshotNotFound{ID: id}
	}
	if err != nil {
		return GraphSnapshot{}, fmt.Errorf("get snapshot %q: %w", id, err)
	}
	return s, nil
}

func (r *sqliteRepository) LatestSnapshotMeta(ctx context.Context, scenario string) (GraphSnapshotMeta, error) {
	if err := r.ensureSourceFingerprintColumn(ctx); err != nil {
		return GraphSnapshotMeta{}, err
	}
	row := r.db.QueryRowContext(ctx, selectLatestSnapshotMetaSQL, scenario)
	meta, err := scanSnapshotMeta(row)
	if errors.Is(err, sql.ErrNoRows) {
		return GraphSnapshotMeta{}, ErrSnapshotNotFound{ID: "scenario=" + scenario}
	}
	if err != nil {
		return GraphSnapshotMeta{}, fmt.Errorf("latest snapshot meta: %w", err)
	}
	return meta, nil
}

func (r *sqliteRepository) FindByHash(ctx context.Context, scenario, contentHash string) (GraphSnapshot, error) {
	if err := r.ensureSourceFingerprintColumn(ctx); err != nil {
		return GraphSnapshot{}, err
	}
	row := r.db.QueryRowContext(ctx, selectSnapshotByHashSQL, scenario, contentHash)
	s, err := scanSnapshot(row)
	if errors.Is(err, sql.ErrNoRows) {
		return GraphSnapshot{}, ErrSnapshotNotFound{ID: fmt.Sprintf("scenario=%s hash=%s", scenario, contentHash)}
	}
	if err != nil {
		return GraphSnapshot{}, fmt.Errorf("find by hash: %w", err)
	}
	return s, nil
}

func (r *sqliteRepository) FindBySourceFingerprint(ctx context.Context, scenario, sourceFingerprint string) (GraphSnapshot, error) {
	if err := r.ensureSourceFingerprintColumn(ctx); err != nil {
		return GraphSnapshot{}, err
	}
	row := r.db.QueryRowContext(ctx, selectSnapshotBySourceFingerprintSQL, scenario, sourceFingerprint)
	s, err := scanSnapshot(row)
	if errors.Is(err, sql.ErrNoRows) {
		return GraphSnapshot{}, ErrSnapshotNotFound{ID: fmt.Sprintf("scenario=%s source_fingerprint=%s", scenario, sourceFingerprint)}
	}
	if err != nil {
		return GraphSnapshot{}, fmt.Errorf("find by source fingerprint: %w", err)
	}
	return s, nil
}

func (r *sqliteRepository) ListSnapshots(ctx context.Context, f ListSnapshotsFilter) (SnapshotPage, error) {
	if err := r.ensureSourceFingerprintColumn(ctx); err != nil {
		return SnapshotPage{}, err
	}
	limit := f.PageSize
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, listSnapshotsSQL, f.Scenario, f.Scenario, limit)
	if err != nil {
		return SnapshotPage{}, fmt.Errorf("list snapshots: %w", err)
	}
	defer rows.Close()

	var out []GraphSnapshot
	for rows.Next() {
		s, err := scanSnapshot(rows)
		if err != nil {
			return SnapshotPage{}, err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return SnapshotPage{}, fmt.Errorf("iterate snapshots: %w", err)
	}
	return SnapshotPage{Snapshots: out}, nil
}

func (r *sqliteRepository) ClearSnapshots(ctx context.Context, scenario string) (int, error) {
	if err := r.ensureSourceFingerprintColumn(ctx); err != nil {
		return 0, err
	}
	res, err := r.db.ExecContext(ctx, clearSnapshotsSQL, scenario)
	if err != nil {
		return 0, fmt.Errorf("clear snapshots: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	return int(n), nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

type snapshotPayload struct {
	Languages []Language    `json:"languages"`
	Files     []FileNode    `json:"files"`
	Packages  []PackageNode `json:"packages"`
	Symbols   []SymbolNode  `json:"symbols"`
	Imports   []ImportEdge  `json:"imports"`
	Skipped   []string      `json:"skipped_adapters,omitempty"`
}

func scanSnapshot(s rowScanner) (GraphSnapshot, error) {
	var (
		snap         GraphSnapshot
		stored       []byte
		codec        string
		extractedRaw string
	)
	if err := s.Scan(
		&snap.ID, &snap.Scenario, &snap.ContentHash, &snap.SourceFingerprint,
		&stored, &codec, &extractedRaw, &snap.ExtractionMS,
	); err != nil {
		return GraphSnapshot{}, err
	}
	payload, err := decodePayload(stored, codec)
	if err != nil {
		return GraphSnapshot{}, fmt.Errorf("decode payload for snapshot %q: %w", snap.ID, err)
	}
	extractedAt, err := time.Parse(snapshotTimeFormat, extractedRaw)
	if err != nil {
		return GraphSnapshot{}, fmt.Errorf("parse extracted_at: %w", err)
	}
	snap.ExtractedAt = extractedAt

	var p snapshotPayload
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &p); err != nil {
			return GraphSnapshot{}, fmt.Errorf("decode payload: %w", err)
		}
		snap.Languages = p.Languages
		snap.Files = p.Files
		snap.Packages = p.Packages
		snap.Symbols = p.Symbols
		snap.Imports = p.Imports
		snap.SkippedAdapters = p.Skipped
	}
	return snap, nil
}

func scanSnapshotMeta(s rowScanner) (GraphSnapshotMeta, error) {
	var (
		meta         GraphSnapshotMeta
		extractedRaw string
	)
	if err := s.Scan(
		&meta.ID, &meta.Scenario, &meta.ContentHash, &meta.SourceFingerprint,
		&extractedRaw, &meta.ExtractionMS, &meta.PayloadBytes,
	); err != nil {
		return GraphSnapshotMeta{}, err
	}
	t, err := time.Parse(snapshotTimeFormat, extractedRaw)
	if err != nil {
		return GraphSnapshotMeta{}, fmt.Errorf("parse extracted_at: %w", err)
	}
	meta.ExtractedAt = t
	return meta, nil
}

func (r *sqliteRepository) ensureSourceFingerprintColumn(ctx context.Context) error {
	if r.sourceFingerprintReady.Load() {
		return nil
	}
	if _, err := r.db.ExecContext(ctx, addSourceFingerprintColumnSQL); err != nil && !isDuplicateColumnError(err) {
		return fmt.Errorf("migrate graph_snapshots.source_fingerprint: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, createSourceFingerprintIndexSQL); err != nil {
		return fmt.Errorf("migrate graph_snapshots source_fingerprint index: %w", err)
	}
	// Existing rows get an empty codec from the column default, which the read
	// path already understands as legacy raw JSON. Nothing is rewritten.
	if _, err := r.db.ExecContext(ctx, addPayloadCodecColumnSQL); err != nil && !isDuplicateColumnError(err) {
		return fmt.Errorf("migrate graph_snapshots.payload_codec: %w", err)
	}
	r.sourceFingerprintReady.Store(true)
	return nil
}

func isDuplicateColumnError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column") || strings.Contains(msg, "already exists")
}
