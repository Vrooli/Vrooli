package sessions

import (
	"context"
	"sync"
	"testing"

	"web-console/internal/backend"
	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/pty"
	"web-console/internal/ptyfake"
	"web-console/internal/sessionstore"
	"web-console/session"
)

func TestAccountBoundOpenCodeCreatePersistsAccountAndInjectsSessionCredentialBeforeSpawn(t *testing.T) {
	t.Setenv("WC_SESSION_STATE_ROOT", t.TempDir())

	var mu sync.Mutex
	bindings := map[string]string{}
	var capturedEnv map[string]string
	factory := func(spec pty.LaunchSpec) (pty.PTY, error) {
		mu.Lock()
		capturedEnv = spec.Env
		mu.Unlock()
		return ptyfake.NewFakePTYWithOutput(), nil
	}
	manager := session.NewManagerWithFactory(factory)
	store := sessionstore.NewInMemory()
	manager.SetStore(store)
	manager.SetEnvForSessionFunc(func(sessionID string) map[string]string {
		mu.Lock()
		defer mu.Unlock()
		return map[string]string{"OPENCODE_CONFIG_CONTENT": bindings[sessionID]}
	})
	adapter := &Adapter{
		Manager: manager, Store: store, Events: events.NewLogger(20), Metrics: metrics.New(),
		PaidCodingCredential: func() (string, error) { return "session-secret", nil },
		RegisterPaidCodingSession: func(sessionID, accountID, token string) {
			mu.Lock()
			defer mu.Unlock()
			bindings[sessionID] = accountID + ":" + token
		},
		RemovePaidCodingSession: func(sessionID string) {
			mu.Lock()
			defer mu.Unlock()
			delete(bindings, sessionID)
		},
	}

	created, err := adapter.Create(context.Background(), CreateInput{
		Shell: "/bin/sh", Cols: 80, Rows: 24, Backend: string(backend.Standard), Origin: "ui",
		AgentType: "opencode", LaunchCommand: "opencode", ExecuteLaunchCommand: true,
		BusinessAccountID: "acct-selected",
	})
	if err != nil {
		t.Fatalf("create account-bound OpenCode session: %v", err)
	}
	t.Cleanup(func() { _ = adapter.Delete(context.Background(), created.ID) })

	mu.Lock()
	gotEnv := capturedEnv["OPENCODE_CONFIG_CONTENT"]
	registered := bindings[created.ID]
	mu.Unlock()
	if registered != "acct-selected:session-secret" {
		t.Fatalf("session binding = %q", registered)
	}
	if gotEnv != registered {
		t.Fatalf("OpenCode PTY config = %q, want the pre-spawn session binding %q", gotEnv, registered)
	}
	metadata, err := store.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("load created session metadata: %v", err)
	}
	if metadata.BusinessAccountID != "acct-selected" || metadata.AgentType != sessionstore.AgentOpenCode {
		t.Fatalf("persisted paid session metadata = %+v", metadata)
	}
}
