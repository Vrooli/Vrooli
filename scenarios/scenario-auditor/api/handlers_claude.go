package main

import (
	"encoding/json"
	"fmt"
	"net/http"
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

const (
	maxBulkFixAgents  = 10
	maxIssuesPerAgent = 50
)

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
		selected, collectedIDs := selectStoredFindings(violations, issueIDs, func(v StandardsViolation) string { return v.ID })
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
		selected, collectedIDs := selectStoredFindings(available, issueIDs, func(v StoredVulnerability) string { return v.ID })
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
	return buildBulkFixPlan(req, plan, extraPrompt, selectedModel, standardsPlanAdapter)
}

func buildVulnerabilityPlan(req *claudeFixRequest, plan *fixPlan, extraPrompt, selectedModel string) (*fixPlan, error) {
	return buildBulkFixPlan(req, plan, extraPrompt, selectedModel, vulnerabilityPlanAdapter)
}

type bulkFixPlanAdapter[T any] struct {
	prepare   func([]claudeFixTarget) ([]T, []string, error)
	count     func([]T) int
	scenarios func([]T) []string
	split     func([]T, int) [][]T
	prompt    func([]T, string) (string, string, map[string]string, error)
	issueIDs  func([]T) []string
	action    string
	noun      string
}

var standardsPlanAdapter = bulkFixPlanAdapter[standardsFixMultiScenario]{
	prepare:   prepareBulkStandardsTargets,
	count:     countStandardsViolations,
	scenarios: extractStandardsScenarioNames,
	split:     splitStandardsTargets,
	prompt:    buildMultiStandardsFixPrompt,
	issueIDs:  collectStandardsIssueIDs,
	action:    agentActionStandardsFix,
	noun:      "standards violations",
}

var vulnerabilityPlanAdapter = bulkFixPlanAdapter[vulnerabilityFixMultiScenario]{
	prepare:   prepareBulkVulnerabilityTargets,
	count:     countVulnerabilityFindings,
	scenarios: extractVulnerabilityScenarioNames,
	split:     splitVulnerabilityTargets,
	prompt:    buildMultiVulnerabilityFixPrompt,
	issueIDs:  collectVulnerabilityIssueIDs,
	action:    agentActionVulnerabilityFix,
	noun:      "vulnerabilities",
}

func buildBulkFixPlan[T any](
	req *claudeFixRequest,
	plan *fixPlan,
	extraPrompt string,
	selectedModel string,
	adapter bulkFixPlanAdapter[T],
) (*fixPlan, error) {
	targets, _, err := adapter.prepare(req.Targets)
	if err != nil {
		return nil, newPlanError(http.StatusBadRequest, err.Error(), err)
	}
	total := adapter.count(targets)
	if total == 0 {
		return nil, newPlanError(http.StatusBadRequest, fmt.Sprintf("No matching %s found to fix", adapter.noun), nil)
	}
	scenarioNames := adapter.scenarios(targets)
	plan.Scenarios = scenarioNames
	plan.IssueCount = total
	requestedAgents := clampAgentCount(req.AgentCount, total)
	if requestedAgents > 1 {
		groups := adapter.split(targets, requestedAgents)
		if len(groups) == 0 {
			return nil, newPlanError(http.StatusInternalServerError, fmt.Sprintf("Failed to partition %s for multi-agent fix", adapter.noun), nil)
		}
		for idx, group := range groups {
			prompt, groupLabel, metadata, err := adapter.prompt(group, extraPrompt)
			if err != nil {
				return nil, newPlanError(http.StatusInternalServerError, "Failed to build agent prompt", err)
			}
			metadata = ensureMetadata(metadata)
			groupScenarios := adapter.scenarios(group)
			metadata["fix_type"] = plan.FixType
			metadata["scenarios"] = strings.Join(groupScenarios, ",")
			metadata["batch_index"] = fmt.Sprintf("%d/%d", idx+1, len(groups))
			metadata["issues_per_agent_cap"] = fmt.Sprintf("%d", maxIssuesPerAgent)
			metadata["model"] = selectedModel
			if extraPrompt != "" {
				metadata["user_instructions"] = extraPrompt
			}
			batchIDs := adapter.issueIDs(group)
			metadata["issue_count"] = fmt.Sprintf("%d", len(batchIDs))
			label := groupLabel
			if len(groups) > 1 {
				label = fmt.Sprintf("%s (batch %d/%d)", groupLabel, idx+1, len(groups))
			}
			plan.AgentEntries = append(plan.AgentEntries, fixAgentEntry{
				Config: AgentStartConfig{
					Label: label, Name: label, Action: adapter.action, Scenario: "multi",
					IssueIDs: batchIDs, Prompt: prompt, Model: selectedModel, Metadata: metadata,
				},
				ErrorLabel: fmt.Sprintf("Failed to start agent batch %d", idx+1), ScenarioSet: groupScenarios,
			})
		}
		plan.Message = fmt.Sprintf("Started %d agents to address %d %s across %d scenario(s)", len(plan.AgentEntries), total, adapter.noun, len(scenarioNames))
		return plan, nil
	}

	prompt, label, metadata, err := adapter.prompt(targets, extraPrompt)
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
	metadata["issue_count"] = fmt.Sprintf("%d", total)
	plan.AgentEntries = append(plan.AgentEntries, fixAgentEntry{
		Config: AgentStartConfig{
			Label: label, Name: fmt.Sprintf("%s (multi)", label), Action: adapter.action,
			Scenario: "multi", IssueIDs: adapter.issueIDs(targets), Prompt: prompt, Model: selectedModel, Metadata: metadata,
		},
		ScenarioSet: scenarioNames,
	})
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

func selectStoredFindings[T any](all []T, ids []string, idOf func(T) string) ([]T, []string) {
	if len(all) == 0 {
		return nil, nil
	}
	if len(ids) == 0 {
		result := append([]T(nil), all...)
		collected := make([]string, 0, len(result))
		for _, item := range result {
			if id := idOf(item); id != "" {
				collected = append(collected, id)
			}
		}
		return result, collected
	}
	allowed := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		allowed[id] = struct{}{}
	}
	var filtered []T
	var collected []string
	for _, item := range all {
		id := idOf(item)
		if _, ok := allowed[id]; ok {
			filtered = append(filtered, item)
			if id != "" {
				collected = append(collected, id)
			}
		}
	}
	return filtered, collected
}

