package validation

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
)

// hygieneHandleCapture migrates scenario-auditor's
// routed_database_handle_capture rule
// (scenarios/scenario-auditor/api/rules/api/routed_database_handle_capture.go →
// CheckRoutedDatabaseHandleCapture). It flags captures of a raw *sql.DB handle
// in a struct field, a package-level var, or a function parameter. A captured
// *sql.DB is a handle to a single pool — handlers wired against it cannot
// participate in the per-request routing *RoutedDB provides.
//
// New code: SQL_DB_HANDLE_CAPTURE (WARNING).
type hygieneHandleCapture struct{}

func init() { register(&hygieneHandleCapture{}) }

func (hygieneHandleCapture) Name() string { return "hygiene-handle-capture" }

func (hygieneHandleCapture) Applies(ac AnalyzerContext) bool { return ac.IsGo() }

func (a hygieneHandleCapture) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	var findings []Finding
	for _, gf := range CollectGoFiles(ac) {
		findings = append(findings, a.analyzeSource(ReadFile(gf.AbsPath), gf.RelPath)...)
	}
	return findings, nil
}

func (a hygieneHandleCapture) analyzeSource(source, relPath string) []Finding {
	if hygieneIsExemptPath(relPath) || hygieneIsAPICorePath(relPath) {
		return nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, relPath, source, 0)
	if err != nil {
		return nil
	}

	var findings []Finding
	report := func(pos token.Pos, where string) {
		findings = append(findings, Finding{
			Code:     "SQL_DB_HANDLE_CAPTURE",
			Severity: SeverityWarning,
			Title:    "Captured *sql.DB bypasses the RoutedDB seam",
			Message: "This file captures a *sql.DB handle in a " + where + ". A *sql.DB is a " +
				"handle to a single pool; handlers wired against it cannot participate in " +
				"the per-request routing that *database.RoutedDB provides, so the scenario " +
				"becomes ineligible for the in-place e2e path.",
			Location:    hygieneLoc(relPath, fset, pos),
			Remediation: "Hold *database.RoutedDB (from github.com/vrooli/api-core/database) instead of *sql.DB; its method surface mirrors *sql.DB so handler bodies need no other change.",
			Analyzer:    a.Name(),
		})
	}

	// Mirrors routed_database_handle_capture.go's ast.Inspect switch:
	// package-level vars, struct fields, and function parameters of *sql.DB.
	ast.Inspect(file, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.GenDecl:
			if decl.Tok != token.VAR {
				return true
			}
			for _, spec := range decl.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || vs.Type == nil {
					continue
				}
				if hygieneIsSQLDBPtr(vs.Type) {
					report(vs.Pos(), "package-level var")
				}
			}
		case *ast.StructType:
			for _, field := range decl.Fields.List {
				if hygieneIsSQLDBPtr(field.Type) {
					report(field.Pos(), "struct field")
				}
			}
		case *ast.FuncDecl:
			if decl.Type != nil && decl.Type.Params != nil {
				for _, field := range decl.Type.Params.List {
					if hygieneIsSQLDBPtr(field.Type) {
						report(field.Pos(), "function parameter")
					}
				}
			}
		}
		return true
	})

	return findings
}
