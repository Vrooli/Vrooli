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
	configureProcessCommand(c)
	c.Env = env
	c.Dir = dir
	c.Stdout = stdout
	c.Stderr = stderr
	if err := c.Start(); err != nil {
		return nil, err
	}
	cleanup, err := assignProcessContainment(c.Process)
	if err != nil {
		_ = c.Process.Kill()
		_ = c.Wait()
		return nil, err
	}
	return &realProcess{cmd: c, cleanup: cleanup}, nil
}

// realProcess wraps exec.Cmd to implement Process interface.
type realProcess struct {
	cmd     *exec.Cmd
	cleanup func()
}

func (p *realProcess) Wait() error {
	err := p.cmd.Wait()
	if p.cleanup != nil {
		p.cleanup()
		p.cleanup = nil
	}
	return err
}

func (p *realProcess) Signal(sig os.Signal) error {
	if p.cmd.Process == nil {
		return os.ErrProcessDone
	}
	return p.cmd.Process.Signal(sig)
}

func (p *realProcess) GracefulStop() error {
	if p.cmd.Process == nil {
		return os.ErrProcessDone
	}
	return gracefulStopProcess(p.cmd.Process)
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

// StopProcess requests a platform-correct graceful stop. Test doubles that do
// not implement the optional method retain the small Process contract.
func StopProcess(p Process) error {
	if graceful, ok := p.(interface{ GracefulStop() error }); ok {
		return graceful.GracefulStop()
	}
	return p.Signal(Interrupt)
}

// ConfigureProcessCommand applies platform process-group/job semantics to a
// command before it starts. Resource and scenario launchers share this seam.
func ConfigureProcessCommand(cmd *exec.Cmd) { configureProcessCommand(cmd) }

// GracefulStopProcess sends the platform-correct group or console shutdown
// request to an already-started process.
func GracefulStopProcess(process *os.Process) error { return gracefulStopProcess(process) }

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
