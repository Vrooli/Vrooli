package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	re "scenario-auditor/internal/ruleengine"
	rulespkg "scenario-auditor/rules"

	"github.com/gorilla/mux"
)

// Persistent store for rule enabled/disabled status
type RuleStateStore struct {
	mu       sync.RWMutex
	states   map[string]bool
	filePath string
}

type ruleStateData struct {
	States     map[string]bool `json:"states"`
	LastUpdate time.Time       `json:"last_update"`
}

var ruleStateStore = initRuleStateStore()

func initRuleStateStore() *RuleStateStore {
	fmt.Fprintf(os.Stderr, "[INIT] Initializing rule state store...\n")
	store := &RuleStateStore{
		states: make(map[string]bool),
	}
	store.enablePersistence()
	return store
}

func (rs *RuleStateStore) enablePersistence() {
	logger := NewLogger()

	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}

	parentDir := filepath.Join(vrooliRoot, ".vrooli", "data")
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			logger.Error(fmt.Sprintf("Failed to create parent data directory %s", parentDir), err)
			logger.Info("Rule state store will operate in memory-only mode (no persistence)")
			return
		}
	}

	dataDir := filepath.Join(parentDir, "scenario-auditor")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Error(fmt.Sprintf("Failed to create scenario-auditor data directory %s", dataDir), err)
		logger.Info("Rule state store will operate in memory-only mode (no persistence)")
		return
	}

	rs.filePath = filepath.Join(dataDir, "rule-preferences.json")

	if err := rs.loadFromFile(); err != nil {
		logger.Error(fmt.Sprintf("Failed to load existing rule states from %s", rs.filePath), err)
	} else {
		if len(rs.states) > 0 {
			logger.Info(fmt.Sprintf("Loaded %d rule state preference entries", len(rs.states)))
		}
	}

	logger.Info(fmt.Sprintf("Rule state persistence enabled at: %s", rs.filePath))
}

func (rs *RuleStateStore) loadFromFile() error {
	if rs.filePath == "" {
		return nil
	}

	rs.mu.Lock()
	defer rs.mu.Unlock()

	data, err := os.ReadFile(rs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var stored ruleStateData
	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}

	if stored.States == nil {
		stored.States = make(map[string]bool)
	}

	rs.states = stored.States
	return nil
}

func (rs *RuleStateStore) saveToFileLocked() error {
	if rs.filePath == "" {
		return nil
	}

	payload := ruleStateData{
		States:     make(map[string]bool, len(rs.states)),
		LastUpdate: time.Now(),
	}
	for id, enabled := range rs.states {
		payload.States[id] = enabled
	}

	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := rs.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, bytes, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, rs.filePath)
}

func (rs *RuleStateStore) SetState(ruleID string, enabled bool) error {
	if ruleID == "" {
		return fmt.Errorf("rule ID is required")
	}

	rs.mu.Lock()
	if rs.states == nil {
		rs.states = make(map[string]bool)
	}
	rs.states[ruleID] = enabled
	err := rs.saveToFileLocked()
	rs.mu.Unlock()
	return err
}

func (rs *RuleStateStore) GetState(ruleID string) (bool, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	enabled, ok := rs.states[ruleID]
	return enabled, ok
}

func (rs *RuleStateStore) GetAllStates() map[string]bool {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	result := make(map[string]bool, len(rs.states))
	for id, enabled := range rs.states {
		result[id] = enabled
	}
	return result
}

type Rule struct {
	rulespkg.Rule
	Targets        []string                `json:"targets,omitempty"`
	TestStatus     *RuleTestStatus         `json:"test_status,omitempty"`
	Implementation re.ImplementationStatus `json:"implementation"`
}

type RuleCategory = re.Category

type RuleTestStatus struct {
	Total     int    `json:"total"`
	Passed    int    `json:"passed"`
	Failed    int    `json:"failed"`
	Warning   string `json:"warning,omitempty"`
	Error     string `json:"error,omitempty"`
	HasIssues bool   `json:"has_issues"`
}

