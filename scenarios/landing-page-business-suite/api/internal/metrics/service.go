// Package metrics owns event validation, idempotent ingestion, and analytics
// aggregation. HTTP handlers remain at the transport edge and depend on this
// package rather than holding metrics rules themselves.
package metrics

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"
)

type Service struct {
	db    Store
	clock schedule.Clock
}

// Store is the persistence boundary for metrics ingestion and reporting.
type Store interface {
	QueryRow(string, ...any) *sql.Row
	Query(string, ...any) (*sql.Rows, error)
	Exec(string, ...any) (sql.Result, error)
}

var validEventTypes = map[string]struct{}{
	"page_view": {}, "scroll_depth": {}, "click": {}, "form_submit": {},
	"conversion": {}, "download": {},
}

// ValidationError identifies a request field that violates the metrics contract.
type ValidationError struct{ Field, Reason string }

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

func NewService(db Store) *Service { return NewServiceWithClock(db, schedule.System()) }

func NewServiceWithClock(db Store, serviceClock schedule.Clock) *Service {
	if serviceClock == nil {
		serviceClock = schedule.System()
	}
	return &Service{db: db, clock: serviceClock}
}

type Event struct {
	EventType    string                 `json:"event_type"`
	VariantSlug  string                 `json:"variant_slug"`
	EventData    map[string]interface{} `json:"event_data,omitempty"`
	SessionID    string                 `json:"session_id"`
	VisitorID    string                 `json:"visitor_id,omitempty"`
	EventID      string                 `json:"event_id,omitempty"`
	ReferrerHost string                 `json:"referrer_host,omitempty"`
	ReferrerKind string                 `json:"referrer_kind,omitempty"`
	UTMSource    string                 `json:"utm_source,omitempty"`
	UTMMedium    string                 `json:"utm_medium,omitempty"`
	UTMCampaign  string                 `json:"utm_campaign,omitempty"`
	LandingPath  string                 `json:"landing_path,omitempty"`
	CountryCode  string                 `json:"country_code,omitempty"`
	DeviceClass  string                 `json:"device_class,omitempty"`
}

type VariantStats struct {
	VariantSlug    string  `json:"variant_slug"`
	VariantName    string  `json:"variant_name"`
	Views          int64   `json:"views"`
	CTAClicks      int64   `json:"cta_clicks"`
	Conversions    int64   `json:"conversions"`
	Downloads      int64   `json:"downloads"`
	Exposures      int64   `json:"exposures"`
	ConversionRate float64 `json:"conversion_rate"`
	Trend          string  `json:"trend,omitempty"`
	AvgScrollDepth float64 `json:"avg_scroll_depth,omitempty"`
}

type AnalyticsSummary struct {
	TotalVisitors  int64          `json:"total_visitors"`
	TotalDownloads int64          `json:"total_downloads"`
	VariantStats   []VariantStats `json:"variant_stats"`
	TopCTA         string         `json:"top_cta,omitempty"`
	TopCTACTR      float64        `json:"top_cta_ctr,omitempty"`
	ObservedAt     *time.Time     `json:"observed_at,omitempty"`
}

type TrafficBreakdownRow struct {
	Key          string  `json:"key"`
	Label        string  `json:"label"`
	Sessions     int64   `json:"sessions"`
	Conversions  int64   `json:"conversions"`
	RevenueMinor int64   `json:"revenue_minor"`
	Share        float64 `json:"share"`
}

type TrafficBreakdown struct {
	Rows          []TrafficBreakdownRow `json:"rows"`
	TotalSessions int64                 `json:"total_sessions"`
	Exhaustive    bool                  `json:"exhaustive"`
	Currency      string                `json:"currency"`
	ObservedAt    time.Time             `json:"observed_at"`
}

type TrafficSeriesPoint struct {
	BucketStart time.Time
	Value       float64
}
type TrafficSeries struct {
	Points     []TrafficSeriesPoint
	Unit       string
	ObservedAt time.Time
}

