package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gorilla/mux"
)

type claudeFixRequest struct {
	ScenarioName string   `json:"scenario_name"`
	FixType      string   `json:"fix_type"`
	IssueIDs     []string `json:"issue_ids"`
}

func triggerClaudeFixHandler(w http.ResponseWriter, r *http.Request) {
	logger := NewLogger()

	var req claudeFixRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid request body", http.StatusBadRequest, err)
		return
	}

	scenarioName := strings.TrimSpace(req.ScenarioName)
	if scenarioName == "" {
		HTTPError(w, "scenario_name is required", http.StatusBadRequest, nil)
		return
	}

	fixType := strings.TrimSpace(req.FixType)
	if fixType == "" {
		fixType = "standards"
	}

	scenarioPath := filepath.Clean(filepath.Join(getScenarioRoot(), "..", scenarioName))
	if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
		HTTPError(w, "Scenario not found", http.StatusNotFound, err)
		return
	}

	issueIDs := normaliseIDs(req.IssueIDs)

	var (
		prompt   string
		label    string
		metadata map[string]string
		action   string
		err      error
		finalIDs []string
	)

	switch fixType {
	case "standards":
		violations := standardsStore.GetViolations(scenarioName)
		selected, ids := selectStandardsViolations(violations, issueIDs)
		if len(selected) == 0 {
			HTTPError(w, "No matching standards violations found to fix", http.StatusBadRequest, nil)
			return
		}
		if len(issueIDs) > 0 {
			if missing := findMissingIDs(issueIDs, ids); len(missing) > 0 {
				HTTPError(w, fmt.Sprintf("Unknown standards violation IDs: %s", strings.Join(missing, ", ")), http.StatusBadRequest, nil)
				return
			}
		}
		prompt, label, metadata, err = buildStandardsFixPrompt(scenarioName, scenarioPath, selected, ids)
		finalIDs = ids
		action = agentActionStandardsFix
	case "vulnerabilities":
		vulnerabilities := vulnStore.GetVulnerabilities(scenarioName)
		selected, ids := selectStoredVulnerabilities(vulnerabilities, issueIDs)
		if len(selected) == 0 {
			HTTPError(w, "No matching vulnerabilities found to fix", http.StatusBadRequest, nil)
			return
		}
		if len(issueIDs) > 0 {
			if missing := findMissingIDs(issueIDs, ids); len(missing) > 0 {
				HTTPError(w, fmt.Sprintf("Unknown vulnerability IDs: %s", strings.Join(missing, ", ")), http.StatusBadRequest, nil)
				return
			}
		}
		prompt, label, metadata, err = buildVulnerabilityFixPrompt(scenarioName, scenarioPath, selected, ids)
		finalIDs = ids
		action = agentActionVulnerabilityFix
	default:
		HTTPError(w, "Unsupported fix_type", http.StatusBadRequest, nil)
		return
	}

	if err != nil {
		HTTPError(w, "Failed to build agent prompt", http.StatusInternalServerError, err)
		return
	}

	if metadata == nil {
		metadata = make(map[string]string)
	}
	metadata["scenario"] = scenarioName
	metadata["scenario_path"] = scenarioPath
	metadata["fix_type"] = fixType

	agentInfo, err := agentManager.StartAgent(AgentStartConfig{
		Label:    label,
		Name:     fmt.Sprintf("%s (%s)", label, scenarioName),
		Action:   action,
		Scenario: scenarioName,
		IssueIDs: finalIDs,
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
		"message":    fmt.Sprintf("Started %s for %s", label, scenarioName),
		"fix_id":     agentInfo.ID,
		"started_at": agentInfo.StartedAt,
		"agent":      agentInfo,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode trigger fix response", err)
	}
}

func getClaudeFixStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fixID := vars["fixId"]
	if strings.TrimSpace(fixID) == "" {
		HTTPError(w, "fixId is required", http.StatusBadRequest, nil)
		return
	}

	if agent, ok := agentManager.GetAgent(fixID); ok {
		writeAgentStatusResponse(w, agent)
		return
	}

	if agent, ok := agentManager.GetAgentHistory(fixID); ok {
		writeAgentStatusResponse(w, agent)
		return
	}

	HTTPError(w, "Agent not found", http.StatusNotFound, nil)
}

func writeAgentStatusResponse(w http.ResponseWriter, agent *AgentInfo) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"status":  agent.Status,
		"agent":   agent,
	})
}

func normaliseIDs(ids []string) []string {
	unique := make(map[string]struct{})
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		unique[trimmed] = struct{}{}
	}
	if len(unique) == 0 {
		return nil
	}
	result := make([]string, 0, len(unique))
	for id := range unique {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

func findMissingIDs(requested, found []string) []string {
	if len(requested) == 0 {
		return nil
	}
	present := make(map[string]struct{}, len(found))
	for _, id := range found {
		present[id] = struct{}{}
	}
	var missing []string
	for _, id := range requested {
		if _, ok := present[id]; !ok {
			missing = append(missing, id)
		}
	}
	return missing
}

func selectStandardsViolations(all []StandardsViolation, ids []string) ([]StandardsViolation, []string) {
	if len(all) == 0 {
		return nil, nil
	}
	if len(ids) == 0 {
		result := append([]StandardsViolation(nil), all...)
		collected := make([]string, 0, len(result))
		for _, v := range result {
			if v.ID != "" {
				collected = append(collected, v.ID)
			}
		}
		return result, collected
	}
	allowed := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		allowed[id] = struct{}{}
	}
	var filtered []StandardsViolation
	var collected []string
	for _, v := range all {
		if _, ok := allowed[v.ID]; ok {
			filtered = append(filtered, v)
			if v.ID != "" {
				collected = append(collected, v.ID)
			}
		}
	}
	return filtered, collected
}

func selectStoredVulnerabilities(all []StoredVulnerability, ids []string) ([]StoredVulnerability, []string) {
	if len(all) == 0 {
		return nil, nil
	}
	if len(ids) == 0 {
		result := append([]StoredVulnerability(nil), all...)
		collected := make([]string, 0, len(result))
		for _, v := range result {
			if v.ID != "" {
				collected = append(collected, v.ID)
			}
		}
		return result, collected
	}
	allowed := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		allowed[id] = struct{}{}
	}
	var filtered []StoredVulnerability
	var collected []string
	for _, v := range all {
		if _, ok := allowed[v.ID]; ok {
			filtered = append(filtered, v)
			if v.ID != "" {
				collected = append(collected, v.ID)
			}
		}
	}
	return filtered, collected
}
