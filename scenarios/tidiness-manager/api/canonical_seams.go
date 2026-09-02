package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const (
	repositorySeamsPath = ".vrooli/canonical-seams.json"
	targetSeamsPath     = ".vrooli/seams.json"
)

type Seam struct {
	ID            string     `json:"id"`
	Canonical     string     `json:"canonical"`
	Why           string     `json:"why"`
	Remediation   string     `json:"remediation"`
	Bypass        SeamBypass `json:"bypass"`
	Scope         SeamScope  `json:"scope"`
	Severity      string     `json:"severity"`
	Budget        int        `json:"budget"`
	Reserve       int        `json:"reserve,omitempty"`
	ReserveReason string     `json:"reserve_reason,omitempty"`
	Resolver      string     `json:"resolver,omitempty"`
	Disabled      bool       `json:"disabled,omitempty"`
}

type SeamBypass struct {
	Kind                   string   `json:"kind"`
	Pattern                string   `json:"pattern"`
	DeclKind               string   `json:"declKind,omitempty"`
	GenericWords           []string `json:"genericWords,omitempty"`
	MinDomainWords         int      `json:"minDomainWords,omitempty"`
	Unit                   string   `json:"unit,omitempty"`
	CanonicalCall          string   `json:"canonicalCall,omitempty"`
	OuterKey               string   `json:"outerKey,omitempty"`
	InnerKey               string   `json:"innerKey,omitempty"`
	ConstructedType        string   `json:"constructedType,omitempty"`
	ExcludeSymbols         []string `json:"excludeSymbols,omitempty"`
	RepeatedAcrossPackages int      `json:"repeatedAcrossPackages,omitempty"`
	PairedPathPatterns     []string `json:"pairedPathPatterns,omitempty"`
	ExcludeAliases         bool     `json:"excludeAliases,omitempty"`
	ShapeKind              string   `json:"shapeKind,omitempty"`
	MinMembers             int      `json:"minMembers,omitempty"`
	RequireFor             string   `json:"requireFor,omitempty"`
	RequirePresent         string   `json:"requirePresent,omitempty"`
	ForbidKind             string   `json:"forbidKind,omitempty"`
}

type SeamScope struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

type seamFile struct {
	Schema        string `json:"$schema,omitempty"`
	SchemaVersion string `json:"schemaVersion"`
	Seams         []Seam `json:"seams"`
}

type SeamHit struct {
	SeamID      string
	Canonical   string
	Why         string
	Remediation string
	Severity    string
	Budget      int
	Path        string
	Symbol      string
	Line        int
	Analyzer    string
}

type compiledSeam struct {
	seam        Seam
	pattern     *regexp.Regexp
	pairedPaths []*regexp.Regexp
}

func LoadSeams(treeRoot string) ([]Seam, error) {
	return loadSeamsFile(filepath.Join(treeRoot, filepath.FromSlash(repositorySeamsPath)), false)
}

func loadSeamsFile(path string, allowDisabled bool) ([]Seam, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) || os.IsPermission(err) {
		return []Seam{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read canonical seams: %w", err)
	}
	var config seamFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("decode canonical seams: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("decode canonical seams: multiple JSON values")
		}
		return nil, fmt.Errorf("decode canonical seams trailing data: %w", err)
	}
	if config.SchemaVersion != "1.0.0" {
		return nil, fmt.Errorf("canonical seams schemaVersion must be 1.0.0")
	}
	seen := map[string]struct{}{}
	for i := range config.Seams {
		seam := &config.Seams[i]
		seam.ID = strings.TrimSpace(seam.ID)
		if _, duplicate := seen[seam.ID]; duplicate {
			return nil, fmt.Errorf("canonical seam id %q is duplicated", seam.ID)
		}
		seen[seam.ID] = struct{}{}
		if seam.Disabled {
			if !allowDisabled {
				return nil, fmt.Errorf("canonical seam %q cannot be disabled in the repository baseline", seam.ID)
			}
			if seam.ID == "" || strings.TrimSpace(seam.ReserveReason) == "" {
				return nil, fmt.Errorf("disabled canonical seam %d requires id and reserve_reason", i)
			}
			continue
		}
		if seam.ID == "" || strings.TrimSpace(seam.Canonical) == "" || strings.TrimSpace(seam.Why) == "" || strings.TrimSpace(seam.Remediation) == "" {
			return nil, fmt.Errorf("canonical seam %d is missing required text", i)
		}
		if !validBypassKind(seam.Bypass.Kind) {
			return nil, fmt.Errorf("canonical seam %q has unsupported bypass kind %q", seam.ID, seam.Bypass.Kind)
		}
		if seam.Resolver != "" && seam.Resolver != "local" && seam.Resolver != "code-facts" {
			return nil, fmt.Errorf("canonical seam %q has invalid resolver %q", seam.ID, seam.Resolver)
		}
		if seam.Bypass.DeclKind != "" && !validDeclarationKind(seam.Bypass.DeclKind) {
			return nil, fmt.Errorf("canonical seam %q has invalid declaration kind %q", seam.ID, seam.Bypass.DeclKind)
		}
		if seam.Bypass.RepeatedAcrossPackages < 0 {
			return nil, fmt.Errorf("canonical seam %q has invalid repeatedAcrossPackages", seam.ID)
		}
		if len(seam.Bypass.PairedPathPatterns) != 0 && len(seam.Bypass.PairedPathPatterns) != 2 {
			return nil, fmt.Errorf("canonical seam %q pairedPathPatterns requires exactly two patterns", seam.ID)
		}
		for _, pathPattern := range seam.Bypass.PairedPathPatterns {
			compiled, err := regexp.Compile(pathPattern)
			if err != nil || compiled.NumSubexp() != 1 {
				return nil, fmt.Errorf("canonical seam %q pairedPathPatterns entries must compile with exactly one capture group", seam.ID)
			}
		}
		if seam.Bypass.MinMembers < 0 {
			return nil, fmt.Errorf("canonical seam %q has invalid minMembers", seam.ID)
		}
		if seam.Bypass.MinDomainWords < 0 {
			return nil, fmt.Errorf("canonical seam %q has invalid minDomainWords", seam.ID)
		}
		if seam.Bypass.Kind == "semantic-naming" && (seam.Bypass.DeclKind == "" || seam.Bypass.MinDomainWords < 1) {
			return nil, fmt.Errorf("canonical seam %q semantic-naming requires declKind and positive minDomainWords", seam.ID)
		}
		if seam.Bypass.Kind == "suppression-breadth" && seam.Bypass.Unit != "" && seam.Bypass.Unit != "declarations" && seam.Bypass.Unit != "directives" {
			return nil, fmt.Errorf("canonical seam %q suppression-breadth has invalid unit %q", seam.ID, seam.Bypass.Unit)
		}
		if seam.Bypass.Kind == "replaced-call" && (strings.TrimSpace(seam.Bypass.CanonicalCall) == "" || seam.Bypass.ShapeKind == "") {
			return nil, fmt.Errorf("canonical seam %q replaced-call requires canonicalCall and shapeKind", seam.ID)
		}
		if seam.Bypass.ShapeKind == "json_nesting" && (strings.TrimSpace(seam.Bypass.OuterKey) == "" || strings.TrimSpace(seam.Bypass.InnerKey) == "") {
			return nil, fmt.Errorf("canonical seam %q json_nesting requires outerKey and innerKey", seam.ID)
		}
		if seam.Bypass.ShapeKind == "constructs_type" && strings.TrimSpace(seam.Bypass.ConstructedType) == "" {
			return nil, fmt.Errorf("canonical seam %q constructs_type requires constructedType", seam.ID)
		}
		if seam.Reserve < 0 {
			return nil, fmt.Errorf("canonical seam %q has invalid reserve", seam.ID)
		}
		if (seam.Budget > 0 || seam.Reserve > 0) && strings.TrimSpace(seam.ReserveReason) == "" {
			return nil, fmt.Errorf("canonical seam %q requires reserve_reason for non-zero budget or reserve", seam.ID)
		}
		if seam.Bypass.Kind == "absence" {
			if seam.Bypass.RequireFor == "" || (seam.Bypass.RequirePresent == "" && seam.Bypass.ForbidKind == "") {
				return nil, fmt.Errorf("canonical seam %q absence requires requireFor and either requirePresent or forbidKind", seam.ID)
			}
			if seam.Bypass.ForbidKind != "" && seam.Bypass.ForbidKind != "executable" {
				return nil, fmt.Errorf("canonical seam %q has unsupported forbidden kind %q", seam.ID, seam.Bypass.ForbidKind)
			}
		}
		if seam.Bypass.ShapeKind != "" && !validShapeKind(seam.Bypass.ShapeKind) {
			return nil, fmt.Errorf("canonical seam %q has invalid shape kind %q", seam.ID, seam.Bypass.ShapeKind)
		}
		if _, err := regexp.Compile(seam.Bypass.Pattern); err != nil {
			return nil, fmt.Errorf("canonical seam %q bypass pattern: %w", seam.ID, err)
		}
		if len(seam.Scope.Include) == 0 || seam.Budget < 0 {
			return nil, fmt.Errorf("canonical seam %q requires include scope and non-negative budget", seam.ID)
		}
		switch seam.Severity {
		case "info", "low", "medium", "high", "critical":
		default:
			return nil, fmt.Errorf("canonical seam %q has invalid severity %q", seam.ID, seam.Severity)
		}
	}
	return config.Seams, nil
}

