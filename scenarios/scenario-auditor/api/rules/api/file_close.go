package api

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

/*
Rule: File Handle Close
Description: Ensure files opened on disk are explicitly closed
Reason: Prevents leaking file descriptors which can exhaust process limits
Category: api
Severity: high
Standard: resource-management-v1
Targets: api

<test-case id="file-not-closed" should-fail="true">
  <description>File opened from disk without a corresponding Close</description>
  <input language="go"><![CDATA[
func readConfig(path string) ([]byte, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    return io.ReadAll(file)
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>File Not Closed</expected-message>
</test-case>

<test-case id="file-closed" should-fail="false">
  <description>File opened and closed with defer after checking errors</description>
  <input language="go"><![CDATA[
func readConfigSafely(path string) ([]byte, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    return io.ReadAll(file)
}
]]></input>
</test-case>

<test-case id="alternate-error-name" should-fail="true">
  <description>Handles alternate error identifier when checking for missing Close</description>
  <input language="go"><![CDATA[
func readConfigWithAlias(path string) ([]byte, error) {
    handle, openErr := os.Open(path)
    if openErr != nil {
        return nil, openErr
    }
    return io.ReadAll(handle)
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>File Not Closed</expected-message>
</test-case>

<test-case id="reassigned-handle" should-fail="true">
  <description>Handles reassignment with '=' for existing variables</description>
  <input language="go"><![CDATA[
func reuseHandle(path string) error {
    var file *os.File
    var err error
    file, err = os.Open(path)
    if err != nil {
        return err
    }
    return nil
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>File Not Closed</expected-message>
</test-case>

<test-case id="blank-identifier" should-fail="true">
  <description>Detects when the handle is discarded via the blank identifier</description>
  <input language="go"><![CDATA[
func discardHandle(path string) error {
    _, err := os.Open(path)
    if err != nil {
        return err
    }
    return nil
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>File handle ignored</expected-message>
</test-case>

<test-case id="defer-func-literal" should-fail="false">
  <description>Accepts defer with function literal that closes the file</description>
  <input language="go"><![CDATA[
func readConfigWithFunc(path string) ([]byte, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer func() {
        _ = f.Close()
    }()
    return io.ReadAll(f)
}
]]></input>
</test-case>

<test-case id="unreachable-close" should-fail="true">
  <description>Close inside an unreachable branch does not satisfy the rule</description>
  <input language="go"><![CDATA[
func unreachableClose(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    if false {
        f.Close()
    }
    return nil
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>File Not Closed</expected-message>
</test-case>

<test-case id="explicit-close" should-fail="false">
  <description>Explicit close on all paths is accepted</description>
  <input language="go"><![CDATA[
func readConfigExplicit(path string) ([]byte, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    data, err := io.ReadAll(f)
    if err != nil {
        f.Close()
        return nil, err
    }
    if closeErr := f.Close(); closeErr != nil {
        return nil, closeErr
    }
    return data, nil
}
]]></input>
</test-case>

<test-case id="struct-field-managed" should-fail="false">
  <description>Handles ownership transfer to a struct field with later close logic</description>
  <input language="go"><![CDATA[
type logger struct {
    file *os.File
}

func newLogger(path string) (*logger, error) {
    l := &logger{}
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    l.file = f
    return l, nil
}

func (l *logger) Close() error {
    if l.file == nil {
        return nil
    }
    return l.file.Close()
}
]]></input>
</test-case>

<test-case id="alias-defer-close" should-fail="false">
  <description>Handles := aliasing when the alias is the variable that gets closed</description>
  <input language="go"><![CDATA[
func aliasClose(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    handle := f
    defer handle.Close()
    return nil
}
]]></input>
</test-case>

<test-case id="struct-literal-transfer" should-fail="false">
  <description>Returning a struct literal that embeds the file is treated as ownership transfer</description>
  <input language="go"><![CDATA[
type reader struct {
    file *os.File
}

func newReader(path string) (*reader, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    return &reader{file: f}, nil
}
]]></input>
</test-case>

<test-case id="return-handle" should-fail="false">
  <description>Returning the open handle to the caller is treated as ownership transfer</description>
  <input language="go"><![CDATA[
func openForCaller(path string) (*os.File, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    return f, nil
}
]]></input>
</test-case>
*/

