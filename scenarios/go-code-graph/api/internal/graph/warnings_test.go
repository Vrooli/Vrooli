package graph

import (
	"testing"

	"golang.org/x/tools/go/packages"
)

// TestPackageWarningsClassifyKinds asserts each packages.Error.Kind
// maps to the right typed WarningKind. The function is unexported, so
// the test uses the internal package name.
func TestPackageWarningsClassifyKinds(t *testing.T) {
	t.Parallel()
	pkg := &packages.Package{
		Errors: []packages.Error{
			{Kind: packages.ParseError, Pos: "/scenario/a.go:1:1", Msg: "expected ';'"},
			{Kind: packages.TypeError, Pos: "/scenario/b.go:5:2", Msg: "undeclared name: Foo"},
			{Kind: packages.ListError, Pos: "", Msg: "could not import missing/pkg"},
		},
	}
	got := packageWarnings(pkg)
	if len(got) != 3 {
		t.Fatalf("want 3 warnings, got %d", len(got))
	}
	gotByKind := map[WarningKind]Warning{}
	for _, w := range got {
		gotByKind[w.Kind] = w
	}
	if _, ok := gotByKind[WarningKindParseError]; !ok {
		t.Errorf("missing parse_error: %+v", got)
	}
	if _, ok := gotByKind[WarningKindTypeCheckFailure]; !ok {
		t.Errorf("missing type_check_failure: %+v", got)
	}
	if _, ok := gotByKind[WarningKindUnresolvedImport]; !ok {
		t.Errorf("missing unresolved_import: %+v", got)
	}
	// Parse-error warning should carry the file path extracted from
	// the position string.
	if w := gotByKind[WarningKindParseError]; w.File != "/scenario/a.go" {
		t.Errorf("parse_error file: want /scenario/a.go, got %q", w.File)
	}
}

// TestPackageWarningsNilPackage exercises the safety branch.
func TestPackageWarningsNilPackage(t *testing.T) {
	t.Parallel()
	if got := packageWarnings(nil); got != nil {
		t.Fatalf("nil package should yield nil warnings, got %+v", got)
	}
	if got := packageWarnings(&packages.Package{}); got != nil {
		t.Fatalf("no errors should yield nil warnings, got %+v", got)
	}
}
