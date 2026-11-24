package facts

import (
	"context"
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

const (
	maxProtoHandlerFiles       = 160
	maxProtoHandlerSourceBytes = 16 << 20
	maxProtoGeneratedFiles     = 80
	maxProtoGeneratedBytes     = 8 << 20
	maxProtoSourceFileBytes    = 2 << 20
)

type ProtoHandlerConformanceResponse struct {
	Scenario     string                `json:"scenario"`
	Scope        string                `json:"scope"`
	Services     []ProtoHandlerService `json:"services"`
	Degradations []string              `json:"degradations"`
}

type ProtoHandlerService struct {
	Service       string               `json:"service"`
	GeneratedPath string               `json:"generated_path"`
	HandlerType   string               `json:"handler_type,omitempty"`
	Methods       []ProtoHandlerMethod `json:"methods"`
	Status        string               `json:"status"`
}

type ProtoHandlerMethod struct {
	Name           string                  `json:"name"`
	Status         string                  `json:"status"`
	Generated      ProtoConformanceSource  `json:"generated"`
	Implementation *ProtoConformanceSource `json:"implementation,omitempty"`
	ExpectedTypes  []string                `json:"expected_types"`
	ActualTypes    []string                `json:"actual_types,omitempty"`
}

type ProtoConformanceSource struct {
	Path   string `json:"path"`
	Line   int    `json:"line"`
	SHA256 string `json:"sha256"`
	Basis  string `json:"basis"`
}

type protoHandlerSourceFile struct {
	absolute string
	relative string
	data     []byte
	fset     *token.FileSet
	file     *ast.File
}

type protoHandlerContract struct {
	service       string
	generatedPath string
	handlerDir    string
	handlerType   string
	ambiguous     bool
	incomplete    bool
	methods       []protoHandlerMethodContract
}

type protoHandlerMethodContract struct {
	name      string
	types     []string
	source    ProtoConformanceSource
	parseOkay bool
}

// ProtoHandlerConformance checks only the Go unary method signatures on one
// scenario's concrete Connect handlers against the generated Connect service
// interfaces they import. It does not inspect endpoint registration, route
// mounting, runtime behavior, or proto-source/generated-code freshness.
func (s *Service) ProtoHandlerConformance(_ context.Context, scenario string) (*ProtoHandlerConformanceResponse, error) {
	scenario = strings.TrimSpace(scenario)
	if !invariantScenarioPattern.MatchString(scenario) {
		return nil, fmt.Errorf("one scenario slug is required")
	}
	target, err := resolveTarget(&factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: scenario})
	if err != nil {
		return nil, err
	}
	scenarioRoot, err := filepath.EvalSymlinks(target.GetRootPath())
	if err != nil {
		return nil, err
	}
	return scanProtoHandlerConformance(scenarioRoot, scenario)
}

func scanProtoHandlerConformance(scenarioRoot, scenario string) (*ProtoHandlerConformanceResponse, error) {
	files, parseErrors, err := loadProtoHandlerSources(scenarioRoot)
	if err != nil {
		return nil, err
	}
	repoRoot := filepath.Clean(filepath.Join(scenarioRoot, "..", ".."))
	contracts, importDegradations, err := discoverProtoHandlerContracts(repoRoot, files)
	if err != nil {
		return nil, err
	}
	response := &ProtoHandlerConformanceResponse{
		Scenario: scenario,
		Scope:    "scenario:" + scenario + "/api/handlers; exact source contract compares concrete unary Go method parameter/result types with imported generated Connect ServiceHandler interfaces; bounded to 160 Go files/16 MiB and generated packages to 80 files/8 MiB; does not prove .proto source parity or runtime behavior",
		Services: []ProtoHandlerService{}, Degradations: []string{},
	}
	response.Degradations = append(response.Degradations, importDegradations...)
	if len(parseErrors) > 0 {
		response.Degradations = append(response.Degradations, "source_parse_incomplete")
	}
	for _, contract := range contracts {
		service := compareProtoHandlerContract(contract, files, len(parseErrors) == 0)
		response.Services = append(response.Services, service)
	}
	if len(contracts) == 0 {
		response.Degradations = append(response.Degradations, "no_generated_connect_service_interface_found")
	}
	sort.Strings(response.Degradations)
	return response, nil
}

