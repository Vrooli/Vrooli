package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (s *Server) handleV2Credentials(w http.ResponseWriter, _ *http.Request) {
	root, err := manifestRoot()
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	models, err := loadScenarioReadModels()
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	closure, err := resolveClosure(root, models)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	credentials := make([]credentialReadiness, 0)
	for _, member := range closure.Scenarios {
		items, loadErr := loadScenarioCredentialReadiness(member.Name)
		if loadErr != nil {
			if writeCatalogDegraded(w, loadErr) {
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": loadErr.Error()})
			return
		}
		credentials = append(credentials, items...)
	}
	for _, member := range closure.Resources {
		items, loadErr := loadCredentialReadiness(member.Name)
		if loadErr != nil {
			if writeCatalogDegraded(w, loadErr) {
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": loadErr.Error()})
			return
		}
		credentials = append(credentials, items...)
	}
	projectCredentials, projectErr := projectCredentialReadiness()
	if projectErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": projectErr.Error()})
		return
	}
	credentials = append(credentials, projectCredentials...)
	sortCredentialReadiness(credentials)
	writeJSON(w, http.StatusOK, map[string]any{"credentials": credentials, "count": len(credentials)})
}

var credentialDoctorCommand = func(ctx context.Context) ([]byte, error) {
	return onboardingDoctorJSON(ctx)
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
	// One probe per request, never one per credential: the cached verdict is
	// what makes a long-lived onboarding process report a store state it
	// observed hours ago.
	recheckCredentialAuthority()
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
