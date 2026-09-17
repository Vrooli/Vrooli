package presentation

import (
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
)

// Validate checks the complete document, including cross-object references.
// It does not normalize or mutate the input. Callers should refuse to publish
// a document when this returns an error.
func Validate(document Document) error {
	issues := &ValidationError{}
	if document.SchemaVersion != SchemaVersion {
		issues.add("schema_version", "unsupported_schema_version", fmt.Sprintf("must be %d", SchemaVersion))
	}
	validateBundle(document.Bundle, issues)

	appsByKey := make(map[string]App, len(document.Apps))
	appsBySlug := make(map[string]string, len(document.Apps))
	appPageOwners := make(map[string][]string)
	capabilities := make(map[string]Capability)
	for i, app := range document.Apps {
		path := fmt.Sprintf("apps[%d]", i)
		validateApp(app, path, issues)
		if _, exists := appsByKey[app.Key]; exists {
			issues.add(path+".key", "duplicate_app_key", "app keys must be unique")
		}
		if previous, exists := appsBySlug[app.Slug]; exists {
			issues.add(path+".slug", "duplicate_slug", fmt.Sprintf("slug is already used by %q", previous))
		}
		appsByKey[app.Key] = app
		appsBySlug[app.Slug] = app.Key
		appPageOwners[app.PageID] = append(appPageOwners[app.PageID], app.Key)
		for j, capability := range app.Capabilities {
			capPath := fmt.Sprintf("%s.capabilities[%d]", path, j)
			validateCapability(capability, capPath, issues)
			if _, exists := capabilities[capability.ID]; exists {
				issues.add(capPath+".id", "duplicate_capability_id", "capability IDs must be globally unique")
			}
			capabilities[capability.ID] = capability
		}
	}

	assets := make(map[string]Asset, len(document.Assets))
	for i, asset := range document.Assets {
		path := fmt.Sprintf("assets[%d]", i)
		validateAsset(asset, path, issues)
		if _, exists := assets[asset.ID]; exists {
			issues.add(path+".id", "duplicate_asset_id", "asset IDs must be unique")
		}
		assets[asset.ID] = asset
	}
	for i, asset := range document.Assets {
		for j, alternative := range asset.ResponsiveAlternatives {
			if _, exists := assets[alternative.AssetID]; !exists {
				issues.add(fmt.Sprintf("assets[%d].responsive_alternatives[%d].asset_id", i, j), "unknown_asset_ref", "responsive alternative references an undeclared asset")
			}
		}
	}
	fixtures := make(map[string]Fixture, len(document.Fixtures))
	for i, fixture := range document.Fixtures {
		path := fmt.Sprintf("fixtures[%d]", i)
		validateFixture(fixture, path, issues)
		if _, exists := fixtures[fixture.ID]; exists {
			issues.add(path+".id", "duplicate_fixture_id", "fixture IDs must be unique")
		}
		fixtures[fixture.ID] = fixture
		if fixture.Backdrop != nil {
			for j, ref := range fixture.Backdrop.AssetRefs {
				if _, exists := assets[ref]; !exists {
					issues.add(fmt.Sprintf("%s.backdrop.asset_refs[%d]", path, j), "unknown_asset_ref", "asset reference is not declared")
				}
			}
		}
	}

	pageKeys := make(map[string]bool, len(document.Pages))
	pageByID := make(map[string][]Page)
	for i, page := range document.Pages {
		path := fmt.Sprintf("pages[%d]", i)
		validatePage(page, path, issues)
		key := page.ID + "\x00" + normalizedLocale(page.Locale)
		if pageKeys[key] {
			issues.add(path, "duplicate_page", "a page ID and locale pair must be unique")
		}
		pageKeys[key] = true
		pageByID[page.ID] = append(pageByID[page.ID], page)
		pageCapabilities := capabilities
		if owners := appPageOwners[page.ID]; len(owners) == 1 {
			pageCapabilities = make(map[string]Capability)
			for _, capability := range appsByKey[owners[0]].Capabilities {
				pageCapabilities[capability.ID] = capability
			}
		}
		validatePageReferences(page, path, appsByKey, pageCapabilities, assets, fixtures, issues)
		validateDisplay(page, path+".display", appsByKey, pageCapabilities, assets, fixtures, issues)
	}
	for pageID, pages := range pageByID {
		configuredDefault := false
		for _, page := range pages {
			if normalizedLocale(page.Locale) == normalizedLocale(document.Bundle.DefaultLocale) {
				configuredDefault = true
				break
			}
		}
		if !configuredDefault {
			issues.add("pages", "missing_default_locale_page", fmt.Sprintf("page %q must define the configured bundle default locale", pageID))
		}
	}
	for pageID, owners := range appPageOwners {
		if len(owners) != 1 {
			issues.add("pages["+pageID+"]", "ambiguous_page_owner", "each app requires its own page identity; localized pages share only that app's identity")
			continue
		}
		for _, page := range pageByID[pageID] {
			hasHero, hasClosing := false, false
			for blockIndex, block := range page.Blocks {
				switch block.Kind {
				case BlockProductHero:
					hasHero = true
				case BlockClosingAction:
					hasClosing = true
				}
				if block.Kind != BlockProductHero {
					continue
				}
				content, err := contentMap(block.Content)
				if err != nil {
					continue
				}
				appKey := stringValue(content["app_key"])
				path := fmt.Sprintf("pages[%s].blocks[%d].content.app_key", pageID, blockIndex)
				if appKey == "" {
					issues.add(path, "missing_product_hero_app_ref", "an app page product hero must identify its owning app")
				} else if appKey != owners[0] {
					issues.add(path, "wrong_product_hero_app_ref", "an app page product hero must identify its owning app")
				}
			}
			pagePath := "pages[" + pageID + "]"
			if !hasHero {
				issues.add(pagePath+".blocks", "missing_product_hero", "an app page requires a configured product hero")
			}
			if !hasClosing {
				issues.add(pagePath+".blocks", "missing_closing_action", "an app page requires a configured closing action")
			}
		}
	}

	for i, app := range document.Apps {
		if !hasPage(pageByID, app.PageID) {
			issues.add(fmt.Sprintf("apps[%d].page_id", i), "unknown_page_ref", "app page_id does not identify a page")
		}
	}
	if !hasPage(pageByID, document.Bundle.PageID) {
		issues.add("bundle.page_id", "unknown_page_ref", "bundle page_id does not identify a page")
	}
	if !hasPage(pageByID, document.Bundle.EmptyPageID) {
		issues.add("bundle.empty_page_id", "unknown_page_ref", "empty_page_id does not identify a page")
	}
	locales := sortedStringKeys(document.Strings)
	for _, locale := range locales {
		values := document.Strings[locale]
		validateLocale(locale, "strings["+locale+"]", issues)
		for _, key := range sortedStringKeys(values) {
			value := values[key]
			validateID(key, "strings["+locale+"]."+key, issues)
			validateText(value, "strings["+locale+"]."+key, issues, true)
		}
	}
	for i, key := range document.Bundle.AppOrder {
		if _, exists := appsByKey[key]; !exists {
			issues.add(fmt.Sprintf("bundle.app_order[%d]", i), "unknown_app_ref", "app_order references an undeclared app")
		}
	}
	if len(issues.Issues) != 0 {
		return issues
	}
	return nil
}

