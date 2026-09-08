// Package gotest owns conservative Go test-quality syntax analysis. It never
// invokes go list, downloads dependencies, or treats syntax as runtime proof.
package gotest

import (
	"fmt"
	"github.com/vrooli/api-core/relationshiprefs"
	"go/ast"
	"go/build"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
	"unit-health/internal/testquality"
)

type Input struct {
	DefaultTestKind                         string
	Root, Workspace, GOOS, GOARCH           string
	Files, SelectedTests, RegisteredHelpers []string
	TestKinds                               map[string]string
}
type source struct {
	path, pkg string
	ast       *ast.File
	active    bool
	imports   map[string]string
}
type function struct {
	file *source
	decl *ast.FuncDecl
}
type analyzer struct {
	links      *[]testquality.TestLinks
	tags       map[string][]string
	registered map[string]bool
	input      Input
	set        *token.FileSet
	functions  map[string]map[string][]function
	results    []testquality.Result
}

func Analyze(input Input) ([]testquality.Result, error) {
	return analyze(input, nil)
}

func AnalyzeEvidence(input Input) ([]testquality.Result, []testquality.TestLinks, error) {
	var links []testquality.TestLinks
	rows, err := analyze(input, &links)
	if err != nil {
		return nil, nil, err
	}
	return rows, links, nil
}

