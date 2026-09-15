// Package presentation owns the pure, transport-independent LPBS product
// presentation contract. It deliberately has no knowledge of storage,
// commerce, delivery, HTTP, or the filesystem.
package presentation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

const SchemaVersion = 1

type Mode string

const (
	ModeEmpty     Mode = "empty"
	ModeSingleApp Mode = "single_app"
	ModeBundle    Mode = "bundle"
	ModeAppDetail Mode = "app_detail"
)

type Scope string

const (
	ScopeBundle Scope = "bundle"
	ScopeApp    Scope = "app"
)

type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

type Publication string

const (
	PublicationDraft     Publication = "draft"
	PublicationPublished Publication = "published"
)

type CapabilityStatus string

const (
	CapabilityAvailable  CapabilityStatus = "available"
	CapabilityPreview    CapabilityStatus = "preview"
	CapabilityComingSoon CapabilityStatus = "coming-soon"
)

type ActionKind string

const (
	ActionOpen          ActionKind = "open"
	ActionDownload      ActionKind = "download"
	ActionPurchase      ActionKind = "purchase"
	ActionRequestAccess ActionKind = "request-access"
	ActionUnavailable   ActionKind = "unavailable"
	ActionAnchor        ActionKind = "anchor"
	ActionAppDetail     ActionKind = "app-detail"
)

type BlockKind string

const (
	BlockProductHero       BlockKind = "product-hero"
	BlockBundleHero        BlockKind = "bundle-hero"
	BlockCapabilityStrip   BlockKind = "capability-strip"
	BlockProductStory      BlockKind = "product-story"
	BlockProductDemo       BlockKind = "product-demo"
	BlockAppSpotlights     BlockKind = "app-spotlights"
	BlockArtifactExplorer  BlockKind = "artifact-explorer"
	BlockVoiceStory        BlockKind = "voice-story"
	BlockDeviceStory       BlockKind = "device-story"
	BlockCapabilityRoadmap BlockKind = "capability-roadmap"
	BlockPricing           BlockKind = "pricing"
	BlockClosingAction     BlockKind = "closing-action"
	BlockFAQ               BlockKind = "faq"
	BlockFooter            BlockKind = "footer"
)

type ArtifactKind string

const (
	ArtifactPlan        ArtifactKind = "plan"
	ArtifactImage       ArtifactKind = "image"
	ArtifactHTMLPreview ArtifactKind = "html-preview"
	ArtifactVideo       ArtifactKind = "video"
	ArtifactAudio       ArtifactKind = "audio"
	ArtifactCode        ArtifactKind = "code"
	ArtifactPDF         ArtifactKind = "pdf"
)

// Document is the immutable, versioned presentation input owned by the
// configuration layer. Any value stored in Content must satisfy the v1 block
// contract for its Kind.
type Document struct {
	SchemaVersion int                          `json:"schema_version"`
	Bundle        Bundle                       `json:"bundle"`
	Apps          []App                        `json:"apps"`
	Pages         []Page                       `json:"pages"`
	Assets        []Asset                      `json:"assets"`
	Fixtures      []Fixture                    `json:"fixtures,omitempty"`
	Strings       map[string]map[string]string `json:"strings,omitempty"`
}

func (document Document) Validate() error { return Validate(document) }

type Bundle struct {
	Key           string   `json:"key"`
	Name          string   `json:"name"`
	AppOrder      []string `json:"app_order"`
	MaxAppSlides  int      `json:"max_app_slides"`
	PageID        string   `json:"page_id"`
	EmptyPageID   string   `json:"empty_page_id"`
	DefaultLocale string   `json:"default_locale"`
	Locales       []string `json:"locales"`
}

type App struct {
	Key             string       `json:"key"`
	Slug            string       `json:"slug"`
	Name            string       `json:"name"`
	Enabled         bool         `json:"enabled"`
	Visibility      Visibility   `json:"visibility"`
	Publication     Publication  `json:"publication"`
	PageID          string       `json:"page_id"`
	Tagline         string       `json:"tagline"`
	Description     string       `json:"description"`
	Capabilities    []Capability `json:"capabilities,omitempty"`
	PreservationRef string       `json:"preservation_ref,omitempty"`
}

