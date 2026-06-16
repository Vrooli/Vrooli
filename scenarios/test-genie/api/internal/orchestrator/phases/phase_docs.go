// DOC: docs/phases/docs/README.md#summary-metrics
package phases

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"

	"github.com/vrooli/api-core/discovery"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	kov1 "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1"
	kov1connect "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1/knowledgeobservatoryv1connect"
)

// DocsSummary is the metric rollup persisted to phase pointers. It
// mirrors knowledge-observatory's DocHealthCounts so reporting code
// keeps the same field shape after the migration off in-process
// validators. Field names match the legacy `docs.Summary` shape so
// downstream pointer/json consumers do not break.
type DocsSummary struct {
	FilesChecked     int `json:"filesChecked"`
	ExternalLinks    int `json:"externalLinks"`
	LocalLinks       int `json:"localLinks"`
	BrokenLinks      int `json:"brokenLinks"`
	MermaidValidated int `json:"mermaidValidated"`
	AbsolutePathHits int `json:"absolutePathHits"`
	MarkdownWarnings int `json:"markdownWarnings"`
	MarkdownFailures int `json:"markdownFailures"`
	ExternalWarnings int `json:"externalWarnings"`
	ExternalFailures int `json:"externalFailures"`
	MermaidFailures  int `json:"mermaidFailures"`
	AbsoluteFailures int `json:"absoluteFailures"`

	CodeRefsFound     int `json:"codeRefsFound"`
	CodeRefsBroken    int `json:"codeRefsBroken"`
	DocRefsFound      int `json:"docRefsFound"`
	DocRefsBroken     int `json:"docRefsBroken"`
	CodeFilesScanned  int `json:"codeFilesScanned"`
	MarkedRefsFound   int `json:"markedRefsFound"`
	MarkedRefsBroken  int `json:"markedRefsBroken"`
	MarkedRefsSkipped int `json:"markedRefsSkipped"`
	MarkedRefsUnknown int `json:"markedRefsUnknown"`

	DocsInManifest    int    `json:"docsInManifest"`
	DocsNotInManifest int    `json:"docsNotInManifest"`
	LocalCurrentLevel string `json:"local_current_level,omitempty"`
	LocalNextLevel    string `json:"local_next_level,omitempty"`
}

// String returns a short human-readable summary.
func (s DocsSummary) String() string {
	base := fmt.Sprintf("%d files, %d broken links, %d mermaid errors, %d markdown errors",
		s.FilesChecked, s.BrokenLinks, s.MermaidFailures, s.MarkdownFailures)
	if s.CodeRefsFound > 0 || s.DocRefsFound > 0 || s.MarkedRefsFound > 0 {
		base += fmt.Sprintf(", code refs: %d found/%d broken, doc refs: %d found/%d broken, marked refs: %d found/%d broken",
			s.CodeRefsFound, s.CodeRefsBroken, s.DocRefsFound, s.DocRefsBroken, s.MarkedRefsFound, s.MarkedRefsBroken)
	}
	if s.LocalCurrentLevel != "" || s.LocalNextLevel != "" {
		base += fmt.Sprintf(", local=%s next=%s", s.LocalCurrentLevel, s.LocalNextLevel)
	}
	return base
}

// docsRunResult mirrors the legacy `docs.RunResult` shape so RunPhase /
// ExtractWithSummary continue to work unchanged.
type docsRunResult struct {
	shared.RunResult[DocsSummary]
}

var (
	// Seam for tests: resolveDocHealthBaseURL returns the
	// knowledge-observatory base URL. discovery is the only production
	// path; tests can override.
	resolveDocHealthBaseURL = func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, "knowledge-observatory")
	}
	// Seam for tests: docHealthHTTPClient lets tests substitute a fake
	// connect.HTTPClient instead of dialing the real service.
	docHealthHTTPClient connect.HTTPClient = &http.Client{Timeout: 60 * time.Second}
)

