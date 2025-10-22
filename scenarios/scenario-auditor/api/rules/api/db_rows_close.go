package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

/*
Rule: Database Rows Close
Description: Ensure database query result sets are closed
Reason: Prevents exhausting database connections by leaking result cursors
Category: api
Severity: high
Standard: database-v1
Targets: api

<test-case id="rows-not-closed" should-fail="true">
  <description>Query rows without a corresponding Close call</description>
  <input language="go"><![CDATA[
func listUsers(db *sql.DB) ([]User, error) {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return nil, err
    }
    var out []User
    for rows.Next() {
        var u User
        rows.Scan(&u.ID)
        out = append(out, u)
    }
    return out, nil
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Database Rows Not Closed</expected-message>
</test-case>

<test-case id="rows-closed" should-fail="false">
  <description>Query rows with defer Close immediately after the error check</description>
  <input language="go"><![CDATA[
func listUsersSafely(db *sql.DB) ([]User, error) {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []User
    for rows.Next() {
        var u User
        rows.Scan(&u.ID)
        out = append(out, u)
    }
    return out, rows.Err()
}
]]></input>
</test-case>

<test-case id="rows-returned" should-fail="false">
  <description>Helper returns rows to caller who owns the Close responsibility</description>
  <input language="go"><![CDATA[
func listUsersRaw(db *sql.DB) (*sql.Rows, error) {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return nil, err
    }
    return rows, nil
}
]]></input>
</test-case>

<test-case id="rows-blank-identifier" should-fail="false">
  <description>Blank identifier discard should not trigger Close enforcement</description>
  <input language="go"><![CDATA[
func countUsers(db *sql.DB) error {
    _, err := db.Query("SELECT id FROM users")
    return err
}
]]></input>
</test-case>

<test-case id="rows-helper-close" should-fail="false">
  <description>Custom helper used to close rows should be accepted</description>
  <input language="go"><![CDATA[
func listUsersWithHelper(db *sql.DB) ([]User, error) {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return nil, err
    }
    defer closeRows(rows)
    return consume(rows)
}

func closeRows(rows *sql.Rows) {
    rows.Close()
}

func consume(rows *sql.Rows) ([]User, error) {
    return nil, nil
}
]]></input>
</test-case>

<test-case id="rows-helper-consume" should-fail="false">
  <description>Helper without close keyword in name still closes rows</description>
  <input language="go"><![CDATA[
func listUsersWithConsumer(db *sql.DB) ([]User, error) {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return nil, err
    }
    defer drainRows(rows)
    return consume(rows)
}

func drainRows(rows *sql.Rows) {
    rows.Close()
}

func consume(rows *sql.Rows) ([]User, error) {
    return nil, nil
}
]]></input>
</test-case>

<test-case id="rows-else-branch-close" should-fail="false">
  <description>Error branch returns early, else branch defers Close</description>
  <input language="go"><![CDATA[
func listUsersConditional(db *sql.DB) ([]User, error) {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return nil, err
    } else {
        defer rows.Close()
        for rows.Next() {
            var id int
            rows.Scan(&id)
        }
        return nil, rows.Err()
    }
}
]]></input>
</test-case>

<test-case id="rows-if-cleanup" should-fail="false">
  <description>Cleanup inside if err == nil branch without else</description>
  <input language="go"><![CDATA[
func listUsersIfSafe(db *sql.DB) error {
    rows, err := db.Query("SELECT id FROM users")
    if err == nil {
        defer rows.Close()
        for rows.Next() {
            var id int
            rows.Scan(&id)
        }
        return rows.Err()
    }
    return err
}
]]></input>
</test-case>

<test-case id="rows-branch-assignment" should-fail="false">
  <description>Rows reassigned inside branches but closed after the conditional</description>
  <input language="go"><![CDATA[
func branchClose(db *sql.DB, stats bool) error {
    var rows *sql.Rows
    var err error

    if stats {
        rows, err = db.Query("SELECT 1")
    } else {
        rows, err = db.Query("SELECT 2")
    }
    if err != nil {
        return err
    }
    defer rows.Close()
    return nil
}
]]></input>
</test-case>

<test-case id="rows-conditional-defer" should-fail="true">
  <description>Conditional defer should still require guaranteed close</description>
  <input language="go"><![CDATA[
func conditionalClose(db *sql.DB, enabled bool) error {
    rows, err := db.Query("SELECT 1")
    if err != nil {
        return err
    }
    if enabled {
        defer rows.Close()
    }
    return nil
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Database Rows Not Closed</expected-message>
</test-case>

<test-case id="rows-guarded-and" should-fail="false">
  <description>Rows closed within guard that checks err and rows</description>
  <input language="go"><![CDATA[
func guardedClose(db *sql.DB) (map[string]int, error) {
    rows, err := db.Query("SELECT 1")
    counts := map[string]int{}
    if err == nil && rows != nil {
        defer rows.Close()
        for rows.Next() {
            var key string
            var value int
            if err := rows.Scan(&key, &value); err == nil {
                counts[key] = value
            }
        }
    }
    return counts, err
}
]]></input>
</test-case>

<test-case id="rows-non-row-name-leak" should-fail="true">
  <description>Non-row identifier must still be closed</description>
  <input language="go"><![CDATA[
func leakedCursor(db *sql.DB) error {
    cursor, err := db.Query("SELECT 1")
    if err != nil {
        return err
    }
    return cursor.Err()
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Database Rows Not Closed</expected-message>
</test-case>

<test-case id="rows-non-row-name-closed" should-fail="false">
  <description>Non-row identifier closed properly</description>
  <input language="go"><![CDATA[
func closedCursor(db *sql.DB) error {
    cursor, err := db.Query("SELECT 1")
    if err != nil {
        return err
    }
    defer cursor.Close()
    return cursor.Err()
}
]]></input>
</test-case>

<test-case id="rows-helper-body-scan" should-fail="false">
  <description>Helper without keyword closes rows internally</description>
  <input language="go"><![CDATA[
func withHelper(db *sql.DB) error {
    rows, err := db.Query("SELECT 1")
    if err != nil {
        return err
    }
    defer handleRows(rows)
    return nil
}

func handleRows(r *sql.Rows) {
    if r != nil {
        r.Close()
    }
}
]]></input>
</test-case>

<test-case id="rows-queryx-context" should-fail="true">
  <description>sqlx-style QueryxContext without close should fail</description>
  <input language="go"><![CDATA[
func leakQueryx(db interface{ QueryxContext(interface{}, string, ...interface{}) (*sql.Rows, error) }) error {
    rows, err := db.QueryxContext(nil, "SELECT 1")
    if err != nil {
        return err
    }
    return rows.Err()
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Database Rows Not Closed</expected-message>
</test-case>

<test-case id="rows-struct-selector" should-fail="true">
  <description>Rows fetched via struct field without close</description>
  <input language="go"><![CDATA[
func structLeak(proc *Processor) error {
    rows, err := proc.db.Query("SELECT id FROM users")
    if err != nil {
        return err
    }
    return nil
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Database Rows Not Closed</expected-message>
</test-case>

<test-case id="rows-multiline-call" should-fail="true">
  <description>Multiline selector call without close</description>
  <input language="go"><![CDATA[
func multilineLeak(db *sql.DB) error {
    rows, err := db.
        Query("SELECT id FROM users")
    if err != nil {
        return err
    }
    return nil
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Database Rows Not Closed</expected-message>
</test-case>

<test-case id="rows-comment" should-fail="false">
  <description>Comment containing query should be ignored</description>
  <input language="go"><![CDATA[
func commentOnly() {
    // rows, err := db.Query("SELECT 1")
}
]]></input>
</test-case>

<test-case id="rows-defer-nonclosing" should-fail="true">
  <description>Deferred helper without closing semantics should fail</description>
  <input language="go"><![CDATA[
func leakWithLogger(db *sql.DB) error {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return err
    }
    defer logRows(rows)
    return nil
}

func logRows(rows *sql.Rows) {
    _ = rows
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Database Rows Not Closed</expected-message>
</test-case>

<test-case id="http-request-query" should-fail="false">
  <description>HTTP URL query helpers should not require Close</description>
  <input language="go"><![CDATA[
func inspectQuery(req *http.Request) {
    q := req.URL.Query()
    _ = q.Get("id")
}
]]></input>
</test-case>

<test-case id="rows-nested-scope-leak" should-fail="true">
  <description>Nested scope redeclares rows without closing it</description>
  <input language="go"><![CDATA[
func usersNested(db *sql.DB, include bool) error {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return err
    }
    defer rows.Close()

    if include {
        rows, err := db.Query("SELECT id FROM admins")
        if err != nil {
            return err
        }
        for rows.Next() {
            var id int
            rows.Scan(&id)
        }
    }
    return nil
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Database Rows Not Closed</expected-message>
</test-case>
*/

