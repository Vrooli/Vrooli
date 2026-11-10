package lifecycleinference

import (
	"context"
	"go/ast"
	"go/token"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

type functionFlow struct {
	persistence []Evidence
	readModels  []Evidence
	transient   []Evidence
	writes      []Evidence
	otherUses   []Evidence
	origins     []Evidence
}

func scenarioTypeGraph(files []parsedSource, candidate Candidate, candidateImport string) map[string]bool {
	contained := map[string]bool{candidate.TypeName: true}
	for changed := true; changed; {
		changed = false
		for _, source := range files {
			if source.packageDir != filepath.ToSlash(candidate.Package) {
				continue
			}
			for _, declaration := range source.file.Decls {
				gen, ok := declaration.(*ast.GenDecl)
				if !ok || gen.Tok != token.TYPE {
					continue
				}
				for _, spec := range gen.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok || contained[typeSpec.Name.Name] {
						continue
					}
					if astContainsCandidateType(typeSpec.Type, source, candidate, candidateImport, contained) {
						contained[typeSpec.Name.Name] = true
						changed = true
					}
				}
			}
		}
	}
	return contained
}

func astContainsCandidateType(expr ast.Expr, source parsedSource, candidate Candidate, candidateImport string, contained map[string]bool) bool {
	switch node := expr.(type) {
	case *ast.Ident:
		return source.packageDir == filepath.ToSlash(candidate.Package) && contained[node.Name]
	case *ast.SelectorExpr:
		return source.imports[exprIdent(node.X)] == candidateImport && contained[node.Sel.Name]
	case *ast.StarExpr:
		return astContainsCandidateType(node.X, source, candidate, candidateImport, contained)
	case *ast.ArrayType:
		return astContainsCandidateType(node.Elt, source, candidate, candidateImport, contained)
	case *ast.MapType:
		return astContainsCandidateType(node.Key, source, candidate, candidateImport, contained) || astContainsCandidateType(node.Value, source, candidate, candidateImport, contained)
	case *ast.StructType:
		for _, field := range node.Fields.List {
			if astContainsCandidateType(field.Type, source, candidate, candidateImport, contained) {
				return true
			}
		}
		return false
	case *ast.ChanType:
		return astContainsCandidateType(node.Value, source, candidate, candidateImport, contained)
	case *ast.ParenExpr:
		return astContainsCandidateType(node.X, source, candidate, candidateImport, contained)
	default:
		return false
	}
}

