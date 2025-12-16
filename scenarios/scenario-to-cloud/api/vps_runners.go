package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type NetResolver struct{}

func (NetResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, host)
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
		"-p", strconv.Itoa(cfg.Port),
	}
	if cfg.KeyPath != "" {
		args = append(args, "-i", cfg.KeyPath)
	}
	target := fmt.Sprintf("%s@%s", cfg.User, cfg.Host)
	args = append(args, target, "--", "bash", "-lc", command)

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

type ExecSCPRunner struct{}

func (ExecSCPRunner) Copy(ctx context.Context, cfg SSHConfig, localPath, remotePath string) error {
	copyCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	args := []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-P", strconv.Itoa(cfg.Port),
	}
	if cfg.KeyPath != "" {
		args = append(args, "-i", cfg.KeyPath)
	}
	target := fmt.Sprintf("%s@%s:%s", cfg.User, cfg.Host, remotePath)
	args = append(args, localPath, target)

	cmd := exec.CommandContext(copyCtx, "scp", args...)
	return cmd.Run()
}