var (
	queryFuncNames = map[string]struct{}{
		"Query":             {},
		"QueryContext":      {},
		"Queryx":            {},
		"QueryxContext":     {},
		"NamedQuery":        {},
		"NamedQueryContext": {},
	}
	cleanupHelperKeywords = []string{"close", "cleanup", "drain", "consume", "release", "finish"}
)

// CheckDBRowsClose ensures sql.Rows results are properly closed.
func CheckDBRowsClose(content []byte, filePath string) []Violation {
	if !strings.HasSuffix(strings.ToLower(filePath), ".go") {
		return nil
	}
	if isTestFile(filePath) {
		return nil
	}

	source := string(content)
	if strings.TrimSpace(source) == "" {
		return nil
	}

	lines := strings.Split(source, "\n")
	fset := token.NewFileSet()
	file, lineOffset, err := parseGoSource(fset, filePath, source)
	if err != nil {
		return nil
	}

	ctx := &rowsRuleContext{
		fset:       fset,
		filePath:   filePath,
		lines:      lines,
		lineOffset: lineOffset,
		funcDecls:  collectFuncDecls(file),
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		analyzeRowsBlock(fn.Body, ctx, nil)
	}

	return ctx.violations
}

type rowsRuleContext struct {
	fset       *token.FileSet
	filePath   string
	lines      []string
	lineOffset int
	violations []Violation
	funcDecls  map[string]*ast.FuncDecl
}

