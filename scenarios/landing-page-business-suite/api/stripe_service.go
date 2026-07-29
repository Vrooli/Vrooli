package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/envx"
)

// StripeService handles Stripe payment integration
type StripeService struct {
	db                StripeServiceStore
	planService       *PlanService
	paymentSettings   *PaymentSettingsService
	paymentAnomaly    *commerce.PaymentAnomalyService
	checkoutCacheTTL  time.Duration
	httpClient        *http.Client
	apiBase           string
	mu                sync.RWMutex
	runtimeConfig     stripeRuntimeConfig
	configLoader      stripeConfigLoader
	introCouponConfig IntroCouponConfig
}

// StripeServiceStore is the transaction-capable persistence boundary shared by
// checkout, subscription, credit, coupon, and webhook workflows.
type StripeServiceStore interface {
	QueryRow(string, ...any) *sql.Row
	QueryRowContext(context.Context, string, ...any) *sql.Row
	Exec(string, ...any) (sql.Result, error)
	Begin() (*sql.Tx, error)
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// SetPaymentAnomaly wires the anomaly service after construction. Required
// because PaymentAnomalyService is constructed alongside StripeService but the
// two are wired in Server to avoid a circular dependency at boot.
func (s *StripeService) SetPaymentAnomaly(a *commerce.PaymentAnomalyService) {
	s.paymentAnomaly = a
}

type stripeRuntimeConfig struct {
	publishableKey string
	secretKey      string
	webhookSecret  string
	source         string
	hasPublishable bool
	hasSecret      bool
	hasWebhook     bool
	apiBase        string
}

type stripeConfigLoader func(ctx context.Context) (stripeRuntimeConfig, error)

const (
	sessionTypeSubscription          = "subscription"
	sessionTypeCreditsTopup          = "credits_topup"
	sessionTypeSupporterContribution = "supporter_contribution"
)

// IntroCouponConfig holds the configuration for intro pricing coupons.
// Coupons are created in Stripe Dashboard with duration=once.
type IntroCouponConfig struct {
	Enabled   bool              // Whether intro pricing is enabled
	CouponMap map[string]string // tier -> coupon_id mapping
}

// loadIntroCouponConfig loads intro coupon configuration from environment variables.
func loadIntroCouponConfig() IntroCouponConfig {
	enabled := strings.ToLower(strings.TrimSpace(envx.Get("INTRO_ENABLED"))) == "true"
	if !enabled {
		return IntroCouponConfig{Enabled: false}
	}

	couponMap := make(map[string]string)
	tierEnvMap := map[string]string{
		"solo":     "INTRO_COUPON_SOLO",
		"pro":      "INTRO_COUPON_PRO",
		"studio":   "INTRO_COUPON_STUDIO",
		"business": "INTRO_COUPON_BUSINESS",
	}

	for tier, envVar := range tierEnvMap {
		if couponID := strings.TrimSpace(envx.Get(envVar)); couponID != "" {
			couponMap[tier] = couponID
		}
	}

	return IntroCouponConfig{
		Enabled:   enabled && len(couponMap) > 0,
		CouponMap: couponMap,
	}
}

// GetCouponForTier returns the coupon ID for a given plan tier, or empty string if none.
func (c *IntroCouponConfig) GetCouponForTier(tier string) string {
	if !c.Enabled || c.CouponMap == nil {
		return ""
	}
	return c.CouponMap[strings.ToLower(strings.TrimSpace(tier))]
}

// StripeConfigError indicates missing or invalid Stripe configuration.
type StripeConfigError struct {
	MissingKey string
}

func (e *StripeConfigError) Error() string {
	return fmt.Sprintf("stripe configuration missing %s", e.MissingKey)
}

// StripeAPIError represents a Stripe API error response.
type StripeAPIError struct {
	Status    int
	Message   string
	Type      string
	Code      string
	Param     string
	RequestID string
}

func (e *StripeAPIError) Error() string {
	msg := strings.TrimSpace(e.Message)
	if msg == "" {
		msg = "stripe api error"
	}
	if e.RequestID != "" {
		return fmt.Sprintf("stripe api error (%d): %s (request_id=%s)", e.Status, msg, e.RequestID)
	}
	return fmt.Sprintf("stripe api error (%d): %s", e.Status, msg)
}

// StripeBundleProductNotFoundError indicates the configured bundle product is missing in Stripe.
type StripeBundleProductNotFoundError struct {
	BundleKey string
	ProductID string
}

func (e *StripeBundleProductNotFoundError) Error() string {
	if e.BundleKey == "" {
		return fmt.Sprintf("stripe product %s not found for bundle", e.ProductID)
	}
	return fmt.Sprintf("stripe product %s not found for bundle %s", e.ProductID, e.BundleKey)
}

// NewStripeService creates a new Stripe service instance.
func NewStripeService(db StripeServiceStore) *StripeService {
	return NewStripeServiceWithSettings(db, NewPlanService(db), NewPaymentSettingsService(db))
}

// NewStripeServiceWithSettings wires explicit plan/payment dependencies (used by server).
func NewStripeServiceWithSettings(db StripeServiceStore, planService *PlanService, paymentSettings *PaymentSettingsService) *StripeService {
	if planService == nil {
		planService = NewPlanService(db)
	}
	if paymentSettings == nil {
		paymentSettings = NewPaymentSettingsService(db)
	}

	service := &StripeService{
		db:                db,
		planService:       planService,
		paymentSettings:   paymentSettings,
		checkoutCacheTTL:  60 * time.Second,
		introCouponConfig: loadIntroCouponConfig(),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
	service.configLoader = service.loadStripeConfig

	if err := service.RefreshConfig(context.Background()); err != nil {
		logStructured("failed to initialize Stripe config", map[string]interface{}{
			"level": "warn",
			"error": err.Error(),
		})
	}

	// Log intro coupon config status
	if service.introCouponConfig.Enabled {
		logStructured("intro_coupon_config_loaded", map[string]interface{}{
			"level":        "info",
			"enabled":      true,
			"coupon_count": len(service.introCouponConfig.CouponMap),
		})
	}

	return service
}

// RefreshConfig reloads Stripe credentials from DB/env.
func (s *StripeService) RefreshConfig(ctx context.Context) error {
	s.mu.RLock()
	loader := s.configLoader
	s.mu.RUnlock()
	if loader == nil {
		loader = s.loadStripeConfig
	}

	cfg, err := loader(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.runtimeConfig = cfg
	if cfg.apiBase != "" {
		s.apiBase = cfg.apiBase
	}
	s.mu.Unlock()
	return nil
}

// UseHTTPClient allows tests to substitute a mock HTTP client for Stripe calls.
func (s *StripeService) UseHTTPClient(client *http.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.httpClient = client
}

// UseConfigLoader overrides how Stripe runtime configuration is loaded (primarily for tests).
func (s *StripeService) UseConfigLoader(loader stripeConfigLoader) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if loader == nil {
		s.configLoader = s.loadStripeConfig
		return
	}
	s.configLoader = loader
}

// UseIntroCouponConfig overrides intro coupon configuration (for tests).
func (s *StripeService) UseIntroCouponConfig(config IntroCouponConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.introCouponConfig = config
}

func (s *StripeService) loadStripeConfig(ctx context.Context) (stripeRuntimeConfig, error) {
	// Start with environment defaults.
	envPublishable := strings.TrimSpace(envx.Get("STRIPE_PUBLISHABLE_KEY"))
	envSecret := strings.TrimSpace(envx.Get("STRIPE_SECRET_KEY"))
	envWebhook := strings.TrimSpace(envx.Get("STRIPE_WEBHOOK_SECRET"))
	apiBase := strings.TrimSpace(envx.Get("STRIPE_API_BASE"))
	if apiBase == "" {
		apiBase = "https://api.stripe.com"
	}

	cfg := stripeRuntimeConfig{
		publishableKey: envPublishable,
		secretKey:      envSecret,
		webhookSecret:  envWebhook,
		apiBase:        apiBase,
		hasPublishable: envPublishable != "",
		hasSecret:      envSecret != "",
		hasWebhook:     envWebhook != "",
		source:         "env",
	}

	// Overlay database/admin overrides when present; keep env values for fields not provided.
	if s.paymentSettings != nil {
		record, err := s.paymentSettings.GetStripeSettings(ctx)
		if err != nil {
			return cfg, err
		}
		if record != nil {
			fromDB := false
			if publishable := strings.TrimSpace(record.PublishableKey); publishable != "" {
				cfg.publishableKey = publishable
				cfg.hasPublishable = true
				fromDB = true
			}
			if secret := strings.TrimSpace(record.SecretKey); secret != "" {
				cfg.secretKey = secret
				cfg.hasSecret = true
				fromDB = true
			}
			if webhook := strings.TrimSpace(record.WebhookSecret); webhook != "" {
				cfg.webhookSecret = webhook
				cfg.hasWebhook = true
				fromDB = true
			}
			if fromDB {
				cfg.source = "database"
			}
		}
	}

	if !cfg.hasPublishable {
		logStructured("STRIPE_PUBLISHABLE_KEY missing", map[string]interface{}{
			"level":   "warn",
			"message": "STRIPE_PUBLISHABLE_KEY is not set",
		})
	}
	if !cfg.hasSecret {
		logStructured("STRIPE_SECRET_KEY (restricted) missing", map[string]interface{}{
			"level":   "warn",
			"message": "STRIPE_SECRET_KEY (restricted key) is not set",
		})
	}
	if !cfg.hasWebhook {
		logStructured("STRIPE_WEBHOOK_SECRET missing", map[string]interface{}{
			"level":   "warn",
			"message": "STRIPE_WEBHOOK_SECRET is not set",
		})
	}

	return cfg, nil
}

func (s *StripeService) getConfig() stripeRuntimeConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runtimeConfig
}

func (s *StripeService) getHTTPClient() *http.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.httpClient
}

