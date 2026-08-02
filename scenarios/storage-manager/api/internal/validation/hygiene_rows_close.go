package validation

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// hygieneRowsClose migrates scenario-auditor's db_rows_close rule
// (scenarios/scenario-auditor/api/rules/api/db_rows_close.go → CheckDBRowsClose).
// It flags a `rows, err := db.Query(...)` (or QueryContext/Queryx/etc.) result
// set that is not guaranteed to be Closed — a leaked cursor that exhausts the
// pool. The flow analysis is ported faithfully from the original: it tracks the
// rows variable through the following statements, accepting defer rows.Close(),
// a return of the rows to the caller, a re-bind, or a recognised cleanup helper
// (by name keyword or by inspecting the helper's body).
//
// Every helper here is prefixed `hygiene` so it does not collide with the
// schema/isolation analyzer tiers in this package; the logic is otherwise a
// line-for-line port of the original so the parity tests reach identical
// verdicts.
//
// New code: DB_ROWS_NOT_CLOSED (ERROR).
type hygieneRowsClose struct{}

func init() { register(&hygieneRowsClose{}) }

func (hygieneRowsClose) Name() string { return "hygiene-rows-close" }

func (hygieneRowsClose) Applies(ac AnalyzerContext) bool { return ac.IsGo() }

func (a hygieneRowsClose) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	var findings []Finding
	for _, gf := range CollectGoFiles(ac) {
		findings = append(findings, a.analyzeSource(ReadFile(gf.AbsPath), gf.RelPath)...)
	}
	return findings, nil
}

var (
	// hygieneQueryFuncNames mirrors db_rows_close.go's queryFuncNames.
	hygieneQueryFuncNames = map[string]struct{}{
		"Query":             {},
		"QueryContext":      {},
		"Queryx":            {},
		"QueryxContext":     {},
		"NamedQuery":        {},
		"NamedQueryContext": {},
	}
	// hygieneCleanupHelperKeywords mirrors db_rows_close.go's cleanupHelperKeywords.
	hygieneCleanupHelperKeywords = []string{"close", "cleanup", "drain", "consume", "release", "finish"}
)

// analyzeSource is the pure detection core, shared by Analyze and the parity
// tests. relPath is the finding Location and gates exemption.
func (a hygieneRowsClose) analyzeSource(source, relPath string) []Finding {
	if !strings.HasSuffix(strings.ToLower(relPath), ".go") {
		return nil
	}
	if hygieneIsExemptPath(relPath) {
		return nil
	}
	if strings.TrimSpace(source) == "" {
		return nil
	}

	fset := token.NewFileSet()
	file, lineOffset, err := hygieneParseGoSource(fset, relPath, source)
	if err != nil {
		return nil
	}

	ctx := &hygieneRowsCtx{
		fset:       fset,
		relPath:    relPath,
		lineOffset: lineOffset,
		funcDecls:  hygieneCollectFuncDecls(file),
		analyzer:   a.Name(),
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		hygieneAnalyzeBlock(fn.Body, ctx, nil)
	}

	return ctx.findings
}

type hygieneRowsCtx struct {
	fset       *token.FileSet
	relPath    string
	lineOffset int
	findings   []Finding
	funcDecls  map[string]*ast.FuncDecl
	analyzer   string
}

func (ctx *hygieneRowsCtx) addFinding(pos token.Pos, rowsVar string) {
	line := ctx.fset.Position(pos).Line - ctx.lineOffset
	if line < 1 {
		line = 1
	}
	loc := ctx.relPath
	if line > 0 {
		loc = ctx.relPath + ":" + strconv.Itoa(line)
	}
	ctx.findings = append(ctx.findings, Finding{
		Code:     "DB_ROWS_NOT_CLOSED",
		Severity: SeverityError,
		Title:    "Database rows not closed",
		Message: "A query result set (" + rowsVar + ") is opened but never guaranteed to be " +
			"closed on every path. A leaked *sql.Rows cursor holds a connection until the " +
			"GC finalizer runs, exhausting the pool under load.",
		Location:    loc,
		Remediation: "Add `defer " + rowsVar + ".Close()` immediately after the error check, or return the rows to a caller that owns closing them.",
		Analyzer:    ctx.analyzer,
	})
}

