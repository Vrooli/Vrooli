package recoveryhandlers

import (
	"io"
	"time"

	recoveryapp "github.com/vrooli/vrooli/internal/app/recovery"
	"github.com/vrooli/vrooli/internal/baselinefloor"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/recoverycli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
)

// HandlerDeps supplies the context accessors the recovery command group needs.
// Store and Clock are optional seams: nil Store resolves the production
// sudo-aware cache root via baselinefloor.DefaultStore; nil Clock uses time.Now.
type HandlerDeps[C any] struct {
	Stdout       func(C) io.Writer
	Root         func(C) string
	OutputFormat func(C) (cliout.Format, error)
	Store        func(C) (*baselinefloor.Store, error)
	Clock        func() time.Time
}

// RootHandler dispatches `vrooli recovery <subcommand>`.
func RootHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	handlers := commandtree.BuildHandlerMap(buildCommandTable(deps))
	return func(ctx C, args []string) error {
		return rootcli.RunSubcommandSet(ctx, args, recoverycli.RenderCommandHelp, "recovery", handlers, deps.Stdout)
	}
}

func buildCommandTable[C any](deps HandlerDeps[C]) []commandtree.Spec[rootcli.Handler[C]] {
	serviceFactory := func(ctx C, _ cliout.Format) (recoveryapp.Service, error) { return newService(deps, ctx) }
	handlerMap := map[recoverycli.CommandID]rootcli.Handler[C]{
		recoverycli.CommandCapture: recoveryCommand(deps.Stdout,
			deps.OutputFormat,
			serviceFactory,
			func(ctx C, args []string) (recoveryapp.CaptureRequest, error) {
				return recoverycli.ParseCaptureRequest(args)
			},
			func(service recoveryapp.Service, req recoveryapp.CaptureRequest) (recoveryapp.CaptureOutput, error) {
				return service.Capture(req)
			},
			recoverycli.RenderCapture,
		),
		recoverycli.CommandRestore: recoveryCommand(deps.Stdout,
			deps.OutputFormat,
			serviceFactory,
			func(ctx C, args []string) (recoveryapp.RestoreRequest, error) {
				return recoverycli.ParseRestoreRequest(args)
			},
			func(service recoveryapp.Service, req recoveryapp.RestoreRequest) (recoveryapp.RestoreOutput, error) {
				return service.Restore(req)
			},
			recoverycli.RenderRestore,
		),
		recoverycli.CommandWrite: recoveryCommand(deps.Stdout,
			deps.OutputFormat,
			serviceFactory,
			func(ctx C, args []string) (recoveryapp.WriteRequest, error) {
				return recoverycli.ParseWriteRequest(args)
			},
			func(service recoveryapp.Service, req recoveryapp.WriteRequest) (recoveryapp.EngagementView, error) {
				return service.WriteEngagement(req)
			},
			recoverycli.RenderEngagement,
		),
		recoverycli.CommandShow: recoveryCommand(deps.Stdout,
			deps.OutputFormat,
			serviceFactory,
			func(ctx C, args []string) (recoveryapp.Ref, error) {
				return recoverycli.ParseRefRequest(recoverycli.CommandShow, "recovery show", args)
			},
			recoveryapp.Service.ShowEngagement,
			recoverycli.RenderEngagement,
		),
		recoverycli.CommandList: recoveryCommand(deps.Stdout,
			deps.OutputFormat,
			serviceFactory,
			func(ctx C, args []string) (recoveryapp.Ref, error) {
				_, err := recoverycli.ParseRefRequest(recoverycli.CommandList, "recovery list", args)
				return recoveryapp.Ref{}, err
			},
			func(service recoveryapp.Service, _ recoveryapp.Ref) (recoveryapp.ListOutput, error) {
				return service.ListEngagements()
			},
			recoverycli.RenderList,
		),
		recoverycli.CommandTouch: recoveryCommand(deps.Stdout,
			deps.OutputFormat,
			serviceFactory,
			func(ctx C, args []string) (recoveryapp.Ref, error) {
				return recoverycli.ParseRefRequest(recoverycli.CommandTouch, "recovery touch", args)
			},
			recoveryapp.Service.Touch,
			recoverycli.RenderEngagement,
		),
		recoverycli.CommandSetTTL: recoveryCommand(deps.Stdout,
			deps.OutputFormat,
			serviceFactory,
			func(ctx C, args []string) (recoveryapp.SetTTLRequest, error) {
				return recoverycli.ParseSetTTLRequest(args)
			},
			recoveryapp.Service.SetTTL,
			recoverycli.RenderEngagement,
		),
		recoverycli.CommandSetMode: recoveryCommand(deps.Stdout,
			deps.OutputFormat,
			serviceFactory,
			func(ctx C, args []string) (recoveryapp.SetModeRequest, error) {
				return recoverycli.ParseSetModeRequest(args)
			},
			recoveryapp.Service.SetMode,
			recoverycli.RenderEngagement,
		),
		recoverycli.CommandClean: recoveryCommand(deps.Stdout,
			deps.OutputFormat,
			serviceFactory,
			func(ctx C, args []string) (recoveryapp.Ref, error) {
				return recoverycli.ParseRefRequest(recoverycli.CommandClean, "recovery clean", args)
			},
			func(service recoveryapp.Service, req recoveryapp.Ref) (recoveryapp.CleanOutput, error) {
				return service.Clean(req)
			},
			recoverycli.RenderClean,
		),
		recoverycli.CommandMigrate: recoveryCommand(deps.Stdout,
			deps.OutputFormat,
			serviceFactory,
			func(ctx C, args []string) (recoveryapp.MigrateRequest, error) {
				return recoverycli.ParseMigrateRequest(args)
			},
			func(service recoveryapp.Service, req recoveryapp.MigrateRequest) (recoveryapp.MigrateOutput, error) {
				return service.Migrate(req)
			},
			recoverycli.RenderMigrate,
		),
		recoverycli.CommandNamespace: recoveryCommand(deps.Stdout,
			deps.OutputFormat,
			serviceFactory,
			func(ctx C, args []string) (recoveryapp.NamespaceRequest, error) {
				return recoverycli.ParseNamespaceRequest(args)
			},
			func(service recoveryapp.Service, req recoveryapp.NamespaceRequest) (recoveryapp.NamespaceOutput, error) {
				return service.Namespace(req)
			},
			recoverycli.RenderNamespace,
		),
	}
	return commandtree.BindSpecs(recoverycli.CommandSpecs(), handlerMap)
}

func recoveryCommand[C any, Req any, Resp any](
	stdout func(C) io.Writer,
	outputFormat func(C) (cliout.Format, error),
	serviceFactory func(C, cliout.Format) (recoveryapp.Service, error),
	parse func(C, []string) (Req, error),
	call func(recoveryapp.Service, Req) (Resp, error),
	render func(io.Writer, cliout.Format, Resp) error,
) rootcli.Handler[C] {
	return rootcli.BindService(stdout, outputFormat, serviceFactory, parse, call, render)
}

func newService[C any](deps HandlerDeps[C], ctx C) (recoveryapp.Service, error) {
	store, err := resolveStore(deps, ctx)
	if err != nil {
		return recoveryapp.Service{}, err
	}
	return recoveryapp.Service{Root: deps.Root(ctx), Store: store, Clock: deps.Clock}, nil
}

func resolveStore[C any](deps HandlerDeps[C], ctx C) (*baselinefloor.Store, error) {
	if deps.Store != nil {
		return deps.Store(ctx)
	}
	return baselinefloor.DefaultStore()
}
