package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

// newSkillStoreWithCorePack builds a FileSkillStore over a temp dir with an
// active "core" pack (getActivePacks defaults to listing pack dirs when no
// _pack-order.json exists).
func newSkillStoreWithCorePack(t *testing.T) *FileSkillStore {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "skills", "packs", "core"), 0o755); err != nil {
		t.Fatalf("mkdir core pack: %v", err)
	}
	return NewFileSkillStore(dir)
}

func TestRoutedFileSkillStoreWritesOnlyToLeasedConfigRoot(t *testing.T) {
	primary := t.TempDir()
	if err := os.MkdirAll(filepath.Join(primary, "skills", "packs", "local"), 0o755); err != nil {
		t.Fatalf("prepare primary pack: %v", err)
	}
	roots := filerouting.New(storage.Paths{ConfigDir: primary})
	if _, err := roots.InstallLeasedTestRoots("lease", 0, false); err != nil {
		t.Fatalf("install test roots: %v", err)
	}
	defer func() { _ = roots.ClearTestRoots("lease") }()

	s := NewRoutedFileSkillStore(roots)
	ctx := database.WithTestMode(context.Background())
	if err := s.Create(ctx, "local", &Skill{ID: "isolated", Name: "Isolated", Status: StatusActive}, "# Isolated"); err != nil {
		t.Fatalf("create into leased root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(primary, "skills", "packs", "local", "isolated")); !os.IsNotExist(err) {
		t.Fatalf("primary config received isolated write, stat err=%v", err)
	}
	testConfig, err := roots.Pick(ctx, storage.ClassConfig)
	if err != nil {
		t.Fatalf("pick test config: %v", err)
	}
	if _, err := os.Stat(filepath.Join(testConfig, "skills", "packs", "local", "isolated", "skill.json")); err != nil {
		t.Fatalf("leased config missing skill: %v", err)
	}
	if got := roots.LeaseStats().TestRootWrites; got != 1 {
		t.Fatalf("test-root writes = %d, want 1", got)
	}
}

// TestSkillStore_Update_PersistsProgrammaticHomeSetAndClear is the regression
// guard for the bug where FileSkillStore.Update merged field-by-field and never
// applied ProgrammaticHome — so both set-to-value and clear-to-nil silently
// no-oped while still bumping the revision. The store-adapter round-trips the
// full skill on every update, so the field must be applied unconditionally.
func TestSkillStore_Update_PersistsProgrammaticHomeSetAndClear(t *testing.T) {
	s := newSkillStoreWithCorePack(t)
	ctx := context.Background()

	if err := s.Create(ctx, "core", &Skill{ID: "demo", Name: "Demo", Status: StatusActive}, "# Demo"); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Set the pointer.
	val := "test-genie:architecture"
	if err := s.Update(ctx, "demo", &Skill{ID: "demo", Name: "Demo", Status: StatusActive, ProgrammaticHome: &val}, nil); err != nil {
		t.Fatalf("update set: %v", err)
	}
	got, err := s.Get(ctx, "demo")
	if err != nil {
		t.Fatalf("get after set: %v", err)
	}
	if got.ProgrammaticHome == nil || *got.ProgrammaticHome != val {
		t.Fatalf("set did not persist: %v", got.ProgrammaticHome)
	}

	// Clear the pointer (nil) — must actually drop it, not preserve the old value.
	if err := s.Update(ctx, "demo", &Skill{ID: "demo", Name: "Demo", Status: StatusActive, ProgrammaticHome: nil}, nil); err != nil {
		t.Fatalf("update clear: %v", err)
	}
	got, err = s.Get(ctx, "demo")
	if err != nil {
		t.Fatalf("get after clear: %v", err)
	}
	if got.ProgrammaticHome != nil {
		t.Fatalf("clear did not persist; still %q", *got.ProgrammaticHome)
	}
}

// TestSkillStore_Update_PersistsTargetDimensionsClear guards the sibling bug in
// the same merge: TargetDimensions was also never applied, so clearing it
// silently no-oped.
func TestSkillStore_Update_PersistsTargetDimensionsClear(t *testing.T) {
	s := newSkillStoreWithCorePack(t)
	ctx := context.Background()

	if err := s.Create(ctx, "core", &Skill{ID: "demo", Name: "Demo", Status: StatusActive, TargetDimensions: []string{"structure"}}, "# Demo"); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Update to two dimensions.
	if err := s.Update(ctx, "demo", &Skill{ID: "demo", Name: "Demo", Status: StatusActive, TargetDimensions: []string{"structure", "cycles"}}, nil); err != nil {
		t.Fatalf("update dims: %v", err)
	}
	got, err := s.Get(ctx, "demo")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.TargetDimensions) != 2 {
		t.Fatalf("dimension update did not persist: %v", got.TargetDimensions)
	}
	// Clear dimensions.
	if err := s.Update(ctx, "demo", &Skill{ID: "demo", Name: "Demo", Status: StatusActive, TargetDimensions: nil}, nil); err != nil {
		t.Fatalf("update clear dims: %v", err)
	}
	got, err = s.Get(ctx, "demo")
	if err != nil {
		t.Fatalf("get after clear dims: %v", err)
	}
	if len(got.TargetDimensions) != 0 {
		t.Fatalf("dimension clear did not persist: %v", got.TargetDimensions)
	}
}
