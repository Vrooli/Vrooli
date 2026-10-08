package validation

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
)

// seamMethodSet is the comparable portion of a production seam. Signatures
// intentionally omit parameter names: interface methods and function-valued
// fields express the same seam even when their parameter names differ.
type seamMethodSet map[string]string

type seamDeclaration struct {
	Name     string
	File     string
	Line     int
	Methods  seamMethodSet
	Function bool
}

// analyzePackageArchitecture contains checks whose unit of ownership is a Go
// package rather than a scenario surface. Package validation must not inherit
// scenario-only test policy checks, but it must be able to see duplicate
// production seams in the package tree.
func analyzePackageArchitecture(scenario string, workspaces []Workspace, now string) []Finding {
	return analyzePackageArchitectureWithClosure(scenario, workspaces, now, DependencyClosure{})
}

func analyzePackageArchitectureWithClosure(scenario string, workspaces []Workspace, now string, closure DependencyClosure) []Finding {
	var findings []Finding
	for _, ws := range workspaces {
		if ws.Language != "go" {
			continue
		}
		findings = append(findings, findDuplicatedPackageSeams(scenario, ws, now)...)
		findings = append(findings, analyzeGoCompanionDeclarations(scenario, ws, now, closure)...)
	}
	return findings
}

func findDuplicatedPackageSeams(scenario string, ws Workspace, now string) []Finding {
	declarations := collectSeamDeclarations(ws.RootPath)
	if len(declarations) < 2 {
		if len(declarations) == 0 {
			return nil
		}
	}
	// schedule.Clock is the canonical owner established by Phase 5. Keeping
	// its method set here makes this rule useful against both the historical
	// api-core duplicates and any future package that recreates only a subset
	// of the shared clock seam.
	canonical := seamDeclaration{Name: "schedule.Clock", Methods: canonicalClockMethodSet()}
	all := append(append([]seamDeclaration(nil), declarations...), canonical)

	duplicates := make([]bool, len(declarations))
	for i := range declarations {
		for j := i + 1; j < len(all); j++ {
			if methodSetsSubset(declarations[i].Methods, all[j].Methods) ||
				methodSetsSubset(all[j].Methods, declarations[i].Methods) {
				duplicates[i] = true
				if j < len(declarations) {
					duplicates[j] = true
				}
			}
		}
	}

	var findings []Finding
	for i, declaration := range declarations {
		if !duplicates[i] {
			continue
		}
		shared := sharedSeamSymbol(declaration.Methods)
		findings = append(findings, Finding{
			ID:           fmt.Sprintf("%s-%s-%s", codeSeamDuplicatedInPackage, ws.ID, declaration.Name),
			Scenario:     scenario,
			WorkspaceID:  ws.ID,
			Language:     "go",
			Code:         codeSeamDuplicatedInPackage,
			Category:     "architecture",
			Severity:     codeSeverity[codeSeamDuplicatedInPackage],
			FilePath:     declaration.File,
			Symbol:       declaration.Name,
			Message:      fmt.Sprintf("Production seam %q duplicates a compatible method set elsewhere in this package target.", declaration.Name),
			Evidence:     fmt.Sprintf("%s:%d method set {%s}", relTo(ws.RootPath, declaration.File), declaration.Line, formatMethodSet(declaration.Methods)),
			Expected:     "Each production seam has one owning interface and consumers adopt that shared seam.",
			Observed:     fmt.Sprintf("duplicate-compatible method set {%s}", formatMethodSet(declaration.Methods)),
			WhyItMatters: "Duplicated seams force consumers to maintain incompatible fakes and make deterministic testing less portable across packages.",
			Remediation:  fmt.Sprintf("Adopt the shared symbol %s and remove the local seam declaration.", shared),
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

// methodSetsSubset reports whether every method in smaller exists in larger
// with the same signature. Names are part of a method set; interface names are
// deliberately ignored. This catches a Now-only seam beside a Now+NewTimer
// seam, which exact-set matching would miss.
func methodSetsSubset(smaller, larger seamMethodSet) bool {
	if len(smaller) == 0 || len(smaller) > len(larger) {
		return false
	}
	for name, signature := range smaller {
		if larger[name] != signature {
			return false
		}
	}
	return true
}

func collectSeamDeclarations(root string) []seamDeclaration {
	var declarations []seamDeclaration
	walkSourceFiles(root, func(path string) {
		if !isGoSourceFile(path) || filepath.Base(filepath.Dir(path)) == "schedule" {
			return
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil || file == nil {
			return
		}
		for _, declaration := range file.Decls {
			gen, ok := declaration.(*ast.GenDecl)
			if !ok || gen.Tok.String() != "type" {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				switch typeExpr := typeSpec.Type.(type) {
				case *ast.InterfaceType:
					methods := interfaceMethodSet(typeExpr)
					if isClockSeam(methods) {
						declarations = append(declarations, seamDeclaration{
							Name:    typeSpec.Name.Name,
							File:    path,
							Line:    fset.Position(typeSpec.Pos()).Line,
							Methods: methods,
						})
					}
				case *ast.StructType:
					for _, field := range typeExpr.Fields.List {
						fn, ok := field.Type.(*ast.FuncType)
						if !ok || !isFunctionSeam(field) {
							continue
						}
						for _, name := range field.Names {
							methodName := seamMethodName(name.Name)
							declarations = append(declarations, seamDeclaration{
								Name:     typeSpec.Name.Name + "." + name.Name,
								File:     path,
								Line:     fset.Position(field.Pos()).Line,
								Methods:  seamMethodSet{methodName: funcSignature(fn)},
								Function: true,
							})
						}
					}
				}
			}
		}
	})
	sort.SliceStable(declarations, func(i, j int) bool {
		if declarations[i].File != declarations[j].File {
			return declarations[i].File < declarations[j].File
		}
		return declarations[i].Line < declarations[j].Line
	})
	return declarations
}

func interfaceMethodSet(it *ast.InterfaceType) seamMethodSet {
	methods := seamMethodSet{}
	if it == nil || it.Methods == nil {
		return methods
	}
	for _, field := range it.Methods.List {
		if fn, ok := field.Type.(*ast.FuncType); ok {
			for _, name := range field.Names {
				methods[name.Name] = funcSignature(fn)
			}
			continue
		}
		// Preserve embedded interfaces as comparable method-set members. A
		// qualified embedded name cannot be expanded without type checking, but
		// it still must not make two otherwise identical declarations look empty.
		if len(field.Names) == 0 {
			methods["embedded:"+exprString(field.Type)] = "embedded"
		}
	}
	return methods
}

func funcSignature(fn *ast.FuncType) string {
	if fn == nil {
		return ""
	}
	var params []string
	if fn.Params != nil {
		for _, field := range fn.Params.List {
			typeText := exprString(field.Type)
			count := len(field.Names)
			if count == 0 {
				count = 1
			}
			for range count {
				params = append(params, typeText)
			}
		}
	}
	signature := "(" + strings.Join(params, ",") + ")"
	if fn.Results == nil || len(fn.Results.List) == 0 {
		return signature
	}
	var results []string
	for _, field := range fn.Results.List {
		typeText := exprString(field.Type)
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for range count {
			results = append(results, typeText)
		}
	}
	return signature + " " + strings.Join(results, ",")
}

func exprString(expr ast.Expr) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, token.NewFileSet(), expr); err != nil {
		return "?"
	}
	return strings.Join(strings.Fields(buf.String()), " ")
}

func isFunctionSeam(field *ast.Field) bool {
	if field == nil || len(field.Names) == 0 {
		return false
	}
	for _, name := range field.Names {
		if strings.EqualFold(name.Name, "sleeper") || strings.EqualFold(name.Name, "sleep") {
			return true
		}
	}
	return false
}

func isClockSeam(methods seamMethodSet) bool {
	for name := range methods {
		switch name {
		case "Now", "Sleep", "NewTimer", "NewTicker":
			return true
		}
	}
	return false
}

func seamMethodName(name string) string {
	switch strings.ToLower(name) {
	case "now":
		return "Now"
	case "sleep", "sleeper":
		return "Sleep"
	case "newtimer", "timer":
		return "NewTimer"
	case "newticker", "ticker":
		return "NewTicker"
	default:
		return ""
	}
}

func formatMethodSet(methods seamMethodSet) string {
	names := make([]string, 0, len(methods))
	for name, signature := range methods {
		names = append(names, name+signature)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func sharedSeamSymbol(methods seamMethodSet) string {
	for name := range methods {
		if name == "Now" || name == "Sleep" || name == "NewTimer" || name == "NewTicker" {
			return "api-core/schedule.Clock"
		}
	}
	return "the owning package's shared companion interface"
}

func canonicalClockMethodSet() seamMethodSet {
	return seamMethodSet{
		"Now":       "() time.Time",
		"NewTimer":  "(time.Duration) Timer",
		"NewTicker": "(time.Duration) Ticker",
		"Sleep":     "(time.Duration)",
	}
}
