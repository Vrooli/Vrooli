package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type reportIssueRequest struct {
	ReportType          string   `json:"reportType"`
	RuleID              string   `json:"ruleId"`
	CustomInstructions  string   `json:"customInstructions"`
	SelectedScenarios   []string `json:"selectedScenarios"`
}

type issueTrackerResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Error   string                 `json:"error"`
	Data    map[string]interface{} `json:"data"`
}

// reportIssueHandler creates an issue in app-issue-tracker for rule fixes/tests
func reportIssueHandler(w http.ResponseWriter, r *http.Request) {
	logger := NewLogger()

	var req reportIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid request body", http.StatusBadRequest, err)
		return
	}

	// Validate request
	if req.RuleID == "" {
		HTTPError(w, "ruleId is required", http.StatusBadRequest, nil)
		return
	}

	if req.ReportType == "" {
		HTTPError(w, "reportType is required", http.StatusBadRequest, nil)
		return
	}

	validReportTypes := map[string]bool{
		"add_tests":       true,
		"fix_tests":       true,
		"fix_violations":  true,
	}

	if !validReportTypes[req.ReportType] {
		HTTPError(w, fmt.Sprintf("invalid reportType: %s", req.ReportType), http.StatusBadRequest, nil)
		return
	}

	if req.ReportType == "fix_violations" && len(req.SelectedScenarios) == 0 {
		HTTPError(w, "selectedScenarios is required for fix_violations", http.StatusBadRequest, nil)
		return
	}

	// Load rule to get details
	rules, err := LoadRulesFromFiles()
	if err != nil {
		HTTPError(w, "Failed to load rules", http.StatusInternalServerError, err)
		return
	}

	ruleInfo, exists := rules[req.RuleID]
	if !exists {
		HTTPError(w, "Rule not found", http.StatusNotFound, nil)
		return
	}

	// Convert RuleInfo to Rule
	rule := ConvertRuleInfoToRule(ruleInfo)

	// Resolve app-issue-tracker API port
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	trackerPort, err := resolveIssueTrackerPort(ctx)
	if err != nil {
		HTTPError(w, "Failed to locate app-issue-tracker", http.StatusServiceUnavailable, err)
		return
	}

	// Build payload based on report type
	payload, err := buildIssuePayload(req, rule)
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

	// Build issue URL if we have the issue ID
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
		logger.Error("Failed to encode report issue response", err)
	}
}

func buildIssuePayload(req reportIssueRequest, rule Rule) (map[string]interface{}, error) {
	var title, description, issueType, priority string
	var tags []string
	var artifacts []map[string]interface{}

	metadata := map[string]string{
		"reported_by":  "scenario-auditor",
		"report_type":  req.ReportType,
		"rule_id":      rule.ID,
	}

	environment := map[string]string{
		"rule_id":   rule.ID,
		"rule_name": safeFallback(rule.Name, rule.ID),
	}

	switch req.ReportType {
	case "add_tests":
		title = fmt.Sprintf("[scenario-auditor] Add test cases for rule %s", safeFallback(rule.Name, rule.ID))
		description = buildAddTestsDescription(rule, req.CustomInstructions)
		issueType = "task"
		priority = "medium"
		tags = []string{"scenario-auditor", "test-generation", "rule-tests"}

	case "fix_tests":
		title = fmt.Sprintf("[scenario-auditor] Fix failing test cases for rule %s", safeFallback(rule.Name, rule.ID))
		description = buildFixTestsDescription(rule, req.CustomInstructions)
		issueType = "bug"
		priority = "high"
		tags = []string{"scenario-auditor", "test-failures", "rule-tests"}

	case "fix_violations":
		title = fmt.Sprintf("[scenario-auditor] Fix %d violations of %s across %d scenarios",
			len(req.SelectedScenarios), safeFallback(rule.Name, rule.ID), len(req.SelectedScenarios))
		description = buildFixViolationsDescription(rule, req.CustomInstructions, req.SelectedScenarios)
		issueType = "bug"
		priority = mapSeverityToPriority(rule.Severity)
		tags = []string{"scenario-auditor", "rule-violations", "auto-fix"}

		// Add scenarios to metadata
		metadata["scenario_count"] = strconv.Itoa(len(req.SelectedScenarios))
		environment["scenarios"] = strings.Join(req.SelectedScenarios, ",")

		// Create artifact with violation details
		artifactContent := buildViolationArtifact(rule, req.SelectedScenarios)
		artifacts = append(artifacts, map[string]interface{}{
			"name":         fmt.Sprintf("%s-violations.md", sanitizeForFilename(rule.ID)),
			"category":     "rule_violations",
			"content":      artifactContent,
			"encoding":     "plain",
			"content_type": "text/markdown",
		})
	}

	payload := map[string]interface{}{
		"title":          title,
		"description":    description,
		"type":           issueType,
		"priority":       priority,
		"app_id":         "scenario-auditor",
		"status":         "open",
		"tags":           tags,
		"metadata_extra": metadata,
		"environment":    environment,
		"reporter_name":  "Scenario Auditor",
		"reporter_email": "auditor@vrooli.local",
	}

	if len(artifacts) > 0 {
		payload["artifacts"] = artifacts
	}

	return payload, nil
}

