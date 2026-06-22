package capacityhandlers

import (
	"context"
	"io"
	"time"

	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	"github.com/vrooli/vrooli/internal/cli/capacitycli"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
)

// HandlerDeps supplies the context accessors the capacity command group needs.
// Service is an optional seam: a nil ServiceFor uses the production service
// (real ledger, hostinventory source, docker attribution).
type HandlerDeps[C any] struct {
	Stdout       func(C) io.Writer
	OutputFormat func(C) (cliout.Format, error)
	ServiceFor   func(C) capacityapp.Service
}

// RootHandler dispatches `vrooli capacity <subcommand>`.
func RootHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	handlers := commandtree.BuildHandlerMap(buildCommandTable(deps))
	return func(ctx C, args []string) error {
		return rootcli.RunSubcommandSet(ctx, args, capacitycli.RenderCommandHelp, "capacity", handlers, deps.Stdout)
	}
}

func (d HandlerDeps[C]) service(ctx C) capacityapp.Service {
	if d.ServiceFor != nil {
		return d.ServiceFor(ctx)
	}
	return capacityapp.Service{Clock: func() time.Time { return time.Now().UTC() }}
}

func buildCommandTable[C any](deps HandlerDeps[C]) []commandtree.Spec[rootcli.Handler[C]] {
	handlerMap := map[capacitycli.CommandID]rootcli.Handler[C]{
		capacitycli.CommandClaim: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (capacityapp.ClaimRequest, error) {
				return capacitycli.ParseClaimRequest(args)
			},
			func(ctx C, req capacityapp.ClaimRequest) (cliout.Format, capacityapp.ClaimOutput, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", capacityapp.ClaimOutput{}, err
				}
				resp, err := deps.service(ctx).Claim(context.Background(), req)
				return format, resp, err
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
		capacitycli.CommandRelease: refHandler(deps, capacitycli.CommandRelease, "vrooli capacity release",
			func(s capacityapp.Service, ref capacityapp.Ref) (capacityapp.ClaimView, error) {
				return s.Release(context.Background(), ref)
			}),
		capacitycli.CommandList: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (capacityapp.ListRequest, error) { return capacitycli.ParseListRequest(args) },
			func(ctx C, req capacityapp.ListRequest) (cliout.Format, capacityapp.ListOutput, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", capacityapp.ListOutput{}, err
				}
				resp, err := deps.service(ctx).List(context.Background(), req)
				return format, resp, err
			},
			capacitycli.RenderList,
		),
		capacitycli.CommandReconcile: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (struct{}, error) {
				_, err := capacitycli.ParseListRequest(nil) // reconcile takes only --json; parse args for help/validation
				_ = err
				return struct{}{}, nil
			},
			func(ctx C, _ struct{}) (cliout.Format, capacityapp.ReconcileOutput, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", capacityapp.ReconcileOutput{}, err
				}
				resp, err := deps.service(ctx).Reconcile(context.Background())
				return format, resp, err
			},
			capacitycli.RenderReconcile,
		),
		capacitycli.CommandSweep: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (struct{}, error) {
				return struct{}{}, capacitycli.ParseSweepRequest(args)
			},
			func(ctx C, _ struct{}) (cliout.Format, capacityapp.SweepOutput, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", capacityapp.SweepOutput{}, err
				}
				resp, err := deps.service(ctx).Sweep(context.Background())
				return format, resp, err
			},
			capacitycli.RenderSweep,
		),
		capacitycli.CommandPolicy: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (capacitycli.PolicyArgs, error) {
				return capacitycli.ParsePolicyRequest(args)
			},
			func(ctx C, req capacitycli.PolicyArgs) (cliout.Format, capacityapp.PolicyOutput, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", capacityapp.PolicyOutput{}, err
				}
				svc := deps.service(ctx)
				if req.Action == "set" {
					resp, setErr := svc.PolicySet(context.Background(), req.Key, req.Value)
					return format, resp, setErr
				}
				resp, getErr := svc.PolicyGet(context.Background(), req.Key)
				return format, resp, getErr
			},
			capacitycli.RenderPolicy,
		),
	}
	return commandtree.BindSpecs(capacitycli.CommandSpecs(), handlerMap)
}

func refHandler[C any](deps HandlerDeps[C], id capacitycli.CommandID, command string, run func(capacityapp.Service, capacityapp.Ref) (capacityapp.ClaimView, error)) rootcli.Handler[C] {
	return rootcli.BindGlobalCommand(deps.Stdout,
		func(ctx C, args []string) (capacityapp.Ref, error) {
			return capacitycli.ParseRefRequest(id, command, args)
		},
		func(ctx C, ref capacityapp.Ref) (cliout.Format, capacityapp.ClaimView, error) {
			format, err := deps.OutputFormat(ctx)
			if err != nil {
				return "", capacityapp.ClaimView{}, err
			}
			resp, err := run(deps.service(ctx), ref)
			return format, resp, err
		},
		capacitycli.RenderClaimView,
	)
}
