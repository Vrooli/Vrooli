package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	apiDiscovery "github.com/vrooli/api-core/discovery"
)

// DependencyClosure is the portion of a target's dependency DAG relevant to
// companion ownership. Imports are module/package import paths; Targets keeps
// the scenario/resource/package names returned by the analyzer for evidence.
// Available is false when the dependency service could not be consulted. In
// that case the analyzer remains conservative and does not turn a suggestion
// into an error.
type DependencyClosure struct {
	Imports   map[string]bool
	Targets   map[string]bool
	Source    string
	Available bool
}

// DependencyResolver supplies the target-aware closure produced by Scenario
// Dependency Analyzer. It is injected so static analyzer tests never depend
// on a running service.
type DependencyResolver interface {
	Resolve(ctx context.Context, targetKind, targetID, targetRoot string) (DependencyClosure, error)
}

type companionRegistry struct {
	SchemaVersion string            `json:"schema_version"`
	Companions    []companionExport `json:"companions"`
	Seams         []companionExport `json:"seams"`
}

type companionExport struct {
	Owner string `json:"owner"`
	// OwnerImportPath is the package the companion serves. Shape matching uses
	// it to confirm the declaring package actually works with that seam.
	OwnerImportPath string            `json:"owner_import_path,omitempty"`
	ImportPath      string            `json:"import_path"`
	Symbols         []companionSymbol `json:"symbols"`
}

type companionSymbol struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Signature string   `json:"signature,omitempty"`
	Methods   []string `json:"methods,omitempty"`
}

type localDeclaration struct {
	Name      string
	Kind      string
	Signature string
	Methods   map[string]bool
	File      string
	Line      int
	TestOnly  bool
}

// companionMatch is the single export a local declaration was resolved to,
// together with the evidence that resolved it.
type companionMatch struct {
	Export companionExport
	Symbol companionSymbol
	Reason companionMatchReason
}

// companionMatchReason names how a local declaration was tied to an export. It
// is carried into the finding so a reader can tell an exact duplicate from an
// inference and judge it accordingly.
type companionMatchReason string

const (
	// matchByName is an exact duplicate: same name, kind, and shape.
	matchByName companionMatchReason = "name"
	// matchByAdaptation is the same named helper rebuilt around a different input.
	matchByAdaptation companionMatchReason = "adapted"
	// matchByShape is a type rebuilt under a different name.
	matchByShape companionMatchReason = "shape"
)

// sdaDependencyResolver is deliberately a small HTTP adapter around the
// analyzer's stable target-DAG export. The analyzer owns closure semantics;
// Unit Health only consumes names and module nodes and never reconstructs the
// scenario dependency graph itself.
type sdaDependencyResolver struct {
	Resolver interface {
		ResolveScenarioURLDefault(context.Context, string) (string, error)
	}
	HTTPClient *http.Client
}

