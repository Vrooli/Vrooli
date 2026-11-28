package domains

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	maxDriftFiles        = 180
	maxDriftBytes        = 16 << 20
	maxDriftResults      = 10
	maxDriftHandlerFiles = 80
	maxDriftHandlerBytes = 8 << 20
	maxDriftJoinFiles    = 160
	maxDriftJoinBytes    = 16 << 20
)

type DriftEvidence struct {
	Path   string `json:"path"`
	Line   int    `json:"line"`
	SHA256 string `json:"sha256"`
	Basis  string `json:"basis"`
}
type DocCodeDrift struct {
	ID             string          `json:"id"`
	Title          string          `json:"title"`
	Summary        string          `json:"summary"`
	Scenario       string          `json:"scenario"`
	Domain         string          `json:"domain"`
	Claim          string          `json:"claim"`
	Classification string          `json:"classification"`
	Status         string          `json:"status"`
	Documentation  DriftEvidence   `json:"documentation"`
	Code           []DriftEvidence `json:"code"`
	APIStatus      string          `json:"api_status,omitempty"`
	API            []DriftEvidence `json:"api,omitempty"`
	Limits         []string        `json:"limits"`
	Score          float64         `json:"score"`
}
type DocCodeDriftResponse struct {
	Query        string         `json:"query"`
	Scope        string         `json:"scope"`
	Authority    string         `json:"authority"`
	Results      []DocCodeDrift `json:"results"`
	Degradations []string       `json:"degradations"`
}

type driftSource struct {
	path, fact string
	line       int
	hash       string
	test       bool
	functions  []driftFunction
	routes     []driftRoute
}
type driftCall struct {
	importPath, symbol string
	line               int
}
type driftFunction struct {
	name  string
	line  int
	calls []driftCall
}
type driftRoute struct {
	path, method, handler string
	line                  int
	literal               bool
}
type driftDoc struct {
	domain         string
	paths          []string
	responsibility string
	line           int
}

