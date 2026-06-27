package docvalidation

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"

	"knowledge-observatory/internal/doccontract"
	"knowledge-observatory/internal/doctemplates"
)

type MisplacedDoc struct {
	ActualPath   string `json:"actual_path"`
	ExpectedPath string `json:"expected_path"`
	DocType      string `json:"doc_type"`
	Severity     string `json:"severity"`
}

type DocContentIssue struct {
	Path     string `json:"path"`
	DocType  string `json:"doc_type"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type MissingDoc struct {
	DocType    string   `json:"doc_type"`
	Path       string   `json:"path"`
	Severity   string   `json:"severity"`
	Completion string   `json:"completion"`
	RequiredBy []string `json:"required_by"`
}

type Result struct {
	ScenarioName      string                `json:"scenario_name"`
	SourceTemplateID  string                `json:"source_template_id"`
	ManifestPath      string                `json:"manifest_path"`
	ManifestStatus    string                `json:"manifest_status"`
	ContractFindings  []doccontract.Finding `json:"contract_findings,omitempty"`
	MisplacedDocs     []MisplacedDoc        `json:"misplaced_docs"`
	MissingDocs       []string              `json:"missing_docs"`
	MissingDocDetails []MissingDoc          `json:"missing_doc_details,omitempty"`
	ExtraDocs         []string              `json:"extra_docs"`
	TemporaryDocs     []string              `json:"temporary_docs"`
	ContentIssues     []DocContentIssue     `json:"content_issues"`
	HealthScore       float64               `json:"health_score"`
}

func ValidateScenarioDocumentation(scenarioPath string) (*Result, error) {
	scenarioPath = strings.TrimSpace(scenarioPath)
	if scenarioPath == "" {
		return nil, errors.New("scenarioPath is required")
	}
	info, err := os.Stat(scenarioPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("scenarioPath must be a directory")
	}
	repoRoot := filepath.Dir(filepath.Dir(scenarioPath))
	resolved, err := doctemplates.Resolver{RepoRoot: repoRoot}.ResolveScenario(scenarioPath)
	if err != nil {
		return nil, err
	}
	return ValidateScenarioWithContract(scenarioPath, resolved), nil
}

func ValidateScenarioWithContract(scenarioPath string, resolved *doctemplates.Resolved) *Result {
	result := &Result{
		ScenarioName:     filepath.Base(scenarioPath),
		HealthScore:      1,
		ManifestStatus:   "missing",
		ContractFindings: nil,
	}
	if resolved == nil || resolved.Contract == nil {
		result.ContractFindings = append(result.ContractFindings, doccontract.Finding{Code: "contract_missing", Severity: "error", Message: "documentation contract could not be resolved"})
		result.HealthScore = 0
		return result
	}
	result.SourceTemplateID = resolved.Source.TemplateID
	result.ManifestPath = resolved.Source.ScenarioManifestPath
	if resolved.Source.ScenarioManifestUsed {
		result.ManifestStatus = "present"
	} else {
		result.ManifestStatus = "template-fallback"
		repoRoot := filepath.Dir(filepath.Dir(scenarioPath))
		manifestRel, _ := repocontract.ScenarioDocsManifestRel(repoRoot)
		result.ContractFindings = append(result.ContractFindings, doccontract.Finding{
			Code:     "scenario_manifest_missing",
			Severity: "warning",
			Path:     manifestRel,
			Message:  "scenario docs manifest is missing; using source template manifest",
		})
	}
	result.ContractFindings = append(result.ContractFindings, resolved.ContractFindings...)

	registered := map[string]doccontract.Document{}
	for _, doc := range resolved.Contract.Documents {
		registered[doc.ScenarioPath] = doc
		if fileExists(filepath.Join(scenarioPath, filepath.FromSlash(doc.ScenarioPath))) {
			result.ContentIssues = append(result.ContentIssues, validateContent(scenarioPath, doc)...)
			continue
		}
		if doc.Completion == "optional" {
			continue
		}
		result.MissingDocs = append(result.MissingDocs, doc.DocType)
		result.MissingDocDetails = append(result.MissingDocDetails, MissingDoc{
			DocType:    doc.DocType,
			Path:       doc.ScenarioPath,
			Severity:   severityFor(doc),
			Completion: doc.Completion,
			RequiredBy: append([]string{}, doc.RequiredBy...),
		})
	}

	for _, rel := range discoverDocs(scenarioPath) {
		doc, ok := registered[rel]
		if ok {
			_ = doc
			continue
		}
		if expected, ok := misplacedCandidate(rel, resolved.Contract.Documents); ok {
			result.MisplacedDocs = append(result.MisplacedDocs, MisplacedDoc{
				ActualPath:   rel,
				ExpectedPath: expected.ScenarioPath,
				DocType:      expected.DocType,
				Severity:     severityFor(expected),
			})
			continue
		}
		result.ExtraDocs = append(result.ExtraDocs, rel)
	}
	result.TemporaryDocs = findTemporaryDocs(scenarioPath)
	sortResult(result)
	result.HealthScore = computeHealthScore(result)
	return result
}

func validateContent(scenarioPath string, doc doccontract.Document) []DocContentIssue {
	abs := filepath.Join(scenarioPath, filepath.FromSlash(doc.ScenarioPath))
	data, err := os.ReadFile(abs)
	if err != nil {
		return []DocContentIssue{{Path: doc.ScenarioPath, DocType: doc.DocType, Severity: "warning", Message: "could not read doc for content validation"}}
	}
	content := string(data)
	var issues []DocContentIssue
	for _, heading := range doc.Validation.RequiredHeadings {
		if !hasMarkdownHeading(content, heading) {
			issues = append(issues, DocContentIssue{Path: doc.ScenarioPath, DocType: doc.DocType, Severity: "warning", Message: "missing heading: " + heading})
		}
	}
	for _, placeholder := range doc.Validation.ForbiddenPlaceholders {
		if placeholder != "" && strings.Contains(content, placeholder) {
			issues = append(issues, DocContentIssue{Path: doc.ScenarioPath, DocType: doc.DocType, Severity: "warning", Message: "contains forbidden placeholder: " + placeholder})
		}
	}
	for _, link := range doc.Validation.RequiredLinks {
		if link != "" && !strings.Contains(content, link) {
			issues = append(issues, DocContentIssue{Path: doc.ScenarioPath, DocType: doc.DocType, Severity: "warning", Message: "missing required link: " + link})
		}
	}
	for _, contract := range doc.Validation.TableContracts {
		issues = append(issues, validateTableContract(content, doc, contract)...)
	}
	if op := doc.Operations.AppendLog; op != nil && op.Enabled && op.TargetHeading != "" {
		if !hasMarkdownHeading(content, op.TargetHeading) {
			issues = append(issues, DocContentIssue{Path: doc.ScenarioPath, DocType: doc.DocType, Severity: "warning", Message: "missing append-log heading: " + op.TargetHeading})
		}
	}
	return issues
}

func validateTableContract(content string, doc doccontract.Document, contract doccontract.TableContract) []DocContentIssue {
	heading := strings.TrimSpace(contract.AnchorHeading)
	if heading == "" {
		return nil
	}
	table, ok := markdownTableUnderHeading(content, heading)
	if !ok {
		return []DocContentIssue{{Path: doc.ScenarioPath, DocType: doc.DocType, Severity: "error", Message: "missing table under heading: " + heading}}
	}
	if len(table) == 0 {
		return []DocContentIssue{{Path: doc.ScenarioPath, DocType: doc.DocType, Severity: "error", Message: "empty table under heading: " + heading}}
	}
	header := table[0]
	headerByName := map[string]int{}
	for i, cell := range header {
		headerByName[normalizeTableHeader(cell)] = i
	}
	var issues []DocContentIssue
	for _, col := range contract.Columns {
		name := strings.TrimSpace(col.Name)
		if name == "" {
			continue
		}
		idx, exact := headerByName[normalizeTableHeader(name)]
		if !exact {
			for _, alias := range col.Aliases {
				if aliasIdx, ok := headerByName[normalizeTableHeader(alias)]; ok {
					idx = aliasIdx
					exact = true
					issues = append(issues, DocContentIssue{
						Path:     doc.ScenarioPath,
						DocType:  doc.DocType,
						Severity: "warning",
						Message:  fmt.Sprintf("table %q uses alias header %q; canonical header is %q", heading, header[idx], name),
					})
					break
				}
			}
		}
		if !exact {
			if col.Required {
				issues = append(issues, DocContentIssue{Path: doc.ScenarioPath, DocType: doc.DocType, Severity: "error", Message: fmt.Sprintf("table %q missing required column: %s", heading, name)})
			}
			continue
		}
		issues = append(issues, validateTableColumnValues(doc, heading, table[1:], idx, col)...)
	}
	return issues
}

func validateTableColumnValues(doc doccontract.Document, heading string, rows [][]string, idx int, col doccontract.TableColumnContract) []DocContentIssue {
	if col.Type != "enum" && col.Type != "enum-list" {
		return nil
	}
	allowed := map[string]struct{}{}
	for _, value := range col.EnumValues {
		if normalized := normalizeEnumValue(value); normalized != "" {
			allowed[normalized] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		return nil
	}
	var issues []DocContentIssue
	for rowIdx, row := range rows {
		if idx >= len(row) {
			continue
		}
		for _, value := range tableCellValues(row[idx], col.Type == "enum-list") {
			if value == "" || value == "-" || value == "—" {
				continue
			}
			if _, ok := allowed[normalizeEnumValue(value)]; ok {
				continue
			}
			issues = append(issues, DocContentIssue{
				Path:     doc.ScenarioPath,
				DocType:  doc.DocType,
				Severity: "warning",
				Message:  fmt.Sprintf("table %q row %d column %q has value %q outside enum", heading, rowIdx+1, col.Name, value),
			})
		}
	}
	return issues
}

func misplacedCandidate(rel string, docs []doccontract.Document) (doccontract.Document, bool) {
	base := strings.ToLower(path.Base(rel))
	var matches []doccontract.Document
	for _, doc := range docs {
		if strings.ToLower(path.Base(doc.ScenarioPath)) == base && doc.ScenarioPath != rel {
			matches = append(matches, doc)
		}
	}
	if len(matches) == 1 {
		return matches[0], true
	}
	return doccontract.Document{}, false
}

func severityFor(doc doccontract.Document) string {
	switch doc.Completion {
	case "required":
		return "error"
	default:
		return "warning"
	}
}

func discoverDocs(scenarioPath string) []string {
	var docs []string
	if entries, err := os.ReadDir(scenarioPath); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || !isDocFile(entry.Name()) {
				continue
			}
			docs = append(docs, entry.Name())
		}
	}
	docsRoot := filepath.Join(scenarioPath, "docs")
	if info, err := os.Stat(docsRoot); err == nil && info.IsDir() {
		_ = filepath.WalkDir(docsRoot, func(filePath string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasPrefix(d.Name(), ".") || !isDocFile(d.Name()) {
				return nil
			}
			rel, err := filepath.Rel(scenarioPath, filePath)
			if err == nil {
				docs = append(docs, filepath.ToSlash(rel))
			}
			return nil
		})
	}
	return docs
}

func CountDocs(scenarioPath string) int {
	return len(discoverDocs(scenarioPath))
}

func hasMarkdownHeading(content string, heading string) bool {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimLeft(line, "#")
		if strings.TrimSpace(line) == heading {
			return true
		}
	}
	return false
}

func markdownTableUnderHeading(content, heading string) ([][]string, bool) {
	section, ok := markdownSection(content, heading)
	if !ok {
		return nil, false
	}
	var table [][]string
	seenHeader := false
	for _, raw := range strings.Split(section, "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "|") {
			if seenHeader {
				break
			}
			continue
		}
		cells := splitMarkdownTableRow(line)
		if len(cells) == 0 || isMarkdownSeparatorRow(cells) {
			continue
		}
		table = append(table, cells)
		seenHeader = true
	}
	return table, len(table) > 0
}

func markdownSection(content, heading string) (string, bool) {
	lines := strings.Split(content, "\n")
	start := -1
	startLevel := 0
	for i, line := range lines {
		level, title, ok := parseMarkdownHeading(line)
		if ok && strings.EqualFold(title, heading) {
			start = i + 1
			startLevel = level
			break
		}
	}
	if start < 0 {
		return "", false
	}
	end := len(lines)
	for i := start; i < len(lines); i++ {
		level, _, ok := parseMarkdownHeading(lines[i])
		if ok && level <= startLevel {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n"), true
}

func parseMarkdownHeading(line string) (int, string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return 0, "", false
	}
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level == len(trimmed) || trimmed[level] != ' ' {
		return 0, "", false
	}
	return level, strings.TrimSpace(trimmed[level:]), true
}

func splitMarkdownTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.TrimSpace(part))
	}
	return out
}

func isMarkdownSeparatorRow(cells []string) bool {
	for _, cell := range cells {
		c := strings.Trim(cell, " :-")
		if c != "" {
			return false
		}
	}
	return len(cells) > 0
}

func normalizeTableHeader(header string) string {
	header = strings.TrimSpace(strings.ToLower(header))
	if i := strings.Index(header, "("); i >= 0 {
		header = strings.TrimSpace(header[:i])
	}
	return strings.Join(strings.Fields(header), " ")
}

func tableCellValues(cell string, list bool) []string {
	cell = strings.TrimSpace(strings.Trim(cell, "`"))
	if cell == "" {
		return nil
	}
	separators := func(r rune) bool {
		return r == ',' || (list && r == '/')
	}
	parts := strings.FieldsFunc(cell, separators)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(strings.Trim(part, "`"))
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func normalizeEnumValue(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.Trim(value, "`")
	return strings.Join(strings.Fields(value), " ")
}

func isDocFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".md" || ext == ".json"
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

var temporaryDocBaseNames = map[string]struct{}{
	"implementation_plan.md": {},
	"implementation-plan.md": {},
	"temp.md":                {},
	"tmp.md":                 {},
	"scratch.md":             {},
	"wip.md":                 {},
	"todo.md":                {},
}

func findTemporaryDocs(scenarioPath string) []string {
	var temporary []string
	skipDirs := map[string]struct{}{
		".git": {}, ".vrooli": {}, "node_modules": {}, "vendor": {}, "dist": {}, "build": {}, "coverage": {},
	}
	_ = filepath.WalkDir(scenarioPath, func(filePath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := strings.ToLower(strings.TrimSpace(d.Name()))
			if strings.HasPrefix(name, ".") && name != ".well-known" {
				return filepath.SkipDir
			}
			if _, ok := skipDirs[name]; ok {
				return filepath.SkipDir
			}
			return nil
		}
		if !isDocFile(d.Name()) || !isTemporaryDocName(d.Name()) {
			return nil
		}
		if rel, err := filepath.Rel(scenarioPath, filePath); err == nil {
			temporary = append(temporary, filepath.ToSlash(rel))
		}
		return nil
	})
	return temporary
}