func validateBundle(bundle Bundle, issues *ValidationError) {
	validateID(bundle.Key, "bundle.key", issues)
	validateText(bundle.Name, "bundle.name", issues, true)
	if bundle.MaxAppSlides < 0 {
		issues.add("bundle.max_app_slides", "invalid_cap", "must be zero or greater")
	}
	validateID(bundle.PageID, "bundle.page_id", issues)
	validateID(bundle.EmptyPageID, "bundle.empty_page_id", issues)
	validateLocale(bundle.DefaultLocale, "bundle.default_locale", issues)
	if len(bundle.Locales) == 0 {
		issues.add("bundle.locales", "missing_locales", "at least one locale is required")
	}
	seenLocales := map[string]bool{}
	for i, locale := range bundle.Locales {
		path := fmt.Sprintf("bundle.locales[%d]", i)
		validateLocale(locale, path, issues)
		normalized := normalizedLocale(locale)
		if seenLocales[normalized] {
			issues.add(path, "duplicate_locale", "locales must be unique")
		}
		seenLocales[normalized] = true
	}
	if !seenLocales[normalizedLocale(bundle.DefaultLocale)] && len(bundle.Locales) > 0 {
		issues.add("bundle.default_locale", "locale_not_declared", "default_locale must be listed in locales")
	}
	seenApps := map[string]bool{}
	for i, key := range bundle.AppOrder {
		path := fmt.Sprintf("bundle.app_order[%d]", i)
		validateID(key, path, issues)
		if seenApps[key] {
			issues.add(path, "duplicate_app_order", "app_order must not contain duplicates")
		}
		seenApps[key] = true
	}
}

func validateApp(app App, path string, issues *ValidationError) {
	validateID(app.Key, path+".key", issues)
	validateSlug(app.Slug, path+".slug", issues)
	validateText(app.Name, path+".name", issues, true)
	validateText(app.Tagline, path+".tagline", issues, true)
	validateText(app.Description, path+".description", issues, true)
	validateID(app.PageID, path+".page_id", issues)
	if app.Visibility != VisibilityPublic && app.Visibility != VisibilityPrivate {
		issues.add(path+".visibility", "invalid_visibility", "must be public or private")
	}
	if app.Publication != PublicationDraft && app.Publication != PublicationPublished {
		issues.add(path+".publication", "invalid_publication", "must be draft or published")
	}
	if app.PreservationRef != "" {
		validateOpaqueRef(app.PreservationRef, path+".preservation_ref", issues)
	}
}

func validateCapability(capability Capability, path string, issues *ValidationError) {
	validateID(capability.ID, path+".id", issues)
	validateText(capability.Label, path+".label", issues, true)
	if len(capability.Benefits) == 0 {
		issues.add(path+".benefits", "missing_benefits", "at least one benefit is required")
	}
	for i, benefit := range capability.Benefits {
		validateText(benefit, fmt.Sprintf("%s.benefits[%d]", path, i), issues, true)
	}
	if capability.Status != CapabilityAvailable && capability.Status != CapabilityPreview && capability.Status != CapabilityComingSoon {
		issues.add(path+".status", "invalid_capability_status", "must be available, preview, or coming-soon")
	}
	if capability.Status == CapabilityAvailable && len(capability.EvidenceRefs) == 0 {
		issues.add(path+".evidence_refs", "missing_evidence", "available capabilities require private owner evidence")
	}
	if capability.Status == CapabilityAvailable && (capability.OwnerQualification == nil || !capability.OwnerQualification.Qualified) {
		issues.add(path+".owner_qualification", "unverified_owner_qualification", "available claims require an explicit qualified release-owner attestation; an evidence reference alone is insufficient")
	}
	if capability.OwnerQualification != nil {
		qualificationPath := path + ".owner_qualification"
		validateOpaqueRef(capability.OwnerQualification.Owner, qualificationPath+".owner", issues)
		validateOpaqueRef(capability.OwnerQualification.EvidenceRef, qualificationPath+".evidence_ref", issues)
		validateOpaqueRef(capability.OwnerQualification.ReleaseRef, qualificationPath+".release_ref", issues)
	}
	validateRefList(capability.EvidenceRefs, path+".evidence_refs", issues, true)
	validateTextList(capability.Constraints, path+".constraints", issues)
	validateTextList(capability.ProviderRequirements, path+".provider_requirements", issues)
	validateTextList(capability.PlatformRequirements, path+".platform_requirements", issues)
	for _, locale := range sortedStringKeys(capability.LocalizedLabels) {
		label := capability.LocalizedLabels[locale]
		validateLocale(locale, path+".localized_labels["+locale+"]", issues)
		validateText(label, path+".localized_labels["+locale+"]", issues, true)
	}
	for _, locale := range sortedStringKeys(capability.LocalizedBenefits) {
		benefits := capability.LocalizedBenefits[locale]
		validateLocale(locale, path+".localized_benefits["+locale+"]", issues)
		if len(benefits) == 0 {
			issues.add(path+".localized_benefits["+locale+"]", "missing_benefits", "localized benefits must not be empty")
		}
		validateTextList(benefits, path+".localized_benefits["+locale+"]", issues)
	}
	if capability.StatusLabel != "" {
		validateText(capability.StatusLabel, path+".status_label", issues, true)
	}
	for _, locale := range sortedStringKeys(capability.StatusLabels) {
		label := capability.StatusLabels[locale]
		validateLocale(locale, path+".status_labels["+locale+"]", issues)
		validateText(label, path+".status_labels["+locale+"]", issues, true)
	}
}

func validateAsset(asset Asset, path string, issues *ValidationError) {
	validateID(asset.ID, path+".id", issues)
	validateOpaqueRef(asset.ReleaseRef, path+".release_ref", issues)
	if !assetHashPattern.MatchString(asset.ContentHash) {
		issues.add(path+".content_hash", "invalid_content_hash", "must be exactly 64 lowercase hexadecimal SHA-256 digits")
	}
	if asset.Width <= 0 || asset.Height <= 0 {
		issues.add(path, "invalid_dimensions", "width and height must be positive")
	}
	if strings.TrimSpace(asset.MIME) == "" || strings.ContainsAny(asset.MIME, "\r\n") || !strings.Contains(asset.MIME, "/") {
		issues.add(path+".mime", "invalid_mime", "must be a MIME type")
	}
	validateID(asset.Surface, path+".surface", issues)
	if asset.CropPolicy == "" {
		issues.add(path+".crop_policy", "missing_crop_policy", "crop_policy is required")
	}
	validateOpaqueRef(asset.Provenance.Provider, path+".provenance.provider", issues)
	validateOpaqueRef(asset.Provenance.JobRef, path+".provenance.job_ref", issues)
	validateOpaqueRef(asset.Provenance.CandidateRef, path+".provenance.candidate_ref", issues)
	if math.IsNaN(asset.FocalPoint.X) || math.IsInf(asset.FocalPoint.X, 0) || math.IsNaN(asset.FocalPoint.Y) || math.IsInf(asset.FocalPoint.Y, 0) || asset.FocalPoint.X < 0 || asset.FocalPoint.X > 1 || asset.FocalPoint.Y < 0 || asset.FocalPoint.Y > 1 {
		issues.add(path+".focal_point", "invalid_focal_point", "x and y must be between 0 and 1")
	}
	for i, region := range asset.OverlayRegions {
		regionPath := fmt.Sprintf("%s.overlay_regions[%d]", path, i)
		validateText(region.Name, regionPath+".name", issues, true)
		if !validUnitRect(region.X, region.Y, region.Width, region.Height) {
			issues.add(regionPath, "invalid_overlay_region", "overlay region must be a positive rectangle within the unit square")
		}
		validateLegibilityMeasurement(region.Measurement, regionPath+".measurement", issues)
	}
	for i, alternative := range asset.ResponsiveAlternatives {
		altPath := fmt.Sprintf("%s.responsive_alternatives[%d]", path, i)
		validateID(alternative.Surface, altPath+".surface", issues)
		validateID(alternative.AssetID, altPath+".asset_id", issues)
	}
	validateRefList(asset.PrivateEvidenceRefs, path+".private_evidence_refs", issues, true)
	if asset.PublicURL != "" {
		if !isSafePublicURL(asset.PublicURL) {
			issues.add(path+".public_url", "unsafe_asset_url", "asset URLs must be same-origin relative paths")
		}
	}
}

