package mocks

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

func TestFakeDriver_DefaultMountUnmountCycle(t *testing.T) {
	d := NewFakeDriver()
	ctx := context.Background()
	s := &types.Sandbox{ID: uuid.New()}

	mp, err := d.Mount(ctx, s)
	if err != nil {
		t.Fatalf("Mount: %v", err)
	}
	if !d.Mounted {
		t.Error("Mounted flag should be true after Mount")
	}
	if mp.MergedDir == "" {
		t.Error("default MountPaths should populate MergedDir")
	}

	if err := d.Unmount(ctx, s); err != nil {
		t.Fatalf("Unmount: %v", err)
	}
	if d.Mounted {
		t.Error("Mounted flag should be false after Unmount")
	}
}

func TestFakeDriver_MountErrInjection(t *testing.T) {
	d := NewFakeDriver()
	d.MountErr = errors.New("disk full")

	if _, err := d.Mount(context.Background(), &types.Sandbox{ID: uuid.New()}); err == nil {
		t.Fatal("expected MountErr to surface")
	}
	if d.Mounted {
		t.Error("Mount that errored should not flip Mounted to true")
	}
}

func TestFakeDriver_OrphanCleanupRemovesFromList(t *testing.T) {
	d := NewFakeDriver()
	ctx := context.Background()
	a, b := uuid.New(), uuid.New()
	d.ListDirsResult = []uuid.UUID{a, b}

	if err := d.CleanupOrphan(ctx, a); err != nil {
		t.Fatalf("CleanupOrphan: %v", err)
	}
	got, _ := d.ListSandboxDirs(ctx)
	if len(got) != 1 || got[0] != b {
		t.Errorf("CleanupOrphan should remove the cleaned ID, got %v", got)
	}
	if len(d.OrphanCleanups) != 1 || d.OrphanCleanups[0] != a {
		t.Errorf("OrphanCleanups not recorded: %v", d.OrphanCleanups)
	}
}
