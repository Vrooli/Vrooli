package facts

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
)

// cliResolveArgsSchema preserves the legacy direct-literal-only contract used
// by focused callers. Production composition may also resolve a unique local
// helper whose body is exactly one direct return of an ArgSchema literal.
func cliResolveArgsSchema(expression string, file *ast.File, fset *token.FileSet) (string, string, int) {
	unit := cliSourceUnit{file: file, fset: fset}
	result := cliResolveArgsSchemaInPackage(expression, unit, nil)
	return result.state, result.summary, result.line
}

type cliArgsSchemaResolution struct {
	state, summary string
	line           int
	path, hash     string
}

func cliResolveArgsSchemaInPackage(expression string, caller cliSourceUnit, packageUnits []cliSourceUnit, requestLines ...int) cliArgsSchemaResolution {
	expr, err := parser.ParseExpr(expression)
	if err != nil {
		return cliArgsSchemaResolution{state: "UNKNOWN", summary: "Args expression is not parseable"}
	}
	line := caller.fset.Position(expr.Pos()).Line
	if len(requestLines) > 0 && requestLines[0] > 0 {
		line = requestLines[0]
	}
	lit, ok := expr.(*ast.CompositeLit)
	if ok && cliIsArgSchemaType(lit.Type, caller.file) {
		positionals, options, state, reason := cliReadArgSchema(lit)
		if state != "STATIC_SCHEMA_EXACT" {
			return cliArgsSchemaResolution{state: state, summary: reason, line: line, path: caller.path, hash: caller.hash}
		}
		return cliFormatArgSchema(positionals, options, line, caller.path, caller.hash)
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(packageUnits) == 0 {
		return cliArgsSchemaResolution{state: "UNKNOWN", summary: "Args expression is not a direct commandtree.ArgSchema literal or supported local helper", line: line, path: caller.path, hash: caller.hash}
	}
	name, ok := cliLocalFunctionName(call.Fun)
	if !ok {
		return cliArgsSchemaResolution{state: "UNKNOWN", summary: "Args helper is not a local function call", line: line, path: caller.path, hash: caller.hash}
	}
	var helperUnit cliSourceUnit
	var helper *ast.FuncDecl
	count := 0
	for _, unit := range packageUnits {
		for _, decl := range unit.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Recv == nil && fn.Name.Name == name {
				count++
				helperUnit, helper = unit, fn
			}
		}
	}
	if count != 1 {
		state, reason := "UNKNOWN", "local Args helper is missing"
		if count > 1 {
			state, reason = "AMBIGUOUS", "local Args helper has multiple source declarations"
		}
		return cliArgsSchemaResolution{state: state, summary: reason, line: line, path: caller.path, hash: caller.hash}
	}
	if helper.Body == nil || len(helper.Body.List) == 0 {
		return cliArgsSchemaResolution{state: "UNKNOWN", summary: "local Args helper has no analyzable body", line: line, path: caller.path, hash: caller.hash}
	}
	bindings := map[string]string{}
	var params []string
	if helper.Type.Params != nil {
		params = make([]string, 0, helper.Type.Params.NumFields())
		for _, field := range helper.Type.Params.List {
			for _, param := range field.Names {
				params = append(params, param.Name)
			}
		}
	}
	if len(call.Args) != len(params) {
		return cliArgsSchemaResolution{state: "UNKNOWN", summary: "local Args helper does not have one direct result for the supplied arguments", line: line, path: caller.path, hash: caller.hash}
	}
	for i, name := range params {
		value, ok := cliString(call.Args[i])
		if !ok {
			return cliArgsSchemaResolution{state: "UNKNOWN", summary: "local Args helper arguments must be string literals", line: line, path: caller.path, hash: caller.hash}
		}
		bindings[name] = value
	}
	contents, state, reason := cliInterpretArgsHelper(helperUnit, helper, bindings, packageUnits, 0)
	if state != "STATIC_SCHEMA_EXACT" {
		return cliArgsSchemaResolution{state: state, summary: reason, line: helperUnit.fset.Position(helper.Pos()).Line, path: helperUnit.path, hash: helperUnit.hash}
	}
	resolution := cliFormatArgSchema(contents.positionals, contents.options, helperUnit.fset.Position(helper.Pos()).Line, helperUnit.path, helperUnit.hash)
	if len(contents.sources) > 0 {
		resolution.summary += "; helper sources: " + strings.Join(contents.sources, "; ")
	}
	return resolution
}

