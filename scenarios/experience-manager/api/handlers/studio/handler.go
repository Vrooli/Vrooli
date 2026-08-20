package studio

import (
	"context"
	"encoding/json"
	"log"

	"connectrpc.com/connect"

	"experience-manager/internal/authoring"
	"experience-manager/internal/reconcile"
	"experience-manager/internal/spec"
	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract"
)

type handler struct {
	logger  *log.Logger
	service authoring.Service
}

func (h *handler) StartAuthoringSession(ctx context.Context, req *connect.Request[contractv1.StartAuthoringSessionRequest]) (*connect.Response[contractv1.StartAuthoringSessionResponse], error) {
	if err := requireRepository(h.service.Repo); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	session, err := h.service.StartSession(ctx, req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&contractv1.StartAuthoringSessionResponse{Session: protoSession(session, 0)}), nil
}

func (h *handler) SubmitPage(ctx context.Context, req *connect.Request[contractv1.SubmitPageRequest]) (*connect.Response[contractv1.SubmitPageResponse], error) {
	if err := requireRepository(h.service.Repo); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	session, page, err := h.service.SubmitPage(ctx, req.Msg.GetSessionId(), req.Msg.GetPage())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	pages, _ := h.service.Repo.ListPages(ctx, session.ID)
	return connect.NewResponse(&contractv1.SubmitPageResponse{Session: protoSession(session, len(pages)), Page: protoPageDraft(page)}), nil
}

func (h *handler) PreviewSession(ctx context.Context, req *connect.Request[contractv1.PreviewSessionRequest]) (*connect.Response[contractv1.PreviewSessionResponse], error) {
	if err := requireRepository(h.service.Repo); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	preview, err := h.service.Preview(ctx, req.Msg.GetSessionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&contractv1.PreviewSessionResponse{
		Session:    protoSession(preview.Session, len(preview.Diffs)),
		Diffs:      protoDiffs(preview.Diffs),
		Validation: protoValidation(preview.Report),
	}), nil
}

func (h *handler) ApplySession(ctx context.Context, req *connect.Request[contractv1.ApplySessionRequest]) (*connect.Response[contractv1.ApplySessionResponse], error) {
	if err := requireRepository(h.service.Repo); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	preview, err := h.service.Apply(ctx, req.Msg.GetSessionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&contractv1.ApplySessionResponse{
		Session:    protoSession(preview.Session, len(preview.Diffs)),
		Diffs:      protoDiffs(preview.Diffs),
		Validation: protoValidation(preview.Report),
	}), nil
}

func (h *handler) DiscardSession(ctx context.Context, req *connect.Request[contractv1.DiscardSessionRequest]) (*connect.Response[contractv1.DiscardSessionResponse], error) {
	if err := requireRepository(h.service.Repo); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := h.service.Discard(ctx, req.Msg.GetSessionId()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&contractv1.DiscardSessionResponse{SessionId: req.Msg.GetSessionId(), Discarded: true}), nil
}

func (h *handler) ListSpec(ctx context.Context, req *connect.Request[contractv1.ListSpecRequest]) (*connect.Response[contractv1.ListSpecResponse], error) {
	report, err := h.service.ListSpec(ctx, req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resp := &contractv1.ListSpecResponse{Scenario: report.Scenario}
	if report.Spec != nil {
		for _, ref := range report.Spec.Index.Pages {
			resp.Pages = append(resp.Pages, protoSpecDocument(ref))
		}
		for _, ref := range report.Spec.Index.Journeys {
			resp.Journeys = append(resp.Journeys, protoSpecDocument(ref))
		}
		for _, ref := range report.Spec.Index.Components {
			resp.Components = append(resp.Components, protoSpecDocument(ref))
		}
	}
	return connect.NewResponse(resp), nil
}

func (h *handler) ShowSpec(ctx context.Context, req *connect.Request[contractv1.ShowSpecRequest]) (*connect.Response[contractv1.ShowSpecResponse], error) {
	data, err := h.service.ShowPage(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetPage())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&contractv1.ShowSpecResponse{Scenario: req.Msg.GetScenario(), Page: req.Msg.GetPage(), Json: data}), nil
}