func (ctx *rowsRuleContext) addViolation(pos token.Pos, rowsVar string) {
	line := ctx.fset.Position(pos).Line - ctx.lineOffset
	if line < 1 {
		line = 1
	}
	if line > len(ctx.lines) {
		line = len(ctx.lines)
	}
	snippet := ""
	if line-1 >= 0 && line-1 < len(ctx.lines) {
		snippet = ctx.lines[line-1]
	}

	ctx.violations = append(ctx.violations, Violation{
		Type:           "db_rows_close",
		Severity:       "high",
		Title:          "Database Rows Not Closed",
		Description:    "Database Rows Not Closed",
		FilePath:       ctx.filePath,
		LineNumber:     line,
		CodeSnippet:    snippet,
		Recommendation: "Add defer " + rowsVar + ".Close() after error check",
		Standard:       "database-v1",
	})
}

func parseGoSource(fset *token.FileSet, filePath, src string) (*ast.File, int, error) {
	file, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err == nil {
		return file, 0, nil
	}
	wrapped := "package main\n" + src
	file, errWrapped := parser.ParseFile(fset, filePath, wrapped, parser.ParseComments)
	if errWrapped == nil {
		return file, 1, nil
	}
	return nil, 0, err
}

func collectFuncDecls(file *ast.File) map[string]*ast.FuncDecl {
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

func analyzeRowsBlock(block *ast.BlockStmt, ctx *rowsRuleContext, follow []ast.Stmt) {
	if block == nil {
		return
	}

	for idx, stmt := range block.List {
		afterCurrent := mergeStatementSlices(block.List[idx+1:], follow)
		switch s := stmt.(type) {
		case *ast.AssignStmt:
			if name, pos, ok := extractRowsFromAssign(s); ok {
				if !rowsHandled(ctx, block.List[idx+1:], afterCurrent, name) {
					ctx.addViolation(pos, name)
				}
			}
		case *ast.DeclStmt:
			if name, pos, ok := extractRowsFromDecl(s); ok {
				if !rowsHandled(ctx, block.List[idx+1:], afterCurrent, name) {
					ctx.addViolation(pos, name)
				}
			}
		}

		analyzeStmtChildren(stmt, ctx, afterCurrent)
	}
}

func analyzeStmtChildren(stmt ast.Stmt, ctx *rowsRuleContext, follow []ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		analyzeRowsBlock(s, ctx, follow)
	case *ast.IfStmt:
		if s.Init != nil {
			analyzeRowsInitStmt(s.Init, ctx, s.Body, s.Else, follow)
		}
		bodyFollow := mergeStatementSlices(nil, follow)
		analyzeRowsBlock(s.Body, ctx, bodyFollow)
		analyzeElseClause(s.Else, ctx, bodyFollow)
	case *ast.ForStmt:
		if s.Init != nil {
			analyzeRowsInitStmt(s.Init, ctx, s.Body, nil, follow)
		}
		analyzeRowsBlock(s.Body, ctx, follow)
	case *ast.RangeStmt:
		analyzeRangeStmt(s, ctx, follow)
	case *ast.SwitchStmt:
		if s.Init != nil {
			analyzeRowsInitStmt(s.Init, ctx, nil, nil, follow)
		}
		for _, stmt := range s.Body.List {
			if clause, ok := stmt.(*ast.CaseClause); ok {
				analyzeRowsBlock(&ast.BlockStmt{List: clause.Body}, ctx, follow)
			}
		}
	case *ast.TypeSwitchStmt:
		if s.Init != nil {
			analyzeRowsInitStmt(s.Init, ctx, nil, nil, follow)
		}
		for _, stmt := range s.Body.List {
			if clause, ok := stmt.(*ast.CaseClause); ok {
				analyzeRowsBlock(&ast.BlockStmt{List: clause.Body}, ctx, follow)
			}
		}
	case *ast.SelectStmt:
		for _, stmt := range s.Body.List {
			if clause, ok := stmt.(*ast.CommClause); ok {
				analyzeRowsBlock(&ast.BlockStmt{List: clause.Body}, ctx, follow)
			}
		}
	}
}

