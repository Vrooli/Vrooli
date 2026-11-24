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
	"regexp"
	"sort"
	"strconv"
	"strings"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

const maxPersistencePatternHits = 5

var (
	createTablePattern           = regexp.MustCompile(`(?i)^\s*CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(.+?)\s*(?:\(|;)?\s*$`)
	persistenceIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	sqlStatementPattern          = regexp.MustCompile(`(?is)^\s*(?:SELECT|INSERT|UPDATE|DELETE|REPLACE|WITH)\b`)
	sqlTableReferencePattern     = regexp.MustCompile(`(?i)\b(?:INSERT\s+INTO|UPDATE|FROM|JOIN)\s+([A-Za-z_][A-Za-z0-9_]*)\b`)
)

type PersistenceEvidence struct {
	Layer  string `json:"layer"`
	Path   string `json:"path"`
	Line   int    `json:"line"`
	SHA256 string `json:"sha256"`
	Basis  string `json:"basis"`
	Status string `json:"status"`
}

type PersistencePattern struct {
	ID             string              `json:"id"`
	Title          string              `json:"title"`
	Summary        string              `json:"summary"`
	Scenario       string              `json:"scenario"`
	Entity         string              `json:"entity"`
	SourcePath     string              `json:"source_path"`
	Line           int                 `json:"line"`
	SourceSHA256   string              `json:"source_sha256"`
	Classification string              `json:"classification"`
	Schema         PersistenceEvidence `json:"schema"`
	SchemaBinding  PersistenceEvidence `json:"schema_binding"`
	Migration      PersistenceEvidence `json:"migration"`
	Repository     PersistenceEvidence `json:"repository"`
	Limits         []string            `json:"limits"`
	Score          float64             `json:"score"`
}

type PersistenceUnknown struct {
	Entity string              `json:"entity"`
	Status string              `json:"status"`
	Reason string              `json:"reason"`
	Schema PersistenceEvidence `json:"schema"`
}

type PersistencePatternResponse struct {
	Query        string               `json:"query"`
	Scenario     string               `json:"scenario"`
	Scope        string               `json:"scope"`
	Results      []PersistencePattern `json:"results"`
	Unknowns     []PersistenceUnknown `json:"unknowns"`
	Degradations []string             `json:"degradations"`
}

// SearchPersistencePatterns joins static schema, migration registration, and
// SQLite repository references by the exact declared table identifier. It
// does not inspect or claim the state of a running database.
func (s *Service) SearchPersistencePatterns(_ context.Context, query, scenario string, limit int) (*PersistencePatternResponse, error) {
	query, scenario = strings.TrimSpace(query), strings.TrimSpace(scenario)
	if query == "" || !invariantScenarioPattern.MatchString(scenario) {
		return nil, fmt.Errorf("query and one scenario slug are required")
	}
	if limit <= 0 || limit > maxPersistencePatternHits {
		limit = maxPersistencePatternHits
	}
	target, err := resolveTarget(&factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: scenario})
	if err != nil {
		return nil, err
	}
	root, err := filepath.EvalSymlinks(target.GetRootPath())
	if err != nil {
		return nil, err
	}
	return scanPersistencePatterns(root, scenario, query, limit)
}

