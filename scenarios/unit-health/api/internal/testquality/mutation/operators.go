package mutation

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	OperatorBoundary        = "boundary"
	OperatorNegateCondition = "negate-condition"
	OperatorReturnConstant  = "return-constant"
)

// Candidate is a deterministic, source-level mutation site.
type Candidate struct {
	ID         string
	Operator   string
	File       string
	Line       int
	OwningTest string
	ordinal    int
}

// Discover returns stable mutation sites below packageRoot. Test files are
// read to assign an owning test, but are never mutated.
func Discover(packageRoot, workspaceRoot string, operators []string) ([]Candidate, error) {
	allowed := normalizeOperators(operators)
	if len(allowed) == 0 {
		return nil, fmt.Errorf("at least one supported mutation operator is required")
	}
	files := make([]string, 0)
	testsByDir := make(map[string][]string)
	err := filepath.WalkDir(packageRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != packageRoot && (entry.Name() == "vendor" || entry.Name() == "testdata" || strings.HasPrefix(entry.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || filepath.Ext(path) != ".go" {
			return nil
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			for _, name := range testNames(path) {
				dir := filepath.Dir(path)
				testsByDir[dir] = append(testsByDir[dir], name)
			}
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	for dir := range testsByDir {
		sort.Strings(testsByDir[dir])
	}

	var out []Candidate
	for _, file := range files {
		fset := token.NewFileSet()
		fileAST, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}
		occurrences := make(map[string]int)
		ast.Inspect(fileAST, func(node ast.Node) bool {
			if node == nil {
				return false
			}
			operator := operatorFor(node, allowed)
			if operator == "" {
				return true
			}
			position := fset.Position(node.Pos())
			key := fmt.Sprintf("%d:%s", position.Line, operator)
			occurrences[key]++
			relative, _ := filepath.Rel(workspaceRoot, file)
			test := owningTest(filepath.Dir(file), packageRoot, testsByDir)
			out = append(out, Candidate{ID: fmt.Sprintf("mutant-%04d", len(out)+1), Operator: operator, File: filepath.ToSlash(relative), Line: position.Line, OwningTest: test, ordinal: occurrences[key]})
			return true
		})
	}
	return out, nil
}

func normalizeOperators(input []string) map[string]bool {
	out := make(map[string]bool)
	for _, value := range input {
		switch strings.TrimSpace(strings.ToLower(value)) {
		case OperatorBoundary, "boundary-comparison":
			out[OperatorBoundary] = true
		case OperatorNegateCondition, "negate":
			out[OperatorNegateCondition] = true
		case OperatorReturnConstant, "return-bool":
			out[OperatorReturnConstant] = true
		}
	}
	return out
}

func operatorFor(node ast.Node, allowed map[string]bool) string {
	switch typed := node.(type) {
	case *ast.BinaryExpr:
		if allowed[OperatorBoundary] && (typed.Op == token.LSS || typed.Op == token.LEQ || typed.Op == token.GTR || typed.Op == token.GEQ) {
			return OperatorBoundary
		}
	case *ast.IfStmt:
		if allowed[OperatorNegateCondition] {
			if unary, ok := typed.Cond.(*ast.UnaryExpr); ok && unary.Op == token.NOT {
				return ""
			}
			return OperatorNegateCondition
		}
	case *ast.ReturnStmt:
		if allowed[OperatorReturnConstant] && len(typed.Results) == 1 {
			if ident, ok := typed.Results[0].(*ast.Ident); ok && (ident.Name == "true" || ident.Name == "false") {
				return OperatorReturnConstant
			}
		}
	}
	return ""
}

func testNames(path string) []string {
	fset := token.NewFileSet()
	fileAST, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil
	}
	var names []string
	for _, decl := range fileAST.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Test") {
			names = append(names, fn.Name.Name)
		}
	}
	return names
}

func owningTest(dir, packageRoot string, testsByDir map[string][]string) string {
	if names := testsByDir[dir]; len(names) > 0 {
		return names[0]
	}
	for current := dir; strings.HasPrefix(current, packageRoot) && current != filepath.Dir(packageRoot); current = filepath.Dir(current) {
		if names := testsByDir[current]; len(names) > 0 {
			return names[0]
		}
	}
	return ""
}

// Apply applies exactly the candidate identified by file, operator, line, and
// ordinal to a copied source file.
func Apply(path string, candidate Candidate) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	fileAST, err := parser.ParseFile(fset, path, source, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse mutant source: %w", err)
	}
	occurrences := 0
	changed := false
	ast.Inspect(fileAST, func(node ast.Node) bool {
		if node == nil || changed {
			return false
		}
		if operatorFor(node, map[string]bool{candidate.Operator: true}) != candidate.Operator {
			return true
		}
		if fset.Position(node.Pos()).Line != candidate.Line {
			return true
		}
		occurrences++
		if occurrences != candidate.ordinal {
			return true
		}
		switch typed := node.(type) {
		case *ast.BinaryExpr:
			typed.Op = boundaryMutation(typed.Op)
		case *ast.IfStmt:
			typed.Cond = &ast.UnaryExpr{Op: token.NOT, X: typed.Cond}
		case *ast.ReturnStmt:
			ident := typed.Results[0].(*ast.Ident)
			if ident.Name == "true" {
				ident.Name = "false"
			} else {
				ident.Name = "true"
			}
		}
		changed = true
		return false
	})
	if !changed {
		return fmt.Errorf("mutation site %s:%d was not found", candidate.File, candidate.Line)
	}
	var formatted bytes.Buffer
	if err := format.Node(&formatted, fset, fileAST); err != nil {
		return fmt.Errorf("format mutant source: %w", err)
	}
	return os.WriteFile(path, formatted.Bytes(), 0o644)
}

func boundaryMutation(op token.Token) token.Token {
	switch op {
	case token.LSS:
		return token.LEQ
	case token.LEQ:
		return token.LSS
	case token.GTR:
		return token.GEQ
	case token.GEQ:
		return token.GTR
	default:
		return op
	}
}

func CandidateOrder(seed string, candidates []Candidate) []Candidate {
	out := append([]Candidate(nil), candidates...)
	sort.SliceStable(out, func(i, j int) bool {
		left := sha256.Sum256([]byte(seed + "\x00" + out[i].File + fmt.Sprintf(":%d:%s:%d", out[i].Line, out[i].Operator, out[i].ordinal)))
		right := sha256.Sum256([]byte(seed + "\x00" + out[j].File + fmt.Sprintf(":%d:%s:%d", out[j].Line, out[j].Operator, out[j].ordinal)))
		return string(left[:]) < string(right[:])
	})
	return out
}
