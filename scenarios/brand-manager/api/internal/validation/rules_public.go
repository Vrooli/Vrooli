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

// publicRefFix is one reference that is not in its canonical public form.
type publicRefFix struct {
	From string
	To   string
	// Attr is the index.html attribute the reference lives in ("href" for
	// head links, "content" for og/twitter images). One asset can be named by
	// both with different canonical forms, so a rewrite must stay within it.
	Attr string
}

// publicConventionOffenders is the shared detection both the rule and its fixer
// consume (so "advertised == implemented" is true by construction): branding
// asset files served at the publicDir root, references that are not in their
// canonical public form, and a public manifest whose launch target is not the
// app root.
type publicConventionOffenders struct {
	rootFiles      []string       // scenario-relative file paths, e.g. "ui/public/favicon.svg"
	headRefs       []publicRefFix // index.html link/meta references, e.g. "/favicon.svg" -> "public/favicon.svg"
	manifestIcons  []publicRefFix // manifest icon srcs, e.g. "/public/icon-192.png" -> "icon-192.png"
	manifestLaunch bool           // a manifest under /public/ whose start_url/scope is not ManifestAppRoot
}

func (o publicConventionOffenders) empty() bool {
	return len(o.rootFiles) == 0 && len(o.headRefs) == 0 && len(o.manifestIcons) == 0 && !o.manifestLaunch
}

// refs lists every non-canonical reference, for evidence and previews.
func (o publicConventionOffenders) refs() []string {
	out := make([]string, 0, len(o.headRefs)+len(o.manifestIcons))
	for _, fix := range append(append([]publicRefFix(nil), o.headRefs...), o.manifestIcons...) {
		out = append(out, fix.From)
	}
	return out
}

// detectPublicConventionOffenders scans the scenario tree for branding assets
// and references that violate the /public/ convention
// (docs/concepts/PUBLIC_ASSETS.md §"Adopter requirements"): head links are
// document-relative under public/, og:image and twitter:image keep the
// absolute /public/ path, manifest icon srcs are relative to the manifest, and
// a manifest under /public/ launches at "../". It reads the tree fresh (no scan
// cache) so the fixer, which re-detects after writing, sees a consistent view.
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

	// (b) index.html references to branding assets not in canonical form. The
	// link's rel (or og/twitter property) already says the target is branding,
	// so a custom filename such as "<scenario>-icon.svg" counts as much as a
	// conventional one. A root-served target joins the relocation set so its
	// repointed link lands on a real file.
	seen := map[string]bool{}
	relocating := map[string]bool{}
	for _, rel := range o.rootFiles {
		relocating[rel] = true
	}
	addHead := func(ref, attr string, canonical func(string) string) {
		ref = strings.TrimSpace(ref)
		if !isLocalAssetRef(ref) {
			return
		}
		rootFile, placeable := publicPlacement(root, ref)
		if !placeable {
			return
		}
		if rootFile != "" && !relocating[rootFile] {
			relocating[rootFile] = true
			o.rootFiles = append(o.rootFiles, rootFile)
		}
		if want := canonical(ref); want != ref && !seen[attr+" "+ref] {
			seen[attr+" "+ref] = true
			o.headRefs = append(o.headRefs, publicRefFix{From: ref, To: want, Attr: attr})
		}
	}
	c := newScanContext("", root)
	h := c.head()
	// Every branding link counts: a head usually carries several icon links.
	brandingRels := map[string]bool{"icon": true, "shortcut icon": true, "apple-touch-icon": true, "mask-icon": true, "manifest": true}
	for _, link := range h.links {
		if brandingRels[strings.ToLower(strings.TrimSpace(link.rel))] {
			addHead(link.href, "href", brandsurface.PublicHeadRef)
		}
	}
	if v, ok := h.metaByProperty("og:image"); ok {
		addHead(v, "content", brandsurface.PublicSocialImageURL)
	}
	if v, ok := h.metaByName("twitter:image"); ok {
		addHead(v, "content", brandsurface.PublicSocialImageURL)
	}

	// (c) The manifest: root-absolute icon srcs, and the launch target once it
	// lives under /public/.
	if rel, obj, _, present := c.manifest(); present {
		if icons, ok := obj["icons"].([]any); ok {
			for _, ic := range icons {
				m, ok := ic.(map[string]any)
				if !ok {
					continue
				}
				if src, _ := m["src"].(string); isRootAbsoluteRef(src) {
					o.manifestIcons = append(o.manifestIcons, publicRefFix{From: src, To: brandsurface.PublicManifestIconSrc(src)})
				}
			}
		}
		if strings.HasPrefix(rel, brandsurface.PublicAssetSourceDir+"/") {
			for _, key := range []string{"start_url", "scope"} {
				if v, _ := obj[key].(string); v != brandsurface.ManifestAppRoot {
					o.manifestLaunch = true
				}
			}
		}
	}
	return o
}

