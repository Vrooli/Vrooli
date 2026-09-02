package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

const (
	webConsoleSubscriptionIdentity = credentialclient.SubscriptionIdentity
	webConsoleSubscriptionField    = credentialclient.SubscriptionField
	webConsoleOpenRouterIdentity   = "vrooli/openrouter"
	webConsoleOpenRouterField      = "api-key"
)

type credentialProvisionRequest struct {
	Identity string `json:"identity"`
	Field    string `json:"field"`
	Value    string `json:"value"`
}

// credentialProvisionHandler is deliberately metadata-only on output. The
// authority is the only durable secret store; the browser receives status,
// never a value or a derived representation of one.
func (s *Server) credentialProvisionHandler(w http.ResponseWriter, r *http.Request) {
	if !webConsoleSameOrigin(w, r) {
		return
	}
	if s.credentialClient == nil {
		http.Error(w, "credential authority unavailable", http.StatusServiceUnavailable)
		return
	}
	var request credentialProvisionRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil || strings.TrimSpace(request.Value) == "" {
		http.Error(w, "identity, field, and value are required", http.StatusBadRequest)
		return
	}
	refs, err := s.credentialClient.List(r.Context())
	if err != nil {
		http.Error(w, "credential inventory unavailable", http.StatusBadGateway)
		return
	}
	declared := false
	for _, ref := range refs {
		if ref.LogicalID == strings.TrimSpace(request.Identity) && ref.Field == strings.TrimSpace(request.Field) {
			declared = true
			break
		}
	}
	if !declared {
		http.Error(w, "credential is not declared by web-console", http.StatusForbidden)
		return
	}
	response, err := s.credentialClient.Provision(r.Context(), credentialclient.ProvisionRequest{
		Identity: strings.TrimSpace(request.Identity), Field: strings.TrimSpace(request.Field), Value: request.Value,
	})
	if err != nil {
		http.Error(w, "credential could not be stored", http.StatusBadGateway)
		return
	}
	writeCredentialJSON(w, http.StatusCreated, map[string]any{"identity": response.Identity, "field": response.Field, "configured": true})
}

func (s *Server) credentialDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !webConsoleSameOrigin(w, r) {
		return
	}
	if s.credentialClient == nil {
		http.Error(w, "credential authority unavailable", http.StatusServiceUnavailable)
		return
	}
	var request credentialProvisionRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&request); err != nil || strings.TrimSpace(request.Identity) == "" || strings.TrimSpace(request.Field) == "" {
		http.Error(w, "identity and field are required", http.StatusBadRequest)
		return
	}
	if err := s.credentialClient.Delete(r.Context(), strings.TrimSpace(request.Identity), strings.TrimSpace(request.Field)); err != nil {
		http.Error(w, "credential could not be removed", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) credentialTestHandler(w http.ResponseWriter, r *http.Request) {
	if !webConsoleSameOrigin(w, r) {
		return
	}
	if s == nil || s.credentialClient == nil {
		http.Error(w, "credential authority unavailable", http.StatusServiceUnavailable)
		return
	}
	key, err := s.credentialClient.Resolve(r.Context(), webConsoleOpenRouterIdentity, webConsoleOpenRouterField)
	if err != nil || strings.TrimSpace(key) == "" {
		writeCredentialJSON(w, http.StatusNotFound, map[string]any{"valid": false, "source": "none", "checked_at": time.Now().UTC()})
		return
	}
	requestCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		http.Error(w, "could not test OpenRouter key", http.StatusBadGateway)
		return
	}
	request.Header.Set("Authorization", "Bearer "+key)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		writeCredentialJSON(w, http.StatusBadGateway, map[string]any{"valid": false, "source": "credential_authority", "checked_at": time.Now().UTC()})
		return
	}
	defer response.Body.Close()
	writeCredentialJSON(w, http.StatusOK, map[string]any{"valid": response.StatusCode >= 200 && response.StatusCode < 300, "source": "credential_authority", "checked_at": time.Now().UTC()})
}

func webConsoleSameOrigin(w http.ResponseWriter, r *http.Request) bool {
	// Loopback is necessary for the internal probe, but an explicit browser
	// Origin must still match the local host to prevent local cross-origin use.
	if rawOrigin := strings.TrimSpace(r.Header.Get("Origin")); rawOrigin != "" {
		origin, err := url.Parse(rawOrigin)
		if err != nil || origin.Host == "" || origin.Host != strings.TrimSpace(r.Host) {
			http.Error(w, "same-origin credential request required", http.StatusForbidden)
			return false
		}
	}
	peer := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(peer); err == nil {
		peer = host
	} else {
		peer = strings.Trim(peer, "[]")
	}
	if ip := net.ParseIP(peer); ip != nil && ip.IsLoopback() {
		return true
	}
	if strings.TrimSpace(r.Header.Get("Origin")) == "" {
		http.Error(w, "same-origin credential request required", http.StatusForbidden)
		return false
	}
	return true
}

func writeCredentialJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (s *Server) subscriptionSessionModule() *monetization.SessionModule {
	if s == nil {
		return monetization.NewSessionModule(nil, webConsoleSubscriptionIdentity, webConsoleSubscriptionField)
	}
	return monetization.NewSessionModule(s.credentialClient, webConsoleSubscriptionIdentity, webConsoleSubscriptionField)
}

func (s *Server) subscriptionSummaryHandler(w http.ResponseWriter, r *http.Request) {
	if !webConsoleSameOrigin(w, r) {
		return
	}
	if s == nil || s.subscriptionResolver == nil || s.entitlements == nil {
		http.Error(w, "subscription authority unavailable", http.StatusServiceUnavailable)
		return
	}
	access, err := s.subscriptionResolver.Resolve(r.Context())
	if err != nil {
		http.Error(w, "subscription session is not configured", http.StatusUnauthorized)
		return
	}
	identity, err := resolveLPBSIdentity(r.Context(), s.subscriptionResolver.LPBSBaseURL, access.AccessToken)
	if err != nil {
		http.Error(w, "subscription identity unavailable", http.StatusServiceUnavailable)
		return
	}
	lease, err := s.entitlements.GetWithAccess(r.Context(), identity, access.AccessToken)
	if err != nil {
		http.Error(w, "entitlement lease unavailable", http.StatusServiceUnavailable)
		return
	}
	writeCredentialJSON(w, http.StatusOK, map[string]any{
		"configured": true, "status": lease.Status, "plan_tier": lease.PlanTier,
		"credits": lease.Credits, "pending_sync": 0, "not_after": lease.NotAfter,
	})
}