func scanPersistencePatterns(root, scenario, query string, limit int) (*PersistencePatternResponse, error) {
	if scenario != "code-facts" {
		return nil, fmt.Errorf("persistence pattern source currently supports scenario code-facts only")
	}
	base := filepath.Join(root, "api", "internal", "catalog")
	read := func(name string) ([]byte, error) { return os.ReadFile(filepath.Join(base, name)) }
	schema, err := read("schema.sql")
	if err != nil {
		return nil, err
	}
	schemaGo, err := read("schema.go")
	if err != nil {
		return nil, err
	}
	repository, err := read("sqlite.go")
	if err != nil {
		return nil, err
	}
	response := &PersistencePatternResponse{Query: query, Scenario: scenario, Scope: "scenario:code-facts; static catalog schema + exact schema.go embed/Schema binding + version-1 migration AST + SQL-literal repository references; only unquoted single-token table names are joined; no live database state or migration completion is inferred", Results: []PersistencePattern{}, Unknowns: []PersistenceUnknown{}, Degradations: []string{}}
	sh, mh, rh := sha256.Sum256(schema), sha256.Sum256(schemaGo), sha256.Sum256(repository)
	schemaHash, migrationHash, repoHash := hex.EncodeToString(sh[:]), hex.EncodeToString(mh[:]), hex.EncodeToString(rh[:])
	lines := strings.Split(string(schema), "\n")
	terms := strings.Fields(strings.ToLower(query))
	embedLine, embedOK := exactSchemaBinding(schemaGo)
	migrationLine, migrationReason := exactVersionOneSchemaRegistration(repository)
	repositoryRefs, repositoryErr := exactSQLRepositoryReferences(repository)
	declarationCounts := map[string]int{}
	for _, line := range lines {
		if match := createTablePattern.FindStringSubmatch(line); len(match) == 2 {
			entity := strings.TrimSpace(match[1])
			if persistenceIdentifierPattern.MatchString(entity) {
				declarationCounts[entity]++
			}
		}
	}
	duplicateUnknowns := map[string]bool{}
	for lineNo, line := range lines {
		match := createTablePattern.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}
		entity := strings.TrimSpace(match[1])
		searchEntity := strings.Trim(entity, "\"'`")
		if tokenScore(terms, strings.ToLower(searchEntity+" catalog sqlite migration generation repository schema")) <= 0 {
			continue
		}
		schemaEvidence := PersistenceEvidence{Layer: "schema", Path: "api/internal/catalog/schema.sql", Line: lineNo + 1, SHA256: schemaHash, Basis: "CREATE TABLE declaration", Status: "observed"}
		if !persistenceIdentifierPattern.MatchString(entity) {
			response.Unknowns = append(response.Unknowns, PersistenceUnknown{Entity: searchEntity, Status: "UNKNOWN", Reason: "unsupported_schema_identifier_form", Schema: schemaEvidence})
			continue
		}
		if declarationCounts[entity] > 1 {
			if !duplicateUnknowns[entity] {
				response.Unknowns = append(response.Unknowns, PersistenceUnknown{Entity: entity, Status: "UNKNOWN", Reason: "duplicate_schema_declaration", Schema: schemaEvidence})
				duplicateUnknowns[entity] = true
			}
			continue
		}
		score := tokenScore(terms, strings.ToLower(entity+" catalog sqlite migration generation repository schema"))
		if score <= 0 {
			continue
		}
		reason := ""
		switch {
		case !embedOK:
			reason = "schema_embed_or_Schema_binding_not_proven"
		case migrationReason != "":
			reason = migrationReason
		case repositoryErr != nil:
			reason = "repository_source_parse_unavailable"
		case repositoryRefs[entity] == 0:
			reason = "exact_repository_table_reference_not_proven"
		}
		if reason != "" {
			response.Unknowns = append(response.Unknowns, PersistenceUnknown{Entity: entity, Status: "UNKNOWN", Reason: reason, Schema: schemaEvidence})
			continue
		}
		repoLine := repositoryRefs[entity]
		response.Results = append(response.Results, PersistencePattern{ID: "code-facts:persistence:" + entity, Title: "SQLite catalog persistence: " + entity, Summary: "The schema declaration, exact embed/Schema binding, version-1 migration registration, and SQL-literal repository reference join on this exact table name.", Scenario: scenario, Entity: entity, SourcePath: "api/internal/catalog/sqlite.go", Line: repoLine, SourceSHA256: repoHash, Classification: "JOINED_STATIC_SOURCE_EVIDENCE", Score: score, Schema: schemaEvidence, SchemaBinding: PersistenceEvidence{Layer: "schema_embedding", Path: "api/internal/catalog/schema.go", Line: embedLine, SHA256: migrationHash, Basis: "exact //go:embed schema.sql variable returned by Schema()", Status: "source_binding"}, Migration: PersistenceEvidence{Layer: "migration_registration", Path: "api/internal/catalog/sqlite.go", Line: migrationLine, SHA256: repoHash, Basis: "Migrate passes Version 1 with SQL: Schema() to ApplyMigrations", Status: "registered_in_source"}, Repository: PersistenceEvidence{Layer: "repository", Path: "api/internal/catalog/sqlite.go", Line: repoLine, SHA256: repoHash, Basis: "SQL string literal contains an exact table identifier in a supported repository clause", Status: "observed"}, Limits: []string{"Migration source registration does not prove it ran against a particular database.", "Static table-name joining does not establish active deployment, data contents, or runtime transaction outcome."}})
	}
	sort.Slice(response.Results, func(i, j int) bool {
		if response.Results[i].Score != response.Results[j].Score {
			return response.Results[i].Score > response.Results[j].Score
		}
		return response.Results[i].Entity < response.Results[j].Entity
	})
	if len(response.Results) > limit {
		response.Results = response.Results[:limit]
	}
	if len(response.Unknowns) > 0 {
		response.Degradations = append(response.Degradations, "lineage_unknown: schema candidates were found but an exact embed, version-1 migration, repository identifier, or supported identifier form could not be proven")
	}
	if len(response.Results) == 0 && len(response.Unknowns) == 0 {
		response.Degradations = append(response.Degradations, "no_joined_static_pattern: no exact schema/migration/repository table join matched this query")
	}
	return response, nil
}

