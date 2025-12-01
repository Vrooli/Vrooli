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

	"github.com/google/uuid"
)

type issueTrackerAPIResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Error   string                 `json:"error"`
	Data    map[string]interface{} `json:"data"`
}

type IssueTrackerResult struct {
	IssueID  string
	IssueURL string
	Message  string
}

func submitTestGenerationIssue(ctx context.Context, requestID uuid.UUID, req GenerateTestSuiteRequest) (*IssueTrackerResult, error) {
	port, err := resolveScenarioPortViaCLI(ctx, "app-issue-tracker", "API_PORT")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve app-issue-tracker API port: %w", err)
	}

	payload, err := buildIssueTrackerPayload(requestID, req)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal issue payload: %w", err)
	}

	endpoint := fmt.Sprintf("http://localhost:%d/api/v1/issues", port)

	operation := func() (interface{}, error) {
		httpCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
		defer cancel()

		request, err := http.NewRequestWithContext(httpCtx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create issue request: %w", err)
		}
		request.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 25 * time.Second}
		response, err := client.Do(request)
		if err != nil {
			return nil, fmt.Errorf("failed to call app-issue-tracker: %w", err)
		}
		defer response.Body.Close()

		if response.StatusCode < 200 || response.StatusCode >= 300 {
			data, _ := io.ReadAll(response.Body)
			return nil, fmt.Errorf("app-issue-tracker returned status %d: %s", response.StatusCode, strings.TrimSpace(string(data)))
		}

		var trackerResp issueTrackerAPIResponse
		if err := json.NewDecoder(response.Body).Decode(&trackerResp); err != nil {
			return nil, fmt.Errorf("failed to decode app-issue-tracker response: %w", err)
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

		result := &IssueTrackerResult{Message: strings.TrimSpace(trackerResp.Message)}
		if trackerResp.Data != nil {
			if value, ok := trackerResp.Data["issue_id"].(string); ok {
				result.IssueID = value
			} else if value, ok := trackerResp.Data["issueId"].(string); ok {
				result.IssueID = value
			} else if rawIssue, ok := trackerResp.Data["issue"]; ok {
				switch v := rawIssue.(type) {
				case map[string]interface{}:
					if nested, ok := v["id"].(string); ok {
						result.IssueID = nested
					}
				}
			}
		}

		if result.Message == "" {
			result.Message = "Issue created successfully"
		}

		if result.IssueID != "" {
			if issueURL, urlErr := buildIssueTrackerURL(ctx, result.IssueID); urlErr == nil {
				result.IssueURL = issueURL
			}
		}

		return result, nil
	}

	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var raw interface{}
	if issueTrackerCircuitBreaker != nil {
		wrappedResult, execErr := issueTrackerCircuitBreaker.Execute(execCtx, operation)
		if execErr != nil {
			return nil, execErr
		}
		raw = wrappedResult
	} else {
		unwrapped, execErr := operation()
		if execErr != nil {
			return nil, execErr
		}
		raw = unwrapped
	}

	result, ok := raw.(*IssueTrackerResult)
	if !ok || result == nil {
		return nil, errors.New("unexpected response from issue tracker operation")
	}

	return result, nil
}