func validateLegibilityMeasurement(measurement LegibilityMeasurement, path string, issues *ValidationError) {
	if measurement.ContrastRatio <= 0 || math.IsInf(measurement.ContrastRatio, 0) || math.IsNaN(measurement.ContrastRatio) {
		issues.add(path+".contrast_ratio", "invalid_measurement", "contrast_ratio must be finite and positive")
	}
	if measurement.MinimumContrastRatio <= 0 || math.IsInf(measurement.MinimumContrastRatio, 0) || math.IsNaN(measurement.MinimumContrastRatio) {
		issues.add(path+".minimum_contrast_ratio", "invalid_measurement", "minimum_contrast_ratio must be finite and positive")
	}
	if measurement.Threshold <= 0 || math.IsInf(measurement.Threshold, 0) || math.IsNaN(measurement.Threshold) {
		issues.add(path+".threshold", "invalid_measurement", "threshold must be finite and positive")
	}
	if measurement.Verdict != LegibilityPass && measurement.Verdict != LegibilityFail && measurement.Verdict != LegibilityNotMeasured {
		issues.add(path+".verdict", "invalid_measurement_verdict", "must be pass, fail, or not_measured")
	}
	validateOpaqueRef(measurement.MeasurementRef, path+".measurement_ref", issues)
	if measurement.Verdict == LegibilityPass && measurement.ContrastRatio < measurement.MinimumContrastRatio {
		issues.add(path, "inconsistent_measurement", "a passing measurement must meet its minimum contrast ratio")
	}
}

func validateFixture(fixture Fixture, path string, issues *ValidationError) {
	validateID(fixture.ID, path+".id", issues)
	if fixture.Kind != FixtureWorkspace && fixture.Kind != FixtureBackdrop && fixture.Kind != FixtureWorkflow && fixture.Kind != FixtureMonitor {
		issues.add(path+".kind", "unsupported_fixture_kind", "fixture kind is not registered")
		return
	}
	if fixture.Kind == FixtureWorkspace && fixture.Workspace == nil || fixture.Kind == FixtureBackdrop && fixture.Backdrop == nil || fixture.Kind == FixtureWorkflow && fixture.Workflow == nil || fixture.Kind == FixtureMonitor && fixture.Monitor == nil {
		issues.add(path, "missing_fixture_data", "the selected fixture kind requires its typed data")
	}
	if fixture.Workspace != nil {
		validateWorkspaceFixture(*fixture.Workspace, path+".workspace", issues)
	}
	if fixture.Backdrop != nil {
		validateBackdropFixture(*fixture.Backdrop, path+".backdrop", issues)
	}
	if fixture.Workflow != nil {
		validateWorkflowFixture(*fixture.Workflow, path+".workflow", issues)
	}
	if fixture.Monitor != nil {
		validateMonitorFixture(*fixture.Monitor, path+".monitor", issues)
	}
}

func validateMonitorFixture(fixture MonitorFixture, path string, issues *ValidationError) {
	for key, value := range map[string]string{"title": fixture.Title, "host": fixture.Host, "metrics_label": fixture.MetricsLabel, "investigation_label": fixture.InvestigationLabel, "investigation_title": fixture.InvestigationTitle, "investigation_body": fixture.InvestigationBody, "action_note": fixture.ActionNote, "status": fixture.Status, "uptime": fixture.Uptime} {
		validateText(value, path+"."+key, issues, true)
	}
	if len(fixture.Metrics) == 0 {
		issues.add(path+".metrics", "empty_fixture_data", "must contain at least one metric")
	}
	for i, metric := range fixture.Metrics {
		p := fmt.Sprintf("%s.metrics[%d]", path, i)
		validateID(metric.ID, p+".id", issues)
		for key, value := range map[string]string{"label": metric.Label, "value": metric.Value, "status": metric.Status} {
			validateText(value, p+"."+key, issues, true)
		}
		for key, value := range map[string]string{"unit": metric.Unit, "detail": metric.Detail} {
			validateText(value, p+"."+key, issues, false)
		}
		if len(metric.Trend) == 0 {
			issues.add(p+".trend", "empty_fixture_data", "must contain at least one trend sample")
		}
		for j, sample := range metric.Trend {
			if math.IsNaN(sample) || math.IsInf(sample, 0) || sample < 0 || sample > 1 {
				issues.add(fmt.Sprintf("%s.trend[%d]", p, j), "invalid_fixture_data", "trend samples must be normalized between 0 and 1")
			}
		}
	}
	if len(fixture.Findings) == 0 {
		issues.add(path+".findings", "empty_fixture_data", "must contain at least one finding")
	}
	validateTextList(fixture.Findings, path+".findings", issues)
}

func validateWorkspaceFixture(fixture WorkspaceFixture, path string, issues *ValidationError) {
	values := map[string]string{"title": fixture.Title, "group": fixture.Group, "group_label": fixture.GroupLabel, "sessions_label": fixture.SessionsLabel, "role": fixture.Role, "model": fixture.Model, "reviewer": fixture.Reviewer, "reviewer_model": fixture.ReviewerModel, "branch": fixture.Branch, "prompt": fixture.Prompt, "answer": fixture.Answer, "file_label": fixture.FileLabel, "command": fixture.Command, "ready": fixture.Ready, "composer": fixture.Composer, "return_label": fixture.ReturnLabel, "return_title": fixture.ReturnTitle, "review_message": fixture.ReviewMessage, "message_label": fixture.MessageLabel, "reply_label": fixture.ReplyLabel, "status": fixture.Status, "today": fixture.Today}
	for key, value := range values {
		validateText(value, path+"."+key, issues, true)
	}
	for key, values := range map[string][]string{"groups": fixture.Groups, "sessions": fixture.Sessions, "files": fixture.Files, "diff": fixture.Diff, "checks": fixture.Checks, "keyboard": fixture.Keyboard} {
		if len(values) == 0 {
			issues.add(path+"."+key, "empty_fixture_data", "must contain at least one item")
		}
		validateTextList(values, path+"."+key, issues)
	}
}

func validateBackdropFixture(fixture BackdropFixture, path string, issues *ValidationError) {
	for key, value := range map[string]string{"title": fixture.Title, "label": fixture.Label, "selected": fixture.Selected, "surface": fixture.Surface, "palette": fixture.Palette, "panel": fixture.Panel, "caption": fixture.Caption, "export": fixture.Export, "badge": fixture.Badge} {
		validateText(value, path+"."+key, issues, true)
	}
	for key, values := range map[string][]string{"styles": fixture.Styles, "options": fixture.Options, "asset_refs": fixture.AssetRefs} {
		if len(values) == 0 {
			issues.add(path+"."+key, "empty_fixture_data", "must contain at least one item")
		}
		validateTextList(values, path+"."+key, issues)
	}
	if len(fixture.Styles) != len(fixture.AssetRefs) {
		issues.add(path, "backdrop_style_asset_mismatch", "styles and asset_refs must contain the same number of entries in positional order")
	}
	selected := strings.TrimSpace(fixture.Selected)
	if selected != "" {
		found := false
		for _, style := range fixture.Styles {
			if style == fixture.Selected {
				found = true
				break
			}
		}
		if !found {
			issues.add(path+".selected", "backdrop_selected_style_not_found", "selected must exactly match one of styles")
		}
	}
}

func validateWorkflowFixture(fixture WorkflowFixture, path string, issues *ValidationError) {
	validateText(fixture.Title, path+".title", issues, true)
	validateText(fixture.Label, path+".label", issues, true)
	validateText(fixture.BrowserTitle, path+".browser_title", issues, true)
	validateText(fixture.Note, path+".note", issues, true)
	if len(fixture.Steps) == 0 {
		issues.add(path+".steps", "empty_fixture_data", "must contain at least one step")
	}
	for i, step := range fixture.Steps {
		p := fmt.Sprintf("%s.steps[%d]", path, i)
		validateText(step.Number, p+".number", issues, true)
		validateText(step.Title, p+".title", issues, true)
		validateText(step.Description, p+".description", issues, true)
	}
	validateTextList(fixture.BrowserRows, path+".browser_rows", issues)
}

