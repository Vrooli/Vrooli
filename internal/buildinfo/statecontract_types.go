package buildinfo

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var errNoGoSources = errors.New("no Go sources")

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
		_, ignored, err := buildConstraints(src)
		if err != nil {
			return nil, fmt.Errorf("source %s: %w", path, err)
		}
		if ignored {
			continue
		}
		file, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			return nil, err
		}
		sources = append(sources, goSource{path: path, src: src, file: file})
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("%w match %s", errNoGoSources, filepath.Join(dir, prefix+"*"))
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
	fset      *token.FileSet
	root      string
	types     map[string]packageDecl
	allTypes  map[string][]packageDecl
	values    map[string]packageDecl
	allValues map[string][]packageDecl
	funcs     map[string]packageDecl
	allFuncs  map[string][]packageDecl
	methods   map[string][]packageDecl // method name -> declarations on any type
	topSpec   map[ast.Node]bool
	// imports maps a file to its import names and paths.
	imports    map[string]map[string]string
	dotImports map[string][]string
	// violations collects codec closures that call into another module.
	violations             *[]string
	allowExternal          func(string, string) bool
	strictForeignSelectors bool
	foreignActive          map[string]bool
}

func indexPackage(fset *token.FileSet, root string, sources []goSource) codecPackage {
	violations := []string{}
	pkg := codecPackage{fset: fset, root: root, types: map[string]packageDecl{}, allTypes: map[string][]packageDecl{}, values: map[string]packageDecl{}, allValues: map[string][]packageDecl{}, funcs: map[string]packageDecl{}, allFuncs: map[string][]packageDecl{}, methods: map[string][]packageDecl{}, topSpec: map[ast.Node]bool{}, imports: map[string]map[string]string{}, dotImports: map[string][]string{}, violations: &violations}
	for _, source := range sources {
		names := map[string]string{}
		for _, spec := range source.file.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			name := filepath.Base(path)
			if spec.Name != nil {
				name = spec.Name.Name
			}
			names[name] = path
			if name == "." {
				pkg.dotImports[source.path] = append(pkg.dotImports[source.path], path)
			}
		}
		pkg.imports[source.path] = names
		for _, decl := range source.file.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					pkg.topSpec[spec] = true
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						entry := packageDecl{"type " + canonicalSource(fset, source.src, spec), spec}
						pkg.types[spec.Name.Name] = entry
						pkg.allTypes[spec.Name.Name] = append(pkg.allTypes[spec.Name.Name], entry)
					case *ast.ValueSpec:
						text := decl.Tok.String() + " " + canonicalSource(fset, source.src, spec)
						for _, name := range spec.Names {
							entry := packageDecl{text, spec}
							pkg.values[name.Name] = entry
							pkg.allValues[name.Name] = append(pkg.allValues[name.Name], entry)
						}
					}
				}
			case *ast.FuncDecl:
				entry := packageDecl{"func " + canonicalSource(fset, source.src, decl), decl}
				if decl.Recv == nil {
					pkg.funcs[decl.Name.Name] = entry
					pkg.allFuncs[decl.Name.Name] = append(pkg.allFuncs[decl.Name.Name], entry)
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
	pkg.strictForeignSelectors = true
	pkg.foreignActive = map[string]bool{}
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
				for _, variant := range pkg.allTypes[typeName] {
					parts[typeName] = append(parts[typeName], variant.text)
					parts[typeName] = append(parts[typeName], pkg.closure(variant.node)...)
				}
				if codecEncoderMethods[decl.Name.Name] {
					parts[typeName] = append(parts[typeName], "encoder "+canonicalSource(fset, source.src, decl))
				}
				parts[typeName] = append(parts[typeName], pkg.closure(decl.Body)...)
			}
		}
	}
	digests := make(map[string]string, len(parts))
	// Record even purely promoted codecs. Reflection cannot distinguish an
	// override from promotion, so absence of this classification must fail closed.
	for name, variants := range pkg.allTypes {
		for _, decl := range variants {
			spec := decl.node.(*ast.TypeSpec)
			if structure, ok := spec.Type.(*ast.StructType); ok {
				for _, field := range structure.Fields.List {
					if len(field.Names) == 0 {
						if _, exists := parts[name]; !exists {
							parts[name] = []string{"promoted-codec-classification"}
						}
						break
					}
				}
			}
		}
	}
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
		// An unqualified dot-import reference cannot be attributed to its
		// declaration by this syntax-only closure. Do not qualify a codec
		// containing a non-standard dot import.
		if pkg.strictForeignSelectors {
			for _, path := range pkg.dotImports[pkg.fset.Position(node.Pos()).Filename] {
				if !standardImport(path) {
					*pkg.violations = append(*pkg.violations, "codec-cross-package@"+pkg.location(node))
				}
			}
		}
		ast.Inspect(node, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				if !pkg.strictForeignSelectors {
					if selector, ok := n.Fun.(*ast.SelectorExpr); ok {
						if ident, ok := selector.X.(*ast.Ident); ok && ident.Obj == nil {
							path := pkg.imports[pkg.fset.Position(n.Pos()).Filename][ident.Name]
							if path != "" && !standardImport(path) && (pkg.allowExternal == nil || !pkg.allowExternal(path, selector.Sel.Name)) {
								*pkg.violations = append(*pkg.violations, "codec-cross-package@"+pkg.location(n))
							}
						}
					}
				}
			case *ast.SelectorExpr:
				// Imported constants, variables and function values influence the
				// codec just as calls do. The foreign declaration is not in this
				// closure, so qualification must fail unless explicitly bound.
				if ident, ok := n.X.(*ast.Ident); pkg.strictForeignSelectors && ok && ident.Obj == nil {
					path := pkg.imports[pkg.fset.Position(n.Pos()).Filename][ident.Name]
					if path != "" && !standardImport(path) && (pkg.allowExternal == nil || !pkg.allowExternal(path, n.Sel.Name)) {
						if bound, ok := pkg.bindForeignType(path, n.Sel.Name); ok {
							out = append(out, bound...)
						} else {
							*pkg.violations = append(*pkg.violations, "codec-cross-package@"+pkg.location(n))
						}
					} else if path == "" {
						// A package can declare a name unlike the import path's base.
						// Unknown selector bases may be such imports: fail closed.
						*pkg.violations = append(*pkg.violations, "codec-cross-package@"+pkg.location(n))
					}
				}
				if methods, ok := pkg.methods[n.Sel.Name]; ok {
					add("method "+n.Sel.Name, methods...)
				}
				queue = append(queue, n.X)
				return false
			case *ast.Ident:
				if n.Obj != nil && !pkg.topSpec[declNode(n.Obj)] {
					return true
				}
				if decls := pkg.allValues[n.Name]; len(decls) > 0 {
					add("value "+n.Name, decls...)
				}
				if decls := pkg.allTypes[n.Name]; len(decls) > 0 {
					add("type "+n.Name, decls...)
				}
				if decls := pkg.allFuncs[n.Name]; len(decls) > 0 {
					add("func "+n.Name, decls...)
				}
			}
			return true
		})
	}
	return out
}

