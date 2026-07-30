package main

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

var descriptorPath = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$`)

// receiptCapturePolicy is the only public policy resource for API Core
// observation. It deliberately has no source selector: receipt eligibility is
// based on the receiving API operation and verified attribution is carried by
// the event, not guessed from a scenario name.
type receiptCapturePolicy struct {
	PolicyID string `json:"policy_id"`
	Enabled  bool   `json:"enabled"`
	Selector struct {
		TargetScenario, Operation, Protocol, EventType string `json:"-"`
	} `json:"-"`
	ResponseType            string   `json:"response_type"`
	ResponseProjectionPaths []string `json:"response_projection_paths"`
	RetentionDays           int      `json:"retention_days"`
	Access                  struct {
		ReadPrincipals []string `json:"read_principals"`
	} `json:"access"`
	Version string `json:"version,omitempty"`
}

func (p receiptCapturePolicy) MarshalJSON() ([]byte, error) {
	type selector struct {
		TargetScenario string `json:"target_scenario"`
		Operation      string `json:"operation"`
		Protocol       string `json:"protocol"`
		EventType      string `json:"event_type"`
	}
	return json.Marshal(struct {
		PolicyID                string   `json:"policy_id"`
		Enabled                 bool     `json:"enabled"`
		Selector                selector `json:"selector"`
		ResponseType            string   `json:"response_type"`
		ResponseProjectionPaths []string `json:"response_projection_paths"`
		RetentionDays           int      `json:"retention_days"`
		Access                  struct {
			ReadPrincipals []string `json:"read_principals"`
		} `json:"access"`
		Version string `json:"version,omitempty"`
	}{PolicyID: p.PolicyID, Enabled: p.Enabled, Selector: selector{p.Selector.TargetScenario, p.Selector.Operation, p.Selector.Protocol, p.Selector.EventType}, ResponseType: p.ResponseType, ResponseProjectionPaths: p.ResponseProjectionPaths, RetentionDays: p.RetentionDays, Access: struct {
		ReadPrincipals []string `json:"read_principals"`
	}{p.Access.ReadPrincipals}, Version: p.Version})
}

// UnmarshalJSON is intentionally explicit for the nested selector so the
// public wire shape exactly mirrors ReceiptCapturePolicy.
func (p *receiptCapturePolicy) UnmarshalJSON(data []byte) error {
	type wire struct {
		PolicyID string `json:"policy_id"`
		Enabled  bool   `json:"enabled"`
		Selector struct {
			TargetScenario string `json:"target_scenario"`
			Operation      string `json:"operation"`
			Protocol       string `json:"protocol"`
			EventType      string `json:"event_type"`
		} `json:"selector"`
		ResponseType            string   `json:"response_type"`
		ResponseProjectionPaths []string `json:"response_projection_paths"`
		RetentionDays           int      `json:"retention_days"`
		Access                  struct {
			ReadPrincipals []string `json:"read_principals"`
		} `json:"access"`
		Version string `json:"version"`
	}
	var value wire
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	p.PolicyID, p.Enabled, p.ResponseType, p.ResponseProjectionPaths, p.RetentionDays, p.Version = value.PolicyID, value.Enabled, value.ResponseType, value.ResponseProjectionPaths, value.RetentionDays, value.Version
	p.Selector.TargetScenario, p.Selector.Operation, p.Selector.Protocol, p.Selector.EventType = value.Selector.TargetScenario, value.Selector.Operation, value.Selector.Protocol, value.Selector.EventType
	p.Access.ReadPrincipals = value.Access.ReadPrincipals
	return nil
}

func (p receiptCapturePolicy) rule() policy.ReceiptProjectionRule {
	return policy.ReceiptProjectionRule{PolicyID: p.PolicyID, SourceScenario: "*", TargetScenario: p.Selector.TargetScenario, OperationPattern: p.Selector.Operation, Protocol: p.Selector.Protocol, EventType: p.Selector.EventType, ResponseType: p.ResponseType, ResponseFields: p.ResponseProjectionPaths, ReadPrincipals: p.Access.ReadPrincipals, MaxBytes: 64 * 1024, SamplePerTenK: 10000, RetentionDays: p.RetentionDays, Enabled: p.Enabled}
}
func capturePolicy(rule policy.ReceiptProjectionRule, version string) receiptCapturePolicy {
	var result receiptCapturePolicy
	result.PolicyID, result.Enabled, result.ResponseType, result.ResponseProjectionPaths, result.RetentionDays, result.Version = rule.PolicyID, rule.Enabled, rule.ResponseType, rule.ResponseFields, rule.RetentionDays, version
	result.Selector.TargetScenario, result.Selector.Operation, result.Selector.Protocol, result.Selector.EventType = rule.TargetScenario, rule.OperationPattern, rule.Protocol, rule.EventType
	result.Access.ReadPrincipals = rule.ReadPrincipals
	return result
}

func validateCapturePolicy(p receiptCapturePolicy) string {
	if strings.TrimSpace(p.PolicyID) == "" || strings.TrimSpace(p.Selector.TargetScenario) == "" || strings.TrimSpace(p.Selector.Operation) == "" || strings.TrimSpace(p.ResponseType) == "" {
		return "policy_id, selector target_scenario/operation, and response_type are required"
	}
	if p.Selector.Protocol != "connect" || p.Selector.EventType != receiptEventType {
		return "selector protocol must be connect and event_type must be vrooli.events.receipt.v1"
	}
	if p.RetentionDays <= 0 {
		return "retention_days must be > 0"
	}
	for _, path := range p.ResponseProjectionPaths {
		if !descriptorPath.MatchString(path) {
			return "response_projection_paths must be canonical descriptor paths"
		}
	}
	return ""
}

func (s *Server) handleCreateCapturePolicy(w http.ResponseWriter, r *http.Request) {
	var p receiptCapturePolicy
	if !decodeJSONBody(w, r, &p) {
		return
	}
	if message := validateCapturePolicy(p); message != "" {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, message)
		return
	}
	rule := p.rule()
	// A declaration reconcile may safely submit the same policy repeatedly.
	// policy_id is the declaration identity, so update in place instead of
	// accumulating duplicate rules with ambiguous matching precedence.
	existing, err := s.policyStore.ListReceiptProjections(r.Context(), policy.ReceiptProjectionFilters{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodePolicyRead, "failed to inspect receipt capture policies")
		return
	}
	for _, candidate := range existing {
		if candidate.PolicyID != rule.PolicyID {
			continue
		}
		rule.ID = candidate.ID
		if err := s.policyStore.UpdateReceiptProjection(r.Context(), rule); err != nil {
			writeError(w, http.StatusInternalServerError, ErrCodePolicyWrite, "failed to reconcile receipt capture policy")
			return
		}
		s.broadcastPolicySnapshot(r.Context())
		writeJSON(w, http.StatusOK, map[string]any{"id": rule.ID, "policy_id": p.PolicyID, "reconciled": true})
		return
	}
	id, err := s.policyStore.CreateReceiptProjection(r.Context(), rule)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodePolicyWrite, "failed to create receipt capture policy")
		return
	}
	s.broadcastPolicySnapshot(r.Context())
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "policy_id": p.PolicyID})
}
func (s *Server) handleListCapturePolicies(w http.ResponseWriter, r *http.Request) {
	enabled := true
	rules, err := s.policyStore.ListReceiptProjections(r.Context(), policy.ReceiptProjectionFilters{Enabled: &enabled})
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodePolicyRead, "failed to list receipt capture policies")
		return
	}
	version := "policy-v" + strconv.FormatInt(s.policyVersion.Load(), 10)
	result := make([]receiptCapturePolicy, 0, len(rules))
	for _, rule := range rules {
		result = append(result, capturePolicy(rule, version))
	}
	writeJSON(w, http.StatusOK, result)
}