func exactSchemaBinding(source []byte) (int, bool) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "schema.go", source, parser.ParseComments)
	if err != nil {
		return 0, false
	}
	importsEmbed := false
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err == nil && path == "embed" {
			importsEmbed = true
			break
		}
	}
	if !importsEmbed {
		return 0, false
	}

	embedNames := []string{}
	embedLines := []int{}
	seenEmbedComment := map[token.Pos]bool{}
	for _, declaration := range file.Decls {
		gen, ok := declaration.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			comments := []*ast.CommentGroup{gen.Doc, value.Doc}
			for _, group := range comments {
				if group == nil {
					continue
				}
				for _, comment := range group.List {
					fields := strings.Fields(comment.Text)
					if len(fields) != 2 || fields[0] != "//go:embed" || fields[1] != "schema.sql" || seenEmbedComment[comment.Pos()] {
						continue
					}
					seenEmbedComment[comment.Pos()] = true
					if len(gen.Specs) != 1 || len(value.Names) != 1 || value.Type == nil || !isIdent(value.Type, "string") || len(value.Values) != 0 {
						return 0, false
					}
					embedNames = append(embedNames, value.Names[0].Name)
					embedLines = append(embedLines, fset.Position(comment.Pos()).Line)
				}
			}
		}
	}
	if len(embedNames) != 1 {
		return 0, false
	}

	var schemaFunctions []*ast.FuncDecl
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil && function.Name.Name == "Schema" {
			schemaFunctions = append(schemaFunctions, function)
		}
	}
	if len(schemaFunctions) != 1 {
		return 0, false
	}
	function := schemaFunctions[0]
	if function.Type.Params == nil || len(function.Type.Params.List) != 0 || function.Type.Results == nil || len(function.Type.Results.List) != 1 || !isIdent(function.Type.Results.List[0].Type, "string") || function.Body == nil || len(function.Body.List) != 1 {
		return 0, false
	}
	returnStatement, ok := function.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returnStatement.Results) != 1 {
		return 0, false
	}
	returned, ok := returnStatement.Results[0].(*ast.Ident)
	if !ok || returned.Name != embedNames[0] {
		return 0, false
	}
	return embedLines[0], true
}

func exactVersionOneSchemaRegistration(source []byte) (int, string) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sqlite.go", source, 0)
	if err != nil {
		return 0, "migration_source_parse_unavailable"
	}
	var lines []int
	ast.Inspect(file, func(node ast.Node) bool {
		function, ok := node.(*ast.FuncDecl)
		if !ok || function.Name.Name != "Migrate" || function.Body == nil {
			return true
		}
		ast.Inspect(function.Body, func(child ast.Node) bool {
			call, ok := child.(*ast.CallExpr)
			if !ok || !isIdent(call.Fun, "ApplyMigrations") {
				return true
			}
			for _, arg := range call.Args {
				array, ok := arg.(*ast.CompositeLit)
				if !ok || !isMigrationArray(array.Type) {
					continue
				}
				for _, element := range array.Elts {
					entry, ok := element.(*ast.CompositeLit)
					if !ok {
						continue
					}
					versionOne, schemaCall := false, false
					for _, field := range entry.Elts {
						kv, ok := field.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						key, ok := kv.Key.(*ast.Ident)
						if !ok {
							continue
						}
						switch key.Name {
						case "Version":
							lit, ok := kv.Value.(*ast.BasicLit)
							versionOne = ok && lit.Kind == token.INT && lit.Value == "1"
						case "SQL":
							value, ok := kv.Value.(*ast.CallExpr)
							schemaCall = ok && isIdent(value.Fun, "Schema") && len(value.Args) == 0
						}
					}
					if versionOne && schemaCall {
						lines = append(lines, fset.Position(entry.Pos()).Line)
					}
				}
			}
			return true
		})
		return false
	})
	switch len(lines) {
	case 0:
		return 0, "version_1_schema_migration_registration_not_proven"
	case 1:
		return lines[0], ""
	default:
		return 0, "version_1_schema_migration_registration_ambiguous"
	}
}

