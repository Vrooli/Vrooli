package capacityhandlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	engine "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/cli/capacitycli"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/clock"
)

// RootHandler dispatches `vrooli capacity <subcommand>`.
func RootHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	specs := buildCommandTable(deps)
	return func(ctx C, args []string) error {
		return rootcli.RunSubcommandSpecs(ctx, args, capacitycli.RenderCommandHelp, "capacity", specs, deps.Stdout)
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
		capacitycli.CommandDiskReserve:      diskReserveHandler(deps),
		capacitycli.CommandDiskHeartbeat:    diskHeartbeatHandler(deps),
		capacitycli.CommandDiskExtend:       diskExtendHandler(deps),
		capacitycli.CommandDiskComplete:     diskCompleteHandler(deps),
		capacitycli.CommandDiskList:         diskListHandler(deps),
		capacitycli.CommandDiskReservations: diskReservationsHandler(deps),
		capacitycli.CommandDiskReconcile:    diskReconcileHandler(deps),
		capacitycli.CommandDiskObserve:      diskObserveHandler(deps),
		capacitycli.CommandDiskSample:       diskSampleHandler(deps),
	}
	handlerMap[capacitycli.CommandResidency] = residencyHandler(deps)
	return commandtree.BindSpecs(capacitycli.CommandSpecs(), handlerMap)
}

type diskLeaseFile struct {
	ID           string `json:"id"`
	Capability   string `json:"capability"`
	ProcessID    int    `json:"process_id"`
	ProcessStart string `json:"process_start"`
}

func diskReserveHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		format, err := deps.OutputFormat(ctx)
		if err != nil {
			return err
		}
		svc, err := newService(deps, ctx, format)
		if err != nil {
			return err
		}
		parsed, err := capacitycli.ParseDiskReserveRequest(args)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(parsed.RequestFile)
		if err != nil {
			return fmt.Errorf("read disk job request: %w", err)
		}
		var req engine.DiskJobRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			return fmt.Errorf("decode disk job request: %w", err)
		}
		if req.ProcessID == 0 || req.ProcessStart == "" {
			return errors.New("disk-reserve request must bind process_id and process_start to the long-running job")
		}
		receipt, cred, err := svc.DiskJobReserve(context.Background(), req)
		if err != nil {
			return err
		}
		lease := diskLeaseFile{ID: cred.ID, Capability: cred.Capability, ProcessID: cred.ProcessID, ProcessStart: cred.ProcessStart}
		if err := writePrivateLeaseFile(parsed.LeaseFile, lease); err != nil {
			_ = os.Remove(parsed.LeaseFile)
			_, completeErr := svc.DiskJobComplete(context.Background(), cred)
			if completeErr != nil {
				return errors.Join(fmt.Errorf("disk lease admitted but credential persistence failed; lease remains active %s: %w", receipt.ID, err), fmt.Errorf("complete admission with verified no-write cleanup: %w", completeErr))
			}
			return fmt.Errorf("disk lease credential could not be persisted; the empty admission was completed safely: %w", err)
		}
		return renderDiskOutput(deps.Stdout(ctx), format, receipt)
	}
}

func diskHeartbeatHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		format, err := deps.OutputFormat(ctx)
		if err != nil {
			return err
		}
		svc, err := newService(deps, ctx, format)
		if err != nil {
			return err
		}
		parsed, err := capacitycli.ParseDiskLeaseRequest(capacitycli.CommandDiskHeartbeat, args)
		if err != nil {
			return err
		}
		cred, err := readPrivateLeaseFile(parsed.LeaseFile)
		if err != nil {
			return err
		}
		receipt, err := svc.DiskJobHeartbeat(context.Background(), cred)
		if err != nil {
			return err
		}
		return renderDiskOutput(deps.Stdout(ctx), format, receipt)
	}
}

func diskExtendHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		format, err := deps.OutputFormat(ctx)
		if err != nil {
			return err
		}
		svc, err := newService(deps, ctx, format)
		if err != nil {
			return err
		}
		parsed, err := capacitycli.ParseDiskExtendRequest(args)
		if err != nil {
			return err
		}
		cred, err := readPrivateLeaseFile(parsed.LeaseFile)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(parsed.PathsFile)
		if err != nil {
			return err
		}
		var paths []engine.DiskJobPath
		if err := json.Unmarshal(raw, &paths); err != nil {
			return fmt.Errorf("decode extension paths: %w", err)
		}
		receipt, err := svc.DiskJobExtend(context.Background(), cred, paths)
		if err != nil {
			return err
		}
		return renderDiskOutput(deps.Stdout(ctx), format, receipt)
	}
}

func diskCompleteHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		format, err := deps.OutputFormat(ctx)
		if err != nil {
			return err
		}
		svc, err := newService(deps, ctx, format)
		if err != nil {
			return err
		}
		parsed, err := capacitycli.ParseDiskCompleteRequest(args)
		if err != nil {
			return err
		}
		cred, err := readPrivateLeaseFile(parsed.LeaseFile)
		if err != nil {
			return err
		}
		receipt, err := svc.DiskJobComplete(context.Background(), cred)
		if err != nil {
			return err
		}
		if err := renderDiskOutput(deps.Stdout(ctx), format, receipt); err != nil {
			return err
		}
		if err := os.Remove(parsed.LeaseFile); err != nil {
			return fmt.Errorf("lease completed but private credential file could not be removed: %w", err)
		}
		return nil
	}
}

func diskListHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		format, err := deps.OutputFormat(ctx)
		if err != nil {
			return err
		}
		svc, err := newService(deps, ctx, format)
		if err != nil {
			return err
		}
		parsed, err := capacitycli.ParseDiskListRequest(args)
		if err != nil {
			return err
		}
		if parsed.Observe {
			out, err := svc.DiskHeadroom(context.Background())
			if err != nil {
				return err
			}
			return renderDiskOutput(deps.Stdout(ctx), format, out)
		}
		rows, err := svc.DiskJobList(context.Background(), parsed.Active)
		if err != nil {
			return err
		}
		return renderDiskOutput(deps.Stdout(ctx), format, rows)
	}
}

func diskReservationsHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		format, err := deps.OutputFormat(ctx)
		if err != nil {
			return err
		}
		svc, err := newService(deps, ctx, format)
		if err != nil {
			return err
		}
		parsed, err := capacitycli.ParseDiskReservationsRequest(args)
		if err != nil {
			return err
		}
		rows, err := svc.DiskReservationList(context.Background(), engine.DiskReservationFilter{ActiveOnly: parsed.Active, OwnerID: parsed.Owner, FilesystemID: parsed.Filesystem})
		if err != nil {
			return err
		}
		return renderDiskOutput(deps.Stdout(ctx), format, rows)
	}
}

func diskReconcileHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		format, err := deps.OutputFormat(ctx)
		if err != nil {
			return err
		}
		svc, err := newService(deps, ctx, format)
		if err != nil {
			return err
		}
		parsed, err := capacitycli.ParseDiskReconcileRequest(args)
		if err != nil {
			return err
		}
		receipt, err := svc.DiskJobReconcile(context.Background(), parsed.LeaseID, parsed.ProcessStopped, parsed.CleanupVerified)
		if err != nil {
			return err
		}
		return renderDiskOutput(deps.Stdout(ctx), format, receipt)
	}
}

func diskObserveHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		format, err := deps.OutputFormat(ctx)
		if err != nil {
			return err
		}
		svc, err := newService(deps, ctx, format)
		if err != nil {
			return err
		}
		parsed, err := capacitycli.ParseDiskObserveRequest(args)
		if err != nil {
			return err
		}
		cred, err := readPrivateLeaseFile(parsed.LeaseFile)
		if err != nil {
			return err
		}
		out, err := svc.DiskJobObserve(context.Background(), cred)
		if err != nil {
			return err
		}
		return renderDiskOutput(deps.Stdout(ctx), format, out)
	}
}

func diskSampleHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		format, err := deps.OutputFormat(ctx)
		if err != nil {
			return err
		}
		svc, err := newService(deps, ctx, format)
		if err != nil {
			return err
		}
		parsed, err := capacitycli.ParseDiskSampleRequest(args)
		if err != nil {
			return err
		}
		cred, err := readPrivateLeaseFile(parsed.LeaseFile)
		if err != nil {
			return err
		}
		receipt, err := svc.DiskJobSamplePath(context.Background(), cred, parsed.Path, parsed.DockerContainer, parsed.DockerDestination, parsed.DockerUpperDir, parsed.DockerRootDir, parsed.Bytes, parsed.ExtendPeak)
		if err != nil {
			return err
		}
		return renderDiskOutput(deps.Stdout(ctx), format, receipt)
	}
}

func writePrivateLeaseFile(path string, lease diskLeaseFile) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("lease file path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.Marshal(lease)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err = f.Write(raw); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func readPrivateLeaseFile(path string) (capacityapp.DiskJobCredential, error) {
	pathInfo, err := os.Lstat(path)
	if err != nil {
		return capacityapp.DiskJobCredential{}, err
	}
	if !pathInfo.Mode().IsRegular() {
		return capacityapp.DiskJobCredential{}, fmt.Errorf("lease credential file must be a regular file with mode 0600")
	}
	f, err := os.Open(path)
	if err != nil {
		return capacityapp.DiskJobCredential{}, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return capacityapp.DiskJobCredential{}, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || !os.SameFile(pathInfo, info) {
		return capacityapp.DiskJobCredential{}, fmt.Errorf("lease credential file must be a regular file with mode 0600")
	}
	raw, err := io.ReadAll(f)
	if err != nil {
		return capacityapp.DiskJobCredential{}, err
	}
	var lease diskLeaseFile
	if err := json.Unmarshal(raw, &lease); err != nil {
		return capacityapp.DiskJobCredential{}, err
	}
	if lease.ID == "" || lease.Capability == "" || lease.ProcessID <= 0 || lease.ProcessStart == "" {
		return capacityapp.DiskJobCredential{}, errors.New("lease credential file is incomplete")
	}
	return capacityapp.DiskJobCredential{ID: lease.ID, Capability: lease.Capability, ProcessID: lease.ProcessID, ProcessStart: lease.ProcessStart}, nil
}

func renderDiskOutput(w io.Writer, format cliout.Format, out any) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteJSON(w, out) }, func(w io.Writer) error { _, err := fmt.Fprintln(w, "disk job receipt recorded"); return err })
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
