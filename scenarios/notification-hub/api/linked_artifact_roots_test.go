package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"notification-hub/internal/linkedartifact"
)

func TestLinkedArtifactRootsResolveRepositoryRelativeDecisionLinks(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".vrooli", "work"), 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo, "scenarios", "notification-hub", "README.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# Notification Hub"), 0o600); err != nil {
		t.Fatal(err)
	}

	resolver := linkedartifact.New(linkedArtifactRoots(repo, "", "", "")...)
	file, info, err := resolver.Open("scenarios/notification-hub/README.md")
	if err != nil {
		t.Fatalf("open repository-relative decision link: %v", err)
	}
	defer file.Close()
	if info.Name() != "README.md" {
		t.Fatalf("opened %q, want README.md", info.Name())
	}

	outside := filepath.Join(t.TempDir(), "private.txt")
	if err := os.WriteFile(outside, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := resolver.Open(outside); err == nil {
		t.Fatal("opened a file outside the configured repository root")
	}
}

func TestLinkedArtifactRootsOpenResearchDecisionPackets(t *testing.T) {
	repo, home, research := "/repo", "/home/op", "/home/op/.vrooli/research"
	extra := "/srv/a" + string(filepath.ListSeparator) + "/srv/b"
	got := linkedArtifactRoots(repo, home, research, extra)
	for _, want := range []string{
		"/repo/.vrooli/work",
		"/repo",
		"/home/op/.vrooli/work",
		"/home/op/.vrooli/portfolio/steward/delegations",
		"/home/op/.vrooli/research",
		"/srv/a",
		"/srv/b",
	} {
		if !slices.Contains(got, want) {
			t.Errorf("roots %v lack %s", got, want)
		}
	}
}

func TestLinkedArtifactRootsSkipUnknownLocations(t *testing.T) {
	if got := linkedArtifactRoots("", "", "", ""); len(got) != 0 {
		t.Fatalf("roots = %v, want none", got)
	}
}