func prepareBulkStandardsTargets(targets []claudeFixTarget) ([]standardsFixMultiScenario, []string, error) {
	return prepareBulkTargets[StandardsViolation, standardsFixMultiScenario](
		targets,
		standardsStore.GetViolations,
		func(all []StandardsViolation, ids []string) ([]StandardsViolation, []string) {
			return selectStoredFindings(all, ids, func(v StandardsViolation) string { return v.ID })
		},
		func(scenario, path string, findings []StandardsViolation) standardsFixMultiScenario {
			return standardsFixMultiScenario{Scenario: scenario, ScenarioPath: path, Violations: findings}
		},
		func(group *standardsFixMultiScenario, findings []StandardsViolation) {
			group.Violations = append(group.Violations, findings...)
		},
		"standards violations",
	)
}

func prepareBulkVulnerabilityTargets(targets []claudeFixTarget) ([]vulnerabilityFixMultiScenario, []string, error) {
	return prepareBulkTargets[StoredVulnerability, vulnerabilityFixMultiScenario](
		targets,
		vulnStore.GetVulnerabilities,
		func(all []StoredVulnerability, ids []string) ([]StoredVulnerability, []string) {
			return selectStoredFindings(all, ids, func(v StoredVulnerability) string { return v.ID })
		},
		func(scenario, path string, findings []StoredVulnerability) vulnerabilityFixMultiScenario {
			return vulnerabilityFixMultiScenario{Scenario: scenario, ScenarioPath: path, Findings: findings}
		},
		func(group *vulnerabilityFixMultiScenario, findings []StoredVulnerability) {
			group.Findings = append(group.Findings, findings...)
		},
		"vulnerabilities",
	)
}

func prepareBulkTargets[T any, R any](
	targets []claudeFixTarget,
	load func(string) []T,
	selectFindings func([]T, []string) ([]T, []string),
	newGroup func(scenario, scenarioPath string, findings []T) R,
	appendFindings func(*R, []T),
	label string,
) ([]R, []string, error) {
	var result []R
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
		selected, ids := selectFindings(load(scenario), normaliseIDs(target.IssueIDs))
		if len(selected) == 0 {
			continue
		}
		if idx, exists := indexByScenario[scenario]; exists {
			appendFindings(&result[idx], selected)
		} else {
			indexByScenario[scenario] = len(result)
			result = append(result, newGroup(scenario, scenarioPath, selected))
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
		return nil, nil, fmt.Errorf("no matching %s found for provided targets", label)
	}
	sort.Strings(collectedIDs)
	return result, collectedIDs, nil
}

func resolveScenarioPath(scenario string) (string, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return "", fmt.Errorf("scenario name is empty")
	}
	ctx, err := repoContext()
	if err != nil {
		return "", err
	}
	return ctx.ResolveScenarioPath(scenario)
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
	return countFixTargets(targets, standardsFixTargetOps)
}

func collectStandardsIssueIDs(targets []standardsFixMultiScenario) []string {
	return collectFixTargetIssueIDs(targets, standardsFixTargetOps)
}

func splitStandardsTargets(targets []standardsFixMultiScenario, agentCount int) [][]standardsFixMultiScenario {
	return splitFixTargets(targets, agentCount, standardsFixTargetOps)
}

func countVulnerabilityFindings(targets []vulnerabilityFixMultiScenario) int {
	return countFixTargets(targets, vulnerabilityFixTargetOps)
}

