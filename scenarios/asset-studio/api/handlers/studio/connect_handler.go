package studio

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	core "asset-studio/internal/studio"
	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/blobstore"
	studiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/studio"
)

type connectHandler struct {
	mu           sync.Mutex
	studio       *core.Studio
	store        core.StateStore
	blobs        blobstore.BlobStore
	dispatcher   core.RenderDispatcher
	analyzer     core.AdvisoryAnalyzer
	commissioner AgentCommissioner
}
type AgentCommissioner interface {
	CreateTask(context.Context, string, string) (string, error)
}
type unavailableCommissioner struct{}

func (unavailableCommissioner) CreateTask(context.Context, string, string) (string, error) {
	return "", fmt.Errorf("Agent Manager commissioner is not configured")
}

func NewConnectHandler(s *core.Studio) *connectHandler {
	return &connectHandler{studio: s, dispatcher: core.UnavailableRenderDispatcher{}, analyzer: core.UnavailableAdvisoryAnalyzer{}, commissioner: unavailableCommissioner{}}
}

func NewConnectHandlerWithStore(s *core.Studio, store core.StateStore, blobs blobstore.BlobStore) *connectHandler {
	return NewConnectHandlerWithDispatcher(s, store, blobs, core.UnavailableRenderDispatcher{})
}

func NewConnectHandlerWithDispatcher(s *core.Studio, store core.StateStore, blobs blobstore.BlobStore, dispatcher core.RenderDispatcher) *connectHandler {
	if dispatcher == nil {
		dispatcher = core.UnavailableRenderDispatcher{}
	}
	return &connectHandler{studio: s, store: store, blobs: blobs, dispatcher: dispatcher, analyzer: core.UnavailableAdvisoryAnalyzer{}, commissioner: unavailableCommissioner{}}
}

func (h *connectHandler) SetAdvisoryAnalyzer(analyzer core.AdvisoryAnalyzer) {
	if analyzer == nil {
		analyzer = core.UnavailableAdvisoryAnalyzer{}
	}
	h.analyzer = analyzer
}

func (h *connectHandler) SetAgentCommissioner(c AgentCommissioner) {
	if c == nil {
		c = unavailableCommissioner{}
	}
	h.commissioner = c
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
	if err := h.studio.AuthorizeRender(spec.CampaignRef, req.Msg.EstimatedCost, req.Msg.GetConfirmOverBudget(), req.Msg.GetBudgetConfirmationActorId(), time.Now()); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	count := req.Msg.CandidateCount
	if count < 1 {
		count = 1
	}
	producer := core.ProducerKind(req.Msg.GetProducerKind())
	if producer == "" {
		producer = core.ProducerImage
	}
	if !producer.Valid() {
		return nil, invalid(fmt.Errorf("unsupported producer kind %q", producer))
	}
	if producer == core.ProducerCapture && req.Msg.GetCaptureUrl() == "" {
		return nil, invalid(fmt.Errorf("capture_url is required for capture producer"))
	}
	if producer == core.ProducerCompose {
		seen := map[string]bool{}
		for _, slot := range req.Msg.GetCompositionSlots() {
			if slot.GetName() == "" || slot.GetAssetId() == "" || seen[slot.GetName()] || h.studio.Assets[slot.GetAssetId()] == nil {
				return nil, invalid(fmt.Errorf("composition slots require unique names and existing asset ids"))
			}
			seen[slot.GetName()] = true
		}
		if len(seen) == 0 {
			return nil, invalid(fmt.Errorf("composition producer requires named slots"))
		}
	}
	parentReference := ""
	if producer == core.ProducerRefine {
		parent := h.studio.Assets[req.Msg.GetParentAssetId()]
		if parent == nil {
			return nil, invalid(fmt.Errorf("refinement parent asset %q not found", req.Msg.GetParentAssetId()))
		}
		parentReference = parent.BlobKey
	}
	frameCount := int(req.Msg.GetFrameCount())
	if frameCount < 1 {
		frameCount = 1
	}
	prompt, err := spec.Resolve(h.studio.Identities)
	if err != nil {
		return nil, invalid(err)
	}
	conditioning := make([]core.ConditioningReference, 0)
	for _, identityID := range spec.IdentityVersionIDs {
		conditioning = append(conditioning, h.studio.Identities[identityID].ConditioningReferences...)
	}
	render := &core.Render{ID: uuid.NewString(), Status: core.RenderQueued, EstimatedCost: req.Msg.EstimatedCost, Prompt: prompt, CandidateCount: int(count), Producer: producer, FrameCount: frameCount, ParentAssetID: req.Msg.GetParentAssetId(), ParentAssetReference: parentReference, CaptureURL: req.Msg.GetCaptureUrl(), ConditioningReferences: conditioning, Provenance: &core.Provenance{SpecID: spec.ID, IdentityVersionIDs: spec.IdentityVersionIDs, Parameters: "resolved"}}
	h.studio.Renders[render.ID] = render
	if err := h.persist(ctx); err != nil {
		return nil, err
	}
	go h.executeRender(render.ID)
	return connect.NewResponse(&studiov1.CreateRenderResponse{RenderId: render.ID, Status: string(render.Status)}), nil
}