func loadProtoHandlerSources(scenarioRoot string) ([]*protoHandlerSourceFile, []string, error) {
	apiRoot := filepath.Join(scenarioRoot, "api", "handlers")
	rootInfo, err := os.Lstat(apiRoot)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return nil, nil, fmt.Errorf("scenario API handlers directory is unavailable")
	}
	files := make([]*protoHandlerSourceFile, 0, 32)
	parseErrors := []string{}
	var totalBytes int64
	err = filepath.WalkDir(apiRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") || strings.HasSuffix(entry.Name(), ".pb.go") || strings.HasSuffix(entry.Name(), ".connect.go") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > maxProtoSourceFileBytes || len(files) >= maxProtoHandlerFiles || totalBytes+info.Size() > maxProtoHandlerSourceBytes {
			return fmt.Errorf("scenario API handler source exceeds bounded scan limits")
		}
		data, err := os.ReadFile(path)
		if err != nil || int64(len(data)) != info.Size() {
			return fmt.Errorf("scenario API handler source changed or could not be read")
		}
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, data, parser.AllErrors)
		relative, _ := filepath.Rel(scenarioRoot, path)
		source := &protoHandlerSourceFile{absolute: path, relative: filepath.ToSlash(relative), data: data, fset: fset, file: file}
		files = append(files, source)
		totalBytes += info.Size()
		if parseErr != nil {
			parseErrors = append(parseErrors, source.relative)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(parseErrors)
	return files, parseErrors, nil
}

func discoverProtoHandlerContracts(repoRoot string, files []*protoHandlerSourceFile) ([]protoHandlerContract, []string, error) {
	byDir := map[string][]*protoHandlerSourceFile{}
	for _, file := range files {
		byDir[filepath.Dir(file.absolute)] = append(byDir[filepath.Dir(file.absolute)], file)
	}
	dirs := make([]string, 0, len(byDir))
	for dir := range byDir {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	contractsByKey := map[string]protoHandlerContract{}
	degradations := []string{}
	for _, dir := range dirs {
		dirFiles := byDir[dir]
		for _, source := range dirFiles {
			if source.file == nil {
				continue
			}
			imports := sourceImportPaths(source.file)
			for _, decl := range source.file.Decls {
				moduleFn, ok := decl.(*ast.FuncDecl)
				if !ok || moduleFn.Name.Name != "Module" || moduleFn.Body == nil {
					continue
				}
				ast.Inspect(moduleFn.Body, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok {
						return true
					}
					selector, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || len(call.Args) != 1 {
						return true
					}
					alias, ok := selector.X.(*ast.Ident)
					if !ok || !strings.HasPrefix(selector.Sel.Name, "New") || !strings.HasSuffix(selector.Sel.Name, "Handler") {
						return true
					}
					importPath := imports[alias.Name]
					if !strings.HasSuffix(importPath, "_v1connect") {
						return true
					}
					serviceName := strings.TrimSuffix(strings.TrimPrefix(selector.Sel.Name, "New"), "Handler")
					if serviceName == "" || strings.TrimSpace(serviceName) != serviceName {
						return true
					}
					generatedDir, generatedRel, safe := generatedConnectDir(repoRoot, importPath)
					if !safe {
						degradations = append(degradations, "generated_package_outside_proto_tree")
						return true
					}
					generated, incomplete, err := loadGeneratedServiceContract(generatedDir, generatedRel, serviceName)
					if err != nil {
						degradations = append(degradations, "generated_contract_unavailable:"+serviceName)
						return true
					}
					if incomplete {
						degradations = append(degradations, "generated_contract_incomplete:"+serviceName)
						generated.incomplete = true
					}
					key := dir + "\x00" + importPath + "\x00" + serviceName
					if existing, exists := contractsByKey[key]; exists {
						degradations = append(degradations, "multiple_generated_service_mounts:"+serviceName)
						existing.ambiguous = true
						contractsByKey[key] = existing
						return true
					}
					generated.handlerType = protoHandlerTypeForExpr(call.Args[0], dirFiles, moduleFn, map[string]bool{})
					generated.handlerDir = dir
					contractsByKey[key] = generated
					return true
				})
			}
		}
	}
	contracts := make([]protoHandlerContract, 0, len(contractsByKey))
	for _, contract := range contractsByKey {
		contracts = append(contracts, contract)
	}
	sort.Slice(contracts, func(i, j int) bool {
		if contracts[i].generatedPath != contracts[j].generatedPath {
			return contracts[i].generatedPath < contracts[j].generatedPath
		}
		return contracts[i].service < contracts[j].service
	})
	sort.Strings(degradations)
	return contracts, degradations, nil
}

