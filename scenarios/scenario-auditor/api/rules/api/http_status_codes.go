package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"strings"
)

/*
Rule: HTTP Status Codes
Description: Use proper HTTP status codes in API responses
Reason: Ensures consistent API behavior and proper client error handling
Category: api
Severity: low
Standard: api-design-v1
Targets: api

<test-case id="raw-numeric-status" should-fail="true">
  <description>Using raw numeric HTTP status codes</description>
  <input language="go">
func handleRequest(w http.ResponseWriter, r *http.Request) {
    if err := validateInput(r); err != nil {
        w.WriteHeader(400)
        json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
        return
    }
    w.WriteHeader(200)
    json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>Raw HTTP Status Code</expected-message>
</test-case>

<test-case id="status-ok-with-error" should-fail="true">
  <description>Returning StatusOK (200) when there's an error</description>
  <input language="go">
func handleError(w http.ResponseWriter, r *http.Request) {
    err := processRequest(r)
    if err != nil {
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
        return
    }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Incorrect Status Code</expected-message>
</test-case>

<test-case id="proper-status-constants" should-fail="false">
  <description>Using proper HTTP status constants</description>
  <input language="go">
func handleRequest(w http.ResponseWriter, r *http.Request) {
    if err := validateInput(r); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
        return
    }

    data, err := processData(r)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]string{"error": "Internal error"})
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(data)
}
  </input>
</test-case>

<test-case id="gin-proper-status" should-fail="false">
  <description>Proper status codes with Gin framework</description>
  <input language="go">
func handleGinRequest(c *gin.Context) {
    var input RequestData
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    result, err := processInput(input)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Processing failed"})
        return
    }

    c.JSON(http.StatusCreated, result)
}
  </input>
</test-case>
*/

// CheckHTTPStatusCodes validates proper usage of HTTP status codes
// NOTE: This now uses AST-based analysis instead of regex for improved accuracy
func CheckHTTPStatusCodes(content []byte, filePath string) []Violation {
	// Use the AST-based implementation
	return CheckHTTPStatusCodesAST(content, filePath)
}

// CheckHTTPStatusCodesRegex is the legacy regex-based implementation
// DEPRECATED: Kept for reference only. Has known false positives.
func CheckHTTPStatusCodesRegex(content []byte, filePath string) []Violation {
	var violations []Violation
	contentStr := string(content)

	// Only check Go API files
	if !strings.HasSuffix(filePath, ".go") || !isAPIHandler(contentStr) {
		return violations
	}

	// Check for raw numeric status codes (anti-pattern)
	rawStatusPattern := regexp.MustCompile(`\.WriteHeader\((\d{3})\)`)

	lines := strings.Split(contentStr, "\n")
	rawStatusLines := make(map[int]bool)
	incorrectStatusLines := make(map[int]bool)
	depth := 0
	var errBlockDepths []int

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Adjust block depth for closing braces before evaluating the line
		closeCount := strings.Count(line, "}")
		if closeCount > 0 {
			depth -= closeCount
			if depth < 0 {
				depth = 0
			}
			for len(errBlockDepths) > 0 && errBlockDepths[len(errBlockDepths)-1] > depth {
				errBlockDepths = errBlockDepths[:len(errBlockDepths)-1]
			}
		}

		// Check for raw numeric codes
		if matches := rawStatusPattern.FindStringSubmatch(line); matches != nil {
			if !rawStatusLines[i] {
				violations = append(violations, Violation{
					Type:           "http_status_codes",
					Severity:       "low",
					Title:          "Raw HTTP Status Code",
					Description:    "Raw HTTP Status Code",
					FilePath:       filePath,
					LineNumber:     i + 1,
					CodeSnippet:    line,
					Recommendation: "Use http.Status* constants (e.g., http.StatusOK, http.StatusNotFound)",
					Standard:       "api-design-v1",
				})
				rawStatusLines[i] = true
			}
		}

		if strings.Contains(line, "http.StatusOK") {
			inErrorBlock := len(errBlockDepths) > 0
			containsErrorText := strings.Contains(line, "error")
			if !containsErrorText {
				for j := i; j < len(lines) && j <= i+3; j++ {
					if strings.Contains(lines[j], "\"error\"") || strings.Contains(lines[j], "err") {
						containsErrorText = true
						break
					}
				}
			}

			if (inErrorBlock || containsErrorText) && !incorrectStatusLines[i] {
				violations = append(violations, Violation{
					Type:           "http_status_codes",
					Severity:       "medium",
					Title:          "Incorrect Status Code",
					Description:    "Incorrect Status Code",
					FilePath:       filePath,
					LineNumber:     i + 1,
					CodeSnippet:    line,
					Recommendation: "Use appropriate error status codes (4xx/5xx) for errors",
					Standard:       "api-design-v1",
				})
				incorrectStatusLines[i] = true
			}
		}

		openCount := strings.Count(line, "{")
		depth += openCount
		if strings.Contains(trimmed, "if err != nil") {
			errBlockDepths = append(errBlockDepths, depth)
		}
	}

	return violations
}

