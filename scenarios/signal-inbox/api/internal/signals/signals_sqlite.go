package signals

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

func NewSQLiteRepository(db SQLExecutor, clk schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

const signalTimeFormat = time.RFC3339Nano

func (r *sqliteRepository) Append(ctx context.Context, signal Signal) (CaptureResult, error) {
	if signal.ID == "" {
		signal.ID = uuid.NewString()
	}
	if signal.CapturedAt.IsZero() {
		signal.CapturedAt = r.clock.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO signal (id, source_kind, source_identity, source_url, raw_payload_ref, extracted_content, content_hash, needs_attention, captured_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		signal.ID, signal.SourceKind, signal.SourceIdentity, signal.SourceURL, signal.RawPayloadRef,
		signal.ExtractedContent, signal.ContentHash, signal.NeedsAttention, signal.CapturedAt.Format(signalTimeFormat))
	if err != nil {
		if isUniqueConstraint(err) {
			existing, getErr := r.getByHash(ctx, signal.ContentHash)
			if getErr != nil {
				return CaptureResult{}, fmt.Errorf("read duplicate signal: %w", getErr)
			}
			return CaptureResult{Signal: existing, Duplicate: true}, nil
		}
		return CaptureResult{}, fmt.Errorf("append signal: %w", err)
	}
	if signal.RawPayloadRef != "" {
		if _, err := r.db.ExecContext(ctx, `INSERT INTO signal_media (signal_id, payload_ref) VALUES (?, ?)`, signal.ID, signal.RawPayloadRef); err != nil {
			return CaptureResult{}, fmt.Errorf("append signal media: %w", err)
		}
	}
	if signal.CaptureNote != "" {
		if _, err := r.db.ExecContext(ctx, `INSERT INTO signal_annotations (id, signal_id, kind, body, created_at) VALUES (?, ?, 'capture_note', ?, ?)`, uuid.NewString(), signal.ID, signal.CaptureNote, signal.CapturedAt.Format(signalTimeFormat)); err != nil {
			return CaptureResult{}, fmt.Errorf("append capture annotation: %w", err)
		}
	}
	for _, tag := range signal.Tags {
		if _, err := r.db.ExecContext(ctx, `INSERT INTO signal_tag (signal_id, tag) VALUES (?, ?)`, signal.ID, tag); err != nil {
			return CaptureResult{}, fmt.Errorf("append signal tag: %w", err)
		}
	}
	return CaptureResult{Signal: signal}, nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Signal, error) {
	signal, err := scanSignal(r.db.QueryRowContext(ctx, signalSelect+` WHERE s.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Signal{}, ErrSignalNotFound{ID: id}
	}
	if err != nil {
		return Signal{}, fmt.Errorf("get signal %q: %w", id, err)
	}
	return signal, nil
}

func (r *sqliteRepository) getByHash(ctx context.Context, contentHash string) (Signal, error) {
	return scanSignal(r.db.QueryRowContext(ctx, signalSelect+` WHERE s.content_hash = ?`, contentHash))
}

func (r *sqliteRepository) List(ctx context.Context, limit int) ([]Signal, error) {
	rows, err := r.db.QueryContext(ctx, signalSelect+` ORDER BY s.captured_at DESC, s.id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list signals: %w", err)
	}
	defer rows.Close()
	var signals []Signal
	for rows.Next() {
		signal, err := scanSignal(rows)
		if err != nil {
			return nil, fmt.Errorf("scan signal: %w", err)
		}
		signals = append(signals, signal)
	}
	return signals, rows.Err()
}

const signalSelect = `
SELECT s.id, s.source_kind, s.source_identity, s.source_url, s.raw_payload_ref,
       s.extracted_content, s.content_hash, s.needs_attention, s.captured_at,
       COALESCE((SELECT a.body FROM signal_annotations a WHERE a.signal_id = s.id AND a.kind = 'capture_note' ORDER BY a.created_at ASC LIMIT 1), ''),
       COALESCE((SELECT group_concat(tag, char(31)) FROM signal_tag t WHERE t.signal_id = s.id), '')
FROM signal s`

type scanner interface{ Scan(...any) error }

func scanSignal(row scanner) (Signal, error) {
	var signal Signal
	var capturedAt string
	var tags string
	if err := row.Scan(&signal.ID, &signal.SourceKind, &signal.SourceIdentity, &signal.SourceURL, &signal.RawPayloadRef, &signal.ExtractedContent, &signal.ContentHash, &signal.NeedsAttention, &capturedAt, &signal.CaptureNote, &tags); err != nil {
		return Signal{}, err
	}
	parsed, err := time.Parse(signalTimeFormat, capturedAt)
	if err != nil {
		return Signal{}, fmt.Errorf("parse captured_at: %w", err)
	}
	signal.CapturedAt = parsed
	if tags != "" {
		signal.Tags = strings.Split(tags, "\x1f")
	}
	return signal, nil
}

func isUniqueConstraint(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}
