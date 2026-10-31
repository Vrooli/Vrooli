// Integration coverage for the copy-driver (macOS-representative) sandbox
// shape as seen through the agent-manager protected-launch stack.
//
// The unit tests elsewhere pin the pieces in isolation: the selector's
// containment-gap warn uses a stub reporter (launcher_selector_test.go), and
// the identity-layout launch (host-path workdir, no translation) is pinned in
// sandbox_launcher_test.go. This file ties them together against the REAL
// WorkspaceSandboxProvider talking to a workspace-sandbox simulator reporting
// the copy-driver layout (workspacePath == mergedDir, pathIllusion=false,
// containment backend "none"):
//
// Protected selection must refuse this uncontained provider before a process
// starts. Explicit tracking mode retains provenance and identity-layout launch.
package sandbox

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// integrationSink records emitted run events so the test can assert on the
// containment-gap warning. Implements runner.EventSink.
type integrationSink struct {
	mu     sync.Mutex
	events []*domain.RunEvent
}

func (s *integrationSink) Emit(e *domain.RunEvent) error {
	s.mu.Lock()
	s.events = append(s.events, e)
	s.mu.Unlock()
	return nil
}

func (s *integrationSink) Close() error { return nil }

func (s *integrationSink) warnContains(needle string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.events {
		if e == nil {
			continue
		}
		if data, ok := e.Data.(*domain.LogEventData); ok && data != nil {
			if data.Level == "warn" && strings.Contains(data.Message, needle) {
				return true
			}
		}
	}
	return false
}

func TestCopyDriverSandboxShape_ProtectedRefusesAndTrackingLaunches(t *testing.T) {
	const host = "/var/lib/workspace-sandbox/sb-copy-e2e/merged"

	mock := newSandboxTestServer(4242)
	mock.hostMergedDir = host
	mock.identityLayout = true // copy-driver: workspacePath == mergedDir, backend none
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	sandboxID := uuid.New()

	// The provider is both the launcher factory and the containment reporter.
	selector := runner.NewLauncherSelector(runner.NewHostLauncher(), provider)

	sink := &integrationSink{}
	cfg := &domain.RunConfig{
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
	}
	runID := uuid.New()
	protected := selector.PickFor(context.Background(), runID, cfg, &sandboxID, sink)
	if _, err := protected.Launch(context.Background(), runner.LaunchRequest{Command: "agent"}); err == nil {
		t.Fatal("protected launch must refuse an uncontained provider")
	}
	mock.mu.Lock()
	started := mock.startProcessBody != nil
	mock.mu.Unlock()
	if started {
		t.Fatal("refused launch started a process")
	}
	cfg.SandboxConfig.Mode = domain.SandboxModeTracking

	// --- (1) Selection: real ContainmentFor drives the degradation warn. ---
	picked := selector.PickFor(context.Background(), runID, cfg, &sandboxID, sink)
	sbLauncher, ok := picked.(*SandboxLauncher)
	if !ok {
		t.Fatalf("PickFor returned %T; want *SandboxLauncher for explicit tracking", picked)
	}

	// The provider must independently report the uncontained shape.
	cont, ok := provider.ContainmentFor(context.Background(), sandboxID)
	if !ok || cont == nil {
		t.Fatalf("ContainmentFor ok=%v cont=%v; want a report", ok, cont)
	}
	if cont.Backend != "none" || cont.Level != "none" {
		t.Errorf("containment = backend %q level %q; want none/none", cont.Backend, cont.Level)
	}
	if len(cont.MissingProtectedEnforcements()) != len(cont.Enforcements)+2 {
		// backend none provides nothing, so both protected enforcements are missing.
		t.Errorf("missing enforcements = %v; want both protected enforcements", cont.MissingProtectedEnforcements())
	}

	// The gap warn must name both missing protected enforcements and the backend.
	for _, needle := range []string{
		runner.EnforcementFilesystemWriteContainment,
		runner.EnforcementNetworkDeny,
		"backend=none",
	} {
		if !sink.warnContains(needle) {
			t.Errorf("degradation warn missing %q; events=%v", needle, sink.events)
		}
	}

	// --- (2) Launch: identity layout keeps the host merged path (no translation). ---
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := sbLauncher.Launch(ctx, runner.LaunchRequest{
		Command:    "claude",
		WorkingDir: host,
		Env:        []string{"VROOLI_SANDBOX_MERGED=" + host},
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())

	mock.mu.Lock()
	body := mock.startProcessBody
	mock.mu.Unlock()
	if got, _ := body["workingDir"].(string); got != host {
		t.Errorf("workingDir = %q; want %q (identity layout: no translation)", got, host)
	}
	envAny, _ := body["env"].(map[string]any)
	if got, _ := envAny["VROOLI_SANDBOX_MERGED"].(string); got != host {
		t.Errorf("env VROOLI_SANDBOX_MERGED = %q; want %q (identity layout)", got, host)
	}
	if lvl, _ := body["isolationLevel"].(string); lvl != "vrooli-aware" {
		t.Errorf("isolationLevel = %q; want vrooli-aware", lvl)
	}

	go func() {
		time.Sleep(40 * time.Millisecond)
		mock.markExited(remoteExitInfo{ExitCode: 0})
	}()
	if err := proc.Wait(); err != nil {
		t.Errorf("Wait: %v", err)
	}
}
