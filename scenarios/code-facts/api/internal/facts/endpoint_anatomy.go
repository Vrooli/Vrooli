package facts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

const maxEndpointAnatomyResults = 10
const maxEndpointRouteCandidates = 12

type EndpointSource struct {
	Path   string `json:"path"`
	Line   int    `json:"line"`
	SHA256 string `json:"sha256"`
	Basis  string `json:"basis"`
}

type EndpointAnatomy struct {
	ID                     string                   `json:"id"`
	Title                  string                   `json:"title"`
	Summary                string                   `json:"summary"`
	Scenario               string                   `json:"scenario"`
	RoutePath              string                   `json:"route_path"`
	Method                 string                   `json:"method"`
	Handler                string                   `json:"handler"`
	SourcePath             string                   `json:"source_path"`
	HandlerLine            int                      `json:"handler_line"`
	SourceSHA256           string                   `json:"source_sha256"`
	State                  string                   `json:"state"`
	Declaration            EndpointSource           `json:"declaration"`
	Registration           EndpointSource           `json:"registration"`
	HandlerSource          EndpointSource           `json:"handler_source"`
	Transport              string                   `json:"transport,omitempty"`
	ConnectService         string                   `json:"connect_service,omitempty"`
	ConnectMethod          string                   `json:"connect_method,omitempty"`
	ConnectChain           []EndpointSource         `json:"connect_chain,omitempty"`
	RouteCandidates        []EndpointRouteCandidate `json:"route_candidates,omitempty"`
	RouteCandidateTotal    int                      `json:"route_candidate_total,omitempty"`
	RouteCandidatesOmitted int                      `json:"route_candidates_omitted,omitempty"`
	Limits                 []string                 `json:"limits"`
	Score                  float64                  `json:"score"`
}

// EndpointRouteCandidate is one source-backed registration matching a declared
// HTTP endpoint. Multiple candidates are kept explicit; source order is not a
// safe basis for selecting a handler.
type EndpointRouteCandidate struct {
	Handler       string         `json:"handler"`
	Registration  EndpointSource `json:"registration"`
	HandlerSource EndpointSource `json:"handler_source"`
}

type EndpointAnatomyResponse struct {
	Query        string            `json:"query"`
	Scenario     string            `json:"scenario"`
	Scope        string            `json:"scope"`
	Results      []EndpointAnatomy `json:"results"`
	Degradations []string          `json:"degradations"`
}

type apiDeclaration struct {
	id, path, method, summary string
	pathExpr                  string
	line                      int
}
type apiRoute struct {
	path, method, handler string
	line                  int
}

// SearchEndpointAnatomy joins literal HTTP endpoint descriptors in a scenario's
// facts module with mux registration and the handler function definition. It is
// static source evidence; it does not prove the module is mounted at runtime.
func (s *Service) SearchEndpointAnatomy(_ context.Context, query, scenario string, limit int) (*EndpointAnatomyResponse, error) {
	query, scenario = strings.TrimSpace(query), strings.TrimSpace(scenario)
	if query == "" || !invariantScenarioPattern.MatchString(scenario) {
		return nil, fmt.Errorf("query and one scenario slug are required")
	}
	if limit <= 0 || limit > maxEndpointAnatomyResults {
		limit = maxEndpointAnatomyResults
	}
	target, err := resolveTarget(&factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: scenario})
	if err != nil {
		return nil, err
	}
	root, err := filepath.EvalSymlinks(target.GetRootPath())
	if err != nil {
		return nil, err
	}
	return scanEndpointAnatomy(root, scenario, query, limit)
}

