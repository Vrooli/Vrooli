package aisearch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestParseHelpTreeCached_SkipsHelpInvocationOnUnchangedMtime verifies that a
// second call against the same binary (unchanged mtime) avoids spawning new
// help subprocesses. The cache lives on the FilesystemDiscoverySource.
func TestParseHelpTreeCached_SkipsHelpInvocationOnUnchangedMtime(t *testing.T) {
	dir := t.TempDir()
	// Use a fake binary file so we have a stable mtime we can stat.
	bin := filepath.Join(dir, "fake-cli")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake bin: %v", err)
	}

	src := &FilesystemDiscoverySource{RepoRoot: dir}
	// Replace the helpRunner with a counting fake. We do this by directly
	// calling ParseHelpTree through parseHelpTreeCached — but the cache only
	// invokes ParseHelpTree on miss. So we shim by hijacking via the cache
	// directly: prime the cache, then verify a second call returns it.
	// Simulate first-call: invoke parseHelpTreeCached, which will execute the
	// fake binary's --help (returning empty, hence a help-failed stub). That
	// stub is what gets cached.
	first := src.parseHelpTreeCached(context.Background(), bin, "fake")
	if len(first) == 0 {
		t.Fatalf("first call returned no records")
	}

	// Mutate the cached records to a sentinel so we can detect cache hit.
	src.mu.Lock()
	entry := src.helpCache[bin]
	entry.records = []CommandRecord{{Origin: "fake", Name: "sentinel", FullPath: "fake sentinel", Source: SourceHelp}}
	src.helpCache[bin] = entry
	src.mu.Unlock()

	second := src.parseHelpTreeCached(context.Background(), bin, "fake")
	if len(second) != 1 || second[0].Name != "sentinel" {
		t.Errorf("second call did not hit the cache; got %+v", second)
	}

	// Touch the binary so mtime changes. The next call must bypass the cache
	// (returning real records again, not the sentinel).
	future := entry.mtime.Add(2 * 1000 * 1000 * 1000) // +2s
	if err := os.Chtimes(bin, future, future); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	third := src.parseHelpTreeCached(context.Background(), bin, "fake")
	if len(third) == 1 && third[0].Name == "sentinel" {
		t.Errorf("mtime change should bust the cache; got cached sentinel back")
	}
}