func cliFormatArgSchema(positionals []cliSchemaPositional, options []cliSchemaOption, line int, path, hash string, states ...string) cliArgsSchemaResolution {
	state, reason := "STATIC_SCHEMA_EXACT", "literal fields only"
	if len(states) > 0 {
		state = states[0]
	}
	if len(states) > 1 {
		reason = states[1]
	}
	if state != "STATIC_SCHEMA_EXACT" {
		return cliArgsSchemaResolution{state: state, summary: reason, line: line, path: path, hash: hash}
	}
	min := 0
	max := len(positionals)
	for _, p := range positionals {
		if p.required {
			min++
		}
	}
	unbounded := len(positionals) > 0 && positionals[len(positionals)-1].repeatable
	if unbounded {
		max--
	}
	var parts []string
	if unbounded {
		parts = append(parts, fmt.Sprintf("positionals=%d..unbounded", min))
	} else {
		parts = append(parts, fmt.Sprintf("positionals=%d..%d", min, max))
	}
	for _, p := range positionals {
		parts = append(parts, fmt.Sprintf("%s(required=%t,repeatable=%t; description=%q)", p.name, p.required, p.repeatable, p.description))
	}
	for _, o := range options {
		spellings := append([]string{o.name}, o.aliases...)
		parts = append(parts, fmt.Sprintf("option[%s](value=%q,repeatable=%t; description=%q)", strings.Join(spellings, "|"), o.valueName, o.repeatable, o.description))
	}
	return cliArgsSchemaResolution{state: state, summary: strings.Join(parts, "; "), line: line, path: path, hash: hash}
}

type cliArgSchemaContents struct {
	positionals []cliSchemaPositional
	options     []cliSchemaOption
	sources     []string
}

// cliInterpretArgsHelper supports either one direct ArgSchema return or the
// bounded straight-line form used by resourceStartArgs: initialize a local
// schema from another exact helper, append literal options, then return it.
func cliInterpretArgsHelper(unit cliSourceUnit, helper *ast.FuncDecl, bindings map[string]string, packageUnits []cliSourceUnit, depth int) (cliArgSchemaContents, string, string) {
	if depth >= 4 || helper == nil || helper.Body == nil {
		return cliArgSchemaContents{}, "UNKNOWN", "Args helper is missing or exceeds the direct-helper depth cap"
	}
	if len(helper.Body.List) == 1 {
		ret, ok := helper.Body.List[0].(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			return cliArgSchemaContents{}, "UNKNOWN", "Args helper is not a single direct return"
		}
		literal, ok := ret.Results[0].(*ast.CompositeLit)
		if !ok || !cliIsArgSchemaType(literal.Type, unit.file) {
			return cliArgSchemaContents{}, "UNKNOWN", "Args helper result is not a direct commandtree.ArgSchema literal"
		}
		positionals, options, state, reason := cliReadArgSchemaWithBindings(literal, bindings)
		return cliArgSchemaContents{positionals: positionals, options: options}, state, reason
	}
	if len(helper.Body.List) < 3 {
		return cliArgSchemaContents{}, "UNKNOWN", "Args helper builder must initialize, append literal options, and return one schema"
	}
	initialize, ok := helper.Body.List[0].(*ast.AssignStmt)
	if !ok || initialize.Tok != token.DEFINE || len(initialize.Lhs) != 1 || len(initialize.Rhs) != 1 {
		return cliArgSchemaContents{}, "UNKNOWN", "Args helper builder does not start with one local schema initializer"
	}
	variable, ok := initialize.Lhs[0].(*ast.Ident)
	if !ok || variable.Name == "" {
		return cliArgSchemaContents{}, "UNKNOWN", "Args helper schema initializer target is not a local identifier"
	}
	contents, source, state, reason := cliResolveSchemaHelperCall(unit, initialize.Rhs[0], packageUnits, bindings, depth+1)
	if state != "STATIC_SCHEMA_EXACT" {
		return cliArgSchemaContents{}, state, reason
	}
	if source != "" {
		contents.sources = append(contents.sources, source)
	}
	for _, statement := range helper.Body.List[1 : len(helper.Body.List)-1] {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
			return cliArgSchemaContents{}, "UNKNOWN", "Args helper builder contains an unsupported statement"
		}
		field, ok := assignment.Lhs[0].(*ast.SelectorExpr)
		if !ok {
			return cliArgSchemaContents{}, "UNKNOWN", "Args helper assignment target is not a local schema field"
		}
		base, baseOK := field.X.(*ast.Ident)
		call, callOK := assignment.Rhs[0].(*ast.CallExpr)
		if !baseOK || base.Name != variable.Name || !callOK || !cliIdentCallName(call.Fun, "append") || len(call.Args) != 2 {
			return cliArgSchemaContents{}, "UNKNOWN", "Args helper builder supports only direct append to local schema slices"
		}
		appendField, appendOK := call.Args[0].(*ast.SelectorExpr)
		if !appendOK {
			return cliArgSchemaContents{}, "UNKNOWN", "Args helper append does not target a local schema field"
		}
		appendBase, appendBaseOK := appendField.X.(*ast.Ident)
		if !appendBaseOK || appendBase.Name != variable.Name || appendField.Sel.Name != field.Sel.Name {
			return cliArgSchemaContents{}, "UNKNOWN", "Args helper append target does not match its local schema field"
		}
		row, rowOK := call.Args[1].(*ast.CompositeLit)
		if !rowOK {
			return cliArgSchemaContents{}, "UNKNOWN", "Args helper append element is not a direct literal"
		}
		switch field.Sel.Name {
		case "Options":
			option, rowState, rowReason := cliReadOptionWithBindings(row, bindings)
			if rowState != "STATIC_SCHEMA_EXACT" {
				return cliArgSchemaContents{}, rowState, rowReason
			}
			contents.options = append(contents.options, option)
		case "Positionals":
			positional, rowState, rowReason := cliReadPositionalWithBindings(row, bindings)
			if rowState != "STATIC_SCHEMA_EXACT" {
				return cliArgSchemaContents{}, rowState, rowReason
			}
			contents.positionals = append(contents.positionals, positional)
		default:
			return cliArgSchemaContents{}, "PARTIAL", "Args helper appends unsupported schema field " + field.Sel.Name
		}
	}
	returned, ok := helper.Body.List[len(helper.Body.List)-1].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return cliArgSchemaContents{}, "UNKNOWN", "Args helper builder does not end in one return"
	}
	returnedVariable, ok := returned.Results[0].(*ast.Ident)
	if !ok || returnedVariable.Name != variable.Name {
		return cliArgSchemaContents{}, "UNKNOWN", "Args helper returns a value other than its analyzed schema"
	}
	state, reason = cliValidateArgSchema(contents.positionals, contents.options)
	return contents, state, reason
}

