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
