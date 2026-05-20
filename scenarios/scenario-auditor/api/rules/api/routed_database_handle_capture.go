package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"html"
)

/*
Rule: Routed Database Handle Capture
Description: Scenarios must use *database.RoutedDB rather than capturing *sql.DB directly in struct fields, package-level vars, or function parameters.
Reason: A *sql.DB field is a captured handle to a single pool — handlers wired against it cannot participate in the per-request routing that *RoutedDB provides. Scenarios that hold raw *sql.DB are ineligible for the routed e2e path.
Category: api
Severity: medium
Standard: routed-test-db-v1
Targets: api

This rule complements routed_database_drivers (high). Together the two rules
gate test-genie's eligibility check: any high-severity violation from either
rule, plus any medium-severity from this rule, forces the fallback (restart)
path in the playbooks phase.

<test-case id="PASS-uses-routed-db-field" should-fail="false" path="api/main.go">
  <description>✅ SHOULD PASS: Field is *database.RoutedDB</description>
  <input language="go">
package main

import "github.com/vrooli/api-core/database"

type Server struct {
    DB *database.RoutedDB
}
  </input>
</test-case>

<test-case id="FAIL-struct-field-sqldb" should-fail="true" path="api/server.go">
  <description>❌ SHOULD FAIL: Struct field holds *sql.DB</description>
  <input language="go">
package main

import "database/sql"

type Server struct {
    DB *sql.DB
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>RoutedDB</expected-message>
</test-case>

<test-case id="FAIL-package-var-sqldb" should-fail="true" path="api/db.go">
  <description>❌ SHOULD FAIL: Package-level var of type *sql.DB</description>
  <input language="go">
package main

import "database/sql"

var db *sql.DB
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>RoutedDB</expected-message>
</test-case>

<test-case id="FAIL-func-param-sqldb" should-fail="true" path="api/handlers.go">
  <description>❌ SHOULD FAIL: Function parameter of type *sql.DB</description>
  <input language="go">
package main

import "database/sql"

func handle(db *sql.DB) error { return nil }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>RoutedDB</expected-message>
</test-case>

<test-case id="PASS-test-file-exempt" should-fail="false" path="api/handlers_test.go">
  <description>✅ SHOULD PASS: Test files exempt</description>
  <input language="go">
package main

import (
    "database/sql"
    "testing"
)

type fixture struct {
    DB *sql.DB
}

func TestX(t *testing.T) {
    _ = fixture{}
}
  </input>
</test-case>

*/

// CheckRoutedDatabaseHandleCapture flags captures of *sql.DB in struct
// fields, package-level vars, and function parameters.
func CheckRoutedDatabaseHandleCapture(content []byte, filePath string) []Violation {
	if isExemptPath(filePath) || isAPICorePath(filePath) {
		return nil
	}

	fset := token.NewFileSet()
	source := html.UnescapeString(string(content))
	file, err := parser.ParseFile(fset, filePath, source, 0)
	if err != nil {
		return nil
	}

	var violations []Violation

	report := func(line int, where string) {
		violations = append(violations, Violation{
			Type:        "routed_database_handle_capture",
			Severity:    "medium",
			Title:       "Captured *sql.DB bypasses RoutedDB seam",
			Description: "This file captures a *sql.DB handle in a " + where + ". Handlers wired against a raw *sql.DB cannot participate in per-request routing; the scenario is ineligible for the in-place e2e path.",
			FilePath:    filePath,
			LineNumber:  line,
			Recommendation: `Use *database.RoutedDB instead:

import "github.com/vrooli/api-core/database"

type Server struct {
    DB *database.RoutedDB
}

The *RoutedDB method surface mirrors *sql.DB's, so handler bodies need no other change. See docs/agent-system/routed-test-db.md.`,
			Standard: "routed-test-db-v1",
		})
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.GenDecl:
			// var declarations at package level.
			if decl.Tok != token.VAR {
				return true
			}
			for _, spec := range decl.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || vs.Type == nil {
					continue
				}
				if isSQLDBPtr(vs.Type) {
					report(fset.Position(vs.Pos()).Line, "package-level var")
				}
			}
		case *ast.StructType:
			for _, field := range decl.Fields.List {
				if isSQLDBPtr(field.Type) {
					report(fset.Position(field.Pos()).Line, "struct field")
				}
			}
		case *ast.FuncDecl:
			if decl.Type != nil && decl.Type.Params != nil {
				for _, field := range decl.Type.Params.List {
					if isSQLDBPtr(field.Type) {
						report(fset.Position(field.Pos()).Line, "function parameter")
					}
				}
			}
		}
		return true
	})

	return violations
}

// isSQLDBPtr returns true for *sql.DB expressions.
func isSQLDBPtr(expr ast.Expr) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != "DB" {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "sql"
}
