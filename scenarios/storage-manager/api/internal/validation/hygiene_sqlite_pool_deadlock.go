package validation

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
)

// hygieneSQLitePoolDeadlock is a NEW analyzer (no scenario-auditor predecessor).
// It catches the specific deadlock the project memory records: a
// single-connection SQLite pool (MaxOpenConns: 1) combined with a NESTED query
// issued while an outer rows cursor is still open. With one connection, the
// inner query can never get a connection (the outer rows holds it), so the
// goroutine deadlocks.
//
// CRITICAL: `MaxOpenConns: 1` ALONE is NOT a finding — it is the standard,
// correct SQLite config (storage-manager itself uses it). The finding requires
// BOTH signals in the same file:
//   - a `MaxOpenConns: 1` composite-literal element (the pool-of-1 signal), AND
//   - a `for rows.Next()` loop whose body issues ANOTHER Query/Exec/QueryRow
//     call (the nested query that needs a second connection).
//
// New code: SQLITE_POOL_DEADLOCK (ERROR).
type hygieneSQLitePoolDeadlock struct{}

func init() { register(&hygieneSQLitePoolDeadlock{}) }

func (hygieneSQLitePoolDeadlock) Name() string { return "hygiene-sqlite-pool-deadlock" }

func (hygieneSQLitePoolDeadlock) Applies(ac AnalyzerContext) bool { return ac.IsGo() }

func (a hygieneSQLitePoolDeadlock) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	var findings []Finding
	for _, gf := range CollectGoFiles(ac) {
		findings = append(findings, a.analyzeSource(ReadFile(gf.AbsPath), gf.RelPath)...)
	}
	return findings, nil
}

// hygieneNestedQueryMethods are the query methods whose appearance inside an
// open `for rows.Next()` loop constitutes the nested query.
var hygieneNestedQueryMethods = map[string]struct{}{
	"Query": {}, "QueryContext": {}, "QueryRow": {}, "QueryRowContext": {},
	"Exec": {}, "ExecContext": {},
	"Queryx": {}, "QueryxContext": {}, "NamedQuery": {}, "NamedQueryContext": {},
}

func (a hygieneSQLitePoolDeadlock) analyzeSource(source, relPath string) []Finding {
	if hygieneIsExemptPath(relPath) || hygieneIsAPICorePath(relPath) {
		return nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, relPath, source, 0)
	if err != nil {
		return nil
	}

	// Signal 1: pool of exactly one. Required — without it, a nested query in
	// an open-rows loop is at most a performance smell, not a deadlock.
	if !hygieneHasMaxOpenConnsOne(file) {
		return nil
	}

	// Signal 2: a nested query inside an open `for rows.Next()` loop.
	var pos token.Pos
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}
		forStmt, ok := n.(*ast.ForStmt)
		if !ok || !hygieneIsRowsNextLoop(forStmt) || forStmt.Body == nil {
			return true
		}
		ast.Inspect(forStmt.Body, func(inner ast.Node) bool {
			if found {
				return false
			}
			call, ok := inner.(*ast.CallExpr)
			if !ok {
				return true
			}
			if hygieneIsNestedQueryCall(call) {
				pos, found = call.Pos(), true
				return false
			}
			return true
		})
		return true
	})
	if !found {
		return nil
	}

	return []Finding{{
		Code:     "SQLITE_POOL_DEADLOCK",
		Severity: SeverityError,
		Title:    "Nested query inside an open rows loop on a single-connection SQLite pool",
		Message: "This file configures a single-connection SQLite pool (MaxOpenConns: 1) AND " +
			"issues another query inside a `for rows.Next()` loop while the outer rows " +
			"cursor still holds the only connection. The inner query can never acquire a " +
			"connection, so the goroutine deadlocks. (MaxOpenConns: 1 on its own is the " +
			"correct SQLite config — the nested query is the bug.)",
		Location:    hygieneLoc(relPath, fset, pos),
		Remediation: "Read the outer rows fully (collect the ids into a slice) BEFORE issuing the inner query, or raise MaxOpenConns above 1.",
		Analyzer:    a.Name(),
	}}
}

// hygieneHasMaxOpenConnsOne reports whether the file contains a composite-literal
// element `MaxOpenConns: 1` (the pool-of-one signal). It also accepts a method
// call `<x>.SetMaxOpenConns(1)`.
func hygieneHasMaxOpenConnsOne(file *ast.File) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}
		switch node := n.(type) {
		case *ast.KeyValueExpr:
			if key, ok := node.Key.(*ast.Ident); ok && key.Name == "MaxOpenConns" && hygieneIsIntLit(node.Value, 1) {
				found = true
				return false
			}
		case *ast.CallExpr:
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "SetMaxOpenConns" &&
				len(node.Args) == 1 && hygieneIsIntLit(node.Args[0], 1) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func hygieneIsIntLit(expr ast.Expr, want int) bool {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return false
	}
	return lit.Value == itoaSmall(want)
}

// itoaSmall renders a small non-negative int without importing strconv into
// this file (strconv is already pulled by the package elsewhere, but keeping the
// helper local avoids a second import line just for one literal compare).
func itoaSmall(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// hygieneIsRowsNextLoop reports whether forStmt is `for <x>.Next() { ... }` (an
// open rows cursor loop). It matches any selector `.Next()` condition, which in
// a data-access file is the rows/cursor iteration.
func hygieneIsRowsNextLoop(forStmt *ast.ForStmt) bool {
	if forStmt == nil || forStmt.Cond == nil {
		return false
	}
	call, ok := forStmt.Cond.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Next"
}

// hygieneIsNestedQueryCall reports whether call is `<x>.Query/Exec/...(...)` —
// a query method invocation (the nested query). Detection is by method name on a
// selector; we do not require a SQL-string arg here because the inner query may
// pass a prepared statement or a built query string variable.
func hygieneIsNestedQueryCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	_, ok = hygieneNestedQueryMethods[sel.Sel.Name]
	return ok
}
