package billing

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"landing-page-business-suite-api/internal/commerce"
)

// UsageDependencies are API composition concerns used by the usage transport.
// Metering policy remains owned by internal/commerce.
type UsageDependencies struct {
	UserEmail              func(context.Context) string
	UserID                 func(context.Context) string
	SubscriptionTier       func(context.Context, string) (string, error)
	ResolveBusinessAccount func(context.Context, string, string, string) (string, error)
	WriteError             func(http.ResponseWriter, int, string, string)
	LogError               func(string, map[string]any)
}

// GetBusinessAccountUsageSummary resolves membership and subscription from the
// authenticated account context before returning the account's own metering
// view. A caller-supplied tier is never treated as authority.
func GetBusinessAccountUsageSummary(svc *commerce.UsageService, accountSvc *commerce.Service, deps UsageDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIdentity := deps.UserEmail(r.Context())
		userID := deps.UserID(r.Context())
		if userIdentity == "" || userID == "" {
			deps.WriteError(w, http.StatusUnauthorized, "Authentication required", "unauthorized")
			return
		}
		requestedID := strings.TrimSpace(r.URL.Query().Get("business_account_id"))
		if requestedID == "" || deps.ResolveBusinessAccount == nil || accountSvc == nil {
			deps.WriteError(w, http.StatusBadRequest, "business_account_id is required", "validation")
			return
		}
		accountID, err := deps.ResolveBusinessAccount(r.Context(), userID, userIdentity, requestedID)
		if err != nil {
			deps.WriteError(w, http.StatusForbidden, "Business account access is not available", "forbidden")
			return
		}
		subscription, err := accountSvc.GetSubscriptionForBusinessAccountContext(r.Context(), userIdentity, accountID)
		if err != nil {
			deps.LogError("get_business_account_subscription_failed", map[string]any{"error": err.Error(), "business_account_id": accountID})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to get business account subscription", "server_error")
			return
		}
		tier := "free"
		if subscription != nil && subscription.PlanTier != nil && strings.TrimSpace(*subscription.PlanTier) != "" {
			tier = strings.TrimSpace(*subscription.PlanTier)
		}
		summary, err := svc.GetBusinessAccountUsageSummary(r.Context(), accountID, tier)
		if err != nil {
			deps.LogError("get_business_account_usage_summary_failed", map[string]any{"error": err.Error(), "business_account_id": accountID})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to get business account usage summary", "server_error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(summary); err != nil {
			deps.LogError("encode_response_failed", map[string]any{"error": err.Error()})
		}
	}
}

// Fixed diagnostic fields avoid copying private causes or request identity into logs.
func logDirectUsagePreflightFailure(deps UsageDependencies, authority, reason string) {
	if deps.LogError != nil {
		deps.LogError("report_usage_preflight_failed", map[string]any{
			"phase": "preflight", "authority": authority, "reason": reason,
			"code": "meter_authority_unavailable",
		})
	}
}

