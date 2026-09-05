package monetization

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

const sessionRequestLimit = 16 << 10

// SessionModule mounts the shared device/token-paste subscription session
// endpoints. It stores only the refresh credential through credential
// authority and never returns the credential value.
type SessionModule struct {
	Client   credentialclient.Client
	Identity string
	Field    string
}

func NewSessionModule(client credentialclient.Client, identity, field string) *SessionModule {
	return &SessionModule{Client: client, Identity: strings.TrimSpace(identity), Field: strings.TrimSpace(field)}
}

func (m *SessionModule) Provision(w http.ResponseWriter, r *http.Request) {
	if !sessionOriginOK(w, r) {
		return
	}
	if m == nil || m.Client == nil {
		http.Error(w, "credential authority unavailable", http.StatusServiceUnavailable)
		return
	}
	var request struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, sessionRequestLimit)).Decode(&request); err != nil || strings.TrimSpace(request.RefreshToken) == "" {
		http.Error(w, "refresh_token is required", http.StatusBadRequest)
		return
	}
	response, err := m.Client.Provision(r.Context(), credentialclient.ProvisionRequest{Identity: m.Identity, Field: m.Field, Value: request.RefreshToken})
	if err != nil {
		http.Error(w, "subscription session could not be stored", http.StatusBadGateway)
		return
	}
	writeSessionJSON(w, http.StatusCreated, map[string]any{"identity": response.Identity, "field": response.Field, "configured": true})
}

func (m *SessionModule) Status(w http.ResponseWriter, r *http.Request) {
	if !sessionOriginOK(w, r) {
		return
	}
	if m == nil || m.Client == nil {
		http.Error(w, "credential authority unavailable", http.StatusServiceUnavailable)
		return
	}
	status, err := m.Client.Status(r.Context(), m.Identity, m.Field)
	if err != nil {
		http.Error(w, "subscription session status unavailable", http.StatusServiceUnavailable)
		return
	}
	writeSessionJSON(w, http.StatusOK, map[string]any{"identity": m.Identity, "field": m.Field, "configured": status.Configured, "provider_state": status.ProviderState, "provider_detail": status.ProviderDetail})
}

func (m *SessionModule) Delete(w http.ResponseWriter, r *http.Request) {
	if !sessionOriginOK(w, r) {
		return
	}
	if m == nil || m.Client == nil {
		http.Error(w, "credential authority unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := m.Client.Delete(r.Context(), m.Identity, m.Field); err != nil {
		http.Error(w, "subscription session could not be removed", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func sessionOriginOK(w http.ResponseWriter, r *http.Request) bool {
	peer := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(peer); err == nil {
		peer = host
	} else {
		peer = strings.Trim(peer, "[]")
	}
	if ip := net.ParseIP(peer); ip != nil && ip.IsLoopback() {
		return true
	}
	origin, err := url.Parse(strings.TrimSpace(r.Header.Get("Origin")))
	if err != nil || origin.Host == "" || origin.Host != strings.TrimSpace(r.Host) {
		http.Error(w, "same-origin session request required", http.StatusForbidden)
		return false
	}
	return true
}

func writeSessionJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// ConsumerSession is the shared paid-surface session adapter. It keeps the
// refresh credential in credential authority and exposes only the short-lived
// access token needed by LPBS and entitlementclient.
type ConsumerSession struct {
	Resolver  *credentialclient.ConsumerSessionResolver
	Authority string
}

func NewConsumerSession(resolver *credentialclient.ConsumerSessionResolver, authority string) *ConsumerSession {
	return &ConsumerSession{Resolver: resolver, Authority: strings.TrimRight(strings.TrimSpace(authority), "/")}
}

func (s *ConsumerSession) Resolve(ctx context.Context) (credentialclient.ConsumerAccess, error) {
	if s == nil || s.Resolver == nil {
		return credentialclient.ConsumerAccess{}, credentialclient.ErrCredentialAuthorityUnavailable
	}
	return s.Resolver.ResolveAt(ctx, s.Authority)
}

func (s *ConsumerSession) Clear() {
	if s != nil && s.Resolver != nil {
		s.Resolver.Clear()
	}
}
