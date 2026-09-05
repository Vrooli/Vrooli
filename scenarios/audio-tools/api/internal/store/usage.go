package store

import (
	"context"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
)

// UsageRow is a single accounted operation.
type UsageRow struct {
	OperationID          string
	EmittedAt            time.Time
	Capability           string // stt | tts | summarize | audio
	Operation            string // transcribe | synthesize | summarize | transcode | ...
	ProviderTier         string
	ProviderID           string
	ModelID              string
	LatencyMs            float64
	CreditsCharged       int32
	PromptTokens         int32
	OutputTokens         int32
	AudioDurationSeconds float64
	Error                string
	FallbackReason       string
	UserIdentity         string
}

// UsageSummary is the aggregated view returned by GetSummary.
type UsageSummary struct {
	Since           time.Time
	Until           time.Time
	OperationsTotal int64
	CreditsTotal    int64
	Distribution    []ProviderDist
	FallbackReasons []FallbackReasonCount
	ErrorCount      int64
}

type ProviderDist struct {
	ProviderTier string
	ProviderID   string
	Count        int64
	CreditsTotal int64
	AvgLatencyMs float64
}

type FallbackReasonCount struct {
	Reason string
	Count  int64
}

type UsageStore struct{ db *database.RoutedDB }

func NewUsageStore(db *database.RoutedDB) *UsageStore { return &UsageStore{db: db} }

// Insert is idempotent on OperationID. Re-inserts with the same id are
// silently ignored so async retries cannot create duplicates.
func (s *UsageStore) Insert(ctx context.Context, r UsageRow) error {
	if r.EmittedAt.IsZero() {
		r.EmittedAt = now()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO usage_rows(
			operation_id, emitted_at, capability, operation, provider_tier, provider_id, model_id,
			latency_ms, credits_charged, prompt_tokens, output_tokens, audio_duration_seconds,
			error, fallback_reason, user_identity
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`,
		r.OperationID, r.EmittedAt.UTC().Format(time.RFC3339Nano), r.Capability, r.Operation,
		r.ProviderTier, r.ProviderID, r.ModelID,
		r.LatencyMs, r.CreditsCharged, r.PromptTokens, r.OutputTokens, r.AudioDurationSeconds,
		r.Error, r.FallbackReason, r.UserIdentity,
	)
	return err
}

// ListRecent returns rows newer than `since`, optionally filtered by
// capability and provider tier, ordered newest-first.
func (s *UsageStore) ListRecent(ctx context.Context, since time.Time, limit int, capability, providerTier string) ([]UsageRow, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	q := strings.Builder{}
	q.WriteString(`SELECT operation_id, emitted_at, capability, operation, provider_tier, provider_id, model_id,
		latency_ms, credits_charged, prompt_tokens, output_tokens, audio_duration_seconds,
		error, fallback_reason, user_identity
		FROM usage_rows WHERE emitted_at >= ?`)
	args := []any{since.UTC().Format(time.RFC3339Nano)}
	if capability != "" {
		q.WriteString(" AND capability = ?")
		args = append(args, capability)
	}
	if providerTier != "" {
		q.WriteString(" AND provider_tier = ?")
		args = append(args, providerTier)
	}
	q.WriteString(" ORDER BY emitted_at DESC LIMIT ?")
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, q.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UsageRow
	for rows.Next() {
		var r UsageRow
		var emitted string
		if err := rows.Scan(&r.OperationID, &emitted, &r.Capability, &r.Operation,
			&r.ProviderTier, &r.ProviderID, &r.ModelID,
			&r.LatencyMs, &r.CreditsCharged, &r.PromptTokens, &r.OutputTokens, &r.AudioDurationSeconds,
			&r.Error, &r.FallbackReason, &r.UserIdentity); err != nil {
			return nil, err
		}
		r.EmittedAt, _ = time.Parse(time.RFC3339Nano, emitted)
		out = append(out, r)
	}
	return out, rows.Err()
}

// Summary aggregates rows in [since, now] for the optional capability.
func (s *UsageStore) Summary(ctx context.Context, since time.Time, capability string) (UsageSummary, error) {
	until := now()
	out := UsageSummary{Since: since.UTC(), Until: until}

	args := []any{since.UTC().Format(time.RFC3339Nano)}
	whereCap := ""
	if capability != "" {
		whereCap = " AND capability = ?"
		args = append(args, capability)
	}

	totalsQuery := `SELECT COUNT(*), COALESCE(SUM(credits_charged),0),
		COALESCE(SUM(CASE WHEN error<>'' THEN 1 ELSE 0 END),0)
		FROM usage_rows WHERE emitted_at >= ?` + whereCap
	if err := s.db.QueryRowContext(ctx, totalsQuery, args...).Scan(&out.OperationsTotal, &out.CreditsTotal, &out.ErrorCount); err != nil {
		return UsageSummary{}, err
	}

	distQuery := `SELECT provider_tier, provider_id, COUNT(*), COALESCE(SUM(credits_charged),0), COALESCE(AVG(latency_ms),0)
		FROM usage_rows WHERE emitted_at >= ?` + whereCap +
		` GROUP BY provider_tier, provider_id ORDER BY COUNT(*) DESC`
	dRows, err := s.db.QueryContext(ctx, distQuery, args...)
	if err != nil {
		return UsageSummary{}, err
	}
	defer dRows.Close()
	for dRows.Next() {
		var d ProviderDist
		if err := dRows.Scan(&d.ProviderTier, &d.ProviderID, &d.Count, &d.CreditsTotal, &d.AvgLatencyMs); err != nil {
			return UsageSummary{}, err
		}
		out.Distribution = append(out.Distribution, d)
	}
	if err := dRows.Err(); err != nil {
		return UsageSummary{}, err
	}

	fbQuery := `SELECT fallback_reason, COUNT(*) FROM usage_rows
		WHERE emitted_at >= ? AND fallback_reason <> ''` + whereCap +
		` GROUP BY fallback_reason ORDER BY COUNT(*) DESC`
	fRows, err := s.db.QueryContext(ctx, fbQuery, args...)
	if err != nil {
		return UsageSummary{}, err
	}
	defer fRows.Close()
	for fRows.Next() {
		var f FallbackReasonCount
		if err := fRows.Scan(&f.Reason, &f.Count); err != nil {
			return UsageSummary{}, err
		}
		out.FallbackReasons = append(out.FallbackReasons, f)
	}
	return out, fRows.Err()
}