func cliResolveSchemaHelperCall(caller cliSourceUnit, expression ast.Expr, packageUnits []cliSourceUnit, bindings map[string]string, depth int) (cliArgSchemaContents, string, string, string) {
	call, ok := expression.(*ast.CallExpr)
	if !ok {
		return cliArgSchemaContents{}, "", "UNKNOWN", "schema initializer is not a supported helper call"
	}
	name, ok := cliLocalFunctionName(call.Fun)
	if !ok {
		return cliArgSchemaContents{}, "", "UNKNOWN", "schema initializer is not a same-package helper call"
	}
	var targetUnit cliSourceUnit
	var target *ast.FuncDecl
	count := 0
	for _, candidate := range packageUnits {
		if filepath.Dir(candidate.path) != filepath.Dir(caller.path) {
			continue
		}
		for _, declaration := range candidate.file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok && function.Recv == nil && function.Name.Name == name {
				count++
				targetUnit, target = candidate, function
			}
		}
	}
	if count != 1 {
		if count > 1 {
			return cliArgSchemaContents{}, "", "AMBIGUOUS", "schema helper has multiple declarations"
		}
		return cliArgSchemaContents{}, "", "UNKNOWN", "schema helper declaration is unavailable in the selected package"
	}
	if target.Type.Params == nil || target.Type.Params.NumFields() != len(call.Args) {
		return cliArgSchemaContents{}, "", "UNKNOWN", "schema helper arguments do not match its literal parameters"
	}
	nextBindings := map[string]string{}
	for key, value := range bindings {
		nextBindings[key] = value
	}
	parameterIndex := 0
	for _, field := range target.Type.Params.List {
		if len(field.Names) != 1 || parameterIndex >= len(call.Args) {
			return cliArgSchemaContents{}, "", "UNKNOWN", "schema helper parameters must be individually named string literals"
		}
		value, ok := cliStringBound(call.Args[parameterIndex], bindings)
		if !ok {
			return cliArgSchemaContents{}, "", "UNKNOWN", "schema helper arguments must be literal strings"
		}
		nextBindings[field.Names[0].Name] = value
		parameterIndex++
	}
	contents, state, reason := cliInterpretArgsHelper(targetUnit, target, nextBindings, packageUnits, depth)
	if state != "STATIC_SCHEMA_EXACT" {
		return cliArgSchemaContents{}, "", state, reason
	}
	note := fmt.Sprintf("%s:%d#sha256:%s", targetUnit.path, targetUnit.fset.Position(target.Pos()).Line, targetUnit.hash)
	return contents, note, state, reason
}