// runDocsPhase calls knowledge-observatory's DocHealth RPC and translates
// the response into Observations + Summary consumed by phase reporting.
// There is no fallback: when knowledge-observatory is unreachable, the
// phase fails fast (Greenfield — single source of truth).
func runDocsPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if os.Getenv("TEST_GENIE_SKIP_DOCS") == "1" {
		return RunReport{
			Observations: []Observation{
				NewSkipObservation("docs phase disabled via TEST_GENIE_SKIP_DOCS"),
			},
		}
	}
	var summary DocsSummary
	var archFindings []*architecturev1.ArchitectureFinding

	report := RunPhase(ctx, logWriter, "docs",
		func() (*docsRunResult, error) {
			baseURL, err := resolveDocHealthBaseURL(ctx)
			if err != nil {
				return nil, fmt.Errorf("resolve knowledge-observatory URL: %w", err)
			}
			if strings.TrimSpace(baseURL) == "" {
				return nil, errors.New("knowledge-observatory base URL is empty — start the scenario via 'vrooli scenario start knowledge-observatory'")
			}
			client := kov1connect.NewKnowledgeObservatoryServiceClient(docHealthHTTPClient, baseURL)
			resp, err := client.DocHealth(ctx, connect.NewRequest(&kov1.DocHealthRequest{
				ScenarioName: env.ScenarioName,
			}))
			if err != nil {
				return &docsRunResult{
					RunResult: shared.RunResult[DocsSummary]{
						Success:      false,
						Error:        fmt.Errorf("knowledge-observatory DocHealth RPC failed: %w", err),
						FailureClass: shared.FailureClassMissingDependency,
						Remediation:  "Ensure knowledge-observatory is running ('vrooli scenario start knowledge-observatory') and reachable.",
					},
				}, nil
			}
			result := translateDocHealth(resp.Msg)
			archFindings = docsArchFindings(env.ScenarioName, resp.Msg)
			return result, nil
		},
		func(r *docsRunResult) PhaseResult[shared.Observation] {
			var result shared.RunResult[DocsSummary]
			summaryText := ""
			if r != nil {
				result = r.RunResult
				summary = r.Summary
				summaryText = r.Summary.String()
			}
			return ExtractWithSummary(
				result.Success,
				result.Error,
				result.FailureClass,
				result.Remediation,
				result.Observations,
				"📄",
				fmt.Sprintf("Docs validation completed (%s)", summaryText),
			)
		},
	)

	report.Findings = archFindings
	writePhasePointer(env, "docs", report, map[string]any{"summary": summary}, logWriter)
	logPhaseStep(logWriter, "Docs summary: %s (abs_hits=%d, abs_blocked=%d)", summary.String(), summary.AbsolutePathHits, summary.AbsoluteFailures)
	return report
}

// docSeverityToken maps a knowledge-observatory severity into the string
// vocabulary normalizeSeverity understands.
func docSeverityToken(s kov1.DocHealthSeverity) string {
	switch s {
	case kov1.DocHealthSeverity_DOC_HEALTH_SEVERITY_FAILURE:
		return "error"
	case kov1.DocHealthSeverity_DOC_HEALTH_SEVERITY_WARNING:
		return "warning"
	default:
		return "info"
	}
}

// docLocation renders a doc finding's path (+line) into a single location.
func docLocation(path string, line int32) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if line > 0 {
		return fmt.Sprintf("%s:%d", path, line)
	}
	return path
}

