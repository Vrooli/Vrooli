package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

/*
Rule: Binary Content-Type Headers
Description: Binary file downloads must set appropriate Content-Type headers
Reason: Ensures browsers handle downloads correctly and prevents security issues
Category: api
Severity: high
Standard: api-design-v1
Targets: api

IMPLEMENTATION NOTES:
This rule analyses HTTP handlers that send downloadable responses. It focuses on
paths that explicitly set a `Content-Disposition` header (the clearest signal of
an attachment) and verifies that every execution path leading to that header has
already established an appropriate `Content-Type`.

The rule understands common delivery patterns:
- Direct header setting via `w.Header().Set` or `hdr.Set`.
- Gin helpers such as `c.Header`, `c.Data`, `c.File`, and `c.FileAttachment`.
- Shared helpers whose names imply they take ownership of response headers
  (e.g. `WriteJSON`, `RenderPDF`, `SendDownload`).

If any execution branch can reach `Content-Disposition` without a prior
`Content-Type`, a violation is reported. Serves that delegate to `http.ServeFile`
or Gin’s file helpers are treated as compliant because those APIs populate the
header automatically.

<test-case id="binary-missing-content-type" should-fail="true" path="api/download.go">
  <description>Content-Disposition set without Content-Type</description>
  <input language="go">
package main

import "net/http"

func downloadReport(w http.ResponseWriter, r *http.Request) {
    data := []byte("report")
    w.Header().Set("Content-Disposition", "attachment; filename=\"report.bin\"")
    w.Write(data)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Content-Disposition header detected without corresponding Content-Type</expected-message>
</test-case>

<test-case id="binary-with-inline-content-type" should-fail="false" path="api/download.go">
  <description>Content-Type and Content-Disposition set together</description>
  <input language="go">
package main

import "net/http"

func exportCSV(w http.ResponseWriter, r *http.Request) {
    rows := []byte("id,name\n1,Ada")
    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-Disposition", "attachment; filename=\"users.csv\"")
    w.Write(rows)
}
  </input>
</test-case>

<test-case id="binary-with-constant-header" should-fail="false" path="api/download.go">
  <description>Header key provided via constant</description>
  <input language="go">
package main

import "net/http"

const headerContentType = "Content-Type"

func downloadAsset(w http.ResponseWriter, r *http.Request) {
    payload := []byte("asset-data")
    w.Header().Set(headerContentType, "application/octet-stream")
    w.Header().Set("Content-Disposition", "attachment; filename=\"asset.bin\"")
    w.Write(payload)
}
  </input>
</test-case>

<test-case id="binary-serve-file" should-fail="false" path="api/download.go">
  <description>http.ServeFile handles Content-Type automatically</description>
  <input language="go">
package main

import "net/http"

func downloadManual(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "./manual.pdf")
}
  </input>
</test-case>

<test-case id="binary-header-alias-missing-content-type" should-fail="true" path="api/download.go">
  <description>Header alias without Content-Type should be flagged</description>
  <input language="go">
package main

import "net/http"

func exportBinary(w http.ResponseWriter, r *http.Request) {
    hdr := w.Header()
    hdr.Set("Content-Disposition", "attachment; filename=\"data.bin\"")
    w.Write([]byte("payload"))
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Content-Disposition header detected without corresponding Content-Type</expected-message>
</test-case>

<test-case id="binary-content-type-after-body" should-fail="true" path="api/download.go">
  <description>Setting Content-Type after writing the body must be flagged</description>
  <input language="go">
package main

import "net/http"

func exportLateHeader(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Disposition", "attachment; filename=\"late.csv\"")
    w.Write([]byte("id,name\n1,Ada"))
    w.Header().Set("Content-Type", "text/csv")
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Content-Disposition header detected without corresponding Content-Type</expected-message>
</test-case>

<test-case id="binary-gin-missing-content-type" should-fail="true" path="api/download.go">
  <description>gin.Context Header usage must set Content-Type</description>
  <input language="go">
package main

import "github.com/gin-gonic/gin"

func exportGin(c *gin.Context) {
    c.Header("Content-Disposition", "attachment; filename=\"report.csv\"")
    c.String(200, "data")
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Content-Disposition header detected without corresponding Content-Type</expected-message>
</test-case>

<test-case id="binary-nosniff-without-content-type" should-fail="true" path="api/download.go">
  <description>X-Content-Type-Options header alone is insufficient</description>
  <input language="go">
package main

import "net/http"

func exportNosniffOnly(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Header().Set("Content-Disposition", "attachment; filename=\"report.bin\"")
    w.Write([]byte("data"))
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Content-Disposition header detected without corresponding Content-Type</expected-message>
</test-case>

<test-case id="binary-helper-sets-content-type" should-fail="false" path="api/download.go">
  <description>Helper function that sets Content-Type should pass</description>
  <input language="go">
package main

import (
    "net/http"
)

func WriteJSON(w http.ResponseWriter, status int, payload any) error {
    w.Header().Set("Content-Type", "application/json")
    if status > 0 {
        w.WriteHeader(status)
    }
    return nil
}

func exportJSON(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Disposition", "attachment; filename=\"export.json\"")
    _ = WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
  </input>
</test-case>

<test-case id="binary-helper-missing-content-type" should-fail="true" path="api/download.go">
  <description>Helper that writes bytes without Content-Type must be flagged</description>
  <input language="go">
package main

import "net/http"

func WriteCSV(w http.ResponseWriter, data []byte) error {
    _, _ = w.Write(data)
    return nil
}

func exportCSV(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Disposition", "attachment; filename=\"export.csv\"")
    _ = WriteCSV(w, []byte("a,b\n1,2"))
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Content-Disposition header detected without corresponding Content-Type</expected-message>
</test-case>

<test-case id="binary-conditional-content-type" should-fail="true" path="api/download.go">
  <description>Guarded Content-Type must still trigger when other branches omit it</description>
  <input language="go">
package main

import "net/http"

func downloadConditional(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Disposition", "attachment; filename=\"report.bin\"")
    if r.URL.Query().Get("format") == "pdf" {
        w.Header().Set("Content-Type", "application/pdf")
    }
    w.Write([]byte("data"))
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Content-Disposition header detected without corresponding Content-Type</expected-message>
</test-case>

<test-case id="binary-gin-data-with-content-type" should-fail="false" path="api/download.go">
  <description>Gin Data helper provides content type without manual header</description>
  <input language="go">
package main

import "github.com/gin-gonic/gin"

func exportGinData(c *gin.Context) {
    c.Header("Content-Disposition", "attachment; filename=\"export.json\"")
    c.Data(200, "application/json", []byte(`{"ok":true}`))
}
  </input>
</test-case>

<test-case id="binary-gin-data-from-reader" should-fail="false" path="api/download.go">
  <description>Gin DataFromReader with explicit content type should pass</description>
  <input language="go">
package main

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

func exportGinReader(c *gin.Context) {
    reader := strings.NewReader("payload")
    c.Header("Content-Disposition", "attachment; filename=\"export.bin\"")
    c.DataFromReader(http.StatusOK, "application/octet-stream", reader, nil)
}
  </input>
</test-case>

<test-case id="binary-header-map-missing-content-type" should-fail="true" path="api/download.go">
  <description>Header map assignment without Content-Type must be flagged</description>
  <input language="go">
package main

import "net/http"

func exportMapMissing(w http.ResponseWriter, r *http.Request) {
    hdr := w.Header()
    hdr["Content-Disposition"] = []string{"attachment; filename=\"missing.bin\""}
    w.Write([]byte("payload"))
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Content-Disposition header detected without corresponding Content-Type</expected-message>
</test-case>

<test-case id="binary-header-map-content-type" should-fail="false" path="api/download.go">
  <description>Header map assignment covering Content-Type should pass</description>
  <input language="go">
package main

import "net/http"

func exportMapHeaders(w http.ResponseWriter, r *http.Request) {
    hdr := w.Header()
    hdr["Content-Type"] = []string{"application/pdf"}
    hdr["Content-Disposition"] = []string{"attachment; filename=\"report.pdf\""}
    w.Write([]byte("pdf"))
}
  </input>
</test-case>

<test-case id="binary-canonical-header" should-fail="false" path="api/download.go">
  <description>Canonical header helpers should satisfy Content-Type detection</description>
  <input language="go">
package main

import "net/http"

func exportCanonical(w http.ResponseWriter, r *http.Request) {
    hdr := w.Header()
    key := http.CanonicalHeaderKey("Content-Type")
    hdr[key] = []string{"application/json"}
    hdr.Set("Content-Disposition", "attachment; filename=\"canonical.json\"")
    w.Write([]byte("{}"))
}
  </input>
</test-case>
*/

