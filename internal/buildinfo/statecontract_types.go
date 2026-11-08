package buildinfo

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	maxCodecPackageFiles = 2048
	maxCodecSourceBytes  = 1 << 20
)

var (
	codecEncoderMethods = map[string]bool{"MarshalJSON": true, "MarshalText": true}
	codecMethods        = map[string]bool{"MarshalJSON": true, "UnmarshalJSON": true, "MarshalText": true, "UnmarshalText": true}
)

// goSource is one parsed non-test Go file.
type goSource struct {
	path string
	src  []byte
	file *ast.File
}

// loadGoSources parses the non-test Go files of dir whose names start with
// prefix. Files of every build configuration are included so a stamp is
// target-independent; files constrained to `ignore` are never compiled and
// are skipped.
func loadGoSources(fset *token.FileSet, dir, prefix string) ([]goSource, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	if len(entries) > maxCodecPackageFiles {
		return nil, fmt.Errorf("package exceeds its inventory bound")
	}
	var sources []goSource
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || !strings.HasPrefix(name, prefix) {
			continue
		}
		path := filepath.Join(dir, name)
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() || info.Size() > maxCodecSourceBytes {
			return nil, fmt.Errorf("source %s is not a bounded regular file", name)
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if ignoredBuildFile(src) {
			continue
		}
		file, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			return nil, err
		}
		sources = append(sources, goSource{path: path, src: src, file: file})
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("no Go sources match %s", filepath.Join(dir, prefix+"*"))
	}
	return sources, nil
}

// packageDecl is one package-level declaration and its canonical source.
type packageDecl struct {
	text string
	node ast.Node
}

// codecPackage indexes a package's package-level declarations by name.
type codecPackage struct {
	fset    *token.FileSet
	root    string
	types   map[string]packageDecl
	values  map[string]packageDecl
	funcs   map[string]packageDecl
	methods map[string][]packageDecl // method name -> declarations on any type
	topSpec map[ast.Node]bool
	// imports maps a file to its import names and paths.
	imports map[string]map[string]string
	// violations collects codec closures that call into another module.
	violations *[]string
}

func indexPackage(fset *token.FileSet, root string, sources []goSource) codecPackage {
	violations := []string{}
	pkg := codecPackage{fset: fset, root: root, types: map[string]packageDecl{}, values: map[string]packageDecl{}, funcs: map[string]packageDecl{}, methods: map[string][]packageDecl{}, topSpec: map[ast.Node]bool{}, imports: map[string]map[string]string{}, violations: &violations}
	for _, source := range sources {
		names := map[string]string{}
		for _, spec := range source.file.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			name := filepath.Base(path)
			if spec.Name != nil {
				name = spec.Name.Name
			}
			names[name] = path
		}
		pkg.imports[source.path] = names
		for _, decl := range source.file.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					pkg.topSpec[spec] = true
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						pkg.types[spec.Name.Name] = packageDecl{"type " + canonicalSource(fset, source.src, spec), spec}
					case *ast.ValueSpec:
						text := decl.Tok.String() + " " + canonicalSource(fset, source.src, spec)
						for _, name := range spec.Names {
							pkg.values[name.Name] = packageDecl{text, spec}
						}
					}
				}
			case *ast.FuncDecl:
				entry := packageDecl{"func " + canonicalSource(fset, source.src, decl), decl}
				if decl.Recv == nil {
					pkg.funcs[decl.Name.Name] = entry
					pkg.topSpec[decl] = true
				} else {
					pkg.methods[decl.Name.Name] = append(pkg.methods[decl.Name.Name], entry)
				}
			}
		}
	}
	return pkg
}

// location renders a node position relative to the source root.
func (pkg codecPackage) location(node ast.Node) string {
	position := pkg.fset.Position(node.Pos())
	path := position.Filename
	if rel, err := filepath.Rel(pkg.root, path); err == nil && pkg.root != "" {
		path = filepath.ToSlash(rel)
	}
	return fmt.Sprintf("%s:%d", path, position.Line)
}