type SeamTargetKind string

const (
	SeamTargetScenario     SeamTargetKind = "scenario"
	SeamTargetControlPlane SeamTargetKind = "control-plane"
)

type SeamResolution struct {
	ScanRoot string
	Files    []string
}

func ResolveSeams(scanRoot string, targetKind SeamTargetKind) ([]Seam, SeamResolution, error) {
	scanRoot = filepath.Clean(scanRoot)
	repositoryRoot := seamRepositoryRoot(scanRoot)
	effectiveScanRoot := scanRoot
	if targetKind == SeamTargetControlPlane {
		effectiveScanRoot = repositoryRoot
	}
	resolution := SeamResolution{ScanRoot: effectiveScanRoot}
	baselinePath := filepath.Join(repositoryRoot, filepath.FromSlash(repositorySeamsPath))
	baseline, err := loadSeamsFile(baselinePath, false)
	if err != nil {
		return nil, resolution, err
	}
	if pathExists(baselinePath) {
		resolution.Files = append(resolution.Files, filepath.ToSlash(baselinePath))
	}
	if targetKind == SeamTargetScenario && repositoryRoot != scanRoot {
		baseline = rebaseRepositorySeams(baseline, repositoryRoot, scanRoot)
	}
	overlayPath := filepath.Join(scanRoot, filepath.FromSlash(targetSeamsPath))
	if !pathExists(overlayPath) {
		return baseline, resolution, nil
	}
	overlay, err := loadSeamsFile(overlayPath, true)
	if err != nil {
		return nil, resolution, err
	}
	merged, err := mergeSeams(baseline, overlay)
	if err != nil {
		return nil, resolution, err
	}
	resolution.Files = append(resolution.Files, filepath.ToSlash(overlayPath))
	return merged, resolution, nil
}

func rebaseRepositorySeams(seams []Seam, repositoryRoot, targetRoot string) []Seam {
	targetRelative, err := filepath.Rel(repositoryRoot, targetRoot)
	if err != nil || targetRelative == "." || strings.HasPrefix(targetRelative, "..") {
		return nil
	}
	prefix := filepath.ToSlash(targetRelative) + "/"
	rebased := make([]Seam, 0, len(seams))
	for _, seam := range seams {
		includes := make([]string, 0, len(seam.Scope.Include))
		for _, pattern := range seam.Scope.Include {
			if strings.HasPrefix(pattern, prefix) {
				includes = append(includes, strings.TrimPrefix(pattern, prefix))
			}
		}
		if len(includes) == 0 {
			continue
		}
		seam.Scope.Include = includes
		for index, pattern := range seam.Scope.Exclude {
			if strings.HasPrefix(pattern, prefix) {
				seam.Scope.Exclude[index] = strings.TrimPrefix(pattern, prefix)
			}
		}
		rebased = append(rebased, seam)
	}
	return rebased
}

