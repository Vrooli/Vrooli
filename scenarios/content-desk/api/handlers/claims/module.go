// Package claims mounts the shared-claim library Connect surface.
package claims

import (
	"context"
	"errors"
	"time"

	internalclaims "content-desk/internal/claims"
	"content-desk/internal/module"

	"connectrpc.com/connect"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	claimsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/claims"
	claimsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/claims/claims_v1connect"
)

type handler struct{ library internalclaims.Library }

var _ claimsconnect.ClaimsServiceHandler = handler{}

func (h handler) ListClaims(ctx context.Context, _ *connect.Request[claimsv1.ListClaimsRequest]) (*connect.Response[claimsv1.ListClaimsResponse], error) {
	claims, err := h.library.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &claimsv1.ListClaimsResponse{}
	for _, claim := range claims {
		response.Claims = append(response.Claims, claimMessage(claim))
	}
	return connect.NewResponse(response), nil
}

func (h handler) ListDraftClaims(ctx context.Context, request *connect.Request[claimsv1.ListDraftClaimsRequest]) (*connect.Response[claimsv1.ListDraftClaimsResponse], error) {
	claims, err := h.library.ListForDraft(ctx, request.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &claimsv1.ListDraftClaimsResponse{}
	for _, claim := range claims {
		response.Claims = append(response.Claims, claimMessage(claim))
	}
	return connect.NewResponse(response), nil
}

func (h handler) CreateClaim(ctx context.Context, request *connect.Request[claimsv1.CreateClaimRequest]) (*connect.Response[claimsv1.CreateClaimResponse], error) {
	var observedAt time.Time
	var err error
	if request.Msg.ObservedAt != "" {
		observedAt, err = time.Parse(time.RFC3339Nano, request.Msg.ObservedAt)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	claim, err := h.library.Create(ctx, internalclaims.Claim{Statement: request.Msg.Statement, Kind: request.Msg.Kind}, internalclaims.Evidence{Kind: request.Msg.EvidenceKind, Reference: request.Msg.Reference, Command: request.Msg.Command, ExpectedResult: request.Msg.ExpectedResult, ObservedAt: observedAt})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&claimsv1.CreateClaimResponse{Claim: claimMessage(claim)}), nil
}

func (h handler) CiteClaim(ctx context.Context, request *connect.Request[claimsv1.CiteClaimRequest]) (*connect.Response[claimsv1.CiteClaimResponse], error) {
	err := h.library.Cite(ctx, internalclaims.Citation{DraftID: request.Msg.DraftId, ClaimID: request.Msg.ClaimId, Start: int(request.Msg.SpanStart), End: int(request.Msg.SpanEnd)}, request.Msg.Body)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&claimsv1.CiteClaimResponse{}), nil
}

func (h handler) VerifyClaim(ctx context.Context, request *connect.Request[claimsv1.VerifyClaimRequest]) (*connect.Response[claimsv1.VerifyClaimResponse], error) {
	claim, err := h.library.Verify(ctx, request.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&claimsv1.VerifyClaimResponse{Claim: claimMessage(claim)}), nil
}

func (h handler) SweepClaims(ctx context.Context, _ *connect.Request[claimsv1.SweepClaimsRequest]) (*connect.Response[claimsv1.SweepClaimsResponse], error) {
	claims, err := h.library.Sweep(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	response := &claimsv1.SweepClaimsResponse{}
	for _, claim := range claims {
		response.Claims = append(response.Claims, claimMessage(claim))
	}
	return connect.NewResponse(response), nil
}

func (h handler) GetClaimCoverage(ctx context.Context, request *connect.Request[claimsv1.GetClaimCoverageRequest]) (*connect.Response[claimsv1.GetClaimCoverageResponse], error) {
	if request.Msg.DraftId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("draft_id is required"))
	}
	supported, uncovered, err := h.library.Coverage(ctx, request.Msg.DraftId, request.Msg.Body)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	response := &claimsv1.GetClaimCoverageResponse{}
	for _, span := range supported {
		response.SupportedSpans = append(response.SupportedSpans, &claimsv1.TextSpan{Start: int32(span.Start), End: int32(span.End), ClaimId: span.ClaimID, Supported: true})
	}
	for _, span := range uncovered {
		response.UncoveredSpans = append(response.UncoveredSpans, &claimsv1.TextSpan{Start: int32(span.Start), End: int32(span.End), Supported: false})
	}
	return connect.NewResponse(response), nil
}

