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
	Model        string            `json:"model"`
}

type claudeFixTarget struct {
	Scenario string   `json:"scenario"`
	IssueIDs []string `json:"issue_ids"`
}

const maxBulkFixAgents = 10
const maxIssuesPerAgent = 50

type fixPlan struct {
	FixType           string
	Model             string
	ExtraPrompt       string
	IssueCount        int
	IssuesPerAgentCap int
	Scenarios         []string
	AgentEntries      []fixAgentEntry
	Message           string
}

type fixAgentEntry struct {
	Config      AgentStartConfig
	ErrorLabel  string
	ScenarioSet []string
}

type fixPlanError struct {
	status  int
	message string
	err     error
}

func (e *fixPlanError) Error() string {
	return e.message
}

func newPlanError(status int, message string, err error) *fixPlanError {
	return &fixPlanError{status: status, message: message, err: err}
}

func triggerClaudeFixHandler(w http.ResponseWriter, r *http.Request) {

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
	selectedModel := normalizeAgentModel(req.Model)

	plan, err := buildFixPlan(&req, fixType, issueIDs, extraPrompt, selectedModel)
	if err != nil {
		if pe, ok := err.(*fixPlanError); ok {
			HTTPError(w, pe.message, pe.status, pe.err)
			return
		}
		HTTPError(w, "Failed to build agent prompt", http.StatusInternalServerError, err)
		return
	}

	startedAgents := make([]*AgentInfo, 0, len(plan.AgentEntries))
	fixIDs := make([]string, 0, len(plan.AgentEntries))
	for idx, entry := range plan.AgentEntries {
		agentInfo, startErr := agentManager.StartAgent(entry.Config)
		if startErr != nil {
			errorLabel := entry.ErrorLabel
			if errorLabel == "" {
				errorLabel = "Failed to start agent"
			}
			HTTPError(w, errorLabel, http.StatusInternalServerError, startErr)
			return
		}
		startedAgents = append(startedAgents, agentInfo)
		fixIDs = append(fixIDs, agentInfo.ID)
		if idx == 0 {
			// ensure first agent timestamp used if needed later
		}
	}

	response := map[string]any{
		"success":              true,
		"message":              plan.Message,
		"scenarios":            plan.Scenarios,
		"agent_count":          len(startedAgents),
		"fix_ids":              fixIDs,
		"model":                plan.Model,
		"issues_per_agent_cap": plan.IssuesPerAgentCap,
	}

	if len(startedAgents) > 0 {
		response["fix_id"] = startedAgents[0].ID
	}

	if plan.ExtraPrompt != "" {
		response["user_instructions"] = plan.ExtraPrompt
	}

	if plan.IssueCount > 0 {
		response["issue_count"] = plan.IssueCount
	}

	if len(startedAgents) == 1 {
		agentInfo := startedAgents[0]
		response["agent"] = agentInfo
		response["started_at"] = agentInfo.StartedAt
	} else {
		response["agents"] = startedAgents
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode trigger fix response", err)
	}
}