func analyzeFunctionFlow(ctx context.Context, fn *ast.FuncDecl, source parsedSource, allFiles []parsedSource, candidate Candidate, candidateImport string, typeGraph map[string]bool) functionFlow {
	flow := functionFlow{}
	variables := map[string]bool{}
	persistedFieldValues := map[string]bool{}
	variableTypes := map[string]ast.Expr{}
	structFields := map[string]map[string]ast.Expr{}
	if source.packageDir == filepath.ToSlash(candidate.Package) {
		for _, declaration := range source.file.Decls {
			gen, ok := declaration.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structure, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				fields := map[string]ast.Expr{}
				for _, field := range structure.Fields.List {
					for _, name := range field.Names {
						fields[name.Name] = field.Type
					}
				}
				structFields[typeSpec.Name.Name] = fields
			}
		}
	}
	addVariable := func(name string, expr ast.Expr) bool {
		if name == "" || name == "_" {
			return false
		}
		changed := false
		if expr != nil {
			variableTypes[name] = expr
			if astContainsCandidateType(expr, source, candidate, candidateImport, typeGraph) && !variables[name] {
				variables[name], changed = true, true
			}
		}
		return changed
	}
	if fn.Recv != nil {
		for _, field := range fn.Recv.List {
			for _, name := range field.Names {
				if addVariable(name.Name, field.Type) {
					evidence := sourceEvidence(source.fset, source.path, field, "typed receiver contains candidate type")
					flow.origins = append(flow.origins, evidence)
					flow.otherUses = append(flow.otherUses, evidence)
				}
			}
		}
	}
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			for _, name := range field.Names {
				if addVariable(name.Name, field.Type) {
					evidence := sourceEvidence(source.fset, source.path, field, "typed parameter contains candidate type")
					flow.origins = append(flow.origins, evidence)
					flow.otherUses = append(flow.otherUses, evidence)
				}
			}
		}
	}
	serializers := fieldSerializerFunctions(allFiles)

	for pass := 0; pass < 4; pass++ {
		changed := false
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if ctx.Err() != nil {
				return false
			}
			switch current := node.(type) {
			case *ast.RangeStmt:
				if expressionCarriesCandidate(current.X, source, candidate, candidateImport, typeGraph, variables, structFields) {
					if current.Key != nil {
						if name, ok := current.Key.(*ast.Ident); ok {
							changed = addVariable(name.Name, nil) || changed
						}
					}
					if name, ok := current.Value.(*ast.Ident); ok {
						if name.Name != "" && name.Name != "_" && !variables[name.Name] {
							variables[name.Name] = true
							flow.origins = append(flow.origins, sourceEvidence(source.fset, source.path, current, "range value is candidate-containing"))
							changed = true
						}
					}
				}
			case *ast.ValueSpec:
				for index, name := range current.Names {
					var value ast.Expr
					if index < len(current.Values) {
						value = current.Values[index]
					}
					if expressionCarriesCandidate(value, source, candidate, candidateImport, typeGraph, variables, structFields) || astContainsCandidateType(current.Type, source, candidate, candidateImport, typeGraph) {
						changed = addVariable(name.Name, current.Type) || changed
						if !variables[name.Name] {
							variables[name.Name] = true
							changed = true
						}
					}
					if expressionPersistsCandidateField(value, source, allFiles, candidate, candidateImport, typeGraph, variables, structFields, persistedFieldValues, serializers) && !persistedFieldValues[name.Name] {
						persistedFieldValues[name.Name] = true
						changed = true
					}
				}
			case *ast.AssignStmt:
				for index, left := range current.Lhs {
					if index >= len(current.Rhs) {
						continue
					}
					if expressionPersistsCandidateField(current.Rhs[index], source, allFiles, candidate, candidateImport, typeGraph, variables, structFields, persistedFieldValues, serializers) {
						if name, ok := left.(*ast.Ident); ok && !persistedFieldValues[name.Name] {
							persistedFieldValues[name.Name] = true
							changed = true
						}
					}
					if !expressionCarriesCandidate(current.Rhs[index], source, candidate, candidateImport, typeGraph, variables, structFields) {
						continue
					}
					if name, ok := left.(*ast.Ident); ok {
						changed = addVariable(name.Name, variableTypes[name.Name]) || changed
						if !variables[name.Name] {
							variables[name.Name] = true
							changed = true
						}
					}
				}
			}
			return true
		})
		if !changed {
			break
		}
	}

	queryVars := map[string]bool{}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if ctx.Err() != nil {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		method := selectorMethod(call.Fun)
		receiver := selectorReceiver(call.Fun)
		if isSQLReadSink(method) && isSQLReceiver(receiver, source, variableTypes, structFields) {
			if name, ok := parentAssignedName(fn.Body, call); ok {
				queryVars[name] = true
			}
			return true
		}
		if isSQLWriteSink(method) && isSQLReceiver(receiver, source, variableTypes, structFields) && expressionListPersistsCandidateField(call.Args, source, allFiles, candidate, candidateImport, typeGraph, variables, structFields, persistedFieldValues, serializers) {
			flow.persistence = append(flow.persistence, flow.origins...)
			flow.persistence = append(flow.persistence, sourceEvidence(source.fset, source.path, call, "candidate lifecycle field value or serialized containing record passed to database/sql."+method))
			return true
		}
		if method == "Scan" && queryVars[exprIdent(receiver)] && expressionListContainsCandidateField(call.Args, source, candidate, candidateImport, typeGraph, variables, structFields) {
			flow.readModels = append(flow.readModels, sourceEvidence(source.fset, source.path, call, "database/sql row scan populates the candidate lifecycle field"))
			return true
		}
		if isConnectResponse(call, source) && expressionListCarriesCandidate(call.Args, source, candidate, candidateImport, typeGraph, variables, structFields) {
			flow.transient = append(flow.transient, sourceEvidence(source.fset, source.path, call, "candidate-containing value passed to Connect response constructor"))
		}
		return true
	})

	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if ctx.Err() != nil {
			return false
		}
		switch current := node.(type) {
		case *ast.AssignStmt:
			for _, left := range current.Lhs {
				selector, ok := left.(*ast.SelectorExpr)
				if ok && selector.Sel.Name == candidate.FieldName && expressionCarriesCandidate(selector.X, source, candidate, candidateImport, typeGraph, variables, structFields) {
					flow.writes = append(flow.writes, sourceEvidence(source.fset, source.path, current, "candidate lifecycle field assignment"))
				}
			}
		case *ast.CompositeLit:
			if astContainsCandidateType(current.Type, source, candidate, candidateImport, typeGraph) {
				for _, element := range current.Elts {
					pair, ok := element.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					key, keyOK := pair.Key.(*ast.Ident)
					if keyOK && key.Name == candidate.FieldName {
						flow.writes = append(flow.writes, sourceEvidence(source.fset, source.path, pair, "keyed candidate lifecycle field initialization"))
					}
				}
			}
		case *ast.CallExpr:
			if expressionListCarriesCandidate(current.Args, source, candidate, candidateImport, typeGraph, variables, structFields) {
				flow.otherUses = append(flow.otherUses, sourceEvidence(source.fset, source.path, current, "candidate-containing value used by an unsupported call"))
			}
		}
		return true
	})
	return flow
}

