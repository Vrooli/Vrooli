package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	webConsoleSubscriptionIdentity = credentialclient.SubscriptionIdentity
	webConsoleSubscriptionField    = credentialclient.SubscriptionField
	webConsoleOpenRouterIdentity   = "vrooli/openrouter"
	webConsoleOpenRouterField      = "api-key"
	webConsoleOpenRouterConnector  = "openrouter"
)

type credentialProvisionRequest struct {
	Identity string `json:"identity"`
	Field    string `json:"field"`
	Value    string `json:"value"`
}

type connectionMetadata struct {
	ID               string   `json:"id"`
	Provider         string   `json:"provider"`
	ConnectionName   string   `json:"connection_name"`
	Status           string   `json:"status"`
	Bindings         []string `json:"bindings,omitempty"`
	NextAction       string   `json:"next_action,omitempty"`
	SupportedActions []string `json:"supported_actions,omitempty"`
}

// connectionsHandler projects credential-authority status into the
// provider-neutral connected-account view. It never returns a credential
// value, fingerprint, or other derived secret material.
func (s *Server) connectionsHandler(w http.ResponseWriter, r *http.Request) {
	if !webConsoleSameOrigin(w, r) {
		return
	}
	if s == nil {
		http.Error(w, "connection authority unavailable", http.StatusServiceUnavailable)
		return
	}
	if strings.TrimSpace(s.integrationHubURL) != "" {
		connections, err := s.listHubConnections(r.Context(), r)
		if err != nil {
			http.Error(w, "connection status unavailable", http.StatusBadGateway)
			return
		}
		projected := make([]connectionMetadata, 0, len(connections))
		for _, connection := range connections {
			projected = append(projected, projectHubConnection(connection))
		}
		writeCredentialJSON(w, http.StatusOK, map[string]any{"connections": projected})
		return
	}
	if s.credentialClient == nil {
		http.Error(w, "connection authority unavailable", http.StatusServiceUnavailable)
		return
	}
	status, err := s.credentialClient.Status(r.Context(), webConsoleOpenRouterIdentity, webConsoleOpenRouterField)
	if err != nil {
		http.Error(w, "connection status unavailable", http.StatusBadGateway)
		return
	}
	if !status.Configured && status.ProviderState == "unavailable" {
		// Do not turn an authority outage into a misleading empty state, and do
		// not return provider diagnostics to the browser.
		http.Error(w, "connection status unavailable", http.StatusBadGateway)
		return
	}
	connections := []connectionMetadata{}
	if status.Configured {
		connections = append(connections, connectionMetadata{
			ID:               webConsoleOpenRouterIdentity,
			Provider:         "OpenRouter",
			ConnectionName:   "OpenRouter API credential",
			Status:           "connected",
			Bindings:         []string{"ai-command-generation"},
			NextAction:       "Manage this credential in Account settings.",
			SupportedActions: []string{"test", "delete"},
		})
	}
	writeCredentialJSON(w, http.StatusOK, map[string]any{"connections": connections})
}

func projectHubConnection(connection *commonv1.Connection) connectionMetadata {
	if connection == nil {
		return connectionMetadata{}
	}
	bindings := make([]string, 0, len(connection.GetBindings()))
	for _, binding := range connection.GetBindings() {
		if slug := strings.TrimSpace(binding.GetScenarioSlug()); slug != "" {
			bindings = append(bindings, slug)
		}
	}
	actions := make([]string, 0, len(connection.GetSupportedActions()))
	for _, action := range connection.GetSupportedActions() {
		actions = append(actions, strings.ToLower(strings.TrimPrefix(action.String(), "CONNECTION_ACTION_KIND_")))
	}
	return connectionMetadata{
		ID: connection.GetId(), Provider: connection.GetConnectorName(), ConnectionName: connection.GetDisplayName(),
		Status:   projectHubStatus(connection.GetStatus()),
		Bindings: bindings, NextAction: connection.GetNextAction(), SupportedActions: actions,
	}
}

