package capabilityhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	climanifest "github.com/vrooli/vrooli/cli"
	capabilityapp "github.com/vrooli/vrooli/internal/app/capability"
	"github.com/vrooli/vrooli/internal/cli/manifestdispatch"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/operatorcapability"
)

type capabilityService struct {
	run func([]string) error
}

const (
	capabilityLedgerCommand = "ledger"
	capabilityFleetCommand  = "fleet"
	capabilityJSONFlag      = "--json"
	capabilityHelpFlag      = "--help"
)

var capabilityCommandNames = []string{capabilityLedgerCommand, capabilityFleetCommand}

const capabilityHelpText = "Usage: vrooli capability ledger|fleet [query] [--json]\n  vrooli capability conformance [--json|--declarations-only]\n  vrooli capability auth-conformance [--json]\n  vrooli capability catalog|status [--json]\n  vrooli capability preview|apply [--json] < action JSON\n"

// RegisteredCommandPaths returns the child paths bound by the capability handler.
func RegisteredCommandPaths() []string {
	paths := make([]string, 0, len(capabilityCommandNames))
	for _, name := range capabilityCommandNames {
		paths = append(paths, "capability "+name)
	}
	return paths
}

// RootHandler dispatches `vrooli capability` through the capability app.
func RootHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return rootcli.BindService(deps.Stdout,
		func(C) (cliout.Format, error) { return cliout.FormatHuman, nil },
		func(ctx C, _ cliout.Format) (capabilityService, error) {
			commandCtx := &rootcli.CommandContext{Root: deps.Root(ctx), Globals: deps.Globals(ctx), Stdin: deps.Stdin(ctx), Stdout: deps.Stdout(ctx), Stderr: deps.Stderr(ctx), Context: rootcli.ResolveOperationContext(deps, ctx)}
			app := &capabilityapp.Service{}
			return capabilityService{run: func(args []string) error {
				if len(args) == 0 || manifestdispatch.WantsHelp(args) {
					return writeCapabilityHelp(deps.Stdout(ctx))
				}
				if args[0] != capabilityLedgerCommand && args[0] != capabilityFleetCommand {
					return runLegacyCapability(deps.OperationContext(ctx), app, commandCtx, args)
				}
				group, err := cliapp.LoadFromManifest(climanifest.Bytes(), "capability", capabilityBindings(deps.OperationContext(ctx), app, commandCtx, capabilityCommandNames))
				if err != nil {
					return err
				}
				core := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli capability", Commands: []cliapp.CommandGroup{{Commands: group.Subcommands}}})
				return core.RunWithWriters(manifestdispatch.WithJSON(args, commandCtx.Globals.JSON), deps.Stdout(ctx), deps.Stderr(ctx))
			}}, nil
		},
		func(_ C, args []string) ([]string, error) { return args, nil },
		func(service capabilityService, args []string) (struct{}, error) { return struct{}{}, service.run(args) },
		func(io.Writer, cliout.Format, struct{}) error { return nil },
	)
}

func capabilityBindings(operationCtx context.Context, app *capabilityapp.Service, ctx *rootcli.CommandContext, names []string) map[string]func(cliapp.RunContext) error {
	bindings := make(map[string]func(cliapp.RunContext) error, len(names))
	for _, name := range names {
		command := name
		bindings[name] = func(command string) func(cliapp.RunContext) error {
			return func(runCtx cliapp.RunContext) error {
				commandCtx := *ctx
				commandCtx.Globals.JSON = commandCtx.Globals.JSON || runCtx.JSON()
				query := ""
				if command == capabilityFleetCommand {
					query = strings.TrimSpace(runCtx.Positional("query"))
					if !validFleetQuery(query) {
						return fmt.Errorf("unknown capability fleet query %q", query)
					}
				}
				return app.Ledger(operationCtx, commandCtx.Stdout, capabilityapp.LedgerOptions{Fleet: command == capabilityFleetCommand, Query: query, JSON: commandCtx.Globals.JSON})
			}
		}(command)
	}
	return bindings
}

