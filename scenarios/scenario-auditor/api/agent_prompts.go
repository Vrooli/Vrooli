package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

type RuleCreationSpec struct {
	Name        string
	Description string
	Category    string
	Severity    string
	Motivation  string
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
		if result.ExecutionOutput != nil && result.ExecutionOutput.Method != "" {
			builder.WriteString(fmt.Sprintf("  Execution method: %s (exit_code=%d)\n", result.ExecutionOutput.Method, result.ExecutionOutput.ExitCode))
		}
		if result.ExecutionOutput != nil {
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

func buildRuleCreationPrompt(spec RuleCreationSpec) (string, string, map[string]string, error) {
	spec.Name = strings.TrimSpace(spec.Name)
	spec.Description = strings.TrimSpace(spec.Description)
	spec.Category = strings.TrimSpace(spec.Category)
	spec.Severity = strings.TrimSpace(spec.Severity)
	spec.Motivation = strings.TrimSpace(spec.Motivation)

	if spec.Name == "" {
		return "", "", nil, fmt.Errorf("rule name is required")
	}
	if spec.Description == "" {
		return "", "", nil, fmt.Errorf("rule description is required")
	}
	if spec.Category == "" {
		return "", "", nil, fmt.Errorf("rule category is required")
	}
	if spec.Severity == "" {
		spec.Severity = "medium"
	}

	instructionsPath := filepath.Join(getScenarioRoot(), "prompts", "rule-creation.txt")
	instructionsBytes, err := os.ReadFile(instructionsPath)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to load rule creation instructions: %w", err)
	}

	ruleID := slugifyRuleID(spec.Name)
	if ruleID == "" {
		ruleID = slugifyRuleID(spec.Description)
	}
	if ruleID == "" {
		ruleID = fmt.Sprintf("%s_rule", spec.Category)
	}

	fileName := fmt.Sprintf("%s.go", ruleID)
	relativePath := filepath.Join("rules", spec.Category, fileName)
	absolutePath := filepath.Join(getScenarioRoot(), relativePath)

	if _, err := os.Stat(absolutePath); err == nil {
		return "", "", nil, fmt.Errorf("a rule already exists at %s", relativePath)
	}

	structName := pascalCaseFromID(ruleID) + "Rule"

	var builder strings.Builder
	builder.WriteString(string(instructionsBytes))
	builder.WriteString("\n\n## Request Summary\n")
	builder.WriteString(fmt.Sprintf("- Rule name: %s\n", spec.Name))
	builder.WriteString(fmt.Sprintf("- Proposed rule ID: %s\n", ruleID))
	builder.WriteString(fmt.Sprintf("- Category: %s\n", spec.Category))
	builder.WriteString(fmt.Sprintf("- Severity: %s\n", spec.Severity))
	builder.WriteString(fmt.Sprintf("- Destination file: %s\n", relativePath))
	builder.WriteString(fmt.Sprintf("- Expected struct name: %s\n", structName))

	builder.WriteString("\n## Rule Behaviour Goals\n")
	builder.WriteString(spec.Description)

	if spec.Motivation != "" {
		builder.WriteString("\n\n## Additional Context\n")
		builder.WriteString(spec.Motivation)
	}

	builder.WriteString("\n\n## Implementation Requirements\n")
	builder.WriteString("- Create the Go file at the destination path above.\n")
	builder.WriteString("- Define `type " + structName + " struct{}` and implement `func (r *" + structName + ") Check(content string, filepath string, scenario string) ([]Violation, error)` since the runtime always calls this signature. The `scenario` parameter will be empty for playground runs.\n")
	builder.WriteString("- Register the rule ID in `api/rule_registry.go` so it is discoverable.\n")
	builder.WriteString("- Include a metadata comment block and representative `<test-case>` sections like existing rules.\n")
	builder.WriteString("- Behaviour must rely only on the provided `content` (raw file contents) and `filepath` hints. Do not expect additional context.\n")
	builder.WriteString("- Ensure imported helpers exist or add new ones within the same file if required.\n")
	builder.WriteString("- Finish by running `scenario-auditor test " + ruleID + "` and summarise the result in your final response.\n")

	metadata := map[string]string{
		"rule_id":        ruleID,
		"rule_path":      relativePath,
		"category":       spec.Category,
		"severity":       spec.Severity,
		"requested_name": spec.Name,
	}

	return builder.String(), fmt.Sprintf("Create %s", spec.Name), metadata, nil
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func slugifyRuleID(name string) string {
	trimmed := strings.TrimSpace(strings.ToLower(name))
	if trimmed == "" {
		return ""
	}
	slug := nonAlphaNum.ReplaceAllString(trimmed, "_")
	slug = strings.Trim(slug, "_")
	return slug
}

func pascalCaseFromID(id string) string {
	if id == "" {
		return "Rule"
	}
	parts := strings.Split(id, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		parts[i] = string(runes)
	}
	return strings.Join(parts, "")
}
