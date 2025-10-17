package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

/*
Rule: Lifecycle Protection
Description: Ensure main.go files have VROOLI_LIFECYCLE_MANAGED environment check with proper security enforcement
Reason: Prevents direct execution of services, enforcing proper process management through the Vrooli lifecycle system
Category: api
Severity: critical
Standard: vrooli-lifecycle-v1
Targets: main_go

Security Guarantees (AST-based validation):
- Verifies lifecycle check is the first substantive statement in main()
- Validates conditional logic prevents execution when VROOLI_LIFECYCLE_MANAGED is unset or not "true"
- Ensures os.Exit() is called with non-zero exit code to prevent execution
- Confirms error message contains required instructional text with correct scenario name
- Detects bypass patterns including:
  * Missing os.Exit() (execution continues)
  * Wrong conditional logic (== "false" allows unset bypass)
  * Late check placement (business logic runs first)
  * Commented-out protection code
  * Incorrect exit codes (os.Exit(0) indicates success)

<test-case id="missing-lifecycle-check" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>main.go without lifecycle protection</description>
  <input language="go">
package main

import (
    "fmt"
    "net/http"
    "os"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    fmt.Printf("Starting server on port %s\n", port)
    http.ListenAndServe(":"+port, nil)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Lifecycle Protection</expected-message>
</test-case>

<test-case id="proper-lifecycle-check" should-fail="false" path="api/main.go" scenario="demo-app">
  <description>main.go with proper lifecycle protection</description>
  <input language="go">
package main

import (
    "fmt"
    "net/http"
    "os"
)

func main() {
    // Protect against direct execution - must be run through lifecycle system
    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
        fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start demo-app

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
        os.Exit(1)
    }

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    fmt.Printf("Starting server on port %s\n", port)
    http.ListenAndServe(":"+port, nil)
}
  </input>
</test-case>

<test-case id="incorrect-lifecycle-message" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>lifecycle check present but message differs from required instructional text</description>
  <input language="go">
package main

import (
    "fmt"
    "net/http"
    "os"
)

func main() {
    // Protect against direct execution - must be run through lifecycle system
    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
        fmt.Fprintln(os.Stderr, "Error: Direct execution not allowed")
        fmt.Fprintln(os.Stderr, "Use: vrooli scenario start prompt-manager")
        os.Exit(1)
    }

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    fmt.Printf("Starting server on port %s\n", port)
    http.ListenAndServe(":"+port, nil)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Lifecycle Protection</expected-message>
</test-case>

<test-case id="lifecycle-check-with-lookupenv" should-fail="false" path="api/main.go" scenario="demo-app">
  <description>main.go using os.LookupEnv for lifecycle check</description>
  <input language="go">
package main

import (
    "fmt"
    "os"
)

func main() {
    // Check lifecycle management
    managed, exists := os.LookupEnv("VROOLI_LIFECYCLE_MANAGED")
    if !exists || managed != "true" {
        fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start demo-app

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
        os.Exit(1)
    }

    startApplication()
}
  </input>
</test-case>

<test-case id="missing-lifecycle-check-condition" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>instructional message present without enforcing lifecycle condition</description>
  <input language="go">
package main

import (
    "fmt"
    "net/http"
)

func main() {
    fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start demo-app

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)

    port := "8080"
    fmt.Printf("Starting server on port %s\n", port)
    http.ListenAndServe(":"+port, nil)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Lifecycle Protection</expected-message>
</test-case>

<test-case id="missing-os-exit" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>lifecycle check present but missing os.Exit() - CRITICAL VULNERABILITY</description>
  <input language="go">
package main

import (
    "fmt"
    "os"
)

func main() {
    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
        fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start demo-app

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
        // Missing os.Exit(1) - execution continues!
    }

    fmt.Println("Server starting anyway!")
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing os.Exit</expected-message>
</test-case>

<test-case id="wrong-conditional-logic" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>checks == false instead of != true - allows execution when unset</description>
  <input language="go">
package main

import (
    "fmt"
    "os"
)

func main() {
    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") == "false" {
        fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start demo-app

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
        os.Exit(1)
    }

    fmt.Println("Server starting (unprotected when var is unset)!")
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Incorrect conditional logic</expected-message>
</test-case>

<test-case id="late-lifecycle-check" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>lifecycle check present but business logic runs first</description>
  <input language="go">
package main

import (
    "fmt"
    "os"
)

func main() {
    fmt.Println("Starting initialization...")
    startServer()

    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
        fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start demo-app

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
        os.Exit(1)
    }
}

func startServer() {
    fmt.Println("Critical operation executed without protection!")
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Lifecycle check not first statement</expected-message>
</test-case>

<test-case id="variable-assignment-pattern" should-fail="false" path="api/main.go" scenario="demo-app">
  <description>lifecycle check using variable assignment - should be accepted</description>
  <input language="go">
package main

import (
    "fmt"
    "os"
)

func main() {
    lifecycleManaged := os.Getenv("VROOLI_LIFECYCLE_MANAGED")
    if lifecycleManaged != "true" {
        fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start demo-app

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
        os.Exit(1)
    }

    startServer()
}
  </input>
</test-case>

<test-case id="os-exit-zero" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>os.Exit(0) indicates success - should fail</description>
  <input language="go">
package main

import (
    "fmt"
    "os"
)

func main() {
    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
        fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start demo-app

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
        os.Exit(0) // Wrong - indicates success!
    }

    startServer()
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>os.Exit must use non-zero</expected-message>
</test-case>

<test-case id="deferred-exit" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>defer os.Exit(1) doesn't stop execution - CRITICAL VULNERABILITY</description>
  <input language="go">
package main

import (
    "fmt"
    "os"
)

func main() {
    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
        fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start demo-app

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
        defer os.Exit(1)  // DEFERRED - execution continues!
    }

    fmt.Println("This code still runs even without lifecycle!")
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing os.Exit</expected-message>
</test-case>

<test-case id="conditional-exit" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>os.Exit inside nested condition - might not execute</description>
  <input language="go">
package main

import (
    "fmt"
    "os"
)

func main() {
    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
        fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start demo-app

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
        // Nested condition - exit might not happen!
        if true {
            os.Exit(1)
        }
    }

    fmt.Println("Server starting anyway!")
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing os.Exit</expected-message>
</test-case>

<test-case id="message-in-comment" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>Required message in comment but wrong message in code - should fail with proper block extraction</description>
  <input language="go">
package main

import (
    "fmt"
    "os"
)

// ❌ This binary must be run through the Vrooli lifecycle system.
// 🚀 Instead, use: vrooli scenario start demo-app
// Required message is here in comment!

func main() {
    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
        fmt.Fprintln(os.Stderr, "Nope!")  // Wrong message!
        os.Exit(1)
    }

    fmt.Println("Running...")
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Lifecycle Protection</expected-message>
</test-case>

<test-case id="log-setflags-before-check" should-fail="false" path="api/main.go" scenario="demo-app">
  <description>log.SetFlags before lifecycle check - should PASS (whitelisted harmless setup)</description>
  <input language="go">
package main

import (
    "fmt"
    "log"
    "os"
)

func main() {
    log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
        fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start demo-app

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
        os.Exit(1)
    }

    log.Println("Starting...")
}
  </input>
</test-case>

<test-case id="variable-assignment-with-intervening-code" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>Variable assignment with business logic before check - should fail</description>
  <input language="go">
package main

import (
    "fmt"
    "os"
)

func main() {
    lifecycleManaged := os.Getenv("VROOLI_LIFECYCLE_MANAGED")

    // Business logic runs before the check!
    fmt.Println("Initializing...")
    doSomethingDangerous()

    if lifecycleManaged != "true" {
        fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start demo-app

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
        os.Exit(1)
    }
}

func doSomethingDangerous() {
    fmt.Println("Critical operation executed without protection!")
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Lifecycle check not first statement</expected-message>
</test-case>
*/