func submitTestEnhancementIssue(ctx context.Context, requestID uuid.UUID, req GenerateTestSuiteRequest) (*IssueTrackerResult, error) {
	port, err := resolveScenarioPortViaCLI(ctx, "app-issue-tracker", "API_PORT")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve app-issue-tracker API port: %w", err)
	}

	payload, err := buildIssueTrackerEnhancementPayload(requestID, req)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal issue payload: %w", err)
	}

	endpoint := fmt.Sprintf("http://localhost:%d/api/v1/issues", port)

	operation := func() (interface{}, error) {
		httpCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
		defer cancel()

		request, err := http.NewRequestWithContext(httpCtx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create issue request: %w", err)
		}
		request.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 25 * time.Second}
		response, err := client.Do(request)
		if err != nil {
			return nil, fmt.Errorf("failed to call app-issue-tracker: %w", err)
		}
		defer response.Body.Close()

		if response.StatusCode < 200 || response.StatusCode >= 300 {
			data, _ := io.ReadAll(response.Body)
			return nil, fmt.Errorf("app-issue-tracker returned status %d: %s", response.StatusCode, strings.TrimSpace(string(data)))
		}

		var trackerResp issueTrackerAPIResponse
		if err := json.NewDecoder(response.Body).Decode(&trackerResp); err != nil {
			return nil, fmt.Errorf("failed to decode app-issue-tracker response: %w", err)
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

		result := &IssueTrackerResult{Message: strings.TrimSpace(trackerResp.Message)}
		if trackerResp.Data != nil {
			if value, ok := trackerResp.Data["issue_id"].(string); ok {
				result.IssueID = value
			} else if value, ok := trackerResp.Data["issueId"].(string); ok {
				result.IssueID = value
			} else if rawIssue, ok := trackerResp.Data["issue"]; ok {
				switch v := rawIssue.(type) {
				case map[string]interface{}:
					if nested, ok := v["id"].(string); ok {
						result.IssueID = nested
					}
				}
			}
		}

		if result.Message == "" {
			result.Message = "Enhancement issue created successfully"
		}

		if result.IssueID != "" {
			if issueURL, urlErr := buildIssueTrackerURL(ctx, result.IssueID); urlErr == nil {
				result.IssueURL = issueURL
			}
		}

		return result, nil
	}

	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var raw interface{}
	if issueTrackerCircuitBreaker != nil {
		wrappedResult, execErr := issueTrackerCircuitBreaker.Execute(execCtx, operation)
		if execErr != nil {
			return nil, execErr
		}
		raw = wrappedResult
	} else {
		unwrapped, execErr := operation()
		if execErr != nil {
			return nil, execErr
		}
		raw = unwrapped
	}

	result, ok := raw.(*IssueTrackerResult)
	if !ok || result == nil {
		return nil, errors.New("unexpected response from issue tracker operation")
	}

	return result, nil
}

func buildIssueTrackerEnhancementPayload(requestID uuid.UUID, req GenerateTestSuiteRequest) (map[string]interface{}, error) {
	scenario := strings.TrimSpace(req.ScenarioName)
	if scenario == "" {
		return nil, errors.New("scenario name is required for issue payload")
	}

	testTypes := make([]string, 0, len(req.TestTypes))
	for _, t := range req.TestTypes {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" {
			testTypes = append(testTypes, trimmed)
		}
	}

	if len(testTypes) == 0 {
		testTypes = []string{"unit"}
	}

	requestSpec := map[string]interface{}{
		"request_id":      requestID.String(),
		"request_type":    "enhancement",
		"scenario":        scenario,
		"requested_at":    time.Now().UTC().Format(time.RFC3339),
		"focus_areas":     testTypes,
		"coverage_target": req.CoverageTarget,
		"options":         req.Options,
	}

	specBytes, err := json.MarshalIndent(requestSpec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request spec: %w", err)
	}

	description := buildIssueTrackerEnhancementDescription(scenario, testTypes, req)

	metadata := map[string]string{
		"requested_by":   "test-genie",
		"request_id":     requestID.String(),
		"request_type":   "enhancement",
		"scenario":       scenario,
		"target_primary": scenario,
	}

	environment := map[string]string{
		"scenario":        scenario,
		"focus_areas":     strings.Join(testTypes, ","),
		"coverage_target": fmt.Sprintf("%.2f", req.CoverageTarget),
	}

	artifactName := fmt.Sprintf("%s-test-enhancement.json", sanitizeForFilename(scenario))
	artifacts := []map[string]interface{}{
		{
			"name":         artifactName,
			"category":     "test_enhancement",
			"content":      string(specBytes),
			"encoding":     "plain",
			"content_type": "application/json",
		},
	}

	targets := buildTestGenieTargets(scenario)
	if len(targets) == 0 {
		return nil, errors.New("no valid targets available for enhancement payload")
	}
	metadata["target_count"] = strconv.Itoa(len(targets))

	payload := map[string]interface{}{
		"title":          fmt.Sprintf("Enhance test suite for %s", scenario),
		"description":    description,
		"type":           "task",
		"priority":       "medium",
		"status":         "open",
		"tags":           []string{"test-genie", "test-enhancement", "quality-improvement"},
		"metadata_extra": metadata,
		"environment":    environment,
		"artifacts":      artifacts,
		"reporter_name":  "Test Genie",
		"reporter_email": "test-genie@vrooli.local",
		"targets":        targets,
	}

	if req.Model != "" {
		metadata["preferred_model"] = strings.TrimSpace(req.Model)
	}

	return payload, nil
}

