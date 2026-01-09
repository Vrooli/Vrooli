package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"html"
	"strconv"
	"strings"
)

/*
Rule: Use api-core Database Package
Description: Database connections must use the api-core/database package for automatic retry with backoff
Reason: Ensures consistent retry behavior with exponential backoff and jitter across all scenarios
Category: api
Severity: high
Standard: service-reliability-v1
Targets: api

<test-case id="PASS-uses-api-core-database" should-fail="false">
  <description>✅ SHOULD PASS: Uses api-core/database.Connect() for database connection</description>
  <input language="go">
package main

import (
    "context"
    "log"

    "github.com/vrooli/api-core/database"
    _ "github.com/lib/pq"
)

func main() {
    db, err := database.Connect(context.Background(), database.Config{
        Driver: "postgres",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
}
  </input>
</test-case>

<test-case id="PASS-uses-api-core-database-with-options" should-fail="false">
  <description>✅ SHOULD PASS: Uses api-core/database.Connect() with custom options</description>
  <input language="go">
package main

import (
    "context"
    "log"
    "time"

    "github.com/vrooli/api-core/database"
    "github.com/vrooli/api-core/retry"
    _ "github.com/lib/pq"
)

func main() {
    db, err := database.Connect(context.Background(), database.Config{
        Driver:          "postgres",
        MaxOpenConns:    50,
        MaxIdleConns:    10,
        ConnMaxLifetime: 10 * time.Minute,
        Retry: &amp;retry.Config{
            MaxAttempts: 5,
            BaseDelay:   time.Second,
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
}
  </input>
</test-case>

<test-case id="FAIL-direct-sql-open" should-fail="true" path="api/main.go">
  <description>❌ SHOULD FAIL: Uses sql.Open() directly without api-core</description>
  <input language="go">
package main

import (
    "database/sql"
    "log"

    _ "github.com/lib/pq"
)

func main() {
    db, err := sql.Open("postgres", "postgres://user:pass@host:5432/db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>api-core/database</expected-message>
</test-case>

<test-case id="FAIL-manual-backoff-loop" should-fail="true" path="api/main.go">
  <description>❌ SHOULD FAIL: Manual backoff implementation instead of api-core</description>
  <input language="go">
package main

import (
    "database/sql"
    "fmt"
    "math"
    "math/rand"
    "time"
)

var db *sql.DB

func connectWithBackoff() error {
    db, _ = sql.Open("postgres", "dsn")

    maxRetries := 5
    baseDelay := 200 * time.Millisecond
    maxDelay := 5 * time.Second

    for attempt := 0; attempt &lt; maxRetries; attempt++ {
        if err := db.Ping(); err == nil {
            return nil
        }

        delay := time.Duration(math.Min(float64(baseDelay)*math.Pow(2, float64(attempt)), float64(maxDelay)))
        jitterRange := float64(delay) * 0.25
        jitter := time.Duration(jitterRange * rand.Float64())
        actualDelay := delay + jitter
        time.Sleep(actualDelay)
    }

    return fmt.Errorf("database unavailable after retries")
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>api-core/database</expected-message>
</test-case>

<test-case id="FAIL-sql-open-with-ping-no-apicore" should-fail="true" path="api/main.go">
  <description>❌ SHOULD FAIL: sql.Open with Ping but no api-core import</description>
  <input language="go">
package main

import (
    "database/sql"
    "log"

    _ "github.com/lib/pq"
)

func main() {
    db, err := sql.Open("postgres", "dsn")
    if err != nil {
        log.Fatal(err)
    }
    if err := db.Ping(); err != nil {
        log.Fatal(err)
    }
    defer db.Close()
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>api-core/database</expected-message>
</test-case>

<test-case id="PASS-test-file-with-sql-open" should-fail="false" path="main_test.go">
  <description>✅ SHOULD PASS: Test files can use sql.Open directly for mocking</description>
  <input language="go">
package main

import (
    "database/sql"
    "testing"

    _ "github.com/mattn/go-sqlite3"
)

func TestDatabaseLogic(t *testing.T) {
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()
}
  </input>
</test-case>

<test-case id="PASS-migration-script" should-fail="false" path="migrations/001_initial.go">
  <description>✅ SHOULD PASS: Migration scripts are exempt (single-instance execution)</description>
  <input language="go">
package migrations

import (
    "database/sql"

    _ "github.com/lib/pq"
)

func Run(db *sql.DB) error {
    _, err := db.Exec("CREATE TABLE users (id SERIAL PRIMARY KEY)")
    return err
}
  </input>
</test-case>

<test-case id="PASS-initialization-script" should-fail="false" path="initialization/setup.go">
  <description>✅ SHOULD PASS: Initialization scripts are exempt</description>
  <input language="go">
package initialization

import (
    "database/sql"

    _ "github.com/lib/pq"
)

func Setup() (*sql.DB, error) {
    return sql.Open("postgres", "dsn")
}
  </input>
</test-case>

<test-case id="PASS-test-helpers-file" should-fail="false" path="api/test_helpers.go">
  <description>✅ SHOULD PASS: Test helper files are exempt</description>
  <input language="go">
package main

import (
    "database/sql"

    _ "github.com/lib/pq"
)

func SetupTestDB() (*sql.DB, error) {
    return sql.Open("postgres", "postgres://test:test@localhost/test")
}
  </input>
</test-case>

*/

