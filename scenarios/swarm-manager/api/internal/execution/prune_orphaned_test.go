package execution

import (
	"context"
	"path/filepath"
	"swarm-manager/internal/promptmanager"
	"testing"
)

// TestPruneOrphanedPending_DropsRecordsForMissingBacklogItems verifies that a
// pending record whose spec.json is missing is removed, while records for
// items that still exist on disk are preserved.
func TestPruneOrphanedPending_DropsRecordsForMissingBacklogItems(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "real-item", map[string]any{
		"name":     "real-item",
		"title":    "Real",
		"status":   "backlog",
		"priority": 3,
		"tags":     []string{},
	})

	service := NewService(ServiceConfig{
		RootDir:      root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	})

	// Seed the store with one orphaned pending, one live pending, and one
	// completed record that references a missing item (should be kept —
	// pruning only applies to pending).
	seed := []Record{
		{ExecutionID: "orphan-1", BacklogKind: "idea", BacklogName: "deleted-ghost", Status: StatusPending, Mode: ModeManual},
		{ExecutionID: "live-1", BacklogKind: "idea", BacklogName: "real-item", Status: StatusPending, Mode: ModeManual},
		{ExecutionID: "done-1", BacklogKind: "idea", BacklogName: "also-deleted", Status: StatusCompleted},
	}
	if err := service.store.Save(seed); err != nil {
		t.Fatalf("save seed: %v", err)
	}

	pruned, err := service.PruneOrphanedPending()
	if err != nil {
		t.Fatalf("PruneOrphanedPending error: %v", err)
	}
	if pruned != 1 {
		t.Fatalf("expected 1 pruned record, got %d", pruned)
	}

	after, err := service.store.Load()
	if err != nil {
		t.Fatalf("load after prune: %v", err)
	}
	if len(after) != 2 {
		t.Fatalf("expected 2 records after prune, got %d", len(after))
	}
	ids := map[string]bool{}
	for _, r := range after {
		ids[r.ExecutionID] = true
	}
	if ids["orphan-1"] {
		t.Errorf("orphaned pending record should have been pruned")
	}
	if !ids["live-1"] || !ids["done-1"] {
		t.Errorf("non-orphan records should remain, got %v", ids)
	}
}

// TestQueueBacklog_PrunesOrphansBeforeDepthCheck reproduces the user-facing
// bug: when the store is full of pending records for items that no longer
// exist, queuing a real item should still succeed.
func TestQueueBacklog_PrunesOrphansBeforeDepthCheck(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "real-item", map[string]any{
		"name":     "real-item",
		"title":    "Real",
		"status":   "backlog",
		"priority": 3,
		"tags":     []string{},
	})
	mustWriteDeliverableFile(t, root, "idea", "real-item")

	service := NewService(ServiceConfig{
		RootDir:            root,
		StorePath:          filepath.Join(root, ".vrooli", "execution-runs.json"),
		GovernanceProvider: &stubGovernanceProvider{settings: GovernanceSettings{MaxQueueDepth: 3}},
		PromptClient:       &promptmanager.MockClient{Result: "test prompt"},
	})

	seed := make([]Record, 0, 3)
	for i := 0; i < 3; i++ {
		seed = append(seed, Record{
			ExecutionID: "orphan-" + string(rune('a'+i)),
			BacklogKind: "idea",
			BacklogName: "ghost",
			Status:      StatusPending,
			Mode:        ModeManual,
		})
	}
	if err := service.store.Save(seed); err != nil {
		t.Fatalf("save seed: %v", err)
	}

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "real-item",
		Mode:        ModeManual,
	})
	if err != nil {
		t.Fatalf("QueueBacklog should succeed after pruning orphans, got: %v", err)
	}
	if record.Status != StatusPending {
		t.Fatalf("expected pending status, got %s", record.Status)
	}
}