func buildIssueTrackerPayload(requestID uuid.UUID, req GenerateTestSuiteRequest) (map[string]interface{}, error) {
	scenario := strings.TrimSpace(req.ScenarioName)
	if scenario == "" {
		return nil, errors.New("scenario name is required for issue payload")
	}

	testTypes := make([]string, 0, len(req.TestTypes))
	for _, t := range req.TestTypes {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" {
			testTypes = append(testTypes, trimmed)
		}
	}

	if len(testTypes) == 0 {
		testTypes = []string{"unit"}
	}

	requestSpec := map[string]interface{}{
		"request_id":      requestID.String(),
		"scenario":        scenario,
		"requested_at":    time.Now().UTC().Format(time.RFC3339),
		"test_types":      testTypes,
		"coverage_target": req.CoverageTarget,
		"options":         req.Options,
	}

	specBytes, err := json.MarshalIndent(requestSpec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request spec: %w", err)
	}

	description := buildIssueTrackerDescription(scenario, testTypes, req)

	metadata := map[string]string{
		"requested_by":   "test-genie",
		"request_id":     requestID.String(),
		"scenario":       scenario,
		"target_primary": scenario,
	}

	environment := map[string]string{
		"scenario":             scenario,
		"requested_test_types": strings.Join(testTypes, ","),
		"coverage_target":      fmt.Sprintf("%.2f", req.CoverageTarget),
	}

	artifactName := fmt.Sprintf("%s-test-request.json", sanitizeForFilename(scenario))
	artifacts := []map[string]interface{}{
		{
			"name":         artifactName,
			"category":     "test_generation",
			"content":      string(specBytes),
			"encoding":     "plain",
			"content_type": "application/json",
		},
	}

	targets := buildTestGenieTargets(scenario)
	if len(targets) == 0 {
		return nil, errors.New("no valid targets available for issue payload")
	}
	metadata["target_count"] = strconv.Itoa(len(targets))

	payload := map[string]interface{}{
		"title":          fmt.Sprintf("Generate automated tests for %s", scenario),
		"description":    description,
		"type":           "task",
		"priority":       "high",
		"status":         "open",
		"tags":           []string{"test-genie", "test-generation", "automation"},
		"metadata_extra": metadata,
		"environment":    environment,
		"artifacts":      artifacts,
		"reporter_name":  "Test Genie",
		"reporter_email": "test-genie@vrooli.local",
		"targets":        targets,
	}

	if req.Model != "" {
		metadata["preferred_model"] = strings.TrimSpace(req.Model)
	}

	return payload, nil
}

