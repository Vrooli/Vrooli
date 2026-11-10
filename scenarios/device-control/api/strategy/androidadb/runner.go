package androidadb

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Runner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

type Process interface {
	Start() error
	Interrupt() error
	Kill() error
	Wait() error
}

type commandProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stderr cappedBuffer
}

func (p *commandProcess) Start() error {
	p.stderr.limit = 2048
	p.cmd.Stderr = &p.stderr
	stdin, err := p.cmd.StdinPipe()
	if err != nil {
		return err
	}
	p.stdin = stdin
	return p.cmd.Start()
}

func (p *commandProcess) Interrupt() error {
	if p.stdin == nil {
		return fmt.Errorf("process has not started")
	}
	_, err := p.stdin.Write([]byte{3})
	return err
}

func (p *commandProcess) Kill() error {
	if p.cmd.Process == nil {
		return fmt.Errorf("process has not started")
	}
	return p.cmd.Process.Kill()
}

func (p *commandProcess) Wait() error {
	err := p.cmd.Wait()
	if err != nil {
		if detail := strings.TrimSpace(p.stderr.buffer.String()); detail != "" {
			return fmt.Errorf("%w: %s", err, detail)
		}
	}
	return err
}

type ProcessRunner interface {
	Start(string, ...string) (Process, error)
}

type commandRunner struct{ requireDefaultADBServer bool }

type boundedRunner interface {
	RunBounded(context.Context, string, int, ...string) ([]byte, error)
}

const wirelessADBPort = "5555"

func (commandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil && ctx.Err() != nil {
		return output, fmt.Errorf("command failed (%v) after context ended: %w", err, ctx.Err())
	}
	return output, err
}

func (commandRunner) RunBounded(ctx context.Context, name string, maxBytes int, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr cappedBuffer
	stdout.limit, stderr.limit = maxBytes, 8*1024
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if stdout.exceeded || stderr.exceeded {
		return nil, fmt.Errorf("command output exceeded configured limit")
	}
	if err != nil && ctx.Err() != nil {
		return stdout.buffer.Bytes(), fmt.Errorf("command failed (%v) after context ended: %w", err, ctx.Err())
	}
	return stdout.buffer.Bytes(), err
}

func (commandRunner) Start(name string, args ...string) (Process, error) {
	return &commandProcess{cmd: exec.Command(name, args...)}, nil
}
