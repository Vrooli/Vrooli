package providers

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"storage-manager/internal/cleanup"
	cleanupfakes "storage-manager/internal/testutil/cleanup"
)

const cacheHome = "/home/u/.vrooli/cache"

// nestedCacheTree mirrors the runtime-home cache incident: one top-level
// entry ("vrooli") holds another owner's governed root beside stale siblings.
func nestedCacheTree() *cleanupfakes.FileSystem {
	old := at(1, 0)
	files := []cleanup.FileInfo{
		dir(cacheHome+"/vrooli", old),
		dir(cacheHome+"/vrooli/scenario-to-desktop", old),
		dir(cacheHome+"/vrooli/scenario-to-desktop/staging", old),
		file(cacheHome+"/vrooli/scenario-to-desktop/staging/bundle.tar", 40<<30, old),
		file(cacheHome+"/vrooli/scenario-to-desktop/notes.txt", 10, old),
		dir(cacheHome+"/vrooli/abandoned", old),
		file(cacheHome+"/vrooli/abandoned/blob", 2<<30, old),
		dir(cacheHome+"/shared-packages", old),
		file(cacheHome+"/shared-packages/pkg.tgz", 1<<30, old),
	}
	indexed := make(map[string]cleanup.FileInfo, len(files))
	for _, f := range files {
		indexed[f.Path] = f
	}
	return &cleanupfakes.FileSystem{Root: cacheHome, Files: indexed, AllowRemove: true}
}

func runtimeHomeCache(fsys cleanup.FileSystem) cleanup.Provider {
	return NewRuntimeHomeProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{
		ID: "runtime-home-cache", Roots: []string{cacheHome}, TopLevelEntries: true,
		RetentionMaxAge: 24 * time.Hour,
	})
}

func governedPreviewPaths(t *testing.T, provider cleanup.Provider) map[string]int64 {
	t.Helper()
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Scope: cleanup.ObservationScope{Now: now}, Policy: cleanup.ProviderPolicy{Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]int64{}
	for _, item := range preview.Items {
		out[item.Path] = item.Bytes
	}
	return out
}

func TestNestedGovernedRootFromAnotherProviderIsNeverPartOfAnAncestorUnit(t *testing.T) {
	fsys := nestedCacheTree()
	parent := runtimeHomeCache(fsys)
	staging := NewCacheProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{ID: "scenario-desktop-staging", Roots: []string{cacheHome + "/vrooli/scenario-to-desktop/staging"}})
	excludeNestedGovernedRoots([]cleanup.Provider{parent, staging}, nil)

	got := governedPreviewPaths(t, parent)
	for _, forbidden := range []string{cacheHome + "/vrooli", cacheHome + "/vrooli/scenario-to-desktop", cacheHome + "/vrooli/scenario-to-desktop/staging"} {
		if _, ok := got[forbidden]; ok {
			t.Fatalf("offered %s, which contains or is another provider's governed root: %v", forbidden, got)
		}
	}
	for _, want := range []string{cacheHome + "/vrooli/abandoned", cacheHome + "/shared-packages", cacheHome + "/vrooli/scenario-to-desktop/notes.txt"} {
		if _, ok := got[want]; !ok {
			t.Fatalf("sibling %s outside every governed root is no longer reclaimable: %v", want, got)
		}
	}
}

func TestDeclaredOwnerRootIsWithheldFromAncestorProvider(t *testing.T) {
	fsys := nestedCacheTree()
	parent := runtimeHomeCache(fsys)
	// No provider governs the staging root; only an owner declaration does.
	excludeNestedGovernedRoots([]cleanup.Provider{parent}, []string{cacheHome + "/vrooli/scenario-to-desktop/staging"})

	got := governedPreviewPaths(t, parent)
	if _, ok := got[cacheHome+"/vrooli"]; ok {
		t.Fatalf("declared owner root was reclaimable through its ancestor: %v", got)
	}
	if _, ok := got[cacheHome+"/vrooli/abandoned"]; !ok {
		t.Fatalf("undeclared sibling was withheld too: %v", got)
	}
}

func TestApplyRefusesPersistedUnitThatContainsAGovernedRoot(t *testing.T) {
	fsys := nestedCacheTree()
	parent := runtimeHomeCache(fsys)
	excludeNestedGovernedRoots([]cleanup.Provider{parent}, []string{cacheHome + "/vrooli/scenario-to-desktop/staging"})

	// A preview persisted before the root was declared still names the parent.
	forged := cleanup.Preview{Items: []cleanup.PreviewItem{
		{ID: "a", Path: cacheHome + "/vrooli", Bytes: 42 << 30},
		{ID: "b", Path: cacheHome + "/vrooli/scenario-to-desktop/staging/bundle.tar", Bytes: 40 << 30},
	}}
	result, err := parent.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: parent.Metadata().Version, IdempotencyKey: "replay", ApprovalMode: cleanup.ApprovalModeNone, Preview: forged})
	if err != nil {
		t.Fatal(err)
	}
	if len(fsys.Removed) != 0 || result.ReclaimedBytes != 0 || len(result.SkippedItems) != 2 {
		t.Fatalf("governed root removed through an ancestor: removed=%v result=%#v", fsys.Removed, result)
	}
}

func TestPerFileProviderSkipsNestedGovernedRootAndExcludesItFromUsage(t *testing.T) {
	fsys := nestedCacheTree()
	// Per-file selection with a 2 GiB ceiling. Counting the nested 40 GiB root
	// would put this root permanently over budget and select its files.
	parent := NewCacheProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{
		ID: "parent", Roots: []string{cacheHome}, Tier: cleanup.SafetyTierRegenerable, RetentionMaxBytes: 2 << 30,
	})
	excludeNestedGovernedRoots([]cleanup.Provider{parent}, []string{cacheHome + "/vrooli/scenario-to-desktop/staging"})

	got := governedPreviewPaths(t, parent)
	for path := range got {
		if filepath.Dir(path) == cacheHome+"/vrooli/scenario-to-desktop/staging" {
			t.Fatalf("selected a file inside another owner's root: %v", got)
		}
	}
	var total int64
	for _, bytes := range got {
		total += bytes
	}
	// Usage outside the nested root is 3 GiB + 10 bytes; reclaiming down to
	// the 2 GiB ceiling takes about 1 GiB, never the nested 40 GiB.
	if total == 0 || total > 2<<30 {
		t.Fatalf("selection was not sized from usage outside nested roots: %d bytes %v", total, got)
	}
}