// hygieneParseGoSource mirrors db_rows_close.go's parseGoSource: parse as-is,
// and on failure retry with a `package main\n` wrapper so bare function
// fragments (the doc-test inputs) parse. Returns the line offset to subtract.
func hygieneParseGoSource(fset *token.FileSet, relPath, src string) (*ast.File, int, error) {
	file, err := parser.ParseFile(fset, relPath, src, parser.ParseComments)
	if err == nil {
		return file, 0, nil
	}
	wrapped := "package main\n" + src
	file, errWrapped := parser.ParseFile(fset, relPath, wrapped, parser.ParseComments)
	if errWrapped == nil {
		return file, 1, nil
	}
	return nil, 0, err
}

func hygieneCollectFuncDecls(file *ast.File) map[string]*ast.FuncDecl {
	decls := make(map[string]*ast.FuncDecl)
	if file == nil {
		return decls
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil {
			continue
		}
		if _, exists := decls[fn.Name.Name]; !exists {
			decls[fn.Name.Name] = fn
		}
	}
	return decls
}

func hygieneAnalyzeBlock(block *ast.BlockStmt, ctx *hygieneRowsCtx, follow []ast.Stmt) {
	if block == nil {
		return
	}
	for idx, stmt := range block.List {
		afterCurrent := hygieneMergeStmts(block.List[idx+1:], follow)
		switch s := stmt.(type) {
		case *ast.AssignStmt:
			if name, pos, ok := hygieneExtractRowsFromAssign(s); ok {
				if !hygieneRowsHandled(ctx, block.List[idx+1:], afterCurrent, name) {
					ctx.addFinding(pos, name)
				}
			}
		case *ast.DeclStmt:
			if name, pos, ok := hygieneExtractRowsFromDecl(s); ok {
				if !hygieneRowsHandled(ctx, block.List[idx+1:], afterCurrent, name) {
					ctx.addFinding(pos, name)
				}
			}
		}
		hygieneAnalyzeStmtChildren(stmt, ctx, afterCurrent)
	}
}

func hygieneAnalyzeStmtChildren(stmt ast.Stmt, ctx *hygieneRowsCtx, follow []ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		hygieneAnalyzeBlock(s, ctx, follow)
	case *ast.IfStmt:
		if s.Init != nil {
			hygieneAnalyzeInitStmt(s.Init, ctx, s.Body, s.Else, follow)
		}
		bodyFollow := hygieneMergeStmts(nil, follow)
		hygieneAnalyzeBlock(s.Body, ctx, bodyFollow)
		hygieneAnalyzeElse(s.Else, ctx, bodyFollow)
	case *ast.ForStmt:
		if s.Init != nil {
			hygieneAnalyzeInitStmt(s.Init, ctx, s.Body, nil, follow)
		}
		hygieneAnalyzeBlock(s.Body, ctx, follow)
	case *ast.RangeStmt:
		if s.Body != nil {
			hygieneAnalyzeBlock(s.Body, ctx, follow)
		}
	case *ast.SwitchStmt:
		if s.Init != nil {
			hygieneAnalyzeInitStmt(s.Init, ctx, nil, nil, follow)
		}
		for _, st := range s.Body.List {
			if clause, ok := st.(*ast.CaseClause); ok {
				hygieneAnalyzeBlock(&ast.BlockStmt{List: clause.Body}, ctx, follow)
			}
		}
	case *ast.TypeSwitchStmt:
		if s.Init != nil {
			hygieneAnalyzeInitStmt(s.Init, ctx, nil, nil, follow)
		}
		for _, st := range s.Body.List {
			if clause, ok := st.(*ast.CaseClause); ok {
				hygieneAnalyzeBlock(&ast.BlockStmt{List: clause.Body}, ctx, follow)
			}
		}
	case *ast.SelectStmt:
		for _, st := range s.Body.List {
			if clause, ok := st.(*ast.CommClause); ok {
				hygieneAnalyzeBlock(&ast.BlockStmt{List: clause.Body}, ctx, follow)
			}
		}
	}
}