// SearchDocCodeDrift compares authored API source-path claims from one curated
// DOMAINS.md with a bounded API-source inventory. It reports structural path
// candidates only, never semantic or runtime conformance. Supported literal
// route syntax includes Gorilla Methods chains and statically identified
// net/http ServeMux registrations.
func SearchDocCodeDrift(root string, domainMap DerivedDomainMap, query string, limit int) (*DocCodeDriftResponse, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if limit <= 0 || limit > maxDriftResults {
		limit = maxDriftResults
	}
	response := &DocCodeDriftResponse{Query: query, Scope: fmt.Sprintf("scenario:%s; curated docs claims vs bounded api/internal and api/handlers source files; max %d files/%d MiB; literal Gorilla and statically typed net/http ServeMux routes only; path structure does not prove semantic behavior", domainMap.Scenario, maxDriftFiles, maxDriftBytes>>20), Authority: string(domainMap.Authority), Results: []DocCodeDrift{}, Degradations: []string{}}
	if domainMap.Authority != SourceDomainsDoc || domainMap.AuthorityConfidence != ConfidenceHigh {
		response.Degradations = append(response.Degradations, "docs_authority_unknown: no high-confidence curated DOMAINS.md authority; derived/fallback names cannot be treated as documented claims")
		return response, nil
	}
	docPath := filepath.Join(root, "docs", "concepts", "DOMAINS.md")
	docInfo, err := os.Lstat(docPath)
	if err != nil {
		return nil, err
	}
	if !docInfo.Mode().IsRegular() || docInfo.Size() > 1<<20 {
		return nil, fmt.Errorf("DOMAINS.md missing, non-regular, or exceeds 1 MiB cap")
	}
	docBytes, err := os.ReadFile(docPath)
	if err != nil {
		return nil, err
	}
	docHash := driftDigest(docBytes)
	docClaims, docClaimsParsed := driftDocRows(string(docBytes), domainMap)
	if !docClaimsParsed {
		response.Degradations = append(response.Degradations, "docs_claim_parse_unknown: structured Domain Inventory headers could not be matched; no missing or undocumented claim is inferred")
		return response, nil
	}
	sources, capped, err := scanDriftAPISource(root)
	if err != nil {
		return nil, err
	}
	if capped {
		response.Degradations = append(response.Degradations, "source_scan_capped: missing-path results are UNKNOWN because the bounded Go inventory reached its cap")
	}
	handlerSources, handlerCapped, err := scanDriftHandlerAPI(root, domainMap.Scenario)
	if err != nil {
		return nil, err
	}
	if handlerCapped {
		response.Degradations = append(response.Degradations, "api_handler_scan_capped: route-to-handler linkage is UNKNOWN because bounded handler evidence was incomplete")
	}
	joinCache := &driftJoinCache{root: root, packages: map[string]driftPackageScan{}}
	for _, doc := range docClaims {
		for _, rawPath := range doc.paths {
			pattern := scenarioRelativePattern(domainMap.Scenario, rawPath)
			if !isDriftAPIPattern(pattern) {
				continue
			}
			patternSupported := driftPatternSupported(pattern)
			matches := []driftSource{}
			if patternSupported {
				matches = matchingDriftSources(pattern, sources)
			}
			classification, status := "STRUCTURAL_PATH_MATCH", "PARTIAL"
			if !patternSupported {
				classification, status = "UNSUPPORTED_PATH_PATTERN", "UNKNOWN_PATTERN"
			} else if len(matches) == 0 {
				classification, status = "MISSING_SOURCE_CANDIDATE", "UNKNOWN_SCAN_CAPPED"
				if !capped {
					status = "CONTRADICTION_CANDIDATE"
				}
			}
			if patternSupported && len(matches) > 0 {
				allTests := true
				for _, match := range matches {
					if !match.test {
						allTests = false
						break
					}
				}
				if allTests {
					classification = "TEST_ONLY_SOURCE_CANDIDATE"
				}
			}
			row := DocCodeDrift{ID: "cartographer:doc-code:" + doc.domain + ":" + driftSafeID(pattern), Title: doc.domain + " · " + pattern, Summary: doc.responsibility, Scenario: domainMap.Scenario, Domain: doc.domain, Claim: "DOMAINS.md Source Paths includes `" + pattern + "`", Classification: classification, Status: status, Documentation: DriftEvidence{Path: "docs/concepts/DOMAINS.md", Line: doc.line, SHA256: docHash, Basis: "authored Domain Inventory row and Source Paths cell"}, Code: []DriftEvidence{}, Limits: []string{"Path correspondence is structural evidence only; it does not establish that code implements the documented responsibility.", "The scan includes source files under api/internal and api/handlers only; other documented surfaces are out of scope.", "A missing candidate is UNKNOWN when inventory caps are reached."}}
			row.APIStatus, row.API = driftAPIPathEvidence(pattern, domainMap.Scenario, handlerSources, handlerCapped, patternSupported, joinCache)
			if strings.HasPrefix(row.APIStatus, "UNKNOWN") {
				row.Limits = append(row.Limits, "API route linkage is unknown unless an exact supported literal route → handler → direct internal-package call → target declaration chain is shown; generated Connect, custom routers and dynamic routing are not proved.")
				response.Degradations = append(response.Degradations, "api_linkage_unknown:"+row.ID)
			}
			if !patternSupported {
				row.Limits = append(row.Limits, "Documented glob syntax is outside the path matcher; neither match nor absence is inferred.")
			}
			for _, source := range matches {
				if len(row.Code) >= 3 {
					break
				}
				basis := "Source candidate: " + source.fact
				if source.test {
					basis = "test-only source candidate: " + source.fact
				}
				row.Code = append(row.Code, DriftEvidence{Path: source.path, Line: source.line, SHA256: source.hash, Basis: basis})
			}
			row.Score = driftScore(query, doc.domain+" "+pattern+" "+doc.responsibility+" "+classification)
			if row.Score > 0 {
				response.Results = append(response.Results, row)
			}
		}
	}
	claimed := []string{}
	for _, doc := range docClaims {
		for _, p := range doc.paths {
			p = scenarioRelativePattern(domainMap.Scenario, p)
			if isDriftAPIPattern(p) {
				claimed = append(claimed, p)
			}
		}
	}
	seenFolders := map[string]bool{}
	for _, source := range sources {
		if !strings.HasPrefix(source.path, "api/internal/") || source.test {
			continue
		}
		covered := false
		for _, pattern := range claimed {
			if driftPathMatches(pattern, source.path) {
				covered = true
				break
			}
		}
		if covered {
			continue
		}
		folder := strings.Split(strings.TrimPrefix(source.path, "api/internal/"), "/")[0]
		if seenFolders[folder] {
			continue
		}
		seenFolders[folder] = true
		headerLine := domainInventoryHeaderLine(string(docBytes))
		status, classification := "PARTIAL", "UNDOCUMENTED_SOURCE_CANDIDATE"
		if capped {
			status, classification = "UNKNOWN_SCAN_CAPPED", "UNDOCUMENTED_SOURCE_CANDIDATE"
		}
		row := DocCodeDrift{ID: "cartographer:doc-code:undocumented:" + driftSafeID(folder), Title: "Undocumented API source path · " + folder, Summary: "A source package exists under api/internal but no curated API Source Paths claim matched it.", Scenario: domainMap.Scenario, Claim: "No API Source Paths claim matched this observed package in the curated inventory", Classification: classification, Status: status, Domain: folder, Documentation: DriftEvidence{Path: "docs/concepts/DOMAINS.md", Line: headerLine, SHA256: docHash, Basis: "Domain Inventory table scope; absence is bounded to parsed authored rows"}, Code: []DriftEvidence{{Path: source.path, Line: source.line, SHA256: source.hash, Basis: "Source candidate: " + source.fact}}, Limits: []string{"Undocumented status is a structural candidate, not proof that a domain must be documented; shared/internal utility packages may be intentional.", "Path correspondence does not establish semantic ownership or runtime behavior."}}
		row.APIStatus, row.API = driftAPIPathEvidence("api/internal/"+folder+"/", domainMap.Scenario, handlerSources, handlerCapped, true, joinCache)
		if strings.HasPrefix(row.APIStatus, "UNKNOWN") {
			row.Limits = append(row.Limits, "API route linkage is unknown unless an exact supported literal route → handler → direct internal-package call → target declaration chain is shown.")
			response.Degradations = append(response.Degradations, "api_linkage_unknown:"+row.ID)
		}
		row.Score = driftScore(query, row.Title+" "+row.Summary)
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
		response.Degradations = append(response.Degradations, "no_matching_api_doc_claim: no in-scope API path claim matched the query")
	}
	return response, nil
}

