package conversationsearch

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

const (
	telemetryRetention    = 30 * 24 * time.Hour
	telemetryMaxRows      = 100000
	telemetryReclaimEvery = 256
)

// SearchTelemetry contains only categorical and aggregate request evidence.
// It deliberately has no query, snippet, content, regex, or raw-path field.
type SearchTelemetry struct {
	RequestID           string
	SessionToken        string
	Mode                string
	Sort                string
	FilterFamilies      []string
	Duration            time.Duration
	CandidateCount      int
	ResultCount         int
	ResultStableHitIDs  []string
	WeakOnly            bool
	DegradationReasons  []string
	FreshnessBand       string
	ErrorCategory       string
	LexicalContributed  bool
	SemanticContributed bool
	CreatedAt           time.Time
}

type SearchInteraction struct {
	RequestID    string
	SessionToken string
	Reformulated bool
	SelectedRank int
	StableHitID  string
}

type TelemetryAggregate struct {
	Queries, NoResult, WeakOnly, Reformulated, Selected, Degraded, Errors int64
	LexicalContributed, SemanticContributed                               int64
	P50LatencyMS, P95LatencyMS, P50SelectedRank, P95SelectedRank          float64
	Truncated                                                             bool
}

type TelemetryRepository interface {
	AppendSearchTelemetry(context.Context, SearchTelemetry) error
	RecordSearchInteraction(context.Context, SearchInteraction) (bool, error)
	AggregateSearchTelemetry(context.Context, time.Time, time.Time, int) (TelemetryAggregate, error)
	ReclaimSearchTelemetry(context.Context, time.Time, int) (int64, error)
}

func newTelemetryID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("search-%d", time.Now().UnixNano())
	}
	return "search-" + hex.EncodeToString(buf)
}

func hashTelemetrySession(key []byte, token string) string {
	if token == "" {
		return ""
	}
	digest := hmac.New(sha256.New, key)
	_, _ = digest.Write([]byte(token))
	return hex.EncodeToString(digest.Sum(nil))
}