func scanEndpointAnatomy(root, scenario, query string, limit int) (*EndpointAnatomyResponse, error) {
	if scenario != "code-facts" {
		return nil, fmt.Errorf("endpoint anatomy is currently bounded to scenario code-facts")
	}
	modulePath := filepath.Join(root, "api", "handlers", "facts", "module.go")
	moduleInfo, err := os.Lstat(modulePath)
	if err != nil {
		return nil, err
	}
	if !moduleInfo.Mode().IsRegular() || moduleInfo.Size() > 2<<20 {
		return nil, fmt.Errorf("endpoint module source exceeds 2 MiB cap")
	}
	moduleBytes, err := os.ReadFile(modulePath)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, modulePath, moduleBytes, parser.AllErrors)
	if err != nil {
		return nil, err
	}
	moduleHash := hashBytes(moduleBytes)
	declarations := declaredEndpoints(file, fset)
	routes := registeredRoutes(file, fset)
	response := &EndpointAnatomyResponse{Query: query, Scenario: scenario, Scope: "scenario:code-facts/api/handlers/facts only; literal HTTP JSON routes and bounded Connect service/method declarations with explicit local RegisterServices mount chain; module file max 2 MiB, handler lookup max 160 files/16 MiB, generated Connect source lookup is package-local; runtime module mounting remains unverified", Results: []EndpointAnatomy{}, Degradations: []string{}}
	terms := strings.Fields(strings.ToLower(query))
	connectRows, connectDegradations := scanConnectEndpointAnatomy(root, file, fset, moduleHash, declarations, terms)
	response.Results = append(response.Results, connectRows...)
	response.Degradations = append(response.Degradations, connectDegradations...)
	for _, declaration := range declarations {
		if declaration.pathExpr != "" {
			continue
		}
		if !strings.HasPrefix(declaration.path, "/api/v1/") {
			continue
		}
		row := EndpointAnatomy{ID: "code-facts:endpoint:" + declaration.id, Title: declaration.method + " " + declaration.path, Scenario: scenario, RoutePath: declaration.path, Method: declaration.method, State: "DECLARED_ONLY", Declaration: EndpointSource{Path: "api/handlers/facts/module.go", Line: declaration.line, SHA256: moduleHash, Basis: "module.EndpointDescriptor literal"}, Limits: []string{"Static source correspondence does not prove runtime route mounting or successful request handling.", "Only literal HTTP JSON routes in this one module are in scope; Connect/protobuf endpoints are not classified.", "Handler lookup is capped at 160 source files and 16 MiB."}}
		row.Summary = declaration.summary
		var matchingRoutes []apiRoute
		for _, route := range routes {
			if route.path == declaration.path && strings.EqualFold(route.method, declaration.method) {
				matchingRoutes = append(matchingRoutes, route)
			}
		}
		if len(matchingRoutes) > 1 {
			row.State = "ROUTE_AMBIGUOUS"
			row.RouteCandidateTotal = len(matchingRoutes)
			candidateLimit := len(matchingRoutes)
			if candidateLimit > maxEndpointRouteCandidates {
				candidateLimit = maxEndpointRouteCandidates
				row.RouteCandidatesOmitted = len(matchingRoutes) - candidateLimit
			}
			handlerSources := make(map[string]EndpointSource)
			for _, route := range matchingRoutes[:candidateLimit] {
				handlerSource, cached := handlerSources[route.handler]
				if !cached {
					handlerSource = findHandlerSource(root, route.handler)
					handlerSources[route.handler] = handlerSource
				}
				row.RouteCandidates = append(row.RouteCandidates, EndpointRouteCandidate{
					Handler:       route.handler,
					Registration:  EndpointSource{Path: "api/handlers/facts/module.go", Line: route.line, SHA256: moduleHash, Basis: "mux Handle(...).Methods(...) registration candidate"},
					HandlerSource: handlerSource,
				})
			}
			row.Limits = append(row.Limits, "Multiple literal mux registrations match this declared method and path; no handler was selected and runtime routing precedence is unknown.")
			if row.RouteCandidatesOmitted > 0 {
				row.Limits = append(row.Limits, fmt.Sprintf("Candidate provenance is capped at %d of %d matching registrations; %d candidates were omitted.", candidateLimit, len(matchingRoutes), row.RouteCandidatesOmitted))
			}
		} else if len(matchingRoutes) == 1 {
			route := matchingRoutes[0]
			row.Handler = route.handler
			row.State = "ROUTE_REGISTERED"
			row.Registration = EndpointSource{Path: "api/handlers/facts/module.go", Line: route.line, SHA256: moduleHash, Basis: "mux Handle(...).Methods(...) registration"}
		}
		if row.Handler != "" {
			row.HandlerSource = findHandlerSource(root, row.Handler)
			row.SourcePath = row.HandlerSource.Path
			row.HandlerLine = row.HandlerSource.Line
			row.SourceSHA256 = row.HandlerSource.SHA256
		}
		if row.State == "ROUTE_AMBIGUOUS" {
			// Keep the row partial: candidate provenance is useful evidence, but
			// neither source order nor handler lookup resolves dispatch.
		} else if row.State == "ROUTE_REGISTERED" && row.HandlerSource.Path != "" {
			row.State = "DECLARATION_ROUTE_HANDLER_JOINED"
		} else if row.State == "ROUTE_REGISTERED" {
			row.Limits = append(row.Limits, "Registered handler symbol definition was not found in the bounded handler package scan.")
		} else {
			row.Limits = append(row.Limits, "No matching literal mux registration was found in module.go; this does not rule out indirect registration.")
		}
		haystack := strings.ToLower(strings.Join([]string{row.ID, row.Title, row.Summary, row.Handler, row.State}, " "))
		row.Score = tokenScore(terms, haystack)
		if row.Score > 0 {
			response.Results = append(response.Results, row)
		}
	}
	sort.Slice(response.Results, func(i, j int) bool {
		if response.Results[i].Score != response.Results[j].Score {
			return response.Results[i].Score > response.Results[j].Score
		}
		return response.Results[i].ID < response.Results[j].ID
	})
	if len(response.Results) > limit {
		response.Results = response.Results[:limit]
	}
	if len(response.Results) == 0 {
		response.Degradations = append(response.Degradations, "no_matching_declared_endpoint: query matched no in-scope literal endpoint descriptor")
	}
	return response, nil
}

