package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var credentialDoctorCommand = func(ctx context.Context) ([]byte, error) {
	// --check-writes is deliberate here: the wizard's next action is a
	// provisioning write, so "can this host store a credential" is the actual
	// question. Routine health reads must not pass it — the probe writes to the
	// operator's real store.
	return onboardingDoctorJSON(ctx)
}

var credentialKeyringCommand = func(ctx context.Context, action string) ([]byte, error) {
	return onboardingKeyringJSON(ctx, action)
}

type credentialProvisionRequest struct {
	LogicalID string `json:"logical_id"`
	Field     string `json:"field"`
	Value     string `json:"value"`
}

var credentialProvisionCommand = func(ctx context.Context, logicalID, field, value string) error {
	if err := onboardingProvision(ctx, logicalID, field, value); err != nil {
		return fmt.Errorf("credential authority rejected provisioning: %w", err)
	}
	return nil
}

// handleV2CredentialProvision is an ephemeral control-plane relay. The value
// is accepted only in the request body, forwarded over stdin, and never added
// to command arguments, responses, logs, or operator state.
func (s *Server) handleV2CredentialProvision(w http.ResponseWriter, r *http.Request) {
	var request credentialProvisionRequest
	if !decodeJSONBody(w, r, &request) {
		return
	}
	request.LogicalID = strings.TrimSpace(request.LogicalID)
	request.Field = strings.TrimSpace(request.Field)
	if request.Field == "" {
		request.Field = "value"
	}
	if request.LogicalID == "" || strings.TrimSpace(request.Value) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "logical_id and a non-empty credential value are required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if err := credentialProvisionCommand(ctx, request.LogicalID, request.Field, request.Value); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "native credential authority could not provision this credential"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "provisioned", "logical_id": request.LogicalID, "field": request.Field})
}

func (s *Server) handleV2CredentialDoctor(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	output, err := credentialDoctorCommand(ctx)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credential diagnosis is unavailable"})
		return
	}
	var payload json.RawMessage
	if err := json.Unmarshal(output, &payload); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credential diagnosis returned invalid data"})
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleV2CredentialKeyringInspect(w http.ResponseWriter, r *http.Request) {
	s.relayCredentialKeyring(w, r, "inspect", false)
}

func (s *Server) handleV2CredentialKeyringRepair(w http.ResponseWriter, r *http.Request) {
	var request map[string]json.RawMessage
	if !decodeJSONBody(w, r, &request) {
		return
	}
	confirmed, ok := request["confirm"]
	var confirm bool
	if !ok || json.Unmarshal(confirmed, &confirm) != nil || !confirm {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "confirm=true is required to repair the keyring"})
		return
	}
	s.relayCredentialKeyring(w, r, "repair", true)
}

func (s *Server) relayCredentialKeyring(w http.ResponseWriter, r *http.Request, action string, _ bool) {
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	output, err := credentialKeyringCommand(ctx, action)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "keyring diagnostic is unavailable"})
		return
	}
	var payload json.RawMessage
	if err := json.Unmarshal(output, &payload); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "keyring diagnostic returned invalid data"})
		return
	}
	writeJSON(w, http.StatusOK, payload)
}
