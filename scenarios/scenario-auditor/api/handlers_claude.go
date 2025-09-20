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
	ScenarioName string            `json:"scenario_name"`
	FixType      string            `json:"fix_type"`
	IssueIDs     []string          `json:"issue_ids"`
	Targets      []claudeFixTarget `json:"targets"`
}

type claudeFixTarget struct {
	Scenario string   `json:"scenario"`
	IssueIDs []string `json:"issue_ids"`
}

func triggerClaudeFixHandler(w http.ResponseWriter, r *http.Request) {
	logger := NewLogger()

	var req claudeFixRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid request body", http.StatusBadRequest, err)
		return
	}

	fixType := strings.TrimSpace(req.FixType)
	if fixType == "" {
		fixType = "standards"
	}

	issueIDs := normaliseIDs(req.IssueIDs)

	var (
		prompt            string
		label             string
		metadata          map[string]string
		action            string
		err               error
		finalIDs          []string
		agentCfg          AgentStartConfig
		responseMessage   string
		responseScenarios []string
	)

	if len(req.Targets) > 0 {
		switch fixType {
		case "standards":
			multiTargets, collectedIDs, err := prepareBulkStandardsTargets(req.Targets)
			if err != nil {
				HTTPError(w, err.Error(), http.StatusBadRequest, nil)
				return
			}
			prompt, label, metadata, err = buildMultiStandardsFixPrompt(multiTargets)
			if err != nil {
				HTTPError(w, "Failed to build agent prompt", http.StatusInternalServerError, err)
				return
			}
			metadata["fix_type"] = fixType
			scenarioNames := extractStandardsScenarioNames(multiTargets)
			metadata["scenarios"] = strings.Join(scenarioNames, ",")
			action = agentActionStandardsFix
			agentCfg = AgentStartConfig{
				Label:    label,
				Name:     fmt.Sprintf("%s (multi)", label),
				Action:   action,
				Scenario: "multi",
				IssueIDs: collectedIDs,
				Prompt:   prompt,
				Model:    openRouterModel,
				Metadata: metadata,
			}
			responseMessage = fmt.Sprintf("Started %s across %d scenario(s)", label, len(multiTargets))
			responseScenarios = scenarioNames
		case "vulnerabilities":
			multiTargets, collectedIDs, err := prepareBulkVulnerabilityTargets(req.Targets)
			if err != nil {
				HTTPError(w, err.Error(), http.StatusBadRequest, nil)
				return
			}
			prompt, label, metadata, err = buildMultiVulnerabilityFixPrompt(multiTargets)
			if err != nil {
				HTTPError(w, "Failed to build agent prompt", http.StatusInternalServerError, err)
				return
			}
			metadata["fix_type"] = fixType
			scenarioNames := extractVulnerabilityScenarioNames(multiTargets)
			metadata["scenarios"] = strings.Join(scenarioNames, ",")
			action = agentActionVulnerabilityFix
			agentCfg = AgentStartConfig{
				Label:    label,
				Name:     fmt.Sprintf("%s (multi)", label),
				Action:   action,
				Scenario: "multi",
				IssueIDs: collectedIDs,
				Prompt:   prompt,
				Model:    openRouterModel,
				Metadata: metadata,
			}
			responseMessage = fmt.Sprintf("Started %s across %d scenario(s)", label, len(multiTargets))
			responseScenarios = scenarioNames
		default:
			HTTPError(w, "Unsupported fix_type", http.StatusBadRequest, nil)
			return
		}
	} else {
		scenarioName := strings.TrimSpace(req.ScenarioName)
		if scenarioName == "" {
			HTTPError(w, "scenario_name is required", http.StatusBadRequest, nil)
			return
		}

		scenarioPath := filepath.Clean(filepath.Join(getScenarioRoot(), "..", scenarioName))
		if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
			HTTPError(w, "Scenario not found", http.StatusNotFound, err)
			return
		}

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

		agentCfg = AgentStartConfig{
			Label:    label,
			Name:     fmt.Sprintf("%s (%s)", label, scenarioName),
			Action:   action,
			Scenario: scenarioName,
			IssueIDs: finalIDs,
			Prompt:   prompt,
			Model:    openRouterModel,
			Metadata: metadata,
		}
		responseMessage = fmt.Sprintf("Started %s for %s", label, scenarioName)
		responseScenarios = []string{scenarioName}
	}

	agentInfo, err := agentManager.StartAgent(agentCfg)
	if err != nil {
		HTTPError(w, "Failed to start agent", http.StatusInternalServerError, err)
		return
	}

	response := map[string]interface{}{
		"success":    true,
		"message":    responseMessage,
		"fix_id":     agentInfo.ID,
		"started_at": agentInfo.StartedAt,
		"agent":      agentInfo,
		"scenarios":  responseScenarios,
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

func prepareBulkStandardsTargets(targets []claudeFixTarget) ([]standardsFixMultiScenario, []string, error) {
	var result []standardsFixMultiScenario
	seenIssues := make(map[string]struct{})
	var collectedIDs []string

	indexByScenario := make(map[string]int)

	for _, target := range targets {
		scenario := strings.TrimSpace(target.Scenario)
		if scenario == "" {
			continue
		}

		scenarioPath := filepath.Clean(filepath.Join(getScenarioRoot(), "..", scenario))
		if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("scenario %s not found", scenario)
		}

		available := standardsStore.GetViolations(scenario)
		issueIDs := normaliseIDs(target.IssueIDs)
		selected, ids := selectStandardsViolations(available, issueIDs)
		if len(selected) == 0 {
			continue
		}

		if idx, exists := indexByScenario[scenario]; exists {
			existing := result[idx].Violations
			existing = append(existing, selected...)
			result[idx].Violations = existing
		} else {
			indexByScenario[scenario] = len(result)
			result = append(result, standardsFixMultiScenario{
				Scenario:     scenario,
				ScenarioPath: scenarioPath,
				Violations:   selected,
			})
		}

		for _, id := range ids {
			if id == "" {
				continue
			}
			if _, exists := seenIssues[id]; !exists {
				seenIssues[id] = struct{}{}
				collectedIDs = append(collectedIDs, id)
			}
		}
	}

	if len(result) == 0 {
		return nil, nil, fmt.Errorf("no matching standards violations found for provided targets")
	}

	sort.Strings(collectedIDs)
	return result, collectedIDs, nil
}