// scanCodecPackage returns, per named type declared in dir, a digest of the
// source that decides its stored values beyond the compiled shape:
//
//   - every constant of that type (iota position and value expression) and
//     the same-package declarations those expressions reach;
//   - for a type with a custom JSON/text codec: its type declaration, the full
//     source of its encoders, and the transitive same-package closure
//     (consts, vars, types, functions and named methods) of every codec
//     method, decoders included. Decoder bodies themselves do not
//     participate: the prior release decodes with its own decoder.
//
// Comments and formatting do not participate. Types with none of these have
// no entry. Values reached through another package are not followed.
func scanCodecPackage(root, dir string) (map[string]string, []string, error) {
	fset := token.NewFileSet()
	sources, err := loadGoSources(fset, filepath.Join(root, filepath.FromSlash(dir)), "")
	if err != nil {
		return nil, nil, err
	}
	pkg := indexPackage(fset, root, sources)
	parts := map[string][]string{}
	for _, source := range sources {
		for _, decl := range source.file.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				if decl.Tok == token.CONST {
					pkg.collectTypedConstants(source.src, decl, parts)
				}
			case *ast.FuncDecl:
				typeName := receiverTypeName(decl)
				if _, declared := pkg.types[typeName]; !declared || !codecMethods[decl.Name.Name] {
					continue
				}
				parts[typeName] = append(parts[typeName], pkg.types[typeName].text)
				if codecEncoderMethods[decl.Name.Name] {
					parts[typeName] = append(parts[typeName], "encoder "+canonicalSource(fset, source.src, decl))
				}
				parts[typeName] = append(parts[typeName], pkg.closure(decl.Body, pkg.types[typeName].node)...)
			}
		}
	}
	digests := make(map[string]string, len(parts))
	for typeName, items := range parts {
		sort.Strings(items)
		items = compactStrings(items)
		sum := sha256.Sum256([]byte(strings.Join(items, "\n")))
		digests[typeName] = hex.EncodeToString(sum[:16])
	}
	return digests, *pkg.violations, nil
}

// collectTypedConstants assigns each constant spec to its named type,
// including implicit repetition (`A Phase = iota; B; C`) and conversions
// (`A = Phase("a")`), with the declarations its value expression reaches
// (`A Phase = prefix + "a"`). Untyped constants belong to no stored type.
func (pkg codecPackage) collectTypedConstants(src []byte, decl *ast.GenDecl, parts map[string][]string) {
	lastType, lastValues := "", ""
	var lastExprs []ast.Expr
	for index, spec := range decl.Specs {
		value, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		switch {
		case value.Type != nil:
			lastType = ""
			if ident, ok := value.Type.(*ast.Ident); ok {
				if _, declared := pkg.types[ident.Name]; declared {
					lastType = ident.Name
				}
			}
			lastValues, lastExprs = canonicalExprs(pkg.fset, src, value.Values), value.Values
		case len(value.Values) > 0:
			lastType = ""
			if call, ok := value.Values[0].(*ast.CallExpr); ok && len(value.Values) == 1 && len(call.Args) == 1 {
				if ident, ok := call.Fun.(*ast.Ident); ok {
					if _, declared := pkg.types[ident.Name]; declared {
						lastType = ident.Name
					}
				}
			}
			lastValues, lastExprs = canonicalExprs(pkg.fset, src, value.Values), value.Values
		}
		if lastType == "" {
			continue
		}
		parts[lastType] = append(parts[lastType], fmt.Sprintf("const %d %s", index, lastValues))
		for _, expr := range lastExprs {
			parts[lastType] = append(parts[lastType], pkg.closure(expr)...)
		}
	}
}

// closure returns the canonical source of every package-level declaration
// transitively reachable from the given nodes by name: consts, vars, types,
// functions, and methods named in selectors (x.String()) on any type.
// Identifiers the parser resolved to a local declaration are not followed.
func (pkg codecPackage) closure(nodes ...ast.Node) []string {
	seen := map[string]bool{}
	var out []string
	queue := append([]ast.Node(nil), nodes...)
	add := func(key string, decls ...packageDecl) {
		if seen[key] {
			return
		}
		seen[key] = true
		for _, decl := range decls {
			out = append(out, decl.text)
			queue = append(queue, decl.node)
		}
	}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		if node == nil {
			continue
		}
		ast.Inspect(node, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				// A codec that calls into another module package runs code
				// the stamp cannot see; fail closed rather than follow it.
				if selector, ok := n.Fun.(*ast.SelectorExpr); ok {
					if ident, ok := selector.X.(*ast.Ident); ok && ident.Obj == nil {
						path := pkg.imports[pkg.fset.Position(n.Pos()).Filename][ident.Name]
						if path != "" && !standardImport(path) {
							*pkg.violations = append(*pkg.violations, "codec-cross-package@"+pkg.location(n))
						}
					}
				}
			case *ast.SelectorExpr:
				if methods, ok := pkg.methods[n.Sel.Name]; ok {
					add("method "+n.Sel.Name, methods...)
				}
				queue = append(queue, n.X)
				return false
			case *ast.Ident:
				if n.Obj != nil && !pkg.topSpec[declNode(n.Obj)] {
					return true
				}
				if decl, ok := pkg.values[n.Name]; ok {
					add("value "+n.Name, decl)
				} else if decl, ok := pkg.types[n.Name]; ok {
					add("type "+n.Name, decl)
				} else if decl, ok := pkg.funcs[n.Name]; ok {
					add("func "+n.Name, decl)
				}
			}
			return true
		})
	}
	return out
}

// declNode maps a parser object to the top-level node that declares it.
func declNode(obj *ast.Object) ast.Node {
	switch decl := obj.Decl.(type) {
	case *ast.ValueSpec:
		return decl
	case *ast.TypeSpec:
		return decl
	case *ast.FuncDecl:
		return decl
	default:
		return nil
	}
}

