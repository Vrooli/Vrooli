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

	"github.com/vrooli/ai-provider-go/providererror"
	"github.com/vrooli/cli-core/cliutil"
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
	// Fact marks a read-only policy or inventory query whose result a
	// FactCache may reuse. Commands that act (execute, pull, mutate) are never
	// facts.
	Fact bool
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

// maxFailureDetail bounds a logged failure reason.
const maxFailureDetail = 600

// FailureDetail is a failure's reason on one bounded line for operator logs.
// A resource command failure is reduced to its code, exit status and the
// resource's own message, which the runner has already redacted; the command
// line is left out because it carries no reason.
func FailureDetail(err error) string {
	if err == nil {
		return ""
	}
	detail := err.Error()
	var cmdErr *CommandError
	if errors.As(err, &cmdErr) {
		reason := strings.TrimSpace(cmdErr.Stderr)
		if reason == "" && cmdErr.Err != nil {
			reason = cmdErr.Err.Error()
		}
		detail = fmt.Sprintf("code=%s exit=%d %s", cmdErr.Code, cmdErr.ExitCode, reason)
	}
	detail = strings.Join(strings.Fields(detail), " ")
	if len(detail) > maxFailureDetail {
		detail = detail[:maxFailureDetail] + "..."
	}
	return detail
}

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
	// Every resource command runs inside a request deadline. A resource CLI
	// whose sources changed would otherwise rebuild itself in the request
	// path (tens of seconds), so commands run latency-bound: a stale CLI
	// answers as-is and refreshes in the background for the next call.
	cmd.Env = append(cmd.Env, cliutil.LatencyBoundEnvVar+"=1")
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
		Stdout: redactOutput(stdout.String()),
		Stderr: redactOutput(rawStderr),
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

var (
	secretishPattern            = regexp.MustCompile(`(?i)(api[_-]?key|authorization|bearer|token|secret)(["'=:\s]+)(bearer\s+)?[^,\s"]+`)
	secretFieldPattern          = regexp.MustCompile(`(?i)(api[_-]?key|authorization|bearer|token|secret)`)
	credentialIdentifierPattern = regexp.MustCompile(`^[A-Z0-9_]*(API[_-]?KEY|TOKEN|SECRET|AUTHORIZATION)$`)
	tokenCountFieldPattern      = regexp.MustCompile(`(?i)^[a-z0-9_]*_tokens$|^tokens$`)
)

func redact(s string) string {
	return secretishPattern.ReplaceAllString(s, "$1$2[redacted]")
}

func redactOutput(s string) string {
	var value any
	if err := json.Unmarshal([]byte(s), &value); err != nil {
		return redact(s)
	}
	value = redactJSONValue(value, "")
	redacted, err := json.Marshal(value)
	if err != nil {
		return redact(s)
	}
	return string(redacted)
}

func redactJSONValue(value any, field string) any {
	if isSecretField(field) && !isTokenCount(field, value) {
		return "[redacted]"
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			typed[key] = redactJSONValue(child, key)
		}
	case []any:
		for index, child := range typed {
			if field == "missing_credentials" && isCredentialIdentifier(child) {
				continue
			}
			typed[index] = redactJSONValue(child, field)
		}
	case string:
		if isCredentialIdentifier(typed) {
			return typed
		}
		return redact(typed)
	}
	return value
}

func isSecretField(field string) bool {
	return secretFieldPattern.MatchString(field)
}

// isTokenCount reports a numeric usage or limit field such as input_tokens,
// output_tokens or max_tokens. These are ordinary metadata, not credentials: a
// substring match would redact them into strings and make otherwise valid
// resource JSON (a decision's usage block, a role's token cap) impossible for
// downstream adapters to decode. A credential is a string, and the singular
// "token" field stays secret.
func isTokenCount(field string, value any) bool {
	if _, numeric := value.(float64); !numeric {
		return false
	}
	return tokenCountFieldPattern.MatchString(field)
}

func isCredentialIdentifier(value any) bool {
	text, ok := value.(string)
	return ok && credentialIdentifierPattern.MatchString(text)
}