// RegenerateRender creates a fresh receipt from a successful render's stored,
// resolved intent. It intentionally does not copy outputs or treat a policy
// routed execution as a claim that the original provider/model remains
// available: the new receipt records the newly resolved producer outcome.
func (h *connectHandler) RegenerateRender(ctx context.Context, req *connect.Request[studiov1.RegenerateRenderRequest]) (*connect.Response[studiov1.RegenerateRenderResponse], error) {
	h.mu.Lock()
	source := h.studio.Renders[req.Msg.GetSourceRenderId()]
	if source == nil {
		h.mu.Unlock()
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("source render %q not found", req.Msg.GetSourceRenderId()))
	}
	if source.Status != core.RenderSucceeded || source.Provenance == nil || !source.ActualCostRecorded {
		h.mu.Unlock()
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("source render %q does not have successful recorded provenance", source.ID))
	}
	if _, ok := h.studio.Specs[source.Provenance.SpecID]; !ok {
		h.mu.Unlock()
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("source render %q references unavailable spec %q", source.ID, source.Provenance.SpecID))
	}
	for _, identityID := range source.Provenance.IdentityVersionIDs {
		if _, ok := h.studio.Identities[identityID]; !ok {
			h.mu.Unlock()
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("source render %q references unavailable identity version %q", source.ID, identityID))
		}
	}
	spec := h.studio.Specs[source.Provenance.SpecID]
	if err := h.studio.AuthorizeRender(spec.CampaignRef, source.EstimatedCost, req.Msg.GetConfirmOverBudget(), req.Msg.GetBudgetConfirmationActorId(), time.Now()); err != nil {
		h.mu.Unlock()
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	provenance := *source.Provenance
	provenance.IdentityVersionIDs = append([]string(nil), source.Provenance.IdentityVersionIDs...)
	render := &core.Render{ID: uuid.NewString(), Status: core.RenderQueued, EstimatedCost: source.EstimatedCost, Prompt: source.Prompt, CandidateCount: source.CandidateCount, Producer: source.Producer, FrameCount: source.FrameCount, ParentAssetID: source.ParentAssetID, ParentAssetReference: source.ParentAssetReference, CaptureURL: source.CaptureURL, ConditioningReferences: append([]core.ConditioningReference(nil), source.ConditioningReferences...), Provenance: &provenance}
	h.studio.Renders[render.ID] = render
	if err := h.persist(ctx); err != nil {
		h.mu.Unlock()
		return nil, err
	}
	h.mu.Unlock()
	go h.executeRender(render.ID)
	return connect.NewResponse(&studiov1.RegenerateRenderResponse{RenderId: render.ID, Status: string(render.Status), SourceRenderId: source.ID}), nil
}

