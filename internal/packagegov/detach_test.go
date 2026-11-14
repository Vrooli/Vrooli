package packagegov

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hardlinkedConsumer builds a package shipping dist/ and a consumer whose
// installed index.js is a hard link to the package's, as a file: install makes.
func hardlinkedConsumer(t *testing.T) (packageRoot, consumerFile string) {
	t.Helper()
	root := t.TempDir()
	packageRoot = filepath.Join(root, "packages", "demo")
	mustWrite(t, filepath.Join(packageRoot, "package.json"), `{"name":"@vrooli/demo","files":["dist"]}`)
	mustWrite(t, filepath.Join(packageRoot, "dist", "index.js"), "export const v = 'old';\n")
	consumerFile = filepath.Join(root, "scenarios", "alpha", "ui", "node_modules", "@vrooli", "demo", "dist", "index.js")
	if err := os.MkdirAll(filepath.Dir(consumerFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(packageRoot, "dist", "index.js"), consumerFile); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}
	return packageRoot, consumerFile
}

func TestRunCommandsLeavesHardlinkedConsumerCopiesOnTheirSnapshot(t *testing.T) {
	packageRoot, consumerFile := hardlinkedConsumer(t)
	// An in-place rewrite plus an added module: what tsc does on a rebuild.
	rebuild := CommandSpec{Name: "build", Run: []string{"sh", "-c", "printf \"export * from './added.js';\\n\" > dist/index.js && printf 'export const a = 1;\\n' > dist/added.js"}}
	if err := RunCommandsWithOptions(packageRoot, []CommandSpec{rebuild}, io.Discard, io.Discard, CommandOptions{Stdin: strings.NewReader("")}); err != nil {
		t.Fatalf("RunCommandsWithOptions: %v", err)
	}

	if got := readString(t, filepath.Join(packageRoot, "dist", "index.js")); !strings.Contains(got, "added.js") {
		t.Fatalf("package output = %q, want the rebuilt module", got)
	}
	if got := readString(t, consumerFile); got != "export const v = 'old';\n" {
		t.Fatalf("consumer copy = %q, want its old snapshot untouched", got)
	}
}

func TestDetachShippedFilesKeepsSingleLinkFilesAndUnshippedPaths(t *testing.T) {
	packageRoot, _ := hardlinkedConsumer(t)
	mustWrite(t, filepath.Join(packageRoot, "src", "index.ts"), "x")
	src := filepath.Join(packageRoot, "src", "index.ts")
	srcLink := filepath.Join(t.TempDir(), "src-link.ts")
	if err := os.Link(src, srcLink); err != nil {
		t.Fatal(err)
	}
	before, _ := os.Stat(src)
	if err := DetachShippedFiles(packageRoot); err != nil {
		t.Fatalf("DetachShippedFiles: %v", err)
	}
	after, _ := os.Stat(src)
	if !os.SameFile(before, after) {
		t.Fatal("an unshipped path was detached")
	}
	shipped, _ := os.Stat(filepath.Join(packageRoot, "dist", "index.js"))
	if linkCount(shipped) != 1 {
		t.Fatalf("shipped file still has %d links", linkCount(shipped))
	}
	entries, _ := os.ReadDir(filepath.Join(packageRoot, "dist"))
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".detach-") {
			t.Fatalf("temporary file %s left behind", entry.Name())
		}
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readString(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
