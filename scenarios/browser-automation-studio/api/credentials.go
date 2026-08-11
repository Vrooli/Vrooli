package main

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

type credentialProvisionRequest struct {
	Identity string `json:"identity"`
	Field    string `json:"field"`
	Value    string `json:"value"`
}

const (
	lpbsAccountIdentity = "vrooli/lpbs-account"
	lpbsAccountField    = "refresh-token"
)

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// requireSessionOrigin prevents a remote page from silently replacing the
// shared subscription credential. Loopback development requests are allowed;
// remote deployments must present a same-origin browser request.
func requireSessionOrigin(w http.ResponseWriter, r *http.Request) bool {
	// Use the peer address, never the Host header, for the loopback exception.
	// Host is caller-controlled and accepting `Host: 127.0.0.1` would let a
	// remote browser bypass the session replacement CSRF boundary.
	peer := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(peer); err == nil {
		peer = host
	} else {
		peer = strings.Trim(peer, "[]")
	}
	if ip := net.ParseIP(peer); ip != nil && ip.IsLoopback() {
		return true
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		http.Error(w, "same-origin session request required", http.StatusForbidden)
		return false
	}
	// Do not trust X-Forwarded-Host from an arbitrary caller. A deployment
	// terminating TLS at a proxy must preserve the original Host header; if it
	// cannot, it must establish a trusted same-origin policy before forwarding.
	trustedHost := strings.TrimSpace(r.Host)
	if parsed.Host != trustedHost {
		http.Error(w, "same-origin session request required", http.StatusForbidden)
		return false
	}
	return true
}

// subscriptionSessionHandler is the device/token-paste seam. It accepts only
// the rotating refresh credential and never returns it or stores an access
// token in the scenario process.
func subscriptionSessionHandler(client credentialclient.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !requireSessionOrigin(w, r) {
			return
		}
		if client == nil {
			http.Error(w, "credential authority unavailable", http.StatusServiceUnavailable)
			return
		}
		var request struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil || strings.TrimSpace(request.RefreshToken) == "" {
			http.Error(w, "refresh_token is required", http.StatusBadRequest)
			return
		}
		response, err := client.Provision(r.Context(), credentialclient.ProvisionRequest{Identity: lpbsAccountIdentity, Field: lpbsAccountField, Value: request.RefreshToken})
		if err != nil {
			http.Error(w, "subscription session could not be stored", http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"identity": response.Identity, "field": response.Field, "configured": true})
	}
}

func subscriptionSessionStatusHandler(client credentialclient.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !requireSessionOrigin(w, r) {
			return
		}
		if client == nil {
			http.Error(w, "credential authority unavailable", http.StatusServiceUnavailable)
			return
		}
		status, err := client.Status(r.Context(), lpbsAccountIdentity, lpbsAccountField)
		if err != nil {
			http.Error(w, "subscription session status unavailable", http.StatusServiceUnavailable)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"identity": lpbsAccountIdentity, "field": lpbsAccountField, "configured": status.Configured, "provider_state": status.ProviderState, "provider_detail": status.ProviderDetail})
	}
}

func subscriptionSessionDeleteHandler(client credentialclient.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !requireSessionOrigin(w, r) {
			return
		}
		if client == nil {
			http.Error(w, "credential authority unavailable", http.StatusServiceUnavailable)
			return
		}
		if err := client.Delete(r.Context(), lpbsAccountIdentity, lpbsAccountField); err != nil {
			http.Error(w, "subscription session could not be removed", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// credentialProvisionHandler permits only credentials declared by this
// scenario and never includes a credential value in its response.
func credentialProvisionHandler(client credentialclient.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !requireSessionOrigin(w, r) {
			return
		}
		if client == nil {
			http.Error(w, "credential authority unavailable", http.StatusServiceUnavailable)
			return
		}
		var request credentialProvisionRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil {
			http.Error(w, "invalid credential request", http.StatusBadRequest)
			return
		}
		request.Identity = strings.TrimSpace(request.Identity)
		request.Field = strings.TrimSpace(request.Field)
		if request.Identity == "" || request.Field == "" || strings.TrimSpace(request.Value) == "" {
			http.Error(w, "identity, field, and value are required", http.StatusBadRequest)
			return
		}
		refs, err := client.List(r.Context())
		if err != nil {
			http.Error(w, "credential declarations unavailable", http.StatusInternalServerError)
			return
		}
		declared := false
		for _, ref := range refs {
			if ref.LogicalID == request.Identity && ref.Field == request.Field {
				declared = true
				break
			}
		}
		if !declared {
			http.Error(w, "credential is not declared by this scenario", http.StatusForbidden)
			return
		}
		response, err := client.Provision(r.Context(), credentialclient.ProvisionRequest{Identity: request.Identity, Field: request.Field, Value: request.Value})
		if err != nil {
			http.Error(w, "credential provisioning failed", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Identity   string `json:"identity"`
			Field      string `json:"field"`
			Configured bool   `json:"configured"`
		}{Identity: response.Identity, Field: response.Field, Configured: true})
	}
}