func projectHubStatus(status commonv1.ConnectionStatus) string {
	switch status {
	case commonv1.ConnectionStatus_CONNECTION_STATUS_NEEDS_REAUTHORIZATION,
		commonv1.ConnectionStatus_CONNECTION_STATUS_EXPIRED,
		commonv1.ConnectionStatus_CONNECTION_STATUS_INSUFFICIENT_SCOPE,
		commonv1.ConnectionStatus_CONNECTION_STATUS_PROVIDER_OUTAGE,
		commonv1.ConnectionStatus_CONNECTION_STATUS_PROVIDER_UNAVAILABLE:
		return "needs_attention"
	default:
		return strings.ToLower(strings.TrimPrefix(status.String(), "CONNECTION_STATUS_"))
	}
}

func hubIdentityPresent(r *http.Request) bool {
	if r == nil {
		return false
	}
	return strings.TrimSpace(r.Header.Get("X-Vrooli-Identity")) != "" || strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Header.Get("Authorization"))), "bearer ")
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
	if strings.TrimSpace(s.integrationHubURL) != "" && request.Identity == webConsoleOpenRouterIdentity && request.Field == webConsoleOpenRouterField && hubIdentityPresent(r) {
		if err := s.createHubConnection(r.Context(), r, request.Value); err != nil {
			http.Error(w, "credential could not be stored", http.StatusBadGateway)
			return
		}
		writeCredentialJSON(w, http.StatusCreated, map[string]any{"identity": request.Identity, "field": request.Field, "configured": true})
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
	if strings.TrimSpace(s.integrationHubURL) != "" && request.Identity == webConsoleOpenRouterIdentity && request.Field == webConsoleOpenRouterField && hubIdentityPresent(r) {
		connections, err := s.listHubConnections(r.Context(), r)
		if err != nil {
			http.Error(w, "credential inventory unavailable", http.StatusBadGateway)
			return
		}
		for _, connection := range connections {
			if connection.GetConnectorId() == webConsoleOpenRouterConnector {
				if err := s.deleteHubConnection(r.Context(), r, connection.GetId()); err != nil {
					http.Error(w, "credential could not be removed", http.StatusBadGateway)
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		// A legacy credential may predate Integration Hub. Preserve the
		// existing declared-authority deletion path below until it is migrated.
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

// commercialContextHandler is a same-origin, optional projection for bundled
// consumers. LPBS remains the authority: Web Console forwards the caller's
// access token and returns only the typed, presentation-safe response.
func (s *Server) commercialContextHandler(w http.ResponseWriter, r *http.Request) {
	if !webConsoleSameOrigin(w, r) {
		return
	}
	if s == nil || s.subscriptionResolver == nil {
		http.Error(w, "commercial context unavailable", http.StatusServiceUnavailable)
		return
	}
	access, err := s.subscriptionResolver.Resolve(r.Context())
	if err != nil || strings.TrimSpace(access.AccessToken) == "" {
		http.Error(w, "commercial context requires an account", http.StatusUnauthorized)
		return
	}
	client := lpbsconnect.NewAccountServiceClient(http.DefaultClient, strings.TrimRight(s.subscriptionResolver.LPBSBaseURL, "/"), connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+access.AccessToken)
			return next(ctx, req)
		}
	})))
	var request lpbsv1.CommercialContextRequest
	if placement := strings.TrimSpace(r.URL.Query().Get("placement")); placement != "" {
		request.Placement = placement
	}
	if capability := strings.TrimSpace(r.URL.Query().Get("capability_id")); capability != "" {
		request.CapabilityId = capability
	}
	response, err := client.GetCommercialContext(r.Context(), connect.NewRequest(&request))
	if err != nil {
		http.Error(w, "commercial context unavailable", http.StatusBadGateway)
		return
	}
	body, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(response.Msg)
	if err != nil {
		http.Error(w, "commercial context could not be encoded", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, max-age=60, stale-while-revalidate=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