func (s *Service) RecordExposure(visitorID, variantSlug, weightFingerprint string) error {
	if strings.TrimSpace(visitorID) == "" || strings.TrimSpace(variantSlug) == "" || strings.TrimSpace(weightFingerprint) == "" {
		return nil
	}
	_, err := s.db.Exec(`INSERT INTO experiment_exposures (visitor_id, variant_slug, weight_fingerprint) VALUES ($1, $2, $3) ON CONFLICT (visitor_id, variant_slug, weight_fingerprint) DO NOTHING`, visitorID, variantSlug, weightFingerprint)
	if err != nil {
		return fmt.Errorf("record experiment exposure: %w", err)
	}
	return nil
}

// AdminRevenue is the canonical producer-owned revenue projection. Monetary
// values are in the declared currency and unit; sample_size is active MRR
// subscriptions included in the rollup.
type AdminRevenue struct {
	MRR        float64    `json:"mrr"`
	MRRUnit    string     `json:"mrr_unit"`
	Today      float64    `json:"today"`
	TodayUnit  string     `json:"today_unit"`
	Currency   string     `json:"currency"`
	SampleSize int64      `json:"sample_size"`
	ObservedAt *time.Time `json:"observed_at"`
}

// RevenueSummary is the complete finance-owned aggregate. Money is expressed
// in minor settlement-currency units; rates are percentages from 0 to 100.
type RevenueSummary struct {
	Currency                   string     `json:"currency"`
	MRRUnit                    string     `json:"mrr_unit"`
	RevenueTodayUnit           string     `json:"revenue_today_unit"`
	RevenueWindowUnit          string     `json:"revenue_window_unit"`
	CreditUnit                 string     `json:"credit_unit"`
	CurrencyExcludedCount      int64      `json:"currency_excluded_count"`
	MRRMinor                   int64      `json:"mrr_minor"`
	RevenueTodayMinor          int64      `json:"revenue_today_minor"`
	RevenueWindowMinor         int64      `json:"revenue_window_minor"`
	ActiveSubscriptions        int64      `json:"active_subscriptions"`
	SubscriptionsChurnedWindow int64      `json:"subscriptions_churned_window"`
	ChurnRatePercent           float64    `json:"churn_rate_percent"`
	CreditBalanceTotal         int64      `json:"credit_balance_total"`
	CreditBurnedWindow         int64      `json:"credit_burned_window"`
	UsageRecordsWindow         int64      `json:"usage_records_window"`
	SampleSize                 int64      `json:"sample_size"`
	TrialsWithoutPaymentMethod int64      `json:"trials_without_payment_method"`
	ObservedAt                 *time.Time `json:"observed_at"`
}

