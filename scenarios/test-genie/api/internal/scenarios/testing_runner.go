package scenarios

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// TestingRunner executes scenario testing commands described by TestingCapabilities.
type TestingRunner struct {
	Timeout time.Duration
	Output  io.Writer
	LogDir  string
}

type TestingRunnerResult struct {
	Command []string
	LogPath string
}

const defaultTestingTimeout = 10 * time.Minute

// Run executes the preferred testing command (or specific type when provided).
func (r TestingRunner) Run(ctx context.Context, caps TestingCapabilities, preferred string) (*TestingRunnerResult, error) {
	return r.RunWithArgs(ctx, caps, preferred, nil)
}

// RunWithArgs executes the preferred testing command and appends extra args (e.g., path filters).
func (r TestingRunner) RunWithArgs(ctx context.Context, caps TestingCapabilities, preferred string, extraArgs []string) (*TestingRunnerResult, error) {
	cmdSpec := caps.SelectCommand(preferred)
	if cmdSpec == nil {
		return nil, fmt.Errorf("no testing commands available for scenario")
	}

	timeout := r.Timeout
	if timeout <= 0 {
		timeout = defaultTestingTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if len(cmdSpec.Command) == 0 {
		return nil, fmt.Errorf("testing command for %s is empty", cmdSpec.Type)
	}
	name := cmdSpec.Command[0]
	args := cmdSpec.Command[1:]
	if len(extraArgs) > 0 {
		args = append(args, extraArgs...)
	}

	command := exec.CommandContext(runCtx, name, args...)
	if cmdSpec.WorkingDir != "" {
		command.Dir = cmdSpec.WorkingDir
	}
	var logFile *os.File
	var logPath string
	if r.LogDir != "" {
		filename := fmt.Sprintf("scenario-tests-%d.log", time.Now().UnixNano())
		logPath = filepath.Join(r.LogDir, filename)
		if err := os.MkdirAll(r.LogDir, 0o755); err == nil {
			if f, err := os.Create(logPath); err == nil {
				logFile = f
				defer logFile.Close()
				command.Stdout = logFile
				command.Stderr = logFile
			}
		}
	}
	if command.Stdout == nil && r.Output != nil {
		command.Stdout = r.Output
		command.Stderr = r.Output
	}

	if err := command.Run(); err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("%s testing command timed out after %s", cmdSpec.Type, timeout)
		}
		return nil, fmt.Errorf("%s testing command failed: %w", cmdSpec.Type, err)
	}

	return &TestingRunnerResult{
		Command: cmdSpec.Command,
		LogPath: logPath,
	}, nil
}