func (s *Service) RecordSearchTelemetry(ctx context.Context, record SearchTelemetry) (string, error) {
	if record.RequestID == "" {
		record.RequestID = newTelemetryID()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	record.SessionToken = hashTelemetrySession(s.telemetryKey, record.SessionToken)
	if s.telemetry == nil {
		return record.RequestID, nil
	}
	if err := s.telemetry.AppendSearchTelemetry(ctx, record); err != nil {
		return record.RequestID, err
	}
	if s.telemetryAppends.Add(1)%telemetryReclaimEvery == 0 {
		_, _ = s.telemetry.ReclaimSearchTelemetry(context.WithoutCancel(ctx), time.Now().UTC().Add(-telemetryRetention), telemetryMaxRows)
	}
	return record.RequestID, nil
}

func (s *Service) RecordSearchInteraction(ctx context.Context, interaction SearchInteraction) (bool, error) {
	interaction.SessionToken = hashTelemetrySession(s.telemetryKey, interaction.SessionToken)
	if s.telemetry == nil {
		return false, nil
	}
	return s.telemetry.RecordSearchInteraction(ctx, interaction)
}

func (s *Service) AggregateSearchTelemetry(ctx context.Context, from, to time.Time, limit int) (TelemetryAggregate, error) {
	if s.telemetry == nil {
		return TelemetryAggregate{}, errorsUnavailable("conversation search telemetry repository is unavailable")
	}
	return s.telemetry.AggregateSearchTelemetry(ctx, from, to, limit)
}

func errorsUnavailable(message string) error { return fmt.Errorf("%s", message) }

func (r *SQLiteRepository) AppendSearchTelemetry(ctx context.Context, record SearchTelemetry) error {
	filters, err := json.Marshal(compactSorted(record.FilterFamilies))
	if err != nil {
		return fmt.Errorf("encode conversation search filter families: %w", err)
	}
	degradations, err := json.Marshal(compactSorted(record.DegradationReasons))
	if err != nil {
		return fmt.Errorf("encode conversation search degradations: %w", err)
	}
	resultIDs, err := json.Marshal(compactSorted(record.ResultStableHitIDs))
	if err != nil {
		return fmt.Errorf("encode conversation search result identifiers: %w", err)
	}
	created := record.CreatedAt.UTC().Format(time.RFC3339Nano)
	_, err = r.db.ExecContext(ctx, `INSERT INTO conversation_search_telemetry (
request_id, session_hash, mode, sort_order, filter_families_json, duration_ms,
candidate_count, result_count, result_stable_hit_ids_json, weak_only, degradation_reasons_json, freshness_band,
error_category, lexical_contributed, semantic_contributed, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, record.RequestID, record.SessionToken,
		record.Mode, record.Sort, string(filters), maxInt64(0, record.Duration.Milliseconds()),
		maxInt(0, record.CandidateCount), maxInt(0, record.ResultCount), string(resultIDs), boolInt(record.WeakOnly), string(degradations),
		record.FreshnessBand, record.ErrorCategory, boolInt(record.LexicalContributed), boolInt(record.SemanticContributed), created, created)
	if err != nil {
		return fmt.Errorf("append conversation search telemetry: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) RecordSearchInteraction(ctx context.Context, interaction SearchInteraction) (bool, error) {
	if interaction.RequestID == "" {
		return false, ErrInvalidRequest
	}
	var result sql.Result
	var err error
	now := time.Now().UTC().Format(time.RFC3339Nano)
	match := `request_id = ? AND session_hash = ?`
	if interaction.Reformulated {
		result, err = r.db.ExecContext(ctx, `UPDATE conversation_search_telemetry SET reformulated = 1, updated_at = ? WHERE `+match,
			now, interaction.RequestID, interaction.SessionToken)
	} else {
		if interaction.SelectedRank < 1 || interaction.SelectedRank > 100 || interaction.StableHitID == "" {
			return false, ErrInvalidRequest
		}
		result, err = r.db.ExecContext(ctx, `UPDATE conversation_search_telemetry SET selected_rank = ?, selected_stable_hit_id = ?, updated_at = ? WHERE `+match+` AND EXISTS (SELECT 1 FROM json_each(result_stable_hit_ids_json) WHERE value = ?)`,
			interaction.SelectedRank, interaction.StableHitID, now, interaction.RequestID, interaction.SessionToken, interaction.StableHitID)
	}
	if err != nil {
		return false, fmt.Errorf("record conversation search interaction: %w", err)
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

func (r *SQLiteRepository) AggregateSearchTelemetry(ctx context.Context, from, to time.Time, limit int) (TelemetryAggregate, error) {
	if limit <= 0 || limit > 10000 {
		limit = 10000
	}
	args := []any{from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano), limit + 1}
	var rows []telemetryAggregateRow
	if err := r.db.SelectContext(ctx, &rows, `SELECT duration_ms, result_count, weak_only,
reformulated, selected_rank, lexical_contributed, semantic_contributed,
CASE WHEN degradation_reasons_json <> '[]' THEN 1 ELSE 0 END AS degraded,
CASE WHEN error_category <> '' THEN 1 ELSE 0 END AS errored
FROM conversation_search_telemetry WHERE created_at >= ? AND created_at < ?
ORDER BY created_at DESC, request_id DESC LIMIT ?`, args...); err != nil {
		return TelemetryAggregate{}, fmt.Errorf("aggregate conversation search telemetry: %w", err)
	}
	aggregate := TelemetryAggregate{Truncated: len(rows) > limit}
	if aggregate.Truncated {
		rows = rows[:limit]
	}
	latencies := make([]int64, 0, len(rows))
	ranks := make([]int64, 0, len(rows))
	for _, row := range rows {
		aggregate.Queries++
		if row.ResultCount == 0 {
			aggregate.NoResult++
		}
		aggregate.WeakOnly += row.WeakOnly
		aggregate.Reformulated += row.Reformulated
		aggregate.LexicalContributed += row.LexicalContributed
		aggregate.SemanticContributed += row.SemanticContributed
		aggregate.Degraded += row.Degraded
		aggregate.Errors += row.Errored
		latencies = append(latencies, row.DurationMS)
		if row.SelectedRank.Valid {
			aggregate.Selected++
			ranks = append(ranks, row.SelectedRank.Int64)
		}
	}
	aggregate.P50LatencyMS, aggregate.P95LatencyMS = percentilePair(latencies)
	aggregate.P50SelectedRank, aggregate.P95SelectedRank = percentilePair(ranks)
	return aggregate, nil
}

func (r *SQLiteRepository) ReclaimSearchTelemetry(ctx context.Context, before time.Time, maxRows int) (int64, error) {
	if maxRows <= 0 || maxRows > telemetryMaxRows {
		maxRows = telemetryMaxRows
	}
	result, err := r.db.ExecContext(ctx, `DELETE FROM conversation_search_telemetry
WHERE created_at < ? OR request_id IN (
  SELECT request_id FROM conversation_search_telemetry
  ORDER BY created_at DESC, request_id DESC LIMIT -1 OFFSET ?
)`, before.UTC().Format(time.RFC3339Nano), maxRows)
	if err != nil {
		return 0, fmt.Errorf("reclaim conversation search telemetry: %w", err)
	}
	rows, err := result.RowsAffected()
	return rows, err
}

type telemetryAggregateRow struct {
	DurationMS          int64         `db:"duration_ms"`
	ResultCount         int64         `db:"result_count"`
	WeakOnly            int64         `db:"weak_only"`
	Reformulated        int64         `db:"reformulated"`
	SelectedRank        sql.NullInt64 `db:"selected_rank"`
	LexicalContributed  int64         `db:"lexical_contributed"`
	SemanticContributed int64         `db:"semantic_contributed"`
	Degraded            int64         `db:"degraded"`
	Errored             int64         `db:"errored"`
}

func percentilePair(values []int64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return float64(values[percentileIndex(len(values), 0.50)]), float64(values[percentileIndex(len(values), 0.95)])
}

func percentileIndex(length int, percentile float64) int {
	index := int(float64(length)*percentile+0.999999) - 1
	if index < 0 {
		return 0
	}
	if index >= length {
		return length - 1
	}
	return index
}

func compactSorted(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != "" {
			set[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