func seamRepositoryRoot(scanRoot string) string {
	for candidate := filepath.Clean(scanRoot); ; candidate = filepath.Dir(candidate) {
		if pathExists(filepath.Join(candidate, filepath.FromSlash(repositorySeamsPath))) {
			return candidate
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return scanRoot
		}
	}
}

func mergeSeams(baseline, overlay []Seam) ([]Seam, error) {
	merged := append([]Seam(nil), baseline...)
	indexes := make(map[string]int, len(merged))
	for index, seam := range merged {
		indexes[seam.ID] = index
	}
	for _, seam := range overlay {
		index, exists := indexes[seam.ID]
		if seam.Disabled {
			if !exists {
				return nil, fmt.Errorf("disabled canonical seam %q does not exist in the repository baseline", seam.ID)
			}
			merged = append(merged[:index], merged[index+1:]...)
			indexes = make(map[string]int, len(merged))
			for replacementIndex, remaining := range merged {
				indexes[remaining.ID] = replacementIndex
			}
			continue
		}
		if exists {
			merged[index] = seam
			continue
		}
		indexes[seam.ID] = len(merged)
		merged = append(merged, seam)
	}
	return merged, nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// seamTreeRoot keeps ordinary scenario scans local while allowing the control
// plane's bounded internal/ target to consume repository-level seam rules.
func seamTreeRoot(scanRoot string) string {
	if _, err := os.Stat(filepath.Join(scanRoot, filepath.FromSlash(repositorySeamsPath))); err == nil {
		return scanRoot
	}
	if filepath.Base(filepath.Clean(scanRoot)) != "internal" {
		return scanRoot
	}
	parent := filepath.Dir(filepath.Clean(scanRoot))
	if _, err := os.Stat(filepath.Join(parent, filepath.FromSlash(repositorySeamsPath))); err == nil {
		return parent
	}
	return scanRoot
}

func ScanSeams(treeRoot string, seams []Seam) ([]SeamHit, error) {
	if len(seams) == 0 {
		return []SeamHit{}, nil
	}
	compiled := make([]compiledSeam, 0, len(seams))
	for _, seam := range seams {
		pattern, err := regexp.Compile(seam.Bypass.Pattern)
		if err != nil {
			return nil, fmt.Errorf("canonical seam %q bypass pattern: %w", seam.ID, err)
		}
		candidate := compiledSeam{seam: seam, pattern: pattern}
		for _, pathPattern := range seam.Bypass.PairedPathPatterns {
			candidate.pairedPaths = append(candidate.pairedPaths, regexp.MustCompile(pathPattern))
		}
		compiled = append(compiled, candidate)
	}
	absenceHits, err := scanAbsenceSeams(treeRoot, compiled)
	if err != nil {
		return nil, err
	}
	declarationPackages, err := collectDeclarationPackages(treeRoot, compiled)
	if err != nil {
		return nil, err
	}
	shapeGroups, err := collectShapeGroups(treeRoot, compiled)
	if err != nil {
		return nil, err
	}
	brokeredHits, err := scanCodeFactsSeams(treeRoot, compiled)
	if err != nil {
		return nil, err
	}
	hits := append(absenceHits, brokeredHits...)
	err = filepath.WalkDir(treeRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != treeRoot && (entry.Name() == ".git" || entry.Name() == "node_modules" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		rel, err := filepath.Rel(treeRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		applicable := make([]compiledSeam, 0, len(compiled))
		for _, candidate := range compiled {
			if candidate.seam.Bypass.Kind == "absence" {
				continue
			}
			if seamPathIncluded(rel, candidate.seam.Scope) {
				applicable = append(applicable, candidate)
			}
		}
		if len(applicable) == 0 {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parse %s: %w", rel, err)
		}
		aliases := seamImportAliases(file)
		for _, commentGroup := range file.Comments {
			for _, comment := range commentGroup.List {
				for _, candidate := range applicable {
					if (candidate.seam.Bypass.Kind != "directive" && candidate.seam.Bypass.Kind != "suppression-breadth") || !candidate.pattern.MatchString(strings.TrimSpace(comment.Text)) {
						continue
					}
					position := fset.Position(comment.Pos())
					cost := 1
					if candidate.seam.Bypass.Kind == "suppression-breadth" && candidate.seam.Bypass.Unit != "directives" && comment.Pos() < file.Package {
						cost = topLevelDeclarationCount(file)
					}
					for range cost {
						hits = append(hits, SeamHit{SeamID: candidate.seam.ID, Canonical: candidate.seam.Canonical, Why: candidate.seam.Why, Remediation: candidate.seam.Remediation, Severity: candidate.seam.Severity, Budget: candidate.seam.Budget, Path: rel, Symbol: strings.TrimSpace(comment.Text), Line: position.Line})
					}
				}
			}
		}
		constStack := make([]bool, 0, 16)
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				constStack = constStack[:len(constStack)-1]
				return true
			}
			inConst := len(constStack) > 0 && constStack[len(constStack)-1]
			if declaration, ok := node.(*ast.GenDecl); ok && declaration.Tok == token.CONST {
				inConst = true
			}
			constStack = append(constStack, inConst)
			for _, candidate := range applicable {
				if candidate.seam.Resolver == "code-facts" {
					continue
				}
				if seamDeclaresCanonical(file, candidate.seam.Canonical) {
					continue
				}
				var symbols []string
				if candidate.seam.Bypass.Kind == "declaration" {
					symbols = declarationMatches(node, candidate.pattern, candidate.seam.Bypass.DeclKind, inConst, candidate.seam.Bypass.ExcludeAliases)
				} else if candidate.seam.Bypass.Kind == "semantic-naming" {
					symbols = semanticNamingMatches(node, candidate.pattern, candidate.seam.Bypass, inConst)
				} else if candidate.seam.Bypass.Kind == "replaced-call" {
					symbols = replacedCallMatches(node, candidate.pattern, candidate.seam.Bypass.ShapeKind)
				} else if candidate.seam.Bypass.Kind == "shape" {
					symbols = shapeMatches(node, candidate.seam.Bypass, candidate.pattern, shapeGroups[candidate.seam.ID])
				} else {
					symbols = seamNodeMatches(node, candidate.seam.Bypass.Kind, candidate.pattern, aliases, inConst)
				}
				for _, symbol := range symbols {
					if symbolExcluded(symbol, candidate.seam.Bypass.ExcludeSymbols) {
						continue
					}
					if minimum := candidate.seam.Bypass.RepeatedAcrossPackages; minimum > 0 && len(declarationPackages[candidate.seam.ID+"\x00"+symbol]) < minimum {
						continue
					}
					if len(candidate.pairedPaths) == 2 {
						if !hasPairedPathDeclarations(declarationPackages[candidate.seam.ID+"\x00"+symbol], candidate.pairedPaths, rel) || pairedPathSide(candidate.pairedPaths, rel) != 1 {
							continue
						}
					}
					position := fset.Position(node.Pos())
					hits = append(hits, SeamHit{SeamID: candidate.seam.ID, Canonical: candidate.seam.Canonical, Why: candidate.seam.Why, Remediation: candidate.seam.Remediation, Severity: candidate.seam.Severity, Budget: candidate.seam.Budget, Path: rel, Symbol: symbol, Line: position.Line})
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].SeamID != hits[j].SeamID {
			return hits[i].SeamID < hits[j].SeamID
		}
		if hits[i].Path != hits[j].Path {
			return hits[i].Path < hits[j].Path
		}
		return hits[i].Line < hits[j].Line
	})
	return hits, nil
}

var supportedBypassKinds = []string{"call", "literal", "declaration", "semantic-naming", "replaced-call", "shape", "directive", "suppression-breadth", "absence"}

func validBypassKind(kind string) bool {
	for _, supported := range supportedBypassKinds {
		if kind == supported {
			return true
		}
	}
	return false
}

func topLevelDeclarationCount(file *ast.File) int {
	count := 0
	for _, declaration := range file.Decls {
		switch value := declaration.(type) {
		case *ast.FuncDecl:
			count++
		case *ast.GenDecl:
			if value.Tok != token.IMPORT {
				count += len(value.Specs)
			}
		}
	}
	if count == 0 {
		return 1
	}
	return count
}

func replacedCallMatches(node ast.Node, pattern *regexp.Regexp, shapeKind string) []string {
	function, ok := node.(*ast.FuncDecl)
	if !ok || function.Body == nil || ast.IsExported(function.Name.Name) || !pattern.MatchString(function.Name.Name) {
		return nil
	}
	hasCall := false
	ast.Inspect(function.Body, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if builtin, ok := call.Fun.(*ast.Ident); ok && (builtin.Name == "len" || builtin.Name == "cap") {
				return true
			}
			hasCall = true
			return false
		}
		return !hasCall
	})
	if hasCall || shapeKind != "nested_swap_loop" || !hasNestedSwapLoop(function.Body) {
		return nil
	}
	return []string{function.Name.Name}
}

