package retention

import (
	"path/filepath"
	"testing"
)

func TestPathContainsAcceptsShortChildNames(t *testing.T) {
	// Regression. The copy of this predicate that guarded os.RemoveAll carried
	// an extra `len(rel) >= 3` term, so every child of a protected root whose
	// name was one or two characters long was reported as not contained -- and
	// therefore not protected. ~/.vrooli/bin/go is such a child on a real host.
	root := filepath.Join(string(filepath.Separator), "home", "u", ".vrooli", "bin")
	for _, name := range []string{"a", "go", "ai", "cli", "codex", "vrooli"} {
		if !PathContains(root, filepath.Join(root, name)) {
			t.Fatalf("PathContains(%q, %q/%s) = false, want true", root, root, name)
		}
	}
}

func TestPathContainsRejectsNonDescendants(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "home", "u", ".vrooli", "bin")
	cases := []struct {
		name  string
		child string
	}{
		{"same path", root},
		{"parent", filepath.Dir(root)},
		{"sibling", filepath.Join(filepath.Dir(root), "cache")},
		{"prefix collision", root + "-backup"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if PathContains(root, tc.child) {
				t.Fatalf("PathContains(%q, %q) = true, want false", root, tc.child)
			}
		})
	}
}

func TestProtectedPathOverlapCoversSelfChildrenAndAncestors(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "home", "u", ".vrooli", "bin")
	cases := []struct {
		name      string
		candidate string
		want      bool
	}{
		{"the root itself", root, true},
		{"a child", filepath.Join(root, "plan-manager"), true},
		{"a short-named child", filepath.Join(root, "go"), true},
		{"a deep descendant", filepath.Join(root, "nested", "deeper"), true},
		// An ancestor matters because pruning it removes the protected root as
		// one top-level entry.
		{"an ancestor", filepath.Dir(root), true},
		{"a sibling", filepath.Join(filepath.Dir(root), "cache"), false},
		{"a name-prefix sibling", root + "-backup", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ProtectedPathOverlap(tc.candidate, []string{root}); got != tc.want {
				t.Fatalf("ProtectedPathOverlap(%q) = %v, want %v", tc.candidate, got, tc.want)
			}
		})
	}
}

func TestNormalizeProtectedRootsRejectsRelativeRoots(t *testing.T) {
	// A relative root would resolve against whichever working directory the
	// process happened to have, which is how a protection stops protecting
	// without anyone changing it.
	if _, err := NormalizeProtectedRoots([]string{"relative/bin"}); err == nil {
		t.Fatal("NormalizeProtectedRoots(relative) = nil error, want rejection")
	}
}

func TestNormalizeProtectedRootsDropsBlanks(t *testing.T) {
	absolute := filepath.Join(string(filepath.Separator), "home", "u", ".vrooli", "bin")
	got, err := NormalizeProtectedRoots([]string{"", "   ", absolute})
	if err != nil {
		t.Fatalf("NormalizeProtectedRoots: %v", err)
	}
	if len(got) != 1 || got[0] != absolute {
		t.Fatalf("NormalizeProtectedRoots = %v, want [%s]", got, absolute)
	}
}
