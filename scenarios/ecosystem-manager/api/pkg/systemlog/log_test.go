package systemlog

import (
	"path/filepath"
	"sync"
	"testing"
)

// resetLogState allows tests to reinitialize the logger with a clean slate.
func resetLogState() {
	Close()

	mu = sync.Mutex{}
	recentMu = sync.RWMutex{}
	initOnce = sync.Once{}
	logDir = ""
	logFile = nil
	logWriter = nil
	recentEntries = nil
	writeCounter = 0
}

func setupTestLogger(t *testing.T) string {
	t.Helper()
	resetLogState()

	baseDir := t.TempDir()
	InitWithBaseDir(baseDir)
	t.Cleanup(resetLogState)
	return baseDir
}

func TestRecentEntriesCacheAndTailFallback(t *testing.T) {
	baseDir := setupTestLogger(t)

	Info("first entry")
	Info("second entry")

	entries, err := RecentEntries(0) // cache path
	if err != nil {
		t.Fatalf("RecentEntries returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries from cache, got %d", len(entries))
	}
	if entries[0].Message != "first entry" || entries[1].Message != "second entry" {
		t.Fatalf("unexpected cache entries: %+v", entries)
	}

	// Clear cache to force file tail path.
	recentMu.Lock()
	recentEntries = nil
	recentMu.Unlock()

	tailEntries, err := RecentEntries(5)
	if err != nil {
		t.Fatalf("RecentEntries tail fallback returned error: %v", err)
	}
	if len(tailEntries) < 2 {
		t.Fatalf("expected at least 2 tail entries, got %d", len(tailEntries))
	}
	if tailEntries[len(tailEntries)-1].Message != "second entry" {
		t.Fatalf("tail fallback should return latest entry, got %+v", tailEntries[len(tailEntries)-1])
	}

	logFiles, err := filepath.Glob(filepath.Join(baseDir, "logs", "*.log"))
	if err != nil {
		t.Fatalf("failed to glob log files: %v", err)
	}
	if len(logFiles) == 0 {
		t.Fatalf("expected log file to be created in %s/logs", baseDir)
	}
}
