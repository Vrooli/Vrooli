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
	return commandtree.BindSpecs(contractcli.CommandSpecs(), handlerMap)
}

func buildResolveCommandTable[C any](deps HandlerDeps[C]) []commandtree.Spec[rootcli.Handler[C]] {
	handlerMap := map[contractcli.CommandID]rootcli.Handler[C]{
		contractcli.CommandResolveScenario: resolveScenarioHandler(deps),
	}
	return commandtree.BindSpecs(contractcli.ResolveCommandSpecs(), handlerMap)
}

func validateHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return bindContractCommand(deps,
		contractcli.ParseValidateRequest,
		func(ctx C, _ contractcli.NoArgsRequest) (contractapp.ValidationOutput, error) {
			return deps.Service(ctx).Validate()
		},
		func(w io.Writer, format cliout.Format, output contractapp.ValidationOutput) error {
			return contractcli.RenderValidate(w, format, output)
		},
		rootError,
		func(output contractapp.ValidationOutput) error {
			if output.Success {
				return nil
			}
			return rootcli.ExitCodeError{Code: 1, Silent_: true}
		},
	)
}

func showHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return bindContractCommand(deps,
		contractcli.ParseShowRequest,
		func(ctx C, _ contractcli.NoArgsRequest) (contractapp.ShowOutput, error) {
			return deps.Service(ctx).Show()
		},
		contractcli.RenderShow,
		rootError,
		nil,
	)
}

func resolveHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	resolveHandlers := commandtree.BuildHandlerMap(buildResolveCommandTable(deps))
	return func(ctx C, args []string) error {
		return rootcli.RunSubcommandSet(ctx, args, contractcli.RenderResolveHelp, "contract resolve", resolveHandlers, deps.Stdout)
	}
}

func resolveScenarioHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return bindContractCommand(deps,
		contractcli.ParseResolveScenarioRequest,
		func(ctx C, req contractapp.ResolveScenarioRequest) (contractapp.ResolveScenarioOutput, error) {
			return deps.Service(ctx).ResolveScenario(req)
		},
		contractcli.RenderResolveScenario,
		rootError,
		nil,
	)
}

func matchGlobHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return bindContractCommand(deps,
		contractcli.ParseMatchGlobRequest,
		func(ctx C, req contractapp.MatchGlobRequest) (contractapp.MatchGlobOutput, error) {
			return deps.Service(ctx).MatchGlob(req)
		},
		contractcli.RenderMatchGlob,
		nil,
		nil,
	)
}

func bindContractCommand[C any, Req any, Resp any](
	deps HandlerDeps[C],
	parse func([]string) (Req, error),
	run func(C, Req) (Resp, error),
	render func(io.Writer, cliout.Format, Resp) error,
	mapErr func(error) error,
	after func(Resp) error,
) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		req, err := parse(args)
		if err != nil {
			if rootcli.HandleHelp(deps.Stdout(ctx), err) {
				return nil
			}
			return rootcli.UsageErrorf("", err.Error())
		}
		output, err := run(ctx, req)
		if err != nil {
			if mapErr != nil {
				return mapErr(err)
			}
			return err
		}
		format, err := deps.OutputFormat(ctx)
		if err != nil {
			return err
		}
		if err := render(deps.Stdout(ctx), format, output); err != nil {
			return err
		}
		if after != nil {
			return after(output)
		}
		return nil
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
