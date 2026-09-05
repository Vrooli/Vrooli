package store

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

func TestFileTeamStore_RoutesTestModeTeamAndSharedFileWrites(t *testing.T) {
	ctx := context.Background()
	primary := t.TempDir()
	roots := filerouting.New(storage.Paths{
		ConfigDir: primary + "/config",
		DataDir:   primary + "/data",
		CacheDir:  primary + "/cache",
	})
	if _, err := roots.InstallLeasedTestRoots("team-isolation", time.Minute, true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = roots.ClearTestRoots("team-isolation") })

	store := NewFileTeamStore(primary+"/config", primary+"/data", nil, roots)
	testCtx := database.WithTestMode(ctx)
	team := newIndependentTestTeam("test-only", "Test-only team")
	if err := store.Create(testCtx, team); err != nil {
		t.Fatalf("create test-mode team: %v", err)
	}
	if err := store.WriteSharedFile(testCtx, team.ID, "TEAM.md", "isolated"); err != nil {
		t.Fatalf("write test-mode shared file: %v", err)
	}
	if err := store.SaveTaskBoard(testCtx, team.ID, &TeamTaskBoard{Tasks: []TeamTask{{ID: "task", Title: "isolated"}}}); err != nil {
		t.Fatalf("write test-mode task board: %v", err)
	}
	if _, err := store.Get(ctx, team.ID); err == nil {
		t.Fatal("test-mode team write leaked into primary config root")
	}
	if _, err := store.ReadSharedFile(ctx, team.ID, "TEAM.md"); err == nil {
		t.Fatal("test-mode shared-file write leaked into primary config root")
	}
	primaryTasks, err := store.GetTaskBoard(ctx, team.ID)
	if err != nil || len(primaryTasks.Tasks) != 0 {
		t.Fatalf("test-mode task board leaked into primary runtime root: %#v, %v", primaryTasks, err)
	}
	content, err := store.ReadSharedFile(testCtx, team.ID, "TEAM.md")
	if err != nil || content != "isolated" {
		t.Fatalf("test-mode shared file = %q, %v; want isolated", content, err)
	}
	testTasks, err := store.GetTaskBoard(testCtx, team.ID)
	if err != nil || len(testTasks.Tasks) != 1 {
		t.Fatalf("test-mode task board = %#v, %v; want one task", testTasks, err)
	}
}
