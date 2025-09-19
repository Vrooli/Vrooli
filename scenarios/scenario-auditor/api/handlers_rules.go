package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// Global state for rule enabled/disabled status
var ruleStates = struct {
	mu     sync.RWMutex
	states map[string]bool
}{
	states: make(map[string]bool),
}

type Rule struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Severity    string          `json:"severity"`
	Enabled     bool            `json:"enabled"`
	Standard    string          `json:"standard"`
	TestStatus  *RuleTestStatus `json:"test_status,omitempty"`
}

type RuleCategory struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

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

	// Load rules from files
	ruleInfos, err := LoadRulesFromFiles()
	if err != nil {
		logger.Error("Failed to load rules from files", err)
		// Fall back to hardcoded rules if loading fails
		ruleInfos = getDefaultRules()
	}

	// Convert RuleInfo to Rule and apply enabled states
	rules := make(map[string]Rule)
	ruleStates.mu.RLock()
	for id, info := range ruleInfos {
		rule := ConvertRuleInfoToRule(info)
		// Check if we have a stored enabled state for this rule
		if enabled, exists := ruleStates.states[id]; exists {
			rule.Enabled = enabled
		}
		rules[id] = rule
	}
	ruleStates.mu.RUnlock()

	testStatuses := computeRuleTestStatuses(ruleInfos)
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
	categories := GetRuleCategories()

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

func computeRuleTestStatuses(ruleInfos map[string]RuleInfo) map[string]RuleTestStatus {
	statuses := make(map[string]RuleTestStatus)
	if len(ruleInfos) == 0 {
		return statuses
	}

	testRunner := NewTestRunner()

	for ruleID, ruleInfo := range ruleInfos {
		status := RuleTestStatus{}
		results, err := testRunner.RunAllTests(ruleID, ruleInfo)
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

		if _, implemented := RuleRegistry[ruleID]; !implemented {
			if status.Warning != "" {
				status.Warning += "; "
			}
			status.Warning += "Rule implementation not registered"
			status.HasIssues = true
		}

		statuses[ruleID] = status
	}

	return statuses
}

// createRuleHandler creates a new rule (not yet implemented)
func createRuleHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Rule creation not yet implemented", http.StatusNotImplemented, nil)
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

	// For now, we'll store the state in memory
	// In a production system, this would be persisted to a database
	ruleStates.mu.Lock()
	ruleStates.states[ruleID] = toggleReq.Enabled
	ruleStates.mu.Unlock()

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

	// Load rules from files
	ruleInfos, err := LoadRulesFromFiles()
	if err != nil {
		logger.Error("Failed to load rules from files", err)
		ruleInfos = getDefaultRules()
	}

	// Find the specific rule
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
	rule := ConvertRuleInfoToRule(ruleInfo)

	if status, ok := computeRuleTestStatuses(map[string]RuleInfo{ruleID: ruleInfo})[ruleID]; ok {
		s := status
		rule.TestStatus = &s
	}

	executionInfo := buildRuleExecutionInfo(ruleInfo)

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

	categories := GetRuleCategories()
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

	// Load rules to get the specific rule
	ruleInfos, err := LoadRulesFromFiles()
	if err != nil {
		HTTPError(w, "Failed to load rules", http.StatusInternalServerError, err)
		return
	}

	ruleInfo, exists := ruleInfos[ruleID]
	if !exists {
		HTTPError(w, "Rule not found", http.StatusNotFound, nil)
		return
	}

	// Get or create test runner
	testRunner := NewTestRunner()

	// Run all tests for the rule
	testResults, err := testRunner.RunAllTests(ruleID, ruleInfo)
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
	fileHash, _ := testRunner.GetFileHash(ruleInfo.FilePath)
	cached, _ := testRunner.GetCachedResults(ruleID, fileHash)
	wasCached := cached != nil

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

	// Load rules to get the specific rule
	ruleInfos, err := LoadRulesFromFiles()
	if err != nil {
		HTTPError(w, "Failed to load rules", http.StatusInternalServerError, err)
		return
	}

	ruleInfo, exists := ruleInfos[ruleID]
	if !exists {
		HTTPError(w, "Rule not found", http.StatusNotFound, nil)
		return
	}

	// Get or create test runner
	testRunner := NewTestRunner()

	// Validate the custom input
	result, err := testRunner.ValidateCustomInput(ruleID, ruleInfo, validateReq.Code, validateReq.Language)
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

	// Get or create test runner
	testRunner := NewTestRunner()

	if ruleID != "" {
		logger.Info("Clearing test cache for rule: " + ruleID)
		testRunner.ClearCache(ruleID)

		response := map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Test cache cleared for rule %s", ruleID),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	} else {
		logger.Info("Clearing all test cache")
		testRunner.ClearAllCache()

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

	// Load all rules
	ruleInfos, err := LoadRulesFromFiles()
	if err != nil {
		logger.Error("Failed to load rules", err)
		HTTPError(w, "Failed to load rules", http.StatusInternalServerError, err)
		return
	}

	// Get test runner
	testRunner := NewTestRunner()

	// Calculate coverage metrics
	totalRules := len(ruleInfos)
	rulesWithTests := 0
	totalTestCases := 0
	passingTests := 0
	failingTests := 0
	rulesCoverage := make(map[string]interface{})
	categoryCoverage := make(map[string]map[string]int)

	// Initialize category maps
	categories := GetRuleCategories()
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
		rule := ConvertRuleInfoToRule(ruleInfo)
		category := rule.Category

		// Increment total for category
		if catStats, exists := categoryCoverage[category]; exists {
			catStats["total"]++
		}

		// Read rule file to check for test cases
		content, err := os.ReadFile(ruleInfo.FilePath)
		if err != nil {
			continue
		}

		// Extract test cases
		testCases, err := testRunner.ExtractTestCases(string(content))
		if err != nil {
			continue
		}

		hasTests := len(testCases) > 0
		if hasTests {
			rulesWithTests++
			if catStats, exists := categoryCoverage[category]; exists {
				catStats["with_tests"]++
				catStats["test_cases"] += len(testCases)
			}
		}

		totalTestCases += len(testCases)

		// Run tests if cache is available (don't run new tests, just check cache)
		fileHash, _ := testRunner.GetFileHash(ruleInfo.FilePath)
		if cached, valid := testRunner.GetCachedResults(ruleID, fileHash); valid {
			for _, result := range cached.TestResults {
				if result.Passed {
					passingTests++
					if catStats, exists := categoryCoverage[category]; exists {
						catStats["passing"]++
					}
				} else {
					failingTests++
					if catStats, exists := categoryCoverage[category]; exists {
						catStats["failing"]++
					}
				}
			}
		}

		// Store rule-specific coverage
		rulesCoverage[ruleID] = map[string]interface{}{
			"has_tests":  hasTests,
			"test_count": len(testCases),
			"category":   category,
			"severity":   rule.Severity,
			"name":       rule.Name,
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

	// Build response
	response := map[string]interface{}{
		"coverage_metrics": map[string]interface{}{
			"total_rules":            totalRules,
			"rules_with_tests":       rulesWithTests,
			"rules_without_tests":    totalRules - rulesWithTests,
			"coverage_percentage":    coveragePercent,
			"total_test_cases":       totalTestCases,
			"average_tests_per_rule": float64(totalTestCases) / float64(totalRules),
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