func hygieneAnalyzeElse(stmt ast.Stmt, ctx *hygieneRowsCtx, follow []ast.Stmt) {
	switch v := stmt.(type) {
	case *ast.BlockStmt:
		hygieneAnalyzeBlock(v, ctx, follow)
	case *ast.IfStmt:
		hygieneAnalyzeStmtChildren(v, ctx, follow)
	}
}

func hygieneAnalyzeInitStmt(init ast.Stmt, ctx *hygieneRowsCtx, body *ast.BlockStmt, elseStmt ast.Stmt, follow []ast.Stmt) {
	if init == nil {
		return
	}
	if name, pos, ok := hygieneExtractRowsFromStmt(init); ok {
		var handled bool
		if body != nil {
			handled = handled || hygieneRowsHandled(ctx, body.List, hygieneMergeStmts(nil, follow), name)
		}
		if !handled && elseStmt != nil {
			switch e := elseStmt.(type) {
			case *ast.BlockStmt:
				handled = handled || hygieneRowsHandled(ctx, e.List, hygieneMergeStmts(nil, follow), name)
			case *ast.IfStmt:
				handled = handled || hygieneRowsHandled(ctx, e.Body.List, hygieneMergeStmts(nil, follow), name)
			}
		}
		if !handled {
			ctx.addFinding(pos, name)
		}
	}
}

func hygieneExtractRowsFromStmt(stmt ast.Stmt) (string, token.Pos, bool) {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		return hygieneExtractRowsFromAssign(s)
	case *ast.DeclStmt:
		return hygieneExtractRowsFromDecl(s)
	default:
		return "", 0, false
	}
}

func hygieneExtractRowsFromAssign(assign *ast.AssignStmt) (string, token.Pos, bool) {
	if assign == nil || len(assign.Rhs) != 1 {
		return "", 0, false
	}
	if len(assign.Lhs) < 2 {
		return "", 0, false
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok || !hygieneIsQueryCall(call.Fun) {
		return "", 0, false
	}
	if ident := hygieneSelectRowsIdent(assign.Lhs); ident != nil {
		return ident.Name, assign.Pos(), true
	}
	return "", 0, false
}

func hygieneExtractRowsFromDecl(decl *ast.DeclStmt) (string, token.Pos, bool) {
	gen, ok := decl.Decl.(*ast.GenDecl)
	if !ok || gen.Tok != token.VAR {
		return "", 0, false
	}
	for _, spec := range gen.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok || len(valueSpec.Values) != 1 {
			continue
		}
		if len(valueSpec.Names) < 2 {
			continue
		}
		call, ok := valueSpec.Values[0].(*ast.CallExpr)
		if !ok || !hygieneIsQueryCall(call.Fun) {
			continue
		}
		if name := hygieneSelectRowsName(valueSpec.Names); name != "" {
			return name, valueSpec.Pos(), true
		}
	}
	return "", 0, false
}

func hygieneIsQueryCall(fun ast.Expr) bool {
	sel, ok := fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if _, ok = hygieneQueryFuncNames[sel.Sel.Name]; !ok {
		return false
	}
	// Network and model clients commonly expose Query methods that return a
	// payload and an error, not *sql.Rows. We do not have type information in
	// this AST-only pass, but a receiver named *Client is strong evidence that
	// this is not a database cursor and avoids a dangerous false positive.
	return !hygieneReceiverIsClient(sel.X)
}

func hygieneReceiverIsClient(expr ast.Expr) bool {
	switch receiver := expr.(type) {
	case *ast.Ident:
		return strings.Contains(strings.ToLower(receiver.Name), "client")
	case *ast.SelectorExpr:
		return strings.Contains(strings.ToLower(receiver.Sel.Name), "client") || hygieneReceiverIsClient(receiver.X)
	case *ast.ParenExpr:
		return hygieneReceiverIsClient(receiver.X)
	default:
		return false
	}
}

func hygieneRowsHandled(ctx *hygieneRowsCtx, stmts []ast.Stmt, follow []ast.Stmt, rowsName string) bool {
	if rowsName == "" {
		return true
	}
	if handled, stop := hygieneScanStmts(ctx, stmts, rowsName); handled || stop {
		return handled
	}
	if handled, _ := hygieneScanStmts(ctx, follow, rowsName); handled {
		return true
	}
	return false
}

