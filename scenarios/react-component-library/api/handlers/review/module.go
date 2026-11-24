// Package review exposes the server-owned findings queue over Connect-RPC.
//
// internal/review owns findings and finding_dispositions; this package maps
// those records onto the generated service and keeps every triage write
// explicit (decision + reason + actor).
package review

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review"
	reviewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review/review_v1connect"

	"react-component-library/internal/module"
	domain "react-component-library/internal/review"
)

// Service is the review surface the handler needs. *domain.Service implements
// it; tests inject a narrow fake.
type Service interface {
	List(ctx context.Context, filter domain.Filter, limit, offset int) (domain.Page, error)
	Get(ctx context.Context, id string) (domain.Finding, error)
	RecordDisposition(ctx context.Context, id, decision, reason, actor string) (domain.Finding, error)
	Assign(ctx context.Context, id, owner string) (domain.Finding, error)
	Summary(ctx context.Context) (domain.Summary, error)
}

// Handler adapts the review domain onto the generated Connect interface.
type Handler struct {
	service Service
	ports   Ports
}

// NewHandler wires a service implementation.
func NewHandler(service Service) *Handler { return &Handler{service: service} }

// NewHandlerWithPorts mounts findings plus candidate acceptance and visual review
// operations. The injected ports keep those workflows on their owning domains.
func NewHandlerWithPorts(service Service, ports Ports) *Handler {
	return &Handler{service: service, ports: ports}
}

// Module mounts ReviewService over the scenario database. Evidence projection
// reads catalog_gate_evidence through the same pool the rest of the API uses.
// database is the request-aware SQL surface used by the review queue and its
// evidence projector. *sql.DB remains useful for fixtures; production passes
// api-core's RoutedDB so each request context selects its pool.
type database interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

func Module(db database) module.Module {
	return ModuleFromService(domain.NewService(db, domain.NewEvidenceProjector(db)))
}

// ModuleFromService is the constructor tests and alternate stores use.
func ModuleFromService(service Service) module.Module {
	return ModuleFromServiceWithPorts(service, Ports{})
}

// ModuleFromServiceWithPorts mounts ReviewService without routing through the
// workspace candidate service handler. Acceptance and critique dependencies
// stay explicit.
func ModuleFromServiceWithPorts(service Service, ports Ports) module.Module {
	path, handler := reviewconnect.NewReviewServiceHandler(NewHandlerWithPorts(service, ports))
	return module.Module{
		Name: "review",
		Mount: func(router *mux.Router) {
			connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internal/review.Schema for the modules registry.
func Schema() string { return domain.Schema() }

// ListFindings returns findings newest-first with optional filters.
func (h *Handler) ListFindings(ctx context.Context, req *connect.Request[reviewv1.ListFindingsRequest]) (*connect.Response[reviewv1.ListFindingsResponse], error) {
	if h == nil || h.service == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("review service is not configured"))
	}
	status, err := statusFromProto(req.Msg.GetStatus())
	if err != nil {
		return nil, err
	}
	severity, err := severityFromProto(req.Msg.GetSeverity())
	if err != nil {
		return nil, err
	}
	page, err := h.service.List(ctx, domain.Filter{
		Status:      status,
		Severity:    severity,
		SubjectKind: strings.TrimSpace(req.Msg.GetSubjectKind()),
		Query:       strings.TrimSpace(req.Msg.GetQuery()),
	}, int(req.Msg.GetLimit()), int(req.Msg.GetOffset()))
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*reviewv1.Finding, 0, len(page.Findings))
	for _, finding := range page.Findings {
		out = append(out, toProtoFinding(finding))
	}
	return connect.NewResponse(&reviewv1.ListFindingsResponse{
		Findings: out,
		Total:    int32(page.Total),
	}), nil
}

// GetFinding returns one finding and its durable disposition history.
func (h *Handler) GetFinding(ctx context.Context, req *connect.Request[reviewv1.GetFindingRequest]) (*connect.Response[reviewv1.GetFindingResponse], error) {
	if h == nil || h.service == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("review service is not configured"))
	}
	finding, err := h.service.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&reviewv1.GetFindingResponse{Finding: toProtoFinding(finding)}), nil
}