func validatePage(page Page, path string, issues *ValidationError) {
	validateID(page.ID, path+".id", issues)
	validateLocale(page.Locale, path+".locale", issues)
	// A zero-app page is allowed to have zero blocks, but it is still a real
	// configured page. Its shell must be complete so it cannot silently fall
	// back to an arbitrary renderer or locale.
	validateText(page.Title, path+".title", issues, true)
	validateText(page.Description, path+".description", issues, true)
	validateTheme(page.Theme, path+".theme", issues)
	validateNavigation(page.Navigation, path+".navigation", issues)
	validateFooter(page.Footer, path+".footer", issues)
	seen := map[string]bool{}
	spotlightCollections := 0
	for i, block := range page.Blocks {
		path := fmt.Sprintf("%s.blocks[%d]", path, i)
		validateBlock(block, path, issues)
		if seen[block.ID] {
			issues.add(path+".id", "duplicate_block_id", "block IDs must be unique within a page")
		}
		seen[block.ID] = true
		if block.Kind == BlockAppSpotlights {
			spotlightCollections++
			if spotlightCollections > 1 {
				issues.add(fmt.Sprintf("%s.blocks[%d]", path, i), "duplicate_spotlight_collection", "a page may contain only one app spotlight collection")
			}
		}
	}
}

func validateTheme(theme Theme, path string, issues *ValidationError) {
	validTheme := map[string]bool{"signal": true, "studio": true}
	if !validTheme[theme.Variant] {
		issues.add(path+".variant", "invalid_theme_variant", "must be signal or studio")
	}
	for name, value := range map[string]string{"primary": theme.Primary, "background": theme.Background, "accent": theme.Accent} {
		if !colorPattern.MatchString(value) {
			issues.add(path+"."+name, "invalid_theme_token", "must be a six-digit hexadecimal color")
		}
	}
}

func validateNavigation(navigation Navigation, path string, issues *ValidationError) {
	validateText(navigation.Label, path+".label", issues, true)
	for i, item := range navigation.Items {
		validateNavigationItem(item, fmt.Sprintf("%s.items[%d]", path, i), issues)
	}
}

func validateFooter(footer Footer, path string, issues *ValidationError) {
	validateText(footer.Label, path+".label", issues, true)
	for i, item := range footer.Links {
		validateNavigationItem(item, fmt.Sprintf("%s.links[%d]", path, i), issues)
	}
}

func validateNavigationItem(item NavigationItem, path string, issues *ValidationError) {
	validateText(item.Label, path+".label", issues, true)
	validateText(item.AccessibleLabel, path+".accessible_label", issues, true)
	if !isSafeTarget(item.Target) {
		issues.add(path+".target", "unsafe_target", "target must be a safe relative route or anchor")
	}
}

func validateBlock(block Block, path string, issues *ValidationError) {
	validateID(block.ID, path+".id", issues)
	if !validKinds[block.Kind] {
		issues.add(path+".kind", "unsupported_block_kind", "kind is not registered for schema version 1")
		return
	}
	if block.Version != SchemaVersion {
		issues.add(path+".version", "unsupported_block_version", "must be 1")
	}
	if !validVariants[block.Kind][block.Variant] {
		issues.add(path+".variant", "unsupported_block_variant", "variant is not registered for this block kind")
	}
	content, err := contentMap(block.Content)
	if err != nil {
		issues.add(path+".content", "invalid_content", err.Error())
		return
	}
	validateContent(block.Kind, block.Variant, content, path+".content", issues)
}

var contentFields = map[BlockKind]map[string]bool{
	BlockProductHero:       {"app_key": true, "eyebrow": true, "title": true, "description": true, "visual_ref": true, "fixture_ref": true, "accessibility_label": true, "actions": true},
	BlockBundleHero:        {"eyebrow": true, "title": true, "description": true, "accessibility_label": true, "hero_items": true, "actions": true},
	BlockCapabilityStrip:   {"heading": true, "items": true},
	BlockProductStory:      {"heading": true, "body": true, "items": true},
	BlockProductDemo:       {"heading": true, "description": true, "renderer_ref": true, "fixture_ref": true, "poster_ref": true, "media_ref": true, "alt_text": true, "playback": true},
	BlockAppSpotlights:     {"heading": true, "app_keys": true, "detail_link_label": true},
	BlockArtifactExplorer:  {"heading": true, "capability_id": true, "examples": true, "selected_example_id": true},
	BlockVoiceStory:        {"eyebrow": true, "heading": true, "body": true, "features": true, "note": true, "input_label": true, "transcript": true, "summary_label": true, "summary_title": true, "summary_items": true, "output_label": true, "demo_note": true, "provider_qualification": true, "waveform": true, "capability_ids": true},
	BlockDeviceStory:       {"heading": true, "description": true, "device": true, "visual_ref": true, "alt_text": true},
	BlockCapabilityRoadmap: {"heading": true, "capability_ids": true, "status_label": true, "description": true},
	BlockPricing:           {"heading": true, "description": true, "plan_refs": true, "actions": true},
	BlockClosingAction:     {"heading": true, "description": true, "actions": true, "visual_ref": true},
	BlockFAQ:               {"heading": true, "items": true},
	BlockFooter:            {"label": true, "links": true},
}

var requiredContentFields = map[BlockKind][]string{
	BlockProductHero:     {"title", "description", "accessibility_label", "actions"},
	BlockBundleHero:      {"title", "description", "accessibility_label", "hero_items", "actions"},
	BlockCapabilityStrip: {"heading", "items"}, BlockProductStory: {"heading", "body", "items"},
	BlockProductDemo:       {"heading", "description", "renderer_ref", "alt_text"},
	BlockAppSpotlights:     {"heading", "app_keys", "detail_link_label"},
	BlockArtifactExplorer:  {"heading", "capability_id", "examples", "selected_example_id"},
	BlockVoiceStory:        {"heading", "body", "features", "note", "input_label", "transcript", "summary_label", "summary_title", "summary_items", "output_label", "demo_note", "provider_qualification", "waveform", "capability_ids"},
	BlockDeviceStory:       {"heading", "description", "device", "alt_text"},
	BlockCapabilityRoadmap: {"heading", "capability_ids", "status_label", "description"},
	BlockPricing:           {"heading", "description", "plan_refs", "actions"},
	BlockClosingAction:     {"heading", "description", "actions"}, BlockFAQ: {"heading", "items"}, BlockFooter: {"label", "links"},
}

func validateContent(kind BlockKind, variant string, content map[string]any, path string, issues *ValidationError) {
	allowed := contentFields[kind]
	for _, key := range sortedStringKeys(content) {
		if unsafeContentKey(key) {
			issues.add(path+"."+key, "unsafe_content", "executable markup and scripts are not valid presentation content")
			continue
		}
		if !allowed[key] {
			issues.add(path+"."+key, "unknown_content_field", "field is not part of the registered block contract")
		}
	}
	for _, key := range requiredContentFields[kind] {
		value, exists := content[key]
		if !exists || value == nil || (isString(value) && strings.TrimSpace(value.(string)) == "") {
			issues.add(path+"."+key, "missing_content_field", "field is required")
		}
	}
	for _, key := range sortedStringKeys(content) {
		value := content[key]
		if isString(value) {
			validateText(value.(string), path+"."+key, issues, false)
		}
	}
	validateContentShapes(kind, variant, content, path, issues)
}

