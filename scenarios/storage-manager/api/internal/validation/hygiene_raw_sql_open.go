package validation

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
)

// hygieneRawSQLOpen migrates scenario-auditor's database_backoff rule
// (scenarios/scenario-auditor/api/rules/api/database_backoff.go → CheckDatabaseBackoff).
// It flags a direct database/sql `sql.Open(...)` call in non-test, non-substrate
// code that does NOT import the api-core/database seam. The api-core seam wraps
// connection setup with routed-pool retry/backoff; a raw sql.Open bypasses it.
//
// Detection (faithful to the original AST logic):
//   - parse the file; if it imports api-core/database → compliant (no findings)
//   - if it does NOT import database/sql → nothing to check
//   - otherwise every `sql.Open(...)` call expression is a finding.
//
// New code: RAW_SQL_OPEN (ERROR).
type hygieneRawSQLOpen struct{}

func init() { register(&hygieneRawSQLOpen{}) }

func (hygieneRawSQLOpen) Name() string { return "hygiene-raw-sql-open" }

func (hygieneRawSQLOpen) Applies(ac AnalyzerContext) bool { return ac.IsGo() }

func (a hygieneRawSQLOpen) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	var findings []Finding
	for _, gf := range CollectGoFiles(ac) {
		findings = append(findings, a.analyzeSource(ReadFile(gf.AbsPath), gf.RelPath)...)
	}
	return findings, nil
}

// analyzeSource is the pure detection core, shared by Analyze and the parity
// tests. relPath is used only as the finding Location + for exemption (callers
// of CollectGoFiles already filter exempt paths, but the parity tests feed raw
// paths so this re-checks defensively).
func (a hygieneRawSQLOpen) analyzeSource(source, relPath string) []Finding {
	if hygieneIsExemptPath(relPath) || hygieneIsAPICorePath(relPath) {
		return nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, relPath, source, 0)
	if err != nil {
		return nil
	}
	if hygieneImportsAPICoreDatabase(file) {
		return nil
	}
	if !hygieneImportsDatabaseSQL(file) {
		return nil
	}

	var findings []Finding
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if hygieneIsSelectorCall(call, "sql", "Open") {
			findings = append(findings, Finding{
				Code:     "RAW_SQL_OPEN",
				Severity: SeverityError,
				Title:    "Direct sql.Open() without the api-core/database seam",
				Message: "This file calls sql.Open() directly instead of routing through " +
					"github.com/vrooli/api-core/database. The api-core seam configures the " +
					"connection pool with retry/backoff/jitter and exposes *RoutedDB, the " +
					"seam test-genie uses to install a runtime test pool without restarting " +
					"the scenario. A raw sql.Open bypasses both.",
				Location:    hygieneLoc(relPath, fset, call.Pos()),
				Remediation: "Replace sql.Open(...) with database.Open(ctx, database.Config{Driver: ...}) from github.com/vrooli/api-core/database.",
				Analyzer:    a.Name(),
			})
		}
		return true
	})
	return findings
}