func buildAddTestsDescription(rule Rule, customInstructions string) string {
	var b strings.Builder
	b.WriteString("Scenario Auditor requests automated test generation for this rule.\n\n")
	b.WriteString(fmt.Sprintf("**Rule**: %s\n", safeFallback(rule.Name, rule.ID)))
	b.WriteString(fmt.Sprintf("**Category**: %s\n", rule.Category))
	b.WriteString(fmt.Sprintf("**Severity**: %s\n", rule.Severity))
	if rule.Description != "" {
		b.WriteString(fmt.Sprintf("**Description**: %s\n", rule.Description))
	}

	b.WriteString("\n### Requirements\n")
	b.WriteString("- Analyze the rule implementation and create comprehensive test coverage\n")
	b.WriteString("- Generate both positive and negative test cases\n")
	b.WriteString("- Test should be deterministic and fast\n")
	b.WriteString("- Follow existing test patterns in the codebase\n")

	if customInstructions != "" {
		b.WriteString("\n### Custom Instructions\n")
		b.WriteString(customInstructions)
		b.WriteString("\n")
	}

	return b.String()
}

func buildFixTestsDescription(rule Rule, customInstructions string) string {
	var b strings.Builder
	b.WriteString("Scenario Auditor detected failing test cases for this rule.\n\n")
	b.WriteString(fmt.Sprintf("**Rule**: %s\n", safeFallback(rule.Name, rule.ID)))
	b.WriteString(fmt.Sprintf("**Category**: %s\n", rule.Category))

	b.WriteString("\n### Requirements\n")
	b.WriteString("- Investigate why tests are failing\n")
	b.WriteString("- Fix the tests or update the rule implementation as needed\n")
	b.WriteString("- Ensure all test cases pass\n")
	b.WriteString("- Maintain backward compatibility\n")

	if customInstructions != "" {
		b.WriteString("\n### Custom Instructions\n")
		b.WriteString(customInstructions)
		b.WriteString("\n")
	}

	return b.String()
}

func buildFixViolationsDescription(rule Rule, customInstructions string, scenarios []string) string {
	var b strings.Builder
	b.WriteString("Scenario Auditor detected rule violations in multiple scenarios.\n\n")
	b.WriteString(fmt.Sprintf("**Rule**: %s\n", safeFallback(rule.Name, rule.ID)))
	b.WriteString(fmt.Sprintf("**Severity**: %s\n", rule.Severity))
	b.WriteString(fmt.Sprintf("**Scenarios Affected**: %d\n\n", len(scenarios)))

	b.WriteString("### Affected Scenarios\n")
	for _, scenario := range scenarios {
		b.WriteString(fmt.Sprintf("- %s\n", scenario))
	}

	b.WriteString("\n### Requirements\n")
	b.WriteString("- Review the violation details in the attached artifact\n")
	b.WriteString("- Fix all violations across the affected scenarios\n")
	b.WriteString("- Ensure fixes align with the rule's intent\n")
	b.WriteString("- Run tests after applying fixes\n")

	if customInstructions != "" {
		b.WriteString("\n### Custom Instructions\n")
		b.WriteString(customInstructions)
		b.WriteString("\n")
	}

	return b.String()
}

