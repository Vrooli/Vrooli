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

// RequiredManifestKeys are the manifest fields a complete, installable manifest
// must declare. theme_color/background_color come from the visual system, not
// identity, so they are required-present here but their value is owned by the
// theme-color rules.
var RequiredManifestKeys = []string{
	"name", "short_name", "description",
	"theme_color", "background_color",
	"display", "start_url", "id", "icons",
}
