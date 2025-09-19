package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
)

var agentManager = NewAgentManager()

type startAgentRequest struct {
	Action string `json:"action"`
	RuleID string `json:"rule_id"`
	Label  string `json:"label"`
}

func startAgentHandler(w http.ResponseWriter, r *http.Request) {
	logger := NewLogger()

	var req startAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid request body", http.StatusBadRequest, err)
		return
	}

	if req.Action == "" {
		HTTPError(w, "action is required", http.StatusBadRequest, nil)
		return
	}

	if req.RuleID == "" {
		// Allow path parameter as fallback
		if vars := mux.Vars(r); vars != nil {
			req.RuleID = vars["ruleId"]
		}
	}

	switch req.Action {
	case agentActionAddRuleTests, agentActionFixRuleTests:
		if req.RuleID == "" {
			HTTPError(w, "rule_id is required for rule test agents", http.StatusBadRequest, nil)
			return
		}
	default:
		HTTPError(w, fmt.Sprintf("unsupported agent action: %s", req.Action), http.StatusBadRequest, nil)
		return
	}

	rules, err := LoadRulesFromFiles()
	if err != nil {
		HTTPError(w, "Failed to load rules", http.StatusInternalServerError, err)
		return
	}

	rule, exists := rules[req.RuleID]
	if !exists {
		HTTPError(w, "Rule not found", http.StatusNotFound, nil)
		return
	}

	prompt, derivedLabel, metadata, err := buildRuleAgentPrompt(rule, req.Action)
	if err != nil {
		HTTPError(w, "Failed to build agent prompt", http.StatusInternalServerError, err)
		return
	}

	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = derivedLabel
	}

	name := fmt.Sprintf("%s (%s)", label, safeFallback(rule.Name, rule.ID))

	agentInfo, err := agentManager.StartAgent(AgentStartConfig{
		Label:    label,
		Name:     name,
		Action:   req.Action,
		RuleID:   rule.ID,
		Prompt:   prompt,
		Model:    openRouterModel,
		Metadata: metadata,
	})
	if err != nil {
		HTTPError(w, "Failed to start agent", http.StatusInternalServerError, err)
		return
	}

	response := map[string]interface{}{
		"success":    true,
		"agent":      agentInfo,
		"message":    fmt.Sprintf("Agent %s started", agentInfo.Name),
		"prompt_len": agentInfo.PromptLength,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode agent start response", err)
	}
}

func getAgentsHandler(w http.ResponseWriter, r *http.Request) {
	agents := agentManager.ListAgents()
	response := map[string]interface{}{
		"count":  len(agents),
		"agents": agents,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger := NewLogger()
		logger.Error("Failed to encode active agents response", err)
	}
}

func stopAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agentId"]
	if agentID == "" {
		HTTPError(w, "agentId is required", http.StatusBadRequest, nil)
		return
	}

	if err := agentManager.StopAgent(agentID); err != nil {
		HTTPError(w, err.Error(), http.StatusNotFound, err)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Agent %s stopped", agentID),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getAgentLogsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agentId"]
	if agentID == "" {
		HTTPError(w, "agentId is required", http.StatusBadRequest, nil)
		return
	}

	logPath := agentManager.AgentLogPath(agentID)
	data, err := os.ReadFile(logPath)
	if err != nil {
		HTTPError(w, "Failed to read agent logs", http.StatusNotFound, err)
		return
	}

	response := map[string]interface{}{
		"agent_id": agentID,
		"logs":     string(data),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
