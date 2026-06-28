package generation

import (
	"context"
	"log"
	"strings"
)

// BrandStore is the cross-domain seam Service uses to read a brand and apply
// generated text facets. Implemented at the composition root
// (handlers/generation/module.go) over the brands domain's service, so applying
// facets goes through the brands partial-merge + version-snapshot path.
type BrandStore interface {
	// Get returns the brand projection the generator needs, or ErrBrandNotFound
	// when no brand matches.
	Get(ctx context.Context, brandID string) (BrandView, error)
	// ApplyElements merges the non-nil generated facets onto the brand and
	// returns the brand's new version. Partial-merge: a nil facet leaves the
	// stored value unchanged.
	ApplyElements(ctx context.Context, in ApplyElementsInput) (newVersion int, err error)
}

// AssetStore is the cross-domain seam Service uses to persist a generated
// image. Implemented at the composition root over the assets domain's service,
// so a re-generated image upserts by (brand_id, filename) like any other asset.
type AssetStore interface {
	// Store persists the image bytes and returns the stored catalog metadata.
	Store(ctx context.Context, in AssetUpload) (StoredAsset, error)
}

// supportedElements is the set of text facets the generator knows how to build.
// Anything else is reported per-element as "unsupported".
var supportedElements = map[string]bool{
	"colors":     true,
	"typography": true,
	"voice":      true,
}

// imageSpec maps an image type to its prompt builder and target dimensions.
type imageSpec struct {
	width  int
	height int
}

var imageSpecs = map[string]imageSpec{
	"logo":    {width: 512, height: 512},
	"favicon": {width: 64, height: 64},
}

// Service is the application-layer surface the generation handlers depend on.
// It orchestrates the provider chain and the brand/asset cross-domain seams; it
// owns no persistence of its own. The handler is thin around it: decode → call
// service → translate errors.
type Service interface {
	// ProviderStatus reports the configured provider chain's readiness.
	ProviderStatus(ctx context.Context) (available bool, statuses []ProviderStatus)

	// GenerateElements generates the requested text facets and applies them to
	// the brand. Returns ErrInvalidGeneration (bad input), ErrBrandNotFound
	// (unknown brand), or ErrProvidersUnavailable (no provider reachable).
	// Per-element provider/parse failures are reported in the result, not as a
	// call error.
	GenerateElements(ctx context.Context, brandID string, elements []string, model string) (ElementsResult, error)

	// GenerateImage generates a logo or favicon and stores it as a brand asset.
	GenerateImage(ctx context.Context, brandID, imageType, model string) (ImageResult, error)
}

type service struct {
	providers Providers
	brands    BrandStore
	assets    AssetStore
	logger    *log.Logger
}

// NewService constructs the production Service. A nil logger defaults to
// log.Default().
func NewService(providers Providers, brands BrandStore, assets AssetStore, logger *log.Logger) Service {
	if logger == nil {
		logger = log.Default()
	}
	return &service{providers: providers, brands: brands, assets: assets, logger: logger}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) ProviderStatus(ctx context.Context) (bool, []ProviderStatus) {
	return s.providers.Available(ctx), s.providers.Statuses(ctx)
}