func ReportUsage(svc *commerce.UsageService, deps UsageDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIdentity := deps.UserEmail(r.Context())
		if userIdentity == "" {
			deps.WriteError(w, http.StatusUnauthorized, "Authentication required", "unauthorized")
			return
		}

		// The endpoint is intentionally batch-shaped. A single object remains
		// accepted as a one-item batch so older clients fail closed on identity
		// while they migrate to the durable outbox protocol.
		var raw json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}

		var requests []commerce.UsageReportRequest
		if len(raw) > 0 && raw[0] == '[' {
			if err := json.Unmarshal(raw, &requests); err != nil {
				deps.WriteError(w, http.StatusBadRequest, "Invalid request body", "validation")
				return
			}
		} else {
			var request commerce.UsageReportRequest
			if err := json.Unmarshal(raw, &request); err != nil {
				deps.WriteError(w, http.StatusBadRequest, "Invalid request body", "validation")
				return
			}
			requests = []commerce.UsageReportRequest{request}
		}
		if len(requests) == 0 {
			deps.WriteError(w, http.StatusBadRequest, "At least one usage item is required", "validation")
			return
		}

		// SubscriptionTier is composed from the authenticated subject's subscription.
		// No body tier, account or app field can grant metering authority.
		if deps.SubscriptionTier == nil {
			logDirectUsagePreflightFailure(deps, "subscription_tier", "missing_callback")
			deps.WriteError(w, http.StatusServiceUnavailable, commerce.ErrDirectUsageAuthorityUnavailable.Error(), "meter_authority_unavailable")
			return
		}
		tier, err := deps.SubscriptionTier(r.Context(), userIdentity)
		if err != nil || strings.TrimSpace(tier) == "" {
			reason := "empty_tier"
			if err != nil {
				reason = "lookup_failed"
			}
			logDirectUsagePreflightFailure(deps, "subscription_tier", reason)
			deps.WriteError(w, http.StatusServiceUnavailable, commerce.ErrDirectUsageAuthorityUnavailable.Error(), "meter_authority_unavailable")
			return
		}
		// Validate the entire batch before the first RecordUsage call. A rejected
		// cost/app/unknown item must never leave an earlier count item written.
		for i := range requests {
			requests[i].UserIdentity = userIdentity
			requests[i].Tier = tier
			if err := svc.ValidateDirectUsage(r.Context(), requests[i]); err != nil {
				if errors.Is(err, commerce.ErrDirectUsageScopeUnavailable) {
					deps.WriteError(w, http.StatusServiceUnavailable, err.Error(), "billing_scope_unavailable")
				} else if errors.Is(err, commerce.ErrDirectUsageAuthorityUnavailable) {
					logDirectUsagePreflightFailure(deps, "tier_limits", "lookup_unavailable")
					deps.WriteError(w, http.StatusServiceUnavailable, commerce.ErrDirectUsageAuthorityUnavailable.Error(), "meter_authority_unavailable")
				} else {
					deps.WriteError(w, http.StatusBadRequest, err.Error(), "validation")
				}
				return
			}
		}
		for i := range requests {
			if err := svc.RecordUsage(r.Context(), requests[i]); err != nil {
				deps.LogError("report_usage_failed", map[string]any{"error": err.Error(), "user_identity": userIdentity, "limit_key": requests[i].LimitKey, "operation_id": requests[i].OperationID})
				if errors.Is(err, commerce.ErrDirectUsageScopeUnavailable) {
					deps.WriteError(w, http.StatusServiceUnavailable, err.Error(), "billing_scope_unavailable")
				} else if errors.Is(err, commerce.ErrDirectUsageAuthorityUnavailable) {
					deps.WriteError(w, http.StatusServiceUnavailable, commerce.ErrDirectUsageAuthorityUnavailable.Error(), "meter_authority_unavailable")
				} else {
					deps.WriteError(w, http.StatusBadRequest, err.Error(), "validation")
				}
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"success": true, "recorded": len(requests)}); err != nil {
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

type reservationRequest struct {
	LimitKey      string `json:"limit_key"`
	Amount        int64  `json:"amount"`
	WindowSeconds int64  `json:"window_seconds,omitempty"`
}

// ReserveCredits exposes the authenticated reservation boundary. Tier is
// resolved from LPBS subscription state; clients cannot choose a stronger
// plan by putting one in the request body.
func ReserveCredits(svc *commerce.UsageService, accountSvc *commerce.Service, deps UsageDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIdentity := deps.UserEmail(r.Context())
		if userIdentity == "" {
			deps.WriteError(w, http.StatusUnauthorized, "Authentication required", "unauthorized")
			return
		}
		var request reservationRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}
		tier := ""
		if deps.SubscriptionTier != nil {
			var err error
			tier, err = quotaTier(r.Context(), userIdentity, deps)
			if err != nil {
				deps.WriteError(w, http.StatusServiceUnavailable, "Subscription service unavailable", "server_error")
				return
			}
		} else if accountSvc != nil {
			subscription, err := accountSvc.GetSubscriptionContext(r.Context(), userIdentity)
			if err != nil {
				deps.WriteError(w, http.StatusServiceUnavailable, "Subscription service unavailable", "server_error")
				return
			}
			if subscription != nil {
				tier = subscription.GetPlanTier()
			}
		}
		if request.WindowSeconds < 0 || request.WindowSeconds > int64((24*time.Hour)/time.Second) {
			deps.WriteError(w, http.StatusBadRequest, "reservation window exceeds maximum of 24h0m0s", "reservation")
			return
		}
		kind, err := svc.MeterKind(r.Context(), tier, request.LimitKey)
		if err != nil {
			deps.WriteError(w, http.StatusServiceUnavailable, "Meter limit authority is unavailable", "server_error")
			return
		}
		if kind != "count_based" {
			deps.WriteError(w, http.StatusServiceUnavailable, "Verified billing scope is unavailable", "server_error")
			return
		}
		reservationID, err := svc.ReserveQuota(r.Context(), userIdentity, tier, request.LimitKey, request.Amount, time.Duration(request.WindowSeconds)*time.Second)
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(strings.ToLower(err.Error()), "insufficient credits") {
				status = http.StatusPaymentRequired
			} else if !strings.Contains(strings.ToLower(err.Error()), "required") && !strings.Contains(strings.ToLower(err.Error()), "positive") {
				status = http.StatusInternalServerError
			}
			deps.LogError("reserve_credits_failed", map[string]any{"error": err.Error(), "user_identity": userIdentity, "limit_key": request.LimitKey})
			deps.WriteError(w, status, err.Error(), "reservation")
			return
		}
		writeReservationJSON(w, map[string]any{"reservation_id": reservationID, "status": "pending"})
	}
}