// bindForeignType includes a repository-local foreign type and its source
// closure, including custom codec methods. Unresolvable or ambiguous symbols
// fail closed. The package's codec methods are all included conservatively so
// nested local types cannot hide an encoder or decoder dependency.
func (pkg codecPackage) bindForeignType(path, name string) ([]string, bool) {
	module, err := goModulePath(filepath.Join(pkg.root, "go.mod"))
	if err != nil {
		return nil, false
	}
	rel, ok := strings.CutPrefix(path, module+"/")
	if !ok {
		return nil, false
	}
	dir := filepath.Join(pkg.root, filepath.FromSlash(rel))
	resolved, err := sourceImportPath(pkg.root, filepath.ToSlash(rel))
	if err != nil || resolved != path {
		return nil, false
	}
	key := path + "." + name
	if pkg.foreignActive == nil || pkg.foreignActive[key] || len(pkg.foreignActive) >= 64 {
		return nil, false
	}
	fset := token.NewFileSet()
	sources, err := loadGoSources(fset, dir, "")
	if err != nil {
		return nil, false
	}
	foreign := indexPackage(fset, pkg.root, sources)
	declarations := foreign.allTypes[name]
	if len(declarations) == 0 || len(foreign.allValues[name]) != 0 || len(foreign.allFuncs[name]) != 0 {
		return nil, false
	}
	foreign.strictForeignSelectors = true
	foreign.foreignActive = pkg.foreignActive
	pkg.foreignActive[key] = true
	defer delete(pkg.foreignActive, key)
	parts := []string{"foreign-type " + key}
	var nodes []ast.Node
	for _, decl := range declarations {
		parts = append(parts, decl.text)
		nodes = append(nodes, decl.node)
	}
	for method, decls := range foreign.methods {
		if codecMethods[method] {
			for _, decl := range decls {
				parts = append(parts, decl.text)
				nodes = append(nodes, decl.node)
			}
		}
	}
	parts = append(parts, foreign.closure(nodes...)...)
	if len(*foreign.violations) > 0 {
		return nil, false
	}
	sort.Strings(parts)
	return compactStrings(parts), true
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
// capacitySQLWriterDigest binds the bodies that supply persisted SQL values,
// including timestamp framing at each call site. Pure readers and unrelated
// sampling/decision functions do not participate.
func capacitySQLWriterDigest(root string) (string, error) {
	fset := token.NewFileSet()
	sources, err := loadGoSources(fset, filepath.Join(root, "internal/capacity"), "")
	if err != nil {
		return "", err
	}
	pkg := indexPackage(fset, root, sources)
	// Explicitly classify calls reached by the writer closure. Time encoding is
	// whole-source stamped; x/sys is bound to go.mod/go.sum below. Locking,
	// retry intervals and default durations do not define a persisted encoding.
	allowed := map[string]map[string]bool{
		"github.com/vrooli/vrooli/internal/storagetime":  {"FormatUTC": true, "FormatOptionalUTC": true},
		"github.com/vrooli/vrooli/internal/artifactlock": {"AcquireWithPathContext": true},
		// Runtime-home path identity is governed by the compiled home contract.
		"github.com/vrooli/vrooli/internal/config": {"HomeDir": true, "EnsureOwnedDir": true},
		"github.com/vrooli/repo-contract-go":       {"RuntimeHomeEntryPath": true},
		// Host sensing, command execution and admission thresholds provide live
		// observations; they do not encode the ledger's stored representation.
		"github.com/vrooli/vrooli/internal/shell":         {"NewCommandContext": true},
		"github.com/vrooli/vrooli/internal/hostinventory": {"Collect": true, "SystemCollector": true},
		"github.com/vrooli/vrooli/internal/hostpressure":  {"NewUnread": true, "NewRead": true, "ParsePSISomeAvg10": true},
		"github.com/vrooli/vrooli/internal/setpoint":      {"Resolve": true},
		"golang.org/x/sys/unix":                           {"Stat": true, "Statfs": true},
		"github.com/vrooli/vrooli/internal/tuning":        {"FastPersistenceRetryInterval": true, "FastHealthPollInterval": true, "CapacityHeartbeatTTL": true, "ReachabilityTimeout": true, "CredentialServiceTimeout": true, "CapacityObservedPeakHalflife": true, "DailyRetentionWindow": true},
	}
	usesUnix := false
	pkg.allowExternal = func(path, name string) bool {
		if path == "golang.org/x/sys/unix" {
			usesUnix = true
		}
		return allowed[path][name]
	}
	type writerFunc struct {
		source goSource
		fn     *ast.FuncDecl
	}
	var funcs []writerFunc
	selected := map[*ast.FuncDecl]bool{}
	selectedNames := map[string]bool{}
	for _, source := range sources {
		for _, decl := range source.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			funcs = append(funcs, writerFunc{source, fn})
			writes := false
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if ok && (selector.Sel.Name == "ExecContext" || selector.Sel.Name == "Exec") {
					writes = true
				}
				return true
			})
			if writes || fn.Name.Name == "formatTime" || fn.Name.Name == "formatOptionalTime" {
				selected[fn] = true
				// Callers of time formatters are not writers merely by reading time.
				if writes {
					selectedNames[fn.Name.Name] = true
				}
			}
		}
	}
	// Include callers that can transform arguments before passing them to a
	// writer helper. Name-based matching is conservative for methods.
	for changed := true; changed; {
		changed = false
		for _, entry := range funcs {
			if selected[entry.fn] {
				continue
			}
			callsWriter := false
			ast.Inspect(entry.fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				name := ""
				switch target := call.Fun.(type) {
				case *ast.Ident:
					name = target.Name
				case *ast.SelectorExpr:
					name = target.Sel.Name
				}
				if selectedNames[name] {
					callsWriter = true
				}
				return true
			})
			if callsWriter {
				selected[entry.fn] = true
				if !ast.IsExported(entry.fn.Name.Name) {
					selectedNames[entry.fn.Name.Name] = true
				}
				changed = true
			}
		}
	}
	var items []string
	for _, entry := range funcs {
		if selected[entry.fn] {
			items = append(items, filepath.Base(entry.source.path)+":"+entry.fn.Name.Name+":"+canonicalSource(fset, entry.source.src, entry.fn))
			items = append(items, pkg.closure(entry.fn.Body)...)
		}
	}
	if len(items) == 0 {
		return "", fmt.Errorf("capacity SQL writer inventory is empty")
	}
	if len(*pkg.violations) != 0 {
		return "", fmt.Errorf("capacity writer dependency not covered: %s", strings.Join(compactStrings(sortedCopy(*pkg.violations)), ", "))
	}
	// Filesystem identity uses x/sys/unix; bind its selected module and sums.
	for _, path := range []string{"go.mod", "go.sum"} {
		if !usesUnix {
			break
		}
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			return "", err
		}
		found := false
		for _, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, "golang.org/x/sys ") {
				items = append(items, path+":"+strings.TrimSpace(line))
				found = true
			}
		}
		if !found {
			return "", fmt.Errorf("missing x/sys dependency evidence in %s", path)
		}
	}
	sort.Strings(items)
	items = compactStrings(items)
	sum := sha256.Sum256([]byte(strings.Join(items, "\n")))
	return hex.EncodeToString(sum[:16]), nil
}

