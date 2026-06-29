package generation

import (
	"context"
	"log"
	"strings"

	"github.com/google/uuid"
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

// AssetStore is the cross-domain seam Service uses to persist and load brand
// images. Implemented at the composition root over the assets domain's service,
// so a stored image upserts by (brand_id, filename) like any other asset.
type AssetStore interface {
	// Store persists the image bytes and returns the stored catalog metadata
	// (upsert by (brand_id, filename)).
	Store(ctx context.Context, in AssetUpload) (StoredAsset, error)
	// Read loads an existing asset's bytes by id, or ErrSourceAssetNotFound.
	Read(ctx context.Context, assetID string) (AssetBytes, error)
	// Exists reports whether the brand already has an asset stored under filename
	// (used to decide whether to auto-promote a first generation to canonical).
	Exists(ctx context.Context, brandID, filename string) (bool, error)
}

// supportedElements is the set of text facets the generator knows how to build.
// Anything else is reported per-element as "unsupported".
var supportedElements = map[string]bool{
	"colors":     true,
	"typography": true,
	"voice":      true,
}

// imageSpec maps an image type to its target generation dimensions.
type imageSpec struct {
	width  int
	height int
}

// imageSpecs are the text_to_image canvas sizes per type. A favicon is generated
// at a model-friendly size and then downscaled via DeriveIcons; a 16/32px canvas
// is far too small for a diffusion model to produce anything legible.
var imageSpecs = map[string]imageSpec{
	"logo":    {width: 512, height: 512},
	"favicon": {width: 512, height: 512},
}

// Canonical kinds + their stable per-kind filenames.
const (
	kindLogo            = "logo"
	kindFavicon         = "favicon"
	kindLogoTransparent = "logo-transparent"

	// defaultIconBackground is the solid background used for opaque icon variants
	// when the brand has no usable primary color.
	defaultIconBackground = "#ffffff"
)

// logoNegativePrompt steers logo generation away from common diffusion failure
// modes. Lives in the service (a brand prompt recipe), not in image-tools.
const logoNegativePrompt = "photo, photograph, realistic, 3d render, text, watermark, signature, busy background, clutter, low quality, blurry"

// Service is the application-layer surface the generation handlers depend on. It
// orchestrates the text provider chain (facets), the image-tools backend
// (images), and the brand/asset cross-domain seams; it owns no persistence of
// its own. The handler is thin around it: decode → call service → translate
// errors.
type Service interface {
	// ProviderStatus reports the configured TEXT provider chain's readiness.
	ProviderStatus(ctx context.Context) (available bool, statuses []ProviderStatus)

	// ImageBackendStatus reports image-tools' reachability + per-operation
	// readiness. Never errors (an unreachable backend is Available=false).
	ImageBackendStatus(ctx context.Context) ImageBackendStatus

	// GenerateElements generates the requested text facets and applies them to
	// the brand. Returns ErrInvalidGeneration (bad input), ErrBrandNotFound
	// (unknown brand), or ErrProvidersUnavailable (no text provider reachable).
	GenerateElements(ctx context.Context, brandID string, elements []string, model string) (ElementsResult, error)

	// GenerateImage generates a logo or favicon through image-tools and stores it
	// as a brand asset.
	GenerateImage(ctx context.Context, in GenerateImageInput) (ImageResult, error)

	// EditImage edits an existing brand image by instruction through image-tools
	// (edit_instruct) and stores the result as a new asset.
	EditImage(ctx context.Context, in EditImageInput) (ImageResult, error)

	// RemoveBackground isolates the mark in an existing brand image through
	// image-tools (background_removal) and stores the transparent cutout.
	RemoveBackground(ctx context.Context, in RemoveBackgroundInput) (ImageResult, error)

	// DeriveIcons produces a deterministic set of platform icon variants from a
	// source asset using image-tools' deterministic ops (resize / flatten).
	DeriveIcons(ctx context.Context, in DeriveIconsInput) (icons []ImageResult, warnings []string, err error)
}

// GenerateImageInput is the GenerateImage request in domain terms.
type GenerateImageInput struct {
	BrandID        string
	Type           string // logo | favicon
	ModelOverride  string
	AllowBYOK      bool
	QualityPolicy  string
	FallbackPolicy string
	Priority       string
	AllowReclaim   *bool
	Seed           int64
	SetCanonical   bool
}

// EditImageInput is the EditImage request in domain terms.
type EditImageInput struct {
	BrandID        string
	SourceAssetID  string
	Instruction    string
	ModelOverride  string
	AllowBYOK      bool
	QualityPolicy  string
	FallbackPolicy string
	Priority       string
	AllowReclaim   *bool
	Seed           int64
	SetCanonical   bool
}

// RemoveBackgroundInput is the RemoveBackground request in domain terms.
type RemoveBackgroundInput struct {
	BrandID       string
	SourceAssetID string
	ModelOverride string
	AllowBYOK     bool
	SetCanonical  bool
}

// DeriveIconsInput is the DeriveIcons request in domain terms. When none of the
// include flags is set, all variant families are produced.
type DeriveIconsInput struct {
	BrandID           string
	SourceAssetID     string
	IncludeMaskable   bool
	IncludeAppleTouch bool
	IncludeFavicon    bool
}

// iconVariant is one deterministic derived icon: a target size and whether it is
// flattened onto a solid brand-color background (opaque) or kept transparent.
type iconVariant struct {
	kind     string
	filename string
	width    int
	height   int
	solid    bool
}

var (
	faviconVariants = []iconVariant{
		{kind: "favicon-16", filename: "favicon-16.png", width: 16, height: 16},
		{kind: "favicon-32", filename: "favicon-32.png", width: 32, height: 32},
		{kind: "favicon-196", filename: "favicon-196.png", width: 196, height: 196},
	}
	appleTouchVariants = []iconVariant{
		{kind: "apple-touch-icon", filename: "apple-touch-icon.png", width: 180, height: 180, solid: true},
	}
	maskableVariants = []iconVariant{
		{kind: "maskable-192", filename: "maskable-icon-192.png", width: 192, height: 192, solid: true},
		{kind: "maskable-512", filename: "maskable-icon-512.png", width: 512, height: 512, solid: true},
	}
)

type service struct {
	providers Providers
	images    ImageBackend
	brands    BrandStore
	assets    AssetStore
	idGen     func() string
	logger    *log.Logger
}

// NewService constructs the production Service. A nil logger defaults to
// log.Default(); a nil idGen defaults to a random uuid.
func NewService(providers Providers, images ImageBackend, brands BrandStore, assets AssetStore, idGen func() string, logger *log.Logger) Service {
	if logger == nil {
		logger = log.Default()
	}
	if idGen == nil {
		idGen = func() string { return uuid.NewString()[:8] }
	}
	return &service{providers: providers, images: images, brands: brands, assets: assets, idGen: idGen, logger: logger}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) ProviderStatus(ctx context.Context) (bool, []ProviderStatus) {
	return s.providers.Available(ctx), s.providers.Statuses(ctx)
}

func (s *service) ImageBackendStatus(ctx context.Context) ImageBackendStatus {
	return s.images.Status(ctx)
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

func (s *service) GenerateImage(ctx context.Context, in GenerateImageInput) (ImageResult, error) {
	brandID := strings.TrimSpace(in.BrandID)
	if brandID == "" {
		return ImageResult{}, ErrInvalidGeneration{Field: "brand_id", Reason: "required"}
	}
	imageType := strings.ToLower(strings.TrimSpace(in.Type))
	spec, ok := imageSpecs[imageType]
	if !ok {
		return ImageResult{}, ErrInvalidGeneration{Field: "type", Reason: "must be 'logo' or 'favicon'"}
	}

	brand, err := s.brands.Get(ctx, brandID)
	if err != nil {
		return ImageResult{}, err
	}

	prompt := logoPrompt(brand.Name, brand.Description, brand.PrimaryColor)
	if imageType == kindFavicon {
		prompt = faviconPrompt(brand.Name, brand.PrimaryColor)
	}

	out, err := s.images.Generate(ctx, ImageGenerateRequest{
		Prompt:         prompt,
		NegativePrompt: logoNegativePrompt,
		Width:          spec.width,
		Height:         spec.height,
		ModelOverride:  in.ModelOverride,
		AllowBYOK:      brandImageAllowBYOK(in.AllowBYOK, in.FallbackPolicy),
		QualityPolicy:  brandImageQualityPolicy(in.QualityPolicy),
		FallbackPolicy: brandImageFallbackPolicy(in.FallbackPolicy),
		Priority:       brandImagePriority(in.Priority),
		AllowReclaim:   brandImageAllowReclaim(in.AllowReclaim),
		Seed:           in.Seed,
	})
	if err != nil {
		return ImageResult{}, err
	}
	return s.storeImage(ctx, brandID, imageType, out, in.SetCanonical)
}

func (s *service) EditImage(ctx context.Context, in EditImageInput) (ImageResult, error) {
	brandID := strings.TrimSpace(in.BrandID)
	if brandID == "" {
		return ImageResult{}, ErrInvalidGeneration{Field: "brand_id", Reason: "required"}
	}
	instruction := strings.TrimSpace(in.Instruction)
	if instruction == "" {
		return ImageResult{}, ErrInvalidGeneration{Field: "instruction", Reason: "required"}
	}
	src, err := s.loadSource(ctx, brandID, in.SourceAssetID)
	if err != nil {
		return ImageResult{}, err
	}

	out, err := s.images.Edit(ctx, ImageEditRequest{
		Source:         src.Content,
		Instruction:    instruction,
		ModelOverride:  in.ModelOverride,
		AllowBYOK:      brandImageAllowBYOK(in.AllowBYOK, in.FallbackPolicy),
		QualityPolicy:  brandImageQualityPolicy(in.QualityPolicy),
		FallbackPolicy: brandImageFallbackPolicy(in.FallbackPolicy),
		Priority:       brandImagePriority(in.Priority),
		AllowReclaim:   brandImageAllowReclaim(in.AllowReclaim),
		Seed:           in.Seed,
	})
	if err != nil {
		return ImageResult{}, err
	}
	return s.storeImage(ctx, brandID, kindLogo, out, in.SetCanonical)
}

func brandImageAllowBYOK(_ bool, fallbackPolicy string) bool {
	if strings.EqualFold(strings.TrimSpace(fallbackPolicy), "local_only") {
		return false
	}
	return true
}

func brandImageQualityPolicy(policy string) string {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case "fast", "balanced":
		return strings.ToLower(strings.TrimSpace(policy))
	default:
		return "quality"
	}
}

func brandImageFallbackPolicy(policy string) string {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case "local_only", "cloud_allowed":
		return strings.ToLower(strings.TrimSpace(policy))
	default:
		return "any"
	}
}