func (r sdaDependencyResolver) Resolve(ctx context.Context, targetKind, targetID, _ string) (DependencyClosure, error) {
	resolver := r.Resolver
	if resolver == nil {
		resolver = apiDiscovery.NewResolver(apiDiscovery.ResolverConfig{})
	}
	client := r.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	expression := targetID
	if targetKind != "" && targetKind != "scenario" {
		expression = targetKind + ":" + targetID
	}
	base, err := resolver.ResolveScenarioURLDefault(ctx, "scenario-dependency-analyzer")
	if err != nil {
		return DependencyClosure{}, fmt.Errorf("resolve scenario-dependency-analyzer: %w", err)
	}
	endpoint := strings.TrimRight(base, "/") + "/targets/" + url.PathEscape(expression) + "/dag/export?recursive=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return DependencyClosure{}, fmt.Errorf("build dependency DAG request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return DependencyClosure{}, fmt.Errorf("request dependency DAG: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return DependencyClosure{}, fmt.Errorf("dependency DAG returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var payload struct {
		DAG []dependencyNode `json:"dag"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return DependencyClosure{}, fmt.Errorf("decode dependency DAG: %w", err)
	}
	closure := DependencyClosure{Imports: map[string]bool{}, Targets: map[string]bool{}, Source: "scenario-dependency-analyzer", Available: true}
	for _, node := range payload.DAG {
		collectDependencyNode(node, &closure)
	}
	return closure, nil
}

type dependencyNode struct {
	Name     string           `json:"name"`
	Children []dependencyNode `json:"children"`
}

func collectDependencyNode(node dependencyNode, closure *DependencyClosure) {
	if strings.TrimSpace(node.Name) != "" {
		closure.Targets[node.Name] = true
		if strings.Contains(node.Name, "/") || strings.Contains(node.Name, ".") {
			closure.Imports[node.Name] = true
		}
	}
	for _, child := range node.Children {
		collectDependencyNode(child, closure)
	}
}

func loadCompanionRegistry(root string) (companionRegistry, bool) {
	dir := root
	for i := 0; i < 12 && dir != ""; i++ {
		path := filepath.Join(dir, ".vrooli", "test-companions.json")
		raw, err := os.ReadFile(path)
		if err == nil {
			var registry companionRegistry
			if json.Unmarshal(raw, &registry) == nil && registry.SchemaVersion == "1.0.0" {
				return registry, true
			}
			return companionRegistry{}, false
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return companionRegistry{}, false
}

func collectLocalDeclarations(root string) []localDeclaration {
	var declarations []localDeclaration
	walkSourceFiles(root, func(path string) {
		if !strings.HasSuffix(path, ".go") {
			return
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil || file == nil {
			return
		}
		testOnly := isTestSupportFile(path)
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv != nil || d.Name == nil {
					continue
				}
				declarations = append(declarations, localDeclaration{
					Name: d.Name.Name, Kind: "function", Signature: nodeSignature(d.Type),
					File: path, Line: fset.Position(d.Pos()).Line, TestOnly: testOnly,
				})
			case *ast.GenDecl:
				if d.Tok != token.TYPE {
					continue
				}
				for _, spec := range d.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok || typeSpec.Name == nil {
						continue
					}
					kind := "type"
					var methods map[string]bool
					if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
						kind = "interface"
						methods = interfaceMethodNames(interfaceType)
					}
					declarations = append(declarations, localDeclaration{
						Name: typeSpec.Name.Name, Kind: kind,
						Methods: methods,
						File:    path, Line: fset.Position(typeSpec.Pos()).Line, TestOnly: testOnly,
					})
				}
			}
		}
	})
	enrichMethodSets(root, declarations)
	sort.SliceStable(declarations, func(i, j int) bool {
		if declarations[i].File != declarations[j].File {
			return declarations[i].File < declarations[j].File
		}
		return declarations[i].Line < declarations[j].Line
	})
	return declarations
}

func nodeSignature(node ast.Node) string {
	var buf strings.Builder
	if err := format.Node(&buf, token.NewFileSet(), node); err != nil {
		return ""
	}
	return strings.Join(strings.Fields(buf.String()), " ")
}

// enrichMethodSets attaches each named type's exported method set. Methods are
// keyed by declaring directory as well as by receiver name: a workspace holds
// many packages, and two unrelated packages routinely declare the same
// unexported fake. Keying on the receiver alone merged their method sets, which
// both invented methods a type does not have and hid the ones it does.
func enrichMethodSets(root string, declarations []localDeclaration) {
	byReceiver := map[string]map[string]bool{}
	walkSourceFiles(root, func(path string) {
		if !strings.HasSuffix(path, ".go") {
			return
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil || file == nil {
			return
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 || fn.Name == nil {
				continue
			}
			receiver := receiverName(fn.Recv.List[0].Type)
			if receiver == "" {
				continue
			}
			key := methodSetKey(path, receiver)
			if byReceiver[key] == nil {
				byReceiver[key] = map[string]bool{}
			}
			if ast.IsExported(fn.Name.Name) {
				byReceiver[key][fn.Name.Name] = true
			}
		}
	})
	for i := range declarations {
		if declarations[i].Kind == "type" {
			declarations[i].Methods = byReceiver[methodSetKey(declarations[i].File, declarations[i].Name)]
			if declarations[i].Methods == nil {
				declarations[i].Methods = map[string]bool{}
			}
		}
	}
}

func methodSetKey(file, receiver string) string {
	return filepath.Dir(file) + "\x00" + receiver
}

// isTestSupportFile reports whether a Go file is test scaffolding: a _test.go
// file, or a file in a package whose whole job is test support.
//
// Structural matching is restricted to these files on purpose. A production
// type that happens to share a fake's shape is not a duplicate — the shared
// package convention deliberately keeps runtime implementations such as
// blobstore.Memory in the production package — so accusing production code of
// reimplementing a test companion would be wrong, not merely noisy.
func isTestSupportFile(file string) bool {
	if isGoTestFile(file) {
		return true
	}
	for _, part := range strings.Split(filepath.ToSlash(filepath.Dir(file)), "/") {
		switch part {
		case "testutil", "testutils", "testkit", "testing", "testhelper", "testhelpers", "testsupport", "mocks", "fixtures":
			return true
		}
	}
	return false
}

func interfaceMethodNames(interfaceType *ast.InterfaceType) map[string]bool {
	methods := map[string]bool{}
	if interfaceType == nil || interfaceType.Methods == nil {
		return methods
	}
	for _, field := range interfaceType.Methods.List {
		for _, name := range field.Names {
			if ast.IsExported(name.Name) {
				methods[name.Name] = true
			}
		}
	}
	return methods
}

// receiverName resolves the type name a method is declared on, unwrapping the
// pointer and any type parameters. A generic receiver such as
// `(r *SliceRepo[T])` parses as an index expression, and failing to unwrap it
// dropped every method of every generic type — which read as "this type exposes
// nothing" rather than as a parse gap.
func receiverName(expr ast.Expr) string {
	for {
		switch typed := expr.(type) {
		case *ast.StarExpr:
			expr = typed.X
		case *ast.IndexExpr:
			expr = typed.X
		case *ast.IndexListExpr:
			expr = typed.X
		case *ast.Ident:
			return typed.Name
		default:
			return ""
		}
	}
}

func importReachable(importPath string, closure DependencyClosure) bool {
	if closure.Imports[importPath] {
		return true
	}
	for module := range closure.Imports {
		if strings.HasPrefix(importPath, module+"/") || strings.HasPrefix(module, importPath+"/") {
			return true
		}
	}
	return false
}

func companionSymbolMatches(local localDeclaration, symbol companionSymbol) bool {
	if local.Name != symbol.Name || local.Kind != symbol.Kind {
		return false
	}
	if symbol.Signature != "" && local.Signature != normalizeSignature(symbol.Signature) {
		return false
	}
	if len(symbol.Methods) == 0 {
		return true
	}
	if len(local.Methods) != len(symbol.Methods) {
		return false
	}
	for _, method := range symbol.Methods {
		if !local.Methods[method] {
			return false
		}
	}
	return true
}

// packageImportIndex caches each directory's import set for the life of one
// analysis. Shape matching asks the same question of every declaration in a
// package, and re-reading the package per declaration is wasted work.
type packageImportIndex map[string]map[string]bool

func (index packageImportIndex) imports(dir string) map[string]bool {
	if set, ok := index[dir]; ok {
		return set
	}
	set := map[string]bool{}
	if entries, err := os.ReadDir(dir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
				continue
			}
			for _, imported := range goImports(filepath.Join(dir, entry.Name())) {
				set[imported] = true
			}
		}
	}
	index[dir] = set
	return set
}

// declarationWorksWithOwner reports whether the package that declares a local
// type also works with the package the companion serves.
//
// Shape matching infers identity from method names alone, which is the weakest
// evidence the matcher uses and the only tier that can mistake one domain for
// another: swarm-manager's fakeGoalReader exposes {Get, List}, a subset of
// databasetest.SliceRepo, while having nothing to do with a repository. The
// declaring package there never imports api-core/database, whereas every real
// clock fake sits in a package that imports api-core/schedule. Requiring that
// the seam is actually in use is what separates the two, and it does so without
// a method-count floor high enough to miss the two-method fakes that make up
// most of the fleet's real duplicates.
func declarationWorksWithOwner(index packageImportIndex, local localDeclaration, export companionExport) bool {
	imported := index.imports(filepath.Dir(local.File))
	if export.OwnerImportPath != "" && imported[export.OwnerImportPath] {
		return true
	}
	return imported[export.ImportPath]
}

// companionSymbolReimplemented reports whether a test-only declaration rebuilds
// a registered companion type's shape. Name matching alone cannot see this:
// the ordinary Go idiom is an unexported fake, so `fakeClock` never matches
// `FakeClock` no matter how exactly it reimplements it.
//
// The floor of two shared methods is deliberate. One shared method is not
// evidence — nearly every fake in the fleet has a Get or a Now, and reporting
// all of them would bury the real duplicates. A one-method fake is therefore
// treated as unprovable rather than accused.
func companionSymbolReimplemented(local localDeclaration, symbol companionSymbol) bool {
	if local.Kind != symbol.Kind {
		return false
	}
	if symbol.Kind != "type" && symbol.Kind != "interface" {
		return false
	}
	if len(local.Methods) < 2 || len(local.Methods) > len(symbol.Methods) {
		return false
	}
	owned := make(map[string]bool, len(symbol.Methods))
	for _, method := range symbol.Methods {
		owned[method] = true
	}
	for method := range local.Methods {
		if !owned[method] {
			return false
		}
	}
	return true
}

// companionHelperAdapted reports whether a test-only declaration is a
// registered helper rebuilt around a different input. Exact matching cannot see
// these: vrooli-autoheal's MustDecodeJSON takes the recorder where the
// companion takes the body, which is the same duty adapted to a local idiom,
// not a different helper.
//
// Both sides must take a testing handle. That is what separates a test helper
// from an ordinary constructor, and without it a companion named New or
// Response would collide with a large share of the fleet's unrelated functions
// — measured against this tree, every same-named local New is a plain
// constructor and none takes a testing handle.
func companionHelperAdapted(local localDeclaration, symbol companionSymbol) bool {
	if symbol.Kind != "function" || local.Kind != symbol.Kind || local.Name != symbol.Name {
		return false
	}
	return takesTestingHandle(local.Signature) && takesTestingHandle(symbol.Signature)
}

func takesTestingHandle(signature string) bool {
	return strings.Contains(signature, "*testing.T") || strings.Contains(signature, "testing.TB")
}

// bestCompanionMatch resolves a local declaration to at most one registered
// export, strongest evidence first: an exact match, then a test helper rebuilt
// around a different input, then a type rebuilt under another name. Seams are
// considered before companions so a local copy of an owned interface reports as
// a seam rather than as the fake built on top of it. Returning a single match
// keeps one declaration from producing a finding per registered export.
func bestCompanionMatch(local localDeclaration, registry companionRegistry, index packageImportIndex) (companionMatch, bool) {
	exports := registryExports(registry)
	for _, export := range exports {
		for _, symbol := range export.Symbols {
			if companionSymbolMatches(local, symbol) {
				return companionMatch{Export: export, Symbol: symbol, Reason: matchByName}, true
			}
		}
	}
	if !local.TestOnly {
		return companionMatch{}, false
	}
	for _, export := range exports {
		for _, symbol := range export.Symbols {
			if companionHelperAdapted(local, symbol) {
				return companionMatch{Export: export, Symbol: symbol, Reason: matchByAdaptation}, true
			}
		}
	}
	for _, export := range exports {
		if !declarationWorksWithOwner(index, local, export) {
			continue
		}
		for _, symbol := range export.Symbols {
			if companionSymbolReimplemented(local, symbol) {
				return companionMatch{Export: export, Symbol: symbol, Reason: matchByShape}, true
			}
		}
	}
	return companionMatch{}, false
}

// registryExports lists seams before companions so seam ownership wins ties.
func registryExports(registry companionRegistry) []companionExport {
	exports := make([]companionExport, 0, len(registry.Seams)+len(registry.Companions))
	exports = append(exports, registry.Seams...)
	exports = append(exports, registry.Companions...)
	return exports
}

func normalizeSignature(signature string) string { return strings.Join(strings.Fields(signature), " ") }

func importsRegisteredCompanion(imports map[string]bool, registry companionRegistry) bool {
	for _, companion := range registry.Companions {
		if imports[companion.ImportPath] {
			return true
		}
	}
	return false
}

func analyzeGoCompanionDeclarations(scenario string, ws Workspace, now string, closure DependencyClosure) []Finding {
	registry, ok := loadCompanionRegistry(ws.RootPath)
	if !ok {
		return nil
	}
	return analyzeCompanionDeclarations(scenario, ws, now, registry, closure)
}

func analyzeCompanionDeclarations(scenario string, ws Workspace, now string, registry companionRegistry, closure DependencyClosure) []Finding {
	closure = mergeModuleClosure(ws.RootPath, closure)
	declarations := collectLocalDeclarations(ws.RootPath)
	index := packageImportIndex{}
	var findings []Finding
	for _, declaration := range declarations {
		if isCanonicalCompanionDeclaration(ws.RootPath, declaration.File, registry) {
			continue
		}
		match, ok := bestCompanionMatch(declaration, registry, index)
		if !ok {
			continue
		}
		kind := "companion"
		if containsExport(registry.Seams, match.Export) {
			kind = "seam"
		}
		code := codeCompanionAvailable
		if closure.Available && importReachable(match.Export.ImportPath, closure) {
			if kind == "seam" {
				code = codeSeamReimplemented
			} else {
				code = codeCompanionReimplemented
			}
		}
		owner := match.Export.ImportPath + "." + match.Symbol.Name
		var message, observed string
		switch match.Reason {
		case matchByAdaptation:
			message = fmt.Sprintf("Test helper %q reimplements %s around a different input.", declaration.Name, owner)
			observed = fmt.Sprintf("local %s%s against companion %s", declaration.Name, trimFuncKeyword(declaration.Signature), trimFuncKeyword(match.Symbol.Signature))
		case matchByShape:
			message = fmt.Sprintf("Test %s %q reimplements the shape of %s.", declaration.Kind, declaration.Name, owner)
			observed = fmt.Sprintf("local %s exposes {%s}, contained in the method set of %s", declaration.Name, strings.Join(sortedMethodNames(declaration.Methods), ", "), owner)
		default:
			message = fmt.Sprintf("Target declares %s %q also exported by %s.", kind, declaration.Name, match.Export.ImportPath)
			observed = fmt.Sprintf("local %s declaration matches the registered %s export", declaration.Name, kind)
		}
		detail := fmt.Sprintf("declaration=%s kind=%s match=%s", declaration.Name, declaration.Kind, match.Reason)
		if match.Reason == matchByShape {
			detail += fmt.Sprintf(" methods={%s}", strings.Join(sortedMethodNames(declaration.Methods), ", "))
		}
		findings = append(findings, Finding{
			ID:           fmt.Sprintf("%s-%s-%s-%d", code, ws.ID, declaration.Name, declaration.Line),
			Scenario:     scenario,
			WorkspaceID:  ws.ID,
			Language:     "go",
			Code:         code,
			Category:     "architecture",
			Severity:     codeSeverity[code],
			FilePath:     declaration.File,
			Symbol:       declaration.Name,
			Message:      message,
			Evidence:     fmt.Sprintf("%s:%d %s closure_source=%s", relTo(ws.RootPath, declaration.File), declaration.Line, detail, orDefault(closure.Source, "unavailable")),
			Expected:     fmt.Sprintf("Adopt %s instead of declaring a local equivalent.", owner),
			Observed:     observed,
			WhyItMatters: "One owned seam and one companion implementation keep fakes consistent, portable, and maintainable across targets.",
			Remediation:  fmt.Sprintf("Import %s and adopt %s; one shared %s improves consistency and portability across platforms and consumers.", match.Export.ImportPath, match.Symbol.Name, kind),
			CreatedAt:    now,
		})
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].FilePath != findings[j].FilePath {
			return findings[i].FilePath < findings[j].FilePath
		}
		return findings[i].Symbol < findings[j].Symbol
	})
	return findings
}

func containsExport(exports []companionExport, candidate companionExport) bool {
	for _, export := range exports {
		if export.ImportPath == candidate.ImportPath {
			return true
		}
	}
	return false
}

// isCanonicalCompanionDeclaration reports whether a file is part of a
// registered export's own package, so a companion never reports itself when the
// package that owns it is the validation target.
//
// Ownership is decided by import path rather than by directory name. A name
// test would also exclude any scenario package that happens to share a base
// name with a registered export — `internal/schedule` is exactly the local
// duplicate of `api-core/schedule` these rules exist to catch. Deriving this
// from the registry is also what lets a newly registered companion work with no
// matching code change.
func isCanonicalCompanionDeclaration(root, file string, registry companionRegistry) bool {
	declared := declarationImportPath(root, file)
	if declared == "" {
		return false
	}
	for _, export := range registryExports(registry) {
		if export.ImportPath == declared {
			return true
		}
	}
	return false
}

// declarationImportPath resolves the import path of the package a file belongs
// to, using the workspace's own module path. It returns "" when the workspace
// declares no module, which leaves every declaration eligible for analysis.
func declarationImportPath(root, file string) string {
	module := goModulePath(root)
	if module == "" {
		return ""
	}
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Dir(filepath.Clean(file)))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	if rel == "." {
		return module
	}
	return module + "/" + filepath.ToSlash(rel)
}

// trimFuncKeyword renders a signature for a message where the name already
// supplies the "func" part.
func trimFuncKeyword(signature string) string {
	return strings.TrimPrefix(signature, "func")
}

func sortedMethodNames(methods map[string]bool) []string {
	names := make([]string, 0, len(methods))
	for name := range methods {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// mergeModuleClosure returns closure widened with the module graph the
// workspace's own go.mod declares. It never mutates its argument: companion
// analysis runs once per workspace and each workspace owns a distinct module,
// so a shared map would leak one workspace's requirements into the next.
//
// The go.mod evidence is deliberately independent of Scenario Dependency
// Analyzer. A `require github.com/vrooli/api-core` line proves on its own that
// the companion is importable here, so reachability must not collapse to
// "unknown" whenever the analyzer happens to be stopped — that silently demotes
// every COMPANION_REIMPLEMENTED error to a COMPANION_AVAILABLE suggestion and
// makes the rule non-gating without saying so. The analyzer remains the
// authority on the wider target DAG; go.mod is the always-available floor.
//
// Only a go.mod at root counts. Walking up to an ancestor would let a workspace
// with no module of its own inherit the control plane's requirements and claim
// a reachability it does not have.
func mergeModuleClosure(root string, closure DependencyClosure) DependencyClosure {
	merged := DependencyClosure{
		Imports:   copyStringSet(closure.Imports),
		Targets:   copyStringSet(closure.Targets),
		Source:    closure.Source,
		Available: closure.Available,
	}
	raw := readFileString(filepath.Join(root, "go.mod"))
	if raw == "" {
		return merged
	}
	for _, path := range goModPaths(raw) {
		merged.Imports[path] = true
	}
	merged.Available = true
	merged.Source = mergeClosureSource(closure.Source, "go.mod")
	return merged
}

// goModPaths returns the module path and every required module path declared in
// a go.mod, in declaration order. Replace directives are ignored: they retarget
// a requirement, they do not add one.
func goModPaths(raw string) []string {
	var paths []string
	inRequire := false
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
		switch {
		case line == "require (":
			inRequire = true
			continue
		case inRequire && line == ")":
			inRequire = false
			continue
		case strings.HasPrefix(line, "module "):
			if path := strings.TrimSpace(strings.TrimPrefix(line, "module ")); path != "" {
				paths = append(paths, path)
			}
			continue
		case strings.HasPrefix(line, "require "):
			line = strings.TrimSpace(strings.TrimPrefix(line, "require "))
		case !inRequire:
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 && !strings.HasPrefix(fields[0], "(") {
			paths = append(paths, fields[0])
		}
	}
	return paths
}

// mergeClosureSource records every evidence source that contributed to a
// closure so a finding's evidence line stays auditable.
func mergeClosureSource(existing, added string) string {
	existing = strings.TrimSpace(existing)
	if existing == "" {
		return added
	}
	for _, part := range strings.Split(existing, "+") {
		if part == added {
			return existing
		}
	}
	return existing + "+" + added
}

func copyStringSet(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