func exactSQLRepositoryReferences(source []byte) (map[string]int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sqlite.go", source, 0)
	if err != nil {
		return nil, err
	}
	refs := map[string]int{}
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 || !isSQLRepositoryCall(call.Fun) {
			return true
		}
		literal, ok := call.Args[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		text, err := strconv.Unquote(literal.Value)
		if err != nil {
			return true
		}
		masked := maskSQLNonCode(text)
		if !sqlStatementPattern.MatchString(masked) {
			return true
		}
		for _, loc := range sqlTableReferencePattern.FindAllStringSubmatchIndex(masked, -1) {
			entity := masked[loc[2]:loc[3]]
			if refs[entity] == 0 {
				line := fset.Position(literal.Pos()).Line
				if strings.HasPrefix(literal.Value, "`") {
					line += strings.Count(masked[:loc[0]], "\n")
				}
				refs[entity] = line
			}
		}
		return true
	})
	return refs, nil
}

func isSQLRepositoryCall(expression ast.Expr) bool {
	selector, ok := expression.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	switch selector.Sel.Name {
	case "Exec", "ExecContext", "Query", "QueryContext", "QueryRow", "QueryRowContext":
		return true
	default:
		return false
	}
}

// maskSQLNonCode preserves line/column offsets while removing SQL comments and
// quoted values/identifiers so table-like words inside them cannot form joins.
func maskSQLNonCode(source string) string {
	masked := []byte(source)
	const (
		normal = iota
		singleQuote
		doubleQuote
		backtickQuote
		bracketQuote
		lineComment
		blockComment
	)
	state := normal
	for i := 0; i < len(masked); i++ {
		current := masked[i]
		next := byte(0)
		if i+1 < len(masked) {
			next = masked[i+1]
		}
		if state == normal {
			switch {
			case current == '-' && next == '-':
				state = lineComment
				masked[i], masked[i+1] = ' ', ' '
				i++
			case current == '/' && next == '*':
				state = blockComment
				masked[i], masked[i+1] = ' ', ' '
				i++
			case current == '\'':
				state = singleQuote
				masked[i] = ' '
			case current == '"':
				state = doubleQuote
				masked[i] = ' '
			case current == '`':
				state = backtickQuote
				masked[i] = ' '
			case current == '[':
				state = bracketQuote
				masked[i] = ' '
			}
			continue
		}
		if current != '\n' && current != '\r' {
			masked[i] = ' '
		}
		switch state {
		case singleQuote:
			if current == '\'' {
				if next == '\'' {
					masked[i+1] = ' '
					i++
				} else {
					state = normal
				}
			}
		case doubleQuote:
			if current == '"' {
				if next == '"' {
					masked[i+1] = ' '
					i++
				} else {
					state = normal
				}
			}
		case backtickQuote:
			if current == '`' {
				state = normal
			}
		case bracketQuote:
			if current == ']' {
				state = normal
			}
		case lineComment:
			if current == '\n' {
				state = normal
			}
		case blockComment:
			if current == '*' && next == '/' {
				masked[i+1] = ' '
				i++
				state = normal
			}
		}
	}
	return string(masked)
}

func isIdent(expression ast.Expr, name string) bool {
	ident, ok := expression.(*ast.Ident)
	return ok && ident.Name == name
}

func isMigrationArray(expression ast.Expr) bool {
	array, ok := expression.(*ast.ArrayType)
	return ok && isIdent(array.Elt, "Migration")
}
func tokenScore(terms []string, text string) float64 {
	if len(terms) == 0 {
		return 0
	}
	hit := 0
	for _, term := range terms {
		if len(term) > 2 && strings.Contains(text, term) {
			hit++
		}
	}
	if hit == 0 {
		return 0
	}
	return float64(hit) / float64(len(terms))
}