func analyze(input Input, links *[]testquality.TestLinks) ([]testquality.Result, error) {
	if input.Root == "" || input.Workspace == "" || len(input.Files) > 1000 {
		return nil, fmt.Errorf("root, workspace and bounded file inventory required")
	}
	a := analyzer{input: input, links: links, tags: map[string][]string{}, set: token.NewFileSet(), functions: map[string]map[string][]function{}}
	ctx := build.Default
	if input.GOOS != "" {
		ctx.GOOS = input.GOOS
	} else {
		ctx.GOOS = runtime.GOOS
	}
	if input.GOARCH != "" {
		ctx.GOARCH = input.GOARCH
	}
	var files []*source
	canonicalRoot, err := filepath.Abs(input.Root)
	if err != nil {
		return nil, err
	}
	canonicalRoot, err = filepath.EvalSymlinks(canonicalRoot)
	if err != nil {
		return nil, err
	}
	totalBytes := 0
	for _, path := range input.Files {
		if !filepath.IsLocal(path) {
			return nil, fmt.Errorf("source must be root-relative")
		}
		full := filepath.Join(input.Root, path)
		resolved, err := filepath.EvalSymlinks(full)
		if err != nil {
			return nil, err
		}
		resolved, err = filepath.Abs(resolved)
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(canonicalRoot, resolved)
		if err != nil || !filepath.IsLocal(rel) {
			return nil, fmt.Errorf("source escapes root")
		}
		info, err := os.Stat(resolved)
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("source must be a regular file")
		}
		reader, err := os.Open(resolved)
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(reader, 1024*1024+1))
		reader.Close()
		if readErr != nil {
			return nil, readErr
		}
		totalBytes += len(data)
		if len(data) > 1024*1024 || totalBytes > 16*1024*1024 {
			return nil, fmt.Errorf("Go source analysis byte limit exceeded")
		}
		tree, err := parser.ParseFile(a.set, full, data, parser.ParseComments)
		if err != nil {
			location := testquality.Location{Line: 1, Column: 1}
			if list, ok := err.(scanner.ErrorList); ok && len(list) > 0 {
				location.Line, location.Column = list[0].Pos.Line, list[0].Pos.Column
			}
			a.results = append(a.results, testquality.Result{RuleID: "assertion-observation", RuleVersion: "1", Target: testquality.Target{Workspace: input.Workspace, File: path, Scope: "file"}, SupportProfile: "go-syntax-v1", Status: testquality.Unknown, Reason: testquality.ParseFailure, EvidenceKind: testquality.Static, Enforcement: testquality.Advisory, Severity: testquality.Warning, Location: location, Limitations: []string{err.Error()}})
			continue
		}
		active := true
		if strings.HasSuffix(path, ".go") {
			active, err = ctx.MatchFile(filepath.Dir(full), filepath.Base(full))
			if err != nil {
				return nil, err
			}
		}
		f := &source{path: path, pkg: filepath.Dir(path) + ":" + tree.Name.Name, ast: tree, active: active, imports: map[string]string{}}
		for _, imp := range tree.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			name := filepath.Base(p)
			if imp.Name != nil {
				name = imp.Name.Name
			}
			f.imports[name] = p
		}
		files = append(files, f)
		if !active {
			continue
		}
		if a.functions[f.pkg] == nil {
			a.functions[f.pkg] = map[string][]function{}
		}
		for _, decl := range tree.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Body != nil {
				a.functions[f.pkg][fn.Name.Name] = append(a.functions[f.pkg][fn.Name.Name], function{f, fn})
			}
		}
	}
	a.registered = registeredProcessHelpers(a.functions)
	for _, f := range files {
		for _, decl := range f.ast.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Body == nil || !testName(fn.Name.Name) {
				continue
			}
			bindings := testingBindings(f, fn.Type)
			if len(bindings) == 0 && fn.Name.Name != "TestMain" {
				continue
			}
			if fn.Name.Name != "TestMain" && (len(fn.Type.Params.List) != 1 || len(fn.Type.Params.List[0].Names) != 1 || fn.Type.Results != nil || fn.Type.TypeParams != nil) {
				continue
			}
			if strings.HasSuffix(f.path, ".go") && !strings.HasSuffix(f.path, "_test.go") {
				continue
			}
			var tags []string
			if fn.Doc != nil {
				tags = append(tags, relationshiprefs.ExtractTestRequirementIDs(fn.Doc.Text())...)
			}
			// The documented Go convention permits a marker on the declaration
			// line. Comments elsewhere in the body cannot tag unrelated siblings.
			line := a.set.Position(fn.Pos()).Line
			for _, group := range f.ast.Comments {
				if a.set.Position(group.Pos()).Line == line {
					tags = append(tags, relationshiprefs.ExtractTestRequirementIDs(group.Text())...)
				}
			}
			a.tags[f.path+":"+fn.Name.Name] = tags
			a.assess(f, fn.Name.Name, fn.Body, fn.Pos(), bindings, !f.active)
		}
	}
	return a.results, nil
}
func testName(name string) bool {
	if !strings.HasPrefix(name, "Test") {
		return false
	}
	tail := strings.TrimPrefix(name, "Test")
	if tail == "" {
		return true
	}
	r, _ := utf8.DecodeRuneInString(tail)
	return !unicode.IsLower(r)
}
func testingBindings(f *source, typ *ast.FuncType) map[*ast.Object]bool {
	out := map[*ast.Object]bool{}
	if typ.Params == nil {
		return out
	}
	for _, field := range typ.Params.List {
		ptr, ok := field.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		sel, ok := ptr.X.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok || f.imports[id.Name] != "testing" || sel.Sel.Name != "T" {
			continue
		}
		for _, name := range field.Names {
			if name.Obj != nil {
				out[name.Obj] = true
			}
		}
	}
	return out
}
func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
func (a *analyzer) assess(f *source, name string, body *ast.BlockStmt, pos token.Pos, bindings map[*ast.Object]bool, inactive bool) {
	kind := a.input.TestKinds[name]
	if kind == "" {
		kind = a.input.DefaultTestKind
		if kind == "" {
			kind = "unit"
		}
	}
	status, reason := testquality.Unknown, testquality.ReasonNone
	switch {
	case inactive || name == "TestMain" || contains(a.input.RegisteredHelpers, name) || a.registered[f.pkg+":"+name] || kind == "compile-contract":
		status, reason = testquality.NotApplicable, testquality.UnsupportedTestKind
	case kind != "unit" && kind != "local-integration":
		reason = testquality.UnsupportedTestKind
	default:
		signals := a.inspect(f, body, bindings, map[string]bool{}, 0)
		switch {
		case signals.assertion:
			status = testquality.CheckedClean
		case signals.reason != "":
			reason = signals.reason
		case signals.children > 0:
			status = testquality.CheckedClean
		default:
			status = testquality.Violation
		}
	}
	if len(a.input.SelectedTests) == 0 || contains(a.input.SelectedTests, name) {
		// A default inventory reports subtests individually rather than treating a
		// parent's children as assertions in every sibling.
		children := a.children(f, name, body, bindings, inactive)
		if children == 0 || len(a.input.SelectedTests) > 0 {
			if a.links != nil {
				*a.links = append(*a.links, testquality.TestLinks{Target: testquality.Target{Workspace: a.input.Workspace, File: f.path, TestID: name}, IDs: append([]string(nil), a.tags[f.path+":"+name]...), EvidenceKind: testquality.Static, Execution: testquality.ExecutionNotRun})
			}
			p := a.set.Position(pos)
			a.results = append(a.results, testquality.Result{RuleID: "assertion-observation", RuleVersion: "1", Target: testquality.Target{Workspace: a.input.Workspace, File: f.path, TestID: name}, TestKind: kind, SupportProfile: "go-syntax-v1", Status: status, Reason: reason, EvidenceKind: testquality.Static, Enforcement: testquality.Advisory, Severity: testquality.Warning, Location: testquality.Location{Line: p.Line, Column: p.Column}, Limitations: []string{"Supported source assertion observations are not proof of execution or behavioral adequacy."}})
			if declared, placeholder := skipDeclaration(body, bindings); declared {
				row := a.results[len(a.results)-1]
				row.RuleID = "skip-declaration"
				row.Status, row.Reason = testquality.Unknown, testquality.NotExecuted
				if status == testquality.NotApplicable {
					row.Status, row.Reason = status, reason
				} else if kind != "unit" && kind != "local-integration" {
					row.Reason = testquality.UnsupportedTestKind
				} else if placeholder {
					row.Status, row.Reason = testquality.Violation, testquality.ReasonNone
				}
				row.Limitations = []string{"Static direct skip declaration only; conditional declarations do not establish runtime skipped outcomes. Only a sole direct skip statement establishes an unconditional placeholder. Absence of a row does not prove absence of delegated or dynamic skips."}
				a.results = append(a.results, row)
			}
		}
	} else {
		a.children(f, name, body, bindings, inactive)
	}
}

