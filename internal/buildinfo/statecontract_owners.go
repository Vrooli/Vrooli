package buildinfo

import (
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// ownerAnalysis is the stamp-time classification and check of one precise
// (or constants-stamped) owner's source packages.
type ownerAnalysis struct {
	values     map[string]string // stored untyped string constant -> digest
	versions   []string          // format-version constants in use
	layouts    []string          // canonical time layouts formatted or parsed
	literals   []string          // stored-field literal candidates (literalCandidates)
	violations []string
}

// analyzeOwnerPackages classifies an owner's packages. When vocabulary is
// true, every package-level untyped string constant not classified as not
// stored becomes a stored value. Version writes and time layouts are checked
// for every owner package.
func analyzeOwnerPackages(root string, dirs []string, notStored map[string][]string, vocabulary bool) (ownerAnalysis, error) {
	result := ownerAnalysis{values: map[string]string{}}
	layouts := map[string]bool{}
	for _, dir := range dirs {
		fset := token.NewFileSet()
		sources, err := loadGoSources(fset, filepath.Join(root, filepath.FromSlash(dir)), "")
		if err != nil {
			return result, err
		}
		pkg := indexPackage(fset, root, sources)
		importPath, err := sourceImportPath(root, dir)
		if err != nil {
			return result, err
		}
		uses := fieldUses(pkg, importPath, sources)
		evidence := newVocabularyEvidence()
		typedFields(pkg, importPath, evidence.typed)
		evidence.add(importPath, uses)
		result.literals = append(result.literals, literalCandidates(pkg, importPath, uses, evidence, nil)...)
		skip := map[string]bool{}
		for _, name := range notStored[dir] {
			skip[name] = true
		}
		versionFields, versionPositions := versionFieldNames(sources)
		for _, source := range sources {
			for _, decl := range source.file.Decls {
				if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.CONST && vocabulary {
					for _, spec := range gen.Specs {
						value := spec.(*ast.ValueSpec)
						if value.Type != nil {
							continue
						}
						for i, name := range value.Names {
							if i >= len(value.Values) || !stringExpr(value.Values[i]) || skip[name.Name] {
								continue
							}
							items := append([]string{canonicalSource(fset, source.src, value.Values[i])}, pkg.closure(value.Values[i])...)
							sort.Strings(items)
							sum := sha256.Sum256([]byte(strings.Join(items, "\n")))
							result.values[name.Name] = hex.EncodeToString(sum[:16])
						}
					}
				}
			}
			ast.Inspect(source.file, func(node ast.Node) bool {
				switch node := node.(type) {
				case *ast.CompositeLit:
					// Another package's type carries that package's version
					// (a request protocol, a foreign envelope), not this
					// owner's format.
					if _, foreign := node.Type.(*ast.SelectorExpr); foreign {
						return true
					}
					for _, element := range node.Elts {
						if pair, ok := element.(*ast.KeyValueExpr); ok {
							if key, ok := pair.Key.(*ast.Ident); ok && versionFields[key.Name] {
								result.checkVersion(pkg, pair.Value, versionFields)
							}
						}
					}
					// Positional literals of a type with a version field.
					if ident, ok := node.Type.(*ast.Ident); ok && len(node.Elts) > 0 {
						if _, keyed := node.Elts[0].(*ast.KeyValueExpr); !keyed {
							for index := range versionPositions[ident.Name] {
								if index < len(node.Elts) {
									result.checkVersion(pkg, node.Elts[index], versionFields)
								}
							}
						}
					}
				case *ast.AssignStmt:
					for i, lhs := range node.Lhs {
						if field := selectorName(lhs); versionFields[field] && i < len(node.Rhs) {
							result.checkVersion(pkg, node.Rhs[i], versionFields)
						}
					}
				case *ast.BinaryExpr:
					switch node.Op {
					case token.EQL, token.NEQ, token.LSS, token.GTR, token.LEQ, token.GEQ:
						if versionFields[selectorName(node.X)] {
							result.checkVersion(pkg, node.Y, versionFields)
						} else if versionFields[selectorName(node.Y)] {
							result.checkVersion(pkg, node.X, versionFields)
						}
					}
				case *ast.CallExpr:
					if location := storedLiteralCall(pkg, node); location != "" {
						result.violations = append(result.violations, "stored-literal@"+location)
					}
					if layout := timeLayoutArgument(node); layout != nil {
						items := append([]string{canonicalSource(fset, source.src, layout)}, pkg.closure(layout)...)
						sort.Strings(items)
						layouts[strings.Join(items, "\n")] = true
					}
				}
				return true
			})
		}
	}
	for layout := range layouts {
		result.layouts = append(result.layouts, layout)
	}
	sort.Strings(result.layouts)
	sort.Strings(result.versions)
	result.versions = compactStrings(result.versions)
	result.literals = compactStrings(sortedCopy(result.literals))
	return result, nil
}

// fieldUse is one place a package writes, compares or switches on a struct
// field with a value.
type fieldUse struct {
	// key is the field's type, "importpath.Type", or "importpath.*" when the
	// receiver type is not syntactically known (any type of the scanning
	// package with that field).
	key, field string
	write      bool
	named      bool          // the value is a named constant
	literal    *ast.BasicLit // the value is this non-empty string literal
	selector   bool          // an assignment, comparison or switch rather than a composite literal
}

func nonEmptyStringLiteral(expr ast.Expr) *ast.BasicLit {
	if literal, ok := expr.(*ast.BasicLit); ok && literal.Kind == token.STRING && literal.Value != `""` && literal.Value != "``" {
		return literal
	}
	return nil
}

// fieldUses lists a package's valued field uses. A composite literal names its
// type; a selector on a receiver, parameter or variable declared with a named
// type resolves to it; any other selector is recorded under "importpath.*".
func fieldUses(pkg codecPackage, importPath string, sources []goSource) []fieldUse {
	var uses []fieldUse
	for _, source := range sources {
		imports := pkg.imports[source.path]
		typeKey := func(expr ast.Expr) string {
			if star, ok := expr.(*ast.StarExpr); ok {
				expr = star.X
			}
			switch typ := expr.(type) {
			case *ast.Ident:
				if decl, ok := pkg.types[typ.Name]; ok {
					// An alias to another package's type is that type.
					if spec, ok := decl.node.(*ast.TypeSpec); ok && spec.Assign.IsValid() {
						declaring := pkg.imports[pkg.fset.Position(spec.Pos()).Filename]
						if target, ok := spec.Type.(*ast.SelectorExpr); ok {
							if alias, ok := target.X.(*ast.Ident); ok && declaring[alias.Name] != "" {
								return declaring[alias.Name] + "." + target.Sel.Name
							}
						}
					}
					return importPath + "." + typ.Name
				}
			case *ast.SelectorExpr:
				if alias, ok := typ.X.(*ast.Ident); ok && imports[alias.Name] != "" {
					return imports[alias.Name] + "." + typ.Sel.Name
				}
			}
			return ""
		}
		named := func(expr ast.Expr) bool {
			switch expr := expr.(type) {
			case *ast.Ident:
				decl, ok := pkg.values[expr.Name]
				return ok && strings.HasPrefix(decl.text, "const ")
			case *ast.SelectorExpr:
				alias, ok := expr.X.(*ast.Ident)
				return ok && imports[alias.Name] != "" && ast.IsExported(expr.Sel.Name)
			}
			return false
		}
		// elementKey is the named element type of a slice, array or map
		// type expression.
		elementKey := func(expr ast.Expr) string {
			switch typ := expr.(type) {
			case *ast.ArrayType:
				return typeKey(typ.Elt)
			case *ast.MapType:
				return typeKey(typ.Value)
			}
			return ""
		}
		// localField is the declared type of a field of one of the package's
		// own struct types.
		localField := func(key, field string) ast.Expr {
			name, ok := strings.CutPrefix(key, importPath+".")
			if !ok {
				return nil
			}
			spec, ok := pkg.types[name].node.(*ast.TypeSpec)
			if !ok {
				return nil
			}
			structType, ok := spec.Type.(*ast.StructType)
			if !ok {
				return nil
			}
			for _, declared := range structType.Fields.List {
				for _, declaredName := range declared.Names {
					if declaredName.Name == field {
						return declared.Type
					}
				}
			}
			return nil
		}
		// results is the declared result list of the package's function or
		// method of that name (methods by name alone: receivers are not
		// resolved), or nil when there is none or the declarations disagree.
		results := func(call *ast.CallExpr) []*ast.Field {
			name := ""
			switch fun := call.Fun.(type) {
			case *ast.Ident:
				name = fun.Name
				if decl, ok := pkg.funcs[name].node.(*ast.FuncDecl); ok && decl.Type.Results != nil {
					return decl.Type.Results.List
				}
				return nil
			case *ast.SelectorExpr:
				if alias, ok := fun.X.(*ast.Ident); ok && imports[alias.Name] != "" {
					return nil // another package's function
				}
				name = fun.Sel.Name
			}
			var found []*ast.Field
			for _, method := range pkg.methods[name] {
				decl, ok := method.node.(*ast.FuncDecl)
				if !ok || decl.Type.Results == nil {
					return nil
				}
				if found != nil && resultSignature(found) != resultSignature(decl.Type.Results.List) {
					return nil
				}
				found = decl.Type.Results.List
			}
			return found
		}
		for _, decl := range source.file.Decls {
			variables, elements := map[string]string{}, map[string]string{}
			declare := func(name string, typ ast.Expr) {
				if key := typeKey(typ); key != "" {
					variables[name] = key
				}
				if key := elementKey(typ); key != "" {
					elements[name] = key
				}
			}
			if fn, ok := decl.(*ast.FuncDecl); ok {
				for _, list := range []*ast.FieldList{fn.Recv, fn.Type.Params} {
					if list == nil {
						continue
					}
					for _, field := range list.List {
						for _, name := range field.Names {
							declare(name.Name, field.Type)
						}
					}
				}
			}
			// localLiterals are the string literals each local variable of
			// this declaration is assigned, so `status := "running"; ref.Status
			// = status` is held to the same rule as the literal itself.
			localLiterals := map[string][]*ast.BasicLit{}
			ast.Inspect(decl, func(node ast.Node) bool {
				switch node := node.(type) {
				case *ast.AssignStmt:
					if len(node.Lhs) == len(node.Rhs) {
						for i, lhs := range node.Lhs {
							if ident, ok := lhs.(*ast.Ident); ok {
								if literal := nonEmptyStringLiteral(node.Rhs[i]); literal != nil {
									localLiterals[ident.Name] = append(localLiterals[ident.Name], literal)
								}
							}
						}
					}
				case *ast.ValueSpec:
					if len(node.Names) == len(node.Values) {
						for i, name := range node.Names {
							if literal := nonEmptyStringLiteral(node.Values[i]); literal != nil {
								localLiterals[name.Name] = append(localLiterals[name.Name], literal)
							}
						}
					}
				}
				return true
			})
			add := func(key, field string, write, selector bool, value ast.Expr) {
				use := fieldUse{key: key, field: field, write: write, selector: selector, named: named(value), literal: nonEmptyStringLiteral(value)}
				if use.named || use.literal != nil {
					uses = append(uses, use)
					return
				}
				if ident, ok := value.(*ast.Ident); ok {
					for _, literal := range localLiterals[ident.Name] {
						use.literal = literal
						uses = append(uses, use)
					}
				}
			}
			// foreign marks a value of another package's type. Every field
			// reachable from it is declared outside this package too: that
			// package cannot import this one.
			foreign := func(key string) bool {
				return key != "" && !strings.HasPrefix(key, importPath+".")
			}
			// exprType resolves a variable or a field chain from one: the
			// type key, another package's type, or "" when unknown.
			var exprType, containerElement func(ast.Expr) string
			exprType = func(expr ast.Expr) string {
				switch expr := expr.(type) {
				case *ast.Ident:
					return variables[expr.Name]
				case *ast.ParenExpr:
					return exprType(expr.X)
				case *ast.StarExpr:
					return exprType(expr.X)
				case *ast.IndexExpr:
					return containerElement(expr.X)
				case *ast.CallExpr:
					if list := results(expr); len(list) > 0 {
						return typeKey(list[0].Type)
					}
				case *ast.SelectorExpr:
					base := exprType(expr.X)
					if base == "" || foreign(base) {
						return base
					}
					declared := localField(base, expr.Sel.Name)
					if key := typeKey(declared); key != "" {
						return key
					}
					if key := elementKey(declared); foreign(key) {
						return key
					}
				}
				return ""
			}
			// containerElement resolves the element type a range over expr
			// yields.
			containerElement = func(expr ast.Expr) string {
				switch expr := expr.(type) {
				case *ast.Ident:
					return elements[expr.Name]
				case *ast.CallExpr:
					if list := results(expr); len(list) > 0 {
						return elementKey(list[0].Type)
					}
				case *ast.SelectorExpr:
					base := exprType(expr.X)
					if foreign(base) {
						return base
					}
					if base != "" {
						return elementKey(localField(base, expr.Sel.Name))
					}
				}
				return ""
			}
			selected := func(expr ast.Expr) (string, string) {
				sel, ok := expr.(*ast.SelectorExpr)
				if !ok {
					return "", ""
				}
				if key := exprType(sel.X); key != "" {
					return key, sel.Sel.Name
				}
				return importPath + ".*", sel.Sel.Name
			}
			ast.Inspect(decl, func(node ast.Node) bool {
				switch node := node.(type) {
				case *ast.ValueSpec:
					if node.Type != nil {
						for _, name := range node.Names {
							declare(name.Name, node.Type)
						}
					}
				case *ast.RangeStmt:
					value, ok := node.Value.(*ast.Ident)
					if !ok {
						return true
					}
					if key := containerElement(node.X); key != "" {
						variables[value.Name] = key
					}
				case *ast.CompositeLit:
					if key := typeKey(node.Type); key != "" {
						for _, element := range node.Elts {
							if pair, ok := element.(*ast.KeyValueExpr); ok {
								if field, ok := pair.Key.(*ast.Ident); ok {
									add(key, field.Name, true, false, pair.Value)
								}
							}
						}
					}
				case *ast.AssignStmt:
					if call, ok := singleCall(node); ok && node.Tok == token.DEFINE {
						position := 0
						for _, field := range results(call) {
							count := max(len(field.Names), 1)
							for range count {
								if position < len(node.Lhs) {
									if ident, ok := node.Lhs[position].(*ast.Ident); ok {
										declare(ident.Name, field.Type)
									}
								}
								position++
							}
						}
					}
					for i, lhs := range node.Lhs {
						if i >= len(node.Rhs) {
							break
						}
						if ident, ok := lhs.(*ast.Ident); ok && node.Tok == token.DEFINE {
							value := node.Rhs[i]
							if unary, ok := value.(*ast.UnaryExpr); ok && unary.Op == token.AND {
								value = unary.X
							}
							if composite, ok := value.(*ast.CompositeLit); ok {
								if key := typeKey(composite.Type); key != "" {
									variables[ident.Name] = key
								}
							} else if key := exprType(value); key != "" {
								variables[ident.Name] = key
							}
						}
						if key, field := selected(lhs); key != "" {
							add(key, field, true, true, node.Rhs[i])
						}
					}
				case *ast.BinaryExpr:
					if node.Op == token.EQL || node.Op == token.NEQ {
						if key, field := selected(node.X); key != "" {
							add(key, field, false, true, node.Y)
						}
						if key, field := selected(node.Y); key != "" {
							add(key, field, false, true, node.X)
						}
					}
				case *ast.SwitchStmt:
					if key, field := selected(node.Tag); key != "" {
						for _, clause := range node.Body.List {
							for _, value := range clause.(*ast.CaseClause).List {
								add(key, field, false, true, value)
							}
						}
					}
				}
				return true
			})
		}
	}
	return uses
}

// singleCall reports an assignment of several values from one call.
func singleCall(assign *ast.AssignStmt) (*ast.CallExpr, bool) {
	if len(assign.Rhs) != 1 || len(assign.Lhs) < 2 {
		return nil, false
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	return call, ok
}

// resultSignature renders a result list's type expressions for comparison.
func resultSignature(fields []*ast.Field) string {
	var parts []string
	for _, field := range fields {
		parts = append(parts, strconv.Itoa(max(len(field.Names), 1))+":"+typeExprString(field.Type))
	}
	return strings.Join(parts, ",")
}

// typeExprString spells the type expressions result lists use; anything else
// spells as its position, so two declarations never compare equal through it.
func typeExprString(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.StarExpr:
		return "*" + typeExprString(expr.X)
	case *ast.SelectorExpr:
		return typeExprString(expr.X) + "." + expr.Sel.Name
	case *ast.ArrayType:
		if expr.Len == nil {
			return "[]" + typeExprString(expr.Elt)
		}
	case *ast.MapType:
		return "map[" + typeExprString(expr.Key) + "]" + typeExprString(expr.Value)
	}
	return "@" + strconv.Itoa(int(expr.Pos()))
}

// typedFields records the fields a package declares with one of its own
// string types, as "importpath.Type.Field" and "importpath.*.Field".
func typedFields(pkg codecPackage, importPath string, typed map[string]bool) {
	for name, decl := range pkg.types {
		spec, ok := decl.node.(*ast.TypeSpec)
		if !ok {
			continue
		}
		structType, ok := spec.Type.(*ast.StructType)
		if !ok {
			continue
		}
		for _, field := range structType.Fields.List {
			ident, ok := field.Type.(*ast.Ident)
			if !ok {
				continue
			}
			underlying, ok := pkg.types[ident.Name].node.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if base, ok := underlying.Type.(*ast.Ident); ok && base.Name == "string" {
				for _, fieldName := range field.Names {
					typed[importPath+"."+name+"."+fieldName.Name] = true
					typed[importPath+".*."+fieldName.Name] = true
				}
			}
		}
	}
}

// vocabularyEvidence decides which fields are a vocabulary: fields declared
// with one of the package's own string types, and fields the code reads by
// value (compares or switches against a named constant or string literal).
// The compared values are the words a reader accepts, wherever the stored
// value came from (this code, operator input or manifest data), so they and
// every value written to the field must be named constants. A reason or
// message field that is written and displayed but never branched on is not
// one. A use on a resolved type takes that type's evidence; an unresolved use
// takes every read of the field name in its package.
type vocabularyEvidence struct {
	read, typed map[string]bool
}

func newVocabularyEvidence() vocabularyEvidence {
	return vocabularyEvidence{read: map[string]bool{}, typed: map[string]bool{}}
}

func (v vocabularyEvidence) add(importPath string, uses []fieldUse) {
	for _, use := range uses {
		if !use.write {
			v.read[use.key+"."+use.field] = true
			v.read[importPath+"|"+use.field] = true
		}
	}
}

func (v vocabularyEvidence) holds(importPath string, use fieldUse) bool {
	if v.typed[use.key+"."+use.field] {
		return true
	}
	if strings.HasSuffix(use.key, ".*") {
		return v.read[importPath+"|"+use.field]
	}
	return v.read[use.key+"."+use.field]
}

// literalCandidates renders the vocabulary uses whose value is a string
// literal as "importpath.Type.Field@file:line". The builder cannot see which
// types are stored; the executable keeps the candidates whose field is a
// string field of a declared stored type and refuses to qualify while any
// remain (statecontract.LiteralViolations). Only envelopes an owner declares
// are its vocabulary: in an owner package (owned nil) that is the package's
// own types, with every form; a caller elsewhere is held to composite
// literals of the owned packages' types. A type another package declares (a
// manifest, a fetch step) carries that package's vocabulary.
func literalCandidates(pkg codecPackage, importPath string, uses []fieldUse, evidence vocabularyEvidence, owned map[string]bool) []string {
	var found []string
	for _, use := range uses {
		if use.literal == nil || !evidence.holds(importPath, use) {
			continue
		}
		typePackage := use.key[:strings.LastIndex(use.key, ".")]
		if owned == nil && typePackage != importPath || owned != nil && (use.selector || !owned[typePackage]) {
			continue
		}
		found = append(found, use.key+"."+use.field+"@"+pkg.location(use.literal))
	}
	return found
}

// StoredFieldLiteralCandidates returns the literal candidates of the owner
// packages (every form) and of callers in callerDirs, scanned recursively
// (composite literals of an owner package's types). Tests combine them with
// the compiled stored types (statecontract.LiteralViolations) so a literal
// stored vocabulary value fails during development, including in packages the
// build stamp does not scan.
func StoredFieldLiteralCandidates(root string, ownerDirs []string, callerDirs ...string) ([]string, error) {
	type scanned struct {
		pkg        codecPackage
		importPath string
		uses       []fieldUse
		owner      bool
	}
	var packages []scanned
	owned := map[string]bool{}
	evidence := newVocabularyEvidence()
	scan := func(path string, fset *token.FileSet, sources []goSource, owner bool) error {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		importPath, err := sourceImportPath(root, filepath.ToSlash(rel))
		if err != nil {
			return nil // outside any module
		}
		if !owner && owned[importPath] {
			return nil // an owner package, scanned already
		}
		pkg := indexPackage(fset, root, sources)
		uses := fieldUses(pkg, importPath, sources)
		if owner {
			owned[importPath] = true
			typedFields(pkg, importPath, evidence.typed)
		}
		evidence.add(importPath, uses)
		packages = append(packages, scanned{pkg, importPath, uses, owner})
		return nil
	}
	for _, dir := range ownerDirs {
		fset := token.NewFileSet()
		path := filepath.Join(root, filepath.FromSlash(dir))
		sources, err := loadGoSources(fset, path, "")
		if err != nil {
			return nil, err
		}
		if err := scan(path, fset, sources, true); err != nil {
			return nil, err
		}
	}
	err := walkGoPackages(root, callerDirs, func(path string, fset *token.FileSet, sources []goSource) error {
		return scan(path, fset, sources, false)
	})
	var found []string
	for _, p := range packages {
		if p.owner {
			found = append(found, literalCandidates(p.pkg, p.importPath, p.uses, evidence, nil)...)
		} else {
			found = append(found, literalCandidates(p.pkg, p.importPath, p.uses, evidence, owned)...)
		}
	}
	return compactStrings(sortedCopy(found)), err
}

// walkGoPackages visits every directory with Go sources under dirs.
func walkGoPackages(root string, dirs []string, visit func(path string, fset *token.FileSet, sources []goSource) error) error {
	for _, dir := range dirs {
		start := filepath.Join(root, filepath.FromSlash(dir))
		err := filepath.WalkDir(start, func(path string, entry os.DirEntry, err error) error {
			if err != nil || !entry.IsDir() {
				return err
			}
			if name := entry.Name(); name == "testdata" || name == "node_modules" || strings.HasPrefix(name, ".") && path != start {
				return filepath.SkipDir
			}
			fset := token.NewFileSet()
			sources, err := loadGoSources(fset, path, "")
			if err != nil {
				return nil // directories without Go sources
			}
			return visit(path, fset, sources)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (a ownerAnalysis) layoutDigest() string {
	sum := sha256.Sum256([]byte(strings.Join(a.layouts, "\n\n")))
	return hex.EncodeToString(sum[:16])
}

// checkVersion accepts a named package constant (recorded so the executable
// can require it to be a fact), a zero or empty presence check, or another
// version field (copying or comparing stored values). Anything else hides the
// stored format version from the contract.
func (a *ownerAnalysis) checkVersion(pkg codecPackage, value ast.Expr, versionFields map[string]bool) {
	switch value := value.(type) {
	case *ast.BasicLit:
		if value.Value == "0" || value.Value == `""` {
			return
		}
	case *ast.Ident:
		if decl, ok := pkg.values[value.Name]; ok && strings.HasPrefix(decl.text, "const ") {
			a.versions = append(a.versions, value.Name)
			return
		}
	case *ast.SelectorExpr:
		if versionFields[value.Sel.Name] {
			return
		}
	}
	a.violations = append(a.violations, "version-value@"+pkg.location(value))
}

// versionFieldNames are the Go names of struct fields that hold a stored
// format version: json key schema_version, format_version or
// *_schema_version, or an integer field keyed *version. String fields keyed
// version or *_version usually carry resource or artifact versions (data).
func versionFieldNames(sources []goSource) (map[string]bool, map[string]map[int]bool) {
	names := map[string]bool{}
	positions := map[string]map[int]bool{}
	for _, source := range sources {
		ast.Inspect(source.file, func(node ast.Node) bool {
			typeName := ""
			structType, ok := node.(*ast.StructType)
			if spec, isSpec := node.(*ast.TypeSpec); isSpec {
				structType, ok = spec.Type.(*ast.StructType)
				typeName = spec.Name.Name
			}
			if !ok {
				return true
			}
			index := -1
			for _, field := range structType.Fields.List {
				count := len(field.Names)
				if count == 0 {
					count = 1
				}
				index += count
				if field.Tag == nil || len(field.Names) == 0 {
					continue
				}
				tag, err := strconv.Unquote(field.Tag.Value)
				if err != nil {
					continue
				}
				key, _, _ := strings.Cut(reflect.StructTag(tag).Get("json"), ",")
				integer := false
				if ident, ok := field.Type.(*ast.Ident); ok {
					integer = strings.HasPrefix(ident.Name, "int") || strings.HasPrefix(ident.Name, "uint")
				}
				if key == "schema_version" || key == "format_version" || strings.HasSuffix(key, "_schema_version") || strings.HasSuffix(key, "version") && integer {
					for i, name := range field.Names {
						names[name.Name] = true
						if typeName != "" {
							if positions[typeName] == nil {
								positions[typeName] = map[int]bool{}
							}
							positions[typeName][index-len(field.Names)+1+i] = true
						}
					}
				}
			}
			return typeName == ""
		})
	}
	return names, positions
}

func selectorName(expr ast.Expr) string {
	if selector, ok := expr.(*ast.SelectorExpr); ok {
		return selector.Sel.Name
	}
	return ""
}

// stringExpr reports an untyped string constant expression: a string literal
// or a concatenation that contains one.
func stringExpr(expr ast.Expr) bool {
	switch expr := expr.(type) {
	case *ast.BasicLit:
		return expr.Kind == token.STRING
	case *ast.BinaryExpr:
		return expr.Op == token.ADD && (stringExpr(expr.X) || stringExpr(expr.Y))
	case *ast.ParenExpr:
		return stringExpr(expr.X)
	}
	return false
}

// timeLayoutArgument returns the layout argument of x.Format(L),
// x.AppendFormat(b, L), time.Parse(L, v) or time.ParseInLocation(L, v, loc).
func timeLayoutArgument(call *ast.CallExpr) ast.Expr {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	switch selector.Sel.Name {
	case "Format":
		if len(call.Args) == 1 {
			return call.Args[0]
		}
	case "AppendFormat":
		if len(call.Args) == 2 {
			return call.Args[1]
		}
	case "Parse", "ParseInLocation":
		if ident, ok := selector.X.(*ast.Ident); ok && ident.Name == "time" && len(call.Args) >= 2 {
			return call.Args[0]
		}
	}
	return nil
}

// storedLiteralCall reports a string literal passed where
// statecontract.StoredValueCalls declares a stored vocabulary argument.
func storedLiteralCall(pkg codecPackage, call *ast.CallExpr) string {
	name := ""
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		name = fun.Name
	case *ast.SelectorExpr:
		name = fun.Sel.Name
	}
	index, ok := statecontract.StoredValueCalls[name]
	if !ok || index >= len(call.Args) {
		return ""
	}
	if literal, ok := call.Args[index].(*ast.BasicLit); ok && literal.Kind == token.STRING {
		return pkg.location(literal)
	}
	return ""
}

// StoredValueLiteralCalls finds stored vocabulary values written as string
// literals by callers in the given repository directories (recursively), so
// an owner's tests can hold callers outside its package to the same rule.
func StoredValueLiteralCalls(root string, dirs ...string) ([]string, error) {
	var found []string
	err := walkGoPackages(root, dirs, func(_ string, fset *token.FileSet, sources []goSource) error {
		pkg := indexPackage(fset, root, sources)
		for _, source := range sources {
			ast.Inspect(source.file, func(node ast.Node) bool {
				if call, ok := node.(*ast.CallExpr); ok {
					if location := storedLiteralCall(pkg, call); location != "" {
						found = append(found, location)
					}
				}
				return true
			})
		}
		return nil
	})
	sort.Strings(found)
	return found, err
}

// PersistentStateSourceFindings runs, for one persistence owner package, the
// checks the governed build applies when it stamps, plus classification
// completeness: every untyped string constant of a precise owner must be
// listed as stored or not stored, and no classification may be stale. Owner
// packages call it from their own tests so a violation surfaces during
// development rather than as a refused activation.
func PersistentStateSourceFindings(root, dir string) ([]string, error) {
	vocabulary, owned := false, false
	for _, dirs := range statecontract.PreciseSources {
		vocabulary = vocabulary || slices.Contains(dirs, dir)
	}
	for _, dirs := range statecontract.ConstantSources {
		owned = owned || slices.Contains(dirs, dir)
	}
	if !vocabulary && !owned {
		return []string{dir + " is not in statecontract.PreciseSources or ConstantSources"}, nil
	}
	var findings []string
	analysis, err := analyzeOwnerPackages(root, []string{dir}, statecontract.NotStoredConstants, vocabulary)
	if err != nil {
		return nil, err
	}
	violations := append([]string(nil), analysis.violations...)
	if slices.Contains(statecontract.CodecPackages, dir) {
		_, codecViolations, err := scanCodecPackage(root, dir)
		if err != nil {
			return nil, err
		}
		violations = append(violations, codecViolations...)
	}
	for _, violation := range violations {
		findings = append(findings, statecontract.DescribeViolations([]string{violation}))
	}
	declared, err := packageConstantNames(root, dir)
	if err != nil {
		return nil, err
	}
	stored := map[string]bool{}
	for _, name := range statecontract.StoredConstants[dir] {
		stored[name] = true
		if !declared[name] {
			findings = append(findings, "stale classification: "+dir+" lists "+name+" as stored, but no such constant exists")
		}
	}
	for _, name := range statecontract.NotStoredConstants[dir] {
		if !declared[name] {
			findings = append(findings, "stale classification: "+dir+" lists "+name+" as not stored, but no such constant exists")
		}
	}
	if vocabulary {
		for name := range analysis.values {
			if !stored[name] {
				findings = append(findings, "unclassified constant: "+dir+"."+name+" is an untyped string constant; list it in statecontract.StoredConstants (its value reaches stored state) or statecontract.NotStoredConstants")
			}
		}
	}
	sort.Strings(findings)
	return findings, nil
}

func packageConstantNames(root, dir string) (map[string]bool, error) {
	fset := token.NewFileSet()
	sources, err := loadGoSources(fset, filepath.Join(root, filepath.FromSlash(dir)), "")
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, source := range sources {
		for _, decl := range source.file.Decls {
			if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.CONST {
				for _, spec := range gen.Specs {
					for _, name := range spec.(*ast.ValueSpec).Names {
						names[name.Name] = true
					}
				}
			}
		}
	}
	return names, nil
}