// CheckFileClose ensures files opened via os.Open* are eventually closed.
func CheckFileClose(content []byte, filePath string) []Violation {
	if !strings.HasSuffix(strings.ToLower(filePath), ".go") {
		return nil
	}

	fset := token.NewFileSet()
	source := string(content)
	file, err := parser.ParseFile(fset, filePath, source, parser.ParseComments)
	lineOffset := 0
	if (err != nil && file == nil) || len(file.Decls) == 0 {
		wrapped := "package main\n" + source
		lineOffset = 1
		file, err = parser.ParseFile(fset, filePath, wrapped, parser.ParseComments)
		if err != nil && file == nil {
			return nil
		}
	}

	checker := newFileCloseChecker(fset, source, filePath, lineOffset)
	checker.processFile(file)
	return checker.violations
}

type fileCloseChecker struct {
	fset       *token.FileSet
	lines      []string
	filePath   string
	lineOffset int
	violations []Violation
}

type varKey struct {
	name string
	obj  *ast.Object
	pos  token.Pos
}

type openInfo struct {
	key              varKey
	line             int
	snippet          string
	blank            bool
	closed           bool
	closeReachable   bool
	closeDeferred    bool
	unreachableClose bool
	escaped          bool
}

func newFileCloseChecker(fset *token.FileSet, source string, filePath string, lineOffset int) *fileCloseChecker {
	return &fileCloseChecker{
		fset:       fset,
		lines:      strings.Split(source, "\n"),
		filePath:   filePath,
		lineOffset: lineOffset,
	}
}

func (c *fileCloseChecker) processFile(file *ast.File) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		c.processFunc(fn)
	}
}

func (c *fileCloseChecker) processFunc(fn *ast.FuncDecl) {
	openMap := make(map[varKey][]*openInfo)
	var stack []ast.Node

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return false
		}

		stack = append(stack, n)

		switch node := n.(type) {
		case *ast.AssignStmt:
			c.considerOpen(node, openMap)
			c.detectEscapes(node, openMap)
		case *ast.CallExpr:
			c.considerClose(node, stack, openMap)
		case *ast.ReturnStmt:
			c.considerReturn(node, openMap)
		}

		return true
	})

	for _, infos := range openMap {
		for _, info := range infos {
			if info.blank {
				c.addViolation(info, "File handle ignored", "Capture the returned *os.File and close it (for example, assign it to a variable and defer file.Close()).")
				continue
			}

			if info.escaped {
				continue
			}

			if !info.closed {
				msg := fmt.Sprintf("File %s is opened but never closed", info.key.name)
				reco := fmt.Sprintf("Add defer %s.Close() after checking the error.", info.key.name)
				c.addViolation(info, "File Not Closed", reco, msg)
				continue
			}

			if info.closed && !info.closeReachable {
				reco := fmt.Sprintf("Ensure %s.Close() executes on every path (prefer defer).", info.key.name)
				c.addViolation(info, "File Not Closed", reco, "Close call is unreachable; the file may leak")
			}
		}
	}
}

