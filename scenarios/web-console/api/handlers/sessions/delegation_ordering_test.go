package sessions

import (
	"context"
	"sync"
	"testing"

	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/pty"
	"web-console/internal/ptyfake"
	"web-console/internal/sessionstore"
	"web-console/session"
)

// TestCreateMintsDelegationBeforeBuildingEnv is the ordering regression for the
// human-owner delegation fix. A plain-terminal local create carrying an
// operator credential must have its session-scoped delegation minted and stored
// BEFORE the PTY environment is assembled, so the env builder emits
// WC_HUMAN_OWNER_DELEGATION for the fresh session and the launched agent can
// forward it on attach. The original bug minted after Create returned — after
// the PTY (and its environment) had already spawned — leaving the env var
// absent and the agent-manager run owner empty.
func TestCreateMintsDelegationBeforeBuildingEnv(t *testing.T) {
	t.Setenv("WC_SESSION_STATE_ROOT", t.TempDir())

	// Capture the environment the manager hands to the PTY at spawn time.
	var mu sync.Mutex
	var capturedEnv map[string]string
	factory := func(spec pty.LaunchSpec) (pty.PTY, error) {
		mu.Lock()
		capturedEnv = spec.Env
		mu.Unlock()
		return ptyfake.NewFakePTYWithOutput(), nil
	}

	mgr := session.NewManagerWithFactory(factory)

	// Stand in for the in-memory delegation holder + env builder the real server
	// wires: the env builder emits the delegation env var only when the holder
	// has a token for that session id.
	holder := map[string]string{}
	mgr.SetEnvForSessionFunc(func(sessionID string) map[string]string {
		env := map[string]string{}
		mu.Lock()
		if tok, ok := holder[sessionID]; ok {
			env["WC_HUMAN_OWNER_DELEGATION"] = tok
		}
		mu.Unlock()
		return env
	})

	store := sessionstore.NewInMemory()
	mgr.SetStore(store)

	// The minter stands in for mintSessionDelegation: it records the derived
	// token into the holder keyed by session id, exactly as the real mint does
	// via activeSessionDelegations.Put. It asserts a non-empty session id so a
	// pre-id invocation would fail loudly.
	var mintedFor string
	var mintedCred string
	var attemptedCredentials []string
	adapter := &Adapter{
		Manager: mgr,
		Store:   store,
		Events:  events.NewLogger(20),
		Metrics: metrics.New(),
		DelegationMinter: func(_ context.Context, sessionID, rawCredential string) bool {
			if sessionID == "" {
				t.Error("delegation minter invoked with an empty session id")
				return false
			}
			mu.Lock()
			attemptedCredentials = append(attemptedCredentials, rawCredential)
			mu.Unlock()
			// The account bearer is rejected as non-human; the Cloudflare
			// assertion is the first accepted candidate.
			if rawCredential != "cf-access-human-assertion" {
				return false
			}
			mu.Lock()
			mintedFor = sessionID
			mintedCred = rawCredential
			holder[sessionID] = "minted-" + sessionID
			mu.Unlock()
			return true
		},
	}

	sess, err := adapter.Create(context.Background(), CreateInput{
		Origin:                "ui",
		ExecuteLaunchCommand:  true,
		LaunchCommand:         "codex --yolo",
		AgentType:             "codex",
		HumanOwnerCredentials: []string{"Bearer service-token", "cf-access-human-assertion"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = adapter.Delete(context.Background(), sess.ID) })

	mu.Lock()
	defer mu.Unlock()
	if mintedFor != sess.ID {
		t.Fatalf("delegation minted for session %q, want the created session %q", mintedFor, sess.ID)
	}
	// The account bearer was refused, then the Cloudflare assertion succeeded.
	if mintedCred != "cf-access-human-assertion" {
		t.Fatalf("accepted credential = %q, want the Cf-Access assertion", mintedCred)
	}
	if len(attemptedCredentials) != 2 || attemptedCredentials[0] != "Bearer service-token" || attemptedCredentials[1] != "cf-access-human-assertion" {
		t.Fatalf("credential attempts = %v, want rejected bearer followed by accepted Cloudflare assertion", attemptedCredentials)
	}
	if got := capturedEnv["WC_HUMAN_OWNER_DELEGATION"]; got != "minted-"+sess.ID {
		t.Fatalf("delegation env var = %q at PTY-spawn time, want %q — the mint did not run before the env was built",
			got, "minted-"+sess.ID)
	}
}

func TestCreateStopsAfterFirstAcceptedCredential(t *testing.T) {
	t.Setenv("WC_SESSION_STATE_ROOT", t.TempDir())

	mgr := session.NewManagerWithFactory(ptyfake.NewFactory())
	store := sessionstore.NewInMemory()
	mgr.SetStore(store)
	var attempts []string
	var holderMu sync.Mutex
	holder := map[string]string{}
	mgr.SetEnvForSessionFunc(func(sessionID string) map[string]string {
		holderMu.Lock()
		defer holderMu.Unlock()
		if token := holder[sessionID]; token != "" {
			return map[string]string{"WC_HUMAN_OWNER_DELEGATION": token}
		}
		return nil
	})
	adapter := &Adapter{
		Manager: mgr, Store: store, Events: events.NewLogger(5), Metrics: metrics.New(),
		DelegationMinter: func(_ context.Context, sessionID, credential string) bool {
			attempts = append(attempts, credential)
			if credential != "account-token" {
				return false
			}
			holderMu.Lock()
			holder[sessionID] = "delegated-for-" + sessionID
			holderMu.Unlock()
			return true
		},
	}
	sess, err := adapter.Create(context.Background(), CreateInput{
		Origin: "ui", Shell: "/bin/sh", Backend: "persistent",
		HumanOwnerCredentials: []string{"account-token", "cloudflare-fallback"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = adapter.Delete(context.Background(), sess.ID) })
	if len(attempts) != 1 || attempts[0] != "account-token" {
		t.Fatalf("credential attempts = %v, want only the first successful account token", attempts)
	}
}

// TestCreateWithoutCredentialsMintsNothing confirms the mint is inert for a
// create that carries no human credential (a plain shell or a service/agent
// create): no delegation is minted and the env var is absent, exactly the
// prior behaviour.
func TestCreateWithoutCredentialsMintsNothing(t *testing.T) {
	t.Setenv("WC_SESSION_STATE_ROOT", t.TempDir())

	var capturedEnv map[string]string
	mgr := session.NewManagerWithFactory(func(spec pty.LaunchSpec) (pty.PTY, error) {
		capturedEnv = spec.Env
		return ptyfake.NewFakePTYWithOutput(), nil
	})
	mgr.SetEnvForSessionFunc(func(string) map[string]string { return map[string]string{} })
	store := sessionstore.NewInMemory()
	mgr.SetStore(store)

	minted := false
	adapter := &Adapter{
		Manager: mgr,
		Store:   store,
		Events:  events.NewLogger(20),
		Metrics: metrics.New(),
		DelegationMinter: func(context.Context, string, string) bool {
			minted = true
			return true
		},
	}

	sess, err := adapter.Create(context.Background(), CreateInput{Origin: "ui"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = adapter.Delete(context.Background(), sess.ID) })

	if minted {
		t.Fatal("delegation minted for a create with no human credentials")
	}
	if _, ok := capturedEnv["WC_HUMAN_OWNER_DELEGATION"]; ok {
		t.Fatal("delegation env var emitted for a create with no human credentials")
	}
}
