package facts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

const (
	maxAmbientFiles = 160
	maxAmbientBytes = 16 << 20
	maxAmbientHits  = 12
)

type AmbientSeam struct {
	ID                    string                     `json:"id"`
	Title                 string                     `json:"title"`
	Summary               string                     `json:"summary"`
	Scenario              string                     `json:"scenario"`
	Kind                  string                     `json:"kind"`
	Expression            string                     `json:"expression"`
	ContextSymbol         string                     `json:"context_symbol"`
	SourceLayer           string                     `json:"source_layer"`
	BuildConstraint       string                     `json:"build_constraint,omitempty"`
	BuildConstraintStatus string                     `json:"build_constraint_status"`
	BuildConstraintLine   int                        `json:"build_constraint_line,omitempty"`
	SourcePath            string                     `json:"source_path"`
	Line                  int                        `json:"line"`
	SourceSHA256          string                     `json:"source_sha256"`
	ArgumentContextStatus string                     `json:"argument_context_status"`
	ArgumentCallee        string                     `json:"argument_callee,omitempty"`
	ArgumentLine          int                        `json:"argument_line,omitempty"`
	MatchedTerms          []string                   `json:"matched_terms"`
	MatchedFields         []string                   `json:"matched_fields"`
	MatchEvidence         []AmbientSeamMatchEvidence `json:"match_evidence"`
	MatchStatus           string                     `json:"match_status"`
	Basis                 string                     `json:"basis"`
	Limit                 string                     `json:"limit"`
	Score                 float64                    `json:"score"`
}

// AmbientSeamMatchEvidence connects one exact lexical query token to the
// metadata field and source revision that produced this candidate.
type AmbientSeamMatchEvidence struct {
	Term         string `json:"term"`
	Field        string `json:"field"`
	Value        string `json:"value"`
	SourcePath   string `json:"source_path"`
	Line         int    `json:"line"`
	SourceSHA256 string `json:"source_sha256"`
}

type AmbientSeamDegradation struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type AmbientSeamResponse struct {
	Query        string                   `json:"query"`
	Scenario     string                   `json:"scenario"`
	Scope        string                   `json:"scope"`
	MatchStatus  string                   `json:"match_status"`
	Results      []AmbientSeam            `json:"results"`
	Degradations []AmbientSeamDegradation `json:"degradations"`
}

type ambientSource struct {
	path    string
	content []byte
}
type ambientCandidate struct{ seam AmbientSeam }

