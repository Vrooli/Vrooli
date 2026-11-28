package validation

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"
)

// hygieneSQLInHandlers is a NEW analyzer (no scenario-auditor predecessor). It
// flags raw SQL executed directly inside a transport-handler file instead of
// behind a repository layer. Mixing SQL into the transport layer couples the
// request surface to the schema and makes the routed-pool seam harder to thread.
//
// To stay conservative (a WARNING, not a hard gate) it requires BOTH:
//   - the file lives in a transport-handler location (a `handlers/` path
//     segment, or it implements Connect/HTTP handler methods — detected via a
//     `connect.Request`/`http.ResponseWriter` signature), AND
//   - it issues raw SQL: a `<x>.Query/Exec/QueryRow(+Context)` call whose
//     first argument is a SQL string, OR a bare SQL-keyword string literal.
//
// Files that import api-core/database and only call repository methods are not
// flagged — the SQL-call/SQL-literal signal is what trips it.
//
// New code: DIRECT_SQL_IN_HANDLERS (WARNING).
type hygieneSQLInHandlers struct{}

func init() { register(&hygieneSQLInHandlers{}) }

func (hygieneSQLInHandlers) Name() string { return "hygiene-sql-in-handlers" }

func (hygieneSQLInHandlers) Applies(ac AnalyzerContext) bool { return ac.IsGo() }

func (a hygieneSQLInHandlers) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	var findings []Finding
	for _, gf := range CollectGoFiles(ac) {
		findings = append(findings, a.analyzeSource(ReadFile(gf.AbsPath), gf.RelPath)...)
	}
	return findings, nil
}

// hygieneSQLExecMethods are the *sql.DB / repository method names that run a
// query when handed a SQL string.
var hygieneSQLExecMethods = map[string]struct{}{
	"Query": {}, "QueryContext": {}, "QueryRow": {}, "QueryRowContext": {},
	"Exec": {}, "ExecContext": {},
}

func (a hygieneSQLInHandlers) analyzeSource(source, relPath string) []Finding {
	if hygieneIsExemptPath(relPath) || hygieneIsAPICorePath(relPath) {
		return nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, relPath, source, 0)
	if err != nil {
		return nil
	}
	if !hygieneIsHandlerFile(relPath, file) {
		return nil
	}

	// Find the first raw-SQL site so the finding points somewhere concrete.
	var pos token.Pos
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}
		switch node := n.(type) {
		case *ast.CallExpr:
			if hygieneIsRawSQLCall(node) {
				pos, found = node.Pos(), true
				return false
			}
		case *ast.BasicLit:
			if node.Kind == token.STRING && hygieneLooksLikeSQL(node.Value) {
				pos, found = node.Pos(), true
				return false
			}
		}
		return true
	})
	if !found {
		return nil
	}

	return []Finding{{
		Code:     "DIRECT_SQL_IN_HANDLERS",
		Severity: SeverityWarning,
		Title:    "Raw SQL in a transport handler",
		Message: "This transport-handler file issues raw SQL directly instead of calling a " +
			"repository function. Embedding SQL in the request surface couples transport to " +
			"the schema, scatters query logic across handlers, and makes the routed-pool " +
			"seam harder to thread for in-place e2e testing.",
		Location:    hygieneLoc(relPath, fset, pos),
		Remediation: "Move the SQL into a repository function in the domain's data layer and call that from the handler.",
		Analyzer:    a.Name(),
	}}
}

// hygieneIsHandlerFile reports whether relPath/file is a transport handler:
// either it lives under a handlers/ path segment, or it implements a handler
// signature (a parameter typed connect.Request[...] or http.ResponseWriter).
func hygieneIsHandlerFile(relPath string, file *ast.File) bool {
	for _, seg := range strings.Split(relPath, "/") {
		if seg == "handlers" {
			return true
		}
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Type == nil || fn.Type.Params == nil {
			continue
		}
		for _, p := range fn.Type.Params.List {
			if hygieneIsHandlerParam(p.Type) {
				return true
			}
		}
	}
	return false
}

