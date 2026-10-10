package nativeisolation

import (
	"context"
	"github.com/vrooli/api-core/effortauthority"
	"github.com/vrooli/vrooli/internal/nativeisolation/probeprotocol"
	"strings"
	"testing"
	"time"
)

func probeFixture() (ProbeSpec, time.Time) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	h := strings.Repeat("a", 64)
	m := Manifest{Enabled: true, Backend: "linux-systemd-service-v1", SystemdRun: Executable{"/usr/bin/systemd-run", h}, Systemctl: Executable{"/usr/bin/systemctl", h}, RootFS: "/var/lib/vrooli/image", WorkspaceSource: "/var/lib/vrooli/worktree", RootFSDigest: h, NativeUID: 1234, NativeGID: 1234, Isolation: &effortauthority.NativeUIDIsolation{NativeUID: 1234, NativeProbePID: 999, Services: map[string]effortauthority.ProtectedProcess{"issuer": {PID: 7, ExecutableSHA256: h}, "prompt-manager": {PID: 8, ExecutableSHA256: h}, "agent-manager": {PID: 9, ExecutableSHA256: h}}, ProtectedPaths: []string{"/var/lib/issuer/ledger", "/var/lib/issuer/key"}}, Bindings: map[string]Binding{"canary": {PolicyDigest: h, ProfileDigest: h, Repository: "/srv/work/repository", Deadline: now.Add(8 * time.Hour)}}, EnvKeys: []string{"PATH", "HOME"}, NativeBinaries: map[string]string{"/usr/bin/codex": h}}
	return ProbeSpec{Manifest: m, HelperSHA256: strings.Repeat("a", 64), Purpose: ContinuingWitness, Deadline: now.Add(time.Hour)}, now
}

func TestProbeBootstrapIsFixedNonmodelAndBounded(t *testing.T) {
	s, now := probeFixture()
	p, e := s.BootstrapPlan(context.Background(), now)
	if e != nil {
		t.Fatal(e)
	}
	if p.Command != s.Manifest.SystemdRun.Path || p.Unit != s.unit() || strings.Join(p.Args[len(p.Args)-4:], " ") != "-- "+ProbeHelperPath+" --until "+s.Deadline.Format(time.RFC3339Nano) {
		t.Fatal("arbitrary bootstrap command")
	}
	args := strings.Join(p.Args, " ")
	for _, required := range []string{"--property=NoNewPrivileges=yes", "--property=CapabilityBoundingSet=", "--property=AmbientCapabilities=", "--property=Restart=no", "--property=IPAddressDeny=any", "--property=RestrictAddressFamilies=AF_INET AF_INET6", "--property=BindPaths=" + s.Manifest.WorkspaceSource + ":" + s.repository(), "--property=ReadWritePaths=" + s.repository(), "--property=WorkingDirectory=" + s.repository()} {
		if !strings.Contains(args, required) {
			t.Fatal("missing isolation property", required)
		}
	}
	for _, forbidden := range []string{"IPAddressAllow=", "--setenv=", "codex", "claude", "--pipe", "--wait"} {
		if strings.Contains(args, forbidden) {
			t.Fatal("nonmodel probe expanded", forbidden)
		}
	}
	if len(p.Env) != 2 || p.Request.Command != "" {
		t.Fatal("bootstrap accepts native request authority")
	}
	for _, change := range []func(*ProbeSpec){func(s *ProbeSpec) { s.Purpose = FirstQualification }, func(s *ProbeSpec) { s.Deadline = now }, func(s *ProbeSpec) { s.Deadline = now.Add(9 * time.Hour) }, func(s *ProbeSpec) { s.Manifest.NativeBinaries[ProbeHelperPath] = s.HelperSHA256 }, func(s *ProbeSpec) { s.HelperSHA256 = "wrong" }, func(s *ProbeSpec) { s.Manifest.Enabled = false }} {
		copy, _ := cloneSpec(s)
		change(&copy)
		if _, e := copy.BootstrapPlan(context.Background(), now); e == nil {
			t.Fatal("invalid bootstrap accepted")
		}
	}
	s.Purpose = FirstQualification
	s.Deadline = now.Add(5 * time.Minute)
	s.FirstObservation = &FirstObservationContract{CanaryRoot: "/var/lib/vrooli-isolation-canary", CanaryID: strings.Repeat("a", 32), WorkspaceSHA256: probeprotocol.MarkerDigest(strings.Repeat("a", 32), "workspace"), DeniedSHA256: probeprotocol.MarkerDigest(strings.Repeat("a", 32), "denied")}
	if _, e := s.BootstrapPlan(context.Background(), now); e != nil {
		t.Fatal("bounded first probe refused", e)
	}
}