func (c *fileCloseChecker) considerOpen(assign *ast.AssignStmt, openMap map[varKey][]*openInfo) {
	if len(assign.Rhs) != 1 {
		return
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok || !isFileOpenCall(call) {
		return
	}
	if len(assign.Lhs) == 0 {
		return
	}
	lhs, ok := assign.Lhs[0].(*ast.Ident)
	if !ok {
		return
	}

	key := makeVarKey(lhs)
	pos := c.adjustedPosition(assign.Pos())
	info := &openInfo{
		key:     key,
		line:    pos.Line,
		snippet: c.lineText(pos.Line),
		blank:   lhs.Name == "_",
	}

	openMap[key] = append(openMap[key], info)
}

func (c *fileCloseChecker) considerClose(call *ast.CallExpr, stack []ast.Node, openMap map[varKey][]*openInfo) {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel == nil || selector.Sel.Name != "Close" {
		return
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return
	}
	key := makeVarKey(ident)
	infos, ok := openMap[key]
	if !ok {
		return
	}

	reachable := !isInUnreachableBlock(stack)
	isDeferred := hasDeferAncestor(stack)

	pos := call.Lparen
	if pos == token.NoPos {
		pos = call.Pos()
	}
	for i := len(infos) - 1; i >= 0; i-- {
		open := infos[i]
		if open.blank {
			continue
		}
		if openPos := openPosition(open); openPos > pos {
			continue
		}
		if open.closed && open.closeReachable {
			return
		}
		if reachable {
			open.closed = true
			open.closeReachable = true
			open.closeDeferred = isDeferred
		} else {
			open.unreachableClose = true
		}
		return
	}
}

func (c *fileCloseChecker) detectEscapes(assign *ast.AssignStmt, openMap map[varKey][]*openInfo) {
	if len(assign.Rhs) == 0 {
		return
	}

	for _, info := range c.findOpenInfosInExprs(assign.Rhs, openMap) {
		if info.blank {
			continue
		}

		if assign.Pos() == openPosition(info) {
			// This assignment created the open handle; do not treat accompanying
			// result bindings (e.g., error values) as escapes.
			continue
		}

		if assign.Tok == token.DEFINE {
			if len(assign.Lhs) == 1 {
				if ident, ok := assign.Lhs[0].(*ast.Ident); ok && ident.Name != "" && ident.Name != "_" && !isSameVarExpr(assign.Lhs[0], info.key) {
					aliasKey := makeVarKey(ident)
					openMap[aliasKey] = append(openMap[aliasKey], info)
					continue
				}
			}
		}

		if info.escaped {
			continue
		}

		escaped := false
		for _, lhs := range assign.Lhs {
			if isSameVarExpr(lhs, info.key) {
				continue
			}
			escaped = true
			break
		}

		if escaped {
			info.escaped = true
		}
	}
}

func (c *fileCloseChecker) considerReturn(ret *ast.ReturnStmt, openMap map[varKey][]*openInfo) {
	for _, result := range ret.Results {
		c.markReturnEscapes(result, openMap)
	}
}

func (c *fileCloseChecker) markReturnEscapes(expr ast.Expr, openMap map[varKey][]*openInfo) {
	switch node := expr.(type) {
	case *ast.Ident:
		c.markIdentEscape(node, openMap)
	case *ast.UnaryExpr:
		if node.Op == token.AND || node.Op == token.MUL {
			c.markReturnEscapes(node.X, openMap)
		}
	case *ast.ParenExpr:
		c.markReturnEscapes(node.X, openMap)
	case *ast.CompositeLit:
		for _, elt := range node.Elts {
			switch v := elt.(type) {
			case *ast.KeyValueExpr:
				c.markReturnEscapes(v.Value, openMap)
			default:
				c.markReturnEscapes(v, openMap)
			}
		}
	}
}

func (c *fileCloseChecker) markIdentEscape(ident *ast.Ident, openMap map[varKey][]*openInfo) {
	for key, infos := range openMap {
		if len(infos) == 0 || !identMatchesKey(ident, key) {
			continue
		}
		info := latestOpenInfo(infos)
		if info == nil || info.blank || info.escaped {
			continue
		}
		info.escaped = true
	}
}

func (c *fileCloseChecker) addViolation(info *openInfo, title string, recommendation string, descriptions ...string) {
	description := title
	if len(descriptions) > 0 && strings.TrimSpace(descriptions[0]) != "" {
		description = descriptions[0]
	}
	c.violations = append(c.violations, Violation{
		Type:           "file_close",
		Severity:       "high",
		Title:          title,
		Description:    description,
		FilePath:       c.filePath,
		LineNumber:     info.line,
		CodeSnippet:    info.snippet,
		Recommendation: recommendation,
		Standard:       "resource-management-v1",
	})
}

func (c *fileCloseChecker) adjustedPosition(pos token.Pos) token.Position {
	position := c.fset.Position(pos)
	position.Line -= c.lineOffset
	if position.Line < 1 {
		position.Line = 1
	}
	return position
}

func (c *fileCloseChecker) lineText(line int) string {
	if line <= 0 || line > len(c.lines) {
		return ""
	}
	return strings.TrimSpace(c.lines[line-1])
}

func makeVarKey(ident *ast.Ident) varKey {
	key := varKey{name: ident.Name, obj: ident.Obj, pos: ident.Pos()}
	if ident.Obj != nil {
		key.pos = ident.Obj.Pos()
	}
	return key
}

func isFileOpenCall(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkgIdent, ok := selector.X.(*ast.Ident)
	if !ok || pkgIdent.Name != "os" {
		return false
	}
	name := selector.Sel.Name
	return name == "Open" || name == "OpenFile" || name == "Create"
}

func (c *fileCloseChecker) findOpenInfosInExprs(exprs []ast.Expr, openMap map[varKey][]*openInfo) []*openInfo {
	var collected []*openInfo
	seen := make(map[*openInfo]struct{})
	for _, expr := range exprs {
		for _, info := range c.findOpenInfosInExpr(expr, openMap) {
			if _, ok := seen[info]; ok {
				continue
			}
			seen[info] = struct{}{}
			collected = append(collected, info)
		}
	}
	return collected
}

func (c *fileCloseChecker) findOpenInfosInExpr(expr ast.Expr, openMap map[varKey][]*openInfo) []*openInfo {
	if expr == nil {
		return nil
	}

	var matches []*openInfo
	seen := make(map[*openInfo]struct{})
	ast.Inspect(expr, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		for key, infos := range openMap {
			if len(infos) == 0 {
				continue
			}
			if !identMatchesKey(ident, key) {
				continue
			}
			info := latestOpenInfo(infos)
			if info == nil {
				continue
			}
			if _, ok := seen[info]; ok {
				continue
			}
			seen[info] = struct{}{}
			matches = append(matches, info)
		}
		return true
	})
	return matches
}