func expressionCarriesCandidate(expr ast.Expr, source parsedSource, candidate Candidate, candidateImport string, typeGraph map[string]bool, variables map[string]bool, fields map[string]map[string]ast.Expr) bool {
	if expr == nil {
		return false
	}
	if astContainsCandidateType(expr, source, candidate, candidateImport, typeGraph) {
		return true
	}
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		if found {
			return false
		}
		switch current := node.(type) {
		case *ast.Ident:
			found = variables[current.Name]
		case *ast.SelectorExpr:
			if variables[exprIdent(current.X)] || current.Sel.Name == candidate.FieldName && variables[exprIdent(current.X)] {
				found = true
			}
		case *ast.CallExpr:
			for _, argument := range current.Args {
				if expressionCarriesCandidate(argument, source, candidate, candidateImport, typeGraph, variables, fields) {
					found = true
					break
				}
			}
		case *ast.CompositeLit:
			found = astContainsCandidateType(current.Type, source, candidate, candidateImport, typeGraph)
		}
		return !found
	})
	return found
}

func expressionListCarriesCandidate(expressions []ast.Expr, source parsedSource, candidate Candidate, candidateImport string, typeGraph map[string]bool, variables map[string]bool, fields map[string]map[string]ast.Expr) bool {
	for _, expression := range expressions {
		if expressionCarriesCandidate(expression, source, candidate, candidateImport, typeGraph, variables, fields) {
			return true
		}
	}
	return false
}

func expressionListContainsCandidateField(expressions []ast.Expr, source parsedSource, candidate Candidate, candidateImport string, typeGraph map[string]bool, variables map[string]bool, fields map[string]map[string]ast.Expr) bool {
	for _, expression := range expressions {
		found := false
		ast.Inspect(expression, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if ok && selector.Sel.Name == candidate.FieldName && expressionCarriesCandidate(selector.X, source, candidate, candidateImport, typeGraph, variables, fields) {
				found = true
				return false
			}
			return !found
		})
		if found {
			return true
		}
	}
	return false
}

