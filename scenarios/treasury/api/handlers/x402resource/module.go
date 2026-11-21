// Package x402resource exposes service-authenticated, deferred inbound x402
// settlement for LPBS orders. Verification records authority; only the later
// settle call can move value.
package x402resource

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"treasury/internal/module"
	x402rail "treasury/internal/rail/x402"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
)

const (
	pathVerify   = "/api/v1/x402/authorizations/verify"
	pathSettle   = "/api/v1/x402/authorizations/{admission_id}/settle"
	pathCancel   = "/api/v1/x402/authorizations/{admission_id}/cancel"
	audience     = "scenario-authenticator:treasury-x402-order"
	capAuthorize = "treasury:x402-authorize"
	capSettle    = "treasury:x402-settle"
	capCancel    = "treasury:x402-cancel"
	maxBody      = 64 << 10
)

type handler struct{ gate *x402rail.Gate }

func Module(gate *x402rail.Gate) module.Module {
	h := &handler{gate: gate}
	identity := authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{
		Issuer: strings.TrimSpace(os.Getenv("VROOLI_AUTH_SCENARIO_ISSUER")), Audience: audience,
		JWKSURL: strings.TrimSpace(os.Getenv("VROOLI_AUTH_SCENARIO_JWKS_URL")), ExclusiveAudience: true,
	})
	return module.Module{Name: "x402resource", Mount: func(router *mux.Router) {
		router.Handle(pathVerify, authn.Middleware(authn.Config{Providers: []authn.Provider{identity}})(http.HandlerFunc(h.verify))).Methods(http.MethodPost)
		router.Handle(pathSettle, authn.Middleware(authn.Config{Providers: []authn.Provider{identity}})(http.HandlerFunc(h.settle))).Methods(http.MethodPost)
		router.Handle(pathCancel, authn.Middleware(authn.Config{Providers: []authn.Provider{identity}})(http.HandlerFunc(h.cancel))).Methods(http.MethodPost)
	}, Endpoints: Endpoints}
}

type authorizationRequest struct {
	PriceID          string `json:"price_id"`
	JobID            string `json:"job_id"`
	ResourceID       string `json:"resource_id"`
	OrderID          string `json:"order_id"`
	PaymentSignature string `json:"payment_signature"`
}

type authorizationResponse struct {
	AdmissionID   string `json:"admission_id"`
	PriceID       string `json:"price_id"`
	Status        string `json:"status"`
	Payer         string `json:"payer"`
	TransactionID string `json:"transaction_id,omitempty"`
	Network       string `json:"network"`
	Mode          string `json:"mode"`
	JobID         string `json:"job_id"`
	ResourceID    string `json:"resource_id"`
	OrderID       string `json:"order_id"`
	Detail        string `json:"detail,omitempty"`
}

type settlementRequest struct {
	JobID      string `json:"job_id"`
	ResourceID string `json:"resource_id"`
	OrderID    string `json:"order_id"`
}

func (h *handler) verify(w http.ResponseWriter, r *http.Request) {
	if _, err := authn.RequireService(r.Context()); err != nil {
		writeError(w, http.StatusUnauthorized, "verified service identity is required")
		return
	}
	if _, err := authn.RequireCapability(r.Context(), capAuthorize); err != nil {
		writeError(w, http.StatusForbidden, "Treasury x402 authorize capability is required")
		return
	}
	var input authorizationRequest
	if err := decode(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid resource authorization request")
		return
	}
	if h.gate == nil {
		writeError(w, http.StatusServiceUnavailable, "Treasury x402 gate is unavailable")
		return
	}
	value, err := h.gate.VerifyForResource(r.Context(), input.PriceID, input.JobID, input.ResourceID, input.OrderID, input.PaymentSignature)
	if err != nil {
		writeGateError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response(value))
}

func (h *handler) settle(w http.ResponseWriter, r *http.Request) {
	if _, err := authn.RequireService(r.Context()); err != nil {
		writeError(w, http.StatusUnauthorized, "verified service identity is required")
		return
	}
	if _, err := authn.RequireCapability(r.Context(), capSettle); err != nil {
		writeError(w, http.StatusForbidden, "Treasury x402 settle capability is required")
		return
	}
	var input settlementRequest
	if err := decode(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid resource settlement request")
		return
	}
	if h.gate == nil {
		writeError(w, http.StatusServiceUnavailable, "Treasury x402 gate is unavailable")
		return
	}
	value, err := h.gate.SettleAuthorization(r.Context(), strings.TrimSpace(mux.Vars(r)["admission_id"]), input.JobID, input.ResourceID, input.OrderID)
	if err != nil {
		if errors.Is(err, x402rail.ErrUnknown) || errors.Is(err, x402rail.ErrInProgress) {
			writeJSON(w, http.StatusAccepted, response(value))
			return
		}
		writeGateError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response(value))
}

func (h *handler) cancel(w http.ResponseWriter, r *http.Request) {
	if _, err := authn.RequireService(r.Context()); err != nil {
		writeError(w, http.StatusUnauthorized, "verified service identity is required")
		return
	}
	if _, err := authn.RequireCapability(r.Context(), capCancel); err != nil {
		writeError(w, http.StatusForbidden, "Treasury x402 cancel capability is required")
		return
	}
	var input settlementRequest
	if err := decode(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid resource cancellation request")
		return
	}
	if h.gate == nil {
		writeError(w, http.StatusServiceUnavailable, "Treasury x402 gate is unavailable")
		return
	}
	value, err := h.gate.CancelAuthorization(r.Context(), strings.TrimSpace(mux.Vars(r)["admission_id"]), input.JobID, input.ResourceID, input.OrderID)
	if err != nil {
		writeGateError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response(value))
}

func response(value x402rail.Admission) authorizationResponse {
	status := value.Status
	if status == "verified" {
		status = "authorized"
	}
	return authorizationResponse{AdmissionID: value.ID, PriceID: value.PriceID, Status: status, Payer: value.Payer, TransactionID: value.TransactionID, Network: value.Network, Mode: value.Mode, JobID: value.JobID, ResourceID: value.ResourceID, OrderID: value.OrderID, Detail: value.Detail}
}

func decode(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxBody))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeGateError(w http.ResponseWriter, err error) {
	status := http.StatusServiceUnavailable
	if errors.Is(err, x402rail.ErrInvalid) {
		status = http.StatusBadRequest
	}
	if errors.Is(err, x402rail.ErrBindingConflict) {
		status = http.StatusConflict
	}
	if errors.Is(err, x402rail.ErrStateConflict) {
		status = http.StatusConflict
	}
	if errors.Is(err, x402rail.ErrNotFound) {
		status = http.StatusNotFound
	}
	if errors.Is(err, x402rail.ErrUnknown) || errors.Is(err, x402rail.ErrInProgress) {
		status = http.StatusAccepted
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
