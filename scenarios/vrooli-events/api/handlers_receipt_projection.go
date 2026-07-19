package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

func (s *Server) handleCreateReceiptProjection(w http.ResponseWriter, r *http.Request) {
	var rule policy.ReceiptProjectionRule
	if !decodeJSONBody(w, r, &rule) {
		return
	}
	if message := validateReceiptProjectionRule(&rule); message != "" {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, message)
		return
	}
	id, err := s.policyStore.CreateReceiptProjection(r.Context(), rule)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodePolicyWrite, "failed to create receipt projection rule")
		return
	}
	s.broadcastPolicySnapshot(r.Context())
	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleListReceiptProjections(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filters := policy.ReceiptProjectionFilters{Source: q.Get("source"), Target: q.Get("target")}
	if raw := q.Get("enabled"); raw != "" {
		enabled := raw == "true"
		filters.Enabled = &enabled
	}
	rules, err := s.policyStore.ListReceiptProjections(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodePolicyRead, "failed to list receipt projection rules")
		return
	}
	writeJSON(w, http.StatusOK, orEmpty(rules))
}

func (s *Server) handleGetReceiptProjection(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidParam, "invalid receipt projection ID")
		return
	}
	rule, err := s.policyStore.GetReceiptProjection(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, "receipt projection rule not found")
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (s *Server) handleUpdateReceiptProjection(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidParam, "invalid receipt projection ID")
		return
	}
	if _, err := s.policyStore.GetReceiptProjection(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, "receipt projection rule not found")
		return
	}
	var rule policy.ReceiptProjectionRule
	if !decodeJSONBody(w, r, &rule) {
		return
	}
	rule.ID = id
	if message := validateReceiptProjectionRule(&rule); message != "" {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, message)
		return
	}
	if err := s.policyStore.UpdateReceiptProjection(r.Context(), rule); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodePolicyWrite, "failed to update receipt projection rule")
		return
	}
	s.broadcastPolicySnapshot(r.Context())
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleDeleteReceiptProjection(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidParam, "invalid receipt projection ID")
		return
	}
	if err := s.policyStore.DeleteReceiptProjection(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodePolicyWrite, "failed to delete receipt projection rule")
		return
	}
	s.broadcastPolicySnapshot(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

func validateReceiptProjectionRule(rule *policy.ReceiptProjectionRule) string {
	if strings.TrimSpace(rule.SourceScenario) == "" || strings.TrimSpace(rule.TargetScenario) == "" || strings.TrimSpace(rule.OperationPattern) == "" {
		return "source_scenario, target_scenario, and operation_pattern are required"
	}
	if len(rule.ResponseFields) == 0 {
		return "response_fields must declare the approved safe projection"
	}
	if rule.MaxBytes <= 0 || rule.MaxBytes > 64*1024 {
		return "max_bytes must be between 1 and 65536"
	}
	if rule.SamplePerTenK < 0 || rule.SamplePerTenK > 10000 {
		return "sample_per_ten_k must be between 0 and 10000"
	}
	if rule.RetentionDays <= 0 {
		return "retention_days must be > 0"
	}
	return ""
}

// parseProjectionID is retained as a narrow parsing seam for callers that do
// not use the standard mux path variables.
func parseProjectionID(raw string) (int64, error) { return strconv.ParseInt(raw, 10, 64) }
