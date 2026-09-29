package codecs

import (
	"context"
	"errors"
	"flag"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

var (
	liveCodexWritePolicy = flag.Bool("live-codex-write-policy", false, "Check installed Codex within an owner-restricted workspace without a model")
	codexWriteHelper     = flag.Bool("codex-write-helper", false, "Run native write-policy child assertions")
	codexWriteWork       = flag.String("codex-write-work", "", "Native helper workspace")
	codexWriteAddress    = flag.String("codex-write-address", "", "Host network positive control")
	codexWriteSocket     = flag.String("codex-write-socket", "", "Host Unix socket positive control")
	codexWriteAllowed    = flag.Bool("codex-write-allowed", false, "Whether the helper has a source write grant")
)

func TestCodexOwnerWritePolicyOnEveryLaunchPath(t *testing.T) {
	c := NewCodexForTest()
	cfg := &domain.RunConfig{NetworkAccess: domain.NetworkAccessNone,
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected,
			WritePolicy: &domain.WorkspaceWritePolicy{Paths: []string{"src"}}}}
	control, err := c.ControlArgs(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{control,
		c.BuildArgs(c.NewState(), runner.ExecuteRequest{ResolvedConfig: cfg}),
		c.BuildContinueArgs(c.NewState(), runner.ContinueRequest{ResolvedConfig: cfg, SessionID: "retained"})} {
		joined := strings.Join(args, " ")
		if !strings.Contains(joined, `default_permissions="vrooli-network-only"`) ||
			!strings.Contains(joined, "enabled=false") || containsArg(args, "--sandbox") ||
			containsArg(args, "--dangerously-bypass-approvals-and-sandbox") {
			t.Fatalf("native network policy must compose with required outer filesystem grants: %q", args)
		}
	}
	cfg.SandboxConfig.Mode = domain.SandboxModeOff
	if _, err := c.ControlArgs(cfg); err == nil {
		t.Fatal("write grants must require the protected outer sandbox")
	}
}

func TestLiveCodexOwnerWritePolicy(t *testing.T) {
	if !*liveCodexWritePolicy {
		t.Skip("pass -live-codex-write-policy for installed CLI containment")
	}
	root := t.TempDir()
	work, home := filepath.Join(root, "work"), filepath.Join(root, "codex")
	for _, dir := range []string{work, filepath.Join(work, "src"), home} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	for path, value := range map[string]string{
		filepath.Join(work, "control"):           "owner",
		filepath.Join(work, "src", "value"):      "before",
		filepath.Join(home, "config.toml"):       "sandbox_mode='workspace-write'\n[sandbox_workspace_write]\nnetwork_access=true\nwritable_roots=['/']\n",
		filepath.Join(root, "requirements.toml"): "allowed_sandbox_modes=['read-only']\n",
	} {
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	conn, err := net.DialTimeout("tcp4", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
	socketPath := filepath.Join(root, "control.sock")
	unixListener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer unixListener.Close()
	conn, err = net.DialTimeout("unix", socketPath, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := exec.LookPath("codex")
	if err != nil {
		t.Fatal(err)
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, grant   string
		adminReadOnly bool
	}{
		{"directory", "src", false}, {"file", "src/value", false}, {"empty", "", false},
		{"admin-read-only", "src", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			grant := tc.grant
			cfg := &domain.RunConfig{NetworkAccess: domain.NetworkAccessNone,
				SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected, WritePolicy: &domain.WorkspaceWritePolicy{}}}
			// Like WSS, use a private namespace root with read-only host binds;
			// Codex may create its own metadata masks in that private root.
			outer := []string{"--unshare-user", "--unshare-pid", "--die-with-parent", "--tmpfs", "/tmp"}
			for _, path := range []string{"/bin", "/usr", "/lib", "/lib64", "/etc", userHome} {
				if _, err := os.Stat(path); err == nil {
					outer = append(outer, "--ro-bind", path, path)
				}
			}
			outer = append(outer, "--ro-bind", root, root, "--ro-bind", self, self,
				"--bind", home, home, "--proc", "/proc", "--dev", "/dev", "--chdir", work, "--setenv", "CODEX_HOME", home)
			if grant != "" {
				cfg.SandboxConfig.WritePolicy.Paths = []string{grant}
				path := filepath.Join(work, grant)
				outer = append(outer, "--bind", path, path)
			}
			if tc.adminReadOnly {
				outer = append(outer, "--tmpfs", "/etc", "--ro-bind", filepath.Join(root, "requirements.toml"), "/etc/codex/requirements.toml")
			}
			args, err := NewCodexForTest().ControlArgs(cfg)
			if err != nil {
				t.Fatal(err)
			}
			outer = append(outer, "--", binary, "sandbox")
			outer = append(outer, args...)
			outer = append(outer, "--", self, "-test.run=^TestCodexWritePolicyHelper$", "-test.v", "-codex-write-helper",
				"-codex-write-work="+work, "-codex-write-address="+listener.Addr().String(), "-codex-write-socket="+socketPath, "-codex-write-allowed="+strconv.FormatBool(grant != "" && !tc.adminReadOnly))
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			output, err := exec.CommandContext(ctx, "bwrap", outer...).CombinedOutput()
			if tc.adminReadOnly {
				if err == nil || !strings.Contains(string(output), "requirements do not allow") {
					t.Fatalf("conflicting administrator ceiling must refuse launch: %v\n%s", err, output)
				}
				return
			}
			if err != nil {
				t.Fatalf("nested native policy failed: %v\n%s", err, output)
			}
			t.Log(string(output))
		})
	}
}

func TestCodexWritePolicyHelper(t *testing.T) {
	if !*codexWriteHelper {
		t.Skip("native policy child only")
	}
	null, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("standard device access: %v", err)
	}
	if _, err := null.Write([]byte("discard")); err != nil {
		t.Fatal(err)
	}
	_ = null.Close()
	err = os.WriteFile(filepath.Join(*codexWriteWork, "src", "value"), []byte("changed"), 0600)
	if (err == nil) != *codexWriteAllowed {
		t.Fatalf("source write, allowed=%v: %v", *codexWriteAllowed, err)
	}
	if err := os.WriteFile(filepath.Join(*codexWriteWork, "control"), []byte("forbidden"), 0600); err == nil {
		t.Fatal("control write escaped")
	}
	conn, err := net.DialTimeout("tcp4", *codexWriteAddress, time.Second)
	if err == nil {
		conn.Close()
		t.Fatal("host network escaped")
	}
	info, err := os.Stat(*codexWriteSocket)
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		t.Fatalf("host socket must be visible for a meaningful denial probe: %v", err)
	}
	conn, err = net.DialTimeout("unix", *codexWriteSocket, time.Second)
	if err == nil {
		conn.Close()
		t.Fatal("host Unix control socket escaped")
	}
	if !errors.Is(err, syscall.EPERM) && !errors.Is(err, syscall.EACCES) {
		t.Fatalf("expected enforced socket denial, not an unavailable socket: %v", err)
	}
	t.Log("source grant, control denial, host TCP and visible Unix socket denial passed")
}