type Capability struct {
	ID                   string              `json:"id"`
	Label                string              `json:"label"`
	Benefits             []string            `json:"benefits"`
	Status               CapabilityStatus    `json:"status"`
	EvidenceRefs         []string            `json:"evidence_refs,omitempty"`
	OwnerQualification   *OwnerQualification `json:"owner_qualification,omitempty"`
	Constraints          []string            `json:"constraints,omitempty"`
	ProviderRequirements []string            `json:"provider_requirements,omitempty"`
	PlatformRequirements []string            `json:"platform_requirements,omitempty"`
	LocalizedLabels      map[string]string   `json:"localized_labels,omitempty"`
	LocalizedBenefits    map[string][]string `json:"localized_benefits,omitempty"`
	StatusLabel          string              `json:"status_label,omitempty"`
	StatusLabels         map[string]string   `json:"status_labels,omitempty"`
}

// OwnerQualification is an explicit release-owner attestation. EvidenceRefs
// are structural/private references only; they are never sufficient to make
// an available claim. Validate checks this attestation's shape, while the
// owner that supplies it remains responsible for its real-world verification.
type OwnerQualification struct {
	Owner       string `json:"owner"`
	EvidenceRef string `json:"evidence_ref"`
	ReleaseRef  string `json:"release_ref"`
	Qualified   bool   `json:"qualified"`
}

