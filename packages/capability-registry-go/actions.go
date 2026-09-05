package capabilityregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CommandRunner is the narrow execution seam for lifecycle actions. The
// service never accepts an arbitrary command: it derives the argv from a
// declared scenario slug and a fixed lifecycle verb.
type CommandRunner interface {
	Run(context.Context, string, ...string) (CommandResult, error)
}

type CommandResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

type ExecCommandRunner struct{}

func (ExecCommandRunner) Run(ctx context.Context, name string, args ...string) (CommandResult, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	stdout, err := cmd.Output()
	result := CommandResult{Stdout: stdout}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if err == nil {
		return result, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.Stderr = exitErr.Stderr
		result.ExitCode = exitErr.ExitCode()
		return result, err
	}
	result.ExitCode = -1
	return result, err
}

type LifecycleActionRequest struct {
	IntegrationID string
	ActionKind    ActionKind
}

type LifecycleActionResult struct {
	Success       bool
	Status        string
	Message       string
	IntegrationID string
	ActionKind    ActionKind
}

// LifecycleActionService performs only explicit, allowlisted scenario start
// and restart actions. It waits once through the lifecycle CLI before
// returning, so callers receive post-action evidence rather than a launch
// acknowledgement.
type LifecycleActionService struct {
	Defs    []Def
	Runner  CommandRunner
	CLIPath string
	Timeout time.Duration
}

func (s LifecycleActionService) Run(ctx context.Context, req LifecycleActionRequest) (LifecycleActionResult, error) {
	def, ok := definitionByID(s.Defs, req.IntegrationID)
	if !ok {
		return LifecycleActionResult{}, fmt.Errorf("integration %q is not declared", req.IntegrationID)
	}
	if def.DependencyKind != DependencyScenario || def.DependencySlug == "" {
		if req.ActionKind == ActionKindOperatorCommand {
			return s.runOperatorCommand(ctx, req, def)
		}
		return LifecycleActionResult{}, fmt.Errorf("integration %q is not a scenario dependency", req.IntegrationID)
	}
	if def.ActionKind != ActionKindNone && def.ActionKind != req.ActionKind {
		return LifecycleActionResult{}, fmt.Errorf("action %q is not eligible for integration %q", req.ActionKind, req.IntegrationID)
	}
	if req.ActionKind == ActionKindOperatorCommand && def.ActionKind != ActionKindOperatorCommand {
		return LifecycleActionResult{}, fmt.Errorf("integration %q has no operator action", req.IntegrationID)
	}
	verb := ""
	switch req.ActionKind {
	case ActionKindScenarioStart:
		verb = "start"
	case ActionKindScenarioRestart:
		verb = "restart"
	default:
		return LifecycleActionResult{}, fmt.Errorf("unsupported lifecycle action %q", req.ActionKind)
	}
	cli := strings.TrimSpace(s.CLIPath)
	if cli == "" {
		cli = "vrooli"
	}
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = 180 * time.Second
	}
	runner := s.Runner
	if runner == nil {
		runner = ExecCommandRunner{}
	}
	actionCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start, err := runner.Run(actionCtx, cli, "scenario", verb, def.DependencySlug, "--json", "--timeout", fmt.Sprintf("%.0f", timeout.Seconds()))
	if err != nil {
		return lifecycleActionFailure(req, "lifecycle "+verb+" failed", start, err), nil
	}
	wait, err := runner.Run(actionCtx, cli, "scenario", "wait", def.DependencySlug, "--json", "--timeout", fmt.Sprintf("%.0f", timeout.Seconds()))
	if err != nil {
		return lifecycleActionFailure(req, "lifecycle wait failed", wait, err), nil
	}
	return lifecycleActionSuccess(req, wait), nil
}

func (s LifecycleActionService) runOperatorCommand(ctx context.Context, req LifecycleActionRequest, def Def) (LifecycleActionResult, error) {
	if def.OperatorCommand == "" {
		return LifecycleActionResult{}, fmt.Errorf("integration %q has no operator action", req.IntegrationID)
	}
	cli := strings.TrimSpace(s.CLIPath)
	if cli == "" {
		cli = "vrooli"
	}
	args := strings.Fields(def.OperatorCommand)
	if len(args) > 0 && commandBaseName(args[0]) == commandBaseName(cli) {
		args = args[1:]
	}
	if len(args) == 0 {
		return LifecycleActionResult{}, fmt.Errorf("integration %q has an empty operator action", req.IntegrationID)
	}
	runner := s.Runner
	if runner == nil {
		runner = ExecCommandRunner{}
	}
	actionCtx, cancel := context.WithTimeout(ctx, actionTimeout(s.Timeout))
	defer cancel()
	result, err := runner.Run(actionCtx, cli, args...)
	if err != nil {
		return lifecycleActionFailure(req, "operator action failed", result, err), nil
	}
	return lifecycleActionSuccess(req, result), nil
}

func commandBaseName(value string) string {
	value = strings.TrimSpace(value)
	if index := strings.LastIndexAny(value, `/\\`); index >= 0 {
		value = value[index+1:]
	}
	if strings.EqualFold(filepath.Ext(value), ".exe") {
		value = value[:len(value)-len(".exe")]
	}
	return value
}

func actionTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return 180 * time.Second
	}
	return timeout
}

func definitionByID(defs []Def, id string) (Def, bool) {
	for _, def := range defs {
		if def.ID == id {
			return def, true
		}
	}
	return Def{}, false
}

type lifecycleWaitPayload struct {
	Success  bool   `json:"success"`
	Verdict  string `json:"verdict"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error"`
}

func lifecycleActionSuccess(req LifecycleActionRequest, result CommandResult) LifecycleActionResult {
	payload := lifecycleWaitPayload{Success: result.ExitCode == 0}
	_ = json.Unmarshal(result.Stdout, &payload)
	status := payload.Verdict
	if status == "" {
		status = "completed"
	}
	if payload.ExitCode != 0 {
		payload.Success = false
	}
	return LifecycleActionResult{Success: payload.Success, Status: status, Message: payload.Error, IntegrationID: req.IntegrationID, ActionKind: req.ActionKind}
}

func lifecycleActionFailure(req LifecycleActionRequest, prefix string, result CommandResult, err error) LifecycleActionResult {
	detail := strings.TrimSpace(string(result.Stderr))
	if detail == "" {
		detail = strings.TrimSpace(string(result.Stdout))
	}
	if detail == "" && err != nil {
		detail = err.Error()
	}
	return LifecycleActionResult{Success: false, Status: "failed", Message: strings.TrimSpace(prefix + ": " + detail), IntegrationID: req.IntegrationID, ActionKind: req.ActionKind}
}