func analyzeElseClause(stmt ast.Stmt, ctx *rowsRuleContext, follow []ast.Stmt) {
	switch v := stmt.(type) {
	case *ast.BlockStmt:
		analyzeRowsBlock(v, ctx, follow)
	case *ast.IfStmt:
		analyzeStmtChildren(v, ctx, follow)
	}
}

func analyzeRangeStmt(stmt *ast.RangeStmt, ctx *rowsRuleContext, follow []ast.Stmt) {
	if stmt == nil {
		return
	}
	if stmt.Key != nil {
		if ident, ok := stmt.Key.(*ast.Ident); ok {
			if ident.Name != "_" {
				// noop for now
			}
		}
	}
	if stmt.Body != nil {
		analyzeRowsBlock(stmt.Body, ctx, follow)
	}
}

func analyzeRowsInitStmt(init ast.Stmt, ctx *rowsRuleContext, body *ast.BlockStmt, elseStmt ast.Stmt, follow []ast.Stmt) {
	if init == nil {
		return
	}
	if name, pos, ok := extractRowsFromStmt(init); ok {
		var handled bool
		if body != nil {
			handled = handled || rowsHandled(ctx, body.List, mergeStatementSlices(nil, follow), name)
		}
		if !handled && elseStmt != nil {
			switch e := elseStmt.(type) {
			case *ast.BlockStmt:
				handled = handled || rowsHandled(ctx, e.List, mergeStatementSlices(nil, follow), name)
			case *ast.IfStmt:
				handled = handled || rowsHandled(ctx, e.Body.List, mergeStatementSlices(nil, follow), name)
			}
		}
		if !handled {
			ctx.addViolation(pos, name)
		}
	}
}