func brandImagePriority(priority string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "batch", "interactive":
		return strings.ToLower(strings.TrimSpace(priority))
	default:
		return "service"
	}
}

func brandImageAllowReclaim(allow *bool) bool {
	if allow == nil {
		return true
	}
	return *allow
}

func (s *service) RemoveBackground(ctx context.Context, in RemoveBackgroundInput) (ImageResult, error) {
	brandID := strings.TrimSpace(in.BrandID)
	if brandID == "" {
		return ImageResult{}, ErrInvalidGeneration{Field: "brand_id", Reason: "required"}
	}
	src, err := s.loadSource(ctx, brandID, in.SourceAssetID)
	if err != nil {
		return ImageResult{}, err
	}

	out, err := s.images.RemoveBackground(ctx, ImageRemoveBackgroundRequest{
		Source:        src.Content,
		ModelOverride: in.ModelOverride,
		AllowBYOK:     in.AllowBYOK,
	})
	if err != nil {
		return ImageResult{}, err
	}
	return s.storeImage(ctx, brandID, kindLogoTransparent, out, in.SetCanonical)
}

func (s *service) DeriveIcons(ctx context.Context, in DeriveIconsInput) ([]ImageResult, []string, error) {
	brandID := strings.TrimSpace(in.BrandID)
	if brandID == "" {
		return nil, nil, ErrInvalidGeneration{Field: "brand_id", Reason: "required"}
	}
	brand, err := s.brands.Get(ctx, brandID)
	if err != nil {
		return nil, nil, err
	}
	src, err := s.loadSource(ctx, brandID, in.SourceAssetID)
	if err != nil {
		return nil, nil, err
	}

	background := normalizeHexColor(brand.PrimaryColor)
	variants := selectIconVariants(in)

	results := make([]ImageResult, 0, len(variants))
	var warnings []string
	for _, v := range variants {
		var out ImageOutput
		var derr error
		if v.solid {
			out, derr = s.images.Flatten(ctx, src.Content, v.width, v.height, background)
		} else {
			out, derr = s.images.Resize(ctx, src.Content, v.width, v.height)
		}
		if derr != nil {
			return nil, nil, derr
		}
		res, serr := s.storeDerived(ctx, brandID, v, out)
		if serr != nil {
			return nil, nil, serr
		}
		results = append(results, res)
		warnings = append(warnings, out.Warnings...)
	}
	return results, warnings, nil
}