type Page struct {
	Display     PageDisplay `json:"display"`
	ID          string      `json:"id"`
	Locale      string      `json:"locale"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Theme       Theme       `json:"theme"`
	Navigation  Navigation  `json:"navigation"`
	Blocks      []Block     `json:"blocks"`
	Footer      Footer      `json:"footer"`
}

type Theme struct {
	Variant    string `json:"variant"`
	Primary    string `json:"primary"`
	Background string `json:"background"`
	Accent     string `json:"accent"`
}

type Navigation struct {
	Label string           `json:"label"`
	Items []NavigationItem `json:"items"`
}

type NavigationItem struct {
	Label           string `json:"label"`
	AccessibleLabel string `json:"accessible_label"`
	Target          string `json:"target"`
}

type Footer struct {
	Label string           `json:"label"`
	Links []NavigationItem `json:"links"`
}

type Block struct {
	ID      string       `json:"id"`
	Kind    BlockKind    `json:"kind"`
	Version int          `json:"version"`
	Variant string       `json:"variant"`
	Content BlockContent `json:"content"`
}

// BlockContent is the closed v1 renderer input union. JSON decoding chooses a
// concrete member from Block.Kind; callers should not need to type-assert a
// map to render a block.
type BlockContent interface{ presentationBlockContent() }

type unknownBlockContent struct{}

func (unknownBlockContent) presentationBlockContent() {}

// UnmarshalJSON keeps the wire shape ergonomic for Astra while preserving a
// closed typed union in memory. Unknown kinds are retained as an invalid
// sentinel so Validate can return an actionable schema error.
func (b *Block) UnmarshalJSON(data []byte) error {
	var envelope struct {
		ID      string          `json:"id"`
		Kind    BlockKind       `json:"kind"`
		Version int             `json:"version"`
		Variant string          `json:"variant"`
		Content json.RawMessage `json:"content"`
	}
	if err := strictJSONUnmarshal(data, &envelope); err != nil {
		return err
	}
	b.ID, b.Kind, b.Version, b.Variant = envelope.ID, envelope.Kind, envelope.Version, envelope.Variant
	var content BlockContent
	switch envelope.Kind {
	case BlockProductHero:
		content = &ProductHeroContent{}
	case BlockBundleHero:
		content = &BundleHeroContent{}
	case BlockCapabilityStrip:
		content = &CapabilityStripContent{}
	case BlockProductStory:
		content = &ProductStoryContent{}
	case BlockProductDemo:
		content = &ProductDemoContent{}
	case BlockAppSpotlights:
		content = &AppSpotlightsContent{}
	case BlockArtifactExplorer:
		content = &ArtifactExplorerContent{}
	case BlockVoiceStory:
		content = &VoiceStoryContent{}
	case BlockDeviceStory:
		content = &DeviceStoryContent{}
	case BlockCapabilityRoadmap:
		content = &CapabilityRoadmapContent{}
	case BlockPricing:
		content = &PricingContent{}
	case BlockClosingAction:
		content = &ClosingActionContent{}
	case BlockFAQ:
		content = &FAQContent{}
	case BlockFooter:
		content = &FooterContent{}
	default:
		content = unknownBlockContent{}
	}
	if len(envelope.Content) != 0 && string(envelope.Content) != "null" && envelope.Kind != "" && validKinds[envelope.Kind] {
		if err := strictJSONUnmarshal(envelope.Content, content); err != nil {
			return fmt.Errorf("decode %s content: %w", envelope.Kind, err)
		}
	}
	switch typed := content.(type) {
	case *ProductHeroContent:
		b.Content = *typed
	case *BundleHeroContent:
		b.Content = *typed
	case *CapabilityStripContent:
		b.Content = *typed
	case *ProductStoryContent:
		b.Content = *typed
	case *ProductDemoContent:
		b.Content = *typed
	case *AppSpotlightsContent:
		b.Content = *typed
	case *ArtifactExplorerContent:
		b.Content = *typed
	case *VoiceStoryContent:
		b.Content = *typed
	case *DeviceStoryContent:
		b.Content = *typed
	case *CapabilityRoadmapContent:
		b.Content = *typed
	case *PricingContent:
		b.Content = *typed
	case *ClosingActionContent:
		b.Content = *typed
	case *FAQContent:
		b.Content = *typed
	case *FooterContent:
		b.Content = *typed
	default:
		b.Content = content
	}
	return nil
}

func strictJSONUnmarshal(data []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

type Asset struct {
	ID                     string          `json:"id"`
	ReleaseRef             string          `json:"release_ref"`
	ContentHash            string          `json:"content_hash"`
	Width                  int             `json:"width"`
	Height                 int             `json:"height"`
	MIME                   string          `json:"mime"`
	Surface                string          `json:"surface"`
	ResponsiveAlternatives []AssetVariant  `json:"responsive_alternatives,omitempty"`
	FocalPoint             FocalPoint      `json:"focal_point"`
	CropPolicy             string          `json:"crop_policy"`
	Provenance             AssetProvenance `json:"provenance"`
	OverlayRegions         []OverlayRegion `json:"overlay_regions"`
	PublicURL              string          `json:"public_url,omitempty"`
	PrivateEvidenceRefs    []string        `json:"private_evidence_refs,omitempty"`
}

type AssetVariant struct {
	Surface string `json:"surface"`
	AssetID string `json:"asset_id"`
}

type FocalPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type AssetProvenance struct {
	Provider     string `json:"provider"`
	JobRef       string `json:"job_ref"`
	CandidateRef string `json:"candidate_ref"`
}

type OverlayRegion struct {
	Name        string                `json:"name"`
	X           float64               `json:"x"`
	Y           float64               `json:"y"`
	Width       float64               `json:"width"`
	Height      float64               `json:"height"`
	Measurement LegibilityMeasurement `json:"measurement"`
}

type LegibilityVerdict string

const (
	LegibilityPass        LegibilityVerdict = "pass"
	LegibilityFail        LegibilityVerdict = "fail"
	LegibilityNotMeasured LegibilityVerdict = "not_measured"
)

// LegibilityMeasurement records measured contrast for one crop/surface. It
// is intentionally separate from OverlayRegion geometry: reserved space is
// not evidence that copy is readable there.
type LegibilityMeasurement struct {
	ContrastRatio        float64           `json:"contrast_ratio"`
	MinimumContrastRatio float64           `json:"minimum_contrast_ratio"`
	Threshold            float64           `json:"threshold"`
	Verdict              LegibilityVerdict `json:"verdict"`
	MeasurementRef       string            `json:"measurement_ref"`
}

type Action struct {
	Kind            ActionKind `json:"kind"`
	Label           string     `json:"label"`
	AccessibleLabel string     `json:"accessible_label"`
	Target          string     `json:"target,omitempty"`
	PlanRef         string     `json:"plan_ref,omitempty"`
	AppKey          string     `json:"app_key,omitempty"`
	Reason          string     `json:"reason,omitempty"`
}

type HeroItem struct {
	AppKey      string `json:"app_key"`
	VisualRef   string `json:"visual_ref"`
	ExhibitKind string `json:"exhibit_kind"`
	DetailLabel string `json:"detail_label"`
}

type ProductHeroContent struct {
	AppKey             string   `json:"app_key"`
	Eyebrow            string   `json:"eyebrow"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	VisualRef          string   `json:"visual_ref"`
	FixtureRef         string   `json:"fixture_ref,omitempty"`
	AccessibilityLabel string   `json:"accessibility_label"`
	Actions            []Action `json:"actions"`
}

