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
	scenariosRoot    string
	staticCfg        staticConfig
	doer             Doer
	commandValidator CommandReferenceValidator
	diagramValidator DiagramValidator
}

// ServiceOption configures a Service.
type ServiceOption func(*Service)

// WithDoer overrides the HTTP client used for external-link probing. The
// default is *http.Client with a per-request timeout from staticConfig.
func WithDoer(d Doer) ServiceOption {
	return func(s *Service) { s.doer = d }
}

// WithCommandReferenceValidator overrides the CLI Health command-reference
// validator. Tests use this seam; production uses CLI Health over Connect.
func WithCommandReferenceValidator(v CommandReferenceValidator) ServiceOption {
	return func(s *Service) { s.commandValidator = v }
}

// WithDiagramValidator overrides Mermaid parsing. Tests use this seam; production
// executes the pinned Mermaid parser sidecar.
func WithDiagramValidator(v DiagramValidator) ServiceOption {
	return func(s *Service) { s.diagramValidator = v }
}

// HealthResult bundles validation results with doc counts.
type HealthResult struct {
	Validation *docvalidation.Result
	TotalDocs  int
}

// ValidateMarkdownDiagrams validates Mermaid fences in caller-provided Markdown.
func (s *Service) ValidateMarkdownDiagrams(ctx context.Context, content, source string) ([]Finding, string, bool) {
	blocks := extractMermaidBlocks(content)
	if source == "" {
		source = "markdown"
	}
	if len(blocks) == 0 {
		return nil, "", false
	}
	if s.diagramValidator == nil {
		findings := make([]Finding, 0, len(blocks))
		for _, block := range blocks {
			findings = append(findings, Finding{Code: "mermaid_unverified", Severity: SeverityWarning, Message: fmt.Sprintf("%s:%d Mermaid parser unavailable: diagram parser is not configured", source, block.startLine), Path: source, Line: block.startLine})
		}
		return findings, "", true
	}
	input := make([]DiagramBlock, len(blocks))
	for i, block := range blocks {
		input[i] = block.DiagramBlock
	}
	result, err := s.diagramValidator.ValidateDiagrams(ctx, input)
	if err != nil || len(result.Verdicts) != len(blocks) {
		reason := "diagram parser returned incomplete verdicts"
		if err != nil {
			reason = err.Error()
		}
		findings := make([]Finding, 0, len(blocks))
		for _, block := range blocks {
			findings = append(findings, Finding{Code: "mermaid_unverified", Severity: SeverityWarning, Message: fmt.Sprintf("%s:%d Mermaid parser unavailable: %s", source, block.startLine, reason), Path: source, Line: block.startLine})
		}
		return findings, "", true
	}
	var findings []Finding
	for i, v := range result.Verdicts {
		if !v.Valid {
			line := blocks[i].startLine + v.Line
			findings = append(findings, Finding{Code: "mermaid_invalid", Severity: SeverityFailure, Message: fmt.Sprintf("%s:%d %s", source, line, v.Error), Path: source, Line: line})
		}
	}
	return findings, result.Engine, false
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
	s := &Service{
		scenariosRoot:    scenariosRoot,
		staticCfg:        defaultStaticConfig(),
		commandValidator: NewCLIHealthCommandReferenceValidator(),
		diagramValidator: NewMermaidSidecarValidator(),
	}
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

// DocHealth runs the documentation-health suite for a target. With the default
// options it targets a scenario by name and runs every check — structural
// placement (delegated to docvalidation), markdown/mermaid/path content checks,
// link validation, bidirectional references, manifest coverage, and the
// derived-number lint — preserving the original behavior. opts.Scope/Path can
// instead target a project-level docs path (generic checks only), and
// opts.Checks can narrow the run. It is the single source of truth callers use
// for "doc health."
func (s *Service) DocHealth(ctx context.Context, scenarioName string, opts DocHealthOptions) (*DocHealthResult, error) {
	target, err := s.resolveTarget(scenarioName, opts)
	if err != nil {
		return nil, err
	}
	sel, err := newSelection(opts.Checks)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrScenarioNameInvalid, err)
	}

	cfg := s.staticCfg.withOptions(opts)
	result := &DocHealthResult{ScenarioName: target.label}

	// Structural validation (scenario-scoped). Skipped for generic paths.
	var validation *docvalidation.Result
	if sel.runs(checkStructure, target) {
		validation, err = docvalidation.ValidateScenarioDocumentation(target.root)
		if err != nil {
			return nil, err
		}
		result.ScenarioName = validation.ScenarioName
		result.SourceTemplateID = validation.SourceTemplateID
		result.ManifestPath = validation.ManifestPath
		result.ManifestStatus = validation.ManifestStatus
		result.HealthScore = validation.HealthScore
		result.TotalDocs = docvalidation.CountDocs(target.root)
		result.ExtraDocs = append([]string(nil), validation.ExtraDocs...)
		result.TemporaryDocs = append([]string(nil), validation.TemporaryDocs...)
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
	} else {
		// No structural pass evaluated for this target, so there is no
		// contract score to dock — report a clean structural score.
		result.HealthScore = 1.0
		result.ManifestStatus = "not-evaluated"
		result.TotalDocs = docvalidation.CountDocs(target.root)
	}

	// Markdown file set (shared by content / links / refs / numbers / manifest).
	var files []string
	if sel.needsMarkdownFiles(target) {
		files, err = collectMarkdownFiles(target.root, cfg)
		if err != nil {
			return nil, fmt.Errorf("collect markdown files: %w", err)
		}
	}

	// Per-file content + number checks.
	var linkTasks []linkTarget
	for _, file := range files {
		if sel.runs(checkContent, target) {
			findings, summary, links, ioErrs := inspectMarkdownFile(file, cfg, s.diagramValidator)
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
		if sel.runs(checkNumbers, target) {
			numFindings, flagged := scanNumbersFile(file)
			result.ContentFindings = append(result.ContentFindings, numFindings...)
			result.Counts.NumbersFlagged += flagged
		}
	}

	if sel.runs(checkLinks, target) {
		linkFindings, linkSum := validateLinks(ctx, target.root, s.doer, cfg, linkTasks)
		result.ContentFindings = append(result.ContentFindings, linkFindings...)
		result.Counts.LocalLinks += linkSum.LocalLinks
		result.Counts.ExternalLinks += linkSum.ExternalLinks
		result.Counts.BrokenLinks += linkSum.BrokenLinks
		result.Counts.ExternalWarnings += linkSum.ExternalWarnings
		result.Counts.ExternalFailures += linkSum.ExternalFailures
	}

	if sel.runs(checkRefs, target) {
		refFindings, refSum := validateBidirectionalRefs(ctx, target.root, files, cfg, s.commandValidator)
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
	}

	if sel.runs(checkCommands, target) {
		result.ReferenceFindings = append(result.ReferenceFindings, validateCommandSnippets(ctx, target.root, files, s.commandValidator)...)
	}

	if sel.runs(checkManifest, target) {
		manifestRel := cfg.manifestRel
		if validation != nil && validation.ManifestPath != "" {
			// Prefer the manifest path reported by docvalidation (already
			// repo-relative) so both checks point at the same file.
			if rel, err := filepath.Rel(target.root, filepath.Join(s.scenariosRoot, "..", validation.ManifestPath)); err == nil && !strings.HasPrefix(rel, "..") {
				manifestRel = rel
			}
		}
		mfFindings, coverage, _ := checkManifestCoverage(target.root, manifestRel, cfg.requireRegistered, files)
		result.ManifestFindings = append(result.ManifestFindings, mfFindings...)
		result.Counts.DocsInManifest = coverage.InManifest
		result.Counts.DocsNotInManifest = coverage.NotInManifest
	}

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