func buildViolationArtifact(rule Rule, scenarios []string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Violations for Rule: %s\n\n", safeFallback(rule.Name, rule.ID)))
	b.WriteString(fmt.Sprintf("**Rule ID**: %s\n", rule.ID))
	b.WriteString(fmt.Sprintf("**Severity**: %s\n", rule.Severity))
	b.WriteString(fmt.Sprintf("**Category**: %s\n\n", rule.Category))

	if rule.Description != "" {
		b.WriteString(fmt.Sprintf("**Description**: %s\n\n", rule.Description))
	}

	b.WriteString("## Affected Scenarios\n\n")
	for i, scenario := range scenarios {
		b.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, scenario))
	}

	b.WriteString("\n## Next Steps\n\n")
	b.WriteString("1. Run the rule against each scenario individually to see specific violations\n")
	b.WriteString("2. Review the violation details (file paths, line numbers, recommendations)\n")
	b.WriteString("3. Apply fixes systematically\n")
	b.WriteString("4. Re-run the rule to verify fixes\n")

	return b.String()
}

func mapSeverityToPriority(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "high"
	case "high":
		return "high"
	case "medium":
		return "medium"
	default:
		return "low"
	}
}

func sanitizeForFilename(value string) string {
	sanitized := make([]rune, 0, len(value))
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			sanitized = append(sanitized, r)
		case r >= 'A' && r <= 'Z':
			sanitized = append(sanitized, r+('a'-'A'))
		case r >= '0' && r <= '9':
			sanitized = append(sanitized, r)
		case r == '-' || r == '_':
			sanitized = append(sanitized, r)
		default:
			sanitized = append(sanitized, '-')
		}
	}

	result := strings.Trim(strings.ReplaceAll(string(sanitized), "--", "-"), "-")
	if result == "" {
		return "file"
	}
	return result
}

type issueTrackerResult struct {
	IssueID string
	Message string
}

func submitIssueToTracker(ctx context.Context, port int, payload map[string]interface{}) (*issueTrackerResult, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	endpoint := fmt.Sprintf("http://localhost:%d/api/v1/issues", port)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call app-issue-tracker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("app-issue-tracker returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	var trackerResp issueTrackerResponse
	if err := json.NewDecoder(resp.Body).Decode(&trackerResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !trackerResp.Success {
		message := strings.TrimSpace(trackerResp.Error)
		if message == "" {
			message = strings.TrimSpace(trackerResp.Message)
		}
		if message == "" {
			message = "app-issue-tracker rejected the issue"
		}
		return nil, errors.New(message)
	}

	issueID := ""
	if trackerResp.Data != nil {
		if value, ok := trackerResp.Data["issue_id"].(string); ok {
			issueID = value
		} else if value, ok := trackerResp.Data["issueId"].(string); ok {
			issueID = value
		} else if rawIssue, ok := trackerResp.Data["issue"]; ok {
			if issueMap, ok := rawIssue.(map[string]interface{}); ok {
				if id, ok := issueMap["id"].(string); ok {
					issueID = id
				}
			}
		}
	}

	message := strings.TrimSpace(trackerResp.Message)
	if message == "" {
		message = "Issue created successfully"
	}

	return &issueTrackerResult{
		IssueID: issueID,
		Message: message,
	}, nil
}

func resolveIssueTrackerPort(ctx context.Context) (int, error) {
	return resolveScenarioPortViaCLI(ctx, "app-issue-tracker", "API_PORT")
}

func resolveIssueTrackerUIPort(ctx context.Context) (int, error) {
	return resolveScenarioPortViaCLI(ctx, "app-issue-tracker", "UI_PORT")
}

func resolveScenarioPortViaCLI(ctx context.Context, scenarioName, portLabel string) (int, error) {
	if strings.TrimSpace(scenarioName) == "" || strings.TrimSpace(portLabel) == "" {
		return 0, errors.New("scenario and port labels are required")
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", "scenario", "port", scenarioName, portLabel)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("vrooli scenario port %s %s failed: %s", scenarioName, portLabel, strings.TrimSpace(string(output)))
	}

	return parsePortValue(strings.TrimSpace(string(output)))
}

func parsePortValue(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, errors.New("empty port value")
	}

	port, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid port value %q", value)
	}

	if port <= 0 || port > 65535 {
		return 0, fmt.Errorf("port value out of range: %d", port)
	}

	return port, nil
}