func (ProductHeroContent) presentationBlockContent() {}

type BundleHeroContent struct {
	Eyebrow            string     `json:"eyebrow"`
	Title              string     `json:"title"`
	Description        string     `json:"description"`
	AccessibilityLabel string     `json:"accessibility_label"`
	HeroItems          []HeroItem `json:"hero_items"`
	Actions            []Action   `json:"actions"`
}

func (BundleHeroContent) presentationBlockContent() {}

type CapabilityItem struct {
	CapabilityID string `json:"capability_id"`
	Label        string `json:"label"`
	Description  string `json:"description"`
}

func (CapabilityStripContent) presentationBlockContent() {}

type CapabilityStripContent struct {
	Heading string           `json:"heading"`
	Items   []CapabilityItem `json:"items"`
}

func (ProductStoryContent) presentationBlockContent() {}

type StoryItem struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	VisualRef   string `json:"visual_ref,omitempty"`
	AltText     string `json:"alt_text,omitempty"`
}

func (ProductDemoContent) presentationBlockContent() {}

type ProductStoryContent struct {
	Heading string      `json:"heading"`
	Body    string      `json:"body"`
	Items   []StoryItem `json:"items"`
}

func (AppSpotlightsContent) presentationBlockContent() {}

type ProductDemoContent struct {
	Heading     string `json:"heading"`
	Description string `json:"description"`
	RendererRef string `json:"renderer_ref"`
	FixtureRef  string `json:"fixture_ref"`
	PosterRef   string `json:"poster_ref,omitempty"`
	MediaRef    string `json:"media_ref,omitempty"`
	AltText     string `json:"alt_text"`
}

func (ArtifactExplorerContent) presentationBlockContent() {}

type AppSpotlightsContent struct {
	Heading         string   `json:"heading"`
	AppKeys         []string `json:"app_keys"`
	DetailLinkLabel string   `json:"detail_link_label"`
}

func (VoiceStoryContent) presentationBlockContent() {}

type ArtifactExample struct {
	ID         string          `json:"id"`
	Kind       ArtifactKind    `json:"kind"`
	Label      string          `json:"label"`
	Filename   string          `json:"filename"`
	Type       string          `json:"type"`
	Title      string          `json:"title"`
	Body       string          `json:"body"`
	Steps      []string        `json:"steps,omitempty"`
	Checks     []string        `json:"checks,omitempty"`
	Caption    string          `json:"caption"`
	AltText    string          `json:"alt_text"`
	Icon       string          `json:"icon"`
	Width      int             `json:"width"`
	Height     int             `json:"height"`
	SourceRef  string          `json:"source_ref"`
	Source     ArtifactSource  `json:"source"`
	AssetRef   string          `json:"asset_ref,omitempty"`
	Brand      string          `json:"brand,omitempty"`
	Accent     string          `json:"accent,omitempty"`
	Frames     []ArtifactFrame `json:"frames,omitempty"`
	PreviewRef string          `json:"preview_ref,omitempty"`
}

