package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
)

// Fixture commands are deliberately HTTP-visible only on a local development
// authority. They seed the real commerce tables so offline validation exercises
// the same subscription and wallet reads as production.
type fixtureSeedRequest struct {
	Email         string `json:"email"`
	Tier          string `json:"tier"`
	CreditBalance int64  `json:"credit_balance"`
	BundleKey     string `json:"bundle_key"`
}

type fixtureEmailRequest struct {
	Email string `json:"email"`
}

func registerFixtureRoutes(s *Server) {
	s.router.HandleFunc("/api/v1/dev/fixtures/seed", fixtureOnly(s, s.fixtureSeed)).Methods(http.MethodPost)
	s.router.HandleFunc("/api/v1/dev/fixtures/token", fixtureOnly(s, s.fixtureToken)).Methods(http.MethodPost)
	s.router.HandleFunc("/api/v1/dev/fixtures/balance", fixtureOnly(s, s.fixtureBalance)).Methods(http.MethodGet)
	s.router.HandleFunc("/api/v1/dev/fixtures/zero", fixtureOnly(s, s.fixtureZero)).Methods(http.MethodPost)
}

func fixtureOnly(s *Server, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !fixtureRequestAllowed(r) {
			writeJSONError(w, http.StatusForbidden, "fixture commands are restricted to a local development authority", ApiErrorTypeForbidden)
			return
		}
		next(w, r)
	}
}

