package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const canonicalSeamsPath = ".vrooli/canonical-seams.json"

type Seam struct {
	ID          string     `json:"id"`
	Canonical   string     `json:"canonical"`
	Why         string     `json:"why"`
	Remediation string     `json:"remediation"`
	Bypass      SeamBypass `json:"bypass"`
	Scope       SeamScope  `json:"scope"`
	Severity    string     `json:"severity"`
	Budget      int        `json:"budget"`
}

type SeamBypass struct {
	Kind    string `json:"kind"`
	Pattern string `json:"pattern"`
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
}

func LoadSeams(treeRoot string) ([]Seam, error) {
	data, err := os.ReadFile(filepath.Join(treeRoot, filepath.FromSlash(canonicalSeamsPath)))
	if os.IsNotExist(err) {
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
		if seam.ID == "" || strings.TrimSpace(seam.Canonical) == "" || strings.TrimSpace(seam.Why) == "" || strings.TrimSpace(seam.Remediation) == "" {
			return nil, fmt.Errorf("canonical seam %d is missing required text", i)
		}
		if _, duplicate := seen[seam.ID]; duplicate {
			return nil, fmt.Errorf("canonical seam id %q is duplicated", seam.ID)
		}
		seen[seam.ID] = struct{}{}
		if seam.Bypass.Kind != "call" && seam.Bypass.Kind != "literal" {
			return nil, fmt.Errorf("canonical seam %q bypass kind must be call or literal", seam.ID)
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

// seamTreeRoot keeps ordinary scenario scans local while allowing the control
// plane's bounded internal/ target to consume repository-level seam rules.
func seamTreeRoot(scanRoot string) string {
	if _, err := os.Stat(filepath.Join(scanRoot, filepath.FromSlash(canonicalSeamsPath))); err == nil {
		return scanRoot
	}
	if filepath.Base(filepath.Clean(scanRoot)) != "internal" {
		return scanRoot
	}
	parent := filepath.Dir(filepath.Clean(scanRoot))
	if _, err := os.Stat(filepath.Join(parent, filepath.FromSlash(canonicalSeamsPath))); err == nil {
		return parent
	}
	return scanRoot
}

func ScanSeams(treeRoot string, seams []Seam) ([]SeamHit, error) {
	if len(seams) == 0 {
		return []SeamHit{}, nil
	}
	type compiledSeam struct {
		seam    Seam
		pattern *regexp.Regexp
	}
	compiled := make([]compiledSeam, 0, len(seams))
	for _, seam := range seams {
		pattern, err := regexp.Compile(seam.Bypass.Pattern)
		if err != nil {
			return nil, fmt.Errorf("canonical seam %q bypass pattern: %w", seam.ID, err)
		}
		compiled = append(compiled, compiledSeam{seam: seam, pattern: pattern})
	}
	var hits []SeamHit
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
		applicable := make([]compiledSeam, 0, len(compiled))
		for _, candidate := range compiled {
			if seamPathIncluded(rel, candidate.seam.Scope) {
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
		aliases := seamImportAliases(file)
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
				if seamDeclaresCanonical(file, candidate.seam.Canonical) {
					continue
				}
				for _, symbol := range seamNodeMatches(node, candidate.seam.Bypass.Kind, candidate.pattern, aliases, inConst) {
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
	default:
		return nil
	}
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