// CheckDatabaseBackoff verifies that database connections use the api-core/database package.
// Manual sql.Open() calls without api-core are flagged as violations.
func CheckDatabaseBackoff(content []byte, filePath string) []Violation {
	// Skip exempt paths (tests, migrations, initialization scripts)
	if isExemptPath(filePath) {
		return nil
	}

	fset := token.NewFileSet()
	source := html.UnescapeString(string(content))
	file, err := parser.ParseFile(fset, filePath, source, 0)
	if err != nil {
		return nil
	}

	// Check if file imports api-core/database - if so, it's compliant
	if hasAPICoreImport(file) {
		return nil
	}

	// Check if file imports database/sql - if not, nothing to check
	if !hasDatabaseSQLImport(file) {
		return nil
	}

	var violations []Violation

	// Look for sql.Open() calls
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if isSQLOpenCall(call) {
			line := fset.Position(call.Pos()).Line
			violations = append(violations, Violation{
				Type:        "database_backoff",
				Severity:    "high",
				Title:       "Direct sql.Open() Without api-core",
				Description: "Database connections must use the api-core/database package instead of calling sql.Open() directly. The api-core package provides automatic retry with exponential backoff and jitter to prevent thundering herd issues.",
				FilePath:    filePath,
				LineNumber:  line,
				Recommendation: `Replace sql.Open() with database.Connect():

import "github.com/vrooli/api-core/database"

db, err := database.Connect(ctx, database.Config{
    Driver: "postgres",
})

This automatically:
- Reads POSTGRES_* environment variables from lifecycle
- Retries with exponential backoff (10 attempts, 500ms base)
- Adds random jitter to prevent thundering herd
- Configures connection pooling

See: packages/api-core/README.md`,
				Standard: "service-reliability-v1",
			})
		}

		return true
	})

	return deduplicateBackoffViolations(violations)
}

// isExemptPath returns true for paths that are exempt from this rule.
// These include test files, migrations, and initialization scripts.
func isExemptPath(path string) bool {
	lowerPath := strings.ToLower(path)

	// Test files can use sql.Open() directly for mocking/test databases
	if strings.HasSuffix(lowerPath, "_test.go") {
		return true
	}

	// Test helper files (test_helpers.go, test_utils.go, etc.)
	base := lowerPath
	if idx := strings.LastIndex(lowerPath, "/"); idx >= 0 {
		base = lowerPath[idx+1:]
	}
	if strings.HasPrefix(base, "test_") {
		return true
	}

	// Check for exempt directories (handles both absolute and relative paths)
	exemptDirs := []string{
		"test",
		"migrate",
		"migration",
		"migrations",
		"initialization",
		"init",
		"scripts",
		"tools",
	}

	for _, dir := range exemptDirs {
		// Match /dir/ (middle of path)
		if strings.Contains(lowerPath, "/"+dir+"/") {
			return true
		}
		// Match dir/ at start of relative path
		if strings.HasPrefix(lowerPath, dir+"/") {
			return true
		}
	}

	return false
}

// hasAPICoreImport checks if the file imports github.com/vrooli/api-core/database
func hasAPICoreImport(file *ast.File) bool {
	for _, imp := range file.Imports {
		if imp.Path != nil {
			path := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(path, "api-core/database") {
				return true
			}
		}
	}
	return false
}

// hasDatabaseSQLImport checks if the file imports database/sql
func hasDatabaseSQLImport(file *ast.File) bool {
	for _, imp := range file.Imports {
		if imp.Path != nil {
			path := strings.Trim(imp.Path.Value, `"`)
			if path == "database/sql" {
				return true
			}
		}
	}
	return false
}

// isSQLOpenCall checks if a call expression is sql.Open()
func isSQLOpenCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Check for sql.Open
	if sel.Sel.Name != "Open" {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Name == "sql"
}

func deduplicateBackoffViolations(items []Violation) []Violation {
	if len(items) == 0 {
		return items
	}
	seen := make(map[string]bool)
	result := make([]Violation, 0, len(items))
	for _, v := range items {
		key := v.Title + "|" + v.FilePath + "|" + strconv.Itoa(v.LineNumber)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, v)
	}
	return result
}
