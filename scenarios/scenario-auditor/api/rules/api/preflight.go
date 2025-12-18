package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

/*
Rule: Preflight Checks
Description: Ensure main.go files use api-core preflight package for staleness detection and lifecycle protection
Reason: Provides automatic binary staleness detection, rebuild, and lifecycle management verification
Category: api
Severity: critical
Standard: vrooli-preflight-v1
Targets: main_go

Security Guarantees (AST-based validation):
- Verifies preflight.Run() or preflight.MustRun() is called at the start of main()
- Validates preflight.Run() is wrapped in if-statement with return
- Ensures ScenarioName is provided in the Config
- Detects bypass patterns including:
  * Missing preflight import
  * Preflight call after business logic
  * Missing return after preflight.Run()
  * Empty ScenarioName

<test-case id="missing-preflight" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>main.go without preflight checks</description>
  <input language="go">
package main

import (
    "fmt"
    "net/http"
)

func main() {
    fmt.Println("Starting server...")
    http.ListenAndServe(":8080", nil)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Preflight Check</expected-message>
</test-case>

<test-case id="proper-preflight-run" should-fail="false" path="api/main.go" scenario="demo-app">
  <description>main.go with proper preflight.Run() pattern</description>
  <input language="go">
package main

import (
    "fmt"
    "net/http"

    "github.com/vrooli/api-core/preflight"
)

func main() {
    if preflight.Run(preflight.Config{
        ScenarioName: "demo-app",
    }) {
        return
    }

    fmt.Println("Starting server...")
    http.ListenAndServe(":8080", nil)
}
  </input>
</test-case>

<test-case id="proper-preflight-mustrun" should-fail="false" path="api/main.go" scenario="demo-app">
  <description>main.go with preflight.MustRun() pattern</description>
  <input language="go">
package main

import (
    "github.com/vrooli/api-core/preflight"
)

func main() {
    preflight.MustRun(preflight.Config{
        ScenarioName: "demo-app",
    })

    startServer()
}
  </input>
</test-case>

<test-case id="missing-return-after-run" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>preflight.Run() without return statement</description>
  <input language="go">
package main

import (
    "github.com/vrooli/api-core/preflight"
)

func main() {
    if preflight.Run(preflight.Config{
        ScenarioName: "demo-app",
    }) {
        // Missing return!
    }

    startServer()
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing return after preflight.Run</expected-message>
</test-case>

<test-case id="preflight-not-first" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>preflight call after business logic</description>
  <input language="go">
package main

import (
    "fmt"

    "github.com/vrooli/api-core/preflight"
)

func main() {
    fmt.Println("Initializing...")
    doSomething()

    if preflight.Run(preflight.Config{
        ScenarioName: "demo-app",
    }) {
        return
    }

    startServer()
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Preflight check not first statement</expected-message>
</test-case>

<test-case id="missing-scenario-name" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>preflight without ScenarioName</description>
  <input language="go">
package main

import (
    "github.com/vrooli/api-core/preflight"
)

func main() {
    if preflight.Run(preflight.Config{}) {
        return
    }

    startServer()
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing ScenarioName</expected-message>
</test-case>

<test-case id="wrong-scenario-name" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>preflight with wrong ScenarioName</description>
  <input language="go">
package main

import (
    "github.com/vrooli/api-core/preflight"
)

func main() {
    if preflight.Run(preflight.Config{
        ScenarioName: "wrong-name",
    }) {
        return
    }

    startServer()
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>ScenarioName mismatch</expected-message>
</test-case>

<test-case id="preflight-run-not-in-if" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>preflight.Run() called without if statement</description>
  <input language="go">
package main

import (
    "github.com/vrooli/api-core/preflight"
)

func main() {
    preflight.Run(preflight.Config{
        ScenarioName: "demo-app",
    })

    startServer()
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>preflight.Run must be wrapped in if statement</expected-message>
</test-case>

<test-case id="preflight-with-options" should-fail="false" path="api/main.go" scenario="demo-app">
  <description>preflight.Run() with additional options</description>
  <input language="go">
package main

import (
    "github.com/vrooli/api-core/preflight"
)

func main() {
    if preflight.Run(preflight.Config{
        ScenarioName:     "demo-app",
        DisableStaleness: false,
        SkipRebuild:      false,
    }) {
        return
    }

    startServer()
}
  </input>
</test-case>

<test-case id="log-setflags-before-preflight" should-fail="false" path="api/main.go" scenario="demo-app">
  <description>log.SetFlags before preflight - should PASS (whitelisted)</description>
  <input language="go">
package main

import (
    "log"

    "github.com/vrooli/api-core/preflight"
)

func main() {
    log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

    if preflight.Run(preflight.Config{
        ScenarioName: "demo-app",
    }) {
        return
    }

    startServer()
}
  </input>
</test-case>
*/

