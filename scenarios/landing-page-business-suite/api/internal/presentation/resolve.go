package presentation

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Resolve selects one immutable document view and deterministically projects
// it into a public-safe result. It never reads external state; commerce and
// delivery owners join their authoritative values after this pure operation.
func Resolve(document Document, request ResolveRequest) (ResolveResult, error) {
	if err := Validate(document); err != nil {
		return ResolveResult{}, err
	}

	preview, revision, err := previewSettings(request)
	if err != nil {
		return ResolveResult{}, err
	}
	if preview && !requestPreviewAuthorized(request) {
		return ResolveResult{}, ErrPreviewUnauthorized
	}

	route, routeKind, routeSlug := normalizeRoute(request.Route)
	if routeKind == "" {
		return ResolveResult{}, ErrNotFound
	}
	requestedLocale, err := requestedLocaleFor(document, request.Locale)
	if err != nil {
		return ResolveResult{}, err
	}
	requestedVariant, resolvedVariant, fallbackReason, err := resolveVariant(request)
	if err != nil {
		return ResolveResult{}, err
	}
	resolvedRevision := request.ResolvedRevision
	if resolvedRevision != "" && !idPattern.MatchString(resolvedRevision) {
		return ResolveResult{}, requestIssue("resolved_revision", "invalid_revision", "must be a safe revision identifier")
	}
	if resolvedRevision == "" {
		resolvedRevision = revision
	}

	appsByKey := map[string]App{}
	appsBySlug := map[string]App{}
	for _, app := range document.Apps {
		appsByKey[app.Key] = app
		appsBySlug[app.Slug] = app
	}
	// A root draft preview simulates public membership exactly. Authorization
	// grants inspection of a private app only on its explicit detail route;
	// it must not quietly turn every disabled profile into a bundle member.
	eligible := eligibleApps(document.Bundle.AppOrder, appsByKey, false)
	if preview && routeKind == "detail" {
		if app, ok := appsBySlug[routeSlug]; ok {
			found := false
			for _, member := range eligible {
				if member.Key == app.Key {
					found = true
				}
			}
			if !found {
				eligible = append(eligible, app)
			}
		}
	}
	eligibleKeys := appKeys(eligible)
	eligibleSet := make(map[string]bool, len(eligibleKeys))
	for _, key := range eligibleKeys {
		eligibleSet[key] = true
	}

	mode := ModeEmpty
	scope := ScopeBundle
	appKey := ""
	selected := []string{}
	pageID := document.Bundle.EmptyPageID
	if routeKind == "detail" {
		app, ok := appsBySlug[routeSlug]
		if !ok || !eligibleSet[app.Key] {
			return ResolveResult{}, ErrNotFound
		}
		mode, scope, appKey, pageID = ModeAppDetail, ScopeApp, app.Key, app.PageID
		selected = []string{app.Key}
	} else {
		switch len(eligible) {
		case 0:
			mode, scope, pageID = ModeEmpty, ScopeBundle, document.Bundle.EmptyPageID
		case 1:
			mode, scope, appKey, pageID = ModeSingleApp, ScopeApp, eligible[0].Key, eligible[0].PageID
			selected = []string{eligible[0].Key}
		default:
			mode, scope, pageID = ModeBundle, ScopeBundle, document.Bundle.PageID
			selected = capKeys(eligibleKeys, document.Bundle.MaxAppSlides)
		}
	}

	page, pageFallback, err := selectPage(document.Pages, pageID, requestedLocale, document.Bundle.DefaultLocale)
	if err != nil {
		return ResolveResult{}, err
	}
	if pageFallback != "" {
		fallbackReason = joinReasons(fallbackReason, pageFallback)
	}
	publicSlugs := make(map[string]bool, len(eligible))
	for _, app := range eligible {
		publicSlugs[app.Slug] = true
	}
	resolvedPage, err := projectPage(page, eligibleSet, selected, mode, preview, publicSlugs)
	if err != nil {
		return ResolveResult{}, err
	}
	publicFixtures, err := resolvedFixtures(document.Fixtures, resolvedPage)
	if err != nil {
		return ResolveResult{}, fmt.Errorf("resolve presentation fixtures: %w", err)
	}

	profileKeys := pageAppKeys(resolvedPage, selected)
	result := ResolveResult{
		SchemaVersion:   SchemaVersion,
		Mode:            mode,
		Scope:           scope,
		AppKey:          appKey,
		Page:            resolvedPage,
		SelectedAppKeys: copyStrings(selected),
		Spotlights:      resolvedSpotlights(profileKeys, appsByKey),
		Capabilities:    resolvedCapabilities(profileKeys, mode, appsByKey, resolvedPage.Locale),
		Assets:          resolvedAssets(document.Assets, resolvedPage, publicFixtures),
		Fixtures:        publicFixtures,
	}
	closeDisplayResources(&result.Page.Display, result.Fixtures, result.Assets)
	result.Diagnostics = Diagnostics{
		RequestedRoute:    request.Route,
		ResolvedRoute:     route,
		RequestedVariant:  requestedVariant,
		ResolvedVariant:   resolvedVariant,
		RequestedRevision: revision,
		ResolvedRevision:  resolvedRevision,
		Locale:            resolvedPage.Locale,
		BundleKey:         document.Bundle.Key,
		AppKey:            appKey,
		Mode:              mode,
		Fallback:          fallbackReason != "",
		FallbackReason:    fallbackReason,
		Preview:           preview,
		NoIndex:           preview,
		NoStore:           preview,
		EligibleAppKeys:   copyStrings(eligibleKeys),
		AssetReleaseRefs:  assetReleaseRefs(result.Assets),
	}
	result.Diagnostics.BlockDigest, err = ContentDigest(result.Page)
	if err != nil {
		return ResolveResult{}, fmt.Errorf("resolve presentation digest: %w", err)
	}
	return result, nil
}

