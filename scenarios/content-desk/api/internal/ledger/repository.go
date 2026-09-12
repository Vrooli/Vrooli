package ledger

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	RecordPublish(context.Context, PublishRecord) (PublishRecord, error)
	RecordReleaseReceipt(context.Context, ReleaseReceipt) (PublishRecord, error)
	IngestMetricSample(context.Context, MetricSample) (MetricSample, error)
	CreateRemediation(context.Context, Remediation) (Remediation, error)
	ResolveRemediation(context.Context, string) (Remediation, error)
	ListRemediations(context.Context, string, bool) ([]Remediation, error)
	ListPublishHistory(context.Context, int) ([]PublishRecord, error)
	SubjectFamiliarity(context.Context, string, string) (SubjectFamiliarity, error)
	NarratedForScenario(context.Context, string) ([]NarratedItem, error)
	ContaminatedByClaim(context.Context, string) ([]PublishRecord, error)
	Coverage(context.Context, time.Duration) ([]CoverageCell, error)
}

func (r *sqliteRepository) CreateRemediation(ctx context.Context, remediation Remediation) (Remediation, error) {
	if remediation.PublishRecordID == "" || remediation.Kind == "" {
		return Remediation{}, fmt.Errorf("remediation requires publish record and kind")
	}
	if remediation.ID == "" {
		remediation.ID = uuid.NewString()
	}
	if remediation.Status == "" {
		remediation.Status = "open"
	}
	if remediation.CreatedAt.IsZero() {
		remediation.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO ledger_remediations(id, publish_record_id, kind, status, note, created_at, resolved_at) VALUES (?, ?, ?, ?, ?, ?, NULL)`, remediation.ID, remediation.PublishRecordID, remediation.Kind, remediation.Status, remediation.Note, remediation.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return Remediation{}, fmt.Errorf("create remediation: %w", err)
	}
	return remediation, nil
}

func (r *sqliteRepository) ResolveRemediation(ctx context.Context, id string) (Remediation, error) {
	when := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, `UPDATE ledger_remediations SET status = 'resolved', resolved_at = ? WHERE id = ? AND status <> 'resolved'`, when.Format(time.RFC3339Nano), id)
	if err != nil {
		return Remediation{}, fmt.Errorf("resolve remediation: %w", err)
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return Remediation{}, fmt.Errorf("open remediation %q not found", id)
	}
	var remediation Remediation
	var created string
	err = r.db.QueryRowContext(ctx, `SELECT id, publish_record_id, kind, status, note, created_at FROM ledger_remediations WHERE id = ?`, id).Scan(&remediation.ID, &remediation.PublishRecordID, &remediation.Kind, &remediation.Status, &remediation.Note, &created)
	if err != nil {
		return Remediation{}, err
	}
	remediation.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	remediation.ResolvedAt = when
	return remediation, nil
}

// ListRemediations keeps remediation history visible. When openOnly is true,
// it is the operator's active correction queue; history is never deleted.
func (r *sqliteRepository) ListRemediations(ctx context.Context, publishRecordID string, openOnly bool) ([]Remediation, error) {
	query := `SELECT id, publish_record_id, kind, status, note, created_at, resolved_at FROM ledger_remediations`
	var args []any
	clauses := make([]string, 0, 2)
	if publishRecordID != "" {
		clauses = append(clauses, "publish_record_id = ?")
		args = append(args, publishRecordID)
	}
	if openOnly {
		clauses = append(clauses, "status <> 'resolved'")
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY created_at DESC, id DESC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list remediations: %w", err)
	}
	defer rows.Close()
	var remediations []Remediation
	for rows.Next() {
		var remediation Remediation
		var created string
		var resolved sql.NullString
		if err := rows.Scan(&remediation.ID, &remediation.PublishRecordID, &remediation.Kind, &remediation.Status, &remediation.Note, &created, &resolved); err != nil {
			return nil, fmt.Errorf("scan remediation: %w", err)
		}
		if remediation.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
			return nil, fmt.Errorf("parse remediation creation: %w", err)
		}
		if resolved.Valid {
			if remediation.ResolvedAt, err = time.Parse(time.RFC3339Nano, resolved.String); err != nil {
				return nil, fmt.Errorf("parse remediation resolution: %w", err)
			}
		}
		remediations = append(remediations, remediation)
	}
	return remediations, rows.Err()
}

func (r *sqliteRepository) IngestMetricSample(ctx context.Context, sample MetricSample) (MetricSample, error) {
	if sample.SampleID == "" || sample.ReleaseID == "" || sample.DraftID == "" || sample.Metric == "" || sample.ObservedAt.IsZero() {
		return MetricSample{}, fmt.Errorf("metric sample requires identity, release, draft, metric, and observation time")
	}
	var existing MetricSample
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT sample_id, release_id, draft_id, metric, value, observed_at FROM ledger_metric_samples WHERE sample_id = ?`, sample.SampleID).Scan(&existing.SampleID, &existing.ReleaseID, &existing.DraftID, &existing.Metric, &existing.Value, &raw)
	if err == nil {
		existing.ObservedAt, err = time.Parse(time.RFC3339Nano, raw)
		return existing, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return MetricSample{}, fmt.Errorf("lookup metric sample: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO ledger_metric_samples(sample_id, release_id, draft_id, metric, value, observed_at, received_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, sample.SampleID, sample.ReleaseID, sample.DraftID, sample.Metric, sample.Value, sample.ObservedAt.UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return MetricSample{}, fmt.Errorf("ingest metric sample: %w", err)
	}
	return sample, nil
}

// RecordReleaseReceipt is Content Desk's idempotent inbox for Channel Manager
// publication outcomes. It intentionally accepts only completed or partial
// publication results, never an identity, session, or executor credential.
func (r *sqliteRepository) RecordReleaseReceipt(ctx context.Context, receipt ReleaseReceipt) (PublishRecord, error) {
	if receipt.ReceiptID == "" || receipt.DraftID == "" || receipt.PlatformPostID == "" || receipt.PublishedURL == "" {
		return PublishRecord{}, fmt.Errorf("release receipt requires id, draft, platform post id, and URL")
	}
	if receipt.Status != "published" && receipt.Status != "partial" {
		return PublishRecord{}, fmt.Errorf("release receipt status %q is not publishable", receipt.Status)
	}
	var existing PublishRecord
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT id, COALESCE(draft_id,''), COALESCE(series_id,''), channel, audience, published_url, platform_post_id, source_kind, published_at FROM ledger_publish_records WHERE import_key = ?`, "channel-manager:"+receipt.ReceiptID).Scan(&existing.ID, &existing.DraftID, &existing.SeriesID, &existing.Channel, &existing.Audience, &existing.PublishedURL, &existing.PlatformPostID, &existing.SourceKind, &raw)
	if err == nil {
		existing.PublishedAt, err = time.Parse(time.RFC3339Nano, raw)
		return existing, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return PublishRecord{}, fmt.Errorf("lookup release receipt: %w", err)
	}
	when := receipt.PublishedAt
	if when.IsZero() {
		when = time.Now().UTC()
	}
	record := PublishRecord{ID: uuid.NewString(), DraftID: receipt.DraftID, Channel: receipt.Channel, PublishedURL: receipt.PublishedURL, PlatformPostID: receipt.PlatformPostID, SourceKind: "channel-manager", PublishedAt: when}
	_, err = r.db.ExecContext(ctx, `INSERT INTO ledger_publish_records (id, import_key, draft_id, series_id, channel, audience, published_url, platform_post_id, source_kind, published_at, payload_json) VALUES (?, ?, ?, NULL, ?, '', ?, ?, ?, ?, ?)`, record.ID, "channel-manager:"+receipt.ReceiptID, record.DraftID, record.Channel, record.PublishedURL, record.PlatformPostID, record.SourceKind, record.PublishedAt.UTC().Format(time.RFC3339Nano), `{"status":"`+receipt.Status+`"}`)
	if err != nil {
		return PublishRecord{}, fmt.Errorf("record release receipt: %w", err)
	}
	return record, nil
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