func previewClaudeFixHandler(w http.ResponseWriter, r *http.Request) {
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
	selectedModel := normalizeAgentModel(req.Model)

	plan, err := buildFixPlan(&req, fixType, issueIDs, extraPrompt, selectedModel)
	if err != nil {
		if pe, ok := err.(*fixPlanError); ok {
			HTTPError(w, pe.message, pe.status, pe.err)
			return
		}
		HTTPError(w, "Failed to build agent prompt", http.StatusInternalServerError, err)
		return
	}

	prompts := make([]map[string]any, 0, len(plan.AgentEntries))
	for _, entry := range plan.AgentEntries {
		scenarios := entry.ScenarioSet
		if len(scenarios) == 0 && entry.Config.Metadata != nil {
			if raw := entry.Config.Metadata["scenarios"]; raw != "" {
				scenarios = strings.Split(raw, ",")
			}
		}
		promptInfo := map[string]any{
			"label":     entry.Config.Label,
			"scenario":  entry.Config.Scenario,
			"action":    entry.Config.Action,
			"prompt":    entry.Config.Prompt,
			"issue_ids": entry.Config.IssueIDs,
		}
		if len(scenarios) > 0 {
			promptInfo["scenarios"] = scenarios
		}
		if entry.Config.Metadata != nil {
			promptInfo["metadata"] = entry.Config.Metadata
		}
		prompts = append(prompts, promptInfo)
	}

	response := map[string]any{
		"success":              true,
		"message":              plan.Message,
		"prompts":              prompts,
		"scenarios":            plan.Scenarios,
		"agent_count":          len(plan.AgentEntries),
		"issue_count":          plan.IssueCount,
		"issues_per_agent_cap": plan.IssuesPerAgentCap,
		"model":                plan.Model,
		"fix_type":             plan.FixType,
	}
	if plan.ExtraPrompt != "" {
		response["user_instructions"] = plan.ExtraPrompt
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		HTTPError(w, "Failed to encode preview response", http.StatusInternalServerError, err)
	}
}

func buildFixPlan(req *claudeFixRequest, fixType string, issueIDs []string, extraPrompt, selectedModel string) (*fixPlan, error) {
	plan := &fixPlan{
		FixType:           fixType,
		Model:             selectedModel,
		ExtraPrompt:       extraPrompt,
		IssuesPerAgentCap: maxIssuesPerAgent,
	}

	if len(req.Targets) > 0 {
		switch fixType {
		case "standards":
			return buildStandardsPlan(req, plan, extraPrompt, selectedModel)
		case "vulnerabilities":
			return buildVulnerabilityPlan(req, plan, extraPrompt, selectedModel)
		default:
			return nil, newPlanError(http.StatusBadRequest, "Unsupported fix_type", nil)
		}
	}

	scenarioName := strings.TrimSpace(req.ScenarioName)
	if scenarioName == "" {
		return nil, newPlanError(http.StatusBadRequest, "scenario_name is required", nil)
	}

	scenarioPath, err := resolveScenarioPath(scenarioName)
	if err != nil {
		return nil, newPlanError(http.StatusNotFound, "Scenario not found", err)
	}

	plan.Scenarios = []string{scenarioName}

	var (
		prompt   string
		label    string
		metadata map[string]string
		action   string
		ids      []string
	)

	switch fixType {
	case "standards":
		violations := standardsStore.GetViolations(scenarioName)
		selected, collectedIDs := selectStandardsViolations(violations, issueIDs)
		if len(selected) == 0 {
			return nil, newPlanError(http.StatusBadRequest, "No matching standards violations found to fix", nil)
		}
		if len(issueIDs) > 0 {
			if missing := findMissingIDs(issueIDs, collectedIDs); len(missing) > 0 {
				return nil, newPlanError(http.StatusBadRequest, fmt.Sprintf("Unknown standards violation IDs: %s", strings.Join(missing, ", ")), nil)
			}
		}
		prompt, label, metadata, err = buildStandardsFixPrompt(scenarioName, scenarioPath, selected, collectedIDs)
		ids = collectedIDs
		action = agentActionStandardsFix
	case "vulnerabilities":
		available := vulnStore.GetVulnerabilities(scenarioName)
		selected, collectedIDs := selectStoredVulnerabilities(available, issueIDs)
		if len(selected) == 0 {
			return nil, newPlanError(http.StatusBadRequest, "No matching vulnerabilities found to fix", nil)
		}
		if len(issueIDs) > 0 {
			if missing := findMissingIDs(issueIDs, collectedIDs); len(missing) > 0 {
				return nil, newPlanError(http.StatusBadRequest, fmt.Sprintf("Unknown vulnerability IDs: %s", strings.Join(missing, ", ")), nil)
			}
		}
		prompt, label, metadata, err = buildVulnerabilityFixPrompt(scenarioName, scenarioPath, selected, collectedIDs)
		ids = collectedIDs
		action = agentActionVulnerabilityFix
	default:
		return nil, newPlanError(http.StatusBadRequest, "Unsupported fix_type", nil)
	}

	if err == nil && extraPrompt != "" {
		prompt = appendUserInstructions(prompt, extraPrompt)
	}

	if err != nil {
		return nil, newPlanError(http.StatusInternalServerError, "Failed to build agent prompt", err)
	}

	if metadata == nil {
		metadata = make(map[string]string)
	}
	metadata["scenario"] = scenarioName
	metadata["scenario_path"] = scenarioPath
	metadata["fix_type"] = fixType
	metadata["issues_per_agent_cap"] = fmt.Sprintf("%d", maxIssuesPerAgent)
	metadata["model"] = selectedModel
	if extraPrompt != "" {
		metadata["user_instructions"] = extraPrompt
	}
	if len(ids) > 0 {
		metadata["issue_count"] = fmt.Sprintf("%d", len(ids))
	}

	plan.IssueCount = len(ids)

	entry := fixAgentEntry{
		Config: AgentStartConfig{
			Label:    label,
			Name:     fmt.Sprintf("%s (%s)", label, scenarioName),
			Action:   action,
			Scenario: scenarioName,
			IssueIDs: ids,
			Prompt:   prompt,
			Model:    selectedModel,
			Metadata: metadata,
		},
		ScenarioSet: []string{scenarioName},
	}
	plan.AgentEntries = append(plan.AgentEntries, entry)
	plan.Message = fmt.Sprintf("Started %s for %s", label, scenarioName)

	return plan, nil
}

