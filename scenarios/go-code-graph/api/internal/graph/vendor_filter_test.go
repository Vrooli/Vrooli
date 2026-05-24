package graph

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"
)

// TestFilterVendorPackages exercises the directory-based vendor-filter
// used to honor LoadOptions.IncludeVendor=false (REQ-P1-003). It runs
// against the production filterVendorPackages helper directly to keep
// the test in-package and synthetic — building a real Go module with a
// populated vendor/ tree just to validate this filter would be overkill.
func TestFilterVendorPackages(t *testing.T) {
	root := filepath.FromSlash("/scenario")
	pkgs := []*packages.Package{
		{
			PkgPath: "example.com/app",
			GoFiles: []string{filepath.Join(root, "app", "main.go")},
		},
		{
			PkgPath: "github.com/foo/bar",
			GoFiles: []string{filepath.Join(root, "vendor", "github.com", "foo", "bar", "bar.go")},
		},
		{
			PkgPath: "example.com/app/internal/util",
			GoFiles: []string{filepath.Join(root, "internal", "util", "util.go")},
		},
		{
			PkgPath:         "github.com/baz/qux",
			CompiledGoFiles: []string{filepath.Join(root, "vendor", "github.com", "baz", "qux", "qux.go")},
		},
		// Package whose name happens to contain "vendor" but is not
		// actually a vendored package — must be preserved.
		{
			PkgPath: "example.com/app/myvendorthing",
			GoFiles: []string{filepath.Join(root, "myvendorthing", "x.go")},
		},
	}

	out := filterVendorPackages(pkgs)
	gotPaths := make([]string, 0, len(out))
	for _, p := range out {
		gotPaths = append(gotPaths, p.PkgPath)
	}

	wantKept := map[string]bool{
		"example.com/app":                     true,
		"example.com/app/internal/util":       true,
		"example.com/app/myvendorthing":       true,
	}
	wantDropped := map[string]bool{
		"github.com/foo/bar": true,
		"github.com/baz/qux": true,
	}

	for _, p := range gotPaths {
		if wantDropped[p] {
			t.Errorf("vendored package not filtered: %s", p)
		}
		delete(wantKept, p)
	}
	for p := range wantKept {
		t.Errorf("non-vendored package was filtered: %s", p)
	}
}