func (h handler) ExtractClaimProposals(ctx context.Context, request *connect.Request[claimsv1.ExtractClaimProposalsRequest]) (*connect.Response[claimsv1.ExtractClaimProposalsResponse], error) {
	proposals, err := h.library.ExtractProposals(ctx, request.Msg.DraftId, request.Msg.Body)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	response := &claimsv1.ExtractClaimProposalsResponse{}
	for _, proposal := range proposals {
		response.Proposals = append(response.Proposals, proposalMessage(proposal))
	}
	return connect.NewResponse(response), nil
}

func (h handler) ListClaimProposals(ctx context.Context, request *connect.Request[claimsv1.ListClaimProposalsRequest]) (*connect.Response[claimsv1.ListClaimProposalsResponse], error) {
	proposals, err := h.library.ListProposals(ctx, request.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &claimsv1.ListClaimProposalsResponse{}
	for _, proposal := range proposals {
		response.Proposals = append(response.Proposals, proposalMessage(proposal))
	}
	return connect.NewResponse(response), nil
}

func (h handler) DecideClaimProposal(ctx context.Context, request *connect.Request[claimsv1.DecideClaimProposalRequest]) (*connect.Response[claimsv1.DecideClaimProposalResponse], error) {
	proposal, err := h.library.DecideProposal(ctx, request.Msg.Id, request.Msg.Status)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&claimsv1.DecideClaimProposalResponse{Proposal: proposalMessage(proposal)}), nil
}

func claimMessage(claim internalclaims.Claim) *claimsv1.Claim {
	return &claimsv1.Claim{Id: claim.ID, Statement: claim.Statement, VerificationStatus: claim.VerificationStatus, Kind: claim.Kind}
}

func proposalMessage(proposal internalclaims.Proposal) *claimsv1.ClaimProposal {
	message := &claimsv1.ClaimProposal{Id: proposal.ID, DraftId: proposal.DraftID, Statement: proposal.Statement, Status: proposal.Status, CreatedAt: proposal.CreatedAt.UTC().Format(time.RFC3339Nano)}
	if !proposal.DecidedAt.IsZero() {
		message.DecidedAt = proposal.DecidedAt.UTC().Format(time.RFC3339Nano)
	}
	return message
}

func Module(db *database.RoutedDB) module.Module {
	path, h := claimsconnect.NewClaimsServiceHandler(handler{library: internalclaims.NewLibrary(db, internalclaims.LocalRunner{})})
	return module.Module{Name: "claims", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}
func Schema() string { return internalclaims.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "claims_list", Path: claimsconnect.ClaimsServiceListClaimsProcedure, Method: "POST", Summary: "List claims", Category: "claims"},
	{ID: "claims_list_draft", Path: claimsconnect.ClaimsServiceListDraftClaimsProcedure, Method: "POST", Summary: "List claims cited by a draft", Category: "claims"},
	{ID: "claims_create", Path: claimsconnect.ClaimsServiceCreateClaimProcedure, Method: "POST", Summary: "Create claim with evidence", Category: "claims"},
	{ID: "claims_cite", Path: claimsconnect.ClaimsServiceCiteClaimProcedure, Method: "POST", Summary: "Cite a claim at a draft span", Category: "claims"},
	{ID: "claims_verify", Path: claimsconnect.ClaimsServiceVerifyClaimProcedure, Method: "POST", Summary: "Run claim verification", Category: "claims"},
	{ID: "claims_sweep", Path: claimsconnect.ClaimsServiceSweepClaimsProcedure, Method: "POST", Summary: "Verify all check-backed claims", Category: "claims"},
	{ID: "claims_coverage", Path: claimsconnect.ClaimsServiceGetClaimCoverageProcedure, Method: "POST", Summary: "Get supported and uncovered citation spans for a draft", Category: "claims"},
	{ID: "claims_extract_proposals", Path: claimsconnect.ClaimsServiceExtractClaimProposalsProcedure, Method: "POST", Summary: "Create review-only claim proposals", Category: "claims"},
	{ID: "claims_list_proposals", Path: claimsconnect.ClaimsServiceListClaimProposalsProcedure, Method: "POST", Summary: "List claim proposals for a draft", Category: "claims"},
	{ID: "claims_decide_proposal", Path: claimsconnect.ClaimsServiceDecideClaimProposalProcedure, Method: "POST", Summary: "Accept or reject a claim proposal", Category: "claims"},
}
