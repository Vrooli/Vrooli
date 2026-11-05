// Package objectives exposes the generated Connect transport edge for the
// objective authority domain. It owns no domain logic: every RPC delegates to
// the single objective application service so all clients observe one
// validation, one revision scheme and one ordering authority.
package objectives

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	objectivesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/objectives"
	objectivesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/objectives/objectives_v1connect"

	domain "prompt-manager/internal/objectives"
)

type connectHandler struct {
	objectivesconnect.UnimplementedObjectivesServiceHandler
	service *domain.Service
}

// NewConnectMount builds the Connect service mount (procedure path + handler)
// for registration on the existing router. The registered path is generated
// from the objectives proto service name.
func NewConnectMount(service *domain.Service, opts ...connect.HandlerOption) (string, http.Handler) {
	return objectivesconnect.NewObjectivesServiceHandler(&connectHandler{service: service}, opts...)
}

func (h *connectHandler) ListObjectives(ctx context.Context, _ *connect.Request[objectivesv1.ListObjectivesRequest]) (*connect.Response[objectivesv1.ListObjectivesResponse], error) {
	objs, err := h.service.ListObjectives(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(objectiveListToProto(objs)), nil
}

func (h *connectHandler) GetObjective(ctx context.Context, req *connect.Request[objectivesv1.GetObjectiveRequest]) (*connect.Response[objectivesv1.Objective], error) {
	o, ok, err := h.service.GetObjective(ctx, req.Msg.GetId())
	if err != nil {
		return nil, mapError(err)
	}
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("objective %q not found", req.Msg.GetId()))
	}
	return connect.NewResponse(objectiveToProto(o)), nil
}

func (h *connectHandler) UpsertObjective(ctx context.Context, req *connect.Request[objectivesv1.UpsertObjectiveRequest]) (*connect.Response[objectivesv1.Objective], error) {
	o := objectiveFromInput(req.Msg.GetObjective())
	result, err := h.service.UpsertObjective(ctx, o, req.Msg.GetExpectedMeaningRevision())
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(objectiveToProto(result)), nil
}

func (h *connectHandler) DeleteObjective(ctx context.Context, req *connect.Request[objectivesv1.DeleteObjectiveRequest]) (*connect.Response[objectivesv1.DeleteObjectiveResponse], error) {
	if err := h.service.DeleteObjective(ctx, req.Msg.GetId(), req.Msg.GetExpectedMeaningRevision()); err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&objectivesv1.DeleteObjectiveResponse{}), nil
}

func (h *connectHandler) ReorderObjectives(ctx context.Context, req *connect.Request[objectivesv1.ReorderObjectivesRequest]) (*connect.Response[objectivesv1.ListObjectivesResponse], error) {
	objs, err := h.service.ReorderObjectives(ctx, req.Msg.GetObjectiveIds())
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(objectiveListToProto(objs)), nil
}

func (h *connectHandler) ListTeamAttachments(ctx context.Context, req *connect.Request[objectivesv1.ListTeamAttachmentsRequest]) (*connect.Response[objectivesv1.ListTeamAttachmentsResponse], error) {
	teamID := req.Msg.GetTeamId()
	atts, err := h.service.ListTeamAttachments(ctx, teamID)
	if err != nil {
		return nil, mapError(err)
	}
	revision, err := h.service.TeamAttachmentRevision(ctx, teamID)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(teamAttachmentsToProto(teamID, revision, atts)), nil
}

func (h *connectHandler) GetTeamAttachmentRevision(ctx context.Context, req *connect.Request[objectivesv1.GetTeamAttachmentRevisionRequest]) (*connect.Response[objectivesv1.GetTeamAttachmentRevisionResponse], error) {
	revision, err := h.service.TeamAttachmentRevision(ctx, req.Msg.GetTeamId())
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&objectivesv1.GetTeamAttachmentRevisionResponse{
		TeamId:             req.Msg.GetTeamId(),
		AttachmentRevision: revision,
	}), nil
}

