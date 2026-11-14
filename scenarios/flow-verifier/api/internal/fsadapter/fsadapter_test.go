package fsadapter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindReturnsSortedMatchesAndSkipsIgnoredDirs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "b/example.flow.json", "{}")
	writeFile(t, root, "a/example.flow.json", "{}")
	writeFile(t, root, "node_modules/hidden.flow.json", "{}")
	writeFile(t, root, "coverage/hidden.flow.json", "{}")

	got, err := Find(root, ".flow.json")
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	want := []string{"a/example.flow.json", "b/example.flow.json"}
	if len(got) != len(want) {
		t.Fatalf("Find() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Find()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTreeSHA256IgnoresTestsGeneratedModelsAndTestdata(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/tool\n")
	writeFile(t, root, "main.go", "package main\n")
	writeFile(t, root, "flow.schema.json", "{}")

	before, err := TreeSHA256(root)
	if err != nil {
		t.Fatalf("TreeSHA256() error = %v", err)
	}

	writeFile(t, root, "main_test.go", "package main\n")
	writeFile(t, root, "testdata/input.json", "{}")
	writeFile(t, root, "model.qnt", "module Model")
	writeFile(t, root, "model.formal.generated.json", "{}")

	after, err := TreeSHA256(root)
	if err != nil {
		t.Fatalf("TreeSHA256() error = %v", err)
	}
	if before != after {
		t.Fatalf("TreeSHA256 changed after ignored files: before=%s after=%s", before, after)
	}
}

func TestHashHelpersAreStable(t *testing.T) {
	if SHA256String("abc") != SHA256Bytes([]byte("abc")) {
		t.Fatal("SHA256String and SHA256Bytes disagree")
	}
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	if err := os.WriteFile(path, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, err := FileSHA256(path)
	if err != nil {
		t.Fatalf("FileSHA256() error = %v", err)
	}
	if hash != SHA256String("abc") {
		t.Fatalf("FileSHA256() = %s, want %s", hash, SHA256String("abc"))
	}
}

func writeFile(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