// skipDeclaration recognizes import-bound testing receivers, not names or skip
// text. Nested test callbacks own their declarations separately. Deliberately
// narrow placeholder support avoids treating prerequisite checks as defects.
func skipDeclaration(body *ast.BlockStmt, bindings map[*ast.Object]bool) (declared, placeholder bool) {
	isSkip := func(expr ast.Expr) bool {
		call, ok := expr.(*ast.CallExpr)
		if !ok {
			return false
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		return ok && testingReceiver(selector, bindings) && (selector.Sel.Name == "Skip" || selector.Sel.Name == "Skipf" || selector.Sel.Name == "SkipNow")
	}
	if len(body.List) == 1 {
		if statement, ok := body.List[0].(*ast.ExprStmt); ok && isSkip(statement.X) {
			return true, true
		}
	}
	nodes := 0
	ast.Inspect(body, func(node ast.Node) bool {
		nodes++
		if declared || nodes > 4096 {
			return false
		}
		if _, nested := node.(*ast.FuncLit); nested {
			return false
		}
		if expr, ok := node.(ast.Expr); ok && isSkip(expr) {
			declared = true
			return false
		}
		return true
	})
	return declared, false
}
func (a *analyzer) children(f *source, name string, body *ast.BlockStmt, bindings map[*ast.Object]bool, inactive bool) int {
	count := 0
	var scan func(*ast.BlockStmt, int)
	scan = func(block *ast.BlockStmt, depth int) {
		if depth >= 8 {
			return
		}
		bounded := beforeReturn(block)
		ast.Inspect(&bounded, func(node ast.Node) bool {
			if branch, ok := node.(*ast.IfStmt); ok {
				if condition, ok := branch.Cond.(*ast.Ident); ok && condition.Obj == nil && (condition.Name == "true" || condition.Name == "false") {
					if branch.Init != nil {
						scan(&ast.BlockStmt{List: []ast.Stmt{branch.Init}}, depth+1)
					}
					if condition.Name == "true" {
						scan(branch.Body, depth+1)
					} else if branch.Else != nil {
						scan(&ast.BlockStmt{List: []ast.Stmt{branch.Else}}, depth+1)
					}
					return false
				}
			}
			if nested, ok := node.(*ast.BlockStmt); ok && nested != &bounded {
				scan(nested, depth+1)
				return false
			}
			if _, ok := node.(*ast.FuncLit); ok {
				return false
			}
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Run" || !testingReceiver(sel, bindings) || len(call.Args) != 2 {
				return true
			}
			label, ok := call.Args[0].(*ast.BasicLit)
			if !ok || label.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(label.Value)
			if err != nil {
				return true
			}
			callback, ok := call.Args[1].(*ast.FuncLit)
			if !ok {
				return true
			}
			count++
			a.tags[f.path+":"+name+"/"+value] = append(append([]string(nil), a.tags[f.path+":"+name]...), relationshiprefs.ExtractTestRequirementIDs(value)...)
			a.assess(f, name+"/"+value, callback.Body, call.Pos(), testingBindings(f, callback.Type), inactive)
			return false
		})
	}
	scan(body, 0)
	return count
}

