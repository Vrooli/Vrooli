package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

const (
	ErrCodeNotFound    = "NOT_FOUND"
	ErrCodePolicyWrite = "POLICY_WRITE_ERROR"
	ErrCodePolicyRead  = "POLICY_READ_ERROR"
	ErrCodeValidation  = "VALIDATION_ERROR"
)

// handlePolicySnapshot returns one complete, versioned capture-policy
// generation. API Core replaces this value atomically and never evaluates
// request traffic through this endpoint.
func (s *Server) handlePolicySnapshot(w http.ResponseWriter, r *http.Request) {
	enabled := true
	rules, err := s.policyStore.ListReceiptProjections(r.Context(), policy.ReceiptProjectionFilters{Enabled: &enabled})
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodePolicyRead, "failed to load receipt capture policy snapshot")
		return
	}
	version := fmt.Sprintf("policy-v%d", s.policyVersion.Load())
	writeJSON(w, http.StatusOK, map[string]any{"version": version, "receipt_capture_policies": capturePolicies(rules, version)})
}

func capturePolicies(rules []policy.ReceiptProjectionRule, version string) []map[string]any {
	result := make([]map[string]any, 0, len(rules))
	for _, rule := range rules {
		result = append(result, map[string]any{"policy_id": rule.PolicyID, "enabled": rule.Enabled, "selector": map[string]any{"target_scenario": rule.TargetScenario, "operation": rule.OperationPattern, "protocol": rule.Protocol, "event_type": rule.EventType}, "response_type": rule.ResponseType, "response_projection_paths": rule.ResponseFields, "retention_days": rule.RetentionDays, "access": map[string]any{"read_principals": rule.ReadPrincipals}, "version": version})
	}
	return result
}

func parsePathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

func (s *Server) handlePolicySubscribe(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, ErrCodeSSEUnsupported, "streaming not supported")
		return
	}
	id, ch := s.policyBroadcaster.Subscribe()
	defer s.policyBroadcaster.Unsubscribe(id)
	writeSSEHeaders(w, flusher, s.config.SSERetryMs)
	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("policy SSE marshal error: %v", err)
				continue
			}
			fmt.Fprintf(w, "event: snapshot\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (s *Server) broadcastPolicySnapshot(ctx context.Context) {
	if s.policyBroadcaster == nil {
		return
	}
	enabled := true
	rules, err := s.policyStore.ListReceiptProjections(ctx, policy.ReceiptProjectionFilters{Enabled: &enabled})
	if err != nil {
		log.Printf("receipt capture policy snapshot broadcast error: %v", err)
		return
	}
	s.policyBroadcaster.BroadcastSnapshot(s.policyVersion.Add(1), nil, orEmpty(rules))
}