// CheckBinaryContentTypeHeaders validates binary file download Content-Type headers
func CheckBinaryContentTypeHeaders(content []byte, filePath string) []Violation {
	if !strings.HasSuffix(filePath, ".go") || isTestFile(filePath) {
		return nil
	}

	originalLines := strings.Split(string(content), "\n")
	trimmed := strings.TrimSpace(string(content))
	parsedSource := string(content)
	lineOffset := 0
	if !strings.HasPrefix(trimmed, "package ") {
		parsedSource = "package main\n" + parsedSource
		lineOffset = 1
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, parsedSource, parser.ParseComments)
	if err != nil {
		return nil
	}

	var violations []Violation

	functionDecls := make(map[string]*ast.FuncDecl)
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil {
			continue
		}
		functionDecls[strings.ToLower(fn.Name.Name)] = fn
	}

	globalContentAliases, globalDispositionAliases := collectGlobalAliasInfo(file)

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		info := analyzeBinaryDownloadFunction(fn, functionDecls, globalContentAliases, globalDispositionAliases)
		if len(info.missingDispositionPositions) == 0 {
			continue
		}
		for _, pos := range info.missingDispositionPositions {
			line := fset.Position(pos).Line - lineOffset
			if line < 1 {
				line = 1
			}
			snippet := ""
			if line-1 >= 0 && line-1 < len(originalLines) {
				snippet = strings.TrimSpace(originalLines[line-1])
			}

			violations = append(violations, Violation{
				Type:           "content_type_binary",
				Severity:       "high",
				Title:          "Missing binary Content-Type header",
				Description:    "Content-Disposition header detected without corresponding Content-Type",
				Message:        "Content-Disposition header detected without corresponding Content-Type",
				FilePath:       filePath,
				LineNumber:     line,
				CodeSnippet:    snippet,
				Recommendation: "Set the Content-Type header (e.g., application/pdf, text/csv) before writing binary responses",
				Standard:       "api-design-v1",
			})
		}
	}

	return violations
}

