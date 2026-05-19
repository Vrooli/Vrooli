// Package schema is the CLI surface for the BAS SchemaService Connect-RPC.
//
// Each subcommand dispatches through cli-core's Connect HTTP client. Output
// preserves the historical shapes (raw JSON schema dump, JSON node-types
// envelope, text/markdown/json step rendering) so agents wrapping this CLI
// keep working.
package schema

import (
	"browser-automation-studio/cli/internal/appctx"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"

	schemav1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/schema"
	schemaconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/schema/schemaconnect"

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

func newClient(ctx *appctx.Context) schemaconnect.SchemaServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(ctx.Core)
	return schemaconnect.NewSchemaServiceClient(httpClient, baseURL)
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

	req := &schemav1.GetWorkflowSchemaRequest{}
	if nodes != "" {
		for _, tok := range strings.Split(nodes, ",") {
			if t := strings.TrimSpace(tok); t != "" {
				req.NodeTypes = append(req.NodeTypes, t)
			}
		}
	}

	resp, err := newClient(ctx).GetWorkflowSchema(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("schema workflow", err, nil)
	}
	body, err := json.MarshalIndent(resp.Msg.GetSchema().AsMap(), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}

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
	if len(args) > 0 {
		return fmt.Errorf("unexpected argument: %s", args[0])
	}
	resp, err := newClient(ctx).GetNodeTypes(context.Background(),
		connect.NewRequest(&schemav1.GetNodeTypesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("schema node-types", err, nil)
	}
	out, err := json.MarshalIndent(map[string]any{"node_types": resp.Msg.GetNodeTypes()}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal node types: %w", err)
	}
	fmt.Println(string(out))
	return nil
}
