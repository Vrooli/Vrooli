package retention

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newFixtureDir builds a directory of top-level snapshot entries, each holding
// one file of the given size, with mod times spaced apart and entry 0 oldest.
func newFixtureDir(t *testing.T, entries int, size int, spacing time.Duration) string {
	t.Helper()
	root := t.TempDir()
	payload := []byte(strings.Repeat("x", size))
	for i := range entries {
		dir := filepath.Join(root, "snapshot-"+string(rune('a'+i%26))+"-"+itoa(i))
		if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "nested", "data.bin"), payload, 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		modTime := fixtureClock.Add(-spacing * time.Duration(entries-i))
		if err := os.Chtimes(dir, modTime, modTime); err != nil {
			t.Fatalf("chtimes: %v", err)
		}
	}
	return root
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

func newDirPruner(t *testing.T, root string) *DirectoryPruner {
	t.Helper()
	p, err := NewDirectoryPruner(DirectoryConfig{Path: root, Now: func() time.Time { return fixtureClock }})
	if err != nil {
		t.Fatalf("NewDirectoryPruner: %v", err)
	}
	return p
}

func countEntries(t *testing.T, root string) int {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	return len(entries)
}

func TestDirectoryMeasureCountsNestedBytes(t *testing.T) {
	root := newFixtureDir(t, 4, 1000, time.Hour)
	usage, err := newDirPruner(t, root).Measure(context.Background())
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	if usage.Items != 4 {
		t.Errorf("Items = %d, want 4 top-level entries", usage.Items)
	}
	if usage.Bytes != 4000 {
		t.Errorf("Bytes = %d, want 4000; nested files were not counted", usage.Bytes)
	}
}

func TestDirectoryMissingDirectoryMeasuresEmpty(t *testing.T) {
	// A component that has not written anything is trivially within budget;
	// erroring would make an unused budget look broken.
	p, err := NewDirectoryPruner(DirectoryConfig{Path: filepath.Join(t.TempDir(), "never-created")})
	if err != nil {
		t.Fatalf("NewDirectoryPruner: %v", err)
	}
	usage, err := p.Measure(context.Background())
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	if usage != (Usage{}) {
		t.Fatalf("Usage = %+v, want zero", usage)
	}
}

func TestDirectoryPruneByAgeRemovesOldestWholeEntries(t *testing.T) {
	root := newFixtureDir(t, 10, 1000, time.Hour)
	result, err := newDirPruner(t, root).Prune(context.Background(), Budget{Name: "snapshots", MaxAge: 5 * time.Hour})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	// Entries are 10h..1h old; the five strictly older than the 5h horizon go.
	if got := countEntries(t, root); got != 5 {
		t.Fatalf("entries remaining = %d, want 5", got)
	}
	if result.Deleted != 5 {
		t.Errorf("Deleted = %d, want 5", result.Deleted)
	}
	if result.BoundBy != BoundAge {
		t.Errorf("BoundBy = %v, want age", result.BoundBy)
	}
	if result.FreedBytes != 5000 {
		t.Errorf("FreedBytes = %d, want 5000", result.FreedBytes)
	}
}

func TestDirectoryPruneBySizeStopsAtTheBound(t *testing.T) {
	root := newFixtureDir(t, 10, 1000, time.Hour)
	result, err := newDirPruner(t, root).Prune(context.Background(), Budget{Name: "snapshots", MaxBytes: 3000})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if got := countEntries(t, root); got != 3 {
		t.Fatalf("entries remaining = %d, want 3 to fit a 3000-byte ceiling", got)
	}
	if result.BoundBy != BoundBytes {
		t.Errorf("BoundBy = %v, want bytes", result.BoundBy)
	}
	if result.After.Bytes > 3000 {
		t.Errorf("After.Bytes = %d, want at or below the 3000 ceiling", result.After.Bytes)
	}
}

func TestDirectoryPruneBySizeBindsWhenAgeCannot(t *testing.T) {
	// Everything is minutes old against a 30-day horizon, so only the ceiling
	// can act. This is the directory-target form of the autoheal failure.
	root := newFixtureDir(t, 10, 1000, time.Minute)
	result, err := newDirPruner(t, root).Prune(context.Background(), Budget{
		Name: "snapshots", MaxAge: 30 * 24 * time.Hour, MaxBytes: 4000,
	})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if result.BoundBy != BoundBytes {
		t.Fatalf("BoundBy = %v, want bytes", result.BoundBy)
	}
	if got := countEntries(t, root); got != 4 {
		t.Fatalf("entries remaining = %d, want 4", got)
	}
}

func TestDirectoryPruneWithinBudgetRemovesNothing(t *testing.T) {
	root := newFixtureDir(t, 3, 1000, time.Minute)
	result, err := newDirPruner(t, root).Prune(context.Background(), Budget{
		Name: "snapshots", MaxAge: 30 * 24 * time.Hour, MaxBytes: 1 << 30,
	})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if result.Deleted != 0 || countEntries(t, root) != 3 {
		t.Fatalf("Deleted = %d with %d entries left, want 0 and 3", result.Deleted, countEntries(t, root))
	}
	if result.BoundBy != BoundNone {
		t.Errorf("BoundBy = %v, want none", result.BoundBy)
	}
}

func TestDirectoryPruneDeletesWholeEntriesNotIndividualFiles(t *testing.T) {
	// A half-deleted subtree is harder to reason about than a missing one.
	root := newFixtureDir(t, 4, 1000, time.Hour)
	if _, err := newDirPruner(t, root).Prune(context.Background(), Budget{Name: "s", MaxBytes: 2000}); err != nil {
		t.Fatalf("Prune: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		nested := filepath.Join(root, e.Name(), "nested", "data.bin")
		if _, err := os.Stat(nested); err != nil {
			t.Fatalf("surviving entry %s is hollowed out: %v", e.Name(), err)
		}
	}
}

func TestDirectoryCancelledContextReportsIncomplete(t *testing.T) {
	root := newFixtureDir(t, 10, 1000, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := newDirPruner(t, root).Prune(ctx, Budget{Name: "s", MaxAge: time.Hour})
	if err == nil {
		t.Fatal("expected a cancellation error")
	}
	if !result.Incomplete {
		t.Fatal("Incomplete = false after a cancelled prune")
	}
	if countEntries(t, root) != 10 {
		t.Fatal("entries were removed despite an already-cancelled context")
	}
}

func TestDirectoryPrunerRequiresAnAbsolutePath(t *testing.T) {
	// A relative path would resolve against the process working directory
	// rather than the component's storage namespace, which is exactly how a
	// shadow variant ends up pruning live's data.
	if _, err := NewDirectoryPruner(DirectoryConfig{Path: "snapshots"}); err == nil {
		t.Fatal("expected a relative path to be rejected")
	}
	if _, err := NewDirectoryPruner(DirectoryConfig{Path: ""}); err == nil {
		t.Fatal("expected an empty path to be rejected")
	}
}