// CheckPreflight validates that main functions include proper preflight checks
// Uses AST-based analysis to ensure proper implementation
func CheckPreflight(content []byte, filePath string, scenario string) *Violation {
	contentStr := string(content)

	// Skip rule definition files
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

	// This rule is for long-running services (APIs), not CLIs
	normalized := filepath.ToSlash(filePath)
	if strings.HasPrefix(normalized, "cli/") || strings.Contains(normalized, "/cli/") {
		return nil
	}

	return checkPreflightAST(content, filePath, scenario)
}

// preflightCheckInfo tracks the preflight implementation
type preflightCheckInfo struct {
	found            bool
	position         int    // statement index in main()
	isMustRun        bool   // true for MustRun, false for Run
	hasIfWrapper     bool   // Run must be in if statement
	hasReturn        bool   // if block must have return
	scenarioName     string // extracted ScenarioName value
	lineNumber       int
	missingComponent string
}

func checkPreflightAST(content []byte, filePath string, scenario string) *Violation {
	contentStr := string(content)

	// Ensure content has package declaration for parsing
	if !strings.HasPrefix(strings.TrimSpace(contentStr), "package ") {
		contentStr = "package main\n" + contentStr
	}

	// Parse the file into an AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, []byte(contentStr), parser.ParseComments)
	if err != nil {
		return nil
	}

	// Check for preflight import
	hasPreflightImport := false
	for _, imp := range file.Imports {
		if imp.Path != nil {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if importPath == "github.com/vrooli/api-core/preflight" {
				hasPreflightImport = true
				break
			}
		}
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
		return nil // No main function
	}

	scenarioName := resolveScenarioName(scenario, filePath)
	return analyzeMainForPreflight(mainFunc, fset, scenarioName, filePath, contentStr, hasPreflightImport)
}

func analyzeMainForPreflight(mainFunc *ast.FuncDecl, fset *token.FileSet, scenarioName, filePath, content string, hasPreflightImport bool) *Violation {
	if mainFunc.Body == nil || len(mainFunc.Body.List) == 0 {
		return createPreflightViolation("Missing Preflight Check", "Empty main() function", fset.Position(mainFunc.Pos()).Line, scenarioName, filePath)
	}

	info := &preflightCheckInfo{}
	firstBusinessLogicPos := -1

	// Analyze each statement in main()
	for i, stmt := range mainFunc.Body.List {
		// Check for if-statement with preflight.Run()
		if ifStmt, ok := stmt.(*ast.IfStmt); ok {
			if callExpr := extractPreflightRunCall(ifStmt.Cond); callExpr != nil {
				info.found = true
				info.position = i
				info.isMustRun = false
				info.hasIfWrapper = true
				info.lineNumber = fset.Position(ifStmt.Pos()).Line
				info.scenarioName = extractScenarioNameFromCall(callExpr)
				info.hasReturn = blockHasReturn(ifStmt.Body)
				break
			}
		}

		// Check for standalone preflight.MustRun() or preflight.Run() call
		if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
			if callExpr, isMustRun := extractPreflightCall(exprStmt.X); callExpr != nil {
				info.found = true
				info.position = i
				info.isMustRun = isMustRun
				info.hasIfWrapper = false
				info.lineNumber = fset.Position(exprStmt.Pos()).Line
				info.scenarioName = extractScenarioNameFromCall(callExpr)
				break
			}
		}

		// Track business logic before preflight
		if !info.found && !isHarmlessSetupStmt(stmt) && firstBusinessLogicPos == -1 {
			firstBusinessLogicPos = i
		}
	}

	// Validate findings
	if !info.found {
		if !hasPreflightImport {
			return createPreflightViolation("Missing Preflight Check",
				"No preflight import found. Add: import \"github.com/vrooli/api-core/preflight\"",
				fset.Position(mainFunc.Pos()).Line, scenarioName, filePath)
		}
		return createPreflightViolation("Missing Preflight Check",
			"No preflight.Run() or preflight.MustRun() call found in main()",
			fset.Position(mainFunc.Pos()).Line, scenarioName, filePath)
	}

	// Check positioning
	if firstBusinessLogicPos != -1 && firstBusinessLogicPos < info.position {
		return createPreflightViolation("Preflight check not first statement",
			"Business logic executes before preflight check",
			info.lineNumber, scenarioName, filePath)
	}

	// For preflight.Run(), must be in if statement with return
	if !info.isMustRun {
		if !info.hasIfWrapper {
			return createPreflightViolation("preflight.Run must be wrapped in if statement",
				"preflight.Run() returns a bool - must check it: if preflight.Run(...) { return }",
				info.lineNumber, scenarioName, filePath)
		}
		if !info.hasReturn {
			return createPreflightViolation("Missing return after preflight.Run",
				"The if block must contain a return statement to exit after re-exec",
				info.lineNumber, scenarioName, filePath)
		}
	}

	// Validate ScenarioName
	if info.scenarioName == "" {
		return createPreflightViolation("Missing ScenarioName in preflight config",
			"ScenarioName must be provided for proper error messages",
			info.lineNumber, scenarioName, filePath)
	}

	if info.scenarioName != scenarioName {
		return createPreflightViolation("ScenarioName mismatch",
			"ScenarioName \""+info.scenarioName+"\" doesn't match expected \""+scenarioName+"\"",
			info.lineNumber, scenarioName, filePath)
	}

	return nil
}

