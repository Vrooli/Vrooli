package studio

import (
	"context"
	"io"
	"testing"

	core "asset-studio/internal/studio"
	"connectrpc.com/connect"
	"github.com/vrooli/api-core/blobstore"
	studiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/studio"
)

func TestProductIdentityToReleasedReference(t *testing.T) {
	blobs := blobstore.NewMemoryBlobStore()
	h := NewConnectHandlerWithStore(core.New(), nil, blobs)
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
	assetID := render.Msg.Candidates[0].Id
	reader, mime, err := blobs.Get(context.Background(), h.studio.Assets[assetID].BlobKey)
	if err != nil || mime != "text/plain" {
		t.Fatalf("candidate blob = %v, %q", err, mime)
	}
	body, _ := io.ReadAll(reader)
	_ = reader.Close()
	if string(body) != "asset-studio generated candidate\n" {
		t.Fatalf("candidate bytes = %q", body)
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
