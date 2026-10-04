package runner

import (
	"agent-manager/internal/domain"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/effortauthority"
	isolation "github.com/vrooli/vrooli/packages/nativeisolation"
	"strings"
	"testing"
	"time"
)

func finiteLaunchFixture() (isolation.Manifest, *domain.RunConfig, uuid.UUID, LaunchRequest) {
	h := strings.Repeat("a", 64)
	id := uuid.New()
	deadline := time.Now().Add(time.Hour)
	repo := "/srv/work/repository"
	m := isolation.Manifest{Enabled: true, Backend: "linux-systemd-service-v1", SystemdRun: isolation.Executable{Path: "/usr/bin/systemd-run", SHA256: h}, Systemctl: isolation.Executable{Path: "/usr/bin/systemctl", SHA256: h}, RootFS: "/var/lib/vrooli/finite-auth02/image", WorkspaceSource: "/var/lib/vrooli/finite-auth02/worktree", RootFSDigest: h, NativeUID: 1234, NativeGID: 1234, Isolation: &effortauthority.NativeUIDIsolation{NativeUID: 1234, ProtectedPaths: []string{"/var/lib/issuer/key", "/var/lib/issuer/ledger"}}, Bindings: map[string]isolation.Binding{"canary": {PolicyDigest: h, ProfileDigest: h, Repository: repo, Deadline: deadline}}, EnvKeys: []string{"PATH", "HOME", "VROOLI_AGENT_IDENTITY_TOKEN"}, NativeBinaries: map[string]string{"/usr/bin/codex": h}}
	cfg := &domain.RunConfig{Timeout: time.Minute, SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}, Admission: &domain.RunAdmission{Effort: &effortauthority.Binding{PolicyID: "canary", PolicyDigest: h, Deadline: deadline}, EffortIntent: &effortauthority.Intent{Effect: "run.create", ProfileDigest: h, Repository: repo}}}
	req := BuildEnvWrappedLaunchRequest("CODEX_AGENT_TAG", "/usr/bin/codex", []string{"--native-option"}, id.String(), "disposable prompt", []string{"PATH=/usr/bin:/bin", "HOME=" + repo + "/.native-home", "VROOLI_AGENT_IDENTITY_TOKEN=exact-parent"}, repo)
	req.RunID = id
	return m, cfg, id, req
}
func TestFiniteSelectorUsesConcreteIsolationBeforeOffMode(t *testing.T) {
	m, cfg, id, req := finiteLaunchFixture()
	f, e := NewFiniteNativeFactory(m)
	if e != nil {
		t.Fatal(e)
	}
	s := NewLauncherSelector(nil, nil)
	s.SetFiniteNativeFactory(f)
	picked := s.PickFor(context.Background(), id, cfg, nil, nil)
	l, ok := picked.(*finiteNativeLauncher)
	if !ok {
		t.Fatalf("finite off mode chose %T", picked)
	}
	r := l.request
	r.Command = req.Command
	r.Args = req.Args
	r.Env = req.Env
	r.WorkingDir = req.WorkingDir
	p, e := l.runtime.Plan(context.Background(), r)
	if e != nil || !strings.Contains(strings.Join(p.Args, " "), "CODEX_AGENT_TAG="+id.String()) {
		t.Fatalf("actual native env shim route was not preserved: %v", e)
	}
}
func TestFiniteMissingDisabledAndWrongRouteNeverFallback(t *testing.T) {
	for _, kind := range []string{"missing", "disabled", "child", "recovery", "protected", "policy-files", "write-policy"} {
		t.Run(kind, func(t *testing.T) {
			m, cfg, id, req := finiteLaunchFixture()
			s := NewLauncherSelector(nil, nil)
			if kind != "missing" {
				if kind == "disabled" {
					m.Enabled = false
				}
				f, e := NewFiniteNativeFactory(m)
				if e != nil {
					t.Fatal(e)
				}
				s.SetFiniteNativeFactory(f)
			}
			switch kind {
			case "child":
				cfg.Admission.EffortIntent.Effect = "run.child"
			case "recovery":
				cfg.Admission.EffortIntent.Effect = "run.recover"
			case "protected":
				cfg.SandboxConfig.Mode = domain.SandboxModeProtected
			case "write-policy":
				cfg.SandboxConfig.WritePolicy = &domain.WorkspaceWritePolicy{Paths: []string{"/srv/work/repository"}}
			case "policy-files":
				req.PolicyFiles = []PolicyFile{{}}
			}
			picked := s.PickFor(context.Background(), id, cfg, nil, nil)
			if _, ok := picked.(*HostLauncher); ok {
				t.Fatal("finite host fallback")
			}
			if _, e := picked.Launch(context.Background(), req); e == nil {
				t.Fatal("unqualified finite route started")
			}
		})
	}
}
func TestOrdinaryOffModeStillUsesHost(t *testing.T) {
	s := NewLauncherSelector(nil, nil)
	if _, ok := s.PickFor(context.Background(), uuid.New(), &domain.RunConfig{SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}}, nil, nil).(*HostLauncher); !ok {
		t.Fatal(fmt.Sprintf("ordinary route changed"))
	}
}