// GetRevenueSummary computes the documented tenant-wide rollup in one
// producer-owned projection. The SQL keeps currency and sample counts
// explicit, so an empty tenant returns zeros rather than an error.
func (s *Service) GetRevenueSummary() (*RevenueSummary, error) {
	var out RevenueSummary
	var mrr, today, window float64
	var active, churned, trials, creditBalance, creditBurned, usage, currencies int64
	var currency string
	if err := s.db.QueryRow(`SELECT
		COALESCE(SUM(CASE WHEN sub.status = 'active' OR (sub.status = 'trialing' AND sub.customer_id IS NOT NULL)
			THEN GREATEST(0, (CASE WHEN bp.billing_interval = 'year' THEN CASE WHEN bp.intro_enabled THEN COALESCE(NULLIF(bp.intro_amount_cents, 0), bp.amount_cents) ELSE bp.amount_cents END / 12.0
				ELSE CASE WHEN bp.intro_enabled THEN COALESCE(NULLIF(bp.intro_amount_cents, 0), bp.amount_cents) ELSE bp.amount_cents END END)
				* (1 - COALESCE(NULLIF(bp.metadata->>'discount_percent', '')::numeric, 0) / 100)
				- COALESCE(NULLIF(bp.metadata->>'discount_amount_cents', '')::numeric, 0)) ELSE 0 END), 0),
		COALESCE(COUNT(*) FILTER (WHERE sub.status = 'active' OR (sub.status = 'trialing' AND sub.customer_id IS NOT NULL)), 0),
		COALESCE(COUNT(*) FILTER (WHERE sub.status = 'trialing' AND sub.customer_id IS NULL), 0),
		COALESCE((SELECT NULLIF(bp2.currency, '') FROM subscriptions sub2 LEFT JOIN bundle_prices bp2 ON bp2.stripe_price_id = sub2.price_id
			WHERE sub2.status IN ('active','trialing') AND COALESCE(bp2.billing_interval, 'one_time') IN ('month','year')
			GROUP BY bp2.currency ORDER BY COUNT(*) DESC, bp2.currency LIMIT 1), 'usd'),
		COALESCE(COUNT(DISTINCT NULLIF(bp.currency, '')), 0)
		FROM subscriptions sub LEFT JOIN bundle_prices bp ON bp.stripe_price_id = sub.price_id
		WHERE sub.status IN ('active','trialing') AND COALESCE(bp.billing_interval, 'one_time') IN ('month','year')`).Scan(&mrr, &active, &trials, &currency, &currencies); err != nil {
		return nil, fmt.Errorf("compute revenue summary subscriptions: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(amount_cents), 0) FROM checkout_sessions WHERE status IN ('paid','complete') AND created_at >= CURRENT_DATE`).Scan(&today); err != nil {
		return nil, fmt.Errorf("compute revenue summary today: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(amount_cents), 0) FROM checkout_sessions WHERE status IN ('paid','complete') AND created_at >= CURRENT_DATE - INTERVAL '30 days'`).Scan(&window); err != nil {
		return nil, fmt.Errorf("compute revenue summary window: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FILTER (WHERE canceled_at >= CURRENT_DATE - INTERVAL '30 days') FROM subscriptions WHERE canceled_at IS NOT NULL`).Scan(&churned); err != nil {
		return nil, fmt.Errorf("compute revenue summary churn: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(balance_credits + bonus_credits), 0) FROM credit_wallets`).Scan(&creditBalance); err != nil {
		return nil, fmt.Errorf("compute revenue summary credits: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(ABS(amount_credits)), 0) FROM credit_transactions WHERE transaction_type IN ('usage','debit','consumption') AND created_at >= CURRENT_DATE - INTERVAL '30 days'`).Scan(&creditBurned); err != nil {
		return nil, fmt.Errorf("compute revenue summary credit usage: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM usage_records WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'`).Scan(&usage); err != nil {
		return nil, fmt.Errorf("compute revenue summary usage: %w", err)
	}
	if active+churned > 0 {
		out.ChurnRatePercent = float64(churned) * 100 / float64(active+churned)
	}
	out.Currency, out.MRRMinor, out.RevenueTodayMinor, out.RevenueWindowMinor = currency, int64(math.Round(mrr)), int64(math.Round(today)), int64(math.Round(window))
	out.MRRUnit, out.RevenueTodayUnit, out.RevenueWindowUnit, out.CreditUnit = "minor_currency", "minor_currency", "minor_currency", "credits"
	if currencies > 1 {
		out.CurrencyExcludedCount = currencies - 1
	}
	out.ActiveSubscriptions, out.SubscriptionsChurnedWindow, out.CreditBalanceTotal = active, churned, creditBalance
	out.CreditBurnedWindow, out.UsageRecordsWindow, out.SampleSize, out.TrialsWithoutPaymentMethod = creditBurned, usage, active, trials
	now := s.clock.Now().UTC()
	out.ObservedAt = &now
	return &out, nil
}

// GetAdminRevenue computes MRR once from active Stripe subscriptions. Annual
// plans are normalized to one twelfth of their price; one-time plans are
// excluded from MRR. This definition is documented in docs/concepts/MRR.md.
func (s *Service) GetAdminRevenue() (*AdminRevenue, error) {
	summary, err := s.GetRevenueSummary()
	if err != nil {
		return nil, err
	}
	return &AdminRevenue{MRR: float64(summary.MRRMinor) / 100, MRRUnit: "currency", Today: float64(summary.RevenueTodayMinor) / 100, TodayUnit: "currency", Currency: summary.Currency, SampleSize: summary.SampleSize, ObservedAt: summary.ObservedAt}, nil
}

func (s *Service) TrackEvent(event Event) error {
	if err := ValidateEvent(event); err != nil {
		return err
	}
	eventID := event.EventID
	if eventID == "" {
		eventID = GenerateEventIDAt(event, s.clock.Now())
	}

	var err error
	if event.EventData == nil {
		event.EventData = make(map[string]interface{})
	}
	event.EventData["event_id"] = eventID
	eventDataJSON, err := json.Marshal(event.EventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event_data: %w", err)
	}
	if _, err = s.db.Exec(`INSERT INTO metrics_events
		(variant_slug, event_type, event_data, event_id, session_id, visitor_id,
		 referrer_host, referrer_kind, utm_source, utm_medium, utm_campaign,
		 landing_path, country_code, device_class)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''), NULLIF($12, ''), NULLIF($13, ''), NULLIF($14, ''))
		ON CONFLICT (event_id) WHERE event_id IS NOT NULL DO NOTHING`, event.VariantSlug, event.EventType, eventDataJSON,
		eventID, event.SessionID, event.VisitorID, event.ReferrerHost, event.ReferrerKind,
		event.UTMSource, event.UTMMedium, event.UTMCampaign, event.LandingPath,
		event.CountryCode, event.DeviceClass); err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}
	return nil
}

// GetTrafficBreakdown returns one safe, parameterised projection for all
// supported traffic dimensions. The identifier is selected from a fixed map;
// callers cannot inject SQL through the dimension.
func (s *Service) GetTrafficBreakdown(dimension string, startDate, endDate time.Time, limit int) (*TrafficBreakdown, error) {
	columns := map[string]string{
		"country": "country_code", "referrer_kind": "referrer_kind", "utm_source": "utm_source",
		"utm_campaign": "utm_campaign", "device_class": "device_class", "landing_path": "landing_path", "variant": "variant_slug",
	}
	column, ok := columns[strings.ToLower(dimension)]
	if !ok {
		return nil, fmt.Errorf("unsupported traffic dimension %q", dimension)
	}
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	query := fmt.Sprintf(`WITH visitor_dimensions AS (
		SELECT DISTINCT COALESCE(NULLIF(%s, ''), 'unknown') AS dimension_key,
			COALESCE(NULLIF(visitor_id, ''), session_id) AS visitor_key
		FROM metrics_events WHERE created_at >= $1 AND created_at <= $2
	), grouped AS (
		SELECT COALESCE(NULLIF(e.%s, ''), 'unknown') AS dimension_key,
			COUNT(DISTINCT COALESCE(NULLIF(e.visitor_id, ''), e.session_id)) AS sessions,
			COUNT(DISTINCT CASE WHEN e.event_type = 'conversion' THEN COALESCE(NULLIF(e.visitor_id, ''), e.session_id) END) AS conversions
		FROM metrics_events e WHERE e.created_at >= $1 AND e.created_at <= $2 GROUP BY 1
	), revenue AS (
		SELECT vd.dimension_key, COALESCE(SUM(c.amount_cents), 0) AS revenue_minor
		FROM visitor_dimensions vd JOIN checkout_sessions c ON c.visitor_id = vd.visitor_key
		WHERE c.status IN ('paid', 'complete') GROUP BY vd.dimension_key
	)
	SELECT g.dimension_key, g.sessions, g.conversions, COALESCE(r.revenue_minor, 0),
		SUM(g.sessions) OVER () AS total_sessions, COUNT(*) OVER () AS total_groups
	FROM grouped g LEFT JOIN revenue r USING (dimension_key)
	ORDER BY g.sessions DESC, g.dimension_key LIMIT $3`, column, column)
	rows, err := s.db.Query(query, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("query traffic breakdown: %w", err)
	}
	defer rows.Close()
	out := &TrafficBreakdown{Currency: "usd", ObservedAt: s.clock.Now().UTC()}
	for rows.Next() {
		var row TrafficBreakdownRow
		var totalGroups int64
		if err := rows.Scan(&row.Key, &row.Sessions, &row.Conversions, &row.RevenueMinor, &out.TotalSessions, &totalGroups); err != nil {
			return nil, fmt.Errorf("scan traffic breakdown: %w", err)
		}
		row.Label = row.Key
		out.Rows = append(out.Rows, row)
		out.Exhaustive = totalGroups <= int64(limit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate traffic breakdown: %w", err)
	}
	for i := range out.Rows {
		if out.TotalSessions > 0 {
			out.Rows[i].Share = float64(out.Rows[i].Sessions) / float64(out.TotalSessions)
		}
	}
	return out, nil
}

func (s *Service) GetTrafficSeries(metric string, startDate, endDate time.Time, bucket string) (*TrafficSeries, error) {
	metricSQL := map[string]string{"visitors": "COUNT(DISTINCT COALESCE(NULLIF(visitor_id, ''), session_id))", "sessions": "COUNT(DISTINCT session_id)", "conversions": "COUNT(*) FILTER (WHERE event_type = 'conversion')"}
	expr, ok := metricSQL[strings.ToLower(metric)]
	if !ok {
		return nil, fmt.Errorf("unsupported traffic metric %q", metric)
	}
	if bucket == "" {
		bucket = "day"
	}
	if bucket != "day" {
		return nil, fmt.Errorf("unsupported traffic bucket %q", bucket)
	}
	query := fmt.Sprintf("SELECT date_trunc('day', created_at), %s FROM metrics_events WHERE created_at >= $1 AND created_at <= $2 GROUP BY 1 ORDER BY 1", expr)
	rows, err := s.db.Query(query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("query traffic series: %w", err)
	}
	defer rows.Close()
	out := &TrafficSeries{Unit: "count", ObservedAt: s.clock.Now().UTC()}
	for rows.Next() {
		var p TrafficSeriesPoint
		if err := rows.Scan(&p.BucketStart, &p.Value); err != nil {
			return nil, fmt.Errorf("scan traffic series: %w", err)
		}
		out.Points = append(out.Points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate traffic series: %w", err)
	}
	return out, nil
}

func ValidateEvent(event Event) error {
	if strings.TrimSpace(event.EventType) == "" {
		return &ValidationError{Field: "event_type", Reason: "event_type is required"}
	}
	if _, ok := validEventTypes[event.EventType]; !ok {
		return &ValidationError{Field: "event_type", Reason: fmt.Sprintf("invalid event_type: %s", event.EventType)}
	}
	if strings.TrimSpace(event.VariantSlug) == "" {
		return &ValidationError{Field: "variant_slug", Reason: "variant_slug is required"}
	}
	if strings.TrimSpace(event.SessionID) == "" {
		return &ValidationError{Field: "session_id", Reason: "session_id is required"}
	}
	return nil
}

func GenerateEventIDAt(event Event, now time.Time) string {
	input := fmt.Sprintf("%s:%s:%s:%d", event.SessionID, event.EventType, event.VariantSlug, now.Unix())
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:16])
}

func (s *Service) GetVariantStats(startDate, endDate time.Time, variantSlug string) ([]VariantStats, error) {
	query := `SELECT e.variant_slug,
		COALESCE(SUM(CASE WHEN event_type = 'page_view' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN event_type = 'click' AND event_data->>'element_type' = 'cta' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN event_type = 'conversion' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN event_type = 'download' THEN 1 ELSE 0 END), 0),
		COALESCE(x.exposures, 0)
		FROM metrics_events e
		LEFT JOIN (
			SELECT variant_slug, COUNT(*) AS exposures
			FROM experiment_exposures
			WHERE first_seen_at >= $1 AND first_seen_at <= $2
			GROUP BY variant_slug
		) x ON x.variant_slug = e.variant_slug
		WHERE e.created_at >= $1 AND e.created_at <= $2`
	args := []interface{}{startDate, endDate}
	if variantSlug != "" {
		query += " AND e.variant_slug = $3"
		args = append(args, variantSlug)
	}
	query += " GROUP BY e.variant_slug, x.exposures ORDER BY e.variant_slug"
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query variant stats: %w", err)
	}
	defer rows.Close()
	var stats []VariantStats
	for rows.Next() {
		var stat VariantStats
		if err := rows.Scan(&stat.VariantSlug, &stat.Views, &stat.CTAClicks, &stat.Conversions, &stat.Downloads, &stat.Exposures); err != nil {
			return nil, fmt.Errorf("failed to scan variant stats: %w", err)
		}
		stat.VariantName = stat.VariantSlug
		if stat.Exposures > 0 {
			stat.ConversionRate = float64(stat.Conversions) / float64(stat.Exposures) * 100
		}
		stats = append(stats, stat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate variant stats: %w", err)
	}
	return stats, nil
}

func (s *Service) GetAnalyticsSummary(startDate, endDate time.Time) (*AnalyticsSummary, error) {
	stats, err := s.GetVariantStats(startDate, endDate, "")
	if err != nil {
		return nil, err
	}
	var totalVisitors int64
	if err = s.db.QueryRow(`SELECT COUNT(DISTINCT COALESCE(NULLIF(visitor_id, ''), session_id)) FROM metrics_events WHERE event_type = 'page_view' AND created_at >= $1 AND created_at <= $2`, startDate, endDate).Scan(&totalVisitors); err != nil {
		return nil, fmt.Errorf("failed to count visitors: %w", err)
	}
	var totalDownloads int64
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM metrics_events WHERE event_type = 'download' AND created_at >= $1 AND created_at <= $2`, startDate, endDate).Scan(&totalDownloads); err != nil {
		return nil, fmt.Errorf("failed to count downloads: %w", err)
	}
	var topCTA string
	var topCTAClicks, topCTAViews int64
	err = s.db.QueryRow(`SELECT m.event_data->>'element_id', COUNT(*),
		(SELECT COUNT(DISTINCT COALESCE(NULLIF(visitor_id, ''), session_id)) FROM metrics_events WHERE event_type = 'page_view' AND created_at >= $1 AND created_at <= $2)
		FROM metrics_events m WHERE m.event_type = 'click' AND m.event_data->>'element_type' = 'cta' AND m.created_at >= $1 AND m.created_at <= $2
		GROUP BY m.event_data->>'element_id' ORDER BY COUNT(*) DESC LIMIT 1`, startDate, endDate).Scan(&topCTA, &topCTAClicks, &topCTAViews)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to query top CTA: %w", err)
	}
	var topCTACTR float64
	if topCTAViews > 0 {
		topCTACTR = float64(topCTAClicks) / float64(topCTAViews) * 100
	}
	observedAt := time.Now().UTC()
	return &AnalyticsSummary{TotalVisitors: totalVisitors, TotalDownloads: totalDownloads, VariantStats: stats, TopCTA: topCTA, TopCTACTR: topCTACTR, ObservedAt: &observedAt}, nil
}
