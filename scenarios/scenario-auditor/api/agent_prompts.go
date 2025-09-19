package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type ruleAgentPromptData struct {
	RuleID               string
	RuleName             string
	Category             string
	Severity             string
	Standard             string
	RuleFile             string
	RuleImplementation   string
	ExistingTestsSummary string
	ExistingTestsDetail  string
	FailingTestsDetail   string
}

func buildRuleAgentPrompt(rule RuleInfo, action string) (string, string, map[string]string, error) {
	ruleSource, err := os.ReadFile(rule.FilePath)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to read rule file: %w", err)
	}

	testRunner := NewTestRunner()
	testCases, err := testRunner.ExtractTestCases(string(ruleSource))
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to parse test cases: %w", err)
	}

	existingSummary := describeExistingTests(rule.ID, testCases)
	existingDetail := describeTestDetails(testCases)

	var failingDetail string
	var testRunErr error
	var label string

	switch action {
	case agentActionAddRuleTests:
		label = "Add Rule Test Cases"
	case agentActionFixRuleTests:
		label = "Fix Rule Test Cases"
	default:
		return "", "", nil, fmt.Errorf("unsupported agent action: %s", action)
	}

	var templateName string
	switch action {
	case agentActionAddRuleTests:
		templateName = "rule-add-test-cases.tmpl"
	case agentActionFixRuleTests:
		templateName = "rule-fix-test-cases.tmpl"
	}

	templatePath := filepath.Join(getScenarioRoot(), "prompts", templateName)
	tplBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to load prompt template: %w", err)
	}

	// Always attempt to run tests so we can provide failing context (even if there are currently zero cases)
	results, runErr := testRunner.RunAllTests(rule.ID, rule)
	if runErr != nil {
		testRunErr = runErr
	}
	if len(results) > 0 {
		failingDetail = describeFailingTests(results)
	}
	if testRunErr != nil {
		failingDetail += fmt.Sprintf("\nNOTE: Test execution error encountered: %v", testRunErr)
	}

	data := ruleAgentPromptData{
		RuleID:               rule.ID,
		RuleName:             safeFallback(rule.Name, rule.ID),
		Category:             safeFallback(rule.Category, "uncategorized"),
		Severity:             safeFallback(rule.Severity, "unknown"),
		Standard:             safeFallback(rule.Standard, "unspecified"),
		RuleFile:             relativeRulePath(rule.FilePath),
		RuleImplementation:   string(ruleSource),
		ExistingTestsSummary: existingSummary,
		ExistingTestsDetail:  existingDetail,
		FailingTestsDetail:   failingDetail,
	}

	tpl, err := template.New(templateName).Parse(string(tplBytes))
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to parse prompt template: %w", err)
	}

	var buffer bytes.Buffer
	if err := tpl.Execute(&buffer, data); err != nil {
		return "", "", nil, fmt.Errorf("failed to render prompt template: %w", err)
	}

	metadata := map[string]string{
		"rule_id":   rule.ID,
		"rule_file": data.RuleFile,
		"action":    action,
	}

	if testRunErr != nil {
		metadata["test_run_error"] = testRunErr.Error()
	}

	return buffer.String(), label, metadata, nil
}

func describeExistingTests(ruleID string, tests []TestCase) string {
	if len(tests) == 0 {
		return fmt.Sprintf("Rule %s currently has no embedded test cases.", ruleID)
	}
	summary := fmt.Sprintf("Rule %s currently has %d test case(s).", ruleID, len(tests))
	return summary
}

func describeTestDetails(tests []TestCase) string {
	if len(tests) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, tc := range tests {
		builder.WriteString("- ")
		builder.WriteString(tc.ID)
		if tc.Description != "" {
			builder.WriteString(": ")
			builder.WriteString(tc.Description)
		}
		builder.WriteString(fmt.Sprintf(" (language=%s, should_fail=%t, expected_violations=%d)", safeFallback(tc.Language, "text"), tc.ShouldFail, tc.ExpectedViolations))
		if tc.ExpectedMessage != "" {
			builder.WriteString(" — expects message containing \"")
			builder.WriteString(tc.ExpectedMessage)
			builder.WriteString("\"")
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

func describeFailingTests(results []TestResult) string {
	var builder strings.Builder
	for _, result := range results {
		if result.Passed {
			continue
		}
		builder.WriteString(fmt.Sprintf("- %s failed. ShouldFail=%t, expected violations=%d, actual=%d\n",
			result.TestCase.ID,
			result.TestCase.ShouldFail,
			result.TestCase.ExpectedViolations,
			len(result.ActualViolations),
		))
		if result.TestCase.ExpectedMessage != "" {
			builder.WriteString(fmt.Sprintf("  Expected message containing: %s\n", result.TestCase.ExpectedMessage))
		}
		if len(result.ActualViolations) > 0 {
			builder.WriteString("  Actual violations:\n")
			for _, violation := range result.ActualViolations {
				builder.WriteString(fmt.Sprintf("    - [%s] %s\n", safeFallback(violation.Severity, "unknown"), violation.Message))
			}
		}
		if result.ExecutionOutput.Method != "" {
			builder.WriteString(fmt.Sprintf("  Execution method: %s (exit_code=%d)\n", result.ExecutionOutput.Method, result.ExecutionOutput.ExitCode))
		}
		if trimmed := trimForPrompt(result.ExecutionOutput.Stdout); trimmed != "" {
			builder.WriteString("  Stdout snippet:\n")
			builder.WriteString(indentForPrompt(trimmed))
			builder.WriteString("\n")
		}
		if trimmedErr := trimForPrompt(result.ExecutionOutput.Stderr); trimmedErr != "" {
			builder.WriteString("  Stderr snippet:\n")
			builder.WriteString(indentForPrompt(trimmedErr))
			builder.WriteString("\n")
		}
		if result.Error != "" {
			builder.WriteString(fmt.Sprintf("  Error: %s\n", result.Error))
		}
	}

	return builder.String()
}

func trimForPrompt(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	const maxLen = 600
	if len([]rune(trimmed)) <= maxLen {
		return trimmed
	}
	runes := []rune(trimmed)
	return string(runes[:maxLen]) + "…"
}

func indentForPrompt(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = "    " + line
	}
	return strings.Join(lines, "\n")
}

func safeFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func relativeRulePath(absPath string) string {
	root := getScenarioRoot()
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		return absPath
	}
	return rel
}