// selectIconVariants resolves the requested variant families. When no include
// flag is set, every family is produced (empty selection = all).
func selectIconVariants(in DeriveIconsInput) []iconVariant {
	all := !in.IncludeMaskable && !in.IncludeAppleTouch && !in.IncludeFavicon
	var variants []iconVariant
	if all || in.IncludeFavicon {
		variants = append(variants, faviconVariants...)
	}
	if all || in.IncludeAppleTouch {
		variants = append(variants, appleTouchVariants...)
	}
	if all || in.IncludeMaskable {
		variants = append(variants, maskableVariants...)
	}
	return variants
}

// loadSource validates the brand exists and loads the source asset bytes.
func (s *service) loadSource(ctx context.Context, brandID, sourceAssetID string) (AssetBytes, error) {
	sourceAssetID = strings.TrimSpace(sourceAssetID)
	if sourceAssetID == "" {
		return AssetBytes{}, ErrInvalidGeneration{Field: "source_asset_id", Reason: "required"}
	}
	if _, err := s.brands.Get(ctx, brandID); err != nil {
		return AssetBytes{}, err
	}
	src, err := s.assets.Read(ctx, sourceAssetID)
	if err != nil {
		return AssetBytes{}, err
	}
	return src, nil
}

// storeImage persists an exploratory image under a unique per-kind filename
// (never clobbering a prior result), then promotes it to the canonical per-kind
// asset when set_canonical is true OR the brand has no canonical for this kind
// yet (so a first generation is usable while an existing one is preserved).
func (s *service) storeImage(ctx context.Context, brandID, kind string, out ImageOutput, setCanonical bool) (ImageResult, error) {
	mime := out.MimeType
	if mime == "" {
		mime = "image/png"
	}
	stored, err := s.assets.Store(ctx, AssetUpload{
		BrandID:  brandID,
		Filename: s.uniqueFilename(kind),
		MimeType: mime,
		Content:  out.Data,
	})
	if err != nil {
		return ImageResult{}, err
	}

	canonical := setCanonical
	if !canonical {
		exists, existsErr := s.assets.Exists(ctx, brandID, canonicalFilename(kind))
		if existsErr != nil {
			return ImageResult{}, existsErr
		}
		canonical = !exists
	}
	if canonical {
		if _, err := s.assets.Store(ctx, AssetUpload{
			BrandID:  brandID,
			Filename: canonicalFilename(kind),
			MimeType: mime,
			Content:  out.Data,
		}); err != nil {
			return ImageResult{}, err
		}
	}

	return ImageResult{
		BrandID:   brandID,
		AssetID:   stored.ID,
		Kind:      kind,
		Filename:  stored.Filename,
		MimeType:  stored.MimeType,
		Size:      stored.Size,
		ModelID:   out.ModelID,
		Tier:      out.Tier,
		Canonical: canonical,
		Warnings:  out.Warnings,
	}, nil
}