func buildIssueTrackerEnhancementDescription(scenario string, testTypes []string, req GenerateTestSuiteRequest) string {
	var builder strings.Builder
	builder.WriteString("Test Genie requests test suite enhancement and quality improvement for the following scenario.\n\n")
	builder.WriteString(fmt.Sprintf("Scenario: %s\n", scenario))
	builder.WriteString(fmt.Sprintf("Focus areas: %s\n", strings.Join(testTypes, ", ")))
	if req.CoverageTarget > 0 {
		builder.WriteString(fmt.Sprintf("Target coverage: %.2f%%\n", req.CoverageTarget))
	}

	builder.WriteString("\n## 🎯 YOUR TASK: IMPLEMENT IMPROVEMENTS (NOT JUST INVESTIGATE)\n\n")
	builder.WriteString("**This is an IMPLEMENTATION task, not an investigation task.**\n\n")
	builder.WriteString("You must:\n")
	builder.WriteString("1. **Assess**: Run existing tests, generate coverage reports, identify gaps\n")
	builder.WriteString("2. **Implement**: Write new tests and improve existing ones following gold standards\n")
	builder.WriteString("3. **Verify**: Run full test suite, confirm coverage increases, all tests pass\n")
	builder.WriteString("4. **Report**: Document what you implemented and the coverage improvement\n\n")
	builder.WriteString("**DO NOT** just create recommendations or analysis documents. Write the actual test code.\n\n")

	builder.WriteString("## 🚨 CRITICAL SAFETY BOUNDARIES\n\n")
	builder.WriteString("**NEVER use git commands**:\n")
	builder.WriteString("- ❌ NO `git commit`, `git add`, `git push`, `git rebase`, or ANY git operations\n")
	builder.WriteString("- ❌ NO git rollbacks or history manipulation\n")
	builder.WriteString("- ✅ Edit files directly, verify tests pass, then STOP\n")
	builder.WriteString("- ✅ Report your changes; committing is the human's responsibility\n\n")

	builder.WriteString("**Stay within scenario boundaries**:\n")
	builder.WriteString(fmt.Sprintf("- ✅ ONLY modify files within `scenarios/%s/`\n", scenario))
	builder.WriteString("- ❌ DO NOT touch shared libraries, other scenarios, or root-level code\n")
	builder.WriteString("- ❌ DO NOT modify centralized testing infrastructure in `scripts/scenarios/testing/`\n")
	builder.WriteString("- ⚠️  If you need changes outside this scenario, STOP and report the dependency\n\n")

	builder.WriteString("**Cross-scenario awareness**:\n")
	builder.WriteString("- This scenario may be consumed by other scenarios or the ecosystem\n")
	builder.WriteString("- Do not break existing API contracts, endpoints, or behavior\n")
	builder.WriteString("- Tests should verify behavior, not change it\n")
	builder.WriteString("- If tests reveal bugs, note them but do not fix production code without explicit approval\n\n")

	builder.WriteString("## 📋 Implementation Workflow\n\n")
	builder.WriteString("**Step 1: Assessment** (understand current state)\n")
	builder.WriteString("```bash\n")
	builder.WriteString(fmt.Sprintf("cd scenarios/%s\n", scenario))
	builder.WriteString("make test  # Run existing tests\n")
	builder.WriteString("# Review test output and coverage reports\n")
	builder.WriteString("# Identify gaps in coverage, missing error cases, untested functions\n")
	builder.WriteString("```\n\n")

	builder.WriteString("**Step 2: Implementation** (write the tests)\n")
	builder.WriteString("- Add missing test cases to existing `*_test.go` files\n")
	builder.WriteString("- Create new test files if covering new modules\n")
	builder.WriteString("- Update `test_helpers.go` with reusable utilities\n")
	builder.WriteString("- Follow visited-tracker patterns (TestScenarioBuilder, ErrorTestPattern)\n")
	builder.WriteString("- Ensure proper cleanup with defer statements\n\n")

	builder.WriteString("**Step 3: Verification** (prove it works)\n")
	builder.WriteString("```bash\n")
	builder.WriteString("make test  # All tests must pass\n")
	builder.WriteString("# Confirm coverage increased\n")
	builder.WriteString("# Document before/after coverage percentages\n")
	builder.WriteString("```\n\n")

	builder.WriteString("**Step 4: Completion** (report and stop)\n")
	builder.WriteString("- Summarize what tests were added/improved\n")
	builder.WriteString("- Report coverage improvement (e.g., \"45% → 78%\")\n")
	builder.WriteString("- List any discovered bugs or issues\n")
	builder.WriteString("- STOP - do not commit, push, or continue iterating\n\n")

	return builder.String() + buildSharedTestingGuidance(scenario, testTypes, req)
}

func buildTestGenieTargets(scenario string) []map[string]string {
	trimmed := strings.TrimSpace(scenario)
	if trimmed == "" {
		return nil
	}

	return []map[string]string{
		{
			"type": "scenario",
			"id":   trimmed,
			"name": trimmed,
		},
	}
}

