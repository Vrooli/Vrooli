package validation

import (
	"regexp"
	"strings"

	"brand-manager/internal/brandsurface"
)

// This file holds cross-surface consistency and template-residue rules. They are
// UI-surface-conditional and warning-severity: a name that disagrees across
// surfaces, or leftover scaffold text, is a real branding defect but not a hard
// gate.

// ruleNameConsistency flags any present name-bearing surface that disagrees with
// service.displayName (the human-authored SSOT): <title>, application-name,
// apple-mobile-web-app-title, and manifest name/short_name. A surface that is
// simply absent is not flagged here (presence is other rules' concern); only
// disagreement among existing surfaces is a consistency defect.
func ruleNameConsistency(c *scanContext) (Finding, bool) {
	id := c.identity()
	if !id.HasIdentity() {
		return Finding{}, false // has-display-name owns the missing-identity case
	}
	mismatches := nameMismatches(c, id)
	if len(mismatches) == 0 {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityWarning,
		Title:                  "Brand name is inconsistent across surfaces",
		Description:            "One or more surfaces render a name that disagrees with service.displayName.",
		FilePath:               indexHTMLRel,
		WhyItMatters:           "A name that differs between the tab title, manifest, and iOS title reads as a half-rebranded app.",
		RecommendedRemediation: "Align every surface to service.displayName.",
		Evidence:               map[string]any{"display_name": id.DisplayName, "mismatches": mismatches},
	}, true
}

// nameMismatches returns the per-surface (surface→actual) values that disagree
// with the display name. Only surfaces that exist are considered.
func nameMismatches(c *scanContext, id brandsurface.Surface) map[string]any {
	want := id.DisplayName
	out := map[string]any{}
	h := c.head()
	if h.title != "" && !strings.EqualFold(h.title, want) {
		out["title"] = h.title
	}
	for _, tag := range id.ConsistencyTags() {
		if v, ok := h.metaByName(tag.Key); ok && !strings.EqualFold(strings.TrimSpace(v), want) {
			out[tag.Key] = v
		}
	}
	if _, obj, _, present := c.manifest(); present {
		// Only the full manifest `name` must equal the display name; `short_name`
		// is an intentional home-screen abbreviation and is not flagged here.
		if v, ok := obj["name"].(string); ok && strings.TrimSpace(v) != "" && !strings.EqualFold(strings.TrimSpace(v), want) {
			out["manifest.name"] = v
		}
	}
	return out
}

// ruleThemeColorConsistency flags a <meta theme-color> that disagrees with the
// manifest theme_color. Presence of each is owned by theme-color-present /
// manifest-completeness; this rule owns only the disagreement.
func ruleThemeColorConsistency(c *scanContext) (Finding, bool) {
	meta, hasMeta := c.head().metaByName("theme-color")
	_, obj, _, present := c.manifest()
	if !hasMeta || !present {
		return Finding{}, false
	}
	mfColor, ok := obj["theme_color"].(string)
	if !ok || strings.TrimSpace(mfColor) == "" {
		return Finding{}, false
	}
	if strings.EqualFold(strings.TrimSpace(meta), strings.TrimSpace(mfColor)) {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityWarning,
		Title:                  "theme-color disagrees between meta and manifest",
		Description:            "The <meta name=\"theme-color\"> value differs from the manifest theme_color, so the chrome and launch colors diverge.",
		FilePath:               indexHTMLRel,
		WhyItMatters:           "The address-bar color and the PWA splash/launch color should be the same brand color.",
		RecommendedRemediation: "Set the manifest theme_color to match the <meta theme-color> (the page is the source of truth).",
		Evidence:               map[string]any{"meta_theme_color": meta, "manifest_theme_color": mfColor},
	}, true
}

