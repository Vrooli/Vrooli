// Package brandsurface is the single source of truth (SSOT) for the canonical
// branding "surface" a scenario should project from its own identity: the
// expected <head> meta tags, <title>, manifest scalar fields, and social/link
// preview metadata.
//
// Both the validation rules (does the actual surface match the projection?) and
// the deterministic fixers (re-project the expected surface) consume this one
// package, so "advertised" and "applied" expected values can never drift —
// there is exactly one definition of correct.
//
// The projector is intentionally pure and self-contained: it derives everything
// from the scenario's OWN data (.vrooli/service.json). A richer brand-projection
// path (palette tokens, generated artwork from an assigned BrandView) lives in
// the apply domain; this package only models what is derivable without a brand
// assignment, which is exactly what test-genie's brandless ApplyFix can write.
package brandsurface

import (
	"encoding/json"
	"path"
	"strings"
)

// TagKind distinguishes a `<meta name=...>` tag from a `<meta property=...>` tag
// (Open Graph uses property; most others use name).
type TagKind string

const (
	// KindName is a `<meta name="..." content="...">` tag.
	KindName TagKind = "name"
	// KindProperty is a `<meta property="..." content="...">` tag (Open Graph).
	KindProperty TagKind = "property"
)

// Tag is one expected `<meta>` tag in the projected head.
type Tag struct {
	Kind    TagKind
	Key     string
	Content string
}

// Surface is a scenario's branding identity, parsed from its service.json.
type Surface struct {
	// Slug is the raw scenario id (service.name), e.g. "widget-shop".
	Slug string
	// DisplayName is the human brand name (service.displayName), e.g. "Widget Shop".
	DisplayName string
	// Description is the one-line product description (service.description).
	Description string
}

// ParseService extracts the branding identity from a .vrooli/service.json
// document. It tolerates malformed JSON by returning a best-effort Surface
// (empty fields), so callers can still reason about what is missing.
func ParseService(serviceJSON string) Surface {
	var svc struct {
		Service struct {
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
			Description string `json:"description"`
		} `json:"service"`
	}
	_ = json.Unmarshal([]byte(serviceJSON), &svc)
	return Surface{
		Slug:        strings.TrimSpace(svc.Service.Name),
		DisplayName: strings.TrimSpace(svc.Service.DisplayName),
		Description: strings.TrimSpace(svc.Service.Description),
	}
}

// HasIdentity reports whether the surface carries a usable display name (the
// minimum needed to project consistent branding). A slug-equal or empty display
// name is not a usable identity.
func (s Surface) HasIdentity() bool {
	d := s.DisplayName
	return d != "" && !strings.EqualFold(d, s.Slug) && !strings.Contains(d, "[")
}

// Title is the expected document <title> and the value every name-bearing
// surface (application-name, apple title, manifest name) must agree with.
func (s Surface) Title() string { return s.DisplayName }

// ConsistencyTags are the `<meta name=...>` tags whose content must equal the
// display name for cross-surface name consistency.
func (s Surface) ConsistencyTags() []Tag {
	return []Tag{
		{Kind: KindName, Key: "application-name", Content: s.DisplayName},
		{Kind: KindName, Key: "apple-mobile-web-app-title", Content: s.DisplayName},
	}
}

// OpenGraphTags are the expected Open Graph link-preview tags derived purely
// from the scenario's own identity. og:image is intentionally omitted — pointing
// it at a real asset needs an existing image and so is decided by the caller.
func (s Surface) OpenGraphTags() []Tag {
	return []Tag{
		{Kind: KindProperty, Key: "og:type", Content: "website"},
		{Kind: KindProperty, Key: "og:title", Content: s.DisplayName},
		{Kind: KindProperty, Key: "og:description", Content: s.Description},
		{Kind: KindProperty, Key: "og:site_name", Content: s.DisplayName},
	}
}

// TwitterTags are the expected Twitter (X) summary-card tags, mirroring OG.
func (s Surface) TwitterTags() []Tag {
	return []Tag{
		{Kind: KindName, Key: "twitter:card", Content: "summary_large_image"},
		{Kind: KindName, Key: "twitter:title", Content: s.DisplayName},
		{Kind: KindName, Key: "twitter:description", Content: s.Description},
	}
}

