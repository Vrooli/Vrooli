package hostpaths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScratchRootsResolvesExistingRepoScratch(t *testing.T) {
	repo := t.TempDir()
	scratch := filepath.Join(repo, ScratchRootName)
	if err := os.MkdirAll(scratch, 0o755); err != nil {
		t.Fatal(err)
	}
	got := ScratchRoots(repo)
	if len(got) != 1 || got[0] != scratch {
		t.Fatalf("ScratchRoots(%q) = %#v, want [%q]", repo, got, scratch)
	}
}

func TestScratchRootsRefusesUnusableRoots(t *testing.T) {
	// A missing scratch directory is ordinary, not broken: the provider should
	// report nothing rather than fail. A relative root can never match the
	// FileProvider containment check, so it is dropped rather than resolved
	// against the process working directory.
	repoWithoutScratch := t.TempDir()
	cases := map[string]string{
		"absent":   repoWithoutScratch,
		"relative": "some/relative/path",
		"empty":    "",
		"blank":    "   ",
	}
	for name, root := range cases {
		t.Run(name, func(t *testing.T) {
			if got := ScratchRoots(root); len(got) != 0 {
				t.Fatalf("ScratchRoots(%q) = %#v, want none", root, got)
			}
		})
	}
}

func TestScratchRootsRejectsAFileNamedScratch(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, ScratchRootName), []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ScratchRoots(repo); len(got) != 0 {
		t.Fatalf("ScratchRoots = %#v, want none when scratch is a file", got)
	}
}