func writeCapabilityHelp(out io.Writer) error {
	_, err := io.WriteString(out, capabilityHelpText)
	return err
}

func runLegacyCapability(operationCtx context.Context, app *capabilityapp.Service, ctx *rootcli.CommandContext, args []string) error {
	if len(args) == 0 || manifestdispatch.WantsHelp(args) {
		return writeCapabilityHelp(ctx.Stdout)
	}
	switch args[0] {
	case capabilityLedgerCommand, capabilityFleetCommand:
		query := ""
		jsonOutput := ctx.Globals.JSON
		for _, arg := range args[1:] {
			switch strings.TrimSpace(arg) {
			case capabilityJSONFlag:
				jsonOutput = true
			case capabilityHelpFlag, "-h":
				return writeCapabilityHelp(ctx.Stdout)
			default:
				if args[0] != capabilityFleetCommand || !validFleetQuery(strings.TrimSpace(arg)) {
					return fmt.Errorf("unknown capability ledger option %q", arg)
				}
				query = strings.TrimSpace(arg)
			}
		}
		return app.Ledger(operationCtx, ctx.Stdout, capabilityapp.LedgerOptions{Fleet: args[0] == capabilityFleetCommand, Query: query, JSON: jsonOutput})
	case "conformance":
		jsonOutput, declarationsOnly, err := parseConformanceArgs(args[1:])
		if err != nil {
			return err
		}
		return app.Conformance(operationCtx, ctx.Root, ctx.Stdout, capabilityapp.ConformanceOptions{DeclarationsOnly: declarationsOnly, JSON: ctx.Globals.JSON || jsonOutput})
	case "auth-conformance":
		jsonOutput, _, err := parseConformanceArgs(args[1:])
		if err != nil {
			return err
		}
		return app.AuthConformance(operationCtx, ctx.Root, ctx.Stdout, capabilityapp.ConformanceOptions{JSON: ctx.Globals.JSON || jsonOutput})
	case "catalog", "status", "preview", "apply":
		return runLegacyWorkflow(operationCtx, app, ctx, args[0], args[1:])
	default:
		return fmt.Errorf("unknown capability command %q", args[0])
	}
}

func parseConformanceArgs(args []string) (bool, bool, error) {
	jsonOutput := false
	declarationsOnly := false
	for _, arg := range args {
		switch strings.TrimSpace(arg) {
		case capabilityJSONFlag:
			jsonOutput = true
		case "--declarations-only":
			declarationsOnly = true
		case capabilityHelpFlag, "-h":
			return false, false, nil
		default:
			return false, false, fmt.Errorf("unknown capability conformance option %q", arg)
		}
	}
	return jsonOutput, declarationsOnly, nil
}

func runLegacyWorkflow(operationCtx context.Context, app *capabilityapp.Service, ctx *rootcli.CommandContext, action string, args []string) error {
	jsonOutput := ctx.Globals.JSON
	for _, arg := range args {
		switch strings.TrimSpace(arg) {
		case capabilityJSONFlag:
			jsonOutput = true
		case capabilityHelpFlag, "-h":
			return writeCapabilityHelp(ctx.Stdout)
		default:
			return fmt.Errorf("unknown capability workflow option %q", arg)
		}
	}
	var request operatorcapability.ActionRequest
	if action == "preview" || action == "apply" {
		input := ctx.Stdin
		if input == nil {
			input = os.Stdin
		}
		if err := json.NewDecoder(input).Decode(&request); err != nil {
			return fmt.Errorf("capability action JSON is required on standard input: %w", err)
		}
	}
	return app.Workflow(operationCtx, ctx.Root, ctx.Stdout, capabilityapp.WorkflowOptions{Action: action, JSON: jsonOutput, Request: request})
}

func validFleetQuery(query string) bool {
	if query == "" {
		return true
	}
	switch query {
	case "blocked", "docker", "peerless", "upgrades", "desktop":
		return true
	default:
		return false
	}
}