func beforeReturn(body *ast.BlockStmt) ast.BlockStmt {
	bounded := *body
	for i, stmt := range body.List {
		if _, ok := stmt.(*ast.ReturnStmt); ok {
			bounded.List = body.List[:i+1]
			break
		}
	}
	return bounded
}

type signals struct {
	assertion bool
	reason    testquality.Reason
	children  int
}

func testingReceiver(sel *ast.SelectorExpr, bindings map[*ast.Object]bool) bool {
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Obj != nil && bindings[id.Obj]
}

var failures = map[string]bool{"Error": true, "Errorf": true, "Fatal": true, "Fatalf": true, "Fail": true, "FailNow": true}

func testifyAssertion(name string) bool {
	for _, base := range []string{"Equal", "NotEqual", "NoError", "Error", "ErrorIs", "ErrorAs", "True", "False", "Nil", "NotNil", "Len", "Empty", "NotEmpty", "Contains", "NotContains", "Greater", "Less", "Panics", "NotPanics", "WithinDuration", "Fail", "FailNow"} {
		if name == base || name == base+"f" {
			return true
		}
	}
	return false
}

func helperBindings(helper function, call *ast.CallExpr, caller map[*ast.Object]bool) map[*ast.Object]bool {
	typed := testingBindings(helper.file, helper.decl.Type)
	bound := map[*ast.Object]bool{}
	index := 0
	if helper.decl.Type.Params == nil {
		return bound
	}
	for _, field := range helper.decl.Type.Params.List {
		for _, name := range field.Names {
			if index < len(call.Args) && typed[name.Obj] {
				if arg, ok := call.Args[index].(*ast.Ident); ok && caller[arg.Obj] {
					bound[name.Obj] = true
				}
			}
			index++
		}
	}
	return bound
}

