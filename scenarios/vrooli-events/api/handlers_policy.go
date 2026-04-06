package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

const (
	ErrCodeNotFound      = "NOT_FOUND"
	ErrCodePolicyWrite   = "POLICY_WRITE_ERROR"
	ErrCodePolicyRead    = "POLICY_READ_ERROR"
	ErrCodeValidation    = "VALIDATION_ERROR"
	ErrCodeViolationRead = "VIOLATION_READ_ERROR"
)

// handleCreatePolicy creates a new policy rule.
// DOC: docs/reference/api-endpoints.md#policy-crud
func (s *Server) handleCreatePolicy(w http.ResponseWriter, r *http.Request) {
	var rule policy.Rule
	if !decodeJSONBody(w, r, &rule) {
		return
	}

	if ve := validatePolicyRule(&rule); ve != "" {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, ve)
		return
	}

	id, err := s.policyStore.CreateRule(r.Context(), rule)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodePolicyWrite, "failed to create policy rule")
		log.Printf("policy create error: %v", err)
		return
	}

	rule.ID = id
	s.broadcastPolicyChange("created", id, &rule)

	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

// handleListPolicies lists policy rules with optional filters.
func (s *Server) handleListPolicies(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filters := policy.ListFilters{
		RuleType: policy.RuleType(q.Get("rule_type")),
		Source:   q.Get("source"),
		Target:   q.Get("target"),
	}
	if enabledStr := q.Get("enabled"); enabledStr != "" {
		enabled := enabledStr == "true"
		filters.Enabled = &enabled
	}

	rules, err := s.policyStore.ListRules(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodePolicyRead, "failed to list policy rules")
		log.Printf("policy list error: %v", err)
		return
	}

	if rules == nil {
		rules = []policy.Rule{}
	}
	writeJSON(w, 0, rules)
}

// handleGetPolicy returns a single policy rule by ID.
func (s *Server) handleGetPolicy(w http.ResponseWriter, r *http.Request) {
	_, rule, ok := requireByID(w, r, "id", s.policyStore.GetRule, ErrCodePolicyRead, "policy rule")
	if !ok {
		return
	}
	writeJSON(w, 0, rule)
}

// handleUpdatePolicy updates an existing policy rule.
func (s *Server) handleUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	id, _, ok := requireByID(w, r, "id", s.policyStore.GetRule, ErrCodePolicyRead, "policy rule")
	if !ok {
		return
	}

	var rule policy.Rule
	if !decodeJSONBody(w, r, &rule) {
		return
	}

	rule.ID = id
	if ve := validatePolicyRule(&rule); ve != "" {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, ve)
		return
	}

	if err := s.policyStore.UpdateRule(r.Context(), rule); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodePolicyWrite, "failed to update policy rule")
		log.Printf("policy update error: %v", err)
		return
	}

	s.broadcastPolicyChange("updated", id, &rule)

	writeJSON(w, 0, map[string]string{"status": "updated"})
}

// handleDeletePolicy deletes a policy rule by ID.
func (s *Server) handleDeletePolicy(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidParam, "invalid policy ID")
		return
	}

	if err := s.policyStore.DeleteRule(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodePolicyWrite, "failed to delete policy rule")
		log.Printf("policy delete error: %v", err)
		return
	}

	s.broadcastPolicyChange("deleted", id, nil)

	w.WriteHeader(http.StatusNoContent)
}

// handleListViolations lists policy violations with optional filters.
func (s *Server) handleListViolations(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filters := policy.ViolationFilters{
		Source:   q.Get("source"),
		Target:   q.Get("target"),
		RuleType: policy.RuleType(q.Get("rule_type")),
		Since:    q.Get("since"),
		Until:    q.Get("until"),
	}
	if limitStr := q.Get("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil {
			filters.Limit = n
		}
	}

	violations, err := s.policyStore.ListViolations(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeViolationRead, "failed to list violations")
		log.Printf("violations list error: %v", err)
		return
	}

	if violations == nil {
		violations = []policy.Violation{}
	}
	writeJSON(w, 0, violations)
}

