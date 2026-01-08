package ssh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Runner executes SSH commands on a remote host.
type Runner interface {
	Run(ctx context.Context, cfg Config, command string) (Result, error)
}

// SCPRunner transfers files to a remote host via SCP.
type SCPRunner interface {
	Copy(ctx context.Context, cfg Config, localPath, remotePath string) error
}

// ExecRunner implements Runner using os/exec.
type ExecRunner struct{}

// Run executes an SSH command and returns the result.
func (ExecRunner) Run(ctx context.Context, cfg Config, command string) (Result, error) {
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-o", "ServerAliveInterval=5",
		"-o", "ServerAliveCountMax=1",
		"-o", "StrictHostKeyChecking=accept-new",
		"-p", strconv.Itoa(cfg.Port),
	}
	if cfg.KeyPath != "" {
		args = append(args, "-i", cfg.KeyPath)
	}
	target := fmt.Sprintf("%s@%s", cfg.User, cfg.Host)
	args = append(args, target, "--", command)

	cmd := exec.CommandContext(ctx, "ssh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exitCode = ee.ExitCode()
		} else {
			exitCode = 255
		}
	}
	return Result{
		Stdout:   strings.TrimRight(stdout.String(), "\n"),
		Stderr:   strings.TrimRight(stderr.String(), "\n"),
		ExitCode: exitCode,
	}, err
}

// ExecSCPRunner implements SCPRunner using os/exec.
type ExecSCPRunner struct{}

// Copy transfers a local file to a remote path via SCP.
func (ExecSCPRunner) Copy(ctx context.Context, cfg Config, localPath, remotePath string) error {
	copyCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	args := []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-o", "StrictHostKeyChecking=accept-new",
		"-P", strconv.Itoa(cfg.Port),
	}
	if cfg.KeyPath != "" {
		args = append(args, "-i", cfg.KeyPath)
	}
	// Wrap IPv6 addresses in brackets for scp target format
	host := cfg.Host
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	target := fmt.Sprintf("%s@%s:%s", cfg.User, host, remotePath)
	args = append(args, localPath, target)

	cmd := exec.CommandContext(copyCtx, "scp", args...)
	return cmd.Run()
}
