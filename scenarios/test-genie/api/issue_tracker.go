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
		"requested_by": "test-genie",
		"request_id":   requestID.String(),
		"scenario":     scenario,
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

	payload := map[string]interface{}{
		"title":          fmt.Sprintf("Generate automated tests for %s", scenario),
		"description":    description,
		"type":           "task",
		"priority":       "high",
		"app_id":         scenario,
		"status":         "open",
		"tags":           []string{"test-genie", "test-generation", "automation"},
		"metadata_extra": metadata,
		"environment":    environment,
		"artifacts":      artifacts,
		"reporter_name":  "Test Genie",
		"reporter_email": "test-genie@vrooli.local",
	}

	if req.Model != "" {
		metadata["preferred_model"] = strings.TrimSpace(req.Model)
	}

	return payload, nil
}

func buildIssueTrackerDescription(scenario string, testTypes []string, req GenerateTestSuiteRequest) string {
	var builder strings.Builder
	builder.WriteString("Test Genie requests automated test generation for the following scenario.\n\n")
	builder.WriteString(fmt.Sprintf("Scenario: %s\n", scenario))
	builder.WriteString(fmt.Sprintf("Requested test types: %s\n", strings.Join(testTypes, ", ")))
	if req.CoverageTarget > 0 {
		builder.WriteString(fmt.Sprintf("Target coverage: %.2f%%\n", req.CoverageTarget))
	}
	builder.WriteString("\nExpectations:\n")
	builder.WriteString("- Analyze the scenario codebase and existing tests to design comprehensive coverage.\n")
	builder.WriteString("- Generate runnable test suites with clear structure and deterministic assertions.\n")
	builder.WriteString("- Provide instructions for integrating the new tests back into Test Genie once ready.\n")
	builder.WriteString("- Include any additional validation steps (performance, security) requested in the options.\n")

	if len(req.Options.CustomTestPatterns) > 0 {
		builder.WriteString("\nCustom patterns to consider:\n")
		for _, pattern := range req.Options.CustomTestPatterns {
			trimmed := strings.TrimSpace(pattern)
			if trimmed != "" {
				builder.WriteString(fmt.Sprintf("- %s\n", trimmed))
			}
		}
	}

	if req.Options.IncludePerformanceTests {
		builder.WriteString("\nPerformance testing is requested as part of this suite.\n")
	}
	if req.Options.IncludeSecurityTests {
		builder.WriteString("Security and hardening checks should be included in the generated suites.\n")
	}

	builder.WriteString("\nWhen complete, attach generated artifacts and note the test locations in the issue so Test Genie can import them.")

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