// Resolve is intentionally also available as a small value object for owners
// that prefer dependency injection without changing this package's purity.
type Resolver struct{}

func (Resolver) Validate(document Document) error { return Validate(document) }
func (Resolver) Resolve(document Document, request ResolveRequest) (ResolveResult, error) {
	return Resolve(document, request)
}

func previewSettings(request ResolveRequest) (bool, string, error) {
	preview := request.PreviewRevision != "" || request.PreviewAuthorized
	revision := request.PreviewRevision
	if request.Preview != nil {
		if request.Preview.Revision != "" {
			revision = request.Preview.Revision
		}
		preview = preview || request.Preview.Revision != "" || request.Preview.Authorized
		if request.Preview.Authorized {
			return preview, revision, nil
		}
	}
	if revision != "" && !requestPreviewAuthorized(request) {
		return true, revision, ErrPreviewUnauthorized
	}
	if revision != "" && !idPattern.MatchString(revision) {
		return preview, revision, requestIssue("preview_revision", "invalid_revision", "must be a safe revision identifier")
	}
	return preview, revision, nil
}

func requestPreviewAuthorized(request ResolveRequest) bool {
	return request.PreviewAuthorized || request.Preview != nil && request.Preview.Authorized
}

func requestedLocaleFor(document Document, locale string) (string, error) {
	if locale == "" {
		return normalizedLocale(document.Bundle.DefaultLocale), nil
	}
	if !localePattern.MatchString(locale) {
		return "", requestIssue("locale", "invalid_locale", "must be a safe BCP-47-like locale")
	}
	return normalizedLocale(locale), nil
}

