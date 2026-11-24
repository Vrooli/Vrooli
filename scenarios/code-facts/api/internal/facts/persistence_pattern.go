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
	response := &PersistencePatternResponse{Query: query, Scenario: scenario, Scope: "scenario:code-facts; static catalog schema + exact schema.go embed/Schema binding + version-1 migration AST + SQLite SQL-literal repository references; unqualified single-token identifiers, including simple quoted forms, are joined case-insensitively; CTEs, qualified names, and live database state are not inferred", Results: []PersistencePattern{}, Unknowns: []PersistenceUnknown{}, Degradations: []string{}}
	sh, mh, rh := sha256.Sum256(schema), sha256.Sum256(schemaGo), sha256.Sum256(repository)
	schemaHash, migrationHash, repoHash := hex.EncodeToString(sh[:]), hex.EncodeToString(mh[:]), hex.EncodeToString(rh[:])
	lines := strings.Split(string(schema), "\n")
	terms := strings.Fields(strings.ToLower(query))
	embedLine, embedOK := exactSchemaBinding(schemaGo)
	migrationLine, migrationReason := exactVersionOneSchemaRegistration(repository)
	repositoryRefs, repositoryErr := exactSQLRepositoryReferences(repository)
	declarationCounts := map[string]int{}
	for _, line := range lines {
		if entity, ok := schemaTableIdentifier(line); ok {
			declarationCounts[entity]++
		}
	}
	duplicateUnknowns := map[string]bool{}
	for lineNo, line := range lines {
		match := createTablePattern.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}
		rawEntity := strings.TrimSpace(match[1])
		entity, supportedIdentifier := sqliteIdentifier(rawEntity)
		searchEntity := entity
		if !supportedIdentifier {
			searchEntity = strings.Trim(rawEntity, "\"'`[]")
		}
		if tokenScore(terms, strings.ToLower(searchEntity+" catalog sqlite migration generation repository schema")) <= 0 {
			continue
		}
		schemaEvidence := PersistenceEvidence{Layer: "schema", Path: "api/internal/catalog/schema.sql", Line: lineNo + 1, SHA256: schemaHash, Basis: "CREATE TABLE declaration", Status: "observed"}
		if !supportedIdentifier {
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
		for _, reference := range sqliteTableReferences(text) {
			if refs[reference.name] == 0 {
				line := fset.Position(literal.Pos()).Line + strings.Count(text[:reference.offset], "\n")
				refs[reference.name] = line
			}
		}
		return true
	})
	return refs, nil
}

type sqliteSQLToken struct {
	text   string
	kind   byte
	offset int
	quoted bool
}

type sqliteTableReference struct {
	name   string
	offset int
}

func schemaTableIdentifier(line string) (string, bool) {
	match := createTablePattern.FindStringSubmatch(line)
	if len(match) != 2 {
		return "", false
	}
	return sqliteIdentifier(strings.TrimSpace(match[1]))
}

// sqliteIdentifier accepts one conservative SQLite identifier token. Simple
// delimited names are normalized like SQLite's case-insensitive identifier
// lookup; qualified, escaped, or punctuation-bearing names remain unsupported.
func sqliteIdentifier(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if persistenceIdentifierPattern.MatchString(raw) {
		return strings.ToLower(raw), true
	}
	if len(raw) < 2 {
		return "", false
	}
	open, close := raw[0], raw[len(raw)-1]
	if !((open == '"' && close == '"') || (open == 96 && close == 96) || (open == '[' && close == ']')) {
		return "", false
	}
	inner := raw[1 : len(raw)-1]
	if strings.ContainsRune(inner, rune(close)) || !persistenceIdentifierPattern.MatchString(inner) {
		return "", false
	}
	return strings.ToLower(inner), true
}

