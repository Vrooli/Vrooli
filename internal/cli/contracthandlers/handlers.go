package contracthandlers

import (
	"fmt"
	"io"

	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/contractcli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
)

type HandlerDeps[C any] struct {
	Stdout       func(C) io.Writer
	OutputFormat func(C) (cliout.Format, error)
	Service      func(C) contractapp.Service
}

func RootHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	commandHandlers := commandtree.BuildHandlerMap(buildCommandTable(deps))
	return func(ctx C, args []string) error {
		return rootcli.RunSubcommandSet(ctx, args, contractcli.RenderCommandHelp, "contract", commandHandlers, deps.Stdout)
	}
}

func buildCommandTable[C any](deps HandlerDeps[C]) []commandtree.Spec[rootcli.Handler[C]] {
	handlerMap := map[contractcli.CommandID]rootcli.Handler[C]{
		contractcli.CommandValidate:  validateHandler(deps),
		contractcli.CommandShow:      showHandler(deps),
		contractcli.CommandResolve:   resolveHandler(deps),
		contractcli.CommandMatchGlob: matchGlobHandler(deps),
	}
	source := contractcli.CommandSpecs()
	specs := make([]commandtree.Spec[rootcli.Handler[C]], 0, len(source))
	for _, spec := range source {
		handler, ok := handlerMap[spec.Handler]
		if !ok {
			continue
		}
		specs = append(specs, commandtree.Spec[rootcli.Handler[C]]{
			Name:    spec.Name,
			Summary: spec.Summary,
			Group:   spec.Group,
			Handler: handler,
		})
	}
	return specs
}

func buildResolveCommandTable[C any](deps HandlerDeps[C]) []commandtree.Spec[rootcli.Handler[C]] {
	handlerMap := map[contractcli.CommandID]rootcli.Handler[C]{
		contractcli.CommandResolveScenario: resolveScenarioHandler(deps),
	}
	source := contractcli.ResolveCommandSpecs()
	specs := make([]commandtree.Spec[rootcli.Handler[C]], 0, len(source))
	for _, spec := range source {
		handler, ok := handlerMap[spec.Handler]
		if !ok {
			continue
		}
		specs = append(specs, commandtree.Spec[rootcli.Handler[C]]{
			Name:    spec.Name,
			Summary: spec.Summary,
			Group:   spec.Group,
			Handler: handler,
		})
	}
	return specs
}

func validateHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		if _, err := contractcli.ParseValidateRequest(args); err != nil {
			if _, ok := err.(interface{ HelpText() string }); ok {
				contractcli.RenderValidateHelp(deps.Stdout(ctx))
				return nil
			}
			return rootcli.UsageErrorf("contract validate", err.Error())
		}
		output, err := deps.Service(ctx).Validate()
		if err != nil {
			return rootError(err)
		}
		format, err := deps.OutputFormat(ctx)
		if err != nil {
			return err
		}
		if err := contractcli.RenderValidate(deps.Stdout(ctx), format, output); err != nil {
			return err
		}
		if output.Success {
			return nil
		}
		return rootcli.ExitCodeError{Code: 1, Silent_: true}
	}
}

func showHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		if _, err := contractcli.ParseShowRequest(args); err != nil {
			if _, ok := err.(interface{ HelpText() string }); ok {
				contractcli.RenderShowHelp(deps.Stdout(ctx))
				return nil
			}
			return rootcli.UsageErrorf("contract show", err.Error())
		}
		output, err := deps.Service(ctx).Show()
		if err != nil {
			return rootError(err)
		}
		format, err := deps.OutputFormat(ctx)
		if err != nil {
			return err
		}
		return contractcli.RenderShow(deps.Stdout(ctx), format, output)
	}
}

func resolveHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	resolveHandlers := commandtree.BuildHandlerMap(buildResolveCommandTable(deps))
	return func(ctx C, args []string) error {
		return rootcli.RunSubcommandSet(ctx, args, contractcli.RenderResolveHelp, "contract resolve", resolveHandlers, deps.Stdout)
	}
}

func resolveScenarioHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		req, err := contractcli.ParseResolveScenarioRequest(args)
		if err != nil {
			if _, ok := err.(interface{ HelpText() string }); ok {
				contractcli.RenderResolveScenarioHelp(deps.Stdout(ctx))
				return nil
			}
			return rootcli.UsageErrorf("contract resolve scenario", err.Error())
		}
		output, err := deps.Service(ctx).ResolveScenario(req)
		if err != nil {
			return rootError(err)
		}
		format, err := deps.OutputFormat(ctx)
		if err != nil {
			return err
		}
		return contractcli.RenderResolveScenario(deps.Stdout(ctx), format, output)
	}
}

func matchGlobHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		req, err := contractcli.ParseMatchGlobRequest(args)
		if err != nil {
			if _, ok := err.(interface{ HelpText() string }); ok {
				contractcli.RenderMatchGlobHelp(deps.Stdout(ctx))
				return nil
			}
			return rootcli.UsageErrorf("contract match-glob", err.Error())
		}
		output, err := deps.Service(ctx).MatchGlob(req)
		if err != nil {
			return err
		}
		format, err := deps.OutputFormat(ctx)
		if err != nil {
			return err
		}
		return contractcli.RenderMatchGlob(deps.Stdout(ctx), format, output)
	}
}

func rootError(err error) error {
	return rootcli.NewErrorWithCategory(
		fmt.Errorf("resolve repo contract root: %w", err),
		rootcli.ErrorCategoryEnvironment,
		"Run from a Vrooli repository descendant or set VROOLI_SOURCE_ROOT",
		nil,
	)
}
