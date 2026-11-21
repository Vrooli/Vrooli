package sessions

import (
	"context"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"

	"web-console/internal/backend"
	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/pty"
	"web-console/internal/ptyfake"
	"web-console/internal/sessionstore"
	"web-console/session"
)

func TestConnectRecoverMintsForMatchingVerifiedOwnerBeforeReplacementSpawn(t *testing.T) {
	t.Setenv("WC_SESSION_STATE_ROOT", t.TempDir())

	principal := identity.Principal{
		Kind: identity.ActorHuman, Subject: "account:alice", Verified: true,
		Source: identity.SourceCloudflareAccess, ExpiresAt: time.Now().Add(time.Hour),
	}
	ctx := identity.WithPrincipal(context.Background(), principal)
	store := sessionstore.NewInMemory()
	var mu sync.Mutex
	holder := map[string]string{}
	capturedEnv := map[string]map[string]string{}
	mgr := session.NewManagerWithFactory(func(spec pty.LaunchSpec) (pty.PTY, error) {
		mu.Lock()
		capturedEnv[spec.SessionID] = spec.Env
		mu.Unlock()
		return ptyfake.NewFakePTYWithOutput(), nil
	})
	mgr.SetStore(store)
	mgr.SetEnvForSessionFunc(func(sessionID string) map[string]string {
		mu.Lock()
		defer mu.Unlock()
		if token := holder[sessionID]; token != "" {
			return map[string]string{"WC_HUMAN_OWNER_DELEGATION": token}
		}
		return nil
	})

	var mintedSessionID string
	var mintedCredential string
	var mintCalls int
	adapter := &Adapter{
		Manager: mgr, Store: store,
		Events: events.NewLogger(10), Metrics: metrics.New(),
		AgentHistoryPresent: func(sessionstore.Metadata) bool { return true },
		DelegationMinter: func(_ context.Context, sessionID, credential string) bool {
			mu.Lock()
			defer mu.Unlock()
			mintCalls++
			mintedSessionID = sessionID
			mintedCredential = credential
			if credential != "Bearer account-token" {
				return false
			}
			holder[sessionID] = "delegation-for-" + sessionID
			return true
		},
	}
	old, err := adapter.Create(ctx, CreateInput{
		Shell: "/bin/sh", Cols: 80, Rows: 24, Backend: string(backend.Persistent),
		AgentType: "codex", Origin: "ui",
	})
	if err != nil {
		t.Fatalf("create source session: %v", err)
	}
	replacementID := ""
	t.Cleanup(func() {
		for _, id := range []string{old.ID, replacementID} {
			if id != "" {
				_ = mgr.Delete(context.Background(), id)
			}
		}
	})
	if err := store.UpdateAgentInfo(ctx, old.ID, sessionstore.AgentInfo{
		AgentType: sessionstore.AgentCodex, AgentSessionID: "codex-session-1",
	}); err != nil {
		t.Fatalf("save source agent identity: %v", err)
	}
	if err := store.MarkArchived(ctx, old.ID, time.Now().UTC()); err != nil {
		t.Fatalf("archive source session: %v", err)
	}
	oldRow, err := store.Get(ctx, old.ID)
	if err != nil || oldRow.Owner != principal.OwnerSubject() {
		t.Fatalf("source owner = %q, err=%v; want canonical verified owner %q", oldRow.Owner, err, principal.OwnerSubject())
	}

	handler := NewConnectHandler(Deps{Service: adapter})
	req := connect.NewRequest(&sessionsv1.RecoverRequest{Id: old.ID})
	req.Header().Set("Authorization", "Bearer account-token")
	req.Header().Set("Cf-Access-Jwt-Assertion", "cloudflare-fallback")
	resp, err := handler.Recover(ctx, req)
	if err != nil {
		t.Fatalf("Connect Recover: %v", err)
	}
	newID := resp.Msg.GetNewSessionId()
	if newID == "" || newID == old.ID {
		t.Fatalf("replacement session id = %q, source=%q", newID, old.ID)
	}
	replacementID = newID
	if mintedSessionID != newID || mintedCredential != "Bearer account-token" || mintCalls != 1 {
		t.Fatalf("minted session=%q credential=%q calls=%d; want replacement %q with first successful account credential", mintedSessionID, mintedCredential, mintCalls, newID)
	}
	mu.Lock()
	newEnv := capturedEnv[newID]
	mu.Unlock()
	if got := newEnv["WC_HUMAN_OWNER_DELEGATION"]; got != "delegation-for-"+newID {
		t.Fatalf("replacement PTY delegation = %q; mint did not precede environment construction", got)
	}
	newRow, err := store.Get(ctx, newID)
	if err != nil {
		t.Fatalf("read replacement metadata: %v", err)
	}
	if newRow.Owner != principal.OwnerSubject() {
		t.Fatalf("replacement owner = %q, want %q", newRow.Owner, principal.OwnerSubject())
	}
}

