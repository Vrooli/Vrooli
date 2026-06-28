package generation_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"brand-manager/internal/generation"
	mocks "brand-manager/internal/generation/mocks"

	"github.com/stretchr/testify/require"
)

func newService(t *testing.T, providers *mocks.FakeProviders, brands *mocks.FakeBrandStore, assets *mocks.FakeAssetStore) generation.Service {
	t.Helper()
	return generation.NewService(providers, brands, assets, nil)
}

func TestGenerateElements_AppliesColorsAndBumpsVersion(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme", Description: "desc", Notes: "notes", Version: 3})
	providers := &mocks.FakeProviders{
		AvailableValue: true,
		TextResponder: func(generation.TextRequest) (generation.TextResponse, error) {
			return generation.TextResponse{
				Text:     `{"primary":"#112233","background":"#ffffff","text":"#000000"}`,
				Provider: "ollama",
				Model:    "chat.default",
			}, nil
		},
	}
	assets := &mocks.FakeAssetStore{}
	svc := newService(t, providers, brands, assets)

	res, err := svc.GenerateElements(context.Background(), "b1", []string{"colors"}, "")
	require.NoError(t, err)
	require.Equal(t, []string{"colors"}, res.Applied)
	require.Equal(t, "ollama", res.Provider)
	require.Equal(t, 4, res.Version, "applying one facet bumps the brand version once")
	require.Len(t, res.Results, 1)
	require.Equal(t, generation.StatusApplied, res.Results[0].Status)

	applied := brands.AppliedInputs()
	require.Len(t, applied, 1)
	require.NotNil(t, applied[0].Colors)
	require.Equal(t, "#112233", applied[0].Colors.Primary)
	require.Nil(t, applied[0].Typography, "only the generated facet is staged")
}

func TestGenerateElements_PromptCarriesBrandContext(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme", Description: "a fintech", Notes: "playful"})
	providers := &mocks.FakeProviders{AvailableValue: true}
	svc := newService(t, providers, brands, &mocks.FakeAssetStore{})

	_, err := svc.GenerateElements(context.Background(), "b1", []string{"voice"}, "")
	require.NoError(t, err)

	reqs := providers.TextRequests()
	require.Len(t, reqs, 1)
	require.Contains(t, reqs[0].Prompt, "Acme")
	require.Contains(t, reqs[0].Prompt, "a fintech")
	require.Contains(t, reqs[0].Prompt, "playful")
}

func TestGenerateElements_UnsupportedElementIsReportedNotApplied(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme"})
	providers := &mocks.FakeProviders{AvailableValue: true}
	svc := newService(t, providers, brands, &mocks.FakeAssetStore{})

	res, err := svc.GenerateElements(context.Background(), "b1", []string{"mascot"}, "")
	require.NoError(t, err)
	require.Empty(t, res.Applied)
	require.Equal(t, 0, res.Version, "nothing applied leaves the brand untouched")
	require.Equal(t, generation.StatusUnsupported, res.Results[0].Status)
	require.Equal(t, int64(0), providers.TextCalls.Load(), "no provider call for an unsupported element")
	require.Empty(t, brands.AppliedInputs())
}

func TestGenerateElements_BadJSONIsPerElementFailureNotCallError(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme"})
	providers := &mocks.FakeProviders{
		AvailableValue: true,
		TextResponder: func(generation.TextRequest) (generation.TextResponse, error) {
			return generation.TextResponse{Text: "I cannot help with that.", Provider: "ollama"}, nil
		},
	}
	svc := newService(t, providers, brands, &mocks.FakeAssetStore{})

	res, err := svc.GenerateElements(context.Background(), "b1", []string{"colors"}, "")
	require.NoError(t, err, "a parse failure is reported per-element, not as a call error")
	require.Empty(t, res.Applied)
	require.Equal(t, generation.StatusFailed, res.Results[0].Status)
	require.NotEmpty(t, res.Results[0].Detail)
}

