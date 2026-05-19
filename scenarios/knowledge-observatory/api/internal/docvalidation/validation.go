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
	if op := doc.Operations.AppendLog; op != nil && op.Enabled && op.TargetHeading != "" {
		if !hasMarkdownHeading(content, op.TargetHeading) {
			issues = append(issues, DocContentIssue{Path: doc.ScenarioPath, DocType: doc.DocType, Severity: "warning", Message: "missing append-log heading: " + op.TargetHeading})
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
