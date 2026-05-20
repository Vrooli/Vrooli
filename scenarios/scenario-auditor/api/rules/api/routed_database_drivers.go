package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"html"
	"strings"
)

/*
Rule: Routed Database Drivers
Description: Scenarios must not import database drivers or call sql.OpenDB directly; the api-core/database package is the single substrate for database access, including the routed test-pool seam.
Reason: Direct driver imports and sql.OpenDB calls bypass the *RoutedDB seam, which test-genie uses to install a runtime test pool without restarting the scenario. Scenarios that route around the seam are ineligible for the in-place e2e path.
Category: api
Severity: high
Standard: routed-test-db-v1
Targets: api

Coordinates with database_backoff: that rule already flags raw sql.Open
calls. This rule deliberately does NOT scan for sql.Open to avoid
double-reporting. Both rules consume the same isExemptPath exemption list,
plus this rule additionally exempts packages/api-core/ so the substrate
itself can carry the driver wiring.

<test-case id="PASS-uses-routed-db" should-fail="false">
  <description>✅ SHOULD PASS: Uses api-core/database.Open() and *RoutedDB</description>
  <input language="go">
package main

import (
    "context"
    "log"

    "github.com/vrooli/api-core/database"
)

func main() {
    db, err := database.Open(context.Background(), database.Config{Driver: "postgres"})
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
}
  </input>
</test-case>

<test-case id="FAIL-imports-pgx" should-fail="true" path="api/main.go">
  <description>❌ SHOULD FAIL: Imports github.com/jackc/pgx/v5 directly</description>
  <input language="go">
package main

import (
    "github.com/jackc/pgx/v5"
)

var _ = pgx.Connect
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>routed</expected-message>
</test-case>

<test-case id="FAIL-imports-pgxpool" should-fail="true" path="api/db.go">
  <description>❌ SHOULD FAIL: Imports pgxpool</description>
  <input language="go">
package db

import (
    "github.com/jackc/pgx/v5/pgxpool"
)

var _ = pgxpool.Connect
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>routed</expected-message>
</test-case>

<test-case id="FAIL-imports-lib-pq-non-blank" should-fail="true" path="api/db.go">
  <description>❌ SHOULD FAIL: Imports github.com/lib/pq with a name (not blank)</description>
  <input language="go">
package db

import (
    pq "github.com/lib/pq"
)

var _ = pq.Driver{}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>routed</expected-message>
</test-case>

<test-case id="FAIL-imports-mattn-sqlite3" should-fail="true" path="api/db.go">
  <description>❌ SHOULD FAIL: Imports mattn/go-sqlite3 directly</description>
  <input language="go">
package db

import (
    "github.com/mattn/go-sqlite3"
)

var _ = sqlite3.Version
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>routed</expected-message>
</test-case>

<test-case id="FAIL-imports-modernc-sqlite" should-fail="true" path="api/db.go">
  <description>❌ SHOULD FAIL: Imports modernc.org/sqlite directly (use api-core)</description>
  <input language="go">
package db

import (
    sqlite "modernc.org/sqlite"
)

var _ = sqlite.Version
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>routed</expected-message>
</test-case>

<test-case id="PASS-blank-import-allowed-in-main" should-fail="false" path="api/main.go">
  <description>✅ SHOULD PASS: Blank imports of drivers are permitted (driver registration)</description>
  <input language="go">
package main

import (
    "github.com/vrooli/api-core/database"

    _ "github.com/lib/pq"
)

var _ = database.DriverPostgres
  </input>
</test-case>

<test-case id="FAIL-sql-opendb" should-fail="true" path="api/main.go">
  <description>❌ SHOULD FAIL: Calls sql.OpenDB directly (not handled by database_backoff)</description>
  <input language="go">
package main

import (
    "database/sql"
    "database/sql/driver"
)

var connector driver.Connector

func main() {
    _ = sql.OpenDB(connector)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>routed</expected-message>
</test-case>

<test-case id="PASS-test-file-exempt" should-fail="false" path="api/main_test.go">
  <description>✅ SHOULD PASS: Test files exempt</description>
  <input language="go">
package main

import (
    "testing"

    "github.com/jackc/pgx/v5"
)

func TestX(t *testing.T) { _ = pgx.Connect }
  </input>
</test-case>

*/

// routedDriverPackages lists import paths whose presence in a scenario's
// non-test, non-substrate code is a routed-database violation. Blank imports
// (used purely for driver registration) are not flagged — those are still
// the recommended pattern for wiring drivers into the api-core seam.
var routedDriverPackages = []string{
	"github.com/jackc/pgx",
	"github.com/lib/pq",
	"github.com/mattn/go-sqlite3",
	"modernc.org/sqlite",
}

// CheckRoutedDatabaseDrivers flags direct driver imports and sql.OpenDB
// calls that bypass the *database.RoutedDB seam.
func CheckRoutedDatabaseDrivers(content []byte, filePath string) []Violation {
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

	// Imports.
	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}
		path := strings.Trim(imp.Path.Value, `"`)
		if !matchesRoutedDriver(path) {
			continue
		}
		// Blank imports (e.g. _ "github.com/lib/pq") are allowed; they wire
		// the driver into the database/sql registry without exposing the
		// driver API.
		if imp.Name != nil && imp.Name.Name == "_" {
			continue
		}
		line := fset.Position(imp.Pos()).Line
		violations = append(violations, Violation{
			Type:        "routed_database_drivers",
			Severity:    "high",
			Title:       "Direct database driver import bypasses RoutedDB seam",
			Description: "This scenario imports a database driver package directly. The api-core/database package is the only substrate scenarios should depend on; it exposes *RoutedDB, the seam test-genie uses to install a runtime test pool without restarting the scenario.",
			FilePath:    filePath,
			LineNumber:  line,
			Recommendation: `Remove the named driver import and depend on api-core/database instead:

import "github.com/vrooli/api-core/database"

db, err := database.Open(ctx, database.Config{Driver: "postgres"})

If you need driver registration, use a blank import (_ "github.com/lib/pq"); blank imports are not flagged.

See: docs/agent-system/routed-test-db.md`,
			Standard: "routed-test-db-v1",
		})
	}

	// sql.OpenDB calls — database_backoff covers sql.Open; we cover
	// sql.OpenDB so the disjoint partition catches the second factory.
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !isSQLOpenDBCall(call) {
			return true
		}
		line := fset.Position(call.Pos()).Line
		violations = append(violations, Violation{
			Type:        "routed_database_drivers",
			Severity:    "high",
			Title:       "Direct sql.OpenDB bypasses RoutedDB seam",
			Description: "sql.OpenDB constructs a connection pool outside the api-core/database substrate. *RoutedDB is the seam scenarios must depend on; sql.OpenDB bypasses it and disqualifies the scenario from the routed e2e path.",
			FilePath:    filePath,
			LineNumber:  line,
			Recommendation: `Use database.Open instead:

import "github.com/vrooli/api-core/database"

db, err := database.Open(ctx, database.Config{Driver: "postgres"})

See: docs/agent-system/routed-test-db.md`,
			Standard: "routed-test-db-v1",
		})
		return true
	})

	return violations
}

func matchesRoutedDriver(importPath string) bool {
	for _, prefix := range routedDriverPackages {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return true
		}
	}
	return false
}

func isSQLOpenDBCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != "OpenDB" {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "sql"
}
