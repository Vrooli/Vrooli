package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/consumeridentity"
	"github.com/vrooli/api-core/discovery"
	"landing-page-business-suite-api/internal/aidoor"
	"landing-page-business-suite-api/internal/commerce"
)

func registerAIDoor(s *Server) {
	db := s.primaryDB()
	if db == nil {
		return
	}
	live := isProductionEnvironment()
	door := &aidoor.Door{Keys: aidoor.KeyStore{DB: db}, Wallet: commerce.NewBusinessAccountUsageService(commerce.BusinessAccountUsageOptions{DB: db, Limits: commerce.NewLimitsService(db, "postgres", nil), Dialect: "postgres", LiveMode: &live}), SignCreditAssertion: func(key aidoor.Key, operation, featureID string) (string, error) {
		return s.userAuthService.SignAICreditAssertion(consumeridentity.CreditAssertionExpectation{BusinessAccountID: key.Account, ActorID: key.Actor, AppKey: key.App, FeatureID: featureID, MaxCredits: key.MaxCredits, Audience: consumeridentity.CreditGatewayAudience, OperationID: operation, Source: consumeridentity.CreditAssertionSource})
	}, ResolveGateway: func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
	}}
	go door.RunRecovery(context.Background())
	registerAIDoorRoutes(s, door)
}

func registerAIDoorRoutes(s *Server, door *aidoor.Door) {
	s.router.HandleFunc("/api/v1/ai/keys/device", s.requireUserAuth(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Account    string `json:"business_account_id"`
			App        string `json:"app_bundle_key"`
			MaxCredits int64  `json:"max_credits"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&input) != nil {
			writeJSONError(w, 400, "invalid device key request", "invalid_request")
			return
		}
		if s.businessAccounts == nil || s.accountService == nil || s.userAuthService == nil {
			writeJSONError(w, 503, "app credit authority is unavailable", ApiErrorTypeServerError)
			return
		}
		actor := getUserID(r.Context())
		account, err := s.businessAccounts.ResolveForUser(r.Context(), actor, getUserEmail(r.Context()), strings.TrimSpace(input.Account))
		if err != nil {
			writeJSONError(w, 403, "business account access is not available", ApiErrorTypeForbidden)
			return
		}
		appKey := strings.ToLower(strings.TrimSpace(input.App))
		entitlements, err := s.accountService.GetEntitlementsForBusinessAccountContext(r.Context(), getUserEmail(r.Context()), account.ID)
		if err != nil {
			writeJSONError(w, 503, "app credit entitlement is unavailable", ApiErrorTypeServerError)
			return
		}
		appAllowed := false
		for _, grant := range entitlements.AppEntitlements {
			if grant.AppKey != appKey {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(grant.Status)) {
			case "active", "trialing", "past_due":
				appAllowed = true
			}
		}
		if !appAllowed {
			writeJSONError(w, 403, "active app entitlement is required", ApiErrorTypeForbidden)
			return
		}
		key, err := door.Keys.IssueDevice(r.Context(), account.ID, actor, appKey, input.MaxCredits)
		if err != nil {
			writeJSONError(w, 403, err.Error(), "forbidden")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(key)
	})).Methods("POST")
	s.router.Handle("/api/v1/features/{feature}", door).Methods("POST")
	s.router.Handle("/api/v1/ai/models", door).Methods("GET")
	s.router.Handle("/api/v1/ai/receipt", door).Methods("GET")
	s.router.Handle("/api/v1/ai/responses", door).Methods("POST")
	s.router.Handle("/api/v1/ai/chat/completions", door).Methods("POST")
	s.router.PathPrefix("/api/v1/ai/").Handler(door).Methods("POST")
	s.router.Handle("/api/v1/ai/keys/session", door).Methods("POST")
	s.router.Handle("/api/v1/ai/keys/{id}", door).Methods("DELETE")
}
