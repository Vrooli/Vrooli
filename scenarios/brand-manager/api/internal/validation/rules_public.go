package validation

import (
	"os"
	"path/filepath"
	"strings"

	"brand-manager/internal/brandsurface"
)

// This file holds the /public/ URL-convention rule. The fleet convention
// (docs/concepts/PUBLIC_ASSETS.md) is that anything served under the URL prefix
// /public/ is publicly fetchable by anonymous clients — iOS Add-to-Home-Screen,
// link unfurlers, Open Graph crawlers — so an edge Cloudflare Access bypass
// scoped to <host>/public can serve a scenario's branding/PWA/OG assets without
// weakening Access on the rest of the app. This rule asserts that a scenario's
// public branding assets, and the index.html/manifest references to them, live
// under that convention.
//
// It is advisory (warning, never gating), consistent with the rest of the
// branding rule set: a scenario serving branding at the site root still works
// behind Access for authenticated users; the convention is what makes those
// assets resolve for anonymous system fetchers. The matching fixer
// (fixers_public.go) relocates the files + repoints the references, so the
// advertised AutofixAvailable flag matches what it can deterministically do.

// publicConventionOffenders is the shared detection both the rule and its fixer
// consume (so "advertised == implemented" is true by construction): the
// branding-asset files served at the publicDir root and the index.html/manifest
// references that point outside /public/.
type publicConventionOffenders struct {
	rootFiles []string // scenario-relative file paths, e.g. "ui/public/favicon.svg"
	rootRefs  []string // offending root-absolute URLs, e.g. "/favicon.svg"
}

func (o publicConventionOffenders) empty() bool {
	return len(o.rootFiles) == 0 && len(o.rootRefs) == 0
}

// detectPublicConventionOffenders scans the scenario tree for branding assets
// and references that violate the /public/ convention. It reads the tree fresh
// (no scan cache) so the fixer, which re-detects after writing, sees a
// consistent view.
func detectPublicConventionOffenders(root string) publicConventionOffenders {
	var o publicConventionOffenders

	// (a) Branding-asset files served at the publicDir root (URL "/<base>"). Only
	// the immediate children are considered: anything already nested under
	// ui/public/public/ is, by convention, correct.
	base := filepath.Join(root, filepath.FromSlash(brandsurface.RootAssetSourceDir))
	entries, _ := os.ReadDir(base)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if brandsurface.IsPublicBrandingAsset(e.Name()) {
			o.rootFiles = append(o.rootFiles, brandsurface.RootAssetSourceDir+"/"+e.Name())
		}
	}

	// (b) index.html + manifest references to branding assets not under /public/.
	seen := map[string]bool{}
	addRef := func(ref string) {
		ref = strings.TrimSpace(ref)
		if !isRootAbsoluteRef(ref) || brandsurface.IsUnderPublicPrefix(ref) {
			return
		}
		if !brandsurface.IsPublicBrandingAsset(filepath.Base(ref)) {
			return
		}
		if !seen[ref] {
			seen[ref] = true
			o.rootRefs = append(o.rootRefs, ref)
		}
	}

	c := newScanContext("", root)
	h := c.head()
	for _, rel := range []string{"icon", "shortcut icon", "apple-touch-icon", "mask-icon", "manifest"} {
		addRef(linkHref(h, rel))
	}
	if v, ok := h.metaByProperty("og:image"); ok {
		addRef(v)
	}
	if v, ok := h.metaByName("twitter:image"); ok {
		addRef(v)
	}
	if _, obj, _, present := c.manifest(); present {
		if icons, ok := obj["icons"].([]any); ok {
			for _, ic := range icons {
				if m, ok := ic.(map[string]any); ok {
					if src, _ := m["src"].(string); src != "" {
						addRef(src)
					}
				}
			}
		}
	}
	return o
}

// isRootAbsoluteRef reports whether ref is a local root-absolute URL path (starts
// with "/", not an absolute URL or data: URI). Relative references resolve
// against the document/manifest location and so are convention-neutral here.
func isRootAbsoluteRef(ref string) bool {
	ref = strings.TrimSpace(ref)
	return strings.HasPrefix(ref, "/") && !strings.Contains(ref, "://") && !strings.HasPrefix(ref, "data:")
}

func rulePublicAssetConvention(c *scanContext) (Finding, bool) {
	o := detectPublicConventionOffenders(c.root)
	if o.empty() {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityWarning,
		Title:                  "Public branding assets are not served under the /public/ convention",
		Description:            "Branding/PWA/OG assets — or the index.html/manifest references to them — are served at the site root instead of under the /public/ URL prefix.",
		FilePath:               brandsurface.PublicAssetSourceDir,
		WhyItMatters:           "Only assets under /public/ can be exempted from Cloudflare Access at the edge, so anonymous fetchers (iOS Add-to-Home-Screen, link/OG crawlers) load the real icon and manifest instead of a login page.",
		RecommendedRemediation: "Relocate the assets under ui/public/public/ (served at /public/) and repoint the index.html links + manifest references at /public/.",
		Evidence:               map[string]any{"root_files": o.rootFiles, "root_refs": o.rootRefs},
	}, true
}
