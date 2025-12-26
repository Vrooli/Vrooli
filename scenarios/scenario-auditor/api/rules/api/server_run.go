package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"html"
	"path/filepath"
	"strconv"
	"strings"
)

/*
Rule: Use api-core Server Package
Description: HTTP servers must use the api-core/server package for graceful shutdown and signal handling
Reason: Ensures consistent graceful shutdown behavior with signal handling across all scenarios
Category: api
Severity: high
Standard: service-reliability-v1
Targets: main_go

<test-case id="PASS-uses-api-core-server" should-fail="false" path="api/main.go">
  <description>✅ SHOULD PASS: Uses api-core/server.Run() for HTTP server</description>
  <input language="go">
package main

import (
    "context"
    "log"

    "github.com/gorilla/mux"
    "github.com/vrooli/api-core/server"
)

func main() {
    router := mux.NewRouter()
    router.HandleFunc("/health", healthHandler)

    if err := server.Run(server.Config{
        Handler: router,
    }); err != nil {
        log.Fatal(err)
    }
}
  </input>
</test-case>

<test-case id="PASS-uses-api-core-server-with-cleanup" should-fail="false" path="api/main.go">
  <description>✅ SHOULD PASS: Uses api-core/server.Run() with cleanup callback</description>
  <input language="go">
package main

import (
    "context"
    "log"

    "github.com/vrooli/api-core/server"
)

func main() {
    db := connectDB()

    if err := server.Run(server.Config{
        Handler: router,
        Cleanup: func(ctx context.Context) error {
            return db.Close()
        },
    }); err != nil {
        log.Fatal(err)
    }
}
  </input>
</test-case>

<test-case id="FAIL-http-listen-and-serve" should-fail="true" path="api/main.go">
  <description>❌ SHOULD FAIL: Uses http.ListenAndServe() directly</description>
  <input language="go">
package main

import (
    "log"
    "net/http"
)

func main() {
    http.HandleFunc("/health", healthHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>api-core/server</expected-message>
</test-case>

<test-case id="FAIL-http-server-listen-and-serve" should-fail="true" path="api/main.go">
  <description>❌ SHOULD FAIL: Uses srv.ListenAndServe() directly</description>
  <input language="go">
package main

import (
    "log"
    "net/http"
)

func main() {
    srv := &amp;http.Server{
        Addr:    ":8080",
        Handler: nil,
    }
    log.Fatal(srv.ListenAndServe())
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>api-core/server</expected-message>
</test-case>

<test-case id="FAIL-http-listen-and-serve-tls" should-fail="true" path="api/main.go">
  <description>❌ SHOULD FAIL: Uses http.ListenAndServeTLS() directly</description>
  <input language="go">
package main

import (
    "log"
    "net/http"
)

func main() {
    http.HandleFunc("/health", healthHandler)
    log.Fatal(http.ListenAndServeTLS(":8443", "cert.pem", "key.pem", nil))
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>api-core/server</expected-message>
</test-case>

<test-case id="PASS-test-file" should-fail="false" path="main_test.go">
  <description>✅ SHOULD PASS: Test files can use http.ListenAndServe for test servers</description>
  <input language="go">
package main

import (
    "net/http"
    "testing"
)

func TestServerStart(t *testing.T) {
    go http.ListenAndServe(":0", nil)
}
  </input>
</test-case>

<test-case id="PASS-cli-main" should-fail="false" path="cli/main.go">
  <description>✅ SHOULD PASS: CLI main.go files are exempt (not long-running servers)</description>
  <input language="go">
package main

import (
    "fmt"
    "net/http"
)

func main() {
    // CLI that makes HTTP requests, not a server
    resp, _ := http.Get("http://localhost:8080")
    fmt.Println(resp.Status)
}
  </input>
</test-case>

<test-case id="PASS-no-server" should-fail="false" path="api/main.go">
  <description>✅ SHOULD PASS: File without HTTP server code</description>
  <input language="go">
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
  </input>
</test-case>

*/

