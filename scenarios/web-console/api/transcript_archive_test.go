package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"web-console/internal/sessionstore"
)

func TestClaudeDiscoverySkipsExplicitArchivesAndResumesAfterUnarchive(t *testing.T) {
	ctx := context.Background()
	metadata := sessionstore.NewInMemory()
	home := t.TempDir()
	for _, id := range []string{"live", "archived"} {
		if err := metadata.Save(ctx, sessionstore.Metadata{ID: id, AgentType: sessionstore.AgentClaude, AgentSessionID: id, CWD: "/work", Status: sessionstore.StatusLive}); err != nil {
			t.Fatal(err)
		}
		path := claudeTranscriptPath(home, "/work", id)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("synthetic transcript fixture\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := metadata.MarkArchived(ctx, "archived", time.Now()); err != nil {
		t.Fatal(err)
	}
	source := claudeTranscriptSource{tailer: &ClaudeTailer{server: &Server{sessionStore: metadata}}, homeDir: func() (string, error) { return home, nil }}
	files, err := source.DiscoverFiles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].SessionID != "live" {
		t.Fatalf("archive was scheduled for tailing: %+v", files)
	}
	if err := metadata.MarkUnarchived(ctx, "archived"); err != nil {
		t.Fatal(err)
	}
	files, err = source.DiscoverFiles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatal("explicit unarchive did not resume discovery")
	}
}

func TestClaudeResolverDoesNotScanProcessesForArchivedUnidentifiedPanes(t *testing.T) {
	store := &identityStoreStub{rows: []sessionstore.Metadata{{ID: "archived", AgentType: sessionstore.AgentClaude, Status: sessionstore.StatusLive, ArchivedAt: time.Now()}}}
	resolver := newResolverForTest(t, t.TempDir(), store, nil)
	scans := 0
	resolver.discover = func() ([]agentProcess, error) { scans++; return nil, nil }
	if adopted := resolver.Resolve(context.Background()); adopted != 0 || scans != 0 {
		t.Fatalf("archived pane triggered attribution: adopted=%d process_scans=%d", adopted, scans)
	}
}