// acquisitionGeneratorDigest conservatively binds every generator file. Runtime
// code could read or execute these files dynamically, even though Go cannot
// import a command package. Until that boundary is enforced, changes here must
// be treated as potentially affecting persistent-state behavior.
func acquisitionGeneratorDigest(root string) (string, error) {
	dir := filepath.Join(root, "packages/binaryfetch/cmd/acquisition-schema")
	hash := sha256.New()
	count := 0
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		count++
		if count > maxCodecPackageFiles {
			return fmt.Errorf("generator exceeds inventory bound")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > maxCodecSourceBytes {
			return fmt.Errorf("generator input %s is not a bounded regular file", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		fmt.Fprintf(hash, "%d:%s:%d:", len(rel), filepath.ToSlash(rel), len(data))
		_, err = hash.Write(data)
		return err
	})
	if err != nil {
		return "", err
	}
	if count == 0 {
		return "", fmt.Errorf("generator inventory is empty")
	}
	return hex.EncodeToString(hash.Sum(nil)[:16]), nil
}

func wholeSourceDigest(root string, patterns []string) (string, error) {
	hash := sha256.New()
	excluded := map[string]bool{}
	type sourceEntry struct {
		path   string
		fset   *token.FileSet
		source goSource
	}
	var entries []sourceEntry
	for _, pattern := range patterns {
		// A leading "!" excludes one repository-relative source file from a
		// broader directory inventory. Keeping exclusions exact avoids hiding
		// newly added source files from a whole-source owner.
		if strings.HasPrefix(pattern, "!") {
			relative := filepath.Clean(filepath.FromSlash(strings.TrimPrefix(pattern, "!")))
			if filepath.IsAbs(relative) || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || strings.Contains(relative, "*") {
				return "", fmt.Errorf("invalid whole-source exclusion %q", pattern)
			}
			excluded[filepath.ToSlash(relative)] = true
			continue
		}
		dir, prefix := pattern, ""
		if strings.HasSuffix(pattern, "*") {
			dir, prefix = filepath.Dir(pattern), strings.TrimSuffix(filepath.Base(pattern), "*")
		}
		fset := token.NewFileSet()
		loaded, err := loadGoSources(fset, filepath.Join(root, filepath.FromSlash(dir)), prefix)
		if err != nil {
			return "", err
		}
		for _, source := range loaded {
			rel, err := filepath.Rel(root, source.path)
			if err != nil {
				return "", err
			}
			entries = append(entries, sourceEntry{path: filepath.ToSlash(rel), fset: fset, source: source})
		}
	}
	selected := map[string]bool{}
	for _, entry := range entries {
		selected[entry.path] = true
	}
	for path := range excluded {
		if !selected[path] {
			return "", fmt.Errorf("whole-source exclusion %q did not match a selected source", path)
		}
	}
	remaining := false
	for _, entry := range entries {
		remaining = remaining || !excluded[entry.path]
	}
	if !remaining {
		return "", fmt.Errorf("whole-source inventory excludes every selected source")
	}
	for _, entry := range entries {
		if excluded[entry.path] {
			continue
		}
		fmt.Fprintf(hash, "file %s\n%s\n", entry.path, canonicalSource(entry.fset, entry.source.src, entry.source.file))
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
	// The same token stream can be selected on different targets after a
	// build-tag or filename change. Bind both even for selected subexpressions.
	constraints, _, _ := buildConstraints(src) // validated by loadGoSources
	return "file=" + filepath.Base(fset.Position(node.Pos()).Filename) + " constraints=" + constraints + " " + strings.Join(tokens, " ")
}

// buildConstraints records directives before the package clause. Skip only
// the conventional unconditional ignore tag; expressions such as !ignore
// and linux || ignore can compile under normal configurations.
func buildConstraints(src []byte) (string, bool, error) {
	var directives []string
	var modern []constraint.Expr
	var legacy []constraint.Expr
	file := token.NewFileSet().AddFile("", -1, len(src))
	var lexer scanner.Scanner
	lexer.Init(file, src, nil, scanner.ScanComments)
	for {
		_, tok, text := lexer.Scan()
		if tok == token.PACKAGE || tok == token.EOF {
			break
		}
		if tok != token.COMMENT || (!constraint.IsGoBuild(text) && !constraint.IsPlusBuild(text)) {
			continue
		}
		expr, err := constraint.Parse(text)
		if err != nil {
			return "", false, fmt.Errorf("invalid build constraint %q: %w", text, err)
		}
		if constraint.IsGoBuild(text) {
			modern = append(modern, expr)
			directives = append(directives, "go:"+expr.String())
		} else {
			legacy = append(legacy, expr)
			directives = append(directives, "legacy:"+expr.String())
		}
	}
	if len(modern) > 1 {
		return "", false, fmt.Errorf("multiple go:build constraints")
	}
	onlyIgnore := func(expr constraint.Expr) bool {
		tag, ok := expr.(*constraint.TagExpr)
		return ok && tag.Tag == "ignore"
	}
	ignored := len(modern) == 1 && onlyIgnore(modern[0])
	// Legacy +build directives require a separating blank line; conservatively
	// retain legacy-only files instead of guessing whether Go recognizes one.

	// A disagreement between modern and legacy constraints is invalid in Go's
	// tooling; keep it visible instead of hiding a potentially compiled file.
	if len(modern) == 1 && len(legacy) > 0 {
		ignored = ignored && len(legacy) == 1 && onlyIgnore(legacy[0])
	}
	return strings.Join(directives, ";"), ignored, nil
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