// CheckLifecycleProtection validates that main functions include lifecycle protection
// Uses AST-based analysis to ensure proper security enforcement
func CheckLifecycleProtection(content []byte, filePath string, scenario string) *Violation {
	contentStr := string(content)

	// Skip rule definition files - they contain test cases, not actual code
	if strings.Contains(filePath, "/rules/") {
		return nil
	}

	// Preliminary check: ensure file has func main()
	if !strings.Contains(contentStr, "func main()") {
		return nil
	}

	// Only check Go entrypoints
	if !(strings.HasSuffix(filePath, "main.go") || strings.Contains(contentStr, "package main")) {
		return nil
	}

	// Use AST-based validation for security-critical analysis
	return checkLifecycleProtectionAST(content, filePath, scenario)
}

// AST-based lifecycle protection validator
func checkLifecycleProtectionAST(content []byte, filePath string, scenario string) *Violation {
	contentStr := string(content)

	// Ensure content has package declaration for parsing
	if !strings.HasPrefix(strings.TrimSpace(contentStr), "package ") {
		contentStr = "package main\n" + contentStr
	}

	// Parse the file into an AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, []byte(contentStr), parser.ParseComments)
	if err != nil {
		// If we can't parse it, return nil (could be invalid Go code)
		return nil
	}

	// Find main() function
	var mainFunc *ast.FuncDecl
	ast.Inspect(file, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			if fn.Name.Name == "main" {
				mainFunc = fn
				return false
			}
		}
		return true
	})

	if mainFunc == nil {
		return nil // No main function, not an entrypoint
	}

	// Analyze main() function for lifecycle protection
	scenarioName := resolveScenarioName(scenario, filePath)
	result := analyzeMainFunction(mainFunc, fset, scenarioName, filePath, contentStr)

	return result
}

