package session

import (
	"context"
	"testing"

	"web-console/internal/pty"
	"web-console/internal/ptyfake"
)

// TestCreateRunsPreSpawnHookBeforeBuildingEnv pins the ordering contract the
// human-owner delegation fix depends on: for a local PTY create, the manager
// invokes the create-scoped pre-spawn hook with the fresh session id BEFORE it
// assembles the PTY environment via envForSession. A side effect the hook keys
// by session id (here: recording a token the env builder then reads back) must
// therefore be visible in the spawned environment. Minting after Create
// returned — the old bug — left this env var absent.
func TestCreateRunsPreSpawnHookBeforeBuildingEnv(t *testing.T) {
	useIsolatedSessionState(t)

	// Capture the environment the manager hands to the PTY factory.
	var capturedEnv map[string]string
	capturingFactory := func(spec pty.LaunchSpec) (pty.PTY, error) {
		capturedEnv = spec.Env
		return ptyfake.NewFakePTYWithOutput(), nil
	}

	sm := NewManagerWithFactory(capturingFactory)

	// A stand-in for the delegation holder the real env builder reads. The hook
	// writes it keyed by session id; envForSession reads it back for that id.
	stored := map[string]string{}
	sm.SetEnvForSessionFunc(func(sessionID string) map[string]string {
		env := map[string]string{}
		if tok, ok := stored[sessionID]; ok {
			env["WC_HUMAN_OWNER_DELEGATION"] = tok
		}
		return env
	})

	var hookSawID string
	ctx := WithPreSpawnHook(context.Background(), func(sessionID string) {
		hookSawID = sessionID
		stored[sessionID] = "minted-delegation"
	})

	sess, err := sm.CreateWithOptions(ctx, "", 80, 24, "", nil, t.TempDir(), false)
	if err != nil {
		t.Fatalf("CreateWithOptions: %v", err)
	}
	t.Cleanup(func() { _ = sm.Delete(context.Background(), sess.ID) })

	if hookSawID == "" {
		t.Fatal("pre-spawn hook was never invoked")
	}
	if hookSawID != sess.ID {
		t.Fatalf("pre-spawn hook saw session id %q, want the created session id %q", hookSawID, sess.ID)
	}
	if capturedEnv["WC_HUMAN_OWNER_DELEGATION"] != "minted-delegation" {
		t.Fatalf("delegation env var = %q at PTY-spawn time, want %q — the hook did not run before envForSession",
			capturedEnv["WC_HUMAN_OWNER_DELEGATION"], "minted-delegation")
	}
}

// TestNoPreSpawnHookLeavesEnvUnchanged confirms a create without a hook builds
// the ordinary env (no delegation), so the ordering seam is inert absent a
// human credential — a plain shell, service/agent session, or down authenticator.
func TestNoPreSpawnHookLeavesEnvUnchanged(t *testing.T) {
	useIsolatedSessionState(t)

	var capturedEnv map[string]string
	sm := NewManagerWithFactory(func(spec pty.LaunchSpec) (pty.PTY, error) {
		capturedEnv = spec.Env
		return ptyfake.NewFakePTYWithOutput(), nil
	})
	sm.SetEnvForSessionFunc(func(string) map[string]string {
		return map[string]string{"WC_WEB_CONSOLE_SESSION_ID": "x"}
	})

	sess, err := sm.CreateWithOptions(context.Background(), "", 80, 24, "", nil, t.TempDir(), false)
	if err != nil {
		t.Fatalf("CreateWithOptions: %v", err)
	}
	t.Cleanup(func() { _ = sm.Delete(context.Background(), sess.ID) })

	if _, ok := capturedEnv["WC_HUMAN_OWNER_DELEGATION"]; ok {
		t.Fatal("delegation env var emitted with no pre-spawn hook installed")
	}
}
