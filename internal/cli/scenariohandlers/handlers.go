package scenariohandlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	scenarioapp "github.com/vrooli/vrooli/internal/app/scenario"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	. "github.com/vrooli/vrooli/internal/cli/scenariocli" //nolint:revive // scenariohandlers is a thin glue layer over scenariocli; dot-import keeps wiring readable.
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioexec"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type HandlerDeps[C any] struct {
	Stdout             func(C) io.Writer
	Stderr             func(C) io.Writer
	Root               func(C) string
	Globals            func(C) rootcli.GlobalOptions
	OutputFormat       func(C) (cliout.Format, error)
	HomeDir            func(C) (string, error)
	EnsureCLI          func(C, string) error
	ScenarioOperations func(C) (scenarioapp.ScenarioOperations, error)
	TimingStore        func(C) (*scenarioruntime.SQLiteStore, error)
	LifecycleRunner    func(C) (scenarioapp.PhaseRunner, error)
	EnvValidator       func(C) (scenarioapp.EnvironmentValidator, error)
	OpenURL            func(C, string) error
	LaunchDetached     func(C, ...string) error
	RunSubprocess      func(C, scenarioexec.SubprocessSpec) error
	// RemoteScenarioCall is the one explicit-node dispatch seam. The parser
	// owns address grammar; this seam owns command forwarding and response
	// decoding. Local commands remain on their existing service paths.
	RemoteScenarioCall func(C, string, string, string, []string, bool) ([]byte, error)
	LocateTestGenieCLI func(C) (string, error)
	// LocateBusinessHealthCLI resolves the business-health CLI, which owns
	// the contract-side requirements verbs (validate, report, lint-prd,
	// drift, phase, init, manual-log); sync/snapshot stay with test-genie.
	LocateBusinessHealthCLI func(C) (string, error)
	LocateCompleteCLI       func(C) (string, error)
	CommandEnv              func(C) []string
}

func RootHandler[C any](stdout func(C) io.Writer, lookup func(string) (rootcli.Handler[C], bool), suggest func(string) []string) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		if len(args) == 0 || (len(args) == 1 && HelpOnlyWithoutRoot(args)) {
			RenderCommandHelp(stdout(ctx))
			return nil
		}
		handler, ok := lookup(args[0])
		if !ok {
			return rootcli.NewUnknownScenarioCommandError(args[0], suggest(args[0]))
		}
		return handler(ctx, args[1:])
	}
}