// getRulesHandler returns the list of available rules and categories
func getRulesHandler(w http.ResponseWriter, r *http.Request) {
	logger := NewLogger()
	logger.Info("Fetching available rules")

	svc, err := ruleService()
	if err != nil {
		logger.Error("Failed to initialise rule service", err)
		HTTPError(w, "Failed to initialise rule engine", http.StatusInternalServerError, err)
		return
	}

	ruleInfos, err := svc.Load()
	if err != nil {
		logger.Error("Failed to load rules from service", err)
		HTTPError(w, "Failed to load rules", http.StatusInternalServerError, err)
		return
	}

	rules := make(map[string]Rule)
	states := ruleStateStore.GetAllStates()
	for id, info := range ruleInfos {
		rule := convertInfoToRule(info)
		if enabled, exists := states[id]; exists {
			rule.Enabled = enabled
		}
		rules[id] = rule
	}

	testStatuses := computeRuleTestStatuses(svc, ruleInfos)
	ruleStatusMap := make(map[string]RuleTestStatus)
	for id, rule := range rules {
		status, ok := testStatuses[id]
		if !ok {
			// If we couldn't compute a status (e.g. metadata only), mark as needing attention
			status = RuleTestStatus{
				Warning:   "Rule metadata missing source file",
				HasIssues: true,
			}
		}

		statusCopy := status
		rule.TestStatus = &statusCopy
		rules[id] = rule
		ruleStatusMap[id] = statusCopy
	}

	// Get categories
	categories := re.DefaultCategories()

	// Filter by category if specified
	categoryFilter := r.URL.Query().Get("category")

	filteredRules := make(map[string]Rule)
	if categoryFilter != "" {
		for id, rule := range rules {
			if rule.Category == categoryFilter {
				filteredRules[id] = rule
			}
		}
	} else {
		filteredRules = rules
	}

	// Return the response
	response := map[string]interface{}{
		"rules":         filteredRules,
		"categories":    categories,
		"count":         len(filteredRules),
		"total":         len(rules),
		"rule_statuses": ruleStatusMap,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode rules response", err)
		HTTPError(w, "Failed to encode response", http.StatusInternalServerError, err)
	}
}

func computeRuleTestStatuses(svc *re.Service, ruleInfos map[string]re.Info) map[string]RuleTestStatus {
	statuses := make(map[string]RuleTestStatus)
	if len(ruleInfos) == 0 {
		return statuses
	}

	for ruleID, ruleInfo := range ruleInfos {
		status := RuleTestStatus{}

		if !ruleInfo.Implementation.Valid {
			status.Error = ruleInfo.Implementation.Error
			if status.Error == "" {
				status.Error = "rule implementation failed to load"
			}
			status.Warning = "Rule implementation unavailable"
			status.HasIssues = true
			statuses[ruleID] = status
			continue
		}

		results, err := svc.RunTestsForInfo(ruleID, ruleInfo)
		if err != nil {
			status.Error = err.Error()
			status.HasIssues = true
		} else {
			status.Total = len(results)
			for _, result := range results {
				if result.Passed {
					status.Passed++
				} else {
					status.Failed++
				}
			}

			if status.Total == 0 {
				status.Warning = "Rule has no test cases"
				status.HasIssues = true
			} else if status.Failed > 0 {
				suffix := "s"
				if status.Failed == 1 {
					suffix = ""
				}
				status.Warning = fmt.Sprintf("%d failing test%s", status.Failed, suffix)
				status.HasIssues = true
			} else {
				status.HasIssues = false
			}
		}

		statuses[ruleID] = status
	}

	return statuses
}

