package vps

import (
	"context"
	"testing"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/ssh"
)

func TestRunSystemMetricsDebug_ParsesLinuxMetrics(t *testing.T) {
	t.Parallel()

	runner := &testSSHRunner{
		responses: map[string]ssh.Result{
			"/etc/os-release": {Stdout: "ID=ubuntu\nVERSION_ID=\"24.04\"\n", ExitCode: 0},
			"echo ok":         {Stdout: "ok\n", ExitCode: 0},
			"authorized_keys": {Stdout: "", ExitCode: 0},
			"df -Pk /":        {Stdout: "/dev/sda1 209715200 83886080 125829120 40% /\n", ExitCode: 0},
			"/proc/meminfo":   {Stdout: "MemTotal: 8192000 kB\nMemFree: 1024000 kB\nMemAvailable: 4096000 kB\nSwapTotal: 2097152 kB\nSwapFree: 1048576 kB\n", ExitCode: 0},
			"/proc/loadavg":   {Stdout: "0.20 0.15 0.10 1/100 1000\n", ExitCode: 0},
			"/proc/uptime":    {Stdout: "12345.67 89012.34\n", ExitCode: 0},
			"/proc/cpuinfo":   {Stdout: "4\n", ExitCode: 0}, // used by grep -c processor
			"model name":      {Stdout: "model name\t: Test CPU\n", ExitCode: 0},
			"/proc/stat":      {Stdout: "cpu  100 0 100 1000 0 0 0 0\ncpu  200 0 200 1100 0 0 0 0\n", ExitCode: 0},
		},
	}

	manifest := domain.CloudManifest{
		Target: domain.ManifestTarget{
			Type: "vps",
			VPS: &domain.ManifestVPS{
				Host: "203.0.113.10",
			},
		},
	}

	result := RunSystemMetricsDebug(context.Background(), manifest, runner)

	if !result.OK {
		t.Fatalf("result.OK=false, error=%q", result.Error)
	}
	if result.Collector != "linux" {
		t.Fatalf("collector=%q, want linux", result.Collector)
	}
	if result.OSID != "ubuntu" {
		t.Fatalf("os_id=%q, want ubuntu", result.OSID)
	}
	if result.System.Memory.TotalMB != 8000 {
		t.Fatalf("memory total=%d, want 8000", result.System.Memory.TotalMB)
	}
	if result.System.Memory.UsedMB != 4000 {
		t.Fatalf("memory used=%d, want 4000", result.System.Memory.UsedMB)
	}
	if result.System.Disk.TotalGB <= 0 {
		t.Fatalf("disk total gb not parsed: %+v", result.System.Disk)
	}
	if len(result.Commands) == 0 {
		t.Fatal("expected command debug entries")
	}
}

func TestRunSystemMetricsDebug_SSHFailureSetsNotOK(t *testing.T) {
	t.Parallel()

	runner := &testSSHRunner{
		responses: map[string]ssh.Result{
			"/etc/os-release": {Stdout: "ID=ubuntu\n", ExitCode: 0},
		},
		errs: map[string]error{
			"echo ok": context.DeadlineExceeded,
		},
	}

	manifest := domain.CloudManifest{
		Target: domain.ManifestTarget{
			Type: "vps",
			VPS:  &domain.ManifestVPS{Host: "203.0.113.10"},
		},
	}

	result := RunSystemMetricsDebug(context.Background(), manifest, runner)
	if result.OK {
		t.Fatal("expected result.OK=false when ssh_ping fails")
	}
	if result.Error == "" {
		t.Fatal("expected error message when ssh_ping fails")
	}
}
