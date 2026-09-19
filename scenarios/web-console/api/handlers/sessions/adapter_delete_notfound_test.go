package sessions

import (
	"context"
	"errors"
	"testing"

	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/ptyfake"
	"web-console/internal/sessionstore"
	"web-console/internal/workspace"
	"web-console/session"
)

// Deleting an id that names no session must report not-found. Returning
// success made a mistyped or truncated id look like a completed deletion:
// the caller saw "Deleted session ..." while every row survived. Genuine
// retries are absorbed by the lifecycle receipt replay path, so reporting
// the miss costs no idempotency.
func TestAdapterDeleteUnknownSessionReportsNotFound(t *testing.T) {
	ctx := context.Background()
	mgr := session.NewManagerWithFactory(ptyfake.NewFactory())
	store := sessionstore.NewInMemory()
	a := &Adapter{
		Manager: mgr, Store: store, Events: events.NewLogger(20), Metrics: metrics.New(),
		Conversations: adapterConversationStub{}, Workspace: workspace.NewMemStore(),
	}

	err := a.Delete(ctx, "b1a5e0de-0000-4000-8000-000000000000")
	if err == nil {
		t.Fatal("delete of an unknown session returned success; it must report not-found")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete error = %v, want ErrNotFound", err)
	}
}

// A truncated id is the common way to hit the miss: `session list` prints
// ids that other commands then reject, so the short form must not be
// silently accepted as a successful delete.
func TestAdapterDeleteTruncatedIDReportsNotFound(t *testing.T) {
	ctx := context.Background()
	mgr := session.NewManagerWithFactory(ptyfake.NewFactory())
	store := sessionstore.NewInMemory()
	a := &Adapter{
		Manager: mgr, Store: store, Events: events.NewLogger(20), Metrics: metrics.New(),
		Conversations: adapterConversationStub{}, Workspace: workspace.NewMemStore(),
	}

	created, err := a.Create(ctx, CreateInput{Origin: "ui"})
	if err != nil {
		t.Fatal(err)
	}
	if len(created.ID) < 8 {
		t.Fatalf("created id %q too short to truncate", created.ID)
	}

	if err := a.Delete(ctx, created.ID[:8]); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete with truncated id = %v, want ErrNotFound", err)
	}
	if rows, err := store.List(ctx); err != nil {
		t.Fatal(err)
	} else if len(rows) != 1 {
		t.Fatalf("rows after refused delete = %d, want 1 (the session must survive)", len(rows))
	}
}