// hygieneIsHandlerParam reports whether a parameter type signals an HTTP/Connect
// handler: http.ResponseWriter, *http.Request, or *connect.Request[...].
func hygieneIsHandlerParam(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.SelectorExpr:
		if pkg, ok := t.X.(*ast.Ident); ok {
			if pkg.Name == "http" && (t.Sel.Name == "ResponseWriter" || t.Sel.Name == "Request") {
				return true
			}
		}
	case *ast.StarExpr:
		return hygieneIsHandlerParam(t.X)
	case *ast.IndexExpr: // connect.Request[T]
		return hygieneIsConnectRequest(t.X)
	case *ast.IndexListExpr:
		return hygieneIsConnectRequest(t.X)
	}
	return false
}

func hygieneIsConnectRequest(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "connect" && (sel.Sel.Name == "Request" || sel.Sel.Name == "Response")
}

// hygieneIsRawSQLCall reports whether call is `<x>.Query/Exec/...( "SQL", ... )`
// — a query method whose first argument is a SQL-looking string literal.
func hygieneIsRawSQLCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if _, ok := hygieneSQLExecMethods[sel.Sel.Name]; !ok {
		return false
	}
	for _, arg := range call.Args {
		if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING && hygieneLooksLikeSQL(lit.Value) {
			return true
		}
	}
	return false
}

// hygieneLooksLikeSQL reports whether a Go string-literal token (still quoted)
// has a SQL statement shape. For SELECT without FROM, require a compact
// expression shape or an explicit SQL expression marker; requiring only the
// leading keyword mistakes natural-language messages such as "select an exact
// catalog page-template identity" for queries.
func hygieneLooksLikeSQL(litValue string) bool {
	s := strings.TrimSpace(strings.Trim(litValue, "`\""))
	words := hygieneSQLWords(s)
	if len(words) == 0 {
		return false
	}
	switch words[0] {
	case "SELECT":
		if hygieneHasSQLWord(words, "FROM") {
			return true
		}
		rest := strings.TrimSpace(s[len("SELECT"):])
		if rest == "" {
			return false
		}
		if strings.ContainsAny(rest[:1], "*?0123456789'\"(") {
			return true
		}
		if len(words) < 2 {
			return false
		}
		if words[1] == "CASE" || words[1] == "EXISTS" || strings.HasPrefix(words[1], "CURRENT_") {
			return true
		}
		if strings.ContainsAny(rest, "(),=<>+*/%|") || hygieneHasSQLWord(words, "AS") {
			return true
		}
		if len(words) == 2 {
			return true // e.g. SELECT id or SELECT COUNT(*)
		}
		return words[1] == "DISTINCT" && len(words) == 3
	case "INSERT":
		if len(words) < 2 {
			return false
		}
		return hygieneHasSQLWord(words, "INTO")
	case "UPDATE":
		if len(words) < 2 {
			return false
		}
		return hygieneHasSQLWord(words, "SET")
	case "DELETE":
		if len(words) < 2 {
			return false
		}
		return hygieneHasSQLWord(words, "FROM")
	default:
		return false
	}
}

func hygieneHasSQLWord(words []string, want string) bool {
	for _, word := range words[1:] {
		if word == want {
			return true
		}
	}
	return false
}

// hygieneSQLWords tokenizes identifiers while ignoring quoted SQL strings, so
// a message or value containing a word such as "from" does not create a clause.
func hygieneSQLWords(s string) []string {
	var words []string
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == '\'' || r == '"' || r == '`' {
			quote := r
			i += size
			for i < len(s) {
				next, nextSize := utf8.DecodeRuneInString(s[i:])
				i += nextSize
				if next == quote {
					if i < len(s) && rune(s[i]) == quote {
						i++
						continue
					}
					break
				}
			}
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '$' {
			start := i
			i += size
			for i < len(s) {
				next, nextSize := utf8.DecodeRuneInString(s[i:])
				if !unicode.IsLetter(next) && !unicode.IsDigit(next) && next != '_' && next != '$' {
					break
				}
				i += nextSize
			}
			words = append(words, strings.ToUpper(s[start:i]))
			continue
		}
		i += size
	}
	return words
}
