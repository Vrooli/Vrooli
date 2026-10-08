package manifestvalidation

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func entrypointFindings(manifestPath string) []Finding {
	if strings.TrimSpace(manifestPath) == "" {
		return nil
	}
	mainPath := filepath.Join(filepath.Dir(manifestPath), "main.go")
	src, err := os.ReadFile(mainPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []Finding{{
			Severity:   SeverityWarning,
			Code:       CodeCLIMainUnreadable,
			Location:   mainPath,
			Message:    fmt.Sprintf("cli/main.go could not be read: %v", err),
			Suggestion: "restore readable cli/main.go permissions and rerun cli-health",
		}}
	}
	return analyzeMainEntrypoint(mainPath, src)
}

func analyzeMainEntrypoint(path string, src []byte) []Finding {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return []Finding{{
			Severity:   SeverityWarning,
			Code:       CodeCLIMainUnreadable,
			Location:   path,
			Message:    fmt.Sprintf("cli/main.go could not be parsed: %v", err),
			Suggestion: "fix Go syntax in cli/main.go so cli-health can inspect the CLI entrypoint",
		}}
	}
	if file.Name == nil || file.Name.Name != "main" {
		return nil
	}
	mainFn := findMainFunc(file)
	if mainFn == nil || mainFn.Body == nil {
		return nil
	}

	imports := importNames(file)
	analysis := analyzeMainBody(mainFn.Body, imports)
	if len(analysis.infraCalls) == 0 && (analysis.delegatesToApp || !analysis.tooBusy()) {
		return nil
	}

	reason := "main() should only handle process-boundary concerns and delegate CLI behavior to NewApp/app.Run, cmd.Execute, or an equivalent runner"
	if len(analysis.infraCalls) > 0 {
		reason = fmt.Sprintf("main() performs infrastructure setup directly (%s) instead of delegating through the CLI application layer", strings.Join(analysis.infraCalls, ", "))
	} else if !analysis.delegatesToApp {
		reason = "main() contains multiple operations but no recognized delegation to a CLI app or command runner"
	}

	return []Finding{{
		Severity:   SeverityWarning,
		Code:       CodeCLIMainHeavy,
		Location:   fmt.Sprintf("%s:%d", path, fset.Position(mainFn.Pos()).Line),
		Message:    reason,
		Suggestion: "keep cli/main.go as a thin process boundary: build the scenario app in NewApp or a domain-owned constructor, run it from main(), and move setup/business logic behind testable functions",
	}}
}

type mainAnalysis struct {
	statementCount int
	callCount      int
	delegatesToApp bool
	infraCalls     []string
}

func (a mainAnalysis) tooBusy() bool {
	return a.statementCount > 4 || a.callCount > 8
}

func findMainFunc(file *ast.File) *ast.FuncDecl {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != "main" || fn.Recv != nil {
			continue
		}
		if fn.Type != nil && len(fn.Type.Params.List) == 0 {
			return fn
		}
	}
	return nil
}

func importNames(file *ast.File) map[string]string {
	out := map[string]string{}
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		name := filepath.Base(path)
		if spec.Name != nil && spec.Name.Name != "." && spec.Name.Name != "_" {
			name = spec.Name.Name
		}
		out[name] = path
	}
	return out
}

func analyzeMainBody(body *ast.BlockStmt, imports map[string]string) mainAnalysis {
	analysis := mainAnalysis{statementCount: len(body.List)}
	seenInfra := map[string]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		analysis.callCount++
		name := callName(call.Fun, imports)
		if isDelegationCall(name) {
			analysis.delegatesToApp = true
		}
		if isInfrastructureCall(name) && !seenInfra[name] {
			seenInfra[name] = true
			analysis.infraCalls = append(analysis.infraCalls, name)
		}
		return true
	})
	return analysis
}

func callName(expr ast.Expr, imports map[string]string) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		left, ok := e.X.(*ast.Ident)
		if !ok {
			return e.Sel.Name
		}
		if importPath, imported := imports[left.Name]; imported {
			return importPath + "." + e.Sel.Name
		}
		return left.Name + "." + e.Sel.Name
	default:
		return ""
	}
}

func isDelegationCall(name string) bool {
	switch {
	case name == "NewApp":
		return true
	case strings.HasSuffix(name, ".Run"):
		return true
	case strings.HasSuffix(name, ".Execute"):
		return true
	default:
		return false
	}
}

func isInfrastructureCall(name string) bool {
	switch name {
	case "database/sql.Open",
		"net/http.ListenAndServe",
		"net/http.ListenAndServeTLS",
		"net/http.Serve",
		"net/http.ServeTLS",
		"google.golang.org/grpc.Dial",
		"google.golang.org/grpc.NewServer":
		return true
	default:
		return false
	}
}
