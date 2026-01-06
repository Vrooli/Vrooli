package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// vrooliCommand wraps a vrooli command with PATH setup for SSH non-interactive sessions.
// SSH non-interactive commands don't source .bashrc, so we need to set up PATH explicitly.
// The vrooli CLI is typically installed to ~/.local/bin via the CLI installer.
func vrooliCommand(workdir, cmd string) string {
	// Prepend common user bin directories to PATH before running vrooli commands
	pathSetup := `export PATH="$HOME/.local/bin:$HOME/bin:/usr/local/bin:$PATH"`
	return fmt.Sprintf("%s && cd %s && %s", pathSetup, shellQuoteSingle(workdir), cmd)
}

type SSHConfig struct {
	Host    string
	Port    int
	User    string
	KeyPath string
}

func sshConfigFromManifest(manifest CloudManifest) SSHConfig {
	vps := manifest.Target.VPS
	return SSHConfig{
		Host:    vps.Host,
		Port:    vps.Port,
		User:    vps.User,
		KeyPath: strings.TrimSpace(vps.KeyPath),
	}
}

// SSHConfigDefaults provides common default values for SSH connection parameters.
const (
	DefaultSSHPort = 22
	DefaultSSHUser = "root"
)

// NewSSHConfig creates an SSHConfig with defaults applied for missing values.
// This reduces repeated `if port == 0 { port = 22 }` patterns in handlers.
func NewSSHConfig(host string, port int, user string, keyPath string) SSHConfig {
	if port == 0 {
		port = DefaultSSHPort
	}
	if user == "" {
		user = DefaultSSHUser
	}
	return SSHConfig{
		Host:    host,
		Port:    port,
		User:    user,
		KeyPath: strings.TrimSpace(keyPath),
	}
}

type SSHResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

type SSHRunner interface {
	Run(ctx context.Context, cfg SSHConfig, command string) (SSHResult, error)
}

type SCPRunner interface {
	Copy(ctx context.Context, cfg SSHConfig, localPath, remotePath string) error
}

type ExecSSHRunner struct{}

func (ExecSSHRunner) Run(ctx context.Context, cfg SSHConfig, command string) (SSHResult, error) {
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
	// Pass command directly - SSH runs it through remote user's shell
	// Note: Previously used "bash -lc" wrapper but that broke when SSH
	// concatenated separate args, causing commands with spaces to fail
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
	} else {
		exitCode = 0
	}
	return SSHResult{
		Stdout:   strings.TrimRight(stdout.String(), "\n"),
		Stderr:   strings.TrimRight(stderr.String(), "\n"),
		ExitCode: exitCode,
	}, err
}

// RunSSHWithOutput executes an SSH command and returns an error with output context if it fails.
// This reduces cognitive load by consolidating the repeated pattern of running SSH commands
// and building error messages from stdout/stderr. The returned error includes the last 50 lines
// of stdout (useful because tools like vrooli often redirect stderr to stdout via `2>&1 | tee`).
func RunSSHWithOutput(ctx context.Context, runner SSHRunner, cfg SSHConfig, cmd string) error {
	if containsTildeInSingleQuotes(cmd) {
		msg := fmt.Sprintf("invalid command: tilde inside single quotes prevents home expansion; use $HOME or an absolute path. command: %s", cmd)
		log.Printf("ssh command rejected: %s", msg)
		return fmt.Errorf("%s", msg)
	}
	log.Printf("ssh command: %s", formatSSHCommandForLog(cfg, cmd))
	res, err := runner.Run(ctx, cfg, cmd)
	if err != nil {
		return buildSSHError(err, res)
	}
	return nil
}

// buildSSHError constructs an error with output context from an SSH result.
// Collects both stderr and stdout (limited to last 50 lines) for comprehensive error context.
func buildSSHError(err error, res SSHResult) error {
	var outputParts []string
	if res.Stderr != "" {
		outputParts = append(outputParts, "stderr: "+res.Stderr)
	}
	if res.Stdout != "" {
		lines := strings.Split(res.Stdout, "\n")
		if len(lines) > 50 {
			lines = lines[len(lines)-50:]
			outputParts = append(outputParts, "stdout (last 50 lines): "+strings.Join(lines, "\n"))
		} else {
			outputParts = append(outputParts, "stdout: "+res.Stdout)
		}
	}
	if len(outputParts) > 0 {
		return fmt.Errorf("%w\n%s", err, strings.Join(outputParts, "\n"))
	}
	return err
}

func formatSSHCommandForLog(cfg SSHConfig, cmd string) string {
	parts := []string{
		"ssh",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-o", "ServerAliveInterval=5",
		"-o", "ServerAliveCountMax=1",
		"-o", "StrictHostKeyChecking=accept-new",
		"-p", strconv.Itoa(cfg.Port),
	}
	if cfg.KeyPath != "" {
		parts = append(parts, "-i", "<redacted>")
	}
	target := fmt.Sprintf("%s@%s", cfg.User, cfg.Host)
	parts = append(parts, target, "--", cmd)
	return strings.Join(parts, " ")
}

type ExecSCPRunner struct{}

func (ExecSCPRunner) Copy(ctx context.Context, cfg SSHConfig, localPath, remotePath string) error {
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