func isAPIHandler(content string) bool {
	handlerIndicators := []string{
		"http.ResponseWriter",
		"*http.Request",
		"gin.Context",
		"fiber.Ctx",
		"echo.Context",
	}

	for _, indicator := range handlerIndicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}
	return false
}
// AST-based implementation of HTTP status code checking
// This replaces the regex-based approach with proper syntax tree analysis

type statusCodeVisitor struct {
	violations  []Violation
	filePath    string
	fileSet     *token.FileSet
	content     string

	// Function context
	currentFunc *ast.FuncDecl
	funcStack   []*ast.FuncDecl

	// Scope tracking for error blocks
	errorBlocks      map[ast.Node]bool
	errorBlockStmts  map[*ast.BlockStmt]bool  // Track block statements that are error blocks
	statusCalls      map[ast.Stmt]*statusCallInfo
	writeHeaderCalls map[*ast.BlockStmt]bool  // Track which error blocks have WriteHeader

	// Wrapper function cache
	wrapperFuncs map[*ast.FuncDecl]bool
}

type statusCallInfo struct {
	hasStatusCode bool
	position      token.Pos
}

// CheckHTTPStatusCodesAST is the AST-based implementation
func CheckHTTPStatusCodesAST(content []byte, filePath string) []Violation {
	// Only check Go API files
	if !strings.HasSuffix(filePath, ".go") {
		return nil
	}

	// Parse the file into an AST
	// If content doesn't start with "package", prepend it for parsing
	contentStr := string(content)
	if !strings.HasPrefix(strings.TrimSpace(contentStr), "package ") {
		contentStr = "package main\n" + contentStr
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, []byte(contentStr), parser.ParseComments)
	if err != nil {
		// If we can't parse it, return no violations (could be invalid Go code)
		return nil
	}

	// Check if this is an API handler file
	if !isAPIHandlerAST(file) {
		return nil
	}

	// Initialize visitor
	v := &statusCodeVisitor{
		filePath:         filePath,
		fileSet:          fset,
		content:          contentStr,
		errorBlocks:      make(map[ast.Node]bool),
		errorBlockStmts:  make(map[*ast.BlockStmt]bool),
		statusCalls:      make(map[ast.Stmt]*statusCallInfo),
		writeHeaderCalls: make(map[*ast.BlockStmt]bool),
		wrapperFuncs:     make(map[*ast.FuncDecl]bool),
	}

	// First pass: identify wrapper functions and error blocks
	ast.Inspect(file, v.firstPass)

	// Second pass: check for violations
	ast.Inspect(file, v.secondPass)

	return v.violations
}

// firstPass identifies wrapper functions, error blocks, and WriteHeader calls
func (v *statusCodeVisitor) firstPass(n ast.Node) bool {
	switch node := n.(type) {
	case *ast.FuncDecl:
		// Check if this is a wrapper function
		if isWrapperFuncAST(node) {
			v.wrapperFuncs[node] = true
		}
		v.funcStack = append(v.funcStack, node)
		return true

	case *ast.IfStmt:
		// Check if this is an error check
		if isErrorCheckAST(node.Cond) {
			// Mark all statements in the body as being in an error block
			v.markErrorBlock(node.Body)
			// Track the block statement itself for missing WriteHeader detection
			v.errorBlockStmts[node.Body] = true

			// Check if this error block contains WriteHeader calls
			v.trackWriteHeaderCalls(node.Body)
		}
		return true
	}

	// Continue traversal
	return true
}