// RecordDisposition appends a decision and applies its status transition.
func (h *Handler) RecordDisposition(ctx context.Context, req *connect.Request[reviewv1.RecordDispositionRequest]) (*connect.Response[reviewv1.RecordDispositionResponse], error) {
	if h == nil || h.service == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("review service is not configured"))
	}
	decision, err := decisionFromProto(req.Msg.GetDecision())
	if err != nil {
		return nil, err
	}
	finding, err := h.service.RecordDisposition(ctx, req.Msg.GetId(), decision, req.Msg.GetReason(), req.Msg.GetActor())
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&reviewv1.RecordDispositionResponse{Finding: toProtoFinding(finding)}), nil
}

// AssignFinding sets the accountable owner.
func (h *Handler) AssignFinding(ctx context.Context, req *connect.Request[reviewv1.AssignFindingRequest]) (*connect.Response[reviewv1.AssignFindingResponse], error) {
	if h == nil || h.service == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("review service is not configured"))
	}
	finding, err := h.service.Assign(ctx, req.Msg.GetId(), req.Msg.GetOwner())
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&reviewv1.AssignFindingResponse{Finding: toProtoFinding(finding)}), nil
}

// GetReviewSummary returns queue counts by severity and status.
func (h *Handler) GetReviewSummary(ctx context.Context, _ *connect.Request[reviewv1.GetReviewSummaryRequest]) (*connect.Response[reviewv1.GetReviewSummaryResponse], error) {
	if h == nil || h.service == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("review service is not configured"))
	}
	summary, err := h.service.Summary(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&reviewv1.GetReviewSummaryResponse{
		Summary: &reviewv1.ReviewSummary{
			Open:         int32(summary.Open),
			Blockers:     int32(summary.Blockers),
			High:         int32(summary.High),
			Acknowledged: int32(summary.Acknowledged),
			Accepted:     int32(summary.Accepted),
			Fixed:        int32(summary.Fixed),
		},
	}), nil
}

func toProtoFinding(finding domain.Finding) *reviewv1.Finding {
	dispositions := make([]*reviewv1.FindingDisposition, 0, len(finding.Dispositions))
	for _, disposition := range finding.Dispositions {
		dispositions = append(dispositions, &reviewv1.FindingDisposition{
			Id:       disposition.ID,
			Decision: decisionToProto(disposition.Decision),
			Reason:   disposition.Reason,
			Actor:    disposition.Actor,
			At:       disposition.At,
		})
	}
	return &reviewv1.Finding{
		Id:           finding.ID,
		SubjectKind:  finding.SubjectKind,
		SubjectId:    finding.SubjectID,
		Gate:         finding.Gate,
		Severity:     severityToProto(finding.Severity),
		Status:       statusToProto(finding.Status),
		Message:      finding.Message,
		Remediation:  finding.Remediation,
		EvidenceJson: finding.EvidenceJSON,
		DocsRef:      finding.DocsRef,
		Owner:        finding.Owner,
		CreatedAt:    finding.CreatedAt,
		Dispositions: dispositions,
	}
}

func statusFromProto(value reviewv1.FindingStatus) (string, error) {
	switch value {
	case reviewv1.FindingStatus_FINDING_STATUS_UNSPECIFIED:
		return "", nil
	case reviewv1.FindingStatus_FINDING_STATUS_OPEN:
		return domain.StatusOpen, nil
	case reviewv1.FindingStatus_FINDING_STATUS_ACKNOWLEDGED:
		return domain.StatusAcknowledged, nil
	case reviewv1.FindingStatus_FINDING_STATUS_ACCEPTED:
		return domain.StatusAccepted, nil
	case reviewv1.FindingStatus_FINDING_STATUS_FIXED:
		return domain.StatusFixed, nil
	case reviewv1.FindingStatus_FINDING_STATUS_RETIRED:
		return domain.StatusRetired, nil
	default:
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown finding status %d", value))
	}
}

