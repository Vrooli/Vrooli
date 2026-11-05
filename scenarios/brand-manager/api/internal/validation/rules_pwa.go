package validation

import "strings"

// This file holds the accessibility-depth (dark mode) and PWA/mobile-theming
// rules. They are all UI-surface-conditional (registered with surfaceUI) and
// never error-severity: a missing dark mode or PWA polish is advisory, per the
// branding severity contract.

// ruleDarkModeContrast re-checks the core color pairings against the dark-scheme
// overrides. It only fires when a dark block ships AND a pairing fails there —
// never for the absence of dark mode (that is color-scheme-declared's info nudge).
func ruleDarkModeContrast(c *scanContext) (Finding, bool) {
	content, ok := c.tokenContent()
	if !ok {
		return Finding{}, false
	}
	dark := cssVarsForScheme(content, schemeDark)
	if len(dark) == 0 {
		return Finding{}, false // no dark block — not this rule's concern
	}
	merged := cssVarsForScheme(content, schemeLight)
	for k, v := range dark {
		merged[k] = v
	}
	failures, worst := contrastFailures(merged)
	if len(failures) == 0 {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityWarning,
		Title:                  "Dark-mode color pairings fail WCAG AA contrast",
		Description:            "One or more color pairings fail WCAG 2.1 AA (4.5:1) when the dark-scheme overrides are applied.",
		FilePath:               designSystemCSSRel,
		WhyItMatters:           "Dark mode is shipped but unreadable for low-vision users where its overrides lower contrast.",
		RecommendedRemediation: "Adjust the dark-scheme --color-* overrides so each pairing reaches at least 4.5:1.",
		Evidence:               map[string]any{"failures": failures, "worst_ratio": worst, "scheme": "dark"},
	}, true
}

// colorSchemeDeclared reports whether the scenario declares color-scheme support
// (a <meta name="color-scheme"> or a CSS `color-scheme:` declaration anywhere in
// the design tokens).
func colorSchemeDeclared(c *scanContext) bool {
	if v, ok := c.head().metaByName("color-scheme"); ok && strings.TrimSpace(v) != "" {
		return true
	}
	content, _ := c.tokenContent()
	return strings.Contains(content, "color-scheme:")
}

// ruleColorSchemeDeclared nudges (info) a scenario that ships a dark block but
// never declares color-scheme, so the browser form controls/scrollbars match.
func ruleColorSchemeDeclared(c *scanContext) (Finding, bool) {
	content, ok := c.tokenContent()
	if !ok || !hasDarkScheme(content) {
		return Finding{}, false
	}
	if colorSchemeDeclared(c) {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityInfo,
		Title:                  "Dark mode is shipped without a declared color-scheme",
		Description:            "Design tokens define a dark scheme but no <meta name=\"color-scheme\"> or CSS color-scheme declaration tells the browser to theme native controls.",
		FilePath:               indexHTMLRel,
		WhyItMatters:           "Without color-scheme, native form controls and scrollbars stay light on a dark page.",
		RecommendedRemediation: "Add <meta name=\"color-scheme\" content=\"light dark\"> to ui/index.html.",
	}, true
}

// ruleThemeColorPresent requires a <meta name="theme-color">; when a dark scheme
// ships it additionally requires a dark media variant so the browser chrome
// matches the active scheme.
func ruleThemeColorPresent(c *scanContext) (Finding, bool) {
	metas := c.head().metasByName("theme-color")
	if len(metas) == 0 {
		return Finding{
			Severity:               SeverityWarning,
			Title:                  "No theme-color is declared",
			Description:            "ui/index.html declares no <meta name=\"theme-color\">, so mobile browser chrome falls back to its default.",
			FilePath:               indexHTMLRel,
			WhyItMatters:           "theme-color paints the mobile browser address bar and task-switcher card with the brand color.",
			RecommendedRemediation: "Add <meta name=\"theme-color\" content=\"#...\"> to ui/index.html.",
			Evidence:               map[string]any{"missing": "theme-color"},
		}, true
	}
	content, _ := c.tokenContent()
	if hasDarkScheme(content) && !hasDarkMediaVariant(metas) {
		return Finding{
			Severity:               SeverityWarning,
			Title:                  "theme-color has no dark-scheme variant",
			Description:            "A dark scheme ships but theme-color declares no (prefers-color-scheme: dark) media variant, so the browser chrome stays the light color in dark mode.",
			FilePath:               indexHTMLRel,
			WhyItMatters:           "A single theme-color leaves the mobile chrome mismatched against a dark page.",
			RecommendedRemediation: "Declare light and dark theme-color metas with matching media queries.",
			Evidence:               map[string]any{"variants": len(metas)},
		}, true
	}
	return Finding{}, false
}

func hasDarkMediaVariant(metas []metaTag) bool {
	for _, m := range metas {
		if strings.Contains(strings.ToLower(m.media), "dark") {
			return true
		}
	}
	return false
}

// standaloneMetas are the two capability metas an installable PWA declares.
var standaloneMetas = []string{"mobile-web-app-capable", "apple-mobile-web-app-capable"}

