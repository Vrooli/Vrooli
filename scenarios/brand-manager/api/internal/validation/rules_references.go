package validation

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// This file holds referential integrity for declared brand assets. The presence
// rules (has-favicon, manifest-completeness) check that a scenario DECLARES its
// icons/manifest; this rule checks that every LOCAL asset a scenario points at
// actually resolves to a real, non-empty file on disk — a manifest icon, a head
// icon link, or a social-preview image. A dangling reference renders as a broken
// icon/tab/link-preview while every presence rule still reports green.
//
// It is detect-only: the fix is to ship the missing bytes (an image we cannot
// synthesize deterministically), so it never advertises an autofix. Emptiness of
// the favicon/apple-touch-icon is owned by asset-validity, so this rule only
// checks EXISTENCE for those two surfaces to avoid double-reporting; for surfaces
// nothing else covers (manifest icons, mask-icon, social images) it also checks
// emptiness, and for manifest PNG icons it best-effort verifies the decoded
// dimensions match the declared `sizes`.

func ruleReferencedAssetsExist(c *scanContext) (Finding, bool) {
	issues := referencedAssetIssues(c)
	if len(issues) == 0 {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityWarning,
		Title:                  "Referenced brand assets are missing or empty",
		Description:            "A manifest icon, head icon link, or social-preview image points at a file that does not exist (or is empty) on disk.",
		FilePath:               "ui/public",
		WhyItMatters:           "A dangling asset reference renders as a broken icon, tab mark, or link preview even though the scenario reports the reference as present.",
		RecommendedRemediation: "Ship the referenced asset files (or correct the paths / declared sizes) so every declared reference resolves.",
		Evidence:               map[string]any{"issues": issues},
	}, true
}

// referencedAssetIssues returns the detected per-reference problems (stable keys).
func referencedAssetIssues(c *scanContext) map[string]any {
	out := map[string]any{}
	h := c.head()

	// Head icon links. icon/shortcut icon/apple-touch-icon emptiness is owned by
	// asset-validity, so here they are existence-only; mask-icon is checked fully.
	checkRef(c.root, "link:icon", linkHref(h, "icon"), out, false)
	checkRef(c.root, "link:shortcut-icon", linkHref(h, "shortcut icon"), out, false)
	checkRef(c.root, "link:apple-touch-icon", linkHref(h, "apple-touch-icon"), out, false)
	checkRef(c.root, "link:mask-icon", linkHref(h, "mask-icon"), out, true)

	// Social preview images — not covered by any other rule.
	for _, m := range h.metasByProperty("og:image") {
		checkRef(c.root, nextRefKey(out, "meta:og:image"), m.content, out, true)
	}
	for _, m := range h.metasByName("twitter:image") {
		checkRef(c.root, nextRefKey(out, "meta:twitter:image"), m.content, out, true)
	}

	// Manifest icons: existence + emptiness + best-effort dimension match.
	if _, obj, _, present := c.manifest(); present {
		if icons, ok := obj["icons"].([]any); ok {
			for _, ic := range icons {
				m, ok := ic.(map[string]any)
				if !ok {
					continue
				}
				src, _ := m["src"].(string)
				if !isLocalRef(src) {
					continue
				}
				abs, exists := resolvePublicAsset(c.root, src)
				key := "manifest:" + strings.TrimSpace(src)
				switch {
				case !exists:
					out[key] = "missing"
				case isEmptyFile(abs):
					out[key] = "empty"
				default:
					if sizes, _ := m["sizes"].(string); sizes != "" {
						if mm := iconDimensionMismatch(abs, sizes); mm != "" {
							out[key] = mm
						}
					}
				}
			}
		}
	}
	return out
}

func nextRefKey(out map[string]any, base string) string {
	if _, exists := out[base]; !exists {
		return base
	}
	for i := 1; ; i++ {
		key := fmt.Sprintf("%s[%d]", base, i)
		if _, exists := out[key]; !exists {
			return key
		}
	}
}

// linkHref returns the href of the first <link rel=rel>, or "" when absent.
func linkHref(h *headDoc, rel string) string {
	if l, ok := h.linkByRel(rel); ok {
		return l.href
	}
	return ""
}

// checkRef records a missing (and optionally empty) local asset reference. A
// blank or external (http(s)/data:) reference is skipped — there is nothing
// local to inspect.
func checkRef(root, label, ref string, out map[string]any, checkEmpty bool) {
	if !isLocalRef(ref) {
		return
	}
	abs, exists := resolvePublicAsset(root, ref)
	if !exists {
		out[label] = "missing"
		return
	}
	if checkEmpty && isEmptyFile(abs) {
		out[label] = "empty"
	}
}

// isLocalRef reports whether ref is a local path worth resolving on disk
// (non-empty, not an absolute URL, not a data: URI).
func isLocalRef(ref string) bool {
	ref = strings.TrimSpace(ref)
	return ref != "" && !strings.Contains(ref, "://") && !strings.HasPrefix(ref, "data:")
}

// iconDimensionMismatch best-effort verifies a PNG icon's real dimensions match
// the first WxH token in its declared `sizes`. It returns "" (no finding) for a
// non-PNG, an undecodable file, a "any"/unparseable size, or a match — so it only
// reports a genuine, provable mismatch.
func iconDimensionMismatch(abs, sizes string) string {
	fields := strings.Fields(sizes)
	if len(fields) == 0 {
		return ""
	}
	first := strings.ToLower(fields[0])
	if first == "any" {
		return ""
	}
	wh := strings.SplitN(first, "x", 2)
	if len(wh) != 2 {
		return ""
	}
	dw, err1 := strconv.Atoi(wh[0])
	dh, err2 := strconv.Atoi(wh[1])
	if err1 != nil || err2 != nil {
		return ""
	}
	if strings.ToLower(filepath.Ext(abs)) != ".png" {
		return ""
	}
	f, err := os.Open(abs)
	if err != nil {
		return ""
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		return ""
	}
	if cfg.Width != dw || cfg.Height != dh {
		return fmt.Sprintf("dimension_mismatch declared=%dx%d actual=%dx%d", dw, dh, cfg.Width, cfg.Height)
	}
	return ""
}