func resolveVariant(request ResolveRequest) (string, string, string, error) {
	requested := request.Variant
	resolved := request.ResolvedVariant
	if request.VariantAssignment != "" && !idPattern.MatchString(request.VariantAssignment) {
		return "", "", "", requestIssue("variant_assignment", "invalid_variant_assignment", "must be a safe identifier")
	}
	if requested != "" && !idPattern.MatchString(requested) {
		return "", "", "", requestIssue("variant", "invalid_variant", "must be a safe variant identifier")
	}
	if resolved != "" && !idPattern.MatchString(resolved) {
		return "", "", "", requestIssue("resolved_variant", "invalid_variant", "must be a safe variant identifier")
	}
	if resolved == "" {
		resolved = requested
	}
	return requested, resolved, "", nil
}

func normalizeRoute(route string) (string, string, string) {
	if route == "" {
		route = "/"
	}
	if route == "/" {
		return route, "root", ""
	}
	if !strings.HasPrefix(route, "/apps/") {
		return "", "", ""
	}
	slug := strings.TrimPrefix(route, "/apps/")
	if !slugPattern.MatchString(slug) {
		return "", "", ""
	}
	return "/apps/" + slug, "detail", slug
}

func eligibleApps(order []string, apps map[string]App, preview bool) []App {
	result := make([]App, 0, len(order))
	for _, key := range order {
		app, ok := apps[key]
		if !ok {
			continue
		}
		// Preview is an authenticated inspection of the preserved configuration,
		// including disabled/private/draft profiles. Public eligibility retains
		// the complete delivery gate.
		if !preview && !app.Enabled {
			continue
		}
		if preview || app.Visibility == VisibilityPublic && app.Publication == PublicationPublished {
			result = append(result, app)
		}
	}
	return result
}

func appKeys(apps []App) []string {
	result := make([]string, 0, len(apps))
	for _, app := range apps {
		result = append(result, app.Key)
	}
	return result
}

func copyStrings(values []string) []string {
	result := make([]string, len(values))
	copy(result, values)
	return result
}

func capKeys(keys []string, cap int) []string {
	if cap >= len(keys) {
		return append([]string(nil), keys...)
	}
	if cap <= 0 {
		return []string{}
	}
	return append([]string(nil), keys[:cap]...)
}

func selectPage(pages []Page, pageID, locale, defaultLocale string) (Page, string, error) {
	var defaultPage Page
	for _, page := range pages {
		if page.ID != pageID {
			continue
		}
		if normalizedLocale(page.Locale) == locale {
			return page, "", nil
		}
		if normalizedLocale(page.Locale) == normalizedLocale(defaultLocale) {
			defaultPage = page
		}
	}
	if defaultPage.ID == "" {
		// Never choose an arbitrary first locale. A missing configured default
		// is an unavailable join, not content that may be relabeled.
		return Page{}, "", ErrUnavailable
	}
	return defaultPage, "locale_unavailable", nil
}

func projectPage(page Page, eligible map[string]bool, selected []string, mode Mode, preview bool, publicSlugs map[string]bool) (ResolvedPage, error) {
	// JSON round-tripping a validated page is deliberate here: it gives every
	// response its own backing storage, including nested typed block slices,
	// without importing a mutable renderer model.
	page, err := clonePage(page)
	if err != nil {
		return ResolvedPage{}, fmt.Errorf("clone presentation page: %w", err)
	}
	result := ResolvedPage{ID: page.ID, Locale: normalizedLocale(page.Locale), Title: page.Title, Description: page.Description, Theme: page.Theme, Navigation: publicNavigation(page.Navigation, publicSlugs), Footer: publicFooter(page.Footer, publicSlugs)}
	result.Blocks = make([]ResolvedBlock, 0, len(page.Blocks))
	remainingSpotlights := len(selected)
	for _, block := range page.Blocks {
		content := publicContent(block.Kind, block.Content, eligible, selected, mode, preview, publicSlugs, &remainingSpotlights)
		if content == nil {
			continue
		}
		result.Blocks = append(result.Blocks, ResolvedBlock{ID: block.ID, Kind: block.Kind, Version: block.Version, Variant: block.Variant, Content: content})
	}
	result.Display = projectDisplay(page.Display, result.Blocks, pageAppKeys(result, selected), eligible, publicSlugs)
	return result, nil
}