func (a *analyzer) inspect(f *source, body *ast.BlockStmt, bindings map[*ast.Object]bool, seen map[string]bool, depth int) signals {
	out := signals{}
	if depth >= 8 {
		return signals{reason: testquality.ResolutionLimit}
	}
	nodes := 0
	boundedBody := beforeReturn(body)
	merge := func(nested signals) {
		out.assertion = out.assertion || nested.assertion
		out.children += nested.children
		if nested.reason != "" {
			out.reason = nested.reason
		}
	}
	ast.Inspect(&boundedBody, func(node ast.Node) bool {
		nodes++
		if nodes > 4096 {
			out.reason = testquality.ResolutionLimit
			return false
		}
		if _, ok := node.(*ast.FuncLit); ok {
			return false
		}
		if nested, ok := node.(*ast.BlockStmt); ok && nested != &boundedBody {
			merge(a.inspect(f, nested, bindings, seen, depth+1))
			return false
		}
		if branch, ok := node.(*ast.IfStmt); ok {
			if condition, ok := branch.Cond.(*ast.Ident); ok && (condition.Name == "false" || condition.Name == "true") && condition.Obj == nil {
				if branch.Init != nil {
					merge(a.inspect(f, &ast.BlockStmt{List: []ast.Stmt{branch.Init}}, bindings, seen, depth+1))
				}
				if condition.Name == "true" {
					merge(a.inspect(f, branch.Body, bindings, seen, depth+1))
				} else if branch.Else != nil {
					merge(a.inspect(f, &ast.BlockStmt{List: []ast.Stmt{branch.Else}}, bindings, seen, depth+1))
				}
				return false
			}
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := call.Fun.(type) {
		case *ast.SelectorExpr:
			if testingReceiver(fun, bindings) {
				switch {
				case failures[fun.Sel.Name]:
					out.assertion = true
				case fun.Sel.Name == "Run":
					if len(call.Args) == 2 {
						label, literal := call.Args[0].(*ast.BasicLit)
						if _, ok := call.Args[1].(*ast.FuncLit); ok && literal && label.Kind == token.STRING {
							out.children++
						} else {
							out.reason = testquality.ExternalHelperUnresolved
						}
					}
				case fun.Sel.Name == "Skip" || fun.Sel.Name == "Skipf" || fun.Sel.Name == "SkipNow":
					out.reason = testquality.NotExecuted
				case fun.Sel.Name == "Cleanup":
					out.reason = testquality.ExternalHelperUnresolved
				}
				return true
			}
			id, ok := fun.X.(*ast.Ident)
			if !ok {
				out.reason = testquality.ExternalHelperUnresolved
				return true
			}
			// Import identity, not names such as require, controls recognition. Local
			// shadowing has a parser object and cannot borrow the import's identity.
			imported := f.imports[id.Name]
			if id.Obj == nil && (imported == "github.com/stretchr/testify/require" || imported == "github.com/stretchr/testify/assert") {
				if testifyAssertion(fun.Sel.Name) && len(call.Args) > 0 {
					if receiver, ok := call.Args[0].(*ast.Ident); ok && bindings[receiver.Obj] {
						out.assertion = true
						return true
					}
				}
				out.reason = testquality.ExternalHelperUnresolved
				return true
			}
			if imported != "" && strings.Contains(imported, ".") {
				out.reason = testquality.BuildContextUnavailable
			} else {
				out.reason = testquality.ExternalHelperUnresolved
			}
		case *ast.Ident:
			funcs := a.functions[f.pkg][fun.Name]
			if len(funcs) == 1 && (fun.Obj == nil || fun.Obj.Kind == ast.Fun) {
				key := f.pkg + ":" + fun.Name
				if seen[key] {
					out.reason = testquality.ResolutionLimit
					return false
				}
				seen[key] = true
				helper := funcs[0]
				nested := a.inspect(helper.file, helper.decl.Body, helperBindings(helper, call, bindings), seen, depth+1)
				delete(seen, key)
				out.assertion = out.assertion || nested.assertion
				if nested.reason != "" {
					out.reason = nested.reason
				}
			} else if fun.Obj == nil || fun.Obj.Kind != ast.Typ {
				out.reason = testquality.ExternalHelperUnresolved
			}
		default:
			out.reason = testquality.ExternalHelperUnresolved
		}
		return true
	})
	return out
}
