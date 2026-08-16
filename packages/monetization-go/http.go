package monetization

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type accessTokenContextKey struct{}

// WithAccessToken stores a request-scoped consumer token. It is intentionally
// not a substitute for lease verification and must never be persisted.
func WithAccessToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, accessTokenContextKey{}, strings.TrimSpace(token))
}

// AccessTokenFromContext returns the short-lived consumer token attached by
// InjectEntitlement. Callers still need a verified lease before granting work.
func AccessTokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(accessTokenContextKey{}).(string)
	return strings.TrimSpace(token)
}

// InjectEntitlement carries the bearer token into the request context. It
// deliberately ignores user headers, query parameters, and request bodies.
func InjectEntitlement(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "
		authorization := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(authorization, prefix) {
			next.ServeHTTP(w, r.WithContext(WithAccessToken(r.Context(), strings.TrimSpace(strings.TrimPrefix(authorization, prefix)))))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireActiveSubscription gates a route using only the verified lease for
// the supplied identity. A missing identity cannot grant access.
func RequireActiveSubscription(gate *Gate, identity func(context.Context) string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := ""
		if identity != nil {
			id = strings.TrimSpace(identity(r.Context()))
		}
		decision := gate.Feature(r.Context(), id, "", 0)
		if !decision.Allowed {
			WriteError(w, http.StatusForbidden, ErrorSubscriptionRequired, decision)
			return
		}
		if decision.Warning {
			w.Header().Set("X-Entitlement-Warning", ReasonPastDue)
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRank gates a route at a plan rank from the signed lease.
func RequireRank(gate *Gate, identity func(context.Context) string, rank int32, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := ""
		if identity != nil {
			id = strings.TrimSpace(identity(r.Context()))
		}
		decision := gate.Feature(r.Context(), id, "", rank)
		if !decision.Allowed {
			WriteError(w, http.StatusForbidden, ErrorRankRequired, decision)
			return
		}
		next.ServeHTTP(w, r)
	})
}

const (
	// ErrorUnauthorized is returned when no consumer credential is available.
	ErrorUnauthorized = "unauthorized"
	// ErrorSubscriptionRequired is returned for inactive or missing access.
	ErrorSubscriptionRequired = "subscription_required"
	// ErrorCreditsRequired is returned when a trusted wallet rejects a charge.
	ErrorCreditsRequired = "credits_required"
	// ErrorAuthorityUnavailable is returned when no valid lease is available.
	ErrorAuthorityUnavailable = "authority_unavailable"
	// ErrorRateLimited is returned when an operation is throttled.
	ErrorRateLimited = "rate_limited"
	// ErrorRankRequired is returned when the signed plan rank is insufficient.
	ErrorRankRequired = "rank_required"
)

// ErrorResponse is the stable, non-authority-leaking error shape for paid
// surfaces. The raw LPBS message is never copied into this response.
type ErrorResponse struct {
	Error     string `json:"error"`
	ErrorType string `json:"error_type"`
	Retryable bool   `json:"retryable"`
}

// WriteError writes the shared paid-surface error shape.
func WriteError(w http.ResponseWriter, status int, errorType string, decision Decision) {
	retryable := decision.Reason == ReasonLeaseUnavailable
	if status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable {
		retryable = true
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: errorType, ErrorType: errorType, Retryable: retryable})
}

// CreditsDisplay converts internal credit units to the public display unit.
// The multiplier and label are contract configuration, never a gate decision.
type CreditsDisplay struct {
	Value      float64
	Label      string
	Multiplier float64
}

// DisplayCredits applies a positive display multiplier and stable label.
func DisplayCredits(units int64, multiplier float64, label string) CreditsDisplay {
	if multiplier <= 0 {
		multiplier = 1
	}
	return CreditsDisplay{Value: float64(units) * multiplier, Label: strings.TrimSpace(label), Multiplier: multiplier}
}