func validateContentShapes(kind BlockKind, variant string, content map[string]any, path string, issues *ValidationError) {
	validateActions(content["actions"], path+".actions", issues)
	validateAssetRefs(content, path, issues)
	switch kind {
	case BlockProductHero:
		validateOptionalID(content["app_key"], path+".app_key", issues)
		validateOptionalID(content["fixture_ref"], path+".fixture_ref", issues)
		if strings.TrimSpace(stringValue(content["visual_ref"])) == "" && strings.TrimSpace(stringValue(content["fixture_ref"])) == "" {
			issues.add(path, "missing_visual_source", "a product hero requires a released visual_ref or an illustrative fixture_ref")
		}
	case BlockProductDemo:
		validateIDValue(content["renderer_ref"], path+".renderer_ref", issues)
		validateProductDemo(variant, content, path, issues)
	case BlockBundleHero:
		if len(arrayValue(content["hero_items"])) > 3 {
			issues.add(path+".hero_items", "hero_capacity_exceeded", "version 1 hero compositions support at most three app groups")
		}
		items := arrayValue(content["hero_items"])
		seen := map[string]bool{}
		for i, raw := range items {
			validateHeroItem(raw, fmt.Sprintf("%s.hero_items[%d]", path, i), seen, issues)
		}
	case BlockCapabilityStrip:
		for i, raw := range arrayValue(content["items"]) {
			validateCapabilityItem(raw, fmt.Sprintf("%s.items[%d]", path, i), issues)
		}
	case BlockProductStory:
		for i, raw := range arrayValue(content["items"]) {
			validateStoryItem(raw, fmt.Sprintf("%s.items[%d]", path, i), issues)
		}
	case BlockAppSpotlights:
		validateStringArray(content["app_keys"], path+".app_keys", issues, true)
		seen := map[string]bool{}
		for i, value := range stringArray(content["app_keys"]) {
			if seen[value] {
				issues.add(fmt.Sprintf("%s.app_keys[%d]", path, i), "duplicate_spotlight_app", "spotlight app keys must be unique")
			}
			seen[value] = true
		}
	case BlockArtifactExplorer:
		validateIDValue(content["capability_id"], path+".capability_id", issues)
		seen := map[string]bool{}
		for i, raw := range arrayValue(content["examples"]) {
			validateArtifact(raw, fmt.Sprintf("%s.examples[%d]", path, i), seen, issues)
		}
		validateIDValue(content["selected_example_id"], path+".selected_example_id", issues)
	case BlockVoiceStory:
		validateStringArray(content["capability_ids"], path+".capability_ids", issues, false)
		for i, raw := range arrayValue(content["features"]) {
			validateVoiceFeature(raw, fmt.Sprintf("%s.features[%d]", path, i), issues)
		}
		validateTextArray(content["summary_items"], path+".summary_items", issues, true)
		waveform := arrayValue(content["waveform"])
		if len(waveform) == 0 {
			issues.add(path+".waveform", "invalid_waveform", "waveform must contain samples")
		}
		for i, raw := range waveform {
			if n, ok := raw.(float64); !ok || math.IsNaN(n) || math.IsInf(n, 0) || n < 0 || n > 1 {
				issues.add(fmt.Sprintf("%s.waveform[%d]", path, i), "invalid_waveform", "samples must be finite values between 0 and 1")
			}
		}
	case BlockCapabilityRoadmap:
		validateStringArray(content["capability_ids"], path+".capability_ids", issues, false)
	case BlockPricing:
		// Plan refs are owner price identifiers (Stripe price IDs), not
		// document-local ids, so they validate as opaque references.
		if values, ok := content["plan_refs"].([]any); ok {
			seenRefs := map[string]bool{}
			for i, raw := range values {
				ref := stringValue(raw)
				validateOpaqueRef(ref, fmt.Sprintf("%s.plan_refs[%d]", path, i), issues)
				if seenRefs[ref] {
					issues.add(fmt.Sprintf("%s.plan_refs[%d]", path, i), "duplicate_plan_ref", "plan refs must be unique")
				}
				seenRefs[ref] = true
			}
		} else if content["plan_refs"] != nil {
			issues.add(path+".plan_refs", "invalid_array", "must be an array")
		}
	case BlockFAQ:
		for i, raw := range arrayValue(content["items"]) {
			validateFAQItem(raw, fmt.Sprintf("%s.items[%d]", path, i), issues)
		}
	case BlockFooter:
		validateLinkArray(content["links"], path+".links", issues)
	}
}

func validateProductDemo(variant string, content map[string]any, path string, issues *ValidationError) {
	recorded := variant == "recorded"
	fixtureRef := strings.TrimSpace(stringValue(content["fixture_ref"]))
	posterRef := strings.TrimSpace(stringValue(content["poster_ref"]))
	mediaRef := strings.TrimSpace(stringValue(content["media_ref"]))
	rendererRef := strings.TrimSpace(stringValue(content["renderer_ref"]))
	playback, hasPlayback := content["playback"]

	if recorded {
		if rendererRef != "video" {
			issues.add(path+".renderer_ref", "invalid_recorded_renderer", "recorded product demos must use renderer_ref video")
		}
		if fixtureRef != "" {
			issues.add(path+".fixture_ref", "forbidden_recorded_fixture", "recorded product demos cannot use fixture_ref")
		}
		if mediaRef != "" {
			issues.add(path+".media_ref", "forbidden_recorded_media", "recorded product demos cannot use media_ref")
		}
		if posterRef == "" {
			issues.add(path+".poster_ref", "missing_recorded_poster", "recorded product demos require a released poster_ref")
		}
		if !hasPlayback || playback == nil {
			issues.add(path+".playback", "missing_recorded_playback", "recorded product demos require typed playback")
		} else {
			validateProductDemoPlayback(playback, path+".playback", issues)
		}
		return
	}

	if rendererRef == "video" {
		issues.add(path+".renderer_ref", "invalid_fixture_renderer", "fixture product demos cannot use the recorded video renderer")
	}
	if fixtureRef == "" {
		issues.add(path+".fixture_ref", "missing_fixture_ref", "fixture product demos require fixture_ref")
	}
	if hasPlayback && playback != nil {
		issues.add(path+".playback", "ignored_playback_fields", "fixture product demos cannot carry playback fields")
	}
	if posterRef != "" {
		issues.add(path+".poster_ref", "ignored_media_fields", "fixture product demos cannot carry poster media fields")
	}
	if mediaRef != "" {
		issues.add(path+".media_ref", "ignored_media_fields", "fixture product demos cannot carry media fields")
	}
}

func validateProductDemoPlayback(value any, path string, issues *ValidationError) {
	playback, ok := value.(map[string]any)
	if !ok {
		issues.add(path, "invalid_playback", "playback must be an object")
		return
	}
	provider := stringValue(playback["provider"])
	if provider != "youtube" && provider != "vimeo" {
		issues.add(path+".provider", "unsupported_playback_provider", "provider must be youtube or vimeo")
	}
	if _, err := CanonicalPlaybackURL(provider, stringValue(playback["external_url"])); err != nil {
		issues.add(path+".external_url", "invalid_playback_url", err.Error())
	}
	layout := stringValue(playback["layout"])
	if layout != "stacked" && layout != "split" {
		issues.add(path+".layout", "unsupported_playback_layout", "layout must be stacked or split")
	}
	for _, field := range []string{"play_label", "caption", "unavailable_label"} {
		validateTextValue(playback[field], path+"."+field, issues, true)
	}
}

// CanonicalPlaybackURL validates an external provider URL and returns the
// privacy-enhanced embed URL used only after explicit player activation.
func CanonicalPlaybackURL(provider, externalURL string) (string, error) {
	if provider != "youtube" && provider != "vimeo" {
		return "", fmt.Errorf("provider must be youtube or vimeo")
	}
	if externalURL == "" || strings.HasSuffix(externalURL, "#") || strings.ContainsAny(externalURL, "\\\x00\r\n\t %") {
		return "", fmt.Errorf("URL must be an HTTPS provider URL without unsafe characters")
	}
	parsed, err := url.Parse(externalURL)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Hostname() == "" || parsed.Port() != "" || strings.Contains(parsed.Host, ":") || parsed.Fragment != "" {
		return "", fmt.Errorf("URL must be HTTPS with no credentials, port, or fragment")
	}
	if parsed.RawPath != "" || strings.Contains(parsed.Path, "\\") {
		return "", fmt.Errorf("URL path contains an unsafe character")
	}

	if provider == "youtube" {
		id, ok := youtubePlaybackID(parsed)
		if !ok {
			return "", fmt.Errorf("URL is not a canonical YouTube watch, short, or embed URL")
		}
		return "https://www.youtube-nocookie.com/embed/" + id + "?autoplay=0&controls=1&playsinline=1", nil
	}
	id, ok := vimeoPlaybackID(parsed)
	if !ok {
		return "", fmt.Errorf("URL is not a canonical Vimeo or Vimeo player URL")
	}
	return "https://player.vimeo.com/video/" + id + "?autoplay=0&controls=1&dnt=1", nil
}

