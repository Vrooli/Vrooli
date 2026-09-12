package assets_test

import (
	"context"
	"testing"

	"brand-manager/handlers/assets"
	internalassets "brand-manager/internal/assets"
	mocks "brand-manager/internal/assets/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	assetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assets"
	assetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assets/assets_v1connect"
)

// newClient wires the real internal service over in-memory fakes behind the
// generated Connect handler, exercising handler + adapter + service together.
func newClient(t *testing.T, knownBrands ...string) assetsconnect.AssetsServiceClient {
	t.Helper()
	repo := &mocks.FakeRepository{}
	blobs := &mocks.FakeBlobStore{}
	resolver := mocks.FakeBrandResolver{Known: map[string]bool{}}
	for _, b := range knownBrands {
		resolver.Known[b] = true
	}
	logger, _ := connectxtest.NewLogger(t)
	svc := internalassets.NewService(repo, blobs, resolver, logger)
	path, handler := assetsconnect.NewAssetsServiceHandler(assets.NewConnectHandler(assets.Deps{Service: svc, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return assetsconnect.NewAssetsServiceClient(server.Client(), server.URL)
}

func TestConnect_UploadThenGetAndDownload(t *testing.T) {
	client := newClient(t, "b1")
	ctx := context.Background()

	up, err := client.UploadAsset(ctx, connect.NewRequest(&assetsv1.UploadAssetRequest{
		BrandId:  "b1",
		Filename: "logo.png",
		Content:  []byte("\x89PNGdata"),
	}))
	require.NoError(t, err)
	require.NotEmpty(t, up.Msg.Asset.Id)
	require.Equal(t, "image/png", up.Msg.Asset.MimeType, "mime inferred from extension")
	require.Equal(t, int64(len("\x89PNGdata")), up.Msg.Asset.Size)

	got, err := client.GetAsset(ctx, connect.NewRequest(&assetsv1.GetAssetRequest{Id: up.Msg.Asset.Id}))
	require.NoError(t, err)
	require.Equal(t, "logo.png", got.Msg.Asset.Filename)

	dl, err := client.DownloadAsset(ctx, connect.NewRequest(&assetsv1.DownloadAssetRequest{Id: up.Msg.Asset.Id}))
	require.NoError(t, err)
	require.Equal(t, "image/png", dl.Msg.MimeType)
	require.Equal(t, "\x89PNGdata", string(dl.Msg.Content))
}

func TestConnect_UploadUnknownBrandIsInvalidArgument(t *testing.T) {
	client := newClient(t) // no known brands
	_, err := client.UploadAsset(context.Background(), connect.NewRequest(&assetsv1.UploadAssetRequest{
		BrandId:  "ghost",
		Filename: "logo.png",
		Content:  []byte("x"),
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestConnect_UploadUnsupportedTypeIsInvalidArgument(t *testing.T) {
	client := newClient(t, "b1")
	_, err := client.UploadAsset(context.Background(), connect.NewRequest(&assetsv1.UploadAssetRequest{
		BrandId:  "b1",
		Filename: "notes.txt",
		Content:  []byte("hi"),
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestConnect_GetNotFound(t *testing.T) {
	client := newClient(t, "b1")
	_, err := client.GetAsset(context.Background(), connect.NewRequest(&assetsv1.GetAssetRequest{Id: "ghost"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestConnect_ListFiltersByBrand(t *testing.T) {
	client := newClient(t, "b1", "b2")
	ctx := context.Background()

	_, err := client.UploadAsset(ctx, connect.NewRequest(&assetsv1.UploadAssetRequest{BrandId: "b1", Filename: "a.png", Content: []byte("a")}))
	require.NoError(t, err)
	_, err = client.UploadAsset(ctx, connect.NewRequest(&assetsv1.UploadAssetRequest{BrandId: "b2", Filename: "b.png", Content: []byte("b")}))
	require.NoError(t, err)

	b1, err := client.ListAssets(ctx, connect.NewRequest(&assetsv1.ListAssetsRequest{BrandId: "b1"}))
	require.NoError(t, err)
	require.Len(t, b1.Msg.Assets, 1)
	require.Equal(t, "a.png", b1.Msg.Assets[0].Filename)

	all, err := client.ListAssets(ctx, connect.NewRequest(&assetsv1.ListAssetsRequest{}))
	require.NoError(t, err)
	require.Len(t, all.Msg.Assets, 2, "empty brand filter returns all assets")
}

func TestConnect_DeleteIsIdempotent(t *testing.T) {
	client := newClient(t, "b1")
	ctx := context.Background()

	up, err := client.UploadAsset(ctx, connect.NewRequest(&assetsv1.UploadAssetRequest{BrandId: "b1", Filename: "logo.png", Content: []byte("x")}))
	require.NoError(t, err)

	_, err = client.DeleteAsset(ctx, connect.NewRequest(&assetsv1.DeleteAssetRequest{Id: up.Msg.Asset.Id}))
	require.NoError(t, err)
	_, err = client.DeleteAsset(ctx, connect.NewRequest(&assetsv1.DeleteAssetRequest{Id: up.Msg.Asset.Id}))
	require.NoError(t, err, "deleting a missing asset is a success")
}