func latestOpenInfo(list []*openInfo) *openInfo {
	if len(list) == 0 {
		return nil
	}
	return list[len(list)-1]
}

func identMatchesKey(ident *ast.Ident, key varKey) bool {
	if ident == nil {
		return false
	}
	if key.obj != nil && ident.Obj != nil {
		return ident.Obj == key.obj
	}
	if key.obj != nil && ident.Obj == nil {
		return ident.Name == key.name
	}
	if key.obj == nil && ident.Obj != nil {
		return ident.Obj.Pos() == key.pos && ident.Name == key.name
	}
	return ident.Name == key.name
}

func isSameVarExpr(expr ast.Expr, key varKey) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	return identMatchesKey(ident, key)
}

func isInUnreachableBlock(stack []ast.Node) bool {
	for i := 0; i < len(stack); i++ {
		if ifStmt, ok := stack[i].(*ast.IfStmt); ok {
			if isConstantFalse(ifStmt.Cond) {
				if i+1 < len(stack) && stack[i+1] == ifStmt.Body {
					return true
				}
			}
		}
	}
	return false
}

func hasDeferAncestor(stack []ast.Node) bool {
	for i := len(stack) - 1; i >= 0; i-- {
		if _, ok := stack[i].(*ast.DeferStmt); ok {
			return true
		}
	}
	return false
}

func isConstantFalse(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "false"
	}
	return false
}

func openPosition(info *openInfo) token.Pos {
	return info.key.pos
}