func BuildHandlers[C any](deps HandlerDeps[C]) map[CommandID]rootcli.Handler[C] {
	return map[CommandID]rootcli.Handler[C]{
		CommandList: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (ListRequest, error) { return ParseListRequest(deps.Globals(ctx).JSON, args) },
			func(ctx C, req ListRequest) (cliout.Format, ListResponse, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", ListResponse{}, err
				}
				ops, err := deps.ScenarioOperations(ctx)
				if err != nil {
					return "", ListResponse{}, err
				}
				service := NewStartService(ops, func(url string) error { return deps.OpenURL(ctx, url) })
				resp, err := ListResponseFrom(format, func(req ListRequest) (scenarioapp.ListResponse, error) {
					return service.List(scenarioapp.ListRequest(req))
				}, req)
				return format, resp, err
			},
			RenderListResponse,
		),
		CommandInfo: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (InfoRequest, error) { return ParseInfoRequest(deps.Globals(ctx).JSON, args) },
			func(ctx C, req InfoRequest) (cliout.Format, InfoOutput, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", InfoOutput{}, err
				}
				ops, err := deps.ScenarioOperations(ctx)
				if err != nil {
					return "", InfoOutput{}, err
				}
				service := NewStartService(ops, func(url string) error { return deps.OpenURL(ctx, url) })
				resp, err := InfoResponseFrom(format, func(req InfoRequest) (scenarioapp.InfoOutput, error) {
					return service.Info(scenarioapp.InfoRequest(req))
				}, req)
				return format, resp, err
			},
			RenderInfoResponse,
		),
		CommandTimings: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (TimingsRequest, error) {
				return ParseTimingsRequest(deps.Globals(ctx).JSON, args)
			},
			func(ctx C, req TimingsRequest) (cliout.Format, TimingsResponse, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", TimingsResponse{}, err
				}
				if deps.TimingStore == nil {
					return "", TimingsResponse{}, fmt.Errorf("scenario timings is not configured")
				}
				store, err := deps.TimingStore(ctx)
				if err != nil {
					return "", TimingsResponse{}, err
				}
				defer store.Close()
				rows, err := store.StartTimingSummaries(context.Background(), req.Scenario)
				return format, TimingsResponse{Rows: rows}, err
			},
			RenderTimingsResponse,
		),
		CommandStatus: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (StatusRequest, error) {
				return ParseStatusRequest(deps.Globals(ctx).JSON, args)
			},
			func(ctx C, req StatusRequest) (cliout.Format, StatusResponse, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", StatusResponse{}, err
				}
				if node, remoteScenario, remote := remoteScenarioAddress(req.Name); remote {
					payload, callErr := callRemoteScenario(deps, ctx, node, remoteScenario, "scenario status", nil, req.JSON)
					if callErr != nil {
						emitLifecycleFailure(deps, ctx, "status", []string{req.Name}, callErr)
						return format, StatusResponse{}, silentLifecycleError{inner: callErr}
					}
					return format, StatusResponse{Raw: payload}, callErr
				}
				ops, err := deps.ScenarioOperations(ctx)
				if err != nil {
					return "", StatusResponse{}, err
				}
				service := NewStartService(ops, func(url string) error { return deps.OpenURL(ctx, url) })
				resp, err := StatusResponseFrom(format, func(req StatusRequest) (scenarioapp.StatusResponse, error) {
					return service.Status(scenarioapp.StatusRequest(req))
				}, req)
				return format, resp, err
			},
			RenderStatusResponse,
		),
		CommandWait: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (WaitRequest, error) {
				return ParseWaitRequest(deps.Globals(ctx).JSON, args)
			},
			func(ctx C, req WaitRequest) (cliout.Format, WaitResponse, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", WaitResponse{}, err
				}
				if node, remoteScenario, remote := remoteScenarioAddress(req.Name); remote {
					args := []string{}
					if req.TimeoutSeconds > 0 {
						args = []string{"--timeout", strconv.Itoa(req.TimeoutSeconds)}
					}
					payload, callErr := callRemoteScenario(deps, ctx, node, remoteScenario, "scenario wait", args, true)
					if callErr != nil {
						emitLifecycleFailure(deps, ctx, "wait", []string{req.Name}, callErr)
						return format, WaitResponse{}, silentLifecycleError{inner: callErr}
					}
					resp, decodeErr := decodeRemoteWait(payload)
					if decodeErr != nil {
						emitLifecycleFailure(deps, ctx, "wait", []string{req.Name}, decodeErr)
						return format, WaitResponse{}, silentLifecycleError{inner: decodeErr}
					}
					return format, resp, nil
				}
				runner, err := deps.LifecycleRunner(ctx)
				if err != nil {
					return "", WaitResponse{}, err
				}
				stderr := deps.Stderr(ctx)
				// Inside an agent-manager run, park instead of blocking:
				// agent-manager performs the wait via its lifecycle Waiter and
				// wakes the run with the JSON verdict (zero tokens parked).
				if message, parked := ParkScenarioWait(stderr, req.Name); parked {
					return format, WaitResponse{Scenario: req.Name, ParkedMessage: message}, nil
				}
				WarnIfEagerScenarioWait(stderr, req.Name, time.Now())
				service := NewRunnerService(runner)
				appReq := scenarioapp.WaitRequest{Name: req.Name, TimeoutSeconds: req.TimeoutSeconds}
				if format != cliout.FormatJSON {
					// Human heartbeat: step/dependency transitions of the
					// awaited operation. JSON mode keeps stdout machine-pure.
					out := deps.Stdout(ctx)
					appReq.OnTransition = func(view lifecycle.StartOperationView) {
						if line := view.TransitionLine(); line != "" {
							fmt.Fprintln(out, line)
						}
					}
				}

				// Ctrl-C detaches: the awaited start (owned by another
				// process) is unaffected; print re-attach guidance, exit 0.
				type waitResult struct {
					resp scenarioapp.WaitResponse
					err  error
				}
				resultCh := make(chan waitResult, 1)
				go func() {
					resp, err := service.Wait(appReq)
					resultCh <- waitResult{resp, err}
				}()
				sigCh := make(chan os.Signal, 1)
				signal.Notify(sigCh, os.Interrupt)
				defer signal.Stop(sigCh)
				var resp scenarioapp.WaitResponse
				select {
				case r := <-resultCh:
					if r.err != nil {
						return "", WaitResponse{}, r.err
					}
					resp = r.resp
				case <-sigCh:
					fmt.Fprintf(stderr, "detached from scenario wait; the start (if any) continues — re-attach with `vrooli scenario wait %s --json`\n", req.Name)
					return format, WaitResponse{Scenario: req.Name, Verdict: WaitVerdictDetached, Source: "detached"}, nil
				}
				if resp.Verdict == lifecycle.WaitVerdictTimeout {
					// Ceiling elapsed: this wait detached; the start continues.
					fmt.Fprintf(stderr, "scenario wait: timeout ceiling elapsed after %ds; the start is still running — re-attach with `vrooli scenario wait %s --json` (size --timeout as ETA + 75%% buffer)\n", resp.WaitedSeconds, req.Name)
				} else {
					ClearScenarioWaitAttempt(req.Name)
				}
				return format, WaitResponse{
					Success:       resp.Success,
					Scenario:      resp.Scenario,
					Verdict:       resp.Verdict,
					ExitCode:      resp.ExitCode,
					Source:        resp.Source,
					WaitedSeconds: resp.WaitedSeconds,
					Error:         resp.Error,
					Operation:     resp.Operation,
				}, nil
			},
			RenderWaitResponse,
		),
		CommandValidateEnv: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (ValidateEnvRequest, error) {
				return ParseValidateEnvRequest(deps.Globals(ctx).JSON, args)
			},
			func(ctx C, req ValidateEnvRequest) (cliout.Format, ValidateEnvResponse, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", ValidateEnvResponse{}, err
				}
				validator, err := deps.EnvValidator(ctx)
				if err != nil {
					return "", ValidateEnvResponse{}, err
				}
				service := NewValidatorService(validator)
				resp, err := ValidateEnvResponseFrom(func(req ValidateEnvRequest) (scenarioapp.ValidateEnvResponse, error) {
					return service.ValidateEnv(scenarioapp.ValidateEnvRequest(req))
				}, format, req)
				return format, resp, err
			},
			RenderValidateEnvResponse,
		),
		CommandFreshness: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (FreshnessRequest, error) {
				return ParseFreshnessRequest(deps.Globals(ctx).JSON, args)
			},
			func(ctx C, req FreshnessRequest) (cliout.Format, FreshnessResponse, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", FreshnessResponse{}, err
				}
				runner, err := deps.LifecycleRunner(ctx)
				if err != nil {
					return "", FreshnessResponse{}, err
				}
				report, err := NewRunnerService(runner).Freshness(scenarioapp.FreshnessRequest{Name: req.Name, Path: req.Path, JSON: req.JSON})
				if err != nil {
					return format, FreshnessResponse{}, err
				}
				return format, FreshnessResponse{Report: report, Explain: req.Explain}, nil
			},
			RenderFreshnessResponse,
		),
		CommandRun: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (StartRequest, error) {
				return ParseStartRequest(deps.Globals(ctx).JSON, args)
			},
			func(ctx C, req StartRequest) (cliout.Format, []LifecycleItemOutput, error) {
				if err := ensureScenarioCLIs(deps, ctx, req.Names...); err != nil {
					return "", nil, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", nil, err
				}
				ops, err := deps.ScenarioOperations(ctx)
				if err != nil {
					return "", nil, err
				}
				service := NewStartService(ops, func(url string) error { return deps.OpenURL(ctx, url) })
				items, err := service.Start(scenarioapp.StartRequest(req))
				return format, toCLILifecycleItems(items), err
			},
			WriteLifecycleItems,
		),
		CommandStart: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (StartRequest, error) {
				return ParseStartRequest(deps.Globals(ctx).JSON, args)
			},
			withLifecycleFailureBlock(
				deps,
				"start",
				func(req StartRequest) []string { return append([]string(nil), req.Names...) },
				func(ctx C, req StartRequest) (cliout.Format, []LifecycleItemOutput, error) {
					if node, remoteScenario, remote := remoteStartAddress(req.Names); remote {
						if len(req.Names) != 1 {
							return "", nil, fmt.Errorf("remote scenario start accepts exactly one explicitly addressed scenario")
						}
						format, callErr := deps.OutputFormat(ctx)
						if callErr != nil {
							return "", nil, callErr
						}
						payload, callErr := callRemoteScenario(deps, ctx, node, remoteScenario, "scenario start", remoteStartArgs(req.Options, req.OpenAfter, req.TimeoutSeconds), true)
						if callErr != nil {
							return format, nil, callErr
						}
						items, decodeErr := decodeRemoteLifecycle(payload)
						return format, items, decodeErr
					}
					if req.Options.CustomPath == "" {
						if err := ensureScenarioCLIs(deps, ctx, req.Names...); err != nil {
							return "", nil, err
						}
					}
					format, err := deps.OutputFormat(ctx)
					if err != nil {
						return "", nil, err
					}
					ops, err := deps.ScenarioOperations(ctx)
					if err != nil {
						return "", nil, err
					}
					service := NewStartService(ops, func(url string) error { return deps.OpenURL(ctx, url) })
					items, err := runWithStartCeiling(req.TimeoutSeconds, deps.Stderr(ctx), strings.Join(req.Names, " "), func(operationCtx context.Context) ([]scenarioapp.LifecycleItemOutput, error) {
						req.Options.Context = operationCtx
						return service.Start(scenarioapp.StartRequest(req))
					})
					return format, toCLILifecycleItems(items), err
				},
			),
			WriteLifecycleItems,
		),
		CommandStartAll: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (StartAllRequest, error) {
				return ParseStartAllRequest(deps.Globals(ctx).JSON, args)
			},
			func(ctx C, req StartAllRequest) (cliout.Format, BatchResponse, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", BatchResponse{}, err
				}
				ops, err := deps.ScenarioOperations(ctx)
				if err != nil {
					return "", BatchResponse{}, err
				}
				service := NewStartService(ops, func(url string) error { return deps.OpenURL(ctx, url) })
				resp, err := BatchStartResponseFrom(format, service.StartAll)
				return format, resp, err
			},
			WriteBatchReport,
		),
		CommandSetup: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (SetupRequest, error) {
				return ParseSetupRequest(deps.Globals(ctx).JSON, args)
			},
			func(ctx C, req SetupRequest) (cliout.Format, lifecycle.PhaseResult, error) {
				if req.Opts.CustomPath == "" {
					if err := ensureScenarioCLIs(deps, ctx, req.Name); err != nil {
						return "", lifecycle.PhaseResult{}, err
					}
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", lifecycle.PhaseResult{}, err
				}
				runner, err := deps.LifecycleRunner(ctx)
				if err != nil {
					return "", lifecycle.PhaseResult{}, err
				}
				service := NewRunnerService(runner)
				result, err := service.Setup(scenarioapp.SetupRequest(req))
				return format, result, err
			},
			RenderSetupPhaseResult,
		),
		CommandRestart: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (RestartRequest, error) {
				return ParseRestartRequest(deps.Globals(ctx).JSON, args)
			},
			withLifecycleFailureBlock(
				deps,
				"restart",
				func(req RestartRequest) []string { return []string{req.Name} },
				func(ctx C, req RestartRequest) (cliout.Format, []LifecycleItemOutput, error) {
					if node, remoteScenario, remote := remoteScenarioAddress(req.Name); remote {
						format, callErr := deps.OutputFormat(ctx)
						if callErr != nil {
							return "", nil, callErr
						}
						payload, callErr := callRemoteScenario(deps, ctx, node, remoteScenario, "scenario restart", remoteStartArgs(req.Options, req.OpenAfter, req.TimeoutSeconds), true)
						if callErr != nil {
							return format, nil, callErr
						}
						items, decodeErr := decodeRemoteLifecycle(payload)
						return format, items, decodeErr
					}
					if req.Options.CustomPath == "" {
						if err := ensureScenarioCLIs(deps, ctx, req.Name); err != nil {
							return "", nil, err
						}
					}
					format, err := deps.OutputFormat(ctx)
					if err != nil {
						return "", nil, err
					}
					ops, err := deps.ScenarioOperations(ctx)
					if err != nil {
						return "", nil, err
					}
					service := NewStartService(ops, func(url string) error { return deps.OpenURL(ctx, url) })
					items, err := runWithStartCeiling(req.TimeoutSeconds, deps.Stderr(ctx), req.Name, func(operationCtx context.Context) ([]scenarioapp.LifecycleItemOutput, error) {
						req.Options.Context = operationCtx
						return service.Restart(scenarioapp.RestartRequest(req))
					})
					return format, toCLILifecycleItems(items), err
				},
			),
			WriteLifecycleItems,
		),
		CommandStop: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (StopRequest, error) { return ParseStopRequest(deps.Globals(ctx).JSON, args) },
			withLifecycleFailureBlock(
				deps,
				"stop",
				func(req StopRequest) []string { return []string{req.Name} },
				func(ctx C, req StopRequest) (cliout.Format, []LifecycleItemOutput, error) {
					if node, remoteScenario, remote := remoteScenarioAddress(req.Name); remote {
						format, callErr := deps.OutputFormat(ctx)
						if callErr != nil {
							return "", nil, callErr
						}
						payload, callErr := callRemoteScenario(deps, ctx, node, remoteScenario, "scenario stop", nil, true)
						if callErr != nil {
							return format, nil, callErr
						}
						items, decodeErr := decodeRemoteLifecycle(payload)
						return format, items, decodeErr
					}
					format, err := deps.OutputFormat(ctx)
					if err != nil {
						return "", nil, err
					}
					runner, err := deps.LifecycleRunner(ctx)
					if err != nil {
						return "", nil, err
					}
					service := NewRunnerService(runner)
					items, err := service.Stop(scenarioapp.StopRequest(req))
					return format, toCLILifecycleItems(items), err
				},
			),
			WriteLifecycleItems,
		),
		CommandDelete: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (DeleteRequest, error) {
				return ParseDeleteRequest(deps.Globals(ctx).JSON, args)
			},
			func(ctx C, req DeleteRequest) (cliout.Format, DeleteResponse, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", DeleteResponse{}, err
				}
				root := filepath.Clean(deps.Root(ctx))
				canonical := filepath.Join(root, "scenarios", req.Name)
				resolved, redirected := scenariomodel.ResolveScenarioPath(root, req.Name, scenariomodel.SandboxEnv{})
				if redirected || filepath.Clean(resolved) != canonical {
					return format, DeleteResponse{}, fmt.Errorf("scenario delete refuses redirected scenario path %q", resolved)
				}
				if _, statErr := os.Stat(canonical); statErr == nil {
					runner, runnerErr := deps.LifecycleRunner(ctx)
					if runnerErr != nil {
						return format, DeleteResponse{}, runnerErr
					}
					if _, stopErr := NewRunnerService(runner).Stop(scenarioapp.StopRequest{Name: req.Name}); stopErr != nil {
						return format, DeleteResponse{}, stopErr
					}
				}
				if err := os.RemoveAll(canonical); err != nil {
					return format, DeleteResponse{}, fmt.Errorf("delete scenario source: %w", err)
				}
				home, err := deps.HomeDir(ctx)
				if err != nil {
					return format, DeleteResponse{}, err
				}
				manager, err := cliinstall.NewManager(root, home)
				if err != nil {
					return format, DeleteResponse{}, err
				}
				removal, err := manager.RemoveScenarioCLIReport(req.Name)
				if err != nil {
					return format, DeleteResponse{}, err
				}
				return format, DeleteResponse{Name: req.Name, ScenarioPath: canonical, RemovedArtifacts: removal.Removed, SkippedArtifacts: removal.Skipped}, nil
			},
			RenderDeleteResponse,
		),
		CommandStopAll: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (StopAllRequest, error) {
				return ParseStopAllRequest(deps.Globals(ctx).JSON, args)
			},
			func(ctx C, req StopAllRequest) (cliout.Format, BatchResponse, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", BatchResponse{}, err
				}
				ops, err := deps.ScenarioOperations(ctx)
				if err != nil {
					return "", BatchResponse{}, err
				}
				service := NewStartService(ops, func(url string) error { return deps.OpenURL(ctx, url) })
				resp, err := BatchStopResponseFrom(format, service.StopAll)
				return format, resp, err
			},
			WriteBatchReport,
		),
		CommandTest: TestHandler(deps),
		CommandLogs: LogsHandler(deps),
		CommandScreenshot: func(ctx C, args []string) error {
			req, err := ParseScreenshotRequest(deps.Globals(ctx).JSON, args)
			if err != nil {
				return err
			}
			var command string
			var commandArgs []string
			switch runtime.GOOS {
			case "darwin":
				command, commandArgs = "screencapture", []string{"-x", req.Output}
			case "linux":
				command, commandArgs = "gnome-screenshot", []string{"-f", req.Output}
			default:
				return fmt.Errorf("scenario screenshot: unsupported operating system %q", runtime.GOOS)
			}
			// Capture stderr. Both backends explain a refusal there and say
			// nothing on stdout, so discarding it reduces every failure to a
			// bare "exit status 1" — including the two that actually happen:
			// macOS denying screencapture without a Screen Recording grant or
			// outside a GUI login session, and gnome-screenshot with no
			// reachable display.
			var stderr bytes.Buffer
			capture := exec.Command(command, commandArgs...)
			capture.Stderr = &stderr
			if err := capture.Run(); err != nil {
				if detail := strings.TrimSpace(stderr.String()); detail != "" {
					return fmt.Errorf("scenario screenshot: %s failed: %w: %s", command, err, detail)
				}
				return fmt.Errorf("scenario screenshot: %s failed with no diagnostic output: %w", command, err)
			}
			if _, err := os.Stat(req.Output); err != nil {
				return fmt.Errorf("scenario screenshot: capture did not create %q: %w", req.Output, err)
			}
			if req.JSON {
				_, err = fmt.Fprintf(deps.Stdout(ctx), "{\"path\":%s}\n", strconv.Quote(req.Output))
				return err
			}
			_, err = fmt.Fprintf(deps.Stdout(ctx), "%s\n", req.Output)
			return err
		},
		CommandOpen: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (OpenRequest, error) { return ParseOpenRequest(deps.Globals(ctx).JSON, args) },
			func(ctx C, req OpenRequest) (cliout.Format, OpenOutput, error) {
				ops, err := deps.ScenarioOperations(ctx)
				if err != nil {
					return "", OpenOutput{}, err
				}
				service := NewStartService(ops, func(url string) error { return deps.OpenURL(ctx, url) })
				return OpenResponseFrom(func(req OpenRequest) (scenarioapp.OpenOutput, error) {
					return service.Open(scenarioapp.OpenRequest(req))
				}, req)
			},
			func(w io.Writer, _ cliout.Format, resp OpenOutput) error { return RenderOpenResponse(w, resp) },
		),
		CommandPort: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (PortRequest, error) { return ParsePortRequest(deps.Globals(ctx).JSON, args) },
			func(ctx C, req PortRequest) (cliout.Format, PortResponse, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", PortResponse{}, err
				}
				if node, remoteScenario, remote := remoteScenarioAddress(req.ScenarioName); remote {
					args := []string{}
					if req.PortName != "" {
						args = append(args, req.PortName)
					}
					if req.Path != "" {
						args = append(args, "--path", req.Path)
					}
					payload, callErr := callRemoteScenario(deps, ctx, node, remoteScenario, "scenario port", args, true)
					if callErr != nil {
						emitLifecycleFailure(deps, ctx, "query port", []string{req.ScenarioName}, callErr)
						return format, PortResponse{}, silentLifecycleError{inner: callErr}
					}
					resp, decodeErr := decodeRemotePort(payload, req.PortName)
					if decodeErr != nil {
						emitLifecycleFailure(deps, ctx, "query port", []string{req.ScenarioName}, decodeErr)
						return format, PortResponse{}, silentLifecycleError{inner: decodeErr}
					}
					return format, resp, nil
				}
				ops, err := deps.ScenarioOperations(ctx)
				if err != nil {
					return "", PortResponse{}, err
				}
				service := NewStartService(ops, func(url string) error { return deps.OpenURL(ctx, url) })
				resp, err := PortResponseFrom(format, func(req PortRequest) (scenarioapp.PortResponse, error) {
					return service.Port(scenarioapp.PortRequest(req))
				}, req)
				return format, resp, err
			},
			RenderPortResponse,
		),
		CommandRequirements: RequirementsHandler(deps),
		CommandCompleteness: CompletenessHandler(deps),
		CommandHealFromSandbox: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (HealFromSandboxRequest, error) {
				return ParseHealFromSandboxRequest(strings.TrimSpace(os.Getenv("SANDBOX_MERGED_DIR")), args)
			},
			func(ctx C, req HealFromSandboxRequest) (cliout.Format, HealFromSandboxResponse, error) {
				return HealFromSandboxHandlerResponse(deps, ctx, req)
			},
			RenderHealFromSandboxResponse,
		),
	}
}

