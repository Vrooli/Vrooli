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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	rulespkg "scenario-auditor/rules"
)

type reportIssueRequest struct {
	ReportType         string   `json:"reportType"`
	RuleID             string   `json:"ruleId"`
	CustomInstructions string   `json:"customInstructions"`
	SelectedScenarios  []string `json:"selectedScenarios"`
}

type createRuleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	Motivation  string `json:"motivation"`
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
		"add_tests":      true,
		"fix_tests":      true,
		"fix_violations": true,
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
	payload, err := buildIssuePayload(req, rule, ruleInfo)
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

func buildIssuePayload(req reportIssueRequest, rule Rule, ruleInfo RuleInfo) (map[string]interface{}, error) {
	var title, description, issueType, priority string
	var tags []string
	var artifacts []map[string]interface{}

	metadata := map[string]string{
		"reported_by": "scenario-auditor",
		"report_type": req.ReportType,
		"rule_id":     rule.ID,
	}

	environment := map[string]string{
		"rule_id":   rule.ID,
		"rule_name": safeFallback(rule.Name, rule.ID),
	}

	switch req.ReportType {
	case "add_tests":
		title = fmt.Sprintf("[scenario-auditor] Add test cases for rule %s", safeFallback(rule.Name, rule.ID))
		description = buildAddTestsDescription(rule, ruleInfo, req.CustomInstructions)
		issueType = "task"
		priority = "medium"
		tags = []string{"scenario-auditor", "test-generation", "rule-tests"}

	case "fix_tests":
		title = fmt.Sprintf("[scenario-auditor] Fix failing test cases for rule %s", safeFallback(rule.Name, rule.ID))
		description = buildFixTestsDescription(rule, ruleInfo, req.CustomInstructions)
		issueType = "bug"
		priority = "high"
		tags = []string{"scenario-auditor", "test-failures", "rule-tests"}

	case "fix_violations":
		title = fmt.Sprintf("[scenario-auditor] Fix %d violations of %s across %d scenarios",
			len(req.SelectedScenarios), safeFallback(rule.Name, rule.ID), len(req.SelectedScenarios))
		description = buildFixViolationsDescription(rule, ruleInfo, req.CustomInstructions, req.SelectedScenarios)
		issueType = "bug"
		priority = mapSeverityToPriority(rule.Severity)
		tags = []string{"scenario-auditor", "rule-violations", "auto-fix"}

		// Add scenarios to metadata
		metadata["scenario_count"] = strconv.Itoa(len(req.SelectedScenarios))
		environment["scenarios"] = strings.Join(req.SelectedScenarios, ",")

		// Create artifact with violation details
		artifactContent := buildViolationArtifact(rule, req.SelectedScenarios, ruleInfo)
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

func buildAddTestsDescription(rule Rule, ruleInfo RuleInfo, customInstructions string) string {
	var b strings.Builder
	b.WriteString("Scenario Auditor requests automated test generation for this rule.\n\n")
	b.WriteString(fmt.Sprintf("**Rule**: %s\n", safeFallback(rule.Name, rule.ID)))
	b.WriteString(fmt.Sprintf("**Rule ID**: %s\n", rule.ID))
	b.WriteString(fmt.Sprintf("**Category**: %s\n", rule.Category))
	b.WriteString(fmt.Sprintf("**Severity**: %s\n", rule.Severity))
	if rule.Description != "" {
		b.WriteString(fmt.Sprintf("**Description**: %s\n", rule.Description))
	}

	// Rule implementation context
	b.WriteString("\n## Rule Implementation Context\n\n")
	b.WriteString(fmt.Sprintf("**Rule File**: `%s`\n\n", relativePathFrom(ruleInfo.FilePath, getScenarioRoot())))

	// Read and include rule implementation
	if ruleSource, err := os.ReadFile(ruleInfo.FilePath); err == nil {
		b.WriteString("### Current Implementation\n")
		b.WriteString("```go\n")
		b.WriteString(string(ruleSource))
		b.WriteString("\n```\n\n")
	}

	// Extract and describe existing test cases
	existingTests, _ := extractRuleTestCases(ruleInfo)
	if len(existingTests) > 0 {
		b.WriteString("### Existing Test Cases\n")
		b.WriteString(fmt.Sprintf("Rule currently has %d test case(s):\n\n", len(existingTests)))
		for _, tc := range existingTests {
			b.WriteString(fmt.Sprintf("- **%s**: %s (should-fail=%t, language=%s)\n",
				tc.ID, tc.Description, tc.ShouldFail, safeFallback(tc.Language, "text")))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("### Existing Test Cases\n")
		b.WriteString("Rule currently has **no embedded test cases**.\n\n")
	}

	// Test case format requirements
	b.WriteString("## Test Case Requirements\n\n")
	b.WriteString("**Format**: Test cases MUST be embedded as XML comments in the rule file using this structure:\n")
	b.WriteString("```xml\n")
	b.WriteString("<test-case id=\"unique-test-id\" should-fail=\"true|false\" path=\"relative/file/path\">\n")
	b.WriteString("  <description>Clear description of what this test validates</description>\n")
	b.WriteString("  <input language=\"go|json|text\">\n")
	b.WriteString("    ... realistic test input code/content ...\n")
	b.WriteString("  </input>\n")
	b.WriteString("  <expected-violations>0|1|2...</expected-violations>\n")
	b.WriteString("  <expected-message>partial message to match (optional)</expected-message>\n")
	b.WriteString("</test-case>\n")
	b.WriteString("```\n\n")

	b.WriteString("**Attributes**:\n")
	b.WriteString("- `id`: Unique kebab-case identifier (e.g., \"missing-health-endpoint\")\n")
	b.WriteString("- `should-fail`: \"true\" if rule should detect violations, \"false\" if valid\n")
	b.WriteString("- `path`: Simulated file path matching rule's target (e.g., \"api/main.go\", \".vrooli/service.json\")\n")
	b.WriteString("- `language`: Code language for syntax context (\"go\", \"json\", \"text\", etc.)\n\n")

	b.WriteString("**Test Coverage Requirements**:\n")
	b.WriteString("1. **Happy Path**: At least 1 test where rule passes (should-fail=\"false\")\n")
	b.WriteString("2. **Violation Detection**: At least 2-3 tests detecting different violation types (should-fail=\"true\")\n")
	b.WriteString("3. **Edge Cases**: Boundary conditions, empty inputs, comments vs actual code\n")
	b.WriteString("4. **False Positives**: Tests ensuring rule doesn't trigger incorrectly\n\n")

	b.WriteString("**Validation**: Tests validate:\n")
	b.WriteString("- Violation count matches `<expected-violations>`\n")
	b.WriteString("- Violation message contains `<expected-message>` substring (if specified)\n")
	b.WriteString("- Rule executes without errors\n\n")

	// Testing integration
	b.WriteString("## Testing Integration\n\n")
	b.WriteString(fmt.Sprintf("**Run Tests**: After adding test cases, execute:\n```bash\nscenario-auditor test %s\n```\n\n", rule.ID))
	b.WriteString("**Verification**: All tests must pass before considering task complete.\n\n")

	// Reference examples
	b.WriteString("## Reference Examples\n\n")
	b.WriteString(fmt.Sprintf("**Gold Standard Example**: See `api/rules/api/health_check.go` for comprehensive test coverage (5+ test cases covering various scenarios).\n\n"))
	if rule.Category != "" {
		b.WriteString(fmt.Sprintf("**Category**: This is a `%s` rule. Review other rules in `api/rules/%s/` for similar patterns.\n\n", rule.Category, rule.Category))
	}

	// File modification instructions
	b.WriteString("## File Modification Instructions\n\n")
	b.WriteString("1. **DO NOT create separate test files** - Tests MUST be embedded in the rule file itself\n")
	b.WriteString("2. Add test-case XML blocks in comments AFTER the rule metadata but BEFORE the implementation\n")
	b.WriteString("3. Maintain existing code style and structure\n")
	b.WriteString("4. Ensure test inputs are realistic and representative of actual scenario code\n\n")

	// Rule engine context
	b.WriteString("## Rule Engine Context\n\n")
	b.WriteString("- Rules implement `Check(content, filepath, scenario) ([]Violation, error)`\n")
	b.WriteString("- Rules receive file content as raw string\n")
	b.WriteString("- Rules must be deterministic and fast\n")
	b.WriteString("- Rules registered in `api/rule_registry.go`\n\n")

	if customInstructions != "" {
		b.WriteString("## Custom Instructions\n\n")
		b.WriteString(customInstructions)
		b.WriteString("\n\n")
	}

	// Success criteria
	b.WriteString("## Success Criteria\n\n")
	b.WriteString(fmt.Sprintf("- [ ] All rule test cases pass (`scenario-auditor test %s`)\n", rule.ID))
	b.WriteString("- [ ] Test coverage includes happy path + violations + edge cases\n")
	b.WriteString("- [ ] Tests are deterministic and fast\n")
	b.WriteString("- [ ] Test inputs are realistic and representative\n")

	return b.String()
}

func buildFixTestsDescription(rule Rule, ruleInfo RuleInfo, customInstructions string) string {
	var b strings.Builder
	b.WriteString("Scenario Auditor detected failing test cases for this rule.\n\n")
	b.WriteString(fmt.Sprintf("**Rule**: %s\n", safeFallback(rule.Name, rule.ID)))
	b.WriteString(fmt.Sprintf("**Rule ID**: %s\n", rule.ID))
	b.WriteString(fmt.Sprintf("**Category**: %s\n", rule.Category))
	b.WriteString(fmt.Sprintf("**Severity**: %s\n", rule.Severity))

	// Rule implementation context
	b.WriteString("\n## Rule Implementation Context\n\n")
	b.WriteString(fmt.Sprintf("**Rule File**: `%s`\n\n", relativePathFrom(ruleInfo.FilePath, getScenarioRoot())))

	// Read and include rule implementation
	if ruleSource, err := os.ReadFile(ruleInfo.FilePath); err == nil {
		b.WriteString("### Rule Implementation\n")
		b.WriteString("```go\n")
		b.WriteString(string(ruleSource))
		b.WriteString("\n```\n\n")
	}

	// Run tests and get failure details
	b.WriteString("## Failing Test Details\n\n")
	testResults, err := runRuleTests(rule.ID, ruleInfo)
	if err != nil {
		b.WriteString(fmt.Sprintf("**Test Execution Error**: %v\n\n", err))
	}

	if len(testResults) > 0 {
		failingCount := 0
		passingCount := 0
		for _, result := range testResults {
			if !result.Passed {
				failingCount++
			} else {
				passingCount++
			}
		}

		b.WriteString(fmt.Sprintf("**Summary**: %d failing, %d passing (total: %d tests)\n\n", failingCount, passingCount, len(testResults)))

		if failingCount > 0 {
			b.WriteString("### Failing Tests:\n\n")
			for _, result := range testResults {
				if !result.Passed {
					b.WriteString(fmt.Sprintf("#### Test: `%s`\n", result.TestCase.ID))
					if result.TestCase.Description != "" {
						b.WriteString(fmt.Sprintf("**Description**: %s\n\n", result.TestCase.Description))
					}
					b.WriteString(fmt.Sprintf("- **Should Fail**: %t\n", result.TestCase.ShouldFail))
					b.WriteString(fmt.Sprintf("- **Expected Violations**: %d\n", result.TestCase.ExpectedViolations))
					b.WriteString(fmt.Sprintf("- **Actual Violations**: %d\n", len(result.ActualViolations)))

					if result.TestCase.ExpectedMessage != "" {
						b.WriteString(fmt.Sprintf("- **Expected Message**: \"%s\"\n", result.TestCase.ExpectedMessage))
					}

					if len(result.ActualViolations) > 0 {
						b.WriteString("\n**Actual Violations Found**:\n")
						for i, v := range result.ActualViolations {
							b.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, safeFallback(v.Severity, "unknown"), v.Message))
						}
					}

					if result.Error != "" {
						b.WriteString(fmt.Sprintf("\n**Error**: %s\n", result.Error))
					}
					b.WriteString("\n")
				}
			}
		}
	} else {
		b.WriteString("No test results available. Tests may not be running correctly.\n\n")
	}

	// Diagnostic approach
	b.WriteString("## Diagnostic Approach\n\n")
	b.WriteString("1. **Analyze Discrepancy**: Determine if:\n")
	b.WriteString("   - Test expectations are wrong (update test-case)\n")
	b.WriteString("   - Rule implementation has bug (fix Check() method)\n")
	b.WriteString("   - Test input doesn't match rule's target context\n\n")

	b.WriteString("2. **Common Failure Patterns**:\n")
	b.WriteString("   - Rule too strict: Adjust detection logic\n")
	b.WriteString("   - Rule too lenient: Strengthen validation\n")
	b.WriteString("   - Test input unrealistic: Update test scenarios\n")
	b.WriteString("   - Message mismatch: Align violation messages with expected-message\n\n")

	b.WriteString("3. **Verification**:\n")
	b.WriteString(fmt.Sprintf("   - Run `scenario-auditor test %s` after fixes\n", rule.ID))
	b.WriteString("   - Ensure ALL test cases pass\n")
	b.WriteString("   - Verify no regressions in previously passing tests\n\n")

	// Test case modification format
	b.WriteString("## Test Case Modification Format\n\n")
	b.WriteString("Test cases are embedded XML in rule file comments:\n")
	b.WriteString("```xml\n")
	b.WriteString("<test-case id=\"{id}\" should-fail=\"{true|false}\" path=\"{path}\">\n")
	b.WriteString("  <description>{description}</description>\n")
	b.WriteString("  <input language=\"{language}\">...</input>\n")
	b.WriteString("  <expected-violations>{count}</expected-violations>\n")
	b.WriteString("  <expected-message>{partial-match}</expected-message>\n")
	b.WriteString("</test-case>\n")
	b.WriteString("```\n\n")

	// Rule implementation signature
	b.WriteString("## Rule Implementation Signature\n\n")
	b.WriteString("```go\n")
	b.WriteString("func (r *{RuleStruct}) Check(content string, filepath string, scenario string) ([]Violation, error)\n")
	b.WriteString("```\n\n")
	b.WriteString("- `content`: Raw file content as string\n")
	b.WriteString("- `filepath`: Relative file path for context\n")
	b.WriteString("- `scenario`: Scenario name (empty during tests)\n")
	b.WriteString("- Returns: Slice of violations or error\n\n")

	if customInstructions != "" {
		b.WriteString("## Custom Instructions\n\n")
		b.WriteString(customInstructions)
		b.WriteString("\n\n")
	}

	// Success criteria
	b.WriteString("## Success Criteria\n\n")
	b.WriteString(fmt.Sprintf("- [ ] All test cases pass (`scenario-auditor test %s`)\n", rule.ID))
	b.WriteString("- [ ] No regressions in previously passing tests\n")
	b.WriteString("- [ ] Test expectations match rule behavior\n")
	b.WriteString("- [ ] Rule implementation is correct and maintainable\n")

	return b.String()
}

func buildFixViolationsDescription(rule Rule, ruleInfo RuleInfo, customInstructions string, scenarios []string) string {
	var b strings.Builder
	b.WriteString("Scenario Auditor detected rule violations in multiple scenarios.\n\n")

	// Rule details
	b.WriteString("## Rule Details\n\n")
	b.WriteString(fmt.Sprintf("**Rule**: %s\n", safeFallback(rule.Name, rule.ID)))
	b.WriteString(fmt.Sprintf("**ID**: %s\n", rule.ID))
	b.WriteString(fmt.Sprintf("**Severity**: %s\n", rule.Severity))
	b.WriteString(fmt.Sprintf("**Category**: %s\n\n", rule.Category))

	if rule.Description != "" {
		b.WriteString("### Rule Purpose\n")
		b.WriteString(fmt.Sprintf("%s\n\n", rule.Description))
	}

	if ruleInfo.Reason != "" {
		b.WriteString("### Why This Matters\n")
		b.WriteString(fmt.Sprintf("%s\n\n", ruleInfo.Reason))
	}

	// Affected scenarios
	b.WriteString(fmt.Sprintf("## Affected Scenarios (%d)\n\n", len(scenarios)))
	for _, scenario := range scenarios {
		b.WriteString(fmt.Sprintf("- `%s`\n", scenario))
	}
	b.WriteString("\n")

	// Note about detailed violations
	b.WriteString("**Note**: Detailed violation information (file paths, line numbers, specific messages) is available ")
	b.WriteString("in the attached artifact. Review the artifact for complete violation details per scenario.\n\n")

	// NEW: Pre-Fix Investigation Checklist
	b.WriteString("## Pre-Fix Investigation\n\n")
	b.WriteString("Before modifying code, understand the current implementation:\n\n")
	b.WriteString("- [ ] **Locate violations**: Check attached artifact for exact file paths and line numbers\n")
	b.WriteString("- [ ] **Understand the rule**: Review \"What This Rule Detects\" section in artifact\n")
	b.WriteString("- [ ] **Study examples**: Compare your code against passing/failing examples in artifact\n")
	b.WriteString("- [ ] **Review context**: Read surrounding code to understand the pattern\n")
	b.WriteString("- [ ] **Check dependencies**: Identify required imports or helpers needed\n\n")

	// NEW: Rule-specific detailed fix patterns
	buildRuleSpecificFixPatterns(&b, rule, ruleInfo)

	// Fix validation requirements
	b.WriteString("## Fix Validation Requirements\n\n")
	b.WriteString("After applying fixes to each scenario:\n\n")
	b.WriteString("1. **Re-run Rule**: Execute rule against fixed scenario:\n")
	b.WriteString("   ```bash\n")
	b.WriteString(fmt.Sprintf("   scenario-auditor scan {scenario-name} --rule %s\n", rule.ID))
	b.WriteString("   ```\n\n")

	b.WriteString("2. **Verify Zero Violations**: Ensure rule no longer triggers\n\n")

	b.WriteString("3. **Run Scenario Tests**: Execute scenario's test suite:\n")
	b.WriteString("   ```bash\n")
	b.WriteString("   cd scenarios/{scenario-name} && make test\n")
	b.WriteString("   ```\n\n")

	b.WriteString("4. **Check Service Health**: Start scenario and verify it works:\n")
	b.WriteString("   ```bash\n")
	b.WriteString("   vrooli scenario run {scenario-name}\n")
	b.WriteString("   vrooli scenario status {scenario-name}\n")
	b.WriteString("   ```\n\n")

	// Common fix patterns by category
	if rule.Category != "" {
		b.WriteString(fmt.Sprintf("## Common Fix Patterns for %s Rules\n\n", rule.Category))

		switch rule.Category {
		case "api":
			b.WriteString("**API Standards Fixes**:\n")
			b.WriteString("- Health endpoints: Add `r.HandleFunc(\"/health\", healthHandler).Methods(\"GET\")`\n")
			b.WriteString("- HTTP status codes: Use appropriate codes (200, 201, 400, 404, 500)\n")
			b.WriteString("- Content-Type headers: Set `w.Header().Set(\"Content-Type\", \"application/json\")`\n")
			b.WriteString("- Error handling: Return proper error responses with consistent format\n\n")

		case "config":
			b.WriteString("**Configuration Fixes**:\n")
			b.WriteString("- service.json: Ensure valid JSON with required lifecycle fields\n")
			b.WriteString("- Ports: Use approved ranges (API: 15000-19999, UI: 35000-39999)\n")
			b.WriteString("- Environment variables: Use standard names (API_PORT, UI_PORT)\n")
			b.WriteString("- Makefile: Include required targets (run, test, logs, stop, clean)\n\n")

		case "ui":
			b.WriteString("**UI Standards Fixes**:\n")
			b.WriteString("- Security headers: Configure helmet.js with frameAncestors\n")
			b.WriteString("- Element IDs: Add unique data-testid attributes\n")
			b.WriteString("- Accessibility: Use semantic HTML and ARIA labels\n")
			b.WriteString("- Tunnel integration: Use iframe bridge for secure preview\n\n")

		case "test":
			b.WriteString("**Testing Fixes**:\n")
			b.WriteString("- Test structure: Organize tests in test/phases/\n")
			b.WriteString("- Exit codes: Ensure test scripts return non-zero on failure\n")
			b.WriteString("- Coverage: Add test coverage reporting and thresholds\n")
			b.WriteString("- Integration: Use centralized testing helpers\n\n")

		case "cli":
			b.WriteString("**CLI Fixes**:\n")
			b.WriteString("- Lightweight main: Keep CLI entry point simple, delegate to libraries\n")
			b.WriteString("- Structured logging: Use consistent log format with levels\n")
			b.WriteString("- Error handling: Exit with appropriate codes (0=success, 1=error)\n")
			b.WriteString("- Help text: Include usage examples and flag descriptions\n\n")

		case "structure":
			b.WriteString("**Structure Fixes**:\n")
			b.WriteString("- Required layout: Ensure api/, cli/, ui/, test/ directories exist\n")
			b.WriteString("- Documentation: Include PRD.md, README.md in scenario root\n")
			b.WriteString("- Lifecycle files: Add .vrooli/service.json, Makefile\n")
			b.WriteString("- Test coverage: Include coverage reports in test/coverage/\n\n")
		}
	}

	// Rule implementation reference
	b.WriteString("## Rule Implementation Reference\n\n")
	b.WriteString("To understand exact detection logic, see rule implementation:\n")
	b.WriteString(fmt.Sprintf("**File**: `%s`\n\n", relativePathFrom(ruleInfo.FilePath, getScenarioRoot())))

	// Read and include relevant portions of rule Check() method
	if ruleSource, err := os.ReadFile(ruleInfo.FilePath); err == nil {
		// Try to extract just the Check method for brevity
		checkMethodPattern := `func \([^)]+\) Check\([^)]+\) \([^)]+\) \{`
		if matched, _ := regexp.MatchString(checkMethodPattern, string(ruleSource)); matched {
			b.WriteString("<details>\n")
			b.WriteString("<summary>Click to view rule detection logic</summary>\n\n")
			b.WriteString("```go\n")
			// Include full source for now - could be optimized to extract just Check method
			b.WriteString(string(ruleSource))
			b.WriteString("\n```\n")
			b.WriteString("</details>\n\n")
		}
	}

	if customInstructions != "" {
		b.WriteString("## Custom Instructions\n\n")
		b.WriteString(customInstructions)
		b.WriteString("\n\n")
	}

	// Success criteria
	b.WriteString("## Success Criteria\n\n")
	b.WriteString("- [ ] All violations fixed across all affected scenarios\n")
	b.WriteString(fmt.Sprintf("- [ ] Rule `%s` passes on all scenarios\n", rule.ID))
	b.WriteString("- [ ] Scenario tests pass after fixes\n")
	b.WriteString("- [ ] Scenarios start and function correctly\n")
	b.WriteString("- [ ] Fixes align with rule's intent and best practices\n")

	return b.String()
}

// buildRuleSpecificFixPatterns adds detailed, rule-specific fix patterns with before/after examples
func buildRuleSpecificFixPatterns(b *strings.Builder, rule Rule, ruleInfo RuleInfo) {
	// Try to generate rule-specific patterns based on rule ID and test cases
	testCases, _ := extractRuleTestCases(ruleInfo)

	// If we have test cases, use them to generate detailed patterns
	if len(testCases) > 0 {
		b.WriteString("## Detailed Fix Patterns\n\n")
		b.WriteString(fmt.Sprintf("Based on rule `%s` test cases, here are the specific patterns to fix:\n\n", rule.ID))

		// Extract patterns from test cases
		passingExamples := []string{}
		failingExamples := []string{}

		for _, tc := range testCases {
			if tc.ShouldFail {
				if len(failingExamples) < 3 && tc.Input != "" {
					failingExamples = append(failingExamples, tc.Input)
				}
			} else {
				if len(passingExamples) < 2 && tc.Input != "" {
					passingExamples = append(passingExamples, tc.Input)
				}
			}
		}

		// Show before/after patterns
		for i, failing := range failingExamples {
			if i < len(passingExamples) {
				b.WriteString(fmt.Sprintf("### Pattern %d\n\n", i+1))
				b.WriteString("**Before** (❌ Violation):\n")
				lang := "go"
				if len(testCases) > 0 {
					lang = safeFallback(testCases[0].Language, "go")
				}
				b.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", lang, truncateCode(failing, 15)))
				b.WriteString("**After** (✅ Correct):\n")
				b.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", lang, truncateCode(passingExamples[i], 15)))
			}
		}

		b.WriteString("**Key Changes**:\n")

		// Add rule-specific guidance based on rule ID
		switch {
		case strings.Contains(rule.ID, "backoff") || strings.Contains(rule.ID, "jitter"):
			b.WriteString("- Add `import \"math/rand\"` or `import \"time\"` for jitter generation\n")
			b.WriteString("- Calculate exponential delay: `delay := baseDelay * math.Pow(2, float64(attempt))`\n")
			b.WriteString("- Add random jitter: `jitter := time.Duration(rand.Float64() * float64(delay) * 0.25)`\n")
			b.WriteString("- Apply both: `time.Sleep(delay + jitter)`\n\n")

			b.WriteString("**Common Pitfalls**:\n")
			b.WriteString("- ❌ Deterministic jitter: `jitter := delay * (attempt / maxRetries)` (same for all instances!)\n")
			b.WriteString("- ✅ Random jitter: `jitter := time.Duration(rand.Float64() * float64(delay))`\n\n")

		case strings.Contains(rule.ID, "health"):
			b.WriteString("- Add health endpoint: `r.HandleFunc(\"/health\", healthHandler).Methods(\"GET\")`\n")
			b.WriteString("- Return JSON: `w.Header().Set(\"Content-Type\", \"application/json\")`\n")
			b.WriteString("- Include status: `{\"status\": \"healthy\", \"timestamp\": \"...\"}`\n\n")

		case strings.Contains(rule.ID, "content-type") || strings.Contains(rule.ID, "header"):
			b.WriteString("- Set Content-Type before writing response\n")
			b.WriteString("- Use appropriate MIME type: `application/json`, `text/html`, etc.\n")
			b.WriteString("- Example: `w.Header().Set(\"Content-Type\", \"application/json\")`\n\n")

		case strings.Contains(rule.ID, "status") && strings.Contains(rule.ID, "code"):
			b.WriteString("- Use semantic HTTP status codes:\n")
			b.WriteString("  - 200 OK: Successful GET/PUT\n")
			b.WriteString("  - 201 Created: Successful POST creating resource\n")
			b.WriteString("  - 400 Bad Request: Invalid input\n")
			b.WriteString("  - 404 Not Found: Resource doesn't exist\n")
			b.WriteString("  - 500 Internal Server Error: Server failure\n\n")

		case strings.Contains(rule.ID, "port"):
			b.WriteString("- Use approved port ranges:\n")
			b.WriteString("  - API services: 15000-19999\n")
			b.WriteString("  - UI services: 35000-39999\n")
			b.WriteString("- Update service.json with correct ports\n\n")

		case strings.Contains(rule.ID, "makefile"):
			b.WriteString("- Add required targets: `run`, `test`, `logs`, `stop`, `clean`\n")
			b.WriteString("- Use `.PHONY:` declarations for non-file targets\n")
			b.WriteString("- Follow existing Makefile patterns from other scenarios\n\n")

		case strings.Contains(rule.ID, "test") && strings.Contains(rule.ID, "exit"):
			b.WriteString("- Ensure test scripts exit with non-zero on failure\n")
			b.WriteString("- Use `set -e` at top of bash scripts for early exit\n")
			b.WriteString("- Return proper exit codes: `exit 0` (success), `exit 1` (failure)\n\n")

		default:
			// Generic guidance if no specific pattern found
			b.WriteString("- Review the passing examples in the attached artifact\n")
			b.WriteString("- Apply the same pattern to your violations\n")
			b.WriteString("- Verify with the rule scanner after changes\n\n")
		}
	}
}

// truncateCode truncates code to max lines for readability
func truncateCode(code string, maxLines int) string {
	lines := strings.Split(code, "\n")
	if len(lines) <= maxLines {
		return code
	}
	return strings.Join(lines[:maxLines], "\n") + "\n// ... (truncated)"
}

func buildViolationArtifact(rule Rule, scenarios []string, ruleInfo RuleInfo) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Violations for Rule: %s\n\n", safeFallback(rule.Name, rule.ID)))
	b.WriteString(fmt.Sprintf("**Rule ID**: %s\n", rule.ID))
	b.WriteString(fmt.Sprintf("**Severity**: %s\n", rule.Severity))
	b.WriteString(fmt.Sprintf("**Category**: %s\n\n", rule.Category))

	if rule.Description != "" {
		b.WriteString(fmt.Sprintf("**Description**: %s\n\n", rule.Description))
	}

	// NEW: Include detection logic summary
	b.WriteString("## What This Rule Detects\n\n")
	summary := extractRuleDetectionSummary(ruleInfo)
	if summary != "" {
		b.WriteString(summary)
		b.WriteString("\n\n")
	} else {
		b.WriteString("See rule implementation for detection logic.\n\n")
	}

	// NEW: Include code examples from rule test cases
	b.WriteString("## Passing vs Failing Examples\n\n")
	testCases, _ := extractRuleTestCases(ruleInfo)
	if len(testCases) > 0 {
		// Show 1 passing example
		foundPassingExample := false
		for _, tc := range testCases {
			if !tc.ShouldFail {
				b.WriteString("### ✅ Correct Implementation\n")
				b.WriteString(fmt.Sprintf("_%s_\n\n", tc.Description))
				lang := safeFallback(tc.Language, "go")
				b.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", lang, tc.Input))
				foundPassingExample = true
				break
			}
		}

		// Show 1-2 failing examples
		if foundPassingExample {
			failCount := 0
			for _, tc := range testCases {
				if tc.ShouldFail && failCount < 2 {
					b.WriteString(fmt.Sprintf("### ❌ Violation: %s\n", tc.Description))
					lang := safeFallback(tc.Language, "go")
					b.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", lang, tc.Input))
					failCount++
				}
			}
		}
	} else {
		b.WriteString("_See rule implementation file for code examples._\n\n")
	}

	b.WriteString("## Detailed Violations by Scenario\n\n")

	// NEW: Actually scan each scenario and include violation details!
	for i, scenario := range scenarios {
		b.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, scenario))

		// Run the rule against this scenario to get actual violations
		violations := scanScenarioForRule(scenario, rule.ID, ruleInfo)

		if len(violations) == 0 {
			b.WriteString("_No violations found (may have been fixed since scan)_\n\n")
			continue
		}

		for _, violation := range violations {
			relPath := violation.FilePath
			if scenarioRoot := getScenarioRoot(); scenarioRoot != "" {
				if rel, err := filepath.Rel(scenarioRoot, violation.FilePath); err == nil {
					relPath = rel
				}
			}

			b.WriteString(fmt.Sprintf("#### File: `%s:%d`\n\n", relPath, violation.LineNumber))

			if violation.Title != "" {
				b.WriteString(fmt.Sprintf("**Issue**: %s\n\n", violation.Title))
			}

			if violation.Description != "" {
				b.WriteString(fmt.Sprintf("**Description**: %s\n\n", violation.Description))
			}

			if violation.CodeSnippet != "" {
				b.WriteString("**Code Context**:\n")
				b.WriteString(fmt.Sprintf("```go\n%s\n```\n\n", violation.CodeSnippet))
			}

			if violation.Recommendation != "" {
				b.WriteString(fmt.Sprintf("**Fix**: %s\n\n", violation.Recommendation))
			}

			b.WriteString("---\n\n")
		}
	}

	b.WriteString("## Fix Workflow\n\n")
	b.WriteString("For each violation above:\n\n")
	b.WriteString("1. **Locate**: Navigate to the file and line number shown\n")
	b.WriteString("2. **Understand**: Review the code context and violation description\n")
	b.WriteString("3. **Apply Fix**: Use the recommended fix pattern or follow the passing examples\n")
	b.WriteString("4. **Verify**: Run `scenario-auditor scan " + scenarios[0] + " --rule " + rule.ID + "` to confirm fix\n")
	b.WriteString("5. **Test**: Run scenario tests: `cd scenarios/{scenario} && make test`\n")
	b.WriteString("6. **Validate**: Verify service still starts and functions correctly\n\n")

	b.WriteString("**Repeat for all scenarios listed above.**\n")

	return b.String()
}