func buildIssueTrackerDescription(scenario string, testTypes []string, req GenerateTestSuiteRequest) string {
	var builder strings.Builder
	builder.WriteString("Test Genie requests automated test generation for the following scenario.\n\n")
	builder.WriteString(fmt.Sprintf("Scenario: %s\n", scenario))
	builder.WriteString(fmt.Sprintf("Requested test types: %s\n", strings.Join(testTypes, ", ")))
	if req.CoverageTarget > 0 {
		builder.WriteString(fmt.Sprintf("Target coverage: %.2f%%\n", req.CoverageTarget))
	}

	builder.WriteString("\n## 🎯 YOUR TASK: IMPLEMENT COMPREHENSIVE TESTS\n\n")
	builder.WriteString("**This is an IMPLEMENTATION task.** Generate complete, working test suites that:\n")
	builder.WriteString("1. Follow gold standard patterns from visited-tracker\n")
	builder.WriteString("2. Achieve the target coverage\n")
	builder.WriteString("3. Pass all tests when run with `make test`\n")
	builder.WriteString("4. Include helpers, patterns, and phase integration\n\n")
	builder.WriteString("**DO NOT** just analyze or recommend - write the actual test files.\n\n")

	builder.WriteString("## 🚨 CRITICAL SAFETY BOUNDARIES\n\n")
	builder.WriteString("**NEVER use git commands**:\n")
	builder.WriteString("- ❌ NO `git commit`, `git add`, `git push`, `git rebase`, or ANY git operations\n")
	builder.WriteString("- ❌ NO git rollbacks or history manipulation\n")
	builder.WriteString("- ✅ Create test files, verify they pass, then STOP\n")
	builder.WriteString("- ✅ Report your work; committing is the human's responsibility\n\n")

	builder.WriteString("**Stay within scenario boundaries**:\n")
	builder.WriteString(fmt.Sprintf("- ✅ ONLY create/modify files within `scenarios/%s/`\n", scenario))
	builder.WriteString("- ❌ DO NOT touch shared libraries, other scenarios, or root-level code\n")
	builder.WriteString("- ❌ DO NOT modify centralized testing infrastructure in `scripts/scenarios/testing/`\n")
	builder.WriteString("- ⚠️  If you need changes outside this scenario, STOP and report the dependency\n\n")

	builder.WriteString("**Cross-scenario awareness**:\n")
	builder.WriteString("- This scenario may be consumed by other scenarios or the ecosystem\n")
	builder.WriteString("- Tests verify behavior without changing it\n")
	builder.WriteString("- If tests reveal bugs, document them but do not fix production code without explicit approval\n\n")

	return builder.String() + buildSharedTestingGuidance(scenario, testTypes, req)
}

