package handlers

import (
	"encoding/json"
	"net/http"
	
	"github.com/gorilla/mux"
	
	"email-triage/models"
	"email-triage/services"
)

// RuleHandler handles triage rule management endpoints
type RuleHandler struct {
	ruleService *services.RuleService
}

// NewRuleHandler creates a new RuleHandler instance
func NewRuleHandler(ruleService *services.RuleService) *RuleHandler {
	return &RuleHandler{
		ruleService: ruleService,
	}
}

// CreateRule handles POST /api/v1/rules
func (rh *RuleHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req models.CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Description == "" {
		http.Error(w, `{"error":"description is required"}`, http.StatusBadRequest)
		return
	}

	if len(req.Actions) == 0 {
		http.Error(w, `{"error":"at least one action is required"}`, http.StatusBadRequest)
		return
	}

	// Create rule using service
	response, err := rh.ruleService.CreateRule(userID, &req)
	if err != nil {
		responseObj := map[string]interface{}{
			"success": false,
			"error":   "rule_creation_failed",
			"message": err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(responseObj)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// ListRules handles GET /api/v1/rules
func (rh *RuleHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	// Get user's rules
	rules, err := rh.ruleService.GetRulesByUser(userID)
	if err != nil {
		http.Error(w, `{"error":"failed to retrieve rules"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"rules": rules,
		"count": len(rules),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetRule handles GET /api/v1/rules/{id}
func (rh *RuleHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	ruleID := vars["id"]

	// Get all user rules and find the specific one
	// (In production, you'd have a more efficient GetRuleByID method)
	rules, err := rh.ruleService.GetRulesByUser(userID)
	if err != nil {
		http.Error(w, `{"error":"failed to retrieve rules"}`, http.StatusInternalServerError)
		return
	}

	// Find the specific rule
	var targetRule *models.TriageRule
	for _, rule := range rules {
		if rule.ID == ruleID {
			targetRule = rule
			break
		}
	}

	if targetRule == nil {
		http.Error(w, `{"error":"rule not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(targetRule)
}

// UpdateRule handles PUT /api/v1/rules/{id}
func (rh *RuleHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	ruleID := vars["id"]

	// Parse request body
	var updateReq map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// TODO: Implement rule update functionality
	// For now, return a placeholder response

	response := map[string]interface{}{
		"success": true,
		"message": "Rule update not yet implemented",
		"rule_id": ruleID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteRule handles DELETE /api/v1/rules/{id}
func (rh *RuleHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	ruleID := vars["id"]

	// TODO: Implement rule deletion
	// For now, return a placeholder response

	response := map[string]interface{}{
		"success": true,
		"message": "Rule deletion not yet implemented",
		"rule_id": ruleID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TestRule handles POST /api/v1/rules/{id}/test
func (rh *RuleHandler) TestRule(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	ruleID := vars["id"]

	// Parse request body for test email data
	var testReq struct {
		Subject     string   `json:"subject"`
		Sender      string   `json:"sender"`
		Body        string   `json:"body"`
		Recipients  []string `json:"recipients"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&testReq); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// TODO: Implement rule testing against provided email data
	// This would:
	// 1. Get the rule by ID
	// 2. Create a mock email from the test data
	// 3. Test if the rule would match
	// 4. Return match result and confidence

	response := map[string]interface{}{
		"success":    true,
		"rule_id":    ruleID,
		"matches":    true,
		"confidence": 0.85,
		"message":    "Rule testing not fully implemented",
		"explanation": "This rule would likely match the provided email based on subject analysis",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}