func prepareBulkVulnerabilityTargets(targets []claudeFixTarget) ([]vulnerabilityFixMultiScenario, []string, error) {
	var result []vulnerabilityFixMultiScenario
	seenIssues := make(map[string]struct{})
	var collectedIDs []string

	indexByScenario := make(map[string]int)

	for _, target := range targets {
		scenario := strings.TrimSpace(target.Scenario)
		if scenario == "" {
			continue
		}

		scenarioPath := filepath.Clean(filepath.Join(getScenarioRoot(), "..", scenario))
		if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("scenario %s not found", scenario)
		}

		available := vulnStore.GetVulnerabilities(scenario)
		issueIDs := normaliseIDs(target.IssueIDs)
		selected, ids := selectStoredVulnerabilities(available, issueIDs)
		if len(selected) == 0 {
			continue
		}

		if idx, exists := indexByScenario[scenario]; exists {
			existing := result[idx].Findings
			existing = append(existing, selected...)
			result[idx].Findings = existing
		} else {
			indexByScenario[scenario] = len(result)
			result = append(result, vulnerabilityFixMultiScenario{
				Scenario:     scenario,
				ScenarioPath: scenarioPath,
				Findings:     selected,
			})
		}

		for _, id := range ids {
			if id == "" {
				continue
			}
			if _, exists := seenIssues[id]; !exists {
				seenIssues[id] = struct{}{}
				collectedIDs = append(collectedIDs, id)
			}
		}
	}

	if len(result) == 0 {
		return nil, nil, fmt.Errorf("no matching vulnerabilities found for provided targets")
	}

	sort.Strings(collectedIDs)
	return result, collectedIDs, nil
}

func extractStandardsScenarioNames(targets []standardsFixMultiScenario) []string {
	set := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		set[target.Scenario] = struct{}{}
	}
	return sortedKeys(set)
}

func extractVulnerabilityScenarioNames(targets []vulnerabilityFixMultiScenario) []string {
	set := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		set[target.Scenario] = struct{}{}
	}
	return sortedKeys(set)
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