func clonePage(page Page) (Page, error) {
	data, err := json.Marshal(page)
	if err != nil {
		return Page{}, err
	}
	var cloned Page
	if err := strictJSONUnmarshal(data, &cloned); err != nil {
		return Page{}, err
	}
	return cloned, nil
}

func publicNavigation(navigation Navigation, publicSlugs map[string]bool) Navigation {
	navigation.Items = publicLinks(navigation.Items, publicSlugs)
	return navigation
}

func publicFooter(footer Footer, publicSlugs map[string]bool) Footer {
	footer.Links = publicLinks(footer.Links, publicSlugs)
	return footer
}

func publicLinks(items []NavigationItem, publicSlugs map[string]bool) []NavigationItem {
	result := make([]NavigationItem, 0, len(items))
	for _, item := range items {
		if strings.HasPrefix(item.Target, "/apps/") {
			slug := strings.TrimPrefix(item.Target, "/apps/")
			if !publicSlugs[slug] {
				continue
			}
		}
		result = append(result, item)
	}
	return result
}

func publicContent(kind BlockKind, content BlockContent, eligible map[string]bool, selected []string, mode Mode, preview bool, publicSlugs map[string]bool, remainingSpotlights *int) BlockContent {
	switch typed := content.(type) {
	case ProductHeroContent:
		if typed.AppKey != "" && !eligible[typed.AppKey] {
			// A private app hero is a narrative leak even if its app key is
			// removed. Drop the block; validation rejects this on app-owned pages.
			return nil
		}
		typed.Actions = publicActions(typed.Actions, eligible, publicSlugs)
		return typed
	case BundleHeroContent:
		items := make([]HeroItem, 0, len(typed.HeroItems))
		for _, item := range typed.HeroItems {
			if eligible[item.AppKey] {
				items = append(items, item)
			}
		}
		typed.HeroItems = items
		typed.Actions = publicActions(typed.Actions, eligible, publicSlugs)
		return typed
	case AppSpotlightsContent:
		configured := make(map[string]bool, len(typed.AppKeys))
		for _, key := range typed.AppKeys {
			configured[key] = true
		}
		keys := make([]string, 0, len(selected))
		if remainingSpotlights != nil && *remainingSpotlights > 0 {
			for _, key := range selected {
				if *remainingSpotlights == 0 {
					break
				}
				if configured[key] {
					keys = append(keys, key)
					*remainingSpotlights--
				}
			}
		}
		typed.AppKeys = keys
		return typed
	case PricingContent:
		typed.Actions = publicActions(typed.Actions, eligible, publicSlugs)
		return typed
	case ClosingActionContent:
		typed.Actions = publicActions(typed.Actions, eligible, publicSlugs)
		return typed
	default:
		_ = kind
		_ = mode
		return content
	}
}

func publicActions(actions []Action, eligible map[string]bool, publicSlugs map[string]bool) []Action {
	result := make([]Action, 0, len(actions))
	for _, action := range actions {
		if action.AppKey != "" && !eligible[action.AppKey] {
			continue
		}
		if strings.HasPrefix(action.Target, "/apps/") {
			slug := strings.TrimPrefix(action.Target, "/apps/")
			if !publicSlugs[slug] {
				continue
			}
		}
		result = append(result, action)
	}
	return result
}

func resolvedSpotlights(keys []string, apps map[string]App) []ResolvedAppSpotlight {
	result := make([]ResolvedAppSpotlight, 0, len(keys))
	for _, key := range keys {
		app, ok := apps[key]
		if !ok {
			continue
		}
		result = append(result, ResolvedAppSpotlight{AppKey: app.Key, Slug: app.Slug, Name: app.Name, Tagline: app.Tagline, Description: app.Description, DetailRoute: "/apps/" + app.Slug})
	}
	return result
}

