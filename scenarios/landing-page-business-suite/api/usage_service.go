package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/commerce"
)

type (
	UsageStore          = commerce.UsageStore
	UsageService        = commerce.UsageService
	UsageServiceOptions = commerce.UsageServiceOptions
	UsageRecord         = commerce.UsageRecord
	UsageReportRequest  = commerce.UsageReportRequest
	UsageSummary        = commerce.UsageSummary
	UsageHealthStatus   = commerce.UsageHealthStatus
)

func NewUsageService(db UsageStore, limitsSvc LimitsServicer, dialect string) *UsageService {
	return NewUsageServiceWithOptions(UsageServiceOptions{DB: db, LimitsService: limitsSvc, Dialect: dialect})
}

func NewUsageServiceWithOptions(opts UsageServiceOptions) *UsageService {
	if opts.ServiceToken == "" {
		opts.ServiceToken = resolveSecret("LPBS_SERVICE_SECRET")
		if opts.ServiceToken == "" {
			logStructured("usage_service_no_token", map[string]interface{}{
				"level":   "warn",
				"message": "LPBS_SERVICE_SECRET not set; service-to-service auth disabled",
			})
		}
	}
	if opts.Log == nil {
		opts.Log = logStructured
	}
	if opts.InsufficientCredits == nil {
		opts.InsufficientCredits = ErrInsufficientCredits
	}
	return commerce.NewUsageServiceWithOptions(opts)
}

// API handlers remain at the HTTP edge; metering policy lives in commerce.

// requireUsageServiceAuth is API-edge middleware for service-to-service authentication.
func requireUsageServiceAuth(svc *UsageService, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract Bearer token
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeJSONError(w, http.StatusUnauthorized, "Missing or invalid authorization header", ApiErrorTypeUnauthorized)
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if !svc.ValidateServiceToken(token) {
			writeJSONError(w, http.StatusUnauthorized, "Invalid service token", ApiErrorTypeUnauthorized)
			return
		}

		next(w, r)
	}
}

func handleReportUsage(svc *UsageService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req UsageReportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		if err := svc.RecordUsage(r.Context(), req); err != nil {
			logStructuredError("report_usage_failed", map[string]interface{}{
				"error":         err.Error(),
				"user_identity": req.UserIdentity,
				"limit_key":     req.LimitKey,
			})
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		}); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func handleGetUsageSummary(svc *UsageService, accountSvc *AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIdentity := getUserEmail(r.Context())
		if userIdentity == "" {
			writeJSONError(w, http.StatusUnauthorized, "Authentication required", ApiErrorTypeUnauthorized)
			return
		}
		tier := r.URL.Query().Get("tier")

		// If no tier provided, try to get it from the account service
		if tier == "" && accountSvc != nil {
			// Try to get subscription info to determine tier
			sub, err := accountSvc.GetSubscriptionContext(r.Context(), userIdentity)
			if err == nil && sub != nil && sub.PlanTier != nil {
				tier = *sub.PlanTier
			}
		}

		summary, err := svc.GetUsageSummary(r.Context(), userIdentity, tier)
		if err != nil {
			logStructuredError("get_usage_summary_failed", map[string]interface{}{
				"error":         err.Error(),
				"user_identity": userIdentity,
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to get usage summary", ApiErrorTypeServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(summary); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func handleAdminUsageSummary(svc *UsageService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		billingPeriod := r.URL.Query().Get("period")

		records, err := svc.GetAllUsageForPeriod(r.Context(), billingPeriod)
		if err != nil {
			logStructuredError("admin_usage_summary_failed", map[string]interface{}{
				"error":          err.Error(),
				"billing_period": billingPeriod,
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to get usage summary", ApiErrorTypeServerError)
			return
		}

		if records == nil {
			records = []UsageRecord{}
		}

		// Aggregate by user
		userTotals := make(map[string]int64)
		appTotals := make(map[string]int64)
		for _, record := range records {
			userTotals[record.UserIdentity] += record.UsageAmount
			if record.AppBundleKey != nil {
				appTotals[*record.AppBundleKey] += record.UsageAmount
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"billing_period": billingPeriod,
			"records":        records,
			"user_totals":    userTotals,
			"app_totals":     appTotals,
			"total_users":    len(userTotals),
			"total_records":  len(records),
		}); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleCheckLimit checks if a user can perform an operation (for entitlements endpoint).
func handleCheckLimit(svc *UsageService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIdentity := getUserEmail(r.Context())
		if userIdentity == "" {
			writeJSONError(w, http.StatusUnauthorized, "Authentication required", ApiErrorTypeUnauthorized)
			return
		}
		tier := r.URL.Query().Get("tier")
		limitKey := r.URL.Query().Get("limit_key")

		if limitKey == "" {
			writeJSONError(w, http.StatusBadRequest, "limit_key is required", ApiErrorTypeValidation)
			return
		}

		// Default check for 1 unit
		amount := int64(1)

		canProceed, remaining, err := svc.CheckLimit(r.Context(), userIdentity, tier, limitKey, amount)
		if err != nil {
			logStructuredError("check_limit_failed", map[string]interface{}{
				"error":         err.Error(),
				"user_identity": userIdentity,
				"limit_key":     limitKey,
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to check limit", ApiErrorTypeServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"can_proceed": canProceed,
			"remaining":   remaining,
			"limit_key":   limitKey,
		}); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleUsageHealth returns the health status of the usage service.
// This endpoint is unauthenticated for monitoring/observability purposes.
func handleUsageHealth(svc *UsageService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		status, err := svc.HealthCheck(ctx)
		if err != nil {
			logStructuredError("usage_health_check_failed", map[string]interface{}{
				"error": err.Error(),
			})
			w.WriteHeader(http.StatusServiceUnavailable)
			if encErr := json.NewEncoder(w).Encode(map[string]interface{}{
				"healthy": false,
				"error":   err.Error(),
			}); encErr != nil {
				logStructuredError("encode_response_failed", map[string]interface{}{"error": encErr.Error()})
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if !status.Healthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		if err := json.NewEncoder(w).Encode(status); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}
