package schema

import (
	"browser-automation-studio/cli/internal/api"
	"browser-automation-studio/cli/internal/appctx"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// StepDefinition mirrors the API's step definition structure.
type StepDefinition struct {
	Type         string         `json:"type"`
	Description  string         `json:"description"`
	Positional   *PositionalDef `json:"positional,omitempty"`
	RequiredKVs  []KVDef        `json:"requiredKVs,omitempty"`
	OptionalKVs  []KVDef        `json:"optionalKVs,omitempty"`
	RequireOneOf [][]string     `json:"requireOneOf,omitempty"`
	Examples     []StepExample  `json:"examples,omitempty"`
	CLISupported bool           `json:"cliSupported"`
}

// PositionalDef defines a positional argument.
type PositionalDef struct {
	Name        string `json:"name"`
	MapsTo      string `json:"mapsTo"`
	Description string `json:"description"`
}

// KVDef defines a key-value argument.
type KVDef struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// StepExample provides an example CLI usage.
type StepExample struct {
	Description string `json:"description"`
	CLI         string `json:"cli"`
}

// StepsResponse is the API response structure.
type StepsResponse struct {
	Steps []StepDefinition `json:"steps"`
}

func runSchemaSteps(ctx *appctx.Context, args []string) error {
	types := ""
	format := "text"
	cliOnly := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--types":
			if i+1 >= len(args) {
				return fmt.Errorf("--types requires a value")
			}
			types = args[i+1]
			i++
		case "--format":
			if i+1 >= len(args) {
				return fmt.Errorf("--format requires a value")
			}
			format = args[i+1]
			if format != "text" && format != "json" && format != "markdown" {
				return fmt.Errorf("--format must be text, json, or markdown")
			}
			i++
		case "--cli-only":
			cliOnly = true
		default:
			if strings.HasPrefix(args[i], "--") {
				return fmt.Errorf("unknown option: %s", args[i])
			}
			return fmt.Errorf("unexpected argument: %s", args[i])
		}
	}

	// Build query parameters
	query := url.Values{}
	if types != "" {
		query.Set("types", types)
	}
	if cliOnly {
		query.Set("cli_only", "true")
	}

	path := "/schema/steps"
	status, body, err := api.Do(ctx, "GET", path, query, nil, nil)
	if err != nil {
		return err
	}

	if status != 200 {
		return fmt.Errorf("failed to get step definitions (status %d): %s", status, string(body))
	}

	var resp StepsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	// Output in requested format
	switch format {
	case "json":
		// Re-marshal with indentation
		output, err := json.MarshalIndent(resp.Steps, "", "  ")
		if err != nil {
			return fmt.Errorf("format json: %w", err)
		}
		fmt.Println(string(output))
	case "markdown":
		fmt.Println(formatStepsMarkdown(resp.Steps))
	default:
		fmt.Println(formatStepsText(resp.Steps))
	}

	return nil
}

// formatStepsText formats step definitions as readable text.
func formatStepsText(steps []StepDefinition) string {
	var sb strings.Builder

	for i, step := range steps {
		if i > 0 {
			sb.WriteString("\n")
		}

		// Type and description
		sb.WriteString(step.Type)
		sb.WriteString("\n")
		sb.WriteString("  ")
		sb.WriteString(step.Description)
		sb.WriteString("\n")

		// Positional argument
		if step.Positional != nil {
			sb.WriteString("\n")
			sb.WriteString("  Positional: <")
			sb.WriteString(step.Positional.Name)
			sb.WriteString(">\n")
			sb.WriteString("    ")
			sb.WriteString(step.Positional.Description)
			sb.WriteString("\n")
		}

		// Required key-value pairs
		if len(step.RequiredKVs) > 0 {
			sb.WriteString("\n")
			sb.WriteString("  Required:\n")
			for _, kv := range step.RequiredKVs {
				sb.WriteString(fmt.Sprintf("    %s (%s): %s\n", kv.Key, kv.Type, kv.Description))
			}
		}

		// Optional key-value pairs
		if len(step.OptionalKVs) > 0 {
			sb.WriteString("\n")
			sb.WriteString("  Optional:\n")
			for _, kv := range step.OptionalKVs {
				sb.WriteString(fmt.Sprintf("    %s (%s): %s\n", kv.Key, kv.Type, kv.Description))
			}
		}

		// Require one of constraints
		if len(step.RequireOneOf) > 0 {
			sb.WriteString("\n")
			sb.WriteString("  Requires one of:\n")
			for _, group := range step.RequireOneOf {
				sb.WriteString("    - ")
				sb.WriteString(strings.Join(group, " OR "))
				sb.WriteString("\n")
			}
		}

		// Examples
		if len(step.Examples) > 0 {
			sb.WriteString("\n")
			sb.WriteString("  Examples:\n")
			for _, ex := range step.Examples {
				sb.WriteString("    # ")
				sb.WriteString(ex.Description)
				sb.WriteString("\n")
				sb.WriteString("    ")
				sb.WriteString(ex.CLI)
				sb.WriteString("\n")
			}
		}

		// CLI support note
		if !step.CLISupported {
			sb.WriteString("\n")
			sb.WriteString("  Note: This step type requires workflow JSON (not supported via --step flag)\n")
		}
	}

	return sb.String()
}

// formatStepsMarkdown formats step definitions as markdown.
func formatStepsMarkdown(steps []StepDefinition) string {
	var sb strings.Builder

	sb.WriteString("# Step Definitions\n\n")

	for _, step := range steps {
		// Type heading
		sb.WriteString("## ")
		sb.WriteString(step.Type)
		if !step.CLISupported {
			sb.WriteString(" (JSON only)")
		}
		sb.WriteString("\n\n")

		// Description
		sb.WriteString(step.Description)
		sb.WriteString("\n\n")

		// Positional argument
		if step.Positional != nil {
			sb.WriteString("### Positional Argument\n\n")
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n\n", step.Positional.Name, step.Positional.Description))
		}

		// Required key-value pairs
		if len(step.RequiredKVs) > 0 {
			sb.WriteString("### Required Parameters\n\n")
			sb.WriteString("| Key | Type | Description |\n")
			sb.WriteString("|-----|------|-------------|\n")
			for _, kv := range step.RequiredKVs {
				sb.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", kv.Key, kv.Type, kv.Description))
			}
			sb.WriteString("\n")
		}

		// Optional key-value pairs
		if len(step.OptionalKVs) > 0 {
			sb.WriteString("### Optional Parameters\n\n")
			sb.WriteString("| Key | Type | Description |\n")
			sb.WriteString("|-----|------|-------------|\n")
			for _, kv := range step.OptionalKVs {
				sb.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", kv.Key, kv.Type, kv.Description))
			}
			sb.WriteString("\n")
		}

		// Require one of constraints
		if len(step.RequireOneOf) > 0 {
			sb.WriteString("### Constraints\n\n")
			sb.WriteString("Requires one of:\n")
			for _, group := range step.RequireOneOf {
				sb.WriteString("- ")
				for i, key := range group {
					if i > 0 {
						sb.WriteString(" OR ")
					}
					sb.WriteString("`")
					sb.WriteString(key)
					sb.WriteString("`")
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}

		// Examples
		if len(step.Examples) > 0 {
			sb.WriteString("### Examples\n\n")
			for _, ex := range step.Examples {
				sb.WriteString(ex.Description)
				sb.WriteString(":\n")
				sb.WriteString("```bash\n")
				sb.WriteString(ex.CLI)
				sb.WriteString("\n```\n\n")
			}
		}

		sb.WriteString("---\n\n")
	}

	return sb.String()
}
