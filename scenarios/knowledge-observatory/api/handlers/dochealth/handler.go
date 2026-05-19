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
	"time"

	"connectrpc.com/connect"

	"knowledge-observatory/internal/services/dochealth"

	kov1 "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1"
)

// Handler implements the generated KnowledgeObservatoryServiceHandler.
type Handler struct {
	service *dochealth.Service
	now     func() time.Time
}

// New builds a Connect handler backed by the provided dochealth service.
// service must be non-nil; production wires the singleton created in
// server.setupServices.
func New(service *dochealth.Service) *Handler {
	return &Handler{service: service, now: time.Now}
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
	}
	result, err := h.service.DocHealth(ctx, in.GetScenarioName(), opts)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(translate(result, h.now())), nil
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

func translate(r *dochealth.DocHealthResult, now time.Time) *kov1.DocHealthResponse {
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
	return resp
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
