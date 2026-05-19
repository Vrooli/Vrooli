// Package dev_tools wraps the local development-tool CLIs
// (scenario-auditor, test-genie, scenario-completeness-scoring) behind
// the validation_run.ToolRunner seam.
//
// The runner is intentionally narrow: it shells out to a tool binary
// resolved from PATH, captures exit + stderr, and produces a
// vrun.ToolResult. Per-tool expectation parsing (e.g., reading a
// structured JSON output) is followup work documented in PROBLEMS.md.
package dev_tools

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"development-toolchain-validator/internal/clock"
	vrun "development-toolchain-validator/internal/validation_run"
)

// Options configures the dev-tool runner.
type Options struct {
	Clock clock.Clock

	// CommandRunner is the seam used to exec the tool binary. nil
	// defaults to a real exec.CommandContext.
	CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)
}

// Runner implements vrun.ToolRunner by shelling out to one of the
// dev-tool CLIs.
type Runner struct {
	opts Options
}

// New constructs a Runner with the given options.
func New(opts Options) *Runner {
	if opts.Clock == nil {
		opts.Clock = clock.System{}
	}
	if opts.CommandRunner == nil {
		opts.CommandRunner = defaultCommandRunner
	}
	return &Runner{opts: opts}
}

var _ vrun.ToolRunner = (*Runner)(nil)

// Invoke runs the named tool against the given golden path. The tool
// is expected to exit zero on pass; non-zero exit (or executable
// failure) yields a ToolResult{Succeeded: false} with the captured
// stderr as ErrorReason.
func (r *Runner) Invoke(ctx context.Context, toolName, goldenPath string) (vrun.ToolResult, error) {
	if !isKnownTool(toolName) {
		return vrun.ToolResult{Name: toolName, Succeeded: false, ErrorReason: "unknown tool"}, nil
	}
	start := r.opts.Clock.Now().UTC()
	_, err := r.opts.CommandRunner(ctx, toolName, goldenPath)
	end := r.opts.Clock.Now().UTC()
	if err != nil {
		return vrun.ToolResult{
			Name:        toolName,
			Succeeded:   false,
			StartedAt:   start,
			EndedAt:     end,
			ErrorReason: fmt.Sprintf("%s exited with error: %v", toolName, err),
		}, nil
	}
	return vrun.ToolResult{
		Name:      toolName,
		Succeeded: true,
		StartedAt: start,
		EndedAt:   end,
	}, nil
}

func defaultCommandRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

// knownTools is the closed set of tool names this adapter recognizes.
// Keeps a misconfigured caller from arbitrarily executing PATH lookups.
var knownTools = map[string]struct{}{
	"scenario-auditor":              {},
	"test-genie":                    {},
	"scenario-completeness-scoring": {},
}

func isKnownTool(name string) bool {
	_, ok := knownTools[name]
	return ok
}

// Compile-time hint that time.Duration is in scope; the package is
// kept thin enough that this import would otherwise be elided.
var _ time.Duration
