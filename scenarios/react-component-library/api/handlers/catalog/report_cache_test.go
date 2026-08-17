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

func TestFingerprintToleratesMissingTrees(t *testing.T) {
	// A fresh checkout may not have every tree present; a missing directory is
	// a legitimate state and must not fail the coverage request.
	if _, err := fingerprint(t.TempDir()); err != nil {
		t.Fatalf("fingerprint over an empty root should succeed, got %v", err)
	}
}
