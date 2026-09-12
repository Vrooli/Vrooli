package development

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/connectx"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct{ service *Service }

func RegisterRoutes(router *mux.Router, service *Service, authentication authn.Config) {
	path, handler := apiconnect.NewDevelopmentServiceHandler(&Handler{service: service})
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: authn.Middleware(authentication)(handler)})
}

// RuntimeBlockers describes missing qualifications, not an operator grant.
// Keep preview, retained review and board consumers on the same projection.
func RuntimeBlockers() []string {
	return []string{
		"Authoritative outcome receipt resolvers are not yet configured for product acceptance.",
	}
}

func (h *Handler) GetDevelopment(ctx context.Context, req *connect.Request[api.GetDevelopmentRequest]) (*connect.Response[api.DevelopmentResponse], error) {
	if req == nil || req.Msg == nil || !validItem(req.Msg.WorkItem) {
		return nil, transportError(ErrInvalid)
	}
	state, err := h.service.Get(ctx, req.Msg.WorkItem)
	return h.response(ctx, state, err)
}

func (h *Handler) ApproveDevelopment(ctx context.Context, req *connect.Request[api.ApproveDevelopmentRequest]) (*connect.Response[api.DevelopmentResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, transportError(ErrInvalid)
	}
	p, err := ProposalFromProto(req.Msg.Proposal)
	if err != nil {
		return nil, transportError(err)
	}
	state, err := h.service.Approve(ctx, p, req.Msg.ReviewedDigest, req.Msg.ExpectedVersion, req.Msg.Reason)
	return h.response(ctx, state, err)
}

func (h *Handler) RevokeDevelopment(ctx context.Context, req *connect.Request[api.RevokeDevelopmentRequest]) (*connect.Response[api.DevelopmentResponse], error) {
	if req == nil || req.Msg == nil || !validItem(req.Msg.WorkItem) {
		return nil, transportError(ErrInvalid)
	}
	state, err := h.service.Revoke(ctx, req.Msg.WorkItem, req.Msg.ExpectedVersion, req.Msg.Reason)
	return h.response(ctx, state, err)
}

func (h *Handler) AcceptDevelopment(ctx context.Context, req *connect.Request[api.AcceptDevelopmentRequest]) (*connect.Response[api.DevelopmentResponse], error) {
	if req == nil || req.Msg == nil || !validItem(req.Msg.WorkItem) {
		return nil, transportError(ErrInvalid)
	}
	state, err := h.service.Accept(ctx, req.Msg.WorkItem, req.Msg.ExpectedVersion, req.Msg.EvidenceRefs)
	return h.response(ctx, state, err)
}

func (h *Handler) GetDevelopmentArtifact(ctx context.Context, req *connect.Request[api.GetDevelopmentArtifactRequest]) (*connect.Response[api.GetDevelopmentArtifactResponse], error) {
	if req == nil || req.Msg == nil || !validItem(req.Msg.WorkItem) {
		return nil, transportError(ErrInvalid)
	}
	state, err := h.service.Get(ctx, req.Msg.WorkItem)
	if err != nil {
		return nil, transportError(err)
	}
	known := false
	for _, a := range state.Approvals {
		if a.Digest == req.Msg.Digest {
			known = true
			break
		}
	}
	if !known {
		return nil, transportError(ErrNotFound)
	}
	snapshot, err := h.service.Snapshot(ctx, req.Msg.Digest)
	if err != nil {
		return nil, transportError(err)
	}
	for _, a := range snapshot.Artifacts {
		if a.Path != req.Msg.Path {
			continue
		}
		for _, content := range snapshot.Contents {
			if content.Path == a.Path {
				return connect.NewResponse(&api.GetDevelopmentArtifactResponse{Artifact: artifactProto(a), Content: content.Bytes}), nil
			}
		}
	}
	return nil, transportError(ErrNotFound)
}