func declaredEndpoints(file *ast.File, fset *token.FileSet) []apiDeclaration {
	var out []apiDeclaration
	ast.Inspect(file, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if name, ok := call.Fun.(*ast.Ident); ok && (name.Name == "indexEndpoint" || name.Name == "sharedControlEndpoint") && len(call.Args) >= 3 {
				row := apiDeclaration{id: stringValue(call.Args[0]), method: "POST", summary: stringValue(call.Args[2]), pathExpr: expressionName(call.Args[1]), line: fset.Position(call.Pos()).Line}
				if row.id != "" && row.pathExpr != "" {
					out = append(out, row)
				}
			}
		}
		lit, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		typ, ok := lit.Type.(*ast.SelectorExpr)
		if !ok || typ.Sel.Name != "EndpointDescriptor" {
			return true
		}
		row := apiDeclaration{line: fset.Position(lit.Pos()).Line}
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			switch key.Name {
			case "ID":
				row.id = stringValue(kv.Value)
			case "Path":
				row.path = stringValue(kv.Value)
				if row.path == "" {
					row.pathExpr = expressionName(kv.Value)
				}
			case "Method":
				row.method = stringValue(kv.Value)
			case "Summary":
				row.summary = stringValue(kv.Value)
			}
		}
		if row.id != "" && (row.path != "" || row.pathExpr != "") && row.method != "" {
			out = append(out, row)
		}
		return true
	})
	return out
}

func registeredRoutes(file *ast.File, fset *token.FileSet) []apiRoute {
	var out []apiRoute
	ast.Inspect(file, func(node ast.Node) bool {
		outer, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := outer.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Methods" || len(outer.Args) == 0 {
			return true
		}
		inner, ok := sel.X.(*ast.CallExpr)
		if !ok {
			return true
		}
		handle, ok := inner.Fun.(*ast.SelectorExpr)
		if !ok || handle.Sel.Name != "Handle" || len(inner.Args) < 2 {
			return true
		}
		path := stringValue(inner.Args[0])
		methods := methodValue(outer.Args[0])
		handler := firstCallIdentifier(inner.Args[1])
		if path != "" && methods != "" && handler != "" {
			out = append(out, apiRoute{path: path, method: strings.ToUpper(methods), handler: handler, line: fset.Position(inner.Pos()).Line})
		}
		return true
	})
	return out
}

func findHandlerSource(root, name string) EndpointSource {
	dir := filepath.Join(root, "api", "handlers", "facts")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) > 160 {
		return EndpointSource{}
	}
	var total int64
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.Size() > 2<<20 || total+info.Size() > 16<<20 {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil || int64(len(data)) != info.Size() {
			continue
		}
		total += info.Size()
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, data, parser.AllErrors)
		if err != nil {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Name.Name == name {
				sum := sha256.Sum256(data)
				rel, _ := filepath.Rel(root, path)
				return EndpointSource{Path: filepath.ToSlash(rel), Line: fset.Position(fn.Pos()).Line, SHA256: hex.EncodeToString(sum[:]), Basis: "handler function definition"}
			}
		}
	}
	return EndpointSource{}
}

func stringValue(expr ast.Expr) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return ""
	}
	return value
}
func firstCallIdentifier(expr ast.Expr) string {
	var result string
	ast.Inspect(expr, func(node ast.Node) bool {
		if result != "" {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			result = fn.Name
		case *ast.SelectorExpr:
			result = fn.Sel.Name
		}
		return false
	})
	return result
}
func methodValue(expr ast.Expr) string {
	sel, ok := expr.(*ast.SelectorExpr)
	if ok && strings.HasPrefix(sel.Sel.Name, "Method") {
		return strings.TrimPrefix(sel.Sel.Name, "Method")
	}
	return stringValue(expr)
}
func hashBytes(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