// trackWriteHeaderCalls marks error blocks that contain WriteHeader calls
func (v *statusCodeVisitor) trackWriteHeaderCalls(block *ast.BlockStmt) {
	ast.Inspect(block, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "WriteHeader" {
					v.writeHeaderCalls[block] = true
					return false  // Found it, stop searching this block
				}
			}
		}
		return true
	})
}

// secondPass checks for violations
func (v *statusCodeVisitor) secondPass(n ast.Node) bool {
	switch node := n.(type) {
	case *ast.FuncDecl:
		v.currentFunc = node
		return true

	case *ast.CallExpr:
		v.checkStatusCodeCall(node)
		v.checkMissingStatusCode(node)
		return true
	}

	return true
}

// checkStatusCodeCall analyzes a function call for status code issues
func (v *statusCodeVisitor) checkStatusCodeCall(call *ast.CallExpr) {
	framework, argPos, isStatusCall := isStatusCodeCallAST(call)
	if !isStatusCall {
		return
	}

	// Check if we have enough arguments
	if len(call.Args) <= argPos {
		return
	}

	arg := call.Args[argPos]

	// Check for raw numeric literals
	if num, isLiteral := isRawNumberAST(arg); isLiteral && num >= 100 && num <= 599 {
		// Check if we're inside a wrapper function
		if v.currentFunc != nil && v.wrapperFuncs[v.currentFunc] {
			return // Wrapper functions are exempt
		}

		pos := v.fileSet.Position(call.Pos())
		v.violations = append(v.violations, Violation{
			Type:           "http_status_codes",
			Severity:       "low",
			Title:          "Raw HTTP Status Code",
			Description:    "Raw HTTP Status Code in " + framework + " framework",
			FilePath:       v.filePath,
			LineNumber:     pos.Line,
			CodeSnippet:    getCodeSnippet(v.fileSet, call),
			Recommendation: getRecommendation(framework),
			Standard:       "api-design-v1",
		})
		return
	}

	// Check for StatusOK in error context
	if isStatusOKAST(arg) {
		// Check if this call is in an error block
		if v.isInErrorBlock(call) {
			pos := v.fileSet.Position(call.Pos())
			v.violations = append(v.violations, Violation{
				Type:           "http_status_codes",
				Severity:       "medium",
				Title:          "Incorrect Status Code",
				Description:    "Returning StatusOK (200) in error context",
				FilePath:       v.filePath,
				LineNumber:     pos.Line,
				CodeSnippet:    getCodeSnippet(v.fileSet, call),
				Recommendation: "Use appropriate error status codes (4xx/5xx) for errors",
				Standard:       "api-design-v1",
			})
		}
	}
}

// checkMissingStatusCode detects json.Encode calls in error blocks without prior WriteHeader (stdlib only)
func (v *statusCodeVisitor) checkMissingStatusCode(call *ast.CallExpr) {
	// Only check stdlib (frameworks handle this automatically)
	if !strings.Contains(v.content, "http.ResponseWriter") {
		return
	}

	// Check if this is a json.NewEncoder(...).Encode(...) call
	if !isJSONEncodeCall(call) {
		return
	}

	// Check if we're in an error block
	inErrorBlock := v.isInErrorBlock(call)
	if !inErrorBlock {
		return
	}

	// Find which error block this is in
	errorBlock := v.findEnclosingErrorBlock(call)
	if errorBlock == nil {
		return
	}

	// Check if this error block has a WriteHeader call
	if v.writeHeaderCalls[errorBlock] {
		return  // WriteHeader was called, no violation
	}

	// No WriteHeader found - report violation
	pos := v.fileSet.Position(call.Pos())
	v.violations = append(v.violations, Violation{
		Type:           "http_status_codes",
		Severity:       "high",
		Title:          "Missing Status Code",
		Description:    "Returning response in error block without setting status code (will default to 200)",
		FilePath:       v.filePath,
		LineNumber:     pos.Line,
		CodeSnippet:    getCodeSnippet(v.fileSet, call),
		Recommendation: "Add w.WriteHeader(http.StatusInternalServerError) or appropriate error status before encoding response",
		Standard:       "api-design-v1",
	})
}