func severityFromProto(value reviewv1.FindingSeverity) (string, error) {
	switch value {
	case reviewv1.FindingSeverity_FINDING_SEVERITY_UNSPECIFIED:
		return "", nil
	case reviewv1.FindingSeverity_FINDING_SEVERITY_BLOCKER:
		return domain.SeverityBlocker, nil
	case reviewv1.FindingSeverity_FINDING_SEVERITY_HIGH:
		return domain.SeverityHigh, nil
	case reviewv1.FindingSeverity_FINDING_SEVERITY_MEDIUM:
		return domain.SeverityMedium, nil
	case reviewv1.FindingSeverity_FINDING_SEVERITY_LOW:
		return domain.SeverityLow, nil
	case reviewv1.FindingSeverity_FINDING_SEVERITY_INFO:
		return domain.SeverityInfo, nil
	default:
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown finding severity %d", value))
	}
}

func decisionFromProto(value reviewv1.DispositionDecision) (string, error) {
	switch value {
	case reviewv1.DispositionDecision_DISPOSITION_DECISION_UNSPECIFIED:
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("decision is required"))
	case reviewv1.DispositionDecision_DISPOSITION_DECISION_ACKNOWLEDGE:
		return domain.DecisionAcknowledge, nil
	case reviewv1.DispositionDecision_DISPOSITION_DECISION_ACCEPT_RISK:
		return domain.DecisionAcceptRisk, nil
	case reviewv1.DispositionDecision_DISPOSITION_DECISION_FIX:
		return domain.DecisionFix, nil
	case reviewv1.DispositionDecision_DISPOSITION_DECISION_RETIRE:
		return domain.DecisionRetire, nil
	default:
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown disposition decision %d", value))
	}
}

func severityToProto(value string) reviewv1.FindingSeverity {
	switch value {
	case domain.SeverityBlocker:
		return reviewv1.FindingSeverity_FINDING_SEVERITY_BLOCKER
	case domain.SeverityHigh:
		return reviewv1.FindingSeverity_FINDING_SEVERITY_HIGH
	case domain.SeverityMedium:
		return reviewv1.FindingSeverity_FINDING_SEVERITY_MEDIUM
	case domain.SeverityLow:
		return reviewv1.FindingSeverity_FINDING_SEVERITY_LOW
	case domain.SeverityInfo:
		return reviewv1.FindingSeverity_FINDING_SEVERITY_INFO
	default:
		return reviewv1.FindingSeverity_FINDING_SEVERITY_UNSPECIFIED
	}
}

func statusToProto(value string) reviewv1.FindingStatus {
	switch value {
	case domain.StatusOpen:
		return reviewv1.FindingStatus_FINDING_STATUS_OPEN
	case domain.StatusAcknowledged:
		return reviewv1.FindingStatus_FINDING_STATUS_ACKNOWLEDGED
	case domain.StatusAccepted:
		return reviewv1.FindingStatus_FINDING_STATUS_ACCEPTED
	case domain.StatusFixed:
		return reviewv1.FindingStatus_FINDING_STATUS_FIXED
	case domain.StatusRetired:
		return reviewv1.FindingStatus_FINDING_STATUS_RETIRED
	default:
		return reviewv1.FindingStatus_FINDING_STATUS_UNSPECIFIED
	}
}

func decisionToProto(value string) reviewv1.DispositionDecision {
	switch value {
	case domain.DecisionAcknowledge:
		return reviewv1.DispositionDecision_DISPOSITION_DECISION_ACKNOWLEDGE
	case domain.DecisionAcceptRisk:
		return reviewv1.DispositionDecision_DISPOSITION_DECISION_ACCEPT_RISK
	case domain.DecisionFix:
		return reviewv1.DispositionDecision_DISPOSITION_DECISION_FIX
	case domain.DecisionRetire:
		return reviewv1.DispositionDecision_DISPOSITION_DECISION_RETIRE
	default:
		return reviewv1.DispositionDecision_DISPOSITION_DECISION_UNSPECIFIED
	}
}

// mapError preserves the two actionable failure modes and hides storage
// details behind Internal.
func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, errors.New("finding not found"))
	case errors.Is(err, domain.ErrInvalid):
		message := strings.TrimPrefix(err.Error(), domain.ErrInvalid.Error()+": ")
		return connect.NewError(connect.CodeInvalidArgument, errors.New(message))
	default:
		return connect.NewError(connect.CodeInternal, errors.New("review storage failure"))
	}
}

