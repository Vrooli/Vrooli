package validation

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// hygieneRoutedDriver migrates scenario-auditor's routed_database_drivers rule
// (scenarios/scenario-auditor/api/rules/api/routed_database_drivers.go →
// CheckRoutedDatabaseDrivers). It flags two disjoint things that bypass the
// *database.RoutedDB seam:
//   - a NON-BLANK import of a database driver package (pgx, lib/pq,
//     mattn/go-sqlite3, modernc.org/sqlite). Blank imports (`_ "..."`) are
//     allowed — they register the driver without exposing its API.
//   - a direct sql.OpenDB(...) call (sql.Open is left to RAW_SQL_OPEN so the
//     two analyzers don't double-report the same construct).
//
// New code: ROUTED_DRIVER_IMPORT (ERROR).
type hygieneRoutedDriver struct{}

func init() { register(&hygieneRoutedDriver{}) }

// hygieneRoutedDriverPackages mirrors routed_database_drivers.go's
// routedDriverPackages.
var hygieneRoutedDriverPackages = []string{
	"github.com/jackc/pgx",
	"github.com/lib/pq",
	"github.com/mattn/go-sqlite3",
	"modernc.org/sqlite",
}

func (hygieneRoutedDriver) Name() string { return "hygiene-routed-driver" }

func (hygieneRoutedDriver) Applies(ac AnalyzerContext) bool { return ac.IsGo() }

func (a hygieneRoutedDriver) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	var findings []Finding
	for _, gf := range CollectGoFiles(ac) {
		findings = append(findings, a.analyzeSource(ReadFile(gf.AbsPath), gf.RelPath)...)
	}
	return findings, nil
}

func (a hygieneRoutedDriver) analyzeSource(source, relPath string) []Finding {
	if hygieneIsExemptPath(relPath) || hygieneIsAPICorePath(relPath) {
		return nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, relPath, source, 0)
	if err != nil {
		return nil
	}

	var findings []Finding

	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}
		path := strings.Trim(imp.Path.Value, `"`)
		if !hygieneMatchesRoutedDriver(path) {
			continue
		}
		// Blank imports are the recommended driver-registration pattern.
		if imp.Name != nil && imp.Name.Name == "_" {
			continue
		}
		findings = append(findings, Finding{
			Code:     "ROUTED_DRIVER_IMPORT",
			Severity: SeverityError,
			Title:    "Direct database-driver import bypasses the RoutedDB seam",
			Message: "This file imports the database driver " + path + " directly. The " +
				"api-core/database package is the only substrate scenarios should depend " +
				"on; it exposes *RoutedDB, the seam test-genie uses to install a runtime " +
				"test pool without restarting the scenario. Importing the driver directly " +
				"routes around the seam and disqualifies the scenario from the in-place " +
				"e2e path. (A blank import `_ \"" + path + "\"` for driver registration is fine.)",
			Location:    hygieneLoc(relPath, fset, imp.Pos()),
			Remediation: "Remove the named driver import and depend on github.com/vrooli/api-core/database (database.Open). If you only need driver registration, use a blank import.",
			Analyzer:    a.Name(),
		})
	}

	// sql.OpenDB — RAW_SQL_OPEN covers sql.Open; we cover OpenDB so the
	// disjoint partition catches the second pool factory.
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if hygieneIsSelectorCall(call, "sql", "OpenDB") {
			findings = append(findings, Finding{
				Code:     "ROUTED_DRIVER_IMPORT",
				Severity: SeverityError,
				Title:    "Direct sql.OpenDB bypasses the RoutedDB seam",
				Message: "sql.OpenDB constructs a connection pool outside the " +
					"api-core/database substrate. *RoutedDB is the seam scenarios must " +
					"depend on; sql.OpenDB bypasses it and disqualifies the scenario from " +
					"the routed e2e path.",
				Location:    hygieneLoc(relPath, fset, call.Pos()),
				Remediation: "Use database.Open(ctx, database.Config{...}) from github.com/vrooli/api-core/database instead of sql.OpenDB.",
				Analyzer:    a.Name(),
			})
		}
		return true
	})

	return findings
}

// hygieneMatchesRoutedDriver mirrors routed_database_drivers.go's
// matchesRoutedDriver: exact match or prefix-with-slash.
func hygieneMatchesRoutedDriver(importPath string) bool {
	for _, prefix := range hygieneRoutedDriverPackages {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return true
		}
	}
	return false
}
