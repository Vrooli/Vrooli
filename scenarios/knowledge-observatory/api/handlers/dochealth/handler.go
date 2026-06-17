// Package dochealth hosts the KnowledgeObservatoryService.DocHealth
// Connect-RPC handler — pure transport translation between the proto
// surface and the in-process dochealth.Service domain layer.
//
// DOC: docs/concepts/ARCHITECTURE.md#documentation-health
// DOC: docs/internal/SEAMS.md#dochealth
package dochealth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"

	"knowledge-observatory/internal/services/dochealth"

	"github.com/vrooli/maturity-go/assessment"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	kov1 "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// Handler implements the generated KnowledgeObservatoryServiceHandler.
type Handler struct {
	service *dochealth.Service
	spec    *assessment.Spec
	now     func() time.Time
}

type Deps struct {
	Service      *dochealth.Service
	MaturitySpec *assessment.Spec
}

// New builds a Connect handler backed by the provided dochealth service.
// service must be non-nil; production wires the singleton created in
// server.setupServices.
func New(service *dochealth.Service) *Handler {
	return NewWithDeps(Deps{Service: service})
}

func NewWithDeps(deps Deps) *Handler {
	return &Handler{service: deps.Service, spec: deps.MaturitySpec, now: time.Now}
}

// WithClock overrides the timestamp source (tests).
func (h *Handler) WithClock(now func() time.Time) *Handler {
	h.now = now
	return h
}

// DocHealth runs the full documentation-health suite for a scenario.
func (h *Handler) DocHealth(ctx context.Context, req *connect.Request[kov1.DocHealthRequest]) (*connect.Response[kov1.DocHealthResponse], error) {
	if h.service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("documentation health service unavailable"))
	}
	in := req.Msg
	opts := dochealth.DocHealthOptions{
		StrictExternalLinks:      in.StrictExternalLinks,
		RequireAllDocsRegistered: in.RequireAllDocsRegistered,
		SkipExternalLinks:        in.SkipExternalLinks,
		Scope:                    in.GetScope(),
		Path:                     in.GetPath(),
		Checks:                   in.GetChecks(),
	}
	result, err := h.service.DocHealth(ctx, in.GetScenarioName(), opts)
	if err != nil {
		return nil, mapError(err)
	}
	resp, err := translate(result, h.now(), h.spec)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build docs maturity assessment: %w", err))
	}
	return connect.NewResponse(resp), nil
}