func buildStandardsPlan(req *claudeFixRequest, plan *fixPlan, extraPrompt, selectedModel string) (*fixPlan, error) {
	multiTargets, _, err := prepareBulkStandardsTargets(req.Targets)
	if err != nil {
		return nil, newPlanError(http.StatusBadRequest, err.Error(), err)
	}

	totalViolations := countStandardsViolations(multiTargets)
	if totalViolations == 0 {
		return nil, newPlanError(http.StatusBadRequest, "No matching standards violations found to fix", nil)
	}

	scenarioNames := extractStandardsScenarioNames(multiTargets)
	plan.Scenarios = scenarioNames
	plan.IssueCount = totalViolations

	requestedAgents := clampAgentCount(req.AgentCount, totalViolations)
	if requestedAgents > 1 {
		groups := splitStandardsTargets(multiTargets, requestedAgents)
		if len(groups) == 0 {
			return nil, newPlanError(http.StatusInternalServerError, "Failed to partition standards violations for multi-agent fix", nil)
		}
		for idx, group := range groups {
			prompt, groupLabel, metadata, err := buildMultiStandardsFixPrompt(group, extraPrompt)
			if err != nil {
				return nil, newPlanError(http.StatusInternalServerError, "Failed to build agent prompt", err)
			}
			metadata = ensureMetadata(metadata)
			groupScenarios := extractStandardsScenarioNames(group)
			metadata["fix_type"] = plan.FixType
			metadata["scenarios"] = strings.Join(groupScenarios, ",")
			metadata["batch_index"] = fmt.Sprintf("%d/%d", idx+1, len(groups))
			metadata["issues_per_agent_cap"] = fmt.Sprintf("%d", maxIssuesPerAgent)
			metadata["model"] = selectedModel
			if extraPrompt != "" {
				metadata["user_instructions"] = extraPrompt
			}
			batchIDs := collectStandardsIssueIDs(group)
			metadata["issue_count"] = fmt.Sprintf("%d", len(batchIDs))

			labelWithBatch := groupLabel
			if len(groups) > 1 {
				labelWithBatch = fmt.Sprintf("%s (batch %d/%d)", groupLabel, idx+1, len(groups))
			}

			entry := fixAgentEntry{
				Config: AgentStartConfig{
					Label:    labelWithBatch,
					Name:     labelWithBatch,
					Action:   agentActionStandardsFix,
					Scenario: "multi",
					IssueIDs: batchIDs,
					Prompt:   prompt,
					Model:    selectedModel,
					Metadata: metadata,
				},
				ErrorLabel:  fmt.Sprintf("Failed to start agent batch %d", idx+1),
				ScenarioSet: groupScenarios,
			}
			plan.AgentEntries = append(plan.AgentEntries, entry)
		}

		plan.Message = fmt.Sprintf("Started %d agents to address %d violations across %d scenario(s)", len(plan.AgentEntries), totalViolations, len(scenarioNames))
		return plan, nil
	}

	prompt, label, metadata, err := buildMultiStandardsFixPrompt(multiTargets, extraPrompt)
	if err != nil {
		return nil, newPlanError(http.StatusInternalServerError, "Failed to build agent prompt", err)
	}

	metadata = ensureMetadata(metadata)
	metadata["fix_type"] = plan.FixType
	metadata["scenarios"] = strings.Join(scenarioNames, ",")
	metadata["issues_per_agent_cap"] = fmt.Sprintf("%d", maxIssuesPerAgent)
	metadata["model"] = selectedModel
	if extraPrompt != "" {
		metadata["user_instructions"] = extraPrompt
	}
	metadata["issue_count"] = fmt.Sprintf("%d", totalViolations)

	entry := fixAgentEntry{
		Config: AgentStartConfig{
			Label:    label,
			Name:     fmt.Sprintf("%s (multi)", label),
			Action:   agentActionStandardsFix,
			Scenario: "multi",
			IssueIDs: collectStandardsIssueIDs(multiTargets),
			Prompt:   prompt,
			Model:    selectedModel,
			Metadata: metadata,
		},
		ScenarioSet: scenarioNames,
	}

	plan.AgentEntries = append(plan.AgentEntries, entry)
	plan.Message = fmt.Sprintf("Started %s across %d scenario(s)", label, len(scenarioNames))
	return plan, nil
}