func buildSharedTestingGuidance(scenario string, testTypes []string, req GenerateTestSuiteRequest) string {
	var builder strings.Builder

	builder.WriteString("\n## Testing Infrastructure Integration\n\n")
	builder.WriteString("**Centralized Testing Library**: All tests must integrate with Vrooli's centralized testing infrastructure at `scripts/scenarios/testing/`:\n")
	builder.WriteString("- Source unit test runners from `scripts/scenarios/testing/unit/run-all.sh`\n")
	builder.WriteString("- Use phase helpers from `scripts/scenarios/testing/shell/phase-helpers.sh`\n")
	builder.WriteString("- Coverage thresholds: `--coverage-warn 80 --coverage-error 50`\n\n")

	builder.WriteString("**Test Organization Requirements**:\n")
	builder.WriteString("```\n")
	builder.WriteString(fmt.Sprintf("scenarios/%s/\n", scenario))
	builder.WriteString("├── api/\n")
	builder.WriteString("│   ├── test_helpers.go         # Reusable test utilities\n")
	builder.WriteString("│   ├── test_patterns.go        # Systematic error patterns\n")
	builder.WriteString("│   ├── main_test.go            # Comprehensive tests\n")
	builder.WriteString("│   └── *_test.go               # Additional test files\n")
	builder.WriteString("├── test/\n")
	builder.WriteString("│   ├── phases/\n")
	builder.WriteString("│   │   ├── test-unit.sh        # Must source centralized runners\n")
	builder.WriteString("│   │   ├── test-integration.sh\n")
	builder.WriteString("│   │   └── test-*.sh           # Other test phases\n")
	builder.WriteString("│   └── cli/\n")
	builder.WriteString("│       └── run-cli-tests.sh    # BATS CLI integration tests\n")
	builder.WriteString("```\n\n")

	builder.WriteString("## Language-Specific Patterns (Gold Standard: visited-tracker)\n\n")
	builder.WriteString("### Go Testing Requirements\n")
	builder.WriteString("1. **Helper Library** (`api/test_helpers.go`):\n")
	builder.WriteString("   - `setupTestLogger()` - Controlled logging during tests\n")
	builder.WriteString("   - `setupTestDirectory()` - Isolated test environments with cleanup\n")
	builder.WriteString("   - `makeHTTPRequest()` - Simplified HTTP request creation\n")
	builder.WriteString("   - `assertJSONResponse()` - Validate JSON responses\n")
	builder.WriteString("   - `assertErrorResponse()` - Validate error responses\n\n")

	builder.WriteString("2. **Pattern Library** (`api/test_patterns.go`):\n")
	builder.WriteString("   - `TestScenarioBuilder` - Fluent interface for building test scenarios\n")
	builder.WriteString("   - `ErrorTestPattern` - Systematic error condition testing\n")
	builder.WriteString("   - `HandlerTestSuite` - Comprehensive HTTP handler testing\n\n")

	builder.WriteString("3. **Test Structure**:\n")
	builder.WriteString("   ```go\n")
	builder.WriteString("   func TestHandler(t *testing.T) {\n")
	builder.WriteString("       cleanup := setupTestLogger()\n")
	builder.WriteString("       defer cleanup()\n")
	builder.WriteString("       \n")
	builder.WriteString("       env := setupTestDirectory(t)\n")
	builder.WriteString("       defer env.Cleanup()\n")
	builder.WriteString("       \n")
	builder.WriteString("       t.Run(\"Success\", func(t *testing.T) { /* happy path */ })\n")
	builder.WriteString("       t.Run(\"ErrorPaths\", func(t *testing.T) { /* systematic errors */ })\n")
	builder.WriteString("       t.Run(\"EdgeCases\", func(t *testing.T) { /* boundaries */ })\n")
	builder.WriteString("   }\n")
	builder.WriteString("   ```\n\n")

	builder.WriteString("### Node.js Testing Requirements\n")
	builder.WriteString("- Jest/Vitest configuration with coverage thresholds\n")
	builder.WriteString("- React Testing Library for components\n")
	builder.WriteString("- Mock fetch/API calls globally\n")
	builder.WriteString("- Factory functions for test data\n")
	builder.WriteString("- Coverage: 80% minimum across branches/functions/lines/statements\n\n")

	builder.WriteString("### Python Testing Requirements\n")
	builder.WriteString("- pytest with parametrize decorators\n")
	builder.WriteString("- Fixtures for common setup/teardown\n")
	builder.WriteString("- unittest.mock for external dependencies\n")
	builder.WriteString("- Coverage: 80% minimum with `--cov-fail-under=80`\n\n")

	builder.WriteString("## Test Quality Standards\n\n")
	builder.WriteString("**Each test must include**:\n")
	builder.WriteString("1. **Setup Phase**: Logger, isolated directory, test data\n")
	builder.WriteString("2. **Success Cases**: Happy path with complete assertions\n")
	builder.WriteString("3. **Error Cases**: Invalid inputs, missing resources, malformed data\n")
	builder.WriteString("4. **Edge Cases**: Empty inputs, boundary conditions, null values\n")
	builder.WriteString("5. **Cleanup**: Always defer cleanup to prevent test pollution\n\n")

	builder.WriteString("**HTTP Handler Testing**:\n")
	builder.WriteString("- Validate BOTH status code AND response body\n")
	builder.WriteString("- Test all HTTP methods (GET, POST, PUT, DELETE)\n")
	builder.WriteString("- Test invalid UUIDs, non-existent resources, malformed JSON\n")
	builder.WriteString("- Use table-driven tests for multiple scenarios\n\n")

	builder.WriteString("**Error Testing Patterns**:\n")
	builder.WriteString("```go\n")
	builder.WriteString("patterns := NewTestScenarioBuilder().\n")
	builder.WriteString("    AddInvalidUUID(\"/api/v1/endpoint/invalid-uuid\").\n")
	builder.WriteString("    AddNonExistentCampaign(\"/api/v1/endpoint/{id}\").\n")
	builder.WriteString("    AddInvalidJSON(\"/api/v1/endpoint/{id}\").\n")
	builder.WriteString("    Build()\n")
	builder.WriteString("```\n\n")

	builder.WriteString("## Integration with Test Phases\n\n")
	builder.WriteString("**test/phases/test-unit.sh** must:\n")
	builder.WriteString("```bash\n")
	builder.WriteString("#!/bin/bash\n")
	builder.WriteString("APP_ROOT=\"${APP_ROOT:-$(cd \"${BASH_SOURCE[0]%/*}/../../../..\" && pwd)}\"\n")
	builder.WriteString("source \"${APP_ROOT}/scripts/lib/utils/var.sh\"\n")
	builder.WriteString("source \"${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh\"\n\n")
	builder.WriteString("testing::phase::init --target-time \"60s\"\n")
	builder.WriteString("source \"${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh\"\n\n")
	builder.WriteString("cd \"$TESTING_PHASE_SCENARIO_DIR\"\n\n")
	builder.WriteString("testing::unit::run_all_tests \\\n")
	builder.WriteString("    --go-dir \"api\" \\\n")
	builder.WriteString("    --node-dir \"ui\" \\\n")
	builder.WriteString("    --skip-python \\\n")
	builder.WriteString("    --coverage-warn 80 \\\n")
	builder.WriteString("    --coverage-error 50\n\n")
	builder.WriteString("testing::phase::end_with_summary \"Unit tests completed\"\n")
	builder.WriteString("```\n\n")

	builder.WriteString("## Documentation References\n\n")
	builder.WriteString("- **Gold Standard**: `/scenarios/visited-tracker/` - 79.4% Go coverage, comprehensive test suite\n")
	builder.WriteString("- **Testing Guide**: `/docs/testing/guides/scenario-unit-testing.md`\n")
	builder.WriteString("- **Helper Library**: Review visited-tracker's `test_helpers.go` and `test_patterns.go`\n")
	builder.WriteString("- **Test Patterns**: `/scenarios/visited-tracker/api/TESTING_GUIDE.md`\n\n")

	builder.WriteString("## Success Criteria\n\n")
	builder.WriteString("- [ ] Tests achieve ≥80% coverage (70% absolute minimum)\n")
	builder.WriteString("- [ ] All tests use centralized testing library integration\n")
	builder.WriteString("- [ ] Helper functions extracted for reusability\n")
	builder.WriteString("- [ ] Systematic error testing using TestScenarioBuilder\n")
	builder.WriteString("- [ ] Proper cleanup with defer statements\n")
	builder.WriteString("- [ ] Integration with phase-based test runner\n")
	builder.WriteString("- [ ] Complete HTTP handler testing (status + body validation)\n")
	builder.WriteString("- [ ] Tests complete in <60 seconds\n\n")

	if len(req.Options.CustomTestPatterns) > 0 {
		builder.WriteString("## Custom patterns to consider:\n")
		for _, pattern := range req.Options.CustomTestPatterns {
			trimmed := strings.TrimSpace(pattern)
			if trimmed != "" {
				builder.WriteString(fmt.Sprintf("- %s\n", trimmed))
			}
		}
		builder.WriteString("\n")
	}

	if req.Options.IncludePerformanceTests {
		builder.WriteString("## Performance Testing\n")
		builder.WriteString("Performance testing is requested as part of this suite.\n\n")
	}
	if req.Options.IncludeSecurityTests {
		builder.WriteString("## Security Testing\n")
		builder.WriteString("Security and hardening checks should be included in the generated suites.\n\n")
	}

	builder.WriteString("## Completion Instructions\n\n")
	builder.WriteString("When complete, attach generated artifacts and note the test locations in the issue so Test Genie can import them.")

	return builder.String()
}