type binaryFunctionInfo struct {
	missingDispositionPositions []token.Pos
	headerAliases               map[string]struct{}
	responseWriterNames         map[string]struct{}
	contextNames                map[string]struct{}
	recorded                    map[token.Pos]struct{}
	contentTypeKeyAliases       map[string]struct{}
	dispositionKeyAliases       map[string]struct{}
}

type flowState struct {
	hasContentType bool
	bodyWritten    bool
	pending        []token.Pos
}

func (info *binaryFunctionInfo) recordViolation(pos token.Pos) {
	if info.recorded == nil {
		info.recorded = make(map[token.Pos]struct{})
	}
	if _, ok := info.recorded[pos]; ok {
		return
	}
	info.recorded[pos] = struct{}{}
	info.missingDispositionPositions = append(info.missingDispositionPositions, pos)
}

func (info *binaryFunctionInfo) markContentTypeKey(name string) {
	if info.contentTypeKeyAliases == nil {
		info.contentTypeKeyAliases = make(map[string]struct{})
	}
	info.contentTypeKeyAliases[name] = struct{}{}
}

func (info *binaryFunctionInfo) markDispositionKey(name string) {
	if info.dispositionKeyAliases == nil {
		info.dispositionKeyAliases = make(map[string]struct{})
	}
	info.dispositionKeyAliases[name] = struct{}{}
}

func (info *binaryFunctionInfo) isContentTypeKey(name string) bool {
	if info.contentTypeKeyAliases == nil {
		return false
	}
	_, ok := info.contentTypeKeyAliases[name]
	return ok
}

func (info *binaryFunctionInfo) isDispositionKey(name string) bool {
	if info.dispositionKeyAliases == nil {
		return false
	}
	_, ok := info.dispositionKeyAliases[name]
	return ok
}

func cloneState(state flowState) flowState {
	if len(state.pending) == 0 {
		return state
	}
	copyPending := make([]token.Pos, len(state.pending))
	copy(copyPending, state.pending)
	state.pending = copyPending
	return state
}

func mergePending(a, b []token.Pos) []token.Pos {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	seen := make(map[token.Pos]struct{}, len(a)+len(b))
	merged := make([]token.Pos, 0, len(a)+len(b))
	for _, pos := range a {
		if _, ok := seen[pos]; ok {
			continue
		}
		seen[pos] = struct{}{}
		merged = append(merged, pos)
	}
	for _, pos := range b {
		if _, ok := seen[pos]; ok {
			continue
		}
		seen[pos] = struct{}{}
		merged = append(merged, pos)
	}
	return merged
}

func (state flowState) addPending(pos token.Pos) flowState {
	newState := cloneState(state)
	newState.pending = append(newState.pending, pos)
	return newState
}

func (state flowState) clearPending() flowState {
	if len(state.pending) == 0 {
		return state
	}
	state.pending = nil
	return state
}

func (state flowState) flushPending(info *binaryFunctionInfo) flowState {
	if len(state.pending) == 0 {
		return state
	}
	for _, pos := range state.pending {
		info.recordViolation(pos)
	}
	state.pending = nil
	return state
}