func TestCodexNoNetworkPolicyIsExplicitOnEveryLaunchPath(t *testing.T) {
	c := NewCodexForTest()
	cfg := &domain.RunConfig{NetworkAccess: domain.NetworkAccessNone}
	interactive, err := c.ControlArgs(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for name, args := range map[string][]string{
		"interactive": interactive,
		"execute":     c.BuildArgs(c.NewState(), runner.ExecuteRequest{ResolvedConfig: cfg}),
		"continue":    c.BuildContinueArgs(c.NewState(), runner.ContinueRequest{ResolvedConfig: cfg, SessionID: "retained-thread"}),
	} {
		t.Run(name, func(t *testing.T) {
			// Selecting workspace-write alone inherits network_access from the
			// user's configuration. An owner no-network policy must override it.
			for _, required := range []string{"workspace-write", "sandbox_workspace_write.network_access=false", "sandbox_workspace_write.writable_roots=[]", "approval_policy=never"} {
				if !containsArg(args, required) {
					t.Errorf("missing explicit %q in %q", required, args)
				}
			}
			if containsArg(args, "--dangerously-bypass-approvals-and-sandbox") {
				t.Fatalf("no-network launch bypasses sandbox: %q", args)
			}
		})
	}
}

func TestCodexExtraFlagsCannotOverrideOwnerPolicy(t *testing.T) {
	// The generic validator permits flag=value forms. Allowing raw -c would
	// therefore bypass typed model, effort, filesystem and network controls.
	for _, flag := range NewCodexForTest().Capabilities().AllowedExtraFlags {
		if flag == "-c" || flag == "--config" {
			t.Fatalf("raw config override %q must not be an extra flag", flag)
		}
	}
}

func TestClaudeControlArgsCarriesSkipPermissionsForInteractive(t *testing.T) {
	c := NewClaudeForTest()
	withSkip, err := c.ControlArgs(&domain.RunConfig{Model: "haiku", SkipPermissionPrompt: true})
	if err != nil {
		t.Fatal(err)
	}
	if !containsArg(withSkip, "--dangerously-skip-permissions") {
		t.Fatalf("skip permissions not forwarded to the interactive launch: %q", withSkip)
	}
	withoutSkip, err := c.ControlArgs(&domain.RunConfig{Model: "haiku"})
	if err != nil {
		t.Fatal(err)
	}
	if containsArg(withoutSkip, "--dangerously-skip-permissions") {
		t.Fatalf("skip permissions must not be added when unset: %q", withoutSkip)
	}
}

func TestCodexControlArgsFailsClosedOnUnsupportedEffort(t *testing.T) {
	c := NewCodexForTest()
	args, err := c.ControlArgs(&domain.RunConfig{Model: "gpt-6-luna", Effort: domain.EffortHigh})
	if err != nil {
		t.Fatal(err)
	}
	if !containsArg(args, "model_reasoning_effort=high") {
		t.Fatalf("codex high effort not translated: %q", args)
	}

	if _, err := c.ControlArgs(&domain.RunConfig{Model: "gpt-6-luna", Effort: domain.EffortMax}); err == nil ||
		!strings.Contains(err.Error(), "no native reasoning effort") {
		t.Fatalf("codex max effort must fail closed, got %v", err)
	}

	args, err = c.ControlArgs(&domain.RunConfig{Model: "gpt-6-luna"})
	if err != nil {
		t.Fatal(err)
	}
	if containsArg(args, "-c") {
		t.Fatalf("unset effort must not emit a reasoning-effort config: %q", args)
	}
}

func TestOpenCodeControlArgsUsesOnlyDocumentedProviderVariants(t *testing.T) {
	c := NewOpenCodeForTest()
	tests := []struct {
		name   string
		model  string
		effort domain.Effort
		want   bool
	}{
		{"anthropic high", "anthropic/claude-sonnet", domain.EffortHigh, true},
		{"anthropic max", "anthropic/claude-sonnet", domain.EffortMax, true},
		{"anthropic low", "anthropic/claude-sonnet", domain.EffortLow, false},
		{"openai xhigh", "openai/gpt-5", domain.EffortXHigh, true},
		{"openai max", "openai/gpt-5", domain.EffortMax, false},
		{"google low", "google/gemini", domain.EffortLow, true},
		{"google medium", "google/gemini", domain.EffortMedium, false},
		{"local provider", "ollama/qwen", domain.EffortHigh, false},
		{"missing provider", "gpt-5", domain.EffortHigh, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := c.ControlArgs(&domain.RunConfig{RunnerType: domain.RunnerTypeOpenCode, Model: tt.model, Effort: tt.effort})
			if tt.want {
				if err != nil || !containsArg(args, "--variant") {
					t.Fatalf("ControlArgs() = %q, %v; want documented variant", args, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "no documented variant") {
				t.Fatalf("ControlArgs() error = %v, want provider-domain rejection", err)
			}
		})
	}
}