func (h *connectHandler) AttachObjective(ctx context.Context, req *connect.Request[objectivesv1.AttachObjectiveRequest]) (*connect.Response[objectivesv1.Attachment], error) {
	result, err := h.service.Attach(ctx, attachmentFromInput(req.Msg.GetAttachment()), req.Msg.GetExpectedTeamRevision())
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(attachmentToProto(result)), nil
}

func (h *connectHandler) UpdateAttachment(ctx context.Context, req *connect.Request[objectivesv1.UpdateAttachmentRequest]) (*connect.Response[objectivesv1.Attachment], error) {
	result, err := h.service.UpdateAttachment(ctx, attachmentFromInput(req.Msg.GetAttachment()), req.Msg.GetExpectedTeamRevision())
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(attachmentToProto(result)), nil
}

func (h *connectHandler) DetachObjective(ctx context.Context, req *connect.Request[objectivesv1.DetachObjectiveRequest]) (*connect.Response[objectivesv1.DetachObjectiveResponse], error) {
	if err := h.service.Detach(ctx, req.Msg.GetObjectiveId(), req.Msg.GetTeamId(), req.Msg.GetExpectedTeamRevision()); err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&objectivesv1.DetachObjectiveResponse{}), nil
}

func (h *connectHandler) ReorderTeamAttachments(ctx context.Context, req *connect.Request[objectivesv1.ReorderTeamAttachmentsRequest]) (*connect.Response[objectivesv1.ListTeamAttachmentsResponse], error) {
	teamID := req.Msg.GetTeamId()
	atts, err := h.service.ReorderTeamAttachments(ctx, teamID, req.Msg.GetObjectiveIds(), req.Msg.GetExpectedTeamRevision())
	if err != nil {
		return nil, mapError(err)
	}
	revision, err := h.service.TeamAttachmentRevision(ctx, teamID)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(teamAttachmentsToProto(teamID, revision, atts)), nil
}

func (h *connectHandler) AcknowledgeObjective(ctx context.Context, req *connect.Request[objectivesv1.AcknowledgeObjectiveRequest]) (*connect.Response[objectivesv1.Attachment], error) {
	result, err := h.service.Acknowledge(ctx, req.Msg.GetObjectiveId(), req.Msg.GetTeamId(), req.Msg.GetRevision())
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(attachmentToProto(result)), nil
}

func (h *connectHandler) ListRelations(ctx context.Context, _ *connect.Request[objectivesv1.ListRelationsRequest]) (*connect.Response[objectivesv1.ListRelationsResponse], error) {
	relations, err := h.service.ListRelations(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	out := &objectivesv1.ListRelationsResponse{}
	for _, relation := range relations {
		out.Relations = append(out.Relations, &objectivesv1.Relation{
			FromObjectiveId: relation.FromObjectiveID,
			ToObjectiveId:   relation.ToObjectiveID,
		})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) AddRelation(ctx context.Context, req *connect.Request[objectivesv1.AddRelationRequest]) (*connect.Response[objectivesv1.AddRelationResponse], error) {
	if err := h.service.AddRelation(ctx, req.Msg.GetFromObjectiveId(), req.Msg.GetToObjectiveId()); err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&objectivesv1.AddRelationResponse{}), nil
}

func (h *connectHandler) DeleteRelation(ctx context.Context, req *connect.Request[objectivesv1.DeleteRelationRequest]) (*connect.Response[objectivesv1.DeleteRelationResponse], error) {
	if err := h.service.DeleteRelation(ctx, req.Msg.GetFromObjectiveId(), req.Msg.GetToObjectiveId()); err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&objectivesv1.DeleteRelationResponse{}), nil
}

