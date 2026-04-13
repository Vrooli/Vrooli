package main

import (
	"fmt"
	"io"

	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/contractcli"
)

type contractCommandHandler func(ctx *commandContext, args []string) error

var (
	contractCommandTable           = buildContractCommandTable()
	contractResolveCommandTable    = buildContractResolveCommandTable()
	contractCommandHandlers        = buildContractCommandMap(contractCommandTable)
	contractResolveCommandHandlers = buildContractCommandMap(contractResolveCommandTable)
)

func buildContractCommandMap(descriptors []commandtree.Spec[contractCommandHandler]) map[string]contractCommandHandler {
	return commandtree.BuildHandlerMap(descriptors)
}

func runContractCommandSet(ctx *commandContext, args []string, usage func(io.Writer), command string, handlers map[string]contractCommandHandler) error {
	if len(args) == 0 || wantsCommandHelp(args) {
		usage(ctx.Stdout)
		return nil
	}
	handler, ok := handlers[commandtree.NormalizeName(args[0])]
	if !ok {
		return usageErrorf(command, "unknown %s command: %s", command, args[0])
	}
	return handler(ctx, args[1:])
}

func runContractCommandWithApp(_ *App, ctx *commandContext, args []string) error {
	return runContractCommandSet(ctx, args, showContractHelp, "contract", contractCommandHandlers)
}

func runContractValidateCommand(ctx *commandContext, args []string) error {
	if _, err := contractcli.ParseValidateRequest(args); err != nil {
		if _, ok := err.(interface{ HelpText() string }); ok {
			contractcli.RenderValidateHelp(ctx.Stdout)
			return nil
		}
		return usageErrorf("contract validate", err.Error())
	}

	output, err := newContractCommandService().Validate()
	if err != nil {
		return contractRootError(err)
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
	if _, err := contractcli.ParseShowRequest(args); err != nil {
		if _, ok := err.(interface{ HelpText() string }); ok {
			contractcli.RenderShowHelp(ctx.Stdout)
			return nil
		}
		return usageErrorf("contract show", err.Error())
	}

	output, err := newContractCommandService().Show()
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
	return runContractCommandSet(ctx, args, contractcli.RenderResolveHelp, "contract resolve", contractResolveCommandHandlers)
}

func runContractResolveScenarioCommand(ctx *commandContext, args []string) error {
	req, err := contractcli.ParseResolveScenarioRequest(args)
	if err != nil {
		if _, ok := err.(interface{ HelpText() string }); ok {
			contractcli.RenderResolveScenarioHelp(ctx.Stdout)
			return nil
		}
		return usageErrorf("contract resolve scenario", err.Error())
	}

	output, err := newContractCommandService().ResolveScenario(contractapp.ResolveScenarioRequest{
		ScenarioName: req.ScenarioName,
		FileKey:      req.FileKey,
	})
	if err != nil {
		return contractRootError(err)
	}

	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return err
	}
	return contractcli.RenderResolveScenario(ctx.Stdout, format, output)
}

func runContractMatchGlobCommand(ctx *commandContext, args []string) error {
	req, err := contractcli.ParseMatchGlobRequest(args)
	if err != nil {
		if _, ok := err.(interface{ HelpText() string }); ok {
			contractcli.RenderMatchGlobHelp(ctx.Stdout)
			return nil
		}
		return usageErrorf("contract match-glob", err.Error())
	}

	output, err := newContractCommandService().MatchGlob(contractapp.MatchGlobRequest{Pattern: req.Pattern, Path: req.Path})
	if err != nil {
		return err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return err
	}
	return contractcli.RenderMatchGlob(ctx.Stdout, format, output)
}

func contractRootError(err error) error {
	return newErrorWithCategory(fmt.Errorf("resolve repo contract root: %w", err), errorCategoryEnvironment, "Run from a Vrooli repository descendant or set VROOLI_SOURCE_ROOT", nil)
}

func showContractHelp(w io.Writer) {
	contractcli.RenderCommandHelp(w)
}

func showContractValidateHelp(w io.Writer) {
	contractcli.RenderValidateHelp(w)
}

func showContractShowHelp(w io.Writer) {
	contractcli.RenderShowHelp(w)
}

func showContractResolveHelp(w io.Writer) {
	contractcli.RenderResolveHelp(w)
}

func showContractResolveScenarioHelp(w io.Writer) {
	contractcli.RenderResolveScenarioHelp(w)
}

func showContractMatchGlobHelp(w io.Writer) {
	contractcli.RenderMatchGlobHelp(w)
}

func buildContractCommandTable() []commandtree.Spec[contractCommandHandler] {
	handlerMap := map[contractcli.CommandID]contractCommandHandler{
		contractcli.CommandValidate:  runContractValidateCommand,
		contractcli.CommandShow:      runContractShowCommand,
		contractcli.CommandResolve:   runContractResolveCommand,
		contractcli.CommandMatchGlob: runContractMatchGlobCommand,
	}
	source := contractcli.CommandSpecs()
	specs := make([]commandtree.Spec[contractCommandHandler], 0, len(source))
	for _, spec := range source {
		handler, ok := handlerMap[spec.Handler]
		if !ok {
			continue
		}
		specs = append(specs, commandtree.Spec[contractCommandHandler]{
			Name:    spec.Name,
			Summary: spec.Summary,
			Group:   spec.Group,
			Handler: handler,
		})
	}
	return specs
}

func buildContractResolveCommandTable() []commandtree.Spec[contractCommandHandler] {
	handlerMap := map[contractcli.CommandID]contractCommandHandler{
		contractcli.CommandResolveScenario: runContractResolveScenarioCommand,
	}
	source := contractcli.ResolveCommandSpecs()
	specs := make([]commandtree.Spec[contractCommandHandler], 0, len(source))
	for _, spec := range source {
		handler, ok := handlerMap[spec.Handler]
		if !ok {
			continue
		}
		specs = append(specs, commandtree.Spec[contractCommandHandler]{
			Name:    spec.Name,
			Summary: spec.Summary,
			Group:   spec.Group,
			Handler: handler,
		})
	}
	return specs
}
