package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/operatorcapability"
	"github.com/vrooli/vrooli/internal/operatorinput"
)

const capabilityActionTimeout = 30 * time.Second

func (s *Server) handleV2Capabilities(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), capabilityActionTimeout)
	defer cancel()
	output, err := (controlPlaneExecutor{}).runNamedWithInput(ctx, nil, "vrooli", "capability", "status", "--json")
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "capability status is unavailable"})
		return
	}
	var statuses []operatorcapability.Status
	if err := json.Unmarshal(output, &statuses); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "capability status returned invalid metadata"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"capabilities": statuses, "count": len(statuses)})
}

func (s *Server) handleV2CapabilityPreview(w http.ResponseWriter, r *http.Request) {
	var request operatorcapability.ActionRequest
	if !decodeJSONBody(w, r, &request) {
		return
	}
	if err := validateCapabilityRequest(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), capabilityActionTimeout)
	defer cancel()
	preview, err := (controlPlaneExecutor{}).previewCapability(ctx, request)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "capability preview failed"})
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (s *Server) handleV2CapabilityApply(w http.ResponseWriter, r *http.Request) {
	var request operatorcapability.ActionRequest
	if !decodeJSONBody(w, r, &request) {
		return
	}
	if err := validateCapabilityRequest(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if !request.Confirm {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "confirm=true is required for capability apply"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), capabilityActionTimeout)
	defer cancel()
	result, err := (controlPlaneExecutor{}).applyCapability(ctx, request)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, result)
		return
	}
	status := http.StatusOK
	if result.State == operatorcapability.StateRetryableFailure || result.State == operatorcapability.StateDegraded {
		status = http.StatusUnprocessableEntity
	} else if result.State == operatorcapability.StateReady {
		if err := operatorinput.RemoveCapability(request.CapabilityID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "capability applied but pending operator metadata could not be reconciled"})
			return
		}
	}
	writeJSON(w, status, result)
}

func validateCapabilityRequest(request *operatorcapability.ActionRequest) error {
	if request == nil {
		return fmt.Errorf("capability action is required")
	}
	if strings.TrimSpace(request.CapabilityID) == "" {
		return fmt.Errorf("capability_id is required")
	}
	if request.IdempotencyKey == "" {
		request.IdempotencyKey = operatorcapability.StableIdempotencyKey(request.CapabilityID, request.Inputs)
	}
	return request.Validate()
}
