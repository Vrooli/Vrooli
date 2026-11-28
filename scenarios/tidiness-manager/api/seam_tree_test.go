package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func listingOf(rels ...string) gitSourceListing {
	return func(string) ([]byte, error) { return []byte(strings.Join(rels, "\x00") + "\x00"), nil }
}

func noGit(string) ([]byte, error) { return nil, os.ErrNotExist }

// writeExecutableFixture writes a file that isExecutableArtifact recognises
// as a compiled binary.
func writeExecutableFixture(t *testing.T, root, rel string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("\x7fELF\x02\x01\x01\x00padding"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// The executable verdict must not depend on whether Git or the walk fallback
// enumerated the tree: both see the committed residue and neither sees the
// build output the repository ignores.
func TestSeamTreeExecutableVerdictMatchesAcrossEnumerationPaths(t *testing.T) {
	root := t.TempDir()
	residue := "scenarios/app/tools/legacy-helper"
	writeExecutableFixture(t, root, residue)
	for _, ignored := range []string{
		"scenarios/app/api/bin/api",
		"scenarios/app/api/.venv/bin/python",
		"scenarios/app/platforms/electron/dist-electron/app",
		"scenarios/app/platforms/electron/bundle/runtime/app",
	} {
		writeExecutableFixture(t, root, ignored)
	}
	writeSeamTestFile(t, root, "scenarios/app/api/main.go", "package main\n")

	executables := func(list gitSourceListing) []string {
		t.Helper()
		files, err := seamTreeFilesWith(root, list)
		if err != nil {
			t.Fatal(err)
		}
		var found []string
		for _, file := range files {
			if ok, err := isExecutableArtifact(file.path); err != nil {
				t.Fatal(err)
			} else if ok {
				found = append(found, file.rel)
			}
		}
		return found
	}
	// Git's view of this tree: the source files, without the ignored output.
	viaGit := executables(listingOf("scenarios/app/api/main.go", residue))
	viaWalk := executables(noGit)

	if want := []string{residue}; !reflect.DeepEqual(viaGit, want) || !reflect.DeepEqual(viaWalk, want) {
		t.Fatalf("executables via git = %v, via walk = %v, want %v from both", viaGit, viaWalk, want)
	}
}

// The inventory skips vendored dependency trees and symlinks in either mode.
func TestSeamTreeFilesSkipDependencyTreesAndSymlinks(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"api/main.go", "api/vendor/dep/dep.go", "ui/node_modules/pkg/index.js", "docs/notes.md"} {
		writeSeamTestFile(t, root, rel, "x\n")
	}
	if err := os.Symlink(filepath.Join(root, "api", "main.go"), filepath.Join(root, "api", "link.go")); err != nil {
		t.Fatal(err)
	}
	files, err := seamTreeFilesWith(root, noGit)
	if err != nil {
		t.Fatal(err)
	}
	var rels []string
	for _, file := range files {
		rels = append(rels, file.rel)
	}
	if want := []string{"api/main.go", "docs/notes.md"}; !reflect.DeepEqual(rels, want) {
		t.Fatalf("inventory = %v, want %v", rels, want)
	}
}

// A checkout without Git that holds a local build reports no executables: the
// walk skips the virtualenv and packaged desktop output, while a tracked-looking
// .vrooli file stays visible to the seams that govern it.
func TestSeamTreeWalkIgnoresLocalBuildOutput(t *testing.T) {
	root := t.TempDir()
	writeExecutableFixture(t, root, "scenarios/app/api/.venv/bin/python")
	writeExecutableFixture(t, root, "scenarios/app/platforms/electron/dist-electron/linux-unpacked/app")
	writeExecutableFixture(t, root, "scenarios/app/cli/bin/cli")
	writeSeamTestFile(t, root, "scenarios/app/.vrooli/service.json", "{}\n")

	files, err := seamTreeFilesWith(root, noGit)
	if err != nil {
		t.Fatal(err)
	}
	var rels []string
	for _, file := range files {
		if ok, err := isExecutableArtifact(file.path); err != nil {
			t.Fatal(err)
		} else if ok {
			t.Errorf("walk counted build output %s as an executable", file.rel)
		}
		rels = append(rels, file.rel)
	}
	if want := []string{"scenarios/app/.vrooli/service.json"}; !reflect.DeepEqual(rels, want) {
		t.Fatalf("inventory = %v, want %v", rels, want)
	}
}