// scanScenarioForRule runs a specific rule against a scenario and returns violations
func scanScenarioForRule(scenarioName, ruleID string, ruleInfo RuleInfo) []rulespkg.Violation {
	logger := NewLogger()

	if !ruleInfo.Implementation.Valid {
		logger.Warn(fmt.Sprintf("Rule %s implementation not valid, skipping scan", ruleID), nil)
		return nil
	}

	scenarioRoot := getScenarioRoot()
	scenarioPath := filepath.Join(scenarioRoot, "scenarios", scenarioName)

	// Check if scenario exists
	if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
		logger.Warn(fmt.Sprintf("Scenario path does not exist: %s", scenarioPath), nil)
		return nil
	}

	var violations []rulespkg.Violation

	// Walk the scenario directory
	err := filepath.Walk(scenarioPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil // Skip errors, continue walking
		}

		if info.IsDir() {
			// Skip common directories that don't contain source code
			base := filepath.Base(path)
			if base == "node_modules" || base == ".git" || base == "vendor" ||
				base == "dist" || base == "build" || base == ".next" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file matches rule targets
		ext := filepath.Ext(path)
		shouldCheck := false
		for _, target := range ruleInfo.Targets {
			switch target {
			case "api":
				if ext == ".go" && strings.Contains(path, "/api/") {
					shouldCheck = true
				}
			case "config":
				if ext == ".json" || ext == ".yaml" || ext == ".yml" ||
					strings.HasSuffix(path, "Makefile") {
					shouldCheck = true
				}
			case "ui":
				if ext == ".js" || ext == ".jsx" || ext == ".ts" || ext == ".tsx" {
					shouldCheck = true
				}
			case "cli":
				if ext == ".sh" || ext == ".go" && strings.Contains(path, "/cli/") {
					shouldCheck = true
				}
			case "test":
				if ext == ".go" && strings.Contains(path, "_test.go") {
					shouldCheck = true
				}
			}
		}

		if !shouldCheck {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		// Run rule check
		ruleViolations, execErr := ruleInfo.Check(string(content), path, scenarioName)
		if execErr != nil {
			logger.Warn(fmt.Sprintf("Rule execution failed on %s: %v", path, execErr), nil)
			return nil
		}

		// Convert to our Violation type
		for _, rv := range ruleViolations {
			violations = append(violations, convertRuleViolationToOurType(rv))
		}

		return nil
	})

	if err != nil {
		logger.Warn(fmt.Sprintf("Error walking scenario %s: %v", scenarioName, err), nil)
	}

	return violations
}