// ruleStandaloneCapable requires BOTH the standard and Apple "web app capable"
// metas so an installed PWA launches without browser chrome on every platform.
func ruleStandaloneCapable(c *scanContext) (Finding, bool) {
	var missing []string
	for _, name := range standaloneMetas {
		if v, ok := c.head().metaByName(name); !ok || strings.TrimSpace(v) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityInfo,
		Title:                  "Not declared as a standalone-capable web app",
		Description:            "ui/index.html is missing a web-app-capable meta, so an installed PWA may launch inside browser chrome.",
		FilePath:               indexHTMLRel,
		WhyItMatters:           "Both mobile-web-app-capable and apple-mobile-web-app-capable are needed for a chrome-free install across platforms.",
		RecommendedRemediation: "Add the missing web-app-capable metas to ui/index.html.",
		Evidence:               map[string]any{"missing": missing},
	}, true
}

const (
	statusBarStyleMeta  = "apple-mobile-web-app-status-bar-style"
	translucentStyle    = "black-translucent"
	viewportFitCover    = "viewport-fit=cover"
	safeAreaInsetMarker = "safe-area-inset"
)

// ruleIOSStatusBarSafeArea fires when the translucent iOS status-bar style is
// requested but the page does not opt into the full-bleed layout it requires:
// viewport-fit=cover AND real env(safe-area-inset-*) usage. Without both, content
// renders under the status bar / notch.
func ruleIOSStatusBarSafeArea(c *scanContext) (Finding, bool) {
	style, ok := c.head().metaByName(statusBarStyleMeta)
	if !ok || strings.TrimSpace(style) != translucentStyle {
		return Finding{}, false // only the translucent style needs safe-area handling
	}
	viewport, _ := c.head().viewportContent()
	missing := map[string]any{}
	if !strings.Contains(strings.ReplaceAll(viewport, " ", ""), viewportFitCover) {
		missing["viewport_fit_cover"] = true
	}
	if !c.uiCSSContains(safeAreaInsetMarker) {
		missing["safe_area_inset_usage"] = true
	}
	if len(missing) == 0 {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityWarning,
		Title:                  "Translucent iOS status bar without safe-area handling",
		Description:            "apple-mobile-web-app-status-bar-style is black-translucent but the layout does not opt into the safe area it requires (viewport-fit=cover + env(safe-area-inset-*)).",
		FilePath:               indexHTMLRel,
		WhyItMatters:           "Under a translucent status bar, content renders beneath the clock/notch unless the page reserves the safe-area insets.",
		RecommendedRemediation: "Add viewport-fit=cover to the viewport meta and pad the layout with env(safe-area-inset-*).",
		Evidence:               missing,
	}, true
}

// ruleManifestCompleteness requires the web-app manifest to declare every field
// needed for an installable, shareable PWA.
func ruleManifestCompleteness(c *scanContext) (Finding, bool) {
	rel, obj, _, present := c.manifest()
	if !present {
		return Finding{
			Severity:               SeverityWarning,
			Title:                  "No web-app manifest",
			Description:            "The scenario ships no site.webmanifest/manifest.json, so it cannot be installed as a PWA.",
			FilePath:               rel,
			WhyItMatters:           "An installable manifest is what lets users add the app to a home screen with the right name, icons, and colors.",
			RecommendedRemediation: "Add ui/public/site.webmanifest with name, icons (192/512 + maskable), theme/background color, and display: standalone.",
			Evidence:               map[string]any{"manifest_present": false},
		}, true
	}
	var missing []string
	for _, key := range manifestRequiredKeys {
		if !manifestHasKey(obj, key) {
			missing = append(missing, key)
		}
	}
	if !manifestHasMaskableIcon(obj) {
		missing = append(missing, "icons[maskable]")
	}
	if len(missing) == 0 {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityWarning,
		Title:                  "Web-app manifest is incomplete",
		Description:            "The manifest is missing fields needed for an installable, shareable PWA.",
		FilePath:               rel,
		WhyItMatters:           "Incomplete manifests degrade the install prompt, home-screen icon, and themed launch.",
		RecommendedRemediation: "Fill the missing manifest fields; generate the 192/512 + maskable icon set from the logo.",
		Evidence:               map[string]any{"missing": missing},
	}, true
}

// manifestRequiredKeys is the scalar+icons key set a complete manifest declares.
var manifestRequiredKeys = []string{
	"name", "short_name", "description",
	"theme_color", "background_color",
	"display", "start_url", "id", "icons",
}

func manifestHasKey(obj map[string]any, key string) bool {
	v, ok := obj[key]
	if !ok || v == nil {
		return false
	}
	if s, isStr := v.(string); isStr {
		return strings.TrimSpace(s) != ""
	}
	if arr, isArr := v.([]any); isArr {
		return len(arr) > 0
	}
	return true
}

func manifestHasMaskableIcon(obj map[string]any) bool {
	icons, ok := obj["icons"].([]any)
	if !ok {
		return false
	}
	for _, ic := range icons {
		m, ok := ic.(map[string]any)
		if !ok {
			continue
		}
		if purpose, _ := m["purpose"].(string); strings.Contains(strings.ToLower(purpose), "maskable") {
			return true
		}
	}
	return false
}
