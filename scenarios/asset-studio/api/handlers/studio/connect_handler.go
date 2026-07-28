package studio

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	core "asset-studio/internal/studio"
	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/blobstore"
	studiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/studio"
)

type connectHandler struct {
	mu     sync.Mutex
	studio *core.Studio
	store  core.StateStore
	blobs  blobstore.BlobStore
}

func NewConnectHandler(s *core.Studio) *connectHandler {
	return &connectHandler{studio: s}
}

func NewConnectHandlerWithStore(s *core.Studio, store core.StateStore, blobs blobstore.BlobStore) *connectHandler {
	return &connectHandler{studio: s, store: store, blobs: blobs}
}

func (h *connectHandler) persist(ctx context.Context) error {
	if h.store == nil {
		return nil
	}
	if err := h.store.Save(ctx, h.studio); err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	return nil
}

func (h *connectHandler) ListIdentities(_ context.Context, req *connect.Request[studiov1.ListIdentitiesRequest]) (*connect.Response[studiov1.ListIdentitiesResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := &studiov1.ListIdentitiesResponse{}
	for _, identity := range h.studio.Identities {
		if req.Msg.Kind == "" || string(identity.Kind) == req.Msg.Kind {
			out.Identities = append(out.Identities, toProtoIdentity(identity))
		}
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreateIdentity(ctx context.Context, req *connect.Request[studiov1.CreateIdentityRequest]) (*connect.Response[studiov1.CreateIdentityResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	identity := fromProtoIdentity(req.Msg.Identity)
	if identity.ID == "" {
		identity.ID = uuid.NewString()
	}
	if err := h.studio.Author(identity, req.Msg.ActorId, core.ActorKind(req.Msg.ActorKind), time.Now()); err != nil {
		return nil, invalid(err)
	}
	if err := h.persist(ctx); err != nil {
		return nil, err
	}
	return connect.NewResponse(&studiov1.CreateIdentityResponse{Identity: toProtoIdentity(h.studio.Identities[identity.ID])}), nil
}

func (h *connectHandler) ReviseIdentity(ctx context.Context, req *connect.Request[studiov1.ReviseIdentityRequest]) (*connect.Response[studiov1.ReviseIdentityResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	identity, err := h.studio.Revise(fromProtoIdentity(req.Msg.Identity), req.Msg.ActorId, core.ActorKind(req.Msg.ActorKind), time.Now())
	if err != nil {
		return nil, invalid(err)
	}
	if err := h.persist(ctx); err != nil {
		return nil, err
	}
	return connect.NewResponse(&studiov1.ReviseIdentityResponse{Identity: toProtoIdentity(identity)}), nil
}

func (h *connectHandler) ResolveSpec(ctx context.Context, req *connect.Request[studiov1.ResolveSpecRequest]) (*connect.Response[studiov1.ResolveSpecResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	spec := core.Spec{ID: uuid.NewString(), Template: req.Msg.Template, Fields: req.Msg.Fields, IdentityVersionIDs: req.Msg.IdentityVersionIds, CampaignRef: req.Msg.CampaignRef}
	payload, err := spec.Resolve(h.studio.Identities)
	if err != nil {
		return nil, invalid(err)
	}
	h.studio.Specs[spec.ID] = spec
	if err := h.persist(ctx); err != nil {
		return nil, err
	}
	return connect.NewResponse(&studiov1.ResolveSpecResponse{SpecId: spec.ID, ResolvedPayload: payload}), nil
}

func (h *connectHandler) CreateRender(ctx context.Context, req *connect.Request[studiov1.CreateRenderRequest]) (*connect.Response[studiov1.CreateRenderResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	spec, ok := h.studio.Specs[req.Msg.SpecId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("spec %q not found", req.Msg.SpecId))
	}
	count := req.Msg.CandidateCount
	if count < 1 {
		count = 1
	}
	render := &core.Render{ID: uuid.NewString(), Status: core.RenderQueued, EstimatedCost: req.Msg.EstimatedCost, ActualCost: req.Msg.EstimatedCost, ActualCostRecorded: true, Provenance: &core.Provenance{SpecID: spec.ID, IdentityVersionIDs: spec.IdentityVersionIDs, Backend: "ai-gateway", Model: "pending", Seed: "pending", Parameters: "resolved"}}
	if err := render.Transition(core.RenderRunning); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := render.Transition(core.RenderSucceeded); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	h.studio.Renders[render.ID] = render
	response := &studiov1.CreateRenderResponse{RenderId: render.ID, Status: string(render.Status)}
	for i := int32(0); i < count; i++ {
		assetID := uuid.NewString()
		blobKey := "renders/" + render.ID + "/" + assetID + ".txt"
		if h.blobs != nil {
			if err := h.blobs.Put(ctx, blobKey, bytes.NewBufferString("asset-studio generated candidate\n"), "text/plain"); err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		}
		a := &core.Asset{ID: assetID, RenderID: render.ID, BlobKey: blobKey, Status: core.Produced, AltText: "Generated asset", Disclosure: "AI-generated disclosure required", AIgenerated: true, IdentityVersionIDs: spec.IdentityVersionIDs}
		h.studio.Assets[a.ID] = a
		render.AssetIDs = append(render.AssetIDs, a.ID)
		response.Candidates = append(response.Candidates, toProtoAsset(a))
	}
	if err := h.persist(ctx); err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) SelectCandidate(ctx context.Context, req *connect.Request[studiov1.SelectCandidateRequest]) (*connect.Response[studiov1.SelectCandidateResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.studio.Select(req.Msg.AssetId); err != nil {
		return nil, invalid(err)
	}
	if err := h.persist(ctx); err != nil {
		return nil, err
	}
	return connect.NewResponse(&studiov1.SelectCandidateResponse{Selected: toProtoAsset(h.studio.Assets[req.Msg.AssetId])}), nil
}

func (h *connectHandler) JudgeConformance(ctx context.Context, req *connect.Request[studiov1.JudgeConformanceRequest]) (*connect.Response[studiov1.JudgeConformanceResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	err := h.studio.Judge(core.Verdict{AssetID: req.Msg.AssetId, IdentityVersionID: req.Msg.IdentityVersionId, ActorID: req.Msg.ActorId, ActorKind: core.ActorKind(req.Msg.ActorKind), Passed: req.Msg.Passed, Basis: core.VerdictBasis(req.Msg.Basis), At: time.Now()})
	if err != nil {
		return nil, invalid(err)
	}
	if err := h.persist(ctx); err != nil {
		return nil, err
	}
	return connect.NewResponse(&studiov1.JudgeConformanceResponse{}), nil
}

func (h *connectHandler) ReleaseAsset(ctx context.Context, req *connect.Request[studiov1.ReleaseAssetRequest]) (*connect.Response[studiov1.ReleaseAssetResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.studio.Release(req.Msg.AssetId); err != nil {
		return nil, invalid(err)
	}
	if err := h.persist(ctx); err != nil {
		return nil, err
	}
	return connect.NewResponse(&studiov1.ReleaseAssetResponse{Asset: toProtoAsset(h.studio.Assets[req.Msg.AssetId])}), nil
}

func (h *connectHandler) GetReleasedAssetReference(_ context.Context, req *connect.Request[studiov1.GetReleasedAssetReferenceRequest]) (*connect.Response[studiov1.GetReleasedAssetReferenceResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	asset := h.studio.Assets[req.Msg.AssetId]
	if asset == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("asset %q not found", req.Msg.AssetId))
	}
	if asset.Status != core.Released {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("asset %q is not released", req.Msg.AssetId))
	}
	return connect.NewResponse(&studiov1.GetReleasedAssetReferenceResponse{Asset: toProtoAsset(asset)}), nil
}

func (h *connectHandler) ImportCanon(ctx context.Context, req *connect.Request[studiov1.ImportCanonRequest]) (*connect.Response[studiov1.ImportCanonResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := h.studio.ImportCanon(req.Msg.Root, "canon-import", time.Now())
	if err := h.persist(ctx); err != nil {
		return nil, err
	}
	return connect.NewResponse(&studiov1.ImportCanonResponse{Created: int32(result.Created), Revised: int32(result.Revised), Errors: result.Errors}), nil
}
func invalid(err error) error { return connect.NewError(connect.CodeInvalidArgument, err) }
func fromProtoIdentity(p *studiov1.Identity) core.Identity {
	if p == nil {
		return core.Identity{}
	}
	refs := make([]core.ConditioningReference, 0, len(p.ConditioningReferences))
	for _, r := range p.ConditioningReferences {
		refs = append(refs, core.ConditioningReference{Kind: r.Kind, ID: r.Id, Version: r.Version})
	}
	return core.Identity{ID: p.Id, Name: p.Name, Kind: core.IdentityKind(p.Kind), Version: int(p.Version), Traits: p.Traits, ReferenceImages: p.ReferenceImages, ConditioningReferences: refs, CredentialClaims: p.CredentialClaims}
}

func toProtoIdentity(i core.Identity) *studiov1.Identity {
	refs := make([]*studiov1.ConditioningReference, 0, len(i.ConditioningReferences))
	for _, r := range i.ConditioningReferences {
		refs = append(refs, &studiov1.ConditioningReference{Kind: r.Kind, Id: r.ID, Version: r.Version})
	}
	return &studiov1.Identity{Id: i.ID, Name: i.Name, Kind: string(i.Kind), Version: int32(i.Version), Traits: i.Traits, ReferenceImages: i.ReferenceImages, ConditioningReferences: refs, CredentialClaims: i.CredentialClaims, Referenced: i.Referenced}
}

func toProtoAsset(a *core.Asset) *studiov1.AssetReference {
	return &studiov1.AssetReference{Id: a.ID, Status: string(a.Status), AltText: a.AltText, Disclosure: a.Disclosure, AiGenerated: a.AIgenerated, MediaType: "image/png"}
}