// SearchAmbientSeams classifies authored Go call sites by source-file role and
// reports declared build constraints. It does not claim that a condition matches
// a target build, that a non-test file is linked into a production binary, or
// that a test file is executed or substitutes a production dependency.
func (s *Service) SearchAmbientSeams(_ context.Context, query, scenario string, limit int) (*AmbientSeamResponse, error) {
	query, scenario = strings.TrimSpace(query), strings.TrimSpace(scenario)
	if query == "" || !invariantScenarioPattern.MatchString(scenario) {
		return nil, fmt.Errorf("query and one scenario slug are required")
	}
	if limit <= 0 || limit > maxAmbientHits {
		limit = maxAmbientHits
	}
	target, err := resolveTarget(&factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: scenario})
	if err != nil {
		return nil, err
	}
	root, err := filepath.EvalSymlinks(target.GetRootPath())
	if err != nil {
		return nil, err
	}
	response, err := scanAmbientSeams(root, scenario, query, limit)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func scanAmbientSeams(root, scenario, query string, limit int) (*AmbientSeamResponse, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	response := &AmbientSeamResponse{Query: strings.TrimSpace(query), Scenario: scenario, Results: []AmbientSeam{}, Degradations: []AmbientSeamDegradation{}, Scope: fmt.Sprintf("scenario:%s Go source only; at most %d files/%d MiB; test-file suffix and declared go:build/+build conditions are static source facts; declared conditions are not evaluated against target GOOS/GOARCH/tags; no binary membership, runtime use, or test substitution is inferred", scenario, maxAmbientFiles, maxAmbientBytes>>20), MatchStatus: "UNKNOWN"}
	if ambientQueryUnsupported(query) {
		response.Degradations = append(response.Degradations, AmbientSeamDegradation{"query_unsupported_syntax", "Boolean, quoted, wildcard, and scoped query syntax is unsupported; no lexical match was asserted."})
		return response, nil
	}
	files, err := ambientSourceFiles(root)
	if err != nil {
		return nil, err
	}
	terms := ambientTokens(query)
	if len(terms) == 0 {
		response.Degradations = append(response.Degradations, AmbientSeamDegradation{"query_unusable", "Query contains no searchable terms."})
		return response, nil
	}
	var candidates []ambientCandidate
	for _, source := range files {
		found, err := ambientFileCandidates(scenario, source)
		if err != nil {
			response.Degradations = append(response.Degradations, AmbientSeamDegradation{"source_parse_degraded", "Some bounded Go files could not be parsed; returned source evidence may be incomplete."})
			continue
		}
		for _, seam := range found {
			score, matchedTerms, matchedFields, matchEvidence := ambientMatchDetails(terms, seam)
			if score == 0 {
				continue
			}
			seam.Score = score
			seam.MatchedTerms = matchedTerms
			seam.MatchedFields = matchedFields
			seam.MatchEvidence = matchEvidence
			candidates = append(candidates, ambientCandidate{seam: seam})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].seam.Score != candidates[j].seam.Score {
			return candidates[i].seam.Score > candidates[j].seam.Score
		}
		if candidates[i].seam.SourcePath != candidates[j].seam.SourcePath {
			return candidates[i].seam.SourcePath < candidates[j].seam.SourcePath
		}
		return candidates[i].seam.Line < candidates[j].seam.Line
	})
	response.MatchStatus = "MATCHED"
	for start := 0; start < len(candidates); {
		end := start + 1
		for end < len(candidates) && candidates[end].seam.Score == candidates[start].seam.Score {
			end++
		}
		if end-start > 1 {
			response.MatchStatus = "UNKNOWN"
			response.Degradations = append(response.Degradations, AmbientSeamDegradation{"ambiguous_match_tie", "Multiple source candidates have equal lexical overlap; their relative result order is unknown."})
			for index := start; index < end; index++ {
				candidates[index].seam.MatchStatus = "UNKNOWN"
			}
		}
		start = end
	}
	for _, candidate := range candidates {
		if len(response.Results) == limit {
			break
		}
		if candidate.seam.MatchStatus == "" {
			candidate.seam.MatchStatus = "MATCHED"
		}
		response.Results = append(response.Results, candidate.seam)
	}
	if len(response.Results) == 0 {
		response.MatchStatus = "UNKNOWN"
		response.Degradations = append(response.Degradations, AmbientSeamDegradation{"no_seam_match", "No recognized ambient seam matched within the bounded Go source scope."})
	}
	return response, nil
}

func ambientSourceFiles(root string) ([]ambientSource, error) {
	var files []ambientSource
	var total int64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == ".git" || entry.Name() == "vendor" || entry.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(path, ".go") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		if len(files)+1 > maxAmbientFiles || total > maxAmbientBytes {
			return fmt.Errorf("ambient seam source exceeds bounded scan (%d files, %d bytes)", maxAmbientFiles, maxAmbientBytes)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, ambientSource{path: filepath.ToSlash(rel), content: content})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("scenario contains no bounded Go source files")
	}
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	return files, nil
}

