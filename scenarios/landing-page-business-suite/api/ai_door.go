package main

import (
	"context"
	"encoding/json"
	"github.com/vrooli/api-core/discovery"
	"landing-page-business-suite-api/internal/aidoor"
	"landing-page-business-suite-api/internal/commerce"
	"net/http"
)

func registerAIDoor(s *Server) {
	db := s.primaryDB()
	if db == nil {
		return
	}
	live := isProductionEnvironment()
	door := &aidoor.Door{Keys: aidoor.KeyStore{DB: db}, Wallet: commerce.NewBusinessAccountUsageService(commerce.BusinessAccountUsageOptions{DB: db, Limits: commerce.NewLimitsService(db, "postgres", nil), Dialect: "postgres", LiveMode: &live}), ResolveGateway: func(ctx context.Context) (string, error) {
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
		// A public client label is attribution, never app billing authority.
		// Until app-bound issuance has an authenticated owner, this is a global
		// account credential; configured app caps fail closed at admission.
		key, err := door.Keys.IssueDevice(r.Context(), input.Account, getUserID(r.Context()), "", input.MaxCredits)
		if err != nil {
			writeJSONError(w, 403, err.Error(), "forbidden")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(key)
	})).Methods("POST")
	s.router.Handle("/api/v1/features/{feature}", door).Methods("POST")
	s.router.Handle("/api/v1/ai/models", door).Methods("GET")
	s.router.Handle("/api/v1/ai/responses", door).Methods("POST")
	s.router.Handle("/api/v1/ai/chat/completions", door).Methods("POST")
	s.router.Handle("/api/v1/ai/keys/session", door).Methods("POST")
	s.router.Handle("/api/v1/ai/keys/{id}", door).Methods("DELETE")
}