func fixtureRequestAllowed(r *http.Request) bool {
	if isProductionEnvironment() || strings.EqualFold(strings.TrimSpace(resolveConfig("LPBS_FIXTURE_MODE")), "false") {
		return false
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.Host))
	if err != nil {
		host = strings.TrimSpace(r.Host)
	}
	host = strings.Trim(host, "[]")
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func (s *Server) fixtureSeed(w http.ResponseWriter, r *http.Request) {
	var request fixtureSeedRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid fixture seed payload", ApiErrorTypeValidation)
		return
	}
	request.Email = NormalizeEmail(request.Email)
	request.Tier = strings.ToLower(strings.TrimSpace(request.Tier))
	request.BundleKey = strings.TrimSpace(request.BundleKey)
	if request.Email == "" || request.Tier == "" || request.BundleKey == "" || request.CreditBalance < 0 {
		writeJSONError(w, http.StatusBadRequest, "email, tier, bundle_key, and non-negative credit_balance are required", ApiErrorTypeValidation)
		return
	}
	if !fixtureTier(request.Tier) {
		writeJSONError(w, http.StatusBadRequest, "unsupported fixture tier", ApiErrorTypeValidation)
		return
	}

	ctx := r.Context()
	tx, err := s.primaryDB().BeginTx(ctx, nil)
	if err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, "fixture database unavailable", ApiErrorTypeServerError)
		return
	}
	defer func() { _ = tx.Rollback() }()
	var userID string
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO users (email, email_verified, updated_at, last_login_at)
		VALUES ($1, TRUE, NOW(), NOW())
		ON CONFLICT (email) DO UPDATE SET email_verified = TRUE, updated_at = NOW()
		RETURNING id`, request.Email).Scan(&userID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "seed fixture user failed", ApiErrorTypeServerError)
		return
	}
	fixtureID := fixtureSubscriptionID(request.Email)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, price_id, bundle_key, created_at, updated_at)
		VALUES ($1, $2, $3, 'active', $4, $5, $6, NOW(), NOW())
		ON CONFLICT (subscription_id) DO UPDATE SET customer_email = EXCLUDED.customer_email, status = 'active', plan_tier = EXCLUDED.plan_tier, bundle_key = EXCLUDED.bundle_key, updated_at = NOW()`,
		fixtureID, userID, request.Email, request.Tier, "fixture_"+request.Tier, request.BundleKey); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "seed fixture subscription failed", ApiErrorTypeServerError)
		return
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO credit_wallets (customer_email, balance_credits, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (customer_email) DO UPDATE SET balance_credits = EXCLUDED.balance_credits, updated_at = NOW()`, request.Email, request.CreditBalance); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "seed fixture wallet failed", ApiErrorTypeServerError)
		return
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO subscription_tier_limits (tier_id, limit_type, limit_key, limit_value, app_bundle_key, reset_period)
		VALUES ($1, 'cost_based', 'ai_credits', $2, $3, 'monthly')
		ON CONFLICT (tier_id, limit_type, limit_key, app_bundle_key) DO UPDATE SET limit_value = EXCLUDED.limit_value, updated_at = NOW()`,
		request.Tier, request.CreditBalance, request.BundleKey); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "seed fixture tier limit failed", ApiErrorTypeServerError)
		return
	}
	leaseBundleKey := strings.TrimSpace(resolveConfig("BUNDLE_KEY"))
	if leaseBundleKey == "" {
		leaseBundleKey = "business_suite"
	}
	workflowLimit := int64(0)
	if request.Tier != "free" {
		workflowLimit = 100
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO subscription_tier_limits (tier_id, limit_type, limit_key, limit_value, app_bundle_key, reset_period)
		VALUES ($1, 'count_based', 'workflow_executions', $2, $3, 'monthly')
		ON CONFLICT (tier_id, limit_type, limit_key, app_bundle_key) DO UPDATE SET limit_value = EXCLUDED.limit_value, updated_at = NOW()`,
		request.Tier, workflowLimit, leaseBundleKey); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "seed fixture workflow limit failed", ApiErrorTypeServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "commit fixture seed failed", ApiErrorTypeServerError)
		return
	}
	writeJSON(w, map[string]any{"email": request.Email, "tier": request.Tier, "credit_balance": request.CreditBalance, "bundle_key": request.BundleKey, "subscription_id": fixtureID, "idempotent": true})
}

func (s *Server) fixtureToken(w http.ResponseWriter, r *http.Request) {
	var request fixtureEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid fixture token payload", ApiErrorTypeValidation)
		return
	}
	user, err := s.userAuthService.GetUserByEmail(r.Context(), NormalizeEmail(request.Email))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "fixture account not found", ApiErrorTypeNotFound)
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "load fixture account failed", ApiErrorTypeServerError)
		return
	}
	tokenPair, err := s.userAuthService.CreateSession(r.Context(), user, "127.0.0.1", "vrooli-local-fixture")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "mint fixture token failed", ApiErrorTypeServerError)
		return
	}
	writeJSON(w, map[string]any{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"token_type":    tokenPair.TokenType,
		"expires_at":    tokenPair.ExpiresAt.UTC(),
		"email":         user.Email,
	})
}

func (s *Server) fixtureBalance(w http.ResponseWriter, r *http.Request) {
	email := NormalizeEmail(r.URL.Query().Get("email"))
	if email == "" {
		writeJSONError(w, http.StatusBadRequest, "email is required", ApiErrorTypeValidation)
		return
	}
	var balance int64
	if err := s.primaryDB().QueryRowContext(r.Context(), `SELECT COALESCE(balance_credits, 0) FROM credit_wallets WHERE customer_email = $1`, email).Scan(&balance); err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusInternalServerError, "read fixture balance failed", ApiErrorTypeServerError)
		return
	}
	writeJSON(w, map[string]any{"email": email, "credit_balance": balance})
}

func (s *Server) fixtureZero(w http.ResponseWriter, r *http.Request) {
	var request fixtureEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid fixture zero payload", ApiErrorTypeValidation)
		return
	}
	email := NormalizeEmail(request.Email)
	if email == "" {
		writeJSONError(w, http.StatusBadRequest, "email is required", ApiErrorTypeValidation)
		return
	}
	if _, err := s.primaryDB().ExecContext(r.Context(), `UPDATE credit_wallets SET balance_credits = 0, updated_at = NOW() WHERE customer_email = $1`, email); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "zero fixture balance failed", ApiErrorTypeServerError)
		return
	}
	writeJSON(w, map[string]any{"email": email, "credit_balance": 0})
}

func fixtureTier(tier string) bool {
	switch tier {
	case "free", "solo", "pro", "studio", "business":
		return true
	default:
		return false
	}
}

func fixtureSubscriptionID(email string) string {
	digest := sha256.Sum256([]byte(NormalizeEmail(email)))
	return "fixture_" + hex.EncodeToString(digest[:16])
}