// ArtifactSource is safe, non-executable source metadata. HTML examples use
// PreviewRef/Source.Isolation to point at the product viewer's isolated
// preview contract; they never carry executable markup in this domain.
type ArtifactSource struct {
	Ref           string `json:"ref"`
	MediaRef      string `json:"media_ref,omitempty"`
	TextExcerpt   string `json:"text_excerpt,omitempty"`
	IntegrityHash string `json:"integrity_hash,omitempty"`
	Isolation     string `json:"isolation,omitempty"`
}

type ArtifactFrame struct {
	AssetRef string `json:"asset_ref"`
	Time     string `json:"time"`
	Label    string `json:"label"`
}

func (DeviceStoryContent) presentationBlockContent() {}

type ArtifactExplorerContent struct {
	Heading           string            `json:"heading"`
	CapabilityID      string            `json:"capability_id"`
	Examples          []ArtifactExample `json:"examples"`
	SelectedExampleID string            `json:"selected_example_id"`
}

func (CapabilityRoadmapContent) presentationBlockContent() {}

type VoiceStoryContent struct {
	Eyebrow               string         `json:"eyebrow,omitempty"`
	Heading               string         `json:"heading"`
	Body                  string         `json:"body"`
	Features              []VoiceFeature `json:"features"`
	Note                  string         `json:"note"`
	InputLabel            string         `json:"input_label"`
	Transcript            string         `json:"transcript"`
	SummaryLabel          string         `json:"summary_label"`
	SummaryTitle          string         `json:"summary_title"`
	SummaryItems          []string       `json:"summary_items"`
	OutputLabel           string         `json:"output_label"`
	DemoNote              string         `json:"demo_note"`
	ProviderQualification string         `json:"provider_qualification"`
	Waveform              []float64      `json:"waveform"`
	CapabilityIDs         []string       `json:"capability_ids"`
}

type VoiceFeature struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	CapabilityID string `json:"capability_id"`
}

func (PricingContent) presentationBlockContent() {}

type DeviceStoryContent struct {
	Heading     string `json:"heading"`
	Description string `json:"description"`
	Device      string `json:"device"`
	VisualRef   string `json:"visual_ref"`
	AltText     string `json:"alt_text"`
}

func (ClosingActionContent) presentationBlockContent() {}

type CapabilityRoadmapContent struct {
	Heading       string   `json:"heading"`
	CapabilityIDs []string `json:"capability_ids"`
	StatusLabel   string   `json:"status_label"`
	Description   string   `json:"description"`
}

func (FAQContent) presentationBlockContent() {}

type PricingContent struct {
	Heading     string   `json:"heading"`
	Description string   `json:"description"`
	PlanRefs    []string `json:"plan_refs"`
	Actions     []Action `json:"actions"`
}

func (FooterContent) presentationBlockContent() {}

type ClosingActionContent struct {
	Heading     string   `json:"heading"`
	Description string   `json:"description"`
	Actions     []Action `json:"actions"`
	VisualRef   string   `json:"visual_ref,omitempty"`
}

type FAQItem struct {
	Question        string `json:"question"`
	Answer          string `json:"answer"`
	AccessibleLabel string `json:"accessible_label"`
}

type FAQContent struct {
	Heading string    `json:"heading"`
	Items   []FAQItem `json:"items"`
}

type FooterContent struct {
	Label string           `json:"label"`
	Links []NavigationItem `json:"links"`
}

type FixtureKind string

const (
	FixtureWorkspace FixtureKind = "workspace"
	FixtureBackdrop  FixtureKind = "backdrop"
	FixtureWorkflow  FixtureKind = "workflow"
)

// Fixture is public illustrative data selected by a finite renderer. It is
// stored in the document so every demo label, row, and example remains
// editable; it is never executable product UI.
type Fixture struct {
	ID        string            `json:"id"`
	Kind      FixtureKind       `json:"kind"`
	Workspace *WorkspaceFixture `json:"workspace,omitempty"`
	Backdrop  *BackdropFixture  `json:"backdrop,omitempty"`
	Workflow  *WorkflowFixture  `json:"workflow,omitempty"`
}