func analyzeBinaryDownloadFunction(fn *ast.FuncDecl, functions map[string]*ast.FuncDecl, globalContentAliases, globalDispositionAliases map[string]struct{}) binaryFunctionInfo {
	info := binaryFunctionInfo{
		headerAliases:         make(map[string]struct{}),
		responseWriterNames:   collectResponseWriterParams(fn),
		contextNames:          collectContextParams(fn),
		contentTypeKeyAliases: cloneStringSet(globalContentAliases),
		dispositionKeyAliases: cloneStringSet(globalDispositionAliases),
	}

	visited := make(map[string]struct{})
	finalState := analyzeStmtList(fn.Body.List, flowState{}, &info, functions, visited)
	finalState = finalState.flushPending(&info)

	return info
}

var (
	dispositionKeywords = []string{"content-disposition", "contentdisposition"}
	contentTypeKeywords = []string{"content-type", "contenttype"}
	trustedHelperNames  = map[string]struct{}{
		"writejson":            {},
		"respondjson":          {},
		"renderjson":           {},
		"jsonresponse":         {},
		"handlers.writejson":   {},
		"handlers.respondjson": {},
		"handlers.renderpdf":   {},
	}
)

func handleHeaderAssignments(assign *ast.AssignStmt, state flowState, info *binaryFunctionInfo) flowState {
	if len(assign.Lhs) == 0 {
		return state
	}
	for i, lhs := range assign.Lhs {
		var rhs ast.Expr
		if i < len(assign.Rhs) {
			rhs = assign.Rhs[i]
		}
		state = processHeaderAssignment(lhs, rhs, state, info)
	}
	return state
}

func processHeaderAssignment(lhs ast.Expr, _ ast.Expr, state flowState, info *binaryFunctionInfo) flowState {
	indexExpr, ok := lhs.(*ast.IndexExpr)
	if !ok {
		return state
	}
	if !isHeaderLikeExpr(indexExpr.X, info) {
		return state
	}
	if headerKeyMatches(indexExpr.Index, contentTypeKeywords, info) {
		state.hasContentType = true
		return state.clearPending()
	}
	if headerKeyMatches(indexExpr.Index, dispositionKeywords, info) {
		if !state.hasContentType {
			return state.addPending(indexExpr.Pos())
		}
	}
	return state
}

func isHeaderLikeExpr(expr ast.Expr, info *binaryFunctionInfo) bool {
	if isHeaderSelector(expr) {
		return true
	}
	name := strings.ToLower(exprToString(expr))
	if name == "" {
		return false
	}
	_, ok := info.headerAliases[name]
	return ok
}

func isServeFileName(name string) bool {
	return strings.HasSuffix(name, ".servefile") || strings.HasSuffix(name, ".servecontent") || strings.HasSuffix(name, "servefile") || strings.HasSuffix(name, "servecontent")
}

func isBodyWriteCall(call *ast.CallExpr, info *binaryFunctionInfo) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	method := strings.ToLower(sel.Sel.Name)
	if isResponseWriterExpr(sel.X, info.responseWriterNames) {
		switch method {
		case "write", "writeheader", "writebytes", "writestring", "writejson", "writebinary":
			return true
		}
	}
	if isContextExpr(sel.X, info.contextNames) {
		switch method {
		case "data", "string", "file", "fileattachment", "filefromfs", "datafromreader", "datafromreaderwithheaders", "ssevent", "stream":
			return true
		}
	}
	return false
}

func analyzeStmtList(stmts []ast.Stmt, state flowState, info *binaryFunctionInfo, functions map[string]*ast.FuncDecl, visited map[string]struct{}) flowState {
	for _, stmt := range stmts {
		state = analyzeStmt(stmt, state, info, functions, visited)
	}
	return state
}

func analyzeStmt(stmt ast.Stmt, state flowState, info *binaryFunctionInfo, functions map[string]*ast.FuncDecl, visited map[string]struct{}) flowState {
	switch node := stmt.(type) {
	case *ast.BlockStmt:
		return analyzeStmtList(node.List, state, info, functions, visited)
	case *ast.ExprStmt:
		return handleCallExpr(node.X, state, info, functions, visited)
	case *ast.AssignStmt:
		recordHeaderKeyAliases(info, node.Lhs, node.Rhs)
		recordAssignmentAliases(info, node)
		state = handleHeaderAssignments(node, state, info)
		for _, rhs := range node.Rhs {
			state = handleCallExpr(rhs, state, info, functions, visited)
		}
		return state
	case *ast.DeclStmt:
		return analyzeDeclStmt(node, state, info, functions, visited)
	case *ast.DeferStmt:
		return handleCallExpr(node.Call, state, info, functions, visited)
	case *ast.GoStmt:
		return handleCallExpr(node.Call, state, info, functions, visited)
	case *ast.IfStmt:
		return analyzeIfStmt(node, state, info, functions, visited)
	case *ast.ForStmt:
		if node.Init != nil {
			state = analyzeStmt(node.Init, state, info, functions, visited)
		}
		if node.Cond != nil {
			state = analyzeExpr(node.Cond, state, info, functions, visited)
		}
		bodyState := analyzeStmtList(node.Body.List, state, info, functions, visited)
		if node.Post != nil {
			_ = analyzeStmt(node.Post, bodyState, info, functions, visited)
		}
		return state
	case *ast.RangeStmt:
		_ = analyzeStmtList(node.Body.List, state, info, functions, visited)
		return state
	case *ast.SwitchStmt:
		return analyzeSwitchStmt(node, state, info, functions, visited)
	case *ast.TypeSwitchStmt:
		return state
	case *ast.ReturnStmt:
		return state.flushPending(info)
	default:
		return state
	}
}