func ambientFileCandidates(scenario string, source ambientSource) ([]AmbientSeam, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, source.path, source.content, parser.AllErrors)
	if err != nil {
		return nil, err
	}
	buildConstraint := ambientBuildConstraintForSource(source.content)
	imports := map[string]string{}
	for _, spec := range file.Imports {
		importPath := strings.Trim(spec.Path.Value, "\"")
		alias := filepath.Base(importPath)
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		imports[alias] = importPath
	}
	hash := sha256.Sum256(source.content)
	digest := hex.EncodeToString(hash[:])
	functions := []struct {
		name       string
		start, end token.Pos
	}{}
	ast.Inspect(file, func(node ast.Node) bool {
		if fn, ok := node.(*ast.FuncDecl); ok {
			name := fn.Name.Name
			if fn.Recv != nil {
				name = ambientReceiverName(fn.Recv) + "." + name
			}
			functions = append(functions, struct {
				name       string
				start, end token.Pos
			}{name, fn.Pos(), fn.End()})
		}
		if literal, ok := node.(*ast.FuncLit); ok {
			functions = append(functions, struct {
				name       string
				start, end token.Pos
			}{"<function literal>", literal.Pos(), literal.End()})
		}
		return true
	})
	var calls []*ast.CallExpr
	ast.Inspect(file, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			calls = append(calls, call)
		}
		return true
	})
	var results []AmbientSeam
	seen := map[string]bool{}
	add := func(node ast.Node, expr string, kind string) {
		pos := fset.Position(node.Pos())
		symbol := "package initializer"
		functionSpan := token.Pos(1 << 62)
		for _, fn := range functions {
			if fn.start <= node.Pos() && node.End() <= fn.end && fn.end-fn.start < functionSpan {
				symbol = fn.name
				functionSpan = fn.end - fn.start
			}
		}
		layer := "NON_TEST_SOURCE_CANDIDATE"
		if strings.HasSuffix(source.path, "_test.go") {
			layer = "TEST_SOURCE_CANDIDATE"
		}
		argumentCallee := ""
		argumentLine := 0
		argumentSpan := token.Pos(1 << 62)
		for _, call := range calls {
			for _, arg := range call.Args {
				if arg.Pos() <= node.Pos() && node.End() <= arg.End() && call.End()-call.Pos() < argumentSpan {
					var rendered strings.Builder
					if format.Node(&rendered, fset, call.Fun) == nil {
						argumentCallee = rendered.String()
						argumentLine = fset.Position(call.Pos()).Line
						argumentSpan = call.End() - call.Pos()
					}
				}
			}
		}
		argumentContextStatus := "UNKNOWN_STATIC_CONTEXT_ONLY"
		if argumentCallee != "" {
			argumentContextStatus = "STATIC_ARGUMENT_RELATIONSHIP"
		}
		key := fmt.Sprintf("%s:%d:%s", source.path, pos.Line, expr)
		if seen[key] {
			return
		}
		seen[key] = true
		title := kind + " seam: " + expr
		summary := fmt.Sprintf("%s in %s at %s:%d within %s; build condition %s; argument context %s", kind, layer, source.path, pos.Line, symbol, buildConstraint.status, argumentContextStatus)
		if buildConstraint.expression != "" {
			summary += "; declared build condition " + buildConstraint.expression
		}
		if argumentCallee != "" {
			summary += fmt.Sprintf("; passed as an argument to %s at line %d", argumentCallee, argumentLine)
		}
		results = append(results, AmbientSeam{ID: fmt.Sprintf("%s:ambient:%s:%d:%s", scenario, source.path, pos.Line, kind), Title: title, Summary: summary, Scenario: scenario, Kind: kind, Expression: expr, ContextSymbol: symbol, SourceLayer: layer, BuildConstraint: buildConstraint.expression, BuildConstraintStatus: buildConstraint.status, BuildConstraintLine: buildConstraint.line, SourcePath: source.path, Line: pos.Line, SourceSHA256: digest, ArgumentContextStatus: argumentContextStatus, ArgumentCallee: argumentCallee, ArgumentLine: argumentLine, Basis: "go_ast_call_or_provider_expression", Limit: "Declared constraints and file suffixes are not evaluated for a target build; source location does not prove binary membership, production/test dispatch, runtime use, or replacement of this dependency."})
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.CallExpr:
			if expr, kind, ok := ambientCall(value.Fun, imports); ok {
				add(value, expr, kind)
			}
		case *ast.SelectorExpr:
			if expr, kind, ok := ambientValue(value, imports); ok {
				add(value, expr, kind)
			}
		case *ast.CompositeLit:
			if expr, kind, ok := ambientClientLiteral(value.Type, imports); ok {
				add(value, expr, kind)
			}
		}
		return true
	})
	return results, nil
}

type ambientBuildConstraintInfo struct {
	expression string
	status     string
	line       int
}

