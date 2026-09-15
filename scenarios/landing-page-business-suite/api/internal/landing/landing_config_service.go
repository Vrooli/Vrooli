// DOC: docs/reference/api/landing.md - Public landing page configuration API
// DOC: docs/concepts/CONCEPTS.md#data-flow-architecture - Data flow overview
// DOC: PRD.md#OT-P0-031 - API-driven typed landing configuration
package landing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/delivery"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/presentation"
)

// LandingConfigService resolves published presentation data and joins its
// public commerce facts.
type LandingConfigService struct {
	planService                  *commerce.PlanService
	downloadService              *delivery.CatalogService
	configStore                  *experimentation.ConfigStore
	presentationOwnerJoin        func(context.Context, string) (*commerce.PricingOverview, []delivery.App, error)
	presentationExposureRecorder interface {
		RecordPresentationExposure(context.Context, string, string, string, string, string, string, string) (bool, error)
	}
}

type PresentationAssignmentSource string

const (
	PresentationAssignmentWeightedVisitor PresentationAssignmentSource = "weighted_visitor"
	PresentationAssignmentExplicitURL     PresentationAssignmentSource = "explicit_url"
)

// PresentationExposureRequest is the schema-free domain proof passed by the
// public exposure RPC. The generated protocol owns its wire equivalent.
type PresentationExposureRequest struct {
	VisitorID         string
	VariantSlug       string
	Revision          string
	Route             string
	Locale            string
	BlockDigest       string
	WeightFingerprint string
	Source            PresentationAssignmentSource
}

func (s *LandingConfigService) UsePresentationExposureRecorder(recorder interface {
	RecordPresentationExposure(context.Context, string, string, string, string, string, string, string) (bool, error)
}) {
	s.presentationExposureRecorder = recorder
}

// LandingConfigResponse is returned by the typed public landing service.
type LandingConfigResponse struct {
	Pricing      *commerce.PricingOverview   `json:"pricing"`
	Downloads    []delivery.App              `json:"downloads"`
	Fallback     bool                        `json:"fallback"`
	Presentation *presentation.ResolveResult `json:"presentation,omitempty"`
}

// NewLandingConfigServiceWithConfigStore creates a LandingConfigService using ConfigStore (JSON files as source of truth)
func NewLandingConfigServiceWithConfigStore(
	configStore *experimentation.ConfigStore,
	planService *commerce.PlanService,
	downloadService *delivery.CatalogService,
) *LandingConfigService {
	service := &LandingConfigService{
		configStore:     configStore,
		planService:     planService,
		downloadService: downloadService,
	}
	service.presentationOwnerJoin = service.joinPresentationOwners
	return service
}

// GetLandingConfigForRequest resolves the typed presentation path only when
// an immutable published revision exists. Missing publication fails closed so
// mutable legacy marketing content cannot become a public substitute.
func (s *LandingConfigService) GetLandingConfigForRequest(ctx context.Context, variantSlug, route, locale, visitorID string) (*LandingConfigResponse, error) {
	selectedVariant, variants := s.presentationVariant(variantSlug, visitorID)
	if selectedVariant == "" {
		if presentationDetailRoute(route) {
			return nil, presentation.ErrNotFound
		}
		return nil, fmt.Errorf("%w: no eligible published presentation variant", presentation.ErrUnavailable)
	}

	loaded, err := s.configStore.GetPublishedPresentation(ctx, selectedVariant)
	if err != nil {
		if errors.Is(err, experimentation.ErrPresentationNotFound) {
			if presentationDetailRoute(route) {
				return nil, presentation.ErrNotFound
			}
			return nil, fmt.Errorf("%w: published presentation unavailable for variant %q", presentation.ErrUnavailable, selectedVariant)
		}
		return nil, err
	}
	resolved, err := presentation.Resolve(loaded.Document, presentation.ResolveRequest{
		Route: route, Locale: locale, Variant: variantSlug,
		ResolvedVariant: selectedVariant, ResolvedRevision: loaded.Revision,
	})
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(variantSlug) != "" {
		resolved.Diagnostics.AssignmentSource = string(PresentationAssignmentExplicitURL)
	} else if strings.TrimSpace(visitorID) != "" {
		resolved.Diagnostics.AssignmentSource = string(PresentationAssignmentWeightedVisitor)
		resolved.Diagnostics.WeightFingerprint = experimentation.WeightFingerprint(variants)
	}
	pricing, downloads, ownerErr := s.presentationOwners(ctx, loaded.Document.Bundle.Key)
	if ownerErr == nil {
		// Bundle validation above is intentionally performed against the raw
		// owner response. The typed presentation path then narrows commerce to
		// its immutable public facts before snapshots, action joins, or wire
		// conversion can observe arbitrary owner metadata or hidden plans.
		pricing = sanitizePresentationPricing(pricing)
		downloads = filterPresentationDownloads(resolved, downloads)
		resolved.Diagnostics.CommerceSnapshotRef = publicOwnerSnapshotRef(pricing, downloads)
	} else {
		pricing, downloads = nil, nil
	}
	resolved.Actions = presentation.ResolveActions(resolved)
	presentation.JoinActionOwners(&resolved, presentationOwnerObservations(resolved, pricing, downloads))
	return &LandingConfigResponse{
		Pricing: pricing, Downloads: downloads, Presentation: &resolved,
		Fallback: resolved.Diagnostics.Fallback,
	}, nil
}

