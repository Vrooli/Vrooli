package procsampler

import (
	"context"
	"testing"
)

// fakeDocker is a DockerFallback that returns a fixed owner for specific pids.
type fakeDocker struct{ byPID map[int]string }

func (f fakeDocker) Attribute(_ context.Context, pid int) string { return f.byPID[pid] }

func TestAttributor_DirectCwdMatch(t *testing.T) {
	a := NewAttributor(nil)
	samples := []ProcessSample{
		{PID: 100, PPID: 1, Comm: "node", Cwd: "/home/u/Vrooli/scenarios/system-monitor/api"},
		{PID: 101, PPID: 1, Comm: "bash", Cwd: "/home/u"},
	}
	a.Attribute(context.Background(), samples)

	if samples[0].Owner != "system-monitor" {
		t.Fatalf("cwd match: got owner %q, want system-monitor", samples[0].Owner)
	}
	if samples[1].Owner != OwnerUnknown {
		t.Fatalf("non-scenario proc: got owner %q, want unknown", samples[1].Owner)
	}
}

func TestAttributor_BinaryNameMatch(t *testing.T) {
	a := NewAttributor(nil)
	samples := []ProcessSample{
		{PID: 200, PPID: 1, Comm: "security-health-api", Cwd: "/opt/somewhere"},
		{PID: 201, PPID: 1, Comm: "node", Cmdline: "/usr/bin/node /home/u/Vrooli/scenarios/web-console/api/index.js"},
	}
	a.Attribute(context.Background(), samples)

	if samples[0].Owner != "security-health" {
		t.Fatalf("binary name: got owner %q, want security-health", samples[0].Owner)
	}
	if samples[1].Owner != "web-console" {
		t.Fatalf("cmdline scenarios path: got owner %q, want web-console", samples[1].Owner)
	}
}

func TestAttributor_PPIDWalkInheritsOwner(t *testing.T) {
	a := NewAttributor(nil)
	// security-health-api (300) spawns osv-scanner (301) which spawns a helper
	// (302). Both children should attribute to security-health via the PPID walk.
	samples := []ProcessSample{
		{PID: 300, PPID: 1, Comm: "security-health-api"},
		{PID: 301, PPID: 300, Comm: "osv-scanner"},
		{PID: 302, PPID: 301, Comm: "sh"},
	}
	a.Attribute(context.Background(), samples)

	for _, s := range samples {
		if s.Owner != "security-health" {
			t.Fatalf("pid %d: got owner %q, want security-health (PPID walk)", s.PID, s.Owner)
		}
	}
}

func TestAttributor_UnknownFallbackAndDockerBranch(t *testing.T) {
	docker := fakeDocker{byPID: map[int]string{402: "whisper"}}
	a := NewAttributor(docker)
	samples := []ProcessSample{
		{PID: 400, PPID: 1, Comm: "sshd"},   // genuinely unknown
		{PID: 401, PPID: 400, Comm: "bash"}, // child of unknown -> unknown
		{PID: 402, PPID: 1, Comm: "python"}, // resolved only by docker fallback
	}
	a.Attribute(context.Background(), samples)

	if samples[0].Owner != OwnerUnknown {
		t.Fatalf("host proc: got %q, want unknown", samples[0].Owner)
	}
	if samples[1].Owner != OwnerUnknown {
		t.Fatalf("child of unknown: got %q, want unknown", samples[1].Owner)
	}
	if samples[2].Owner != "whisper" {
		t.Fatalf("docker fallback: got %q, want whisper", samples[2].Owner)
	}
}

func TestAttributor_PPIDCycleGuard(t *testing.T) {
	a := NewAttributor(nil)
	// Pathological self/cyclic parent links must not loop forever.
	samples := []ProcessSample{
		{PID: 500, PPID: 501, Comm: "a"},
		{PID: 501, PPID: 500, Comm: "b"},
	}
	a.Attribute(context.Background(), samples)
	for _, s := range samples {
		if s.Owner != OwnerUnknown {
			t.Fatalf("cycle pid %d: got %q, want unknown", s.PID, s.Owner)
		}
	}
}

// TestAttributor_SpawnerWinsOverScannedDirCwd is the regression guard for the
// attribution-precedence fix: security-health's reconcile runs osv-scanner with
// cwd INSIDE the scenario being scanned (e.g. scenarios/agent-manager). A
// cwd-first rule would misattribute that CPU to the innocent scanned scenario;
// ancestry must win so the osv-scanner attributes to security-health (its
// spawner) — the osv-scanner→security-health link the plan targets.
func TestAttributor_SpawnerWinsOverScannedDirCwd(t *testing.T) {
	a := NewAttributor(nil)
	samples := []ProcessSample{
		{PID: 600, PPID: 1, Comm: "security-health", Cmdline: "./security-health-api", Cwd: "/home/u/Vrooli/scenarios/security-health/api"},
		// child osv-scanner, cwd in a DIFFERENT scenario it is scanning:
		{PID: 601, PPID: 600, Comm: "osv-scanner", Cmdline: "osv-scanner scan --format json -r .", Cwd: "/home/u/Vrooli/scenarios/agent-manager"},
		{PID: 602, PPID: 600, Comm: "osv-scanner", Cmdline: "osv-scanner scan --format json -r .", Cwd: "/home/u/Vrooli/scenarios/app-monitor"},
	}
	a.Attribute(context.Background(), samples)
	for _, s := range samples {
		if s.Owner != "security-health" {
			t.Fatalf("pid %d (cwd=%s): got owner %q, want security-health (spawner, not scanned dir)", s.PID, s.Cwd, s.Owner)
		}
	}
}