func hygieneStmtProvidesCleanup(ctx *hygieneRowsCtx, stmt ast.Stmt, rowsName string) bool {
	switch s := stmt.(type) {
	case *ast.DeferStmt:
		return hygieneDeferClosesRows(ctx, s, rowsName)
	case *ast.ExprStmt:
		if call, ok := s.X.(*ast.CallExpr); ok {
			return hygieneCallClosesRows(ctx, call, rowsName)
		}
	case *ast.BlockStmt:
		handled, _ := hygieneScanStmts(ctx, s.List, rowsName)
		return handled
	case *ast.IfStmt:
		bodyUses := hygieneBlockUsesRows(s.Body, rowsName)
		elseUses := hygieneClauseUsesRows(s.Else, rowsName)

		if s.Else == nil {
			if !bodyUses {
				return false
			}
			if !hygieneBlockProvidesCleanup(ctx, s.Body, rowsName) {
				return false
			}
			if !hygieneConditionAllowsCleanup(s.Cond, rowsName) {
				return false
			}
			return true
		}

		if !bodyUses && !elseUses {
			return false
		}

		bodyClean := !bodyUses || hygieneBlockProvidesCleanup(ctx, s.Body, rowsName)
		elseClean := !elseUses || hygieneClauseProvidesCleanup(ctx, s.Else, rowsName)
		return bodyClean && elseClean
	case *ast.SwitchStmt:
		return hygieneClausesProvideCleanup(ctx, s.Body.List, rowsName)
	case *ast.TypeSwitchStmt:
		return hygieneClausesProvideCleanup(ctx, s.Body.List, rowsName)
	case *ast.SelectStmt:
		return hygieneCommClausesProvideCleanup(ctx, s.Body.List, rowsName)
	}
	return false
}

func hygieneStmtReturnsRows(stmt ast.Stmt, rowsName string) bool {
	returned := false
	ast.Inspect(stmt, func(n ast.Node) bool {
		if returned {
			return false
		}
		ret, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		for _, result := range ret.Results {
			if hygieneExprIsIdent(result, rowsName) {
				returned = true
				break
			}
		}
		return false
	})
	return returned
}

func hygieneMergeStmts(primary, secondary []ast.Stmt) []ast.Stmt {
	if len(primary) == 0 {
		if len(secondary) == 0 {
			return nil
		}
		return append([]ast.Stmt(nil), secondary...)
	}
	merged := make([]ast.Stmt, 0, len(primary)+len(secondary))
	merged = append(merged, primary...)
	merged = append(merged, secondary...)
	return merged
}

func hygieneScanStmts(ctx *hygieneRowsCtx, stmts []ast.Stmt, rowsName string) (handled, stop bool) {
	for _, stmt := range stmts {
		if hygieneStmtProvidesCleanup(ctx, stmt, rowsName) {
			return true, true
		}
		if hygieneStmtReturnsRows(stmt, rowsName) {
			return true, true
		}
		if hygieneStmtRebindsRows(stmt, rowsName) {
			return false, true
		}
	}
	return false, false
}

func hygieneBlockProvidesCleanup(ctx *hygieneRowsCtx, block *ast.BlockStmt, rowsName string) bool {
	if block == nil {
		return false
	}
	handled, _ := hygieneScanStmts(ctx, block.List, rowsName)
	return handled
}

func hygieneClauseProvidesCleanup(ctx *hygieneRowsCtx, stmt ast.Stmt, rowsName string) bool {
	switch v := stmt.(type) {
	case *ast.BlockStmt:
		return hygieneBlockProvidesCleanup(ctx, v, rowsName)
	case *ast.IfStmt:
		return hygieneStmtProvidesCleanup(ctx, v, rowsName)
	case *ast.CaseClause:
		handled, _ := hygieneScanStmts(ctx, v.Body, rowsName)
		return handled
	case *ast.CommClause:
		handled, _ := hygieneScanStmts(ctx, v.Body, rowsName)
		return handled
	default:
		return hygieneStmtProvidesCleanup(ctx, stmt, rowsName)
	}
}

func hygieneBlockUsesRows(block *ast.BlockStmt, rowsName string) bool {
	if block == nil {
		return false
	}
	for _, stmt := range block.List {
		if hygieneStmtUsesRows(stmt, rowsName) {
			return true
		}
	}
	return false
}

