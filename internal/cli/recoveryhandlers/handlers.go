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
	handlerMap := map[recoverycli.CommandID]rootcli.Handler[C]{
		recoverycli.CommandCapture: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (recoveryapp.CaptureRequest, error) {
				return recoverycli.ParseCaptureRequest(args)
			},
			func(ctx C, req recoveryapp.CaptureRequest) (cliout.Format, recoveryapp.CaptureOutput, error) {
				format, service, err := newService(deps, ctx)
				if err != nil {
					return "", recoveryapp.CaptureOutput{}, err
				}
				resp, err := service.Capture(req)
				return format, resp, err
			},
			recoverycli.RenderCapture,
		),
		recoverycli.CommandRestore: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (recoveryapp.RestoreRequest, error) {
				return recoverycli.ParseRestoreRequest(args)
			},
			func(ctx C, req recoveryapp.RestoreRequest) (cliout.Format, recoveryapp.RestoreOutput, error) {
				format, service, err := newService(deps, ctx)
				if err != nil {
					return "", recoveryapp.RestoreOutput{}, err
				}
				resp, err := service.Restore(req)
				return format, resp, err
			},
			recoverycli.RenderRestore,
		),
		recoverycli.CommandWrite: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (recoveryapp.WriteRequest, error) {
				return recoverycli.ParseWriteRequest(args)
			},
			func(ctx C, req recoveryapp.WriteRequest) (cliout.Format, recoveryapp.EngagementView, error) {
				format, service, err := newService(deps, ctx)
				if err != nil {
					return "", recoveryapp.EngagementView{}, err
				}
				resp, err := service.WriteEngagement(req)
				return format, resp, err
			},
			recoverycli.RenderEngagement,
		),
		recoverycli.CommandShow: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (recoveryapp.Ref, error) {
				return recoverycli.ParseRefRequest(recoverycli.CommandShow, "recovery show", args)
			},
			func(ctx C, req recoveryapp.Ref) (cliout.Format, recoveryapp.EngagementView, error) {
				format, service, err := newService(deps, ctx)
				if err != nil {
					return "", recoveryapp.EngagementView{}, err
				}
				resp, err := service.ShowEngagement(req)
				return format, resp, err
			},
			recoverycli.RenderEngagement,
		),
		recoverycli.CommandList: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (recoveryapp.Ref, error) {
				_, err := recoverycli.ParseRefRequest(recoverycli.CommandList, "recovery list", args)
				return recoveryapp.Ref{}, err
			},
			func(ctx C, _ recoveryapp.Ref) (cliout.Format, recoveryapp.ListOutput, error) {
				format, service, err := newService(deps, ctx)
				if err != nil {
					return "", recoveryapp.ListOutput{}, err
				}
				resp, err := service.ListEngagements()
				return format, resp, err
			},
			recoverycli.RenderList,
		),
		recoverycli.CommandTouch: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (recoveryapp.Ref, error) {
				return recoverycli.ParseRefRequest(recoverycli.CommandTouch, "recovery touch", args)
			},
			func(ctx C, req recoveryapp.Ref) (cliout.Format, recoveryapp.EngagementView, error) {
				format, service, err := newService(deps, ctx)
				if err != nil {
					return "", recoveryapp.EngagementView{}, err
				}
				resp, err := service.Touch(req)
				return format, resp, err
			},
			recoverycli.RenderEngagement,
		),
		recoverycli.CommandSetTTL: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (recoveryapp.SetTTLRequest, error) {
				return recoverycli.ParseSetTTLRequest(args)
			},
			func(ctx C, req recoveryapp.SetTTLRequest) (cliout.Format, recoveryapp.EngagementView, error) {
				format, service, err := newService(deps, ctx)
				if err != nil {
					return "", recoveryapp.EngagementView{}, err
				}
				resp, err := service.SetTTL(req)
				return format, resp, err
			},
			recoverycli.RenderEngagement,
		),
		recoverycli.CommandClean: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (recoveryapp.Ref, error) {
				return recoverycli.ParseRefRequest(recoverycli.CommandClean, "recovery clean", args)
			},
			func(ctx C, req recoveryapp.Ref) (cliout.Format, recoveryapp.CleanOutput, error) {
				format, service, err := newService(deps, ctx)
				if err != nil {
					return "", recoveryapp.CleanOutput{}, err
				}
				resp, err := service.Clean(req)
				return format, resp, err
			},
			recoverycli.RenderClean,
		),
		recoverycli.CommandMigrate: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (recoveryapp.MigrateRequest, error) {
				return recoverycli.ParseMigrateRequest(args)
			},
			func(ctx C, req recoveryapp.MigrateRequest) (cliout.Format, recoveryapp.MigrateOutput, error) {
				format, service, err := newService(deps, ctx)
				if err != nil {
					return "", recoveryapp.MigrateOutput{}, err
				}
				resp, err := service.Migrate(req)
				return format, resp, err
			},
			recoverycli.RenderMigrate,
		),
		recoverycli.CommandNamespace: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (recoveryapp.NamespaceRequest, error) {
				return recoverycli.ParseNamespaceRequest(args)
			},
			func(ctx C, req recoveryapp.NamespaceRequest) (cliout.Format, recoveryapp.NamespaceOutput, error) {
				format, service, err := newService(deps, ctx)
				if err != nil {
					return "", recoveryapp.NamespaceOutput{}, err
				}
				resp, err := service.Namespace(req)
				return format, resp, err
			},
			recoverycli.RenderNamespace,
		),
	}
	return commandtree.BindSpecs(recoverycli.CommandSpecs(), handlerMap)
}

func newService[C any](deps HandlerDeps[C], ctx C) (cliout.Format, recoveryapp.Service, error) {
	format, err := deps.OutputFormat(ctx)
	if err != nil {
		return "", recoveryapp.Service{}, err
	}
	store, err := resolveStore(deps, ctx)
	if err != nil {
		return "", recoveryapp.Service{}, err
	}
	return format, recoveryapp.Service{Root: deps.Root(ctx), Store: store, Clock: deps.Clock}, nil
}

func resolveStore[C any](deps HandlerDeps[C], ctx C) (*baselinefloor.Store, error) {
	if deps.Store != nil {
		return deps.Store(ctx)
	}
	return baselinefloor.DefaultStore()
}
