// Package artifacts mounts the draft-lifecycle Connect surface.
package artifacts

import (
	"context"
	"fmt"
	"time"

	agentmanager "content-desk/integrations/agentmanager"
	assetstudio "content-desk/integrations/assetstudio"
	channelmanager "content-desk/integrations/channelmanager"
	internalartifacts "content-desk/internal/artifacts"
	"content-desk/internal/module"

	"connectrpc.com/connect"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/artifacts"
	artifactsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/artifacts/artifacts_v1connect"
)

type handler struct {
	repo        internalartifacts.Repository
	submitter   channelmanager.Submitter
	eligibility channelmanager.EligibilityChecker
	assets      assetstudio.Resolver
	agents      agentmanager.Runner
}

var _ artifactsconnect.ArtifactsServiceHandler = handler{}

func (h handler) ListDrafts(ctx context.Context, _ *connect.Request[artifactsv1.ListDraftsRequest]) (*connect.Response[artifactsv1.ListDraftsResponse], error) {
	drafts, err := h.repo.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &artifactsv1.ListDraftsResponse{}
	for _, draft := range drafts {
		response.Drafts = append(response.Drafts, draftMessage(draft))
	}
	return connect.NewResponse(response), nil
}

func (h handler) CreateDraft(ctx context.Context, request *connect.Request[artifactsv1.CreateDraftRequest]) (*connect.Response[artifactsv1.CreateDraftResponse], error) {
	draft, err := h.repo.Create(ctx, internalartifacts.Draft{CampaignID: request.Msg.CampaignId, PostTypeID: request.Msg.PostTypeId, Body: request.Msg.Body, Channel: request.Msg.Channel, Format: request.Msg.Format, Lane: request.Msg.Lane, SKU: request.Msg.Sku, ScenarioName: request.Msg.ScenarioName})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&artifactsv1.CreateDraftResponse{Draft: draftMessage(draft)}), nil
}

func (h handler) UpdateDraftBody(ctx context.Context, request *connect.Request[artifactsv1.UpdateDraftBodyRequest]) (*connect.Response[artifactsv1.UpdateDraftBodyResponse], error) {
	draft, err := h.repo.UpdateBody(ctx, request.Msg.Id, request.Msg.Body)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&artifactsv1.UpdateDraftBodyResponse{Draft: draftMessage(draft)}), nil
}

func (h handler) AttachReleasedAsset(ctx context.Context, request *connect.Request[artifactsv1.AttachReleasedAssetRequest]) (*connect.Response[artifactsv1.AttachReleasedAssetResponse], error) {
	if h.assets == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("asset studio integration is unavailable"))
	}
	asset, err := h.assets.ResolveReleasedAsset(ctx, request.Msg.AssetId)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if asset.ID != request.Msg.AssetId {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("asset studio returned a mismatched asset reference"))
	}
	attachment, err := h.repo.Attach(ctx, internalartifacts.Attachment{DraftID: request.Msg.DraftId, AssetID: asset.ID, Role: request.Msg.Role, AspectRatio: request.Msg.AspectRatio, AltText: request.Msg.AltText, Position: int(request.Msg.Position)})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&artifactsv1.AttachReleasedAssetResponse{Attachment: attachmentMessage(attachment)}), nil
}

func (h handler) ListDraftAttachments(ctx context.Context, request *connect.Request[artifactsv1.ListDraftAttachmentsRequest]) (*connect.Response[artifactsv1.ListDraftAttachmentsResponse], error) {
	attachments, err := h.repo.ListAttachments(ctx, request.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &artifactsv1.ListDraftAttachmentsResponse{}
	for _, attachment := range attachments {
		response.Attachments = append(response.Attachments, attachmentMessage(attachment))
	}
	return connect.NewResponse(response), nil
}

func (h handler) CommissionAgentWork(ctx context.Context, request *connect.Request[artifactsv1.CommissionAgentWorkRequest]) (*connect.Response[artifactsv1.CommissionAgentWorkResponse], error) {
	if h.agents == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("agent manager integration is unavailable"))
	}
	draft, err := h.repo.Get(ctx, request.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	receipt, err := h.agents.Commission(ctx, agentmanager.Commission{DraftID: draft.ID, Action: request.Msg.Action, Body: draft.Body})
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	commission, err := h.repo.RecordAgentCommission(ctx, internalartifacts.AgentCommission{DraftID: draft.ID, Action: request.Msg.Action, TaskID: receipt.TaskID, RunID: receipt.RunID, Status: receipt.Status})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&artifactsv1.CommissionAgentWorkResponse{CommissionId: commission.ID, TaskId: commission.TaskID, RunId: commission.RunID, Status: commission.Status}), nil
}

