package vps

import (
	"context"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/ssh"
	"scenario-to-cloud/sshidentity"
	"strings"
	"sync"
	"testing"
)

type recordingSSHRunner struct {
	mu    sync.Mutex
	calls []string
}

func (r *recordingSSHRunner) Run(ctx context.Context, cfg ssh.Config, command string, opts ssh.RunOptions) (ssh.Result, error) {
	r.mu.Lock()
	r.calls = append(r.calls, command)
	r.mu.Unlock()

	switch {
	case strings.Contains(command, "cat /etc/os-release"):
		return ssh.Result{Stdout: "ID=ubuntu\nVERSION_ID=\"24.04\"\n", ExitCode: 0}, nil
	case strings.Contains(command, "ps aux --no-headers"):
		return ssh.Result{Stdout: "root 1 0.0 0.1 1000 100 ? Ss 00:00 0:00 /sbin/init\n", ExitCode: 0}, nil
	case strings.Contains(command, "ss -tlnp"):
		return ssh.Result{Stdout: "LISTEN 0 4096 0.0.0.0:22 0.0.0.0:* users:((\"sshd\",pid=1,fd=3))\n", ExitCode: 0}, nil
	case strings.Contains(command, "echo ok"):
		return ssh.Result{Stdout: "ok\n", ExitCode: 0}, nil
	case strings.Contains(command, "scenario status"):
		return ssh.Result{Stdout: "{\"scenarios\":[]}\n", ExitCode: 0}, nil
	case strings.Contains(command, "resource status"):
		return ssh.Result{Stdout: "{\"resources\":[]}\n", ExitCode: 0}, nil
	case strings.Contains(command, "cat /etc/caddy/Caddyfile"):
		return ssh.Result{Stdout: "", ExitCode: 0}, nil
	case strings.Contains(command, "pgrep -x caddy"):
		return ssh.Result{Stdout: "stopped\n", ExitCode: 0}, nil
	case strings.Contains(command, "cat ~/.ssh/authorized_keys"):
		return ssh.Result{Stdout: "", ExitCode: 0}, nil
	case strings.Contains(command, "test -d"):
		return ssh.Result{Stdout: "scenario:test-scenario:exists\n", ExitCode: 0}, nil
	case strings.Contains(command, "df -Pk /"):
		return ssh.Result{Stdout: "/dev/vda1 1000000 500000 500000 50% /\n", ExitCode: 0}, nil
	case strings.Contains(command, "cat /proc/meminfo"):
		return ssh.Result{Stdout: "MemTotal: 1000000 kB\nMemAvailable: 600000 kB\nSwapTotal: 0 kB\nSwapFree: 0 kB\n", ExitCode: 0}, nil
	case strings.Contains(command, "cat /proc/loadavg"):
		return ssh.Result{Stdout: "0.10 0.20 0.30 1/100 1234\n", ExitCode: 0}, nil
	case strings.Contains(command, "cat /proc/uptime"):
		return ssh.Result{Stdout: "1000.00 900.00\n", ExitCode: 0}, nil
	case strings.Contains(command, "grep -c processor"):
		return ssh.Result{Stdout: "1\n", ExitCode: 0}, nil
	case strings.Contains(command, "grep 'model name'"):
		return ssh.Result{Stdout: "model name\t: Test CPU\n", ExitCode: 0}, nil
	case strings.Contains(command, "for i in 1 2 3 4; do cat /proc/stat"):
		return ssh.Result{Stdout: "cpu  100 0 100 1000 0 0 0 0\ncpu  101 0 101 1099 0 0 0 0\ncpu  102 0 102 1198 0 0 0 0\ncpu  103 0 103 1297 0 0 0 0\n", ExitCode: 0}, nil
	default:
		return ssh.Result{Stdout: "", ExitCode: 0}, nil
	}
}

func TestRunLiveStateInspection_CPUSamplingRunsAfterConcurrentCommands(t *testing.T) {
	t.Parallel()

	runner := &recordingSSHRunner{}
	manifest := domain.CloudManifest{
		Scenario: domain.ManifestScenario{ID: "test-scenario"},
		Target: domain.ManifestTarget{
			Type: "vps",
			VPS: &domain.ManifestVPS{
				Host:    "127.0.0.1",
				Port:    22,
				User:    "root",
				Workdir: "/root/Vrooli",
			},
		},
	}
	identity := sshidentity.DeploymentSSHIdentity{
		AuthMode:          sshidentity.AuthModeUnknown,
		VerificationState: sshidentity.VerificationUnknown,
	}

	result := RunLiveStateInspection(context.Background(), manifest, identity, runner)
	if !result.OK {
		t.Fatalf("expected live state OK, got error: %s", result.Error)
	}
	if result.System == nil {
		t.Fatalf("expected system metrics")
	}

	runner.mu.Lock()
	defer runner.mu.Unlock()
	if len(runner.calls) == 0 {
		t.Fatalf("expected commands to be executed")
	}

	last := runner.calls[len(runner.calls)-1]
	if !strings.Contains(last, "for i in 1 2 3 4; do cat /proc/stat") {
		t.Fatalf("expected last command to be smoothed cpuusage command, got: %s", last)
	}
}

func TestCPUUsageCommandForLiveState_UpgradesProcStatSampling(t *testing.T) {
	t.Parallel()

	base := "cat /proc/stat | head -1; sleep 1; cat /proc/stat | head -1"
	got := cpuUsageCommandForLiveState(base)
	wantSubstring := "for i in 1 2 3 4; do cat /proc/stat"
	if !strings.Contains(got, wantSubstring) {
		t.Fatalf("expected upgraded command to contain %q, got %q", wantSubstring, got)
	}
}