func (h *handler) ListEvidence(ctx context.Context, req *connect.Request[contractv1.ListEvidenceRequest]) (*connect.Response[contractv1.ListEvidenceResponse], error) {
	if err := requireEvidenceRepository(h.service.Evidence); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var report spec.Report
	var evidence []reconcile.Evidence
	var err error
	if componentID := req.Msg.GetComponent(); componentID != "" {
		report, evidence, err = h.service.ListComponentEvidence(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), componentID, req.Msg.GetClaim(), int(req.Msg.GetLimit()))
	} else {
		report, evidence, err = h.service.ListEvidence(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetPage(), req.Msg.GetClaim(), int(req.Msg.GetLimit()))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resp := &contractv1.ListEvidenceResponse{Scenario: report.Scenario, Page: req.Msg.GetPage()}
	for _, item := range evidence {
		resp.Evidence = append(resp.Evidence, protoEvidence(item))
	}
	return connect.NewResponse(resp), nil
}

func (h *handler) SuggestBindings(ctx context.Context, req *connect.Request[contractv1.SuggestBindingsRequest]) (*connect.Response[contractv1.SuggestBindingsResponse], error) {
	suggestions, err := h.service.SuggestBindings(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetPage(), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resp := &contractv1.SuggestBindingsResponse{Scenario: req.Msg.GetScenario(), Page: req.Msg.GetPage()}
	for _, s := range suggestions {
		resp.Suggestions = append(resp.Suggestions, &contractv1.BindingSuggestion{
			ElementId:      s.ElementID,
			Testid:         s.TestID,
			Role:           s.Role,
			AccessibleName: s.AccessibleName,
			Source:         s.Source,
		})
	}
	return connect.NewResponse(resp), nil
}

func (h *handler) RenderSpec(ctx context.Context, req *connect.Request[contractv1.RenderSpecRequest]) (*connect.Response[contractv1.RenderSpecResponse], error) {
	result, err := h.service.RenderSpec(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetPage(), req.Msg.GetMode())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&contractv1.RenderSpecResponse{
		Scenario:       result.Scenario,
		Page:           result.PageID,
		Mode:           result.Mode,
		Html:           result.HTML,
		ArtifactPath:   result.ArtifactPath,
		DegradedReason: result.DegradedReason,
	}), nil
}

func (h *handler) CompareVariants(ctx context.Context, req *connect.Request[contractv1.CompareVariantsRequest]) (*connect.Response[contractv1.CompareVariantsResponse], error) {
	result, err := h.service.CompareVariants(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetPage(), req.Msg.GetMode(), req.Msg.GetVariants())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resp := &contractv1.CompareVariantsResponse{
		Scenario:       result.Scenario,
		Page:           result.PageID,
		Mode:           result.Mode,
		Html:           result.HTML,
		ArtifactPath:   result.ArtifactPath,
		DegradedReason: result.DegradedReason,
	}
	for _, variant := range result.Variants {
		resp.Variants = append(resp.Variants, protoRenderedVariant(variant))
	}
	return connect.NewResponse(resp), nil
}

func (h *handler) PromoteVariant(ctx context.Context, req *connect.Request[contractv1.PromoteVariantRequest]) (*connect.Response[contractv1.PromoteVariantResponse], error) {
	result, err := h.service.PromoteVariant(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetPage(), req.Msg.GetVariant())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&contractv1.PromoteVariantResponse{
		Scenario:   result.Scenario,
		Page:       result.PageID,
		Variant:    protoRenderedVariant(result.Variant),
		Diffs:      protoDiffs(result.Diffs),
		Validation: protoValidation(result.Report),
	}), nil
}

func protoSession(s authoring.Session, count int) *contractv1.AuthoringSession {
	return &contractv1.AuthoringSession{
		Id:         s.ID,
		Scenario:   s.Scenario,
		TargetPath: s.TargetPath,
		Status:     s.Status,
		PageCount:  int32(count),
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
	}
}

