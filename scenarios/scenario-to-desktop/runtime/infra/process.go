package infra

import (
	"context"
	"io"
	"os"
	"os/exec"
)

// ProcessRunner abstracts process execution for testing.
type ProcessRunner interface {
	// Start starts a process with the given command and arguments.
	Start(ctx context.Context, cmd string, args []string, env []string, dir string, stdout, stderr io.Writer) (Process, error)
}

// Process represents a running process.
type Process interface {
	// Wait waits for the process to exit and returns any error.
	Wait() error
	// Signal sends a signal to the process.
	Signal(sig os.Signal) error
	// Kill forcefully terminates the process.
	Kill() error
	// Pid returns the process ID.
	Pid() int
}

// CommandRunner abstracts command execution for health checks.
type CommandRunner interface {
	// Run executes a command and returns its exit status.
	Run(ctx context.Context, name string, args []string) error
	// LookPath searches for an executable in the system PATH.
	LookPath(file string) (string, error)
	// Output runs the command and returns its standard output.
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
}

// RealProcessRunner implements ProcessRunner using os/exec.
type RealProcessRunner struct{}

// Start starts a process with the given command and arguments.
func (RealProcessRunner) Start(ctx context.Context, cmd string, args []string, env []string, dir string, stdout, stderr io.Writer) (Process, error) {
	c := exec.CommandContext(ctx, cmd, args...)
	c.Env = env
	c.Dir = dir
	c.Stdout = stdout
	c.Stderr = stderr
	if err := c.Start(); err != nil {
		return nil, err
	}
	return &realProcess{cmd: c}, nil
}

// realProcess wraps exec.Cmd to implement Process interface.
type realProcess struct {
	cmd *exec.Cmd
}

func (p *realProcess) Wait() error {
	return p.cmd.Wait()
}

func (p *realProcess) Signal(sig os.Signal) error {
	if p.cmd.Process == nil {
		return os.ErrProcessDone
	}
	return p.cmd.Process.Signal(sig)
}

func (p *realProcess) Kill() error {
	if p.cmd.Process == nil {
		return os.ErrProcessDone
	}
	return p.cmd.Process.Kill()
}

func (p *realProcess) Pid() int {
	if p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

// Ensure RealProcessRunner implements ProcessRunner.
var _ ProcessRunner = RealProcessRunner{}

// RealCommandRunner implements CommandRunner using os/exec.
type RealCommandRunner struct{}

func (RealCommandRunner) Run(ctx context.Context, name string, args []string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Run()
}

func (RealCommandRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (RealCommandRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}

// Ensure RealCommandRunner implements CommandRunner.
var _ CommandRunner = RealCommandRunner{}
