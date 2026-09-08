package brief

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	agentbrief "github.com/vrooli/agentbrief-go"
	"github.com/vrooli/api-core/schedule"
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

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	if clock == nil {
		clock = schedule.System()
	}
	return &sqliteRepository{db: db, clock: clock}
}

func (r *sqliteRepository) Save(ctx context.Context, record Record) error {
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = r.clock.Now().UTC()
	}
	providers, err := json.Marshal(record.QueriedProviders)
	if err != nil {
		return fmt.Errorf("encode brief providers: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO briefs(id,consumer,verdict,reason,chat_id,message_id,harness,session_ref,prompt_digest,effective_query,rendered,queried_providers_json,max_trust_class,degraded,latency_ms,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, record.ID, record.Consumer, record.Verdict, record.Reason, record.ChatID, record.MessageID, record.Harness, record.SessionRef, record.PromptDigest, record.EffectiveQuery, record.Rendered, string(providers), record.MaxTrustClass, boolInt(record.Degraded), record.LatencyMS, record.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save brief: %w", err)
	}
	for index, item := range record.Items {
		if _, err := r.db.ExecContext(ctx, `INSERT INTO brief_items(brief_id,item_index,provider_id,type,title,snippet,path,score,rerank_score,trust_class,suggested_command) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, record.ID, index, item.ProviderID, item.Type, item.Title, item.Snippet, item.Path, item.Score, item.RerankScore, item.TrustClass, item.SuggestedCommand); err != nil {
			return fmt.Errorf("save brief item: %w", err)
		}
	}
	return nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Record, error) {
	var record Record
	var degraded int
	var created string
	var providersJSON string
	err := r.db.QueryRowContext(ctx, `SELECT id,consumer,verdict,reason,COALESCE(chat_id,''),COALESCE(message_id,''),harness,session_ref,prompt_digest,effective_query,rendered,queried_providers_json,max_trust_class,degraded,latency_ms,created_at FROM briefs WHERE id=?`, id).Scan(&record.ID, &record.Consumer, &record.Verdict, &record.Reason, &record.ChatID, &record.MessageID, &record.Harness, &record.SessionRef, &record.PromptDigest, &record.EffectiveQuery, &record.Rendered, &providersJSON, &record.MaxTrustClass, &degraded, &record.LatencyMS, &created)
	if err != nil {
		return Record{}, err
	}
	record.Degraded = degraded != 0
	if strings.TrimSpace(providersJSON) != "" {
		if err := json.Unmarshal([]byte(providersJSON), &record.QueriedProviders); err != nil {
			return Record{}, fmt.Errorf("decode brief providers: %w", err)
		}
	}
	record.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return Record{}, err
	}
	if err = r.readItems(ctx, &record); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (r *sqliteRepository) List(ctx context.Context, input ListInput) ([]Record, error) {
	limit := input.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `SELECT id FROM briefs WHERE 1=1`
	args := []any{}
	if input.Consumer != "" {
		query += ` AND consumer=?`
		args = append(args, input.Consumer)
	}
	if strings.TrimSpace(input.ChatID) != "" {
		query += ` AND chat_id=?`
		args = append(args, input.ChatID)
	}
	if strings.TrimSpace(input.SessionRef) != "" {
		query += ` AND session_ref=?`
		args = append(args, input.SessionRef)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]Record, 0, len(ids))
	for _, id := range ids {
		record, err := r.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, nil
}

func (r *sqliteRepository) readItems(ctx context.Context, record *Record) error {
	rows, err := r.db.QueryContext(ctx, `SELECT provider_id,type,title,snippet,path,score,rerank_score,trust_class,suggested_command FROM brief_items WHERE brief_id=? ORDER BY item_index`, record.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item agentbrief.Item
		if err := rows.Scan(&item.ProviderID, &item.Type, &item.Title, &item.Snippet, &item.Path, &item.Score, &item.RerankScore, &item.TrustClass, &item.SuggestedCommand); err != nil {
			return err
		}
		record.Items = append(record.Items, item)
		record.QueriedProviders = appendUnique(record.QueriedProviders, item.ProviderID)
	}
	return rows.Err()
}

func (r *sqliteRepository) RecordUse(ctx context.Context, input UseInput, at time.Time) (bool, error) {
	result, err := r.db.ExecContext(ctx, `INSERT INTO brief_uses(brief_id,item_index,kind,occurred_at) SELECT ?,?,?,? WHERE (? = -1 OR EXISTS(SELECT 1 FROM brief_items WHERE brief_id=? AND item_index=?)) AND NOT EXISTS(SELECT 1 FROM brief_uses WHERE brief_id=? AND item_index=? AND kind=?)`, input.BriefID, input.ItemIndex, input.Kind, at.UTC().Format(time.RFC3339Nano), input.ItemIndex, input.BriefID, input.ItemIndex, input.BriefID, input.ItemIndex, input.Kind)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count > 0, err
}

func (r *sqliteRepository) Stats(ctx context.Context, input StatsInput, now time.Time) ([]StatsRow, error) {
	window := input.WindowDays
	if window <= 0 {
		window = 7
	}
	cutoff := now.UTC().Add(-time.Duration(window) * 24 * time.Hour).Format(time.RFC3339Nano)
	query := `SELECT consumer, verdict, COUNT(*) FROM briefs WHERE created_at >= ?`
	args := []any{cutoff}
	if input.Consumer != "" {
		query += " AND consumer=?"
		args = append(args, input.Consumer)
	}
	query += " GROUP BY consumer, verdict ORDER BY consumer, verdict"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byConsumer := map[agentbrief.Consumer]*StatsRow{}
	order := []agentbrief.Consumer{}
	for rows.Next() {
		var consumer agentbrief.Consumer
		var verdict agentbrief.Verdict
		var count int64
		if err := rows.Scan(&consumer, &verdict, &count); err != nil {
			return nil, err
		}
		row := byConsumer[consumer]
		if row == nil {
			row = &StatsRow{Consumer: consumer, WithheldByVerdict: map[agentbrief.Verdict]int64{}}
			byConsumer[consumer] = row
			order = append(order, consumer)
		}
		row.BriefsBuilt += count
		if verdict == agentbrief.VerdictDeliver {
			row.BriefsDelivered += count
		} else {
			row.WithheldByVerdict[verdict] += count
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, consumer := range order {
		row := byConsumer[consumer]
		if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM brief_items i JOIN briefs b ON b.id=i.brief_id WHERE b.consumer=? AND b.created_at >= ?`, consumer, cutoff).Scan(&row.ItemsDelivered); err != nil {
			return nil, err
		}
		if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM brief_uses u JOIN briefs b ON b.id=u.brief_id WHERE b.consumer=? AND b.created_at >= ? AND u.item_index >= 0`, consumer, cutoff).Scan(&row.ItemsUsed); err != nil {
			return nil, err
		}
		if row.ItemsDelivered > 0 {
			row.UsageRate = float64(row.ItemsUsed) / float64(row.ItemsDelivered)
		}
		if row.BriefsBuilt > 0 {
			row.WithheldRate = float64(row.BriefsBuilt-row.BriefsDelivered) / float64(row.BriefsBuilt)
		}
	}
	out := make([]StatsRow, 0, len(order))
	for _, consumer := range order {
		out = append(out, *byConsumer[consumer])
	}
	return out, nil
}

func (r *sqliteRepository) DeleteBefore(ctx context.Context, cutoff time.Time) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM briefs WHERE created_at < ?`, cutoff.UTC().Format(time.RFC3339Nano))
	return err
}
func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	if value != "" {
		return append(values, value)
	}
	return values
}