func cliIdentCallName(expression ast.Expr, name string) bool {
	identifier, ok := expression.(*ast.Ident)
	return ok && identifier.Name == name
}

func cliLocalFunctionName(expr ast.Expr) (string, bool) {
	id, ok := expr.(*ast.Ident)
	if !ok || id.Name == "" {
		return "", false
	}
	return id.Name, true
}

type cliSchemaPositional struct {
	name, description    string
	required, repeatable bool
}
type cliSchemaOption struct {
	name, description string
	aliases           []string
	valueName         string
	repeatable        bool
}

func cliIsArgSchemaType(expr ast.Expr, file *ast.File) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "ArgSchema" {
		return false
	}
	qualifier, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	for _, imported := range file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil || !strings.HasSuffix(path, "/internal/cli/commandtree") {
			continue
		}
		name := "commandtree"
		if imported.Name != nil {
			name = imported.Name.Name
		}
		return qualifier.Name == name
	}
	return false
}

func cliReadArgSchema(lit *ast.CompositeLit) ([]cliSchemaPositional, []cliSchemaOption, string, string) {
	return cliReadArgSchemaWithBindings(lit, nil)
}

func cliReadArgSchemaWithBindings(lit *ast.CompositeLit, bindings map[string]string) ([]cliSchemaPositional, []cliSchemaOption, string, string) {
	var positionals []cliSchemaPositional
	var options []cliSchemaOption
	seenFields := map[string]bool{}
	for _, raw := range lit.Elts {
		field, ok := raw.(*ast.KeyValueExpr)
		if !ok {
			return nil, nil, "UNKNOWN", "ArgSchema contains an unkeyed element"
		}
		key, ok := field.Key.(*ast.Ident)
		if !ok {
			return nil, nil, "UNKNOWN", "ArgSchema field name is dynamic"
		}
		if seenFields[key.Name] {
			return nil, nil, "AMBIGUOUS", "ArgSchema repeats " + key.Name
		}
		seenFields[key.Name] = true
		switch key.Name {
		case "Positionals":
			rows, ok := field.Value.(*ast.CompositeLit)
			if !ok {
				return nil, nil, "UNKNOWN", "Positionals is not a direct literal"
			}
			for _, item := range rows.Elts {
				row, ok := item.(*ast.CompositeLit)
				if !ok {
					return nil, nil, "PARTIAL", "Positionals contains an unresolved helper or element"
				}
				p, state, reason := cliReadPositionalWithBindings(row, bindings)
				if state != "STATIC_SCHEMA_EXACT" {
					return nil, nil, state, reason
				}
				positionals = append(positionals, p)
			}
		case "Options":
			rows, ok := field.Value.(*ast.CompositeLit)
			if !ok {
				return nil, nil, "UNKNOWN", "Options is not a direct literal"
			}
			for _, item := range rows.Elts {
				row, ok := item.(*ast.CompositeLit)
				if !ok {
					return nil, nil, "PARTIAL", "Options contains an unresolved helper or element"
				}
				o, state, reason := cliReadOptionWithBindings(row, bindings)
				if state != "STATIC_SCHEMA_EXACT" {
					return nil, nil, state, reason
				}
				options = append(options, o)
			}
		default:
			return nil, nil, "PARTIAL", "unsupported ArgSchema field " + key.Name
		}
	}
	state, reason := cliValidateArgSchema(positionals, options)
	if state != "STATIC_SCHEMA_EXACT" {
		return nil, nil, state, reason
	}
	return positionals, options, "STATIC_SCHEMA_EXACT", "literal fields only"
}

func cliValidateArgSchema(positionals []cliSchemaPositional, options []cliSchemaOption) (string, string) {
	spellings := map[string]bool{}
	for _, option := range options {
		for _, spelling := range append([]string{option.name}, option.aliases...) {
			if spelling == "" {
				return "UNKNOWN", "option spelling is empty"
			}
			if spellings[spelling] {
				return "AMBIGUOUS", "duplicate option spelling " + spelling
			}
			spellings[spelling] = true
		}
	}
	for i, p := range positionals {
		if p.name == "" {
			return "UNKNOWN", "positional name is empty"
		}
		if p.repeatable && i != len(positionals)-1 {
			return "INVALID", "repeatable positional is not final"
		}
	}
	return "STATIC_SCHEMA_EXACT", ""
}

