// Package dev_tools runs the local development-tool CLIs (test-genie,
// scenario-completeness-scoring) against a golden scenario behind the
// validation_run.ToolRunner seam.
//
// Unlike the original exit-code-only stub, the runner builds the correct
// per-tool argv, captures stdout/stderr separately, and applies a
// per-tool expectation (test-genie: every phase passes;
// completeness-scoring: score >= floor). It distinguishes "the tool
// could not run" (run failure) from "the tool ran and the expectation
// did not hold" (tool/template regression) so the evaluator can map the
// two-layer signal to the right verdict.
package dev_tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"

	vrun "development-toolchain-validator/internal/validation_run"

	"github.com/vrooli/api-core/schedule"
)

// CommandResult captures one tool process execution.
type CommandResult struct {
	Stdout []byte
	Stderr []byte
	// ExitCode is the process exit code (meaningful only when Launched).
	ExitCode int
	// Launched is true when the process actually started and exited
	// (even non-zero); false when it could not be started at all or was
	// killed by a context deadline before completing.
	Launched bool
}

// Options configures the dev-tool runner.
type Options struct {
	Clock schedule.Clock

	// CommandRunner is the seam used to exec a tool command. nil defaults
	// to a real exec.CommandContext capturing stdout/stderr separately.
	CommandRunner func(ctx context.Context, name string, args ...string) (CommandResult, error)

	// ExpectationsDir overrides the directory holding <tool>.json
	// expectation files. Empty resolves to the committed repo directory.
	ExpectationsDir string

	// Registry overrides the tool registry. nil uses defaultRegistry.
	Registry map[string]toolSpec
}

// Runner implements vrun.ToolRunner by shelling out to a registered
// dev-tool CLI and applying its expectation.
type Runner struct {
	opts     Options
	registry map[string]toolSpec
}

// New constructs a Runner with the given options.
func New(opts Options) *Runner {
	if opts.Clock == nil {
		opts.Clock = schedule.System()
	}
	if opts.CommandRunner == nil {
		opts.CommandRunner = defaultCommandRunner
	}
	registry := opts.Registry
	if registry == nil {
		registry = defaultRegistry()
	}
	return &Runner{opts: opts, registry: registry}
}

var _ vrun.ToolRunner = (*Runner)(nil)

// Invoke runs the named tool against the golden and returns a ToolResult
// carrying the two-layer signal (Ran / ExpectationMet) plus the captured
// output. It returns a non-nil error only when the tool is unknown — an
// unrecoverable misconfiguration. All other failure modes are encoded in
// the ToolResult (Ran=false for "couldn't run", ExpectationMet=false for
// "ran but failed"), so the evaluator is the single place verdicts are
// decided.
func (r *Runner) Invoke(ctx context.Context, toolName, goldenSlug, goldenPath string) (vrun.ToolResult, error) {
	start := r.opts.Clock.Now().UTC()

	spec, ok := r.registry[toolName]
	if !ok {
		return vrun.ToolResult{
			Name:        toolName,
			Ran:         false,
			StartedAt:   start,
			EndedAt:     r.opts.Clock.Now().UTC(),
			ErrorReason: "unknown tool",
		}, fmt.Errorf("unknown tool %q", toolName)
	}

	exp, err := resolveExpectation(r.opts.ExpectationsDir, toolName)
	if err != nil {
		return vrun.ToolResult{
			Name:        toolName,
			Ran:         false,
			StartedAt:   start,
			EndedAt:     r.opts.Clock.Now().UTC(),
			ErrorReason: "load expectation: " + err.Error(),
		}, nil
	}

	absPath, err := resolveAbsGoldenPath(goldenPath)
	if err != nil {
		return vrun.ToolResult{
			Name:        toolName,
			Ran:         false,
			StartedAt:   start,
			EndedAt:     r.opts.Clock.Now().UTC(),
			ErrorReason: "resolve golden path: " + err.Error(),
		}, nil
	}

	cmds := spec.commands(goldenSlug, absPath, exp)
	var rawAll bytes.Buffer
	var finalRes CommandResult
	for i, args := range cmds {
		res, runErr := r.opts.CommandRunner(ctx, toolName, args...)
		appendRawOutput(&rawAll, toolName, args, res)
		isFinal := i == len(cmds)-1

		if runErr != nil || !res.Launched {
			reason := fmt.Sprintf("%s could not run (%s)", toolName, strings.Join(args, " "))
			if runErr != nil {
				reason += ": " + runErr.Error()
			}
			return vrun.ToolResult{
				Name:        toolName,
				Ran:         false,
				StartedAt:   start,
				EndedAt:     r.opts.Clock.Now().UTC(),
				RawOutput:   rawAll.Bytes(),
				ErrorReason: reason,
			}, nil
		}
		// A preparatory (non-final) command that exits non-zero means we
		// could not complete the measurement → run failure. The final
		// command's non-zero exit is allowed: it is the expectation signal.
		if !isFinal && res.ExitCode != 0 {
			return vrun.ToolResult{
				Name:        toolName,
				Ran:         false,
				StartedAt:   start,
				EndedAt:     r.opts.Clock.Now().UTC(),
				RawOutput:   rawAll.Bytes(),
				ErrorReason: fmt.Sprintf("%s preparatory step (%s) exited %d", toolName, strings.Join(args, " "), res.ExitCode),
			}, nil
		}
		if isFinal {
			finalRes = res
		}
	}

	met, detail := spec.evaluate(finalRes, exp)
	result := vrun.ToolResult{
		Name:           toolName,
		Ran:            true,
		ExpectationMet: met,
		Detail:         detail,
		RawOutput:      rawAll.Bytes(),
		StartedAt:      start,
		EndedAt:        r.opts.Clock.Now().UTC(),
	}
	if !met {
		result.ErrorReason = detail
	}
	return result, nil
}