// convertRuleViolationToOurType converts from rules package Violation to our Violation type
func convertRuleViolationToOurType(rv interface{}) rulespkg.Violation {
	v := rulespkg.Violation{}

	// The Check method returns []rulespkg.Violation, which has the same structure
	// We can use type assertion since they have identical fields
	type sourceViolation struct {
		ID             string
		RuleID         string
		Type           string
		Severity       string
		Title          string
		Message        string
		Description    string
		File           string
		FilePath       string
		Line           int
		LineNumber     int
		CodeSnippet    string
		Recommendation string
		Standard       string
		Category       string
	}

	// Try direct struct conversion first
	switch src := rv.(type) {
	case sourceViolation:
		v.ID = src.ID
		v.RuleID = src.RuleID
		v.Type = src.Type
		v.Severity = src.Severity
		v.Title = src.Title
		v.Message = src.Message
		v.Description = src.Description
		v.File = src.File
		v.FilePath = src.FilePath
		v.Line = src.Line
		v.LineNumber = src.LineNumber
		v.CodeSnippet = src.CodeSnippet
		v.Recommendation = src.Recommendation
		v.Standard = src.Standard
		v.Category = src.Category
	}

	// Ensure FilePath is set (some rules use FilePath, others use File)
	if v.FilePath == "" && v.File != "" {
		v.FilePath = v.File
	}

	// Ensure LineNumber is set (some rules use LineNumber, others use Line)
	if v.LineNumber == 0 && v.Line != 0 {
		v.LineNumber = v.Line
	}

	return v
}

