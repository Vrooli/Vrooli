// Package profiles is the single source of truth for what an icon target
// profile means: the file, format, exact size, variant and wiring of every icon
// a scenario ships. Render, apply and validation all read this package, so a
// size is never hard-coded twice.
package profiles

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Variant is how a mark is composed for a target.
type Variant string

const (
	// VariantRounded is the mark on a rounded tile.
	VariantRounded Variant = "rounded"
	// VariantFullBleed is an opaque square ground.
	VariantFullBleed Variant = "full_bleed"
	// VariantMaskable keeps the mark inside the W3C safe zone.
	VariantMaskable Variant = "maskable"
	// VariantSocialCard is the wide Open Graph card.
	VariantSocialCard Variant = "social_card"
	// VariantVector is the SVG mark passed through unchanged.
	VariantVector Variant = "vector"
)

// Wiring is how a target is referenced from the scenario document.
type Wiring string

const (
	WiringLinkIcon         Wiring = "link_icon"
	WiringLinkAppleTouch   Wiring = "link_apple_touch"
	WiringManifestAny      Wiring = "manifest_any"
	WiringManifestMaskable Wiring = "manifest_maskable"
	WiringManifestFile     Wiring = "manifest_file"
	WiringMetaOG           Wiring = "meta_og"
	WiringNone             Wiring = "none"
)

// Target is one file a scenario must ship.
type Target struct {
	ID               string  `json:"id"`
	Path             string  `json:"path"`
	Format           string  `json:"format"` // svg|png|ico|icns|json
	Width            int     `json:"width,omitempty"`
	Height           int     `json:"height,omitempty"`
	Variant          Variant `json:"variant,omitempty"`
	SmallMarkAllowed bool    `json:"small_mark_allowed,omitempty"`
	Wiring           Wiring  `json:"wiring"`
	// Sizes, for container targets (ICO/ICNS), lists the exact pixel sizes to
	// pack. Empty for single-image targets.
	Sizes []int `json:"sizes,omitempty"`
}

// Profile is a named, versioned list of targets under one root.
type Profile struct {
	ID      string
	Version int
	Root    string
	Targets []Target
	// requiresDir, when non-empty, is a scenario-relative directory that must
	// exist for the profile to apply (electron-v1 applies only when
	// platforms/electron exists).
	requiresDir string
}

// WebPublicV1 is the `/public/*` PWA set under ui/public/public.
var WebPublicV1 = Profile{
	ID:      "web-public-v1",
	Version: 1,
	Root:    "ui/public/public",
	Targets: []Target{
		{ID: "logo", Path: "logo.svg", Format: "svg", Width: 512, Height: 512, Variant: VariantVector, Wiring: WiringLinkIcon},
		{ID: "favicon-16", Path: "favicon-16.png", Format: "png", Width: 16, Height: 16, Variant: VariantRounded, SmallMarkAllowed: true, Wiring: WiringLinkIcon},
		{ID: "favicon-32", Path: "favicon-32.png", Format: "png", Width: 32, Height: 32, Variant: VariantRounded, SmallMarkAllowed: true, Wiring: WiringLinkIcon},
		{ID: "favicon-196", Path: "favicon-196.png", Format: "png", Width: 196, Height: 196, Variant: VariantRounded, Wiring: WiringNone},
		{ID: "icon-192", Path: "icon-192.png", Format: "png", Width: 192, Height: 192, Variant: VariantRounded, Wiring: WiringManifestAny},
		{ID: "icon-512", Path: "icon-512.png", Format: "png", Width: 512, Height: 512, Variant: VariantRounded, Wiring: WiringManifestAny},
		{ID: "maskable-192", Path: "maskable-icon-192.png", Format: "png", Width: 192, Height: 192, Variant: VariantMaskable, Wiring: WiringManifestMaskable},
		{ID: "maskable-512", Path: "maskable-icon-512.png", Format: "png", Width: 512, Height: 512, Variant: VariantMaskable, Wiring: WiringManifestMaskable},
		{ID: "apple-touch", Path: "apple-touch-icon.png", Format: "png", Width: 180, Height: 180, Variant: VariantFullBleed, Wiring: WiringLinkAppleTouch},
		{ID: "og-image", Path: "og-image.png", Format: "png", Width: 1200, Height: 630, Variant: VariantSocialCard, Wiring: WiringMetaOG},
		{ID: "manifest", Path: "site.webmanifest", Format: "json", Wiring: WiringManifestFile},
	},
}

