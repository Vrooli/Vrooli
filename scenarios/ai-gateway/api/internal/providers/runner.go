package providers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const DefaultCommandTimeout = 5 * time.Second

// seam: CommandRunner executes resource-owned CLI commands. Production wires
// ExecRunner; tests wire providers/mocks.FakeRunner.
type CommandRunner interface {
	Run(ctx context.Context, command Command) (Result, error)
}

type Command struct {
	Name    string
	Args    []string
	Stdin   string
	Timeout time.Duration
}

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type CommandError struct {
	Code     string
	Command  string
	ExitCode int
	Stderr   string
	Err      error
}

func (e *CommandError) Error() string {
	if e == nil {
		return ""
	}
	detail := strings.TrimSpace(e.Stderr)
	if detail == "" && e.Err != nil {
		detail = e.Err.Error()
	}
	if detail == "" {
		detail = e.Code
	}
	return fmt.Sprintf("%s: %s", e.Command, detail)
}

func (e *CommandError) Unwrap() error { return e.Err }

type ExecRunner struct{}

var _ CommandRunner = ExecRunner{}

func (ExecRunner) Run(ctx context.Context, command Command) (Result, error) {
	if !allowedResourceCommand(command.Name) {
		return Result{}, &CommandError{Code: "unsupported_command", Command: command.String(), ExitCode: -1, Err: fmt.Errorf("unsupported resource command %q", command.Name)}
	}
	timeout := command.Timeout
	if timeout <= 0 {
		timeout = DefaultCommandTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command.Name, command.Args...) // #nosec G204 -- command.Name is restricted by allowedResourceCommand; args are fixed by adapters, prompt data is passed via stdin.
	cmd.Env = os.Environ()
	if command.Stdin != "" {
		cmd.Stdin = strings.NewReader(command.Stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{
		Stdout: redact(stdout.String()),
		Stderr: redact(stderr.String()),
	}
	if err == nil {
		return result, nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return result, &CommandError{Code: "timeout", Command: command.String(), ExitCode: -1, Stderr: result.Stderr, Err: ctx.Err()}
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, &CommandError{Code: "exit_error", Command: command.String(), ExitCode: result.ExitCode, Stderr: result.Stderr, Err: err}
	}
	return result, &CommandError{Code: "missing_binary", Command: command.String(), ExitCode: -1, Stderr: result.Stderr, Err: err}
}

func (c Command) String() string {
	parts := append([]string{c.Name}, c.Args...)
	return strings.Join(parts, " ")
}

func allowedResourceCommand(name string) bool {
	switch name {
	case "resource-ollama", "resource-openrouter":
		return true
	default:
		return false
	}
}

var secretishPattern = regexp.MustCompile(`(?i)(api[_-]?key|authorization|bearer|token|secret)(["'=:\s]+)(bearer\s+)?[^,\s"]+`)

func redact(s string) string {
	return secretishPattern.ReplaceAllString(s, "$1$2[redacted]")
}
