package capacityhandlers

import (
	"context"
	"fmt"

	"github.com/vrooli/vrooli/internal/cli/capacitycli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

func residencyHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		format, err := deps.OutputFormat(ctx)
		if err != nil {
			return err
		}
		req, err := capacitycli.ParseResidencyRequest(args)
		if err != nil {
			return err
		}
		svc, err := newService(deps, ctx, format)
		if err != nil {
			return err
		}
		switch req.Action {
		case "begin", "complete", "reconcile", "recover":
			transition, err := capacitycli.ReadResidencyRequest(req.RequestFile)
			if err != nil {
				return err
			}
			if req.Action == "begin" {
				out, err := svc.ResidencyBegin(context.Background(), transition)
				if err != nil {
					return err
				}
				return capacitycli.RenderResidencyRecord(deps.Stdout(ctx), format, out)
			}
			if req.Action == "reconcile" {
				out, err := svc.ResidencyReconcile(context.Background(), transition)
				if err != nil {
					return err
				}
				return capacitycli.RenderResidencyRecord(deps.Stdout(ctx), format, out)
			}
			if req.Action == "recover" {
				out, err := svc.ResidencyRecover(context.Background(), transition)
				if err != nil {
					return err
				}
				return capacitycli.RenderResidencyRecord(deps.Stdout(ctx), format, out)
			}
			out, err := svc.ResidencyComplete(context.Background(), transition)
			if err != nil {
				return err
			}
			return capacitycli.RenderResidencyRecord(deps.Stdout(ctx), format, out)
		case "get":
			out, err := svc.ResidencyGet(context.Background(), req.TransitionID)
			if err != nil {
				return err
			}
			return capacitycli.RenderResidencyRecord(deps.Stdout(ctx), format, out)
		case "history":
			out, err := svc.ResidencyHistory(context.Background(), req.TransitionID)
			if err != nil {
				return err
			}
			return capacitycli.RenderResidencyHistory(deps.Stdout(ctx), format, out)
		case "release":
			release, err := capacitycli.ReadResidencyReleaseRequest(req.RequestFile)
			if err != nil {
				return err
			}
			out, err := svc.ResidencyRelease(context.Background(), release)
			if err != nil {
				return err
			}
			return capacitycli.RenderResidencyClaim(deps.Stdout(ctx), format, out)
		default:
			return fmt.Errorf("unsupported residency action %q", req.Action)
		}
	}
}