func ensureScenarioCLIs[C any](deps HandlerDeps[C], ctx C, names ...string) error {
	if deps.EnsureCLI == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		if err := deps.EnsureCLI(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

func bindGlobal[C any, Req any, Resp any](
	stdout func(C) io.Writer,
	parse func(C, []string) (Req, error),
	run func(C, Req) (cliout.Format, Resp, error),
	render func(io.Writer, cliout.Format, Resp) error,
) rootcli.Handler[C] {
	return rootcli.BindGlobalCommand(stdout, parse, run, render)
}

// remoteScenarioAddress recognizes only an explicit node/name address. Local
// scenario names deliberately remain on the existing control-plane path.
func remoteScenarioAddress(name string) (node, scenario string, remote bool) {
	node, scenario, variant, err := cliutil.SplitAddress(strings.TrimSpace(name))
	if err != nil || strings.TrimSpace(node) == "" {
		return "", "", false
	}
	scenario = strings.TrimSpace(scenario)
	if variant != "" {
		scenario += "@" + variant
	}
	return strings.TrimSpace(node), scenario, true
}

func remoteStartAddress(names []string) (node, scenario string, remote bool) {
	if len(names) == 0 {
		return "", "", false
	}
	return remoteScenarioAddress(names[0])
}

func callRemoteScenario[C any](deps HandlerDeps[C], ctx C, node, scenario, command string, args []string, jsonOutput bool) ([]byte, error) {
	if deps.RemoteScenarioCall == nil {
		return nil, fmt.Errorf("remote scenario %s is not configured", command)
	}
	payload, err := deps.RemoteScenarioCall(ctx, node, scenario, command, args, jsonOutput)
	if err != nil {
		return nil, fmt.Errorf("remote %s %s: %w", command, scenario, err)
	}
	if len(bytes.TrimSpace(payload)) == 0 {
		return nil, fmt.Errorf("remote %s %s returned an empty response", command, scenario)
	}
	return payload, nil
}

func remoteStartArgs(options lifecycle.StartOptions, openAfter bool, timeoutSeconds int) []string {
	args := []string{}
	if options.CustomPath != "" {
		args = append(args, "--path", options.CustomPath)
	}
	if options.BestEffort {
		args = append(args, "--best-effort")
	}
	if options.CleanStale {
		args = append(args, "--clean-stale")
	}
	if options.ForceSetup {
		args = append(args, "--force")
	}
	if openAfter {
		args = append(args, "--open")
	}
	if timeoutSeconds > 0 {
		args = append(args, "--timeout", strconv.Itoa(timeoutSeconds))
	}
	return args
}

func decodeRemoteLifecycle(payload []byte) ([]LifecycleItemOutput, error) {
	var response cliv1.ScenarioLifecycleResponse
	if err := protojson.Unmarshal(payload, &response); err != nil {
		return nil, fmt.Errorf("decode remote lifecycle response: %w", err)
	}
	items := make([]LifecycleItemOutput, 0, len(response.GetScenarios()))
	for _, item := range response.GetScenarios() {
		if item == nil {
			continue
		}
		ports := map[string]int{}
		for key, port := range item.GetPorts() {
			ports[key] = int(port)
		}
		endpoints := make([]EndpointOutput, 0, len(item.GetEndpoints()))
		for _, endpoint := range item.GetEndpoints() {
			if endpoint == nil {
				continue
			}
			endpoints = append(endpoints, EndpointOutput{
				Name: endpoint.GetName(), Key: endpoint.GetKey(), Description: endpoint.GetDescription(),
				Port: int(endpoint.GetPort()), URL: endpoint.GetUrl(),
			})
		}
		items = append(items, LifecycleItemOutput{
			Name: item.GetName(), Status: item.GetStatus(), Health: item.GetHealth(), Ports: ports,
			Endpoints: endpoints, FailedDependencies: CopyStrings(item.GetFailedDependencies()),
			FailedResources: CopyStrings(item.GetFailedResources()), Verdict: item.GetVerdict(),
			Operation: remoteOperationView(item.GetOperation()),
		})
	}
	return items, nil
}

func decodeRemoteWait(payload []byte) (WaitResponse, error) {
	var response cliv1.ScenarioWaitResponse
	if err := protojson.Unmarshal(payload, &response); err != nil {
		return WaitResponse{}, fmt.Errorf("decode remote wait response: %w", err)
	}
	return WaitResponse{
		Success: response.GetSuccess(), Scenario: response.GetScenario(), Verdict: response.GetVerdict(),
		ExitCode: int(response.GetExitCode()), Source: response.GetSource(),
		WaitedSeconds: int(response.GetWaitedSeconds()), Operation: remoteOperationView(response.GetOperation()),
	}, nil
}

func decodeRemotePort(payload []byte, portName string) (PortResponse, error) {
	if strings.TrimSpace(portName) == "" {
		var response cliv1.ScenarioPortList
		if err := protojson.Unmarshal(payload, &response); err != nil {
			return PortResponse{}, fmt.Errorf("decode remote port list response: %w", err)
		}
		ports := make([]ListPortOutput, 0, len(response.GetPorts()))
		for _, port := range response.GetPorts() {
			if port == nil {
				continue
			}
			ports = append(ports, ListPortOutput{Key: port.GetKey(), Step: port.GetStep(), Port: int(port.GetPort()), ListenerStatus: port.GetListenerStatus()})
		}
		return PortResponse{List: &PortListOutput{Success: response.GetSuccess(), Scenario: response.GetScenario(), Ports: ports, Metadata: intMap(response.GetMetadata()), Error: response.GetError()}}, nil
	}
	var response cliv1.ScenarioPortSingle
	if err := protojson.Unmarshal(payload, &response); err != nil {
		return PortResponse{}, fmt.Errorf("decode remote port response: %w", err)
	}
	return PortResponse{Single: &PortSingleOutput{Success: response.GetSuccess(), Scenario: response.GetScenario(), PortName: response.GetPortName(), Step: response.GetStep(), Port: int(response.GetPort()), Error: response.GetError()}}, nil
}

func intMap(values map[string]int32) map[string]int {
	out := make(map[string]int, len(values))
	for key, value := range values {
		out[key] = int(value)
	}
	return out
}

func remoteOperationView(operation *cliv1.ScenarioStartOperation) *lifecycle.StartOperationView {
	if operation == nil {
		return nil
	}
	view := &lifecycle.StartOperationView{
		OperationID: operation.GetOperationId(), Scenario: operation.GetScenario(), Variant: operation.GetVariant(),
		Operation: operation.GetOperation(), Status: operation.GetStatus(), Verdict: operation.GetVerdict(),
		Error: operation.GetError(), CurrentStep: operation.GetCurrentStep(), DependencyCurrent: operation.GetDependencyCurrent(),
		DependencyIndex: int(operation.GetDependencyIndex()), DependencyTotal: int(operation.GetDependencyTotal()),
		ElapsedSeconds: int(operation.GetElapsedSeconds()), ETAKnown: operation.GetEtaKnown(), ETASeconds: int(operation.GetEtaSeconds()),
		RecommendedNextCheckSeconds: int(operation.GetRecommendedNextCheckSeconds()), InitiatorPID: int(operation.GetInitiatorPid()),
	}
	view.StartedAt, _ = time.Parse(time.RFC3339Nano, operation.GetStartedAt())
	if finished, err := time.Parse(time.RFC3339Nano, operation.GetFinishedAt()); err == nil && operation.GetFinishedAt() != "" {
		view.FinishedAt = &finished
	}
	view.Steps = make([]scenarioruntime.StartOperationStep, 0, len(operation.GetSteps()))
	for _, step := range operation.GetSteps() {
		if step == nil {
			continue
		}
		converted := scenarioruntime.StartOperationStep{Name: step.GetName(), Status: step.GetStatus()}
		converted.StartedAt, _ = time.Parse(time.RFC3339Nano, step.GetStartedAt())
		if ended, err := time.Parse(time.RFC3339Nano, step.GetEndedAt()); err == nil && step.GetEndedAt() != "" {
			converted.EndedAt = &ended
		}
		view.Steps = append(view.Steps, converted)
	}
	return view
}

func HealFromSandboxHandlerResponse[C any](deps HandlerDeps[C], ctx C, req HealFromSandboxRequest) (cliout.Format, HealFromSandboxResponse, error) {
	home, err := deps.HomeDir(ctx)
	if err != nil {
		return "", HealFromSandboxResponse{}, err
	}
	root := deps.Root(ctx)
	svc := orchestrator.New(root, home, deps.Stdout(ctx), deps.Stderr(ctx))
	affected, err := svc.SandboxAffectedScenarios(context.Background(), req.MergedPath)
	if err != nil {
		return "", HealFromSandboxResponse{}, err
	}
	resp := scenarioapp.HealFromSandboxResponse{Affected: append([]string(nil), affected...), DryRun: req.DryRun}
	if len(affected) == 0 || req.DryRun {
		return HealFromSandboxResponseFrom(func(HealFromSandboxRequest) (scenarioapp.HealFromSandboxResponse, error) { return resp, nil }, req)
	}
	runner, err := deps.LifecycleRunner(ctx)
	if err != nil {
		return "", HealFromSandboxResponse{}, err
	}
	for _, name := range affected {
		if stopErr := runner.Stop(name, lifecycle.StopOptions{}); stopErr != nil {
			_, _ = io.WriteString(deps.Stderr(ctx), "heal-from-sandbox: stop "+name+" failed: "+stopErr.Error()+"\n")
		}
	}
	for _, name := range affected {
		if startErr := deps.LaunchDetached(ctx, "start", name); startErr != nil {
			return "", HealFromSandboxResponse{}, startErr
		}
		resp.StoppedCount++
	}
	return HealFromSandboxResponseFrom(func(HealFromSandboxRequest) (scenarioapp.HealFromSandboxResponse, error) { return resp, nil }, req)
}