// RecordPresentationExposure validates a read-only public resolution against
// the current deterministic assignment and active published revision before
// delegating the idempotent write to the context-aware metrics owner.
func (s *LandingConfigService) RecordPresentationExposure(ctx context.Context, request PresentationExposureRequest) (bool, error) {
	if request.Source != PresentationAssignmentWeightedVisitor {
		return false, fmt.Errorf("%w: explicit URL assignments are not exposures", presentation.ErrUnavailable)
	}
	visitorID := strings.TrimSpace(request.VisitorID)
	variantSlug := strings.TrimSpace(request.VariantSlug)
	revision := strings.TrimSpace(request.Revision)
	route := normalizePresentationExposureRoute(request.Route)
	locale := strings.TrimSpace(request.Locale)
	blockDigest := strings.TrimSpace(request.BlockDigest)
	weightFingerprint := strings.TrimSpace(request.WeightFingerprint)
	if visitorID == "" || variantSlug == "" || revision == "" || blockDigest == "" || weightFingerprint == "" {
		return false, fmt.Errorf("%w: incomplete presentation exposure proof", presentation.ErrUnavailable)
	}
	if s.configStore == nil || s.presentationExposureRecorder == nil {
		return false, fmt.Errorf("%w: presentation exposure owner is unavailable", presentation.ErrUnavailable)
	}
	variants := s.configStore.ListVariants()
	selectable := false
	for _, variant := range variants {
		if experimentation.VariantWeight(variant) > 0 {
			selectable = true
			break
		}
	}
	if !selectable {
		return false, fmt.Errorf("%w: no selectable presentation variant", presentation.ErrUnavailable)
	}
	selected := experimentation.SelectVariantForVisitor(variants, visitorID)
	if selected == nil || selected.Variant.Slug != variantSlug {
		return false, fmt.Errorf("%w: variant is not the deterministic assignment for visitor", presentation.ErrUnavailable)
	}
	if expected := experimentation.WeightFingerprint(variants); expected != weightFingerprint {
		return false, fmt.Errorf("%w: assignment weights changed", presentation.ErrUnavailable)
	}
	loaded, err := s.configStore.GetPublishedPresentation(ctx, variantSlug)
	if err != nil {
		if errors.Is(err, experimentation.ErrPresentationNotFound) {
			return false, fmt.Errorf("%w: published presentation is unavailable", presentation.ErrUnavailable)
		}
		return false, err
	}
	if loaded.Revision != revision {
		return false, fmt.Errorf("%w: published revision changed", presentation.ErrUnavailable)
	}
	if locale == "" {
		locale = loaded.Document.Bundle.DefaultLocale
	}
	resolved, err := presentation.Resolve(loaded.Document, presentation.ResolveRequest{
		Route: route, Locale: locale, ResolvedVariant: variantSlug, ResolvedRevision: revision,
	})
	if err != nil {
		return false, err
	}
	if resolved.Diagnostics.ResolvedRoute != route || resolved.Diagnostics.Locale != locale || resolved.Diagnostics.ResolvedVariant != variantSlug || resolved.Diagnostics.ResolvedRevision != revision || resolved.Diagnostics.BlockDigest != blockDigest {
		return false, fmt.Errorf("%w: public presentation proof does not match current resolution", presentation.ErrUnavailable)
	}
	return s.presentationExposureRecorder.RecordPresentationExposure(ctx, visitorID, variantSlug, revision, route, locale, blockDigest, weightFingerprint)
}

