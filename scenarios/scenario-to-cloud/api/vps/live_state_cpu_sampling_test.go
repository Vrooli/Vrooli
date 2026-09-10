package vps

import (
	"context"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/reach/reachtest"
	"scenario-to-cloud/sshidentity"
)

func liveStateTarget() identity.TargetRef {
	return identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "127.0.0.1", Port: 22, User: "root", Workdir: "/root/Vrooli"}}
}

func liveStateManifest() domain.CloudManifest {
	return domain.CloudManifest{
		Scenario: domain.ManifestScenario{ID: "test-scenario"},
		Target:   domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: "127.0.0.1", Port: 22, Workdir: "/root/Vrooli"}},
	}
}

func liveStateReach() *reachtest.Scripted {
	return &reachtest.Scripted{Answers: map[string]reachtest.Answer{
		"cat /etc/os-release": {Result: reach.Result{Stdout: "ID=ubuntu\nVERSION_ID=\"24.04\"\n"}},
		"ps aux --no-headers": {Result: reach.Result{Stdout: "root 1 0.0 0.1 1000 100 ? Ss 00:00 0:00 /sbin/init\n"}},
		"ss -tlnp":            {Result: reach.Result{Stdout: "LISTEN 0 4096 0.0.0.0:22 0.0.0.0:* users:((\"sshd\",pid=1,fd=3))\n"}},
		"uname -s":            {Result: reach.Result{Stdout: "Linux\n"}},
		"vrooli scenario status test-scenario --json":        {Result: reach.Result{Stdout: "{\"scenarios\":[]}\n"}},
		"vrooli resource status --json":                      {Result: reach.Result{Stdout: "{\"resources\":[]}\n"}},
		"cat /etc/caddy/Caddyfile":                           {Result: reach.Result{ExitCode: 1, Stderr: "No such file"}},
		"pgrep -x caddy":                                     {Result: reach.Result{ExitCode: 1}},
		"cat /root/.ssh/authorized_keys":                     {Result: reach.Result{Stdout: ""}},
		"stat -c %n -- /root/Vrooli/scenarios/test-scenario": {Result: reach.Result{Stdout: "/root/Vrooli/scenarios/test-scenario\n"}},
		"df -Pk /":                           {Result: reach.Result{Stdout: "Filesystem 1024-blocks Used Available Capacity Mounted on\n/dev/vda1 1000000 500000 500000 50% /\n"}},
		"cat /proc/meminfo":                  {Result: reach.Result{Stdout: "MemTotal: 1000000 kB\nMemAvailable: 600000 kB\nSwapTotal: 0 kB\nSwapFree: 0 kB\n"}},
		"cat /proc/loadavg":                  {Result: reach.Result{Stdout: "0.10 0.20 0.30 1/100 1234\n"}},
		"cat /proc/uptime":                   {Result: reach.Result{Stdout: "1000.00 900.00\n"}},
		"grep -c processor /proc/cpuinfo":    {Result: reach.Result{Stdout: "1\n"}},
		"grep -m 1 model name /proc/cpuinfo": {Result: reach.Result{Stdout: "model name\t: Test CPU\n"}},
		"cat /proc/stat":                     {Result: reach.Result{Stdout: "cpu  100 0 100 1000 0 0 0 0\ncpu0 100 0 100 1000 0 0 0 0\n"}},
	}}
}

// [REQ:STC-P0-024] Live state is gathered through typed reads only: every
// probe is a vrooli verb or an observation program, no argument carries
// shell syntax, and the CPU sample runs after the concurrent probes so it
// is not inflated by them.
func TestRunLiveStateInspection_ProbesAreTypedAndCPUSamplesLast(t *testing.T) {
	cpuSampleInterval = time.Millisecond
	r := liveStateReach()
	identity := sshidentity.DeploymentSSHIdentity{AuthMode: sshidentity.AuthModeUnknown, VerificationState: sshidentity.VerificationUnknown}

	result := RunLiveStateInspection(context.Background(), liveStateManifest(), identity, Prober{Reach: r, Target: liveStateTarget()})
	if !result.OK {
		t.Fatalf("expected live state OK, got error: %s", result.Error)
	}
	if result.System == nil || !result.System.SSH.Connected {
		t.Fatalf("expected system metrics with a connected transport, got %+v", result.System)
	}
	if result.System.Disk.TotalGB <= 0 {
		t.Fatalf("df header line must be skipped: %+v", result.System.Disk)
	}
	if len(result.Expected) == 0 || !result.Expected[0].DirectoryExists {
		t.Fatalf("stat probe must mark the scenario directory as existing: %+v", result.Expected)
	}
	if result.Caddy == nil || result.Caddy.Running {
		t.Fatalf("pgrep exit 1 means caddy is not running: %+v", result.Caddy)
	}

	keys := r.CallKeys()
	if len(keys) == 0 {
		t.Fatal("expected probes to be issued")
	}
	for _, c := range r.Calls {
		if c.Effectful {
			t.Fatalf("live state issued an effectful command: %+v", c)
		}
		if c.IsObservation() && !reach.ObservationPrograms[c.Program] {
			t.Fatalf("live state used a non-observation program: %+v", c)
		}
		for _, a := range c.Args {
			if strings.ContainsAny(a, "|;&$`") {
				t.Fatalf("probe argument carries shell syntax: %q", a)
			}
		}
	}
	last := keys[len(keys)-1]
	if last != "cat /proc/stat" || keys[len(keys)-2] != "cat /proc/stat" {
		t.Fatalf("expected the two /proc/stat samples last, got %v", keys[len(keys)-2:])
	}
}

// [REQ:STC-P0-024] The published probe list is what the inspection runs;
// the architecture test pins it to the observation allowlist.
func TestLiveStateProbesAreAllReads(t *testing.T) {
	for _, cmd := range LiveStateProbes(liveStateManifest(), "deploy") {
		if err := reach.ValidateCommand(cmd); err != nil {
			t.Fatalf("probe %v is invalid: %v", cmd.Argv(), err)
		}
		if cmd.Effectful {
			t.Fatalf("probe %v is effectful", cmd.Argv())
		}
		if cmd.Program == "cat" && len(cmd.Args) == 1 && strings.HasSuffix(cmd.Args[0], "authorized_keys") && !strings.HasPrefix(cmd.Args[0], "/home/deploy/") {
			t.Fatalf("authorized_keys must be read from the bound user's home: %v", cmd.Args)
		}
	}
}