// ruleThemeColorDesignToken flags mobile/browser chrome colors that agree with
// each other but disagree with the scenario's root DESIGN.md surface token.
// DESIGN.md is the design language contract; if a scenario intentionally wants
// launch chrome to diverge, it must document that with the override marker below.
func ruleThemeColorDesignToken(c *scanContext) (Finding, bool) {
	design, ok := c.designSurfaceColor()
	if !ok {
		return Finding{}, false
	}
	if c.designThemeColorOverride() {
		return Finding{}, false
	}
	meta, hasMeta := c.head().metaByName("theme-color")
	_, obj, _, hasManifest := c.manifest()
	manifestColor, _ := obj["theme_color"].(string)
	mismatches := map[string]string{}
	if hasMeta && !sameColor(meta, design) {
		mismatches["meta_theme_color"] = strings.TrimSpace(meta)
	}
	if hasManifest && strings.TrimSpace(manifestColor) != "" && !sameColor(manifestColor, design) {
		mismatches["manifest_theme_color"] = strings.TrimSpace(manifestColor)
	}
	if len(mismatches) == 0 {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityWarning,
		Title:                  "theme-color disagrees with the design-kit surface token",
		Description:            "The declared browser/PWA theme color differs from DESIGN.md's surface color token.",
		FilePath:               "DESIGN.md",
		WhyItMatters:           "Mobile browser chrome and launch surfaces should match the scenario's canonical surface color unless a deliberate exception is documented.",
		RecommendedRemediation: "Set <meta theme-color> and manifest theme_color to the DESIGN.md surface token, or add a DESIGN.md note containing brand-manager:theme-color-token-override with the reason.",
		Evidence: map[string]any{
			"design_surface": design,
			"mismatches":     mismatches,
		},
	}, true
}

const themeColorDesignOverrideMarker = "brand-manager:theme-color-token-override"

var designSurfaceRe = regexp.MustCompile(`(?m)^\s*surface:\s*["']?(#[0-9a-fA-F]{3,8})["']?\s*$`)

func (c *scanContext) designSurfaceColor() (string, bool) {
	content, ok := c.read("DESIGN.md")
	if !ok {
		return "", false
	}
	if m := designSurfaceRe.FindStringSubmatch(content); len(m) > 1 {
		return strings.ToLower(strings.TrimSpace(m[1])), true
	}
	return "", false
}

func (c *scanContext) designThemeColorOverride() bool {
	content, ok := c.read("DESIGN.md")
	return ok && strings.Contains(content, themeColorDesignOverrideMarker)
}

func sameColor(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// templateResidueMarkers are case-insensitive substrings that betray unbranded
// scaffold output in the document title.
var templateTitleMarkers = []string{"vite + react", "vite app", "react app", "create react app"}

// ruleNoTemplateResidue detects leftover scaffold branding: a default Vite
// favicon (vite.svg), a template <title>, lorem-ipsum copy, or a vite.svg asset.
func ruleNoTemplateResidue(c *scanContext) (Finding, bool) {
	residues := templateResidues(c)
	if len(residues) == 0 {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityWarning,
		Title:                  "Template/scaffold residue is still shipped",
		Description:            "The UI still ships default scaffold branding (Vite favicon/title or placeholder copy).",
		FilePath:               indexHTMLRel,
		WhyItMatters:           "Scaffold residue makes the product look unfinished and unbranded.",
		RecommendedRemediation: "Replace the template title/favicon with the brand's and remove placeholder copy.",
		Evidence:               map[string]any{"residues": residues},
	}, true
}

// templateResidues returns the detected residue kinds (stable keys).
func templateResidues(c *scanContext) map[string]any {
	out := map[string]any{}
	h := c.head()
	for _, l := range h.links {
		if strings.Contains(strings.ToLower(l.href), "vite.svg") {
			out["vite_favicon"] = l.href
		}
	}
	lowerTitle := strings.ToLower(h.title)
	for _, m := range templateTitleMarkers {
		if strings.Contains(lowerTitle, m) {
			out["template_title"] = h.title
		}
	}
	if _, ok := anyFileMatches(c.root, []string{"ui/public", "public"}, []string{"vite.svg"}); ok {
		out["vite_asset"] = "ui/public/vite.svg"
	}
	if c.uiCSSContains("lorem ipsum") || htmlContainsLorem(h.raw) {
		out["lorem_ipsum"] = true
	}
	return out
}

func htmlContainsLorem(raw string) bool {
	return strings.Contains(strings.ToLower(raw), "lorem ipsum")
}