func (h handler) GetAgentWorkResult(ctx context.Context, request *connect.Request[artifactsv1.GetAgentWorkResultRequest]) (*connect.Response[artifactsv1.GetAgentWorkResultResponse], error) {
	if h.agents == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("agent manager integration is unavailable"))
	}
	commission, err := h.repo.GetAgentCommission(ctx, request.Msg.CommissionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	result, err := h.agents.GetResult(ctx, commission.RunID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(&artifactsv1.GetAgentWorkResultResponse{RunId: commission.RunID, Status: result.Status, Body: result.Body}), nil
}

func (h handler) AdoptAgentSuggestion(ctx context.Context, request *connect.Request[artifactsv1.AdoptAgentSuggestionRequest]) (*connect.Response[artifactsv1.AdoptAgentSuggestionResponse], error) {
	commission, err := h.repo.GetAgentCommission(ctx, request.Msg.CommissionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if request.Msg.Body == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("agent suggestion body is required"))
	}
	draft, err := h.repo.UpdateBody(ctx, commission.DraftID, request.Msg.Body)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if err := h.repo.RecordAgentAdoption(ctx, commission.ID, commission.DraftID, commission.RunID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&artifactsv1.AdoptAgentSuggestionResponse{Draft: draftMessage(draft)}), nil
}

// SubmitReleaseDraft does not publish directly. It verifies the editorial
// lifecycle state and delegates durable queue ownership to Channel Manager.
// A scheduled receipt remains an approved draft until Channel Manager later
// delivers a completed/partial outcome to the ledger inbox.
func (h handler) SubmitReleaseDraft(ctx context.Context, request *connect.Request[artifactsv1.SubmitReleaseDraftRequest]) (*connect.Response[artifactsv1.SubmitReleaseDraftResponse], error) {
	if h.submitter == nil {
		return nil, connect.NewError(connect.CodeUnavailable, nil)
	}
	draft, err := h.repo.RevalidateForRelease(ctx, request.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	attachments, err := h.repo.ListAttachments(ctx, draft.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var assetIDs []string
	for _, attachment := range attachments {
		assetIDs = append(assetIDs, attachment.AssetID)
	}
	receipt, err := h.submitter.SubmitRelease(ctx, channelmanager.Submission{IdentityID: request.Msg.IdentityId, Lane: request.Msg.Lane, DraftID: draft.ID, IdempotencyKey: request.Msg.IdempotencyKey, AssetIDs: assetIDs, DisclosureVisible: request.Msg.DisclosureVisible})
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(&artifactsv1.SubmitReleaseDraftResponse{Draft: draftMessage(draft), ReleaseId: receipt.ID, ActionId: receipt.ActionID, ReleaseStatus: receipt.Status}), nil
}

func (h handler) RecordReleaseOutcome(ctx context.Context, request *connect.Request[artifactsv1.RecordReleaseOutcomeRequest]) (*connect.Response[artifactsv1.RecordReleaseOutcomeResponse], error) {
	publishedAt, err := time.Parse(time.RFC3339Nano, request.Msg.PublishedAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parse published_at: %w", err))
	}
	draft, recordID, err := h.repo.RecordReleaseOutcome(ctx, internalartifacts.ReleaseOutcome{ReceiptID: request.Msg.ReceiptId, DraftID: request.Msg.DraftId, Status: request.Msg.Status, PlatformPostID: request.Msg.PlatformPostId, PublishedURL: request.Msg.PublishedUrl, PublishedAt: publishedAt})
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&artifactsv1.RecordReleaseOutcomeResponse{Draft: draftMessage(draft), PublishRecordId: recordID}), nil
}

func (h handler) TransitionDraft(ctx context.Context, request *connect.Request[artifactsv1.TransitionDraftRequest]) (*connect.Response[artifactsv1.TransitionDraftResponse], error) {
	draft, err := h.repo.Transition(ctx, request.Msg.Id, internalartifacts.DraftEvent(request.Msg.Event))
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&artifactsv1.TransitionDraftResponse{Draft: draftMessage(draft)}), nil
}