func analyzeIfStmt(node *ast.IfStmt, state flowState, info *binaryFunctionInfo, functions map[string]*ast.FuncDecl, visited map[string]struct{}) flowState {
	if node.Init != nil {
		state = analyzeStmt(node.Init, state, info, functions, visited)
	}
	if node.Cond != nil {
		state = analyzeExpr(node.Cond, state, info, functions, visited)
	}
	thenState := analyzeStmtList(node.Body.List, cloneState(state), info, functions, visited)
	elseState := cloneState(state)
	if node.Else != nil {
		elseState = analyzeElse(node.Else, cloneState(state), info, functions, visited)
	}
	state.hasContentType = thenState.hasContentType && elseState.hasContentType
	state.bodyWritten = state.bodyWritten || thenState.bodyWritten || elseState.bodyWritten
	state.pending = mergePending(thenState.pending, elseState.pending)
	return state
}

func analyzeElse(stmt ast.Stmt, state flowState, info *binaryFunctionInfo, functions map[string]*ast.FuncDecl, visited map[string]struct{}) flowState {
	switch node := stmt.(type) {
	case *ast.BlockStmt:
		return analyzeStmtList(node.List, cloneState(state), info, functions, visited)
	case *ast.IfStmt:
		return analyzeIfStmt(node, cloneState(state), info, functions, visited)
	default:
		return state
	}
}

func analyzeSwitchStmt(node *ast.SwitchStmt, state flowState, info *binaryFunctionInfo, functions map[string]*ast.FuncDecl, visited map[string]struct{}) flowState {
	if node.Init != nil {
		state = analyzeStmt(node.Init, state, info, functions, visited)
	}
	if node.Tag != nil {
		state = analyzeExpr(node.Tag, state, info, functions, visited)
	}
	hasDefault := false
	allCasesCT := true
	mergedPending := append([]token.Pos{}, state.pending...)
	for _, clause := range node.Body.List {
		caseClause, ok := clause.(*ast.CaseClause)
		if !ok {
			continue
		}
		if caseClause.List == nil {
			hasDefault = true
		}
		caseState := analyzeStmtList(caseClause.Body, cloneState(state), info, functions, visited)
		allCasesCT = allCasesCT && caseState.hasContentType
		mergedPending = mergePending(mergedPending, caseState.pending)
		state.bodyWritten = state.bodyWritten || caseState.bodyWritten
	}
	if hasDefault {
		state.hasContentType = state.hasContentType && allCasesCT
	}
	state.pending = mergedPending
	return state
}

func analyzeExpr(expr ast.Expr, state flowState, info *binaryFunctionInfo, functions map[string]*ast.FuncDecl, visited map[string]struct{}) flowState {
	switch v := expr.(type) {
	case *ast.CallExpr:
		return handleCallExpr(v, state, info, functions, visited)
	case *ast.BinaryExpr:
		state = analyzeExpr(v.X, state, info, functions, visited)
		return analyzeExpr(v.Y, state, info, functions, visited)
	case *ast.UnaryExpr:
		return analyzeExpr(v.X, state, info, functions, visited)
	case *ast.ParenExpr:
		return analyzeExpr(v.X, state, info, functions, visited)
	default:
		return state
	}
}

func handleCallExpr(expr ast.Expr, state flowState, info *binaryFunctionInfo, functions map[string]*ast.FuncDecl, visited map[string]struct{}) flowState {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return state
	}

	if isHeaderCallWithKey(call, dispositionKeywords, info) {
		if !state.hasContentType {
			state = state.addPending(call.Pos())
		}
		return state
	}

	if isHeaderCallWithKey(call, contentTypeKeywords, info) {
		state.hasContentType = true
		return state.clearPending()
	}

	name := strings.ToLower(exprToString(call.Fun))
	if isServeFileName(name) {
		state.hasContentType = true
		state = state.clearPending()
		state.bodyWritten = true
		return state.flushPending(info)
	}

	if helperCallSetsContentType(call, info, functions, visited) {
		state.hasContentType = true
		return state.clearPending()
	}

	if ginCallSetsContentType(call, info.contextNames) {
		state.hasContentType = true
		state = state.clearPending()
		state.bodyWritten = true
		return state.flushPending(info)
	}

	if isBodyWriteCall(call, info) {
		state.bodyWritten = true
		return state.flushPending(info)
	}

	return state
}

