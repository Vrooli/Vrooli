package capabilities

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

const defaultLifecycleActionTimeout = 180 * time.Second

// Install outcome statuses.
//
// These are the contract between an installer and every surface that renders
// its result, so they are stable strings rather than an unexported enum.
//
// The distinction that matters is between "the installer exited 0" and "the
// machine now has the thing". Only the second is a success an operator can
// act on, and the two are routinely different: a relayed install can complete
// cleanly while the binary lands somewhere the machine does not look, and a
// machine that reports its inventory on a heartbeat has not necessarily
// reported anything yet by the time the installer returns.
const (
	// InstallStatusInstalled means the target itself now reports the
	// capability as present. It is the ONLY status that may be rendered as a
	// completed install.
	InstallStatusInstalled = "installed"
	// InstallStatusUnconfirmed means the installer completed without error and
	// the target has not (yet) reported the capability. It is deliberately not
	// a failure: the honest answer is "we do not know", and the remedy is to
	// look again rather than to install again.
	InstallStatusUnconfirmed = "unconfirmed"
	// InstallStatusFailed means the installer itself did not succeed.
	InstallStatusFailed = "failed"
	// InstallStatusNotApplicable means the capability cannot exist on this
	// target at all, so there is nothing to install and nothing to retry.
	InstallStatusNotApplicable = "not_applicable"

	// InstallConfirmedState is the state a target reports for a capability it
	// actually has. It matches capabilityprobe.Ready by value rather than by
	// import: this package is depended on by the Bridge-facing paths and must
	// stay free of the probe.
	InstallConfirmedState = "ready"
)

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
	cli := strings.TrimSpace(s.CLIPath)
	if cli == "" {
		cli = "vrooli"
	}
	args := operatorCommandArgs(def.OperatorCommand, cli)
	if len(args) == 0 {
		return LifecycleActionResult{}, fmt.Errorf("capability %q has an empty operator action", req.CapabilityID)
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

// operatorCommandArgs turns a declared operator command into argv for cli.
//
// Def.OperatorCommand serves two masters: it is executed here, and it is
// rendered verbatim in the UI as a line an operator can paste into a shell.
// The second use means it carries the CLI's own name, and passing the whole
// string through as arguments produced `vrooli vrooli resource install codex`
// — which is why installing a coding agent on this machine has never worked.
// Drop exactly one leading token, and only when it names the CLI being run.
func operatorCommandArgs(command, cli string) []string {
	args := strings.Fields(command)
	if len(args) == 0 {
		return nil
	}
	name := commandBaseName(cli)
	if name != "" && commandBaseName(args[0]) == name {
		return args[1:]
	}
	return args
}

// commandBaseName reduces a path or bare name to the executable name, so a
// CLIPath of /usr/local/bin/vrooli still matches a declared "vrooli" prefix.
//
// Both separators are cut on every platform rather than deferring to
// filepath.Base: a backslash is a legal filename character on Unix, so
// filepath.Base leaves a Windows-shaped path intact there, and a comparison
// that only works on the host that produced the path is the kind that fails
// in the field and passes in CI. The .exe suffix goes for the same reason.
func commandBaseName(value string) string {
	value = strings.TrimSpace(value)
	if index := strings.LastIndexAny(value, `/\`); index >= 0 {
		value = value[index+1:]
	}
	if strings.EqualFold(filepath.Ext(value), ".exe") {
		value = value[:len(value)-len(".exe")]
	}
	return value
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