func hygieneClauseUsesRows(stmt ast.Stmt, rowsName string) bool {
	if stmt == nil {
		return false
	}
	return hygieneStmtUsesRows(stmt, rowsName)
}

func hygieneStmtUsesRows(stmt ast.Stmt, rowsName string) bool {
	if stmt == nil || rowsName == "" {
		return false
	}
	found := false
	ast.Inspect(stmt, func(n ast.Node) bool {
		if found {
			return false
		}
		if ident, ok := n.(*ast.Ident); ok && ident.Name == rowsName {
			found = true
			return false
		}
		return true
	})
	return found
}

func hygieneConditionAllowsCleanup(cond ast.Expr, rowsName string) bool {
	ensuresRowsNonNil := false
	ensuresErrNil := false
	ast.Inspect(cond, func(n ast.Node) bool {
		bin, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		switch bin.Op {
		case token.NEQ:
			if (hygieneExprUsesIdent(bin.X, rowsName) && hygieneIsNilIdent(bin.Y)) ||
				(hygieneExprUsesIdent(bin.Y, rowsName) && hygieneIsNilIdent(bin.X)) {
				ensuresRowsNonNil = true
			}
		case token.EQL:
			if (hygieneIdentLooksLikeError(bin.X) && hygieneIsNilIdent(bin.Y)) ||
				(hygieneIdentLooksLikeError(bin.Y) && hygieneIsNilIdent(bin.X)) {
				ensuresErrNil = true
			}
		}
		return true
	})
	return ensuresRowsNonNil || ensuresErrNil
}

func hygieneIsNilIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "nil"
}

func hygieneIdentLooksLikeError(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	return strings.Contains(strings.ToLower(ident.Name), "err")
}

func hygieneClausesProvideCleanup(ctx *hygieneRowsCtx, clauses []ast.Stmt, rowsName string) bool {
	if len(clauses) == 0 {
		return false
	}
	for _, clause := range clauses {
		if !hygieneClauseProvidesCleanup(ctx, clause, rowsName) {
			return false
		}
	}
	return true
}

func hygieneCommClausesProvideCleanup(ctx *hygieneRowsCtx, clauses []ast.Stmt, rowsName string) bool {
	if len(clauses) == 0 {
		return false
	}
	for _, clause := range clauses {
		commClause, ok := clause.(*ast.CommClause)
		if !ok {
			return false
		}
		handled, _ := hygieneScanStmts(ctx, commClause.Body, rowsName)
		if !handled {
			return false
		}
	}
	return true
}

func hygieneStmtRebindsRows(stmt ast.Stmt, rowsName string) bool {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		for _, lhs := range s.Lhs {
			if ident, ok := lhs.(*ast.Ident); ok && ident.Name == rowsName {
				return true
			}
		}
	case *ast.DeclStmt:
		gen, ok := s.Decl.(*ast.GenDecl)
		if !ok {
			return false
		}
		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range valueSpec.Names {
				if name.Name == rowsName {
					return true
				}
			}
		}
	}
	return false
}

func hygieneDeferClosesRows(ctx *hygieneRowsCtx, deferStmt *ast.DeferStmt, rowsName string) bool {
	if deferStmt == nil || deferStmt.Call == nil {
		return false
	}
	call := deferStmt.Call
	if hygieneCallClosesRows(ctx, call, rowsName) {
		return true
	}
	if lit, ok := call.Fun.(*ast.FuncLit); ok {
		return hygieneFuncLiteralClosesRows(lit, call.Args, rowsName)
	}
	return false
}

func hygieneFuncLiteralClosesRows(lit *ast.FuncLit, args []ast.Expr, rowsName string) bool {
	if lit == nil || lit.Body == nil {
		return false
	}
	if hygieneNodeContainsRowsClose(lit.Body, rowsName) {
		return true
	}
	if lit.Type == nil || lit.Type.Params == nil {
		return false
	}
	paramCount := hygieneParametersCount(lit.Type.Params)
	limit := len(args)
	if limit > paramCount {
		limit = paramCount
	}
	for idx := 0; idx < limit; idx++ {
		if !hygieneExprUsesIdent(args[idx], rowsName) {
			continue
		}
		paramName := hygieneParameterNameByIndex(lit.Type.Params, idx)
		if paramName == "" {
			continue
		}
		if hygieneNodeContainsRowsClose(lit.Body, paramName) {
			return true
		}
	}
	return false
}

