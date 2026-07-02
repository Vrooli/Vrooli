package evidence

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// [REQ:BH-EVD-001] Single-writer discipline, enforced structurally: the
// only write primitives this package may reach are the ones inside
// AppendAttestation, and every path literal they touch stays under
// coverage/manual-validations/. Any new os.WriteFile/Create/OpenFile with
// write flags outside AppendAttestation fails this test — test-genie's
// sync artifacts can never be written from here.
func TestEvidencePackageWritesOnlyTheManualLedger(t *testing.T) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	writeFns := map[string]bool{"WriteFile": true, "Create": true, "OpenFile": true, "MkdirAll": true, "Mkdir": true, "Remove": true, "RemoveAll": true, "Rename": true}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Clean(name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		var currentFn string
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.FuncDecl:
				currentFn = node.Name.Name
			case *ast.SelectorExpr:
				ident, ok := node.X.(*ast.Ident)
				if !ok || ident.Name != "os" {
					return true
				}
				if !writeFns[node.Sel.Name] {
					return true
				}
				if currentFn != "AppendAttestation" {
					t.Errorf("%s: os.%s used in %s — evidence writes are only allowed inside AppendAttestation (manual ledger)", name, node.Sel.Name, currentFn)
				}
			}
			return true
		})
	}
}

// The ledger path constant itself must stay under the one directory
// business-health owns.
func TestLedgerPathStaysOwned(t *testing.T) {
	if !strings.HasPrefix(manualLedgerRelPath, "coverage/manual-validations/") {
		t.Fatalf("manual ledger moved to %q — it must stay under coverage/manual-validations/", manualLedgerRelPath)
	}
	if strings.HasPrefix(syncSnapshotRelPath, "coverage/manual-validations/") {
		t.Fatal("sync snapshot path must not live in the business-health-owned directory")
	}
}
