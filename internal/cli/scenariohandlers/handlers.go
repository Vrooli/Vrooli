package scenariohandlers

import (
	"io"
	"os"
	"strings"

	scenarioapp "github.com/vrooli/vrooli/internal/app/scenario"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	. "github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/scenarioexec"
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
	LifecycleRunner    func(C) (scenarioapp.PhaseRunner, error)
	EnvValidator       func(C) (scenarioapp.EnvironmentValidator, error)
	OpenURL            func(C, string) error
	LaunchDetached     func(C, ...string) error
	RunSubprocess      func(C, scenarioexec.SubprocessSpec) error
	LocateTestGenieCLI func(C) (string, error)
	LocateCompleteCLI  func(C) (string, error)
	CommandEnv         func(C) []string
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
		CommandStatus: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (StatusRequest, error) {
				return ParseStatusRequest(deps.Globals(ctx).JSON, args)
			},
			func(ctx C, req StatusRequest) (cliout.Format, StatusResponse, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", StatusResponse{}, err
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
				if err := ensureScenarioCLIs(deps, ctx, req.Name); err != nil {
					return "", lifecycle.PhaseResult{}, err
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
			func(ctx C, req RestartRequest) (cliout.Format, []LifecycleItemOutput, error) {
				if err := ensureScenarioCLIs(deps, ctx, req.Name); err != nil {
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
				items, err := service.Restart(scenarioapp.RestartRequest(req))
				return format, toCLILifecycleItems(items), err
			},
			WriteLifecycleItems,
		),
		CommandStop: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (StopRequest, error) { return ParseStopRequest(deps.Globals(ctx).JSON, args) },
			func(ctx C, req StopRequest) (cliout.Format, []LifecycleItemOutput, error) {
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
			WriteLifecycleItems,
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
		CommandTest: bindGlobal(deps.Stdout,
			func(ctx C, args []string) (TestRequest, error) {
				return ParseTestRequest(deps.Globals(ctx).JSON, deps.Globals(ctx).Verbose, args)
			},
			func(ctx C, req TestRequest) (cliout.Format, struct{}, error) {
				if err := ensureScenarioCLIs(deps, ctx, req.Name); err != nil {
					return "", struct{}{}, err
				}
				runner, err := deps.LifecycleRunner(ctx)
				if err != nil {
					return "", struct{}{}, err
				}
				service := NewRunnerService(runner)
				return TestResponseFrom(func(req TestRequest) error {
					return service.Test(scenarioapp.TestRequest(req))
				}, req)
			},
			func(w io.Writer, _ cliout.Format, _ struct{}) error { return nil },
		),
		CommandLogs: LogsHandler(deps),
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
		CommandUISmoke:      UISmokeHandler(deps),
		CommandRequirements: RequirementsHandler(deps),
		CommandTemplate:     TemplateCommandHandler(deps),
		CommandGenerate:     GenerateHandler(deps),
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

func HealFromSandboxHandlerResponse[C any](deps HandlerDeps[C], ctx C, req HealFromSandboxRequest) (cliout.Format, HealFromSandboxResponse, error) {
	home, err := deps.HomeDir(ctx)
	if err != nil {
		return "", HealFromSandboxResponse{}, err
	}
	affected, err := orchestrator.SandboxAffectedScenarios(home, req.MergedPath)
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