func (h *connectHandler) AnalyzeConformance(ctx context.Context, req *connect.Request[studiov1.AnalyzeConformanceRequest]) (*connect.Response[studiov1.AnalyzeConformanceResponse], error) {
	h.mu.Lock()
	asset := h.studio.Assets[req.Msg.GetAssetId()]
	if asset == nil {
		h.mu.Unlock()
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("asset %q not found", req.Msg.GetAssetId()))
	}
	reference := asset.BlobKey
	analyzer := h.analyzer
	h.mu.Unlock()
	result, err := analyzer.Analyze(ctx, reference)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.studio.RecordAdvisory(core.AdvisoryConformance{AssetID: asset.ID, Source: result.Source, Score: result.Score, Notes: result.Notes, At: time.Now()}); err != nil {
		return nil, invalid(err)
	}
	if err := h.persist(ctx); err != nil {
		return nil, err
	}
	advisory := h.studio.Advisories[len(h.studio.Advisories)-1]
	return connect.NewResponse(&studiov1.AnalyzeConformanceResponse{Advisory: &studiov1.AdvisoryConformance{AssetId: advisory.AssetID, Source: advisory.Source, Score: advisory.Score, Notes: advisory.Notes, RecordedAt: advisory.At.Format(time.RFC3339Nano)}}), nil
}

func (h *connectHandler) SetCampaignBudget(ctx context.Context, req *connect.Request[studiov1.SetCampaignBudgetRequest]) (*connect.Response[studiov1.SetCampaignBudgetResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.studio.SetCampaignBudget(req.Msg.GetCampaignRef(), req.Msg.GetLimitUsd()); err != nil {
		return nil, invalid(err)
	}
	if err := h.persist(ctx); err != nil {
		return nil, err
	}
	budget := h.studio.CampaignBudgets[req.Msg.GetCampaignRef()]
	return connect.NewResponse(&studiov1.SetCampaignBudgetResponse{Budget: &studiov1.CampaignBudget{CampaignRef: budget.CampaignRef, LimitUsd: budget.LimitUSD, SpentUsd: budget.SpentUSD}}), nil
}

func (h *connectHandler) GetRender(_ context.Context, req *connect.Request[studiov1.GetRenderRequest]) (*connect.Response[studiov1.GetRenderResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	render := h.studio.Renders[req.Msg.GetRenderId()]
	if render == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("render %q not found", req.Msg.GetRenderId()))
	}
	return connect.NewResponse(&studiov1.GetRenderResponse{Render: toProtoRender(h.studio, render)}), nil
}

// ReconcileRenders resumes queued and interrupted jobs after a process restart.
// The persisted specification is the source of truth; no render is recreated
// and no success is inferred from a missing producer receipt.
func (h *connectHandler) ReconcileRenders() {
	h.mu.Lock()
	ids := make([]string, 0)
	for id, render := range h.studio.Renders {
		if render != nil && (render.Status == core.RenderQueued || render.Status == core.RenderRunning) {
			ids = append(ids, id)
		}
	}
	h.mu.Unlock()
	for _, id := range ids {
		go h.executeRender(id)
	}
}

