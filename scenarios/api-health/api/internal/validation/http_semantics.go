package validation

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type HTTPSemanticsResult struct {
	InspectedFiles   []string
	Routes           []HTTPRoute
	ResponsePatterns []HTTPResponsePattern
	Diagnostics      []string
}

type HTTPRoute struct {
	Path      string
	Method    string
	Class     string
	Source    string
	Versioned bool
	Exempt    bool
}

type HTTPResponsePattern struct {
	Kind          string
	ContentType   string
	Source        string
	HeaderPresent bool
}

type endpointMetadata struct {
	Path          string
	Method        string
	Category      string
	RestException bool
}

func validateHTTPSemantics(target *Target) []Finding {
	if !target.HasAPIDir {
		return nil
	}

	meta := readEndpointMetadata(target)
	files, err := productionGoFiles(target.APIDir)
	if err != nil {
		target.HTTP.Diagnostics = append(target.HTTP.Diagnostics, err.Error())
		return []Finding{{
			Severity:    SeverityError,
			Code:        CodeUnversionedEndpoint,
			Title:       "HTTP route source unreadable",
			Location:    target.APIDir,
			Message:     "API HTTP semantics could not inspect production Go files",
			Remediation: "provide readable Go source files under api/ for HTTP route and response validation",
		}}
	}

	var findings []Finding
	for _, path := range files {
		file, fset, err := parseGoFile(path)
		if err != nil {
			target.HTTP.Diagnostics = append(target.HTTP.Diagnostics, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		target.HTTP.InspectedFiles = append(target.HTTP.InspectedFiles, path)
		analysis := analyzeHTTPFile(file, fset, path, meta)
		target.HTTP.Routes = append(target.HTTP.Routes, analysis.routes...)
		target.HTTP.ResponsePatterns = append(target.HTTP.ResponsePatterns, analysis.patterns...)
		findings = append(findings, analysis.findings...)
	}
	return findings
}

func readEndpointMetadata(target *Target) map[string]endpointMetadata {
	path := filepath.Join(target.RootPath, ".vrooli", "endpoints.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc struct {
		Endpoints []struct {
			Path          string `json:"path"`
			Method        string `json:"method"`
			Category      string `json:"category"`
			RestException any    `json:"rest_exception"`
		} `json:"endpoints"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		target.HTTP.Diagnostics = append(target.HTTP.Diagnostics, fmt.Sprintf("%s: %v", path, err))
		return nil
	}
	out := map[string]endpointMetadata{}
	for _, ep := range doc.Endpoints {
		if strings.TrimSpace(ep.Path) == "" {
			continue
		}
		out[ep.Path] = endpointMetadata{
			Path:          ep.Path,
			Method:        strings.ToUpper(ep.Method),
			Category:      ep.Category,
			RestException: ep.RestException != nil,
		}
	}
	return out
}

func productionGoFiles(apiDir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(apiDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "testdata" || name == "gen" || name == "vendor" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

type httpFileAnalysis struct {
	routes   []HTTPRoute
	patterns []HTTPResponsePattern
	findings []Finding
}

func analyzeHTTPFile(file *ast.File, fset *token.FileSet, path string, meta map[string]endpointMetadata) httpFileAnalysis {
	versionedRouters := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return true
		}
		ident, ok := assign.Lhs[0].(*ast.Ident)
		if !ok {
			return true
		}
		if callChainHasPrefix(assign.Rhs[0], "PathPrefix", "/api/v") || callChainHasPrefix(assign.Rhs[0], "Group", "/api/v") {
			versionedRouters[ident.Name] = true
		}
		return true
	})

	var out httpFileAnalysis
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if route, ok := routeFromCall(call, fset, path, meta, versionedRouters); ok {
			out.routes = append(out.routes, route)
			if !route.Exempt && !route.Versioned && route.Class == "rest_feature" {
				out.findings = append(out.findings, Finding{
					Severity:    SeverityWarn,
					Code:        CodeUnversionedEndpoint,
					Title:       "Unversioned REST feature endpoint",
					Location:    route.Source,
					Message:     fmt.Sprintf("REST feature endpoint %q is not under /api/vN", route.Path),
					Remediation: "version public REST feature routes, or declare a rest_exception when the route is intentionally unversioned",
				})
			}
		}
		return true
	})

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		funcOut := analyzeHTTPFunction(fn, fset, path)
		out.patterns = append(out.patterns, funcOut.patterns...)
		out.findings = append(out.findings, funcOut.findings...)
	}
	return out
}

func callChainHasPrefix(expr ast.Expr, method, prefix string) bool {
	if callPathPrefix(expr, method, prefix) {
		return true
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return callChainHasPrefix(sel.X, method, prefix)
}

func callPathPrefix(expr ast.Expr, method, prefix string) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok || !selectorMethod(call.Fun, method) || len(call.Args) == 0 {
		return false
	}
	value, ok := stringLiteral(call.Args[0])
	return ok && strings.HasPrefix(value, prefix)
}

func routeFromCall(call *ast.CallExpr, fset *token.FileSet, filePath string, meta map[string]endpointMetadata, versionedRouters map[string]bool) (HTTPRoute, bool) {
	if len(call.Args) == 0 {
		return HTTPRoute{}, false
	}
	path, ok := stringLiteral(call.Args[0])
	if !ok || !strings.HasPrefix(path, "/") {
		return HTTPRoute{}, false
	}

	method := ""
	receiver := ""
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		method = strings.ToUpper(sel.Sel.Name)
		receiver = selectorReceiver(sel.X)
	}
	if method == "" {
		return HTTPRoute{}, false
	}

	isRoute := false
	switch method {
	case "HANDLE", "HANDLEFUNC", "PATHPREFIX":
		isRoute = true
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		isRoute = true
	default:
		return HTTPRoute{}, false
	}
	if !isRoute {
		return HTTPRoute{}, false
	}

	source := sourceLocation(fset, filePath, call.Pos())
	route := HTTPRoute{
		Path:      path,
		Method:    method,
		Source:    source,
		Class:     classifyRoute(path, meta[path]),
		Versioned: strings.HasPrefix(path, "/api/v") || versionedRouters[receiver],
	}
	route.Exempt = route.Class != "rest_feature"
	return route, true
}

func classifyRoute(path string, meta endpointMetadata) string {
	if meta.RestException || meta.Category == "system" || path == "/" || strings.HasPrefix(path, "/debug/pprof/") || path == "/health" || path == "/metrics" || path == "/ready" || path == "/live" {
		return "ops_probe"
	}
	if strings.Contains(path, ".") && strings.Contains(path, "/") {
		return "connect"
	}
	if strings.HasPrefix(path, "/static/") || strings.HasPrefix(path, "/assets/") || strings.HasPrefix(path, "/favicon") {
		return "static_asset"
	}
	if strings.HasPrefix(path, "/api/v") {
		return "rest_feature"
	}
	return "rest_feature"
}

type httpFunctionAnalysis struct {
	patterns []HTTPResponsePattern
	findings []Finding
}

func analyzeHTTPFunction(fn *ast.FuncDecl, fset *token.FileSet, filePath string) httpFunctionAnalysis {
	writers := responseWriterNames(fn)
	if len(writers) == 0 {
		return httpFunctionAnalysis{}
	}
	state := functionHTTPState{writers: writers, headers: collectFunctionContentTypes(fn, writers), fset: fset, filePath: filePath}
	state.inspectBlock(fn.Body, false, false)
	return httpFunctionAnalysis{patterns: state.patterns, findings: state.findings}
}

type functionHTTPState struct {
	writers  map[string]bool
	headers  map[string]string
	fset     *token.FileSet
	filePath string
	patterns []HTTPResponsePattern
	findings []Finding
}

func (s *functionHTTPState) inspectBlock(block *ast.BlockStmt, inErrorBranch bool, errorBranchHasStatus bool) {
	if block == nil {
		return
	}
	for _, stmt := range block.List {
		ifStmt, ok := stmt.(*ast.IfStmt)
		if ok {
			errBranch := isErrorCondition(ifStmt.Cond) || (inErrorBranch && !isErrorRecoveryCondition(ifStmt.Cond))
			s.inspectBlock(ifStmt.Body, errBranch, errBranch && blockHasStatus(ifStmt.Body))
			if ifStmt.Else != nil {
				if elseBlock, ok := ifStmt.Else.(*ast.BlockStmt); ok {
					s.inspectBlock(elseBlock, inErrorBranch, inErrorBranch && blockHasStatus(elseBlock))
				}
			}
			continue
		}
		ast.Inspect(stmt, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			s.inspectCall(call, inErrorBranch, errorBranchHasStatus)
			return true
		})
	}
}

func (s *functionHTTPState) inspectCall(call *ast.CallExpr, inErrorBranch bool, errorBranchHasStatus bool) {
	if finding := rawStatusFinding(call, s.fset, s.filePath); finding != nil {
		s.findings = append(s.findings, *finding)
	}
	if inErrorBranch && statusOKCall(call) {
		s.findings = append(s.findings, Finding{
			Severity:    SeverityError,
			Code:        CodeImplicitErrorSuccess,
			Title:       "Error response uses success status",
			Location:    sourceLocation(s.fset, s.filePath, call.Pos()),
			Message:     "an error branch writes HTTP 200/StatusOK",
			Remediation: "use an explicit 4xx or 5xx status for error responses",
		})
	}
	if kind, writer, ok := responsePattern(call, s.writers); ok {
		contentType, headerPresent := s.contentTypeFor(writer)
		s.patterns = append(s.patterns, HTTPResponsePattern{
			Kind:          kind,
			ContentType:   contentType,
			Source:        sourceLocation(s.fset, s.filePath, call.Pos()),
			HeaderPresent: headerPresent,
		})
		if requiresContentType(kind) && !headerPresent {
			s.findings = append(s.findings, Finding{
				Severity:    SeverityWarn,
				Code:        CodeContentTypeMissing,
				Title:       "Response content type missing",
				Location:    sourceLocation(s.fset, s.filePath, call.Pos()),
				Message:     fmt.Sprintf("%s response writes a body without an explicit Content-Type header", kind),
				Remediation: "set Content-Type before writing obvious JSON, text, CSV, PDF, XML, binary, or event-stream responses",
			})
		}
		if inErrorBranch && !errorBranchHasStatus && kind == "json" {
			s.findings = append(s.findings, Finding{
				Severity:    SeverityError,
				Code:        CodeImplicitErrorSuccess,
				Title:       "Error response omits status",
				Location:    sourceLocation(s.fset, s.filePath, call.Pos()),
				Message:     "an error branch encodes JSON without WriteHeader or http.Error, so stdlib net/http will return HTTP 200",
				Remediation: "write an explicit 4xx or 5xx status before encoding the error response",
			})
		}
	}
}

func responseWriterNames(fn *ast.FuncDecl) map[string]bool {
	out := map[string]bool{}
	if fn.Type == nil || fn.Type.Params == nil {
		return out
	}
	for _, field := range fn.Type.Params.List {
		if !isHTTPResponseWriter(field.Type) {
			continue
		}
		for _, name := range field.Names {
			out[name.Name] = true
		}
	}
	return out
}

func isHTTPResponseWriter(expr ast.Expr) bool {
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		return selectorReceiver(sel.X) == "http" && sel.Sel.Name == "ResponseWriter"
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "ResponseWriter"
	}
	return false
}

func collectFunctionContentTypes(fn *ast.FuncDecl, writers map[string]bool) map[string]string {
	headers := map[string]string{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		if !selectorMethod(call.Fun, "Set") && !selectorMethod(call.Fun, "Add") {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		headerCall, ok := sel.X.(*ast.CallExpr)
		if !ok || !selectorMethod(headerCall.Fun, "Header") {
			return true
		}
		headerSel, ok := headerCall.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		writer, ok := headerSel.X.(*ast.Ident)
		if !ok || !writers[writer.Name] {
			return true
		}
		headerName, ok := stringLiteral(call.Args[0])
		if !ok || !strings.EqualFold(headerName, "Content-Type") {
			return true
		}
		headerValue, ok := stringLiteral(call.Args[1])
		if !ok {
			return true
		}
		headers[writer.Name] = headerValue
		return true
	})
	return headers
}

func rawStatusFinding(call *ast.CallExpr, fset *token.FileSet, filePath string) *Finding {
	var idx int
	switch {
	case selectorMethod(call.Fun, "WriteHeader"):
		idx = 0
	case isHTTPErrorCall(call):
		idx = 2
	case selectorMethod(call.Fun, "JSON"), selectorMethod(call.Fun, "IndentedJSON"):
		idx = 0
	default:
		return nil
	}
	if idx >= len(call.Args) {
		return nil
	}
	lit, ok := call.Args[idx].(*ast.BasicLit)
	if !ok || lit.Kind != token.INT || len(lit.Value) != 3 {
		return nil
	}
	return &Finding{
		Severity:    SeverityWarn,
		Code:        CodeRawStatusCode,
		Title:       "Raw HTTP status code",
		Location:    sourceLocation(fset, filePath, lit.Pos()),
		Message:     "HTTP response uses raw numeric status " + lit.Value,
		Remediation: "use the corresponding net/http Status* constant",
	}
}

func statusOKCall(call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.INT && lit.Value == "200" {
			return true
		}
		if selector, ok := arg.(*ast.SelectorExpr); ok && selector.Sel.Name == "StatusOK" {
			return true
		}
	}
	return false
}

func responsePattern(call *ast.CallExpr, writers map[string]bool) (string, string, bool) {
	if selectorMethod(call.Fun, "Encode") {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if encCall, ok := sel.X.(*ast.CallExpr); ok && selectorMethod(encCall.Fun, "NewEncoder") && len(encCall.Args) > 0 {
				if ident, ok := encCall.Args[0].(*ast.Ident); ok && writers[ident.Name] {
					return encoderKind(encCall), ident.Name, true
				}
			}
		}
	}
	if selectorMethod(call.Fun, "Write") && len(call.Args) > 0 {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok && writers[ident.Name] {
				if kind := writeKind(call.Args[0]); kind != "" {
					return kind, ident.Name, true
				}
			}
		}
	}
	return "", "", false
}

func encoderKind(call *ast.CallExpr) string {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		switch selectorReceiver(sel.X) {
		case "json":
			return "json"
		case "xml":
			return "xml"
		}
	}
	return "encoded"
}

func writeKind(arg ast.Expr) string {
	if call, ok := arg.(*ast.CallExpr); ok && len(call.Args) == 1 {
		if _, ok := call.Fun.(*ast.ArrayType); ok {
			arg = call.Args[0]
		}
	}
	value, ok := stringLiteral(arg)
	if !ok {
		return ""
	}
	trimmed := strings.TrimSpace(value)
	switch {
	case strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "["):
		return "json"
	case strings.HasPrefix(strings.ToLower(trimmed), "<!doctype html") || strings.HasPrefix(strings.ToLower(trimmed), "<html"):
		return "html"
	case strings.HasPrefix(strings.ToLower(trimmed), "<?xml") || strings.HasPrefix(strings.ToLower(trimmed), "<rss"):
		return "xml"
	default:
		return "text"
	}
}

func (s *functionHTTPState) contentTypeFor(writer string) (string, bool) {
	contentType := strings.TrimSpace(s.headers[writer])
	return contentType, contentType != ""
}

func requiresContentType(kind string) bool {
	switch kind {
	case "json", "html", "xml", "text", "csv", "pdf", "binary", "event_stream":
		return true
	default:
		return false
	}
}

func blockHasStatus(block *ast.BlockStmt) bool {
	if block == nil {
		return false
	}
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		found = isHTTPErrorCall(call) || selectorMethod(call.Fun, "WriteHeader")
		return !found
	})
	return found
}

func isHTTPErrorCall(call *ast.CallExpr) bool {
	if !selectorMethod(call.Fun, "Error") {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selectorReceiver(sel.X) == "http"
}

func isErrorCondition(expr ast.Expr) bool {
	text := exprString(expr)
	// A direct nil-check identifies an unhandled error path. errors.Is/As
	// instead classify a known error and may intentionally select a recovery
	// response (for example a preview fallback), so treating them as failures
	// creates false "HTTP 200 on error" findings.
	return strings.Contains(text, "err != nil")
}

// isErrorRecoveryCondition recognizes a branch that classifies a known error
// value. It may legitimately return a cached, partial, or fallback success
// response, so it stops inherited "unhandled error" status tracking.
func isErrorRecoveryCondition(expr ast.Expr) bool {
	text := exprString(expr)
	return strings.Contains(text, "errors.Is") || strings.Contains(text, "errors.As")
}

func exprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		return exprString(e.X) + " " + e.Op.String() + " " + exprString(e.Y)
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprString(e.X) + "." + e.Sel.Name
	case *ast.CallExpr:
		return exprString(e.Fun)
	case *ast.BasicLit:
		return e.Value
	default:
		return ""
	}
}

func selectorMethod(expr ast.Expr, method string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == method
}

func selectorReceiver(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return selectorReceiver(e.X) + "." + e.Sel.Name
	default:
		return ""
	}
}

func stringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return value, true
}

func sourceLocation(fset *token.FileSet, filePath string, pos token.Pos) string {
	line := fset.Position(pos).Line
	if line <= 0 {
		return filePath
	}
	return fmt.Sprintf("%s:%d", filePath, line)
}
