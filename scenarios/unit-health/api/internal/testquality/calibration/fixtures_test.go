package calibration

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unit-health/internal/testquality"
)

func TestDevelopmentFixturesAndCompleteRetainedInventory(t *testing.T) {
	cases, err := LoadCases("../testdata", "development.json", "development")
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 28 {
		t.Fatalf("expected 28 materialized development cases, got %d", len(cases))
	}
	rows, err := Inventory("../testdata", cases)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 80 {
		t.Fatalf("retained required cases lost: %d", len(rows))
	}
	pending, materialized := 0, 0
	for _, row := range rows {
		switch row.Disposition {
		case "pending-fixture":
			pending++
		case "development-fixture", "native-fixture", "development-and-native-fixture", "context-fixture":
			materialized++
		default:
			t.Fatalf("unclassified case %+v", row)
		}
	}
	if pending != 0 || materialized != 80 {
		t.Fatalf("inventory hid unfinished work: pending=%d materialized=%d", pending, materialized)
	}
}

func TestGoFixtureLocationsMatchSyntaxBeforeAnalyzerImplementation(t *testing.T) {
	cases, err := LoadCases("../testdata", "development.json", "development")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		if c.Input.Profile != "go-syntax-v1" {
			continue
		}
		for _, e := range c.Expected {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, filepath.Join("../testdata", c.Input.Root, e.File), nil, parser.ParseComments)
			if e.Reason == testquality.ParseFailure {
				if err == nil {
					t.Fatalf("%s malformed fixture parsed successfully", c.Input.ID)
				}
				continue
			}
			if err != nil {
				t.Fatalf("%s unexpected syntax error: %v", c.Input.ID, err)
			}
			found := false
			parts := strings.Split(e.TestID, "/")
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name.Name != parts[0] {
					continue
				}
				pos := fset.Position(fn.Pos())
				if len(parts) == 1 {
					found = pos.Line == e.Line && pos.Column == e.Column
					continue
				}
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok || len(call.Args) < 2 {
						return true
					}
					selector, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || selector.Sel.Name != "Run" {
						return true
					}
					literal, ok := call.Args[0].(*ast.BasicLit)
					if !ok {
						return true
					}
					name, err := strconv.Unquote(literal.Value)
					if err != nil || name != parts[1] {
						return true
					}
					pos := fset.Position(call.Pos())
					if pos.Line == e.Line && pos.Column == e.Column {
						found = true
					}
					return true
				})
			}
			if !found {
				t.Errorf("%s: %s is not at %s:%d:%d", c.Input.ID, e.TestID, e.File, e.Line, e.Column)
			}
		}
	}
}

func TestCatalogCaseReferencesExistInRetainedInventory(t *testing.T) {
	// Catalog IDs are validated independently of analyzer output.
	rows, err := Inventory("../testdata", nil)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, row := range rows {
		ids[row.ID] = true
	}
	catalog, err := testquality.LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range catalog.Rules {
		for _, id := range rule.CalibrationCaseIDs {
			if !ids[id] {
				t.Fatalf("catalog refers to absent case %s", id)
			}
		}
	}
}
