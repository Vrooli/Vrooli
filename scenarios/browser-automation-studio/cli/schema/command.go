package schema

import (
	"fmt"
	"os"
	"strings"

	"browser-automation-studio/cli/internal/api"
	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

// Commands returns the schema command group.
func Commands(ctx *appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Schema",
		Commands: []cliapp.Command{
			{
				Name:        "schema",
				NeedsAPI:    true,
				Description: "Get workflow schema for reference",
				Run: func(args []string) error {
					return runSchema(ctx, args)
				},
			},
		},
	}
}

func runSchema(ctx *appctx.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s schema <workflow|node-types|steps>", ctx.Name)
	}

	subcommand := args[0]
	switch subcommand {
	case "workflow":
		return runSchemaWorkflow(ctx, args[1:])
	case "node-types":
		return runSchemaNodeTypes(ctx, args[1:])
	case "steps":
		return runSchemaSteps(ctx, args[1:])
	default:
		return fmt.Errorf("unknown schema command: %s (use 'workflow', 'node-types', or 'steps')", subcommand)
	}
}

func runSchemaWorkflow(ctx *appctx.Context, args []string) error {
	nodes := ""
	outputPath := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--nodes":
			if i+1 >= len(args) {
				return fmt.Errorf("--nodes requires a value")
			}
			nodes = args[i+1]
			i++
		case "--output":
			if i+1 >= len(args) {
				return fmt.Errorf("--output requires a value")
			}
			outputPath = args[i+1]
			i++
		default:
			if strings.HasPrefix(args[i], "--") {
				return fmt.Errorf("unknown option: %s", args[i])
			}
			return fmt.Errorf("unexpected argument: %s", args[i])
		}
	}

	// Build query string
	path := "/schema/workflow"
	if nodes != "" {
		path = path + "?nodes=" + nodes
	}

	status, body, err := api.Do(ctx, "GET", path, nil, nil, nil)
	if err != nil {
		return err
	}

	if status != 200 {
		return fmt.Errorf("failed to get schema (status %d): %s", status, string(body))
	}

	// Output to file or stdout
	if outputPath != "" {
		if err := os.WriteFile(outputPath, body, 0o644); err != nil {
			return fmt.Errorf("write to file: %w", err)
		}
		fmt.Printf("Schema written to: %s\n", outputPath)
	} else {
		fmt.Println(string(body))
	}

	return nil
}

func runSchemaNodeTypes(ctx *appctx.Context, args []string) error {
	path := "/schema/workflow/node-types"

	status, body, err := api.Do(ctx, "GET", path, nil, nil, nil)
	if err != nil {
		return err
	}

	if status != 200 {
		return fmt.Errorf("failed to get node types (status %d): %s", status, string(body))
	}

	fmt.Println(string(body))
	return nil
}