func analyzeDeclStmt(declStmt *ast.DeclStmt, state flowState, info *binaryFunctionInfo, functions map[string]*ast.FuncDecl, visited map[string]struct{}) flowState {
	genDecl, ok := declStmt.Decl.(*ast.GenDecl)
	if !ok {
		return state
	}
	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		recordValueSpecAliases(info, valueSpec)
		for _, value := range valueSpec.Values {
			state = handleCallExpr(value, state, info, functions, visited)
		}
	}
	return state
}

func ginCallSetsContentType(call *ast.CallExpr, contexts map[string]struct{}) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if !isContextExpr(sel.X, contexts) {
		return false
	}
	method := strings.ToLower(sel.Sel.Name)
	switch method {
	case "data":
		if len(call.Args) >= 2 {
			if _, ok := stringLiteral(call.Args[1]); ok {
				return true
			}
		}
		return false
	case "datafromreader", "datafromreaderwithheaders":
		if len(call.Args) >= 2 {
			if _, ok := stringLiteral(call.Args[1]); ok {
				return true
			}
		}
		return false
	case "file", "fileattachment", "filefromfs":
		return true
	case "json", "purejson", "indentedjson", "xml", "yaml", "protobuf",
		"asciijson", "securejson":
		return true
	default:
		return false
	}
}

func helperCallSetsContentType(call *ast.CallExpr, info *binaryFunctionInfo, functions map[string]*ast.FuncDecl, visited map[string]struct{}) bool {
	if len(call.Args) == 0 {
		return false
	}
	firstArg := call.Args[0]
	if !isResponseWriterExpr(firstArg, info.responseWriterNames) && !isContextExpr(firstArg, info.contextNames) {
		return false
	}

	helperName := strings.ToLower(exprToString(call.Fun))
	if helperName == "" {
		return false
	}
	base := helperBaseName(helperName)

	if fn, ok := functions[base]; ok && fn != nil {
		if functionSetsContentType(fn, functions, visited) {
			return true
		}
	}

	if isTrustedHelperName(helperName) || isTrustedHelperName(base) {
		return true
	}

	return false
}

func functionSetsContentType(fn *ast.FuncDecl, functions map[string]*ast.FuncDecl, visited map[string]struct{}) bool {
	if fn == nil || fn.Body == nil || fn.Name == nil {
		return false
	}

	name := strings.ToLower(fn.Name.Name)
	if _, seen := visited[name]; seen {
		return false
	}
	visited[name] = struct{}{}

	helperInfo := binaryFunctionInfo{
		headerAliases:         make(map[string]struct{}),
		responseWriterNames:   collectResponseWriterParams(fn),
		contextNames:          collectContextParams(fn),
		contentTypeKeyAliases: make(map[string]struct{}),
		dispositionKeyAliases: make(map[string]struct{}),
	}
	helperState := analyzeStmtList(fn.Body.List, flowState{}, &helperInfo, functions, visited)
	helperState = helperState.flushPending(&helperInfo)
	return helperState.hasContentType
}

func helperBaseName(name string) string {
	if idx := strings.LastIndex(name, "."); idx != -1 && idx+1 < len(name) {
		return name[idx+1:]
	}
	return name
}

func isTrustedHelperName(name string) bool {
	if _, ok := trustedHelperNames[name]; ok {
		return true
	}
	for _, token := range []string{
		"writejson", "renderjson", "respondjson", "setcontenttype", "renderxml", "writexml",
		"writeyaml", "renderyaml", "writeproto", "renderproto", "writepdf", "renderpdf",
		"download", "senddownload", "sendfile",
	} {
		if strings.Contains(name, token) {
			return true
		}
	}
	return false
}

func isHeaderCallWithKey(call *ast.CallExpr, keywords []string, info *binaryFunctionInfo) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	method := strings.ToLower(sel.Sel.Name)
	switch method {
	case "set", "add":
		if !isHeaderSelector(sel.X) && !isHeaderAlias(sel.X, info.headerAliases) {
			return false
		}
	case "header":
		if !isContextExpr(sel.X, info.contextNames) {
			return false
		}
	default:
		return false
	}

	if len(call.Args) == 0 {
		return false
	}

	return headerKeyMatches(call.Args[0], keywords, info)
}

