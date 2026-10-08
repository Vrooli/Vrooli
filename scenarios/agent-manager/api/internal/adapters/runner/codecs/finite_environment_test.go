package codecs

import (
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"context"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/effortauthority"
	isolation "github.com/vrooli/vrooli/packages/nativeisolation"
	"strings"
	"testing"
	"time"
)

func codecFiniteFixture() (isolation.Manifest, *domain.RunConfig, uuid.UUID, runner.LaunchRequest) {
	h := strings.Repeat("a", 64)
	id := uuid.New()
	deadline := time.Now().Add(time.Hour)
	repo := "/srv/work/repository"
	m := isolation.Manifest{Enabled: true, Backend: "linux-systemd-service-v1", SystemdRun: isolation.Executable{Path: "/usr/bin/systemd-run", SHA256: h}, Systemctl: isolation.Executable{Path: "/usr/bin/systemctl", SHA256: h}, RootFS: "/var/lib/vrooli/finite-auth02/image", WorkspaceSource: "/var/lib/vrooli/finite-auth02/worktree", RootFSDigest: h, NativeUID: 1234, NativeGID: 1234, Isolation: &effortauthority.NativeUIDIsolation{NativeUID: 1234, ProtectedPaths: []string{"/var/lib/issuer/key", "/var/lib/issuer/ledger"}}, Bindings: map[string]isolation.Binding{"canary": {PolicyDigest: h, ProfileDigest: h, Repository: repo, Deadline: deadline}}, EnvKeys: []string{"PATH", "HOME", "VROOLI_AGENT_IDENTITY_TOKEN"}, NativeBinaries: map[string]string{"/usr/bin/codex": h}}
	cfg := &domain.RunConfig{Timeout: time.Minute, SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}, Admission: &domain.RunAdmission{Effort: &effortauthority.Binding{PolicyID: "canary", PolicyDigest: h, Deadline: deadline}, EffortIntent: &effortauthority.Intent{Effect: "run.create", ProfileDigest: h, Repository: repo}}}
	req := runner.BuildEnvWrappedLaunchRequest("CODEX_AGENT_TAG", "/usr/bin/codex", []string{"--native-option"}, id.String(), "disposable prompt", []string{"PATH=/usr/bin:/bin", "HOME=" + repo + "/.native-home", "VROOLI_AGENT_IDENTITY_TOKEN=exact-parent"}, repo)
	req.RunID = id
	return m, cfg, id, req
}

func TestActualCodecFiniteEnvironmentMigration(t *testing.T) {
	t.Setenv("HOME", "/home/host-fixture")
	t.Setenv("PATH", "/host/fixture/bin")
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY"} {
		t.Setenv(key, "")
	}
	type environmentCodec interface {
		BuildEnv(string, map[string]string) []string
	}
	codecs := map[string]environmentCodec{"claude": NewClaudeForTest(), "codex": NewCodexForTest(), "opencode": NewOpenCodeForTest()}
	tags := map[string]string{"claude": "CLAUDE_CODE_AGENT_TAG", "codex": "CODEX_AGENT_TAG", "opencode": "OPENCODE_AGENT_TAG"}
	for name, codec := range codecs {
		t.Run(name, func(t *testing.T) {
			m, _, id, req := codecFiniteFixture()
			binary := "/usr/bin/" + name
			m.NativeBinaries = map[string]string{binary: strings.Repeat("a", 64)}
			req = runner.BuildEnvWrappedLaunchRequest(tags[name], binary, []string{"--native-option"}, id.String(), "disposable prompt", nil, req.WorkingDir)
			req.RunID = id
			rt, e := isolation.NewRuntime(m)
			if e != nil {
				t.Fatal(e)
			}
			env := codec.BuildEnv("fixture-tag", map[string]string{"VROOLI_AGENT_IDENTITY_TOKEN": "fixture-parent"})
			narrowed, e := rt.PrepareEnvironment(req.WorkingDir, env)
			if e != nil {
				t.Fatal(e)
			}
			joined := strings.Join(narrowed, ";")
			if strings.Contains(joined, "/home/host-fixture") || strings.Contains(joined, "/host/fixture/bin") || !strings.Contains(joined, "VROOLI_AGENT_IDENTITY_TOKEN=fixture-parent") || !strings.Contains(joined, "HOME="+req.WorkingDir+"/.native-home") {
				t.Fatal(joined)
			}
			r := isolation.Request{RunID: id.String(), PolicyID: "canary", PolicyDigest: strings.Repeat("a", 64), ProfileDigest: strings.Repeat("a", 64), Repository: req.WorkingDir, WorkingDir: req.WorkingDir, Deadline: m.Bindings["canary"].Deadline, Timeout: time.Minute, Command: req.Command, Args: req.Args, Env: narrowed}
			if _, e := rt.Plan(context.Background(), r); e != nil {
				t.Fatal(e)
			}
		})
	}
}
