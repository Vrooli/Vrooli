package validation

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// This file holds brand-asset quality checks that go beyond mere presence:
// referenced icons must be non-empty, the apple-touch-icon must be opaque (iOS
// composites alpha onto black), and a maskable manifest icon should be declared.
// It is detect-only: re-encoding/flattening a PNG is an image transform left to
// the brand-projection/apply path, so this rule never advertises an autofix.

func ruleAssetValidity(c *scanContext) (Finding, bool) {
	issues := assetIssues(c)
	if len(issues) == 0 {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityWarning,
		Title:                  "Brand asset quality issues",
		Description:            "A referenced brand asset is empty, transparent where it must be opaque, or the icon set lacks a maskable entry.",
		FilePath:               "ui/public",
		WhyItMatters:           "iOS fills a transparent apple-touch-icon with black, empty assets break tabs/installs, and a missing maskable icon is cropped on Android.",
		RecommendedRemediation: "Flatten the apple-touch-icon onto a solid background, replace empty assets, and add a maskable icon.",
		Evidence:               map[string]any{"issues": issues},
	}, true
}

// assetIssues returns the detected per-asset problems (stable keys).
func assetIssues(c *scanContext) map[string]any {
	out := map[string]any{}
	h := c.head()

	// apple-touch-icon opacity + emptiness.
	if l, ok := h.linkByRel("apple-touch-icon"); ok {
		if abs, exists := resolvePublicAsset(c.root, l.href); exists {
			if isEmptyFile(abs) {
				out["apple_touch_icon_empty"] = l.href
			} else if hasTransparency(abs) {
				out["apple_touch_icon_transparent"] = l.href
			}
		}
	}

	// favicon emptiness (a zero-byte favicon renders as a broken tab mark).
	if l, ok := h.linkByRel("icon"); ok {
		if abs, exists := resolvePublicAsset(c.root, l.href); exists && isEmptyFile(abs) {
			out["favicon_empty"] = l.href
		}
	}

	// Manifest icon set must include a maskable entry when icons are declared.
	if _, obj, _, present := c.manifest(); present {
		if icons, ok := obj["icons"].([]any); ok && len(icons) > 0 && !manifestHasMaskableIcon(obj) {
			out["no_maskable_icon"] = true
		}
	}
	return out
}

// resolvePublicAsset maps an href (e.g. "/apple-icon-180.png") to a file under
// the scenario's public dir and reports whether it exists. External URLs and
// data: URIs resolve to not-exists (nothing local to inspect).
func resolvePublicAsset(root, href string) (string, bool) {
	href = strings.TrimSpace(href)
	if href == "" || strings.Contains(href, "://") || strings.HasPrefix(href, "data:") {
		return "", false
	}
	rel := strings.TrimPrefix(href, "/")
	for _, base := range []string{"ui/public", "ui/public/public", "public", "ui"} {
		abs := filepath.Join(root, filepath.FromSlash(base), filepath.FromSlash(rel))
		if info, err := os.Stat(abs); err == nil && !info.IsDir() {
			return abs, true
		}
	}
	return "", false
}

func isEmptyFile(abs string) bool {
	info, err := os.Stat(abs)
	return err == nil && info.Size() == 0
}

// hasTransparency reports whether a PNG has any non-opaque pixel. A decode
// failure or a non-PNG returns false (treated as a manual concern, never a
// crash), per the plan's risk mitigation.
func hasTransparency(abs string) bool {
	if strings.ToLower(filepath.Ext(abs)) != ".png" {
		return false
	}
	f, err := os.Open(abs)
	if err != nil {
		return false
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return false
	}
	if _, ok := img.(*image.YCbCr); ok {
		return false // no alpha channel by construction
	}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a < 0xffff {
				return true
			}
		}
	}
	return false
}