func normalizePresentationExposureRoute(route string) string {
	route = strings.TrimSpace(route)
	if route == "" {
		return "/"
	}
	route = strings.TrimRight(route, "/")
	if route == "" {
		return "/"
	}
	return route
}

func presentationDetailRoute(route string) bool {
	clean := strings.TrimRight(strings.TrimSpace(route), "/")
	return strings.HasPrefix(clean, "/apps/") && len(clean) > len("/apps/")
}

func filterPresentationDownloads(result presentation.ResolveResult, downloads []delivery.App) []delivery.App {
	allowed := map[string]bool{}
	if result.Mode == presentation.ModeAppDetail {
		allowed[result.AppKey] = true
	} else {
		for _, key := range result.Diagnostics.EligibleAppKeys {
			allowed[key] = true
		}
	}
	filtered := make([]delivery.App, 0, len(downloads))
	for _, app := range downloads {
		if app.BundleKey == result.Diagnostics.BundleKey && allowed[app.AppKey] {
			filtered = append(filtered, sanitizePresentationDownload(app))
		}
	}
	return filtered
}

func sanitizePresentationDownload(app delivery.App) delivery.App {
	if len(app.Metadata) == 0 {
		return app
	}
	metadata := make(map[string]interface{}, len(app.Metadata))
	for key, value := range app.Metadata {
		metadata[key] = value
	}
	if raw, ok := metadata["web_url"].(string); ok {
		if safe := presentationWebURLValue(raw); safe != "" {
			metadata["web_url"] = safe
		} else {
			delete(metadata, "web_url")
		}
	}
	app.Metadata = metadata
	return app
}

func (s *LandingConfigService) presentationVariant(variantSlug, visitorID string) (string, []*experimentation.VariantSnapshot) {
	if s.configStore == nil {
		return "", nil
	}
	if variantSlug != "" {
		// Retained immutable revisions are history, not a second admission
		// registry. Deleting or archiving a variant must revoke public reads.
		variant, err := s.configStore.GetVariant(variantSlug)
		if err != nil || variant == nil || experimentation.NormalizeVariantStatus(variant.Variant.Status) != "active" {
			return "", nil
		}
		return variantSlug, nil
	}
	variants := s.configStore.ListVariants()
	selectable := false
	for _, variant := range variants {
		if experimentation.VariantWeight(variant) > 0 {
			selectable = true
			break
		}
	}
	if !selectable {
		return "", variants
	}
	selected := experimentation.SelectVariantForVisitor(variants, visitorID)
	if selected == nil {
		return "", variants
	}
	return selected.Variant.Slug, variants
}

func (s *LandingConfigService) presentationOwners(ctx context.Context, bundleKey string) (*commerce.PricingOverview, []delivery.App, error) {
	if s.presentationOwnerJoin == nil {
		return nil, nil, fmt.Errorf("%w: commerce and delivery owner join is unavailable", presentation.ErrUnavailable)
	}
	pricing, downloads, err := s.presentationOwnerJoin(ctx, bundleKey)
	if err != nil {
		return nil, nil, err
	}
	if err := validatePresentationOwnerBundle(bundleKey, pricing, downloads); err != nil {
		return nil, nil, err
	}
	return pricing, downloads, nil
}

func (s *LandingConfigService) joinPresentationOwners(ctx context.Context, bundleKey string) (*commerce.PricingOverview, []delivery.App, error) {
	if s.planService == nil || s.downloadService == nil {
		return nil, nil, fmt.Errorf("%w: commerce and delivery owner join is unavailable", presentation.ErrUnavailable)
	}
	pricing, err := s.planService.GetPricingOverviewForBundle(ctx, bundleKey)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: pricing owner unavailable", presentation.ErrUnavailable)
	}
	downloads, err := s.downloadService.ListAppsContext(ctx, bundleKey)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: delivery owner unavailable", presentation.ErrUnavailable)
	}
	if err := validatePresentationOwnerBundle(bundleKey, pricing, downloads); err != nil {
		return nil, nil, err
	}
	return pricing, downloads, nil
}