// lifecycleCheckInfo tracks the lifecycle protection implementation
type lifecycleCheckInfo struct {
	found              bool
	position           int // statement index in main()
	hasCorrectLogic    bool
	hasOsExit          bool
	exitCodeNonZero    bool
	hasErrorMessage    bool
	varName            string // if using variable assignment
	lineNumber         int
	missingComponent   string // what's wrong
}

// analyzeMainFunction performs deep analysis of the main() function
func analyzeMainFunction(mainFunc *ast.FuncDecl, fset *token.FileSet, scenarioName, filePath, content string) *Violation {
	if mainFunc.Body == nil || len(mainFunc.Body.List) == 0 {
		return createViolation("Missing Lifecycle Protection", "Empty main() function", fset.Position(mainFunc.Pos()).Line, scenarioName, filePath, content)
	}

	info := &lifecycleCheckInfo{}

	// Track variable assignments for lifecycle check
	varAssignments := make(map[string]bool) // varName -> is lifecycle var
	firstBusinessLogicPos := -1              // Track position of first non-lifecycle statement

	// Analyze each statement in main()
	for i, stmt := range mainFunc.Body.List {
		// Check if this is a lifecycle-related assignment
		if assignStmt, ok := stmt.(*ast.AssignStmt); ok {
			if isLifecycleAssignment(assignStmt) {
				if len(assignStmt.Lhs) > 0 {
					if ident, ok := assignStmt.Lhs[0].(*ast.Ident); ok {
						varAssignments[ident.Name] = true
						info.varName = ident.Name
					}
				}
				// This is lifecycle-related, continue
				continue
			}
		}

		// Check if this is the lifecycle protection if-statement
		if ifStmt, ok := stmt.(*ast.IfStmt); ok {
			if isLifecycleIfStmt(ifStmt, varAssignments) {
				info.found = true
				info.position = i
				info.lineNumber = fset.Position(ifStmt.Pos()).Line

				// Validate the condition
				info.hasCorrectLogic = hasCorrectConditionalLogic(ifStmt.Cond, varAssignments)

				// Validate the error block
				info.hasErrorMessage = hasLifecycleErrorMessage(ifStmt.Body, scenarioName, content, fset)
				info.hasOsExit, info.exitCodeNonZero = hasOsExitCall(ifStmt.Body, fset)

				break
			}
		}

		// Track if we see business logic (non-lifecycle code) before finding the check
		if !info.found && !isLifecycleRelated(stmt, varAssignments) && firstBusinessLogicPos == -1 {
			firstBusinessLogicPos = i
		}
	}

	// Validate findings and generate appropriate violation
	if !info.found {
		return createViolation("Missing Lifecycle Protection", "No lifecycle protection check found in main()", fset.Position(mainFunc.Pos()).Line, scenarioName, filePath, content)
	}

	// Check if lifecycle check came after business logic
	if firstBusinessLogicPos != -1 && firstBusinessLogicPos < info.position {
		return createViolation(
			"Lifecycle check not first statement",
			"Business logic executes before lifecycle protection check",
			info.lineNumber,
			scenarioName,
			filePath,
			content,
		)
	}

	if !info.hasCorrectLogic {
		return createViolation(
			"Incorrect conditional logic",
			"Lifecycle check uses wrong conditional logic (must check != \"true\" or equivalent to block when unset)",
			info.lineNumber,
			scenarioName,
			filePath,
			content,
		)
	}

	if !info.hasErrorMessage {
		return createViolation(
			"Missing Lifecycle Protection",
			"Error message missing or incorrect - must mention Vrooli lifecycle system and correct scenario start command",
			info.lineNumber,
			scenarioName,
			filePath,
			content,
		)
	}

	if !info.hasOsExit {
		return createViolation(
			"Missing os.Exit in lifecycle check",
			"Lifecycle check must call os.Exit() to prevent execution",
			info.lineNumber,
			scenarioName,
			filePath,
			content,
		)
	}

	if !info.exitCodeNonZero {
		return createViolation(
			"os.Exit must use non-zero exit code",
			"os.Exit(0) indicates success - must use non-zero code (e.g., os.Exit(1))",
			info.lineNumber,
			scenarioName,
			filePath,
			content,
		)
	}

	// All checks passed
	return nil
}