// createRuleHandler creates an issue in app-issue-tracker for rule creation
func createRuleHandler(w http.ResponseWriter, r *http.Request) {
	logger := NewLogger()

	var req createRuleRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid request body", http.StatusBadRequest, err)
		return
	}

	// Validation
	if req.Name == "" || req.Description == "" {
		HTTPError(w, "name and description are required", http.StatusBadRequest, nil)
		return
	}

	if req.Category == "" {
		req.Category = "api"
	}

	if req.Severity == "" {
		req.Severity = "medium"
	}

	// Resolve app-issue-tracker port
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	trackerPort, err := resolveIssueTrackerPort(ctx)
	if err != nil {
		HTTPError(w, "Failed to locate app-issue-tracker", http.StatusServiceUnavailable, err)
		return
	}

	// Build payload
	payload, err := buildCreateRuleIssuePayload(req)
	if err != nil {
		HTTPError(w, "Failed to build issue payload", http.StatusInternalServerError, err)
		return
	}

	// Submit to app-issue-tracker
	result, err := submitIssueToTracker(ctx, trackerPort, payload)
	if err != nil {
		HTTPError(w, "Failed to create issue in tracker", http.StatusInternalServerError, err)
		return
	}

	// Build issue URL
	issueURL := ""
	if result.IssueID != "" {
		if uiPort, err := resolveIssueTrackerUIPort(ctx); err == nil {
			u := url.URL{
				Scheme: "http",
				Host:   fmt.Sprintf("localhost:%d", uiPort),
			}
			query := u.Query()
			query.Set("issue", result.IssueID)
			u.RawQuery = query.Encode()
			issueURL = u.String()
		} else {
			logger.Warn("Failed to resolve app-issue-tracker UI port", map[string]interface{}{"error": err.Error()})
		}
	}

	response := map[string]interface{}{
		"issueId":  result.IssueID,
		"issueUrl": issueURL,
		"message":  result.Message,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode create rule response", err)
	}
}

// updateRuleHandler updates an existing rule (not yet implemented)
func updateRuleHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Rule update not yet implemented", http.StatusNotImplemented, nil)
}

// deleteRuleHandler deletes a rule (not yet implemented)
func deleteRuleHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Rule deletion not yet implemented", http.StatusNotImplemented, nil)
}