func hasNestedSwapLoop(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		outer, ok := node.(*ast.ForStmt)
		if !ok {
			return !found
		}
		ast.Inspect(outer.Body, func(innerNode ast.Node) bool {
			inner, ok := innerNode.(*ast.ForStmt)
			if !ok || inner == outer {
				return !found
			}
			ast.Inspect(inner.Body, func(candidate ast.Node) bool {
				assignment, ok := candidate.(*ast.AssignStmt)
				if ok && assignmentIsSliceSwap(assignment) {
					found = true
					return false
				}
				return !found
			})
			return !found
		})
		return !found
	})
	return found
}

func assignmentIsSliceSwap(assignment *ast.AssignStmt) bool {
	if assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 2 {
		return false
	}
	leftFirst, okLeftFirst := assignment.Lhs[0].(*ast.IndexExpr)
	leftSecond, okLeftSecond := assignment.Lhs[1].(*ast.IndexExpr)
	rightFirst, okRightFirst := assignment.Rhs[0].(*ast.IndexExpr)
	rightSecond, okRightSecond := assignment.Rhs[1].(*ast.IndexExpr)
	if !okLeftFirst || !okLeftSecond || !okRightFirst || !okRightSecond {
		return false
	}
	collection := fieldTypeText(leftFirst.X)
	return collection != "" &&
		fieldTypeText(leftSecond.X) == collection &&
		fieldTypeText(rightFirst.X) == collection &&
		fieldTypeText(rightSecond.X) == collection &&
		fieldTypeText(leftFirst.Index) == fieldTypeText(rightSecond.Index) &&
		fieldTypeText(leftSecond.Index) == fieldTypeText(rightFirst.Index)
}

var supportedShapeKinds = []string{"switch_on_argv", "interface_method_set", "struct_field_set", "error_boundary", "context_duration_literal", "json_nesting", "constructs_type", "nested_swap_loop"}

func validShapeKind(kind string) bool {
	for _, supported := range supportedShapeKinds {
		if kind == supported {
			return true
		}
	}
	return false
}

func shapeMatches(node ast.Node, bypass SeamBypass, pattern *regexp.Regexp, groups map[string]int) []string {
	switch bypass.ShapeKind {
	case "switch_on_argv":
		switchNode, ok := node.(*ast.SwitchStmt)
		if !ok || switchNode.Tag == nil {
			return nil
		}
		index, ok := switchNode.Tag.(*ast.IndexExpr)
		if !ok || index.Index == nil {
			return nil
		}
		zero, ok := index.Index.(*ast.BasicLit)
		if !ok || zero.Kind != token.INT || zero.Value != "0" {
			return nil
		}
		args, ok := index.X.(*ast.Ident)
		if !ok || args.Name != "args" || !pattern.MatchString("switch_on_argv") {
			return nil
		}
		return []string{"switch_on_argv"}
	case "interface_method_set":
		typeSpec, ok := node.(*ast.TypeSpec)
		if !ok || typeSpec.Type == nil {
			return nil
		}
		interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
		if !ok {
			return nil
		}
		matched := pattern.MatchString(typeSpec.Name.Name)
		for _, field := range interfaceType.Methods.List {
			for _, name := range field.Names {
				matched = matched || pattern.MatchString(name.Name)
			}
		}
		if !matched {
			return nil
		}
		signature := interfaceSignature(interfaceType)
		if bypass.MinMembers < 1 {
			bypass.MinMembers = 2
		}
		if groups[signature] >= bypass.MinMembers {
			return []string{typeSpec.Name.Name}
		}
	case "struct_field_set":
		typeSpec, ok := node.(*ast.TypeSpec)
		if !ok || !pattern.MatchString(typeSpec.Name.Name) {
			return nil
		}
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return nil
		}
		signature := structSignature(structType)
		if bypass.MinMembers < 1 {
			bypass.MinMembers = 2
		}
		if groups[signature] >= bypass.MinMembers {
			return []string{typeSpec.Name.Name}
		}
	case "error_boundary":
		function, ok := node.(*ast.FuncDecl)
		if ok && function.Body != nil && pattern.MatchString(function.Name.Name) && functionHasErrorBoundary(function) {
			return []string{function.Name.Name}
		}
	case "context_duration_literal":
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 || !shapeContextCall(call.Fun, pattern) || !containsDurationLiteral(call.Args[1]) {
			return nil
		}
		return []string{"context deadline duration literal"}
	case "json_nesting":
		if structure, ok := node.(*ast.StructType); ok && structHasJSONNesting(structure, bypass.OuterKey, bypass.InnerKey) {
			return []string{"anonymous nested JSON decoder"}
		}
	case "constructs_type":
		function, ok := node.(*ast.FuncDecl)
		if ok && function.Body != nil && functionConstructsType(function, bypass.ConstructedType) && pattern.MatchString(function.Name.Name) {
			return []string{function.Name.Name}
		}
	default:
		return nil
	}
	return nil
}

