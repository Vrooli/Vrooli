// Package handlers - audit provider HTTP endpoint for scenario-auditor integration.
// [REQ:BM-REQ-AUDIT-PROVIDER]
//
// Implements the externalRuleProvider interface pattern used by scenario-auditor:
// GET /api/v1/audit/rules returns rules in the auditor-compatible format with
// evaluate_url pointing back to this service for per-scenario evaluation.
package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"brand-manager/apierr"
	"brand-manager/domain"

	"github.com/gorilla/mux"
)

// auditRuleResult represents a single rule evaluation result for scenario-auditor.
type auditRuleResult struct {
	RuleID   string `json:"rule_id"`
	Pass     bool   `json:"pass"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// auditProviderResponse is the response format for the audit provider endpoint.
type auditProviderResponse struct {
	Provider string         `json:"provider"`
	Version  string         `json:"version"`
	Rules    []BrandingRule `json:"rules"`
}

// GetAuditRules handles GET /api/v1/audit/rules. [REQ:BM-REQ-AUDIT-PROVIDER]
// Returns branding rules in the scenario-auditor externalRuleProvider format.
func (h *Handlers) GetAuditRules(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, auditProviderResponse{
		Provider: "brand-manager",
		Version:  h.cfg.APIVersion,
		Rules:    standardRules,
	})
}

// ruleEvaluator pairs a rule with a function that checks whether the brand satisfies it.
type ruleEvaluator struct {
	rule  BrandingRule
	check func(b *domain.Brand) bool
}

// ruleEvaluators connects each standard rule to its pass/fail decision.
// This is the single source of truth for what each rule checks.
var ruleEvaluators = []ruleEvaluator{
	{standardRules[0], func(b *domain.Brand) bool { // has-logo
		return b.Identity != nil && b.Identity.LogoPath != ""
	}},
	{standardRules[1], func(b *domain.Brand) bool { // has-favicon
		return b.Identity != nil && b.Identity.FaviconPath != ""
	}},
	{standardRules[2], func(b *domain.Brand) bool { // has-color-system
		return b.Colors != nil && b.Colors.Primary != "" && b.Colors.Background != "" && b.Colors.Surface != "" && b.Colors.Text != ""
	}},
	{standardRules[3], func(b *domain.Brand) bool { // has-display-name
		return b.Identity != nil && b.Identity.DisplayName != ""
	}},
	{standardRules[4], func(b *domain.Brand) bool { // has-typography
		return b.Typography != nil && b.Typography.HeadingFont != "" && b.Typography.BodyFont != ""
	}},
}

// evaluateRules runs the given rule evaluators against a brand (or nil for unassigned scenarios).
func evaluateRules(evaluators []ruleEvaluator, brand *domain.Brand, fallbackMsg string) []auditRuleResult {
	results := make([]auditRuleResult, len(evaluators))
	for i, re := range evaluators {
		pass := brand != nil && re.check(brand)
		msg := fallbackMsg
		if brand != nil {
			if pass {
				msg = re.rule.Name + " is defined"
			} else {
				msg = re.rule.Name + " is not defined"
			}
		}
		results[i] = auditRuleResult{
			RuleID:   re.rule.ID,
			Pass:     pass,
			Severity: re.rule.Severity,
			Message:  msg,
		}
	}
	return results
}

// selectEvaluators returns the evaluators matching the optional ?rule= query parameter.
// When ruleID is empty, all evaluators are returned. When ruleID is non-empty but unknown,
// returns nil to signal a 400 to the caller.
func selectEvaluators(ruleID string) ([]ruleEvaluator, bool) {
	if ruleID == "" {
		return ruleEvaluators, true
	}
	for _, re := range ruleEvaluators {
		if re.rule.ID == ruleID {
			return []ruleEvaluator{re}, true
		}
	}
	return nil, false
}

// EvaluateScenario handles POST /api/v1/audit/evaluate/{scenario}. [REQ:BM-REQ-AUDIT-PROVIDER]
// Evaluates branding rules against a specific scenario's brand assignment.
// Optional `?rule=<id>` query parameter limits evaluation to a single rule.
func (h *Handlers) EvaluateScenario(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario"]

	ruleID := r.URL.Query().Get("rule")
	evaluators, ok := selectEvaluators(ruleID)
	if !ok {
		apierr.Write(w, apierr.Validation("unknown rule: "+ruleID))
		return
	}

	assignment, err := h.assignments.GetByScenario(r.Context(), scenario)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"scenario": scenario,
			"results":  evaluateRules(evaluators, nil, "no brand assigned to scenario"),
		})
		return
	}
	if err != nil {
		apierr.Write(w, apierr.Internal("get assignment", err))
		return
	}

	brand, err := h.brands.GetByID(r.Context(), assignment.BrandID)
	if err != nil {
		apierr.Write(w, apierr.Internal("get brand", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"scenario": scenario,
		"results":  evaluateRules(evaluators, brand, ""),
	})
}