func isHeaderAlias(expr ast.Expr, aliases map[string]struct{}) bool {
	name := strings.ToLower(exprToString(expr))
	if name == "" {
		return false
	}
	_, ok := aliases[name]
	return ok
}

func collectResponseWriterParams(fn *ast.FuncDecl) map[string]struct{} {
	names := make(map[string]struct{})
	if fn.Type == nil || fn.Type.Params == nil {
		return names
	}
	for _, field := range fn.Type.Params.List {
		if !isResponseWriterType(field.Type) {
			continue
		}
		for _, name := range field.Names {
			if name == nil || name.Name == "" || name.Name == "_" {
				continue
			}
			names[strings.ToLower(name.Name)] = struct{}{}
		}
	}
	return names
}

func collectContextParams(fn *ast.FuncDecl) map[string]struct{} {
	names := make(map[string]struct{})
	if fn.Type == nil || fn.Type.Params == nil {
		return names
	}
	for _, field := range fn.Type.Params.List {
		if !isGinContextType(field.Type) {
			continue
		}
		for _, name := range field.Names {
			if name == nil || name.Name == "" || name.Name == "_" {
				continue
			}
			names[strings.ToLower(name.Name)] = struct{}{}
		}
	}
	return names
}

func isResponseWriterType(expr ast.Expr) bool {
	switch v := expr.(type) {
	case *ast.SelectorExpr:
		return strings.EqualFold(v.Sel.Name, "ResponseWriter")
	case *ast.StarExpr:
		return isResponseWriterType(v.X)
	default:
		return false
	}
}

func isGinContextType(expr ast.Expr) bool {
	switch v := expr.(type) {
	case *ast.SelectorExpr:
		return strings.EqualFold(v.Sel.Name, "Context")
	case *ast.StarExpr:
		return isGinContextType(v.X)
	default:
		return false
	}
}

func recordAssignmentAliases(info *binaryFunctionInfo, assign *ast.AssignStmt) {
	if len(assign.Lhs) == 0 || len(assign.Rhs) != 1 {
		return
	}
	rhs := assign.Rhs[0]
	addHeaderAliasIfNeeded(info, assign.Lhs, rhs)
	addResponseWriterAliasIfNeeded(info, assign.Lhs, rhs)
	addContextAliasIfNeeded(info, assign.Lhs, rhs)
}

func recordHeaderKeyAliases(info *binaryFunctionInfo, lhs []ast.Expr, rhs []ast.Expr) {
	if len(lhs) == 0 || len(rhs) == 0 {
		return
	}
	limit := len(lhs)
	if len(rhs) < limit {
		limit = len(rhs)
	}
	for i := 0; i < limit; i++ {
		ident, ok := lhs[i].(*ast.Ident)
		if !ok || ident.Name == "" || ident.Name == "_" {
			continue
		}
		name := strings.ToLower(ident.Name)
		if headerKeyMatches(rhs[i], contentTypeKeywords, info) {
			info.markContentTypeKey(name)
		}
		if headerKeyMatches(rhs[i], dispositionKeywords, info) {
			info.markDispositionKey(name)
		}
	}
}

func recordValueSpecAliases(info *binaryFunctionInfo, spec *ast.ValueSpec) {
	if len(spec.Names) == 0 || len(spec.Values) == 0 {
		return
	}
	targets := exprSliceFromNames(spec.Names)
	recordHeaderKeyAliases(info, targets, spec.Values)
	if len(spec.Values) != 1 {
		return
	}
	rhs := spec.Values[0]
	addHeaderAliasIfNeeded(info, targets, rhs)
	addResponseWriterAliasIfNeeded(info, targets, rhs)
	addContextAliasIfNeeded(info, targets, rhs)
}

func exprSliceFromNames(names []*ast.Ident) []ast.Expr {
	exprs := make([]ast.Expr, 0, len(names))
	for _, name := range names {
		exprs = append(exprs, name)
	}
	return exprs
}

func addHeaderAliasIfNeeded(info *binaryFunctionInfo, lhs []ast.Expr, rhs ast.Expr) {
	if !isHeaderCallExpr(rhs, info) {
		return
	}
	for _, target := range lhs {
		ident, ok := target.(*ast.Ident)
		if !ok || ident.Name == "" || ident.Name == "_" {
			continue
		}
		info.headerAliases[strings.ToLower(ident.Name)] = struct{}{}
	}
}

func addResponseWriterAliasIfNeeded(info *binaryFunctionInfo, lhs []ast.Expr, rhs ast.Expr) {
	if !isResponseWriterExpr(rhs, info.responseWriterNames) {
		return
	}
	for _, target := range lhs {
		ident, ok := target.(*ast.Ident)
		if !ok || ident.Name == "" || ident.Name == "_" {
			continue
		}
		info.responseWriterNames[strings.ToLower(ident.Name)] = struct{}{}
	}
}

