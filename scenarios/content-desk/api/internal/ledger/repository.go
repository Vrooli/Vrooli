package ledger

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	RecordPublish(context.Context, PublishRecord) (PublishRecord, error)
	ListPublishHistory(context.Context, int) ([]PublishRecord, error)
	SubjectFamiliarity(context.Context, string, string) (SubjectFamiliarity, error)
	NarratedForScenario(context.Context, string) ([]NarratedItem, error)
	ContaminatedByClaim(context.Context, string) ([]PublishRecord, error)
	Coverage(context.Context, time.Duration) ([]CoverageCell, error)
}

func (r *sqliteRepository) Coverage(ctx context.Context, staleAfter time.Duration) ([]CoverageCell, error) {
	if staleAfter <= 0 {
		staleAfter = 30 * 24 * time.Hour
	}
	rows, err := r.db.QueryContext(ctx, `SELECT d.campaign_id, d.lane, p.channel, d.sku, COUNT(*), MAX(p.published_at)
		FROM ledger_publish_records p JOIN drafts d ON d.id = p.draft_id
		GROUP BY d.campaign_id, d.lane, p.channel, d.sku ORDER BY d.campaign_id, d.lane, p.channel, d.sku`)
	if err != nil {
		return nil, fmt.Errorf("list coverage: %w", err)
	}
	defer rows.Close()
	cutoff := time.Now().UTC().Add(-staleAfter)
	var out []CoverageCell
	for rows.Next() {
		var cell CoverageCell
		var raw string
		if err := rows.Scan(&cell.CampaignID, &cell.Lane, &cell.Channel, &cell.SKU, &cell.PublishCount, &raw); err != nil {
			return nil, err
		}
		if cell.LastPublishedAt, err = time.Parse(time.RFC3339Nano, raw); err != nil {
			return nil, err
		}
		cell.Stale = cell.LastPublishedAt.Before(cutoff)
		out = append(out, cell)
	}
	return out, rows.Err()
}

// ContaminatedByClaim lists every published record whose draft cites the
// supplied claim. Callers use it when a verification check changes result;
// it intentionally returns history rather than changing publication state.
func (r *sqliteRepository) ContaminatedByClaim(ctx context.Context, claimID string) ([]PublishRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT p.id, COALESCE(p.draft_id,''), COALESCE(p.series_id,''), p.channel, p.audience, p.published_url, p.platform_post_id, p.source_kind, p.published_at FROM ledger_publish_records p JOIN claim_citations c ON c.draft_id = p.draft_id WHERE c.claim_id = ? ORDER BY p.published_at DESC, p.id DESC`, claimID)
	if err != nil {
		return nil, fmt.Errorf("list contaminated publish records: %w", err)
	}
	defer rows.Close()
	var out []PublishRecord
	for rows.Next() {
		var record PublishRecord
		var raw string
		if err := rows.Scan(&record.ID, &record.DraftID, &record.SeriesID, &record.Channel, &record.Audience, &record.PublishedURL, &record.PlatformPostID, &record.SourceKind, &raw); err != nil {
			return nil, err
		}
		if record.PublishedAt, err = time.Parse(time.RFC3339Nano, raw); err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) RecordPublish(ctx context.Context, record PublishRecord) (PublishRecord, error) {
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.PublishedAt.IsZero() {
		record.PublishedAt = time.Now().UTC()
	}
	if record.SourceKind == "" {
		record.SourceKind = "gated"
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO ledger_publish_records
      (id, draft_id, series_id, channel, audience, published_url, platform_post_id, source_kind, published_at, payload_json)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '{}')`, record.ID, nullable(record.DraftID), nullable(record.SeriesID), record.Channel, record.Audience, record.PublishedURL, record.PlatformPostID, record.SourceKind, record.PublishedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return PublishRecord{}, fmt.Errorf("record publish: %w", err)
	}
	return record, nil
}

func (r *sqliteRepository) ListPublishHistory(ctx context.Context, limit int) ([]PublishRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, COALESCE(draft_id,''), COALESCE(series_id,''), channel, audience, published_url, platform_post_id, source_kind, published_at FROM ledger_publish_records ORDER BY published_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list publish history: %w", err)
	}
	defer rows.Close()
	var out []PublishRecord
	for rows.Next() {
		var record PublishRecord
		var raw string
		if err := rows.Scan(&record.ID, &record.DraftID, &record.SeriesID, &record.Channel, &record.Audience, &record.PublishedURL, &record.PlatformPostID, &record.SourceKind, &raw); err != nil {
			return nil, fmt.Errorf("scan publish history: %w", err)
		}
		parsed, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return nil, fmt.Errorf("parse publish timestamp: %w", err)
		}
		record.PublishedAt = parsed
		out = append(out, record)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) SubjectFamiliarity(ctx context.Context, subject, audience string) (SubjectFamiliarity, error) {
	result := SubjectFamiliarity{Subject: subject, Audience: audience}
	var last sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(MAX(is_first_mention), 0), MAX(occurred_at) FROM ledger_subject_mentions WHERE subject = ? AND audience = ?`, subject, audience).Scan(&result.MentionCount, &result.FirstMention, &last)
	if err != nil {
		return SubjectFamiliarity{}, fmt.Errorf("subject familiarity: %w", err)
	}
	if last.Valid {
		result.LastMentionAt, err = time.Parse(time.RFC3339Nano, last.String)
		if err != nil {
			return SubjectFamiliarity{}, fmt.Errorf("parse mention time: %w", err)
		}
	}
	return result, nil
}

func (r *sqliteRepository) NarratedForScenario(ctx context.Context, scenario string) ([]NarratedItem, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, subject, scenario, occurred_at FROM ledger_narrated_items WHERE scenario = ? ORDER BY occurred_at DESC, id DESC`, scenario)
	if err != nil {
		return nil, fmt.Errorf("list narrated items: %w", err)
	}
	defer rows.Close()
	var out []NarratedItem
	for rows.Next() {
		var item NarratedItem
		var raw string
		if err := rows.Scan(&item.ID, &item.Subject, &item.Scenario, &raw); err != nil {
			return nil, err
		}
		item.OccurredAt, err = time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}
