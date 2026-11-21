package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/identity"
	handlerSessions "web-console/handlers/sessions"
	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/pty"
	"web-console/internal/ptyfake"
	"web-console/internal/sessionstore"
	"web-console/session"
)

func TestReservedAdapterCreateDeliversRunIdentityOnlyToCreatedPTY(t *testing.T) {
	t.Setenv("WC_SESSION_STATE_ROOT", t.TempDir())
	type launch struct {
		id  string
		env map[string]string
	}
	var mu sync.Mutex
	var launches []launch
	mgr := session.NewManagerWithFactory(func(spec pty.LaunchSpec) (pty.PTY, error) {
		mu.Lock()
		copyEnv := make(map[string]string, len(spec.Env))
		for key, value := range spec.Env {
			copyEnv[key] = value
		}
		launches = append(launches, launch{id: spec.SessionID, env: copyEnv})
		mu.Unlock()
		return ptyfake.NewFakePTYWithOutput(), nil
	})
	store := sessionstore.NewInMemory()
	mgr.SetStore(store)
	mgr.SetEnvForSessionFunc(defaultSessionEnv)
	adapter := &handlerSessions.Adapter{Manager: mgr, Store: store, Events: events.NewLogger(20), Metrics: metrics.New()}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorService, Subject: "agent-manager", Verified: true,
		Source: identity.SourceScenarioAuthenticator, ExpiresAt: time.Now().Add(time.Hour),
	})
	id := uuid.NewString()
	created, err := adapter.Create(ctx, handlerSessions.CreateInput{
		RequestedSessionID: id, AgentRunIdentityToken: "signed-child-token",
		Origin: "programmatic", Shell: "/bin/sh", Backend: "standard",
	})
	if err != nil {
		t.Fatalf("reserved adapter create: %v", err)
	}
	t.Cleanup(func() { _ = adapter.Delete(context.Background(), created.ID) })
	plain, err := adapter.Create(ctx, handlerSessions.CreateInput{Origin: "ui", Shell: "/bin/sh", Backend: "standard"})
	if err != nil {
		t.Fatalf("plain adapter create: %v", err)
	}
	t.Cleanup(func() { _ = adapter.Delete(context.Background(), plain.ID) })
	mu.Lock()
	defer mu.Unlock()
	if len(launches) != 2 {
		t.Fatalf("PTY launches=%d, want 2", len(launches))
	}
	byID := map[string]map[string]string{}
	for _, item := range launches {
		byID[item.id] = item.env
	}
	if got := byID[id]["VROOLI_AGENT_IDENTITY_TOKEN"]; got != "signed-child-token" {
		t.Fatalf("reserved PTY identity=%q, want the child token", got)
	}
	if _, ok := byID[plain.ID]["VROOLI_AGENT_IDENTITY_TOKEN"]; ok {
		t.Fatal("plain PTY inherited the reserved child identity")
	}
	if _, ok := handlerSessions.TakeSessionRunIdentity(id); ok {
		t.Fatal("reserved child identity remained in the handoff after PTY environment consumption")
	}
}

func TestReservedAdapterCreateFailureClearsRunIdentityHandoff(t *testing.T) {
	t.Setenv("WC_SESSION_STATE_ROOT", t.TempDir())
	mgr := session.NewManagerWithFactory(func(pty.LaunchSpec) (pty.PTY, error) { return nil, errors.New("fixture PTY failure") })
	store := sessionstore.NewInMemory()
	mgr.SetStore(store)
	mgr.SetEnvForSessionFunc(defaultSessionEnv)
	adapter := &handlerSessions.Adapter{Manager: mgr, Store: store, Events: events.NewLogger(5), Metrics: metrics.New()}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorService, Subject: "agent-manager", Verified: true,
		Source: identity.SourceScenarioAuthenticator, ExpiresAt: time.Now().Add(time.Hour),
	})
	id := uuid.NewString()
	if _, err := adapter.Create(ctx, handlerSessions.CreateInput{
		RequestedSessionID: id, AgentRunIdentityToken: "signed-child-token",
		Origin: "programmatic", Shell: "/bin/sh", Backend: "standard",
	}); err == nil {
		t.Fatal("expected PTY failure")
	}
	if _, ok := handlerSessions.TakeSessionRunIdentity(id); ok {
		t.Fatal("failed Create left a child identity in the handoff")
	}
}