func resolveScenarioName(provided string, filePath string) string {
	scenario := strings.TrimSpace(provided)
	if scenario != "" {
		return scenario
	}

	// Normalize path separators
	clean := strings.ReplaceAll(filePath, "\\", "/")
	parts := strings.Split(clean, "/")

	// Strategy 1: Look for "scenarios" directory in path
	for i := 0; i < len(parts); i++ {
		if parts[i] == "scenarios" && i+1 < len(parts) {
			candidate := parts[i+1]
			// Skip common non-scenario directories
			if candidate != "api" && candidate != "cli" && candidate != "ui" {
				return candidate
			}
		}
	}

	// Strategy 2: Look for common project patterns (api, cli, ui subdirectories)
	// If file is in /foo/bar/api/main.go, scenario is likely "bar"
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "api" || parts[i] == "cli" || parts[i] == "ui" {
			if i > 0 {
				candidate := parts[i-1]
				// Skip "scenarios" directory itself
				if candidate != "scenarios" && candidate != "." && candidate != "" {
					return candidate
				}
			}
		}
	}

	// Strategy 3: Use parent directory of file
	parentDir := filepath.Dir(filePath)
	base := filepath.Base(parentDir)

	// If parent is "api", "cli", or "ui", go one level up
	if base == "api" || base == "cli" || base == "ui" {
		grandparentDir := filepath.Dir(parentDir)
		base = filepath.Base(grandparentDir)
	}

	if base != "." && base != "" && base != "scenarios" {
		return base
	}

	// Strategy 4: Try to extract from absolute path
	absPath, err := filepath.Abs(filePath)
	if err == nil {
		absClean := strings.ReplaceAll(absPath, "\\", "/")
		absParts := strings.Split(absClean, "/")
		for i := 0; i < len(absParts); i++ {
			if absParts[i] == "scenarios" && i+1 < len(absParts) {
				return absParts[i+1]
			}
		}
	}

	// Last resort: use generic placeholder
	// This will at least make it obvious something went wrong
	return "SCENARIO_NAME_HERE"
}