func (h *Handler) response(ctx context.Context, state Engagement, err error) (*connect.Response[api.DevelopmentResponse], error) {
	if err != nil {
		return nil, transportError(err)
	}
	snapshot, err := h.service.Snapshot(ctx, state.Digest)
	if err != nil {
		return nil, transportError(err)
	}
	r := &api.DevelopmentResponse{
		WorkItem: state.WorkItem, WorkShape: state.WorkShape, Version: state.Version, Digest: state.Digest, Status: state.Status,
		Used: usageProto(state.Used), Reserved: usageProto(state.Reserved), Checkpoint: state.Checkpoint,
		AcceptedBy: state.AcceptedBy, StopReason: state.StopReason, ApprovedProposal: ProposalProto(snapshot.Proposal),
		GoalMessage: snapshot.GoalMessage, LaunchBlockers: h.service.LaunchBlockers(ctx),
	}
	r.Campaign = campaignProto(state.Campaign)
	if state.Cancellation != nil {
		r.Cancellation = &api.DevelopmentCancellationIntent{OperationId: state.Cancellation.OperationID, Reason: state.Cancellation.Reason, State: state.Cancellation.State, RequestedAt: timestamppb.New(state.Cancellation.RequestedAt)}
		if state.Cancellation.AckAt != nil {
			r.Cancellation.AckAt = timestamppb.New(*state.Cancellation.AckAt)
		}
	}
	if state.AcceptedAt != nil {
		r.AcceptedAt = timestamppb.New(*state.AcceptedAt)
	}
	for _, a := range state.Approvals {
		r.Approvals = append(r.Approvals, &api.DevelopmentApproval{Digest: a.Digest, Actor: a.Actor, Reason: a.Reason, At: timestamppb.New(a.At)})
	}
	for _, a := range state.Attempts {
		attempt := &api.DevelopmentAttempt{Key: a.Key, Digest: a.Digest, WorkflowDigest: a.WorkflowDigest, GrantDigest: a.GrantDigest, Mode: a.Mode, Reserved: usageProto(a.Reserved), Used: usageProto(a.Used), ExecutionId: a.ExecutionID, StartedAt: timestamppb.New(a.StartedAt), Checkpoint: a.Checkpoint, CapabilityRevision: a.CapabilityRevision, SelectionReason: a.SelectionReason}
		if a.SettledAt != nil {
			attempt.SettledAt = timestamppb.New(*a.SettledAt)
		}
		r.Attempts = append(r.Attempts, attempt)
	}
	for _, e := range state.Evidence {
		r.Evidence = append(r.Evidence, &api.DevelopmentEvidence{OutcomeId: e.OutcomeID, Source: e.Source, ResolverId: e.ResolverID, ReceiptSchema: e.ReceiptSchema, Cohort: e.Cohort, ReceiptId: e.ReceiptID, Digest: e.Digest, ExecutionId: e.ExecutionID, SubjectRevision: e.SubjectRevision, ObservedAt: timestamppb.New(e.ObservedAt), FreshUntil: timestamppb.New(e.FreshUntil)})
	}
	for _, a := range snapshot.Artifacts {
		r.Artifacts = append(r.Artifacts, artifactProto(a))
	}
	return connect.NewResponse(r), nil
}

func campaignProto(c CampaignCheckpoint) *api.DevelopmentCampaignCheckpoint {
	r := &api.DevelopmentCampaignCheckpoint{Digest: c.Digest, ApprovalDigest: c.ApprovalDigest, AttemptKey: c.AttemptKey, OwnerExecutionId: c.OwnerExecutionID, Pending: c.Pending, RequiredOutcomeIds: append([]string(nil), c.RequiredOutcomeIDs...), CompletedOutcomeIds: append([]string(nil), c.CompletedOutcomeIDs...), RemainingOutcomeIds: append([]string(nil), c.RemainingOutcomeIDs...), LastOutcome: c.LastOutcome, NoProgressCycles: int32(c.NoProgressCycles)}
	if c.LastCheckpoint != nil {
		r.LastCheckpoint = &api.DevelopmentCheckpointReference{Kind: c.LastCheckpoint.Kind, Value: c.LastCheckpoint.Value, Digest: c.LastCheckpoint.Digest}
	}
	return r
}