func ambientBuildConstraintForSource(content []byte) ambientBuildConstraintInfo {
	result := ambientBuildConstraintInfo{status: "NOT_DECLARED"}
	lines := strings.Split(string(content), "\n")
	goExprs := []constraint.Expr{}
	legacyExprs := []constraint.Expr{}
	inBlockComment := false
	lastDirectiveLine := 0
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if inBlockComment {
			if end := strings.Index(trimmed, "*/"); end >= 0 {
				trimmed = strings.TrimSpace(trimmed[end+2:])
				inBlockComment = false
				if trimmed == "" {
					continue
				}
			} else {
				continue
			}
		}
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "/*") {
			if !strings.Contains(trimmed, "*/") {
				inBlockComment = true
			}
			continue
		}
		if !strings.HasPrefix(trimmed, "//") {
			if lastDirectiveLine > 0 && index > 0 && strings.TrimSpace(lines[index-1]) != "" {
				result.status = "UNKNOWN_DIRECTIVE_PLACEMENT"
				return result
			}
			break
		}
		isGo := strings.HasPrefix(trimmed, "//go:build")
		isLegacy := strings.HasPrefix(trimmed, "// +build")
		if !isGo && !isLegacy {
			continue
		}
		if result.line == 0 {
			result.line = index + 1
		}
		lastDirectiveLine = index + 1
		if len(trimmed) > 2048 || len(goExprs)+len(legacyExprs) >= 32 || (isGo && len(goExprs) > 0) {
			result.status = "UNKNOWN_DIRECTIVE_INCOMPLETE"
			return result
		}
		expr, err := constraint.Parse(trimmed)
		if err != nil || expr == nil {
			result.status = "UNKNOWN_DIRECTIVE_INVALID"
			return result
		}
		if isGo {
			goExprs = append(goExprs, expr)
		} else {
			legacyExprs = append(legacyExprs, expr)
		}
	}
	if len(goExprs) == 0 && len(legacyExprs) == 0 {
		return result
	}
	var goExpr, legacyExpr constraint.Expr
	if len(goExprs) > 0 {
		goExpr = goExprs[0]
	}
	for _, expr := range legacyExprs {
		if legacyExpr == nil {
			legacyExpr = expr
		} else {
			legacyExpr = &constraint.AndExpr{X: legacyExpr, Y: expr}
		}
	}
	if goExpr != nil && legacyExpr != nil && goExpr.String() != legacyExpr.String() {
		result.status = "UNKNOWN_DIRECTIVE_CONFLICT"
		result.expression = goExpr.String()
		return result
	}
	if goExpr != nil {
		result.expression = goExpr.String()
		result.status = "DECLARED_UNEVALUATED"
	} else {
		result.expression = legacyExpr.String()
		result.status = "LEGACY_DECLARED_UNEVALUATED"
	}
	return result
}