// Endpoints is the static surface consumed by `make endpoints`.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "review_findings_list",
		Path:        reviewconnect.ReviewServiceListFindingsProcedure,
		Method:      "POST",
		Summary:     "List findings",
		Description: "Returns one page of the server-owned findings queue newest-first plus the filtered total. status and severity are optional enum filters; subject_kind and query narrow further. limit defaults to 100 and is clamped to 500; offset is zero-based and must not be negative.",
		Category:    "review",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"status": "FindingStatus", "severity": "FindingSeverity", "subject_kind": "string", "query": "string", "limit": "integer (default 100, max 500)", "offset": "integer (default 0, non-negative)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"findings": "array<Finding>", "total": "integer (filtered count before limit/offset)"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "status, severity, or offset is not accepted"}},
	},
	{
		ID:          "review_findings_get",
		Path:        reviewconnect.ReviewServiceGetFindingProcedure,
		Method:      "POST",
		Summary:     "Get a finding",
		Description: "Returns one finding with its full disposition history, newest first.",
		Category:    "review",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"finding": "Finding"}},
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No finding with that id"}},
	},
	{
		ID:          "review_disposition_record",
		Path:        reviewconnect.ReviewServiceRecordDispositionProcedure,
		Method:      "POST",
		Summary:     "Record a disposition",
		Description: "Appends a durable triage decision with a required reason and actor, then applies the status transition: acknowledge -> acknowledged, accept_risk -> accepted, fix -> open, retire -> retired.",
		Category:    "review",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)", "decision": "DispositionDecision (required)", "reason": "string (required)", "actor": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"finding": "Finding"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Decision, reason, or actor is missing or unknown"},
			{Status: 404, Code: "not_found", Description: "No finding with that id"},
		},
	},
	{
		ID:          "review_finding_assign",
		Path:        reviewconnect.ReviewServiceAssignFindingProcedure,
		Method:      "POST",
		Summary:     "Assign a finding",
		Description: "Sets the accountable owner. An empty owner clears the assignment; both forms are explicit writes.",
		Category:    "review",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)", "owner": "string (empty clears)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"finding": "Finding"}},
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No finding with that id"}},
	},
	{
		ID:          "review_summary",
		Path:        reviewconnect.ReviewServiceGetReviewSummaryProcedure,
		Method:      "POST",
		Summary:     "Get the review queue summary",
		Description: "Returns open, unresolved blocker/high, acknowledged, accepted and fixed counts. Blockers and high count statuses open or acknowledged so a triaged item is not mistaken for an untouched one.",
		Category:    "review",
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"summary": "ReviewSummary"}},
	},
	{ID: "candidate_accept_check", Path: reviewconnect.ReviewServiceCheckCandidateAcceptanceProcedure, Method: "POST", Summary: "Check current candidate, render, behavior, calibration and review evidence", Category: "review"},
	{ID: "candidate_accept", Path: reviewconnect.ReviewServiceAcceptCandidateProcedure, Method: "POST", Summary: "Record an idempotent terminal acceptance decision", Category: "review"},
	{ID: "acceptance_get", Path: reviewconnect.ReviewServiceGetAcceptanceProcedure, Method: "POST", Summary: "Read a recorded acceptance decision", Category: "review"},
	{ID: "critique_list", Path: reviewconnect.ReviewServiceListCritiquesProcedure, Method: "POST", Summary: "List verified critiques for an exact candidate render", Category: "review"},
	{ID: "critique_rubric", Path: reviewconnect.ReviewServiceGetCritiqueRubricProcedure, Method: "POST", Summary: "Read the versioned visual rubric and calibration status", Category: "review"},
	{ID: "critique_record", Path: reviewconnect.ReviewServiceRecordCritiqueProcedure, Method: "POST", Summary: "Verify image evidence and record an attributed visual critique", Category: "review"},
	{ID: "critique_get", Path: reviewconnect.ReviewServiceGetCritiqueProcedure, Method: "POST", Summary: "Read an immutable visual review fact", Category: "review"},
}
