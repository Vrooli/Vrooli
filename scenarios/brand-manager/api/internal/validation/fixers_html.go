package validation

import (
	"os"
	"path/filepath"
	"strings"

	"brand-manager/internal/brandsurface"
)

// This file holds the self-contained ui/index.html fixers: they derive the
// correct head tags from the scenario's own identity (via brandsurface) or from
// house defaults, inject only what is missing, and are idempotent (a re-run
// finds the tags present and proposes nothing). All gate on their rule firing so
// the advertised AutofixAvailable flag matches what they can actually do.

// injectMissingMetas builds the candidate for injecting the subset of want tags
// that are missing (or empty) in the head. Returns ok=false when nothing is
// missing or there is no index.html.
func injectMissingMetas(root, ruleID, description string, want []brandsurface.Tag, apply bool) (Candidate, bool, error) {
	if !ruleFiresIsolated(root, ruleID) {
		return Candidate{}, false, nil
	}
	content, ok := loadIndexHTML(root)
	if !ok {
		return Candidate{}, false, nil
	}
	h := parseHead(content, true)
	var lines []string
	for _, t := range want {
		if strings.TrimSpace(t.Content) == "" {
			continue // not derivable (e.g. empty description) — leave to a human
		}
		if v, present := h.metaBy(t.Kind, t.Key); present && strings.TrimSpace(v) != "" {
			continue
		}
		lines = append(lines, metaLine(t))
	}
	if len(lines) == 0 {
		return Candidate{}, false, nil
	}
	updated := injectBeforeHeadClose(content, lines)
	cand := Candidate{FilePath: indexHTMLRel, Description: description, Before: content, After: updated}
	if apply {
		if err := writeIndexHTML(root, updated); err != nil {
			return Candidate{}, false, err
		}
		cand.Applied = true
	}
	return cand, true, nil
}

func fixColorScheme(root string, apply bool) (Candidate, bool, error) {
	return injectMissingMetas(root, "color-scheme-declared",
		"Declare color-scheme so the browser themes native controls for the shipped dark mode.",
		[]brandsurface.Tag{{Kind: brandsurface.KindName, Key: "color-scheme", Content: "light dark"}}, apply)
}

func fixStandaloneCapable(root string, apply bool) (Candidate, bool, error) {
	return injectMissingMetas(root, "standalone-capable",
		"Declare the web-app-capable metas so an installed PWA launches chrome-free.",
		[]brandsurface.Tag{
			{Kind: brandsurface.KindName, Key: "mobile-web-app-capable", Content: "yes"},
			{Kind: brandsurface.KindName, Key: "apple-mobile-web-app-capable", Content: "yes"},
		}, apply)
}

func fixOpenGraph(root string, apply bool) (Candidate, bool, error) {
	c := newScanContext("", root)
	return injectMissingMetas(root, "open-graph",
		"Add Open Graph link-preview tags derived from the scenario identity.",
		c.identity().OpenGraphTags(), apply)
}

func fixTwitterCard(root string, apply bool) (Candidate, bool, error) {
	c := newScanContext("", root)
	return injectMissingMetas(root, "twitter-card",
		"Add Twitter card tags derived from the scenario identity.",
		c.identity().TwitterTags(), apply)
}

// fixThemeColorPresent injects a <meta name="theme-color"> when none exists,
// using the manifest theme_color, then the light --color-primary token, then a
// house default. The dark-variant sub-case is left guidance-only (no reliable
// dark color to derive), so this fixer only fires when theme-color is absent.
func fixThemeColorPresent(root string, apply bool) (Candidate, bool, error) {
	if !ruleFiresIsolated(root, "theme-color-present") {
		return Candidate{}, false, nil
	}
	content, ok := loadIndexHTML(root)
	if !ok {
		return Candidate{}, false, nil
	}
	h := parseHead(content, true)
	if len(h.metasByName("theme-color")) > 0 {
		return Candidate{}, false, nil // dark-variant gap — not self-contained
	}
	line := metaLine(brandsurface.Tag{Kind: brandsurface.KindName, Key: "theme-color", Content: defaultThemeColor(root)})
	updated := injectBeforeHeadClose(content, []string{line})
	cand := Candidate{
		FilePath:    indexHTMLRel,
		Description: "Add a theme-color so mobile browser chrome uses the brand color.",
		Before:      content,
		After:       updated,
	}
	if apply {
		if err := writeIndexHTML(root, updated); err != nil {
			return Candidate{}, false, err
		}
		cand.Applied = true
	}
	return cand, true, nil
}

