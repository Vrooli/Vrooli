//go:build integration

// These tests exercise the interactive substrate against a LIVE web-console.
// Run with: go test -tags=integration ./internal/orchestration/interactive/...
// They skip cleanly when web-console cannot be resolved/reached.
package interactive

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// liveClient resolves and probes web-console, returning a real client or
// skipping the test when it is not reachable.
func liveClient(t *testing.T) *webconsole.Client {
	t.Helper()
	base := webconsole.ResolveBaseURL()
	if base == "" {
		t.Skip("web-console base URL could not be resolved; skipping live integration test")
	}
	client := webconsole.NewClient(base, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// A get on a bogus id proves reachability: NotFound means the server
	// answered; any other error (connection refused, timeout) means skip.
	_, err := client.GetSession(ctx, "00000000-0000-0000-0000-000000000000")
	if err != nil && !errors.Is(err, webconsole.ErrSessionNotFound) {
		t.Skipf("web-console not reachable at %s: %v", base, err)
	}
	return client
}

// pollScreen polls GetScreen until it contains want or the deadline passes.
func pollScreen(t *testing.T, client *webconsole.Client, sessionID, want string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last string
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		screen, err := client.Screen(ctx, sessionID, true)
		cancel()
		if err == nil {
			last = screen
			if strings.Contains(screen, want) {
				return screen
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	return last
}

// TestLive_Client_EnvInjectionAndStop drives the real Connect client: it opens
// a programmatic session whose launch command sets an env var via the inline
// prefix, verifies the var is visible in the pane (proving env injection +
// free-for-all stdin), then exercises interrupt + delete teardown.
func TestLive_Client_EnvInjectionAndStop(t *testing.T) {
	client := liveClient(t)
	ctx := context.Background()

	sessionID, err := client.CreateSession(ctx, webconsole.CreateSessionParams{
		LaunchCommand: "PHASE2_INJECTED=phase2value sh",
		Execute:       true,
		DisplayLabel:  "phase2-live-envtest",
		Backend:       "persistent",
	})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { _ = client.DeleteSession(context.Background(), sessionID) })

	// Wait for the inner shell to be ready, then read the injected env back.
	pollScreen(t, client, sessionID, "$", 10*time.Second)
	if err := client.SendText(ctx, sessionID, "echo INJECTED_IS_$PHASE2_INJECTED\n", "agent-manager:run-test"); err != nil {
		t.Fatalf("SendText: %v", err)
	}
	screen := pollScreen(t, client, sessionID, "INJECTED_IS_phase2value", 10*time.Second)
	if !strings.Contains(screen, "INJECTED_IS_phase2value") {
		t.Fatalf("env var not injected into pane; screen was:\n%s", screen)
	}

	// Interrupt (Escape then Ctrl+C) must not error against a live pane.
	if err := client.Interrupt(ctx, sessionID, "agent-manager:run-test"); err != nil {
		t.Fatalf("Interrupt: %v", err)
	}

	// Delete is the hard-kill fallback; after it the session is gone.
	if err := client.DeleteSession(ctx, sessionID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if _, err := client.GetSession(ctx, sessionID); !errors.Is(err, webconsole.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound after delete, got %v", err)
	}
	// Delete is idempotent.
	if err := client.DeleteSession(ctx, sessionID); err != nil {
		t.Fatalf("second DeleteSession should be a no-op, got %v", err)
	}
}

// TestLive_Substrate_LaunchDiscoversTranscript drives the full Substrate.Launch
// against the live server. The "agent binary" is a small script that writes a
// codex-style rollout under the agent-manager-owned CODEX_HOME (proving the
// env-prefix relocation) then execs a shell (a live interactive session). The
// substrate must create the session and discover that rollout as the run's
// transcript, then Stop must tear it down.
func TestLive_Substrate_LaunchDiscoversTranscript(t *testing.T) {
	client := liveClient(t)

	workDir := t.TempDir()
	runDir := t.TempDir()

	// A fake codex CLI: on launch it writes the rollout into $CODEX_HOME (which
	// the substrate sets to <runDir>/codex) then drops to an interactive shell.
	script := filepath.Join(t.TempDir(), "fake-codex.sh")
	scriptBody := "#!/bin/sh\n" +
		"set -e\n" +
		"day_dir=\"$CODEX_HOME/sessions/2026/07/13\"\n" +
		"mkdir -p \"$day_dir\"\n" +
		"echo '{\"type\":\"session_meta\",\"payload\":{\"id\":\"live-test\"}}' > \"$day_dir/rollout-2026-07-13T10-00-00-live.jsonl\"\n" +
		"exec sh\n"
	if err := os.WriteFile(script, []byte(scriptBody), 0o755); err != nil {
		t.Fatalf("write fake codex script: %v", err)
	}

	sub := NewSubstrate(client, fakeResolver(fakeLaunchInfo{
		rt:     domain.RunnerTypeCodex,
		tagKey: "CODEX_AGENT_TAG",
		binary: script,
	}), WithDiscoveryTimeout(15*time.Second), WithPollInterval(200*time.Millisecond), WithStopGrace(200*time.Millisecond))

	ctx := context.Background()
	res, err := sub.Launch(ctx, LaunchParams{
		RunID:      uuid.New(),
		RunnerType: domain.RunnerTypeCodex,
		Tag:        "phase2-live-run",
		WorkingDir: workDir,
		RunDir:     runDir,
	})
	if res.SessionID != "" {
		t.Cleanup(func() { _ = client.DeleteSession(context.Background(), res.SessionID) })
	}
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if res.SessionID == "" {
		t.Fatal("expected a session id")
	}
	wantPrefix := filepath.Join(runDir, "codex", "sessions")
	if !strings.HasPrefix(res.TranscriptPath, wantPrefix) {
		t.Fatalf("transcript path %q not under run-scoped CODEX_HOME %q", res.TranscriptPath, wantPrefix)
	}
	if _, err := os.Stat(res.TranscriptPath); err != nil {
		t.Fatalf("discovered transcript does not exist: %v", err)
	}
	if res.ExecutionMode != domain.ExecutionModeInteractive {
		t.Fatalf("execution mode: got %q, want interactive", res.ExecutionMode)
	}

	// Durable facts land on a run record.
	run := &domain.Run{ID: uuid.New()}
	ApplyToRun(run, res)
	if run.WebConsoleSessionID != res.SessionID || run.TranscriptPath != res.TranscriptPath {
		t.Fatalf("ApplyToRun mismatch: %+v", run)
	}

	// Stop tears the session down (interrupt + delete).
	if err := sub.Stop(ctx, res.SessionID, "agent-manager:run-test"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if _, err := client.GetSession(ctx, res.SessionID); !errors.Is(err, webconsole.ErrSessionNotFound) {
		t.Fatalf("expected session gone after Stop, got %v", err)
	}
}
