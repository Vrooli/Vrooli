package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