// CheckServerRun verifies that HTTP servers use the api-core/server package.
// Direct http.ListenAndServe() or srv.ListenAndServe() calls are flagged as violations.
func CheckServerRun(content []byte, filePath string) []Violation {
	// Skip exempt paths (tests, CLI, non-main files)
	if isServerRunExemptPath(filePath) {
		return nil
	}

	// Only check main.go files in api directories
	if !isAPIMainFile(filePath) {
		return nil
	}

	fset := token.NewFileSet()
	source := html.UnescapeString(string(content))
	file, err := parser.ParseFile(fset, filePath, source, 0)
	if err != nil {
		return nil
	}

	// Check if file imports api-core/server - if so, it's compliant
	if hasAPICoreServerImport(file) {
		return nil
	}

	// Check if file imports net/http - if not, nothing to check
	if !hasNetHTTPImport(file) {
		return nil
	}

	var violations []Violation

	// Look for http.ListenAndServe(), http.ListenAndServeTLS(), and srv.ListenAndServe() calls
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if isListenAndServeCall(call) {
			line := fset.Position(call.Pos()).Line
			violations = append(violations, Violation{
				Type:        "server_run",
				Severity:    "high",
				Title:       "Direct ListenAndServe() Without api-core",
				Description: "HTTP servers must use the api-core/server package instead of calling ListenAndServe() directly. The api-core package provides graceful shutdown with signal handling (SIGINT/SIGTERM), configurable timeouts, and cleanup callbacks.",
				FilePath:    filePath,
				LineNumber:  line,
				Recommendation: `Replace ListenAndServe() with server.Run():

import "github.com/vrooli/api-core/server"

if err := server.Run(server.Config{
    Handler: router,
    Cleanup: func(ctx context.Context) error {
        return db.Close()
    },
}); err != nil {
    log.Fatal(err)
}

This automatically:
- Reads API_PORT from environment (defaults to 8080)
- Handles SIGINT/SIGTERM signals gracefully
- Waits for in-flight requests to complete (10s default)
- Runs cleanup callbacks before exit
- Configures sensible read/write timeouts

See: packages/api-core/README.md`,
				Standard: "service-reliability-v1",
			})
		}

		return true
	})

	return deduplicateServerViolations(violations)
}

// isServerRunExemptPath returns true for paths that are exempt from this rule.
func isServerRunExemptPath(path string) bool {
	lowerPath := strings.ToLower(path)

	// Test files can use ListenAndServe directly
	if strings.HasSuffix(lowerPath, "_test.go") {
		return true
	}

	// Test helper files
	base := filepath.Base(lowerPath)
	if strings.HasPrefix(base, "test_") {
		return true
	}

	// CLI directories are not long-running servers
	normalized := filepath.ToSlash(lowerPath)
	if strings.HasPrefix(normalized, "cli/") || strings.Contains(normalized, "/cli/") {
		return true
	}

	// Check for other exempt directories
	exemptDirs := []string{
		"test",
		"tools",
		"scripts",
		"examples",
	}

	for _, dir := range exemptDirs {
		if strings.Contains(lowerPath, "/"+dir+"/") {
			return true
		}
		if strings.HasPrefix(lowerPath, dir+"/") {
			return true
		}
	}

	return false
}

// isAPIMainFile checks if the file is a main.go in an api directory
func isAPIMainFile(path string) bool {
	normalized := filepath.ToSlash(strings.ToLower(path))

	// Must be main.go
	if !strings.HasSuffix(normalized, "main.go") {
		return false
	}

	// Must be in api directory
	return strings.HasPrefix(normalized, "api/") || strings.Contains(normalized, "/api/")
}

// hasAPICoreServerImport checks if the file imports github.com/vrooli/api-core/server
func hasAPICoreServerImport(file *ast.File) bool {
	for _, imp := range file.Imports {
		if imp.Path != nil {
			path := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(path, "api-core/server") {
				return true
			}
		}
	}
	return false
}

// hasNetHTTPImport checks if the file imports net/http
func hasNetHTTPImport(file *ast.File) bool {
	for _, imp := range file.Imports {
		if imp.Path != nil {
			path := strings.Trim(imp.Path.Value, `"`)
			if path == "net/http" {
				return true
			}
		}
	}
	return false
}

// isListenAndServeCall checks if a call expression is http.ListenAndServe(),
// http.ListenAndServeTLS(), or srv.ListenAndServe()
func isListenAndServeCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	methodName := sel.Sel.Name

	// Check for ListenAndServe or ListenAndServeTLS
	if methodName != "ListenAndServe" && methodName != "ListenAndServeTLS" {
		return false
	}

	// Check if it's http.ListenAndServe (package-level call)
	if ident, ok := sel.X.(*ast.Ident); ok {
		if ident.Name == "http" {
			return true
		}
	}

	// Any .ListenAndServe() call is likely a server - flag it
	// This catches srv.ListenAndServe(), s.server.ListenAndServe(), etc.
	return true
}

func deduplicateServerViolations(items []Violation) []Violation {
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