type WorkspaceFixture struct {
	Title         string   `json:"title"`
	Group         string   `json:"group"`
	GroupLabel    string   `json:"group_label"`
	Groups        []string `json:"groups"`
	SessionsLabel string   `json:"sessions_label"`
	Sessions      []string `json:"sessions"`
	Role          string   `json:"role"`
	Model         string   `json:"model"`
	Reviewer      string   `json:"reviewer"`
	ReviewerModel string   `json:"reviewer_model"`
	Branch        string   `json:"branch"`
	Prompt        string   `json:"prompt"`
	Answer        string   `json:"answer"`
	Files         []string `json:"files"`
	FileLabel     string   `json:"file_label"`
	Diff          []string `json:"diff"`
	Command       string   `json:"command"`
	Checks        []string `json:"checks"`
	Ready         string   `json:"ready"`
	Composer      string   `json:"composer"`
	ReturnLabel   string   `json:"return_label"`
	ReturnTitle   string   `json:"return_title"`
	ReviewMessage string   `json:"review_message"`
	MessageLabel  string   `json:"message_label"`
	ReplyLabel    string   `json:"reply_label"`
	Status        string   `json:"status"`
	Today         string   `json:"today"`
	Keyboard      []string `json:"keyboard"`
}

type BackdropFixture struct {
	Title     string   `json:"title"`
	Label     string   `json:"label"`
	Selected  string   `json:"selected"`
	Styles    []string `json:"styles"`
	Surface   string   `json:"surface"`
	Palette   string   `json:"palette"`
	Panel     string   `json:"panel"`
	Caption   string   `json:"caption"`
	Export    string   `json:"export"`
	Options   []string `json:"options"`
	Badge     string   `json:"badge"`
	AssetRefs []string `json:"asset_refs"`
}

type WorkflowStep struct {
	Number      string `json:"number"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type WorkflowFixture struct {
	Title        string         `json:"title"`
	Label        string         `json:"label"`
	Steps        []WorkflowStep `json:"steps"`
	BrowserTitle string         `json:"browser_title"`
	BrowserRows  []string       `json:"browser_rows"`
	Note         string         `json:"note"`
}

type PreviewRequest struct {
	Revision   string `json:"revision,omitempty"`
	Authorized bool   `json:"authorized"`
}

type ResolveRequest struct {
	Route             string `json:"route"`
	Locale            string `json:"locale,omitempty"`
	Variant           string `json:"variant,omitempty"`
	VariantAssignment string `json:"variant_assignment,omitempty"`
	// ResolvedVariant and ResolvedRevision are joins supplied by the
	// configuration/assignment owner. The presentation domain records them; it
	// does not maintain a variant registry or invent a fallback assignment.
	ResolvedVariant   string          `json:"resolved_variant,omitempty"`
	ResolvedRevision  string          `json:"resolved_revision,omitempty"`
	PreviewRevision   string          `json:"preview_revision,omitempty"`
	PreviewAuthorized bool            `json:"preview_authorized"`
	Preview           *PreviewRequest `json:"preview,omitempty"`
}

// Request is a short compatibility alias for callers that prefer the concise
// domain name.
type Request = ResolveRequest

type ResolvedBlock struct {
	ID      string       `json:"id"`
	Kind    BlockKind    `json:"kind"`
	Version int          `json:"version"`
	Variant string       `json:"variant"`
	Content BlockContent `json:"content"`
}

type ResolvedPage struct {
	Display     PageDisplay     `json:"display"`
	ID          string          `json:"id"`
	Locale      string          `json:"locale"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Theme       Theme           `json:"theme"`
	Navigation  Navigation      `json:"navigation"`
	Blocks      []ResolvedBlock `json:"blocks"`
	Footer      Footer          `json:"footer"`
}

type ResolvedCapability struct {
	ID                   string           `json:"id"`
	Label                string           `json:"label"`
	Benefits             []string         `json:"benefits"`
	Status               CapabilityStatus `json:"status"`
	StatusLabel          string           `json:"status_label"`
	Constraints          []string         `json:"constraints,omitempty"`
	ProviderRequirements []string         `json:"provider_requirements,omitempty"`
	PlatformRequirements []string         `json:"platform_requirements,omitempty"`
}

