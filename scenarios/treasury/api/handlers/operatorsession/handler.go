package operatorsession

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/browsersession"
	coreidentity "github.com/vrooli/api-core/identity"
	"treasury/internal/budget"
	"treasury/internal/operatorauth"
)

const path = "/api/v1/operator-session"

type verifier interface {
	Verify(context.Context, string) (coreidentity.Principal, error)
}

// Handler owns the same-origin sign-in edge for Treasury's operator UI. The
// browser receives only HttpOnly cookies and bounded session status; bearer
// credentials are never returned in a response body.
type Handler struct {
	Edge     *browsersession.Edge
	Verifier verifier
	Operator operatorauth.Authorizer
	Budgets  interface {
		Headroom(context.Context, string) (budget.Headroom, error)
	}
	OwnsBudget func(context.Context, string, string) bool
}

func (h Handler) Register(mux *http.ServeMux) {
	mux.Handle(path, h)
	mux.Handle(path+"/", h)
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	if h.Edge == nil {
		http.Error(w, "operator sign-in unavailable", http.StatusServiceUnavailable)
		return
	}
	switch {
	case r.URL.Path == path+"/sign-in" && r.Method == http.MethodPost:
		h.signIn(w, r)
	case r.URL.Path == path+"/refresh" && r.Method == http.MethodPost:
		h.refresh(w, r)
	case r.URL.Path == path+"/sign-out" && r.Method == http.MethodPost:
		h.signOut(w, r)
	case r.URL.Path == path && r.Method == http.MethodGet:
		h.status(w, r)
	case r.URL.Path == path+"/budget-headroom" && r.Method == http.MethodGet:
		h.headroom(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.NotFound(w, r)
	}
}

func (h Handler) headroom(w http.ResponseWriter, r *http.Request) {
	if h.Operator == nil || h.Budgets == nil || h.OwnsBudget == nil {
		http.Error(w, "budget headroom unavailable", http.StatusServiceUnavailable)
		return
	}
	operator, err := h.Operator.Authorize(r.Context(), r.Header)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, operatorauth.ErrDenied) {
			status = http.StatusForbidden
		} else if errors.Is(err, operatorauth.ErrUnavailable) {
			status = http.StatusServiceUnavailable
		}
		http.Error(w, err.Error(), status)
		return
	}
	budgetID := strings.TrimSpace(r.URL.Query().Get("budget_id"))
	if budgetID == "" {
		http.Error(w, "budget_id is required", http.StatusBadRequest)
		return
	}
	if !h.OwnsBudget(r.Context(), operator.Subject, budgetID) {
		http.Error(w, "budget headroom could not be read", http.StatusNotFound)
		return
	}
	value, err := h.Budgets.Headroom(r.Context(), budgetID)
	if err != nil {
		http.Error(w, "budget headroom could not be read", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"budget_id": value.BudgetID, "book_id": value.BookID, "currency": value.Currency,
		"total_cap_minor": value.TotalCapMinor, "total_used_minor": value.TotalUsedMinor,
		"total_remaining_minor": value.TotalRemainingMinor, "periodic_cap_minor": value.PeriodicCapMinor,
		"period_used_minor": value.PeriodUsedMinor, "period_remaining_minor": value.PeriodRemainingMinor,
		"per_transaction_cap_minor": value.PerTransactionCapMinor, "available_minor": value.AvailableMinor,
		"period_started_at": value.PeriodStartedAt.UTC().Format(time.RFC3339Nano), "computed_at": value.ComputedAt.UTC().Format(time.RFC3339Nano),
		"funding_required": value.FundingRequired, "funding_credited_minor": value.FundingCreditedMinor,
		"funding_remaining_minor": value.FundingRemainingMinor, "funding_atomic_amount": value.FundingAtomicAmount,
		"funding_asset_id": value.FundingAssetID,
	})
}

type signInRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	TOTPCode     string `json:"totp_code,omitempty"`
	RecoveryCode string `json:"recovery_code,omitempty"`
	MFAChallenge string `json:"mfa_challenge,omitempty"`
}

func (h Handler) signIn(w http.ResponseWriter, r *http.Request) {
	var input signInRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid sign-in request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.Email) == "" || input.Password == "" {
		http.Error(w, "email and password are required", http.StatusBadRequest)
		return
	}
	outcome, err := h.Edge.SignIn(r.Context(), r.Header, w.Header(), browsersession.Credentials{
		Email: input.Email, Password: input.Password, TOTPCode: input.TOTPCode,
		RecoveryCode: input.RecoveryCode, MFAChallenge: input.MFAChallenge,
	})
	if err != nil {
		writeSessionError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"authenticated": !outcome.MFARequired,
		"mfa_required":  outcome.MFARequired,
		"mfa_challenge": outcome.MFAChallenge,
	})
}

func (h Handler) refresh(w http.ResponseWriter, r *http.Request) {
	outcome, err := h.Edge.Renew(r.Context(), r.Header, w.Header())
	if err != nil {
		writeSessionError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"authenticated": !outcome.MFARequired})
}

func (h Handler) signOut(w http.ResponseWriter, r *http.Request) {
	err := h.Edge.SignOut(r.Context(), r.Header, w.Header())
	if err != nil {
		writeSessionError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) status(w http.ResponseWriter, r *http.Request) {
	if h.Verifier == nil {
		http.Error(w, "operator sign-in unavailable", http.StatusServiceUnavailable)
		return
	}
	cookie, err := r.Cookie(browsersession.AccessCookie)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		http.Error(w, "operator session required", http.StatusUnauthorized)
		return
	}
	principal, err := h.Verifier.Verify(r.Context(), strings.TrimSpace(cookie.Value))
	if err != nil || !principal.IsVerified() || !principal.Kind.IsHuman() || strings.TrimSpace(principal.Subject) == "" {
		http.Error(w, "operator session is invalid", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"authenticated": true, "subject": strings.TrimSpace(principal.Subject)})
}

func writeSessionError(w http.ResponseWriter, err error) {
	status := http.StatusServiceUnavailable
	if errors.Is(err, browsersession.ErrInvalidInput) {
		status = http.StatusBadRequest
	} else if errors.Is(err, browsersession.ErrInvalidCredentials) || errors.Is(err, browsersession.ErrNoSession) {
		status = http.StatusUnauthorized
	} else if errors.Is(err, browsersession.ErrRateLimited) {
		status = http.StatusTooManyRequests
	}
	http.Error(w, err.Error(), status)
}
