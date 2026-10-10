//go:build linux

package nativeisolation

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeLeaderAndDescendantsRequireKernelPosture(t *testing.T) {
	raw := fmt.Sprintf("Uid:\t1234\t1234\t1234\t1234\nCapInh:\t00000000\nCapPrm:\t00000000\nCapEff:\t00000000\nCapBnd:\t00000000\nCapAmb:\t00000000\nNoNewPrivs:\t1\n")
	if validatePosture([]byte(raw), 1234) != nil {
		t.Fatal("valid disposable leader/descendant facts refused")
	}
	for _, bad := range []string{raw + "CapAmb:\t0\n", raw + "Uid:\t1234\t1234\t1234\t1234\n", "Uid:\t1234\t0\t1234\t1234\n", raw[:len(raw)-15]} {
		if validatePosture([]byte(bad), 1234) == nil {
			t.Fatal("incomplete or changed descendant posture accepted")
		}
	}
}
func TestWorkspaceAccessAndProtectedAliases(t *testing.T) {
	root := t.TempDir()
	if e := os.WriteFile(filepath.Join(root, "work.go"), []byte("work"), 0600); e != nil {
		t.Fatal(e)
	}
	key := filepath.Join(t.TempDir(), "key")
	if e := os.WriteFile(key, []byte("disposable-only"), 0600); e != nil {
		t.Fatal(e)
	}
	if checkWorkspace(root, []string{key}) != nil {
		t.Fatal("ordinary disposable workspace refused")
	}
	if e := os.Link(key, filepath.Join(root, "alias")); e != nil {
		t.Fatal(e)
	}
	if checkWorkspace(root, []string{key}) == nil {
		t.Fatal("protected inode alias accepted")
	}
	os.Remove(filepath.Join(root, "alias"))
	alias := filepath.Join(t.TempDir(), "repo-link")
	if e := os.Symlink(root, alias); e != nil {
		t.Fatal(e)
	}
	if checkWorkspace(alias, []string{key}) == nil {
		t.Fatal("workspace root symlink accepted")
	}
}
func TestImageContentDriftChangesQualifiedDigest(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "native")
	os.WriteFile(p, []byte("first"), 0700)
	a, e := treeDigest(root)
	if e != nil {
		t.Fatal(e)
	}
	os.WriteFile(p, []byte("changed"), 0700)
	b, e := treeDigest(root)
	if e != nil || a == b {
		t.Fatal("image drift not detected")
	}
}