func driftDocRows(content string, m DerivedDomainMap) ([]driftDoc, bool) {
	lines := strings.Split(content, "\n")
	inSection := false
	header := []string{}
	domainCol, pathCol, respCol := -1, -1, -1
	out := []driftDoc{}
	parsed := false
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.EqualFold(trim, "## Domain Inventory") {
			inSection = true
			continue
		}
		if inSection && (strings.HasPrefix(trim, "## ") || strings.HasPrefix(trim, "# ")) {
			break
		}
		if !inSection || !strings.HasPrefix(trim, "|") {
			continue
		}
		cells := splitTableRow(trim)
		if isSeparatorRow(cells) {
			continue
		}
		if len(header) == 0 {
			header = cells
			for x, h := range header {
				switch strings.ToLower(strings.TrimSpace(h)) {
				case "domain":
					domainCol = x
				case "source paths":
					pathCol = x
				case "responsibility":
					respCol = x
				}
			}
			parsed = domainCol >= 0 && pathCol >= 0
			continue
		}
		if domainCol < 0 || pathCol < 0 || len(cells) != len(header) {
			continue
		}
		name := strings.TrimSpace(cell(cells, domainCol))
		if name == "" {
			continue
		}
		found := false
		for _, d := range m.Domains {
			if d.Name == name {
				for _, origin := range d.Provenance {
					if origin == SourceDomainsDoc {
						found = true
						break
					}
				}
			}
		}
		if !found {
			continue
		}
		resp := ""
		if respCol >= 0 {
			resp = cleanTextCell(cell(cells, respCol))
		}
		out = append(out, driftDoc{domain: name, paths: parsePathList(cell(cells, pathCol)), responsibility: resp, line: i + 1})
	}
	return out, parsed
}

