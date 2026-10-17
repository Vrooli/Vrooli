package studio

import (
	"context"
	"testing"
	"time"

	core "asset-studio/internal/studio"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/blobstore"
	studiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/studio"
)

type successfulDispatcher struct{}

type (
	successfulAdvisoryAnalyzer struct{}
	successfulCommissioner     struct{}
)

func (successfulCommissioner) CreateTask(context.Context, string, string) (string, error) {
	return "agent-task-1", nil
}

func (successfulAdvisoryAnalyzer) Analyze(context.Context, string) (core.AdvisoryResult, error) {
	return core.AdvisoryResult{Source: "image-tools/quality_assessment", Score: 1, Notes: []string{"fixture"}}, nil
}

func (successfulDispatcher) Dispatch(_ context.Context, req core.RenderDispatchRequest) (core.RenderDispatchResult, error) {
	output := core.RenderOutput{Reference: "image-tools://outputs/fixture.png", MediaType: "image/png", Width: 640, Height: 480}
	backend, model := "image-tools", "fixture-image-model"
	if req.Producer == core.ProducerVideo {
		output = core.RenderOutput{Reference: "ai-gateway://outputs/fixture.mp4", MediaType: "video/mp4"}
		backend, model = "ai-gateway", "fixture-video-model"
	}
	return core.RenderDispatchResult{
		Backend: backend, Model: model, Seed: "42", Parameters: "operation=fixture", RouteReceipt: "fixture-job-1", ActualCost: 0, CostRecorded: true,
		Outputs: []core.RenderOutput{output},
	}, nil
}

func TestProductIdentityToReleasedReference(t *testing.T) { // [REQ:ASSET-P0-004] [REQ:ASSET-P0-005] [REQ:ASSET-P0-009] [REQ:ASSET-P0-014] [REQ:ASSET-P0-015] [REQ:ASSET-P0-017]
	blobs := blobstore.NewMemoryBlobStore()
	h := NewConnectHandlerWithDispatcher(core.New(), nil, blobs, successfulDispatcher{})
	created, err := h.CreateIdentity(context.Background(), connect.NewRequest(&studiov1.CreateIdentityRequest{Identity: &studiov1.Identity{Name: "Vrooli", Kind: "product", Traits: map[string]string{"form": "console", "finish": "slate"}, CredentialClaims: ""}, ActorId: "operator-1", ActorKind: "operator"}))
	if err != nil {
		t.Fatal(err)
	}
	spec, err := h.ResolveSpec(context.Background(), connect.NewRequest(&studiov1.ResolveSpecRequest{Template: "{{product}} launch", Fields: map[string]string{"product": "Vrooli"}, IdentityVersionIds: []string{created.Msg.Identity.Id}}))
	if err != nil {
		t.Fatal(err)
	}
	render, err := h.CreateRender(context.Background(), connect.NewRequest(&studiov1.CreateRenderRequest{SpecId: spec.Msg.SpecId, EstimatedCost: 0.02, CandidateCount: 2}))
	if err != nil {
		t.Fatal(err)
	}
	if render.Msg.Status != string(core.RenderQueued) || len(render.Msg.Candidates) != 0 {
		t.Fatalf("initial render response = %#v", render.Msg)
	}
	requireEventually(t, func() bool { return h.studio.Renders[render.Msg.RenderId].Status == core.RenderSucceeded })
	inspected, err := h.GetRender(context.Background(), connect.NewRequest(&studiov1.GetRenderRequest{RenderId: render.Msg.RenderId}))
	if err != nil {
		t.Fatal(err)
	}
	if inspected.Msg.Render.Provenance.Model != "fixture-image-model" || len(inspected.Msg.Render.Candidates) != 1 {
		t.Fatalf("render inspection = %#v", inspected.Msg.Render)
	}
	assetID := h.studio.Renders[render.Msg.RenderId].AssetIDs[0]
	if got := h.studio.Assets[assetID].BlobKey; got != "image-tools://outputs/fixture.png" {
		t.Fatalf("producer reference = %q", got)
	}
	if _, err := h.SelectCandidate(context.Background(), connect.NewRequest(&studiov1.SelectCandidateRequest{AssetId: assetID})); err != nil {
		t.Fatal(err)
	}
	if _, err := h.ReleaseAsset(context.Background(), connect.NewRequest(&studiov1.ReleaseAssetRequest{AssetId: assetID})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unresolved release error = %v", err)
	}
	if _, err := h.JudgeConformance(context.Background(), connect.NewRequest(&studiov1.JudgeConformanceRequest{AssetId: assetID, IdentityVersionId: created.Msg.Identity.Id, ActorId: "operator-1", ActorKind: "operator", Passed: true, Basis: "prose-only"})); err != nil {
		t.Fatal(err)
	}
	if _, err := h.ReleaseAsset(context.Background(), connect.NewRequest(&studiov1.ReleaseAssetRequest{AssetId: assetID})); err != nil {
		t.Fatal(err)
	}
	ref, err := h.GetReleasedAssetReference(context.Background(), connect.NewRequest(&studiov1.GetReleasedAssetReferenceRequest{AssetId: assetID}))
	if err != nil {
		t.Fatal(err)
	}
	if ref.Msg.Asset.Disclosure == "" {
		t.Fatal("released reference omitted disclosure")
	}
}