func (s *StripeService) getAPIBase() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.apiBase != "" {
		return s.apiBase
	}
	return "https://api.stripe.com"
}

func maskValue(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 6 {
		return value
	}
	return value[:4] + "…" + value[len(value)-2:]
}

func (s *StripeService) stripeAPIURL(path string) string {
	base := s.getAPIBase()
	return strings.TrimRight(base, "/") + path
}

func (s *StripeService) doStripeForm(ctx context.Context, method, path string, values url.Values) ([]byte, error) {
	return s.doStripeRequest(ctx, method, path, strings.NewReader(values.Encode()), "application/x-www-form-urlencoded")
}

func (s *StripeService) doStripeRequest(ctx context.Context, method, path string, body io.Reader, contentType string) ([]byte, error) {
	cfg := s.getConfig()
	if !cfg.hasSecret {
		return nil, &StripeConfigError{MissingKey: "secret_key"}
	}

	// #nosec G704 -- stripeAPIURL is derived from controlled Stripe configuration; test
	// fixtures inject a local HTTPS/HTTP server through the service seam.
	req, err := http.NewRequestWithContext(ctx, method, s.stripeAPIURL(path), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.secretKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	client := s.getHTTPClient()
	if client == nil {
		client = http.DefaultClient
	}

	// #nosec G704 -- request target is validated at construction above.
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodySnippet := strings.TrimSpace(string(data))
		if len(bodySnippet) > 512 {
			bodySnippet = bodySnippet[:512] + "…"
		}
		if bodySnippet == "" {
			bodySnippet = "no response body"
		}
		return nil, parseStripeAPIError(resp.StatusCode, bodySnippet, data, resp.Header.Get("Request-Id"))
	}

	return data, nil
}

