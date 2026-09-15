// Package metrics owns event validation, idempotent ingestion, and analytics
// aggregation. HTTP handlers remain at the transport edge and depend on this
// package rather than holding metrics rules themselves.
package metrics

import (
	"context"
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
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	digestdomain "landing-page-business-suite-api/internal/digest"
	"landing-page-business-suite-api/internal/experimentation"
)

type Service struct {
	db        Store
	contextDB ContextStore
	clock     schedule.Clock
	variants  interface {
		ListVariants() []*experimentation.VariantSnapshot
	}
}

// Store is the persistence boundary for metrics ingestion and reporting.
type Store interface {
	QueryRow(string, ...any) *sql.Row
	Query(string, ...any) (*sql.Rows, error)
	Exec(string, ...any) (sql.Result, error)
}

// ContextStore is the request-scoped persistence boundary for operations that
// must honor Test Genie's lease routing. It is intentionally separate from the
// legacy Store contract until the broader metrics read/write migration is
// complete.
type ContextStore interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

var ErrPresentationExposureUnavailable = errors.New("presentation exposure persistence is unavailable")

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
	service := &Service{db: db, clock: serviceClock}
	if contextDB, ok := db.(ContextStore); ok {
		service.contextDB = contextDB
	}
	return service
}

// NewServiceWithContextStore keeps existing metrics reads on their current
// Store seam while routing new request-scoped writes through an explicit
// context-aware owner. A nil context store is a hard error for those writes;
// the method never falls back to Store.Exec.
func NewServiceWithContextStore(db Store, contextDB ContextStore) *Service {
	service := NewService(db)
	service.contextDB = contextDB
	return service
}