func collectVulnerabilityIssueIDs(targets []vulnerabilityFixMultiScenario) []string {
	return collectFixTargetIssueIDs(targets, vulnerabilityFixTargetOps)
}

func splitVulnerabilityTargets(targets []vulnerabilityFixMultiScenario, agentCount int) [][]vulnerabilityFixMultiScenario {
	return splitFixTargets(targets, agentCount, vulnerabilityFixTargetOps)
}

type fixTargetOps[T any, G any] struct {
	items    func(G) []T
	scenario func(G) string
	path     func(G) string
	id       func(T) string
	make     func(scenario, path string, items []T) G
	append   func(*G, T)
}

var standardsFixTargetOps = fixTargetOps[StandardsViolation, standardsFixMultiScenario]{
	items:    func(target standardsFixMultiScenario) []StandardsViolation { return target.Violations },
	scenario: func(target standardsFixMultiScenario) string { return target.Scenario },
	path:     func(target standardsFixMultiScenario) string { return target.ScenarioPath },
	id:       func(item StandardsViolation) string { return item.ID },
	make: func(scenario, path string, items []StandardsViolation) standardsFixMultiScenario {
		return standardsFixMultiScenario{Scenario: scenario, ScenarioPath: path, Violations: items}
	},
	append: func(target *standardsFixMultiScenario, item StandardsViolation) {
		target.Violations = append(target.Violations, item)
	},
}

var vulnerabilityFixTargetOps = fixTargetOps[StoredVulnerability, vulnerabilityFixMultiScenario]{
	items:    func(target vulnerabilityFixMultiScenario) []StoredVulnerability { return target.Findings },
	scenario: func(target vulnerabilityFixMultiScenario) string { return target.Scenario },
	path:     func(target vulnerabilityFixMultiScenario) string { return target.ScenarioPath },
	id:       func(item StoredVulnerability) string { return item.ID },
	make: func(scenario, path string, items []StoredVulnerability) vulnerabilityFixMultiScenario {
		return vulnerabilityFixMultiScenario{Scenario: scenario, ScenarioPath: path, Findings: items}
	},
	append: func(target *vulnerabilityFixMultiScenario, item StoredVulnerability) {
		target.Findings = append(target.Findings, item)
	},
}

func countFixTargets[T any, G any](targets []G, ops fixTargetOps[T, G]) int {
	total := 0
	for _, target := range targets {
		total += len(ops.items(target))
	}
	return total
}

func collectFixTargetIssueIDs[T any, G any](targets []G, ops fixTargetOps[T, G]) []string {
	seen := make(map[string]struct{})
	var ids []string
	for _, target := range targets {
		for _, item := range ops.items(target) {
			id := ops.id(item)
			if id == "" {
				continue
			}
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func extractFixTargetScenarios[G any](targets []G, scenario func(G) string) []string {
	set := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		set[scenario(target)] = struct{}{}
	}
	return sortedKeys(set)
}

func splitFixTargets[T any, G any](targets []G, agentCount int, ops fixTargetOps[T, G]) [][]G {
	total := countFixTargets(targets, ops)
	if total == 0 {
		return nil
	}
	if agentCount <= 1 {
		cloned := make([]G, 0, len(targets))
		for _, target := range targets {
			cloned = append(cloned, ops.make(ops.scenario(target), ops.path(target), append([]T(nil), ops.items(target)...)))
		}
		return [][]G{cloned}
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

	var groups [][]G
	current := make(map[string]*G)
	order := make([]string, 0)
	countInGroup := 0
	flushGroup := func() {
		if len(current) == 0 {
			return
		}
		group := make([]G, 0, len(order))
		for _, scenario := range order {
			entry := current[scenario]
			group = append(group, ops.make(scenario, ops.path(*entry), append([]T(nil), ops.items(*entry)...)))
		}
		groups = append(groups, group)
		current = make(map[string]*G)
		order = make([]string, 0)
		countInGroup = 0
	}

	for _, target := range targets {
		for _, item := range ops.items(target) {
			if countInGroup >= chunkSize && len(groups)+1 < agentCount {
				flushGroup()
			}
			scenario := ops.scenario(target)
			entry, exists := current[scenario]
			if !exists {
				group := ops.make(scenario, ops.path(target), nil)
				entry = &group
				current[scenario] = entry
				order = append(order, scenario)
			}
			ops.append(entry, item)
			countInGroup++
		}
	}
	flushGroup()
	return groups
}

func extractStandardsScenarioNames(targets []standardsFixMultiScenario) []string {
	return extractFixTargetScenarios(targets, standardsFixTargetOps.scenario)
}

func extractVulnerabilityScenarioNames(targets []vulnerabilityFixMultiScenario) []string {
	return extractFixTargetScenarios(targets, vulnerabilityFixTargetOps.scenario)
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
