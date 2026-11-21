package sessionstore

import (
	"context"
	"testing"
	"time"

	"web-console/internal/backend"
	"web-console/internal/policy"
)

func TestStoresCoverRecoveryAndMutationPaths(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	meta := Metadata{
		ID: "detached", Backend: backend.Persistent, Shell: "/bin/zsh", Cols: 120, Rows: 40,
		Policy: policy.Policy{Mode: policy.Preset, Duration: "1h"}, Created: now, Detached: true,
		AgentType: AgentCodex, LaunchCommand: "codex", AgentSessionID: "agent-1", CWD: "/tmp",
		LastRolloutPath: "/tmp/rollout", LastActivityAt: now, Origin: OriginRemote,
	}

	for _, store := range []Store{newSQLStore(t), NewInMemory()} {
		t.Run(string(meta.Backend)+"/"+storeName(store), func(t *testing.T) {
			if err := store.Save(ctx, meta); err != nil {
				t.Fatal(err)
			}
			if err := store.Save(ctx, Metadata{ID: "dismissed", Status: StatusDismissed, Created: now}); err != nil {
				t.Fatal(err)
			}
			if err := store.Save(ctx, Metadata{ID: "live", Detached: true, Created: now.Add(time.Hour)}); err != nil {
				t.Fatal(err)
			}

			if err := store.UpdatePolicy(ctx, meta.ID, policy.Policy{Mode: policy.Custom, Duration: "1h"}); err != nil {
				t.Fatal(err)
			}
			if err := store.UpdateAgentInfo(ctx, meta.ID, AgentInfo{
				AgentType: AgentGrok, LaunchCommand: "grok", AgentSessionID: "agent-2", CWD: "/work",
				LastRolloutPath: "/work/updates", LastActivityAt: now.Add(time.Minute),
			}); err != nil {
				t.Fatal(err)
			}
			if err := store.SetProvenance(ctx, meta.ID, OriginProgrammatic, "owner", "label"); err != nil {
				t.Fatal(err)
			}
			if err := store.UpdateAgentInfo(ctx, meta.ID, AgentInfo{}); err != nil {
				t.Fatal(err)
			}

			if got, err := store.Get(ctx, meta.ID); err != nil || got.AgentType != AgentGrok || got.Owner != "owner" {
				t.Fatalf("updated metadata = %+v, err=%v", got, err)
			}
			if got, err := store.ListDetached(ctx); err != nil || len(got) != 2 {
				t.Fatalf("detached = %+v, err=%v", got, err)
			}

			if err := store.MarkOrphaned(ctx, "live", now); err != nil {
				t.Fatal(err)
			}
			if got, err := store.ListRecoverable(ctx); err != nil || len(got) != 1 {
				t.Fatalf("recoverable = %+v, err=%v", got, err)
			}
			if err := store.MarkLive(ctx, "live"); err != nil {
				t.Fatal(err)
			}
			if err := store.MarkDismissed(ctx, meta.ID, "replacement"); err != nil {
				t.Fatal(err)
			}
			if err := store.MarkArchived(ctx, meta.ID, now.Add(2*time.Hour)); err != nil {
				t.Fatal(err)
			}
			if got, err := store.ListArchived(ctx); err != nil || len(got) < 2 {
				t.Fatalf("archived = %+v, err=%v", got, err)
			}
			if got, err := store.ListRetentionCandidates(ctx); err != nil || len(got) != 1 {
				t.Fatalf("retention = %+v, err=%v", got, err)
			}
			if err := store.MarkUnarchived(ctx, meta.ID); err != nil {
				t.Fatal(err)
			}
			if err := store.Delete(ctx, "dismissed"); err != nil {
				t.Fatal(err)
			}
			if _, err := store.Get(ctx, "dismissed"); err == nil {
				t.Fatal("deleted session was still readable")
			}
			if err := store.MarkArchived(ctx, "missing", now); err == nil {
				t.Fatal("missing archive did not fail")
			}
			if err := store.MarkUnarchived(ctx, "missing"); err == nil {
				t.Fatal("missing unarchive did not fail")
			}
		})
	}
}

func TestStatusTransitionOwnerRejectsUnknownAndTerminalReversal(t *testing.T) {
	ctx := context.Background()
	for _, store := range []Store{newSQLStore(t), NewInMemory()} {
		t.Run(storeName(store), func(t *testing.T) {
			if err := store.Save(ctx, Metadata{ID: "invalid", Status: Status("waiting")}); err == nil {
				t.Fatal("unknown status was persisted")
			}
			if err := store.Save(ctx, Metadata{ID: "terminal", Status: StatusDismissed}); err != nil {
				t.Fatal(err)
			}
			if err := store.MarkLive(ctx, "terminal"); err == nil {
				t.Fatal("dismissed session was reopened")
			}
			if err := store.MarkOrphaned(ctx, "terminal", time.Now().UTC()); err == nil {
				t.Fatal("dismissed session entered recovery")
			}
			got, err := store.Get(ctx, "terminal")
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != StatusDismissed {
				t.Fatalf("rejected transition mutated status to %q", got.Status)
			}
		})
	}
}

func TestStatusTransitionOwnerPreservesRecoveryAndArchivedNoop(t *testing.T) {
	ctx := context.Background()
	for _, store := range []Store{newSQLStore(t), NewInMemory()} {
		t.Run(storeName(store), func(t *testing.T) {
			if err := store.Save(ctx, Metadata{ID: "recoverable", Detached: true}); err != nil {
				t.Fatal(err)
			}
			orphanedAt := time.Now().UTC()
			if err := store.MarkOrphaned(ctx, "recoverable", orphanedAt); err != nil {
				t.Fatal(err)
			}
			got, err := store.Get(ctx, "recoverable")
			if err != nil || got.Status != StatusAwaitingRecovery || !got.OrphanedAt.Equal(orphanedAt) {
				t.Fatalf("orphaned session = %+v, err=%v", got, err)
			}
			if err := store.MarkLive(ctx, "recoverable"); err != nil {
				t.Fatal(err)
			}
			got, err = store.Get(ctx, "recoverable")
			if err != nil || got.Status != StatusLive || !got.OrphanedAt.IsZero() {
				t.Fatalf("recovered session = %+v, err=%v", got, err)
			}
			archivedAt := time.Now().UTC()
			if err := store.MarkArchived(ctx, "recoverable", archivedAt); err != nil {
				t.Fatal(err)
			}
			if err := store.MarkOrphaned(ctx, "recoverable", orphanedAt); err != nil {
				t.Fatal(err)
			}
			got, err = store.Get(ctx, "recoverable")
			if err != nil || got.Status != StatusLive || !got.ArchivedAt.Equal(archivedAt) {
				t.Fatalf("archived session changed during orphan event = %+v, err=%v", got, err)
			}
		})
	}
}

func storeName(store Store) string {
	switch store.(type) {
	case *SQLStore:
		return "sql"
	default:
		return "memory"
	}
}
