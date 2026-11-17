package buildinfo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWholeSourceDigestBindsBuildConstraintsAndNegatedIgnore(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "pkg")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	write := func(name, contents string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
	}
	digest := func() string {
		t.Helper()
		d, err := wholeSourceDigest(root, []string{"pkg"})
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	write("base.go", "package pkg\n")
	write("platform.go", "//go:build linux || darwin\n\npackage pkg\nconst Value = 1\n")
	first := digest()
	write("platform.go", "//go:build linux\n\npackage pkg\nconst Value = 1\n")
	if digest() == first {
		t.Fatal("platform selection did not affect digest")
	}
	write("platform.go", "//go:build !ignore\n\npackage pkg\nconst Value = 1\n")
	first = digest()
	write("platform.go", "//go:build !ignore\n\npackage pkg\nconst Value = 2\n")
	if digest() == first {
		t.Fatal("normally compiled !ignore file was excluded")
	}
	write("platform.go", "// +build ignore\npackage pkg\nconst Value = 1\n")
	first = digest()
	write("platform.go", "// +build ignore\npackage pkg\nconst Value = 2\n")
	if digest() == first {
		t.Fatal("unseparated legacy comment excluded compiled file")
	}
	write("platform.go", "/*\n//go:build ignore\n*/\npackage pkg\nconst Value = 1\n")
	first = digest()
	write("platform.go", "/*\n//go:build ignore\n*/\npackage pkg\nconst Value = 2\n")
	if digest() == first {
		t.Fatal("directive text in block comment excluded compiled file")
	}
	write("platform.go", "//go:build linux || ignore\n\npackage pkg\nconst Value = 1\n")
	first = digest()
	write("platform.go", "//go:build linux || ignore\n\npackage pkg\nconst Value = 2\n")
	if digest() == first {
		t.Fatal("disjunctive ignore file was excluded")
	}
}

func TestCodecStampBindsBuildConstraint(t *testing.T) {
	a := scanFixture(t, map[string]string{"fixture.go": "//go:build linux || darwin\n\n" + codecFixtureBase})
	b := scanFixture(t, map[string]string{"fixture.go": "//go:build linux\n\n" + codecFixtureBase})
	if a["Phase"] == b["Phase"] {
		t.Fatal("codec build constraint did not affect digest")
	}
}
