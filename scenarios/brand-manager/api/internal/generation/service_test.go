package generation_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"brand-manager/internal/generation"
	mocks "brand-manager/internal/generation/mocks"

	"github.com/stretchr/testify/require"
)

// seqIDGen returns a deterministic, monotonically increasing id so unique
// filenames are stable and distinct across calls in a test.
func seqIDGen() func() string {
	var n atomic.Int64
	return func() string { return "u" + strconv.FormatInt(n.Add(1), 10) }
}

func newService(t *testing.T, providers *mocks.FakeProviders, images *mocks.FakeImageBackend, brands *mocks.FakeBrandStore, assets *mocks.FakeAssetStore) generation.Service {
	t.Helper()
	if images == nil {
		images = &mocks.FakeImageBackend{}
	}
	return generation.NewService(providers, images, brands, assets, seqIDGen(), nil)
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
	svc := newService(t, providers, nil, brands, assets)

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
	svc := newService(t, providers, nil, brands, &mocks.FakeAssetStore{})

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
	svc := newService(t, providers, nil, brands, &mocks.FakeAssetStore{})

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
	svc := newService(t, providers, nil, brands, &mocks.FakeAssetStore{})

	res, err := svc.GenerateElements(context.Background(), "b1", []string{"colors"}, "")
	require.NoError(t, err, "a parse failure is reported per-element, not as a call error")
	require.Empty(t, res.Applied)
	require.Equal(t, generation.StatusFailed, res.Results[0].Status)
	require.NotEmpty(t, res.Results[0].Detail)
}

func TestGenerateElements_UnknownBrandIsNotFound(t *testing.T) {
	providers := &mocks.FakeProviders{AvailableValue: true}
	svc := newService(t, providers, nil, &mocks.FakeBrandStore{Known: map[string]generation.BrandView{}}, &mocks.FakeAssetStore{})

	_, err := svc.GenerateElements(context.Background(), "ghost", []string{"colors"}, "")
	var notFound generation.ErrBrandNotFound
	require.ErrorAs(t, err, &notFound)
}

func TestGenerateElements_NoProvidersIsUnavailable(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme"})
	providers := &mocks.FakeProviders{AvailableValue: false}
	svc := newService(t, providers, nil, brands, &mocks.FakeAssetStore{})

	_, err := svc.GenerateElements(context.Background(), "b1", []string{"colors"}, "")
	var unavailable generation.ErrProvidersUnavailable
	require.ErrorAs(t, err, &unavailable)
}

func TestGenerateElements_EmptyElementsIsInvalid(t *testing.T) {
	svc := newService(t, &mocks.FakeProviders{AvailableValue: true}, nil, &mocks.FakeBrandStore{}, &mocks.FakeAssetStore{})
	_, err := svc.GenerateElements(context.Background(), "b1", nil, "")
	var invalid generation.ErrInvalidGeneration
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "elements", invalid.Field)
}

// --- image generation (image-tools backed) -------------------------------

func TestGenerateImage_RoutesThroughImageToolsAndAutoPromotesFirstCanonical(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme", PrimaryColor: "#112233"})
	images := &mocks.FakeImageBackend{
		GenerateResponder: func(req generation.ImageGenerateRequest) (generation.ImageOutput, error) {
			require.Contains(t, req.Prompt, "Acme")
			require.Equal(t, 512, req.Width)
			require.NotEmpty(t, req.NegativePrompt)
			return generation.ImageOutput{Data: []byte("\x89PNGlogo"), MimeType: "image/png", ModelID: "sd-1.5", Tier: "local-gpu"}, nil
		},
	}
	assets := &mocks.FakeAssetStore{}
	svc := newService(t, &mocks.FakeProviders{}, images, brands, assets)

	res, err := svc.GenerateImage(context.Background(), generation.GenerateImageInput{BrandID: "b1", Type: "LOGO"})
	require.NoError(t, err)
	require.Equal(t, "logo", res.Kind, "kind is normalised")
	require.Equal(t, "sd-1.5", res.ModelID)
	require.Equal(t, "local-gpu", res.Tier)
	require.True(t, res.Canonical, "first generation auto-promotes to canonical")
	require.Equal(t, "logo-u1.png", res.Filename, "exploratory asset has a unique filename")

	stored := assets.StoredUploads()
	require.Len(t, stored, 2, "one exploratory asset + the auto-promoted canonical")
	require.Equal(t, "logo-u1.png", stored[0].Filename)
	require.Equal(t, "logo.png", stored[1].Filename)
}

