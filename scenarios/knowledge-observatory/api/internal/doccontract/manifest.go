package doccontract

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	KindScenarioDocs = "scenario-docs"
	SchemaV2         = "scenario-docs-manifest/v2"
)

var identifierPart = regexp.MustCompile(`[^a-z0-9]+`)

type Manifest struct {
	Version         string         `json:"version,omitempty"`
	Title           string         `json:"title,omitempty"`
	Description     string         `json:"description,omitempty"`
	DefaultDocument string         `json:"defaultDocument,omitempty"`
	Contract        Contract       `json:"contract"`
	Sections        []Section      `json:"sections"`
	Navigation      map[string]any `json:"navigation,omitempty"`
}

type Contract struct {
	Kind           string   `json:"kind"`
	Template       string   `json:"template,omitempty"`
	Schema         string   `json:"schema"`
	MaturityValues []string `json:"maturityValues"`
	Stages         []string `json:"stages"`
	ExtensionRule  string   `json:"extensionRule,omitempty"`
}

type Section struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Icon        string     `json:"icon,omitempty"`
	Visibility  string     `json:"visibility,omitempty"`
	Description string     `json:"description,omitempty"`
	Documents   []Document `json:"documents"`
}

type Document struct {
	Path         string     `json:"path"`
	DocType      string     `json:"docType"`
	Title        string     `json:"title"`
	Aliases      []string   `json:"aliases,omitempty"`
	Description  string     `json:"description,omitempty"`
	Audience     []string   `json:"audience,omitempty"`
	CanonicalFor []string   `json:"canonicalFor,omitempty"`
	Maturity     string     `json:"maturity"`
	RequiredBy   []string   `json:"requiredBy"`
	Completion   string     `json:"completion"`
	Condition    string     `json:"condition,omitempty"`
	OwnerSkills  []string   `json:"ownerSkills,omitempty"`
	Operations   Operations `json:"operations,omitempty"`
	Validation   Validation `json:"validation,omitempty"`

	ScenarioPath string `json:"-"`
	SectionID    string `json:"-"`
}

type Operations struct {
	AppendLog *AppendLogOperation `json:"appendLog,omitempty"`
}

type AppendLogOperation struct {
	Enabled       bool               `json:"enabled"`
	TargetHeading string             `json:"targetHeading"`
	Format        string             `json:"format"`
	EmptyMarker   string             `json:"emptyMarker,omitempty"`
	Fields        []string           `json:"fields,omitempty"`
	Retention     AppendLogRetention `json:"retention"`
}

type AppendLogRetention struct {
	SupportsReset      bool   `json:"supportsReset"`
	DateSource         string `json:"dateSource"`
	KeepMinimumEntries int    `json:"keepMinimumEntries,omitempty"`
}

type Validation struct {
	RequiredHeadings      []string `json:"requiredHeadings,omitempty"`
	ForbiddenPlaceholders []string `json:"forbiddenPlaceholders,omitempty"`
	RequiredLinks         []string `json:"requiredLinks,omitempty"`
	AllowNotApplicable    bool     `json:"allowNotApplicable,omitempty"`
}

type ResolvedContract struct {
	Manifest     *Manifest
	ManifestPath string
	TemplateID   string
	Documents    []Document
	byIdentifier map[string]Document
	byPath       map[string]Document
}

type Finding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Path     string `json:"path,omitempty"`
	DocType  string `json:"doc_type,omitempty"`
	Message  string `json:"message"`
}