func (h *connectHandler) executeRender(id string) {
	h.mu.Lock()
	render := h.studio.Renders[id]
	if render == nil || render.Status.Terminal() {
		h.mu.Unlock()
		return
	}
	if render.Status == core.RenderQueued {
		if err := render.Transition(core.RenderRunning); err != nil {
			h.mu.Unlock()
			return
		}
	}
	_ = h.persist(context.Background())
	req := core.RenderDispatchRequest{RenderID: render.ID, Prompt: render.Prompt, CandidateCount: render.CandidateCount, Producer: render.Producer, FrameCount: render.FrameCount, ParentAssetID: render.ParentAssetID, ParentReference: render.ParentAssetReference, CaptureURL: render.CaptureURL, ConditioningReferences: append([]core.ConditioningReference(nil), render.ConditioningReferences...)}
	h.mu.Unlock()

	result, err := h.dispatcher.Dispatch(context.Background(), req)
	h.mu.Lock()
	defer h.mu.Unlock()
	render = h.studio.Renders[id]
	if render == nil || render.Status != core.RenderRunning {
		return
	}
	if err != nil {
		render.ActualCost = 0
		render.ActualCostRecorded = true
		render.FailureCode = "producer_failed"
		_ = render.Transition(core.RenderFailed)
		_ = h.persist(context.Background())
		return
	}
	if !result.CostRecorded || result.Backend == "" || result.Model == "" || len(result.Outputs) == 0 {
		render.ActualCost = 0
		render.ActualCostRecorded = true
		render.FailureCode = "incomplete_producer_receipt"
		_ = render.Transition(core.RenderFailed)
		_ = h.persist(context.Background())
		return
	}
	render.ActualCost = result.ActualCost
	render.ActualCostRecorded = true
	if spec := h.studio.Specs[render.Provenance.SpecID]; spec.ID != "" {
		h.studio.RecordRenderSpend(spec.CampaignRef, result.ActualCost)
	}
	render.Provenance.Backend = result.Backend
	render.Provenance.Model = result.Model
	render.Provenance.Seed = result.Seed
	render.Provenance.Parameters = result.Parameters
	for _, output := range result.Outputs {
		asset := &core.Asset{ID: uuid.NewString(), RenderID: render.ID, BlobKey: output.Reference, Status: core.Produced, AltText: "Generated asset", Disclosure: "AI-generated disclosure required", AIgenerated: true, IdentityVersionIDs: render.Provenance.IdentityVersionIDs, MediaType: output.MediaType, Width: output.Width, Height: output.Height, ParentAssetID: render.ParentAssetID, DerivationOperation: string(render.Producer)}
		h.studio.Assets[asset.ID] = asset
		render.AssetIDs = append(render.AssetIDs, asset.ID)
	}
	if err := render.Transition(core.RenderSucceeded); err != nil {
		render.FailureCode = "render_transition_failed"
	}
	_ = h.persist(context.Background())
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

// IngestExternalAsset admits bytes another scenario produced.
//
// The bytes go through the same blob seam every other asset uses, and the
// resulting record enters `in_review` — not `released`. That is the whole
// design: this is a door into the existing release path, not a way around it.
// An ingested asset still needs its operator verdict and still passes every
// check in Release, so admitting it cannot be used to publish something the
// normal path would have refused.
func (h *connectHandler) IngestExternalAsset(ctx context.Context, req *connect.Request[studiov1.IngestExternalAssetRequest]) (*connect.Response[studiov1.IngestExternalAssetResponse], error) {
	msg := req.Msg
	if len(msg.GetImage()) == 0 {
		return nil, invalid(fmt.Errorf("ingest requires image bytes"))
	}
	provenance := msg.GetProvenance()
	if provenance == nil {
		return nil, invalid(fmt.Errorf("ingest requires provenance: an asset with no recorded origin cannot be disclosed"))
	}
	mediaType := strings.TrimSpace(msg.GetMediaType())
	if mediaType == "" {
		mediaType = "image/png"
	}

	assetID := uuid.NewString()
	key := "external/" + assetID
	// The bytes are stored before the record is written. The other order would
	// leave a record pointing at a blob that failed to save, which reads as a
	// released asset whose bytes cannot be fetched — the one failure mode a
	// provenance store must not have.
	if h.blobs == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("asset blob store is unavailable"))
	}
	if err := h.blobs.Put(ctx, key, bytes.NewReader(msg.GetImage()), mediaType); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("store ingested bytes: %w", err))
	}

	conditioning := core.ConditioningReference{}
	if c := provenance.GetConditioning(); c != nil {
		conditioning = core.ConditioningReference{Kind: c.GetKind(), ID: c.GetId(), Version: c.GetVersion()}
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	asset, err := h.studio.Ingest(assetID, core.IngestRequest{
		BlobKey:    key,
		MediaType:  mediaType,
		AltText:    msg.GetAltText(),
		Decorative: msg.GetDecorative(),
		Width:      int(msg.GetWidth()),
		Height:     int(msg.GetHeight()),
		Provenance: core.ExternalProvenance{
			ProducingScenario: provenance.GetProducingScenario(),
			Strategy:          provenance.GetStrategy(),
			ModelBacked:       provenance.GetModelBacked(),
			Model:             provenance.GetModel(),
			Prompt:            provenance.GetPrompt(),
			NegativePrompt:    provenance.GetNegativePrompt(),
			Seed:              provenance.GetSeed(),
			Conditioning:      conditioning,
			Parameters:        provenance.GetParameters(),
		},
	})
	if err != nil {
		// The stored bytes are removed: a refused ingest that left its blob
		// behind would accumulate orphans that no record ever names, which is
		// indistinguishable from a leak.
		if delErr := h.blobs.Delete(ctx, key); delErr != nil {
			log.Printf("ingest rollback: delete orphaned blob %s: %v", key, delErr)
		}
		return nil, invalid(err)
	}
	if err := h.persist(ctx); err != nil {
		return nil, err
	}
	return connect.NewResponse(&studiov1.IngestExternalAssetResponse{Asset: toProtoAsset(asset)}), nil
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

func (h *connectHandler) CommissionAgent(ctx context.Context, req *connect.Request[studiov1.CommissionAgentRequest]) (*connect.Response[studiov1.CommissionAgentResponse], error) {
	h.mu.Lock()
	for _, id := range req.Msg.GetSourceIdentityVersionIds() {
		if _, ok := h.studio.Identities[id]; !ok {
			h.mu.Unlock()
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("identity version %q is unavailable", id))
		}
	}
	c := h.commissioner
	h.mu.Unlock()
	taskID, err := c.CreateTask(ctx, req.Msg.GetRequest(), req.Msg.GetAgentIdentity())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	commission := core.AgentCommission{ID: uuid.NewString(), AgentTaskID: taskID, AgentIdentity: req.Msg.GetAgentIdentity(), Request: req.Msg.GetRequest(), SourceIdentityVersionIDs: req.Msg.GetSourceIdentityVersionIds(), Status: "proposed", At: time.Now()}
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.studio.RecordCommission(commission); err != nil {
		return nil, invalid(err)
	}
	if err := h.persist(ctx); err != nil {
		return nil, err
	}
	return connect.NewResponse(&studiov1.CommissionAgentResponse{Commission: &studiov1.AgentCommission{Id: commission.ID, AgentTaskId: commission.AgentTaskID, AgentIdentity: commission.AgentIdentity, Request: commission.Request, SourceIdentityVersionIds: commission.SourceIdentityVersionIDs, Status: commission.Status, CreatedAt: commission.At.Format(time.RFC3339Nano)}}), nil
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
	return &studiov1.AssetReference{Id: a.ID, Status: string(a.Status), AltText: a.AltText, Disclosure: a.Disclosure, AiGenerated: a.AIgenerated, MediaType: a.MediaType, Width: int32(a.Width), Height: int32(a.Height), ParentAssetId: a.ParentAssetID, DerivationOperation: a.DerivationOperation}
}

func toProtoRender(state *core.Studio, render *core.Render) *studiov1.Render {
	if render == nil {
		return nil
	}
	out := &studiov1.Render{Id: render.ID, Status: string(render.Status), EstimatedCost: render.EstimatedCost, ActualCost: render.ActualCost, ActualCostRecorded: render.ActualCostRecorded, FailureCode: render.FailureCode, ProducerKind: string(render.Producer), FrameCount: int32(render.FrameCount), ParentAssetId: render.ParentAssetID}
	if render.Provenance != nil {
		out.Provenance = &studiov1.RenderProvenance{SpecId: render.Provenance.SpecID, IdentityVersionIds: render.Provenance.IdentityVersionIDs, Backend: render.Provenance.Backend, Model: render.Provenance.Model, Seed: render.Provenance.Seed, Parameters: render.Provenance.Parameters}
	}
	for _, id := range render.AssetIDs {
		if asset := state.Assets[id]; asset != nil {
			out.Candidates = append(out.Candidates, toProtoAsset(asset))
		}
	}
	return out
}
