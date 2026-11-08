package runtimesupervisor

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

type updateHeartbeatStore struct {
	Store
	heartbeats chan struct{}
}

func (s *updateHeartbeatStore) HeartbeatSupervisorSession(ctx context.Context, id string, ttl time.Duration) (scenarioruntime.SupervisorSession, error) {
	result, err := s.Store.HeartbeatSupervisorSession(ctx, id, ttl)
	if err == nil {
		select {
		case s.heartbeats <- struct{}{}:
		default:
		}
	}
	return result, err
}

func TestManagedUpdateRecoveryDoesNotBlockFleetRenewal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	entered := make(chan struct{})
	heartbeats := make(chan struct{}, 32)
	cfg := Config{DBPath: filepath.Join(t.TempDir(), "runtime.db"), RenewInterval: 10 * time.Millisecond, Stderr: io.Discard, HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot", SessionID: "session"}}, ManagedUpdateReconcile: func(ctx context.Context) error { close(entered); <-ctx.Done(); return ctx.Err() }, StoreFactory: func(ctx context.Context, c scenarioruntime.Config) (Store, error) {
		s, e := scenarioruntime.NewSQLiteStore(ctx, c)
		return &updateHeartbeatStore{Store: s, heartbeats: heartbeats}, e
	}}
	done := make(chan error, 1)
	go func() { done <- Run(ctx, cfg) }()
	select {
	case <-entered:
	case err := <-done:
		t.Fatalf("supervisor exited before recovery worker: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("recovery worker did not start")
	}
	for i := 0; i < 3; i++ {
		select {
		case <-heartbeats:
		case err := <-done:
			t.Fatalf("blocked recovery stopped supervisor: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("blocked update recovery delayed fleet renewal")
		}
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("update worker did not honor shutdown")
	}
}

func TestManagedUpdateRecoveryContainsOwnerPanic(t *testing.T) {
	s := New(Config{ManagedUpdateReconcile: func(context.Context) error { panic("fixture") }})
	if err := s.reconcileManagedUpdatesOnce(context.Background()); err == nil {
		t.Fatal("recovery owner panic escaped containment")
	}
}
