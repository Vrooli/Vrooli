package hostfs

import (
	"context"
	"errors"
	"os/exec"
	"time"

	"storage-manager/internal/authority"
	"storage-manager/internal/cleanup"
)

// ProcessRunner executes fixed, provider-owned argv vectors. It never invokes
// a shell, so workload names cannot become shell syntax. The commands act on
// live host services (Docker, Ollama), so only the host's storage authority
// may run them.
type ProcessRunner struct{ scope authority.Scope }

// NewProcessRunner builds a runner whose scope follows VROOLI_STORAGE_ROOT.
func NewProcessRunner() *ProcessRunner { return &ProcessRunner{} }

func (r ProcessRunner) Run(ctx context.Context, command cleanup.ProcessCommand) (cleanup.ProcessResult, error) {
	started := time.Now()
	if len(command.Args) == 0 {
		return cleanup.ProcessResult{}, errors.New("process command requires arguments")
	}
	if err := r.scope.CheckHostAction("host command " + command.Name); err != nil {
		return cleanup.ProcessResult{}, err
	}
	cmd := exec.CommandContext(ctx, command.Name, command.Args...)
	out, err := cmd.Output()
	result := cleanup.ProcessResult{Stdout: string(out), Duration: time.Since(started)}
	if exit, ok := err.(*exec.ExitError); ok {
		result.Stderr = string(exit.Stderr)
		result.ExitCode = exit.ExitCode()
	}
	return result, err
}