// App summaries also serve the hero. Hero composition is configured separately
// from the page-wide spotlight cap, so its referenced apps remain resolvable
// even when max_app_slides is zero. Hidden/unreferenced profiles stay private.
func pageAppKeys(page ResolvedPage, selected []string) []string {
	result := copyStrings(selected)
	seen := map[string]bool{}
	for _, key := range result {
		seen[key] = true
	}
	for _, block := range page.Blocks {
		if hero, ok := block.Content.(BundleHeroContent); ok {
			for _, item := range hero.HeroItems {
				if !seen[item.AppKey] {
					result = append(result, item.AppKey)
					seen[item.AppKey] = true
				}
			}
		}
	}
	return result
}

func resolvedCapabilities(keys []string, mode Mode, apps map[string]App, locale string) []ResolvedCapability {
	if mode == ModeEmpty {
		return nil
	}
	result := []ResolvedCapability{}
	for _, key := range keys {
		app, ok := apps[key]
		if !ok {
			continue
		}
		for _, capability := range app.Capabilities {
			label := capability.Label
			if value := localizedValue(capability.LocalizedLabels, locale); value != "" {
				label = value
			}
			benefits := append([]string(nil), capability.Benefits...)
			if value := localizedList(capability.LocalizedBenefits, locale); len(value) > 0 {
				benefits = value
			}
			statusLabel := capability.StatusLabel
			if value := localizedValue(capability.StatusLabels, locale); value != "" {
				statusLabel = value
			}
			result = append(result, ResolvedCapability{ID: capability.ID, Label: label, Benefits: benefits, Status: capability.Status, StatusLabel: statusLabel, Constraints: append([]string(nil), capability.Constraints...), ProviderRequirements: append([]string(nil), capability.ProviderRequirements...), PlatformRequirements: append([]string(nil), capability.PlatformRequirements...)})
		}
	}
	return result
}

func localizedList(values map[string][]string, locale string) []string {
	if value := values[locale]; len(value) > 0 {
		return copyStrings(value)
	}
	language := strings.Split(locale, "-")[0]
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if strings.Split(normalizedLocale(key), "-")[0] == language && len(values[key]) > 0 {
			return copyStrings(values[key])
		}
	}
	return nil
}

func localizedValue(values map[string]string, locale string) string {
	if value := values[locale]; value != "" {
		return value
	}
	language := strings.Split(locale, "-")[0]
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := values[key]
		if strings.Split(normalizedLocale(key), "-")[0] == language {
			return value
		}
	}
	return ""
}

func resolvedAssets(assets []Asset, page ResolvedPage, fixtures []Fixture) []ResolvedAsset {
	refs := map[string]bool{}
	for _, app := range page.Display.Apps {
		if app.VisualRef != "" {
			refs[app.VisualRef] = true
		}
	}
	for _, block := range page.Blocks {
		for _, ref := range assetRefs(block.Content) {
			refs[ref] = true
		}
	}
	for _, fixture := range fixtures {
		if fixture.Backdrop != nil {
			for _, ref := range fixture.Backdrop.AssetRefs {
				if ref != "" {
					refs[ref] = true
				}
			}
		}
	}
	// Responsive alternatives are part of the selected public resource's
	// closure, so Astra never receives an asset ID without its corresponding
	// released metadata.
	for changed := true; changed; {
		changed = false
		for _, asset := range assets {
			if !refs[asset.ID] {
				continue
			}
			for _, alternative := range asset.ResponsiveAlternatives {
				if alternative.AssetID != "" && !refs[alternative.AssetID] {
					refs[alternative.AssetID] = true
					changed = true
				}
			}
		}
	}
	result := []ResolvedAsset{}
	for _, asset := range assets {
		if refs[asset.ID] {
			regions := make([]ResolvedOverlayRegion, 0, len(asset.OverlayRegions))
			for _, region := range asset.OverlayRegions {
				regions = append(regions, ResolvedOverlayRegion{
					Name: region.Name, X: region.X, Y: region.Y, Width: region.Width, Height: region.Height,
					Measurement: ResolvedLegibilityMeasurement{
						ContrastRatio:        region.Measurement.ContrastRatio,
						MinimumContrastRatio: region.Measurement.MinimumContrastRatio,
						Threshold:            region.Measurement.Threshold,
						Verdict:              region.Measurement.Verdict,
					},
				})
			}
			alternatives := make([]AssetVariant, len(asset.ResponsiveAlternatives))
			copy(alternatives, asset.ResponsiveAlternatives)
			result = append(result, ResolvedAsset{ID: asset.ID, ReleaseRef: asset.ReleaseRef, ContentHash: asset.ContentHash, Width: asset.Width, Height: asset.Height, MIME: asset.MIME, Surface: asset.Surface, ResponsiveAlternatives: alternatives, FocalPoint: asset.FocalPoint, CropPolicy: asset.CropPolicy, Provenance: ResolvedAssetProvenance{Provider: asset.Provenance.Provider}, OverlayRegions: regions, PublicURL: asset.PublicURL})
		}
	}
	return result
}

