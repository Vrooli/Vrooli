package generation_test

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"

	"brand-manager/handlers/generation"
	internalgeneration "brand-manager/internal/generation"
	mocks "brand-manager/internal/generation/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	generationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/generation"
	generationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/generation/generation_v1connect"
)

func seqIDGen() func() string {
	var n atomic.Int64
	return func() string { return "u" + strconv.FormatInt(n.Add(1), 10) }
}

// newClient wires the real internal service over in-memory fakes behind the
// generated Connect handler, exercising handler + adapter + service together.
// A nil images fake defaults to a benign image-tools that always succeeds.
func newClient(t *testing.T, providers *mocks.FakeProviders, images *mocks.FakeImageBackend, brands *mocks.FakeBrandStore, assets *mocks.FakeAssetStore) generationconnect.GenerationServiceClient {
	t.Helper()
	if images == nil {
		images = &mocks.FakeImageBackend{}
	}
	logger, _ := connectxtest.NewLogger(t)
	svc := internalgeneration.NewService(providers, images, brands, assets, seqIDGen(), logger)
	path, handler := generationconnect.NewGenerationServiceHandler(generation.NewConnectHandler(generation.Deps{Service: svc, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return generationconnect.NewGenerationServiceClient(server.Client(), server.URL)
}

func seededBrands(t *testing.T) *mocks.FakeBrandStore {
	t.Helper()
	b := &mocks.FakeBrandStore{}
	b.Seed(internalgeneration.BrandView{ID: "b1", Name: "Acme", Version: 1})
	return b
}

func TestConnect_GenerateElementsApplies(t *testing.T) {
	providers := &mocks.FakeProviders{
		AvailableValue: true,
		TextResponder: func(internalgeneration.TextRequest) (internalgeneration.TextResponse, error) {
			return internalgeneration.TextResponse{Text: `{"primary":"#112233"}`, Provider: "ollama", Model: "chat.default"}, nil
		},
	}
	client := newClient(t, providers, nil, seededBrands(t), &mocks.FakeAssetStore{})

	resp, err := client.GenerateBrandElements(context.Background(), connect.NewRequest(&generationv1.GenerateBrandElementsRequest{
		BrandId:  "b1",
		Elements: []string{"colors"},
	}))
	require.NoError(t, err)
	require.Equal(t, []string{"colors"}, resp.Msg.Applied)
	require.Equal(t, int32(2), resp.Msg.BrandVersion)
	require.Equal(t, "ollama", resp.Msg.Provider)
}

func TestConnect_GenerateElementsUnknownBrandIsNotFound(t *testing.T) {
	providers := &mocks.FakeProviders{AvailableValue: true}
	client := newClient(t, providers, nil, &mocks.FakeBrandStore{Known: map[string]internalgeneration.BrandView{}}, &mocks.FakeAssetStore{})

	_, err := client.GenerateBrandElements(context.Background(), connect.NewRequest(&generationv1.GenerateBrandElementsRequest{
		BrandId:  "ghost",
		Elements: []string{"colors"},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestConnect_GenerateElementsNoProvidersIsUnavailable(t *testing.T) {
	providers := &mocks.FakeProviders{AvailableValue: false}
	client := newClient(t, providers, nil, seededBrands(t), &mocks.FakeAssetStore{})

	_, err := client.GenerateBrandElements(context.Background(), connect.NewRequest(&generationv1.GenerateBrandElementsRequest{
		BrandId:  "b1",
		Elements: []string{"colors"},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

func TestConnect_GenerateElementsEmptyIsInvalidArgument(t *testing.T) {
	client := newClient(t, &mocks.FakeProviders{AvailableValue: true}, nil, seededBrands(t), &mocks.FakeAssetStore{})

	_, err := client.GenerateBrandElements(context.Background(), connect.NewRequest(&generationv1.GenerateBrandElementsRequest{
		BrandId: "b1",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestConnect_GenerateImageStoresAsset(t *testing.T) {
	providers := &mocks.FakeProviders{AvailableValue: true}
	assets := &mocks.FakeAssetStore{}
	client := newClient(t, providers, nil, seededBrands(t), assets)

	resp, err := client.GenerateBrandImage(context.Background(), connect.NewRequest(&generationv1.GenerateBrandImageRequest{
		BrandId: "b1",
		Type:    "favicon",
	}))
	require.NoError(t, err)
	require.Equal(t, "favicon", resp.Msg.Kind)
	require.Equal(t, "favicon-u1.png", resp.Msg.Filename, "exploratory generations get a unique filename")
	require.True(t, resp.Msg.Canonical, "first favicon auto-promotes to canonical")
	require.NotEmpty(t, resp.Msg.AssetId)
	require.Len(t, assets.StoredUploads(), 2, "exploratory asset + auto-promoted canonical")
}

func TestConnect_EditImageRoutesAndReturnsAsset(t *testing.T) {
	providers := &mocks.FakeProviders{}
	assets := &mocks.FakeAssetStore{}
	assets.SeedAsset("src1", "b1", "logo.png", "image/png", []byte("source"))
	client := newClient(t, providers, nil, seededBrands(t), assets)

	resp, err := client.EditBrandImage(context.Background(), connect.NewRequest(&generationv1.EditBrandImageRequest{
		BrandId:       "b1",
		SourceAssetId: "src1",
		Instruction:   "make it bold",
	}))
	require.NoError(t, err)
	require.Equal(t, "logo", resp.Msg.Kind)
	require.NotEmpty(t, resp.Msg.AssetId)
}

func TestConnect_EditImageMissingSourceIsNotFound(t *testing.T) {
	client := newClient(t, &mocks.FakeProviders{}, nil, seededBrands(t), &mocks.FakeAssetStore{})

	_, err := client.EditBrandImage(context.Background(), connect.NewRequest(&generationv1.EditBrandImageRequest{
		BrandId:       "b1",
		SourceAssetId: "ghost",
		Instruction:   "x",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestConnect_GenerateImageBackendNotReadyIsFailedPrecondition(t *testing.T) {
	images := &mocks.FakeImageBackend{
		GenerateResponder: func(internalgeneration.ImageGenerateRequest) (internalgeneration.ImageOutput, error) {
			return internalgeneration.ImageOutput{}, internalgeneration.ErrImageBackendNotReady{Operation: "text_to_image", Hint: "install sd-1.5"}
		},
	}
	client := newClient(t, &mocks.FakeProviders{}, images, seededBrands(t), &mocks.FakeAssetStore{})

	_, err := client.GenerateBrandImage(context.Background(), connect.NewRequest(&generationv1.GenerateBrandImageRequest{
		BrandId: "b1",
		Type:    "logo",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestConnect_GetImageBackendStatus(t *testing.T) {
	images := &mocks.FakeImageBackend{StatusValue: internalgeneration.ImageBackendStatus{
		Available:  true,
		Operations: []internalgeneration.ImageOperationStatus{{Operation: "generate", Ready: true, ModelID: "sd-1.5", Tier: "local-gpu"}},
	}}
	client := newClient(t, &mocks.FakeProviders{}, images, seededBrands(t), &mocks.FakeAssetStore{})

	resp, err := client.GetImageBackendStatus(context.Background(), connect.NewRequest(&generationv1.GetImageBackendStatusRequest{}))
	require.NoError(t, err)
	require.True(t, resp.Msg.Available)
	require.Len(t, resp.Msg.Operations, 1)
	require.Equal(t, "generate", resp.Msg.Operations[0].Operation)
	require.Equal(t, "sd-1.5", resp.Msg.Operations[0].ModelId)
}

func TestConnect_GenerateImageBadTypeIsInvalidArgument(t *testing.T) {
	client := newClient(t, &mocks.FakeProviders{AvailableValue: true}, nil, seededBrands(t), &mocks.FakeAssetStore{})

	_, err := client.GenerateBrandImage(context.Background(), connect.NewRequest(&generationv1.GenerateBrandImageRequest{
		BrandId: "b1",
		Type:    "banner",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestConnect_GetProviderStatus(t *testing.T) {
	providers := &mocks.FakeProviders{
		AvailableValue: true,
		StatusValues:   []internalgeneration.ProviderStatus{{Name: "ollama", Available: true}, {Name: "openrouter", Available: false}},
	}
	client := newClient(t, providers, nil, seededBrands(t), &mocks.FakeAssetStore{})

	resp, err := client.GetProviderStatus(context.Background(), connect.NewRequest(&generationv1.GetProviderStatusRequest{}))
	require.NoError(t, err)
	require.True(t, resp.Msg.Available)
	require.Len(t, resp.Msg.Providers, 2)
	require.Equal(t, "ollama", resp.Msg.Providers[0].Name)
}
