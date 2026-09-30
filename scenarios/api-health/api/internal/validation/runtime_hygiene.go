package validation

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

type RuntimeHygieneResult struct {
	InspectedFiles []string
	Signals        []RuntimeSignal
	Diagnostics    []string
}

type RuntimeSignal struct {
	Kind   string
	Source string
	Detail string
}

func validateRuntimeHygiene(target *Target) []Finding {
	if !target.HasAPIDir {
		return nil
	}
	files, err := productionGoFiles(target.APIDir)
	if err != nil {
		target.Runtime.Diagnostics = append(target.Runtime.Diagnostics, err.Error())
		return []Finding{{
			Severity:    SeverityError,
			Code:        CodeHTTPClientUnbounded,
			Title:       "Runtime hygiene source unreadable",
			Location:    target.APIDir,
			Message:     "API runtime hygiene could not inspect production Go files",
			Remediation: "provide readable Go source files under api/ for runtime hygiene validation",
		}}
	}

	var findings []Finding
	for _, path := range files {
		file, fset, err := parseGoFile(path)
		if err != nil {
			target.Runtime.Diagnostics = append(target.Runtime.Diagnostics, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		target.Runtime.InspectedFiles = append(target.Runtime.InspectedFiles, path)
		analysis := analyzeRuntimeFile(file, fset, path)
		target.Runtime.Signals = append(target.Runtime.Signals, analysis.signals...)
		findings = append(findings, analysis.findings...)
	}
	return findings
}

type runtimeFileAnalysis struct {
	signals  []RuntimeSignal
	findings []Finding
}

func analyzeRuntimeFile(file *ast.File, fset *token.FileSet, path string) runtimeFileAnalysis {
	state := runtimeState{fset: fset, filePath: path}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		state.analyzeFunction(fn)
	}
	return runtimeFileAnalysis{signals: state.signals, findings: state.findings}
}

type runtimeState struct {
	fset     *token.FileSet
	filePath string
	signals  []RuntimeSignal
	findings []Finding
}

func (s *runtimeState) analyzeFunction(fn *ast.FuncDecl) {
	ctxNames := contextParamNames(fn)
	reqNames := requestParamNames(fn)
	respVars := map[string]token.Pos{}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CompositeLit:
			s.inspectHTTPClientLiteral(node)
		case *ast.CallExpr:
			s.inspectCall(node, ctxNames, reqNames, respVars)
		case *ast.AssignStmt:
			s.inspectAssignment(node, respVars)
		case *ast.GoStmt:
			s.inspectGoroutine(node)
		}
		return true
	})

	for resp, pos := range respVars {
		if !functionClosesResponse(fn.Body, resp) {
			s.findings = append(s.findings, Finding{
				Severity:    SeverityWarn,
				Code:        CodeResponseBodyUnclosed,
				Title:       "HTTP response body is not closed",
				Location:    sourceLocation(s.fset, s.filePath, pos),
				Message:     "outbound HTTP response body is read or returned without a local Close",
				Remediation: "close response bodies after successful outbound requests, or transfer ownership through an explicit helper contract",
			})
		}
	}
}

func (s *runtimeState) inspectHTTPClientLiteral(lit *ast.CompositeLit) {
	if !isHTTPClientType(lit.Type) {
		return
	}
	s.signals = append(s.signals, RuntimeSignal{
		Kind:   "http_client",
		Source: sourceLocation(s.fset, s.filePath, lit.Pos()),
		Detail: "custom http.Client literal",
	})
	if httpClientHasTimeout(lit) {
		return
	}
	s.findings = append(s.findings, Finding{
		Severity:    SeverityWarn,
		Code:        CodeHTTPClientUnbounded,
		Title:       "HTTP client has no timeout",
		Location:    sourceLocation(s.fset, s.filePath, lit.Pos()),
		Message:     "http.Client literal does not set Timeout",
		Remediation: "set an explicit Timeout on production API HTTP clients or prove every request has a bounded context",
	})
}

