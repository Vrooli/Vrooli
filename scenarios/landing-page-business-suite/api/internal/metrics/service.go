// Package metrics owns event validation, idempotent ingestion, and analytics
// aggregation. HTTP handlers remain at the transport edge and depend on this
// package rather than holding metrics rules themselves.
package metrics

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
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
	EventType   string                 `json:"event_type"`
	VariantSlug string                 `json:"variant_slug"`
	EventData   map[string]interface{} `json:"event_data,omitempty"`
	SessionID   string                 `json:"session_id"`
	VisitorID   string                 `json:"visitor_id,omitempty"`
	EventID     string                 `json:"event_id,omitempty"`
}

type VariantStats struct {
	VariantSlug    string  `json:"variant_slug"`
	VariantName    string  `json:"variant_name"`
	Views          int64   `json:"views"`
	CTAClicks      int64   `json:"cta_clicks"`
	Conversions    int64   `json:"conversions"`
	Downloads      int64   `json:"downloads"`
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

	var exists bool
	err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM metrics_events WHERE event_data->>'event_id' = $1)`, eventID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("idempotency check failed: %w", err)
	}
	if exists {
		return nil
	}
	if event.EventData == nil {
		event.EventData = make(map[string]interface{})
	}
	event.EventData["event_id"] = eventID
	eventDataJSON, err := json.Marshal(event.EventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event_data: %w", err)
	}
	if _, err = s.db.Exec(`INSERT INTO metrics_events (variant_slug, event_type, event_data, session_id, visitor_id) VALUES ($1, $2, $3, $4, $5)`, event.VariantSlug, event.EventType, eventDataJSON, event.SessionID, event.VisitorID); err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}
	return nil
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
	query := `SELECT variant_slug,
		COALESCE(SUM(CASE WHEN event_type = 'page_view' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN event_type = 'click' AND event_data->>'element_type' = 'cta' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN event_type = 'conversion' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN event_type = 'download' THEN 1 ELSE 0 END), 0)
		FROM metrics_events WHERE created_at >= $1 AND created_at <= $2`
	args := []interface{}{startDate, endDate}
	if variantSlug != "" {
		query += " AND variant_slug = $3"
		args = append(args, variantSlug)
	}
	query += " GROUP BY variant_slug ORDER BY variant_slug"
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query variant stats: %w", err)
	}
	defer rows.Close()
	var stats []VariantStats
	for rows.Next() {
		var stat VariantStats
		if err := rows.Scan(&stat.VariantSlug, &stat.Views, &stat.CTAClicks, &stat.Conversions, &stat.Downloads); err != nil {
			return nil, fmt.Errorf("failed to scan variant stats: %w", err)
		}
		stat.VariantName = stat.VariantSlug
		if stat.Views > 0 {
			stat.ConversionRate = float64(stat.Conversions) / float64(stat.Views) * 100
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
	if err = s.db.QueryRow(`SELECT COUNT(DISTINCT session_id) FROM metrics_events WHERE event_type = 'page_view' AND created_at >= $1 AND created_at <= $2`, startDate, endDate).Scan(&totalVisitors); err != nil {
		return nil, fmt.Errorf("failed to count visitors: %w", err)
	}
	var totalDownloads int64
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM metrics_events WHERE event_type = 'download' AND created_at >= $1 AND created_at <= $2`, startDate, endDate).Scan(&totalDownloads); err != nil {
		return nil, fmt.Errorf("failed to count downloads: %w", err)
	}
	var topCTA string
	var topCTAClicks, topCTAViews int64
	err = s.db.QueryRow(`SELECT m.event_data->>'element_id', COUNT(*),
		(SELECT COUNT(DISTINCT session_id) FROM metrics_events WHERE event_type = 'page_view' AND created_at >= $1 AND created_at <= $2)
		FROM metrics_events m WHERE m.event_type = 'click' AND m.event_data->>'element_type' = 'cta' AND m.created_at >= $1 AND m.created_at <= $2
		GROUP BY m.event_data->>'element_id' ORDER BY COUNT(*) DESC LIMIT 1`, startDate, endDate).Scan(&topCTA, &topCTAClicks, &topCTAViews)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query top CTA: %w", err)
	}
	var topCTACTR float64
	if topCTAViews > 0 {
		topCTACTR = float64(topCTAClicks) / float64(topCTAViews) * 100
	}
	observedAt := time.Now().UTC()
	return &AnalyticsSummary{TotalVisitors: totalVisitors, TotalDownloads: totalDownloads, VariantStats: stats, TopCTA: topCTA, TopCTACTR: topCTACTR, ObservedAt: &observedAt}, nil
}