// defaultThemeColor picks the most authoritative existing brand color for the
// chrome: manifest theme_color, then the light --color-primary token, then a
// neutral house default.
func defaultThemeColor(root string) string {
	c := newScanContext("", root)
	if _, obj, _, present := c.manifest(); present {
		if v, ok := obj["theme_color"].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	if vars, ok := c.tokens(); ok {
		if p := strings.TrimSpace(vars["--color-primary"]); p != "" {
			return p
		}
	}
	return "#0f172a"
}

// fixSafeArea injects viewport-fit=cover into the viewport meta and a
// --safe-area-inset-top token into the design tokens, the deterministic half of
// the translucent-status-bar safe-area handling.
func fixSafeArea(root string, apply bool) (Candidate, bool, error) {
	if !ruleFiresIsolated(root, "ios-statusbar-safe-area") {
		return Candidate{}, false, nil
	}
	content, ok := loadIndexHTML(root)
	if !ok {
		return Candidate{}, false, nil
	}
	updated, viewportChanged := addViewportFitCover(content)
	c := newScanContext("", root)
	needsToken := !c.uiCSSContains(safeAreaInsetMarker)
	if !viewportChanged && !needsToken {
		return Candidate{}, false, nil
	}
	cand := Candidate{
		FilePath:    indexHTMLRel,
		Description: "Add viewport-fit=cover and a --safe-area-inset-top token so the translucent status bar does not overlap content.",
		Before:      content,
		After:       updated,
	}
	if apply {
		if viewportChanged {
			if err := writeIndexHTML(root, updated); err != nil {
				return Candidate{}, false, err
			}
		}
		if needsToken {
			if err := appendSafeAreaInsetCSS(root); err != nil {
				return Candidate{}, false, err
			}
		}
		cand.Applied = true
	}
	return cand, true, nil
}

const safeAreaInsetCSS = `
/* brand-manager: iOS safe-area inset for the translucent status bar */
:root {
  --safe-area-inset-top: env(safe-area-inset-top);
  --safe-area-inset-bottom: env(safe-area-inset-bottom);
}
`

func appendSafeAreaInsetCSS(root string) error {
	abs := filepath.Join(root, filepath.FromSlash(designSystemCSSRel))
	existing, _ := readFile(root, designSystemCSSRel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return err
	}
	return os.WriteFile(abs, []byte(existing+safeAreaInsetCSS), 0o600)
}

// fixNameConsistency rewrites every present-but-mismatched name surface to
// service.displayName: the <title>, the application-name / apple title metas, and
// the manifest name/short_name.
func fixNameConsistency(root string, apply bool) (Candidate, bool, error) {
	if !ruleFiresIsolated(root, "name-consistency") {
		return Candidate{}, false, nil
	}
	c := newScanContext("", root)
	id := c.identity()
	want := id.DisplayName

	content, hasHTML := loadIndexHTML(root)
	updated := content
	changed := false
	if hasHTML {
		if h := parseHead(content, true); !strings.EqualFold(h.title, want) && h.title != "" {
			if next, ok := setTitle(updated, want); ok {
				updated, changed = next, true
			}
		}
		for _, key := range []string{"application-name", "apple-mobile-web-app-title"} {
			if next, ok := setMetaContent(updated, brandsurface.KindName, key, want); ok {
				updated, changed = next, true
			}
		}
	}

	manifestChanged, err := alignManifestName(root, want, apply)
	if err != nil {
		return Candidate{}, false, err
	}
	if !changed && !manifestChanged {
		return Candidate{}, false, nil
	}
	cand := Candidate{
		FilePath:    indexHTMLRel,
		Description: "Align the tab title, iOS/app name metas, and manifest name to service.displayName.",
		Before:      content,
		After:       updated,
	}
	if apply && changed {
		if err := writeIndexHTML(root, updated); err != nil {
			return Candidate{}, false, err
		}
		cand.Applied = true
	}
	if apply && manifestChanged {
		cand.Applied = true
	}
	return cand, true, nil
}

// fixTemplateResidue removes the deterministic scaffold residue: a template
// <title> (rewritten to the brand) and the default Vite favicon link + asset.
// Lorem-ipsum copy is left as a manual finding.
func fixTemplateResidue(root string, apply bool) (Candidate, bool, error) {
	if !ruleFiresIsolated(root, "no-template-residue") {
		return Candidate{}, false, nil
	}
	content, ok := loadIndexHTML(root)
	if !ok {
		return Candidate{}, false, nil
	}
	c := newScanContext("", root)
	id := c.identity()
	updated := content
	changed := false

	h := parseHead(content, true)
	if id.HasIdentity() && isTemplateTitle(h.title) {
		if next, ok := setTitle(updated, id.DisplayName); ok {
			updated, changed = next, true
		}
	}
	if next, ok := removeLinkMatching(updated, func(l linkTag) bool {
		return strings.Contains(strings.ToLower(l.href), "vite.svg")
	}); ok {
		updated, changed = next, true
	}
	viteAsset, hasViteAsset := anyFileMatches(root, []string{"ui/public", "public"}, []string{"vite.svg"})
	if !changed && !hasViteAsset {
		return Candidate{}, false, nil // only lorem-ipsum left — not deterministic
	}
	cand := Candidate{
		FilePath:    indexHTMLRel,
		Description: "Remove the default Vite favicon/title scaffold residue.",
		Before:      content,
		After:       updated,
	}
	if apply {
		if changed {
			if err := writeIndexHTML(root, updated); err != nil {
				return Candidate{}, false, err
			}
		}
		if hasViteAsset {
			_ = os.Remove(filepath.Join(root, filepath.FromSlash(viteAsset)))
		}
		cand.Applied = true
	}
	return cand, true, nil
}

func isTemplateTitle(title string) bool {
	lower := strings.ToLower(title)
	for _, m := range templateTitleMarkers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}