func (h *connectHandler) ValidateObjectives(ctx context.Context, _ *connect.Request[objectivesv1.ValidateObjectivesRequest]) (*connect.Response[objectivesv1.ValidationResult], error) {
	result, err := h.service.Validate(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	out := &objectivesv1.ValidationResult{Errors: int32(result.Errors), Warnings: int32(result.Warnings)}
	for _, finding := range result.Findings {
		out.Findings = append(out.Findings, &objectivesv1.Finding{
			Rule:        finding.Rule,
			Severity:    finding.Severity,
			ObjectiveId: finding.ObjectiveID,
			TeamId:      finding.TeamID,
			Detail:      finding.Detail,
		})
	}
	return connect.NewResponse(out), nil
}

func objectiveListToProto(objs []domain.Objective) *objectivesv1.ListObjectivesResponse {
	out := &objectivesv1.ListObjectivesResponse{Objectives: make([]*objectivesv1.Objective, 0, len(objs))}
	for _, o := range objs {
		out.Objectives = append(out.Objectives, objectiveToProto(o))
	}
	return out
}

func objectiveToProto(o domain.Objective) *objectivesv1.Objective {
	return &objectivesv1.Objective{
		Id:              o.ID,
		Title:           o.Title,
		Class:           string(o.Class),
		EvidenceSource:  o.EvidenceSource,
		HasEvidence:     o.HasEvidence,
		GapMarker:       o.GapMarker,
		GlobalOrder:     int32(o.GlobalOrder),
		MeaningRevision: o.MeaningRevision,
	}
}

func objectiveFromInput(in *objectivesv1.ObjectiveInput) domain.Objective {
	if in == nil {
		return domain.Objective{}
	}
	return domain.Objective{
		ID:             in.GetId(),
		Title:          in.GetTitle(),
		Class:          domain.Class(in.GetClass()),
		EvidenceSource: in.GetEvidenceSource(),
		GapMarker:      in.GetGapMarker(),
	}
}

func teamAttachmentsToProto(teamID, revision string, atts []domain.Attachment) *objectivesv1.ListTeamAttachmentsResponse {
	out := &objectivesv1.ListTeamAttachmentsResponse{
		TeamId:             teamID,
		AttachmentRevision: revision,
		Attachments:        make([]*objectivesv1.Attachment, 0, len(atts)),
	}
	for _, a := range atts {
		out.Attachments = append(out.Attachments, attachmentToProto(a))
	}
	return out
}

func attachmentToProto(a domain.Attachment) *objectivesv1.Attachment {
	return &objectivesv1.Attachment{
		ObjectiveId:          a.ObjectiveID,
		TeamId:               a.TeamID,
		Role:                 a.Role,
		Coverage:             a.Coverage,
		Note:                 a.Note,
		Priority:             int32(a.Priority),
		AcknowledgedRevision: a.AcknowledgedRevision,
		AttachmentRevision:   a.AttachmentRevision,
		RestatementPending:   a.RestatementPending,
	}
}

func attachmentFromInput(in *objectivesv1.AttachmentInput) domain.Attachment {
	if in == nil {
		return domain.Attachment{}
	}
	return domain.Attachment{
		ObjectiveID: in.GetObjectiveId(),
		TeamID:      in.GetTeamId(),
		Role:        in.GetRole(),
		Coverage:    in.GetCoverage(),
		Note:        in.GetNote(),
	}
}

// mapError translates the objective domain's sentinel errors into Connect
// status codes. Every client therefore observes the same classification, not
// just the same message.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		return connect.NewError(connect.CodeInternal, err)
	}
	code := connect.CodeInternal
	switch {
	case errors.Is(validation.Err, domain.ErrNotFound),
		errors.Is(validation.Err, domain.ErrUnknownObjective),
		errors.Is(validation.Err, domain.ErrUnknownTeam):
		code = connect.CodeNotFound
	case errors.Is(validation.Err, domain.ErrConflict):
		code = connect.CodeAborted
	case errors.Is(validation.Err, domain.ErrDuplicateLink):
		code = connect.CodeAlreadyExists
	case errors.Is(validation.Err, domain.ErrInvalidClass),
		errors.Is(validation.Err, domain.ErrInvalidRole),
		errors.Is(validation.Err, domain.ErrInvalidCoverage),
		errors.Is(validation.Err, domain.ErrOrderMismatch):
		code = connect.CodeInvalidArgument
	case errors.Is(validation.Err, domain.ErrCycle),
		errors.Is(validation.Err, domain.ErrActiveReferences):
		code = connect.CodeFailedPrecondition
	}
	return connect.NewError(code, err)
}