// docsArchFindings maps the DocHealth response into the shared
// ArchitectureFinding contract (source=DOCS). Findings across all families
// (contract/content/reference/manifest) plus structural misplaced/missing
// docs all normalize here; the code is "<family>/<code>" for findings,
// "misplaced_doc"/"missing_doc" for the structural arrays.
func docsArchFindings(scenario string, resp *kov1.DocHealthResponse) []*architecturev1.ArchitectureFinding {
	if resp == nil {
		return nil
	}
	var out []*architecturev1.ArchitectureFinding

	add := func(family string, findings []*kov1.DocHealthFinding) {
		for _, f := range findings {
			if f == nil {
				continue
			}
			code := strings.TrimSpace(f.GetCode())
			if code == "" {
				code = family
			} else {
				code = family + "/" + code
			}
			out = append(out, newFinding(
				scenario,
				architecturev1.FindingSource_FINDING_SOURCE_DOCS,
				code, docSeverityToken(f.GetSeverity()), f.GetMessage(), "",
				nonEmptyLocations(docLocation(f.GetPath(), f.GetLine())), nil,
			))
		}
	}

	for _, m := range resp.GetMisplacedDocs() {
		if m == nil {
			continue
		}
		out = append(out, newFinding(
			scenario,
			architecturev1.FindingSource_FINDING_SOURCE_DOCS,
			"misplaced_doc", docSeverityToken(m.GetSeverity()),
			fmt.Sprintf("misplaced: %s → %s", m.GetActualPath(), m.GetExpectedPath()),
			fmt.Sprintf("move to %s", m.GetExpectedPath()),
			nonEmptyLocations(m.GetActualPath()), nil,
		))
	}
	for _, m := range resp.GetMissingDocs() {
		if m == nil {
			continue
		}
		out = append(out, newFinding(
			scenario,
			architecturev1.FindingSource_FINDING_SOURCE_DOCS,
			"missing_doc", docSeverityToken(m.GetSeverity()),
			fmt.Sprintf("missing doc: %s (%s)", m.GetDocType(), m.GetPath()), "",
			nonEmptyLocations(m.GetPath()), nil,
		))
	}

	add("contract", resp.GetContractFindings())
	add("content", resp.GetContentFindings())
	add("reference", resp.GetReferenceFindings())
	add("manifest", resp.GetManifestFindings())

	return out
}

// translateDocHealth converts the proto response into the test-genie
// RunResult shape (success/error/observations/summary) consumed by
// RunPhase. Severity FAILURE → error observation + Success=false;
// WARNING → warning; INFO → info.
func translateDocHealth(resp *kov1.DocHealthResponse) *docsRunResult {
	out := &docsRunResult{}
	out.Summary = countsToSummary(resp.GetCounts())
	if err := requireProtoProviderAssessment("knowledge-observatory", "docs", resp.GetAssessment()); err != nil {
		out.Success = false
		out.FailureClass = shared.FailureClassMaturityContract
		out.Error = err
		out.Remediation = "Run 'test-genie provider-contract check docs " + resp.GetScenarioName() + " --json' and restart knowledge-observatory if stale."
		return out
	}
	out.Summary.LocalCurrentLevel, out.Summary.LocalNextLevel = localMaturitySummary(resp.GetAssessment())

	failureCount := 0

	appendFindings := func(family string, findings []*kov1.DocHealthFinding) {
		for _, f := range findings {
			if f == nil {
				continue
			}
			msg := formatFinding(family, f)
			switch f.GetSeverity() {
			case kov1.DocHealthSeverity_DOC_HEALTH_SEVERITY_FAILURE:
				out.Observations = append(out.Observations, shared.NewErrorObservation(msg))
				failureCount++
			case kov1.DocHealthSeverity_DOC_HEALTH_SEVERITY_WARNING:
				out.Observations = append(out.Observations, shared.NewWarningObservation(msg))
			default:
				out.Observations = append(out.Observations, shared.NewInfoObservation(msg))
			}
		}
	}

	// Structural sections come back as misplaced/missing arrays separate
	// from findings.
	for _, m := range resp.GetMisplacedDocs() {
		if m == nil {
			continue
		}
		msg := fmt.Sprintf("misplaced: %s → %s", m.GetActualPath(), m.GetExpectedPath())
		switch m.GetSeverity() {
		case kov1.DocHealthSeverity_DOC_HEALTH_SEVERITY_FAILURE:
			out.Observations = append(out.Observations, shared.NewErrorObservation(msg))
			failureCount++
		default:
			out.Observations = append(out.Observations, shared.NewWarningObservation(msg))
		}
	}
	for _, m := range resp.GetMissingDocs() {
		if m == nil {
			continue
		}
		msg := fmt.Sprintf("missing doc: %s (%s)", m.GetDocType(), m.GetPath())
		switch m.GetSeverity() {
		case kov1.DocHealthSeverity_DOC_HEALTH_SEVERITY_FAILURE:
			out.Observations = append(out.Observations, shared.NewErrorObservation(msg))
			failureCount++
		default:
			out.Observations = append(out.Observations, shared.NewWarningObservation(msg))
		}
	}

	appendFindings("contract", resp.GetContractFindings())
	appendFindings("content", resp.GetContentFindings())
	appendFindings("reference", resp.GetReferenceFindings())
	appendFindings("manifest", resp.GetManifestFindings())

	if failureCount > 0 {
		out.Success = false
		out.FailureClass = shared.FailureClassTestFailure
		out.Error = fmt.Errorf("%d documentation finding(s) at FAILURE severity", failureCount)
		out.Remediation = "Run 'knowledge-observatory docs health " + resp.GetScenarioName() + "' for details."
	} else {
		out.Success = true
	}
	return out
}

