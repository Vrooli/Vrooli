package monetization

import (
	"context"
	"encoding/json"
	"errors"
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
	OnDelete func(context.Context) error
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
	if err := m.Remove(r.Context()); err != nil {
		http.Error(w, "subscription session could not be removed", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Remove deletes the durable refresh credential without writing an HTTP
// response. Callers can clear process-local entitlement caches before
// acknowledging sign-out.
func (m *SessionModule) Remove(ctx context.Context) error {
	if m == nil || m.Client == nil {
		return errors.New("credential authority unavailable")
	}
	if err := m.Client.Delete(ctx, m.Identity, m.Field); err != nil {
		return err
	}
	if m.OnDelete != nil {
		return m.OnDelete(ctx)
	}
	return nil
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
