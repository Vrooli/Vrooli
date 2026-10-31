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

	"github.com/vrooli/ai-provider-go/providererror"
	"github.com/vrooli/envkit-go"
)

const DefaultCommandTimeout = 5 * time.Second

// ProviderErrorMarker is the stable line protocol an owner adapter writes to
// stderr when it classifies a provider-limit failure.
const ProviderErrorMarker = providererror.Marker

// seam: CommandRunner executes resource-owned CLI commands. Production wires
// ExecRunner; tests wire providers/mocks.FakeRunner.
type CommandRunner interface {
	Run(ctx context.Context, command Command) (Result, error)
}

type Command struct {
	Name  string
	Args  []string
	Stdin string
	// Env contains ephemeral values for the child resource process. It is never
	// included in Command.String or error messages.
	Env     map[string]string
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
	// HTTPStatus is the provider HTTP status that produced the failure, when the
	// failure came from an HTTP response. 0 when not applicable.
	HTTPStatus int
	// RetryAfter is the observed Retry-After value exactly as the provider sent
	// it (seconds or an HTTP date). Empty means the provider sent none; it is
	// never synthesized from a default cooldown.
	RetryAfter string
	// ResetAt is the observed reset time (RFC3339) when the provider supplied
	// one. Empty means unknown and must stay unknown rather than being guessed.
	ResetAt string
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
	cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, nil)
	for key, value := range command.Env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	if command.Stdin != "" {
		cmd.Stdin = strings.NewReader(command.Stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	rawStderr := stderr.String()
	result := Result{
		Stdout: redact(stdout.String()),
		Stderr: redact(rawStderr),
	}
	if err == nil {
		return result, nil
	}
	var cmdErr *CommandError
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		cmdErr = &CommandError{Code: "timeout", Command: command.String(), ExitCode: -1, Stderr: result.Stderr, Err: ctx.Err()}
	default:
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			cmdErr = &CommandError{Code: "exit_error", Command: command.String(), ExitCode: result.ExitCode, Stderr: result.Stderr, Err: err}
		} else {
			cmdErr = &CommandError{Code: "missing_binary", Command: command.String(), ExitCode: -1, Stderr: result.Stderr, Err: err}
		}
	}
	// The marker is read before redaction: redaction targets free text and
	// corrupts a marker whose message names a credential variable (an
	// "unconfigured: needs EDENAI_API_KEY" refusal). Only the code, status and
	// retry window are taken from it; the retained stderr stays redacted.
	applyProviderErrorMarker(cmdErr, rawStderr)
	return result, cmdErr
}

// applyProviderErrorMarker recovers a typed provider failure an owner adapter
// wrote to stderr. The marker contract lives in ai-provider-go so the resource
// and the gateway cannot drift.
func applyProviderErrorMarker(cmdErr *CommandError, stderr string) {
	if cmdErr == nil {
		return
	}
	envelope, ok := providererror.ParseMarker(stderr)
	if !ok {
		return
	}
	code := providererror.NormalizeCode(envelope.Code)
	if code == "" {
		return
	}
	cmdErr.Code = code
	if envelope.HTTPStatus != 0 {
		cmdErr.HTTPStatus = envelope.HTTPStatus
	}
	if retryAfter := strings.TrimSpace(envelope.RetryAfter); retryAfter != "" {
		cmdErr.RetryAfter = retryAfter
	}
}

func normalizeProviderErrorCode(code string) string {
	return providererror.NormalizeCode(code)
}

func (c Command) String() string {
	parts := append([]string{c.Name}, c.Args...)
	return strings.Join(parts, " ")
}

func allowedResourceCommand(name string) bool {
	_, ok := AllowedCommands()[name]
	return ok
}

var secretishPattern = regexp.MustCompile(`(?i)(api[_-]?key|authorization|bearer|token|secret)(["'=:\s]+)(bearer\s+)?[^,\s"]+`)

func redact(s string) string {
	return secretishPattern.ReplaceAllString(s, "$1$2[redacted]")
}
