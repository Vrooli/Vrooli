package vps

import (
	"context"
	"errors"
	"testing"
	"time"

	"scenario-to-cloud/reach"
	"scenario-to-cloud/reach/reachtest"
	"scenario-to-cloud/sshidentity"
)

func TestRunSystemMetricsDebug_ParsesLinuxMetrics(t *testing.T) {
	cpuSampleInterval = time.Millisecond
	r := &reachtest.Scripted{Answers: map[string]reachtest.Answer{
		"cat /etc/os-release":                {Result: reach.Result{Stdout: "ID=ubuntu\nVERSION_ID=\"24.04\"\n"}},
		"uname -s":                           {Result: reach.Result{Stdout: "Linux\n"}},
		"cat /root/.ssh/authorized_keys":     {Result: reach.Result{Stdout: ""}},
		"df -Pk /":                           {Result: reach.Result{Stdout: "Filesystem 1024-blocks Used Available Capacity Mounted on\n/dev/sda1 209715200 83886080 125829120 40% /\n"}},
		"cat /proc/meminfo":                  {Result: reach.Result{Stdout: "MemTotal: 8192000 kB\nMemFree: 1024000 kB\nMemAvailable: 4096000 kB\nSwapTotal: 2097152 kB\nSwapFree: 1048576 kB\n"}},
		"cat /proc/loadavg":                  {Result: reach.Result{Stdout: "0.20 0.15 0.10 1/100 1000\n"}},
		"cat /proc/uptime":                   {Result: reach.Result{Stdout: "12345.67 89012.34\n"}},
		"grep -c processor /proc/cpuinfo":    {Result: reach.Result{Stdout: "4\n"}},
		"grep -m 1 model name /proc/cpuinfo": {Result: reach.Result{Stdout: "model name\t: Test CPU\n"}},
		"cat /proc/stat":                     {Result: reach.Result{Stdout: "cpu  100 0 100 1000 0 0 0 0\n"}},
	}}

	result := RunSystemMetricsDebug(context.Background(), sshidentity.DeploymentSSHIdentity{}, Prober{Reach: r, Target: liveStateTarget()})

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
	for _, c := range result.Commands {
		if c.Command == "" {
			t.Fatalf("debug entry %s lacks its argv rendering", c.ID)
		}
	}
}

func TestRunSystemMetricsDebug_TransportFailureSetsNotOK(t *testing.T) {
	cpuSampleInterval = time.Millisecond
	r := &reachtest.Scripted{Answers: map[string]reachtest.Answer{
		"cat /etc/os-release": {Result: reach.Result{Stdout: "ID=ubuntu\n"}},
		"uname -s":            {Err: &reach.Error{Kind: reach.KindTargetOffline, Err: errors.New("connection refused")}},
	}}

	result := RunSystemMetricsDebug(context.Background(), sshidentity.DeploymentSSHIdentity{}, Prober{Reach: r, Target: liveStateTarget()})
	if result.OK {
		t.Fatal("expected result.OK=false when the reachability probe fails")
	}
	if result.Error == "" {
		t.Fatal("expected error message when the reachability probe fails")
	}
}