func youtubePlaybackID(parsed *url.URL) (string, bool) {
	host := strings.ToLower(parsed.Hostname())
	if host == "youtu.be" {
		if parsed.RawQuery != "" || parsed.Path == "" || strings.Count(parsed.Path, "/") != 1 {
			return "", false
		}
		id := strings.TrimPrefix(parsed.Path, "/")
		return id, youtubeVideoIDPattern.MatchString(id)
	}
	if host != "www.youtube.com" && host != "youtube.com" && host != "www.youtube-nocookie.com" {
		return "", false
	}
	if host != "www.youtube-nocookie.com" && strings.HasPrefix(parsed.Path, "/watch") {
		if parsed.Path != "/watch" {
			return "", false
		}
		query, err := url.ParseQuery(parsed.RawQuery)
		if err != nil || len(query) != 1 || len(query["v"]) != 1 {
			return "", false
		}
		return query["v"][0], youtubeVideoIDPattern.MatchString(query["v"][0])
	}
	if (host == "youtube.com" || host == "www.youtube.com" || host == "www.youtube-nocookie.com") && strings.HasPrefix(parsed.Path, "/embed/") {
		if parsed.RawQuery != "" {
			return "", false
		}
		id := strings.TrimPrefix(parsed.Path, "/embed/")
		return id, youtubeVideoIDPattern.MatchString(id) && !strings.Contains(id, "/")
	}
	return "", false
}

func vimeoPlaybackID(parsed *url.URL) (string, bool) {
	host := strings.ToLower(parsed.Hostname())
	if parsed.RawQuery != "" || parsed.Path == "" {
		return "", false
	}
	var id string
	switch {
	case (host == "vimeo.com" || host == "www.vimeo.com") && strings.Count(parsed.Path, "/") == 1:
		id = strings.TrimPrefix(parsed.Path, "/")
	case host == "player.vimeo.com" && strings.HasPrefix(parsed.Path, "/video/") && strings.Count(parsed.Path, "/") == 2:
		id = strings.TrimPrefix(parsed.Path, "/video/")
	default:
		return "", false
	}
	return id, vimeoVideoIDPattern.MatchString(id)
}

func validatePageReferences(page Page, path string, apps map[string]App, capabilities map[string]Capability, assets map[string]Asset, fixtures map[string]Fixture, issues *ValidationError) {
	for i, block := range page.Blocks {
		content, err := contentMap(block.Content)
		if err != nil {
			continue
		}
		blockPath := fmt.Sprintf("%s.blocks[%d].content", path, i)
		if appKey, ok := content["app_key"].(string); ok && appKey != "" {
			validateAppRef(appKey, blockPath+".app_key", apps, issues)
		}
		for _, key := range []string{"fixture_ref"} {
			if ref, ok := content[key].(string); ok && ref != "" {
				if _, exists := fixtures[ref]; !exists {
					issues.add(blockPath+"."+key, "unknown_fixture_ref", "fixture reference is not declared")
				}
			}
		}
		for _, key := range []string{"capability_id"} {
			if id, ok := content[key].(string); ok {
				validateCapabilityRef(id, blockPath+"."+key, capabilities, issues)
			}
		}
		for _, key := range []string{"capability_ids"} {
			for j, value := range stringArray(content[key]) {
				validateCapabilityRef(value, fmt.Sprintf("%s.%s[%d]", blockPath, key, j), capabilities, issues)
			}
		}
		for _, key := range []string{"app_keys"} {
			for j, value := range stringArray(content[key]) {
				validateAppRef(value, fmt.Sprintf("%s.%s[%d]", blockPath, key, j), apps, issues)
			}
		}
		if actions, ok := content["actions"]; ok {
			for j, raw := range arrayValue(actions) {
				if action, ok := raw.(map[string]any); ok && stringValue(action["app_key"]) != "" {
					validateAppRef(stringValue(action["app_key"]), fmt.Sprintf("%s.actions[%d].app_key", blockPath, j), apps, issues)
				}
			}
		}
		if block.Kind == BlockCapabilityStrip {
			for j, raw := range arrayValue(content["items"]) {
				if item, ok := raw.(map[string]any); ok {
					validateCapabilityRef(stringValue(item["capability_id"]), fmt.Sprintf("%s.items[%d].capability_id", blockPath, j), capabilities, issues)
				}
			}
		}
		if block.Kind == BlockVoiceStory {
			for j, raw := range arrayValue(content["features"]) {
				if item, ok := raw.(map[string]any); ok {
					validateCapabilityRef(stringValue(item["capability_id"]), fmt.Sprintf("%s.features[%d].capability_id", blockPath, j), capabilities, issues)
				}
			}
		}
		if page.ID == "" {
			continue
		}
		if block.Kind == BlockBundleHero {
			for j, raw := range arrayValue(content["hero_items"]) {
				if item, ok := raw.(map[string]any); ok {
					if appKey, ok := item["app_key"].(string); ok {
						validateAppRef(appKey, fmt.Sprintf("%s.hero_items[%d].app_key", blockPath, j), apps, issues)
					}
				}
			}
		}
		for _, key := range []string{"visual_ref", "poster_ref", "media_ref"} {
			if ref, ok := content[key].(string); ok && ref != "" {
				if _, exists := assets[ref]; !exists {
					issues.add(blockPath+"."+key, "unknown_asset_ref", "asset reference is not declared")
				}
			}
		}
		if block.Kind == BlockBundleHero {
			for j, raw := range arrayValue(content["hero_items"]) {
				if item, ok := raw.(map[string]any); ok {
					if ref, ok := item["visual_ref"].(string); ok && ref != "" {
						if _, exists := assets[ref]; !exists {
							issues.add(fmt.Sprintf("%s.hero_items[%d].visual_ref", blockPath, j), "unknown_asset_ref", "asset reference is not declared")
						}
					}
				}
			}
		}
		if block.Kind == BlockArtifactExplorer {
			for j, raw := range arrayValue(content["examples"]) {
				if item, ok := raw.(map[string]any); ok {
					for _, key := range []string{"asset_ref", "preview_ref"} {
						if ref, ok := item[key].(string); ok && ref != "" {
							if _, exists := assets[ref]; !exists {
								issues.add(fmt.Sprintf("%s.examples[%d].%s", blockPath, j, key), "unknown_asset_ref", "asset reference is not declared")
							}
						}
					}
				}
			}
		}
		if block.Kind == BlockArtifactExplorer {
			for j, raw := range arrayValue(content["examples"]) {
				if item, ok := raw.(map[string]any); ok {
					for k, frameRaw := range arrayValue(item["frames"]) {
						if frame, ok := frameRaw.(map[string]any); ok {
							if ref, ok := frame["asset_ref"].(string); ok {
								if _, exists := assets[ref]; !exists {
									issues.add(fmt.Sprintf("%s.examples[%d].frames[%d].asset_ref", blockPath, j, k), "unknown_asset_ref", "asset reference is not declared")
								}
							}
						}
					}
				}
			}
		}
	}
}

