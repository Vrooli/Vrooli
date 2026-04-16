package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	ctx, err := repoContext()
	if err != nil {
		return absPath
	}
	root := ctx.ScenarioAuditorRoot()
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

	ctx, err := repoContext()
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to resolve repo context: %w", err)
	}

	instructionsPath := filepath.Join(ctx.ScenarioAuditorRoot(), "prompts", "rule-creation.txt")
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
	absolutePath := filepath.Join(ctx.ScenarioAuditorRoot(), relativePath)

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

type RuleEditingSpec struct {
	RuleID      string
	Changes     string
	Motivation  string
	FilePath    string
	RuleContent string
}

func buildRuleEditingPrompt(spec RuleEditingSpec) (string, string, map[string]string, error) {
	spec.RuleID = strings.TrimSpace(spec.RuleID)
	spec.Changes = strings.TrimSpace(spec.Changes)
	spec.Motivation = strings.TrimSpace(spec.Motivation)

	if spec.RuleID == "" {
		return "", "", nil, fmt.Errorf("rule ID is required")
	}
	if spec.Changes == "" {
		return "", "", nil, fmt.Errorf("changes description is required")
	}
	if spec.FilePath == "" {
		return "", "", nil, fmt.Errorf("rule file path is required")
	}
	if spec.RuleContent == "" {
		return "", "", nil, fmt.Errorf("rule content is required")
	}

	ctx, err := repoContext()
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to resolve repo context: %w", err)
	}

	instructionsPath := filepath.Join(ctx.ScenarioAuditorRoot(), "prompts", "rule-editing.txt")
	instructionsBytes, err := os.ReadFile(instructionsPath)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to load rule editing instructions: %w", err)
	}

	relativePath := relativeRulePath(spec.FilePath)

	var builder strings.Builder
	builder.WriteString(string(instructionsBytes))
	builder.WriteString("\n\n## Edit Request Summary\n")
	builder.WriteString(fmt.Sprintf("- Rule ID: %s\n", spec.RuleID))
	builder.WriteString(fmt.Sprintf("- File path: %s\n", relativePath))
	builder.WriteString("\n## Current Rule Implementation\n")
	builder.WriteString("```go\n")
	builder.WriteString(spec.RuleContent)
	builder.WriteString("\n```\n")

	builder.WriteString("\n## Requested Changes\n")
	builder.WriteString(spec.Changes)

	if spec.Motivation != "" {
		builder.WriteString("\n\n## Additional Context\n")
		builder.WriteString(spec.Motivation)
	}

	builder.WriteString("\n\n## Implementation Requirements\n")
	builder.WriteString("- Modify the existing Go file at the path above.\n")
	builder.WriteString("- Preserve the existing `Check` function signature: `func (r *RuleStruct) Check(content string, filepath string, scenario string) ([]Violation, error)`\n")
	builder.WriteString("- Update any `<test-case>` sections if the rule behavior changes.\n")
	builder.WriteString("- Maintain the metadata comment block at the top of the file.\n")
	builder.WriteString("- Only change what is necessary to implement the requested modifications.\n")
	builder.WriteString("- Finish by running `scenario-auditor test " + spec.RuleID + "` and summarise the result in your final response.\n")

	metadata := map[string]string{
		"rule_id":   spec.RuleID,
		"rule_path": relativePath,
	}

	return builder.String(), fmt.Sprintf("Edit %s", spec.RuleID), metadata, nil
}
