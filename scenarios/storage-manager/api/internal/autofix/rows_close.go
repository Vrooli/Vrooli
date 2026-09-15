package autofix

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// previewRowsClose finds every Go file under the scenario's api/ surface that
// opens a `rows, err := …Query(…)` result set in the canonical shape (the query
// assignment is immediately followed by an `if err != nil { … }` guard) with no
// `rows.Close()` anywhere in the enclosing function, and proposes inserting
// `defer rows.Close()` right after the err guard.
//
// The detection deliberately mirrors hygiene_rows_close.go's leading-assignment
// case but only acts on the unambiguous canonical shape — a query whose error is
// checked by the very next statement. Anything more exotic (rows opened inside an
// if-init, returned to a caller, rebound, or guarded oddly) is left for the
// analyzer to report and a human to fix, so the fixer never makes a wrong edit.
func previewRowsClose(root string) ([]Candidate, error) {
	var out []Candidate
	for _, gf := range collectGoFiles(root) {
		src, err := os.ReadFile(gf.abs)
		if err != nil {
			continue
		}
		before := string(src)
		after, changed := fixedRowsClose(before)
		if !changed {
			continue
		}
		out = append(out, Candidate{
			RuleID:      RuleDBRowsNotClosed,
			FilePath:    gf.abs,
			Description: "Insert `defer <rows>.Close()` after the error guard for each query result set that is opened but never closed.",
			Before:      before,
			After:       after,
		})
	}
	return out, nil
}

// canFixRowsClose reports whether the file at findingPath (or, when empty, any
// Go file under root) currently has an unclosed rows result set this fixer would
// remediate.
func canFixRowsClose(root, findingPath string) bool {
	if path := rowsCloseTarget(root, findingPath); path != "" {
		src, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		_, changed := fixedRowsClose(string(src))
		return changed
	}
	for _, gf := range collectGoFiles(root) {
		src, err := os.ReadFile(gf.abs)
		if err != nil {
			continue
		}
		if _, changed := fixedRowsClose(string(src)); changed {
			return true
		}
	}
	return false
}

// rowsCloseTarget resolves a finding Location ("api/foo.go:42" or "api/foo.go")
// to an absolute Go file path under root, or "" when the location is empty or
// not a Go file.
func rowsCloseTarget(root, findingPath string) string {
	findingPath = strings.TrimSpace(findingPath)
	if findingPath == "" {
		return ""
	}
	if i := strings.LastIndex(findingPath, ":"); i >= 0 {
		// Strip a trailing :line suffix (but not a Windows drive letter).
		if suffix := findingPath[i+1:]; suffix != "" && isAllDigits(suffix) {
			findingPath = findingPath[:i]
		}
	}
	if !strings.HasSuffix(findingPath, ".go") {
		return ""
	}
	if filepath.IsAbs(findingPath) {
		return findingPath
	}
	return filepath.Join(root, filepath.FromSlash(findingPath))
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

// fixedRowsClose returns the source with `defer <rows>.Close()` inserted after
// each canonical unclosed-rows query, and whether any insertion was made. It is
// idempotent: a source that already closes every rows yields (src, false).
func fixedRowsClose(src string) (string, bool) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "src.go", src, parser.ParseComments)
	if err != nil {
		return src, false
	}

	type insertion struct {
		offset int // byte offset in src at which to insert the defer line
		indent string
		rows   string
	}
	var inserts []insertion

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			block, ok := n.(*ast.BlockStmt)
			if !ok {
				return true
			}
			for i := 0; i+1 < len(block.List); i++ {
				rows, ok := canonicalQueryRows(block.List[i])
				if !ok {
					continue
				}
				guard, ok := block.List[i+1].(*ast.IfStmt)
				if !ok || !guardChecksErr(guard) {
					continue
				}
				if funcClosesRows(fn.Body, rows) {
					continue
				}
				// Insert immediately after the err-guard's closing brace, on the
				// next line, at the statement's indentation.
				lineStart := fset.Position(block.List[i+1].Pos())
				inserts = append(inserts, insertion{
					offset: byteOffsetAtLineEnd(src, fset, guard.End()),
					indent: leadingIndent(src, fset, lineStart),
					rows:   rows,
				})
			}
			return true
		})
	}

	if len(inserts) == 0 {
		return src, false
	}

	// Apply insertions from the end backwards so earlier offsets stay valid.
	out := src
	for j := len(inserts) - 1; j >= 0; j-- {
		ins := inserts[j]
		line := "\n" + ins.indent + "defer " + ins.rows + ".Close()"
		out = out[:ins.offset] + line + out[ins.offset:]
	}
	return out, out != src
}