func extractRowsFromStmt(stmt ast.Stmt) (string, token.Pos, bool) {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		return extractRowsFromAssign(s)
	case *ast.DeclStmt:
		return extractRowsFromDecl(s)
	default:
		return "", 0, false
	}
}

func extractRowsFromAssign(assign *ast.AssignStmt) (string, token.Pos, bool) {
	if assign == nil || len(assign.Rhs) != 1 {
		return "", 0, false
	}
	if len(assign.Lhs) < 2 {
		return "", 0, false
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok || !isQueryCall(call.Fun) {
		return "", 0, false
	}
	if ident := selectRowsIdent(assign.Lhs); ident != nil {
		return ident.Name, assign.Pos(), true
	}
	return "", 0, false
}

func extractRowsFromDecl(decl *ast.DeclStmt) (string, token.Pos, bool) {
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
		if !ok || !isQueryCall(call.Fun) {
			continue
		}
		if ident := selectRowsName(valueSpec.Names); ident != "" {
			return ident, valueSpec.Pos(), true
		}
	}
	return "", 0, false
}

func isQueryCall(fun ast.Expr) bool {
	sel, ok := fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if _, ok := queryFuncNames[sel.Sel.Name]; !ok {
		return false
	}
	return true
}

func rowsHandled(ctx *rowsRuleContext, stmts []ast.Stmt, follow []ast.Stmt, rowsName string) bool {
	if rowsName == "" {
		return true
	}
	if handled, stop := scanStatements(ctx, stmts, rowsName); handled || stop {
		return handled
	}
	if handled, _ := scanStatements(ctx, follow, rowsName); handled {
		return true
	}
	return false
}

func statementProvidesCleanup(ctx *rowsRuleContext, stmt ast.Stmt, rowsName string) bool {
	switch s := stmt.(type) {
	case *ast.DeferStmt:
		return deferClosesRows(ctx, s, rowsName)
	case *ast.ExprStmt:
		if call, ok := s.X.(*ast.CallExpr); ok {
			return callClosesRows(ctx, call, rowsName)
		}
	case *ast.BlockStmt:
		handled, _ := scanStatements(ctx, s.List, rowsName)
		return handled
	case *ast.IfStmt:
		bodyUses := blockUsesRows(s.Body, rowsName)
		elseUses := clauseUsesRows(s.Else, rowsName)

		if s.Else == nil {
			if !bodyUses {
				return false
			}
			if !blockProvidesCleanup(ctx, s.Body, rowsName) {
				return false
			}
			if blockTerminates(s.Body) || conditionAllowsCleanup(s.Cond, rowsName) {
				return true
			}
			return false
		}

		if !bodyUses && !elseUses {
			return false
		}

		bodyClean := !bodyUses || blockProvidesCleanup(ctx, s.Body, rowsName)
		elseClean := !elseUses || clauseProvidesCleanup(ctx, s.Else, rowsName)
		return bodyClean && elseClean
	case *ast.SwitchStmt:
		return clausesProvideCleanup(ctx, s.Body.List, rowsName)
	case *ast.TypeSwitchStmt:
		return clausesProvideCleanup(ctx, s.Body.List, rowsName)
	case *ast.SelectStmt:
		return commClausesProvideCleanup(ctx, s.Body.List, rowsName)
	}
	return false
}

func statementReturnsRows(stmt ast.Stmt, rowsName string) bool {
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
			if exprIsIdent(result, rowsName) {
				returned = true
				break
			}
		}
		return false
	})
	return returned
}