func scanDriftAPISource(root string) ([]driftSource, bool, error) {
	files := []driftSource{}
	var total int64
	capped := false
	for _, prefix := range []string{"api/internal", "api/handlers"} {
		base := filepath.Join(root, filepath.FromSlash(prefix))
		err := filepath.WalkDir(base, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if !(path == base && os.IsNotExist(walkErr)) {
					capped = true
				}
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				capped = true
				return nil
			}
			if d.IsDir() {
				if d.Name() == ".git" || d.Name() == "vendor" || d.Name() == "node_modules" {
					return fs.SkipDir
				}
				return nil
			}
			info, err := d.Info()
			if err != nil || !info.Mode().IsRegular() {
				return nil
			}
			if len(files) >= maxDriftFiles || info.Size() > 2<<20 || total+info.Size() > maxDriftBytes {
				capped = true
				return fs.SkipAll
			}
			data, err := os.ReadFile(path)
			if err != nil || int64(len(data)) != info.Size() {
				capped = true
				return nil
			}
			if len(data) == 0 {
				return nil
			}
			total += info.Size()
			fact, line := "source file", 1
			ext := strings.ToLower(filepath.Ext(d.Name()))
			var parsed *ast.File
			var fset *token.FileSet
			var parseErr error
			if ext == ".go" {
				fset = token.NewFileSet()
				parsed, parseErr = parser.ParseFile(fset, path, data, parser.AllErrors)
				if parseErr != nil {
					capped = true
					return nil
				}
				fact, line = firstDriftDeclaration(parsed, fset)
				if fact == "" {
					return nil
				}
			}
			if ext != ".go" && strings.IndexByte(string(data), 0) >= 0 {
				capped = true
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}
			source := driftSource{path: filepath.ToSlash(rel), fact: fact, line: line, hash: driftDigest(data), test: strings.HasSuffix(d.Name(), "_test.go")}
			files = append(files, source)
			return nil
		})
		if err != nil && err != fs.SkipAll {
			return nil, capped, err
		}
		if capped {
			break
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	return files, capped, nil
}

