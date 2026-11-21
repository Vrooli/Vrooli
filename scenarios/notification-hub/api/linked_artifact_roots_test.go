package main

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestLinkedArtifactRootsOpenResearchDecisionPackets(t *testing.T) {
	repo, home, research := "/repo", "/home/op", "/home/op/.vrooli/research"
	extra := "/srv/a" + string(filepath.ListSeparator) + "/srv/b"
	got := linkedArtifactRoots(repo, home, research, extra)
	for _, want := range []string{
		"/repo/.vrooli/work",
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