func functionConstructsType(function *ast.FuncDecl, constructedType string) bool {
	parts := strings.Split(constructedType, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	packageName, typeName := parts[0], parts[1]
	found := false
	ast.Inspect(function.Body, func(node ast.Node) bool {
		if literal, ok := node.(*ast.CompositeLit); ok && fieldTypeText(literal.Type) == constructedType {
			found = true
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return !found
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel == nil || selector.Sel.Name != "New"+typeName {
			return !found
		}
		ident, ok := selector.X.(*ast.Ident)
		if ok && ident.Name == packageName {
			found = true
		}
		return !found
	})
	return found
}

func symbolExcluded(symbol string, exclusions []string) bool {
	for _, exclusion := range exclusions {
		if symbol == exclusion {
			return true
		}
	}
	return false
}

func shapeContextCall(node ast.Expr, pattern *regexp.Regexp) bool {
	selector, ok := node.(*ast.SelectorExpr)
	if !ok || selector.Sel == nil {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	return ok && pattern.MatchString(ident.Name+"."+selector.Sel.Name)
}

func containsDurationLiteral(node ast.Expr) bool {
	found := false
	ast.Inspect(node, func(node ast.Node) bool {
		if literal, ok := node.(*ast.BasicLit); ok && (literal.Kind == token.INT || literal.Kind == token.FLOAT) {
			found = true
			return false
		}
		return !found
	})
	return found
}

func structHasJSONNesting(structure *ast.StructType, outer, inner string) bool {
	for _, field := range structure.Fields.List {
		if jsonFieldNameFromAST(field) != outer {
			continue
		}
		nested, ok := field.Type.(*ast.StructType)
		if !ok {
			continue
		}
		for _, nestedField := range nested.Fields.List {
			if jsonFieldNameFromAST(nestedField) == inner {
				return true
			}
		}
	}
	return false
}

func jsonFieldNameFromAST(field *ast.Field) string {
	if field.Tag == nil {
		return ""
	}
	tag, err := strconv.Unquote(field.Tag.Value)
	if err != nil {
		return ""
	}
	for _, part := range strings.Fields(tag) {
		if strings.HasPrefix(part, `json:"`) {
			return strings.Split(strings.TrimSuffix(strings.TrimPrefix(part, `json:"`), `"`), ",")[0]
		}
	}
	return ""
}

func collectShapeGroups(treeRoot string, seams []compiledSeam) (map[string]map[string]int, error) {
	groups := make(map[string]map[string]int)
	for _, candidate := range seams {
		if candidate.seam.Bypass.Kind != "shape" || (candidate.seam.Bypass.ShapeKind != "interface_method_set" && candidate.seam.Bypass.ShapeKind != "struct_field_set") {
			continue
		}
		groups[candidate.seam.ID] = make(map[string]int)
	}
	if len(groups) == 0 {
		return groups, nil
	}
	err := filepath.WalkDir(treeRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != treeRoot && (entry.Name() == ".git" || entry.Name() == "node_modules" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		rel, err := filepath.Rel(treeRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		fileApplicable := make([]compiledSeam, 0)
		for _, candidate := range seams {
			if candidate.seam.Bypass.Kind == "shape" && seamPathIncluded(rel, candidate.seam.Scope) {
				fileApplicable = append(fileApplicable, candidate)
			}
		}
		if len(fileApplicable) == 0 {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", rel, err)
		}
		for _, declaration := range file.Decls {
			gen, ok := declaration.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				for _, candidate := range fileApplicable {
					var signature string
					switch candidate.seam.Bypass.ShapeKind {
					case "interface_method_set":
						if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
							signature = interfaceSignature(interfaceType)
						}
					case "struct_field_set":
						if structType, ok := typeSpec.Type.(*ast.StructType); ok {
							signature = structSignature(structType)
						}
					}
					if signature != "" && candidate.pattern.MatchString(typeSpec.Name.Name) {
						groups[candidate.seam.ID][signature]++
					}
				}
			}
		}
		return nil
	})
	return groups, err
}

func interfaceSignature(interfaceType *ast.InterfaceType) string {
	parts := make([]string, 0, len(interfaceType.Methods.List))
	for _, field := range interfaceType.Methods.List {
		for _, name := range field.Names {
			parts = append(parts, name.Name+fieldTypeText(field.Type))
		}
		if len(field.Names) == 0 {
			parts = append(parts, fieldTypeText(field.Type))
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, ";")
}

func structSignature(structType *ast.StructType) string {
	parts := make([]string, 0, len(structType.Fields.List))
	for _, field := range structType.Fields.List {
		typeText := fieldTypeText(field.Type)
		if len(field.Names) == 0 {
			parts = append(parts, typeText)
			continue
		}
		for _, name := range field.Names {
			parts = append(parts, name.Name+":"+typeText)
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, ";")
}

func fieldTypeText(node ast.Expr) string {
	if function, ok := node.(*ast.FuncType); ok {
		return functionTypeText(function)
	}
	var builder strings.Builder
	if err := printer.Fprint(&builder, token.NewFileSet(), node); err != nil {
		return ""
	}
	return builder.String()
}

func functionTypeText(function *ast.FuncType) string {
	var builder strings.Builder
	builder.WriteString("func(")
	writeFieldTypes(&builder, function.Params)
	builder.WriteByte(')')
	if function.Results != nil {
		builder.WriteByte(' ')
		if len(function.Results.List) == 1 && len(function.Results.List[0].Names) == 0 {
			builder.WriteString(fieldTypeText(function.Results.List[0].Type))
		} else {
			builder.WriteByte('(')
			writeFieldTypes(&builder, function.Results)
			builder.WriteByte(')')
		}
	}
	return builder.String()
}

func writeFieldTypes(builder *strings.Builder, fields *ast.FieldList) {
	if fields == nil {
		return
	}
	values := make([]string, 0, len(fields.List))
	for _, field := range fields.List {
		value := fieldTypeText(field.Type)
		for range field.Names {
			values = append(values, value)
		}
		if len(field.Names) == 0 {
			values = append(values, value)
		}
	}
	builder.WriteString(strings.Join(values, ","))
}

func functionHasErrorBoundary(function *ast.FuncDecl) bool {
	if function.Type.Results == nil {
		return false
	}
	hasErrorResult := false
	for _, field := range function.Type.Results.List {
		if field.Type != nil && fieldTypeText(field.Type) == "error" {
			hasErrorResult = true
		}
	}
	if !hasErrorResult {
		return false
	}
	hasErrorCall := false
	ast.Inspect(function.Body, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if ok && selector.Sel.Name == "Error" {
			hasErrorCall = true
		}
		return !hasErrorCall
	})
	return hasErrorCall
}

func scanAbsenceSeams(treeRoot string, seams []compiledSeam) ([]SeamHit, error) {
	var hits []SeamHit
	for _, candidate := range seams {
		if candidate.seam.Bypass.Kind != "absence" {
			continue
		}
		err := filepath.WalkDir(treeRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if path != treeRoot && (entry.Name() == ".git" || entry.Name() == "node_modules" || entry.Name() == "vendor") {
					return filepath.SkipDir
				}
				if path != treeRoot {
					rel, err := filepath.Rel(treeRoot, path)
					if err != nil {
						return err
					}
					if seamDirectoryExcluded(filepath.ToSlash(rel), candidate.seam.Scope) {
						return filepath.SkipDir
					}
				}
				return nil
			}
			rel, err := filepath.Rel(treeRoot, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if !seamGlob(candidate.seam.Bypass.RequireFor).MatchString(rel) {
				return nil
			}
			if !seamPathIncluded(rel, candidate.seam.Scope) {
				return nil
			}
			if candidate.seam.Bypass.ForbidKind == "executable" {
				executable, err := isExecutableArtifact(path)
				if err != nil {
					return err
				}
				if executable {
					hits = append(hits, SeamHit{SeamID: candidate.seam.ID, Canonical: candidate.seam.Canonical, Why: candidate.seam.Why, Remediation: candidate.seam.Remediation, Severity: candidate.seam.Severity, Budget: candidate.seam.Budget, Path: rel, Symbol: "executable artifact", Line: 1})
				}
				return nil
			}
			present, err := requiredTargetPresent(treeRoot, candidate.seam.Bypass.RequirePresent)
			if candidate.seam.Bypass.RequirePresent == "__json_contract_assertion__" {
				present, err = jsonContractAssertionPresent(treeRoot, rel)
			}
			if strings.ContainsAny(candidate.seam.Bypass.RequirePresent, "*?[") || (strings.HasPrefix(candidate.seam.Bypass.RequireFor, "scenarios/") && !strings.Contains(candidate.seam.Bypass.RequirePresent, "/")) {
				present, err = requiredSiblingPresent(treeRoot, rel, candidate.seam.Bypass.RequirePresent)
			}
			if err != nil {
				return err
			}
			if !present {
				position, _ := entry.Info()
				line := 0
				if position != nil {
					line = 1
				}
				hits = append(hits, SeamHit{SeamID: candidate.seam.ID, Canonical: candidate.seam.Canonical, Why: candidate.seam.Why, Remediation: candidate.seam.Remediation, Severity: candidate.seam.Severity, Budget: candidate.seam.Budget, Path: rel, Symbol: candidate.seam.Bypass.RequirePresent, Line: line})
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return hits, nil
}

func seamDirectoryExcluded(rel string, scope SeamScope) bool {
	probe := strings.TrimSuffix(rel, "/") + "/__seam_probe__"
	for _, pattern := range scope.Exclude {
		if seamGlob(pattern).MatchString(probe) {
			return true
		}
	}
	return false
}

func isExecutableArtifact(path string) (bool, error) {
	pathInfo, err := os.Lstat(path)
	if os.IsNotExist(err) || os.IsPermission(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat executable candidate %s: %w", path, err)
	}
	if !pathInfo.Mode().IsRegular() {
		return false, nil
	}
	if pathInfo.Mode().Perm()&0o111 == 0 && !strings.EqualFold(filepath.Ext(path), ".exe") {
		return false, nil
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("open executable candidate %s: %w", path, err)
	}
	defer file.Close()
	header := make([]byte, 4)
	read, err := io.ReadFull(file, header)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return false, fmt.Errorf("read executable candidate %s: %w", path, err)
	}
	if read >= 2 && bytes.Equal(header[:2], []byte{'M', 'Z'}) {
		return true, nil
	}
	if read < len(header) {
		return false, nil
	}
	switch string(header) {
	case "\x7fELF", "\xfe\xed\xfa\xce", "\xfe\xed\xfa\xcf", "\xce\xfa\xed\xfe", "\xcf\xfa\xed\xfe", "\xca\xfe\xba\xbe", "\xbe\xba\xfe\xca":
		return true, nil
	default:
		return false, nil
	}
}

// jsonContractAssertionPresent enforces the credentials JSON contract
// convention. A source function that writes JSON must be named in a contract
// test; an unrelated test file is not sufficient evidence for a new command.
func jsonContractAssertionPresent(treeRoot, rel string) (bool, error) {
	if strings.HasSuffix(rel, "_test.go") {
		return true, nil
	}
	path := filepath.Join(treeRoot, filepath.FromSlash(rel))
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return false, fmt.Errorf("parse JSON contract source %s: %w", rel, err)
	}
	var writers []string
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		writesJSON := false
		ast.Inspect(function.Body, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if ok && selector.Sel.Name == "WriteJSONValue" {
				writesJSON = true
				return false
			}
			return !writesJSON
		})
		if writesJSON {
			writers = append(writers, function.Name.Name)
		}
	}
	if len(writers) == 0 {
		return true, nil
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		return false, fmt.Errorf("read JSON contract directory for %s: %w", rel, err)
	}
	var tests string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(filepath.Dir(path), entry.Name()))
		if readErr != nil {
			return false, fmt.Errorf("read JSON contract test %s: %w", entry.Name(), readErr)
		}
		tests += string(data)
	}
	for _, writer := range writers {
		if !strings.Contains(tests, writer) {
			return false, nil
		}
	}
	return true, nil
}

