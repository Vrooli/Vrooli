package enrichment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"signal-inbox/internal/signals"
)

const timeFormat = time.RFC3339Nano

type sqliteRepository struct{ db signals.SQLExecutor }

func NewSQLiteRepository(db signals.SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Append(ctx context.Context, record Record) error {
	if record.SignalID == "" {
		return ErrInvalidRecord{Reason: "signal id is required"}
	}
	if record.ContentUnits < 0 {
		return ErrInvalidRecord{Reason: "content units cannot be negative"}
	}
	if record.ContentUnits == 0 && record.ExtractedContent != "" {
		return ErrInvalidRecord{Reason: "zero-unit enrichment cannot store content"}
	}
	if record.ContentUnits > 0 && record.ExtractedContent == "" {
		return ErrInvalidRecord{Reason: "non-zero enrichment requires content"}
	}
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.AttemptedAt.IsZero() {
		record.AttemptedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO signal_enrichment (id, signal_id, extracted_content, content_units, needs_attention, attention_reason, attempted_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		record.ID, record.SignalID, record.ExtractedContent, record.ContentUnits, record.NeedsAttention, record.AttentionReason, record.AttemptedAt.UTC().Format(timeFormat))
	if err != nil {
		return fmt.Errorf("append enrichment: %w", err)
	}
	return nil
}

func (r *sqliteRepository) Latest(ctx context.Context, signalID string) (Record, bool, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, signal_id, extracted_content, content_units, needs_attention, attention_reason, attempted_at
FROM signal_enrichment WHERE signal_id = ?
ORDER BY attempted_at DESC, id DESC LIMIT 1`, signalID)
	var record Record
	var attemptedAt string
	err := row.Scan(&record.ID, &record.SignalID, &record.ExtractedContent, &record.ContentUnits, &record.NeedsAttention, &record.AttentionReason, &attemptedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, fmt.Errorf("read latest enrichment: %w", err)
	}
	parsed, err := time.Parse(timeFormat, attemptedAt)
	if err != nil {
		return Record{}, false, fmt.Errorf("parse enrichment attempted_at: %w", err)
	}
	record.AttemptedAt = parsed
	return record, true, nil
}