func driftAPIMetadata(file *ast.File, fset *token.FileSet, rootPrefix, scenario string) ([]driftFunction, []driftRoute) {
	imports := map[string]string{}
	for _, item := range file.Imports {
		path, err := strconv.Unquote(item.Path.Value)
		if err != nil {
			continue
		}
		alias := filepath.Base(path)
		if item.Name != nil {
			alias = item.Name.Name
		}
		imports[alias] = path
	}
	functions := []driftFunction{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		row := driftFunction{name: fn.Name.Name, line: fset.Position(fn.Pos()).Line}
		if strings.HasPrefix(rootPrefix, "api/handlers") {
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := selector.X.(*ast.Ident)
				if !ok {
					return true
				}
				importPath := imports[pkg.Name]
				if importPath == "" || !strings.HasPrefix(importPath, scenario+"/internal/") {
					return true
				}
				row.calls = append(row.calls, driftCall{importPath: importPath, symbol: selector.Sel.Name, line: fset.Position(call.Pos()).Line})
				return true
			})
		}
		functions = append(functions, row)
	}
	routes := []driftRoute{}
	if strings.HasPrefix(rootPrefix, "api/handlers") {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			serveMuxes := driftServeMuxNames(fn, imports)
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				registration, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || (registration.Sel.Name != "Handle" && registration.Sel.Name != "HandleFunc") || len(call.Args) < 2 {
					return true
				}
				if driftServeMuxRegistration(registration, imports, serveMuxes) {
					path := driftStringValue(call.Args[0])
					method := "ANY"
					literal := path != ""
					if !literal {
						path = driftExpression(call.Args[0])
					}
					if literal {
						method, path = driftServeMuxPattern(path)
					} else {
						method = "ANY"
					}
					handler := driftRouteHandlerName(registration.Sel.Name, call.Args[1])
					if path != "" && handler != "" {
						routes = append(routes, driftRoute{path: path, method: method, handler: handler, line: fset.Position(call.Pos()).Line, literal: literal})
					}
					return true
				}
				return true
			})
		}
		// Preserve the existing Gorilla-style `.Methods("POST")` chain. It
		// is not a net/http ServeMux registration and has its own syntax.
		ast.Inspect(file, func(node ast.Node) bool {
			outer, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			methods, ok := outer.Fun.(*ast.SelectorExpr)
			if !ok || methods.Sel.Name != "Methods" || len(outer.Args) == 0 {
				return true
			}
			inner, ok := methods.X.(*ast.CallExpr)
			if !ok {
				return true
			}
			handle, ok := inner.Fun.(*ast.SelectorExpr)
			if !ok || (handle.Sel.Name != "Handle" && handle.Sel.Name != "HandleFunc") || len(inner.Args) < 2 {
				return true
			}
			path := driftStringValue(inner.Args[0])
			literal := path != ""
			if !literal {
				path = driftExpression(inner.Args[0])
			}
			method := driftStringValue(outer.Args[0])
			handler := driftRouteHandlerName(handle.Sel.Name, inner.Args[1])
			if path != "" && method != "" && handler != "" {
				routes = append(routes, driftRoute{path: path, method: strings.ToUpper(method), handler: handler, line: fset.Position(inner.Pos()).Line, literal: literal})
			}
			return true
		})
	}
	return functions, routes
}

func scanDriftHandlerAPI(root, scenario string) ([]driftSource, bool, error) {
	base := filepath.Join(root, "api", "handlers")
	files := []driftSource{}
	var total int64
	scanned := 0
	capped := false
	err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if !(path == base && os.IsNotExist(walkErr)) {
				capped = true
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			capped = true
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "vendor" || entry.Name() == "node_modules" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return nil
		}
		if scanned >= maxDriftHandlerFiles || info.Size() > 2<<20 || total+info.Size() > maxDriftHandlerBytes {
			capped = true
			return fs.SkipAll
		}
		scanned++
		total += info.Size()
		data, err := os.ReadFile(path)
		if err != nil || int64(len(data)) != info.Size() {
			capped = true
			return nil
		}
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, path, data, parser.AllErrors)
		if err != nil {
			capped = true
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		hash := driftDigest(data)
		functions, routes := driftAPIMetadata(parsed, fset, "api/handlers", scenario)
		files = append(files, driftSource{path: filepath.ToSlash(rel), hash: hash, functions: functions, routes: routes})
		return nil
	})
	if err != nil && err != fs.SkipAll {
		return nil, capped, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	return files, capped, nil
}

type driftPackageScan struct {
	sources []driftSource
	capped  bool
}
type driftJoinCache struct {
	root     string
	packages map[string]driftPackageScan
	files    int
	bytes    int64
}

