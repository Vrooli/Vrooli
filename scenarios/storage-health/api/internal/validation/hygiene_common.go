package validation

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// hygiene_common.go holds the small AST/path helpers shared by the Tier-2
// persistence-hygiene analyzers (hygiene_*.go). Everything here is prefixed
// `hygiene` to avoid collisions with the schema/isolation analyzer tiers that
// register into the same package.

// hygieneExtraExemptSegments are path segments that the original scenario-auditor
// rules (rules/api/types.go isExemptPath) treat as exempt but the storage-health
// shared isExemptPath (scan.go) does not. The hygiene analyzers union these in so
// they reach the SAME exemption verdict as the rules they migrate — e.g. an
// api/internal/testutil/db helper using sql.Open is test scaffolding, not a
// production substrate violation (asserted by database_backoff_unit_test.go).
var hygieneExtraExemptSegments = map[string]struct{}{
	"test": {}, "testutil": {}, "migrate": {}, "migration": {},
	"initialization": {}, "init": {}, "tools": {},
}

// hygieneIsExemptPath reports whether a repo-relative path is exempt from the
// persistence-hygiene analyzers. It is the union of the package-shared
// isExemptPath (scan.go: _test.go, migrations/, scripts/, testdata/, vendored/
// generated) and the original auditor's extra test-infra exemptions, so the
// migrated rules keep their exempt-path semantics exactly.
func hygieneIsExemptPath(rel string) bool {
	if isExemptPath(rel) {
		return true
	}
	lower := strings.ToLower(rel)
	base := lower
	if idx := strings.LastIndex(lower, "/"); idx >= 0 {
		base = lower[idx+1:]
	}
	if strings.HasPrefix(base, "test_") {
		return true
	}
	for _, seg := range strings.Split(lower, "/") {
		if _, ok := hygieneExtraExemptSegments[seg]; ok {
			return true
		}
	}
	return false
}

// hygieneIsAPICorePath reports whether a path is inside packages/api-core/,
// the substrate that owns the RoutedDB seam and is therefore allowed to import
// drivers and call the raw sql factories. Mirrors scenario-auditor's
// isAPICorePath (rules/api/types.go). In practice CollectGoFiles only walks the
// scenario api/ surface so this never trips for collected files; it exists for
// parity with the original rules and for the parity tests that feed raw paths.
func hygieneIsAPICorePath(path string) bool {
	return strings.Contains(strings.ToLower(path), "packages/api-core/")
}

// hygieneImportsAPICoreDatabase reports whether the file imports
// github.com/vrooli/api-core/database (the compliant seam). Mirrors
// database_backoff.go's hasAPICoreImport.
func hygieneImportsAPICoreDatabase(file *ast.File) bool {
	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}
		if strings.Contains(strings.Trim(imp.Path.Value, `"`), "api-core/database") {
			return true
		}
	}
	return false
}

// hygieneImportsDatabaseSQL reports whether the file imports database/sql.
// Mirrors database_backoff.go's hasDatabaseSQLImport.
func hygieneImportsDatabaseSQL(file *ast.File) bool {
	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}
		if strings.Trim(imp.Path.Value, `"`) == "database/sql" {
			return true
		}
	}
	return false
}

// hygieneIsSelectorCall reports whether call is `pkg.fn(...)` (e.g. sql.Open).
func hygieneIsSelectorCall(call *ast.CallExpr, pkg, fn string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != fn {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == pkg
}

// hygieneIsSQLDBPtr reports whether expr is `*sql.DB`. Mirrors
// routed_database_handle_capture.go's isSQLDBPtr.
func hygieneIsSQLDBPtr(expr ast.Expr) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "DB" {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == "sql"
}

// hygieneLoc renders a repo-relative `path:line` location for a finding.
func hygieneLoc(relPath string, fset *token.FileSet, pos token.Pos) string {
	line := fset.Position(pos).Line
	if line <= 0 {
		return relPath
	}
	return relPath + ":" + strconv.Itoa(line)
}