// Helper functions
func findMainFunctionLine(content string) int {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.Contains(line, "func main()") {
			return i + 1
		}
	}
	return 1
}

func extractCodeSnippet(content, pattern string) string {
	idx := strings.Index(content, pattern)
	if idx == -1 {
		return ""
	}

	// Extract a few lines around the pattern
	start := idx - 50
	if start < 0 {
		start = 0
	}
	end := idx + 150
	if end > len(content) {
		end = len(content)
	}

	return content[start:end]
}

// AST helper functions

// isLifecycleAssignment checks if an assignment statement is assigning VROOLI_LIFECYCLE_MANAGED
func isLifecycleAssignment(assign *ast.AssignStmt) bool {
	if len(assign.Rhs) == 0 {
		return false
	}

	// Check for os.Getenv("VROOLI_LIFECYCLE_MANAGED") or os.LookupEnv("VROOLI_LIFECYCLE_MANAGED")
	if call, ok := assign.Rhs[0].(*ast.CallExpr); ok {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "os" {
				if sel.Sel.Name == "Getenv" || sel.Sel.Name == "LookupEnv" {
					// Check if argument is "VROOLI_LIFECYCLE_MANAGED"
					if len(call.Args) > 0 {
						if lit, ok := call.Args[0].(*ast.BasicLit); ok {
							return lit.Value == `"VROOLI_LIFECYCLE_MANAGED"`
						}
					}
				}
			}
		}
	}

	return false
}

// isLifecycleIfStmt checks if an if-statement is checking VROOLI_LIFECYCLE_MANAGED
func isLifecycleIfStmt(ifStmt *ast.IfStmt, varAssignments map[string]bool) bool {
	return containsLifecycleCheck(ifStmt.Cond, varAssignments)
}

// containsLifecycleCheck recursively checks if an expression contains a lifecycle check
func containsLifecycleCheck(expr ast.Expr, varAssignments map[string]bool) bool {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		// Check both sides of binary expression
		return containsLifecycleCheck(e.X, varAssignments) || containsLifecycleCheck(e.Y, varAssignments)

	case *ast.UnaryExpr:
		// Check unary expression (e.g., !exists)
		return containsLifecycleCheck(e.X, varAssignments)

	case *ast.CallExpr:
		// Direct call: os.Getenv("VROOLI_LIFECYCLE_MANAGED") or os.LookupEnv(...)
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "os" {
				if sel.Sel.Name == "Getenv" || sel.Sel.Name == "LookupEnv" {
					if len(e.Args) > 0 {
						if lit, ok := e.Args[0].(*ast.BasicLit); ok {
							return lit.Value == `"VROOLI_LIFECYCLE_MANAGED"`
						}
					}
				}
			}
		}

	case *ast.Ident:
		// Variable reference - check if it's a lifecycle variable
		return varAssignments[e.Name]
	}

	return false
}