func mergeStatementSlices(primary []ast.Stmt, secondary []ast.Stmt) []ast.Stmt {
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

func scanStatements(ctx *rowsRuleContext, stmts []ast.Stmt, rowsName string) (handled bool, stop bool) {
	for _, stmt := range stmts {
		if statementProvidesCleanup(ctx, stmt, rowsName) {
			return true, true
		}
		if statementReturnsRows(stmt, rowsName) {
			return true, true
		}
		if stmtRebindsRows(stmt, rowsName) {
			return false, true
		}
	}
	return false, false
}

func blockProvidesCleanup(ctx *rowsRuleContext, block *ast.BlockStmt, rowsName string) bool {
	if block == nil {
		return false
	}
	handled, _ := scanStatements(ctx, block.List, rowsName)
	return handled
}

func clauseProvidesCleanup(ctx *rowsRuleContext, stmt ast.Stmt, rowsName string) bool {
	switch v := stmt.(type) {
	case *ast.BlockStmt:
		return blockProvidesCleanup(ctx, v, rowsName)
	case *ast.IfStmt:
		return statementProvidesCleanup(ctx, v, rowsName)
	case *ast.CaseClause:
		handled, _ := scanStatements(ctx, v.Body, rowsName)
		return handled
	case *ast.CommClause:
		handled, _ := scanStatements(ctx, v.Body, rowsName)
		return handled
	default:
		return statementProvidesCleanup(ctx, stmt, rowsName)
	}
}

func blockUsesRows(block *ast.BlockStmt, rowsName string) bool {
	if block == nil {
		return false
	}
	for _, stmt := range block.List {
		if stmtUsesRows(stmt, rowsName) {
			return true
		}
	}
	return false
}

func clauseUsesRows(stmt ast.Stmt, rowsName string) bool {
	if stmt == nil {
		return false
	}
	return stmtUsesRows(stmt, rowsName)
}

func stmtUsesRows(stmt ast.Stmt, rowsName string) bool {
	if stmt == nil || rowsName == "" {
		return false
	}
	found := false
	ast.Inspect(stmt, func(n ast.Node) bool {
		if found {
			return false
		}
		ident, ok := n.(*ast.Ident)
		if ok && ident.Name == rowsName {
			found = true
			return false
		}
		return true
	})
	return found
}

func blockTerminates(block *ast.BlockStmt) bool {
	if block == nil || len(block.List) == 0 {
		return false
	}
	last := block.List[len(block.List)-1]
	switch stmt := last.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.ExprStmt:
		if call, ok := stmt.X.(*ast.CallExpr); ok {
			if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" {
				return true
			}
		}
	}
	return false
}

func conditionAllowsCleanup(cond ast.Expr, rowsName string) bool {
	ensuresRowsNonNil := false
	ensuresErrNil := false
	ast.Inspect(cond, func(n ast.Node) bool {
		bin, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		switch bin.Op {
		case token.NEQ:
			if (exprUsesIdent(bin.X, rowsName) && isNilIdent(bin.Y)) || (exprUsesIdent(bin.Y, rowsName) && isNilIdent(bin.X)) {
				ensuresRowsNonNil = true
			}
		case token.EQL:
			if (identLooksLikeError(bin.X) && isNilIdent(bin.Y)) || (identLooksLikeError(bin.Y) && isNilIdent(bin.X)) {
				ensuresErrNil = true
			}
		}
		return true
	})
	return ensuresRowsNonNil || ensuresErrNil
}

func isNilIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "nil"
}

func identLooksLikeError(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	return strings.Contains(strings.ToLower(ident.Name), "err")
}

func clausesProvideCleanup(ctx *rowsRuleContext, clauses []ast.Stmt, rowsName string) bool {
	if len(clauses) == 0 {
		return false
	}
	for _, clause := range clauses {
		if !clauseProvidesCleanup(ctx, clause, rowsName) {
			return false
		}
	}
	return true
}