func hygieneCallClosesRows(ctx *hygieneRowsCtx, call *ast.CallExpr, rowsName string) bool {
	if call == nil {
		return false
	}
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if sel.Sel.Name == "Close" && hygieneExprIsIdent(sel.X, rowsName) {
			return true
		}
	}
	return hygieneHelperLikelyCloses(ctx, call, rowsName)
}

func hygieneHelperLikelyCloses(ctx *hygieneRowsCtx, call *ast.CallExpr, rowsName string) bool {
	if call == nil {
		return false
	}
	argIndex := -1
	for idx, arg := range call.Args {
		if hygieneExprUsesIdent(arg, rowsName) {
			argIndex = idx
			break
		}
	}
	if argIndex == -1 {
		return false
	}
	if ctx != nil {
		switch fun := call.Fun.(type) {
		case *ast.Ident:
			if ctx.helperClosesRows(fun.Name, argIndex) {
				return true
			}
		case *ast.SelectorExpr:
			if ctx.helperClosesRows(fun.Sel.Name, argIndex) {
				return true
			}
		}
	}
	name := strings.ToLower(hygieneCallName(call.Fun))
	if name == "" {
		return false
	}
	for _, keyword := range hygieneCleanupHelperKeywords {
		if strings.Contains(name, keyword) {
			return true
		}
	}
	return false
}

func (ctx *hygieneRowsCtx) helperClosesRows(funcName string, argIndex int) bool {
	if ctx == nil || funcName == "" {
		return false
	}
	decl, ok := ctx.funcDecls[funcName]
	if !ok || decl == nil || decl.Body == nil || decl.Type == nil || decl.Type.Params == nil {
		return false
	}
	paramName := hygieneParameterNameByIndex(decl.Type.Params, argIndex)
	if paramName == "" {
		return false
	}
	return hygieneNodeContainsRowsClose(decl.Body, paramName)
}

func hygieneParameterNameByIndex(fieldList *ast.FieldList, target int) string {
	if fieldList == nil || target < 0 {
		return ""
	}
	index := 0
	for _, field := range fieldList.List {
		if len(field.Names) == 0 {
			if index == target {
				return ""
			}
			index++
			continue
		}
		for _, name := range field.Names {
			if index == target {
				return name.Name
			}
			index++
		}
	}
	return ""
}

func hygieneParametersCount(fieldList *ast.FieldList) int {
	if fieldList == nil {
		return 0
	}
	count := 0
	for _, field := range fieldList.List {
		if len(field.Names) == 0 {
			count++
			continue
		}
		count += len(field.Names)
	}
	return count
}

func hygieneSelectRowsIdent(exprs []ast.Expr) *ast.Ident {
	for _, lhs := range exprs {
		ident, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}
		if ident.Name == "_" {
			continue
		}
		if strings.Contains(strings.ToLower(ident.Name), "err") {
			continue
		}
		return ident
	}
	return nil
}

func hygieneSelectRowsName(idents []*ast.Ident) string {
	for _, ident := range idents {
		if ident == nil {
			continue
		}
		if ident.Name == "" || ident.Name == "_" {
			continue
		}
		if strings.Contains(strings.ToLower(ident.Name), "err") {
			continue
		}
		return ident.Name
	}
	return ""
}

func hygieneCallName(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	default:
		return ""
	}
}

func hygieneNodeContainsRowsClose(node ast.Node, rowsName string) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Close" && hygieneExprIsIdent(sel.X, rowsName) {
			found = true
			return false
		}
		return true
	})
	return found
}

func hygieneExprUsesIdent(expr ast.Expr, name string) bool {
	if expr == nil || name == "" {
		return false
	}
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if found {
			return false
		}
		if ident, ok := n.(*ast.Ident); ok && ident.Name == name {
			found = true
			return false
		}
		return true
	})
	return found
}

func hygieneExprIsIdent(expr ast.Expr, name string) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == name
	}
	if paren, ok := expr.(*ast.ParenExpr); ok {
		return hygieneExprIsIdent(paren.X, name)
	}
	return false
}