// extractPreflightRunCall checks if an expression is preflight.Run() and returns the call
func extractPreflightRunCall(expr ast.Expr) *ast.CallExpr {
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil
	}

	sel, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != "preflight" {
		return nil
	}

	if sel.Sel.Name != "Run" {
		return nil
	}

	return callExpr
}

// extractPreflightCall checks if an expression is preflight.Run() or preflight.MustRun()
func extractPreflightCall(expr ast.Expr) (*ast.CallExpr, bool) {
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, false
	}

	sel, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != "preflight" {
		return nil, false
	}

	switch sel.Sel.Name {
	case "MustRun":
		return callExpr, true
	case "Run":
		return callExpr, false
	}

	return nil, false
}

// extractScenarioNameFromCall extracts the ScenarioName value from a preflight call
func extractScenarioNameFromCall(call *ast.CallExpr) string {
	if len(call.Args) == 0 {
		return ""
	}

	// First arg should be preflight.Config{...}
	compLit, ok := call.Args[0].(*ast.CompositeLit)
	if !ok {
		return ""
	}

	for _, elt := range compLit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		keyIdent, ok := kv.Key.(*ast.Ident)
		if !ok || keyIdent.Name != "ScenarioName" {
			continue
		}

		valueLit, ok := kv.Value.(*ast.BasicLit)
		if !ok || valueLit.Kind != token.STRING {
			continue
		}

		// Remove quotes
		return strings.Trim(valueLit.Value, `"`)
	}

	return ""
}

// blockHasReturn checks if a block statement contains a return statement at the top level
func blockHasReturn(block *ast.BlockStmt) bool {
	if block == nil {
		return false
	}

	for _, stmt := range block.List {
		if _, ok := stmt.(*ast.ReturnStmt); ok {
			return true
		}
	}

	return false
}

// isHarmlessSetupStmt checks if a statement is safe before preflight
func isHarmlessSetupStmt(stmt ast.Stmt) bool {
	exprStmt, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return false
	}

	call, ok := exprStmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Whitelist: log.SetFlags(), log.SetOutput(), log.SetPrefix()
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

// resolveScenarioName extracts the scenario name from the provided value or file path
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
	return "SCENARIO_NAME_HERE"
}

// createPreflightViolation creates a standardized violation for preflight issues
func createPreflightViolation(title, description string, lineNumber int, scenarioName, filePath string) *Violation {
	recommendation := "Add proper preflight check at the start of main():\n\n" +
		"import \"github.com/vrooli/api-core/preflight\"\n\n" +
		"func main() {\n" +
		"    if preflight.Run(preflight.Config{\n" +
		"        ScenarioName: \"" + scenarioName + "\",\n" +
		"    }) {\n" +
		"        return\n" +
		"    }\n\n" +
		"    // ... rest of main()\n" +
		"}\n"

	return &Violation{
		Type:           "preflight",
		Severity:       "critical",
		Title:          title,
		Description:    description,
		FilePath:       filePath,
		LineNumber:     lineNumber,
		Recommendation: recommendation,
		Standard:       "vrooli-preflight-v1",
	}
}
