package runtimeapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func nativeSourcePointerFixture(t *testing.T, home, root string) {
	t.Helper()
	p := repocontract.SourceRootPointerPath(home)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(root+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestResolveNativeOperatorSourceBoundedAndCanonical(t *testing.T) {
	home := t.TempDir()
	for _, value := range []string{"relative", "/source/../other", strings.Repeat("x", 4097), "/source\x00injection"} {
		nativeSourcePointerFixture(t, home, value)
		if _, err := ResolveNativeOperatorSource(home); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
}

func TestResolveNativeOperatorSourceRejectsAlias(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(t.TempDir(), "source-pointer")
	if err := os.WriteFile(target, []byte("/source"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := repocontract.SourceRootPointerPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlink fixture unavailable: %v", err)
	}
	if _, err := ResolveNativeOperatorSource(home); err == nil {
		t.Fatal("source pointer alias accepted")
	}
}