// canonicalQueryRows reports the rows variable name when stmt is a
// `rows, err := …Query(…)` (or QueryContext/Queryx/…) assignment whose first LHS
// ident is the rows handle. Mirrors hygiene_rows_close.go's query detection.
func canonicalQueryRows(stmt ast.Stmt) (string, bool) {
	assign, ok := stmt.(*ast.AssignStmt)
	if !ok || len(assign.Rhs) != 1 || len(assign.Lhs) < 2 {
		return "", false
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok {
		return "", false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	if _, ok := queryFuncNames[sel.Sel.Name]; !ok {
		return "", false
	}
	first, ok := assign.Lhs[0].(*ast.Ident)
	if !ok || first.Name == "_" || strings.Contains(strings.ToLower(first.Name), "err") {
		return "", false
	}
	return first.Name, true
}

var queryFuncNames = map[string]struct{}{
	"Query":             {},
	"QueryContext":      {},
	"Queryx":            {},
	"QueryxContext":     {},
	"NamedQuery":        {},
	"NamedQueryContext": {},
}

// guardChecksErr reports whether the if-statement is a `if err != nil { … }`
// shape (the canonical error guard that follows a query).
func guardChecksErr(guard *ast.IfStmt) bool {
	if guard.Init != nil {
		return false
	}
	bin, ok := guard.Cond.(*ast.BinaryExpr)
	if !ok || bin.Op != token.NEQ {
		return false
	}
	left, lok := bin.X.(*ast.Ident)
	right, rok := bin.Y.(*ast.Ident)
	if lok && rok {
		if strings.Contains(strings.ToLower(left.Name), "err") && right.Name == "nil" {
			return true
		}
		if strings.Contains(strings.ToLower(right.Name), "err") && left.Name == "nil" {
			return true
		}
	}
	return false
}

// funcClosesRows reports whether the function body contains any `<rows>.Close()`
// call (deferred or direct) — the idempotency check that stops a second fix.
func funcClosesRows(body *ast.BlockStmt, rows string) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Close" {
			return true
		}
		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == rows {
			found = true
			return false
		}
		return true
	})
	return found
}

// byteOffsetAtLineEnd returns the byte offset in src at the end of the line that
// contains pos. The insertion lands at the end of the err-guard's closing-brace
// line so the new `defer` appears on its own next line.
func byteOffsetAtLineEnd(src string, fset *token.FileSet, pos token.Pos) int {
	off := fset.Position(pos).Offset
	if off > len(src) {
		off = len(src)
	}
	// Advance to the end of the current line (the byte before the next '\n').
	for off < len(src) && src[off] != '\n' {
		off++
	}
	return off
}

// leadingIndent returns the whitespace prefix of the line containing pos.
func leadingIndent(src string, fset *token.FileSet, pos token.Position) string {
	lineStart := pos.Offset - (pos.Column - 1)
	if lineStart < 0 || lineStart > len(src) {
		return ""
	}
	i := lineStart
	for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
		i++
	}
	return src[lineStart:i]
}

// goFile is a discovered Go source file under the scenario's api/ surface.
type goFile struct {
	abs string
	rel string
}

// collectGoFiles walks root/api and returns non-test, non-exempt Go files,
// mirroring internal/validation/scan.go's exempt-path rules so the fixer's reach
// matches the analyzer's exactly.
func collectGoFiles(root string) []goFile {
	apiDir := filepath.Join(root, "api")
	info, err := os.Stat(apiDir)
	if err != nil || !info.IsDir() {
		return nil
	}
	var out []goFile
	_ = filepath.WalkDir(apiDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if isExemptGoPath(rel) {
			return nil
		}
		out = append(out, goFile{abs: path, rel: rel})
		return nil
	})
	return out
}

// isExemptGoPath mirrors internal/validation/scan.go's isExemptPath: test files,
// migrations/, scripts/, testdata/, vendored and generated code are exempt.
func isExemptGoPath(rel string) bool {
	if strings.HasSuffix(rel, "_test.go") {
		return true
	}
	for _, seg := range strings.Split(rel, "/") {
		switch seg {
		case "migrations", "scripts", "testdata", "node_modules", "vendor", "gen", "dist":
			return true
		}
	}
	return false
}
