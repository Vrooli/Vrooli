package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

var credentialDoctorRelay = func(ctx context.Context) ([]byte, error) {
	return secretsDoctorJSON(ctx)
}

var credentialKeyringRelay = func(ctx context.Context, action string) ([]byte, error) {
	return secretsKeyringJSON(ctx, action)
}

func (h *CredentialHandlers) Doctor(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	output, err := credentialDoctorRelay(ctx)
	if err != nil {
		http.Error(w, "credential diagnosis is unavailable", http.StatusServiceUnavailable)
		return
	}
	writeCredentialJSON(w, output)
}

func (h *CredentialHandlers) KeyringInspect(w http.ResponseWriter, r *http.Request) {
	h.relayKeyring(w, r, "inspect")
}

func (h *CredentialHandlers) KeyringRepair(w http.ResponseWriter, r *http.Request) {
	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "confirm=true is required to repair the keyring", http.StatusBadRequest)
		return
	}
	var confirmed bool
	raw, ok := body["confirm"]
	if !ok || json.Unmarshal(raw, &confirmed) != nil || !confirmed {
		http.Error(w, "confirm=true is required to repair the keyring", http.StatusBadRequest)
		return
	}
	h.relayKeyring(w, r, "repair")
}

func (h *CredentialHandlers) relayKeyring(w http.ResponseWriter, r *http.Request, action string) {
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	output, err := credentialKeyringRelay(ctx, action)
	if err != nil {
		http.Error(w, "keyring diagnostic is unavailable", http.StatusServiceUnavailable)
		return
	}
	writeCredentialJSON(w, output)
}

func writeCredentialJSON(w http.ResponseWriter, output []byte) {
	if !json.Valid(output) {
		http.Error(w, "credential diagnostic returned invalid data", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(strings.TrimSpace(string(output))))
}