func parseStripeAPIError(status int, fallback string, data []byte, requestID string) *StripeAPIError {
	var payload struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
			Param   string `json:"param"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &payload); err == nil && strings.TrimSpace(payload.Error.Message) != "" {
		return &StripeAPIError{
			Status:    status,
			Message:   strings.TrimSpace(payload.Error.Message),
			Type:      strings.TrimSpace(payload.Error.Type),
			Code:      strings.TrimSpace(payload.Error.Code),
			Param:     strings.TrimSpace(payload.Error.Param),
			RequestID: strings.TrimSpace(requestID),
		}
	}

	return &StripeAPIError{
		Status:    status,
		Message:   strings.TrimSpace(fallback),
		RequestID: strings.TrimSpace(requestID),
	}
}

func classifyStripeError(err error) (int, string, string, bool) {
	if err == nil {
		return http.StatusInternalServerError, ApiErrorTypeServerError, "Stripe request failed. Please try again.", false
	}

	var configErr *StripeConfigError
	if errors.As(err, &configErr) {
		return http.StatusBadRequest, ApiErrorTypeValidation,
			"Stripe secret key is missing. Add a restricted key in Billing settings to continue.", true
	}

	var bundleErr *StripeBundleProductNotFoundError
	if errors.As(err, &bundleErr) {
		message := "Stripe product not found for the active bundle. Update the bundle's Stripe product ID or create the product in Stripe."
		if bundleErr.ProductID != "" {
			if bundleErr.BundleKey != "" {
				message = fmt.Sprintf("Stripe product %s was not found for bundle %s. Update the bundle's Stripe product ID or create the product in Stripe.", bundleErr.ProductID, bundleErr.BundleKey)
			} else {
				message = fmt.Sprintf("Stripe product %s was not found. Update the bundle's Stripe product ID or create the product in Stripe.", bundleErr.ProductID)
			}
		}
		return http.StatusUnprocessableEntity, ApiErrorTypeValidation, message, true
	}

	var apiErr *StripeAPIError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.Status == http.StatusUnauthorized || apiErr.Status == http.StatusForbidden:
			return http.StatusBadRequest, ApiErrorTypeValidation,
				"Stripe authentication failed. If using a restricted key, ensure it has the required permissions (e.g., Coupons > Read/Write for coupon operations).", true
		case apiErr.Status == http.StatusNotFound || strings.EqualFold(apiErr.Code, "resource_missing"):
			return http.StatusBadRequest, ApiErrorTypeValidation,
				"Stripe resource not found. Verify the Stripe price or product ID.", true
		case apiErr.Status == http.StatusTooManyRequests:
			return http.StatusTooManyRequests, ApiErrorTypeRateLimited,
				"Stripe rate limit reached. Please try again in a moment.", true
		case apiErr.Status >= 500:
			return http.StatusBadGateway, ApiErrorTypeServerError,
				"Stripe is currently unavailable. Please try again later.", true
		default:
			message := strings.TrimSpace(apiErr.Message)
			if message == "" {
				message = "Stripe request failed. Please check your input and try again."
			}
			return http.StatusBadRequest, ApiErrorTypeValidation, message, true
		}
	}

	return http.StatusInternalServerError, ApiErrorTypeServerError, "Stripe request failed. Please try again.", false
}

// --- Shared types and helpers used by multiple service files ---

type checkoutSessionRecord struct {
	SessionID      string
	Status         string
	PriceID        sql.NullString
	SessionType    sql.NullString
	AmountCents    sql.NullInt64
	ScheduleID     sql.NullString
	CustomerID     sql.NullString
	CustomerEmail  sql.NullString
	SubscriptionID sql.NullString
}

type stripePriceRef struct {
	ID         string `json:"id"`
	Currency   string `json:"currency"`
	UnitAmount int64  `json:"unit_amount"`
	Recurring  struct {
		Interval string `json:"interval"`
	} `json:"recurring"`
	Metadata map[string]interface{} `json:"metadata"`
}

type stripeSubscription struct {
	ID                 string `json:"id"`
	Status             string `json:"status"`
	Customer           string `json:"customer"`
	CustomerEmail      string `json:"customer_email"`
	CancelAtPeriodEnd  bool   `json:"cancel_at_period_end"`
	CanceledAt         int64  `json:"canceled_at"`
	BillingCycleAnchor int64  `json:"billing_cycle_anchor"`
	Items              struct {
		Data []struct {
			Price stripePriceRef `json:"price"`
		} `json:"data"`
	} `json:"items"`
	Metadata map[string]interface{} `json:"metadata"`
}

// extractBillingCycleDay converts a Unix timestamp to day of month (1-28).
// Returns 0 if invalid. Days > 28 are capped to avoid short-month issues.
func extractBillingCycleDay(timestamp int64) int {
	if timestamp <= 0 {
		return 0
	}
	day := time.Unix(timestamp, 0).UTC().Day()
	if day > 28 {
		day = 28
	}
	return day
}

type stripeCustomer struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func (s *StripeService) loadCheckoutSession(sessionID string) (*checkoutSessionRecord, error) {
	record := &checkoutSessionRecord{}
	err := s.db.QueryRow(`
		SELECT session_id, status, price_id, session_type, amount_cents, schedule_id, customer_id, customer_email, subscription_id
		FROM checkout_sessions
		WHERE session_id = $1
	`, sessionID).Scan(
		&record.SessionID,
		&record.Status,
		&record.PriceID,
		&record.SessionType,
		&record.AmountCents,
		&record.ScheduleID,
		&record.CustomerID,
		&record.CustomerEmail,
		&record.SubscriptionID,
	)
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (s *StripeService) fetchSubscription(ctx context.Context, subscriptionID string) (*stripeSubscription, error) {
	body, err := s.doStripeRequest(ctx, http.MethodGet, "/v1/subscriptions/"+url.PathEscape(subscriptionID), nil, "")
	if err != nil {
		return nil, err
	}
	var resp stripeSubscription
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.ID == "" {
		return nil, fmt.Errorf("subscription %s not found", subscriptionID)
	}
	return &resp, nil
}

func (s *StripeService) findCustomerByEmail(ctx context.Context, email string) (*stripeCustomer, error) {
	if strings.TrimSpace(email) == "" {
		return nil, nil
	}
	query := fmt.Sprintf(`email:"%s"`, email)
	params := url.Values{}
	params.Set("query", query)
	params.Set("limit", "1")
	body, err := s.doStripeRequest(ctx, http.MethodGet, "/v1/customers/search?"+params.Encode(), nil, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data []stripeCustomer `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, nil
	}
	return &resp.Data[0], nil
}