func generatedConnectDir(repoRoot, importPath string) (string, string, bool) {
	const marker = "/packages/proto/gen/go/"
	index := strings.Index(importPath, marker)
	if index < 0 {
		return "", "", false
	}
	suffix := strings.TrimPrefix(importPath[index+len(marker):], "/")
	if suffix == "" {
		return "", "", false
	}
	for _, part := range strings.Split(suffix, "/") {
		if part == "" || part == "." || part == ".." {
			return "", "", false
		}
	}
	dir := filepath.Join(repoRoot, "packages", "proto", "gen", "go", filepath.FromSlash(suffix))
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", "", false
	}
	relative, err := filepath.Rel(repoRoot, dir)
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", false
	}
	return dir, filepath.ToSlash(relative), true
}

func loadGeneratedServiceContract(dir, generatedRel, service string) (protoHandlerContract, bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) > maxProtoGeneratedFiles {
		return protoHandlerContract{}, false, fmt.Errorf("generated Connect directory is unavailable or over cap")
	}
	contract := protoHandlerContract{service: service, generatedPath: generatedRel}
	foundInterfaces := 0
	incomplete := false
	var totalBytes int64
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(entry.Name(), ".connect.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxProtoSourceFileBytes || totalBytes+info.Size() > maxProtoGeneratedBytes {
			incomplete = true
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil || int64(len(data)) != info.Size() {
			incomplete = true
			continue
		}
		generatedFileRel := filepath.ToSlash(filepath.Join(generatedRel, entry.Name()))
		totalBytes += info.Size()
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, data, parser.AllErrors)
		if err != nil {
			incomplete = true
			continue
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != service+"Handler" {
					continue
				}
				iface, ok := typeSpec.Type.(*ast.InterfaceType)
				if !ok || iface.Methods == nil {
					incomplete = true
					continue
				}
				foundInterfaces++
				imports := sourceImportPaths(file)
				for _, methodField := range iface.Methods.List {
					if len(methodField.Names) != 1 {
						incomplete = true
						continue
					}
					methodType, ok := methodField.Type.(*ast.FuncType)
					if !ok {
						incomplete = true
						continue
					}
					types, supported := canonicalFuncTypes(methodType, imports)
					if !supported {
						incomplete = true
					}
					contract.methods = append(contract.methods, protoHandlerMethodContract{
						name: methodField.Names[0].Name, types: types, parseOkay: supported,
						source: ProtoConformanceSource{Path: generatedFileRel, Line: fset.Position(methodField.Pos()).Line, SHA256: hashBytes(data), Basis: "generated Connect ServiceHandler method signature"},
					})
				}
			}
		}
	}
	if foundInterfaces != 1 {
		return protoHandlerContract{}, incomplete, fmt.Errorf("generated service interface is missing or ambiguous")
	}
	sort.Slice(contract.methods, func(i, j int) bool { return contract.methods[i].name < contract.methods[j].name })
	return contract, incomplete, nil
}