// defaultCommandRunner executes one tool command, capturing stdout and
// stderr separately so a tool's --json report (stdout) is parseable even
// when it also logs to stderr. A non-zero exit is reported via
// CommandResult.ExitCode with Launched=true (the process ran); only a
// genuine launch failure or a context-deadline kill sets Launched=false.
func defaultCommandRunner(ctx context.Context, name string, args ...string) (CommandResult, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	res := CommandResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), Launched: true}

	if ctxErr := ctx.Err(); ctxErr != nil {
		// Context cancellation/timeout killed the process mid-run: treat as
		// "could not complete" rather than a clean exit.
		res.Launched = false
		return res, ctxErr
	}
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
			return res, nil
		}
		// Failed to start at all (binary missing, etc.).
		res.Launched = false
		return res, runErr
	}
	return res, nil
}

// appendRawOutput accumulates a command's captured output into buf with a
// header, so a persisted multi-command run is legible during triage.
func appendRawOutput(buf *bytes.Buffer, tool string, args []string, res CommandResult) {
	fmt.Fprintf(buf, "$ %s %s\n", tool, strings.Join(args, " "))
	if len(res.Stdout) > 0 {
		buf.Write(res.Stdout)
		if res.Stdout[len(res.Stdout)-1] != '\n' {
			buf.WriteByte('\n')
		}
	}
	if len(res.Stderr) > 0 {
		buf.WriteString("--- stderr ---\n")
		buf.Write(res.Stderr)
		if res.Stderr[len(res.Stderr)-1] != '\n' {
			buf.WriteByte('\n')
		}
	}
}

// resolveAbsGoldenPath turns a materialized golden path into an absolute host
// path the tools can read. Normal runs pass an absolute temp path, while
// retained/generated debug paths may be repo-relative. Relative paths resolve
// from VROOLI_SOURCE_ROOT/VROOLI_ROOT with a cwd fallback. Mirrors
// agent_manager.resolveGoldenRoot.
func resolveAbsGoldenPath(goldenPath string) (string, error) {
	p := strings.TrimSpace(goldenPath)
	if p == "" {
		return "", fmt.Errorf("golden path is empty")
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p), nil
	}
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return "", fmt.Errorf("resolve repo root for golden path %q: %w", p, err)
	}
	return filepath.Join(root, filepath.FromSlash(p)), nil
}
