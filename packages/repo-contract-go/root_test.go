package repocontract

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFindRepoRoot(t *testing.T) {
	root := repoRoot(t)
	start := filepath.Join(root, "packages", "repo-contract-go")

	got, err := FindRepoRoot(start)
	if err != nil {
		t.Fatalf("FindRepoRoot() error = %v", err)
	}
	if got != root {
		t.Fatalf("FindRepoRoot() = %q, want %q", got, root)
	}
}

func TestFindRepoRootWithFileStartPath(t *testing.T) {
	root := repoRoot(t)
	start := filepath.Join(root, "packages", "repo-contract-go", "load.go")

	got, err := FindRepoRoot(start)
	if err != nil {
		t.Fatalf("FindRepoRoot() error = %v", err)
	}
	if got != root {
		t.Fatalf("FindRepoRoot() = %q, want %q", got, root)
	}
}

func TestFindRepoRootRejectsEmptyAndMissingStart(t *testing.T) {
	_, err := FindRepoRoot(" ")
	assertErrorKind(t, err, ErrInvalidInput)

	_, err = FindRepoRoot(filepath.Join(t.TempDir(), "missing"))
	assertErrorKind(t, err, ErrNotFound)
}

func TestFindRepoRootReturnsNotFoundWhenNoMatchingRootExists(t *testing.T) {
	dir := t.TempDir()
	_, err := FindRepoRoot(dir)
	assertErrorKind(t, err, ErrNotFound)
}

func TestFindRepoRootSupportsSeamsForContractAndStatFailures(t *testing.T) {
	root := t.TempDir()
	doc := validContractDoc()
	writeContractFile(t, root, doc)
	for _, dir := range doc.Root.Markers.RequiredDirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	t.Run("load contract failure bubbles", func(t *testing.T) {
		oldLoad := loadContract
		loadContract = func(string) (*Contract, error) {
			return nil, &Error{Kind: ErrInvalidContract, Message: "boom"}
		}
		t.Cleanup(func() { loadContract = oldLoad })

		_, err := FindRepoRoot(root)
		assertErrorKind(t, err, ErrInvalidContract)
	})

	t.Run("stat failure bubbles", func(t *testing.T) {
		oldStat := statPath
		statPath = func(path string) (os.FileInfo, error) {
			if filepath.Base(path) == "scenarios" {
				return nil, errors.New("stat failed")
			}
			return oldStat(path)
		}
		t.Cleanup(func() { statPath = oldStat })

		_, err := candidateMatchesRootMarkers(root, doc.Root.Markers)
		assertErrorKind(t, err, ErrNotFound)
	})
}

func TestCandidateMatchesRootMarkers(t *testing.T) {
	root := t.TempDir()
	markers := RootMarkers{
		RequiredDirs:  []string{".vrooli", "scenarios"},
		RequiredFiles: []string{"go.mod"},
	}

	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.vrooli) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios"), 0o755); err != nil {
		t.Fatalf("MkdirAll(scenarios) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	ok, err := candidateMatchesRootMarkers(root, markers)
	if err != nil || !ok {
		t.Fatalf("candidateMatchesRootMarkers() = %v, %v", ok, err)
	}

	if err := os.Remove(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("Remove(go.mod) error = %v", err)
	}
	ok, err = candidateMatchesRootMarkers(root, markers)
	if err != nil {
		t.Fatalf("candidateMatchesRootMarkers() unexpected error = %v", err)
	}
	if ok {
		t.Fatal("candidateMatchesRootMarkers() = true, want false after removing go.mod")
	}
}
