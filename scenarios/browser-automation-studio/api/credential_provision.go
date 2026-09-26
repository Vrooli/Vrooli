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

func requireSessionOrigin(w http.ResponseWriter, r *http.Request) bool {
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
		writeJSON(w, http.StatusOK, map[string]any{"identity": response.Identity, "field": response.Field, "configured": true})
	}
}