// ValidateScenario adapts DocHealth to the shared ScenarioValidationService
// contract consumed by Test Genie. The full DocHealthResponse is preserved in
// native_detail for knowledge-observatory's own clients.
func (h *Handler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	msg := req.Msg
	if msg == nil {
		msg = &scenariovalidationv1.ValidateScenarioRequest{}
	}
	scenario := strings.TrimSpace(msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	native, err := h.DocHealth(ctx, connect.NewRequest(&kov1.DocHealthRequest{
		ScenarioName: scenario,
	}))
	if err != nil {
		return nil, err
	}
	resp, err := assessment.BuildValidationResponse(native.Msg.GetScenarioName(), native.Msg.GetAssessment(), native.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func mapError(err error) error {
	switch {
	case errors.Is(err, dochealth.ErrScenarioNameInvalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, dochealth.ErrScenarioNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, dochealth.ErrScenarioRootInvalid):
		return connect.NewError(connect.CodeUnavailable, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func translate(r *dochealth.DocHealthResult, now time.Time, spec *assessment.Spec) (*kov1.DocHealthResponse, error) {
	resp := &kov1.DocHealthResponse{
		ScenarioName:  r.ScenarioName,
		HealthScore:   r.HealthScore,
		TotalDocs:     int32(r.TotalDocs),
		ExtraDocs:     append([]string(nil), r.ExtraDocs...),
		TemporaryDocs: append([]string(nil), r.TemporaryDocs...),
		Counts:        translateCounts(r.Counts),
		Timestamp:     now.UTC().Format(time.RFC3339),
	}
	if r.SourceTemplateID != "" {
		v := r.SourceTemplateID
		resp.SourceTemplateId = &v
	}
	if r.ManifestPath != "" {
		v := r.ManifestPath
		resp.ManifestPath = &v
	}
	if r.ManifestStatus != "" {
		v := r.ManifestStatus
		resp.ManifestStatus = &v
	}
	for _, m := range r.MisplacedDocs {
		entry := &kov1.DocHealthMisplacedDoc{
			ActualPath:   m.ActualPath,
			ExpectedPath: m.ExpectedPath,
			Severity:     severityToProto(m.Severity),
		}
		if m.DocType != "" {
			v := m.DocType
			entry.DocType = &v
		}
		if m.Message != "" {
			v := m.Message
			entry.Message = &v
		}
		resp.MisplacedDocs = append(resp.MisplacedDocs, entry)
	}
	for _, m := range r.MissingDocs {
		entry := &kov1.DocHealthMissingDoc{
			DocType:    m.DocType,
			Path:       m.Path,
			Severity:   severityToProto(m.Severity),
			RequiredBy: append([]string(nil), m.RequiredBy...),
		}
		if m.Completion != "" {
			v := m.Completion
			entry.Completion = &v
		}
		resp.MissingDocs = append(resp.MissingDocs, entry)
	}
	resp.ContractFindings = translateFindings(r.ContractFindings)
	resp.ContentFindings = translateFindings(r.ContentFindings)
	resp.ReferenceFindings = translateFindings(r.ReferenceFindings)
	resp.ManifestFindings = translateFindings(r.ManifestFindings)
	a, err := buildMaturityAssessment(r, spec)
	if err != nil {
		return nil, err
	}
	resp.Assessment = a
	return resp, nil
}

func buildMaturityAssessment(r *dochealth.DocHealthResult, spec *assessment.Spec) (*commonv1.MaturityAssessment, error) {
	if spec == nil {
		return nil, fmt.Errorf("maturity spec is required")
	}
	findings := make([]assessment.Finding, 0,
		len(r.MisplacedDocs)+len(r.MissingDocs)+len(r.ExtraDocs)+len(r.TemporaryDocs)+
			len(r.ContractFindings)+len(r.ContentFindings)+len(r.ReferenceFindings)+len(r.ManifestFindings))
	for _, m := range r.MisplacedDocs {
		findings = append(findings, assessment.Finding{
			Code:        "misplaced_doc",
			Severity:    severityToAssessment(m.Severity),
			Title:       "Misplaced documentation",
			Message:     firstNonEmpty(m.Message, "documentation file is in the wrong location"),
			Location:    m.ActualPath,
			Remediation: "Move the document to " + m.ExpectedPath,
			Source:      architecturev1.FindingSource_FINDING_SOURCE_DOCS,
			Phase:       spec.Phase,
		})
	}
	for _, m := range r.MissingDocs {
		findings = append(findings, assessment.Finding{
			Code:        "missing_doc",
			Severity:    severityToAssessment(m.Severity),
			Title:       "Missing documentation",
			Message:     fmt.Sprintf("required %s documentation is missing", firstNonEmpty(m.DocType, "scenario")),
			Location:    m.Path,
			Remediation: "Create the required documentation file.",
			Source:      architecturev1.FindingSource_FINDING_SOURCE_DOCS,
			Phase:       spec.Phase,
		})
	}
	for _, path := range r.ExtraDocs {
		findings = append(findings, assessment.Finding{
			Code:        "extra_doc",
			Severity:    architecturev1.FindingSeverity_FINDING_SEVERITY_INFO.String(),
			Title:       "Extra documentation",
			Message:     "documentation file is outside the scenario documentation contract",
			Location:    path,
			Remediation: "Register the doc in docs/manifest.json or remove it if obsolete.",
			Source:      architecturev1.FindingSource_FINDING_SOURCE_DOCS,
			Phase:       spec.Phase,
		})
	}
	for _, path := range r.TemporaryDocs {
		findings = append(findings, assessment.Finding{
			Code:        "temporary_doc",
			Severity:    architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING.String(),
			Title:       "Temporary documentation",
			Message:     "temporary documentation artifact should be promoted or removed",
			Location:    path,
			Remediation: "Promote durable content into the docs contract or remove the temporary artifact.",
			Source:      architecturev1.FindingSource_FINDING_SOURCE_DOCS,
			Phase:       spec.Phase,
		})
	}
	appendDocFindings := func(in []dochealth.Finding) {
		for _, f := range in {
			findings = append(findings, assessment.Finding{
				Code:     f.Code,
				Severity: severityToAssessment(f.Severity),
				Title:    f.Code,
				Message:  f.Message,
				Location: docLocation(f.Path, f.Line),
				Source:   architecturev1.FindingSource_FINDING_SOURCE_DOCS,
				Phase:    spec.Phase,
			})
		}
	}
	appendDocFindings(r.ContractFindings)
	appendDocFindings(r.ContentFindings)
	appendDocFindings(r.ReferenceFindings)
	appendDocFindings(r.ManifestFindings)
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: r.ScenarioName,
		Spec:     *spec,
		Findings: findings,
	})
}

func severityToAssessment(s dochealth.Severity) string {
	switch s {
	case dochealth.SeverityFailure:
		return architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR.String()
	case dochealth.SeverityWarning:
		return architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING.String()
	case dochealth.SeverityInfo:
		return architecturev1.FindingSeverity_FINDING_SEVERITY_INFO.String()
	default:
		return ""
	}
}

func docLocation(path string, line int) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if line > 0 {
		return fmt.Sprintf("%s:%d", path, line)
	}
	return path
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func translateFindings(in []dochealth.Finding) []*kov1.DocHealthFinding {
	if len(in) == 0 {
		return nil
	}
	out := make([]*kov1.DocHealthFinding, 0, len(in))
	for _, f := range in {
		entry := &kov1.DocHealthFinding{
			Code:     f.Code,
			Severity: severityToProto(f.Severity),
			Message:  f.Message,
		}
		if f.Path != "" {
			v := f.Path
			entry.Path = &v
		}
		if f.DocType != "" {
			v := f.DocType
			entry.DocType = &v
		}
		if f.Line != 0 {
			v := int32(f.Line)
			entry.Line = &v
		}
		if f.Target != "" {
			v := f.Target
			entry.Target = &v
		}
		out = append(out, entry)
	}
	return out
}

func translateCounts(c dochealth.Counts) *kov1.DocHealthCounts {
	return &kov1.DocHealthCounts{
		FilesChecked:      int32(c.FilesChecked),
		MarkdownWarnings:  int32(c.MarkdownWarnings),
		MarkdownFailures:  int32(c.MarkdownFailures),
		LocalLinks:        int32(c.LocalLinks),
		ExternalLinks:     int32(c.ExternalLinks),
		BrokenLinks:       int32(c.BrokenLinks),
		ExternalWarnings:  int32(c.ExternalWarnings),
		ExternalFailures:  int32(c.ExternalFailures),
		MermaidValidated:  int32(c.MermaidValidated),
		MermaidFailures:   int32(c.MermaidFailures),
		AbsolutePathHits:  int32(c.AbsolutePathHits),
		AbsoluteFailures:  int32(c.AbsoluteFailures),
		CodeFilesScanned:  int32(c.CodeFilesScanned),
		CodeRefsFound:     int32(c.CodeRefsFound),
		CodeRefsBroken:    int32(c.CodeRefsBroken),
		DocRefsFound:      int32(c.DocRefsFound),
		DocRefsBroken:     int32(c.DocRefsBroken),
		MarkedRefsFound:   int32(c.MarkedRefsFound),
		MarkedRefsBroken:  int32(c.MarkedRefsBroken),
		MarkedRefsSkipped: int32(c.MarkedRefsSkipped),
		MarkedRefsUnknown: int32(c.MarkedRefsUnknown),
		DocsInManifest:    int32(c.DocsInManifest),
		DocsNotInManifest: int32(c.DocsNotInManifest),
		NumbersFlagged:    int32(c.NumbersFlagged),
	}
}

func severityToProto(s dochealth.Severity) kov1.DocHealthSeverity {
	switch s {
	case dochealth.SeverityInfo:
		return kov1.DocHealthSeverity_DOC_HEALTH_SEVERITY_INFO
	case dochealth.SeverityWarning:
		return kov1.DocHealthSeverity_DOC_HEALTH_SEVERITY_WARNING
	case dochealth.SeverityFailure:
		return kov1.DocHealthSeverity_DOC_HEALTH_SEVERITY_FAILURE
	default:
		// Promote unspecified to WARNING because proto validation rejects 0.
		return kov1.DocHealthSeverity_DOC_HEALTH_SEVERITY_WARNING
	}
}