func (s *service) GenerateElements(ctx context.Context, brandID string, elements []string, model string) (ElementsResult, error) {
	brandID = strings.TrimSpace(brandID)
	if brandID == "" {
		return ElementsResult{}, ErrInvalidGeneration{Field: "brand_id", Reason: "required"}
	}
	if len(elements) == 0 {
		return ElementsResult{}, ErrInvalidGeneration{Field: "elements", Reason: "at least one element is required (colors, typography, voice)"}
	}

	brand, err := s.brands.Get(ctx, brandID)
	if err != nil {
		return ElementsResult{}, err
	}
	if !s.providers.Available(ctx) {
		return ElementsResult{}, ErrProvidersUnavailable{}
	}

	apply := ApplyElementsInput{BrandID: brandID}
	out := ElementsResult{Results: make([]ElementOutcome, 0, len(elements))}

	for _, raw := range elements {
		elem := strings.ToLower(strings.TrimSpace(raw))
		if !supportedElements[elem] {
			out.Results = append(out.Results, ElementOutcome{Element: raw, Status: StatusUnsupported, Detail: "unknown element"})
			continue
		}

		resp, genErr := s.providers.GenerateText(ctx, TextRequest{Prompt: promptFor(elem, brand), Model: model})
		if genErr != nil {
			out.Results = append(out.Results, ElementOutcome{Element: raw, Status: StatusFailed, Detail: genErr.Error()})
			continue
		}
		out.Provider = resp.Provider
		out.Model = resp.Model

		parsed, parseErr := parseGeneratedJSON(resp.Text)
		if parseErr != nil {
			out.Results = append(out.Results, ElementOutcome{Element: raw, Status: StatusFailed, Detail: parseErr.Error()})
			continue
		}

		stageFacet(&apply, elem, parsed)
		out.Applied = append(out.Applied, elem)
		out.Results = append(out.Results, ElementOutcome{Element: raw, Status: StatusApplied})
	}

	if len(out.Applied) > 0 {
		version, err := s.brands.ApplyElements(ctx, apply)
		if err != nil {
			return ElementsResult{}, err
		}
		out.Version = version
	}
	return out, nil
}

func (s *service) GenerateImage(ctx context.Context, brandID, imageType, model string) (ImageResult, error) {
	brandID = strings.TrimSpace(brandID)
	if brandID == "" {
		return ImageResult{}, ErrInvalidGeneration{Field: "brand_id", Reason: "required"}
	}
	imageType = strings.ToLower(strings.TrimSpace(imageType))
	spec, ok := imageSpecs[imageType]
	if !ok {
		return ImageResult{}, ErrInvalidGeneration{Field: "type", Reason: "must be 'logo' or 'favicon'"}
	}

	brand, err := s.brands.Get(ctx, brandID)
	if err != nil {
		return ImageResult{}, err
	}
	if !s.providers.Available(ctx) {
		return ImageResult{}, ErrProvidersUnavailable{}
	}

	prompt := logoPrompt(brand.Name, brand.Description, brand.PrimaryColor)
	if imageType == "favicon" {
		prompt = faviconPrompt(brand.Name, brand.PrimaryColor)
	}

	resp, err := s.providers.GenerateImage(ctx, ImageRequest{Prompt: prompt, Model: model, Width: spec.width, Height: spec.height})
	if err != nil {
		return ImageResult{}, err
	}

	// A stable per-type filename means regenerating replaces the bytes in place
	// (the assets domain upserts by (brand_id, filename)) instead of piling up
	// one row per generation.
	filename := imageType + ".png"
	stored, err := s.assets.Store(ctx, AssetUpload{
		BrandID:  brandID,
		Filename: filename,
		MimeType: resp.MimeType,
		Content:  resp.Data,
	})
	if err != nil {
		return ImageResult{}, err
	}

	return ImageResult{
		BrandID:  brandID,
		AssetID:  stored.ID,
		Type:     imageType,
		Filename: stored.Filename,
		MimeType: stored.MimeType,
		Size:     stored.Size,
		Provider: resp.Provider,
		Model:    resp.Model,
	}, nil
}

// promptFor returns the text-generation prompt for a supported element.
func promptFor(elem string, brand BrandView) string {
	switch elem {
	case "typography":
		return typographyPrompt(brand.Name, brand.Description, brand.Notes)
	case "voice":
		return voicePrompt(brand.Name, brand.Description, brand.Notes)
	default: // "colors"
		return colorPrompt(brand.Name, brand.Description, brand.Notes)
	}
}

// stageFacet attaches the parsed facet to the pending apply input.
func stageFacet(apply *ApplyElementsInput, elem string, parsed map[string]any) {
	switch elem {
	case "typography":
		apply.Typography = typographyFromJSON(parsed)
	case "voice":
		apply.Voice = voiceFromJSON(parsed)
	default: // "colors"
		apply.Colors = colorsFromJSON(parsed)
	}
}