// extractRuleDetectionSummary extracts a human-readable summary from rule comments
func extractRuleDetectionSummary(ruleInfo RuleInfo) string {
	// Read rule file
	content, err := os.ReadFile(ruleInfo.FilePath)
	if err != nil {
		return ""
	}

	source := string(content)

	// Extract from comment block at top of file
	// Look for "Description:" or "Reason:" fields
	lines := strings.Split(source, "\n")
	var summary strings.Builder
	inComment := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Start of comment block
		if strings.HasPrefix(trimmed, "/*") {
			inComment = true
			continue
		}

		// End of comment block
		if strings.HasPrefix(trimmed, "*/") {
			break
		}

		if inComment {
			// Extract Description
			if strings.HasPrefix(trimmed, "Description:") {
				desc := strings.TrimPrefix(trimmed, "Description:")
				summary.WriteString(strings.TrimSpace(desc))
				summary.WriteString("\n\n")
			}

			// Extract Reason
			if strings.HasPrefix(trimmed, "Reason:") {
				reason := strings.TrimPrefix(trimmed, "Reason:")
				summary.WriteString("**Why This Matters**: ")
				summary.WriteString(strings.TrimSpace(reason))
				summary.WriteString("\n")
			}
		}
	}

	result := summary.String()
	if result == "" && ruleInfo.Reason != "" {
		return ruleInfo.Reason
	}

	return strings.TrimSpace(result)
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