type ResolvedAppSpotlight struct {
	AppKey      string `json:"app_key"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Tagline     string `json:"tagline"`
	Description string `json:"description"`
	DetailRoute string `json:"detail_route"`
}

type Diagnostics struct {
	RequestedRoute      string   `json:"requested_route"`
	ResolvedRoute       string   `json:"resolved_route"`
	RequestedVariant    string   `json:"requested_variant"`
	ResolvedVariant     string   `json:"resolved_variant"`
	RequestedRevision   string   `json:"requested_revision,omitempty"`
	ResolvedRevision    string   `json:"resolved_revision,omitempty"`
	Locale              string   `json:"locale"`
	BundleKey           string   `json:"bundle_key"`
	AppKey              string   `json:"app_key,omitempty"`
	Mode                Mode     `json:"mode"`
	Fallback            bool     `json:"fallback"`
	FallbackReason      string   `json:"fallback_reason,omitempty"`
	Preview             bool     `json:"preview"`
	NoIndex             bool     `json:"noindex"`
	NoStore             bool     `json:"no_store"`
	EligibleAppKeys     []string `json:"eligible_app_keys"`
	BlockDigest         string   `json:"block_digest"`
	AssetReleaseRefs    []string `json:"asset_release_refs"`
	CommerceSnapshotRef string   `json:"commerce_snapshot_ref,omitempty"`
}

type ResolveResult struct {
	SchemaVersion   int                    `json:"schema_version"`
	Mode            Mode                   `json:"mode"`
	Scope           Scope                  `json:"scope"`
	AppKey          string                 `json:"app_key,omitempty"`
	Page            ResolvedPage           `json:"page"`
	SelectedAppKeys []string               `json:"selected_app_keys"`
	Spotlights      []ResolvedAppSpotlight `json:"spotlights,omitempty"`
	Capabilities    []ResolvedCapability   `json:"capabilities,omitempty"`
	Assets          []ResolvedAsset        `json:"assets,omitempty"`
	Fixtures        []Fixture              `json:"fixtures,omitempty"`
	Diagnostics     Diagnostics            `json:"diagnostics"`
}

type ResolvedAsset struct {
	ID                     string                  `json:"id"`
	ReleaseRef             string                  `json:"release_ref"`
	ContentHash            string                  `json:"content_hash"`
	Width                  int                     `json:"width"`
	Height                 int                     `json:"height"`
	MIME                   string                  `json:"mime"`
	Surface                string                  `json:"surface"`
	ResponsiveAlternatives []AssetVariant          `json:"responsive_alternatives,omitempty"`
	FocalPoint             FocalPoint              `json:"focal_point"`
	CropPolicy             string                  `json:"crop_policy"`
	Provenance             ResolvedAssetProvenance `json:"provenance"`
	OverlayRegions         []ResolvedOverlayRegion `json:"overlay_regions"`
	PublicURL              string                  `json:"public_url,omitempty"`
}

// ResolvedAssetProvenance contains only the public provider label. Job and
// candidate references remain internal evidence and are intentionally omitted
// from the public response.
type ResolvedAssetProvenance struct {
	Provider string `json:"provider"`
}

type ResolvedOverlayRegion struct {
	Name        string                        `json:"name"`
	X           float64                       `json:"x"`
	Y           float64                       `json:"y"`
	Width       float64                       `json:"width"`
	Height      float64                       `json:"height"`
	Measurement ResolvedLegibilityMeasurement `json:"measurement"`
}

// ResolvedLegibilityMeasurement intentionally omits MeasurementRef, which is
// private evidence metadata and must not cross the public response boundary.
type ResolvedLegibilityMeasurement struct {
	ContrastRatio        float64           `json:"contrast_ratio"`
	MinimumContrastRatio float64           `json:"minimum_contrast_ratio"`
	Threshold            float64           `json:"threshold"`
	Verdict              LegibilityVerdict `json:"verdict"`
}

var (
	ErrNotFound            = errors.New("presentation route not found")
	ErrPreviewUnauthorized = errors.New("presentation preview authorization required")
	ErrUnavailable         = errors.New("presentation unavailable")
)

type ValidationIssue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ValidationError struct {
	Issues []ValidationIssue `json:"issues"`
}

func (e *ValidationError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "presentation validation failed"
	}
	parts := make([]string, 0, len(e.Issues))
	for _, issue := range e.Issues {
		parts = append(parts, fmt.Sprintf("%s: %s", issue.Path, issue.Message))
	}
	return "presentation validation failed: " + strings.Join(parts, "; ")
}

func (e *ValidationError) add(path, code, message string) {
	e.Issues = append(e.Issues, ValidationIssue{Path: path, Code: code, Message: message})
}

var (
	idPattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
	slugPattern   = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	localePattern = regexp.MustCompile(`^[A-Za-z]{2,8}(?:-[A-Za-z0-9]{2,8})*$`)
	hashPattern   = regexp.MustCompile(`^(?:sha256:)?[a-fA-F0-9]{32,128}$`)
	colorPattern  = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

var validKinds = map[BlockKind]bool{
	BlockProductHero: true, BlockBundleHero: true, BlockCapabilityStrip: true,
	BlockProductStory: true, BlockProductDemo: true, BlockAppSpotlights: true,
	BlockArtifactExplorer: true, BlockVoiceStory: true, BlockDeviceStory: true,
	BlockCapabilityRoadmap: true, BlockPricing: true, BlockClosingAction: true,
	BlockFAQ: true, BlockFooter: true,
}

var validVariants = map[BlockKind]map[string]bool{
	BlockProductHero:       {"centered": true},
	BlockBundleHero:        {"editorial": true},
	BlockCapabilityStrip:   {"inline": true},
	BlockProductStory:      {"three-column": true},
	BlockProductDemo:       {"static": true, "interactive": true},
	BlockAppSpotlights:     {"grid": true},
	BlockArtifactExplorer:  {"tabs": true},
	BlockVoiceStory:        {"transcript": true, "waveform": true},
	BlockDeviceStory:       {"phone": true, "desktop": true},
	BlockCapabilityRoadmap: {"roadmap": true, "stacked": true},
	BlockPricing:           {"compact": true},
	BlockClosingAction:     {"plain": true, "artwork": true},
	BlockFAQ:               {"defined": true, "accordion": true},
	BlockFooter:            {"defined": true, "minimal": true},
}

var validArtifactKinds = map[ArtifactKind]bool{
	ArtifactPlan: true, ArtifactImage: true, ArtifactHTMLPreview: true,
	ArtifactVideo: true, ArtifactAudio: true, ArtifactCode: true, ArtifactPDF: true,
}

var validActionKinds = map[ActionKind]bool{
	ActionOpen: true, ActionDownload: true, ActionPurchase: true,
	ActionRequestAccess: true, ActionUnavailable: true, ActionAnchor: true,
	ActionAppDetail: true,
}

// NewBlock marshals a typed v1 content value into the same representation
// used by JSON documents. It is useful to callers constructing fixtures while
// retaining the strict validation performed by Validate.
func NewBlock(kind BlockKind, variant string, content any) (Block, error) {
	if _, ok := validKinds[kind]; !ok {
		return Block{}, fmt.Errorf("unsupported block kind %q", kind)
	}
	if content == nil {
		return Block{}, errors.New("block content is required")
	}
	blockContent, ok := content.(BlockContent)
	if !ok {
		return Block{}, errors.New("block content must be a registered typed content struct")
	}
	return Block{Kind: kind, Version: SchemaVersion, Variant: variant, Content: blockContent}, nil
}

// DecodeDocument is the canonical strict wire decoder for configuration
// stores. Unknown document, block, and typed-content properties are rejected;
// callers can then pass the result to Validate for semantic/reference checks.
func DecodeDocument(data []byte) (Document, error) {
	var document Document
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return Document{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Document{}, errors.New("document contains more than one JSON value")
		}
		return Document{}, err
	}
	return document, nil
}

func contentMap(content any) (map[string]any, error) {
	b, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	if err := json.Unmarshal(b, &value); err != nil || value == nil {
		return nil, errors.New("content must be a JSON object")
	}
	return value, nil
}

func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}