func (c *driftJoinCache) scan(dir string) ([]driftSource, bool) {
	if cached, ok := c.packages[dir]; ok {
		return cached.sources, cached.capped
	}
	if !strings.HasPrefix(dir, "api/internal/") || strings.Contains(dir, "..") {
		c.packages[dir] = driftPackageScan{capped: true}
		return nil, true
	}
	base := filepath.Join(c.root, filepath.FromSlash(dir))
	info, err := os.Lstat(base)
	if os.IsNotExist(err) {
		c.packages[dir] = driftPackageScan{}
		return nil, false
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		c.packages[dir] = driftPackageScan{capped: true}
		return nil, true
	}
	files := []driftSource{}
	capped := false
	err = filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			capped = true
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			capped = true
			return nil
		}
		if entry.IsDir() {
			if path != base || entry.Name() == "vendor" || entry.Name() == ".git" || entry.Name() == "node_modules" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		fileInfo, err := entry.Info()
		if err != nil || !fileInfo.Mode().IsRegular() {
			return nil
		}
		if c.files >= maxDriftJoinFiles || fileInfo.Size() > 2<<20 || c.bytes+fileInfo.Size() > maxDriftJoinBytes {
			capped = true
			return fs.SkipAll
		}
		c.files++
		c.bytes += fileInfo.Size()
		data, err := os.ReadFile(path)
		if err != nil || int64(len(data)) != fileInfo.Size() {
			capped = true
			return nil
		}
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, path, data, parser.AllErrors)
		if err != nil {
			capped = true
			return nil
		}
		rel, err := filepath.Rel(c.root, path)
		if err != nil {
			return nil
		}
		functions, _ := driftAPIMetadata(parsed, fset, dir, "")
		files = append(files, driftSource{path: filepath.ToSlash(rel), hash: driftDigest(data), functions: functions})
		return nil
	})
	if err != nil && err != fs.SkipAll {
		capped = true
	}
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	result := driftPackageScan{sources: files, capped: capped}
	c.packages[dir] = result
	return result.sources, result.capped
}

