// Tests for the runner-fork routing logic in claude_code.go: ensure
// selectLauncher() picks the SandboxLauncher iff
//   - the resolved config requests Mode == Protected, AND
//   - the runner has a SandboxLauncherFactory wired, AND
//   - the ExecuteRequest carries a non-nil SandboxID.
//
// This test pins the contract documented on
// (*ClaudeCodeRunner).selectLauncher and the spec for
// execute/protected-sandbox-agent-launch.

package runner

import (
	"context"
	"sync"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// stubLauncher is a minimal Launcher used to assert which launcher was picked.
type stubLauncher struct {
	tag    string
	called bool
	mu     sync.Mutex
}

func (s *stubLauncher) Launch(ctx context.Context, req LaunchRequest) (LaunchedProcess, error) {
	s.mu.Lock()
	s.called = true
	s.mu.Unlock()
	// Returning nil here would crash the runner; tests only call selectLauncher
	// directly, never the full Execute path.
	return nil, nil
}

// stubFactory records the sandbox ID it was asked about and returns the
// configured launcher.
type stubFactory struct {
	launcher    Launcher
	calledForID *uuid.UUID
}

func (f *stubFactory) LauncherFor(sandboxID uuid.UUID) Launcher {
	id := sandboxID
	f.calledForID = &id
	return f.launcher
}

// recordingSink captures emitted log events so tests can assert on warnings.
type recordingSink struct {
	mu     sync.Mutex
	events []*domain.RunEvent
}

func (s *recordingSink) Emit(e *domain.RunEvent) error {
	s.mu.Lock()
	s.events = append(s.events, e)
	s.mu.Unlock()
	return nil
}
func (s *recordingSink) Close() error { return nil }
func (s *recordingSink) hasWarning(needle string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.events {
		if e == nil {
			continue
		}
		if msg, ok := e.Data.(*domain.LogEventData); ok && msg != nil {
			if msg.Level == "warn" && containsAny(msg.Message, needle) {
				return true
			}
		}
	}
	return false
}

func containsAny(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && (indexOf(haystack, needle) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// TestSelectLauncher_NonProtectedUsesHost asserts that without protected
// mode set, the runner picks the host launcher.
func TestSelectLauncher_NonProtectedUsesHost(t *testing.T) {
	host := &stubLauncher{tag: "host"}
	sandbox := &stubLauncher{tag: "sandbox"}
	factory := &stubFactory{launcher: sandbox}

	r := &ClaudeCodeRunner{hostLauncher: host, sandboxFactory: factory}
	cfg := &domain.RunConfig{
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeTracking},
	}
	picked := r.selectLauncher(context.Background(), ExecuteRequest{ResolvedConfig: cfg})
	if picked != host {
		t.Errorf("non-protected request picked %v; want host launcher", picked)
	}
}

// TestSelectLauncher_UnspecifiedModeUsesHost asserts the empty-mode default
// (treated as tracking by SandboxMode.Effective) routes to host.
func TestSelectLauncher_UnspecifiedModeUsesHost(t *testing.T) {
	host := &stubLauncher{tag: "host"}
	factory := &stubFactory{launcher: &stubLauncher{tag: "sandbox"}}
	r := &ClaudeCodeRunner{hostLauncher: host, sandboxFactory: factory}
	cfg := &domain.RunConfig{SandboxConfig: &domain.SandboxConfig{}} // mode=""
	picked := r.selectLauncher(context.Background(), ExecuteRequest{ResolvedConfig: cfg})
	if picked != host {
		t.Errorf("unspecified mode picked %v; want host launcher", picked)
	}
}

// TestSelectLauncher_ProtectedWithFactoryAndIDPicksSandbox is the load-bearing
// case: protected mode + factory wired + SandboxID in request → SandboxLauncher.
func TestSelectLauncher_ProtectedWithFactoryAndIDPicksSandbox(t *testing.T) {
	host := &stubLauncher{tag: "host"}
	sandboxLauncher := &stubLauncher{tag: "sandbox"}
	factory := &stubFactory{launcher: sandboxLauncher}

	r := &ClaudeCodeRunner{hostLauncher: host, sandboxFactory: factory}
	sandboxID := uuid.New()
	cfg := &domain.RunConfig{
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
	}
	picked := r.selectLauncher(context.Background(), ExecuteRequest{
		ResolvedConfig: cfg,
		SandboxID:      &sandboxID,
	})
	if picked != sandboxLauncher {
		t.Errorf("protected request with factory picked %v; want sandbox launcher", picked)
	}
	if factory.calledForID == nil || *factory.calledForID != sandboxID {
		t.Errorf("factory called with sandboxID = %v; want %v", factory.calledForID, sandboxID)
	}
}

// TestSelectLauncher_ProtectedNoFactoryFallsBackWithWarning asserts the
// graceful fallback: protected requested but no factory → host launcher,
// with a warning event emitted to the sink.
func TestSelectLauncher_ProtectedNoFactoryFallsBackWithWarning(t *testing.T) {
	host := &stubLauncher{tag: "host"}
	r := &ClaudeCodeRunner{hostLauncher: host /* no sandboxFactory */}
	sink := &recordingSink{}
	cfg := &domain.RunConfig{
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
	}
	id := uuid.New()
	picked := r.selectLauncher(context.Background(), ExecuteRequest{
		RunID:          uuid.New(),
		ResolvedConfig: cfg,
		SandboxID:      &id,
		EventSink:      sink,
	})
	if picked != host {
		t.Errorf("no-factory fallback picked %v; want host launcher", picked)
	}
	if !sink.hasWarning("no SandboxLauncherFactory") {
		t.Errorf("no warning emitted; events=%v", sink.events)
	}
}

// TestSelectLauncher_ProtectedNoSandboxIDFallsBackWithWarning verifies the
// SandboxID-missing fallback. Protected mode is requested and a factory is
// wired, but the ExecuteRequest carries no SandboxID (likely an in-place
// run misconfigured to ask for protected) — falls back to host with warn.
func TestSelectLauncher_ProtectedNoSandboxIDFallsBackWithWarning(t *testing.T) {
	host := &stubLauncher{tag: "host"}
	factory := &stubFactory{launcher: &stubLauncher{tag: "sandbox"}}
	r := &ClaudeCodeRunner{hostLauncher: host, sandboxFactory: factory}
	sink := &recordingSink{}
	cfg := &domain.RunConfig{
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
	}
	picked := r.selectLauncher(context.Background(), ExecuteRequest{
		RunID:          uuid.New(),
		ResolvedConfig: cfg,
		// SandboxID intentionally nil
		EventSink: sink,
	})
	if picked != host {
		t.Errorf("nil-SandboxID fallback picked %v; want host launcher", picked)
	}
	if !sink.hasWarning("SandboxID is nil") {
		t.Errorf("no warning emitted; events=%v", sink.events)
	}
	if factory.calledForID != nil {
		t.Error("factory should not have been consulted when SandboxID is nil")
	}
}

// TestSelectLauncher_FactoryReturnsNilFallsBackWithWarning asserts the
// factory-returned-nil case (e.g. provider mismatched to sandbox ID).
func TestSelectLauncher_FactoryReturnsNilFallsBackWithWarning(t *testing.T) {
	host := &stubLauncher{tag: "host"}
	factory := &stubFactory{launcher: nil} // explicitly nil
	r := &ClaudeCodeRunner{hostLauncher: host, sandboxFactory: factory}
	sink := &recordingSink{}
	cfg := &domain.RunConfig{
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
	}
	id := uuid.New()
	picked := r.selectLauncher(context.Background(), ExecuteRequest{
		RunID:          uuid.New(),
		ResolvedConfig: cfg,
		SandboxID:      &id,
		EventSink:      sink,
	})
	if picked != host {
		t.Errorf("factory-nil fallback picked %v; want host launcher", picked)
	}
	if !sink.hasWarning("factory returned nil") {
		t.Errorf("no warning emitted; events=%v", sink.events)
	}
}

// TestSelectLauncher_NoConfigUsesHost asserts the runner is robust when
// ExecuteRequest has no resolved config / no profile — host launcher.
// (GetConfig() returns DefaultRunConfig which has nil SandboxConfig.)
func TestSelectLauncher_NoConfigUsesHost(t *testing.T) {
	host := &stubLauncher{tag: "host"}
	factory := &stubFactory{launcher: &stubLauncher{tag: "sandbox"}}
	r := &ClaudeCodeRunner{hostLauncher: host, sandboxFactory: factory}
	picked := r.selectLauncher(context.Background(), ExecuteRequest{}) // no config
	if picked != host {
		t.Errorf("no-config request picked %v; want host launcher", picked)
	}
}