func validatePresentationOwnerBundle(bundleKey string, pricing *commerce.PricingOverview, downloads []delivery.App) error {
	expected := strings.TrimSpace(bundleKey)
	if expected == "" {
		return fmt.Errorf("%w: resolved presentation bundle is empty", presentation.ErrUnavailable)
	}
	if pricing == nil || pricing.Bundle == nil || pricing.Bundle.BundleKey != expected {
		return fmt.Errorf("%w: pricing owner bundle does not match resolved presentation bundle %q", presentation.ErrUnavailable, expected)
	}
	plans := append(append(append([]*commerce.PlanOption{}, pricing.Monthly...), pricing.Yearly...), pricing.CreditTopups...)
	for _, plan := range plans {
		if plan != nil && plan.BundleKey != expected {
			return fmt.Errorf("%w: pricing plan %q belongs to bundle %q, want %q", presentation.ErrUnavailable, plan.StripePriceId, plan.BundleKey, expected)
		}
	}
	for _, app := range downloads {
		if app.BundleKey != expected {
			return fmt.Errorf("%w: delivery app %q belongs to bundle %q, want %q", presentation.ErrUnavailable, app.AppKey, app.BundleKey, expected)
		}
		for _, asset := range app.Platforms {
			if asset.BundleKey != expected || asset.AppKey != app.AppKey {
				return fmt.Errorf("%w: delivery platform does not belong to its resolved bundle/app", presentation.ErrUnavailable)
			}
		}
	}
	return nil
}

func presentationOwnerObservations(result presentation.ResolveResult, pricing *commerce.PricingOverview, downloads []delivery.App) presentation.ActionOwnerObservations {
	observations := presentation.ActionOwnerObservations{Downloads: map[string]presentation.ActionOwnerObservation{}, Purchases: map[string]presentation.ActionOwnerObservation{}, Opens: map[string]presentation.ActionOwnerObservation{}}
	for _, app := range downloads {
		if app.BundleKey != result.Diagnostics.BundleKey {
			continue
		}
		if webURL := presentationWebURL(app.Metadata); webURL != "" {
			observations.Opens[app.AppKey] = presentation.ActionOwnerObservation{Ready: true, Href: webURL}
		}
		if !presentationDownloadReady(app) {
			continue
		}
		href := presentation.CanonicalAppHref(result, app.AppKey)
		if href != "" {
			observations.Downloads[app.AppKey] = presentation.ActionOwnerObservation{Ready: true, Href: href + "/download"}
		}
	}
	if pricing == nil || pricing.Bundle == nil || pricing.Bundle.BundleKey != result.Diagnostics.BundleKey {
		return observations
	}
	for _, plan := range append(append(append([]*commerce.PlanOption{}, pricing.Monthly...), pricing.Yearly...), pricing.CreditTopups...) {
		if plan == nil || !plan.DisplayEnabled || plan.StripePriceId == "" || plan.BundleKey != result.Diagnostics.BundleKey {
			continue
		}
		observations.Purchases[plan.StripePriceId] = presentation.ActionOwnerObservation{
			Ready: true, Href: "/checkout?price_id=" + url.QueryEscape(plan.StripePriceId),
		}
	}
	return observations
}

func presentationWebURL(metadata map[string]interface{}) string {
	raw, ok := metadata["web_url"].(string)
	if !ok {
		return ""
	}
	return presentationWebURLValue(raw)
}

func presentationWebURLValue(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" || strings.ContainsAny(value, "\\\x00\r\n\t") || strings.HasPrefix(value, "//") {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.User != nil || strings.ContainsAny(parsed.Path, "\\\x00\r\n\t") {
		return ""
	}
	if parsed.IsAbs() {
		if parsed.Scheme != "https" || parsed.Host == "" {
			return ""
		}
		return value
	}
	if parsed.Host != "" || !strings.HasPrefix(value, "/") || strings.Contains(parsed.Path, "..") {
		return ""
	}
	return value
}

func presentationDownloadReady(app delivery.App) bool {
	for _, asset := range app.Platforms {
		if asset.BundleKey != app.BundleKey || asset.AppKey != app.AppKey {
			continue
		}
		if strings.TrimSpace(asset.Platform) == "" || strings.TrimSpace(asset.ReleaseVersion) == "" {
			continue
		}
		if asset.ArtifactURL != "" {
			if delivery.ValidateDirectArtifactURL(asset.ArtifactURL) == nil {
				return true
			}
			continue
		}
		if asset.ArtifactID != nil && *asset.ArtifactID > 0 {
			return true
		}
	}
	return false
}

type publicPlanSnapshot struct {
	StripePriceID string `json:"stripe_price_id"`
	BundleKey     string `json:"bundle_key"`
	PlanName      string `json:"plan_name"`
	PlanTier      string `json:"plan_tier"`
	Billing       string `json:"billing_interval"`
	AmountCents   int64  `json:"amount_cents"`
	Currency      string `json:"currency"`
	Display       bool   `json:"display_enabled"`
}