// ManifestScalars are the expected scalar (non-icon) web-app-manifest fields,
// derived from the scenario's identity. Icon entries are out of scope here
// because they require real image assets. start_url/scope/display/id are house
// defaults for an installable PWA.
func (s Surface) ManifestScalars() map[string]string {
	return map[string]string{
		"name":        s.DisplayName,
		"short_name":  s.DisplayName,
		"description": s.Description,
		"display":     "standalone",
		"start_url":   ".",
		"scope":       ".",
		"id":          "/",
	}
}

// --- /public/ asset convention (SSOT) --------------------------------------

// The fleet "/public/*" convention: anything served under the URL path prefix
// /public/ is publicly fetchable by anonymous clients (iOS Add-to-Home-Screen,
// link unfurlers, Open Graph crawlers). The contract is the URL prefix, NOT any
// framework's source-dir name. For a Vite scenario the publicDir (ui/public)
// serves at URL "/", so nesting a "public" directory inside it
// (ui/public/public) serves those files under "/public/". An edge Cloudflare
// Access bypass scoped to <host>/public can then serve a scenario's branding/
// PWA/OG assets to anonymous fetchers without weakening Access on the rest of
// the app. Both the convention rule and its fixer read these definitions, so
// "what is a public asset" and "where it belongs" have exactly one source.
const (
	// PublicURLPrefix is the canonical public URL path prefix (with trailing
	// slash) under which assets are publicly fetchable by convention.
	PublicURLPrefix = "/public/"
	// RootAssetSourceDir is the Vite publicDir whose files serve at URL root "/".
	RootAssetSourceDir = "ui/public"
	// PublicAssetSourceDir is the directory inside the publicDir whose files
	// serve under PublicURLPrefix (the intentional ui/public/public nesting).
	PublicAssetSourceDir = "ui/public/public"
)

// publicBrandingAssetGlobs are the filename globs for the branding/PWA/OG assets
// that anonymous fetchers load (favicon, apple-touch-icon, PWA icons, the web
// manifest, social-preview images) plus the brand logo — i.e. everything that
// belongs under the /public/ convention so it stays world-readable behind an
// Access-gated app. Matched against a lower-cased basename.
var publicBrandingAssetGlobs = []string{
	"favicon.ico", "favicon.svg", "favicon-*.png", "favicon-*.svg", "favicon.png",
	"logo.svg", "logo.png", "logo-*.svg", "logo-*.png", "*-logo.svg", "*-logo.png",
	"apple-touch-icon.png", "apple-touch-icon-*.png", "apple-icon-*.png",
	"icon-*.png", "icon-*.svg", "manifest-icon-*.png", "maskable-*.png", "maskable.png",
	"site.webmanifest", "manifest.json", "manifest.webmanifest",
	"og-image.*", "og-*.png", "og.png", "*-og.png",
	"twitter-image.*", "twitter-*.png",
}

// IsPublicBrandingAsset reports whether base (a file basename) is a branding/PWA/
// OG asset that belongs under the /public/ convention.
func IsPublicBrandingAsset(base string) bool {
	base = strings.ToLower(strings.TrimSpace(base))
	if base == "" {
		return false
	}
	for _, g := range publicBrandingAssetGlobs {
		if ok, _ := path.Match(g, base); ok {
			return true
		}
	}
	return false
}

// IsUnderPublicPrefix reports whether a root-absolute URL path is served under
// the public convention prefix.
func IsUnderPublicPrefix(urlPath string) bool {
	return strings.HasPrefix(strings.TrimSpace(urlPath), PublicURLPrefix)
}

// PublicURLForRootRef rewrites a root-absolute asset URL ("/favicon.svg") to its
// /public/ equivalent ("/public/favicon.svg"). A path already under the prefix is
// returned unchanged.
func PublicURLForRootRef(urlPath string) string {
	urlPath = strings.TrimSpace(urlPath)
	if IsUnderPublicPrefix(urlPath) {
		return urlPath
	}
	return PublicURLPrefix + strings.TrimPrefix(urlPath, "/")
}

// RequiredManifestKeys are the manifest fields a complete, installable manifest
// must declare. theme_color/background_color come from the visual system, not
// identity, so they are required-present here but their value is owned by the
// theme-color rules.
var RequiredManifestKeys = []string{
	"name", "short_name", "description",
	"theme_color", "background_color",
	"display", "start_url", "id", "icons",
}