func TestRecoverDoesNotMintForMissingOrDifferentVerifiedOwner(t *testing.T) {
	tests := []struct {
		name        string
		principal   *identity.Principal
		credential  string
		expectCalls int
	}{
		{name: "missing principal", credential: "Bearer account-token"},
		{name: "different verified human", principal: &identity.Principal{Kind: identity.ActorHuman, Subject: "account:bob", Verified: true, Source: identity.SourceCloudflareAccess, ExpiresAt: time.Now().Add(time.Hour)}, credential: "Bearer account-token"},
		{name: "service principal", principal: &identity.Principal{Kind: identity.ActorService, Subject: "service:worker", Verified: true, Source: identity.SourceScenarioAuthenticator, ExpiresAt: time.Now().Add(time.Hour)}, credential: "Bearer account-token"},
		{name: "unverified matching subject", principal: &identity.Principal{Kind: identity.ActorHuman, Subject: "account:alice", Verified: false, Source: identity.SourceCloudflareAccess, ExpiresAt: time.Now().Add(time.Hour)}, credential: "Bearer account-token"},
		{name: "matching owner without credentials", principal: &identity.Principal{Kind: identity.ActorHuman, Subject: "account:alice", Verified: true, Source: identity.SourceCloudflareAccess, ExpiresAt: time.Now().Add(time.Hour)}},
		{name: "matching owner with missing expiry", principal: &identity.Principal{Kind: identity.ActorHuman, Subject: "account:alice", Verified: true, Source: identity.SourceCloudflareAccess}, credential: "Bearer account-token"},
		{name: "matching owner with expired proof", principal: &identity.Principal{Kind: identity.ActorHuman, Subject: "account:alice", Verified: true, Source: identity.SourceCloudflareAccess, ExpiresAt: time.Now().Add(-time.Minute)}, credential: "Bearer account-token"},
		{name: "matching owner credential refused", principal: &identity.Principal{Kind: identity.ActorHuman, Subject: "account:alice", Verified: true, Source: identity.SourceCloudflareAccess, ExpiresAt: time.Now().Add(time.Hour)}, credential: "Bearer rejected-token", expectCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("WC_SESSION_STATE_ROOT", t.TempDir())
			owner := identity.Principal{Kind: identity.ActorHuman, Subject: "account:alice", Verified: true, Source: identity.SourceCloudflareAccess, ExpiresAt: time.Now().Add(time.Hour)}
			ownerCtx := identity.WithPrincipal(context.Background(), owner)
			store := sessionstore.NewInMemory()
			mgr := session.NewManagerWithFactory(ptyfake.NewFactory())
			mgr.SetStore(store)
			adapter := &Adapter{
				Manager: mgr, Store: store, Events: events.NewLogger(5), Metrics: metrics.New(),
				AgentHistoryPresent: func(sessionstore.Metadata) bool { return true },
			}
			mintCalls := 0
			adapter.DelegationMinter = func(context.Context, string, string) bool {
				mintCalls++
				return false
			}
			old, err := adapter.Create(ownerCtx, CreateInput{
				Shell: "/bin/sh", Cols: 80, Rows: 24, Backend: string(backend.Persistent),
				AgentType: "codex", Origin: "ui",
			})
			if err != nil {
				t.Fatalf("create source: %v", err)
			}
			replacementID := ""
			t.Cleanup(func() {
				for _, id := range []string{old.ID, replacementID} {
					if id != "" {
						_ = mgr.Delete(context.Background(), id)
					}
				}
			})
			if err := store.UpdateAgentInfo(ownerCtx, old.ID, sessionstore.AgentInfo{AgentType: sessionstore.AgentCodex, AgentSessionID: "codex-session-1"}); err != nil {
				t.Fatalf("save source agent identity: %v", err)
			}
			if err := store.MarkArchived(ownerCtx, old.ID, time.Now().UTC()); err != nil {
				t.Fatalf("archive source: %v", err)
			}

			recoverCtx := context.Background()
			if tt.principal != nil {
				recoverCtx = identity.WithPrincipal(recoverCtx, *tt.principal)
			}
			req := connect.NewRequest(&sessionsv1.RecoverRequest{Id: old.ID})
			if tt.credential != "" {
				req.Header().Set("Authorization", tt.credential)
			}
			resp, err := NewConnectHandler(Deps{Service: adapter}).Recover(recoverCtx, req)
			if err != nil {
				t.Fatalf("recovery should preserve existing recovery behavior: %v", err)
			}
			replacementID = resp.Msg.GetNewSessionId()
			if mintCalls != tt.expectCalls {
				t.Fatalf("delegation mint attempts = %d, want %d", mintCalls, tt.expectCalls)
			}
		})
	}
}
