package billing

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/commerce"
)

// UsageDependencies are API composition concerns used by the usage transport.
// Metering policy remains owned by internal/commerce.
type UsageDependencies struct {
	UserEmail  func(context.Context) string
	WriteError func(http.ResponseWriter, int, string, string)
	LogError   func(string, map[string]any)
}

func RequireUsageServiceAuth(svc *commerce.UsageService, deps UsageDependencies, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			deps.WriteError(w, http.StatusUnauthorized, "Missing or invalid authorization header", "unauthorized")
			return
		}
		if !svc.ValidateServiceToken(strings.TrimPrefix(auth, "Bearer ")) {
			deps.WriteError(w, http.StatusUnauthorized, "Invalid service token", "unauthorized")
			return
		}
		next(w, r)
	}
}

func ReportUsage(svc *commerce.UsageService, deps UsageDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req commerce.UsageReportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}
		if err := svc.RecordUsage(r.Context(), req); err != nil {
			deps.LogError("report_usage_failed", map[string]any{"error": err.Error(), "user_identity": req.UserIdentity, "limit_key": req.LimitKey})
			deps.WriteError(w, http.StatusBadRequest, err.Error(), "validation")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"success": true}); err != nil {
			deps.LogError("encode_response_failed", map[string]any{"error": err.Error()})
		}
	}
}

func GetUsageSummary(svc *commerce.UsageService, accountSvc *commerce.Service, deps UsageDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIdentity := deps.UserEmail(r.Context())
		if userIdentity == "" {
			deps.WriteError(w, http.StatusUnauthorized, "Authentication required", "unauthorized")
			return
		}
		tier := r.URL.Query().Get("tier")
		if tier == "" && accountSvc != nil {
			if sub, err := accountSvc.GetSubscriptionContext(r.Context(), userIdentity); err == nil && sub != nil && sub.PlanTier != nil {
				tier = *sub.PlanTier
			}
		}
		summary, err := svc.GetUsageSummary(r.Context(), userIdentity, tier)
		if err != nil {
			deps.LogError("get_usage_summary_failed", map[string]any{"error": err.Error(), "user_identity": userIdentity})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to get usage summary", "server_error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(summary); err != nil {
			deps.LogError("encode_response_failed", map[string]any{"error": err.Error()})
		}
	}
}

func AdminUsageSummary(svc *commerce.UsageService, deps UsageDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		billingPeriod := r.URL.Query().Get("period")
		records, err := svc.GetAllUsageForPeriod(r.Context(), billingPeriod)
		if err != nil {
			deps.LogError("admin_usage_summary_failed", map[string]any{"error": err.Error(), "billing_period": billingPeriod})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to get usage summary", "server_error")
			return
		}
		if records == nil {
			records = []commerce.UsageRecord{}
		}
		userTotals, appTotals := make(map[string]int64), make(map[string]int64)
		for _, record := range records {
			userTotals[record.UserIdentity] += record.UsageAmount
			if record.AppBundleKey != nil {
				appTotals[*record.AppBundleKey] += record.UsageAmount
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"billing_period": billingPeriod, "records": records, "user_totals": userTotals, "app_totals": appTotals, "total_users": len(userTotals), "total_records": len(records)}); err != nil {
			deps.LogError("encode_response_failed", map[string]any{"error": err.Error()})
		}
	}
}

func CheckLimit(svc *commerce.UsageService, deps UsageDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIdentity := deps.UserEmail(r.Context())
		if userIdentity == "" {
			deps.WriteError(w, http.StatusUnauthorized, "Authentication required", "unauthorized")
			return
		}
		tier, limitKey := r.URL.Query().Get("tier"), r.URL.Query().Get("limit_key")
		if limitKey == "" {
			deps.WriteError(w, http.StatusBadRequest, "limit_key is required", "validation")
			return
		}
		canProceed, remaining, err := svc.CheckLimit(r.Context(), userIdentity, tier, limitKey, 1)
		if err != nil {
			deps.LogError("check_limit_failed", map[string]any{"error": err.Error(), "user_identity": userIdentity, "limit_key": limitKey})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to check limit", "server_error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"can_proceed": canProceed, "remaining": remaining, "limit_key": limitKey}); err != nil {
			deps.LogError("encode_response_failed", map[string]any{"error": err.Error()})
		}
	}
}

// UsageHealth is an unauthenticated monitoring endpoint. Its timeout remains
// request-derived so cancellation propagates to the usage store.
func UsageHealth(svc *commerce.UsageService, deps UsageDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		status, err := svc.HealthCheck(ctx)
		if err != nil {
			deps.LogError("usage_health_check_failed", map[string]any{"error": err.Error()})
			w.WriteHeader(http.StatusServiceUnavailable)
			if encodeErr := json.NewEncoder(w).Encode(map[string]any{"healthy": false, "error": err.Error()}); encodeErr != nil {
				deps.LogError("encode_response_failed", map[string]any{"error": encodeErr.Error()})
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if !status.Healthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		if err := json.NewEncoder(w).Encode(status); err != nil {
			deps.LogError("encode_response_failed", map[string]any{"error": err.Error()})
		}
	}
}