// isStatusCodeCallAST checks if a call expression is a status code setting call
func isStatusCodeCallAST(call *ast.CallExpr) (framework string, argPos int, ok bool) {
	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel {
		return "", 0, false
	}

	methodName := sel.Sel.Name

	// Standard library: w.WriteHeader(status)
	if methodName == "WriteHeader" {
		return "stdlib", 0, true
	}

	// Gin: c.JSON(status, data) or c.AbortWithStatusJSON(status, data) or c.Status(status)
	if methodName == "JSON" || methodName == "AbortWithStatusJSON" {
		return "gin", 0, true
	}
	if methodName == "Status" {
		// Could be Gin or Fiber - both use c.Status()
		return "gin/fiber", 0, true
	}

	// Fiber: c.SendStatus(status)
	if methodName == "SendStatus" {
		return "fiber", 0, true
	}

	// Echo: c.JSON(status, data) or c.String(status, text)
	if methodName == "String" {
		return "echo", 0, true
	}

	// Echo: echo.NewHTTPError(status, ...)
	if methodName == "NewHTTPError" {
		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "echo" {
			return "echo", 0, true
		}
	}

	return "", 0, false
}

// isRawNumberAST checks if an expression is a raw numeric literal
func isRawNumberAST(expr ast.Expr) (value int, ok bool) {
	lit, isLit := expr.(*ast.BasicLit)
	if !isLit || lit.Kind != token.INT {
		return 0, false
	}

	val, err := strconv.Atoi(lit.Value)
	if err != nil {
		return 0, false
	}

	return val, true
}

// isStatusOKAST checks if an expression represents StatusOK (200)
func isStatusOKAST(expr ast.Expr) bool {
	// Check for http.StatusOK or fiber.StatusOK
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if sel.Sel.Name == "StatusOK" {
			if ident, ok := sel.X.(*ast.Ident); ok {
				return ident.Name == "http" || ident.Name == "fiber"
			}
		}
	}

	// Check for literal 200
	if num, ok := isRawNumberAST(expr); ok && num == 200 {
		return true
	}

	return false
}

// isWrapperFuncAST checks if a function is a wrapper/utility function with statusCode parameter
func isWrapperFuncAST(fn *ast.FuncDecl) bool {
	if fn.Type == nil || fn.Type.Params == nil {
		return false
	}

	hasResponseWriter := false
	hasStatusParam := false

	for _, field := range fn.Type.Params.List {
		// Check for http.ResponseWriter parameter
		if star, ok := field.Type.(*ast.StarExpr); ok {
			if sel, ok := star.X.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "ResponseWriter" {
					if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "http" {
						hasResponseWriter = true
					}
				}
			}
		}

		// Check for int parameter with status-related name
		if ident, ok := field.Type.(*ast.Ident); ok && ident.Name == "int" {
			for _, name := range field.Names {
				paramName := strings.ToLower(name.Name)
				if strings.Contains(paramName, "status") ||
				   strings.Contains(paramName, "code") ||
				   paramName == "status" || paramName == "code" {
					hasStatusParam = true
				}
			}
		}
	}

	return hasResponseWriter && hasStatusParam
}

// isErrorCheckAST checks if a condition is an error check
func isErrorCheckAST(cond ast.Expr) bool {
	// Check for: err != nil, e != nil, etc.
	binExpr, ok := cond.(*ast.BinaryExpr)
	if !ok {
		return false
	}

	if binExpr.Op != token.NEQ {
		return false
	}

	// Check if right side is nil
	if ident, ok := binExpr.Y.(*ast.Ident); !ok || ident.Name != "nil" {
		return false
	}

	// Check if left side is error variable (err, e, error, etc.)
	switch left := binExpr.X.(type) {
	case *ast.Ident:
		name := strings.ToLower(left.Name)
		return name == "err" || name == "e" || strings.HasPrefix(name, "err")
	}

	return false
}