func buildVulnerabilityPlan(req *claudeFixRequest, plan *fixPlan, extraPrompt, selectedModel string) (*fixPlan, error) {
	multiTargets, _, err := prepareBulkVulnerabilityTargets(req.Targets)
	if err != nil {
		return nil, newPlanError(http.StatusBadRequest, err.Error(), err)
	}

	totalFindings := countVulnerabilityFindings(multiTargets)
	if totalFindings == 0 {
		return nil, newPlanError(http.StatusBadRequest, "No matching vulnerabilities found to fix", nil)
	}

	scenarioNames := extractVulnerabilityScenarioNames(multiTargets)
	plan.Scenarios = scenarioNames
	plan.IssueCount = totalFindings

	requestedAgents := clampAgentCount(req.AgentCount, totalFindings)
	if requestedAgents > 1 {
		groups := splitVulnerabilityTargets(multiTargets, requestedAgents)
		if len(groups) == 0 {
			return nil, newPlanError(http.StatusInternalServerError, "Failed to partition vulnerabilities for multi-agent fix", nil)
		}
		for idx, group := range groups {
			prompt, groupLabel, metadata, err := buildMultiVulnerabilityFixPrompt(group, extraPrompt)
			if err != nil {
				return nil, newPlanError(http.StatusInternalServerError, "Failed to build agent prompt", err)
			}
			metadata = ensureMetadata(metadata)
			groupScenarios := extractVulnerabilityScenarioNames(group)
			metadata["fix_type"] = plan.FixType
			metadata["scenarios"] = strings.Join(groupScenarios, ",")
			metadata["batch_index"] = fmt.Sprintf("%d/%d", idx+1, len(groups))
			metadata["issues_per_agent_cap"] = fmt.Sprintf("%d", maxIssuesPerAgent)
			metadata["model"] = selectedModel
			if extraPrompt != "" {
				metadata["user_instructions"] = extraPrompt
			}
			batchIDs := collectVulnerabilityIssueIDs(group)
			metadata["issue_count"] = fmt.Sprintf("%d", len(batchIDs))

			labelWithBatch := groupLabel
			if len(groups) > 1 {
				labelWithBatch = fmt.Sprintf("%s (batch %d/%d)", groupLabel, idx+1, len(groups))
			}

			entry := fixAgentEntry{
				Config: AgentStartConfig{
					Label:    labelWithBatch,
					Name:     labelWithBatch,
					Action:   agentActionVulnerabilityFix,
					Scenario: "multi",
					IssueIDs: batchIDs,
					Prompt:   prompt,
					Model:    selectedModel,
					Metadata: metadata,
				},
				ErrorLabel:  fmt.Sprintf("Failed to start agent batch %d", idx+1),
				ScenarioSet: groupScenarios,
			}
			plan.AgentEntries = append(plan.AgentEntries, entry)
		}

		plan.Message = fmt.Sprintf("Started %d agents to address %d vulnerabilities across %d scenario(s)", len(plan.AgentEntries), totalFindings, len(scenarioNames))
		return plan, nil
	}

	prompt, label, metadata, err := buildMultiVulnerabilityFixPrompt(multiTargets, extraPrompt)
	if err != nil {
		return nil, newPlanError(http.StatusInternalServerError, "Failed to build agent prompt", err)
	}

	metadata = ensureMetadata(metadata)
	metadata["fix_type"] = plan.FixType
	metadata["scenarios"] = strings.Join(scenarioNames, ",")
	metadata["issues_per_agent_cap"] = fmt.Sprintf("%d", maxIssuesPerAgent)
	metadata["model"] = selectedModel
	if extraPrompt != "" {
		metadata["user_instructions"] = extraPrompt
	}
	metadata["issue_count"] = fmt.Sprintf("%d", totalFindings)

	entry := fixAgentEntry{
		Config: AgentStartConfig{
			Label:    label,
			Name:     fmt.Sprintf("%s (multi)", label),
			Action:   agentActionVulnerabilityFix,
			Scenario: "multi",
			IssueIDs: collectVulnerabilityIssueIDs(multiTargets),
			Prompt:   prompt,
			Model:    selectedModel,
			Metadata: metadata,
		},
		ScenarioSet: scenarioNames,
	}

	plan.AgentEntries = append(plan.AgentEntries, entry)
	plan.Message = fmt.Sprintf("Started %s across %d scenario(s)", label, len(scenarioNames))
	return plan, nil
}

func ensureMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return make(map[string]string)
	}
	copy := make(map[string]string, len(metadata))
	for k, v := range metadata {
		copy[k] = v
	}
	return copy
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
	json.NewEncoder(w).Encode(map[string]any{
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

		scenarioPath, err := resolveScenarioPath(scenario)
		if err != nil {
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

		scenarioPath, err := resolveScenarioPath(scenario)
		if err != nil {
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

// resolveScenarioPath locates another scenario by joining the shared Vrooli root
// with the scenarios directory. The auditor root is discovered once via getVrooliRoot.
func resolveScenarioPath(scenario string) (string, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return "", fmt.Errorf("scenario name is empty")
	}

	if root := getVrooliRoot(); root != "" {
		candidate := filepath.Join(root, "scenarios", scenario)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("scenario %s not found", scenario)
}

func clampAgentCount(requested, total int) int {
	if total <= 0 {
		return 0
	}

	maxCapacity := maxBulkFixAgents * maxIssuesPerAgent
	if total > maxCapacity {
		total = maxCapacity
	}

	minAgents := (total + maxIssuesPerAgent - 1) / maxIssuesPerAgent
	if minAgents < 1 {
		minAgents = 1
	}

	if requested < minAgents {
		requested = minAgents
	}

	if requested > maxBulkFixAgents {
		requested = maxBulkFixAgents
	}

	if requested > total {
		requested = total
	}

	if requested < 1 {
		requested = 1
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
