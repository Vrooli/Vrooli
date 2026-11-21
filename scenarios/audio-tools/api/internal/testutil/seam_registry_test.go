package testutil_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSeamRegistry is the L5 drift gate for the seam catalog.
//
// It walks every non-test .go file under api/internal and api/integrations,
// collects every interface declaration that carries a `// seam:` comment
// within its leading doc block, and asserts that the interface's package +
// name (e.g. "clock.Clock", "envx.Reader", "sttchain.Provider") appears
// somewhere in docs/internal/SEAMS.md.
//
// Failure modes the test catches:
//   - Someone adds a `// seam:` tag without adding the row to SEAMS.md.
//   - Someone renames or moves a tagged interface without updating SEAMS.md.
//
// It does NOT enforce the reverse direction (every SEAMS.md row has a tag)
// because the markdown lists several non-interface ("concrete") seams
// (sqlite stores, pub/sub session, multipart handlers). Those rows live
// in SEAMS.md without a Go interface to tag.
func TestSeamRegistry(t *testing.T) {
	// We run from internal/testutil/. The api root is two levels up.
	apiRoot := filepath.Join("..", "..")
	seamsMD := mustReadSeamsMD(t)

	tagged := collectTaggedSeams(t, apiRoot)
	if len(tagged) == 0 {
		t.Fatal("found zero `// seam:` tags — registry walk is misconfigured")
	}

	// SEAMS.md cites interfaces two ways: as the qualified Go reference
	// `<pkg>.<Name>` in prose, or in the "Interface" row of a seam table
	// as `path/to/file.go::<Name>`. Accept either form so the markdown
	// stays readable without forcing redundant qualification.
	for _, s := range tagged {
		qualified := s.Package + "." + s.Name
		tableForm := "::" + s.Name
		if !strings.Contains(seamsMD, qualified) && !strings.Contains(seamsMD, tableForm) {
			t.Errorf("seam %s tagged at %s but not mentioned in SEAMS.md (neither %q nor %q found)",
				qualified, s.Pos, qualified, tableForm)
		}
	}
}

type seamTag struct {
	Package string
	Name    string
	Pos     string
}

func mustReadSeamsMD(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "docs", "internal", "SEAMS.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read SEAMS.md (%s): %v", path, err)
	}
	return string(b)
}

func collectTaggedSeams(t *testing.T, root string) []seamTag {
	t.Helper()
	fset := token.NewFileSet()
	var out []seamTag

	for _, sub := range []string{"internal", "integrations"} {
		base := filepath.Join(root, sub)
		if _, err := os.Stat(base); os.IsNotExist(err) {
			continue
		}
		filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				// Skip the testutil subtree itself and any generated/ trees.
				name := d.Name()
				if name == "testutil" || name == "generated" || name == "mocks" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			f, perr := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if perr != nil {
				return nil // tolerate transient parse failures elsewhere
			}
			pkg := f.Name.Name
			for _, decl := range f.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.TYPE {
					continue
				}
				for _, spec := range gen.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if _, isIface := ts.Type.(*ast.InterfaceType); !isIface {
						continue
					}
					doc := docTextFor(gen, ts)
					if !strings.Contains(doc, "seam:") {
						continue
					}
					out = append(out, seamTag{
						Package: pkg,
						Name:    ts.Name.Name,
						Pos:     fset.Position(ts.Pos()).String(),
					})
				}
			}
			return nil
		})
	}
	return out
}

// docTextFor returns the doc-comment text immediately preceding the
// interface declaration. The doc may be on the GenDecl (single-type
// declarations) or on the TypeSpec (grouped declarations).
func docTextFor(gen *ast.GenDecl, ts *ast.TypeSpec) string {
	var parts []string
	if gen.Doc != nil {
		parts = append(parts, gen.Doc.Text())
	}
	if ts.Doc != nil {
		parts = append(parts, ts.Doc.Text())
	}
	return strings.Join(parts, "\n")
}