func requiredSiblingPresent(treeRoot, trigger, pattern string) (bool, error) {
	matches, err := filepath.Glob(filepath.Join(treeRoot, filepath.Dir(filepath.FromSlash(trigger)), pattern))
	if err != nil {
		return false, fmt.Errorf("match absence sibling %s: %w", pattern, err)
	}
	return len(matches) > 0, nil
}

func requiredTargetPresent(treeRoot, target string) (bool, error) {
	path, pointer, _ := strings.Cut(target, "#")
	data, err := os.ReadFile(filepath.Join(treeRoot, filepath.FromSlash(path)))
	if err != nil {
		return false, nil
	}
	if pointer == "" {
		return true, nil
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return false, fmt.Errorf("decode absence target %s: %w", target, err)
	}
	if pointer == "/" {
		return true, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return false, fmt.Errorf("absence target %s has invalid JSON pointer", target)
	}
	for _, part := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		switch current := value.(type) {
		case map[string]any:
			var ok bool
			value, ok = current[part]
			if !ok {
				return false, nil
			}
		default:
			return false, nil
		}
	}
	return true, nil
}

// collectDeclarationPackages computes repetition against package identities,
// rather than directory names. This keeps a declaration seam stable when code
// moves between folders within the same Go package.
func collectDeclarationPackages(treeRoot string, seams []compiledSeam) (map[string]map[string]struct{}, error) {
	packages := make(map[string]map[string]struct{})
	err := filepath.WalkDir(treeRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != treeRoot && (entry.Name() == ".git" || entry.Name() == "node_modules" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		rel, err := filepath.Rel(treeRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		applicable := make([]compiledSeam, 0)
		for _, candidate := range seams {
			if candidate.seam.Bypass.Kind == "declaration" && seamPathIncluded(rel, candidate.seam.Scope) {
				applicable = append(applicable, candidate)
			}
		}
		if len(applicable) == 0 {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", rel, err)
		}
		constStack := make([]bool, 0, 16)
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				constStack = constStack[:len(constStack)-1]
				return true
			}
			inConst := len(constStack) > 0 && constStack[len(constStack)-1]
			if declaration, ok := node.(*ast.GenDecl); ok && declaration.Tok == token.CONST {
				inConst = true
			}
			constStack = append(constStack, inConst)
			for _, candidate := range applicable {
				for _, symbol := range declarationMatches(node, candidate.pattern, candidate.seam.Bypass.DeclKind, inConst, candidate.seam.Bypass.ExcludeAliases) {
					key := candidate.seam.ID + "\x00" + symbol
					if packages[key] == nil {
						packages[key] = make(map[string]struct{})
					}
					packages[key][file.Name.Name] = struct{}{}
					for pairIndex, pathPattern := range candidate.pairedPaths {
						match := pathPattern.FindStringSubmatch(rel)
						if len(match) == 2 {
							packages[key][fmt.Sprintf("pair:%d:%s", pairIndex, match[1])] = struct{}{}
						}
					}
				}
			}
			return true
		})
		return nil
	})
	return packages, err
}