func expressionPersistsCandidateField(expression ast.Expr, source parsedSource, allFiles []parsedSource, candidate Candidate, candidateImport string, typeGraph map[string]bool, variables map[string]bool, fields map[string]map[string]ast.Expr, fieldValues map[string]bool, serializers map[string]bool) bool {
	if expression == nil {
		return false
	}
	if expressionListContainsCandidateField([]ast.Expr{expression}, source, candidate, candidateImport, typeGraph, variables, fields) {
		return true
	}
	if ident, ok := expression.(*ast.Ident); ok && fieldValues[ident.Name] {
		return true
	}
	call, ok := expression.(*ast.CallExpr)
	if !ok {
		return false
	}
	name := selectorMethod(call.Fun)
	if ident, ok := call.Fun.(*ast.Ident); ok {
		name = ident.Name
	}
	if !serializers[name] || candidateFieldExcludedFromRecord(allFiles, candidate) {
		return false
	}
	return expressionListCarriesCandidate(call.Args, source, candidate, candidateImport, typeGraph, variables, fields)
}

func expressionListPersistsCandidateField(expressions []ast.Expr, source parsedSource, allFiles []parsedSource, candidate Candidate, candidateImport string, typeGraph map[string]bool, variables map[string]bool, fields map[string]map[string]ast.Expr, fieldValues map[string]bool, serializers map[string]bool) bool {
	for _, expression := range expressions {
		if expressionPersistsCandidateField(expression, source, allFiles, candidate, candidateImport, typeGraph, variables, fields, fieldValues, serializers) {
			return true
		}
	}
	return false
}

// fieldSerializerFunctions recognizes source-local helpers which pass one of
// their parameters directly to encoding/json.Marshal. Unknown/custom codecs
// are intentionally not treated as field persistence evidence.
func fieldSerializerFunctions(files []parsedSource) map[string]bool {
	serializers := map[string]bool{}
	for _, source := range files {
		for _, declaration := range source.file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Type.Params == nil {
				continue
			}
			parameters := map[string]bool{}
			for _, field := range fn.Type.Params.List {
				for _, name := range field.Names {
					parameters[name.Name] = true
				}
			}
			found := false
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok || len(call.Args) == 0 {
					return !found
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "Marshal" || source.imports[exprIdent(selector.X)] != "encoding/json" {
					return !found
				}
				if ident, ok := call.Args[0].(*ast.Ident); ok && parameters[ident.Name] {
					found = true
				}
				return !found
			})
			if found {
				serializers[fn.Name.Name] = true
			}
		}
	}
	return serializers
}

