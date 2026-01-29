package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// StripeService handles Stripe payment integration
type StripeService struct {
	db                *sql.DB
	planService       *PlanService
	paymentSettings   *PaymentSettingsService
	checkoutCacheTTL  time.Duration
	httpClient        *http.Client
	apiBase           string
	mu                sync.RWMutex
	runtimeConfig     stripeRuntimeConfig
	configLoader      stripeConfigLoader
	introCouponConfig IntroCouponConfig
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
	enabled := strings.ToLower(strings.TrimSpace(os.Getenv("INTRO_ENABLED"))) == "true"
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
		if couponID := strings.TrimSpace(os.Getenv(envVar)); couponID != "" {
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

// StripePriceInfo provides typed price verification results.
type StripePriceInfo struct {
	ID          string `json:"id"`
	LookupKey   string `json:"lookup_key,omitempty"`
	Currency    string `json:"currency"`
	AmountCents int64  `json:"amount_cents"`
	Interval    string `json:"interval,omitempty"`
	Active      bool   `json:"active"`
	ProductName string `json:"product_name,omitempty"`
}

// VerifyStripePriceTyped returns typed price info (testable wrapper).
func (s *StripeService) VerifyStripePriceTyped(key string) (*StripePriceInfo, error) {
	result, err := s.VerifyStripePrice(key)
	if err != nil {
		return nil, err
	}

	info := &StripePriceInfo{}

	if id, ok := result["id"].(string); ok {
		info.ID = id
	}
	if lookupKey, ok := result["lookup_key"].(string); ok {
		info.LookupKey = lookupKey
	}
	if currency, ok := result["currency"].(string); ok {
		info.Currency = currency
	}
	if amountCents, ok := result["amount_cents"].(int64); ok {
		info.AmountCents = amountCents
	}
	if interval, ok := result["interval"].(string); ok {
		info.Interval = interval
	}
	if active, ok := result["active"].(bool); ok {
		info.Active = active
	}
	if productName, ok := result["product"].(string); ok {
		info.ProductName = productName
	}

	return info, nil
}

// NewStripeService creates a new Stripe service instance.
func NewStripeService(db *sql.DB) *StripeService {
	return NewStripeServiceWithSettings(db, NewPlanService(db), NewPaymentSettingsService(db))
}

// NewStripeServiceWithSettings wires explicit plan/payment dependencies (used by server).
func NewStripeServiceWithSettings(db *sql.DB, planService *PlanService, paymentSettings *PaymentSettingsService) *StripeService {
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

func (s *StripeService) loadStripeConfig(ctx context.Context) (stripeRuntimeConfig, error) {
	// Start with environment defaults.
	envPublishable := strings.TrimSpace(os.Getenv("STRIPE_PUBLISHABLE_KEY"))
	envSecret := strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY"))
	envWebhook := strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET"))
	apiBase := strings.TrimSpace(os.Getenv("STRIPE_API_BASE"))
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
		cfg.publishableKey = "pk_test_placeholder"
		logStructured("STRIPE_PUBLISHABLE_KEY missing - using placeholder", map[string]interface{}{
			"level":   "warn",
			"message": "STRIPE_PUBLISHABLE_KEY not set; using placeholder for development",
		})
	}
	if !cfg.hasSecret {
		cfg.secretKey = "sk_test_placeholder"
		logStructured("STRIPE_SECRET_KEY (restricted) missing - using placeholder", map[string]interface{}{
			"level":   "warn",
			"message": "STRIPE_SECRET_KEY (restricted key) not set; using placeholder for development",
		})
	}
	if !cfg.hasWebhook {
		cfg.webhookSecret = "whsec_placeholder"
		logStructured("STRIPE_WEBHOOK_SECRET missing - using placeholder", map[string]interface{}{
			"level":   "warn",
			"message": "STRIPE_WEBHOOK_SECRET not set; using placeholder for development",
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

// GetSecretValue returns the unredacted value for a specific secret field from the merged config.
// This is used by the admin reveal endpoint to show actual secret values.
// Allowed fields: "publishable_key", "secret_key", "webhook_secret"
func (s *StripeService) GetSecretValue(field string) (string, bool) {
	cfg := s.getConfig()
	switch field {
	case "publishable_key":
		return cfg.publishableKey, cfg.hasPublishable
	case "secret_key":
		return cfg.secretKey, cfg.hasSecret
	case "webhook_secret":
		return cfg.webhookSecret, cfg.hasWebhook
	default:
		return "", false
	}
}

// ConfigSnapshot returns a redacted view of the active Stripe configuration.
// Note: PublishableKeyPreview is only set when hasPublishable is true to avoid
// leaking placeholder values that could be mistaken for real configuration.
func (s *StripeService) ConfigSnapshot() *landing_page_react_vite_v1.StripeConfigSnapshot {
	cfg := s.getConfig()
	source := landing_page_react_vite_v1.ConfigSource_CONFIG_SOURCE_ENV
	if cfg.source == "database" {
		source = landing_page_react_vite_v1.ConfigSource_CONFIG_SOURCE_DATABASE
	}

	// Only show preview when a real key is configured (not the placeholder)
	var publishablePreview string
	if cfg.hasPublishable {
		publishablePreview = maskValue(cfg.publishableKey)
	}

	return &landing_page_react_vite_v1.StripeConfigSnapshot{
		PublishableKeyPreview: publishablePreview,
		PublishableKeySet:     cfg.hasPublishable,
		SecretKeySet:          cfg.hasSecret,
		WebhookSecretSet:      cfg.hasWebhook,
		Source:                source,
	}
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

func (s *StripeService) lookupCustomerID(user string) string {
	if strings.TrimSpace(user) == "" {
		return ""
	}
	var customerID sql.NullString
	err := s.db.QueryRow(`
		SELECT customer_id
		FROM subscriptions
		WHERE customer_email = $1 OR customer_id = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, user).Scan(&customerID)
	if err != nil {
		return ""
	}
	if customerID.Valid {
		return customerID.String
	}
	return ""
}

// resolveStripePriceID accepts a Stripe price ID or lookup key and returns a concrete price ID.
func (s *StripeService) resolveStripePriceID(key string) (string, error) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return "", errors.New("stripe price id is required")
	}
	if strings.HasPrefix(trimmed, "price_") {
		return trimmed, nil
	}

	// Treat as lookup key; fetch the first matching price.
	values := url.Values{}
	values.Set("lookup_keys[]", trimmed)
	values.Set("limit", "1")
	values.Set("expand[]", "data.product")
	path := "/v1/prices?" + values.Encode()

	body, err := s.doStripeRequest(context.Background(), http.MethodGet, path, nil, "")
	if err != nil {
		return "", err
	}

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("decode stripe price lookup: %w", err)
	}
	if len(resp.Data) == 0 || strings.TrimSpace(resp.Data[0].ID) == "" {
		return "", fmt.Errorf("no stripe price found for lookup key %s", trimmed)
	}
	return resp.Data[0].ID, nil
}

// VerifyStripePrice fetches price details from Stripe using either a price ID or lookup key.
func (s *StripeService) VerifyStripePrice(key string) (map[string]interface{}, error) {
	resolved, err := s.resolveStripePriceID(key)
	if err != nil {
		return nil, err
	}

	path := "/v1/prices/" + url.PathEscape(resolved) + "?expand[]=product"
	body, err := s.doStripeRequest(context.Background(), http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}

	var price struct {
		ID         string `json:"id"`
		LookupKey  string `json:"lookup_key"`
		Currency   string `json:"currency"`
		UnitAmount int64  `json:"unit_amount"`
		Active     bool   `json:"active"`
		Recurring  *struct {
			Interval string `json:"interval"`
		} `json:"recurring"`
		Product struct {
			Name string `json:"name"`
		} `json:"product"`
	}
	if err := json.Unmarshal(body, &price); err != nil {
		return nil, fmt.Errorf("decode stripe price: %w", err)
	}

	interval := "one_time"
	if price.Recurring != nil && strings.TrimSpace(price.Recurring.Interval) != "" {
		interval = price.Recurring.Interval
	}

	return map[string]interface{}{
		"id":           price.ID,
		"lookup_key":   price.LookupKey,
		"currency":     price.Currency,
		"amount_cents": price.UnitAmount,
		"interval":     interval,
		"active":       price.Active,
		"product":      price.Product.Name,
	}, nil
}

// CreateCheckoutSession creates a Stripe checkout session
// [REQ:STRIPE-ROUTES] POST /api/checkout/create endpoint
func (s *StripeService) CreateCheckoutSession(priceID string, successURL string, cancelURL string, customerEmail string) (*landing_page_react_vite_v1.CheckoutSession, error) {
	ctx := context.Background()
	plan, err := s.planService.GetPlanByPriceID(priceID)
	if err != nil {
		return nil, fmt.Errorf("price %s not found: %w", priceID, err)
	}

	resolvedPriceID, err := s.resolveStripePriceID(priceID)
	if err != nil {
		return nil, fmt.Errorf("resolve price %s: %w", priceID, err)
	}

	if successURL == "" {
		successURL = "/success"
	}
	if cancelURL == "" {
		cancelURL = "/cancel"
	}

	mode := "subscription"
	sessionKind := landing_page_react_vite_v1.SessionKind_SESSION_KIND_SUBSCRIPTION
	sessionType := sessionTypeSubscription
	if plan.Kind == landing_page_react_vite_v1.PlanKind_PLAN_KIND_CREDITS_TOPUP {
		mode = "payment"
		sessionKind = landing_page_react_vite_v1.SessionKind_SESSION_KIND_CREDITS_TOPUP
		sessionType = sessionTypeCreditsTopup
	} else if plan.Kind == landing_page_react_vite_v1.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION {
		mode = "payment"
		sessionKind = landing_page_react_vite_v1.SessionKind_SESSION_KIND_SUPPORTER_CONTRIBUTION
		sessionType = sessionTypeSupporterContribution
	}

	values := url.Values{}
	values.Set("mode", mode)
	values.Set("success_url", successURL)
	values.Set("cancel_url", cancelURL)
	values.Set("line_items[0][price]", resolvedPriceID)
	values.Set("line_items[0][quantity]", "1")
	values.Set("metadata[bundle_key]", plan.BundleKey)
	values.Set("metadata[plan_tier]", plan.PlanTier)
	if customerEmail != "" {
		values.Set("customer_email", customerEmail)
		values.Set("metadata[user_identity]", customerEmail)
	}

	if existingCustomerID := s.lookupCustomerID(customerEmail); existingCustomerID != "" {
		values.Del("customer_email")
		values.Set("customer", existingCustomerID)
	}

	// Apply intro coupon for eligible monthly subscriptions
	var appliedCouponID string
	if mode == "subscription" && plan.BillingInterval == landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_MONTH && customerEmail != "" {
		eligible, eligErr := s.checkIntroEligibility(ctx, customerEmail)
		if eligErr != nil {
			logStructuredError("intro_eligibility_check_failed", map[string]interface{}{
				"email": customerEmail,
				"error": eligErr.Error(),
			})
			// Continue without intro pricing on error
		} else if eligible {
			couponID := s.introCouponConfig.GetCouponForTier(plan.PlanTier)
			if couponID != "" {
				values.Set("discounts[0][coupon]", couponID)
				values.Set("metadata[intro_coupon_applied]", couponID)
				appliedCouponID = couponID
				logStructured("intro_coupon_applying", map[string]interface{}{
					"level":     "info",
					"email":     customerEmail,
					"coupon_id": couponID,
					"plan_tier": plan.PlanTier,
				})
			}
		}
	}

	body, err := s.doStripeForm(ctx, http.MethodPost, "/v1/checkout/sessions", values)
	if err != nil {
		return nil, fmt.Errorf("create checkout for price %s: %w", priceID, err)
	}

	var resp struct {
		ID            string `json:"id"`
		URL           string `json:"url"`
		Status        string `json:"status"`
		Subscription  string `json:"subscription"`
		Customer      string `json:"customer"`
		CustomerEmail string `json:"customer_email"`
		AmountTotal   int64  `json:"amount_total"`
		PaymentStatus string `json:"payment_status"`
		Mode          string `json:"mode"`
		Currency      string `json:"currency"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode stripe response: %w", err)
	}

	amount := resp.AmountTotal
	if amount == 0 {
		amount = plan.AmountCents
	}

	meta := map[string]interface{}{
		"bundle_key": plan.BundleKey,
		"plan_tier":  plan.PlanTier,
		"kind":       plan.Kind.String(),
	}
	if appliedCouponID != "" {
		meta["intro_coupon_applied"] = appliedCouponID
	}
	metaBytes, _ := json.Marshal(meta)

	_, err = s.db.Exec(`
		INSERT INTO checkout_sessions (session_id, customer_email, customer_id, price_id, subscription_id, status, session_type, amount_cents, metadata, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW())
		ON CONFLICT (session_id) DO UPDATE SET
			customer_email = EXCLUDED.customer_email,
			customer_id = EXCLUDED.customer_id,
			price_id = EXCLUDED.price_id,
			subscription_id = EXCLUDED.subscription_id,
			status = EXCLUDED.status,
			session_type = EXCLUDED.session_type,
			amount_cents = EXCLUDED.amount_cents,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`, resp.ID, resp.CustomerEmail, resp.Customer, priceID, resp.Subscription, resp.Status, sessionType, amount, string(metaBytes))
	if err != nil {
		return nil, err
	}

	cfg := s.getConfig()
	session := &landing_page_react_vite_v1.CheckoutSession{
		SessionId:      resp.ID,
		SessionKind:    sessionKind,
		Status:         landing_page_react_vite_v1.CheckoutSessionStatus_CHECKOUT_SESSION_STATUS_OPEN,
		Url:            resp.URL,
		PublishableKey: cfg.publishableKey,
		CustomerEmail:  resp.CustomerEmail,
		StripePriceId:  priceID,
		AmountCents:    amount,
		Currency:       plan.Currency,
		SuccessUrl:     successURL,
		CancelUrl:      cancelURL,
		CreatedAt:      timestamppb.Now(),
	}

	return session, nil
}

// VerifyWebhookSignature validates the Stripe webhook signature
// [REQ:STRIPE-SIG] Webhook signature verification
func (s *StripeService) VerifyWebhookSignature(payload []byte, signature string) bool {
	cfg := s.getConfig()
	// [REQ:STRIPE-CONFIG] Uses webhook secret from environment/admin settings
	if !cfg.hasWebhook {
		logStructured("Stripe webhook secret not configured", map[string]interface{}{"level": "warn"})
		return false
	}

	// Extract timestamp and signature from header
	// Format: t=timestamp,v1=signature
	parts := strings.Split(signature, ",")
	if len(parts) < 2 {
		return false
	}

	var timestamp, sig string
	for _, part := range parts {
		if strings.HasPrefix(part, "t=") {
			timestamp = strings.TrimPrefix(part, "t=")
		} else if strings.HasPrefix(part, "v1=") {
			sig = strings.TrimPrefix(part, "v1=")
		}
	}

	if timestamp == "" || sig == "" {
		return false
	}

	// Construct signed payload: timestamp.payload
	signedPayload := timestamp + "." + string(payload)

	// Compute HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(cfg.webhookSecret))
	mac.Write([]byte(signedPayload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	return hmac.Equal([]byte(sig), []byte(expectedSig))
}

// HandleWebhook processes Stripe webhook events
// [REQ:STRIPE-ROUTES] POST /api/webhooks/stripe endpoint
func (s *StripeService) HandleWebhook(body []byte, signature string) error {
	// [REQ:STRIPE-SIG] Verify signature before processing
	if !s.VerifyWebhookSignature(body, signature) {
		return errors.New("invalid webhook signature")
	}

	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		return err
	}

	eventType, ok := event["type"].(string)
	if !ok {
		return errors.New("missing event type")
	}

	// Extract event ID for idempotency
	eventID, _ := event["id"].(string)

	data, ok := event["data"].(map[string]interface{})
	if !ok {
		return errors.New("missing event data")
	}

	obj, ok := data["object"].(map[string]interface{})
	if !ok {
		return errors.New("missing event object")
	}

	// Handle different event types
	switch eventType {
	case "checkout.session.completed":
		return s.handleCheckoutCompleted(obj, eventID)
	case "customer.subscription.created":
		return s.handleSubscriptionCreated(obj)
	case "customer.subscription.updated":
		return s.handleSubscriptionUpdated(obj)
	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(obj)
	case "invoice.paid":
		return s.handleInvoicePaid(obj)
	case "invoice.payment_failed":
		return s.handleInvoicePaymentFailed(obj)
	default:
		logStructured("Unhandled webhook event", map[string]interface{}{
			"event_type": eventType,
		})
	}

	return nil
}

func (s *StripeService) handleCheckoutCompleted(obj map[string]interface{}, stripeEventID string) error {
	sessionID, ok := obj["id"].(string)
	if !ok {
		return errors.New("missing session id")
	}

	customerEmail, _ := obj["customer_email"].(string)
	customerID, _ := obj["customer"].(string)
	subscriptionID, _ := obj["subscription"].(string)

	// Normalize email for consistency
	customerEmail = NormalizeEmail(customerEmail)

	// Link user account to Stripe customer (creates user if not exists)
	if customerEmail != "" && customerID != "" {
		if err := s.linkUserToStripeCustomer(customerEmail, customerID); err != nil {
			logStructuredError("link_stripe_customer_failed", map[string]interface{}{
				"email":       customerEmail,
				"customer_id": customerID,
				"error":       err.Error(),
			})
			// Continue - don't fail checkout for this
		}
	}

	sessionRec, err := s.loadCheckoutSession(sessionID)
	if err != nil {
		return err
	}

	if sessionRec.Status == "complete" {
		logStructured("checkout.session.completed ignored (duplicate)", map[string]interface{}{
			"session_id": sessionID,
		})
		return nil
	}

	if _, err := s.db.Exec(`
		UPDATE checkout_sessions
		SET status = $1, subscription_id = $2, customer_id = $3, customer_email = $4, updated_at = $5
		WHERE session_id = $6
	`, "complete", subscriptionID, customerID, customerEmail, time.Now(), sessionID); err != nil {
		return err
	}

	var plan *PlanOption
	if sessionRec.PriceID.Valid {
		if p, planErr := s.planService.GetPlanByPriceID(sessionRec.PriceID.String); planErr == nil {
			plan = p
		} else {
			logStructured("plan metadata missing during checkout completion", map[string]interface{}{
				"price_id": sessionRec.PriceID.String,
				"error":    planErr.Error(),
			})
		}
	}

	amountCents := s.extractAmount(obj, sessionRec)

	switch {
	case plan != nil && plan.Kind == landing_page_react_vite_v1.PlanKind_PLAN_KIND_CREDITS_TOPUP:
		return s.handleCreditTopup(customerEmail, amountCents, plan, stripeEventID, map[string]interface{}{
			"session_id": sessionID,
		})
	case plan != nil && plan.Kind == landing_page_react_vite_v1.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION:
		logStructured("supporter contribution received", map[string]interface{}{
			"session_id": sessionID,
			"email":      customerEmail,
			"amount":     amountCents,
		})
		return nil
	default:
		return s.handleSubscriptionCompletion(subscriptionID, customerID, customerEmail, plan, sessionRec, amountCents)
	}
}

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

func (s *StripeService) persistSubscriptionFromStripe(userHint string, sub *stripeSubscription) (*landing_page_react_vite_v1.SubscriptionStatus, error) {
	if sub == nil {
		return nil, nil
	}

	priceID := ""
	if len(sub.Items.Data) > 0 {
		priceID = sub.Items.Data[0].Price.ID
	}

	planTier := ""
	bundleKey := s.planService.BundleKey()

	if planTierVal, ok := sub.Metadata["plan_tier"].(string); ok && planTierVal != "" {
		planTier = planTierVal
	}
	if bundleVal, ok := sub.Metadata["bundle_key"].(string); ok && bundleVal != "" {
		bundleKey = bundleVal
	}

	if priceID != "" {
		if plan, err := s.planService.GetPlanByPriceID(priceID); err == nil {
			if plan.PlanTier != "" {
				planTier = plan.PlanTier
			}
			if plan.BundleKey != "" {
				bundleKey = plan.BundleKey
			}
		} else {
			logStructured("stripe_plan_lookup_failed", map[string]interface{}{
				"level":    "warn",
				"price_id": priceID,
				"error":    err.Error(),
			})
		}
	}
	if strings.TrimSpace(planTier) == "" && strings.TrimSpace(priceID) != "" {
		if inferred, ok := detectTierToken(priceID); ok {
			planTier = inferred
		}
	}
	if strings.TrimSpace(planTier) != "" {
		if _, err := normalizePlanTier(planTier); err != nil {
			logStructured("stripe_subscription_plan_tier_invalid", map[string]interface{}{
				"level":        "warn",
				"plan_tier":    planTier,
				"price_id":     priceID,
				"subscription": sub.ID,
			})
			planTier = ""
		}
	}

	state := mapSubscriptionState(sub.Status)
	now := time.Now()

	var canceledAt *time.Time
	if sub.CanceledAt > 0 {
		ts := time.Unix(sub.CanceledAt, 0)
		canceledAt = &ts
	}

	// Extract billing cycle day from anchor timestamp
	billingCycleStart := extractBillingCycleDay(sub.BillingCycleAnchor)

	_, err := s.db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, price_id, bundle_key, billing_cycle_start, canceled_at, created_at, updated_at)
		VALUES ($1::varchar,$2::varchar,$3::varchar,$4::varchar,$5::varchar,$6::varchar,$7::varchar,$8::int,$9::timestamp,COALESCE((SELECT created_at FROM subscriptions WHERE subscription_id = $1::varchar), NOW()), NOW())
		ON CONFLICT (subscription_id) DO UPDATE SET
			customer_id = EXCLUDED.customer_id,
			customer_email = EXCLUDED.customer_email,
			status = EXCLUDED.status,
			plan_tier = COALESCE(NULLIF(EXCLUDED.plan_tier,''), subscriptions.plan_tier),
			price_id = COALESCE(NULLIF(EXCLUDED.price_id,''), subscriptions.price_id),
			bundle_key = COALESCE(NULLIF(EXCLUDED.bundle_key,''), subscriptions.bundle_key),
			billing_cycle_start = EXCLUDED.billing_cycle_start,
			canceled_at = EXCLUDED.canceled_at,
			updated_at = NOW()
	`, sub.ID, sub.Customer, sub.CustomerEmail, legacyStateLabel(state), planTier, priceID, bundleKey, billingCycleStart, canceledAt)
	if err != nil {
		return nil, err
	}

	user := chooseUserIdentity(userHint, sub)
	status := &landing_page_react_vite_v1.SubscriptionStatus{
		State:        state,
		UserIdentity: user,
		CachedAt:     timestamppb.New(now),
	}
	if sub.ID != "" {
		status.SubscriptionId = proto.String(sub.ID)
	}
	if priceID != "" {
		status.StripePriceId = proto.String(priceID)
	}
	if planTier != "" {
		status.PlanTier = proto.String(planTier)
	}
	if bundleKey != "" {
		status.BundleKey = proto.String(bundleKey)
	}
	if canceledAt != nil {
		status.CanceledAt = timestamppb.New(*canceledAt)
	}

	return status, nil
}

func (s *StripeService) refreshSubscriptionFromStripe(userIdentity string, currentSubscriptionID string) (*landing_page_react_vite_v1.SubscriptionStatus, error) {
	ctx := context.Background()

	if currentSubscriptionID != "" {
		if sub, err := s.fetchSubscription(ctx, currentSubscriptionID); err == nil {
			return s.persistSubscriptionFromStripe(userIdentity, sub)
		} else {
			logStructured("stripe fetch subscription failed", map[string]interface{}{
				"level":   "warn",
				"id":      currentSubscriptionID,
				"error":   err.Error(),
				"user_id": userIdentity,
			})
		}
	}

	if strings.HasPrefix(strings.TrimSpace(userIdentity), "sub_") {
		if sub, err := s.fetchSubscription(ctx, userIdentity); err == nil {
			return s.persistSubscriptionFromStripe(userIdentity, sub)
		}
	}

	if strings.Contains(userIdentity, "@") {
		customer, err := s.findCustomerByEmail(ctx, userIdentity)
		if err != nil {
			return nil, err
		}
		if customer == nil {
			return nil, nil
		}
		sub, err := s.latestSubscriptionForCustomer(ctx, customer.ID)
		if err != nil {
			return nil, err
		}
		return s.persistSubscriptionFromStripe(userIdentity, sub)
	}

	if strings.TrimSpace(userIdentity) != "" {
		sub, err := s.latestSubscriptionForCustomer(ctx, userIdentity)
		if err != nil {
			return nil, err
		}
		return s.persistSubscriptionFromStripe(userIdentity, sub)
	}

	return nil, nil
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

func (s *StripeService) handleSubscriptionCompletion(subscriptionID, customerID, customerEmail string, plan *PlanOption, session *checkoutSessionRecord, amountCents int64) error {
	if plan == nil {
		// Without plan metadata we cannot create enriched entries
		return nil
	}

	if subscriptionID == "" {
		return errors.New("subscription id required for subscription completion")
	}

	if amountCents == 0 {
		amountCents = plan.AmountCents
	}

	now := time.Now()
	_, err := s.db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, price_id, bundle_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (subscription_id) DO UPDATE
		SET status = $4, customer_email = $3, customer_id = $2, plan_tier = $5, price_id = $6, bundle_key = $7, updated_at = $9
	`, subscriptionID, customerID, customerEmail, "active", plan.PlanTier, plan.StripePriceId, plan.BundleKey, now, now)
	if err != nil {
		return err
	}

	if plan.IntroEnabled && plan.BillingInterval == landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_MONTH {
		scheduleID, err := s.createSubscriptionSchedule(subscriptionID, plan, amountCents)
		if err != nil {
			return err
		}
		if scheduleID != "" && session != nil {
			if _, err := s.db.Exec(`
				UPDATE checkout_sessions
				SET schedule_id = $1
				WHERE session_id = $2
			`, scheduleID, session.SessionID); err != nil {
				return err
			}
		}
	}

	meta := map[string]interface{}{
		"customer_email":  customerEmail,
		"customer_id":     customerID,
		"subscription_id": subscriptionID,
		"plan_tier":       plan.PlanTier,
		"price_id":        plan.StripePriceId,
		"session_type":    sessionTypeSubscription,
	}
	if session != nil {
		meta["session_id"] = session.SessionID
	}
	logStructured("Checkout session completed", meta)

	return nil
}

func (s *StripeService) createSubscriptionSchedule(subscriptionID string, plan *PlanOption, amountCents int64) (string, error) {
	if plan == nil || subscriptionID == "" {
		return "", nil
	}

	if amountCents == 0 {
		amountCents = plan.AmountCents
	}

	scheduleID := fmt.Sprintf("sched_%d", time.Now().UnixNano())
	nextBilling := time.Now().Add(s.billingIntervalDuration(plan.BillingInterval))

	meta := map[string]interface{}{
		"plan_rank":          plan.PlanRank,
		"intro_enabled":      plan.IntroEnabled,
		"intro_periods":      plan.IntroPeriods,
		"billing_interval":   billingIntervalLabel(plan.BillingInterval),
		"subscription_price": plan.AmountCents,
	}
	metaBytes, _ := json.Marshal(meta)

	_, err := s.db.Exec(`
		INSERT INTO subscription_schedules (
			schedule_id, subscription_id, price_id, billing_interval,
			intro_enabled, intro_amount_cents, intro_periods, normal_amount_cents,
			next_billing_at, status, metadata, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'active',$10,NOW(),NOW())
		ON CONFLICT (schedule_id) DO UPDATE SET
			subscription_id = EXCLUDED.subscription_id,
			price_id = EXCLUDED.price_id,
			billing_interval = EXCLUDED.billing_interval,
			intro_enabled = EXCLUDED.intro_enabled,
			intro_amount_cents = EXCLUDED.intro_amount_cents,
			intro_periods = EXCLUDED.intro_periods,
			normal_amount_cents = EXCLUDED.normal_amount_cents,
			next_billing_at = EXCLUDED.next_billing_at,
			status = 'active',
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`, scheduleID, subscriptionID, plan.StripePriceId, billingIntervalLabel(plan.BillingInterval),
		plan.IntroEnabled, plan.IntroAmountCents, plan.IntroPeriods, amountCents,
		nextBilling, string(metaBytes))
	if err != nil {
		return "", err
	}

	return scheduleID, nil
}

func (s *StripeService) billingIntervalDuration(interval landing_page_react_vite_v1.BillingInterval) time.Duration {
	switch interval {
	case landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_YEAR:
		return 365 * 24 * time.Hour
	case landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_MONTH:
		return 30 * 24 * time.Hour
	default:
		return 30 * 24 * time.Hour
	}
}

func (s *StripeService) handleCreditTopup(customerEmail string, amountCents int64, plan *PlanOption, stripeEventID string, metadata map[string]interface{}) error {
	if customerEmail == "" {
		return errors.New("customer email required for credit top-up")
	}

	// Normalize email for consistency
	customerEmail = NormalizeEmail(customerEmail)

	if amountCents == 0 {
		amountCents = plan.AmountCents
	}

	bundle, err := s.planService.GetBundleProduct()
	if err != nil {
		return err
	}

	if amountCents == 0 || bundle == nil {
		return nil
	}

	credits := (bundle.CreditsPerUsd * amountCents) / 100
	if credits <= 0 {
		return nil
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["price_id"] = plan.StripePriceId
	metadata["session_type"] = sessionTypeCreditsTopup

	return s.addCredits(customerEmail, credits, "credit_topup", stripeEventID, metadata)
}

func (s *StripeService) addCredits(customerEmail string, amount int64, txnType string, stripeEventID string, metadata map[string]interface{}) error {
	if customerEmail == "" || amount <= 0 {
		return nil
	}

	// Normalize email for consistency
	customerEmail = NormalizeEmail(customerEmail)

	// Idempotency check: if stripeEventID is provided, check if already processed
	if stripeEventID != "" {
		var exists bool
		err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM credit_transactions WHERE stripe_event_id = $1)`, stripeEventID).Scan(&exists)
		if err == nil && exists {
			logStructured("credit_topup_already_processed", map[string]interface{}{
				"level":           "info",
				"stripe_event_id": stripeEventID,
				"customer_email":  customerEmail,
			})
			return nil // Already processed - idempotent success
		}
	}

	_, err := s.db.Exec(`
		INSERT INTO credit_wallets (customer_email, balance_credits, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (customer_email) DO UPDATE
		SET balance_credits = credit_wallets.balance_credits + $2, updated_at = NOW()
	`, customerEmail, amount)
	if err != nil {
		return err
	}

	metaBytes, _ := json.Marshal(metadata)

	// Store stripe_event_id for idempotency (nullable)
	var eventIDParam interface{}
	if stripeEventID != "" {
		eventIDParam = stripeEventID
	}

	_, err = s.db.Exec(`
		INSERT INTO credit_transactions (customer_email, amount_credits, transaction_type, stripe_event_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, customerEmail, amount, txnType, eventIDParam, string(metaBytes))
	return err
}

func (s *StripeService) handleSubscriptionCreated(obj map[string]interface{}) error {
	subscriptionID, ok := obj["id"].(string)
	if !ok {
		return errors.New("missing subscription id")
	}

	status, _ := obj["status"].(string)
	customerID, _ := obj["customer"].(string)

	_, err := s.db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (subscription_id) DO UPDATE
		SET status = $3, updated_at = $5
	`, subscriptionID, customerID, status, time.Now(), time.Now())

	return err
}

func (s *StripeService) handleSubscriptionUpdated(obj map[string]interface{}) error {
	subscriptionID, ok := obj["id"].(string)
	if !ok {
		return errors.New("missing subscription id")
	}

	status, _ := obj["status"].(string)
	if payload, err := json.Marshal(obj); err == nil {
		var sub stripeSubscription
		if err := json.Unmarshal(payload, &sub); err == nil && sub.ID != "" {
			if _, persistErr := s.persistSubscriptionFromStripe("", &sub); persistErr == nil {
				return nil
			}
		}
	}

	// [REQ:SUB-CACHE] Cache invalidation - update subscription status
	_, err := s.db.Exec(`
		UPDATE subscriptions
		SET status = $1, updated_at = $2
		WHERE subscription_id = $3
	`, status, time.Now(), subscriptionID)

	return err
}

func (s *StripeService) handleSubscriptionDeleted(obj map[string]interface{}) error {
	subscriptionID, ok := obj["id"].(string)
	if !ok {
		return errors.New("missing subscription id")
	}

	now := time.Now()
	if payload, err := json.Marshal(obj); err == nil {
		var sub stripeSubscription
		if err := json.Unmarshal(payload, &sub); err == nil && sub.ID != "" {
			sub.Status = "canceled"
			sub.CanceledAt = now.Unix()
			if _, persistErr := s.persistSubscriptionFromStripe("", &sub); persistErr == nil {
				return nil
			}
		}
	}

	_, err := s.db.Exec(`
		UPDATE subscriptions
		SET status = $1, canceled_at = $2, updated_at = $3
		WHERE subscription_id = $4
	`, "canceled", now, now, subscriptionID)

	return err
}

func (s *StripeService) extractInvoicePriceID(obj map[string]interface{}) string {
	// Invoice price may live under lines.data[0].price.id
	lines, ok := obj["lines"].(map[string]interface{})
	if !ok {
		return ""
	}
	rawData, ok := lines["data"].([]interface{})
	if !ok || len(rawData) == 0 {
		return ""
	}
	first, ok := rawData[0].(map[string]interface{})
	if !ok {
		return ""
	}
	price, ok := first["price"].(map[string]interface{})
	if !ok {
		return ""
	}
	if id, ok := price["id"].(string); ok {
		return id
	}
	return ""
}

func (s *StripeService) persistInvoiceStatus(subscriptionID, customerID, customerEmail, priceID, status string) error {
	if subscriptionID == "" {
		return nil
	}

	planTier := ""
	bundleKey := s.planService.BundleKey()
	if priceID != "" {
		if plan, err := s.planService.GetPlanByPriceID(priceID); err == nil {
			planTier = plan.PlanTier
			if plan.BundleKey != "" {
				bundleKey = plan.BundleKey
			}
		} else {
			logStructured("stripe_plan_lookup_failed", map[string]interface{}{
				"level":    "warn",
				"price_id": priceID,
				"error":    err.Error(),
			})
		}
	}

	if strings.TrimSpace(planTier) == "" {
		var current sql.NullString
		if err := s.db.QueryRow(`SELECT plan_tier FROM subscriptions WHERE subscription_id = $1`, subscriptionID).Scan(&current); err == nil && current.Valid {
			planTier = current.String
		}
	}
	if strings.TrimSpace(planTier) == "" && strings.TrimSpace(priceID) != "" {
		if inferred, ok := detectTierToken(priceID); ok {
			planTier = inferred
		}
	}
	if strings.TrimSpace(planTier) != "" {
		if _, err := normalizePlanTier(planTier); err != nil {
			logStructured("stripe_subscription_plan_tier_invalid", map[string]interface{}{
				"level":        "warn",
				"plan_tier":    planTier,
				"price_id":     priceID,
				"subscription": subscriptionID,
			})
			planTier = ""
		}
	}

	now := time.Now()
	_, err := s.db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, price_id, bundle_key, created_at, updated_at)
		VALUES ($1::varchar,$2::varchar,$3::varchar,$4::varchar,$5::varchar,$6::varchar,$7::varchar,COALESCE((SELECT created_at FROM subscriptions WHERE subscription_id = $1::varchar), $8::timestamp), $8::timestamp)
		ON CONFLICT (subscription_id) DO UPDATE SET
			customer_id = COALESCE(EXCLUDED.customer_id, subscriptions.customer_id),
			customer_email = COALESCE(EXCLUDED.customer_email, subscriptions.customer_email),
			status = $4,
			plan_tier = COALESCE(NULLIF($5,''), subscriptions.plan_tier),
			price_id = COALESCE(NULLIF($6,''), subscriptions.price_id),
			bundle_key = COALESCE(NULLIF($7,''), subscriptions.bundle_key),
			updated_at = $8
	`, subscriptionID, customerID, customerEmail, status, planTier, priceID, bundleKey, now)
	return err
}

func (s *StripeService) handleInvoicePaid(obj map[string]interface{}) error {
	subscriptionID, _ := obj["subscription"].(string)
	customerID, _ := obj["customer"].(string)
	customerEmail, _ := obj["customer_email"].(string)
	priceID := s.extractInvoicePriceID(obj)
	billingReason, _ := obj["billing_reason"].(string)

	if err := s.persistInvoiceStatus(subscriptionID, customerID, customerEmail, priceID, "active"); err != nil {
		return err
	}

	// Check if this is a subscription creation with an intro coupon
	// billing_reason == "subscription_create" indicates first invoice
	if billingReason == "subscription_create" && customerEmail != "" {
		couponID := s.extractIntroCouponFromInvoice(obj)
		if couponID != "" {
			// Determine plan tier from price
			planTier := ""
			if priceID != "" {
				if plan, err := s.planService.GetPlanByPriceID(priceID); err == nil {
					planTier = plan.PlanTier
				}
			}

			if err := s.markIntroUsed(context.Background(), customerEmail, customerID, couponID, planTier, subscriptionID); err != nil {
				logStructuredError("mark_intro_used_failed", map[string]interface{}{
					"email":           customerEmail,
					"customer_id":     customerID,
					"coupon_id":       couponID,
					"subscription_id": subscriptionID,
					"error":           err.Error(),
				})
				// Don't fail the webhook - intro tracking is non-critical
			}
		}
	}

	// Proactively refresh from Stripe if we have an ID to backfill details
	if subscriptionID != "" {
		if refreshed, err := s.refreshSubscriptionFromStripe(customerEmail, subscriptionID); err == nil && refreshed != nil {
			return nil
		}
	}
	return nil
}

// extractIntroCouponFromInvoice extracts the intro coupon ID from an invoice object.
// Returns empty string if no intro coupon was applied.
func (s *StripeService) extractIntroCouponFromInvoice(obj map[string]interface{}) string {
	// Check discount object (single discount)
	if discount, ok := obj["discount"].(map[string]interface{}); ok {
		if coupon, ok := discount["coupon"].(map[string]interface{}); ok {
			if couponID, ok := coupon["id"].(string); ok {
				if s.isIntroCoupon(couponID) {
					return couponID
				}
			}
		}
	}

	// Check discounts array (multiple discounts)
	if discounts, ok := obj["discounts"].([]interface{}); ok {
		for _, d := range discounts {
			if discountMap, ok := d.(map[string]interface{}); ok {
				if coupon, ok := discountMap["coupon"].(map[string]interface{}); ok {
					if couponID, ok := coupon["id"].(string); ok {
						if s.isIntroCoupon(couponID) {
							return couponID
						}
					}
				}
			}
		}
	}

	// Check total_discount_amounts for coupon references
	if tdAmounts, ok := obj["total_discount_amounts"].([]interface{}); ok {
		for _, tda := range tdAmounts {
			if tdaMap, ok := tda.(map[string]interface{}); ok {
				if discount, ok := tdaMap["discount"].(map[string]interface{}); ok {
					if coupon, ok := discount["coupon"].(map[string]interface{}); ok {
						if couponID, ok := coupon["id"].(string); ok {
							if s.isIntroCoupon(couponID) {
								return couponID
							}
						}
					}
				}
			}
		}
	}

	return ""
}

// isIntroCoupon checks if a coupon ID matches one of our configured intro coupons.
func (s *StripeService) isIntroCoupon(couponID string) bool {
	if !s.introCouponConfig.Enabled {
		return false
	}
	for _, configuredID := range s.introCouponConfig.CouponMap {
		if configuredID == couponID {
			return true
		}
	}
	return false
}

func (s *StripeService) handleInvoicePaymentFailed(obj map[string]interface{}) error {
	subscriptionID, _ := obj["subscription"].(string)
	customerID, _ := obj["customer"].(string)
	customerEmail, _ := obj["customer_email"].(string)
	priceID := s.extractInvoicePriceID(obj)

	if err := s.persistInvoiceStatus(subscriptionID, customerID, customerEmail, priceID, "past_due"); err != nil {
		return err
	}
	return nil
}

// VerifySubscription checks subscription status for a user
// [REQ:SUB-VERIFY] GET /api/subscription/verify endpoint
func (s *StripeService) VerifySubscription(userIdentity string) (*landing_page_react_vite_v1.SubscriptionStatus, error) {
	user := strings.TrimSpace(userIdentity)
	if user == "" {
		return &landing_page_react_vite_v1.SubscriptionStatus{
			State:        landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE,
			UserIdentity: "",
			Message:      proto.String("user not provided"),
		}, nil
	}

	var status string
	var canceledAt *time.Time
	var updatedAt time.Time
	var priceID sql.NullString
	var planTier sql.NullString
	var bundleKey sql.NullString
	var subscriptionID sql.NullString

	err := s.db.QueryRow(`
		SELECT status, canceled_at, updated_at, price_id, plan_tier, bundle_key, subscription_id
		FROM subscriptions
		WHERE customer_email = $1 OR customer_id = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, user).Scan(&status, &canceledAt, &updatedAt, &priceID, &planTier, &bundleKey, &subscriptionID)

	needRefresh := false
	if err == sql.ErrNoRows {
		needRefresh = true
	} else if err != nil {
		return nil, err
	} else if time.Since(updatedAt) > s.checkoutCacheTTL {
		needRefresh = true
		logStructured("Subscription cache stale", map[string]interface{}{
			"level":         "warn",
			"user_identity": user,
			"cache_age_ms":  time.Since(updatedAt).Milliseconds(),
		})
	}

	if needRefresh {
		if refreshed, err := s.refreshSubscriptionFromStripe(user, subscriptionID.String); err == nil && refreshed != nil {
			return refreshed, nil
		} else if err != nil {
			logStructured("Stripe verification fallback to cache", map[string]interface{}{
				"level": "warn",
				"user":  user,
				"error": err.Error(),
			})
		}
	}

	if err == sql.ErrNoRows {
		return &landing_page_react_vite_v1.SubscriptionStatus{
			State:        landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE,
			UserIdentity: user,
			Message:      proto.String("No subscription found"),
		}, nil
	}

	state := mapSubscriptionState(status)
	result := &landing_page_react_vite_v1.SubscriptionStatus{
		State:        state,
		UserIdentity: user,
		CachedAt:     timestamppb.New(updatedAt),
		CacheAgeMs:   time.Since(updatedAt).Milliseconds(),
	}
	if canceledAt != nil {
		result.CanceledAt = timestamppb.New(*canceledAt)
	}
	if priceID.Valid {
		result.StripePriceId = proto.String(priceID.String)
		if planTier.String == "" {
			if plan, err := s.planService.GetPlanByPriceID(priceID.String); err == nil {
				planTier.String = plan.PlanTier
				bundleKey.String = plan.BundleKey
			}
		}
		if planTier.String != "" {
			if _, err := normalizePlanTier(planTier.String); err != nil {
				logStructured("stripe_subscription_plan_tier_invalid", map[string]interface{}{
					"level":        "warn",
					"plan_tier":    planTier.String,
					"price_id":     priceID.String,
					"subscription": subscriptionID.String,
				})
				planTier.String = ""
			} else if subscriptionID.Valid {
				_, _ = s.db.Exec(`
					UPDATE subscriptions
					SET plan_tier = COALESCE(NULLIF($1,''), plan_tier),
						bundle_key = COALESCE(NULLIF($2,''), bundle_key),
						updated_at = NOW()
					WHERE subscription_id = $3
				`, planTier.String, bundleKey.String, subscriptionID.String)
			}
		}
	}
	if planTier.Valid {
		result.PlanTier = proto.String(planTier.String)
	}
	if bundleKey.Valid {
		result.BundleKey = proto.String(bundleKey.String)
	}
	if subscriptionID.Valid {
		result.SubscriptionId = proto.String(subscriptionID.String)
	}

	return result, nil
}

// CancelSubscription cancels an active subscription
// [REQ:SUB-CANCEL] POST /api/subscription/cancel endpoint
func (s *StripeService) CancelSubscription(userIdentity string) (*landing_page_react_vite_v1.CancelSubscriptionResponse, error) {
	var subscriptionID string
	var status string
	var customerID sql.NullString

	// Find active subscription
	err := s.db.QueryRow(`
		SELECT subscription_id, status, customer_id
		FROM subscriptions
		WHERE (customer_email = $1 OR customer_id = $1)
		AND status IN ('active', 'trialing')
		ORDER BY created_at DESC
		LIMIT 1
	`, userIdentity).Scan(&subscriptionID, &status, &customerID)

	if err == sql.ErrNoRows {
		return nil, errors.New("no active subscription found")
	}

	if err != nil {
		return nil, err
	}

	_, stripeErr := s.doStripeForm(context.Background(), http.MethodPost, "/v1/subscriptions/"+url.PathEscape(subscriptionID), url.Values{
		"cancel_at_period_end": {"true"},
	})
	if stripeErr != nil {
		logStructured("failed to cancel subscription on stripe", map[string]interface{}{
			"level": "warn",
			"id":    subscriptionID,
			"user":  userIdentity,
			"error": stripeErr.Error(),
		})
	}

	now := time.Now()
	_, err = s.db.Exec(`
		UPDATE subscriptions
		SET status = $1, canceled_at = $2, updated_at = $3
		WHERE subscription_id = $4
	`, "canceled", now, now, subscriptionID)
	if err != nil {
		return nil, err
	}

	return &landing_page_react_vite_v1.CancelSubscriptionResponse{
		SubscriptionId: proto.String(subscriptionID),
		State:          mapSubscriptionState("canceled"),
		CanceledAt:     timestamppb.New(now),
		Message:        proto.String("Subscription canceled successfully"),
	}, nil
}

func (s *StripeService) CreateBillingPortalSession(ctx context.Context, userIdentity string, returnURL string) (*landing_page_react_vite_v1.BillingPortalResponse, error) {
	user := strings.TrimSpace(userIdentity)
	if user == "" {
		return nil, errors.New("user identity is required")
	}

	customerID := s.lookupCustomerID(user)
	if customerID == "" {
		if strings.Contains(user, "@") {
			customer, err := s.findCustomerByEmail(ctx, user)
			if err != nil {
				return nil, err
			}
			if customer != nil {
				customerID = customer.ID
			}
		} else {
			customerID = user
		}
	}

	if customerID == "" {
		return nil, errors.New("no Stripe customer found for user")
	}

	values := url.Values{}
	values.Set("customer", customerID)
	if strings.TrimSpace(returnURL) != "" {
		values.Set("return_url", returnURL)
	}

	body, err := s.doStripeForm(ctx, http.MethodPost, "/v1/billing_portal/sessions", values)
	if err != nil {
		return nil, err
	}
	var resp struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.URL == "" {
		return nil, errors.New("Stripe portal URL not returned")
	}

	return &landing_page_react_vite_v1.BillingPortalResponse{Url: resp.URL}, nil
}

// linkUserToStripeCustomer creates or updates a user record with the Stripe customer ID.
// This is called when a checkout completes to link the user account to their Stripe customer.
func (s *StripeService) linkUserToStripeCustomer(email, customerID string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	customerID = strings.TrimSpace(customerID)

	if email == "" || customerID == "" {
		return errors.New("email and customer ID are required")
	}

	// Upsert user with stripe_customer_id
	_, err := s.db.Exec(`
		INSERT INTO users (email, stripe_customer_id, email_verified)
		VALUES ($1, $2, FALSE)
		ON CONFLICT (email) DO UPDATE SET
			stripe_customer_id = $2,
			updated_at = NOW()
	`, email, customerID)
	if err != nil {
		return fmt.Errorf("link user to stripe customer: %w", err)
	}

	logStructured("stripe_customer_linked", map[string]interface{}{
		"level":       "info",
		"email":       email,
		"customer_id": customerID,
	})

	return nil
}

// checkIntroEligibility checks if a user is eligible for the intro pricing coupon.
// Returns true if the user has never used the intro offer before.
func (s *StripeService) checkIntroEligibility(ctx context.Context, email string) (bool, error) {
	email = NormalizeEmail(email)
	if email == "" {
		// No email means we can't track eligibility, so don't apply intro
		return false, nil
	}

	// Check if user exists and has already used intro
	var hasUsedIntro sql.NullBool
	err := s.db.QueryRowContext(ctx, `
		SELECT has_used_intro FROM users WHERE email = $1
	`, email).Scan(&hasUsedIntro)

	if err == sql.ErrNoRows {
		// New user - eligible for intro
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("check intro eligibility: %w", err)
	}

	// User exists - check if they've used intro
	if hasUsedIntro.Valid && hasUsedIntro.Bool {
		return false, nil // Already used intro
	}

	return true, nil // Eligible
}

// markIntroUsed marks that a user has used their intro pricing offer.
// This should be called after successful payment with an intro coupon.
func (s *StripeService) markIntroUsed(ctx context.Context, email, customerID, couponID, planTier, subscriptionID string) error {
	email = NormalizeEmail(email)
	if email == "" {
		return errors.New("email required to mark intro used")
	}

	// Start a transaction to ensure consistency
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Update the user's has_used_intro flag (upsert)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO users (email, has_used_intro, stripe_customer_id)
		VALUES ($1, TRUE, $2)
		ON CONFLICT (email) DO UPDATE SET
			has_used_intro = TRUE,
			stripe_customer_id = COALESCE(NULLIF($2, ''), users.stripe_customer_id),
			updated_at = NOW()
	`, email, customerID)
	if err != nil {
		return fmt.Errorf("update user intro flag: %w", err)
	}

	// Record in audit table
	_, err = tx.ExecContext(ctx, `
		INSERT INTO intro_coupon_usage (email, stripe_customer_id, coupon_id, plan_tier, subscription_id)
		VALUES ($1, $2, $3, $4, $5)
	`, email, customerID, couponID, planTier, subscriptionID)
	if err != nil {
		return fmt.Errorf("insert intro coupon usage: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	// Also update Stripe customer metadata as backup
	if customerID != "" {
		go func() {
			values := url.Values{}
			values.Set("metadata[has_used_intro]", "true")
			values.Set("metadata[intro_coupon_id]", couponID)
			_, updateErr := s.doStripeForm(context.Background(), http.MethodPost, "/v1/customers/"+url.PathEscape(customerID), values)
			if updateErr != nil {
				logStructuredError("stripe_customer_metadata_update_failed", map[string]interface{}{
					"customer_id": customerID,
					"error":       updateErr.Error(),
				})
			}
		}()
	}

	logStructured("intro_coupon_marked_used", map[string]interface{}{
		"level":           "info",
		"email":           email,
		"customer_id":     customerID,
		"coupon_id":       couponID,
		"plan_tier":       planTier,
		"subscription_id": subscriptionID,
	})

	return nil
}

// StripeImportPreview provides a preview of products/prices available for import from Stripe.
type StripeImportPreview struct {
	BundleKey          string                    `json:"bundle_key,omitempty"`
	BundleProductID    string                    `json:"bundle_product_id,omitempty"`
	BundleProductFound bool                      `json:"bundle_product_found"`
	BundlePlanCount    int                       `json:"bundle_plan_count"`
	Products           []StripeProductWithPrices `json:"products"`
	TotalPrices        int                       `json:"total_prices"`
	ConflictCount      int                       `json:"conflict_count"`
	NewCount           int                       `json:"new_count"`
}

// StripeProductWithPrices groups a Stripe product with its prices.
type StripeProductWithPrices struct {
	ProductID       string              `json:"product_id"`
	ProductName     string              `json:"product_name"`
	IsCurrentBundle bool                `json:"is_current_bundle"`
	Prices          []StripePriceImport `json:"prices"`
}

// StripePriceImport represents a price from Stripe that can be imported.
type StripePriceImport struct {
	PriceID       string `json:"price_id"`
	LookupKey     string `json:"lookup_key,omitempty"`
	Currency      string `json:"currency"`
	AmountCents   int64  `json:"amount_cents"`
	Interval      string `json:"interval,omitempty"`
	ProductID     string `json:"product_id"`
	ProductName   string `json:"product_name"`
	Active        bool   `json:"active"`
	ExistsLocally bool   `json:"exists_locally"`
}

// ListStripeProductsWithPrices fetches all products and prices from Stripe for import preview.
func (s *StripeService) ListStripeProductsWithPrices(ctx context.Context, planStore *PlanStore) (*StripeImportPreview, error) {
	// Fetch all active products from Stripe
	products, err := s.fetchStripeProducts(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch stripe products: %w", err)
	}

	bundleKey := ""
	bundleProductID := ""
	bundlePlanCount := 0
	bundleProductFound := false
	// Get existing price IDs from plan store
	existingPriceIDs := make(map[string]bool)
	if planStore != nil {
		if bundle := planStore.GetBundle(); bundle != nil {
			bundleKey = bundle.BundleKey
			bundleProductID = strings.TrimSpace(bundle.StripeProductId)
		}
		plans := planStore.GetPlans()
		bundlePlanCount = len(plans)
		for _, plan := range plans {
			if plan.StripePriceId != "" {
				existingPriceIDs[plan.StripePriceId] = true
			}
		}
	}

	preview := &StripeImportPreview{
		BundleKey:          bundleKey,
		BundleProductID:    bundleProductID,
		BundleProductFound: false,
		BundlePlanCount:    bundlePlanCount,
		Products:           make([]StripeProductWithPrices, 0, len(products)),
	}

	for _, product := range products {
		isCurrentBundle := bundleProductID != "" && product.ID == bundleProductID
		if isCurrentBundle {
			bundleProductFound = true
		}
		// Fetch prices for this product
		prices, err := s.fetchStripePricesForProduct(ctx, product.ID)
		if err != nil {
			logStructuredError("stripe_price_fetch_failed", map[string]interface{}{
				"product_id": product.ID,
				"error":      err.Error(),
			})
			continue
		}

		if len(prices) == 0 {
			continue
		}

		productWithPrices := StripeProductWithPrices{
			ProductID:       product.ID,
			ProductName:     product.Name,
			IsCurrentBundle: isCurrentBundle,
			Prices:          make([]StripePriceImport, 0, len(prices)),
		}

		for _, price := range prices {
			existsLocally := existingPriceIDs[price.ID]
			priceImport := StripePriceImport{
				PriceID:       price.ID,
				LookupKey:     price.LookupKey,
				Currency:      price.Currency,
				AmountCents:   price.UnitAmount,
				Interval:      price.Interval,
				ProductID:     product.ID,
				ProductName:   product.Name,
				Active:        price.Active,
				ExistsLocally: existsLocally,
			}
			productWithPrices.Prices = append(productWithPrices.Prices, priceImport)

			preview.TotalPrices++
			if existsLocally {
				preview.ConflictCount++
			} else {
				preview.NewCount++
			}
		}

		preview.Products = append(preview.Products, productWithPrices)
	}

	preview.BundleProductFound = bundleProductFound
	return preview, nil
}

type stripeProduct struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type stripePrice struct {
	ID         string `json:"id"`
	LookupKey  string `json:"lookup_key"`
	Currency   string `json:"currency"`
	UnitAmount int64  `json:"unit_amount"`
	Active     bool   `json:"active"`
	Interval   string
	ProductID  string `json:"product"`
}

func (s *StripeService) fetchStripeProducts(ctx context.Context) ([]stripeProduct, error) {
	values := url.Values{}
	values.Set("active", "true")
	values.Set("limit", "100")
	path := "/v1/products?" + values.Encode()

	body, err := s.doStripeRequest(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data []stripeProduct `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode stripe products: %w", err)
	}

	return resp.Data, nil
}

func (s *StripeService) fetchStripePricesForProduct(ctx context.Context, productID string) ([]stripePrice, error) {
	values := url.Values{}
	values.Set("product", productID)
	values.Set("limit", "100")
	path := "/v1/prices?" + values.Encode()

	body, err := s.doStripeRequest(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data []struct {
			ID         string `json:"id"`
			LookupKey  string `json:"lookup_key"`
			Currency   string `json:"currency"`
			UnitAmount int64  `json:"unit_amount"`
			Active     bool   `json:"active"`
			Recurring  *struct {
				Interval string `json:"interval"`
			} `json:"recurring"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode stripe prices: %w", err)
	}

	prices := make([]stripePrice, 0, len(resp.Data))
	for _, p := range resp.Data {
		price := stripePrice{
			ID:         p.ID,
			LookupKey:  p.LookupKey,
			Currency:   p.Currency,
			UnitAmount: p.UnitAmount,
			Active:     p.Active,
			ProductID:  productID,
		}
		if p.Recurring != nil {
			price.Interval = p.Recurring.Interval
		} else {
			price.Interval = "one_time"
		}
		prices = append(prices, price)
	}

	return prices, nil
}

// StripeCoupon represents a Stripe coupon for admin management.
type StripeCoupon struct {
	ID               string   `json:"id"`
	Name             string   `json:"name,omitempty"`
	AmountOff        *int64   `json:"amount_off,omitempty"`
	PercentOff       *float64 `json:"percent_off,omitempty"`
	Currency         string   `json:"currency,omitempty"`
	Duration         string   `json:"duration"`
	DurationInMonths *int     `json:"duration_in_months,omitempty"`
	MaxRedemptions   *int     `json:"max_redemptions,omitempty"`
	RedeemBy         *int64   `json:"redeem_by,omitempty"`
	TimesRedeemed    int      `json:"times_redeemed"`
	Valid            bool     `json:"valid"`
	Created          int64    `json:"created"`
	IsIntroCoupon    bool     `json:"is_intro_coupon"`
	IntroTier        string   `json:"intro_tier,omitempty"`
}

// CreateCouponRequest contains parameters for creating a new coupon.
type CreateCouponRequest struct {
	ID               string   `json:"id,omitempty"`
	Name             string   `json:"name,omitempty"`
	AmountOff        *int64   `json:"amount_off,omitempty"`
	PercentOff       *float64 `json:"percent_off,omitempty"`
	Currency         string   `json:"currency,omitempty"`
	Duration         string   `json:"duration"`
	DurationInMonths *int     `json:"duration_in_months,omitempty"`
	MaxRedemptions   *int     `json:"max_redemptions,omitempty"`
	RedeemBy         *int64   `json:"redeem_by,omitempty"`
}

// ListCoupons fetches all coupons from Stripe.
func (s *StripeService) ListCoupons(ctx context.Context) ([]StripeCoupon, error) {
	values := url.Values{}
	values.Set("limit", "100")
	path := "/v1/coupons?" + values.Encode()

	body, err := s.doStripeRequest(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data []struct {
			ID               string   `json:"id"`
			Name             string   `json:"name"`
			AmountOff        *int64   `json:"amount_off"`
			PercentOff       *float64 `json:"percent_off"`
			Currency         string   `json:"currency"`
			Duration         string   `json:"duration"`
			DurationInMonths *int     `json:"duration_in_months"`
			MaxRedemptions   *int     `json:"max_redemptions"`
			RedeemBy         *int64   `json:"redeem_by"`
			TimesRedeemed    int      `json:"times_redeemed"`
			Valid            bool     `json:"valid"`
			Created          int64    `json:"created"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode stripe coupons: %w", err)
	}

	coupons := make([]StripeCoupon, 0, len(resp.Data))
	for _, c := range resp.Data {
		coupon := StripeCoupon{
			ID:               c.ID,
			Name:             c.Name,
			AmountOff:        c.AmountOff,
			PercentOff:       c.PercentOff,
			Currency:         c.Currency,
			Duration:         c.Duration,
			DurationInMonths: c.DurationInMonths,
			MaxRedemptions:   c.MaxRedemptions,
			RedeemBy:         c.RedeemBy,
			TimesRedeemed:    c.TimesRedeemed,
			Valid:            c.Valid,
			Created:          c.Created,
		}
		// Check if this coupon is configured for intro pricing
		coupon.IsIntroCoupon, coupon.IntroTier = s.checkIntroCouponMapping(c.ID)
		coupons = append(coupons, coupon)
	}

	return coupons, nil
}

// GetCoupon fetches a single coupon from Stripe by ID.
func (s *StripeService) GetCoupon(ctx context.Context, couponID string) (*StripeCoupon, error) {
	if strings.TrimSpace(couponID) == "" {
		return nil, errors.New("coupon ID is required")
	}

	path := "/v1/coupons/" + url.PathEscape(couponID)
	body, err := s.doStripeRequest(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}

	var c struct {
		ID               string   `json:"id"`
		Name             string   `json:"name"`
		AmountOff        *int64   `json:"amount_off"`
		PercentOff       *float64 `json:"percent_off"`
		Currency         string   `json:"currency"`
		Duration         string   `json:"duration"`
		DurationInMonths *int     `json:"duration_in_months"`
		MaxRedemptions   *int     `json:"max_redemptions"`
		RedeemBy         *int64   `json:"redeem_by"`
		TimesRedeemed    int      `json:"times_redeemed"`
		Valid            bool     `json:"valid"`
		Created          int64    `json:"created"`
	}
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, fmt.Errorf("decode stripe coupon: %w", err)
	}

	coupon := &StripeCoupon{
		ID:               c.ID,
		Name:             c.Name,
		AmountOff:        c.AmountOff,
		PercentOff:       c.PercentOff,
		Currency:         c.Currency,
		Duration:         c.Duration,
		DurationInMonths: c.DurationInMonths,
		MaxRedemptions:   c.MaxRedemptions,
		RedeemBy:         c.RedeemBy,
		TimesRedeemed:    c.TimesRedeemed,
		Valid:            c.Valid,
		Created:          c.Created,
	}
	coupon.IsIntroCoupon, coupon.IntroTier = s.checkIntroCouponMapping(c.ID)

	return coupon, nil
}

// CreateCoupon creates a new coupon in Stripe.
func (s *StripeService) CreateCoupon(ctx context.Context, req CreateCouponRequest) (*StripeCoupon, error) {
	// Validate discount type
	if req.AmountOff == nil && req.PercentOff == nil {
		return nil, errors.New("either amount_off or percent_off is required")
	}
	if req.AmountOff != nil && req.PercentOff != nil {
		return nil, errors.New("cannot specify both amount_off and percent_off")
	}

	// Validate duration
	duration := strings.TrimSpace(req.Duration)
	if duration == "" {
		return nil, errors.New("duration is required")
	}
	if duration != "once" && duration != "repeating" && duration != "forever" {
		return nil, errors.New("duration must be once, repeating, or forever")
	}
	if duration == "repeating" && (req.DurationInMonths == nil || *req.DurationInMonths <= 0) {
		return nil, errors.New("duration_in_months is required for repeating coupons")
	}

	// Build form values
	values := url.Values{}
	values.Set("duration", duration)

	if id := strings.TrimSpace(req.ID); id != "" {
		values.Set("id", id)
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		values.Set("name", name)
	}
	if req.AmountOff != nil {
		values.Set("amount_off", strconv.FormatInt(*req.AmountOff, 10))
		currency := strings.ToLower(strings.TrimSpace(req.Currency))
		if currency == "" {
			currency = "usd"
		}
		values.Set("currency", currency)
	}
	if req.PercentOff != nil {
		values.Set("percent_off", strconv.FormatFloat(*req.PercentOff, 'f', 2, 64))
	}
	if req.DurationInMonths != nil && duration == "repeating" {
		values.Set("duration_in_months", strconv.Itoa(*req.DurationInMonths))
	}
	if req.MaxRedemptions != nil && *req.MaxRedemptions > 0 {
		values.Set("max_redemptions", strconv.Itoa(*req.MaxRedemptions))
	}
	if req.RedeemBy != nil && *req.RedeemBy > 0 {
		values.Set("redeem_by", strconv.FormatInt(*req.RedeemBy, 10))
	}

	body, err := s.doStripeForm(ctx, http.MethodPost, "/v1/coupons", values)
	if err != nil {
		return nil, err
	}

	var c struct {
		ID               string   `json:"id"`
		Name             string   `json:"name"`
		AmountOff        *int64   `json:"amount_off"`
		PercentOff       *float64 `json:"percent_off"`
		Currency         string   `json:"currency"`
		Duration         string   `json:"duration"`
		DurationInMonths *int     `json:"duration_in_months"`
		MaxRedemptions   *int     `json:"max_redemptions"`
		RedeemBy         *int64   `json:"redeem_by"`
		TimesRedeemed    int      `json:"times_redeemed"`
		Valid            bool     `json:"valid"`
		Created          int64    `json:"created"`
	}
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, fmt.Errorf("decode stripe coupon: %w", err)
	}

	coupon := &StripeCoupon{
		ID:               c.ID,
		Name:             c.Name,
		AmountOff:        c.AmountOff,
		PercentOff:       c.PercentOff,
		Currency:         c.Currency,
		Duration:         c.Duration,
		DurationInMonths: c.DurationInMonths,
		MaxRedemptions:   c.MaxRedemptions,
		RedeemBy:         c.RedeemBy,
		TimesRedeemed:    c.TimesRedeemed,
		Valid:            c.Valid,
		Created:          c.Created,
	}
	coupon.IsIntroCoupon, coupon.IntroTier = s.checkIntroCouponMapping(c.ID)

	return coupon, nil
}

// DeleteCoupon deletes a coupon from Stripe.
func (s *StripeService) DeleteCoupon(ctx context.Context, couponID string) error {
	if strings.TrimSpace(couponID) == "" {
		return errors.New("coupon ID is required")
	}

	path := "/v1/coupons/" + url.PathEscape(couponID)
	_, err := s.doStripeRequest(ctx, http.MethodDelete, path, nil, "")
	return err
}

// checkIntroCouponMapping checks if a coupon ID matches a configured intro coupon.
// Returns (isIntro, tier) where tier is the plan tier name if matched.
func (s *StripeService) checkIntroCouponMapping(couponID string) (bool, string) {
	if !s.introCouponConfig.Enabled || s.introCouponConfig.CouponMap == nil {
		return false, ""
	}
	for tier, configuredID := range s.introCouponConfig.CouponMap {
		if configuredID == couponID {
			return true, tier
		}
	}
	return false, ""
}

// GetIntroCouponMap returns the configured intro coupon mapping.
func (s *StripeService) GetIntroCouponMap() map[string]string {
	if !s.introCouponConfig.Enabled || s.introCouponConfig.CouponMap == nil {
		return nil
	}
	// Return a copy to prevent external modification
	result := make(map[string]string, len(s.introCouponConfig.CouponMap))
	for k, v := range s.introCouponConfig.CouponMap {
		result[k] = v
	}
	return result
}

// FetchStripePriceDetails fetches full details for a single price from Stripe.
func (s *StripeService) FetchStripePriceDetails(ctx context.Context, priceID string) (*StripePriceImport, error) {
	path := "/v1/prices/" + url.PathEscape(priceID) + "?expand[]=product"
	body, err := s.doStripeRequest(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}

	var price struct {
		ID         string `json:"id"`
		LookupKey  string `json:"lookup_key"`
		Currency   string `json:"currency"`
		UnitAmount int64  `json:"unit_amount"`
		Active     bool   `json:"active"`
		Recurring  *struct {
			Interval string `json:"interval"`
		} `json:"recurring"`
		Product struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"product"`
	}
	if err := json.Unmarshal(body, &price); err != nil {
		return nil, fmt.Errorf("decode stripe price: %w", err)
	}

	interval := "one_time"
	if price.Recurring != nil {
		interval = price.Recurring.Interval
	}

	return &StripePriceImport{
		PriceID:     price.ID,
		LookupKey:   price.LookupKey,
		Currency:    price.Currency,
		AmountCents: price.UnitAmount,
		Interval:    interval,
		ProductID:   price.Product.ID,
		ProductName: price.Product.Name,
		Active:      price.Active,
	}, nil
}