func hasPairedPathDeclarations(locations map[string]struct{}, patterns []*regexp.Regexp, path string) bool {
	for _, pattern := range patterns {
		match := pattern.FindStringSubmatch(path)
		if len(match) != 2 {
			continue
		}
		_, left := locations["pair:0:"+match[1]]
		_, right := locations["pair:1:"+match[1]]
		return left && right
	}
	return false
}

func pairedPathSide(patterns []*regexp.Regexp, path string) int {
	for index, pattern := range patterns {
		if len(pattern.FindStringSubmatch(path)) == 2 {
			return index
		}
	}
	return -1
}

func seamNodeMatches(node ast.Node, kind string, pattern *regexp.Regexp, aliases map[string]string, inConst bool) []string {
	switch kind {
	case "call":
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return nil
		}
		name, ok := callName(call.Fun, aliases)
		if !ok || !pattern.MatchString(name) {
			return nil
		}
		return []string{name}
	case "literal":
		if literal, ok := node.(*ast.BasicLit); ok {
			value, ok := seamLiteralValue(literal)
			if ok && pattern.MatchString(value) {
				return []string{value}
			}
		}
		if binary, ok := node.(*ast.BinaryExpr); ok && binary.Op == token.MUL && !inConst {
			if qualified, ok := seamQualifiedLiteral(binary.X, binary.Y, aliases); ok && pattern.MatchString(qualified) {
				return []string{qualified}
			}
			if qualified, ok := seamQualifiedLiteral(binary.Y, binary.X, aliases); ok && pattern.MatchString(qualified) {
				return []string{qualified}
			}
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return nil
		}
		name, ok := callName(call.Fun, aliases)
		if !ok {
			return nil
		}
		var matches []string
		for _, argument := range call.Args {
			literal, ok := argument.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				continue
			}
			value, err := strconv.Unquote(literal.Value)
			if err == nil && pattern.MatchString(name+":"+value) {
				matches = append(matches, value)
			}
		}
		return matches
	case "declaration":
		return declarationMatches(node, pattern, "", inConst, false)
	default:
		return nil
	}
}

