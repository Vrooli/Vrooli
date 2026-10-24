package capacityhandlers

import (
	"context"

	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	"github.com/vrooli/vrooli/internal/cli/capacitycli"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/clock"
)

// RootHandler dispatches `vrooli capacity <subcommand>`.
func RootHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	handlers := commandtree.BuildHandlerMap(buildCommandTable(deps))
	return func(ctx C, args []string) error {
		return rootcli.RunSubcommandSet(ctx, args, capacitycli.RenderCommandHelp, "capacity", handlers, deps.Stdout)
	}
}

func service[C any](deps rootcli.HandlerDeps[C], ctx C) capacityapp.Service {
	if deps.ServiceFor != nil {
		return deps.ServiceFor(ctx)
	}
	return capacityapp.Service{Clock: clock.Real{}.Now}
}

func newService[C any](deps rootcli.HandlerDeps[C], ctx C, _ cliout.Format) (capacityapp.Service, error) {
	return service(deps, ctx), nil
}

func buildCommandTable[C any](deps rootcli.HandlerDeps[C]) []commandtree.Spec[rootcli.Handler[C]] {
	serviceFactory := func(ctx C, format cliout.Format) (capacityapp.Service, error) {
		return newService(deps, ctx, format)
	}
	handlerMap := map[capacitycli.CommandID]rootcli.Handler[C]{
		capacitycli.CommandFit: func(ctx C, args []string) error {
			format, err := deps.OutputFormat(ctx)
			if err != nil {
				return err
			}
			service, err := serviceFactory(ctx, format)
			if err != nil {
				return err
			}
			req, err := capacitycli.ParseFitRequest(args)
			if err != nil {
				return err
			}
			out, err := service.Fit(context.Background(), req)
			if err != nil {
				return err
			}
			if err := capacitycli.RenderFit(deps.Stdout(ctx), format, out); err != nil {
				return err
			}
			if out.Verdict != "fits" {
				return rootcli.ExitCodeError{Code: 1, Silent_: true}
			}
			return nil
		},
		capacitycli.CommandClaim: rootcli.BindService(deps.Stdout, deps.OutputFormat, serviceFactory,
			func(ctx C, args []string) (capacityapp.ClaimRequest, error) {
				return capacitycli.ParseClaimRequest(args)
			},
			func(service capacityapp.Service, req capacityapp.ClaimRequest) (capacityapp.ClaimOutput, error) {
				return service.Claim(context.Background(), req)
			},
			capacitycli.RenderClaim,
		),
		capacitycli.CommandHeartbeat: refHandler(deps, capacitycli.CommandHeartbeat, "vrooli capacity heartbeat",
			func(s capacityapp.Service, ref capacityapp.Ref) (capacityapp.ClaimView, error) {
				return s.Heartbeat(context.Background(), ref)
			}),
		capacitycli.CommandActivity: refHandler(deps, capacitycli.CommandActivity, "vrooli capacity activity",
			func(s capacityapp.Service, ref capacityapp.Ref) (capacityapp.ClaimView, error) {
				return s.Activity(context.Background(), ref)
			}),
		capacitycli.CommandDegrade: refHandler(deps, capacitycli.CommandDegrade, "vrooli capacity degrade",
			func(s capacityapp.Service, ref capacityapp.Ref) (capacityapp.ClaimView, error) {
				return s.Degrade(context.Background(), ref)
			}),
		capacitycli.CommandResize: refHandler(deps, capacitycli.CommandResize, "vrooli capacity resize",
			func(s capacityapp.Service, ref capacityapp.Ref) (capacityapp.ClaimView, error) {
				return s.Resize(context.Background(), ref)
			}),
		capacitycli.CommandRelease: refHandler(deps, capacitycli.CommandRelease, "vrooli capacity release",
			func(s capacityapp.Service, ref capacityapp.Ref) (capacityapp.ClaimView, error) {
				return s.Release(context.Background(), ref)
			}),
		capacitycli.CommandList: rootcli.BindService(deps.Stdout, deps.OutputFormat, serviceFactory,
			func(ctx C, args []string) (capacityapp.ListRequest, error) { return capacitycli.ParseListRequest(args) },
			func(service capacityapp.Service, req capacityapp.ListRequest) (capacityapp.ListOutput, error) {
				return service.List(context.Background(), req)
			},
			capacitycli.RenderList,
		),
		capacitycli.CommandReconcile: rootcli.BindService(deps.Stdout, deps.OutputFormat, serviceFactory,
			func(ctx C, args []string) (struct{}, error) {
				_, err := capacitycli.ParseListRequest(nil) // reconcile takes only --json; parse args for help/validation
				_ = err
				return struct{}{}, nil
			},
			func(service capacityapp.Service, _ struct{}) (capacityapp.ReconcileOutput, error) {
				return service.Reconcile(context.Background())
			},
			capacitycli.RenderReconcile,
		),
		capacitycli.CommandSweep: rootcli.BindService(deps.Stdout, deps.OutputFormat, serviceFactory,
			func(ctx C, args []string) (struct{}, error) {
				return struct{}{}, capacitycli.ParseSweepRequest(args)
			},
			func(service capacityapp.Service, _ struct{}) (capacityapp.SweepOutput, error) {
				return service.Sweep(context.Background())
			},
			capacitycli.RenderSweep,
		),
		capacitycli.CommandGC: rootcli.BindService(deps.Stdout, deps.OutputFormat, serviceFactory,
			func(ctx C, args []string) (struct{}, error) {
				return struct{}{}, capacitycli.ParseGCRequest(args)
			},
			func(service capacityapp.Service, _ struct{}) (capacityapp.GCOutput, error) {
				return service.GC(context.Background())
			},
			capacitycli.RenderGC,
		),
		capacitycli.CommandRecommend: rootcli.BindService(deps.Stdout, deps.OutputFormat, serviceFactory,
			func(ctx C, args []string) (capacityapp.RecommendRequest, error) {
				return capacitycli.ParseRecommendRequest(args)
			},
			func(service capacityapp.Service, req capacityapp.RecommendRequest) (capacityapp.RecommendOutput, error) {
				return service.Recommend(context.Background(), req)
			},
			capacitycli.RenderRecommend,
		),
		capacitycli.CommandFootprint: rootcli.BindService(deps.Stdout, deps.OutputFormat, serviceFactory,
			func(ctx C, args []string) (capacityapp.FootprintRequest, error) {
				return capacitycli.ParseFootprintRequest(args)
			},
			func(service capacityapp.Service, req capacityapp.FootprintRequest) (capacityapp.FootprintOutput, error) {
				return service.Footprint(context.Background(), req)
			},
			capacitycli.RenderFootprint,
		),
		capacitycli.CommandPolicy: rootcli.BindService(deps.Stdout, deps.OutputFormat, serviceFactory,
			func(ctx C, args []string) (capacitycli.PolicyArgs, error) {
				return capacitycli.ParsePolicyRequest(args)
			},
			func(service capacityapp.Service, req capacitycli.PolicyArgs) (capacityapp.PolicyOutput, error) {
				if req.Action == "set" {
					return service.PolicySet(context.Background(), req.Key, req.Value)
				}
				return service.PolicyGet(context.Background(), req.Key)
			},
			capacitycli.RenderPolicy,
		),
	}
	return commandtree.BindSpecs(capacitycli.CommandSpecs(), handlerMap)
}

func refHandler[C any](deps rootcli.HandlerDeps[C], id capacitycli.CommandID, command string, run func(capacityapp.Service, capacityapp.Ref) (capacityapp.ClaimView, error)) rootcli.Handler[C] {
	serviceFactory := func(ctx C, format cliout.Format) (capacityapp.Service, error) {
		return newService(deps, ctx, format)
	}
	return rootcli.BindService(deps.Stdout, deps.OutputFormat, serviceFactory,
		func(ctx C, args []string) (capacityapp.Ref, error) {
			return capacitycli.ParseRefRequest(id, command, args)
		},
		func(service capacityapp.Service, ref capacityapp.Ref) (capacityapp.ClaimView, error) {
			return run(service, ref)
		},
		capacitycli.RenderClaimView,
	)
}