func resolvedFixtures(fixtures []Fixture, page ResolvedPage) ([]Fixture, error) {
	refs := displayFixtureRefs(page.Display)
	for _, block := range page.Blocks {
		if ref := fixtureRef(block.Content); ref != "" {
			refs[ref] = true
		}
	}
	result := []Fixture{}
	for _, fixture := range fixtures {
		if refs[fixture.ID] {
			cloned, err := cloneFixture(fixture)
			if err != nil {
				return nil, err
			}
			result = append(result, cloned)
		}
	}
	return result, nil
}

func cloneFixture(fixture Fixture) (Fixture, error) {
	data, err := json.Marshal(fixture)
	if err != nil {
		return Fixture{}, err
	}
	var cloned Fixture
	if err := strictJSONUnmarshal(data, &cloned); err != nil {
		return Fixture{}, err
	}
	return cloned, nil
}

func fixtureRef(content BlockContent) string {
	switch typed := content.(type) {
	case ProductHeroContent:
		return typed.FixtureRef
	case ProductDemoContent:
		return typed.FixtureRef
	}
	return ""
}

func assetRefs(content BlockContent) []string {
	switch typed := content.(type) {
	case ProductHeroContent:
		return nonEmpty(typed.VisualRef)
	case BundleHeroContent:
		result := []string{}
		for _, item := range typed.HeroItems {
			result = append(result, item.VisualRef)
		}
		return result
	case ProductDemoContent:
		return []string{typed.PosterRef, typed.MediaRef}
	case ProductStoryContent:
		result := []string{}
		for _, item := range typed.Items {
			result = append(result, item.VisualRef)
		}
		return result
	case ArtifactExplorerContent:
		result := []string{}
		for _, item := range typed.Examples {
			result = append(result, item.AssetRef, item.PreviewRef)
			for _, frame := range item.Frames {
				result = append(result, frame.AssetRef)
			}
		}
		return result
	case DeviceStoryContent:
		return nonEmpty(typed.VisualRef)
	case ClosingActionContent:
		return nonEmpty(typed.VisualRef)
	}
	return nil
}

func nonEmpty(values ...string) []string {
	result := []string{}
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func assetReleaseRefs(assets []ResolvedAsset) []string {
	result := []string{}
	seen := map[string]bool{}
	for _, asset := range assets {
		if !seen[asset.ReleaseRef] {
			seen[asset.ReleaseRef] = true
			result = append(result, asset.ReleaseRef)
		}
	}
	return result
}

type requestValidationError struct{ Field, Code, Message string }

func (e requestValidationError) Error() string {
	return fmt.Sprintf("invalid presentation request %s: %s", e.Field, e.Message)
}

func requestIssue(field, code, message string) error {
	return requestValidationError{Field: field, Code: code, Message: message}
}

func joinReasons(values ...string) string {
	result := []string{}
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return strings.Join(result, ",")
}