func formatFinding(family string, f *kov1.DocHealthFinding) string {
	code := strings.TrimSpace(f.GetCode())
	parts := []string{family}
	if code != "" {
		parts = append(parts, code)
	}
	prefix := strings.Join(parts, "/")
	location := ""
	if path := strings.TrimSpace(f.GetPath()); path != "" {
		if line := f.GetLine(); line > 0 {
			location = fmt.Sprintf(" [%s:%d]", path, line)
		} else {
			location = fmt.Sprintf(" [%s]", path)
		}
	}
	target := ""
	if t := strings.TrimSpace(f.GetTarget()); t != "" {
		target = fmt.Sprintf(" → %s", t)
	}
	return fmt.Sprintf("%s: %s%s%s", prefix, f.GetMessage(), location, target)
}

func countsToSummary(c *kov1.DocHealthCounts) DocsSummary {
	if c == nil {
		return DocsSummary{}
	}
	return DocsSummary{
		FilesChecked:      int(c.GetFilesChecked()),
		MarkdownWarnings:  int(c.GetMarkdownWarnings()),
		MarkdownFailures:  int(c.GetMarkdownFailures()),
		LocalLinks:        int(c.GetLocalLinks()),
		ExternalLinks:     int(c.GetExternalLinks()),
		BrokenLinks:       int(c.GetBrokenLinks()),
		ExternalWarnings:  int(c.GetExternalWarnings()),
		ExternalFailures:  int(c.GetExternalFailures()),
		MermaidValidated:  int(c.GetMermaidValidated()),
		MermaidFailures:   int(c.GetMermaidFailures()),
		AbsolutePathHits:  int(c.GetAbsolutePathHits()),
		AbsoluteFailures:  int(c.GetAbsoluteFailures()),
		CodeFilesScanned:  int(c.GetCodeFilesScanned()),
		CodeRefsFound:     int(c.GetCodeRefsFound()),
		CodeRefsBroken:    int(c.GetCodeRefsBroken()),
		DocRefsFound:      int(c.GetDocRefsFound()),
		DocRefsBroken:     int(c.GetDocRefsBroken()),
		MarkedRefsFound:   int(c.GetMarkedRefsFound()),
		MarkedRefsBroken:  int(c.GetMarkedRefsBroken()),
		MarkedRefsSkipped: int(c.GetMarkedRefsSkipped()),
		MarkedRefsUnknown: int(c.GetMarkedRefsUnknown()),
		DocsInManifest:    int(c.GetDocsInManifest()),
		DocsNotInManifest: int(c.GetDocsNotInManifest()),
	}
}