func addContextAliasIfNeeded(info *binaryFunctionInfo, lhs []ast.Expr, rhs ast.Expr) {
	if !isContextExpr(rhs, info.contextNames) {
		return
	}
	for _, target := range lhs {
		ident, ok := target.(*ast.Ident)
		if !ok || ident.Name == "" || ident.Name == "_" {
			continue
		}
		info.contextNames[strings.ToLower(ident.Name)] = struct{}{}
	}
}

func isHeaderCallExpr(expr ast.Expr, info *binaryFunctionInfo) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !strings.EqualFold(sel.Sel.Name, "Header") {
		return false
	}
	name := strings.ToLower(exprToString(sel.X))
	if name == "" {
		return false
	}
	_, ok = info.responseWriterNames[name]
	return ok
}

func isResponseWriterExpr(expr ast.Expr, names map[string]struct{}) bool {
	name := strings.ToLower(exprToString(expr))
	if name == "" {
		return false
	}
	_, ok := names[name]
	return ok
}

func isContextExpr(expr ast.Expr, contexts map[string]struct{}) bool {
	name := strings.ToLower(exprToString(expr))
	if name == "" {
		return false
	}
	_, ok := contexts[name]
	return ok
}

func isHeaderSelector(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return sel.Sel.Name == "Header"
}

func headerKeyMatches(expr ast.Expr, keywords []string, info *binaryFunctionInfo) bool {
	if str, ok := stringLiteral(expr); ok {
		return keywordMatchesValue(str, keywords)
	}

	switch node := expr.(type) {
	case *ast.CallExpr:
		fnName := strings.ToLower(exprToString(node.Fun))
		if strings.Contains(fnName, "canonicalheaderkey") && len(node.Args) > 0 {
			return headerKeyMatches(node.Args[0], keywords, info)
		}
	case *ast.Ident:
		name := strings.ToLower(node.Name)
		if info != nil {
			if keywordRepresentsContentType(keywords) && info.isContentTypeKey(name) {
				return true
			}
			if keywordRepresentsContentDisposition(keywords) && info.isDispositionKey(name) {
				return true
			}
		}
		return keywordMatchesValue(name, keywords)
	}

	name := exprToString(expr)
	if name == "" {
		return false
	}
	return keywordMatchesValue(name, keywords)
}

func keywordMatchesValue(value string, keywords []string) bool {
	normalized := normalizeHeaderKeyword(value)
	for _, keyword := range keywords {
		if normalizeHeaderKeyword(keyword) == normalized {
			return true
		}
	}
	return false
}

func keywordRepresentsContentType(keywords []string) bool {
	for _, keyword := range keywords {
		if normalizeHeaderKeyword(keyword) == "contenttype" {
			return true
		}
	}
	return false
}

func keywordRepresentsContentDisposition(keywords []string) bool {
	for _, keyword := range keywords {
		if normalizeHeaderKeyword(keyword) == "contentdisposition" {
			return true
		}
	}
	return false
}

func normalizeHeaderKeyword(s string) string {
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(lower))
	for _, r := range lower {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func cloneStringSet(src map[string]struct{}) map[string]struct{} {
	if len(src) == 0 {
		return make(map[string]struct{})
	}
	clone := make(map[string]struct{}, len(src))
	for k := range src {
		clone[k] = struct{}{}
	}
	return clone
}

func collectGlobalAliasInfo(file *ast.File) (map[string]struct{}, map[string]struct{}) {
	info := binaryFunctionInfo{
		headerAliases:         make(map[string]struct{}),
		responseWriterNames:   make(map[string]struct{}),
		contextNames:          make(map[string]struct{}),
		contentTypeKeyAliases: make(map[string]struct{}),
		dispositionKeyAliases: make(map[string]struct{}),
	}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			recordHeaderKeyAliases(&info, exprSliceFromNames(valueSpec.Names), valueSpec.Values)
		}
	}
	return info.contentTypeKeyAliases, info.dispositionKeyAliases
}

func exprToString(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		prefix := exprToString(v.X)
		if prefix != "" {
			return prefix + "." + v.Sel.Name
		}
		return v.Sel.Name
	case *ast.CallExpr:
		return exprToString(v.Fun)
	case *ast.IndexExpr:
		return exprToString(v.X)
	case *ast.StarExpr:
		return exprToString(v.X)
	case *ast.BasicLit:
		if v.Kind == token.STRING {
			if unquoted, err := strconv.Unquote(v.Value); err == nil {
				return unquoted
			}
		}
		return v.Value
	default:
		return ""
	}
}

func stringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	val, err := strconv.Unquote(lit.Value)
	if err != nil {
		return lit.Value, true
	}
	return val, true
}
