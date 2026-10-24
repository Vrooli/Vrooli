package catalog

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"react-component-library/internal/catalogcoverage"
)

func writeProbeTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, relative := range fingerprintRoots {
		dir := filepath.Join(root, relative)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "probe.json"), []byte(`{}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestReportCacheServesCachedReportWithoutRecomputing(t *testing.T) {
	root := writeProbeTree(t)
	var calls int64
	compute := func(context.Context) (*catalogcoverage.Report, error) {
		atomic.AddInt64(&calls, 1)
		return &catalogcoverage.Report{}, nil
	}
	cache := &reportCache{}
	for i := 0; i < 5; i++ {
		if _, err := cache.get(context.Background(), root, compute); err != nil {
			t.Fatal(err)
		}
	}
	// The coverage page issues GetCoverage and ListNextWork back to back, and
	// each calls report(). Recomputing per call is what spawned the TypeScript
	// toolchain twice per page view.
	if got := atomic.LoadInt64(&calls); got != 1 {
		t.Fatalf("compute ran %d times, want exactly 1 for an unchanged tree", got)
	}
}

func TestReportCacheServesStaleImmediatelyAndRefreshes(t *testing.T) {
	root := writeProbeTree(t)
	var (
		mu      sync.Mutex
		calls   int
		release = make(chan struct{})
	)
	compute := func(context.Context) (*catalogcoverage.Report, error) {
		mu.Lock()
		calls++
		n := calls
		mu.Unlock()
		if n > 1 {
			// Block the refresh so the test can prove the stale read did not
			// wait on it.
			<-release
		}
		return &catalogcoverage.Report{}, nil
	}
	cache := &reportCache{}
	if _, err := cache.get(context.Background(), root, compute); err != nil {
		t.Fatal(err)
	}

	// Change the tree so the fingerprint no longer matches.
	stalePath := filepath.Join(root, fingerprintRoots[0], "added.json")
	if err := os.WriteFile(stalePath, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := cache.get(context.Background(), root, compute); err != nil {
			t.Error(err)
		}
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("stale read blocked on the background refresh; it must return the previous report immediately")
	}
	close(release)
}

func TestFingerprintChangesWhenSourceChanges(t *testing.T) {
	root := writeProbeTree(t)
	before, err := fingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, fingerprintRoots[1], "extra.tsx"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := fingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("fingerprint did not change after adding a library source file")
	}
}

func TestFingerprintChangesWhenContentChangesWithoutMtimeChange(t *testing.T) {
	root := writeProbeTree(t)
	path := filepath.Join(root, fingerprintRoots[1], "probe.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := fingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"changed":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	after, err := fingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("fingerprint did not change after content changed with the original mtime restored")
	}
}

func TestFingerprintToleratesMissingTrees(t *testing.T) {
	// A fresh checkout may not have every tree present; a missing directory is
	// a legitimate state and must not fail the coverage request.
	if _, err := fingerprint(t.TempDir()); err != nil {
		t.Fatalf("fingerprint over an empty root should succeed, got %v", err)
	}
}

func writeAssetCacheFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	scenarioRoot := filepath.Join(root, "scenarios", "react-component-library")
	if err := os.MkdirAll(filepath.Join(scenarioRoot, "catalog", "assets", "controls"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioRoot, "catalog", "config.json"), []byte(`{"domains":[{"id":"controls","order":1}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	asset := func(id, name string) string {
		return `{"kind":"catalog-asset","asset":{"id":"` + id + `","name":"` + name + `","kind":"component","domain":"controls","target":{"priority":"P1","maturity":"implemented"}}}`
	}
	for _, item := range []struct{ id, name string }{{"controls.button", "Button"}, {"controls.card", "Card"}} {
		if err := os.WriteFile(filepath.Join(scenarioRoot, "catalog", "assets", "controls", item.name+".json"), []byte(asset(item.id, item.name)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range []struct {
		name, catalogID, libraryID, dependency string
	}{
		{"Button", "controls.button", "react-component-library:Button", ""},
		{"Card", "controls.card", "react-component-library:Card", "react-component-library:Button"},
	} {
		versionRoot := filepath.Join(scenarioRoot, "library", "components", item.name, "versions", "1.0.0")
		if err := os.MkdirAll(versionRoot, 0o755); err != nil {
			t.Fatal(err)
		}
		manifest := `{"libraryId":"` + item.libraryID + `","catalogId":"` + item.catalogID + `","latest":"1.0.0"}`
		if item.dependency != "" {
			manifest = `{"libraryId":"` + item.libraryID + `","catalogId":"` + item.catalogID + `","latest":"1.0.0","dependencies":[{"libraryId":"` + item.dependency + `","version":"1.0.0"}]}`
		}
		if err := os.WriteFile(filepath.Join(versionRoot, "component.tsx"), []byte("export const "+item.name+" = () => null;\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		lock := `{"schemaVersion":1,"libraryId":"` + item.libraryID + `","version":"1.0.0","dependencies":[]}`
		if item.dependency != "" {
			lock = `{"schemaVersion":1,"libraryId":"` + item.libraryID + `","version":"1.0.0","dependencies":[{"libraryId":"` + item.dependency + `","version":"1.0.0","rank":4}]}`
		}
		if err := os.WriteFile(filepath.Join(versionRoot, "dependencies.json"), []byte(lock), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(scenarioRoot, "library", "components", item.name, "component.json"), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// The report cache used to carry its own per-asset fingerprint and dependency
// walk, reachable only from this test. The live revision index in
// catalogcoverage now owns both, so this asserts the properties the cache
// actually depends on against the code that is really called.
func TestRevisionIndexTracksSourceAndDependents(t *testing.T) {
	root := writeAssetCacheFixture(t)
	before, err := catalogcoverage.BuildRevisionIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button", "versions", "1.0.0", "component.tsx")
	if err := os.WriteFile(path, []byte("export const Button = () => 'changed';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := catalogcoverage.BuildRevisionIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if before["controls.button"] == after["controls.button"] {
		t.Fatal("asset revision did not change after its source content changed")
	}
	if before["controls.card"] == after["controls.card"] {
		t.Fatal("dependent revision did not change after its dependency changed")
	}
}
