package attestation

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"

	apidb "github.com/vrooli/api-core/database"

	localdb "experience-manager/internal/database"
	testdb "github.com/vrooli/api-core/databasetest"
)

func TestSQLiteRepositoryAppendsAttestations(t *testing.T) { // [REQ:EXPERIEN-P1-004]
	db := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(Schema),
	); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	repo := NewSQLiteRepository(db)
	ctx := context.Background()
	for _, id := range []string{"a1", "a2"} {
		if err := repo.AppendAttestation(ctx, Attestation{
			ID:        id,
			Scenario:  "demo",
			PageID:    "home",
			ClaimID:   "intent-reviewed",
			Author:    "operator",
			Rationale: "reviewed against design notes",
			ExpiresAt: "2027-01-01T00:00:00Z",
			CreatedAt: "2026-07-05T00:00:00Z",
		}); err != nil {
			t.Fatalf("append %s: %v", id, err)
		}
	}
	rows, err := repo.ListAttestations(ctx, Filter{Scenario: "demo", PageID: "home", ClaimID: "intent-reviewed"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
}

func TestSQLiteRepositoryHasSingleWriteSeam(t *testing.T) { // [REQ:EXPERIEN-P1-004]
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test path")
	}
	path := filepath.Join(filepath.Dir(file), "sqlite.go")
	fset := token.NewFileSet()
	fileAST, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse sqlite.go: %v", err)
	}
	var execMethods []string
	ast.Inspect(fileAST, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if ok && sel.Sel.Name == "ExecContext" {
			fn := enclosingFunc(fileAST, call.Pos())
			execMethods = append(execMethods, fn)
		}
		return true
	})
	if len(execMethods) != 1 || execMethods[0] != "AppendAttestation" {
		t.Fatalf("ExecContext writers = %v, want [AppendAttestation]", execMethods)
	}
}

func enclosingFunc(file *ast.File, pos token.Pos) string {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Pos() <= pos && pos <= fn.End() {
			return fn.Name.Name
		}
	}
	return ""
}
