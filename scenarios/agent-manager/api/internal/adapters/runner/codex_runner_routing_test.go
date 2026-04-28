// Routing tests for CodexRunner ↔ launcherSelector wiring.
//
// launcher_selector_test.go pins the routing rules themselves (which
// launcher Pick returns for which input). These tests pin the runner-to-
// selector wiring: that the constructor stores the supplied launchers,
// that SetSandboxLauncherFactory plumbs through, and that Execute consults
// the selector rather than calling exec.Command directly.
//
// The full Execute path is exercised by integration_test.go; here we only
// need to prove the seam is in place.

package runner

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// recordingLauncher is a Launcher that records each Launch call and
// returns a LaunchedProcess that exits cleanly without producing output.
// Useful for asserting which launcher was selected in a routing test.
type recordingLauncher struct {
	tag    string
	mu     sync.Mutex
	calls  []LaunchRequest
	failOn bool // when true, Launch returns an error
}

func (l *recordingLauncher) Launch(ctx context.Context, req LaunchRequest) (LaunchedProcess, error) {
	l.mu.Lock()
	l.calls = append(l.calls, req)
	l.mu.Unlock()
	if l.failOn {
		return nil, errors.New("recordingLauncher: refusing to launch (failOn=true)")
	}
	return newNoopLaunchedProcess(), nil
}

func (l *recordingLauncher) callCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.calls)
}

// noopLaunchedProcess is a LaunchedProcess that exits immediately with
// empty stdout/stderr so the runner's stream loop terminates cleanly.
type noopLaunchedProcess struct {
	stdoutR *io.PipeReader
	stdoutW *io.PipeWriter
	stderrR *io.PipeReader
	stderrW *io.PipeWriter
	done    chan struct{}
	killed  atomic.Bool
}

func newNoopLaunchedProcess() *noopLaunchedProcess {
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	p := &noopLaunchedProcess{
		stdoutR: stdoutR,
		stdoutW: stdoutW,
		stderrR: stderrR,
		stderrW: stderrW,
		done:    make(chan struct{}),
	}
	// Close both pipes immediately so the runner sees clean EOF.
	_ = stdoutW.Close()
	_ = stderrW.Close()
	close(p.done)
	return p
}

func (p *noopLaunchedProcess) Stdout() io.Reader      { return p.stdoutR }
func (p *noopLaunchedProcess) Stderr() io.Reader      { return p.stderrR }
func (p *noopLaunchedProcess) ResetIdleTimer()        {}
func (p *noopLaunchedProcess) TimedOut() bool         { return false }
func (p *noopLaunchedProcess) Kill()                  { p.killed.Store(true) }
func (p *noopLaunchedProcess) Signal(_ time.Duration) {}
func (p *noopLaunchedProcess) Wait() error            { <-p.done; return nil }
func (p *noopLaunchedProcess) PID() int               { return 0 }

func TestCodexRunner_NewWithLaunchers_WiresSelector(t *testing.T) {
	host := &recordingLauncher{tag: "host"}
	factory := &stubFactory{launcher: &recordingLauncher{tag: "sandbox"}}

	r, err := NewCodexRunnerWithLaunchers(host, factory)
	if err != nil {
		t.Fatalf("NewCodexRunnerWithLaunchers: %v", err)
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

func TestCodexRunner_NewCodexRunner_WiresDefaultSelector(t *testing.T) {
	// NewCodexRunner has to always return a usable runner so the
	// default constructor used by main.go (and by NewCodexRunner)
	// doesn't crash on resource-codex absence: but the selector should
	// always be present.
	r, err := NewCodexRunner()
	if err != nil {
		t.Fatalf("NewCodexRunner: %v", err)
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

func TestCodexRunner_SetSandboxLauncherFactory_PlumbsThroughToSelector(t *testing.T) {
	r := NewTestCodexRunner()
	if r.selector.SandboxFactory() != nil {
		t.Fatal("test runner should start without a sandbox factory")
	}
	factory := &stubFactory{launcher: &recordingLauncher{tag: "sandbox"}}
	r.SetSandboxLauncherFactory(factory)
	if got := r.selector.SandboxFactory(); got != factory {
		t.Errorf("after SetSandboxLauncherFactory, selector.SandboxFactory() = %v; want injected factory", got)
	}
}

// TestCodexRunner_ProtectedExecuteRoutesThroughSandboxLauncher is the
// load-bearing assertion for the runner-fork: when the streaming Execute
// path runs with SandboxConfig.Mode == Protected and a sandbox factory is
// wired, the sandbox launcher is invoked and the host launcher is not.
//
// This proves agent processes never bypass the sandbox boundary in
// protected mode regardless of which runner spawned them.
func TestCodexRunner_ProtectedExecuteRoutesThroughSandboxLauncher(t *testing.T) {
	host := &recordingLauncher{tag: "host"}
	sandboxL := &recordingLauncher{tag: "sandbox"}
	factory := &stubFactory{launcher: sandboxL}

	r := NewTestCodexRunner()
	r.selector = newLauncherSelector(host, factory)
	r.available = true // bypass the IsAvailable gate so Execute proceeds

	sandboxID := uuid.New()
	cfg := domain.DefaultRunConfig()
	cfg.SandboxConfig = &domain.SandboxConfig{Mode: domain.SandboxModeProtected}
	req := ExecuteRequest{
		RunID:          uuid.New(),
		ResolvedConfig: cfg,
		SandboxID:      &sandboxID,
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

func TestCodexRunner_TrackingExecuteRoutesThroughHostLauncher(t *testing.T) {
	host := &recordingLauncher{tag: "host"}
	sandboxL := &recordingLauncher{tag: "sandbox"}
	factory := &stubFactory{launcher: sandboxL}

	r := NewTestCodexRunner()
	r.selector = newLauncherSelector(host, factory)
	r.available = true

	cfg := domain.DefaultRunConfig()
	cfg.SandboxConfig = &domain.SandboxConfig{Mode: domain.SandboxModeTracking}
	req := ExecuteRequest{
		RunID:          uuid.New(),
		ResolvedConfig: cfg,
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

// TestCodexRunner_WrapperFallbackAlsoRoutesThroughSelector covers the
// non-JSON wrapper path (executeWithWrapper) which uses the resource-
// codex wrapper rather than the codex CLI directly. The seam is the
// same; this test exists to guard against the wrapper path drifting.
func TestCodexRunner_WrapperFallbackAlsoRoutesThroughSelector(t *testing.T) {
	host := &recordingLauncher{tag: "host"}
	r := NewTestCodexRunner()
	r.selector = newLauncherSelector(host, nil)
	r.available = true
	r.useJSONStream = false // force the wrapper path
	r.binaryPath = "/fake/resource-codex"

	cfg := domain.DefaultRunConfig()
	cfg.SandboxConfig = &domain.SandboxConfig{Mode: domain.SandboxModeTracking}
	req := ExecuteRequest{
		RunID:          uuid.New(),
		ResolvedConfig: cfg,
		WorkingDir:     "/tmp",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = r.Execute(ctx, req)

	if host.callCount() != 1 {
		t.Errorf("wrapper fallback called launcher %d times; want 1 (host)", host.callCount())
	}
	// The wrapper path passes the binary directly without an env shim,
	// so the LaunchRequest's Command should be the resource-codex path
	// rather than `env`. This catches regressions where the wrapper
	// path accidentally adopts the env-wrapped builder.
	if host.calls[0].Command != "/fake/resource-codex" {
		t.Errorf("wrapper LaunchRequest.Command = %q; want %q", host.calls[0].Command, "/fake/resource-codex")
	}
}