func protoPageDraft(p authoring.PageDraft) *contractv1.PageDraft {
	return &contractv1.PageDraft{Id: p.PageID, Path: p.Path, Title: p.Title, Status: p.Status}
}

func protoDiffs(diffs []authoring.Diff) []*contractv1.FileDiff {
	out := make([]*contractv1.FileDiff, 0, len(diffs))
	for _, d := range diffs {
		out = append(out, &contractv1.FileDiff{Path: d.Path, Action: d.Action, Before: d.Before, After: d.After})
	}
	return out
}

func protoRenderedVariant(v authoring.VariantResult) *contractv1.RenderedVariant {
	return &contractv1.RenderedVariant{Id: v.ID, Title: v.Title, Html: v.HTML}
}

func protoSpecDocument(ref spec.DocumentRef) *contractv1.SpecDocument {
	return &contractv1.SpecDocument{Id: ref.ID, Path: ref.Path, Title: ref.Title, Status: ref.Status}
}

func protoEvidence(e reconcile.Evidence) *contractv1.ReconciliationEvidence {
	return &contractv1.ReconciliationEvidence{
		Id:             e.ID,
		Scenario:       e.Scenario,
		DocumentKind:   e.DocumentKind,
		Page:           e.PageID,
		ComponentId:    e.ComponentID,
		ComponentTitle: e.ComponentTitle,
		ExampleName:    e.ExampleName,
		Route:          e.Route,
		State:          e.StateID,
		Viewport:       e.ViewportID,
		ViewportWidth:  int32(e.ViewportWidth),
		ViewportHeight: int32(e.ViewportHeight),
		Claim:          e.ClaimID,
		ClaimType:      e.ClaimType,
		Verdict:        e.Verdict,
		CaptureRef:     e.CaptureRef,
		AxNodeJson:     e.AXNodeJSON,
		Measurement:    protoMeasurement(e.MeasurementJSON),
		Message:        e.Message,
		CheckedAt:      e.CheckedAt,
	}
}

func protoMeasurement(raw string) *contractv1.ClaimMeasurement {
	var measurement reconcile.ClaimMeasurement
	if raw == "" || raw == "{}" || json.Unmarshal([]byte(raw), &measurement) != nil {
		return nil
	}
	out := &contractv1.ClaimMeasurement{
		Metric:     measurement.Metric,
		Observed:   measurement.Observed,
		Required:   measurement.Required,
		Unit:       measurement.Unit,
		Comparator: measurement.Comparator,
	}
	for _, subject := range measurement.Subjects {
		item := &contractv1.MeasuredSubject{ElementId: subject.ElementID, TestId: subject.TestID, ContextId: subject.ContextID, Value: subject.Value}
		if subject.Bounds != nil {
			item.Bounds = &contractv1.MeasuredBounds{X: subject.Bounds.X, Y: subject.Bounds.Y, Width: subject.Bounds.Width, Height: subject.Bounds.Height}
		}
		out.Subjects = append(out.Subjects, item)
	}
	return out
}

func protoValidation(report spec.Report) *contractv1.ValidateScenarioResponse {
	return &contractv1.ValidateScenarioResponse{
		Scenario:   report.Scenario,
		Status:     status(report.Findings),
		TargetPath: report.TargetPath,
		Report:     &contractv1.ExperienceContractReport{Findings: protoFindings(report.Findings)},
	}
}

func protoFindings(findings []spec.Finding) []*contractv1.ExperienceFinding {
	out := make([]*contractv1.ExperienceFinding, 0, len(findings))
	for _, f := range findings {
		out = append(out, &contractv1.ExperienceFinding{
			Code:        f.Code,
			Severity:    f.Severity,
			Message:     f.Message,
			Location:    firstLocation(f.Locations),
			Remediation: f.Suggestion,
		})
	}
	return out
}

func status(findings []spec.Finding) string {
	for _, f := range findings {
		if f.Severity == spec.SeverityError {
			return "FAILED"
		}
	}
	return "PASSED"
}

func firstLocation(locations []string) string {
	if len(locations) == 0 {
		return ""
	}
	return locations[0]
}