func ProposalFromProto(m *api.PreviewDevelopmentRequest) (Proposal, error) {
	if m == nil {
		return Proposal{}, ErrInvalid
	}
	p := Proposal{Scenario: m.Scenario, WorkItem: m.WorkItem, Objective: m.Objective, ArtifactPaths: m.ArtifactPaths, AcceptanceAllow: m.AcceptanceAllow, AcceptanceDeny: m.AcceptanceDeny, AllowedEffects: m.AllowedEffects, MaxTokens: m.MaxTokens, MaxWallSeconds: m.MaxWallSeconds, ExecutionStrategy: m.ExecutionStrategy}
	if m.PlanRef != nil {
		p.PlanRef = &PlanReference{Provider: m.PlanRef.Provider, PlanID: m.PlanRef.PlanId, Slug: m.PlanRef.Slug, Role: m.PlanRef.Role}
	}
	p.BudgetPolicy = m.BudgetPolicy
	if err := validateBudgetPolicy(defaultBudgetPolicy(p.BudgetPolicy)); err != nil {
		return Proposal{}, err
	}
	if g := m.Guidance; g != nil {
		p.Guidance = Guidance{Effort: g.Effort, StartingState: g.StartingState, Validation: g.Validation, RepairRelatedCode: g.RepairRelatedCode, AdditionalInstructions: g.AdditionalInstructions}
	}
	if err := p.Guidance.Validate(); err != nil {
		return Proposal{}, err
	}
	for _, o := range m.Outcomes {
		if o == nil {
			return Proposal{}, fmt.Errorf("outcome cannot be null: %w", ErrInvalid)
		}
		p.Outcomes = append(p.Outcomes, Outcome{ID: o.Id, Criterion: o.Criterion, EvidenceSource: o.EvidenceSource})
	}
	return p, nil
}

func ProposalProto(p Proposal) *api.PreviewDevelopmentRequest {
	m := &api.PreviewDevelopmentRequest{Scenario: p.Scenario, WorkItem: p.WorkItem, Objective: p.Objective, ArtifactPaths: p.ArtifactPaths, AcceptanceAllow: p.AcceptanceAllow, AcceptanceDeny: p.AcceptanceDeny, AllowedEffects: p.AllowedEffects, MaxTokens: p.MaxTokens, MaxWallSeconds: p.MaxWallSeconds, ExecutionStrategy: p.ExecutionStrategy}
	if p.PlanRef != nil {
		m.PlanRef = &shared.PlanRef{Provider: p.PlanRef.Provider, PlanId: p.PlanRef.PlanID, Slug: p.PlanRef.Slug, Role: p.PlanRef.Role}
	}
	m.BudgetPolicy = p.BudgetPolicy
	m.Guidance = &api.DevelopmentGuidance{Effort: p.Guidance.Effort, StartingState: p.Guidance.StartingState, Validation: p.Guidance.Validation, RepairRelatedCode: p.Guidance.RepairRelatedCode, AdditionalInstructions: p.Guidance.AdditionalInstructions}
	for _, o := range p.Outcomes {
		m.Outcomes = append(m.Outcomes, &api.DevelopmentOutcome{Id: o.ID, Criterion: o.Criterion, EvidenceSource: o.EvidenceSource})
	}
	return m
}

func usageProto(u Usage) *api.DevelopmentUsage {
	return &api.DevelopmentUsage{Tokens: u.Tokens, WallSeconds: u.WallSeconds}
}

func artifactProto(a Artifact) *api.DevelopmentArtifact {
	return &api.DevelopmentArtifact{Path: a.Path, Sha256: a.SHA256, SizeBytes: a.SizeBytes}
}

func transportError(err error) error {
	code := connect.CodeInternal
	switch {
	case errors.Is(err, ErrInvalid):
		code = connect.CodeInvalidArgument
	case errors.Is(err, ErrNotFound):
		code = connect.CodeNotFound
	case errors.Is(err, ErrConflict):
		code = connect.CodeAborted
	case errors.Is(err, ErrDenied):
		code = connect.CodeFailedPrecondition
	}
	if code == connect.CodeInternal {
		err = errors.New("development state operation failed")
	}
	return connect.NewError(code, err)
}