func ambientCall(expr ast.Expr, imports map[string]string) (string, string, bool) {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", "", false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", "", false
	}
	path := imports[pkg.Name]
	name := sel.Sel.Name
	switch {
	case path == "time" && oneOf(name, "Now", "Since", "After", "NewTicker", "NewTimer"):
		return pkg.Name + "." + name, "clock", true
	case path == "os" && oneOf(name, "Getenv", "LookupEnv", "Environ"):
		return pkg.Name + "." + name, "environment", true
	case path == "log" && oneOf(name, "Default", "New"):
		return pkg.Name + "." + name, "logger", true
	case path == "log/slog" && oneOf(name, "Default", "New"):
		return pkg.Name + "." + name, "logger", true
	case path == "net/http" && oneOf(name, "DefaultClient"):
		return pkg.Name + "." + name, "http_client", true
	}
	return "", "", false
}
func ambientValue(expr *ast.SelectorExpr, imports map[string]string) (string, string, bool) {
	pkg, ok := expr.X.(*ast.Ident)
	if !ok {
		return "", "", false
	}
	path, name := imports[pkg.Name], expr.Sel.Name
	if path == "net/http" && oneOf(name, "DefaultClient", "DefaultTransport") {
		return pkg.Name + "." + name, "http_client", true
	}
	if path == "time" && name == "Now" {
		return pkg.Name + "." + name, "clock", true
	}
	return "", "", false
}
func ambientClientLiteral(expr ast.Expr, imports map[string]string) (string, string, bool) {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", "", false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", "", false
	}
	if imports[pkg.Name] == "net/http" && oneOf(sel.Sel.Name, "Client", "Transport") {
		return pkg.Name + "." + sel.Sel.Name, "http_client", true
	}
	return "", "", false
}
func ambientReceiverName(field *ast.FieldList) string {
	if field == nil || len(field.List) == 0 {
		return "receiver"
	}
	switch t := field.List[0].Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return "receiver"
}
func ambientMatchDetails(terms []string, seam AmbientSeam) (float64, []string, []string, []AmbientSeamMatchEvidence) {
	// Keep the score equal to the previous lexical term-overlap score while
	// requiring whole-token matches and exposing exact metadata evidence.
	fields := []struct {
		name, value string
		line        int
	}{
		{"kind", seam.Kind, seam.Line},
		{"expression", seam.Expression, seam.Line},
		{"function", seam.ContextSymbol, seam.Line},
		{"source_role", seam.SourceLayer, seam.Line},
		{"source_path", seam.SourcePath, seam.Line},
		{"argument_callee", seam.ArgumentCallee, seam.ArgumentLine},
		{"build_constraint", seam.BuildConstraint, seam.BuildConstraintLine},
	}
	matchedTerms := make([]string, 0, len(terms))
	matchedFields := make([]string, 0, len(fields))
	matchEvidence := make([]AmbientSeamMatchEvidence, 0, len(terms))
	fieldMatches := make([]bool, len(fields))
	for _, term := range terms {
		termEvidenceStart := len(matchEvidence)
		for i, field := range fields {
			if ambientFieldHasTerm(field.value, term) {
				fieldMatches[i] = true
				matchEvidence = append(matchEvidence, AmbientSeamMatchEvidence{Term: term, Field: field.name, Value: field.value, SourcePath: seam.SourcePath, Line: field.line, SourceSHA256: seam.SourceSHA256})
			}
		}
		if len(matchEvidence) > termEvidenceStart {
			matchedTerms = append(matchedTerms, term)
		}
	}
	for i, matched := range fieldMatches {
		if matched {
			matchedFields = append(matchedFields, fields[i].name)
		}
	}
	if len(terms) == 0 {
		return 0, matchedTerms, matchedFields, matchEvidence
	}
	return float64(len(matchedTerms)) / float64(len(terms)), matchedTerms, matchedFields, matchEvidence
}

func ambientFieldHasTerm(value, term string) bool {
	for _, fieldToken := range ambientTokens(value) {
		if fieldToken == term {
			return true
		}
	}
	return false
}

func ambientQueryUnsupported(query string) bool {
	if strings.ContainsAny(query, "\"'`*?()[]{}:|&") {
		return true
	}
	for _, token := range ambientTokens(query) {
		if oneOf(token, "and", "or", "not") {
			return true
		}
	}
	return false
}
func ambientTokens(query string) []string {
	fields := strings.FieldsFunc(query, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
	seen := map[string]bool{}
	var terms []string
	appendTerm := func(term string) {
		term = strings.ToLower(term)
		if len(term) > 1 && !seen[term] {
			seen[term] = true
			terms = append(terms, term)
		}
	}
	for _, field := range fields {
		runes := []rune(field)
		start := 0
		for i := 1; i < len(runes); i++ {
			previous, current := runes[i-1], runes[i]
			if !unicode.IsUpper(current) {
				continue
			}
			boundary := unicode.IsLower(previous) || unicode.IsNumber(previous)
			if !boundary && unicode.IsUpper(previous) && i+1 < len(runes) {
				boundary = unicode.IsLower(runes[i+1])
			}
			if boundary {
				appendTerm(string(runes[start:i]))
				start = i
			}
		}
		appendTerm(string(runes[start:]))
	}
	sort.Strings(terms)
	return terms
}
func oneOf(value string, values ...string) bool {
	for _, want := range values {
		if value == want {
			return true
		}
	}
	return false
}