func driftAPIPathEvidence(pattern, scenario string, handlerSources []driftSource, handlerCapped, patternSupported bool, cache *driftJoinCache) (string, []DriftEvidence) {
	if !patternSupported {
		return "UNKNOWN_UNSUPPORTED_PATH", nil
	}
	if handlerCapped {
		return "UNKNOWN_SOURCE_SCAN_CAPPED", nil
	}
	type candidate struct {
		evidence []DriftEvidence
		state    string
	}
	candidates := []candidate{}
	for _, handlerFile := range handlerSources {
		if !strings.HasPrefix(handlerFile.path, "api/handlers/") || handlerFile.test {
			continue
		}
		for _, handler := range handlerFile.functions {
			for _, call := range handler.calls {
				dir := driftInternalImportDir(scenario, call.importPath)
				if dir == "" {
					continue
				}
				targetSources, targetCapped := cache.scan(dir)
				if targetCapped {
					return "UNKNOWN_SOURCE_SCAN_CAPPED", nil
				}
				targets := []driftSource{}
				for _, targetFile := range targetSources {
					if !driftPathMatches(pattern, targetFile.path) {
						continue
					}
					for _, fn := range targetFile.functions {
						if fn.name == call.symbol {
							targets = append(targets, targetFile)
						}
					}
				}
				if len(targets) == 0 {
					continue
				}
				routes := []struct {
					source driftSource
					route  driftRoute
				}{}
				for _, routeFile := range handlerSources {
					if !strings.HasPrefix(routeFile.path, "api/handlers/") || routeFile.test {
						continue
					}
					for _, route := range routeFile.routes {
						if route.handler == handler.name {
							routes = append(routes, struct {
								source driftSource
								route  driftRoute
							}{routeFile, route})
						}
					}
				}
				base := []DriftEvidence{
					{Path: handlerFile.path, Line: handler.line, SHA256: handlerFile.hash, Basis: "local API handler function definition: " + handler.name},
					{Path: handlerFile.path, Line: call.line, SHA256: handlerFile.hash, Basis: "direct call through imported package " + call.importPath + "." + call.symbol},
				}
				for _, target := range targets {
					line := 1
					for _, fn := range target.functions {
						if fn.name == call.symbol {
							line = fn.line
							break
						}
					}
					base = append(base, DriftEvidence{Path: target.path, Line: line, SHA256: target.hash, Basis: "directly named internal function declaration inside the authored Source Paths pattern"})
				}
				if len(targets) != 1 {
					candidates = append(candidates, candidate{evidence: base, state: "UNKNOWN_AMBIGUOUS_TARGET"})
					continue
				}
				if len(routes) != 1 {
					state := "UNKNOWN_NO_SUPPORTED_LITERAL_ROUTE"
					if len(routes) > 1 {
						state = "UNKNOWN_AMBIGUOUS_ROUTE"
					}
					for _, route := range routes {
						basis := "unsupported dynamic route expression: " + route.route.path
						if route.route.literal {
							basis = route.route.method + " " + route.route.path + " mux registration; method handler matched literally"
						}
						base = append(base, DriftEvidence{Path: route.source.path, Line: route.route.line, SHA256: route.source.hash, Basis: basis})
					}
					candidates = append(candidates, candidate{evidence: base, state: state})
					continue
				}
				route := routes[0]
				if !route.route.literal {
					base = append(base, DriftEvidence{Path: route.source.path, Line: route.route.line, SHA256: route.source.hash, Basis: "unsupported dynamic route expression: " + route.route.path})
					candidates = append(candidates, candidate{evidence: base, state: "UNKNOWN_DYNAMIC_ROUTE"})
					continue
				}
				base = append([]DriftEvidence{{Path: route.source.path, Line: route.route.line, SHA256: route.source.hash, Basis: route.route.method + " " + route.route.path + " literal mux route to " + route.route.handler}}, base...)
				candidates = append(candidates, candidate{evidence: base, state: "ROUTE_HANDLER_DIRECT_CALL"})
			}
		}
	}
	if len(candidates) == 0 {
		return "UNKNOWN_NO_SUPPORTED_DIRECT_ROUTE_CHAIN", nil
	}
	if len(candidates) == 1 {
		return candidates[0].state, candidates[0].evidence
	}
	all := []DriftEvidence{}
	for _, item := range candidates {
		all = append(all, item.evidence...)
	}
	if len(all) > 12 {
		all = all[:12]
	}
	return "UNKNOWN_AMBIGUOUS_ROUTE_OR_CALL", all
}

func driftInternalImportDir(scenario, importPath string) string {
	needle := scenario + "/internal/"
	if !strings.HasPrefix(importPath, needle) {
		return ""
	}
	suffix := importPath[len(needle):]
	for _, part := range strings.Split(suffix, "/") {
		if part == "" || part == "." || part == ".." {
			return ""
		}
	}
	return "api/internal/" + suffix
}

func driftStringValue(expr ast.Expr) string {
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

func driftFirstCallName(expr ast.Expr) string {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return ""
	}
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		return fn.Sel.Name
	}
	return ""
}

func driftRouteHandlerName(registration string, expr ast.Expr) string {
	if registration == "Handle" {
		return driftFirstCallName(expr)
	}
	if registration != "HandleFunc" {
		return ""
	}
	switch handler := expr.(type) {
	case *ast.Ident:
		return handler.Name
	case *ast.SelectorExpr:
		return handler.Sel.Name
	default:
		return ""
	}
}