func TestGenerateImage_DoesNotOverwriteExistingCanonicalUnlessRequested(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme"})
	assets := &mocks.FakeAssetStore{}
	// A user already has a canonical logo.
	assets.SeedAsset("existing", "b1", "logo.png", "image/png", []byte("user-logo"))
	svc := newService(t, &mocks.FakeProviders{}, &mocks.FakeImageBackend{}, brands, assets)

	res, err := svc.GenerateImage(context.Background(), generation.GenerateImageInput{BrandID: "b1", Type: "logo"})
	require.NoError(t, err)
	require.False(t, res.Canonical, "an existing user canonical is preserved by default")

	// Only the exploratory asset was written; logo.png is untouched.
	for _, up := range assets.StoredUploads() {
		require.NotEqual(t, "logo.png", up.Filename, "canonical must not be overwritten without set_canonical")
	}

	// With set_canonical, the canonical is (re)written.
	res2, err := svc.GenerateImage(context.Background(), generation.GenerateImageInput{BrandID: "b1", Type: "logo", SetCanonical: true})
	require.NoError(t, err)
	require.True(t, res2.Canonical)
}

func TestGenerateImage_RejectsUnknownType(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme"})
	svc := newService(t, &mocks.FakeProviders{}, &mocks.FakeImageBackend{}, brands, &mocks.FakeAssetStore{})

	_, err := svc.GenerateImage(context.Background(), generation.GenerateImageInput{BrandID: "b1", Type: "banner"})
	var invalid generation.ErrInvalidGeneration
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "type", invalid.Field)
}

func TestGenerateImage_BackendNotReadySurfacesTyped(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme"})
	images := &mocks.FakeImageBackend{
		GenerateResponder: func(generation.ImageGenerateRequest) (generation.ImageOutput, error) {
			return generation.ImageOutput{}, generation.ErrImageBackendNotReady{Operation: "text_to_image", Hint: "run image-tools models install sd-1.5"}
		},
	}
	svc := newService(t, &mocks.FakeProviders{}, images, brands, &mocks.FakeAssetStore{})

	_, err := svc.GenerateImage(context.Background(), generation.GenerateImageInput{BrandID: "b1", Type: "logo"})
	var notReady generation.ErrImageBackendNotReady
	require.ErrorAs(t, err, &notReady)
	require.Contains(t, notReady.Hint, "models install")
}

func TestGenerateImage_AssetWriteErrorSurfaces(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme"})
	assets := &mocks.FakeAssetStore{StoreErr: errors.New("disk full")}
	svc := newService(t, &mocks.FakeProviders{}, &mocks.FakeImageBackend{}, brands, assets)

	_, err := svc.GenerateImage(context.Background(), generation.GenerateImageInput{BrandID: "b1", Type: "favicon"})
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "disk full"))
}

func TestEditImage_LoadsSourceAndStoresNewAsset(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme"})
	assets := &mocks.FakeAssetStore{}
	assets.SeedAsset("src1", "b1", "logo.png", "image/png", []byte("source-bytes"))
	images := &mocks.FakeImageBackend{
		EditResponder: func(req generation.ImageEditRequest) (generation.ImageOutput, error) {
			require.Equal(t, []byte("source-bytes"), req.Source)
			require.Equal(t, "make it navy", req.Instruction)
			return generation.ImageOutput{Data: []byte("edited"), MimeType: "image/png", ModelID: "ip2p", Tier: "local-gpu"}, nil
		},
	}
	svc := newService(t, &mocks.FakeProviders{}, images, brands, assets)

	res, err := svc.EditImage(context.Background(), generation.EditImageInput{BrandID: "b1", SourceAssetID: "src1", Instruction: "make it navy"})
	require.NoError(t, err)
	require.Equal(t, "logo", res.Kind)
	require.False(t, res.Canonical, "logo.png already exists, so an edit does not auto-overwrite it")
	require.Len(t, images.EditRequests(), 1)
}

func TestEditImage_MissingSourceIsNotFound(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme"})
	svc := newService(t, &mocks.FakeProviders{}, &mocks.FakeImageBackend{}, brands, &mocks.FakeAssetStore{})

	_, err := svc.EditImage(context.Background(), generation.EditImageInput{BrandID: "b1", SourceAssetID: "ghost", Instruction: "x"})
	var notFound generation.ErrSourceAssetNotFound
	require.ErrorAs(t, err, &notFound)
}