func compareProtoHandlerContract(contract protoHandlerContract, files []*protoHandlerSourceFile, sourceComplete bool) ProtoHandlerService {
	out := ProtoHandlerService{Service: contract.service, GeneratedPath: contract.generatedPath, HandlerType: contract.handlerType, Methods: []ProtoHandlerMethod{}, Status: "MATCH"}
	concreteTypeKnown := hasLocalHandlerStruct(contract.handlerType, contract.handlerDir, files)
	if contract.ambiguous {
		out.Status = "UNKNOWN_AMBIGUOUS_MOUNT"
	}
	if len(contract.methods) == 0 {
		out.Status = "UNKNOWN_EMPTY_INTERFACE"
	}
	methods := map[string][]protoHandlerImplementation{}
	for _, source := range files {
		if source.file == nil || filepath.Dir(source.absolute) != contract.handlerDir {
			continue
		}
		imports := sourceImportPaths(source.file)
		for _, decl := range source.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Type == nil {
				continue
			}
			receiver := protoHandlerReceiverTypeName(fn.Recv.List[0].Type)
			if receiver == "" || !handlerReceiverMatches(contract.handlerType, receiver) {
				continue
			}
			types, supported := canonicalFuncTypes(fn.Type, imports)
			methods[fn.Name.Name] = append(methods[fn.Name.Name], protoHandlerImplementation{
				types: types, supported: supported,
				source: ProtoConformanceSource{Path: source.relative, Line: source.fset.Position(fn.Pos()).Line, SHA256: hashBytes(source.data), Basis: "concrete handler method signature"},
			})
		}
	}
	for _, expected := range contract.methods {
		method := ProtoHandlerMethod{Name: expected.name, Generated: expected.source, ExpectedTypes: append([]string(nil), expected.types...)}
		candidates := methods[expected.name]
		switch {
		case contract.ambiguous:
			method.Status = "UNKNOWN_AMBIGUOUS_MOUNT"
		case contract.incomplete || !expected.parseOkay || contract.handlerType == "" || !concreteTypeKnown || !sourceComplete:
			method.Status = "UNKNOWN_SOURCE_INCOMPLETE"
		case len(candidates) == 0:
			if handlerEmbedsUnimplemented(contract.handlerType, contract.handlerDir, contract.service, files) {
				method.Status = "UNKNOWN_INHERITED_UNIMPLEMENTED"
			} else {
				method.Status = "MISSING_EXPLICIT_METHOD"
			}
		case len(candidates) > 1:
			method.Status = "AMBIGUOUS_IMPLEMENTATION"
		default:
			implementation := candidates[0].source
			method.Implementation = &implementation
			method.ActualTypes = append([]string(nil), candidates[0].types...)
			if !candidates[0].supported {
				method.Status = "UNKNOWN_UNSUPPORTED_SIGNATURE"
			} else if sameStrings(expected.types, candidates[0].types) {
				method.Status = "MATCH"
			} else {
				method.Status = "MISMATCH"
			}
		}
		if method.Status != "MATCH" {
			out.Status = "PARTIAL"
		}
		out.Methods = append(out.Methods, method)
	}
	return out
}

func handlerEmbedsUnimplemented(handlerType, handlerDir, service string, files []*protoHandlerSourceFile) bool {
	handlerType = strings.TrimPrefix(handlerType, "*")
	want := "Unimplemented" + service + "Handler"
	for _, source := range files {
		if source.file == nil || filepath.Dir(source.absolute) != handlerDir {
			continue
		}
		for _, decl := range source.file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != handlerType {
					continue
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok || structType.Fields == nil {
					continue
				}
				for _, field := range structType.Fields.List {
					if len(field.Names) != 0 {
						continue
					}
					embedded, supported := canonicalType(field.Type, sourceImportPaths(source.file))
					if !supported {
						continue
					}
					if embedded == want || strings.HasSuffix(embedded, "."+want) {
						return true
					}
				}
			}
		}
	}
	return false
}

func hasLocalHandlerStruct(handlerType, handlerDir string, files []*protoHandlerSourceFile) bool {
	handlerType = strings.TrimPrefix(handlerType, "*")
	if handlerType == "" {
		return false
	}
	for _, source := range files {
		if source.file == nil || filepath.Dir(source.absolute) != handlerDir {
			continue
		}
		for _, decl := range source.file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != handlerType {
					continue
				}
				if _, ok := typeSpec.Type.(*ast.StructType); ok {
					return true
				}
			}
		}
	}
	return false
}

func handlerReceiverMatches(handlerType, receiverType string) bool {
	if handlerType == "" {
		return false
	}
	if strings.HasPrefix(handlerType, "*") {
		base := strings.TrimPrefix(handlerType, "*")
		return receiverType == base || receiverType == "*"+base
	}
	return receiverType == handlerType
}

type protoHandlerImplementation struct {
	types     []string
	supported bool
	source    ProtoConformanceSource
}

func sourceImportPaths(file *ast.File) map[string]string {
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
	return imports
}