func (s *runtimeState) inspectCall(call *ast.CallExpr, ctxNames, reqNames map[string]bool, respVars map[string]token.Pos) {
	if isDefaultHTTPCall(call) {
		s.findings = append(s.findings, Finding{
			Severity:    SeverityWarn,
			Code:        CodeHTTPClientUnbounded,
			Title:       "Package-level HTTP client is unbounded",
			Location:    sourceLocation(s.fset, s.filePath, call.Pos()),
			Message:     "package-level http.Get, http.Post, or http.Head uses the default client without a request context",
			Remediation: "create a request with context and execute it through an http.Client with a timeout",
		})
	}
	if isNewRequestWithoutContext(call) && len(ctxNames) > 0 {
		s.findings = append(s.findings, Finding{
			Severity:    SeverityWarn,
			Code:        CodeRequestContextDrop,
			Title:       "Request context is dropped",
			Location:    sourceLocation(s.fset, s.filePath, call.Pos()),
			Message:     "API runtime code creates an outbound HTTP request without the available request context",
			Remediation: "use http.NewRequestWithContext with the handler or service context",
		})
	}
	if contextBackgroundCall(call) && len(ctxNames) > 0 {
		s.findings = append(s.findings, Finding{
			Severity:    SeverityWarn,
			Code:        CodeRequestContextDrop,
			Title:       "Request context is dropped",
			Location:    sourceLocation(s.fset, s.filePath, call.Pos()),
			Message:     "API runtime code uses context.Background while a request-scoped context is available",
			Remediation: "propagate the request context or derive a bounded child context",
		})
	}
	if isStructuredLogCall(call) {
		s.signals = append(s.signals, RuntimeSignal{
			Kind:   "structured_logging",
			Source: sourceLocation(s.fset, s.filePath, call.Pos()),
			Detail: selectorName(call.Fun),
		})
	}
	if isUnstructuredLogCall(call) {
		s.findings = append(s.findings, Finding{
			Severity:    SeverityWarn,
			Code:        CodeUnstructuredLogging,
			Title:       "Unstructured API logging",
			Location:    sourceLocation(s.fset, s.filePath, call.Pos()),
			Message:     "API runtime path uses fmt.Print or log.Print style operational logging",
			Remediation: "use structured logging such as slog/zap or a structured scenario logger for operational events",
		})
	}
	if len(reqNames) > 0 && requestContextCall(call, reqNames) {
		s.signals = append(s.signals, RuntimeSignal{
			Kind:   "request_context",
			Source: sourceLocation(s.fset, s.filePath, call.Pos()),
			Detail: "request context propagated",
		})
	}
}

func (s *runtimeState) inspectAssignment(assign *ast.AssignStmt, respVars map[string]token.Pos) {
	if len(assign.Lhs) == 0 || len(assign.Rhs) == 0 {
		return
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok || !isOutboundHTTPDo(call) {
		return
	}
	ident, ok := assign.Lhs[0].(*ast.Ident)
	if !ok || ident.Name == "_" {
		return
	}
	respVars[ident.Name] = assign.Pos()
}

func (s *runtimeState) inspectGoroutine(goStmt *ast.GoStmt) {
	lit, ok := goStmt.Call.Fun.(*ast.FuncLit)
	if !ok || lit.Body == nil {
		return
	}
	if !blockHasLoop(lit.Body) {
		return
	}
	s.signals = append(s.signals, RuntimeSignal{
		Kind:   "goroutine_loop",
		Source: sourceLocation(s.fset, s.filePath, goStmt.Pos()),
		Detail: "long-lived goroutine candidate",
	})
	if blockHasCancellation(lit.Body) || funcLitHasContextParam(lit) {
		return
	}
	s.findings = append(s.findings, Finding{
		Severity:    SeverityWarn,
		Code:        CodeGoroutineUncancelled,
		Title:       "Long-lived goroutine lacks cancellation",
		Location:    sourceLocation(s.fset, s.filePath, goStmt.Pos()),
		Message:     "API-started goroutine contains a loop without context or done-channel cancellation evidence",
		Remediation: "pass context.Context or a done channel and exit the loop when cancellation is signaled",
	})
}

func contextParamNames(fn *ast.FuncDecl) map[string]bool {
	out := map[string]bool{}
	if fn.Type == nil || fn.Type.Params == nil {
		return out
	}
	for _, field := range fn.Type.Params.List {
		isContext := isContextType(field.Type)
		isRequest := isHTTPRequestPointer(field.Type)
		for _, name := range field.Names {
			if isContext {
				out[name.Name] = true
			}
			if isRequest {
				out[name.Name] = true
			}
		}
	}
	return out
}

func requestParamNames(fn *ast.FuncDecl) map[string]bool {
	out := map[string]bool{}
	if fn.Type == nil || fn.Type.Params == nil {
		return out
	}
	for _, field := range fn.Type.Params.List {
		if !isHTTPRequestPointer(field.Type) {
			continue
		}
		for _, name := range field.Names {
			out[name.Name] = true
		}
	}
	return out
}

func isContextType(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && selectorReceiver(sel.X) == "context" && sel.Sel.Name == "Context"
}

func isHTTPRequestPointer(expr ast.Expr) bool {
	ptr, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := ptr.X.(*ast.SelectorExpr)
	return ok && selectorReceiver(sel.X) == "http" && sel.Sel.Name == "Request"
}

func isHTTPClientType(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && selectorReceiver(sel.X) == "http" && sel.Sel.Name == "Client"
}

func httpClientHasTimeout(lit *ast.CompositeLit) bool {
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if ok && key.Name == "Timeout" {
			return true
		}
	}
	return false
}

func isDefaultHTTPCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selectorReceiver(sel.X) != "http" {
		return false
	}
	switch sel.Sel.Name {
	case "Get", "Post", "Head":
		return true
	default:
		return false
	}
}

func isOutboundHTTPDo(call *ast.CallExpr) bool {
	if isDefaultHTTPCall(call) {
		return true
	}
	return selectorMethod(call.Fun, "Do")
}

func isNewRequestWithoutContext(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selectorReceiver(sel.X) == "http" && sel.Sel.Name == "NewRequest"
}

func contextBackgroundCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selectorReceiver(sel.X) == "context" && sel.Sel.Name == "Background"
}

func requestContextCall(call *ast.CallExpr, reqNames map[string]bool) bool {
	if !selectorMethod(call.Fun, "Context") {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && reqNames[ident.Name]
}

func functionClosesResponse(block *ast.BlockStmt, resp string) bool {
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok || !selectorMethod(call.Fun, "Close") {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		bodySel, ok := sel.X.(*ast.SelectorExpr)
		if !ok || bodySel.Sel.Name != "Body" {
			return true
		}
		ident, ok := bodySel.X.(*ast.Ident)
		found = ok && ident.Name == resp
		return !found
	})
	return found
}

func blockHasLoop(block *ast.BlockStmt) bool {
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if found {
			return false
		}
		switch n.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			found = true
		}
		return !found
	})
	return found
}

func blockHasCancellation(block *ast.BlockStmt) bool {
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if found {
			return false
		}
		switch node := n.(type) {
		case *ast.CallExpr:
			found = selectorMethod(node.Fun, "Done")
		case *ast.UnaryExpr:
			if node.Op == token.ARROW {
				text := exprString(node.X)
				found = strings.Contains(text, "ctx") || strings.Contains(text, "done")
			}
		}
		return !found
	})
	return found
}

func funcLitHasContextParam(lit *ast.FuncLit) bool {
	if lit.Type == nil || lit.Type.Params == nil {
		return false
	}
	for _, field := range lit.Type.Params.List {
		if isContextType(field.Type) {
			return true
		}
		for _, name := range field.Names {
			if strings.Contains(strings.ToLower(name.Name), "done") {
				return true
			}
		}
	}
	return false
}

func isStructuredLogCall(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		receiver := selectorReceiver(sel.X)
		if receiver == "slog" || strings.HasSuffix(receiver, ".logger") || strings.Contains(strings.ToLower(receiver), "logger") {
			switch sel.Sel.Name {
			case "Info", "InfoContext", "Warn", "WarnContext", "Error", "ErrorContext", "Debug", "DebugContext":
				return true
			}
		}
	}
	return false
}

func isUnstructuredLogCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	receiver := selectorReceiver(sel.X)
	switch receiver {
	case "fmt":
		return strings.HasPrefix(sel.Sel.Name, "Print")
	case "log":
		return strings.HasPrefix(sel.Sel.Name, "Print")
	default:
		return false
	}
}

func selectorName(expr ast.Expr) string {
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		return selectorReceiver(sel.X) + "." + sel.Sel.Name
	}
	return exprString(expr)
}
