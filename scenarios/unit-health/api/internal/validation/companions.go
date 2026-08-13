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
	Owner      string            `json:"owner"`
	ImportPath string            `json:"import_path"`
	Symbols    []companionSymbol `json:"symbols"`
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
}

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
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv != nil || d.Name == nil {
					continue
				}
				declarations = append(declarations, localDeclaration{
					Name: d.Name.Name, Kind: "function", Signature: nodeSignature(d.Type),
					File: path, Line: fset.Position(d.Pos()).Line,
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
						File:    path, Line: fset.Position(typeSpec.Pos()).Line,
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
			if byReceiver[receiver] == nil {
				byReceiver[receiver] = map[string]bool{}
			}
			if ast.IsExported(fn.Name.Name) {
				byReceiver[receiver][fn.Name.Name] = true
			}
		}
	})
	for i := range declarations {
		if declarations[i].Kind == "type" {
			declarations[i].Methods = byReceiver[declarations[i].Name]
			if declarations[i].Methods == nil {
				declarations[i].Methods = map[string]bool{}
			}
		}
	}
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

func receiverName(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
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
	declarations := collectLocalDeclarations(ws.RootPath)
	var findings []Finding
	for _, declaration := range declarations {
		if isCanonicalCompanionDeclaration(ws.RootPath, declaration.File) {
			continue
		}
		for _, export := range append(append([]companionExport(nil), registry.Seams...), registry.Companions...) {
			for _, symbol := range export.Symbols {
				if !companionSymbolMatches(declaration, symbol) {
					continue
				}
				code := codeCompanionAvailable
				severity := codeSeverity[codeCompanionAvailable]
				if closure.Available && importReachable(export.ImportPath, closure) {
					if containsExport(registry.Seams, export) {
						code = codeSeamReimplemented
					} else {
						code = codeCompanionReimplemented
					}
					severity = codeSeverity[code]
				}
				kind := "companion"
				if containsExport(registry.Seams, export) {
					kind = "seam"
				}
				findings = append(findings, Finding{
					ID:           fmt.Sprintf("%s-%s-%s-%d", code, ws.ID, declaration.Name, declaration.Line),
					Scenario:     scenario,
					WorkspaceID:  ws.ID,
					Language:     "go",
					Code:         code,
					Category:     "architecture",
					Severity:     severity,
					FilePath:     declaration.File,
					Symbol:       declaration.Name,
					Message:      fmt.Sprintf("Target declares %s %q also exported by %s.", kind, declaration.Name, export.ImportPath),
					Evidence:     fmt.Sprintf("%s:%d declaration=%s kind=%s closure_source=%s", relTo(ws.RootPath, declaration.File), declaration.Line, declaration.Name, declaration.Kind, orDefault(closure.Source, "unavailable")),
					Expected:     fmt.Sprintf("Adopt %s from %s instead of declaring a local equivalent.", declaration.Name, export.ImportPath),
					Observed:     fmt.Sprintf("local %s declaration matches the registered %s export", declaration.Name, kind),
					WhyItMatters: "One owned seam and one companion implementation keep fakes consistent, portable, and maintainable across targets.",
					Remediation:  fmt.Sprintf("Import %s and adopt %s; one shared %s improves consistency and portability across platforms and consumers.", export.ImportPath, declaration.Name, kind),
					CreatedAt:    now,
				})
				break
			}
		}
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

func isCanonicalCompanionDeclaration(root, file string) bool {
	cleanRoot := filepath.Clean(root)
	cleanFile := filepath.Clean(file)
	if cleanRoot == cleanFile || !strings.HasPrefix(cleanFile, cleanRoot+string(filepath.Separator)) {
		return false
	}
	rel, err := filepath.Rel(cleanRoot, cleanFile)
	if err != nil {
		return false
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, part := range parts {
		switch part {
		case "apihttptest", "databasetest", "servertest", "scheduletest", "cliapptest":
			return true
		}
	}
	return false
}

func mergeModuleClosure(root string, closure DependencyClosure) DependencyClosure {
	if closure.Imports == nil {
		closure.Imports = map[string]bool{}
	}
	module := goModulePath(root)
	if module != "" {
		closure.Imports[module] = true
	}
	raw := readNearestGoMod(root)
	inRequire := false
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
		if line == "require (" {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}
		if strings.HasPrefix(line, "require ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "require "))
		} else if !inRequire {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 && !strings.HasPrefix(fields[0], "(") {
			closure.Imports[fields[0]] = true
		}
	}
	return closure
}

func readNearestGoMod(root string) string {
	dir := root
	for i := 0; i < 12 && dir != ""; i++ {
		if raw := readFileString(filepath.Join(dir, "go.mod")); raw != "" {
			return raw
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