func receiverTypeName(decl *ast.FuncDecl) string {
	if decl.Recv == nil || len(decl.Recv.List) != 1 {
		return ""
	}
	expr := decl.Recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	// Generic receivers (T[K], T[K, V]) belong to their declared type.
	switch generic := expr.(type) {
	case *ast.IndexExpr:
		expr = generic.X
	case *ast.IndexListExpr:
		expr = generic.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

// wholeSourceDigest hashes the complete canonical token stream of every
// non-test Go file selected by the patterns ("dir" or "dir/prefix*").
func wholeSourceDigest(root string, patterns []string) (string, error) {
	hash := sha256.New()
	for _, pattern := range patterns {
		dir, prefix := pattern, ""
		if strings.HasSuffix(pattern, "*") {
			dir, prefix = filepath.Dir(pattern), strings.TrimSuffix(filepath.Base(pattern), "*")
		}
		fset := token.NewFileSet()
		sources, err := loadGoSources(fset, filepath.Join(root, filepath.FromSlash(dir)), prefix)
		if err != nil {
			return "", err
		}
		for _, source := range sources {
			rel, err := filepath.Rel(root, source.path)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(hash, "file %s\n%s\n", filepath.ToSlash(rel), canonicalSource(fset, source.src, source.file))
		}
	}
	return hex.EncodeToString(hash.Sum(nil)[:16]), nil
}

// packageConstantsDigest hashes every package-level constant spec (names,
// types and values, iota position included) of the given package dirs,
// except tunables classified as not stored.
func packageConstantsDigest(root string, dirs []string, notStored map[string][]string) (string, error) {
	var items []string
	for _, dir := range dirs {
		skip := map[string]bool{}
		for _, name := range notStored[dir] {
			skip[name] = true
		}
		fset := token.NewFileSet()
		sources, err := loadGoSources(fset, filepath.Join(root, filepath.FromSlash(dir)), "")
		if err != nil {
			return "", err
		}
		for _, source := range sources {
			for _, decl := range source.file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.CONST {
					continue
				}
				// Implicit repetition depends on the block, so the kept specs
				// of a block form one item with their positions.
				var specs []string
				for index, spec := range gen.Specs {
					value := spec.(*ast.ValueSpec)
					kept := false
					for _, name := range value.Names {
						kept = kept || !skip[name.Name]
					}
					if kept {
						specs = append(specs, strconv.Itoa(index)+":"+canonicalSource(fset, source.src, value))
					}
				}
				if len(specs) > 0 {
					items = append(items, dir+" "+strings.Join(specs, " ; "))
				}
			}
		}
	}
	sort.Strings(items)
	sum := sha256.Sum256([]byte(strings.Join(items, "\n")))
	return hex.EncodeToString(sum[:16]), nil
}

func canonicalExprs(fset *token.FileSet, src []byte, exprs []ast.Expr) string {
	parts := make([]string, 0, len(exprs))
	for _, expr := range exprs {
		parts = append(parts, canonicalSource(fset, src, expr))
	}
	return strings.Join(parts, " , ")
}

// canonicalSource renders a node as its token stream: comments, layout,
// automatic semicolons and trailing commas do not participate.
func canonicalSource(fset *token.FileSet, src []byte, node ast.Node) string {
	pos, end := node.Pos(), node.End()
	if file, ok := node.(*ast.File); ok {
		pos, end = file.FileStart, file.FileEnd
	}
	start, stop := fset.Position(pos).Offset, fset.Position(end).Offset
	if start < 0 || stop > len(src) || start > stop {
		return ""
	}
	fragment := src[start:stop]
	file := token.NewFileSet().AddFile("", -1, len(fragment))
	var s scanner.Scanner
	s.Init(file, fragment, nil, 0)
	var tokens []string
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.SEMICOLON && lit == "\n" {
			continue
		}
		text := tok.String()
		if lit != "" {
			text = lit
		}
		if (tok == token.RPAREN || tok == token.RBRACK || tok == token.RBRACE) && len(tokens) > 0 && tokens[len(tokens)-1] == "," {
			tokens = tokens[:len(tokens)-1]
		}
		tokens = append(tokens, text)
	}
	return strings.Join(tokens, " ")
}

// ignoredBuildFile reports a `//go:build ignore` header before the package
// clause; such files are never compiled.
func ignoredBuildFile(src []byte) bool {
	for _, line := range bytes.Split(src, []byte("\n")) {
		trimmed := strings.TrimSpace(string(line))
		if strings.HasPrefix(trimmed, "package ") {
			return false
		}
		if strings.HasPrefix(trimmed, "//go:build") && strings.Contains(trimmed, "ignore") {
			return true
		}
	}
	return false
}

func compactStrings(items []string) []string {
	out := items[:0]
	for i, item := range items {
		if i == 0 || items[i-1] != item {
			out = append(out, item)
		}
	}
	return out
}

func standardImport(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return !strings.Contains(first, ".")
}
