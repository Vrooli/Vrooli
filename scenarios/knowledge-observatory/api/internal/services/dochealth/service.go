package dochealth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"knowledge-observatory/internal/doclogs"
	"knowledge-observatory/internal/docschema"
	"knowledge-observatory/internal/doctemplates"
	"knowledge-observatory/internal/docvalidation"
)

var (
	ErrScenarioNotFound    = errors.New("scenario not found")
	ErrScenarioNameInvalid = errors.New("scenario name is invalid")
	ErrScenarioRootInvalid = errors.New("scenario root is invalid")
)

// Service provides documentation health operations scoped to scenarios.
type Service struct {
	scenariosRoot string
	staticCfg     staticConfig
	doer          Doer
}

// ServiceOption configures a Service.
type ServiceOption func(*Service)

// WithDoer overrides the HTTP client used for external-link probing. The
// default is *http.Client with a per-request timeout from staticConfig.
func WithDoer(d Doer) ServiceOption {
	return func(s *Service) { s.doer = d }
}

// HealthResult bundles validation results with doc counts.
type HealthResult struct {
	Validation *docvalidation.Result
	TotalDocs  int
}

// NewService initializes the doc health service.
func NewService(scenariosRoot string, opts ...ServiceOption) (*Service, error) {
	scenariosRoot = strings.TrimSpace(scenariosRoot)
	if scenariosRoot == "" {
		return nil, ErrScenarioRootInvalid
	}
	info, err := os.Stat(scenariosRoot)
	if err != nil {
		return nil, ErrScenarioRootInvalid
	}
	if !info.IsDir() {
		return nil, ErrScenarioRootInvalid
	}
	s := &Service{scenariosRoot: scenariosRoot, staticCfg: defaultStaticConfig()}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// ValidateScenario runs documentation structure validation for a scenario.
func (s *Service) ValidateScenario(ctx context.Context, scenarioName string) (*HealthResult, error) {
	_ = ctx
	path, err := s.scenarioPath(scenarioName)
	if err != nil {
		return nil, err
	}
	validation, err := docvalidation.ValidateScenarioDocumentation(path)
	if err != nil {
		return nil, err
	}
	count := docvalidation.CountDocs(path)
	return &HealthResult{Validation: validation, TotalDocs: count}, nil
}

// ResetScenarioDoc applies reset rules to a known document in a scenario.
func (s *Service) ResetScenarioDoc(ctx context.Context, scenarioName string, docID string, config doclogs.ResetConfig) (*doclogs.ResetResult, string, error) {
	_ = ctx
	scenarioPath, err := s.scenarioPath(scenarioName)
	if err != nil {
		return nil, "", err
	}
	resolved, err := doctemplates.NewResolverFromScenariosRoot(s.scenariosRoot).ResolveScenario(scenarioPath)
	if err != nil {
		return nil, "", err
	}
	doc, ok := resolved.Contract.ResolveIdentifier(docID)
	if !ok || doc.Operations.AppendLog == nil || !doc.Operations.AppendLog.Enabled || !doc.Operations.AppendLog.Retention.SupportsReset {
		return nil, "", fmt.Errorf("reset is not supported for document %q", docID)
	}
	result, err := doclogs.Reset(filepath.Join(scenarioPath, filepath.FromSlash(doc.ScenarioPath)), *doc.Operations.AppendLog, config)
	return result, doc.DocType, err
}

// AuditScenario runs a comprehensive documentation audit for a scenario.
func (s *Service) AuditScenario(ctx context.Context, scenarioName string) (*docschema.AuditResult, error) {
	_ = ctx
	path, err := s.scenarioPath(scenarioName)
	if err != nil {
		return nil, err
	}
	return docschema.AuditScenarioDocumentation(path)
}

// DocHealth runs the full documentation-health suite for a scenario:
// structural placement (delegated to docvalidation), markdown/mermaid/path
// content checks, link validation, bidirectional references, and manifest
// coverage. It is the single source of truth callers use for "doc health."
func (s *Service) DocHealth(ctx context.Context, scenarioName string, opts DocHealthOptions) (*DocHealthResult, error) {
	scenarioPath, err := s.scenarioPath(scenarioName)
	if err != nil {
		return nil, err
	}

	cfg := s.staticCfg.withOptions(opts)

	// Structural validation reuses the existing docvalidation package.
	validation, err := docvalidation.ValidateScenarioDocumentation(scenarioPath)
	if err != nil {
		return nil, err
	}
	result := &DocHealthResult{
		ScenarioName:     validation.ScenarioName,
		SourceTemplateID: validation.SourceTemplateID,
		ManifestPath:     validation.ManifestPath,
		ManifestStatus:   validation.ManifestStatus,
		HealthScore:      validation.HealthScore,
		TotalDocs:        docvalidation.CountDocs(scenarioPath),
		ExtraDocs:        append([]string(nil), validation.ExtraDocs...),
		TemporaryDocs:    append([]string(nil), validation.TemporaryDocs...),
	}
	for _, m := range validation.MisplacedDocs {
		result.MisplacedDocs = append(result.MisplacedDocs, MisplacedDoc{
			ActualPath:   m.ActualPath,
			ExpectedPath: m.ExpectedPath,
			Severity:     parseLegacySeverity(m.Severity),
			DocType:      m.DocType,
		})
	}
	for _, m := range validation.MissingDocDetails {
		result.MissingDocs = append(result.MissingDocs, MissingDoc{
			DocType:    m.DocType,
			Path:       m.Path,
			Severity:   parseLegacySeverity(m.Severity),
			Completion: m.Completion,
			RequiredBy: append([]string(nil), m.RequiredBy...),
		})
	}
	for _, f := range validation.ContractFindings {
		result.ContractFindings = append(result.ContractFindings, Finding{
			Code:     codeOrDefault(f.Code, "contract_finding"),
			Severity: parseLegacySeverity(f.Severity),
			Message:  f.Message,
			Path:     f.Path,
		})
	}
	for _, issue := range validation.ContentIssues {
		result.ContentFindings = append(result.ContentFindings, Finding{
			Code:     "content_issue",
			Severity: parseLegacySeverity(issue.Severity),
			Message:  issue.Message,
			Path:     issue.Path,
			DocType:  issue.DocType,
		})
	}

	// Content checks (markdown / mermaid / abs paths / links).
	files, err := collectMarkdownFiles(scenarioPath, cfg)
	if err != nil {
		return nil, fmt.Errorf("collect markdown files: %w", err)
	}
	var linkTasks []linkTarget
	for _, file := range files {
		findings, summary, links, ioErrs := inspectMarkdownFile(file, cfg)
		result.ContentFindings = append(result.ContentFindings, findings...)
		result.Counts.FilesChecked++
		result.Counts.MermaidValidated += summary.MermaidValidated
		result.Counts.MermaidFailures += summary.MermaidFailures
		result.Counts.MarkdownWarnings += summary.MarkdownWarnings
		result.Counts.MarkdownFailures += summary.MarkdownFailures
		result.Counts.AbsoluteFailures += summary.AbsoluteFailures
		result.Counts.AbsolutePathHits += summary.AbsoluteHits
		linkTasks = append(linkTasks, links...)
		for _, msg := range ioErrs {
			result.ContentFindings = append(result.ContentFindings, Finding{
				Code:     "file_read_error",
				Severity: SeverityFailure,
				Message:  msg,
				Path:     file,
			})
		}
	}
	linkFindings, linkSum := validateLinks(ctx, scenarioPath, s.doer, cfg, linkTasks)
	result.ContentFindings = append(result.ContentFindings, linkFindings...)
	result.Counts.LocalLinks += linkSum.LocalLinks
	result.Counts.ExternalLinks += linkSum.ExternalLinks
	result.Counts.BrokenLinks += linkSum.BrokenLinks
	result.Counts.ExternalWarnings += linkSum.ExternalWarnings
	result.Counts.ExternalFailures += linkSum.ExternalFailures

	// Bidirectional references.
	refFindings, refSum := validateBidirectionalRefs(ctx, scenarioPath, files, cfg)
	result.ReferenceFindings = append(result.ReferenceFindings, refFindings...)
	result.Counts.CodeRefsFound = refSum.CodeRefsFound
	result.Counts.CodeRefsBroken = refSum.CodeRefsBroken
	result.Counts.DocRefsFound = refSum.DocRefsFound
	result.Counts.DocRefsBroken = refSum.DocRefsBroken
	result.Counts.CodeFilesScanned = refSum.CodeFilesScanned
	result.Counts.MarkedRefsFound = refSum.MarkedRefsFound
	result.Counts.MarkedRefsBroken = refSum.MarkedRefsBroken
	result.Counts.MarkedRefsSkipped = refSum.MarkedRefsSkipped
	result.Counts.MarkedRefsUnknown = refSum.MarkedRefsUnknown

	// Manifest coverage.
	manifestRel := cfg.manifestRel
	if validation.ManifestPath != "" {
		// Prefer the manifest path reported by docvalidation (already
		// repo-relative) so both checks point at the same file.
		if rel, err := filepath.Rel(scenarioPath, filepath.Join(s.scenariosRoot, "..", validation.ManifestPath)); err == nil && !strings.HasPrefix(rel, "..") {
			manifestRel = rel
		}
	}
	mfFindings, coverage, _ := checkManifestCoverage(scenarioPath, manifestRel, cfg.requireRegistered, files)
	result.ManifestFindings = append(result.ManifestFindings, mfFindings...)
	result.Counts.DocsInManifest = coverage.InManifest
	result.Counts.DocsNotInManifest = coverage.NotInManifest

	return result, nil
}

func parseLegacySeverity(s string) Severity {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "info", "notice":
		return SeverityInfo
	case "warning", "warn":
		return SeverityWarning
	case "error", "failure", "fail", "critical":
		return SeverityFailure
	default:
		return SeverityWarning
	}
}

func codeOrDefault(code, fallback string) string {
	if strings.TrimSpace(code) == "" {
		return fallback
	}
	return code
}

func (s *Service) scenarioPath(scenarioName string) (string, error) {
	name := strings.TrimSpace(scenarioName)
	if name == "" || name == "." || name == ".." {
		return "", ErrScenarioNameInvalid
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "", ErrScenarioNameInvalid
	}
	path := filepath.Join(s.scenariosRoot, name)
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", ErrScenarioNotFound
	}
	return path, nil
}
