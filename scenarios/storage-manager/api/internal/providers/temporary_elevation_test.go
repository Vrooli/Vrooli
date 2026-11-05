package providers

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"storage-manager/internal/cleanup"
	cleanupfakes "storage-manager/internal/testutil/cleanup"
)

// deniedWalkFS fails to walk named entries the way hostfs does for a subtree
// owned by another user or left mode 000: openat ... permission denied.
type deniedWalkFS struct {
	treeWalkFS
	denied map[string]bool
	opens  map[string]int
}

func (f *deniedWalkFS) OpenWalk(ctx context.Context, path string) (cleanup.FileWalk, error) {
	f.opens[path]++
	if f.denied[path] {
		return nil, &fs.PathError{Op: "openat", Path: path + "/private", Err: fs.ErrPermission}
	}
	return f.treeWalkFS.OpenWalk(ctx, path)
}

func TestTemporaryRecoveryClassifiesUnreadableEntriesOnceWithoutBlockingProgress(t *testing.T) {
	foreign := filepath.Join(tmpRoot, "workspace-sandbox")
	stale := filepath.Join(tmpRoot, "old-fixture")
	files := map[string]cleanup.FileInfo{
		foreign:              dir(foreign, at(1, 0)),
		foreign + "/payload": file(foreign+"/payload", 1<<20, at(1, 0)),
		stale:                dir(stale, at(2, 0)),
		stale + "/payload":   file(stale+"/payload", 1<<20, at(2, 0)),
	}
	fsys := &deniedWalkFS{treeWalkFS: treeWalkFS{&cleanupfakes.FileSystem{Root: tmpRoot, Files: files}}, denied: map[string]bool{foreign: true}, opens: map[string]int{}}
	provider := NewTmpProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{ID: "tmp", Roots: []string{tmpRoot}, TopLevelEntries: true})

	first := recoveryPreview(t, provider, 24*time.Hour)
	if len(first.Items) != 1 || first.Items[0].Path != stale {
		t.Fatalf("unreadable entry stalled the removable one: %#v", first)
	}
	if first.BlockedReason != "" {
		t.Fatalf("unreadable entry became the provider's blocked reason: %q", first.BlockedReason)
	}
	notes := elevationWarnings(first)
	if len(notes) != 1 || !strings.Contains(notes[0], "workspace-sandbox") {
		t.Fatalf("elevation was not reported once by name: %v", first.Notices)
	}
	for _, warning := range first.Warnings {
		if strings.Contains(warning, "permission denied") {
			t.Fatalf("raw permission failure surfaced as a warning: %v", first.Warnings)
		}
	}

	// The next full pass neither re-walks nor re-reports the same entry.
	delete(fsys.Files, stale)
	delete(fsys.Files, stale+"/payload")
	second := recoveryPreview(t, provider, 24*time.Hour)
	third := recoveryPreview(t, provider, 24*time.Hour)
	if fsys.opens[foreign] != 1 {
		t.Fatalf("unreadable entry was walked %d times; want one classification", fsys.opens[foreign])
	}
	if len(elevationWarnings(second))+len(elevationWarnings(third)) != 0 {
		t.Fatalf("elevation re-reported every run: %v / %v", second.Notices, third.Notices)
	}
	if strings.Contains(second.BlockedReason, "permission") || strings.Contains(third.BlockedReason, "permission") {
		t.Fatalf("classified entry still blocks the provider: %q / %q", second.BlockedReason, third.BlockedReason)
	}
}

func elevationWarnings(preview cleanup.Preview) []string {
	var out []string
	for _, notice := range preview.Notices {
		if strings.Contains(notice, "need elevation") {
			out = append(out, notice)
		}
	}
	return out
}