func (h handler) ApproveDraft(ctx context.Context, request *connect.Request[artifactsv1.ApproveDraftRequest]) (*connect.Response[artifactsv1.ApproveDraftResponse], error) {
	if request.Msg.IdentityId != "" || request.Msg.Lane != "" {
		if request.Msg.IdentityId == "" || request.Msg.Lane == "" || h.eligibility == nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("targeted approval requires identity and lane eligibility"))
		}
		eligibility, err := h.eligibility.CheckEligibility(ctx, request.Msg.IdentityId, request.Msg.Lane)
		if err != nil {
			_ = h.repo.RecordEligibility(ctx, request.Msg.Id, internalartifacts.ReleaseTarget{IdentityID: request.Msg.IdentityId, Lane: request.Msg.Lane, Eligibility: "unknown"})
			return nil, connect.NewError(connect.CodeUnavailable, err)
		}
		if err := h.repo.RecordEligibility(ctx, request.Msg.Id, internalartifacts.ReleaseTarget{IdentityID: request.Msg.IdentityId, Lane: request.Msg.Lane, Eligibility: eligibility}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if eligibility != "eligible" {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("identity is %s for lane %s", eligibility, request.Msg.Lane))
		}
	}
	draft, err := h.repo.Approve(ctx, request.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&artifactsv1.ApproveDraftResponse{Draft: draftMessage(draft)}), nil
}

func draftMessage(draft internalartifacts.Draft) *artifactsv1.Draft {
	return &artifactsv1.Draft{Id: draft.ID, CampaignId: draft.CampaignID, Status: string(draft.Status), PostTypeId: draft.PostTypeID, Body: draft.Body, Channel: draft.Channel, Format: draft.Format, Lane: draft.Lane, Sku: draft.SKU, ScenarioName: draft.ScenarioName}
}

func attachmentMessage(attachment internalartifacts.Attachment) *artifactsv1.DraftAttachment {
	return &artifactsv1.DraftAttachment{Id: attachment.ID, DraftId: attachment.DraftID, AssetId: attachment.AssetID, Role: attachment.Role, AspectRatio: attachment.AspectRatio, AltText: attachment.AltText, Position: int32(attachment.Position)}
}

func Module(db *database.RoutedDB) module.Module {
	client := channelmanager.NewClient()
	path, h := artifactsconnect.NewArtifactsServiceHandler(handler{repo: internalartifacts.NewSQLiteRepository(db), submitter: client, eligibility: client, assets: assetstudio.NewClient(), agents: agentmanager.NewClient()})
	return module.Module{Name: "artifacts", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}
func Schema() string { return internalartifacts.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "artifacts_approve", Path: artifactsconnect.ArtifactsServiceApproveDraftProcedure, Method: "POST", Summary: "Approve draft after all gates pass", Category: "artifacts"},
	{ID: "artifacts_create", Path: artifactsconnect.ArtifactsServiceCreateDraftProcedure, Method: "POST", Summary: "Create draft", Category: "artifacts"},
	{ID: "artifacts_list", Path: artifactsconnect.ArtifactsServiceListDraftsProcedure, Method: "POST", Summary: "List drafts", Category: "artifacts"},
	{ID: "artifacts_submit_release", Path: artifactsconnect.ArtifactsServiceSubmitReleaseDraftProcedure, Method: "POST", Summary: "Submit an approved draft to Channel Manager", Category: "artifacts"},
	{ID: "artifacts_record_release_outcome", Path: artifactsconnect.ArtifactsServiceRecordReleaseOutcomeProcedure, Method: "POST", Summary: "Record an idempotent Channel Manager publication outcome", Category: "artifacts"},
	{ID: "artifacts_transition", Path: artifactsconnect.ArtifactsServiceTransitionDraftProcedure, Method: "POST", Summary: "Transition draft", Category: "artifacts"},
	{ID: "artifacts_update_body", Path: artifactsconnect.ArtifactsServiceUpdateDraftBodyProcedure, Method: "POST", Summary: "Create attributed draft revision", Category: "artifacts"},
	{ID: "artifacts_attach_asset", Path: artifactsconnect.ArtifactsServiceAttachReleasedAssetProcedure, Method: "POST", Summary: "Attach a released Asset Studio reference", Category: "artifacts"},
	{ID: "artifacts_list_attachments", Path: artifactsconnect.ArtifactsServiceListDraftAttachmentsProcedure, Method: "POST", Summary: "List draft asset references", Category: "artifacts"},
	{ID: "artifacts_commission_agent_work", Path: artifactsconnect.ArtifactsServiceCommissionAgentWorkProcedure, Method: "POST", Summary: "Commission a governed editorial Agent Manager run", Category: "artifacts"},
	{ID: "artifacts_get_agent_work_result", Path: artifactsconnect.ArtifactsServiceGetAgentWorkResultProcedure, Method: "POST", Summary: "Read a governed editorial Agent Manager result", Category: "artifacts"},
	{ID: "artifacts_adopt_agent_suggestion", Path: artifactsconnect.ArtifactsServiceAdoptAgentSuggestionProcedure, Method: "POST", Summary: "Operator-adopt an editable Agent Manager suggestion", Category: "artifacts"},
}