func cliReadPositional(row *ast.CompositeLit) (cliSchemaPositional, string, string) {
	return cliReadPositionalWithBindings(row, nil)
}

func cliReadPositionalWithBindings(row *ast.CompositeLit, bindings map[string]string) (cliSchemaPositional, string, string) {
	var out cliSchemaPositional
	seen := map[string]bool{}
	for _, raw := range row.Elts {
		kv, ok := raw.(*ast.KeyValueExpr)
		if !ok {
			return out, "UNKNOWN", "positional fields are not keyed"
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			return out, "UNKNOWN", "positional field is dynamic"
		}
		if seen[key.Name] {
			return out, "AMBIGUOUS", "positional repeats " + key.Name
		}
		seen[key.Name] = true
		switch key.Name {
		case "Name":
			value, ok := cliStringBound(kv.Value, bindings)
			if !ok {
				return out, "UNKNOWN", "positional Name is not a string literal"
			}
			out.name = value
		case "Description":
			value, ok := cliStringBound(kv.Value, bindings)
			if !ok {
				return out, "UNKNOWN", "positional Description is not a string literal"
			}
			out.description = value
		case "Required":
			value, ok := cliBool(kv.Value)
			if !ok {
				return out, "UNKNOWN", "positional Required is not a bool literal"
			}
			out.required = value
		case "Repeatable":
			value, ok := cliBool(kv.Value)
			if !ok {
				return out, "UNKNOWN", "positional Repeatable is not a bool literal"
			}
			out.repeatable = value
		default:
			return out, "PARTIAL", "unsupported positional field " + key.Name
		}
	}
	return out, "STATIC_SCHEMA_EXACT", ""
}

func cliReadOption(row *ast.CompositeLit) (cliSchemaOption, string, string) {
	return cliReadOptionWithBindings(row, nil)
}

func cliReadOptionWithBindings(row *ast.CompositeLit, bindings map[string]string) (cliSchemaOption, string, string) {
	var out cliSchemaOption
	seen := map[string]bool{}
	for _, raw := range row.Elts {
		kv, ok := raw.(*ast.KeyValueExpr)
		if !ok {
			return out, "UNKNOWN", "option fields are not keyed"
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			return out, "UNKNOWN", "option field is dynamic"
		}
		if seen[key.Name] {
			return out, "AMBIGUOUS", "option repeats " + key.Name
		}
		seen[key.Name] = true
		switch key.Name {
		case "Name":
			value, ok := cliStringBound(kv.Value, bindings)
			if !ok {
				return out, "UNKNOWN", "option Name is not a string literal"
			}
			out.name = value
		case "Description":
			value, ok := cliStringBound(kv.Value, bindings)
			if !ok {
				return out, "UNKNOWN", "option Description is not a string literal"
			}
			out.description = value
		case "Aliases":
			rows, ok := kv.Value.(*ast.CompositeLit)
			if !ok {
				return out, "UNKNOWN", "Aliases is not a direct literal"
			}
			for _, item := range rows.Elts {
				value, ok := cliString(item)
				if !ok {
					return out, "UNKNOWN", "alias is not a string literal"
				}
				out.aliases = append(out.aliases, value)
			}
		case "ValueName":
			value, ok := cliString(kv.Value)
			if !ok {
				return out, "UNKNOWN", "option ValueName is not a string literal"
			}
			out.valueName = value
		case "Repeatable":
			value, ok := cliBool(kv.Value)
			if !ok {
				return out, "UNKNOWN", "option Repeatable is not a bool literal"
			}
			out.repeatable = value
		default:
			return out, "PARTIAL", "unsupported option field " + key.Name
		}
	}
	if out.name == "" {
		return out, "UNKNOWN", "option Name is empty"
	}
	return out, "STATIC_SCHEMA_EXACT", ""
}

func cliString(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	return value, err == nil
}

func cliStringBound(expr ast.Expr, bindings map[string]string) (string, bool) {
	if id, ok := expr.(*ast.Ident); ok {
		value, exists := bindings[id.Name]
		return value, exists
	}
	return cliString(expr)
}
func cliBool(expr ast.Expr) (bool, bool) {
	ident, ok := expr.(*ast.Ident)
	if !ok || (ident.Name != "true" && ident.Name != "false") {
		return false, false
	}
	return ident.Name == "true", true
}