// hasCorrectConditionalLogic validates that the condition properly blocks execution when unset
func hasCorrectConditionalLogic(cond ast.Expr, varAssignments map[string]bool) bool {
	switch e := cond.(type) {
	case *ast.BinaryExpr:
		// Check for != "true" pattern
		if e.Op == token.NEQ {
			// Left side should be Getenv/LookupEnv or lifecycle variable
			// Right side should be "true"
			if isLifecycleExpression(e.X, varAssignments) {
				if lit, ok := e.Y.(*ast.BasicLit); ok {
					return lit.Value == `"true"`
				}
			}
		}

		// Check for == "" pattern (also valid - blocks when unset)
		if e.Op == token.EQL {
			if isLifecycleExpression(e.X, varAssignments) {
				if lit, ok := e.Y.(*ast.BasicLit); ok {
					return lit.Value == `""`
				}
			}
		}

		// Check for || with !exists pattern (for LookupEnv)
		if e.Op == token.LOR {
			// Either side could have the correct check
			return hasCorrectConditionalLogic(e.X, varAssignments) || hasCorrectConditionalLogic(e.Y, varAssignments)
		}

	case *ast.UnaryExpr:
		// Check for !exists pattern
		if e.Op == token.NOT {
			if ident, ok := e.X.(*ast.Ident); ok {
				// This is !exists or !ok pattern from LookupEnv
				return varAssignments[ident.Name] || ident.Name == "exists" || ident.Name == "ok"
			}
		}
	}

	return false
}

// isLifecycleExpression checks if an expression references the lifecycle variable
func isLifecycleExpression(expr ast.Expr, varAssignments map[string]bool) bool {
	switch e := expr.(type) {
	case *ast.CallExpr:
		// Direct os.Getenv/LookupEnv call
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "os" {
				if sel.Sel.Name == "Getenv" || sel.Sel.Name == "LookupEnv" {
					if len(e.Args) > 0 {
						if lit, ok := e.Args[0].(*ast.BasicLit); ok {
							return lit.Value == `"VROOLI_LIFECYCLE_MANAGED"`
						}
					}
				}
			}
		}

	case *ast.Ident:
		// Variable reference
		return varAssignments[e.Name]
	}

	return false
}

// hasOsExitCall checks if the block contains os.Exit() with non-zero code
// Also detects if exit is deferred or conditional (both are security issues)
func hasOsExitCall(block *ast.BlockStmt, fset *token.FileSet) (hasExit bool, nonZeroCode bool) {
	var isDeferred bool
	var isConditional bool

	// Track the depth of if-statements to detect conditional exits
	ifDepth := 0

	ast.Inspect(block, func(n ast.Node) bool {
		// Track when we enter/exit if-statements
		if _, ok := n.(*ast.IfStmt); ok {
			ifDepth++
		}

		// Check for defer statements
		if deferStmt, ok := n.(*ast.DeferStmt); ok {
			// deferStmt.Call is already a *ast.CallExpr
			call := deferStmt.Call
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "os" {
					if sel.Sel.Name == "Exit" {
						hasExit = true
						isDeferred = true
						return false // Found it, stop searching
					}
				}
			}
		}

		// Check for os.Exit() call
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "os" {
					if sel.Sel.Name == "Exit" {
						hasExit = true

						// Check if exit is inside a nested if-statement (conditional)
						// We're already in the lifecycle protection if-block (depth 0)
						// If ifDepth > 0, we're in a nested if
						if ifDepth > 0 {
							isConditional = true
						}

						// Check exit code
						if len(call.Args) > 0 {
							if lit, ok := call.Args[0].(*ast.BasicLit); ok {
								// Check if it's a non-zero literal
								nonZeroCode = lit.Value != "0"
							} else {
								// If it's not a literal (e.g., constant or expression), assume non-zero
								nonZeroCode = true
							}
						}

						return false // Found it, stop searching
					}
				}
			}
		}
		return true
	})

	// If exit is deferred or conditional, treat as if it doesn't exist
	// (because it won't reliably prevent execution)
	if isDeferred || isConditional {
		hasExit = false
		nonZeroCode = false
	}

	return hasExit, nonZeroCode
}

// hasLifecycleErrorMessage checks if the block contains the required error message
func hasLifecycleErrorMessage(block *ast.BlockStmt, scenarioName, content string, fset *token.FileSet) bool {
	// Check for key phrases in the content
	// We require: lifecycle system mention + correct command with scenario name
	requiredPhrases := []string{
		"Vrooli lifecycle system",
		"vrooli scenario start",
		scenarioName, // Must mention the correct scenario name
	}

	blockText := getBlockText(block, content, fset)

	for _, phrase := range requiredPhrases {
		if !strings.Contains(blockText, phrase) {
			return false
		}
	}

	return true
}

