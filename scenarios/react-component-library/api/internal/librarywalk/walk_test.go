package librarywalk

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSourcesExcludesQuarantineDependenciesAndHonorsAssetScope(t *testing.T) {
	root := t.TempDir()
	paths := []string{
		"components/Button/versions/1.0.0/Button.tsx",
		"components/Button/versions/1.0.0/Button.css",
		"components/Card/versions/1.0.0/Card.tsx",
		".retired/Button/versions/0.1.0/Button.tsx",
		"node_modules/ignored.tsx",
	}
	for _, relative := range paths {
		path := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("export {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	all, err := Sources(root, FullCorpus())
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("full source count = %d, want 3: %v", len(all), all)
	}
	scoped, err := Sources(root, Scope{Assets: map[string]struct{}{"controls.button": {}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(scoped) != 2 {
		t.Fatalf("scoped source count = %d, want 2: %v", len(scoped), scoped)
	}
	kinds, err := Kinds(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(kinds) != 1 || kinds[0] != "components" {
		t.Fatalf("kinds = %v, want [components]", kinds)
	}
}

func TestOneWalker(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir(filepath.Join(root, "scenarios", "react-component-library", "api"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") || strings.HasSuffix(filepath.ToSlash(path), "/internal/librarywalk/walk.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(data), "filepath.Walk") {
			t.Fatalf("%s opens a private walker; use internal/librarywalk", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBuildSetScopesVersionDirectoriesAndSources(t *testing.T) {
	root := t.TempDir()
	for _, relative := range []string{
		"components/Button/versions/1.0.0/Button.tsx",
		"components/Card/versions/1.0.0/Card.tsx",
	} {
		path := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("export {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	set, err := BuildSet(context.Background(), root, Scope{Assets: map[string]struct{}{"controls.button": {}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Files) != 1 || len(set.Versions) != 1 {
		t.Fatalf("set = %#v, want one scoped source and version directory", set)
	}
}

func TestFilesClosureIncludesLockedDependencies(t *testing.T) {
	root := t.TempDir()
	for _, relative := range []string{
		"components/Button/versions/1.0.0/Button.tsx",
		"components/Icon/versions/1.0.0/Icon.tsx",
		"components/Card/versions/1.0.0/Card.tsx",
	} {
		path := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("export {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	lock := filepath.Join(root, "components/Button/versions/1.0.0/dependencies.json")
	if err := os.WriteFile(lock, []byte(`{"dependencies":[{"libraryId":"react-component-library:Icon"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	set, err := Files(context.Background(), root, Scope{Assets: map[string]struct{}{"controls.button": {}}}, ReadsClosure)
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Versions) != 2 || len(set.Files) != 2 {
		t.Fatalf("closure set = %#v, want button and icon only", set)
	}
}

func TestTraversalPrimitiveIsCentralized(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	apiRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	count := 0
	var owners []string
	err := filepath.WalkDir(apiRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		n := strings.Count(string(data), "filepath.WalkDir(")
		if n > 0 {
			count += n
			owners = append(owners, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || len(owners) != 1 || owners[0] != filepath.Join(filepath.Dir(thisFile), "walk.go") {
		t.Fatalf("filepath.WalkDir owners = %v (count %d), want only librarywalk/walk.go", owners, count)
	}
}
