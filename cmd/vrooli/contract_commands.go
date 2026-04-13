package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/contractcli"
)

type contractCommandHandler func(ctx *commandContext, args []string) error

type contractCommandDescriptor struct {
	Name    string
	Handler contractCommandHandler
}

var contractCommandTable = []contractCommandDescriptor{
	{Name: "validate", Handler: runContractValidateCommand},
	{Name: "show", Handler: runContractShowCommand},
	{Name: "resolve", Handler: runContractResolveCommand},
	{Name: "match-glob", Handler: runContractMatchGlobCommand},
}

var contractResolveCommandTable = []contractCommandDescriptor{
	{Name: "scenario", Handler: runContractResolveScenarioCommand},
}

var (
	contractCommandHandlers        = buildContractCommandMap(contractCommandTable)
	contractResolveCommandHandlers = buildContractCommandMap(contractResolveCommandTable)
)

func buildContractCommandMap(descriptors []contractCommandDescriptor) map[string]contractCommandHandler {
	handlers := make(map[string]contractCommandHandler, len(descriptors))
	for _, descriptor := range descriptors {
		handlers[descriptor.Name] = descriptor.Handler
	}
	return handlers
}

func runContractCommandSet(ctx *commandContext, args []string, usage func(io.Writer), command string, handlers map[string]contractCommandHandler) error {
	if len(args) == 0 || wantsCommandHelp(args) {
		usage(ctx.Stdout)
		return nil
	}
	handler, ok := handlers[strings.ToLower(strings.TrimSpace(args[0]))]
	if !ok {
		return usageErrorf(command, "unknown %s command: %s", command, args[0])
	}
	return handler(ctx, args[1:])
}

func runContractCommandWithApp(_ *App, ctx *commandContext, args []string) error {
	return runContractCommandSet(ctx, args, showContractHelp, "contract", contractCommandHandlers)
}

func runContractValidateCommand(ctx *commandContext, args []string) error {
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			showContractValidateHelp(ctx.Stdout)
			return nil
		default:
			return unknownOptionError("contract validate", arg)
		}
	}

	root, err := resolveContractRoot()
	if err != nil {
		return contractRootError(err)
	}
	output, err := contractcli.Validate(root)
	if err != nil {
		return err
	}

	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return err
	}
	if err := contractcli.RenderValidate(ctx.Stdout, format, output); err != nil {
		return err
	}

	if output.Success {
		return nil
	}
	return exitCodeError{code: 1, silent: true}
}

func runContractShowCommand(ctx *commandContext, args []string) error {
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			showContractShowHelp(ctx.Stdout)
			return nil
		default:
			return unknownOptionError("contract show", arg)
		}
	}

	output, err := contractcli.LoadShowOutput()
	if err != nil {
		return contractRootError(err)
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return err
	}
	return contractcli.RenderShow(ctx.Stdout, format, output)
}

func runContractResolveCommand(ctx *commandContext, args []string) error {
	return runContractCommandSet(ctx, args, showContractResolveHelp, "contract resolve", contractResolveCommandHandlers)
}

func runContractResolveScenarioCommand(ctx *commandContext, args []string) error {
	if len(args) == 0 {
		return usageErrorf("contract resolve scenario", "contract resolve scenario requires a scenario name")
	}
	scenarioName := strings.TrimSpace(args[0])
	if scenarioName == "" {
		return usageErrorf("contract resolve scenario", "contract resolve scenario requires a scenario name")
	}

	fileKey := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			showContractResolveScenarioHelp(ctx.Stdout)
			return nil
		case "--file":
			if i+1 >= len(args) {
				return usageErrorf("contract resolve scenario", "missing value for --file")
			}
			fileKey = strings.TrimSpace(args[i+1])
			i++
		default:
			return unknownOptionError("contract resolve scenario", args[i])
		}
	}

	root, err := resolveContractRoot()
	if err != nil {
		return contractRootError(err)
	}
	output, err := contractcli.ResolveScenario(root, scenarioName, fileKey)
	if err != nil {
		return err
	}

	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return err
	}
	return contractcli.RenderResolveScenario(ctx.Stdout, format, output)
}

func runContractMatchGlobCommand(ctx *commandContext, args []string) error {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			showContractMatchGlobHelp(ctx.Stdout)
			return nil
		}
	}
	if len(args) != 2 {
		return usageErrorf("contract match-glob", "usage: vrooli contract match-glob <pattern> <path>")
	}

	output, err := contractcli.MatchGlob(args[0], args[1])
	if err != nil {
		return err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return err
	}
	return contractcli.RenderMatchGlob(ctx.Stdout, format, output)
}

func resolveContractRoot() (string, error) {
	return contractcli.ResolveRoot()
}

func contractRootError(err error) error {
	return newErrorWithCategory(fmt.Errorf("resolve repo contract root: %w", err), errorCategoryEnvironment, "Run from a Vrooli repository descendant or set VROOLI_SOURCE_ROOT", nil)
}

func showContractHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "vrooli contract - Inspect and validate the repository contract")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  vrooli contract validate")
	_, _ = fmt.Fprintln(w, "  vrooli contract show")
	_, _ = fmt.Fprintln(w, "  vrooli contract resolve scenario <name> [--file <key>]")
	_, _ = fmt.Fprintln(w, "  vrooli contract match-glob <pattern> <path>")
}

func showContractValidateHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli contract validate [--json]")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Runs schema validation plus in-process semantic and live drift checks.")
}

func showContractShowHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli contract show [--json]")
}

func showContractResolveHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli contract resolve scenario <name> [--file <key>] [--json]")
}

func showContractResolveScenarioHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli contract resolve scenario <name> [--file <key>] [--json]")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Known keys: service, docs, requirements, api, ui, cli, initialization")
}

func showContractMatchGlobHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli contract match-glob <pattern> <path> [--json]")
}
