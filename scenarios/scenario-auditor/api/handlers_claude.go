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
	ExtraPrompt  string            `json:"extra_prompt"`
	AgentCount   int               `json:"agent_count"`
}

type claudeFixTarget struct {
	Scenario string   `json:"scenario"`
	IssueIDs []string `json:"issue_ids"`
}

const maxBulkFixAgents = 10
const maxIssuesPerAgent = 50

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
	extraPrompt := strings.TrimSpace(req.ExtraPrompt)

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
		issueCount        int
	)

	if len(req.Targets) > 0 {
		switch fixType {
		case "standards":
			multiTargets, _, err := prepareBulkStandardsTargets(req.Targets)
			if err != nil {
				HTTPError(w, err.Error(), http.StatusBadRequest, nil)
				return
			}
			totalViolations := countStandardsViolations(multiTargets)
			if totalViolations == 0 {
				HTTPError(w, "No matching standards violations found to fix", http.StatusBadRequest, nil)
				return
			}
			scenarioNames := extractStandardsScenarioNames(multiTargets)
			issueCount = totalViolations
			requestedAgents := clampAgentCount(req.AgentCount, totalViolations)
			if requestedAgents > 1 {
				groups := splitStandardsTargets(multiTargets, requestedAgents)
				if len(groups) == 0 {
					HTTPError(w, "Failed to partition standards violations for multi-agent fix", http.StatusInternalServerError, nil)
					return
				}
				startedAgents := make([]*AgentInfo, 0, len(groups))
				for idx, group := range groups {
					prompt, groupLabel, metadata, err := buildMultiStandardsFixPrompt(group, extraPrompt)
					if err != nil {
						HTTPError(w, "Failed to build agent prompt", http.StatusInternalServerError, err)
						return
					}
					if metadata == nil {
						metadata = make(map[string]string)
					}
					metadata["fix_type"] = fixType
					metadata["scenarios"] = strings.Join(extractStandardsScenarioNames(group), ",")
					metadata["batch_index"] = fmt.Sprintf("%d/%d", idx+1, len(groups))
					if extraPrompt != "" {
						metadata["user_instructions"] = extraPrompt
					}
					batchIDs := collectStandardsIssueIDs(group)
					metadata["issue_count"] = fmt.Sprintf("%d", len(batchIDs))
					metadata["issues_per_agent_cap"] = fmt.Sprintf("%d", maxIssuesPerAgent)
					labelWithBatch := groupLabel
					if len(groups) > 1 {
						labelWithBatch = fmt.Sprintf("%s (batch %d/%d)", groupLabel, idx+1, len(groups))
					}
					agentCfg := AgentStartConfig{
						Label:    labelWithBatch,
						Name:     labelWithBatch,
						Action:   agentActionStandardsFix,
						Scenario: "multi",
						IssueIDs: batchIDs,
						Prompt:   prompt,
						Model:    openRouterModel,
						Metadata: metadata,
					}
					agentInfo, startErr := agentManager.StartAgent(agentCfg)
					if startErr != nil {
						HTTPError(w, fmt.Sprintf("Failed to start agent batch %d", idx+1), http.StatusInternalServerError, startErr)
						return
					}
					startedAgents = append(startedAgents, agentInfo)
				}
				fixIDs := make([]string, 0, len(startedAgents))
				for _, info := range startedAgents {
					fixIDs = append(fixIDs, info.ID)
				}
				message := fmt.Sprintf(
					"Started %d agents to address %d violations across %d scenario(s)",
					len(startedAgents), totalViolations, len(scenarioNames),
				)
				response := map[string]interface{}{
					"success":              true,
					"message":              message,
					"fix_id":               fixIDs[0],
					"fix_ids":              fixIDs,
					"agent_count":          len(startedAgents),
					"agents":               startedAgents,
					"issue_count":          totalViolations,
					"issues_per_agent_cap": maxIssuesPerAgent,
					"scenarios":            scenarioNames,
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(response); err != nil {
					logger.Error("Failed to encode trigger fix response", err)
				}
				return
			}

			prompt, label, metadata, err = buildMultiStandardsFixPrompt(multiTargets, extraPrompt)
			if err != nil {
				HTTPError(w, "Failed to build agent prompt", http.StatusInternalServerError, err)
				return
			}
			if metadata == nil {
				metadata = make(map[string]string)
			}
			metadata["fix_type"] = fixType
			metadata["scenarios"] = strings.Join(scenarioNames, ",")
			if extraPrompt != "" {
				metadata["user_instructions"] = extraPrompt
			}
			metadata["issue_count"] = fmt.Sprintf("%d", totalViolations)
			metadata["issues_per_agent_cap"] = fmt.Sprintf("%d", maxIssuesPerAgent)
			action = agentActionStandardsFix
			agentCfg = AgentStartConfig{
				Label:    label,
				Name:     fmt.Sprintf("%s (multi)", label),
				Action:   action,
				Scenario: "multi",
				IssueIDs: collectStandardsIssueIDs(multiTargets),
				Prompt:   prompt,
				Model:    openRouterModel,
				Metadata: metadata,
			}
			responseMessage = fmt.Sprintf("Started %s across %d scenario(s)", label, len(scenarioNames))
			responseScenarios = scenarioNames
		case "vulnerabilities":
			multiTargets, _, err := prepareBulkVulnerabilityTargets(req.Targets)
			if err != nil {
				HTTPError(w, err.Error(), http.StatusBadRequest, nil)
				return
			}
			totalFindings := countVulnerabilityFindings(multiTargets)
			if totalFindings == 0 {
				HTTPError(w, "No matching vulnerabilities found to fix", http.StatusBadRequest, nil)
				return
			}
			scenarioNames := extractVulnerabilityScenarioNames(multiTargets)
			issueCount = totalFindings
			requestedAgents := clampAgentCount(req.AgentCount, totalFindings)
			if requestedAgents > 1 {
				groups := splitVulnerabilityTargets(multiTargets, requestedAgents)
				if len(groups) == 0 {
					HTTPError(w, "Failed to partition vulnerabilities for multi-agent fix", http.StatusInternalServerError, nil)
					return
				}
				startedAgents := make([]*AgentInfo, 0, len(groups))
				for idx, group := range groups {
					prompt, groupLabel, metadata, err := buildMultiVulnerabilityFixPrompt(group, extraPrompt)
					if err != nil {
						HTTPError(w, "Failed to build agent prompt", http.StatusInternalServerError, err)
						return
					}
					if metadata == nil {
						metadata = make(map[string]string)
					}
					metadata["fix_type"] = fixType
					metadata["scenarios"] = strings.Join(extractVulnerabilityScenarioNames(group), ",")
					metadata["batch_index"] = fmt.Sprintf("%d/%d", idx+1, len(groups))
					if extraPrompt != "" {
						metadata["user_instructions"] = extraPrompt
					}
					batchIDs := collectVulnerabilityIssueIDs(group)
					metadata["issue_count"] = fmt.Sprintf("%d", len(batchIDs))
					metadata["issues_per_agent_cap"] = fmt.Sprintf("%d", maxIssuesPerAgent)
					labelWithBatch := groupLabel
					if len(groups) > 1 {
						labelWithBatch = fmt.Sprintf("%s (batch %d/%d)", groupLabel, idx+1, len(groups))
					}
					agentCfg := AgentStartConfig{
						Label:    labelWithBatch,
						Name:     labelWithBatch,
						Action:   agentActionVulnerabilityFix,
						Scenario: "multi",
						IssueIDs: batchIDs,
						Prompt:   prompt,
						Model:    openRouterModel,
						Metadata: metadata,
					}
					agentInfo, startErr := agentManager.StartAgent(agentCfg)
					if startErr != nil {
						HTTPError(w, fmt.Sprintf("Failed to start agent batch %d", idx+1), http.StatusInternalServerError, startErr)
						return
					}
					startedAgents = append(startedAgents, agentInfo)
				}
				fixIDs := make([]string, 0, len(startedAgents))
				for _, info := range startedAgents {
					fixIDs = append(fixIDs, info.ID)
				}
				message := fmt.Sprintf(
					"Started %d agents to address %d vulnerabilities across %d scenario(s)",
					len(startedAgents), totalFindings, len(scenarioNames),
				)
				response := map[string]interface{}{
					"success":              true,
					"message":              message,
					"fix_id":               fixIDs[0],
					"fix_ids":              fixIDs,
					"agent_count":          len(startedAgents),
					"agents":               startedAgents,
					"issue_count":          totalFindings,
					"issues_per_agent_cap": maxIssuesPerAgent,
					"scenarios":            scenarioNames,
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(response); err != nil {
					logger.Error("Failed to encode trigger fix response", err)
				}
				return
			}

			prompt, label, metadata, err = buildMultiVulnerabilityFixPrompt(multiTargets, extraPrompt)
			if err != nil {
				HTTPError(w, "Failed to build agent prompt", http.StatusInternalServerError, err)
				return
			}
			if metadata == nil {
				metadata = make(map[string]string)
			}
			metadata["fix_type"] = fixType
			metadata["scenarios"] = strings.Join(scenarioNames, ",")
			if extraPrompt != "" {
				metadata["user_instructions"] = extraPrompt
			}
			metadata["issue_count"] = fmt.Sprintf("%d", totalFindings)
			metadata["issues_per_agent_cap"] = fmt.Sprintf("%d", maxIssuesPerAgent)
			action = agentActionVulnerabilityFix
			agentCfg = AgentStartConfig{
				Label:    label,
				Name:     fmt.Sprintf("%s (multi)", label),
				Action:   action,
				Scenario: "multi",
				IssueIDs: collectVulnerabilityIssueIDs(multiTargets),
				Prompt:   prompt,
				Model:    openRouterModel,
				Metadata: metadata,
			}
			responseMessage = fmt.Sprintf("Started %s across %d scenario(s)", label, len(scenarioNames))
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
			issueCount = len(finalIDs)
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
			issueCount = len(finalIDs)
			action = agentActionVulnerabilityFix
		default:
			HTTPError(w, "Unsupported fix_type", http.StatusBadRequest, nil)
			return
		}

		if err == nil && extraPrompt != "" {
			prompt = appendUserInstructions(prompt, extraPrompt)
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
		if extraPrompt != "" {
			metadata["user_instructions"] = extraPrompt
		}
		if issueCount > 0 {
			metadata["issue_count"] = fmt.Sprintf("%d", issueCount)
		}
		metadata["issues_per_agent_cap"] = fmt.Sprintf("%d", maxIssuesPerAgent)

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
		"success":     true,
		"message":     responseMessage,
		"fix_id":      agentInfo.ID,
		"started_at":  agentInfo.StartedAt,
		"agent":       agentInfo,
		"scenarios":   responseScenarios,
		"agent_count": 1,
		"fix_ids":     []string{agentInfo.ID},
	}
	if extraPrompt != "" {
		response["user_instructions"] = extraPrompt
	}
	if issueCount > 0 {
		response["issue_count"] = issueCount
	}
	response["issues_per_agent_cap"] = maxIssuesPerAgent

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

func clampAgentCount(requested, total int) int {
	if total <= 0 {
		return 0
	}
	minAgents := (total + maxIssuesPerAgent - 1) / maxIssuesPerAgent
	if minAgents < 1 {
		minAgents = 1
	}
	if requested < 1 {
		requested = 1
	}
	if requested < minAgents {
		requested = minAgents
	}
	if requested > total {
		requested = total
	}
	if requested > maxBulkFixAgents {
		requested = maxBulkFixAgents
	}
	if requested < minAgents {
		requested = minAgents
	}
	return requested
}

func countStandardsViolations(targets []standardsFixMultiScenario) int {
	total := 0
	for _, target := range targets {
		total += len(target.Violations)
	}
	return total
}

func collectStandardsIssueIDs(targets []standardsFixMultiScenario) []string {
	seen := make(map[string]struct{})
	var ids []string
	for _, target := range targets {
		for _, violation := range target.Violations {
			if violation.ID == "" {
				continue
			}
			if _, exists := seen[violation.ID]; exists {
				continue
			}
			seen[violation.ID] = struct{}{}
			ids = append(ids, violation.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func cloneStandardsTargets(targets []standardsFixMultiScenario) []standardsFixMultiScenario {
	cloned := make([]standardsFixMultiScenario, 0, len(targets))
	for _, target := range targets {
		copyTarget := standardsFixMultiScenario{
			Scenario:     target.Scenario,
			ScenarioPath: target.ScenarioPath,
			Violations:   append([]StandardsViolation(nil), target.Violations...),
		}
		cloned = append(cloned, copyTarget)
	}
	return cloned
}

func splitStandardsTargets(targets []standardsFixMultiScenario, agentCount int) [][]standardsFixMultiScenario {
	total := countStandardsViolations(targets)
	if total == 0 {
		return nil
	}
	if agentCount <= 1 {
		return [][]standardsFixMultiScenario{cloneStandardsTargets(targets)}
	}
	if agentCount > maxBulkFixAgents {
		agentCount = maxBulkFixAgents
	}
	if agentCount > total {
		agentCount = total
	}
	chunkSize := (total + agentCount - 1) / agentCount
	if chunkSize < 1 {
		chunkSize = 1
	}
	if chunkSize > maxIssuesPerAgent {
		chunkSize = maxIssuesPerAgent
	}

	var groups [][]standardsFixMultiScenario
	current := make(map[string]*standardsFixMultiScenario)
	order := make([]string, 0)
	countInGroup := 0

	flushGroup := func() {
		if len(current) == 0 {
			return
		}
		group := make([]standardsFixMultiScenario, 0, len(order))
		for _, scenario := range order {
			entry := current[scenario]
			group = append(group, standardsFixMultiScenario{
				Scenario:     entry.Scenario,
				ScenarioPath: entry.ScenarioPath,
				Violations:   append([]StandardsViolation(nil), entry.Violations...),
			})
		}
		groups = append(groups, group)
		current = make(map[string]*standardsFixMultiScenario)
		order = make([]string, 0)
		countInGroup = 0
	}

	for _, target := range targets {
		for _, violation := range target.Violations {
			if countInGroup >= chunkSize && len(groups)+1 < agentCount {
				flushGroup()
			}
			entry, exists := current[target.Scenario]
			if !exists {
				entry = &standardsFixMultiScenario{
					Scenario:     target.Scenario,
					ScenarioPath: target.ScenarioPath,
					Violations:   []StandardsViolation{},
				}
				current[target.Scenario] = entry
				order = append(order, target.Scenario)
			}
			entry.Violations = append(entry.Violations, violation)
			countInGroup++
		}
	}
	flushGroup()
	return groups
}

func countVulnerabilityFindings(targets []vulnerabilityFixMultiScenario) int {
	total := 0
	for _, target := range targets {
		total += len(target.Findings)
	}
	return total
}

func collectVulnerabilityIssueIDs(targets []vulnerabilityFixMultiScenario) []string {
	seen := make(map[string]struct{})
	var ids []string
	for _, target := range targets {
		for _, finding := range target.Findings {
			if finding.ID == "" {
				continue
			}
			if _, exists := seen[finding.ID]; exists {
				continue
			}
			seen[finding.ID] = struct{}{}
			ids = append(ids, finding.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func cloneVulnerabilityTargets(targets []vulnerabilityFixMultiScenario) []vulnerabilityFixMultiScenario {
	cloned := make([]vulnerabilityFixMultiScenario, 0, len(targets))
	for _, target := range targets {
		copyTarget := vulnerabilityFixMultiScenario{
			Scenario:     target.Scenario,
			ScenarioPath: target.ScenarioPath,
			Findings:     append([]StoredVulnerability(nil), target.Findings...),
		}
		cloned = append(cloned, copyTarget)
	}
	return cloned
}

func splitVulnerabilityTargets(targets []vulnerabilityFixMultiScenario, agentCount int) [][]vulnerabilityFixMultiScenario {
	total := countVulnerabilityFindings(targets)
	if total == 0 {
		return nil
	}
	if agentCount <= 1 {
		return [][]vulnerabilityFixMultiScenario{cloneVulnerabilityTargets(targets)}
	}
	if agentCount > maxBulkFixAgents {
		agentCount = maxBulkFixAgents
	}
	if agentCount > total {
		agentCount = total
	}
	chunkSize := (total + agentCount - 1) / agentCount
	if chunkSize < 1 {
		chunkSize = 1
	}
	if chunkSize > maxIssuesPerAgent {
		chunkSize = maxIssuesPerAgent
	}

	var groups [][]vulnerabilityFixMultiScenario
	current := make(map[string]*vulnerabilityFixMultiScenario)
	order := make([]string, 0)
	countInGroup := 0

	flushGroup := func() {
		if len(current) == 0 {
			return
		}
		group := make([]vulnerabilityFixMultiScenario, 0, len(order))
		for _, scenario := range order {
			entry := current[scenario]
			group = append(group, vulnerabilityFixMultiScenario{
				Scenario:     entry.Scenario,
				ScenarioPath: entry.ScenarioPath,
				Findings:     append([]StoredVulnerability(nil), entry.Findings...),
			})
		}
		groups = append(groups, group)
		current = make(map[string]*vulnerabilityFixMultiScenario)
		order = make([]string, 0)
		countInGroup = 0
	}

	for _, target := range targets {
		for _, finding := range target.Findings {
			if countInGroup >= chunkSize && len(groups)+1 < agentCount {
				flushGroup()
			}
			entry, exists := current[target.Scenario]
			if !exists {
				entry = &vulnerabilityFixMultiScenario{
					Scenario:     target.Scenario,
					ScenarioPath: target.ScenarioPath,
					Findings:     []StoredVulnerability{},
				}
				current[target.Scenario] = entry
				order = append(order, target.Scenario)
			}
			entry.Findings = append(entry.Findings, finding)
			countInGroup++
		}
	}
	flushGroup()
	return groups
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

func appendUserInstructions(basePrompt, extra string) string {
	extra = strings.TrimSpace(extra)
	if extra == "" {
		return basePrompt
	}
	var builder strings.Builder
	builder.WriteString(basePrompt)
	builder.WriteString("\n\n## Additional Guidance\n")
	builder.WriteString(extra)
	builder.WriteString("\n")
	return builder.String()
}
