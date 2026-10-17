package preview

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"
	"google.golang.org/protobuf/types/known/structpb"
	"react-component-library/internal/deps"
	domain "react-component-library/internal/preview"
)

type compositionRenderService struct{ domain.Service }

func (compositionRenderService) GetCompositionBundle(_ context.Context, c domain.Composition) (domain.CompositionBundle, error) {
	return domain.CompositionBundle{Bundle: domain.Bundle{JS: "export default function Composition(){return null}", SHA256: "source-hash"}, Revision: c.Revision}, nil
}
func TestCompositionRenderIdentityTracksFixturesAndAppearance(t *testing.T) {
	h := NewConnectHandler(Deps{Service: compositionRenderService{}})
	bindings, err := structpb.NewStruct(map[string]any{"$template": map[string]any{}, "$labels": map[string]any{"missing": "Missing", "failed": "Failed"}})
	require.NoError(t, err)
	request := &previewv1.RenderCompositionRequest{Composition: &previewv1.GetCompositionBundleRequest{Revision: "revision-a", Template: &previewv1.CompositionAsset{CatalogId: "templates.collection-page", Version: "1.0.7", Export: "CollectionPage"}}, Bindings: bindings, Kit: "none", Theme: "light", Direction: "ltr"}
	render := func() *previewv1.RenderCompositionResponse {
		t.Helper()
		r, e := h.RenderComposition(context.Background(), connect.NewRequest(request))
		require.NoError(t, e)
		return r.Msg
	}
	first := render()
	require.Equal(t, first.RenderHash, render().RenderHash)
	require.Equal(t, "revision-a", first.Target.Revision)
	require.Equal(t, "preview", first.Target.Kind)
	require.Equal(t, first.RenderHash, first.Target.RenderHash)
	require.Contains(t, first.Html, `<meta name="bundle-sha256" content="`+first.RenderHash+`" />`)
	digest := sha256.Sum256([]byte(first.Html))
	require.Equal(t, hex.EncodeToString(digest[:]), first.Target.HtmlSha256)
	originalHarness := harnessJavaScript
	t.Cleanup(func() { harnessJavaScript = originalHarness })
	harnessJavaScript += "\n/* renderer revision */"
	changedRenderer := render()
	require.NotEqual(t, first.RenderHash, changedRenderer.RenderHash)
	require.NotEqual(t, first.Target.HtmlSha256, changedRenderer.Target.HtmlSha256)
	require.Equal(t, first.Target.InputsSha256, changedRenderer.Target.InputsSha256)
	require.Len(t, first.Target.InputHashes, 11)
	require.NotEqual(t, first.Target.InputHashes["harness"], changedRenderer.Target.InputHashes["harness"])
	require.Equal(t, first.Target.InputHashes["composition"], changedRenderer.Target.InputHashes["composition"])
	require.Equal(t, first.Target.InputHashes["bundle"], changedRenderer.Target.InputHashes["bundle"])
	harnessJavaScript = originalHarness
	require.Equal(t, first.RenderHash, render().RenderHash)
	require.Contains(t, first.Html, "connect-src 'none'")
	require.Contains(t, first.Html, "form-action 'none'")
	request.Theme = ""
	request.Direction = ""
	require.Equal(t, first.RenderHash, render().RenderHash)
	request.Direction = "ltr"
	request.Theme = "dark"
	require.NotEqual(t, first.RenderHash, render().RenderHash)
	request.Theme = "light"
	request.Direction = "rtl"
	require.NotEqual(t, first.RenderHash, render().RenderHash)
	request.Direction = "ltr"
	request.Bindings.Fields["$labels"].GetStructValue().Fields["missing"] = structpb.NewStringValue("Unavailable")
	require.NotEqual(t, first.RenderHash, render().RenderHash)
	request.Bindings = nil
	_, err = h.RenderComposition(context.Background(), connect.NewRequest(request))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestCompositionRenderIdentityStableAcrossRepeatedRequests(t *testing.T) {
	h := NewConnectHandler(Deps{Service: compositionRenderService{}})
	bindings, err := structpb.NewStruct(map[string]any{"$template": map[string]any{"data": map[string]any{"items": []any{"One", "Two"}}}, "gate": map[string]any{}, "$labels": map[string]any{"missing": "Missing fixture", "failed": "Region failed"}})
	require.NoError(t, err)
	request := &previewv1.RenderCompositionRequest{Composition: &previewv1.GetCompositionBundleRequest{Revision: "candidate", Template: &previewv1.CompositionAsset{CatalogId: "templates.collection-page", Version: "1.1.0", Export: "CollectionPage"}, Regions: []*previewv1.CompositionRegion{{Id: "gate", Slot: []string{"regions", "header"}, Asset: &previewv1.CompositionAsset{CatalogId: "feedback.permission-state", Version: "1.0.7", Export: "PermissionState"}, Required: true}}}, Bindings: bindings, Kit: "none", Theme: "light", Direction: "ltr"}
	first, err := h.RenderComposition(context.Background(), connect.NewRequest(request))
	require.NoError(t, err)
	for i := 0; i < 128; i++ {
		next, err := h.RenderComposition(context.Background(), connect.NewRequest(request))
		require.NoError(t, err)
		require.Equal(t, first.Msg.Target.InputsSha256, next.Msg.Target.InputsSha256, "inputs changed at request %d", i)
		require.Equal(t, first.Msg.RenderHash, next.Msg.RenderHash, "render identity changed at request %d", i)
		require.Equal(t, first.Msg.Target.HtmlSha256, next.Msg.Target.HtmlSha256, "HTML changed at request %d", i)
	}
}

type parsedDependencyRenderService struct{ compositionRenderService }

func (s parsedDependencyRenderService) GetCompositionBundle(ctx context.Context, c domain.Composition) (domain.CompositionBundle, error) {
	bundle, err := s.compositionRenderService.GetCompositionBundle(ctx, c)
	if err != nil {
		return bundle, err
	}
	declarations, err := deps.ParseHeaderField(`{"tailwind-merge":"^2.2.0","clsx":"^2.1.0"}`)
	if err != nil {
		return bundle, err
	}
	for _, d := range declarations {
		bundle.Dependencies = append(bundle.Dependencies, deps.Declaration{DepName: d.DepName, VersionRange: d.VersionRange, Kind: d.Kind})
	}
	return bundle, nil
}
func TestCompositionIdentityStableWithObjectDependencyHeaders(t *testing.T) {
	h := NewConnectHandler(Deps{Service: parsedDependencyRenderService{}})
	bindings, err := structpb.NewStruct(map[string]any{"$template": map[string]any{}, "$labels": map[string]any{"missing": "Missing", "failed": "Failed"}})
	require.NoError(t, err)
	request := &previewv1.RenderCompositionRequest{Composition: &previewv1.GetCompositionBundleRequest{Revision: "candidate", Template: &previewv1.CompositionAsset{CatalogId: "templates.page", Version: "1.0.0", Export: "Page"}}, Bindings: bindings, Kit: "none"}
	var expected string
	for i := 0; i < 64; i++ {
		result, err := h.RenderComposition(context.Background(), connect.NewRequest(request))
		require.NoError(t, err)
		if i == 0 {
			expected = result.Msg.RenderHash
		}
		require.Equal(t, expected, result.Msg.RenderHash, "dependency object ordering altered render identity")
	}
}