func relativePathFrom(absPath, base string) string {
	rel, err := filepath.Rel(base, absPath)
	if err != nil {
		return absPath
	}
	return rel
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

// buildCreateRuleIssuePayload builds the payload for creating a rule creation issue
func buildCreateRuleIssuePayload(req createRuleRequest) (map[string]interface{}, error) {

	title := fmt.Sprintf("[scenario-auditor] Create new rule: %s", req.Name)
	description := buildCreateRuleDescription(req)

	metadata := map[string]string{
		"reported_by":   "scenario-auditor",
		"report_type":   "create_rule",
		"rule_name":     req.Name,
		"rule_category": req.Category,
		"rule_severity": req.Severity,
	}

	environment := map[string]string{
		"rule_name":     req.Name,
		"rule_category": req.Category,
		"rule_severity": req.Severity,
	}

	// Read and attach rule creation prompt template
	promptPath := filepath.Join(getScenarioRoot(), "prompts", "rule-creation.txt")
	promptContent, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rule creation prompt: %w", err)
	}

	artifacts := []map[string]interface{}{
		{
			"name":         "rule-creation-guidelines.txt",
			"category":     "rule_creation",
			"content":      string(promptContent),
			"encoding":     "plain",
			"content_type": "text/plain",
		},
	}

	payload := map[string]interface{}{
		"title":          title,
		"description":    description,
		"type":           "task",
		"priority":       "medium",
		"app_id":         "scenario-auditor",
		"status":         "open",
		"tags":           []string{"scenario-auditor", "rule-creation", "ai-task"},
		"metadata_extra": metadata,
		"environment":    environment,
		"artifacts":      artifacts,
		"reporter_name":  "Scenario Auditor",
		"reporter_email": "auditor@vrooli.local",
	}

	return payload, nil
}

