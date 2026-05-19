package schema

import (
	"browser-automation-studio/cli/internal/appctx"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	schemav1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/schema"

	"github.com/vrooli/cli-core/cliapp"
)

// StepDefinition mirrors the proto step definition for stable CLI output.
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

	req := &schemav1.GetStepDefinitionsRequest{CliOnly: cliOnly}
	if types != "" {
		for _, tok := range strings.Split(types, ",") {
			if t := strings.TrimSpace(tok); t != "" {
				req.Types = append(req.Types, t)
			}
		}
	}

	resp, err := newClient(ctx).GetStepDefinitions(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("schema steps", err, nil)
	}

	steps := make([]StepDefinition, 0, len(resp.Msg.GetSteps()))
	for _, ps := range resp.Msg.GetSteps() {
		steps = append(steps, fromProto(ps))
	}

	switch format {
	case "json":
		out, err := json.MarshalIndent(steps, "", "  ")
		if err != nil {
			return fmt.Errorf("format json: %w", err)
		}
		fmt.Println(string(out))
	case "markdown":
		fmt.Println(formatStepsMarkdown(steps))
	default:
		fmt.Println(formatStepsText(steps))
	}
	return nil
}

func fromProto(p *schemav1.StepDefinition) StepDefinition {
	out := StepDefinition{
		Type:         p.GetType(),
		Description:  p.GetDescription(),
		CLISupported: p.GetCliSupported(),
	}
	if p.Positional != nil {
		out.Positional = &PositionalDef{
			Name:        p.Positional.GetName(),
			MapsTo:      p.Positional.GetMapsTo(),
			Description: p.Positional.GetDescription(),
		}
	}
	for _, kv := range p.GetRequiredKvs() {
		out.RequiredKVs = append(out.RequiredKVs, KVDef{Key: kv.GetKey(), Type: kv.GetType(), Description: kv.GetDescription()})
	}
	for _, kv := range p.GetOptionalKvs() {
		out.OptionalKVs = append(out.OptionalKVs, KVDef{Key: kv.GetKey(), Type: kv.GetType(), Description: kv.GetDescription()})
	}
	for _, g := range p.GetRequireOneOf() {
		out.RequireOneOf = append(out.RequireOneOf, g.GetKeys())
	}
	for _, ex := range p.GetExamples() {
		out.Examples = append(out.Examples, StepExample{Description: ex.GetDescription(), CLI: ex.GetCli()})
	}
	return out
}

// formatStepsText formats step definitions as readable text.
func formatStepsText(steps []StepDefinition) string {
	var sb strings.Builder
	for i, step := range steps {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(step.Type)
		sb.WriteString("\n  ")
		sb.WriteString(step.Description)
		sb.WriteString("\n")
		if step.Positional != nil {
			sb.WriteString("\n  Positional: <")
			sb.WriteString(step.Positional.Name)
			sb.WriteString(">\n    ")
			sb.WriteString(step.Positional.Description)
			sb.WriteString("\n")
		}
		if len(step.RequiredKVs) > 0 {
			sb.WriteString("\n  Required:\n")
			for _, kv := range step.RequiredKVs {
				sb.WriteString(fmt.Sprintf("    %s (%s): %s\n", kv.Key, kv.Type, kv.Description))
			}
		}
		if len(step.OptionalKVs) > 0 {
			sb.WriteString("\n  Optional:\n")
			for _, kv := range step.OptionalKVs {
				sb.WriteString(fmt.Sprintf("    %s (%s): %s\n", kv.Key, kv.Type, kv.Description))
			}
		}
		if len(step.RequireOneOf) > 0 {
			sb.WriteString("\n  Requires one of:\n")
			for _, group := range step.RequireOneOf {
				sb.WriteString("    - ")
				sb.WriteString(strings.Join(group, " OR "))
				sb.WriteString("\n")
			}
		}
		if len(step.Examples) > 0 {
			sb.WriteString("\n  Examples:\n")
			for _, ex := range step.Examples {
				sb.WriteString("    # ")
				sb.WriteString(ex.Description)
				sb.WriteString("\n    ")
				sb.WriteString(ex.CLI)
				sb.WriteString("\n")
			}
		}
		if !step.CLISupported {
			sb.WriteString("\n  Note: This step type requires workflow JSON (not supported via --step flag)\n")
		}
	}
	return sb.String()
}

// formatStepsMarkdown formats step definitions as markdown.
func formatStepsMarkdown(steps []StepDefinition) string {
	var sb strings.Builder
	sb.WriteString("# Step Definitions\n\n")
	for _, step := range steps {
		sb.WriteString("## ")
		sb.WriteString(step.Type)
		if !step.CLISupported {
			sb.WriteString(" (JSON only)")
		}
		sb.WriteString("\n\n")
		sb.WriteString(step.Description)
		sb.WriteString("\n\n")
		if step.Positional != nil {
			sb.WriteString("### Positional Argument\n\n")
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n\n", step.Positional.Name, step.Positional.Description))
		}
		if len(step.RequiredKVs) > 0 {
			sb.WriteString("### Required Parameters\n\n| Key | Type | Description |\n|-----|------|-------------|\n")
			for _, kv := range step.RequiredKVs {
				sb.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", kv.Key, kv.Type, kv.Description))
			}
			sb.WriteString("\n")
		}
		if len(step.OptionalKVs) > 0 {
			sb.WriteString("### Optional Parameters\n\n| Key | Type | Description |\n|-----|------|-------------|\n")
			for _, kv := range step.OptionalKVs {
				sb.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", kv.Key, kv.Type, kv.Description))
			}
			sb.WriteString("\n")
		}
		if len(step.RequireOneOf) > 0 {
			sb.WriteString("### Constraints\n\nRequires one of:\n")
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
		if len(step.Examples) > 0 {
			sb.WriteString("### Examples\n\n")
			for _, ex := range step.Examples {
				sb.WriteString(ex.Description)
				sb.WriteString(":\n```bash\n")
				sb.WriteString(ex.CLI)
				sb.WriteString("\n```\n\n")
			}
		}
		sb.WriteString("---\n\n")
	}
	return sb.String()
}
