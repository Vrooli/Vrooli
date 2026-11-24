package billing

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"landing-page-business-suite-api/internal/commerce"
)

type AccountCreditLifecycleDependencies struct {
	Service        *commerce.AccountCreditLifecycleService
	UserID         func(r *http.Request) string
	UserIdentity   func(r *http.Request) string
	ResolveAccount func(r *http.Request, requestedID string) (string, error)
	AccountTier    func(r *http.Request, actor, account string) (string, error)
	ResolveApp     func(r *http.Request) (string, error)
	WriteError     func(http.ResponseWriter, int, string, string)
}

var operationIDPattern = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`)

type accountCreditLifecycleHandler struct {
	deps AccountCreditLifecycleDependencies
}

func RegisterAccountCreditLifecycleRoutes(router *mux.Router, deps AccountCreditLifecycleDependencies, requireUserAuth func(http.HandlerFunc) http.HandlerFunc) {
	h := accountCreditLifecycleHandler{deps: deps}
	prefix := "/api/v1/account-credits/reservations"
	// v1 binds the USD ceiling and conversion snapshot into LPBS-owned state.
	// Existing lifecycle transition endpoints remain shared and use the same
	// authenticated actor/app/account binding.
	router.HandleFunc(prefix+"/usd-v1", requireUserAuth(h.reserveUSDBudgetV1)).Methods(http.MethodPost)
	router.HandleFunc(prefix, requireUserAuth(h.reserve)).Methods(http.MethodPost)
	router.HandleFunc(prefix+"/{reservationID}/dispatch", requireUserAuth(h.dispatch)).Methods(http.MethodPost)
	router.HandleFunc(prefix+"/{reservationID}/heartbeat", requireUserAuth(h.heartbeat)).Methods(http.MethodPost)
	router.HandleFunc(prefix+"/{reservationID}/settle", requireUserAuth(h.settle)).Methods(http.MethodPost)
	router.HandleFunc(prefix+"/{reservationID}/settle-usd-v1", requireUserAuth(h.settleMeasuredCostV1)).Methods(http.MethodPost)
	router.HandleFunc(prefix+"/{reservationID}/release", requireUserAuth(h.release)).Methods(http.MethodPost)
	router.HandleFunc(prefix+"/{reservationID}/retain-uncertain", requireUserAuth(h.retain)).Methods(http.MethodPost)
	router.HandleFunc(prefix+"/{reservationID}/recovery", requireUserAuth(h.recover)).Methods(http.MethodGet)
}

func (h accountCreditLifecycleHandler) reserveUSDBudgetV1(w http.ResponseWriter, r *http.Request) {
	actor, app, ok := h.auth(w, r)
	if !ok {
		return
	}
	var body struct {
		OperationID  string `json:"operation_id"`
		BudgetMicros int64  `json:"budget_micros"`
	}
	if !decodeLifecycleBody(w, r, &body) || !operationIDPattern.MatchString(body.OperationID) || body.BudgetMicros <= 0 {
		h.fail(w, http.StatusBadRequest, "a UUIDv4 operation_id and positive budget_micros are required", "validation")
		return
	}
	account, err := h.deps.ResolveAccount(r, "")
	if err != nil || strings.TrimSpace(account) == "" {
		h.fail(w, http.StatusForbidden, "default business account is unavailable", "forbidden")
		return
	}
	tier, err := h.accountTier(r, actor, account)
	if err != nil {
		h.fail(w, http.StatusServiceUnavailable, "account subscription unavailable", "unavailable")
		return
	}
	state, created, err := h.deps.Service.ReserveUSDBudgetV1(r.Context(), account, actor, tier, app, body.OperationID, body.BudgetMicros)
	if err != nil {
		h.lifecycleError(w, err)
		return
	}
	writeLifecycleJSON(w, map[string]any{"reservation_id": body.OperationID, "created": created, "phase": state.Phase, "lease_expires_at": state.LeaseExpires, "contract_version": 1, "reserved_amount": state.Reserved, "budget_micros": body.BudgetMicros, "cost_multiplier": state.CostMultiplier})
}

func (h accountCreditLifecycleHandler) settleMeasuredCostV1(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["reservationID"]
	actor, app, _, ok := h.scope(w, r, id)
	if !ok {
		return
	}
	var body struct {
		CostMicros        int64  `json:"cost_micros"`
		ActiveSeconds     *int64 `json:"active_seconds"`
		PausedSeconds     *int64 `json:"paused_seconds"`
		PromptTokens      int    `json:"prompt_tokens"`
		CompletionTokens  int    `json:"completion_tokens"`
		Model             string `json:"model"`
		Provider          string `json:"provider"`
		ProviderReceiptID string `json:"provider_receipt_id"`
	}
	if !decodeLifecycleBody(w, r, &body) {
		h.fail(w, http.StatusBadRequest, "measured provider usage is required", "validation")
		return
	}
	settlement := commerce.AccountCreditSettlement{CostMicros: body.CostMicros, ActiveSeconds: body.ActiveSeconds, PausedSeconds: body.PausedSeconds, PromptTokens: body.PromptTokens, CompletionTokens: body.CompletionTokens, Model: body.Model, Provider: body.Provider, ProviderReceiptID: body.ProviderReceiptID}
	if err := h.deps.Service.SettleMeasuredCostV1(r.Context(), actor, app, id, settlement); err != nil {
		h.lifecycleError(w, err)
		return
	}
	writeLifecycleJSON(w, map[string]any{"reservation_id": id, "phase": "settled", "contract_version": 1})
}

func (h accountCreditLifecycleHandler) auth(w http.ResponseWriter, r *http.Request) (actor, app string, ok bool) {
	if h.deps.Service == nil || h.deps.UserID == nil || h.deps.UserIdentity == nil || h.deps.ResolveAccount == nil || h.deps.ResolveApp == nil {
		h.fail(w, http.StatusServiceUnavailable, "account-credit authority unavailable", "unavailable")
		return "", "", false
	}
	actor = strings.ToLower(strings.TrimSpace(h.deps.UserIdentity(r)))
	if strings.TrimSpace(h.deps.UserID(r)) == "" || actor == "" {
		h.fail(w, http.StatusUnauthorized, "Authentication required", "unauthorized")
		return "", "", false
	}
	app, err := h.deps.ResolveApp(r)
	if err != nil || strings.TrimSpace(app) == "" {
		h.fail(w, http.StatusForbidden, "app credit lifecycle capability required", "forbidden")
		return "", "", false
	}
	return actor, strings.TrimSpace(app), true
}

func (h accountCreditLifecycleHandler) fail(w http.ResponseWriter, status int, message, kind string) {
	if h.deps.WriteError != nil && w != nil {
		h.deps.WriteError(w, status, message, kind)
	}
}

func (h accountCreditLifecycleHandler) scope(w http.ResponseWriter, r *http.Request, reservationID string) (actor, app, account string, ok bool) {
	actor, app, ok = h.auth(w, r)
	if !ok {
		return "", "", "", false
	}
	account, err := h.deps.Service.ReservationAccount(r.Context(), actor, app, reservationID)
	if err != nil || strings.TrimSpace(account) == "" {
		h.fail(w, http.StatusNotFound, "reservation unavailable", "not_found")
		return "", "", "", false
	}
	resolved, err := h.deps.ResolveAccount(r, account)
	if err != nil || resolved != account {
		h.fail(w, http.StatusForbidden, "reservation is unavailable to this user", "forbidden")
		return "", "", "", false
	}
	return actor, app, account, true
}

func (h accountCreditLifecycleHandler) reserve(w http.ResponseWriter, r *http.Request) {
	actor, app, ok := h.auth(w, r)
	if !ok {
		return
	}
	var body struct {
		OperationID string `json:"operation_id"`
		Amount      int64  `json:"amount"`
	}
	if !decodeLifecycleBody(w, r, &body) || !operationIDPattern.MatchString(body.OperationID) || body.Amount <= 0 {
		h.fail(w, http.StatusBadRequest, "a UUIDv4 operation_id and positive amount are required", "validation")
		return
	}
	account, err := h.deps.ResolveAccount(r, "")
	if err != nil || strings.TrimSpace(account) == "" {
		h.fail(w, http.StatusForbidden, "default business account is unavailable", "forbidden")
		return
	}
	tier := "free"
	if h.deps.Service == nil {
		h.fail(w, http.StatusServiceUnavailable, "account-credit authority unavailable", "unavailable")
		return
	}
	// The account plan is resolved by the composition-owned account service and
	// passed to the reservation owner through the composed callback.
	tier, err = h.accountTier(r, actor, account)
	if err != nil {
		h.fail(w, http.StatusServiceUnavailable, "account subscription unavailable", "unavailable")
		return
	}
	state, created, err := h.deps.Service.Reserve(r.Context(), account, actor, tier, app, body.OperationID, body.Amount)
	if err != nil {
		h.lifecycleError(w, err)
		return
	}
	writeLifecycleJSON(w, map[string]any{"reservation_id": body.OperationID, "created": created, "phase": state.Phase, "lease_expires_at": state.LeaseExpires})
}

func (h accountCreditLifecycleHandler) dispatch(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["reservationID"]
	actor, app, _, ok := h.scope(w, r, id)
	if !ok {
		return
	}
	started, err := h.deps.Service.BeginDispatch(r.Context(), actor, app, id)
	if err != nil {
		h.lifecycleError(w, err)
		return
	}
	writeLifecycleJSON(w, map[string]any{"dispatch_authorized": started, "phase": map[bool]string{true: "dispatched", false: "already_dispatched_or_terminal"}[started]})
}

func (h accountCreditLifecycleHandler) heartbeat(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["reservationID"]
	actor, app, _, ok := h.scope(w, r, id)
	if !ok {
		return
	}
	var body struct {
		LeaseSeconds int `json:"lease_seconds"`
	}
	if !decodeLifecycleBody(w, r, &body) {
		h.fail(w, http.StatusBadRequest, "invalid heartbeat request", "validation")
		return
	}
	until, err := h.deps.Service.Heartbeat(r.Context(), actor, app, id, time.Duration(body.LeaseSeconds)*time.Second)
	if err != nil {
		h.lifecycleError(w, err)
		return
	}
	writeLifecycleJSON(w, map[string]any{"lease_expires_at": until})
}

func (h accountCreditLifecycleHandler) settle(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["reservationID"]
	actor, app, _, ok := h.scope(w, r, id)
	if !ok {
		return
	}
	var body commerce.AccountCreditSettlement
	if !decodeLifecycleBody(w, r, &body) {
		h.fail(w, http.StatusBadRequest, "measured provider usage is required", "validation")
		return
	}
	if err := h.deps.Service.Settle(r.Context(), actor, app, id, body); err != nil {
		h.lifecycleError(w, err)
		return
	}
	writeLifecycleJSON(w, map[string]any{"reservation_id": id, "phase": "settled"})
}

func (h accountCreditLifecycleHandler) release(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["reservationID"]
	actor, app, _, ok := h.scope(w, r, id)
	if !ok {
		return
	}
	if err := h.deps.Service.Release(r.Context(), actor, app, id); err != nil {
		h.lifecycleError(w, err)
		return
	}
	writeLifecycleJSON(w, map[string]any{"reservation_id": id, "phase": "released"})
}

func (h accountCreditLifecycleHandler) retain(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["reservationID"]
	actor, app, _, ok := h.scope(w, r, id)
	if !ok {
		return
	}
	if err := h.deps.Service.RetainUncertain(r.Context(), actor, app, id); err != nil {
		h.lifecycleError(w, err)
		return
	}
	writeLifecycleJSON(w, map[string]any{"reservation_id": id, "phase": "uncertain"})
}

func (h accountCreditLifecycleHandler) recover(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["reservationID"]
	actor, app, _, ok := h.scope(w, r, id)
	if !ok {
		return
	}
	state, err := h.deps.Service.Recover(r.Context(), actor, app, id)
	if err != nil {
		h.lifecycleError(w, err)
		return
	}
	writeLifecycleJSON(w, state)
}

func (h accountCreditLifecycleHandler) accountTier(r *http.Request, actor, account string) (string, error) {
	if h.deps.AccountTier == nil {
		return "", errors.New("account tier resolver unavailable")
	}
	return h.deps.AccountTier(r, actor, account)
}

func (h accountCreditLifecycleHandler) lifecycleError(w http.ResponseWriter, err error) {
	status, kind, message := http.StatusServiceUnavailable, "billing_unavailable", "account-credit operation unavailable"
	switch {
	case errors.Is(err, commerce.ErrCreditLifecycleState):
		status, kind, message = http.StatusConflict, "state_conflict", "reservation lifecycle state does not permit this operation"
	case errors.Is(err, commerce.ErrInvalidCreditSettlement):
		status, kind, message = http.StatusBadRequest, "validation", "measured provider usage is invalid"
	case errors.Is(err, commerce.ErrInvalidCreditLease):
		status, kind, message = http.StatusBadRequest, "validation", "lease must be between 30 seconds and 5 minutes"
	case strings.Contains(strings.ToLower(err.Error()), "insufficient credits"):
		status, kind, message = http.StatusPaymentRequired, "insufficient_credits", "account has insufficient available credits"
	}
	h.fail(w, status, message, kind)
}

func decodeLifecycleBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return false
	}
	return decoder.Decode(new(any)) == io.EOF
}

func writeLifecycleJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