// toggleRuleHandler enables/disables a rule
func toggleRuleHandler(w http.ResponseWriter, r *http.Request) {
	logger := NewLogger()

	// Extract rule ID from URL
	vars := mux.Vars(r)
	ruleID := vars["ruleId"]

	if ruleID == "" {
		HTTPError(w, "Rule ID is required", http.StatusBadRequest, nil)
		return
	}

	// Parse request body
	var toggleReq struct {
		Enabled bool `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&toggleReq); err != nil {
		HTTPError(w, "Invalid request body", http.StatusBadRequest, err)
		return
	}

	logger.Info("Toggling rule " + ruleID + " enabled: " + fmt.Sprintf("%v", toggleReq.Enabled))

	if err := ruleStateStore.SetState(ruleID, toggleReq.Enabled); err != nil {
		logger.Error("Failed to persist rule state", err)
		HTTPError(w, "Failed to persist rule state", http.StatusInternalServerError, err)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"success": true,
		"rule_id": ruleID,
		"enabled": toggleReq.Enabled,
		"message": fmt.Sprintf("Rule %s has been %s", ruleID, map[bool]string{true: "enabled", false: "disabled"}[toggleReq.Enabled]),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode toggle response", err)
		HTTPError(w, "Failed to encode response", http.StatusInternalServerError, err)
	}
}

// getRuleHandler gets a single rule by ID including file content
func getRuleHandler(w http.ResponseWriter, r *http.Request) {
	logger := NewLogger()

	// Extract rule ID from URL
	vars := mux.Vars(r)
	ruleID := vars["ruleId"]

	if ruleID == "" {
		HTTPError(w, "Rule ID is required", http.StatusBadRequest, nil)
		return
	}

	logger.Info("Fetching rule details for " + ruleID)

	svc, err := ruleService()
	if err != nil {
		logger.Error("Failed to initialise rule service", err)
		HTTPError(w, "Failed to initialise rule engine", http.StatusInternalServerError, err)
		return
	}

	ruleInfos, err := svc.Load()
	if err != nil {
		logger.Error("Failed to load rules from service", err)
		HTTPError(w, "Failed to load rule", http.StatusInternalServerError, err)
		return
	}

	ruleInfo, exists := ruleInfos[ruleID]
	if !exists {
		HTTPError(w, "Rule not found", http.StatusNotFound, nil)
		return
	}

	// Read the file content
	fileContent := ""
	if ruleInfo.FilePath != "" {
		content, err := os.ReadFile(ruleInfo.FilePath)
		if err != nil {
			logger.Error("Failed to read rule file", err)
		} else {
			fileContent = string(content)
		}
	}

	// Convert to rule and add file content
	rule := convertInfoToRule(ruleInfo)
	if enabled, ok := ruleStateStore.GetState(ruleID); ok {
		rule.Enabled = enabled
	}

	if status, ok := computeRuleTestStatuses(svc, map[string]re.Info{ruleID: ruleInfo})[ruleID]; ok {
		s := status
		rule.TestStatus = &s
	}

	executionInfo := re.BuildExecutionInfo(ruleInfo)

	response := map[string]interface{}{
		"rule":           rule,
		"file_content":   fileContent,
		"file_path":      ruleInfo.FilePath,
		"execution_info": executionInfo,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode rule response", err)
		HTTPError(w, "Failed to encode response", http.StatusInternalServerError, err)
	}
}

// getRuleCategoriesHandler returns just the categories (not yet implemented)
func getRuleCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Get categories not yet implemented", http.StatusNotImplemented, nil)
}

// createRuleWithAIHandler creates a rule using AI assistance
func createRuleWithAIHandler(w http.ResponseWriter, r *http.Request) {
	logger := NewLogger()

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Severity    string `json:"severity"`
		Motivation  string `json:"motivation"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid request body", http.StatusBadRequest, err)
		return
	}

	categories := re.DefaultCategories()
	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "api"
	}
	if _, ok := categories[category]; !ok {
		HTTPError(w, fmt.Sprintf("Invalid category: %s", category), http.StatusBadRequest, nil)
		return
	}

	severity := strings.ToLower(strings.TrimSpace(req.Severity))
	switch severity {
	case "critical", "high", "medium", "low":
		// accepted
	case "":
		severity = "medium"
	default:
		HTTPError(w, fmt.Sprintf("Invalid severity: %s", severity), http.StatusBadRequest, nil)
		return
	}

	spec := RuleCreationSpec{
		Name:        req.Name,
		Description: req.Description,
		Category:    category,
		Severity:    severity,
		Motivation:  req.Motivation,
	}

	prompt, label, metadata, err := buildRuleCreationPrompt(spec)
	if err != nil {
		HTTPError(w, err.Error(), http.StatusBadRequest, err)
		return
	}

	label = safeFallback(label, "Create rule")
	name := fmt.Sprintf("Create %s", safeFallback(spec.Name, metadata["rule_id"]))

	agentInfo, err := agentManager.StartAgent(AgentStartConfig{
		Label:    label,
		Name:     name,
		Action:   agentActionCreateRule,
		Prompt:   prompt,
		Model:    openRouterModel,
		Metadata: metadata,
	})
	if err != nil {
		HTTPError(w, "Failed to start rule creation agent", http.StatusInternalServerError, err)
		return
	}

	logger.Info(fmt.Sprintf("Started rule creation agent %s for %s", agentInfo.ID, metadata["rule_id"]))

	response := map[string]interface{}{
		"success":  true,
		"message":  fmt.Sprintf("Rule creation agent started for %s", safeFallback(spec.Name, metadata["rule_id"])),
		"agent":    agentInfo,
		"metadata": metadata,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode rule creation response", err)
		HTTPError(w, "Failed to encode response", http.StatusInternalServerError, err)
	}
}

// editRuleWithAIHandler edits a rule using AI assistance (not yet implemented)
func editRuleWithAIHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "AI rule editing not yet implemented", http.StatusNotImplemented, nil)
}

