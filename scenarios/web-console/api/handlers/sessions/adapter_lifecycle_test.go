package sessions

import (
	"context"
	"testing"
	"time"

	"web-console/internal/backend"
	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/ptyfake"
	"web-console/internal/sessionstore"
	"web-console/internal/workspace"
	"web-console/session"
)

type adapterConversationStub struct{}

func (adapterConversationStub) DeleteSession(context.Context, string)             {}
func (adapterConversationStub) CopySession(context.Context, string, string) error { return nil }
func (adapterConversationStub) HasConversationAfter(context.Context, string, time.Time) bool {
	return true
}
func (adapterConversationStub) CountSessionEvents(context.Context, string) int64  { return 2 }
func (adapterConversationStub) SessionStorageBytes(context.Context, string) int64 { return 128 }

func TestAdapterLiveArchiveAndPolicyLifecycle(t *testing.T) {
	ctx := context.Background()
	mgr := session.NewManagerWithFactory(ptyfake.NewFactory())
	store := sessionstore.NewInMemory()
	a := &Adapter{
		Manager: mgr, Store: store, Events: events.NewLogger(20), Metrics: metrics.New(),
		Conversations: adapterConversationStub{}, Workspace: workspace.NewMemStore(),
		AgentHistoryPresent: func(sessionstore.Metadata) bool { return true },
		ArchiveGracePeriod:  -time.Second,
		RetentionPolicy:     func() ArchiveRetentionPolicy { return ArchiveRetentionPolicy{MaxBytes: 64} },
		AgentHistorySize:    func(sessionstore.Metadata) (int64, error) { return 32, nil },
		PruneAgentHistory:   func(sessionstore.Metadata) (int64, error) { return 32, nil },
	}

	created, err := a.Create(ctx, CreateInput{Shell: "/bin/sh", Cols: 80, Rows: 24, Backend: "standard", Origin: "ui", Owner: "test", DisplayLabel: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Delete(ctx, created.ID) }()
	if created.ID == "" || a.Metrics.SessionsCreated.Load() != 1 {
		t.Fatalf("created = %+v", created)
	}
	if err := store.Save(ctx, sessionstore.Metadata{ID: created.ID, Backend: backend.Standard, Shell: "/bin/sh", Created: time.Now().UTC(), Origin: sessionstore.OriginUI, Owner: "test"}); err != nil {
		t.Fatal(err)
	}
	if listed, err := a.List(ctx); err != nil || len(listed) != 1 || listed[0].Owner != "test" {
		t.Fatalf("listed = %+v, err=%v", listed, err)
	}
	if got, err := a.Get(ctx, created.ID); err != nil || got.ID != created.ID {
		t.Fatalf("get = %+v, err=%v", got, err)
	}
	if view, err := a.GetPolicy(ctx, created.ID); err != nil || view.SessionID != created.ID {
		t.Fatalf("policy view = %+v, err=%v", view, err)
	}
	if view, err := a.UpdatePolicy(ctx, created.ID, Policy{Mode: "preset", Duration: "1h"}); err != nil || !view.HasExpiry {
		t.Fatalf("updated policy = %+v, err=%v", view, err)
	}
	if _, err := a.UpdatePolicy(ctx, created.ID, Policy{Mode: "invalid"}); err == nil {
		t.Fatal("invalid policy was accepted")
	}

	archivedID := "archived"
	if err := store.Save(ctx, sessionstore.Metadata{ID: archivedID, Backend: backend.Persistent, Status: sessionstore.StatusDismissed, AgentType: sessionstore.AgentCodex, Created: time.Now().UTC(), ArchivedAt: time.Now().UTC(), LastRolloutPath: "/tmp/history"}); err != nil {
		t.Fatal(err)
	}
	if rows, err := a.ListArchived(ctx); err != nil || len(rows) != 1 || rows[0].ID != archivedID {
		t.Fatalf("archived = %+v, err=%v", rows, err)
	}
	if snap, err := a.GetArchiveRetention(ctx); err != nil || snap.Stats.EntryCount != 1 {
		t.Fatalf("retention = %+v, err=%v", snap, err)
	}
	if dry, err := a.PruneArchive(ctx, false); err != nil || len(dry.Actions) == 0 || !dry.DryRun {
		t.Fatalf("prune dry run = %+v, err=%v", dry, err)
	}
	if err := a.Archive(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
}

func TestAdapterRecoverableListingAndDismissal(t *testing.T) {
	ctx := context.Background()
	store := sessionstore.NewInMemory()
	if err := store.Save(ctx, sessionstore.Metadata{ID: "recoverable", Status: sessionstore.StatusAwaitingRecovery, Detached: true, AgentType: sessionstore.AgentCodex, Created: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	a := &Adapter{Store: store, Manager: emptySessionManager{}, Workspace: workspace.NewMemStore()}
	rows, err := a.ListRecoverable(ctx)
	if err != nil || len(rows) != 1 || !rows[0].Recoverable {
		t.Fatalf("recoverable = %+v, err=%v", rows, err)
	}
	if err := a.DismissRecoverable(ctx, "recoverable"); err != nil {
		t.Fatal(err)
	}
	if err := a.DismissRecoverable(ctx, "recoverable"); err == nil {
		t.Fatal("dismissed session was dismissed twice")
	}
}
