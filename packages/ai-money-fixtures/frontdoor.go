package aimoneyfixtures

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
)

// FakeFrontDoor stands in for the single public LPBS door. It does not emulate
// account authentication, wallet persistence, reserve/finalize, or a real
// router; those require separate integration tests against their owners.
var ErrNoEligibleRoute = errors.New("no eligible route")

// CreditsRequiredBody returns the public refusal body paired with HTTP 402.
func CreditsRequiredBody() PublicError {
	return PublicError{Error: ErrorCreditsRequired, ErrorType: ErrorCreditsRequired}
}

// NoEligibleRouteBody returns the public route refusal body paired with HTTP 402.
func NoEligibleRouteBody() PublicError {
	return PublicError{Error: ErrorNoEligibleRoute, ErrorType: ErrorNoEligibleRoute}
}

type FakeFrontDoor struct {
	Gateway       GatewayPrivate
	UseClass      string // test-controlled server-side value; never accepted from JSON
	mu            sync.Mutex
	Calls         []FeatureRequest
	CreditRefusal bool
}

func (f *FakeFrontDoor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writePublicError(w, http.StatusMethodNotAllowed, "method_not_allowed", false)
		return
	}
	request, featureID, model, status, err := decodeFrontDoorRequest(w, r)
	if err != nil {
		writePublicError(w, status, err.Error(), false)
		return
	}
	f.mu.Lock()
	f.Calls = append(f.Calls, request)
	f.mu.Unlock()
	if f.CreditRefusal {
		writeJSON(w, http.StatusPaymentRequired, CreditsRequiredBody())
		return
	}
	if f.Gateway == nil {
		writeJSON(w, http.StatusServiceUnavailable, PublicError{Error: "gateway_unavailable", ErrorType: "gateway_unavailable", Retryable: true})
		return
	}
	plan, err := f.Gateway.Plan(r.Context(), GatewayPlanRequest{OperationID: r.Header.Get("Idempotency-Key"), FeatureID: featureID, UseClass: f.UseClass, Model: model, BudgetMicros: 1000, PrivacyClass: "standard"})
	if err != nil {
		if errors.Is(err, ErrNoEligibleRoute) {
			writeJSON(w, http.StatusPaymentRequired, NoEligibleRouteBody())
		} else {
			writeJSON(w, http.StatusServiceUnavailable, PublicError{Error: "gateway_unavailable", ErrorType: "gateway_unavailable", Retryable: true})
		}
		return
	}
	result, err := f.Gateway.Execute(r.Context(), GatewayExecuteRequest{OperationID: r.Header.Get("Idempotency-Key"), Plan: plan, Feature: request, Funded: true, BudgetMicros: 1000})
	if err != nil {
		writePublicError(w, http.StatusBadGateway, "upstream_failed", true)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"output": result.Output, "usage": map[string]int64{"input_tokens": result.InputTokens, "output_tokens": result.OutputTokens}})
}

func decodeFrontDoorRequest(w http.ResponseWriter, r *http.Request) (FeatureRequest, string, string, int, error) {
	if r.URL.Path == "/v1/chat/completions" {
		return decodeChatRequest(w, r)
	}
	if !strings.HasPrefix(r.URL.Path, "/v1/features/") {
		return FeatureRequest{}, "", "", http.StatusNotFound, errors.New("not_found")
	}
	featureID := strings.TrimPrefix(r.URL.Path, "/v1/features/")
	if featureID == "" {
		return FeatureRequest{}, "", "", http.StatusBadRequest, errors.New("invalid_feature")
	}
	req, err := decodeFeatureRequest(w, r)
	if err != nil {
		return FeatureRequest{}, "", "", http.StatusBadRequest, err
	}
	return req, featureID, "auto", 0, nil
}

func decodeChatRequest(w http.ResponseWriter, r *http.Request) (FeatureRequest, string, string, int, error) {
	var chat ChatRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&chat); err != nil || len(chat.Messages) != 1 || chat.Messages[0].Role != "user" || strings.TrimSpace(chat.Messages[0].Content) == "" {
		return FeatureRequest{}, "", "", http.StatusBadRequest, errors.New("invalid_request")
	}
	model := strings.TrimSpace(chat.Model)
	if model == "" {
		model = "auto"
	}
	return FeatureRequest{Turn: chat.Messages[0]}, "coding-agent", model, 0, nil
}

func decodeFeatureRequest(w http.ResponseWriter, r *http.Request) (FeatureRequest, error) {
	var req FeatureRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		return FeatureRequest{}, errors.New("invalid_request")
	}
	if strings.TrimSpace(req.Instruction) == "" || req.Turn.Role != "user" || strings.TrimSpace(req.Turn.Content) == "" || len(req.Schema) == 0 {
		return FeatureRequest{}, errors.New("invalid_request")
	}
	return req, nil
}

func writePublicError(w http.ResponseWriter, status int, code string, retryable bool) {
	writeJSON(w, status, PublicError{Error: code, ErrorType: code, Retryable: retryable})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// StubEligible is a deterministic test stub, not the policy implementation.
func StubEligible(allowed bool, reason string) EligibleFunc {
	return func(any, any) (bool, string) { return allowed, reason }
}