// markErrorBlock marks all nodes in a block statement as being in an error block
func (v *statusCodeVisitor) markErrorBlock(block *ast.BlockStmt) {
	if block == nil {
		return
	}

	for _, stmt := range block.List {
		v.errorBlocks[stmt] = true
		// Recursively mark nested blocks
		v.markNestedBlocks(stmt)
	}
}

// markNestedBlocks recursively marks nested block statements
func (v *statusCodeVisitor) markNestedBlocks(stmt ast.Stmt) {
	ast.Inspect(stmt, func(n ast.Node) bool {
		if n != nil {
			v.errorBlocks[n] = true
		}
		return true
	})
}

// isInErrorBlock checks if a node is inside an error block
func (v *statusCodeVisitor) isInErrorBlock(node ast.Node) bool {
	// Walk up the tree to find if any parent is marked as error block
	return v.errorBlocks[node]
}

// findEnclosingErrorBlock finds the error block that contains this node
func (v *statusCodeVisitor) findEnclosingErrorBlock(node ast.Node) *ast.BlockStmt {
	// This is a simplified approach - in a real implementation, we'd track
	// the block hierarchy during traversal
	for block := range v.errorBlockStmts {
		// Check if node is within this block
		found := false
		ast.Inspect(block, func(n ast.Node) bool {
			if n == node {
				found = true
				return false
			}
			return true
		})
		if found {
			return block
		}
	}
	return nil
}

// isJSONEncodeCall checks if a call is json.NewEncoder(...).Encode(...)
func isJSONEncodeCall(call *ast.CallExpr) bool {
	// Check for .Encode() call
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Encode" {
		return false
	}

	// Check if the receiver is json.NewEncoder(w)
	innerCall, ok := sel.X.(*ast.CallExpr)
	if !ok {
		return false
	}

	innerSel, ok := innerCall.Fun.(*ast.SelectorExpr)
	if !ok || innerSel.Sel.Name != "NewEncoder" {
		return false
	}

	// Check if it's json.NewEncoder
	if ident, ok := innerSel.X.(*ast.Ident); ok {
		return ident.Name == "json"
	}

	return false
}

// isAPIHandlerAST checks if a file contains API handler code
func isAPIHandlerAST(file *ast.File) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		// Look for http.ResponseWriter, gin.Context, fiber.Ctx, or echo.Context
		// in function parameters, variable types, or selectors
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if sel.Sel.Name == "ResponseWriter" || sel.Sel.Name == "Context" || sel.Sel.Name == "Ctx" {
				if ident, ok := sel.X.(*ast.Ident); ok {
					if ident.Name == "http" || ident.Name == "gin" || ident.Name == "fiber" || ident.Name == "echo" {
						found = true
						return false // Stop searching
					}
				}
			}
		}

		// Also check for methods like WriteHeader, JSON, etc. which are API-specific
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if sel.Sel.Name == "WriteHeader" || sel.Sel.Name == "JSON" ||
			   sel.Sel.Name == "SendStatus" || sel.Sel.Name == "AbortWithStatusJSON" {
				found = true
				return false
			}
		}

		return true
	})
	return found
}

// getCodeSnippet extracts the code snippet for a node
func getCodeSnippet(fset *token.FileSet, node ast.Node) string {
	start := fset.Position(node.Pos())
	_ = fset.Position(node.End()) // For future use when we extract actual source

	// For now, just return a simple representation
	// In a real implementation, we'd read the actual source
	return "line " + strconv.Itoa(start.Line)
}

// getRecommendation returns the appropriate recommendation based on framework
func getRecommendation(framework string) string {
	switch framework {
	case "stdlib":
		return "Use http.Status* constants (e.g., http.StatusOK, http.StatusBadRequest)"
	case "gin", "gin/fiber":
		return "Use http.Status* constants (e.g., c.JSON(http.StatusBadRequest, ...))"
	case "fiber":
		return "Use fiber.Status* constants (e.g., c.Status(fiber.StatusOK))"
	case "echo":
		return "Use http.Status* constants (e.g., c.JSON(http.StatusOK, ...))"
	default:
		return "Use named status code constants instead of raw numbers"
	}
}