func TestEditImage_EmptyInstructionIsInvalid(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme"})
	svc := newService(t, &mocks.FakeProviders{}, &mocks.FakeImageBackend{}, brands, &mocks.FakeAssetStore{})

	_, err := svc.EditImage(context.Background(), generation.EditImageInput{BrandID: "b1", SourceAssetID: "src", Instruction: "  "})
	var invalid generation.ErrInvalidGeneration
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "instruction", invalid.Field)
}

func TestRemoveBackground_StoresTransparentKind(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme"})
	assets := &mocks.FakeAssetStore{}
	assets.SeedAsset("src1", "b1", "logo.png", "image/png", []byte("src"))
	svc := newService(t, &mocks.FakeProviders{}, &mocks.FakeImageBackend{}, brands, assets)

	res, err := svc.RemoveBackground(context.Background(), generation.RemoveBackgroundInput{BrandID: "b1", SourceAssetID: "src1"})
	require.NoError(t, err)
	require.Equal(t, "logo-transparent", res.Kind)
	require.True(t, res.Canonical, "no canonical logo-transparent yet, so it auto-promotes")
	require.Equal(t, "logo-transparent-u1.png", res.Filename)
}

func TestDeriveIcons_ProducesFullSetWithSolidAndTransparentVariants(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme", PrimaryColor: "#102030"})
	assets := &mocks.FakeAssetStore{}
	assets.SeedAsset("src1", "b1", "logo-transparent.png", "image/png", []byte("mark"))
	images := &mocks.FakeImageBackend{}
	svc := newService(t, &mocks.FakeProviders{}, images, brands, assets)

	icons, _, err := svc.DeriveIcons(context.Background(), generation.DeriveIconsInput{BrandID: "b1", SourceAssetID: "src1"})
	require.NoError(t, err)

	names := map[string]bool{}
	for _, ic := range icons {
		names[ic.Filename] = true
		require.Equal(t, "deterministic", ic.Tier)
		require.True(t, ic.Canonical)
	}
	require.True(t, names["favicon-16.png"])
	require.True(t, names["favicon-32.png"])
	require.True(t, names["favicon-196.png"])
	require.True(t, names["apple-touch-icon.png"])
	require.True(t, names["maskable-icon-192.png"])
	require.True(t, names["maskable-icon-512.png"])

	// Solid variants flatten onto the brand color; favicons stay transparent.
	require.Equal(t, []string{"#102030", "#102030", "#102030"}, images.FlattenBackgrounds(),
		"apple-touch + two maskable icons flatten onto the brand primary color")
}

func TestDeriveIcons_RespectsFaviconOnlySelection(t *testing.T) {
	brands := &mocks.FakeBrandStore{}
	brands.Seed(generation.BrandView{ID: "b1", Name: "Acme"})
	assets := &mocks.FakeAssetStore{}
	assets.SeedAsset("src1", "b1", "logo.png", "image/png", []byte("mark"))
	images := &mocks.FakeImageBackend{}
	svc := newService(t, &mocks.FakeProviders{}, images, brands, assets)

	icons, _, err := svc.DeriveIcons(context.Background(), generation.DeriveIconsInput{BrandID: "b1", SourceAssetID: "src1", IncludeFavicon: true})
	require.NoError(t, err)
	require.Len(t, icons, 3, "only the favicon family")
	require.Empty(t, images.FlattenBackgrounds(), "no solid variants when only favicons requested")
}

func TestImageBackendStatus_PassesThrough(t *testing.T) {
	images := &mocks.FakeImageBackend{StatusValue: generation.ImageBackendStatus{
		Available:  true,
		Operations: []generation.ImageOperationStatus{{Operation: "generate", Ready: true, ModelID: "sd-1.5", Tier: "local-gpu"}},
	}}
	svc := newService(t, &mocks.FakeProviders{}, images, &mocks.FakeBrandStore{}, &mocks.FakeAssetStore{})

	st := svc.ImageBackendStatus(context.Background())
	require.True(t, st.Available)
	require.Len(t, st.Operations, 1)
	require.Equal(t, "generate", st.Operations[0].Operation)
}

func TestProviderStatus_PassesThrough(t *testing.T) {
	providers := &mocks.FakeProviders{
		AvailableValue: true,
		StatusValues:   []generation.ProviderStatus{{Name: "ollama", Available: true}, {Name: "openrouter", Available: false}},
	}
	svc := newService(t, providers, nil, &mocks.FakeBrandStore{}, &mocks.FakeAssetStore{})

	available, statuses := svc.ProviderStatus(context.Background())
	require.True(t, available)
	require.Len(t, statuses, 2)
	require.Equal(t, "ollama", statuses[0].Name)
}