// ElectronV1 is the desktop set under platforms/electron/assets. It applies only
// when a platforms/electron directory exists.
var ElectronV1 = Profile{
	ID:          "electron-v1",
	Version:     1,
	Root:        "platforms/electron/assets",
	requiresDir: "platforms/electron",
	Targets: []Target{
		{ID: "icon", Path: "icon.png", Format: "png", Width: 512, Height: 512, Variant: VariantRounded, Wiring: WiringNone},
		{ID: "icon-16", Path: "icon-16x16.png", Format: "png", Width: 16, Height: 16, Variant: VariantRounded, SmallMarkAllowed: true, Wiring: WiringNone},
		{ID: "icon-32", Path: "icon-32x32.png", Format: "png", Width: 32, Height: 32, Variant: VariantRounded, SmallMarkAllowed: true, Wiring: WiringNone},
		{ID: "icon-48", Path: "icon-48x48.png", Format: "png", Width: 48, Height: 48, Variant: VariantRounded, Wiring: WiringNone},
		{ID: "icon-128", Path: "icon-128x128.png", Format: "png", Width: 128, Height: 128, Variant: VariantRounded, Wiring: WiringNone},
		{ID: "icon-256", Path: "icon-256x256.png", Format: "png", Width: 256, Height: 256, Variant: VariantRounded, Wiring: WiringNone},
		{ID: "icon-512", Path: "icon-512x512.png", Format: "png", Width: 512, Height: 512, Variant: VariantRounded, Wiring: WiringNone},
		{ID: "icon-1024", Path: "icon-1024x1024.png", Format: "png", Width: 1024, Height: 1024, Variant: VariantRounded, Wiring: WiringNone},
		{ID: "ico", Path: "icon.ico", Format: "ico", Variant: VariantRounded, Wiring: WiringNone, Sizes: []int{16, 24, 32, 48, 64, 128, 256}},
		{ID: "icns", Path: "icon.icns", Format: "icns", Variant: VariantRounded, Wiring: WiringNone, Sizes: []int{32, 64, 128, 256, 512, 1024}},
	},
}

// ByID returns the profile with the given id.
func ByID(id string) (Profile, bool) {
	switch id {
	case WebPublicV1.ID:
		return WebPublicV1, true
	case ElectronV1.ID:
		return ElectronV1, true
	default:
		return Profile{}, false
	}
}

// ErrNoBranding reports a scenario that declares no branding block.
var ErrNoBranding = fmt.Errorf("profiles: scenario declares no branding block")

type brandingDoc struct {
	Branding *struct {
		Brand   string   `json:"brand"`
		Targets []string `json:"targets"`
	} `json:"branding"`
}

// Declared parses a scenario's service.json bytes and returns the brand slug and
// the declared profile ids. It is the byte-based counterpart of Resolve for
// callers (apply) that read the file through an abstraction rather than disk.
func Declared(data []byte) (brand string, targetIDs []string, err error) {
	var doc brandingDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", nil, fmt.Errorf("profiles: parse service.json: %w", err)
	}
	if doc.Branding == nil {
		return "", nil, ErrNoBranding
	}
	for _, id := range doc.Branding.Targets {
		if _, ok := ByID(id); !ok {
			return "", nil, fmt.Errorf("profiles: unknown target profile %q", id)
		}
	}
	return doc.Branding.Brand, doc.Branding.Targets, nil
}

// Resolve reads scenarioDir/.vrooli/service.json and returns the declared
// profiles. It drops electron-v1 when platforms/electron does not exist and
// returns ErrNoBranding when the block is absent.
func Resolve(scenarioDir string) ([]Profile, error) {
	data, err := os.ReadFile(filepath.Join(scenarioDir, ".vrooli", "service.json"))
	if err != nil {
		return nil, fmt.Errorf("profiles: read service.json: %w", err)
	}
	var doc brandingDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("profiles: parse service.json: %w", err)
	}
	if doc.Branding == nil || len(doc.Branding.Targets) == 0 {
		return nil, ErrNoBranding
	}
	var out []Profile
	for _, id := range doc.Branding.Targets {
		p, ok := ByID(id)
		if !ok {
			return nil, fmt.Errorf("profiles: unknown target profile %q", id)
		}
		if p.requiresDir != "" {
			if _, err := os.Stat(filepath.Join(scenarioDir, p.requiresDir)); err != nil {
				continue
			}
		}
		out = append(out, p)
	}
	return out, nil
}

// Brand reads just the brand slug from a scenario's branding block.
func Brand(scenarioDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(scenarioDir, ".vrooli", "service.json"))
	if err != nil {
		return "", err
	}
	var doc brandingDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", err
	}
	if doc.Branding == nil {
		return "", ErrNoBranding
	}
	return doc.Branding.Brand, nil
}
