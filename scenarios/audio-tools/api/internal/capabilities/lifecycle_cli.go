package capabilities

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
)

// allowedSlugs is the whitelist of resource slugs CLIController will
// ever pass to `vrooli resource <verb> <slug>`. Production wiring
// constrains the inbound provider_id through ResourceSlugForProviderID;
// this second check is defence-in-depth against future regressions.
var allowedSlugs = map[string]struct{}{
	"whisper":     {},
	"sherpa-onnx": {},
	"ollama":      {},
}

// allowedVerbs is the whitelist of `vrooli resource …` subcommands the
// controller may invoke. Keeping this enumerable means a future plan
// can extend the surface deliberately rather than by accident.
var allowedVerbs = map[string]struct{}{
	"start":   {},
	"stop":    {},
	"restart": {},
	"logs":    {},
}

// CLIController is the production ResourceController. It shells out to
// the `vrooli` CLI located at construction time. If the binary is not
// on PATH it returns ErrControllerUnavailable from every method so
// handlers translate to CodeUnavailable with a stable, actionable
// message.
type CLIController struct {
	// vrooliBin is the absolute path to the `vrooli` binary or empty
	// when the binary was not found at construction.
	vrooliBin string
}

// NewCLIController resolves the `vrooli` binary once at startup. The
// returned controller is safe even when the binary is missing — every
// method returns ErrControllerUnavailable in that case.
func NewCLIController() *CLIController {
	bin, err := exec.LookPath("vrooli")
	if err != nil {
		return &CLIController{}
	}
	return &CLIController{vrooliBin: bin}
}

// Start invokes `vrooli resource start <slug>`.
func (c *CLIController) Start(ctx context.Context, slug string) error {
	return c.runResourceVerb(ctx, "start", slug)
}

// Stop invokes `vrooli resource stop <slug>`.
func (c *CLIController) Stop(ctx context.Context, slug string) error {
	return c.runResourceVerb(ctx, "stop", slug)
}

// Restart invokes `vrooli resource restart <slug>`.
func (c *CLIController) Restart(ctx context.Context, slug string) error {
	return c.runResourceVerb(ctx, "restart", slug)
}

// Logs shells out to `vrooli resource logs <slug> [--follow]
// [--tail N]` and returns its stdout (combined with stderr) as a
// ReadCloser. Closing the reader cancels the child process via the
// passed-in context's deadline mechanism — callers SHOULD pass a
// cancellable ctx and cancel() when done.
func (c *CLIController) Logs(ctx context.Context, slug string, follow bool, tailLines int) (io.ReadCloser, error) {
	if c.vrooliBin == "" {
		return nil, ErrControllerUnavailable
	}
	if _, ok := allowedSlugs[slug]; !ok {
		return nil, fmt.Errorf("disallowed resource slug %q", slug)
	}
	args := []string{"resource", "logs", slug}
	if follow {
		args = append(args, "--follow")
	}
	if tailLines > 0 {
		args = append(args, "--tail", strconv.Itoa(tailLines))
	}
	cmd := exec.CommandContext(ctx, c.vrooliBin, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout // best-effort merge of stderr into stdout pipe
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start vrooli resource logs: %w", err)
	}
	return &cmdReader{ReadCloser: stdout, cmd: cmd}, nil
}

// PullModel shells out to `vrooli resource ollama pull-model <model>`
// if that surface exists at runtime; otherwise falls back to
// `vrooli resource exec ollama -- ollama pull <model>`. The selected
// surface is decided at call time by the platform CLI; we encode the
// preferred shape first and accept either exit status.
func (c *CLIController) PullModel(ctx context.Context, model string) error {
	if c.vrooliBin == "" {
		return ErrControllerUnavailable
	}
	if model == "" {
		return fmt.Errorf("model name required")
	}
	// Prefer the typed subcommand; if the platform CLI exposes it the
	// call succeeds. Otherwise we fall through to `resource exec`.
	args := []string{"resource", "ollama", "pull-model", model}
	cmd := exec.CommandContext(ctx, c.vrooliBin, args...)
	if out, err := cmd.CombinedOutput(); err == nil {
		_ = out
		return nil
	}
	// Fallback: shell out via `resource exec` (every resource exposes
	// this verb). The `--` separator guarantees `ollama pull <model>`
	// is treated as the inner command and never reparsed.
	fallback := []string{"resource", "exec", "ollama", "--", "ollama", "pull", model}
	cmd = exec.CommandContext(ctx, c.vrooliBin, fallback...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("vrooli resource exec ollama -- ollama pull %s: %w: %s", model, err, string(out))
	}
	return nil
}

func (c *CLIController) runResourceVerb(ctx context.Context, verb, slug string) error {
	if c.vrooliBin == "" {
		return ErrControllerUnavailable
	}
	if _, ok := allowedVerbs[verb]; !ok {
		return fmt.Errorf("disallowed resource verb %q", verb)
	}
	if _, ok := allowedSlugs[slug]; !ok {
		return fmt.Errorf("disallowed resource slug %q", slug)
	}
	cmd := exec.CommandContext(ctx, c.vrooliBin, "resource", verb, slug)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("vrooli resource %s %s: %w: %s", verb, slug, err, string(out))
	}
	return nil
}

// cmdReader wraps a pipe so closing it also reaps the child process.
type cmdReader struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (r *cmdReader) Close() error {
	closeErr := r.ReadCloser.Close()
	if r.cmd.Process != nil {
		_ = r.cmd.Process.Kill()
	}
	_ = r.cmd.Wait()
	return closeErr
}