func buildIssueTrackerURL(ctx context.Context, issueID string) (string, error) {
	port, err := resolveScenarioPortViaCLI(ctx, "app-issue-tracker", "UI_PORT")
	if err != nil {
		return "", err
	}

	u := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", port),
	}
	query := u.Query()
	query.Set("issue", issueID)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func sanitizeForFilename(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "request"
	}

	sanitized := make([]rune, 0, len(trimmed))
	for _, r := range trimmed {
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
		return "request"
	}
	return result
}

func resolveScenarioPortViaCLI(ctx context.Context, scenarioName, portLabel string) (int, error) {
	port, err := executeScenarioPortCommand(ctx, scenarioName, portLabel)
	if err == nil {
		return port, nil
	}

	fallbackPorts, fallbackErr := executeScenarioPortList(ctx, scenarioName)
	if fallbackErr == nil {
		candidate := strings.ToUpper(strings.TrimSpace(portLabel))
		if value, ok := fallbackPorts[candidate]; ok && value > 0 {
			return value, nil
		}
	}

	if fallbackErr != nil {
		return 0, fmt.Errorf("%v; fallback error: %w", err, fallbackErr)
	}

	return 0, err
}

func executeScenarioPortCommand(ctx context.Context, scenarioName, portLabel string) (int, error) {
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

	return parsePortValueFromString(strings.TrimSpace(string(output)))
}

func executeScenarioPortList(ctx context.Context, scenarioName string) (map[string]int, error) {
	if strings.TrimSpace(scenarioName) == "" {
		return nil, errors.New("scenario name is required")
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", "scenario", "port", scenarioName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("vrooli scenario port %s failed: %s", scenarioName, strings.TrimSpace(string(output)))
	}

	ports := make(map[string]int)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToUpper(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		port, err := parsePortValueFromString(value)
		if err != nil {
			continue
		}

		if port > 0 {
			ports[key] = port
		}
	}

	if len(ports) == 0 {
		return nil, errors.New("no port mappings returned")
	}

	return ports, nil
}

func parsePortValueFromString(value string) (int, error) {
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