func LoadManifest(filePath string) (*Manifest, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func Resolve(manifest *Manifest, manifestPath string) (*ResolvedContract, []Finding) {
	if manifest == nil {
		return nil, []Finding{{Code: "manifest_missing", Severity: "error", Message: "documentation manifest is missing"}}
	}
	rc := &ResolvedContract{
		Manifest:     manifest,
		ManifestPath: filepath.ToSlash(manifestPath),
		TemplateID:   strings.TrimSpace(manifest.Contract.Template),
		byIdentifier: map[string]Document{},
		byPath:       map[string]Document{},
	}
	findings := ValidateManifest(manifest, manifestPath)
	maturity := stringSet(manifest.Contract.MaturityValues)
	stages := stringSet(manifest.Contract.Stages)
	for _, section := range manifest.Sections {
		for _, doc := range section.Documents {
			doc.SectionID = section.ID
			doc.Path = strings.TrimSpace(doc.Path)
			doc.DocType = strings.TrimSpace(doc.DocType)
			doc.Title = strings.TrimSpace(doc.Title)
			doc.ScenarioPath = NormalizeManifestPath(doc.Path)
			rc.Documents = append(rc.Documents, doc)
			if doc.ScenarioPath != "" {
				rc.byPath[doc.ScenarioPath] = doc
			}
			for _, id := range docIdentifiers(doc) {
				key := NormalizeIdentifier(id)
				if key == "" {
					continue
				}
				if existing, ok := rc.byIdentifier[key]; ok {
					if existing.ScenarioPath == doc.ScenarioPath {
						continue
					}
					findings = append(findings, Finding{
						Code:     "duplicate_identifier",
						Severity: "error",
						Path:     doc.ScenarioPath,
						DocType:  doc.DocType,
						Message:  fmt.Sprintf("identifier %q is shared by %s and %s", key, existing.DocType, doc.DocType),
					})
					continue
				}
				rc.byIdentifier[key] = doc
			}
			if doc.Maturity != "" && !maturity[doc.Maturity] {
				findings = append(findings, Finding{Code: "invalid_maturity", Severity: "error", Path: doc.ScenarioPath, DocType: doc.DocType, Message: "document maturity is not declared in contract.maturityValues"})
			}
			for _, stage := range doc.RequiredBy {
				if !stages[stage] {
					findings = append(findings, Finding{Code: "invalid_required_by", Severity: "error", Path: doc.ScenarioPath, DocType: doc.DocType, Message: fmt.Sprintf("requiredBy stage %q is not declared in contract.stages", stage)})
				}
			}
		}
	}
	sort.Slice(rc.Documents, func(i, j int) bool {
		return rc.Documents[i].ScenarioPath < rc.Documents[j].ScenarioPath
	})
	return rc, findings
}

func ValidateManifest(manifest *Manifest, manifestPath string) []Finding {
	// Run JSON Schema validation first when the schema is locatable. Schema
	// findings catch structural/shape errors authoritatively; the imperative
	// Go rules below remain as defense-in-depth for cross-document checks
	// (e.g., maturity/stage cross-references, appendLog field matrices) that
	// are awkward to express in JSON Schema.
	var findings []Finding
	findings = append(findings, validateAgainstSchema(manifestPath)...)
	if strings.TrimSpace(manifest.Contract.Kind) != KindScenarioDocs {
		findings = append(findings, Finding{Code: "invalid_contract_kind", Severity: "error", Path: filepath.ToSlash(manifestPath), Message: "contract.kind must be scenario-docs"})
	}
	if strings.TrimSpace(manifest.Contract.Schema) != SchemaV2 {
		findings = append(findings, Finding{Code: "invalid_contract_schema", Severity: "error", Path: filepath.ToSlash(manifestPath), Message: "contract.schema must be scenario-docs-manifest/v2"})
	}
	if len(manifest.Contract.MaturityValues) == 0 {
		findings = append(findings, Finding{Code: "missing_maturity_values", Severity: "error", Path: filepath.ToSlash(manifestPath), Message: "contract.maturityValues is required"})
	}
	if len(manifest.Contract.Stages) == 0 {
		findings = append(findings, Finding{Code: "missing_stages", Severity: "error", Path: filepath.ToSlash(manifestPath), Message: "contract.stages is required"})
	}
	for _, section := range manifest.Sections {
		if strings.TrimSpace(section.ID) == "" {
			findings = append(findings, Finding{Code: "missing_section_id", Severity: "error", Path: filepath.ToSlash(manifestPath), Message: "section id is required"})
		}
		if strings.TrimSpace(section.Title) == "" {
			findings = append(findings, Finding{Code: "missing_section_title", Severity: "error", Path: filepath.ToSlash(manifestPath), Message: "section title is required"})
		}
		for _, doc := range section.Documents {
			docPath := NormalizeManifestPath(doc.Path)
			switch {
			case docPath == "":
				findings = append(findings, Finding{Code: "missing_document_path", Severity: "error", Path: filepath.ToSlash(manifestPath), Message: "document path is required"})
			case strings.TrimSpace(doc.DocType) == "":
				findings = append(findings, Finding{Code: "missing_doc_type", Severity: "error", Path: docPath, Message: "document docType is required"})
			case strings.TrimSpace(doc.Title) == "":
				findings = append(findings, Finding{Code: "missing_document_title", Severity: "error", Path: docPath, DocType: doc.DocType, Message: "document title is required"})
			case strings.TrimSpace(doc.Maturity) == "":
				findings = append(findings, Finding{Code: "missing_maturity", Severity: "error", Path: docPath, DocType: doc.DocType, Message: "document maturity is required"})
			case len(doc.RequiredBy) == 0:
				findings = append(findings, Finding{Code: "missing_required_by", Severity: "error", Path: docPath, DocType: doc.DocType, Message: "document requiredBy is required"})
			case strings.TrimSpace(doc.Completion) == "":
				findings = append(findings, Finding{Code: "missing_completion", Severity: "error", Path: docPath, DocType: doc.DocType, Message: "document completion is required"})
			}
			if doc.Completion != "" && doc.Completion != "required" && doc.Completion != "conditional" && doc.Completion != "optional" {
				findings = append(findings, Finding{Code: "invalid_completion", Severity: "error", Path: docPath, DocType: doc.DocType, Message: "document completion must be required, conditional, or optional"})
			}
			if op := doc.Operations.AppendLog; op != nil && op.Enabled {
				findings = append(findings, validateAppendLog(docPath, doc.DocType, op)...)
			}
		}
	}
	return findings
}

func validateAppendLog(docPath, docType string, op *AppendLogOperation) []Finding {
	var findings []Finding
	if strings.TrimSpace(op.TargetHeading) == "" {
		findings = append(findings, Finding{Code: "append_log_missing_heading", Severity: "error", Path: docPath, DocType: docType, Message: "appendLog.targetHeading is required"})
	}
	switch op.Format {
	case "dated-markdown-section":
		if !hasField(op.Fields, "title") {
			findings = append(findings, Finding{Code: "append_log_invalid_fields", Severity: "error", Path: docPath, DocType: docType, Message: "dated-markdown-section append logs require a title field"})
		}
		if op.Retention.DateSource != "" && op.Retention.DateSource != "heading" {
			findings = append(findings, Finding{Code: "append_log_invalid_date_source", Severity: "error", Path: docPath, DocType: docType, Message: "dated-markdown-section dateSource must be heading"})
		}
	case "markdown-table":
		if len(op.Fields) == 0 || op.Fields[0] != "date" {
			findings = append(findings, Finding{Code: "append_log_invalid_fields", Severity: "error", Path: docPath, DocType: docType, Message: "markdown-table append logs require date as the first field"})
		}
		if op.Retention.DateSource != "" && op.Retention.DateSource != "first-column" {
			findings = append(findings, Finding{Code: "append_log_invalid_date_source", Severity: "error", Path: docPath, DocType: docType, Message: "markdown-table dateSource must be first-column"})
		}
	default:
		findings = append(findings, Finding{Code: "append_log_invalid_format", Severity: "error", Path: docPath, DocType: docType, Message: fmt.Sprintf("unsupported appendLog format %q", op.Format)})
	}
	return findings
}

func (c *ResolvedContract) ResolveIdentifier(value string) (Document, bool) {
	if c == nil {
		return Document{}, false
	}
	key := NormalizeIdentifier(value)
	if key == "" {
		return Document{}, false
	}
	doc, ok := c.byIdentifier[key]
	return doc, ok
}

func (c *ResolvedContract) ResolvePath(relPath string) (Document, bool) {
	if c == nil {
		return Document{}, false
	}
	relPath = NormalizeScenarioPath(relPath)
	doc, ok := c.byPath[relPath]
	return doc, ok
}

func NormalizeManifestPath(value string) string {
	value = path.Clean(filepath.ToSlash(strings.TrimSpace(value)))
	value = strings.TrimPrefix(value, "./")
	if value == "." || value == "" {
		return ""
	}
	if strings.HasPrefix(value, "../") {
		return strings.TrimPrefix(value, "../")
	}
	return path.Join("docs", value)
}

func NormalizeScenarioPath(value string) string {
	value = path.Clean(filepath.ToSlash(strings.TrimSpace(value)))
	value = strings.TrimPrefix(value, "./")
	if value == "." {
		return ""
	}
	for strings.HasPrefix(value, "scenarios/") {
		parts := strings.SplitN(value, "/", 3)
		if len(parts) < 3 {
			return value
		}
		value = parts[2]
	}
	return value
}

func NormalizeIdentifier(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimSuffix(value, ".md")
	value = strings.TrimSuffix(value, ".json")
	value = identifierPart.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}

func docIdentifiers(doc Document) []string {
	ids := []string{doc.DocType, doc.Title, path.Base(doc.ScenarioPath)}
	ids = append(ids, doc.Aliases...)
	return ids
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = true
		}
	}
	return out
}

func hasField(fields []string, target string) bool {
	for _, field := range fields {
		if field == target {
			return true
		}
	}
	return false
}

func HasError(findings []Finding) bool {
	for _, finding := range findings {
		if finding.Severity == "error" {
			return true
		}
	}
	return false
}

func ErrorFromFindings(findings []Finding) error {
	for _, finding := range findings {
		if finding.Severity == "error" {
			return errors.New(finding.Message)
		}
	}
	return nil
}