func TestGenerateElements_UnknownBrandIsNotFound(t *testing.T) {
	providers := &mocks.FakeProviders{AvailableValue: true}
	svc := newService(t, providers, &mocks.FakeBrandStore{Known: map[string]generation.BrandView{}}, &mocks.FakeAssetStore{})

	_, err := svc.GenerateElements(context.Background(), "ghost", []string{"colors"}, "")
	var notFound generation.ErrBrandNotFound
	require.ErrorAs(t, err, &notFound)
}

func TestGenerateElements_NoProvidersIsUnavailable(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme"})
	providers := &mocks.FakeProviders{AvailableValue: false}
	svc := newService(t, providers, brands, &mocks.FakeAssetStore{})

	_, err := svc.GenerateElements(context.Background(), "b1", []string{"colors"}, "")
	var unavailable generation.ErrProvidersUnavailable
	require.ErrorAs(t, err, &unavailable)
}

func TestGenerateElements_EmptyElementsIsInvalid(t *testing.T) {
	svc := newService(t, &mocks.FakeProviders{AvailableValue: true}, &mocks.FakeBrandStore{}, &mocks.FakeAssetStore{})
	_, err := svc.GenerateElements(context.Background(), "b1", nil, "")
	var invalid generation.ErrInvalidGeneration
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "elements", invalid.Field)
}

func TestGenerateImage_StoresAssetWithStableFilename(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme", PrimaryColor: "#112233"})
	providers := &mocks.FakeProviders{
		AvailableValue: true,
		ImageResponder: func(req generation.ImageRequest) (generation.ImageResponse, error) {
			require.Contains(t, req.Prompt, "Acme")
			require.Equal(t, 512, req.Width)
			return generation.ImageResponse{Data: []byte("\x89PNGlogo"), MimeType: "image/png", Provider: "openrouter", Model: "dall-e-3"}, nil
		},
	}
	assets := &mocks.FakeAssetStore{}
	svc := newService(t, providers, brands, assets)

	res, err := svc.GenerateImage(context.Background(), "b1", "LOGO", "")
	require.NoError(t, err)
	require.Equal(t, "logo", res.Type, "type is normalised")
	require.Equal(t, "logo.png", res.Filename)
	require.Equal(t, "openrouter", res.Provider)
	require.Equal(t, int64(len("\x89PNGlogo")), res.Size)

	stored := assets.StoredUploads()
	require.Len(t, stored, 1)
	require.Equal(t, "logo.png", stored[0].Filename, "stable filename so regeneration upserts in place")
	require.Equal(t, "b1", stored[0].BrandID)
}

func TestGenerateImage_RejectsUnknownType(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme"})
	svc := newService(t, &mocks.FakeProviders{AvailableValue: true}, brands, &mocks.FakeAssetStore{})

	_, err := svc.GenerateImage(context.Background(), "b1", "banner", "")
	var invalid generation.ErrInvalidGeneration
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "type", invalid.Field)
}

func TestGenerateImage_ApplyErrorSurfaces(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme"})
	assets := &mocks.FakeAssetStore{StoreErr: errors.New("disk full")}
	svc := newService(t, &mocks.FakeProviders{AvailableValue: true}, brands, assets)

	_, err := svc.GenerateImage(context.Background(), "b1", "favicon", "")
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "disk full"))
}

func TestProviderStatus_PassesThrough(t *testing.T) {
	providers := &mocks.FakeProviders{
		AvailableValue: true,
		StatusValues:   []generation.ProviderStatus{{Name: "ollama", Available: true}, {Name: "openrouter", Available: false}},
	}
	svc := newService(t, providers, &mocks.FakeBrandStore{}, &mocks.FakeAssetStore{})

	available, statuses := svc.ProviderStatus(context.Background())
	require.True(t, available)
	require.Len(t, statuses, 2)
	require.Equal(t, "ollama", statuses[0].Name)
}