func (s *StripeService) latestSubscriptionForCustomer(ctx context.Context, customerID string) (*stripeSubscription, error) {
	if strings.TrimSpace(customerID) == "" {
		return nil, nil
	}
	params := url.Values{}
	params.Set("customer", customerID)
	params.Set("limit", "1")
	params.Set("status", "all")
	body, err := s.doStripeRequest(ctx, http.MethodGet, "/v1/subscriptions?"+params.Encode(), nil, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data []stripeSubscription `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, nil
	}
	return &resp.Data[0], nil
}

func chooseUserIdentity(userHint string, sub *stripeSubscription) string {
	if strings.TrimSpace(userHint) != "" {
		return userHint
	}
	if sub == nil {
		return ""
	}
	if strings.TrimSpace(sub.CustomerEmail) != "" {
		return sub.CustomerEmail
	}
	return strings.TrimSpace(sub.Customer)
}

func (s *StripeService) extractAmount(obj map[string]interface{}, session *checkoutSessionRecord) int64 {
	if amount := s.parseStripeAmount(obj["amount_total"]); amount != 0 {
		return amount
	}
	if amount := s.parseStripeAmount(obj["amount"]); amount != 0 {
		return amount
	}
	if session != nil && session.AmountCents.Valid {
		return session.AmountCents.Int64
	}
	return 0
}

func (s *StripeService) parseStripeAmount(value interface{}) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case json.Number:
		if parsed, err := v.Int64(); err == nil {
			return parsed
		}
	case string:
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			return parsed
		}
	}
	return 0
}
