package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/vrooli/vrooli/internal/operatorcapability"
	"github.com/vrooli/vrooli/internal/operatorinput"
)

func (s *Server) handleV2OperatorInputs(w http.ResponseWriter, r *http.Request) {
	queue, err := operatorinput.Load()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	// The target is part of the transport contract. The durable queue remains
	// target-neutral; Bridge resolves the selected node when this request is
	// dispatched, so no node identity is persisted with an answer.
	if target := strings.TrimSpace(r.URL.Query().Get("target")); target != "" && target != "local" {
		w.Header().Set("X-Vrooli-Target", target)
	}
	writeJSON(w, http.StatusOK, queue)
}

func (s *Server) handleV2OperatorInputsResolve(w http.ResponseWriter, r *http.Request) {
	var answers []operatorinput.Answer
	if err := json.NewDecoder(r.Body).Decode(&answers); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "answers must be a JSON array: " + err.Error()})
		return
	}
	queue, err := operatorinput.Load()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if _, err := operatorinput.ResolveWith(answers, func(values map[string]string) error {
		requests, err := buildCapabilityRequests(queue, values)
		if err != nil {
			return err
		}
		defer clearCapabilityRequests(requests)
		for _, request := range requests {
			result, applyErr := (controlPlaneExecutor{}).applyCapability(r.Context(), request)
			if applyErr != nil {
				return fmt.Errorf("apply capability %q: %w", request.CapabilityID, applyErr)
			}
			if result.State != operatorcapability.StateReady {
				return fmt.Errorf("capability %q did not become ready: %s", request.CapabilityID, result.Remediation)
			}
		}
		return nil
	}); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "resolved", "configuration_pending": false})
}

func buildCapabilityRequests(queue operatorinput.Pending, values map[string]string) ([]operatorcapability.ActionRequest, error) {
	groups := map[string]*operatorcapability.ActionRequest{}
	for _, request := range queue.Requests {
		capabilityID := strings.TrimSpace(request.CapabilityID)
		inputID := strings.TrimSpace(request.InputID)
		if capabilityID == "" || inputID == "" {
			return nil, fmt.Errorf("operator input %q is missing generic capability metadata", request.ID)
		}
		value, ok := values[request.ID]
		if !ok {
			continue
		}
		group := groups[capabilityID]
		if group == nil {
			group = &operatorcapability.ActionRequest{CapabilityID: capabilityID, Confirm: false, Inputs: map[string]json.RawMessage{}}
			groups[capabilityID] = group
		}
		if inputID == "confirm" {
			group.Confirm = strings.EqualFold(strings.TrimSpace(value), "true")
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("encode operator input %q: %w", request.ID, err)
		}
		if request.Kind == operatorinput.KindBoolean || request.Kind == operatorinput.KindConfirmation {
			if value != "true" && value != "false" {
				return nil, fmt.Errorf("operator input %q must be boolean", request.ID)
			}
			encoded = json.RawMessage(value)
		}
		group.Inputs[inputID] = encoded
	}
	requests := make([]operatorcapability.ActionRequest, 0, len(groups))
	for _, request := range groups {
		request.IdempotencyKey = operatorcapability.StableIdempotencyKey(request.CapabilityID, request.Inputs)
		requests = append(requests, *request)
	}
	return requests, nil
}

func clearCapabilityRequests(requests []operatorcapability.ActionRequest) {
	for i := range requests {
		for key, value := range requests[i].Inputs {
			for index := range value {
				value[index] = 0
			}
			delete(requests[i].Inputs, key)
		}
	}
}
