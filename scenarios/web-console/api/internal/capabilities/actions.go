package capabilities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultLifecycleActionTimeout = 180 * time.Second

type LifecycleActionRequest struct {
	CapabilityID string
	ActionKind   ActionKind
}

type LifecycleActionResult struct {
	Success      bool
	Status       string
	Message      string
	OperationID  string
	CapabilityID string
	ActionKind   ActionKind
}

type CommandResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (CommandResult, error)
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

type LifecycleActionService struct {
	Defs    []Def
	Runner  CommandRunner
	CLIPath string
	Timeout time.Duration
}

func (s LifecycleActionService) Run(ctx context.Context, req LifecycleActionRequest) (LifecycleActionResult, error) {
	def, ok := s.definition(req.CapabilityID)
	if !ok {
		return LifecycleActionResult{}, fmt.Errorf("capability %q is not declared", req.CapabilityID)
	}
	if req.ActionKind == ActionKindOperatorCommand {
		if def.OperatorCommand == "" {
			return LifecycleActionResult{}, fmt.Errorf("capability %q has no operator action", req.CapabilityID)
		}
		return s.runOperatorCommand(ctx, req, def)
	}
	if def.DependencyKind != DependencyScenario || def.DependencySlug == "" {
		return LifecycleActionResult{}, fmt.Errorf("capability %q is not a scenario dependency", req.CapabilityID)
	}
	verb, err := lifecycleVerb(req.ActionKind)
	if err != nil {
		return LifecycleActionResult{}, err
	}

	cli := strings.TrimSpace(s.CLIPath)
	if cli == "" {
		cli = "vrooli"
	}
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = defaultLifecycleActionTimeout
	}
	runner := s.Runner
	if runner == nil {
		runner = ExecCommandRunner{}
	}

	actionCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := []string{"scenario", verb, def.DependencySlug, "--json", "--timeout", fmt.Sprintf("%.0f", timeout.Seconds())}
	startResult, startErr := runner.Run(actionCtx, cli, start...)
	if startErr != nil {
		return lifecycleFailure(req, "lifecycle "+verb+" failed", startResult, startErr), nil
	}

	wait := []string{"scenario", "wait", def.DependencySlug, "--json", "--timeout", fmt.Sprintf("%.0f", timeout.Seconds())}
	waitResult, waitErr := runner.Run(actionCtx, cli, wait...)
	if waitErr != nil {
		return lifecycleFailure(req, "lifecycle wait failed", waitResult, waitErr), nil
	}

	return lifecycleSuccess(req, waitResult), nil
}

func (s LifecycleActionService) scenarioDef(capabilityID string) (Def, bool) {
	def, ok := s.definition(capabilityID)
	return def, ok && def.DependencyKind == DependencyScenario
}

func (s LifecycleActionService) definition(capabilityID string) (Def, bool) {
	for _, def := range s.Defs {
		if def.ID == capabilityID {
			return def, true
		}
	}
	return Def{}, false
}

func (s LifecycleActionService) runOperatorCommand(ctx context.Context, req LifecycleActionRequest, def Def) (LifecycleActionResult, error) {
	args := strings.Fields(def.OperatorCommand)
	if len(args) == 0 {
		return LifecycleActionResult{}, fmt.Errorf("capability %q has an empty operator action", req.CapabilityID)
	}
	cli := strings.TrimSpace(s.CLIPath)
	if cli == "" {
		cli = "vrooli"
	}
	runner := s.Runner
	if runner == nil {
		runner = ExecCommandRunner{}
	}
	actionCtx, cancel := context.WithTimeout(ctx, actionTimeout(s.Timeout))
	defer cancel()
	result, err := runner.Run(actionCtx, cli, args...)
	if err != nil {
		return lifecycleFailure(req, "operator action failed", result, err), nil
	}
	return lifecycleSuccess(req, result), nil
}

func actionTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return defaultLifecycleActionTimeout
	}
	return timeout
}

func lifecycleVerb(kind ActionKind) (string, error) {
	switch kind {
	case ActionKindScenarioStart:
		return "start", nil
	case ActionKindScenarioRestart:
		return "restart", nil
	default:
		return "", fmt.Errorf("unsupported lifecycle action %q", kind)
	}
}

type waitPayload struct {
	Success  bool   `json:"success"`
	Verdict  string `json:"verdict"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error"`
}

func lifecycleSuccess(req LifecycleActionRequest, wait CommandResult) LifecycleActionResult {
	payload := waitPayload{Success: wait.ExitCode == 0}
	_ = json.Unmarshal(wait.Stdout, &payload)
	status := strings.TrimSpace(payload.Verdict)
	if status == "" {
		status = "completed"
	}
	msg := "lifecycle action completed"
	if payload.Error != "" {
		msg = payload.Error
	}
	if payload.ExitCode != 0 {
		payload.Success = false
	}
	return LifecycleActionResult{
		Success:      payload.Success,
		Status:       status,
		Message:      msg,
		CapabilityID: req.CapabilityID,
		ActionKind:   req.ActionKind,
	}
}

func lifecycleFailure(req LifecycleActionRequest, prefix string, result CommandResult, err error) LifecycleActionResult {
	detail := strings.TrimSpace(string(result.Stderr))
	if detail == "" {
		detail = strings.TrimSpace(string(result.Stdout))
	}
	if detail == "" && err != nil {
		detail = err.Error()
	}
	return LifecycleActionResult{
		Success:      false,
		Status:       "failed",
		Message:      joinStatusMessage(prefix, detail),
		CapabilityID: req.CapabilityID,
		ActionKind:   req.ActionKind,
	}
}