func sqliteTableReferences(source string) []sqliteTableReference {
	tokens := lexSQLiteSQL(source)
	if len(tokens) == 0 || tokens[0].quoted || tokens[0].kind != 'w' {
		return nil
	}
	statement := strings.ToUpper(tokens[0].text)
	switch statement {
	case "SELECT", "INSERT", "UPDATE", "DELETE", "REPLACE":
	case "WITH":
		// CTE aliases can shadow real tables. Until CTE scope is modeled, fail
		// closed rather than treating an alias as a repository table reference.
		return nil
	default:
		return nil
	}
	var refs []sqliteTableReference
	for i := range tokens {
		keyword := strings.ToUpper(tokens[i].text)
		candidate := -1
		switch keyword {
		case "INSERT", "REPLACE":
			j := i + 1
			if keyword == "INSERT" && tokenWord(tokens, j, "OR") {
				j += 2 // INSERT OR <conflict-algorithm>
			}
			if tokenWord(tokens, j, "INTO") {
				candidate = j + 1
			}
		case "UPDATE":
			j := i + 1
			if tokenWord(tokens, j, "OR") {
				j += 2 // UPDATE OR <conflict-algorithm>
			}
			candidate = j
		case "FROM", "JOIN":
			candidate = i + 1
		}
		if candidate < 0 || candidate >= len(tokens) || !isSQLiteIdentifierToken(tokens[candidate]) {
			continue
		}
		// Do not collapse schema-qualified names into a same-named local table.
		if candidate+1 < len(tokens) && tokens[candidate+1].text == "." {
			continue
		}
		name, ok := sqliteIdentifier(tokens[candidate].text)
		if ok {
			refs = append(refs, sqliteTableReference{name: name, offset: tokens[candidate].offset})
		}
	}
	return refs
}

func tokenWord(tokens []sqliteSQLToken, index int, want string) bool {
	return index >= 0 && index < len(tokens) && tokens[index].kind == 'w' && !tokens[index].quoted && strings.EqualFold(tokens[index].text, want)
}

func isSQLiteIdentifierToken(token sqliteSQLToken) bool {
	return token.kind == 'w' || token.kind == 'q'
}

// lexSQLiteSQL keeps quoted identifiers as identifier tokens while skipping
// string values and SQL comments. Offsets refer to the original SQL text.
func lexSQLiteSQL(source string) []sqliteSQLToken {
	var tokens []sqliteSQLToken
	for i := 0; i < len(source); {
		c := source[i]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			i++
			continue
		}
		if c == '-' && i+1 < len(source) && source[i+1] == '-' {
			i += 2
			for i < len(source) && source[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(source) && source[i+1] == '*' {
			i += 2
			for i+1 < len(source) && !(source[i] == '*' && source[i+1] == '/') {
				i++
			}
			if i+1 < len(source) {
				i += 2
			}
			continue
		}
		if c == '\'' {
			start := i
			i++
			for i < len(source) {
				if source[i] == '\'' {
					if i+1 < len(source) && source[i+1] == '\'' {
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
			tokens = append(tokens, sqliteSQLToken{kind: 's', offset: start})
			continue
		}
		if c == '"' || c == 96 || c == '[' {
			start := i
			i++
			close := c
			if c == '[' {
				close = ']'
			}
			contentStart := i
			for i < len(source) && source[i] != close {
				i++
			}
			if i >= len(source) {
				tokens = append(tokens, sqliteSQLToken{text: source[start:], kind: '?', offset: start, quoted: true})
				break
			}
			content := source[contentStart:i]
			i++
			tokens = append(tokens, sqliteSQLToken{text: string(c) + content + string(close), kind: 'q', offset: start, quoted: true})
			continue
		}
		if isSQLWordStart(c) {
			start := i
			i++
			for i < len(source) && isSQLWordPart(source[i]) {
				i++
			}
			tokens = append(tokens, sqliteSQLToken{text: source[start:i], kind: 'w', offset: start})
			continue
		}
		tokens = append(tokens, sqliteSQLToken{text: string(c), kind: 'p', offset: i})
		i++
	}
	return tokens
}

func isSQLWordStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isSQLWordPart(c byte) bool {
	return isSQLWordStart(c) || (c >= '0' && c <= '9')
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
