// Routing tests for OpenCodeRunner ↔ launcherSelector wiring.
//
// Mirrors codex_runner_routing_test.go's structure: launcher_selector_test.go
// pins routing logic; these tests pin the per-runner wiring (constructor
// stores launchers, SetSandboxLauncherFactory plumbs through, Execute
// consults the selector instead of calling exec.Command directly).

package runner

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestOpenCodeRunner_NewWithLaunchers_WiresSelector(t *testing.T) {
	host := &recordingLauncher{tag: "host"}
	factory := &stubFactory{launcher: &recordingLauncher{tag: "sandbox"}}

	r, err := NewOpenCodeRunnerWithLaunchers(host, factory)
	if err != nil {
		t.Fatalf("NewOpenCodeRunnerWithLaunchers: %v", err)
	}
	if r.selector == nil {
		t.Fatal("selector field is nil; constructor did not wire it")
	}
	if got := r.selector.HostLauncher(); got != host {
		t.Errorf("selector.HostLauncher() = %v; want injected host", got)
	}
	if got := r.selector.SandboxFactory(); got != factory {
		t.Errorf("selector.SandboxFactory() = %v; want injected factory", got)
	}
}

func TestOpenCodeRunner_NewOpenCodeRunner_WiresDefaultSelector(t *testing.T) {
	r, err := NewOpenCodeRunner()
	if err != nil {
		t.Fatalf("NewOpenCodeRunner: %v", err)
	}
	if r.selector == nil {
		t.Fatal("default constructor did not initialise selector")
	}
	if r.selector.HostLauncher() == nil {
		t.Fatal("default constructor did not set host launcher")
	}
	if r.selector.SandboxFactory() != nil {
		t.Errorf("default constructor should leave SandboxFactory nil; got %v", r.selector.SandboxFactory())
	}
}

func TestOpenCodeRunner_SetSandboxLauncherFactory_PlumbsThroughToSelector(t *testing.T) {
	r := NewTestOpenCodeRunner()
	if r.selector.SandboxFactory() != nil {
		t.Fatal("test runner should start without a sandbox factory")
	}
	factory := &stubFactory{launcher: &recordingLauncher{tag: "sandbox"}}
	r.SetSandboxLauncherFactory(factory)
	if got := r.selector.SandboxFactory(); got != factory {
		t.Errorf("after SetSandboxLauncherFactory, selector.SandboxFactory() = %v; want injected factory", got)
	}
}

// TestOpenCodeRunner_ProtectedExecuteRoutesThroughSandboxLauncher proves
// the load-bearing protected-mode invariant: opencode agent processes
// run inside the sandbox boundary, never on the host, when protected
// mode is requested and a sandbox factory is wired.
func TestOpenCodeRunner_ProtectedExecuteRoutesThroughSandboxLauncher(t *testing.T) {
	host := &recordingLauncher{tag: "host"}
	sandboxL := &recordingLauncher{tag: "sandbox"}
	factory := &stubFactory{launcher: sandboxL}

	r := NewTestOpenCodeRunner()
	r.selector = newLauncherSelector(host, factory)
	r.available = true

	sandboxID := uuid.New()
	cfg := domain.DefaultRunConfig()
	cfg.SandboxConfig = &domain.SandboxConfig{Mode: domain.SandboxModeProtected}
	req := ExecuteRequest{
		RunID:          uuid.New(),
		ResolvedConfig: cfg,
		SandboxID:      &sandboxID,
		Prompt:         "hello",
		WorkingDir:     "/tmp",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = r.Execute(ctx, req)

	if sandboxL.callCount() != 1 {
		t.Errorf("sandbox launcher called %d times; want 1", sandboxL.callCount())
	}
	if host.callCount() != 0 {
		t.Errorf("host launcher called %d times in protected mode; want 0", host.callCount())
	}
}

func TestOpenCodeRunner_TrackingExecuteRoutesThroughHostLauncher(t *testing.T) {
	host := &recordingLauncher{tag: "host"}
	sandboxL := &recordingLauncher{tag: "sandbox"}
	factory := &stubFactory{launcher: sandboxL}

	r := NewTestOpenCodeRunner()
	r.selector = newLauncherSelector(host, factory)
	r.available = true

	cfg := domain.DefaultRunConfig()
	cfg.SandboxConfig = &domain.SandboxConfig{Mode: domain.SandboxModeTracking}
	req := ExecuteRequest{
		RunID:          uuid.New(),
		ResolvedConfig: cfg,
		Prompt:         "hello",
		WorkingDir:     "/tmp",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = r.Execute(ctx, req)

	if host.callCount() != 1 {
		t.Errorf("host launcher called %d times; want 1", host.callCount())
	}
	if sandboxL.callCount() != 0 {
		t.Errorf("sandbox launcher called %d times in tracking mode; want 0", sandboxL.callCount())
	}
}

// TestOpenCodeRunner_LaunchRequestUsesEnvShim asserts that the OpenCode
// streaming Execute path (unlike codex's wrapper-fallback) goes through
// the env-wrapped LaunchRequest builder so the OPENCODE_AGENT_TAG env arg
// surfaces in /proc/<pid>/cmdline for the reconciler.
func TestOpenCodeRunner_LaunchRequestUsesEnvShim(t *testing.T) {
	host := &recordingLauncher{tag: "host"}
	r := NewTestOpenCodeRunner()
	r.selector = newLauncherSelector(host, nil)
	r.available = true

	cfg := domain.DefaultRunConfig()
	cfg.SandboxConfig = &domain.SandboxConfig{Mode: domain.SandboxModeTracking}
	req := ExecuteRequest{
		RunID:          uuid.New(),
		ResolvedConfig: cfg,
		Prompt:         "hello",
		WorkingDir:     "/tmp",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = r.Execute(ctx, req)

	if host.callCount() != 1 {
		t.Fatalf("expected exactly one launch; got %d", host.callCount())
	}
	if got := host.calls[0].Command; got != "env" {
		t.Errorf("LaunchRequest.Command = %q; want %q (env shim)", got, "env")
	}
	// The first arg after env should be the OPENCODE_AGENT_TAG=<tag>
	// pair; the second should be the runner binary path.
	args := host.calls[0].Args
	if len(args) < 2 {
		t.Fatalf("LaunchRequest.Args has %d elements; want at least 2 (tag + binary)", len(args))
	}
	if got := args[0][:len("OPENCODE_AGENT_TAG=")]; got != "OPENCODE_AGENT_TAG=" {
		t.Errorf("LaunchRequest.Args[0] = %q; want OPENCODE_AGENT_TAG=<...> prefix", args[0])
	}
}