// SetVariantConfigReader supplies the authored active-variant catalog so
// zero-event variants remain visible in reporting.
func (s *Service) SetVariantConfigReader(reader interface {
	ListVariants() []*experimentation.VariantSnapshot
}) {
	s.variants = reader
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
	TrafficClass string                 `json:"traffic_class,omitempty"`
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
	BotEvents      int64          `json:"bot_events,omitempty"`
	InternalEvents int64          `json:"internal_events,omitempty"`
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
	OtherVisitors int64                 `json:"other_visitors"`
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

// GetBusinessDigest is the producer-owned aggregate used by Command Center.
// It deliberately returns only aggregate values and keeps window validation at
// the contract edge.
func (s *Service) GetBusinessDigest(ctx context.Context, days int32) (*lpbsv1.BusinessDigest, error) {
	if days == 0 {
		days = 30
	}
	if days != 7 && days != 30 && days != 90 {
		return nil, fmt.Errorf("window_days must be 7, 30, or 90")
	}
	end := s.clock.Now().UTC()
	start := end.AddDate(0, 0, -int(days))
	summary, err := s.GetAnalyticsSummary(start, end)
	if err != nil {
		return nil, err
	}
	revenue, err := s.GetRevenueSummary()
	if err != nil {
		return nil, err
	}
	var cta, ctaClickers, checkouts, paid, paidCheckoutTotal, unattributedCheckouts, unattributedPaid int64
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM metrics_events WHERE traffic_class='human' AND event_type='click' AND event_data->>'element_type'='cta' AND created_at >= $1 AND created_at <= $2`, start, end).Scan(&cta); err != nil {
		return nil, err
	}
	if err = s.db.QueryRow(`WITH visitors AS (
		SELECT DISTINCT COALESCE(NULLIF(visitor_id,''), session_id) AS visitor_key
		FROM metrics_events
		WHERE traffic_class='human' AND event_type='page_view' AND created_at >= $1 AND created_at < $2
	), cta_clickers AS (
		SELECT DISTINCT COALESCE(NULLIF(visitor_id,''), session_id) AS visitor_key
		FROM metrics_events
		WHERE traffic_class='human' AND event_type='click' AND event_data->>'element_type'='cta' AND created_at >= $1 AND created_at < $2
	)
	SELECT COUNT(*) FROM cta_clickers c JOIN visitors v USING (visitor_key)`, start, end).Scan(&ctaClickers); err != nil {
		return nil, err
	}
	if err = s.db.QueryRow(`WITH visitors AS (
		SELECT DISTINCT COALESCE(NULLIF(visitor_id,''), session_id) AS visitor_key
		FROM metrics_events
		WHERE traffic_class='human' AND event_type='page_view' AND created_at >= $1 AND created_at < $2
	), cta_clickers AS (
		SELECT DISTINCT COALESCE(NULLIF(visitor_id,''), session_id) AS visitor_key
		FROM metrics_events
		WHERE traffic_class='human' AND event_type='click' AND event_data->>'element_type'='cta' AND created_at >= $1 AND created_at < $2
	), checkout_visitors AS (
		SELECT DISTINCT visitor_id AS visitor_key FROM checkout_sessions
		WHERE visitor_id IS NOT NULL AND visitor_id <> '' AND created_at >= $1 AND created_at < $2
	)
	SELECT COUNT(*) FROM checkout_visitors c JOIN cta_clickers a USING (visitor_key) JOIN visitors v USING (visitor_key)`, start, end).Scan(&checkouts); err != nil {
		return nil, err
	}
	if err = s.db.QueryRow(`WITH cta_clickers AS (
		SELECT DISTINCT COALESCE(NULLIF(visitor_id,''), session_id) AS visitor_key
		FROM metrics_events
		WHERE traffic_class='human' AND event_type='click' AND event_data->>'element_type'='cta' AND created_at >= $1 AND created_at < $2
	), checkout_visitors AS (
		SELECT DISTINCT visitor_id AS visitor_key FROM checkout_sessions
		WHERE visitor_id IS NOT NULL AND visitor_id <> '' AND created_at >= $1 AND created_at < $2
	), converted AS (
		SELECT DISTINCT COALESCE(NULLIF(visitor_id,''), session_id) AS visitor_key
		FROM metrics_events
		WHERE traffic_class='human' AND event_type='conversion' AND created_at >= $1 AND created_at < $2
	)
	SELECT COUNT(*) FROM converted p JOIN checkout_visitors c USING (visitor_key) JOIN cta_clickers a USING (visitor_key)`, start, end).Scan(&paid); err != nil {
		return nil, err
	}
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM metrics_events WHERE traffic_class='human' AND event_type='conversion' AND created_at >= $1 AND created_at < $2`, start, end).Scan(&paidCheckoutTotal); err != nil {
		return nil, err
	}
	if err = s.db.QueryRow(`SELECT COUNT(*) FILTER (WHERE NULLIF(visitor_id,'') IS NULL) FROM checkout_sessions WHERE created_at >= $1 AND created_at < $2`, start, end).Scan(&unattributedCheckouts); err != nil {
		return nil, err
	}
	if err = s.db.QueryRow(`SELECT COUNT(*) FILTER (WHERE NULLIF(visitor_id,'') IS NULL) FROM metrics_events WHERE traffic_class='human' AND event_type='conversion' AND created_at >= $1 AND created_at < $2`, start, end).Scan(&unattributedPaid); err != nil {
		return nil, err
	}
	var digestRevenue, digestCredits, digestPurchased, digestOperations, digestConsumers int64
	if err = s.db.QueryRow(`SELECT COALESCE(SUM(amount_cents),0) FROM checkout_sessions WHERE status IN ('paid','complete') AND COALESCE(completed_at, created_at) >= $1 AND COALESCE(completed_at, created_at) <= $2`, start, end).Scan(&digestRevenue); err != nil {
		return nil, err
	}
	if err = s.db.QueryRow(`SELECT COALESCE(SUM(credits),0) FROM usage_events WHERE created_at >= $1 AND created_at <= $2`, start, end).Scan(&digestCredits); err != nil {
		return nil, err
	}
	if err = s.db.QueryRow(`SELECT COUNT(*), COUNT(DISTINCT user_identity) FROM usage_events WHERE created_at >= $1 AND created_at <= $2`, start, end).Scan(&digestOperations, &digestConsumers); err != nil {
		return nil, err
	}
	if err = s.db.QueryRow(`SELECT COALESCE((SELECT SUM(amount_credits) FROM credit_transactions WHERE amount_credits > 0 AND created_at >= $1 AND created_at <= $2), 0) + COALESCE((SELECT SUM(amount_credits) FROM business_account_credit_transactions WHERE amount_credits > 0 AND created_at >= $1 AND created_at <= $2), 0)`, start, end).Scan(&digestPurchased); err != nil {
		return nil, err
	}
	var signups, waitlist, paidSubs, trials int64
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE created_at >= $1 AND created_at <= $2`, start, end).Scan(&signups); err != nil {
		return nil, err
	}
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM waitlist_emails WHERE created_at >= $1 AND created_at <= $2`, start, end).Scan(&waitlist); err != nil {
		return nil, err
	}
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM subscriptions WHERE created_at >= $1 AND created_at <= $2 AND status IN ('active','trialing')`, start, end).Scan(&paidSubs); err != nil {
		return nil, err
	}
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM subscriptions WHERE created_at >= $1 AND created_at <= $2 AND status = 'trialing'`, start, end).Scan(&trials); err != nil {
		return nil, err
	}
	// Retention is a paid-subscription cohort, not a signup cohort. The
	// subscriptions table's created_at is the first-paid timestamp recorded by
	// the current producer; status is evaluated at read time.
	cohortStart := end.AddDate(0, 0, -60)
	cohortEnd := end.AddDate(0, 0, -30)
	var cohortSize, retained int64
	if err = s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(CASE WHEN status = 'active' OR (status = 'trialing' AND customer_id IS NOT NULL) THEN 1 ELSE 0 END), 0) FROM subscriptions WHERE created_at >= $1 AND created_at < $2`, cohortStart, cohortEnd).Scan(&cohortSize, &retained); err != nil {
		return nil, err
	}
	apps := make([]*lpbsv1.AppActivity, 0)
	appRows, err := s.db.Query(`SELECT app_key, COUNT(*) FILTER (WHERE event_type='download_authorized'), COUNT(*) FILTER (WHERE event_type='update_check'), COUNT(*) FILTER (WHERE event_type='update_download') FROM delivery_events WHERE created_at >= $1 AND created_at <= $2 GROUP BY app_key ORDER BY app_key`, start, end)
	if err != nil {
		return nil, err
	}
	for appRows.Next() {
		var app string
		var downloads, checks, updateDownloads int64
		if err := appRows.Scan(&app, &downloads, &checks, &updateDownloads); err != nil {
			appRows.Close()
			return nil, err
		}
		apps = append(apps, &lpbsv1.AppActivity{AppKey: app, AppName: app, Downloads: downloads, UpdateChecks: checks, UpdateDownloads: updateDownloads})
	}
	if err := appRows.Err(); err != nil {
		appRows.Close()
		return nil, err
	}
	appRows.Close()
	byApp := make([]*lpbsv1.CreditUsageRow, 0)
	byModel := make([]*lpbsv1.CreditUsageRow, 0)
	usageRows, err := s.db.Query(`SELECT app_bundle_key, COALESCE(SUM(credits),0), COUNT(*) FROM usage_events WHERE created_at >= $1 AND created_at <= $2 GROUP BY app_bundle_key ORDER BY app_bundle_key`, start, end)
	if err != nil {
		return nil, err
	}
	for usageRows.Next() {
		var key string
		var credits, operations int64
		if err := usageRows.Scan(&key, &credits, &operations); err != nil {
			usageRows.Close()
			return nil, err
		}
		byApp = append(byApp, &lpbsv1.CreditUsageRow{Key: key, Label: key, Credits: credits, Operations: operations})
	}
	if err := usageRows.Err(); err != nil {
		usageRows.Close()
		return nil, err
	}
	usageRows.Close()
	modelRows, err := s.db.Query(`SELECT model, COALESCE(SUM(credits),0), COUNT(*) FROM usage_events WHERE created_at >= $1 AND created_at <= $2 GROUP BY model ORDER BY model`, start, end)
	if err != nil {
		return nil, err
	}
	for modelRows.Next() {
		var key string
		var credits, operations int64
		if err := modelRows.Scan(&key, &credits, &operations); err != nil {
			modelRows.Close()
			return nil, err
		}
		byModel = append(byModel, &lpbsv1.CreditUsageRow{Key: key, Label: key, Credits: credits, Operations: operations})
	}
	if err := modelRows.Err(); err != nil {
		modelRows.Close()
		return nil, err
	}
	modelRows.Close()
	control := "control"
	hasControl := false
	for _, stat := range summary.VariantStats {
		if stat.VariantSlug == control {
			hasControl = true
			break
		}
	}
	if !hasControl {
		for _, stat := range summary.VariantStats {
			if stat.VariantSlug != "" {
				control = stat.VariantSlug
				break
			}
		}
	}
	var controlTrials, controlConversions int64
	for _, stat := range summary.VariantStats {
		if stat.VariantSlug == control {
			controlTrials, controlConversions = stat.Exposures, stat.Conversions
			break
		}
	}
	arms := make([]*lpbsv1.ExperimentArm, 0, len(summary.VariantStats))
	for _, stat := range summary.VariantStats {
		verdict := lpbsv1.ExperimentVerdict_EXPERIMENT_VERDICT_INSUFFICIENT_DATA
		prob := 0.5
		trials := stat.Exposures
		if trials >= 100 && stat.Conversions >= 5 {
			prob = digestdomain.ProbabilityBeatsControl(stat.Conversions, trials, controlConversions, controlTrials)
			if stat.VariantSlug == control {
				verdict = lpbsv1.ExperimentVerdict_EXPERIMENT_VERDICT_CONTROL
			} else if prob >= 0.95 {
				verdict = lpbsv1.ExperimentVerdict_EXPERIMENT_VERDICT_LEADING
			} else if prob <= 0.05 {
				verdict = lpbsv1.ExperimentVerdict_EXPERIMENT_VERDICT_TRAILING
			} else {
				verdict = lpbsv1.ExperimentVerdict_EXPERIMENT_VERDICT_INCONCLUSIVE
			}
		}
		arms = append(arms, &lpbsv1.ExperimentArm{VariantSlug: stat.VariantSlug, VariantName: stat.VariantName, IsControl: stat.VariantSlug == control, CtaClick: &lpbsv1.ArmMetric{Successes: stat.CTAClicks, Trials: trials, RatePercent: percent(stat.CTAClicks, trials), ProbabilityBeatsControl: prob, Verdict: verdict}, Paid: &lpbsv1.ArmMetric{Successes: stat.Conversions, Trials: trials, RatePercent: percent(stat.Conversions, trials), ProbabilityBeatsControl: prob, Verdict: verdict}})
	}
	steps := []*lpbsv1.FunnelStep{{Key: "visitors", Label: "Visitors", Value: summary.TotalVisitors}, {Key: "cta_clickers", Label: "Clicked a CTA", Value: ctaClickers, DenominatorKey: "visitors", Denominator: summary.TotalVisitors}, {Key: "checkouts_started", Label: "Checkouts started", Value: checkouts, DenominatorKey: "cta_clickers", Denominator: ctaClickers}, {Key: "paid", Label: "Paid", Value: paid, DenominatorKey: "checkouts_started", Denominator: checkouts}}
	return &lpbsv1.BusinessDigest{ContractVersion: "business-digest.v1", ObservedAt: timestamppb.New(end), Window: &lpbsv1.DigestWindow{Start: timestamppb.New(start), End: timestamppb.New(end), Days: days}, Funnel: &lpbsv1.Funnel{Steps: steps, UnattributedCheckoutsStarted: unattributedCheckouts, UnattributedPaid: unattributedPaid, PaidCheckoutsTotal: paidCheckoutTotal, PaidRevenueMinor: digestRevenue, Currency: revenue.Currency}, Apps: apps, Credits: &lpbsv1.CreditEconomy{CreditsBurned: digestCredits, CreditsPurchased: digestPurchased, DistinctConsumers: digestConsumers, Operations: digestOperations, ByApp: byApp, ByModel: byModel}, Growth: &lpbsv1.Growth{Signups: signups, WaitlistJoins: waitlist, NewPaidSubscriptions: paidSubs, TrialsStarted: trials}, Experiment: &lpbsv1.Experiment{Arms: arms, ControlSlug: control, MinimumTrials: 100, MinimumSuccesses: 5}, Retention: &lpbsv1.Retention{CohortStart: timestamppb.New(cohortStart), CohortEnd: timestamppb.New(cohortEnd), CohortSize: cohortSize, StillActive: retained}, Exclusions: &lpbsv1.TrafficExclusions{BotEvents: summary.BotEvents, InternalEvents: summary.InternalEvents}, CtaClicks: cta}, nil
}

func percent(successes, trials int64) float64 {
	if trials <= 0 {
		return 0
	}
	return float64(successes) * 100 / float64(trials)
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

// RecordPresentationExposure records a validated public presentation
// assignment using the existing experiment-exposure deduplication key. The
// revision, route, locale, and block digest are proof inputs owned and
// validated by the landing service; this persistence owner deliberately does
// not create a new schema surface before the parent protocol decision.
func (s *Service) RecordPresentationExposure(ctx context.Context, visitorID, variantSlug, revision, route, locale, blockDigest, weightFingerprint string) (bool, error) {
	for field, value := range map[string]string{
		"visitor_id":         visitorID,
		"variant_slug":       variantSlug,
		"revision":           revision,
		"route":              route,
		"locale":             locale,
		"block_digest":       blockDigest,
		"weight_fingerprint": weightFingerprint,
	} {
		if strings.TrimSpace(value) == "" {
			return false, &ValidationError{Field: field, Reason: "is required"}
		}
	}
	if s.contextDB == nil {
		return false, ErrPresentationExposureUnavailable
	}
	result, err := s.contextDB.ExecContext(ctx, `INSERT INTO experiment_exposures (visitor_id, variant_slug, weight_fingerprint) VALUES ($1, $2, $3) ON CONFLICT (visitor_id, variant_slug, weight_fingerprint) DO NOTHING`, strings.TrimSpace(visitorID), strings.TrimSpace(variantSlug), strings.TrimSpace(weightFingerprint))
	if err != nil {
		return false, fmt.Errorf("record presentation exposure: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("inspect presentation exposure result: %w", err)
	}
	return rows > 0, nil
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
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(amount_cents), 0) FROM checkout_sessions WHERE status IN ('paid','complete') AND COALESCE(completed_at, created_at) >= CURRENT_DATE`).Scan(&today); err != nil {
		return nil, fmt.Errorf("compute revenue summary today: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(amount_cents), 0) FROM checkout_sessions WHERE status IN ('paid','complete') AND COALESCE(completed_at, created_at) >= CURRENT_DATE - INTERVAL '30 days'`).Scan(&window); err != nil {
		return nil, fmt.Errorf("compute revenue summary window: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FILTER (WHERE canceled_at >= CURRENT_DATE - INTERVAL '30 days') FROM subscriptions WHERE canceled_at IS NOT NULL`).Scan(&churned); err != nil {
		return nil, fmt.Errorf("compute revenue summary churn: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(balance_credits + bonus_credits), 0) FROM credit_wallets`).Scan(&creditBalance); err != nil {
		return nil, fmt.Errorf("compute revenue summary credits: %w", err)
	}
	var businessCreditBalance int64
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(balance_credits + bonus_credits), 0) FROM business_account_credit_wallets`).Scan(&businessCreditBalance); err != nil {
		return nil, fmt.Errorf("compute revenue summary business credits: %w", err)
	}
	creditBalance += businessCreditBalance
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(credits), 0) FROM usage_events WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'`).Scan(&creditBurned); err != nil {
		return nil, fmt.Errorf("compute revenue summary credit usage: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM usage_events WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'`).Scan(&usage); err != nil {
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
	if event.TrafficClass == "" {
		event.TrafficClass = "human"
	}
	if event.TrafficClass != "human" && event.TrafficClass != "bot" && event.TrafficClass != "internal" {
		return &ValidationError{Field: "traffic_class", Reason: "must be human, bot, or internal"}
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
	var existingEventID string
	if err := s.db.QueryRow(`SELECT event_id FROM metrics_events WHERE event_id = $1`, eventID).Scan(&existingEventID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("idempotency check failed: %w", err)
	}
	if _, err = s.db.Exec(`INSERT INTO metrics_events
		(variant_slug, event_type, event_data, event_id, session_id, visitor_id,
		 referrer_host, referrer_kind, utm_source, utm_medium, utm_campaign,
		landing_path, country_code, device_class, traffic_class)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''), NULLIF($12, ''), NULLIF($13, ''), NULLIF($14, ''), $15)
		ON CONFLICT (event_id) WHERE event_id IS NOT NULL DO NOTHING`, event.VariantSlug, event.EventType, eventDataJSON,
		eventID, event.SessionID, event.VisitorID, event.ReferrerHost, event.ReferrerKind,
		event.UTMSource, event.UTMMedium, event.UTMCampaign, event.LandingPath,
		event.CountryCode, event.DeviceClass, event.TrafficClass); err != nil {
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
		"utm_medium": "utm_medium", "utm_campaign": "utm_campaign", "device_class": "device_class", "landing_path": "landing_path", "variant": "variant_slug",
	}
	column, ok := columns[strings.ToLower(dimension)]
	if !ok {
		return nil, fmt.Errorf("unsupported traffic dimension %q", dimension)
	}
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	query := fmt.Sprintf(`WITH first_touch AS (
		SELECT DISTINCT ON (COALESCE(NULLIF(visitor_id, ''), session_id))
			COALESCE(NULLIF(visitor_id, ''), session_id) AS visitor_key,
			COALESCE(NULLIF(%s, ''), 'unknown') AS dimension_key
		FROM metrics_events
		WHERE traffic_class = 'human' AND event_type = 'page_view' AND created_at >= $1 AND created_at <= $2
		ORDER BY COALESCE(NULLIF(visitor_id, ''), session_id), created_at, event_id
	), conversions AS (
		SELECT DISTINCT COALESCE(NULLIF(visitor_id, ''), session_id) AS visitor_key
		FROM metrics_events
		WHERE traffic_class = 'human' AND event_type = 'conversion' AND created_at >= $1 AND created_at <= $2
	), grouped AS (
		SELECT f.dimension_key, COUNT(*) AS visitors, COUNT(c.visitor_key) AS conversions
		FROM first_touch f LEFT JOIN conversions c USING (visitor_key)
		GROUP BY f.dimension_key
	), revenue AS (
		SELECT f.dimension_key, COALESCE(SUM(c.amount_cents), 0) AS revenue_minor
		FROM first_touch f JOIN checkout_sessions c
			ON COALESCE(NULLIF(c.visitor_id, ''), c.session_id) = f.visitor_key
		WHERE c.status IN ('paid', 'complete') AND COALESCE(c.completed_at, c.created_at) >= $1 AND COALESCE(c.completed_at, c.created_at) <= $2
		GROUP BY f.dimension_key
	)
	SELECT g.dimension_key, g.visitors, g.conversions, COALESCE(r.revenue_minor, 0),
		SUM(g.visitors) OVER () AS total_visitors, COUNT(*) OVER () AS total_groups
	FROM grouped g LEFT JOIN revenue r USING (dimension_key)
	ORDER BY g.visitors DESC, g.dimension_key LIMIT $3`, column)
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
		if !out.Exhaustive {
			out.OtherVisitors = out.TotalSessions
			for _, existing := range out.Rows {
				out.OtherVisitors -= existing.Sessions
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate traffic breakdown: %w", err)
	}
	if len(out.Rows) == 0 {
		out.Exhaustive = true
	}
	for i := range out.Rows {
		if out.TotalSessions > 0 {
			out.Rows[i].Share = float64(out.Rows[i].Sessions) / float64(out.TotalSessions)
		}
	}
	return out, nil
}

func (s *Service) GetTrafficSeries(metric string, startDate, endDate time.Time, bucket string) (*TrafficSeries, error) {
	metricSQL := map[string]string{"visitors": "COUNT(DISTINCT COALESCE(NULLIF(visitor_id, ''), session_id))", "sessions": "COUNT(DISTINCT session_id)", "conversions": "COUNT(*) FILTER (WHERE event_type = 'conversion')", "cta_clicks": "COUNT(*) FILTER (WHERE event_type = 'click' AND event_data->>'element_type' = 'cta')"}
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
	query := fmt.Sprintf("SELECT date_trunc('day', created_at), %s FROM metrics_events WHERE traffic_class = 'human' AND created_at >= $1 AND created_at <= $2 GROUP BY 1 ORDER BY 1", expr)
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
	byDay := make(map[string]TrafficSeriesPoint, len(out.Points))
	for _, point := range out.Points {
		byDay[point.BucketStart.UTC().Format("2006-01-02")] = point
	}
	firstDay := time.Date(startDate.UTC().Year(), startDate.UTC().Month(), startDate.UTC().Day(), 0, 0, 0, 0, time.UTC)
	lastDay := time.Date(endDate.UTC().Year(), endDate.UTC().Month(), endDate.UTC().Day(), 0, 0, 0, 0, time.UTC)
	points := make([]TrafficSeriesPoint, 0, int(lastDay.Sub(firstDay).Hours()/24)+1)
	for day := firstDay; !day.After(lastDay); day = day.AddDate(0, 0, 1) {
		if point, ok := byDay[day.Format("2006-01-02")]; ok {
			points = append(points, point)
		} else {
			points = append(points, TrafficSeriesPoint{BucketStart: day})
		}
	}
	out.Points = points
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
	args := []interface{}{startDate, endDate}
	exposureFingerprint := ""
	if s.variants != nil {
		exposureFingerprint = experimentation.WeightFingerprint(s.variants.ListVariants())
	}
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
			WHERE first_seen_at >= $1 AND first_seen_at <= $2`
	if exposureFingerprint != "" {
		query += ` AND weight_fingerprint = $3`
		args = append(args, exposureFingerprint)
	}
	query += `
			GROUP BY variant_slug
		) x ON x.variant_slug = e.variant_slug
		WHERE e.traffic_class = 'human' AND e.created_at >= $1 AND e.created_at <= $2`
	if variantSlug != "" {
		placeholder := len(args) + 1
		query += fmt.Sprintf(" AND e.variant_slug = $%d", placeholder)
		args = append(args, variantSlug)
	}
	query += " GROUP BY e.variant_slug, x.exposures ORDER BY e.variant_slug"
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query variant stats: %w", err)
	}
	defer rows.Close()
	var stats []VariantStats
	bySlug := make(map[string]int)
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
		bySlug[stat.VariantSlug] = len(stats) - 1
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate variant stats: %w", err)
	}
	if s.variants != nil {
		for _, variant := range s.variants.ListVariants() {
			if variant == nil || (variant.Variant.Status != "" && variant.Variant.Status != "active") || (variantSlug != "" && variant.Variant.Slug != variantSlug) {
				continue
			}
			if _, ok := bySlug[variant.Variant.Slug]; ok {
				stats[bySlug[variant.Variant.Slug]].VariantName = variant.Variant.Name
				continue
			}
			stats = append(stats, VariantStats{VariantSlug: variant.Variant.Slug, VariantName: variant.Variant.Name})
		}
	}
	return stats, nil
}

func (s *Service) GetAnalyticsSummary(startDate, endDate time.Time) (*AnalyticsSummary, error) {
	stats, err := s.GetVariantStats(startDate, endDate, "")
	if err != nil {
		return nil, err
	}
	var totalVisitors int64
	if err = s.db.QueryRow(`SELECT COUNT(DISTINCT COALESCE(NULLIF(visitor_id, ''), session_id)) FROM metrics_events WHERE traffic_class = 'human' AND event_type = 'page_view' AND created_at >= $1 AND created_at <= $2`, startDate, endDate).Scan(&totalVisitors); err != nil {
		return nil, fmt.Errorf("failed to count visitors: %w", err)
	}
	var totalDownloads int64
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM metrics_events WHERE traffic_class = 'human' AND event_type = 'download' AND created_at >= $1 AND created_at <= $2`, startDate, endDate).Scan(&totalDownloads); err != nil {
		return nil, fmt.Errorf("failed to count downloads: %w", err)
	}
	var topCTA string
	var topCTAClicks, topCTAViews int64
	err = s.db.QueryRow(`SELECT COALESCE(m.event_data->>'element_id', 'unknown'), COUNT(*),
		(SELECT COUNT(DISTINCT COALESCE(NULLIF(visitor_id, ''), session_id)) FROM metrics_events WHERE traffic_class = 'human' AND event_type = 'page_view' AND created_at >= $1 AND created_at <= $2)
		FROM metrics_events m WHERE m.traffic_class = 'human' AND m.event_type = 'click' AND m.event_data->>'element_type' = 'cta' AND m.created_at >= $1 AND m.created_at <= $2
		GROUP BY COALESCE(m.event_data->>'element_id', 'unknown') ORDER BY COUNT(*) DESC LIMIT 1`, startDate, endDate).Scan(&topCTA, &topCTAClicks, &topCTAViews)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to query top CTA: %w", err)
	}
	var topCTACTR float64
	if topCTAViews > 0 {
		topCTACTR = float64(topCTAClicks) / float64(topCTAViews) * 100
	}
	observedAt := s.clock.Now().UTC()
	var botEvents, internalEvents int64
	_ = s.db.QueryRow(`SELECT COUNT(*) FILTER (WHERE traffic_class='bot'), COUNT(*) FILTER (WHERE traffic_class='internal') FROM metrics_events WHERE created_at >= $1 AND created_at <= $2`, startDate, endDate).Scan(&botEvents, &internalEvents)
	return &AnalyticsSummary{TotalVisitors: totalVisitors, TotalDownloads: totalDownloads, VariantStats: stats, TopCTA: topCTA, TopCTACTR: topCTACTR, BotEvents: botEvents, InternalEvents: internalEvents, ObservedAt: &observedAt}, nil
}