// storeDerived persists a deterministic icon variant under its stable canonical
// filename (idempotent: re-deriving overwrites byte-identically).
func (s *service) storeDerived(ctx context.Context, brandID string, v iconVariant, out ImageOutput) (ImageResult, error) {
	mime := out.MimeType
	if mime == "" {
		mime = "image/png"
	}
	stored, err := s.assets.Store(ctx, AssetUpload{
		BrandID:  brandID,
		Filename: v.filename,
		MimeType: mime,
		Content:  out.Data,
	})
	if err != nil {
		return ImageResult{}, err
	}
	return ImageResult{
		BrandID:   brandID,
		AssetID:   stored.ID,
		Kind:      v.kind,
		Filename:  stored.Filename,
		MimeType:  stored.MimeType,
		Size:      stored.Size,
		Tier:      "deterministic",
		Canonical: true,
	}, nil
}

func (s *service) uniqueFilename(kind string) string {
	return kind + "-" + s.idGen() + ".png"
}

func canonicalFilename(kind string) string {
	return kind + ".png"
}

// normalizeHexColor returns a "#rrggbb" color for an icon background, defaulting
// to white when the brand color is absent or malformed.
func normalizeHexColor(c string) string {
	c = strings.TrimSpace(c)
	if len(c) == 7 && c[0] == '#' && isHex(c[1:]) {
		return c
	}
	return defaultIconBackground
}

func isHex(s string) bool {
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
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