func validateActions(value any, path string, issues *ValidationError) {
	if value == nil {
		return
	}
	items, ok := value.([]any)
	if !ok {
		issues.add(path, "invalid_actions", "actions must be an array")
		return
	}
	seen := map[ActionKind]bool{}
	seenPurchaseRefs := map[string]bool{}
	for i, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			issues.add(fmt.Sprintf("%s[%d]", path, i), "invalid_action", "action must be an object")
			continue
		}
		p := fmt.Sprintf("%s[%d]", path, i)
		kind, _ := item["kind"].(string)
		actionKind := ActionKind(kind)
		if !validActionKinds[actionKind] {
			issues.add(p+".kind", "invalid_action_kind", "action kind is not supported")
		}
		// A purchase action targets one plan, so a block offering several
		// plans repeats the kind; each plan may carry only one purchase CTA.
		if actionKind == ActionPurchase {
			planRef := stringValue(item["plan_ref"])
			if seenPurchaseRefs[planRef] {
				issues.add(p+".plan_ref", "duplicate_purchase_plan", "each plan may carry only one purchase action per block")
			}
			seenPurchaseRefs[planRef] = true
		} else {
			if seen[actionKind] {
				issues.add(p+".kind", "duplicate_action_kind", "an action kind may appear only once per block")
			}
			seen[actionKind] = true
		}
		validateTextValue(item["label"], p+".label", issues, true)
		validateTextValue(item["accessible_label"], p+".accessible_label", issues, true)
		for _, key := range []string{"target", "plan_ref", "app_key", "reason"} {
			if value, ok := item[key].(string); ok {
				validateText(value, p+"."+key, issues, false)
			}
		}
		if planRef := stringValue(item["plan_ref"]); planRef != "" {
			validateOpaqueRef(planRef, p+".plan_ref", issues)
		}
		switch actionKind {
		case ActionUnavailable:
			if strings.TrimSpace(stringValue(item["reason"])) == "" {
				issues.add(p+".reason", "missing_action_reason", "unavailable actions require a visible reason")
			}
		case ActionPurchase:
			if strings.TrimSpace(stringValue(item["plan_ref"])) == "" {
				issues.add(p+".plan_ref", "missing_plan_ref", "purchase actions require a plan reference")
			}
			if item["target"] != nil {
				issues.add(p+".target", "invalid_purchase_target", "purchase destinations come from commerce owners")
			}
		case ActionDownload:
			validateIDValue(item["app_key"], p+".app_key", issues)
			if stringValue(item["target"]) != "" {
				issues.add(p+".target", "invalid_download_target", "download destinations come from delivery owners")
			}
		case ActionOpen:
			if stringValue(item["app_key"]) != "" {
				validateIDValue(item["app_key"], p+".app_key", issues)
				if stringValue(item["target"]) != "" {
					issues.add(p+".target", "invalid_open_target", "app launch destinations come from delivery owners")
				}
			} else if !isSafeTarget(stringValue(item["target"])) {
				issues.add(p+".target", "unsafe_target", "action target must be a safe relative route or anchor")
			}
		case ActionRequestAccess:
			if !isSafeTarget(stringValue(item["target"])) {
				issues.add(p+".target", "unsafe_target", "action target must be a safe relative route or anchor")
			}
		case ActionAnchor:
			target := stringValue(item["target"])
			if !strings.HasPrefix(target, "#") || len(target) < 2 || !idPattern.MatchString(target[1:]) {
				issues.add(p+".target", "invalid_anchor", "anchor actions require a local #target")
			}
		case ActionAppDetail:
			validateIDValue(item["app_key"], p+".app_key", issues)
		}
	}
}

func validateHeroItem(value any, path string, seen map[string]bool, issues *ValidationError) {
	item, ok := value.(map[string]any)
	if !ok {
		issues.add(path, "invalid_hero_item", "hero items must be objects")
		return
	}
	appKey := stringValue(item["app_key"])
	validateID(appKey, path+".app_key", issues)
	if seen[appKey] {
		issues.add(path+".app_key", "duplicate_hero_app", "one visual group is allowed per app")
	}
	seen[appKey] = true
	if ref := stringValue(item["visual_ref"]); ref != "" {
		validateID(ref, path+".visual_ref", issues)
	}
	validateTextValue(item["detail_label"], path+".detail_label", issues, true)
	valid := map[string]bool{"artwork": true, "screenshot": true, "product-view": true, "visual": true}
	if !valid[stringValue(item["exhibit_kind"])] {
		issues.add(path+".exhibit_kind", "unsupported_exhibit_kind", "hero exhibit kind is not registered")
	}
}

func validateCapabilityItem(value any, path string, issues *ValidationError) {
	item, ok := value.(map[string]any)
	if !ok {
		issues.add(path, "invalid_capability_item", "items must be objects")
		return
	}
	validateID(stringValue(item["capability_id"]), path+".capability_id", issues)
	validateTextValue(item["label"], path+".label", issues, true)
	validateTextValue(item["description"], path+".description", issues, true)
}

func validateStoryItem(value any, path string, issues *ValidationError) {
	item, ok := value.(map[string]any)
	if !ok {
		issues.add(path, "invalid_story_item", "items must be objects")
		return
	}
	validateTextValue(item["title"], path+".title", issues, true)
	validateTextValue(item["description"], path+".description", issues, true)
}

func validateFAQItem(value any, path string, issues *ValidationError) {
	item, ok := value.(map[string]any)
	if !ok {
		issues.add(path, "invalid_faq_item", "items must be objects")
		return
	}
	validateTextValue(item["question"], path+".question", issues, true)
	validateTextValue(item["answer"], path+".answer", issues, true)
	validateTextValue(item["accessible_label"], path+".accessible_label", issues, true)
}

func validateArtifact(value any, path string, seen map[string]bool, issues *ValidationError) {
	item, ok := value.(map[string]any)
	if !ok {
		issues.add(path, "invalid_artifact", "examples must be objects")
		return
	}
	id := stringValue(item["id"])
	validateID(id, path+".id", issues)
	if seen[id] {
		issues.add(path+".id", "duplicate_artifact_id", "artifact IDs must be unique")
	}
	seen[id] = true
	kind := ArtifactKind(stringValue(item["kind"]))
	if !validArtifactKinds[kind] {
		issues.add(path+".kind", "unsupported_artifact_kind", "artifact kind is not registered")
	}
	validateTextValue(item["label"], path+".label", issues, true)
	validateTextValue(item["filename"], path+".filename", issues, true)
	validateTextValue(item["type"], path+".type", issues, true)
	validateTextValue(item["title"], path+".title", issues, true)
	validateTextValue(item["body"], path+".body", issues, true)
	validateTextValue(item["caption"], path+".caption", issues, true)
	validateTextValue(item["alt_text"], path+".alt_text", issues, true)
	validateTextValue(item["icon"], path+".icon", issues, true)
	validateIDValue(item["source_ref"], path+".source_ref", issues)
	width, wok := numberValue(item["width"])
	height, hok := numberValue(item["height"])
	if !wok || width <= 0 {
		issues.add(path+".width", "invalid_dimensions", "width must be positive")
	}
	if !hok || height <= 0 {
		issues.add(path+".height", "invalid_dimensions", "height must be positive")
	}
	if source, ok := item["source"].(map[string]any); !ok || strings.TrimSpace(stringValue(source["ref"])) == "" {
		issues.add(path+".source.ref", "missing_source_ref", "artifacts require typed source metadata")
	} else {
		validateOpaqueRef(stringValue(source["ref"]), path+".source.ref", issues)
		if mediaRef := stringValue(source["media_ref"]); mediaRef != "" {
			validateID(mediaRef, path+".source.media_ref", issues)
		}
		if integrityHash := stringValue(source["integrity_hash"]); integrityHash != "" && !hashPattern.MatchString(integrityHash) {
			issues.add(path+".source.integrity_hash", "invalid_content_hash", "must be a hexadecimal content hash")
		}
		if isolation := stringValue(source["isolation"]); isolation != "" && isolation != "sandbox" && isolation != "plain" {
			issues.add(path+".source.isolation", "invalid_source_isolation", "must be sandbox or plain")
		}
	}
	if kind == ArtifactHTMLPreview && (item["preview_ref"] == nil || strings.TrimSpace(stringValue(item["preview_ref"])) == "") {
		issues.add(path+".preview_ref", "missing_preview_ref", "HTML previews require a safe isolated preview reference")
	}
	if kind == ArtifactHTMLPreview {
		source, _ := item["source"].(map[string]any)
		if stringValue(source["isolation"]) != "sandbox" {
			issues.add(path+".source.isolation", "unsafe_html_source", "HTML previews require sandbox isolation")
		}
	}
	for i, raw := range arrayValue(item["steps"]) {
		validateTextValue(raw, fmt.Sprintf("%s.steps[%d]", path, i), issues, true)
	}
	for i, raw := range arrayValue(item["checks"]) {
		validateTextValue(raw, fmt.Sprintf("%s.checks[%d]", path, i), issues, true)
	}
	for i, raw := range arrayValue(item["frames"]) {
		frame, ok := raw.(map[string]any)
		framePath := fmt.Sprintf("%s.frames[%d]", path, i)
		if !ok {
			issues.add(framePath, "invalid_frame", "frame must be an object")
			continue
		}
		validateIDValue(frame["asset_ref"], framePath+".asset_ref", issues)
		validateTextValue(frame["time"], framePath+".time", issues, true)
		validateTextValue(frame["label"], framePath+".label", issues, true)
	}
}

