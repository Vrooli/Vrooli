package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type credentialProvisionRequest struct {
	LogicalID string `json:"logical_id"`
	Field     string `json:"field"`
	Value     string `json:"value"`
}

var credentialProvisionCommand = func(ctx context.Context, logicalID, field, value string) error {
	command := exec.CommandContext(ctx, "vrooli", "credentials", "provision", "--identity", logicalID, "--field", field)
	command.Stdin = strings.NewReader(value)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
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