// getBlockText extracts the text content of a block statement using AST positions
func getBlockText(block *ast.BlockStmt, fullContent string, fset *token.FileSet) string {
	if block == nil || fset == nil {
		return ""
	}

	// Use fset to convert token positions to byte offsets
	startPos := fset.Position(block.Pos())
	endPos := fset.Position(block.End())

	// For safety, scan through the content to find byte offsets
	// (since Position gives us line:column, not byte offset)
	lines := strings.Split(fullContent, "\n")

	// Calculate byte offset for start position
	startOffset := 0
	for i := 1; i < startPos.Line && i <= len(lines); i++ {
		startOffset += len(lines[i-1]) + 1 // +1 for newline
	}
	startOffset += startPos.Column - 1

	// Calculate byte offset for end position
	endOffset := 0
	for i := 1; i < endPos.Line && i <= len(lines); i++ {
		endOffset += len(lines[i-1]) + 1 // +1 for newline
	}
	endOffset += endPos.Column - 1

	// Ensure offsets are within bounds
	if startOffset < 0 || endOffset > len(fullContent) || startOffset >= endOffset {
		// Fallback: search for the required phrases in full content
		// This maintains backward compatibility but is less precise
		return fullContent
	}

	// Extract just the block content
	return fullContent[startOffset:endOffset]
}

// isLifecycleRelated checks if a statement is related to lifecycle checking
func isLifecycleRelated(stmt ast.Stmt, varAssignments map[string]bool) bool {
	// Check if it's an assignment of lifecycle variable
	if assignStmt, ok := stmt.(*ast.AssignStmt); ok {
		return isLifecycleAssignment(assignStmt)
	}

	// Check if it's the lifecycle if statement
	if ifStmt, ok := stmt.(*ast.IfStmt); ok {
		return isLifecycleIfStmt(ifStmt, varAssignments)
	}

	// Check if it's a whitelisted harmless setup call
	if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
		return isHarmlessSetup(exprStmt)
	}

	return false
}

// isHarmlessSetup checks if a statement is a harmless setup call that's safe before lifecycle check
func isHarmlessSetup(exprStmt *ast.ExprStmt) bool {
	call, ok := exprStmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Whitelist of harmless setup patterns
	// log.SetFlags(), log.SetOutput(), log.SetPrefix()
	if ident, ok := sel.X.(*ast.Ident); ok {
		if ident.Name == "log" {
			switch sel.Sel.Name {
			case "SetFlags", "SetOutput", "SetPrefix":
				return true
			}
		}
	}

	return false
}

// createViolation creates a standardized violation for lifecycle protection issues
func createViolation(title, description string, lineNumber int, scenarioName, filePath, content string) *Violation {
	recommendation := "Add proper lifecycle check at the start of main():\n\n" +
		"if os.Getenv(\"VROOLI_LIFECYCLE_MANAGED\") != \"true\" {\n" +
		"    fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.\n\n" +
		"🚀 Instead, use:\n" +
		"   vrooli scenario start " + scenarioName + "\n\n" +
		"💡 The lifecycle system provides environment variables, port allocation,\n" +
		"   and dependency management automatically. Direct execution is not supported.\n" +
		"`)\n" +
		"    os.Exit(1)\n" +
		"}\n"

	return &Violation{
		Type:           "lifecycle_protection",
		Severity:       "critical",
		Title:          title,
		Description:    description,
		FilePath:       filePath,
		LineNumber:     lineNumber,
		CodeSnippet:    extractCodeSnippet(content, "func main()"),
		Recommendation: recommendation,
		Standard:       "vrooli-lifecycle-v1",
	}
}