type publicAssetSnapshot struct {
	AppKey              string `json:"app_key"`
	Platform            string `json:"platform"`
	ReleaseVersion      string `json:"release_version"`
	ReleaseNotes        string `json:"release_notes,omitempty"`
	Checksum            string `json:"checksum,omitempty"`
	RequiresEntitlement bool   `json:"requires_entitlement"`
}

type publicAppSnapshot struct {
	AppKey      string                `json:"app_key"`
	Name        string                `json:"name"`
	Tagline     string                `json:"tagline,omitempty"`
	Description string                `json:"description,omitempty"`
	WebURL      string                `json:"web_url,omitempty"`
	Platforms   []publicAssetSnapshot `json:"platforms"`
}

func publicOwnerSnapshotRef(pricing *commerce.PricingOverview, downloads []delivery.App) string {
	type snapshot struct {
		Bundle    string               `json:"bundle"`
		Monthly   []publicPlanSnapshot `json:"monthly"`
		Yearly    []publicPlanSnapshot `json:"yearly"`
		Topups    []publicPlanSnapshot `json:"credit_topups"`
		Downloads []publicAppSnapshot  `json:"downloads"`
	}
	value := snapshot{
		Monthly:   []publicPlanSnapshot{},
		Yearly:    []publicPlanSnapshot{},
		Topups:    []publicPlanSnapshot{},
		Downloads: []publicAppSnapshot{},
	}
	if pricing != nil && pricing.Bundle != nil {
		value.Bundle = pricing.Bundle.BundleKey
		appendPlans := func(target *[]publicPlanSnapshot, plans []*commerce.PlanOption) {
			for _, plan := range plans {
				if plan == nil {
					continue
				}
				*target = append(*target, publicPlanSnapshot{StripePriceID: plan.StripePriceId, BundleKey: plan.BundleKey, PlanName: plan.PlanName, PlanTier: plan.PlanTier, Billing: plan.BillingInterval.String(), AmountCents: plan.AmountCents, Currency: plan.Currency, Display: plan.DisplayEnabled})
			}
		}
		appendPlans(&value.Monthly, pricing.Monthly)
		appendPlans(&value.Yearly, pricing.Yearly)
		appendPlans(&value.Topups, pricing.CreditTopups)
	}
	planLess := func(left, right publicPlanSnapshot) bool {
		if left.StripePriceID != right.StripePriceID {
			return left.StripePriceID < right.StripePriceID
		}
		if left.BundleKey != right.BundleKey {
			return left.BundleKey < right.BundleKey
		}
		if left.Billing != right.Billing {
			return left.Billing < right.Billing
		}
		return left.PlanTier < right.PlanTier
	}
	sort.Slice(value.Monthly, func(i, j int) bool { return planLess(value.Monthly[i], value.Monthly[j]) })
	sort.Slice(value.Yearly, func(i, j int) bool { return planLess(value.Yearly[i], value.Yearly[j]) })
	sort.Slice(value.Topups, func(i, j int) bool { return planLess(value.Topups[i], value.Topups[j]) })
	for _, app := range downloads {
		projection := publicAppSnapshot{AppKey: app.AppKey, Name: app.Name, Tagline: app.Tagline, Description: app.Description, WebURL: presentationWebURL(app.Metadata), Platforms: []publicAssetSnapshot{}}
		for _, asset := range app.Platforms {
			projection.Platforms = append(projection.Platforms, publicAssetSnapshot{AppKey: asset.AppKey, Platform: asset.Platform, ReleaseVersion: asset.ReleaseVersion, ReleaseNotes: asset.ReleaseNotes, Checksum: asset.Checksum, RequiresEntitlement: asset.RequiresEntitlement})
		}
		value.Downloads = append(value.Downloads, projection)
	}
	for index := range value.Downloads {
		sort.Slice(value.Downloads[index].Platforms, func(i, j int) bool {
			left, right := value.Downloads[index].Platforms[i], value.Downloads[index].Platforms[j]
			if left.Platform != right.Platform {
				return left.Platform < right.Platform
			}
			if left.ReleaseVersion != right.ReleaseVersion {
				return left.ReleaseVersion < right.ReleaseVersion
			}
			return left.Checksum < right.Checksum
		})
	}
	sort.Slice(value.Downloads, func(i, j int) bool { return value.Downloads[i].AppKey < value.Downloads[j].AppKey })
	data, _ := json.Marshal(value)
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