func TestCreateRenderPersistsMultiFrameProducerIntent(t *testing.T) { // [REQ:ASSET-P1-001] [REQ:ASSET-P1-002] [REQ:ASSET-P1-004] [REQ:ASSET-P1-013]
	h := NewConnectHandlerWithDispatcher(core.New(), nil, nil, successfulDispatcher{})
	identity, err := h.CreateIdentity(context.Background(), connect.NewRequest(&studiov1.CreateIdentityRequest{Identity: &studiov1.Identity{Name: "Vrooli", Kind: "product", Traits: map[string]string{"form": "console", "finish": "slate"}}, ActorId: "operator", ActorKind: "operator"}))
	if err != nil {
		t.Fatal(err)
	}
	spec, err := h.ResolveSpec(context.Background(), connect.NewRequest(&studiov1.ResolveSpecRequest{Template: "{{product}} launch", Fields: map[string]string{"product": "Vrooli"}, IdentityVersionIds: []string{identity.Msg.Identity.Id}}))
	if err != nil {
		t.Fatal(err)
	}
	created, err := h.CreateRender(context.Background(), connect.NewRequest(&studiov1.CreateRenderRequest{SpecId: spec.Msg.SpecId, CandidateCount: 1, ProducerKind: "video", FrameCount: 12}))
	if err != nil {
		t.Fatal(err)
	}
	requireEventually(t, func() bool { return h.studio.Renders[created.Msg.RenderId].Status == core.RenderSucceeded })
	got, err := h.GetRender(context.Background(), connect.NewRequest(&studiov1.GetRenderRequest{RenderId: created.Msg.RenderId}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.Render.ProducerKind != "video" || got.Msg.Render.FrameCount != 12 || got.Msg.Render.Provenance.IdentityVersionIds[0] != identity.Msg.Identity.Id {
		t.Fatalf("media render = %#v", got.Msg.Render)
	}
}

func TestRefinementLinksParentAndRequiresItsOwnReview(t *testing.T) { // [REQ:ASSET-P1-013]
	h := NewConnectHandlerWithDispatcher(core.New(), nil, nil, successfulDispatcher{})
	identity, err := h.CreateIdentity(context.Background(), connect.NewRequest(&studiov1.CreateIdentityRequest{Identity: &studiov1.Identity{Name: "Vrooli", Kind: "product", Traits: map[string]string{"form": "console", "finish": "slate"}}, ActorId: "operator", ActorKind: "operator"}))
	if err != nil {
		t.Fatal(err)
	}
	spec, err := h.ResolveSpec(context.Background(), connect.NewRequest(&studiov1.ResolveSpecRequest{Template: "{{name}}", Fields: map[string]string{"name": "Vrooli"}, IdentityVersionIds: []string{identity.Msg.Identity.Id}}))
	if err != nil {
		t.Fatal(err)
	}
	parent, err := h.CreateRender(context.Background(), connect.NewRequest(&studiov1.CreateRenderRequest{SpecId: spec.Msg.SpecId}))
	if err != nil {
		t.Fatal(err)
	}
	requireEventually(t, func() bool { return h.studio.Renders[parent.Msg.RenderId].Status == core.RenderSucceeded })
	parentID := h.studio.Renders[parent.Msg.RenderId].AssetIDs[0]
	child, err := h.CreateRender(context.Background(), connect.NewRequest(&studiov1.CreateRenderRequest{SpecId: spec.Msg.SpecId, ProducerKind: "refine", ParentAssetId: parentID}))
	if err != nil {
		t.Fatal(err)
	}
	requireEventually(t, func() bool { return h.studio.Renders[child.Msg.RenderId].Status == core.RenderSucceeded })
	asset := h.studio.Assets[h.studio.Renders[child.Msg.RenderId].AssetIDs[0]]
	if asset.ParentAssetID != parentID || asset.Status != core.Produced {
		t.Fatalf("refined asset=%#v", asset)
	}
	if err := h.studio.Release(asset.ID); err == nil {
		t.Fatal("refined asset must not inherit parent release authority")
	}
}

func TestRegenerateRenderCreatesFreshReceiptFromSuccessfulProvenance(t *testing.T) { // [REQ:ASSET-P1-005]
	h := NewConnectHandlerWithDispatcher(core.New(), nil, nil, successfulDispatcher{})
	identity, err := h.CreateIdentity(context.Background(), connect.NewRequest(&studiov1.CreateIdentityRequest{Identity: &studiov1.Identity{Name: "Vrooli", Kind: "product", Traits: map[string]string{"form": "console", "finish": "slate"}}, ActorId: "operator", ActorKind: "operator"}))
	if err != nil {
		t.Fatal(err)
	}
	spec, err := h.ResolveSpec(context.Background(), connect.NewRequest(&studiov1.ResolveSpecRequest{Template: "{{name}} launch", Fields: map[string]string{"name": "Vrooli"}, IdentityVersionIds: []string{identity.Msg.Identity.Id}}))
	if err != nil {
		t.Fatal(err)
	}
	source, err := h.CreateRender(context.Background(), connect.NewRequest(&studiov1.CreateRenderRequest{SpecId: spec.Msg.SpecId, CandidateCount: 1}))
	if err != nil {
		t.Fatal(err)
	}
	requireEventually(t, func() bool { return h.studio.Renders[source.Msg.RenderId].Status == core.RenderSucceeded })
	regenerated, err := h.RegenerateRender(context.Background(), connect.NewRequest(&studiov1.RegenerateRenderRequest{SourceRenderId: source.Msg.RenderId}))
	if err != nil {
		t.Fatal(err)
	}
	if regenerated.Msg.SourceRenderId != source.Msg.RenderId || regenerated.Msg.RenderId == source.Msg.RenderId {
		t.Fatalf("regeneration receipt = %#v", regenerated.Msg)
	}
	requireEventually(t, func() bool { return h.studio.Renders[regenerated.Msg.RenderId].Status == core.RenderSucceeded })
	if got, want := h.studio.Renders[regenerated.Msg.RenderId].Prompt, h.studio.Renders[source.Msg.RenderId].Prompt; got != want {
		t.Fatalf("regenerated prompt = %q, want %q", got, want)
	}
}

func TestCampaignBudgetRequiresRecordedConfirmation(t *testing.T) { // [REQ:ASSET-P1-006]
	h := NewConnectHandlerWithDispatcher(core.New(), nil, nil, successfulDispatcher{})
	identity, err := h.CreateIdentity(context.Background(), connect.NewRequest(&studiov1.CreateIdentityRequest{Identity: &studiov1.Identity{Name: "Vrooli", Kind: "product", Traits: map[string]string{"form": "console", "finish": "slate"}}, ActorId: "operator", ActorKind: "operator"}))
	if err != nil {
		t.Fatal(err)
	}
	spec, err := h.ResolveSpec(context.Background(), connect.NewRequest(&studiov1.ResolveSpecRequest{Template: "{{name}}", Fields: map[string]string{"name": "Vrooli"}, IdentityVersionIds: []string{identity.Msg.Identity.Id}, CampaignRef: "launch"}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = h.SetCampaignBudget(context.Background(), connect.NewRequest(&studiov1.SetCampaignBudgetRequest{CampaignRef: "launch", LimitUsd: 0.01})); err != nil {
		t.Fatal(err)
	}
	if _, err = h.CreateRender(context.Background(), connect.NewRequest(&studiov1.CreateRenderRequest{SpecId: spec.Msg.SpecId, EstimatedCost: 0.02})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unconfirmed budget error = %v", err)
	}
	if _, err = h.CreateRender(context.Background(), connect.NewRequest(&studiov1.CreateRenderRequest{SpecId: spec.Msg.SpecId, EstimatedCost: 0.02, ConfirmOverBudget: true, BudgetConfirmationActorId: "operator"})); err != nil {
		t.Fatal(err)
	}
	if got := len(h.studio.CampaignBudgets["launch"].Confirmations); got != 1 {
		t.Fatalf("confirmations = %d", got)
	}
}

func TestAdvisoryConformanceNeverReplacesOperatorVerdict(t *testing.T) { // [REQ:ASSET-P1-010]
	h := NewConnectHandlerWithDispatcher(core.New(), nil, nil, successfulDispatcher{})
	h.SetAdvisoryAnalyzer(successfulAdvisoryAnalyzer{})
	identity, err := h.CreateIdentity(context.Background(), connect.NewRequest(&studiov1.CreateIdentityRequest{Identity: &studiov1.Identity{Name: "Vrooli", Kind: "product", Traits: map[string]string{"form": "console", "finish": "slate"}}, ActorId: "operator", ActorKind: "operator"}))
	if err != nil {
		t.Fatal(err)
	}
	spec, err := h.ResolveSpec(context.Background(), connect.NewRequest(&studiov1.ResolveSpecRequest{Template: "{{name}}", Fields: map[string]string{"name": "Vrooli"}, IdentityVersionIds: []string{identity.Msg.Identity.Id}}))
	if err != nil {
		t.Fatal(err)
	}
	render, err := h.CreateRender(context.Background(), connect.NewRequest(&studiov1.CreateRenderRequest{SpecId: spec.Msg.SpecId}))
	if err != nil {
		t.Fatal(err)
	}
	requireEventually(t, func() bool { return h.studio.Renders[render.Msg.RenderId].Status == core.RenderSucceeded })
	assetID := h.studio.Renders[render.Msg.RenderId].AssetIDs[0]
	if _, err = h.SelectCandidate(context.Background(), connect.NewRequest(&studiov1.SelectCandidateRequest{AssetId: assetID})); err != nil {
		t.Fatal(err)
	}
	advisory, err := h.AnalyzeConformance(context.Background(), connect.NewRequest(&studiov1.AnalyzeConformanceRequest{AssetId: assetID}))
	if err != nil || advisory.Msg.GetAdvisory().GetScore() != 1 {
		t.Fatalf("advisory = %#v, err=%v", advisory.Msg, err)
	}
	if _, err = h.ReleaseAsset(context.Background(), connect.NewRequest(&studiov1.ReleaseAssetRequest{AssetId: assetID})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("advisory bypassed operator verdict: %v", err)
	}
}

func TestCommissionAgentPersistsUntrustedProposal(t *testing.T) { // [REQ:ASSET-P1-011]
	h := NewConnectHandlerWithDispatcher(core.New(), nil, nil, successfulDispatcher{})
	h.SetAgentCommissioner(successfulCommissioner{})
	identity, err := h.CreateIdentity(context.Background(), connect.NewRequest(&studiov1.CreateIdentityRequest{Identity: &studiov1.Identity{Name: "Vrooli", Kind: "product", Traits: map[string]string{"form": "console", "finish": "slate"}}, ActorId: "operator", ActorKind: "operator"}))
	if err != nil {
		t.Fatal(err)
	}
	got, err := h.CommissionAgent(context.Background(), connect.NewRequest(&studiov1.CommissionAgentRequest{Request: "propose campaign concepts", AgentIdentity: "marketing-agent", SourceIdentityVersionIds: []string{identity.Msg.Identity.Id}}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetCommission().GetAgentTaskId() != "agent-task-1" || got.Msg.GetCommission().GetStatus() != "proposed" {
		t.Fatalf("commission=%#v", got.Msg.GetCommission())
	}
	if len(h.studio.Commissions) != 1 {
		t.Fatal("commission was not persisted")
	}
}

func requireEventually(t *testing.T, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if done() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition did not complete")
}
