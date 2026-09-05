package validation

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// This file holds the webfont-loading rule. A --font-* token can name a custom
// family (e.g. "Inter") that the browser silently falls back from unless the
// scenario actually loads it. The rule fires (info) only in the conservative case
// that is almost always a real bug: a token's PRIMARY family is non-system AND no
// font-loading mechanism (an @font-face, a Google-Fonts / webfont <link>, or a
// bundled font file) exists anywhere in the scenario's own sources.
//
// It is detect-only: font selection is a design choice and fetching a typeface is
// not a deterministic edit, so no autofix is advertised. The system-font set
// below is the one deliberately-opinionated input — families on it need no shipped
// file; anything else is treated as custom.

// systemFontFamilies are the CSS generic keywords and ubiquitous platform/web-safe
// families that require no shipped font file. Compared case-insensitively against
// the dequoted primary family of each --font-* stack.
var systemFontFamilies = map[string]bool{
	// CSS generics + global keywords.
	"sans-serif": true, "serif": true, "monospace": true, "cursive": true,
	"fantasy": true, "system-ui": true, "ui-sans-serif": true, "ui-serif": true,
	"ui-monospace": true, "ui-rounded": true, "math": true, "emoji": true,
	"inherit": true, "initial": true, "unset": true, "revert": true,
	// Platform UI + web-safe families.
	"-apple-system": true, "blinkmacsystemfont": true, "segoe ui": true,
	"roboto": true, "oxygen": true, "ubuntu": true, "cantarell": true,
	"fira sans": true, "droid sans": true, "helvetica neue": true,
	"helvetica": true, "arial": true, "tahoma": true, "verdana": true,
	"georgia": true, "times new roman": true, "times": true,
	"sfmono-regular": true, "menlo": true, "monaco": true, "consolas": true,
	"liberation mono": true, "courier new": true, "courier": true,
	"apple color emoji": true, "segoe ui emoji": true, "noto color emoji": true,
}

func ruleCustomFontLoaded(c *scanContext) (Finding, bool) {
	vars, ok := c.tokens()
	if !ok {
		return Finding{}, false // no tokens — has-color-system / has-typography own that
	}
	customSet := map[string]bool{}
	for name, val := range vars {
		if !strings.HasPrefix(name, "--font-") {
			continue
		}
		fam := primaryFontFamily(val)
		if fam == "" || systemFontFamilies[fam] {
			continue
		}
		customSet[fam] = true
	}
	if len(customSet) == 0 {
		return Finding{}, false
	}
	if fontLoadSignalPresent(c) {
		return Finding{}, false
	}
	custom := make([]string, 0, len(customSet))
	for f := range customSet {
		custom = append(custom, f)
	}
	sort.Strings(custom)
	return Finding{
		Severity:               SeverityInfo,
		Title:                  "Custom font is referenced but never loaded",
		Description:            "A --font-* token names a custom family, but no @font-face, webfont <link>, or bundled font file loads it — the browser silently falls back.",
		FilePath:               designSystemCSSRel,
		WhyItMatters:           "An unloaded custom font means the rendered type silently differs from the brand's intended typeface (FOUT / wrong fallback).",
		RecommendedRemediation: "Load the font (self-host an @font-face + bundled woff2, or add a Google-Fonts <link>) or use a system-font stack.",
		Evidence:               map[string]any{"unloaded_families": custom},
	}, true
}

// primaryFontFamily returns the first (dequoted, lowercased) family in a CSS font
// stack — the family the browser actually tries first.
func primaryFontFamily(stack string) string {
	first := stack
	if i := strings.IndexByte(stack, ','); i >= 0 {
		first = stack[:i]
	}
	first = strings.TrimSpace(first)
	first = strings.Trim(first, `"'`)
	return strings.ToLower(strings.TrimSpace(first))
}

var googleFontsRe = regexp.MustCompile(`(?i)fonts\.(googleapis|gstatic)\.com`)

// fontLoadSignalPresent reports whether the scenario loads webfonts by any common
// mechanism: a Google-Fonts reference in the head or app CSS, an @font-face
// declaration, or a bundled font file.
func fontLoadSignalPresent(c *scanContext) bool {
	if googleFontsRe.MatchString(c.head().raw) {
		return true
	}
	m := c.appCSSMatches([]string{"@font-face", "fonts.googleapis.com", "fonts.gstatic.com"})
	if m["@font-face"] || m["fonts.googleapis.com"] || m["fonts.gstatic.com"] {
		return true
	}
	return fontFilePresent(c.root)
}

var fontFileExts = map[string]bool{".woff": true, ".woff2": true, ".ttf": true, ".otf": true, ".eot": true}

// fontFilePresent reports whether any bundled font file lives in a brand asset
// directory (a self-hosted webfont).
func fontFilePresent(root string) bool {
	found := false
	for _, dir := range brandAssetDirs {
		base := filepath.Join(root, filepath.FromSlash(dir))
		_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil || found {
				return nil
			}
			if d.IsDir() {
				if skipScanDir(d.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if fontFileExts[strings.ToLower(filepath.Ext(path))] {
				found = true
			}
			return nil
		})
		if found {
			break
		}
	}
	return found
}