func commClausesProvideCleanup(ctx *rowsRuleContext, clauses []ast.Stmt, rowsName string) bool {
	if len(clauses) == 0 {
		return false
	}
	for _, clause := range clauses {
		commClause, ok := clause.(*ast.CommClause)
		if !ok {
			return false
		}
		handled, _ := scanStatements(ctx, commClause.Body, rowsName)
		if !handled {
			return false
		}
	}
	return true
}

func stmtRebindsRows(stmt ast.Stmt, rowsName string) bool {
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

func deferClosesRows(ctx *rowsRuleContext, deferStmt *ast.DeferStmt, rowsName string) bool {
	if deferStmt == nil {
		return false
	}
	call := deferStmt.Call
	if call == nil {
		return false
	}
	if callClosesRows(ctx, call, rowsName) {
		return true
	}
	if lit, ok := call.Fun.(*ast.FuncLit); ok {
		return nodeContainsRowsClose(lit.Body, rowsName)
	}
	return false
}

func callClosesRows(ctx *rowsRuleContext, call *ast.CallExpr, rowsName string) bool {
	if call == nil {
		return false
	}
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if sel.Sel.Name == "Close" && exprIsIdent(sel.X, rowsName) {
			return true
		}
	}
	if helperLikelyCloses(ctx, call, rowsName) {
		return true
	}
	return false
}

func helperLikelyCloses(ctx *rowsRuleContext, call *ast.CallExpr, rowsName string) bool {
	if call == nil {
		return false
	}
	argIndex := -1
	for idx, arg := range call.Args {
		if exprUsesIdent(arg, rowsName) {
			argIndex = idx
			break
		}
	}
	if argIndex == -1 {
		return false
	}
	if ctx != nil {
		if ident, ok := call.Fun.(*ast.Ident); ok {
			if ctx.helperClosesRows(ident.Name, argIndex, rowsName) {
				return true
			}
		}
	}
	name := strings.ToLower(callName(call.Fun))
	if name == "" {
		return false
	}
	for _, keyword := range cleanupHelperKeywords {
		if strings.Contains(name, keyword) {
			return true
		}
	}
	return false
}

func (ctx *rowsRuleContext) helperClosesRows(funcName string, argIndex int, rowsName string) bool {
	if ctx == nil || funcName == "" {
		return false
	}
	decl, ok := ctx.funcDecls[funcName]
	if !ok || decl == nil || decl.Body == nil || decl.Type == nil || decl.Type.Params == nil {
		return false
	}
	paramName := parameterNameByIndex(decl.Type.Params, argIndex)
	if paramName == "" {
		return false
	}
	return nodeContainsRowsClose(decl.Body, paramName)
}

func parameterNameByIndex(fieldList *ast.FieldList, target int) string {
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

func callUsesIdent(args []ast.Expr, rowsName string) bool {
	for _, arg := range args {
		if exprUsesIdent(arg, rowsName) {
			return true
		}
	}
	return false
}

func selectRowsIdent(exprs []ast.Expr) *ast.Ident {
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

func selectRowsName(idents []*ast.Ident) string {
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

func callName(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	default:
		return ""
	}
}

func nodeContainsRowsClose(node ast.Node, rowsName string) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Close" && exprIsIdent(sel.X, rowsName) {
			found = true
			return false
		}
		return true
	})
	return found
}

func exprUsesIdent(expr ast.Expr, name string) bool {
	if expr == nil || name == "" {
		return false
	}
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if found {
			return false
		}
		ident, ok := n.(*ast.Ident)
		if ok && ident.Name == name {
			found = true
			return false
		}
		return true
	})
	return found
}

func exprIsIdent(expr ast.Expr, name string) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == name
	}
	if paren, ok := expr.(*ast.ParenExpr); ok {
		return exprIsIdent(paren.X, name)
	}
	return false
}