func validateVoiceFeature(value any, path string, issues *ValidationError) {
	item, ok := value.(map[string]any)
	if !ok {
		issues.add(path, "invalid_voice_feature", "feature rows must be objects")
		return
	}
	validateTextValue(item["title"], path+".title", issues, true)
	validateTextValue(item["description"], path+".description", issues, true)
	validateIDValue(item["capability_id"], path+".capability_id", issues)
}

func validateLinkArray(value any, path string, issues *ValidationError) {
	items, ok := value.([]any)
	if !ok {
		issues.add(path, "invalid_links", "links must be an array")
		return
	}
	for i, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			issues.add(fmt.Sprintf("%s[%d]", path, i), "invalid_link", "link must be an object")
			continue
		}
		p := fmt.Sprintf("%s[%d]", path, i)
		validateTextValue(item["label"], p+".label", issues, true)
		validateTextValue(item["accessible_label"], p+".accessible_label", issues, true)
		if !isSafeTarget(stringValue(item["target"])) {
			issues.add(p+".target", "unsafe_target", "link target must be a safe relative route or anchor")
		}
	}
}

func validateAssetRefs(content map[string]any, path string, issues *ValidationError) {
	for _, key := range sortedStringKeys(content) {
		value := content[key]
		if strings.HasSuffix(key, "_ref") && value != nil {
			if ref, ok := value.(string); ok && ref != "" {
				validateID(ref, path+"."+key, issues)
			}
		}
	}
}

func validateStringArray(value any, path string, issues *ValidationError, nonEmpty bool) {
	values, ok := value.([]any)
	if !ok {
		issues.add(path, "invalid_array", "must be an array")
		return
	}
	if nonEmpty && len(values) == 0 {
		issues.add(path, "empty_array", "must contain at least one item")
	}
	for i, v := range values {
		validateID(stringValue(v), fmt.Sprintf("%s[%d]", path, i), issues)
	}
}

func validateRefList(values []string, path string, issues *ValidationError, required bool) {
	if required && len(values) == 0 {
		issues.add(path, "missing_ref", "at least one reference is required")
	}
	for i, v := range values {
		validateOpaqueRef(v, fmt.Sprintf("%s[%d]", path, i), issues)
	}
}

func validateTextList(values []string, path string, issues *ValidationError) {
	for i, v := range values {
		validateText(v, fmt.Sprintf("%s[%d]", path, i), issues, true)
	}
}

func validateTextArray(value any, path string, issues *ValidationError, nonEmpty bool) {
	values, ok := value.([]any)
	if !ok {
		issues.add(path, "invalid_array", "must be an array")
		return
	}
	if nonEmpty && len(values) == 0 {
		issues.add(path, "empty_array", "must contain at least one item")
	}
	for i, value := range values {
		validateTextValue(value, fmt.Sprintf("%s[%d]", path, i), issues, true)
	}
}

func validateOptionalID(value any, path string, issues *ValidationError) {
	if value != nil {
		validateIDValue(value, path, issues)
	}
}

func validateIDValue(value any, path string, issues *ValidationError) {
	validateID(stringValue(value), path, issues)
}

func validateTextValue(value any, path string, issues *ValidationError, required bool) {
	if value == nil {
		if required {
			issues.add(path, "missing_text", "text is required")
		}
		return
	}
	text, ok := value.(string)
	if !ok {
		issues.add(path, "invalid_text", "must be a string")
		return
	}
	validateText(text, path, issues, required)
}

func validateText(value, path string, issues *ValidationError, required bool) {
	if required && strings.TrimSpace(value) == "" {
		issues.add(path, "missing_text", "must not be empty")
	}
	if strings.ContainsAny(value, "\x00\r\n") {
		issues.add(path, "invalid_text", "must not contain control characters or line breaks")
	}
	if len([]rune(value)) > 10000 {
		issues.add(path, "text_too_long", "must be at most 10000 characters")
	}
}

func validateID(value, path string, issues *ValidationError) {
	if !idPattern.MatchString(value) {
		issues.add(path, "invalid_identifier", "must match lowercase identifier syntax")
	}
}

func validateSlug(value, path string, issues *ValidationError) {
	if !slugPattern.MatchString(value) {
		issues.add(path, "invalid_slug", "must be a lowercase URL slug")
	}
}

func validateLocale(value, path string, issues *ValidationError) {
	if !localePattern.MatchString(value) {
		issues.add(path, "invalid_locale", "must be a safe BCP-47-like locale")
	}
}

func validateOpaqueRef(value, path string, issues *ValidationError) {
	if strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\x00\r\n") || strings.Contains(value, "..") || strings.ContainsAny(value, "/\\") {
		issues.add(path, "invalid_reference", "must be a non-empty opaque reference")
	}
}

func validateAppRef(key, path string, apps map[string]App, issues *ValidationError) {
	if _, ok := apps[key]; !ok {
		issues.add(path, "unknown_app_ref", "app reference is not declared")
	}
}

func validateCapabilityRef(id, path string, caps map[string]Capability, issues *ValidationError) {
	if _, ok := caps[id]; !ok {
		issues.add(path, "unknown_capability_ref", "capability reference is not declared")
	}
}
func hasPage(pages map[string][]Page, id string) bool { return len(pages[id]) > 0 }
func normalizedLocale(value string) string {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) == 0 {
		return ""
	}
	parts[0] = strings.ToLower(parts[0])
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) == 2 || (len(parts[i]) == 3 && isAllDigits(parts[i])) {
			parts[i] = strings.ToUpper(parts[i])
		} else {
			parts[i] = strings.ToLower(parts[i])
		}
	}
	return strings.Join(parts, "-")
}

func isAllDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}

func validUnitRect(x, y, w, h float64) bool {
	return !math.IsNaN(x) && !math.IsNaN(y) && !math.IsNaN(w) && !math.IsNaN(h) && x >= 0 && y >= 0 && w > 0 && h > 0 && x+w <= 1 && y+h <= 1
}

func unsafeContentKey(key string) bool {
	switch strings.ToLower(key) {
	case "html", "raw_html", "markup", "jsx", "script", "javascript", "css", "onload", "onclick":
		return true
	}
	return false
}
func isString(value any) bool      { _, ok := value.(string); return ok }
func stringValue(value any) string { text, _ := value.(string); return text }
func arrayValue(value any) []any   { values, _ := value.([]any); return values }

func sortedStringKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func stringArray(value any) []string {
	var result []string
	for _, item := range arrayValue(value) {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return result
}
func numberValue(value any) (float64, bool) { number, ok := value.(float64); return number, ok }
func isSafePublicURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && !parsed.IsAbs() && parsed.Host == "" && parsed.User == nil && strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") && !strings.ContainsAny(value, "\\\x00\r\n\t") && !strings.ContainsAny(parsed.Path, "\\\x00\r\n\t")
}

// isSafeAssetPath narrows isSafePublicURL for configured brand-logo images:
// a relative, same-origin path with no traversal or query/fragment injection.
func isSafeAssetPath(value string) bool {
	if !isSafePublicURL(value) || strings.Contains(value, "..") {
		return false
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.RawQuery == "" && parsed.Fragment == ""
}

func isSafeTarget(value string) bool {
	if value == "" || strings.ContainsAny(value, "\\\x00\r\n\t") || strings.HasPrefix(value, "//") {
		return false
	}
	if strings.HasPrefix(value, "#") {
		return len(value) > 1 && !strings.ContainsAny(value[1:], " #?")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.User != nil || strings.ContainsAny(parsed.Path, "\\\x00\r\n\t") {
		return false
	}
	if parsed.IsAbs() {
		return parsed.Scheme == "https" && parsed.Host != ""
	}
	return parsed.Host == "" && strings.HasPrefix(value, "/") && !strings.Contains(parsed.Path, "..")
}
