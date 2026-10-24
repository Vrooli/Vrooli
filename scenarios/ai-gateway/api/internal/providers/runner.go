package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/envkit-go"
)

const DefaultCommandTimeout = 5 * time.Second

// ProviderErrorMarker is the stable line protocol an owner adapter writes to
// stderr when it classifies a provider-limit failure. The AI Gateway recovers
// the typed code and observed recovery window from it so a resource-command
// boundary does not flatten an account-credit exhaustion or a rate limit into a
// generic exit error. Keep in sync with
// resources/openrouter/cli/internal/health.ProviderErrorMarker.
const ProviderErrorMarker = "VROOLI_PROVIDER_ERROR "

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
	result := Result{
		Stdout: redact(stdout.String()),
		Stderr: redact(stderr.String()),
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
	applyProviderErrorMarker(cmdErr, result.Stderr)
	return result, cmdErr
}

// providerErrorEnvelope is the JSON payload an owner adapter writes after the
// ProviderErrorMarker. It mirrors the marker shape in the OpenRouter resource.
type providerErrorEnvelope struct {
	Code       string `json:"code"`
	HTTPStatus int    `json:"http_status"`
	RetryAfter string `json:"retry_after"`
	Message    string `json:"message"`
}

// applyProviderErrorMarker recovers a typed provider failure an owner adapter
// wrote to stderr, preserving the observed HTTP status and Retry-After instead
// of flattening it into a generic exit error. It is best-effort: a missing or
// malformed marker leaves the original CommandError untouched.
func applyProviderErrorMarker(cmdErr *CommandError, stderr string) {
	if cmdErr == nil {
		return
	}
	for _, line := range strings.Split(stderr, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, ProviderErrorMarker) {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(line, ProviderErrorMarker))
		var envelope providerErrorEnvelope
		if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
			continue
		}
		code := normalizeProviderErrorCode(envelope.Code)
		if code == "" {
			continue
		}
		cmdErr.Code = code
		if envelope.HTTPStatus != 0 {
			cmdErr.HTTPStatus = envelope.HTTPStatus
		}
		if retryAfter := strings.TrimSpace(envelope.RetryAfter); retryAfter != "" {
			cmdErr.RetryAfter = retryAfter
		}
		return
	}
}

func normalizeProviderErrorCode(code string) string {
	switch strings.TrimSpace(code) {
	case CodeInsufficientCredits, CodeRateLimited, CodeProviderOverloaded,
		CodeProviderFailed, CodeStreamFailed, CodeUnreachable:
		return strings.TrimSpace(code)
	default:
		return ""
	}
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