// handleEvaluatePolicy evaluates a policy for a given request context.
func (s *Server) handleEvaluatePolicy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source   string `json:"source"`
		Target   string `json:"target"`
		Endpoint string `json:"endpoint"`
	}
	if !decodeJSONBody(w, r, &req) {
		return
	}
	if req.Source == "" || req.Target == "" {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, "source and target are required")
		return
	}

	decision := s.policyEval.Evaluate(r.Context(), policy.EvalRequest{
		Source:   req.Source,
		Target:   req.Target,
		Endpoint: req.Endpoint,
	})

	// Log violation if denied
	if !decision.Allowed {
		_ = s.policyStore.LogViolation(r.Context(), policy.Violation{
			SourceScenario: req.Source,
			TargetScenario: req.Target,
			Endpoint:       req.Endpoint,
			RuleID:         decision.RuleID,
			RuleType:       decision.RuleType,
			Reason:         decision.Reason,
		})
	}

	writeJSON(w, 0, decision)
}

// handleOverrideCircuitBreaker manually overrides a circuit breaker's state.
// DOC: docs/reference/api-endpoints.md#circuit-breaker-override
func (s *Server) handleOverrideCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	id, rule, ok := requireByID(w, r, "id", s.policyStore.GetRule, ErrCodePolicyRead, "policy rule")
	if !ok {
		return
	}
	if rule.RuleType != policy.RuleTypeCircuitBreaker {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, "only circuit_breaker rules can be overridden")
		return
	}

	var req struct {
		State      string `json:"state"`
		TTLSeconds int    `json:"ttl_seconds"`
	}
	if !decodeJSONBody(w, r, &req) {
		return
	}

	state := policy.CircuitState(req.State)
	switch state {
	case policy.CircuitOpen, policy.CircuitClosed, policy.CircuitHalfOpen:
		// valid
	default:
		writeError(w, http.StatusBadRequest, ErrCodeValidation, "state must be one of: open, closed, half_open")
		return
	}

	ttl := req.TTLSeconds
	if ttl <= 0 {
		ttl = 3600 // default 1 hour
	}

	if err := s.policyStore.SetCircuitBreakerOverride(r.Context(), id, state, ttl); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodePolicyWrite, "failed to set override")
		log.Printf("circuit breaker override error: %v", err)
		return
	}

	writeJSON(w, 0, map[string]any{
		"status":      "overridden",
		"rule_id":     id,
		"state":       req.State,
		"ttl_seconds": ttl,
	})
}

// parsePathID extracts a numeric ID from the URL path.
// Go 1.22+ ServeMux patterns like "/api/v1/policies/{id}" put the value
// in r.PathValue("id").
func parsePathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

// handlePolicySubscribe streams policy change events to SSE clients.
// DOC: docs/reference/api-endpoints.md#policy-subscribe
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
		case evt, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(evt)
			if err != nil {
				log.Printf("policy SSE marshal error: %v", err)
				continue
			}
			fmt.Fprintf(w, "event: policy_change\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// broadcastPolicyChange sends a policy change event if the broadcaster is configured.
func (s *Server) broadcastPolicyChange(action string, ruleID int64, rule *policy.Rule) {
	if s.policyBroadcaster != nil {
		s.policyBroadcaster.Broadcast(policy.PolicyEvent{
			Action: action,
			RuleID: ruleID,
			Rule:   rule,
		})
	}
}

// validatePolicyRule checks required fields on a policy rule.
// Common fields are validated first, then type-specific rules are checked.
func validatePolicyRule(r *policy.Rule) string {
	// Common required fields
	if r.RuleType == "" {
		return "rule_type is required"
	}
	if r.SourceScenario == "" {
		return "source_scenario is required"
	}
	if r.TargetScenario == "" {
		return "target_scenario is required"
	}

	// Type-specific validation
	switch r.RuleType {
	case policy.RuleTypeAccess:
		return validateAccessRule(r)
	case policy.RuleTypeRateLimit:
		return validateRateLimitRule(r)
	case policy.RuleTypeCircuitBreaker:
		return validateCircuitBreakerRule(r)
	default:
		return "rule_type must be one of: access, rate_limit, circuit_breaker"
	}
}

func validateAccessRule(r *policy.Rule) string {
	if r.Effect == "" {
		return "effect is required for access rules"
	}
	if r.Effect != policy.EffectAllow && r.Effect != policy.EffectDeny {
		return "effect must be 'allow' or 'deny'"
	}
	return ""
}

func validateRateLimitRule(r *policy.Rule) string {
	if r.MaxRequests <= 0 {
		return "max_requests must be > 0 for rate_limit rules"
	}
	if r.WindowSeconds <= 0 {
		return "window_seconds must be > 0 for rate_limit rules"
	}
	return ""
}

func validateCircuitBreakerRule(r *policy.Rule) string {
	if r.FailureThreshold <= 0 {
		return "failure_threshold must be > 0 for circuit_breaker rules"
	}
	return ""
}