// buildCreateRuleDescription builds the detailed description for rule creation issue
func buildCreateRuleDescription(req createRuleRequest) string {
	var b strings.Builder

	b.WriteString("Scenario Auditor requests creation of a new quality rule.\n\n")

	// Rule Specification
	b.WriteString("## Rule Specification\n\n")
	b.WriteString(fmt.Sprintf("**Name**: %s\n", req.Name))
	b.WriteString(fmt.Sprintf("**Description**: %s\n", req.Description))
	b.WriteString(fmt.Sprintf("**Category**: %s\n", req.Category))
	b.WriteString(fmt.Sprintf("**Severity**: %s\n\n", req.Severity))

	if req.Motivation != "" {
		b.WriteString("### Motivation\n")
		b.WriteString(fmt.Sprintf("%s\n\n", req.Motivation))
	}

	// Implementation Requirements
	b.WriteString("## Implementation Requirements\n\n")
	b.WriteString("### 1. Create Rule File\n")
	b.WriteString(fmt.Sprintf("**Location**: `api/rules/%s/{descriptive-name}.go`\n\n", req.Category))
	b.WriteString("**Structure**:\n")
	b.WriteString("```go\n")
	b.WriteString("package " + req.Category + "\n\n")
	b.WriteString("/*\n")
	b.WriteString(fmt.Sprintf("Rule: %s\n", req.Name))
	b.WriteString(fmt.Sprintf("Description: %s\n", req.Description))
	b.WriteString(fmt.Sprintf("Category: %s\n", req.Category))
	b.WriteString(fmt.Sprintf("Severity: %s\n", req.Severity))
	b.WriteString("Targets: {file_pattern}\n")
	b.WriteString("Version: 1.0.0\n\n")
	b.WriteString("// Add comprehensive test-case blocks here following this format:\n")
	b.WriteString("<test-case id=\"{id}\" should-fail=\"{true|false}\" path=\"{relative/path}\">\n")
	b.WriteString("  <description>{description}</description>\n")
	b.WriteString("  <input language=\"{language}\">...</input>\n")
	b.WriteString("  <expected-violations>{count}</expected-violations>\n")
	b.WriteString("  <expected-message>{partial-match}</expected-message>\n")
	b.WriteString("</test-case>\n")
	b.WriteString("*/\n\n")
	b.WriteString("func Check" + sanitizeForFunctionName(req.Name) + "(content []byte, filePath string) []Violation {\n")
	b.WriteString("    // Implementation here\n")
	b.WriteString("    return nil\n")
	b.WriteString("}\n")
	b.WriteString("```\n\n")

	// Test Requirements
	b.WriteString("### 2. Embedded Test Cases\n")
	b.WriteString("**Minimum Coverage**:\n")
	b.WriteString("- At least 1 happy path test (should-fail=\"false\")\n")
	b.WriteString("- At least 2-3 violation detection tests (should-fail=\"true\")\n")
	b.WriteString("- Edge cases and false positive prevention\n\n")

	// Registration
	b.WriteString("### 3. Rule Registration\n")
	b.WriteString("Add to `api/rule_registry.go`:\n")
	b.WriteString("```go\n")
	b.WriteString(fmt.Sprintf("registry.Register(\"%s\", %s.Check%s)\n", generateRuleID(req.Name, req.Category), req.Category, sanitizeForFunctionName(req.Name)))
	b.WriteString("```\n\n")

	// Reference Materials
	b.WriteString("## Reference Materials\n\n")
	b.WriteString("**Gold Standard Example**: `api/rules/api/health_check.go`\n")
	b.WriteString("- Comprehensive test coverage (5+ test cases)\n")
	b.WriteString("- Clear violation messages with recommendations\n")
	b.WriteString("- Context-aware detection logic\n\n")

	b.WriteString("**Rule Creation Guidelines**: See attached `rule-creation-guidelines.txt` artifact\n\n")

	// Category-Specific Guidance
	b.WriteString(fmt.Sprintf("## Category-Specific Guidance: %s\n\n", req.Category))
	switch req.Category {
	case "api":
		b.WriteString("**API Rules Focus On**:\n")
		b.WriteString("- Go best practices and patterns\n")
		b.WriteString("- HTTP standards (status codes, headers, CORS)\n")
		b.WriteString("- Security patterns (input validation, sanitization)\n")
		b.WriteString("- Health check endpoints\n")
		b.WriteString("- Error handling consistency\n\n")
	case "config":
		b.WriteString("**Config Rules Focus On**:\n")
		b.WriteString("- service.json schema validation\n")
		b.WriteString("- Lifecycle completeness (health, setup, develop, test, stop)\n")
		b.WriteString("- Port range compliance\n")
		b.WriteString("- Environment variable naming\n")
		b.WriteString("- Makefile required targets\n\n")
	case "ui":
		b.WriteString("**UI Rules Focus On**:\n")
		b.WriteString("- Testing best practices\n")
		b.WriteString("- Accessibility standards (ARIA, semantic HTML)\n")
		b.WriteString("- Security headers configuration\n")
		b.WriteString("- Iframe bridge integration\n")
		b.WriteString("- Element testability (data-testid)\n\n")
	case "test":
		b.WriteString("**Test Rules Focus On**:\n")
		b.WriteString("- Phase-based test structure\n")
		b.WriteString("- Integration with centralized testing library\n")
		b.WriteString("- Coverage reporting and thresholds\n")
		b.WriteString("- Exit code correctness\n")
		b.WriteString("- Test runner compatibility\n\n")
	default:
		b.WriteString("**General Rule Guidelines**:\n")
		b.WriteString("- Clear, specific detection criteria\n")
		b.WriteString("- Actionable violation messages\n")
		b.WriteString("- Comprehensive test coverage\n")
		b.WriteString("- Performance-conscious implementation\n\n")
	}

	// Validation Workflow
	b.WriteString("## Validation Workflow\n\n")
	b.WriteString("1. **Test Rule Tests**:\n")
	b.WriteString("   ```bash\n")
	b.WriteString(fmt.Sprintf("   scenario-auditor test %s\n", generateRuleID(req.Name, req.Category)))
	b.WriteString("   ```\n\n")

	b.WriteString("2. **Scan Test Scenario**:\n")
	b.WriteString("   ```bash\n")
	b.WriteString(fmt.Sprintf("   scenario-auditor scan {test-scenario} --rule %s\n", generateRuleID(req.Name, req.Category)))
	b.WriteString("   ```\n\n")

	b.WriteString("3. **Verify Rule Registration**:\n")
	b.WriteString("   ```bash\n")
	b.WriteString("   scenario-auditor list-rules | grep " + generateRuleID(req.Name, req.Category) + "\n")
	b.WriteString("   ```\n\n")

	// Success Criteria
	b.WriteString("## Success Criteria\n\n")
	b.WriteString("- [ ] Rule file created with proper structure\n")
	b.WriteString("- [ ] All embedded test cases pass\n")
	b.WriteString("- [ ] Rule registered in rule_registry.go\n")
	b.WriteString("- [ ] Rule compiles without errors\n")
	b.WriteString("- [ ] Rule detects violations correctly\n")
	b.WriteString("- [ ] No false positives on valid code\n")
	b.WriteString("- [ ] Clear, actionable violation messages\n")

	return b.String()
}

// generateRuleID creates a rule ID from name and category
func generateRuleID(name, category string) string {
	// Convert "Health Check Endpoint" → "health-check-endpoint"
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "-")
	// Remove special characters
	id = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(id, "")
	return category + "-" + id
}

// sanitizeForFunctionName converts a rule name to a valid Go function name
func sanitizeForFunctionName(name string) string {
	// Convert to title case and remove spaces
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	funcName := strings.Join(words, "")
	// Remove non-alphanumeric characters
	funcName = regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(funcName, "")
	return funcName
}