func FinalizeReservation(svc *commerce.UsageService, deps UsageDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIdentity := deps.UserEmail(r.Context())
		if userIdentity == "" {
			deps.WriteError(w, http.StatusUnauthorized, "Authentication required", "unauthorized")
			return
		}
		var request struct {
			ActualAmount int64 `json:"actual_amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}
		reservationID := mux.Vars(r)["reservationID"]
		tier, err := quotaTier(r.Context(), userIdentity, deps)
		if err != nil {
			deps.WriteError(w, http.StatusServiceUnavailable, "Subscription service unavailable", "server_error")
			return
		}
		if err := svc.FinalizeQuota(r.Context(), userIdentity, tier, reservationID, request.ActualAmount); err != nil {
			reservationError(w, deps, err, "finalize_reservation_failed")
			return
		}
		writeReservationJSON(w, map[string]any{"reservation_id": reservationID, "status": "finalized"})
	}
}

func ReleaseReservation(svc *commerce.UsageService, deps UsageDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIdentity := deps.UserEmail(r.Context())
		if userIdentity == "" {
			deps.WriteError(w, http.StatusUnauthorized, "Authentication required", "unauthorized")
			return
		}
		reservationID := mux.Vars(r)["reservationID"]
		tier, err := quotaTier(r.Context(), userIdentity, deps)
		if err != nil {
			deps.WriteError(w, http.StatusServiceUnavailable, "Subscription service unavailable", "server_error")
			return
		}
		if err := svc.ReleaseQuota(r.Context(), userIdentity, tier, reservationID); err != nil {
			reservationError(w, deps, err, "release_reservation_failed")
			return
		}
		writeReservationJSON(w, map[string]any{"reservation_id": reservationID, "status": "released"})
	}
}

func quotaTier(ctx context.Context, identity string, deps UsageDependencies) (string, error) {
	if deps.SubscriptionTier == nil {
		return "", errors.New("subscription authority is unavailable")
	}
	tier, err := deps.SubscriptionTier(ctx, identity)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(tier) == "" {
		return "free", nil
	}
	return tier, nil
}

func reservationError(w http.ResponseWriter, deps UsageDependencies, err error, event string) {
	status := http.StatusInternalServerError
	code := "server_error"
	if errors.Is(err, commerce.ErrReservationNotOwned) {
		status, code = http.StatusForbidden, "forbidden"
	} else if strings.Contains(strings.ToLower(err.Error()), "required") || strings.Contains(strings.ToLower(err.Error()), "non-negative") {
		status, code = http.StatusBadRequest, "validation"
	} else if errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "not found") {
		status, code = http.StatusNotFound, "not_found"
	}
	deps.LogError(event, map[string]any{"error": err.Error()})
	deps.WriteError(w, status, err.Error(), code)
}

func writeReservationJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
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