func protoHandlerTypeForExpr(expr ast.Expr, files []*protoHandlerSourceFile, module *ast.FuncDecl, seen map[string]bool) string {
	switch value := expr.(type) {
	case *ast.ParenExpr:
		return protoHandlerTypeForExpr(value.X, files, module, seen)
	case *ast.UnaryExpr:
		if value.Op == token.AND {
			return "*" + protoHandlerTypeName(value.X)
		}
	case *ast.CompositeLit:
		return protoHandlerTypeName(value.Type)
	case *ast.CallExpr:
		if ident, ok := value.Fun.(*ast.Ident); ok {
			if seen[ident.Name] {
				return ""
			}
			seen[ident.Name] = true
			for _, source := range files {
				if source.file == nil {
					continue
				}
				for _, decl := range source.file.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Recv != nil || fn.Name.Name != ident.Name || fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
						continue
					}
					return protoHandlerConcreteTypeName(fn.Type.Results.List[0].Type)
				}
			}
		}
	case *ast.Ident:
		var found string
		ambiguous := false
		if module != nil && module.Body != nil {
			ast.Inspect(module.Body, func(node ast.Node) bool {
				assign, ok := node.(*ast.AssignStmt)
				if !ok {
					return true
				}
				for i, lhs := range assign.Lhs {
					id, ok := lhs.(*ast.Ident)
					if !ok || id.Name != value.Name || i >= len(assign.Rhs) {
						continue
					}
					candidate := protoHandlerTypeForExpr(assign.Rhs[i], files, module, seen)
					if candidate != "" {
						if found != "" && found != candidate {
							ambiguous = true
							return false
						}
						found = candidate
					}
				}
				return true
			})
		}
		if ambiguous {
			return ""
		}
		return found
	}
	return ""
}

func protoHandlerTypeName(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	if selector, ok := expr.(*ast.SelectorExpr); ok {
		return selector.Sel.Name
	}
	if literal, ok := expr.(*ast.CompositeLit); ok {
		return protoHandlerTypeName(literal.Type)
	}
	return ""
}

func protoHandlerConcreteTypeName(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		return "*" + protoHandlerTypeName(star.X)
	}
	return protoHandlerTypeName(expr)
}

func protoHandlerReceiverTypeName(expr ast.Expr) string { return protoHandlerConcreteTypeName(expr) }

func canonicalFuncTypes(fn *ast.FuncType, imports map[string]string) ([]string, bool) {
	params, paramsOK := canonicalFieldTypes(fn.Params, imports)
	results, resultsOK := canonicalFieldTypes(fn.Results, imports)
	return append(params, results...), paramsOK && resultsOK
}

func canonicalFieldTypes(fields *ast.FieldList, imports map[string]string) ([]string, bool) {
	if fields == nil {
		return []string{}, true
	}
	out := []string{}
	supported := true
	for _, field := range fields.List {
		value, ok := canonicalType(field.Type, imports)
		if !ok {
			supported = false
		}
		n := len(field.Names)
		if n == 0 {
			n = 1
		}
		for i := 0; i < n; i++ {
			out = append(out, value)
		}
	}
	return out, supported
}

func canonicalType(expr ast.Expr, imports map[string]string) (string, bool) {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name, true
	case *ast.SelectorExpr:
		alias, ok := value.X.(*ast.Ident)
		if !ok {
			return "", false
		}
		importPath := imports[alias.Name]
		if importPath == "" {
			return "", false
		}
		return importPath + "." + value.Sel.Name, true
	case *ast.StarExpr:
		inner, ok := canonicalType(value.X, imports)
		return "*" + inner, ok
	case *ast.IndexExpr:
		left, leftOK := canonicalType(value.X, imports)
		right, rightOK := canonicalType(value.Index, imports)
		return left + "[" + right + "]", leftOK && rightOK
	case *ast.IndexListExpr:
		left, ok := canonicalType(value.X, imports)
		parts := make([]string, 0, len(value.Indices))
		for _, index := range value.Indices {
			part, partOK := canonicalType(index, imports)
			parts = append(parts, part)
			ok = ok && partOK
		}
		return left + "[" + strings.Join(parts, ",") + "]", ok
	case *ast.Ellipsis:
		inner, ok := canonicalType(value.Elt, imports)
		return "..." + inner, false && ok
	default:
		return "", false
	}
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