func isTemporaryDocName(name string) bool {
	candidate := strings.ToLower(strings.TrimSpace(name))
	if _, ok := temporaryDocBaseNames[candidate]; ok {
		return true
	}
	trimmed := strings.TrimSuffix(candidate, filepath.Ext(candidate))
	for _, marker := range []string{"implementation_plan", "implementation-plan", "temporary", "scratch", "-tmp", "_tmp", "-temp", "_temp", "wip"} {
		if strings.Contains(trimmed, marker) {
			return true
		}
	}
	return false
}

func sortResult(result *Result) {
	sort.Slice(result.MisplacedDocs, func(i, j int) bool { return result.MisplacedDocs[i].ActualPath < result.MisplacedDocs[j].ActualPath })
	sort.Strings(result.MissingDocs)
	sort.Slice(result.MissingDocDetails, func(i, j int) bool { return result.MissingDocDetails[i].DocType < result.MissingDocDetails[j].DocType })
	sort.Strings(result.ExtraDocs)
	sort.Strings(result.TemporaryDocs)
	sort.Slice(result.ContentIssues, func(i, j int) bool {
		if result.ContentIssues[i].Path != result.ContentIssues[j].Path {
			return result.ContentIssues[i].Path < result.ContentIssues[j].Path
		}
		return result.ContentIssues[i].Message < result.ContentIssues[j].Message
	})
}

func computeHealthScore(result *Result) float64 {
	score := 1.0
	for _, finding := range result.ContractFindings {
		if finding.Severity == "error" {
			score -= 0.15
		} else {
			score -= 0.03
		}
	}
	score -= float64(len(result.MissingDocs)) * 0.08
	score -= float64(len(result.MisplacedDocs)) * 0.05
	score -= float64(len(result.ExtraDocs)) * 0.02
	score -= float64(len(result.TemporaryDocs)) * 0.01
	score -= float64(len(result.ContentIssues)) * 0.02
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

func FindingMessage(path, message string) string {
	if path == "" {
		return message
	}
	return fmt.Sprintf("%s: %s", path, message)
}