func validDeclarationKind(kind string) bool {
	switch kind {
	case "func", "method", "type", "interface", "const", "var":
		return true
	default:
		return false
	}
}

func declarationMatches(node ast.Node, pattern *regexp.Regexp, declKind string, inConst, excludeAliases bool) []string {
	want := ""
	name := ""
	switch declaration := node.(type) {
	case *ast.FuncDecl:
		name = declaration.Name.Name
		if declaration.Recv == nil {
			want = "func"
		} else {
			want = "method"
		}
	case *ast.TypeSpec:
		if excludeAliases && declaration.Assign.IsValid() {
			return nil
		}
		name = declaration.Name.Name
		want = "type"
		if _, ok := declaration.Type.(*ast.InterfaceType); ok {
			want = "interface"
		}
	case *ast.ValueSpec:
		want = "var"
		if inConst {
			want = "const"
		}
		if len(declaration.Names) == 0 {
			return nil
		}
		matches := make([]string, 0, len(declaration.Names))
		for _, identifier := range declaration.Names {
			if (declKind == "" || declKind == want) && pattern.MatchString(identifier.Name) {
				matches = append(matches, identifier.Name)
			}
		}
		return matches
	default:
		return nil
	}
	if (declKind == "" || declKind == want) && pattern.MatchString(name) {
		return []string{name}
	}
	return nil
}

func semanticNamingMatches(node ast.Node, pattern *regexp.Regexp, bypass SeamBypass, inConst bool) []string {
	declaration, ok := node.(*ast.ValueSpec)
	if !ok || (bypass.DeclKind == "const" && !inConst) || (bypass.DeclKind == "var" && inConst) {
		return nil
	}
	generic := make(map[string]struct{}, len(bypass.GenericWords))
	for _, word := range bypass.GenericWords {
		generic[strings.ToLower(strings.TrimSpace(word))] = struct{}{}
	}
	var matches []string
	for _, identifier := range declaration.Names {
		// The blank identifier discards a value; it does not name a domain concept.
		if identifier.Name == "_" {
			continue
		}
		if !pattern.MatchString(identifier.Name) {
			continue
		}
		domainWords := 0
		for _, word := range splitIdentifierWords(identifier.Name) {
			lower := strings.ToLower(word)
			_, listed := generic[lower]
			if !listed && !genericIdentifierWord(word) {
				domainWords++
			}
		}
		if domainWords < bypass.MinDomainWords {
			matches = append(matches, identifier.Name)
		}
	}
	return matches
}

func genericIdentifierWord(word string) bool {
	runes := []rune(word)
	if len(runes) == 1 {
		return true
	}
	for _, current := range runes {
		if !unicode.IsDigit(current) {
			return false
		}
	}
	return true
}

func splitIdentifierWords(identifier string) []string {
	runes := []rune(identifier)
	words := make([]string, 0, 4)
	start := 0
	flush := func(end int) {
		if end > start {
			words = append(words, string(runes[start:end]))
		}
		start = end
	}
	for index, current := range runes {
		if current == '_' {
			flush(index)
			start = index + 1
			continue
		}
		if index == start {
			continue
		}
		previous := runes[index-1]
		nextIsLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
		caseBoundary := unicode.IsUpper(current) && (unicode.IsLower(previous) || unicode.IsDigit(previous) || (unicode.IsUpper(previous) && nextIsLower))
		digitBoundary := unicode.IsDigit(current) != unicode.IsDigit(previous)
		if caseBoundary || digitBoundary {
			flush(index)
		}
	}
	flush(len(runes))
	return words
}

func seamLiteralValue(literal *ast.BasicLit) (string, bool) {
	if literal.Kind != token.STRING {
		return literal.Value, true
	}
	value, err := strconv.Unquote(literal.Value)
	return value, err == nil
}

func seamQualifiedLiteral(literalExpr, qualifierExpr ast.Expr, aliases map[string]string) (string, bool) {
	literal, ok := literalExpr.(*ast.BasicLit)
	if !ok {
		return "", false
	}
	value, ok := seamLiteralValue(literal)
	if !ok {
		return "", false
	}
	qualifier, ok := callName(qualifierExpr, aliases)
	if !ok {
		return "", false
	}
	return qualifier + ":" + value, true
}

func callName(expr ast.Expr, aliases map[string]string) (string, bool) {
	switch value := expr.(type) {
	case *ast.Ident:
		if canonical, ok := aliases[value.Name]; ok {
			return canonical, true
		}
		return value.Name, true
	case *ast.SelectorExpr:
		prefix, ok := callName(value.X, aliases)
		if !ok {
			return "", false
		}
		return prefix + "." + value.Sel.Name, true
	default:
		return "", false
	}
}

func seamImportAliases(file *ast.File) map[string]string {
	aliases := make(map[string]string, len(file.Imports))
	for _, imported := range file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			continue
		}
		name := filepath.Base(path)
		if imported.Name != nil {
			name = imported.Name.Name
		}
		if name != "_" && name != "." {
			aliases[name] = filepath.Base(path)
		}
	}
	return aliases
}

func seamDeclaresCanonical(file *ast.File, canonical string) bool {
	parts := strings.Split(canonical, ".")
	if len(parts) < 2 || file.Name.Name != parts[len(parts)-2] {
		return false
	}
	want := parts[len(parts)-1]
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil && function.Name.Name == want {
			return true
		}
	}
	return false
}

func seamPathIncluded(path string, scope SeamScope) bool {
	included := false
	for _, pattern := range scope.Include {
		if seamGlob(pattern).MatchString(path) {
			included = true
			break
		}
	}
	if !included {
		return false
	}
	for _, pattern := range scope.Exclude {
		if seamGlob(pattern).MatchString(path) {
			return false
		}
	}
	return true
}

func seamGlob(pattern string) *regexp.Regexp {
	quoted := regexp.QuoteMeta(filepath.ToSlash(strings.TrimSpace(pattern)))
	quoted = strings.ReplaceAll(quoted, `\*\*`, `.*`)
	quoted = strings.ReplaceAll(quoted, `\*`, `[^/]*`)
	quoted = strings.ReplaceAll(quoted, `\?`, `[^/]`)
	return regexp.MustCompile(`^` + quoted + `$`)
}