// publicPlacement reports whether the convention can place a branding
// reference on the public surface, and the root-served file that must move
// there first, if any. The target is placeable when it already lives under
// ui/public/public, when it is a file served at the publicDir root (it is
// relocated), or when its name is a conventional branding asset (the link is
// canonicalised even though the file is missing, so the finding stays visible).
// Anything else — a bundled /assets/ path, say — is not the convention's to
// move, and rewriting its link would break it.
func publicPlacement(root, ref string) (rootFile string, placeable bool) {
	assetPath := brandsurface.PublicAssetPath(ref)
	if assetPath == "" {
		return "", false
	}
	if regularFileExists(filepath.Join(root, filepath.FromSlash(brandsurface.PublicAssetSourceDir), filepath.FromSlash(assetPath))) {
		return "", true
	}
	if !strings.Contains(assetPath, "/") && regularFileExists(filepath.Join(root, filepath.FromSlash(brandsurface.RootAssetSourceDir), assetPath)) {
		return brandsurface.RootAssetSourceDir + "/" + assetPath, true
	}
	return "", brandsurface.IsPublicBrandingAsset(filepath.Base(assetPath))
}

func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// isLocalAssetRef reports whether ref is a non-empty local reference: not an
// absolute URL, protocol-relative URL or data: URI.
func isLocalAssetRef(ref string) bool {
	ref = strings.TrimSpace(ref)
	return ref != "" && !strings.Contains(ref, "://") && !strings.HasPrefix(ref, "//") && !strings.HasPrefix(ref, "data:")
}

// isRootAbsoluteRef reports whether ref is a local root-absolute URL path (starts
// with "/", not an absolute URL or data: URI).
func isRootAbsoluteRef(ref string) bool {
	ref = strings.TrimSpace(ref)
	return strings.HasPrefix(ref, "/") && isLocalAssetRef(ref)
}

func rulePublicAssetConvention(c *scanContext) (Finding, bool) {
	o := detectPublicConventionOffenders(c.root)
	if o.empty() {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityWarning,
		Title:                  "Public branding assets do not follow the /public/ convention",
		Description:            "Branding/PWA/OG assets are served at the site root, or their index.html/manifest references are not in the canonical public form.",
		FilePath:               brandsurface.PublicAssetSourceDir,
		WhyItMatters:           "Only assets under /public/ can be exempted from Cloudflare Access at the edge, so anonymous fetchers (iOS Add-to-Home-Screen, link/OG crawlers) load the real icon and manifest instead of a login page. Relative head links reach /public/ from the host root and stay correct behind a path-prefix proxy.",
		RecommendedRemediation: "Relocate the assets under ui/public/public/, point head links at public/... (og:image and twitter:image at /public/...), keep manifest icon srcs relative, and set the manifest start_url and scope to ../.",
		Evidence:               map[string]any{"root_files": o.rootFiles, "root_refs": o.refs(), "manifest_launch": o.manifestLaunch},
	}, true
}