// driftServeMuxNames returns parameters and receivers statically typed as
// net/http ServeMux. Local assignment flow is intentionally not inferred, so
// shadowed or custom routers cannot inherit a name accidentally.
func driftServeMuxNames(fn *ast.FuncDecl, imports map[string]string) map[string]struct{} {
	names := map[string]struct{}{}
	addFields := func(fields *ast.FieldList) {
		if fields == nil {
			return
		}
		for _, field := range fields.List {
			if !driftIsHTTPServeMuxType(field.Type, imports) {
				continue
			}
			for _, name := range field.Names {
				names[name.Name] = struct{}{}
			}
		}
	}
	addFields(fn.Recv)
	addFields(fn.Type.Params)
	return names
}

func driftIsHTTPServeMuxType(expr ast.Expr, imports map[string]string) bool {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "ServeMux" {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && imports[pkg.Name] == "net/http"
}

func driftServeMuxRegistration(registration *ast.SelectorExpr, imports map[string]string, names map[string]struct{}) bool {
	switch receiver := registration.X.(type) {
	case *ast.Ident:
		if imports[receiver.Name] == "net/http" {
			return true // package-level http.Handle / HandleFunc
		}
		_, ok := names[receiver.Name]
		return ok
	case *ast.SelectorExpr:
		pkg, ok := receiver.X.(*ast.Ident)
		return ok && imports[pkg.Name] == "net/http" && receiver.Sel.Name == "DefaultServeMux"
	}
	return false
}

func driftServeMuxPattern(pattern string) (method, path string) {
	parts := strings.Fields(pattern)
	if len(parts) == 2 && driftHTTPMethod(parts[0]) {
		return strings.ToUpper(parts[0]), parts[1]
	}
	return "ANY", pattern
}

func driftHTTPMethod(value string) bool {
	switch strings.ToUpper(value) {
	case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "CONNECT", "OPTIONS", "TRACE":
		return true
	default:
		return false
	}
}

func driftExpression(expr ast.Expr) string {
	var output bytes.Buffer
	if err := printer.Fprint(&output, token.NewFileSet(), expr); err != nil {
		return "unsupported"
	}
	return output.String()
}

func firstDriftDeclaration(file *ast.File, fset *token.FileSet) (string, int) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			return d.Name.Name, fset.Position(d.Pos()).Line
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					return s.Name.Name, fset.Position(s.Pos()).Line
				case *ast.ValueSpec:
					if len(s.Names) > 0 {
						return s.Names[0].Name, fset.Position(s.Pos()).Line
					}
				}
			}
		}
	}
	return "", 0
}
func scenarioRelativePattern(scenario, path string) string {
	path = NormalizePath(path)
	return strings.TrimPrefix(path, "scenarios/"+scenario+"/")
}
func isDriftAPIPattern(path string) bool {
	return strings.HasPrefix(path, "api/internal/") || strings.HasPrefix(path, "api/handlers/")
}
func matchingDriftSources(pattern string, sources []driftSource) []driftSource {
	out := []driftSource{}
	for _, source := range sources {
		if driftPathMatches(pattern, source.path) {
			out = append(out, source)
		}
	}
	return out
}
func driftPathMatches(pattern, path string) bool {
	return PathMatches(path, pattern)
}
func driftPatternSupported(pattern string) bool {
	if strings.ContainsAny(pattern, "?[]{}") {
		return false
	}
	if strings.Contains(pattern, "*") && pattern != "**" && !strings.HasSuffix(pattern, "/**") && !strings.HasSuffix(pattern, "/*") {
		return false
	}
	return strings.TrimSpace(pattern) != ""
}
func domainInventoryHeaderLine(content string) int {
	for i, line := range strings.Split(content, "\n") {
		if strings.EqualFold(strings.TrimSpace(line), "## Domain Inventory") {
			return i + 1
		}
	}
	return 1
}
func driftScore(query, text string) float64 {
	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		return 0
	}
	hit := 0
	text = strings.ToLower(text)
	for _, term := range terms {
		if len(term) > 1 && strings.Contains(text, term) {
			hit++
		}
	}
	return float64(hit) / float64(len(terms))
}
func driftSafeID(text string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(text) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
func driftDigest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