// testRuleHandler runs all test cases for a specific rule
func testRuleHandler(w http.ResponseWriter, r *http.Request) {
	logger := NewLogger()

	// Extract rule ID from URL
	vars := mux.Vars(r)
	ruleID := vars["ruleId"]

	if ruleID == "" {
		HTTPError(w, "Rule ID is required", http.StatusBadRequest, nil)
		return
	}

	logger.Info("Running tests for rule: " + ruleID)

	svc, err := ruleService()
	if err != nil {
		HTTPError(w, "Failed to initialise rule engine", http.StatusInternalServerError, err)
		return
	}

	ruleInfos, err := svc.Load()
	if err != nil {
		HTTPError(w, "Failed to load rules", http.StatusInternalServerError, err)
		return
	}

	ruleInfo, exists := ruleInfos[ruleID]
	if !exists {
		HTTPError(w, "Rule not found", http.StatusNotFound, nil)
		return
	}

	// Run all tests for the rule
	testResults, err := svc.RunTestsForInfo(ruleID, ruleInfo)
	if err != nil {
		HTTPError(w, "Failed to run tests", http.StatusInternalServerError, err)
		return
	}

	// Calculate summary statistics
	totalTests := len(testResults)
	passedTests := 0
	failedTests := 0

	for _, result := range testResults {
		if result.Passed {
			passedTests++
		} else {
			failedTests++
		}
	}

	// Check if results were cached
	fileHash, wasCached, hashErr := svc.TestCacheInfo(ruleID, ruleInfo)
	if hashErr != nil {
		logger.Error("Failed to inspect test cache", hashErr)
	}

	// Determine if there's a warning (no tests)
	warning := ""
	if totalTests == 0 {
		warning = "No test cases found for this rule"
	}

	response := map[string]interface{}{
		"rule_id":     ruleID,
		"file_hash":   fileHash,
		"total_tests": totalTests,
		"passed":      passedTests,
		"failed":      failedTests,
		"tests":       testResults,
		"cached":      wasCached,
		"warning":     warning,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode test response", err)
		HTTPError(w, "Failed to encode response", http.StatusInternalServerError, err)
	}
}