func candidateFieldExcludedFromRecord(files []parsedSource, candidate Candidate) bool {
	for _, source := range files {
		if source.packageDir != filepath.ToSlash(candidate.Package) {
			continue
		}
		for _, declaration := range source.file.Decls {
			gen, ok := declaration.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != candidate.TypeName {
					continue
				}
				structure, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, field := range structure.Fields.List {
					for _, name := range field.Names {
						if name.Name != candidate.FieldName || field.Tag == nil {
							continue
						}
						raw, err := strconv.Unquote(field.Tag.Value)
						if err != nil {
							return true
						}
						for _, key := range []string{"json", "db", "gorm", "bson"} {
							if strings.Split(reflect.StructTag(raw).Get(key), ",")[0] == "-" {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}

func isSQLWriteSink(method string) bool {
	switch method {
	case "Exec", "ExecContext":
		return true
	default:
		return false
	}
}

func isSQLReadSink(method string) bool {
	switch method {
	case "Query", "QueryContext", "QueryRow", "QueryRowContext":
		return true
	default:
		return false
	}
}

func isSQLReceiver(receiver ast.Expr, source parsedSource, variableTypes map[string]ast.Expr, fields map[string]map[string]ast.Expr) bool {
	var typeExpr ast.Expr
	switch value := receiver.(type) {
	case *ast.Ident:
		typeExpr = variableTypes[value.Name]
	case *ast.SelectorExpr:
		if base, ok := value.X.(*ast.Ident); ok {
			if receiverType := variableTypes[base.Name]; receiverType != nil {
				typeName := typeName(receiverType)
				if typeName != "" {
					typeExpr = fields[typeName][value.Sel.Name]
				}
			}
		}
	}
	selector, ok := typeExpr.(*ast.StarExpr)
	if !ok {
		return false
	}
	qualified, ok := selector.X.(*ast.SelectorExpr)
	if !ok || source.imports[exprIdent(qualified.X)] != "database/sql" {
		return false
	}
	switch qualified.Sel.Name {
	case "DB", "Tx", "Conn", "Stmt":
		return true
	default:
		return false
	}
}

func isConnectResponse(call *ast.CallExpr, source parsedSource) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "NewResponse" && source.imports[exprIdent(selector.X)] == "connectrpc.com/connect"
}

func selectorMethod(expr ast.Expr) string {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	return selector.Sel.Name
}

func selectorReceiver(expr ast.Expr) ast.Expr {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	return selector.X
}

func exprIdent(expr ast.Expr) string {
	ident, _ := expr.(*ast.Ident)
	if ident == nil {
		return ""
	}
	return ident.Name
}

func parentAssignedName(body *ast.BlockStmt, target ast.Node) (string, bool) {
	name := ""
	ast.Inspect(body, func(node ast.Node) bool {
		switch current := node.(type) {
		case *ast.ValueSpec:
			for index, value := range current.Values {
				if value == target && index < len(current.Names) {
					name = current.Names[index].Name
					return false
				}
			}
		case *ast.AssignStmt:
			for index, value := range current.Rhs {
				if value == target && index < len(current.Lhs) {
					name = exprIdent(current.Lhs[index])
					return false
				}
			}
		}
		return name == ""
	})
	return name, name != ""
}

func unknownWithEvidence(unknown PersistenceClassification, reason string, evidence []Evidence) PersistenceClassification {
	unknown.Reason = reason
	unknown.Evidence = evidence
	return unknown
}

func classificationWithScope(class PersistenceClass, reason string, source scenarioSource, evidence []Evidence) PersistenceClassification {
	result := classification(class, reason, evidence)
	result.Scope = source.scope
	result.FilesScanned = len(source.files)
	result.ScanComplete = source.loadError == "" && len(source.parseErrors) == 0
	return result
}

func candidateDeclarationOnly(ctx context.Context, files []parsedSource, candidate Candidate, candidateImport string, typeGraph map[string]bool) bool {
	for _, source := range files {
		if ctx.Err() != nil {
			return false
		}
		for _, declaration := range source.file.Decls {
			gen, ok := declaration.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if ok && typeSpec.Name.Name != candidate.TypeName && astContainsCandidateType(typeSpec.Type, source, candidate, candidateImport, typeGraph) {
					return false
				}
			}
		}
		for _, declaration := range source.file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			flow := analyzeFunctionFlow(ctx, fn, source, files, candidate, candidateImport, typeGraph)
			if len(flow.otherUses)+len(flow.writes)+len(flow.persistence)+len(flow.readModels)+len(flow.transient) > 0 {
				return false
			}
		}
	}
	return true
}

func candidateHasStorageTag(files []parsedSource, candidate Candidate) bool {
	for _, source := range files {
		if source.packageDir != filepath.ToSlash(candidate.Package) {
			continue
		}
		for _, declaration := range source.file.Decls {
			gen, ok := declaration.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != candidate.TypeName {
					continue
				}
				structure, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, field := range structure.Fields.List {
					if field.Tag == nil {
						continue
					}
					for _, name := range field.Names {
						if name.Name == candidate.FieldName {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func sourceEvidence(fset *token.FileSet, path string, node ast.Node, text string) Evidence {
	position := fset.Position(node.Pos())
	return Evidence{Path: filepath.ToSlash(path), Line: position.Line, Text: text}
}

func _unusedString(s string) string { return strings.TrimSpace(s) }
