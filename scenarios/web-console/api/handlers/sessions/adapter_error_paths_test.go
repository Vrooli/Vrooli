package sessions

import (
	"context"
	"errors"
	"testing"
	"time"

	"web-console/internal/backend"
	"web-console/internal/policy"
	"web-console/internal/sessionstore"
	"web-console/session"
)

type emptySessionManager struct{}

func (emptySessionManager) Create(context.Context, string, uint16, uint16, backend.ID, *policy.Policy) (*session.Session, error) {
	return nil, session.ErrBackendUnknown
}

func (emptySessionManager) CreateWithWorkingDir(context.Context, string, uint16, uint16, backend.ID, *policy.Policy, string) (*session.Session, error) {
	return nil, session.ErrBackendUnavailable
}

func (emptySessionManager) CreateWithOptions(context.Context, string, uint16, uint16, backend.ID, *policy.Policy, string, bool) (*session.Session, error) {
	return nil, session.ErrBackendUnavailable
}
func (emptySessionManager) Get(string) (*session.Session, bool)   { return nil, false }
func (emptySessionManager) List() []*session.Session              { return nil }
func (emptySessionManager) Delete(context.Context, string) error  { return nil }
func (emptySessionManager) Archive(context.Context, string) error { return nil }
func (emptySessionManager) RecoveryProgress() session.RecoveryProgress {
	return session.RecoveryProgress{InProgress: true, Total: 1, StartedAt: time.Now()}
}

func TestAdapterErrorAndRemotePaths(t *testing.T) {
	ctx := context.Background()
	a := &Adapter{Manager: emptySessionManager{}}
	if _, err := a.Create(ctx, CreateInput{Backend: "standard", Policy: Policy{Mode: "bad"}, HasPolicy: true}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("invalid policy: %v", err)
	}
	if _, err := a.Create(ctx, CreateInput{TargetID: "node"}); !errors.Is(err, ErrRemoteUnavailable) {
		t.Fatalf("remote without service: %v", err)
	}
	if _, err := a.Get(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing get: %v", err)
	}
	if stat := a.RecoveryStatus(ctx); !stat.InProgress || stat.Total != 1 || stat.StartedAtUnixMs == 0 {
		t.Fatalf("recovery: %+v", stat)
	}
	if _, err := a.GetArchiveRetention(ctx); err != nil {
		t.Fatal(err)
	}
	if rows, err := a.ListArchived(ctx); err != nil || rows != nil {
		t.Fatalf("archived nil store: %v %v", rows, err)
	}
	if rows, err := a.ListRecoverable(ctx); err != nil || rows != nil {
		t.Fatalf("recoverable nil store: %v %v", rows, err)
	}
	if err := a.DismissRecoverable(ctx, "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("dismiss nil store: %v", err)
	}
	if _, err := a.Recover(ctx, RecoverInput{ID: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("recover nil store: %v", err)
	}
	if _, err := a.GetPolicy(ctx, "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("policy missing: %v", err)
	}
	if _, err := a.UpdatePolicy(ctx, "x", Policy{Mode: "never"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("policy update missing: %v", err)
	}
	if err := a.Archive(ctx, "x"); !errors.Is(err, ErrInternal) {
		t.Fatalf("archive nil store: %v", err)
	}
	if err := a.Unarchive(ctx, "x"); !errors.Is(err, ErrInternal) {
		t.Fatalf("unarchive nil store: %v", err)
	}
	if _, err := a.PruneArchive(ctx, false); err != nil {
		t.Fatal(err)
	}
	_ = sessionstore.StatusLive
}