func testRuleOnScenarioHandler(w http.ResponseWriter, r *http.Request) {
	logger := NewLogger()

	vars := mux.Vars(r)
	ruleID := strings.TrimSpace(vars["ruleId"])
	if ruleID == "" {
		HTTPError(w, "Rule ID is required", http.StatusBadRequest, nil)
		return
	}

	var req struct {
		Scenario  string   `json:"scenario"`   // Single scenario (deprecated, for backward compatibility)
		Scenarios []string `json:"scenarios"`  // Multiple scenarios (new)
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid request body", http.StatusBadRequest, err)
		return
	}

	// Support both single and batch modes
	scenarios := req.Scenarios
	if len(scenarios) == 0 && req.Scenario != "" {
		// Backward compatibility: single scenario
		scenarios = []string{strings.TrimSpace(req.Scenario)}
	}

	if len(scenarios) == 0 {
		HTTPError(w, "At least one scenario is required", http.StatusBadRequest, nil)
		return
	}

	svc, err := ruleService()
	if err != nil {
		HTTPError(w, "Failed to initialise rule engine", http.StatusInternalServerError, err)
		return
	}

	ruleInfo, exists, err := svc.Get(ruleID)
	if err != nil {
		HTTPError(w, "Failed to load rule", http.StatusInternalServerError, err)
		return
	}
	if !exists {
		HTTPError(w, "Rule not found", http.StatusNotFound, nil)
		return
	}

	if !ruleInfo.Implementation.Valid {
		HTTPError(w, fmt.Sprintf("Rule implementation unavailable: %s", firstNonEmpty(ruleInfo.Implementation.Error, "failed to load")), http.StatusUnprocessableEntity, nil)
		return
	}

	normalizedTargets := ruleInfo.Targets
	if len(normalizedTargets) == 0 {
		normalizedTargets = defaultTargetsForRule(ruleInfo)
	} else {
		normalizedTargets = normalizeTargets(normalizedTargets)
	}

	if len(normalizedTargets) == 0 {
		HTTPError(w, "Rule has no targets configured; add a Targets: metadata entry to run it against scenarios.", http.StatusUnprocessableEntity, nil)
		return
	}

	sortedTargets := append([]string(nil), normalizedTargets...)
	sort.Strings(sortedTargets)
	containsStructure := false
	for _, target := range sortedTargets {
		if target == targetStructure {
			containsStructure = true
			break
		}
	}

	// Process scenarios (single or batch)
	results := []map[string]interface{}{}
	overallStart := time.Now()

	for _, scenarioName := range scenarios {
		scenarioName = strings.TrimSpace(scenarioName)
		if scenarioName == "" {
			continue
		}

		start := time.Now()
		violations, filesScanned, _, evalErr := evaluateRuleOnScenario(ruleInfo, scenarioName)

		result := map[string]interface{}{
			"rule_id":       ruleID,
			"scenario":      scenarioName,
			"files_scanned": filesScanned,
			"targets":       sortedTargets,
			"duration_ms":   time.Since(start).Milliseconds(),
		}

		if evalErr != nil {
			if os.IsNotExist(evalErr) {
				result["error"] = "Scenario not found"
				result["violations"] = []StandardsViolation{}
			} else {
				logger.Error(fmt.Sprintf("Failed to execute rule %s on scenario %s", ruleID, scenarioName), evalErr)
				result["error"] = fmt.Sprintf("Failed to evaluate rule: %v", evalErr)
				result["violations"] = []StandardsViolation{}
			}
		} else {
			warning := ""
			if filesScanned == 0 && len(violations) == 0 && !containsStructure {
				warning = fmt.Sprintf("No matching files were scanned for this rule in the selected scenario. Targets evaluated: %s.", strings.Join(sortedTargets, ", "))
			}

			if violations == nil {
				violations = []StandardsViolation{}
			}

			result["violations"] = violations
			if warning != "" {
				result["warning"] = warning
			}
		}

		results = append(results, result)
	}

	// Return appropriate response format
	var response interface{}
	if len(results) == 1 {
		// Single scenario: backward compatible response
		response = results[0]
	} else {
		// Multiple scenarios: batch response
		response = map[string]interface{}{
			"rule_id":           ruleID,
			"total_scenarios":   len(results),
			"total_duration_ms": time.Since(overallStart).Milliseconds(),
			"results":           results,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode scenario test response", err)
		HTTPError(w, "Failed to encode response", http.StatusInternalServerError, err)
		return
	}
}

// validateRuleHandler validates custom input against a rule
func validateRuleHandler(w http.ResponseWriter, r *http.Request) {
	logger := NewLogger()

	// Extract rule ID from URL
	vars := mux.Vars(r)
	ruleID := vars["ruleId"]

	if ruleID == "" {
		HTTPError(w, "Rule ID is required", http.StatusBadRequest, nil)
		return
	}

	// Parse request body
	var validateReq struct {
		Code     string `json:"code"`
		Language string `json:"language"`
	}

	if err := json.NewDecoder(r.Body).Decode(&validateReq); err != nil {
		HTTPError(w, "Invalid request body", http.StatusBadRequest, err)
		return
	}

	if validateReq.Code == "" {
		HTTPError(w, "Code input is required", http.StatusBadRequest, nil)
		return
	}

	if validateReq.Language == "" {
		validateReq.Language = "text" // Default language
	}

	logger.Info("Validating custom input for rule: " + ruleID)

	svc, err := ruleService()
	if err != nil {
		HTTPError(w, "Failed to initialise rule engine", http.StatusInternalServerError, err)
		return
	}

	ruleInfos, err := svc.Load()
	if err != nil {
		HTTPError(w, "Failed to load rules", http.StatusInternalServerError, err)
		return
	}

	if _, exists := ruleInfos[ruleID]; !exists {
		HTTPError(w, "Rule not found", http.StatusNotFound, nil)
		return
	}

	// Validate the custom input
	result, err := svc.Validate(ruleID, validateReq.Code, validateReq.Language)
	if err != nil {
		HTTPError(w, "Failed to validate input", http.StatusInternalServerError, err)
		return
	}

	response := map[string]interface{}{
		"rule_id":     ruleID,
		"test_result": result,
		"violations":  result.ActualViolations,
		"passed":      len(result.ActualViolations) == 0,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode validation response", err)
		HTTPError(w, "Failed to encode response", http.StatusInternalServerError, err)
	}
}

// clearTestCacheHandler clears test cache for a specific rule or all rules
func clearTestCacheHandler(w http.ResponseWriter, r *http.Request) {
	logger := NewLogger()

	// Extract rule ID from URL (optional)
	vars := mux.Vars(r)
	ruleID := vars["ruleId"]

	svc, err := ruleService()
	if err != nil {
		HTTPError(w, "Failed to initialise rule engine", http.StatusInternalServerError, err)
		return
	}

	if ruleID != "" {
		logger.Info("Clearing test cache for rule: " + ruleID)
		svc.ClearTestCache(ruleID)

		response := map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Test cache cleared for rule %s", ruleID),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	} else {
		logger.Info("Clearing all test cache")
		svc.ClearTestCache("")

		response := map[string]interface{}{
			"success": true,
			"message": "All test caches cleared",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// getTestCoverageHandler returns test coverage metrics for all rules
func getTestCoverageHandler(w http.ResponseWriter, r *http.Request) {
	logger := NewLogger()
	logger.Info("Fetching test coverage metrics")

	svc, err := ruleService()
	if err != nil {
		logger.Error("Failed to initialise rule engine", err)
		HTTPError(w, "Failed to initialise rule engine", http.StatusInternalServerError, err)
		return
	}

	ruleInfos, err := svc.Load()
	if err != nil {
		logger.Error("Failed to load rules", err)
		HTTPError(w, "Failed to load rules", http.StatusInternalServerError, err)
		return
	}

	totalRules := len(ruleInfos)
	rulesWithTests := 0
	totalTestCases := 0
	passingTests := 0
	failingTests := 0
	rulesCoverage := make(map[string]interface{})
	categoryCoverage := make(map[string]map[string]int)

	// Initialize category maps
	categories := re.DefaultCategories()
	for catID := range categories {
		categoryCoverage[catID] = map[string]int{
			"total":      0,
			"with_tests": 0,
			"test_cases": 0,
			"passing":    0,
			"failing":    0,
		}
	}

	// Process each rule
	for ruleID, ruleInfo := range ruleInfos {
		rule := convertInfoToRule(ruleInfo)
		category := rule.Category
		catStats, exists := categoryCoverage[category]
		if !exists {
			catStats = map[string]int{
				"total":      0,
				"with_tests": 0,
				"test_cases": 0,
				"passing":    0,
				"failing":    0,
			}
			categoryCoverage[category] = catStats
		}
		catStats["total"]++

		testCases, err := svc.ExtractTestCases(ruleInfo)
		if err != nil {
			testCases = nil
		}

		hasTests := len(testCases) > 0
		if hasTests {
			rulesWithTests++
			catStats["with_tests"]++
			catStats["test_cases"] += len(testCases)
		}

		totalTestCases += len(testCases)

		results, err := svc.RunTestsForInfo(ruleID, ruleInfo)
		ruleError := ""
		if err != nil {
			ruleError = err.Error()
		} else {
			for _, result := range results {
				if result.Passed {
					passingTests++
					catStats["passing"]++
				} else {
					failingTests++
					catStats["failing"]++
				}
			}
		}

		rulesCoverage[ruleID] = map[string]interface{}{
			"has_tests":  hasTests,
			"test_count": len(testCases),
			"category":   category,
			"severity":   rule.Severity,
			"name":       rule.Name,
			"error":      ruleError,
		}
	}

	// Calculate percentages
	coveragePercent := float64(0)
	if totalRules > 0 {
		coveragePercent = (float64(rulesWithTests) / float64(totalRules)) * 100
	}

	testPassRate := float64(0)
	totalTestsRun := passingTests + failingTests
	if totalTestsRun > 0 {
		testPassRate = (float64(passingTests) / float64(totalTestsRun)) * 100
	}

	averageTestsPerRule := float64(0)
	if totalRules > 0 {
		averageTestsPerRule = float64(totalTestCases) / float64(totalRules)
	}

	// Build response
	response := map[string]interface{}{
		"coverage_metrics": map[string]interface{}{
			"total_rules":            totalRules,
			"rules_with_tests":       rulesWithTests,
			"rules_without_tests":    totalRules - rulesWithTests,
			"coverage_percentage":    coveragePercent,
			"total_test_cases":       totalTestCases,
			"average_tests_per_rule": averageTestsPerRule,
		},
		"test_results": map[string]interface{}{
			"total_run": totalTestsRun,
			"passing":   passingTests,
			"failing":   failingTests,
			"pass_rate": testPassRate,
		},
		"category_breakdown": categoryCoverage,
		"rules_coverage":     rulesCoverage,
		"timestamp":          time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode test coverage response", err)
		HTTPError(w, "Failed to encode response", http.StatusInternalServerError, err)
	}
}

func convertInfoToRule(info re.Info) Rule {
	copyTargets := append([]string(nil), info.Targets...)
	return Rule{
		Rule:           info.Rule,
		Targets:        copyTargets,
		Implementation: info.Implementation,
	}
}
