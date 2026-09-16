package apply

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"

	"brand-manager/internal/profiles"
	"brand-manager/internal/render"
)

// Managed icon paths. The /public/* layout serves ui/public/public at /public.
const (
	indexHTMLPath = "ui/index.html"
	markerStart   = "<!-- brand-manager:icons:start"
	markerEnd     = "<!-- brand-manager:icons:end -->"
)

// applyIcons resolves the scenario's declared target profiles, renders every
// target from the brand's vector mark and container style, and writes them. It
// replaces the old root-layout writer: nothing writes ui/public directly or a
// root-relative manifest src.
func (s *service) applyIcons(ctx context.Context, brand BrandView, scenario string, write bool) ([]Action, *Skip, error) {
	if s.renderer == nil {
		return nil, &Skip{Element: ElementIcons, Reason: "icon rendering is not configured"}, nil
	}
	serviceJSON, err := s.workspace.ReadFile(ctx, scenario, ".vrooli/service.json")
	if err != nil {
		return nil, nil, err
	}
	brandSlug, targetIDs, err := profiles.Declared(serviceJSON)
	if err != nil {
		return nil, &Skip{Element: ElementIcons, Reason: "scenario declares no branding targets"}, nil
	}
	if strings.TrimSpace(brand.MarkAssetID) == "" {
		return nil, &Skip{Element: ElementIcons, Reason: "brand has no picked mark"}, nil
	}

	mark, found, err := s.assets.ReadByID(ctx, brand.MarkAssetID)
	if err != nil {
		return nil, nil, err
	}
	if !found {
		return nil, &Skip{Element: ElementIcons, Reason: "mark asset not found"}, nil
	}
	var small []byte
	if brand.SmallMarkAssetID != "" {
		if sm, ok, serr := s.assets.ReadByID(ctx, brand.SmallMarkAssetID); serr == nil && ok {
			small = sm.Bytes
		}
	}

	style := s.resolveStyle(ctx, brand.ContainerStyleID)
	renderStyle := toRenderStyle(style)

	var actions []Action
	for _, id := range targetIDs {
		profile, ok := profiles.ByID(id)
		if !ok {
			continue
		}
		var written []profiles.Target
		for _, target := range profile.Targets {
			if target.ID == "manifest" {
				continue // written once below with the complete icon list
			}
			useSmall := target.SmallMarkAllowed && len(small) > 0 && target.Width > 0 && target.Width <= style.SmallMarkThresholdPx
			data, rerr := s.renderTarget(ctx, renderStyle, mark.Bytes, small, target, useSmall)
			if rerr != nil {
				return nil, nil, rerr
			}
			rel := path.Join(profile.Root, target.Path)
			if write {
				if werr := s.workspace.WriteFile(ctx, scenario, rel, data); werr != nil {
					return nil, nil, werr
				}
			}
			actions = append(actions, Action{Type: ActionAsset, File: rel, Element: ElementIcons})
			written = append(written, target)
		}
		if profile.ID == profiles.WebPublicV1.ID {
			if write {
				if werr := s.writeWebPublicWiring(ctx, scenario, brandSlug, brand, written); werr != nil {
					return nil, nil, werr
				}
			}
			actions = append(actions,
				Action{Type: ActionJSON, File: path.Join(profile.Root, "site.webmanifest"), Element: ElementIcons},
				Action{Type: ActionAsset, File: indexHTMLPath, Element: ElementIcons},
			)
		}
	}
	if len(actions) == 0 {
		return nil, &Skip{Element: ElementIcons, Reason: "no declared targets"}, nil
	}
	return actions, nil, nil
}

// renderTarget composes and rasterizes one target from the mark.
func (s *service) renderTarget(ctx context.Context, style render.Style, mark, small []byte, target profiles.Target, useSmall bool) ([]byte, error) {
	edge := target.Width
	if edge == 0 {
		edge = 512
	}
	variant := toRenderVariant(target.Variant)
	switch target.Format {
	case "svg":
		return render.Compose(mark, small, style, render.Vector, edge, false)
	case "png":
		svg, err := render.Compose(mark, small, style, variant, edge, useSmall)
		if err != nil {
			return nil, err
		}
		h := target.Height
		if h == 0 {
			h = edge
		}
		return s.renderer.Rasterize(ctx, svg, target.Width, h, "")
	case "ico", "icns":
		svg, err := render.Compose(mark, small, style, render.Rounded, 512, useSmall)
		if err != nil {
			return nil, err
		}
		return s.renderer.IconContainer(ctx, svg, target.Format, target.Sizes)
	default:
		return nil, fmt.Errorf("apply: unsupported target format %q", target.Format)
	}
}

// writeWebPublicWiring rewrites the marked index.html link block and the
// relative-src site.webmanifest for the web-public-v1 profile.
func (s *service) writeWebPublicWiring(ctx context.Context, scenario, brandSlug string, brand BrandView, written []profiles.Target) error {
	// site.webmanifest with relative srcs and correct purposes.
	manifest := map[string]any{
		"name":             firstNonEmpty(brand.DisplayName, brandSlug),
		"short_name":       firstNonEmpty(brand.DisplayName, brandSlug),
		// theme_color tints the status bar and launch screen, so it is the brand
		// background (the page's <meta theme-color>), not the accent-like primary.
		"theme_color": firstNonEmpty(firstNonEmpty(brand.Colors.Background, brand.Colors.Primary), "#0f172a"),
		"background_color": firstNonEmpty(brand.Colors.Background, "#0f172a"),
		"icons":            manifestIcons(written),
	}
	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := s.workspace.WriteFile(ctx, scenario, path.Join(profiles.WebPublicV1.Root, "site.webmanifest"), append(out, '\n')); err != nil {
		return err
	}

	// index.html marked block.
	existing, err := s.workspace.ReadFile(ctx, scenario, indexHTMLPath)
	if err != nil {
		return err
	}
	if len(existing) == 0 {
		return nil
	}
	updated := rewriteIconBlock(string(existing), brandSlug, brand.Version)
	updated = rewriteMetaImage(updated, "og:image", "/public/og-image.png")
	updated = rewriteMetaImage(updated, "twitter:image", "/public/og-image.png")
	return s.workspace.WriteFile(ctx, scenario, indexHTMLPath, []byte(updated))
}

func (s *service) resolveStyle(ctx context.Context, styleID string) ContainerStyleView {
	if s.styles != nil && styleID != "" {
		if st, ok, err := s.styles.ContainerStyle(ctx, styleID); err == nil && ok {
			return st
		}
	}
	return defaultContainerStyle()
}

// defaultContainerStyle is the constellation-midnight values, used when a brand
// has no style or the style store is unavailable.
func defaultContainerStyle() ContainerStyleView {
	return ContainerStyleView{
		Shape:                "rounded_square",
		CornerRatio:          0.21875,
		BackgroundTop:        "#15243c",
		BackgroundBottom:     "#0b1728",
		MarkScale:            0.86,
		MaskableScale:        0.40,
		AccentColor:          "#22d3ee",
		Glow:                 []GlowLayerView{{Width: 2, Opacity: 0.45}, {Width: 5, Opacity: 0.22}, {Width: 9, Opacity: 0.10}},
		SmallMarkThresholdPx: 32,
	}
}

func toRenderStyle(st ContainerStyleView) render.Style {
	glow := make([]render.GlowLayer, 0, len(st.Glow))
	for _, g := range st.Glow {
		glow = append(glow, render.GlowLayer{Width: g.Width, Opacity: g.Opacity})
	}
	return render.Style{
		Shape:            st.Shape,
		CornerRatio:      st.CornerRatio,
		BackgroundTop:    st.BackgroundTop,
		BackgroundBottom: st.BackgroundBottom,
		MarkScale:        st.MarkScale,
		MaskableScale:    st.MaskableScale,
		AccentColor:      st.AccentColor,
		Glow:             glow,
	}
}

func toRenderVariant(v profiles.Variant) render.Variant {
	switch v {
	case profiles.VariantFullBleed:
		return render.FullBleed
	case profiles.VariantMaskable:
		return render.Maskable
	case profiles.VariantSocialCard:
		return render.SocialCard
	case profiles.VariantVector:
		return render.Vector
	default:
		return render.Rounded
	}
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// manifestIcons builds the manifest icons array from the written targets.
func manifestIcons(written []profiles.Target) []any {
	var icons []any
	for _, t := range written {
		if t.Format != "png" || t.Wiring == profiles.WiringNone {
			continue
		}
		purpose := ""
		switch t.Wiring {
		case profiles.WiringManifestAny:
			purpose = "any"
		case profiles.WiringManifestMaskable:
			purpose = "maskable"
		default:
			continue
		}
		entry := map[string]any{
			"src":     t.Path,
			"sizes":   fmt.Sprintf("%dx%d", t.Width, t.Height),
			"type":    "image/png",
			"purpose": purpose,
		}
		icons = append(icons, entry)
	}
	return icons
}

var (
	iconLinkPattern = regexp.MustCompile(`(?i)<link\b[^>]*\brel=["'](?:icon|apple-touch-icon|manifest)["'][^>]*>`)
	blockPattern    = regexp.MustCompile(`(?s)<!-- brand-manager:icons:start.*?<!-- brand-manager:icons:end -->`)
	headClose       = regexp.MustCompile(`(?i)</head>`)
	metaPattern     = regexp.MustCompile(`(?i)(<meta[^>]*(?:property|name)=["'])([^"']*)(["'][^>]*?content=["'])([^"']*)(["'])`)
)

// rewriteIconBlock replaces the marked block, or (on first apply) removes the
// existing icon/apple-touch/manifest link tags and inserts the block.
func rewriteIconBlock(html, brandSlug string, version int) string {
	block := buildIconBlock(brandSlug, version)
	if blockPattern.MatchString(html) {
		return blockPattern.ReplaceAllString(html, block)
	}
	stripped := iconLinkPattern.ReplaceAllString(html, "")
	if headClose.MatchString(stripped) {
		return headClose.ReplaceAllString(stripped, block+"\n</head>")
	}
	return stripped + "\n" + block
}

func buildIconBlock(brandSlug string, version int) string {
	lines := []string{
		fmt.Sprintf(`<!-- brand-manager:icons:start profile=web-public-v1 brand=%s version=%d -->`, brandSlug, version),
		`<link rel="icon" type="image/svg+xml" href="/public/logo.svg" />`,
		`<link rel="icon" type="image/png" sizes="32x32" href="/public/favicon-32.png" />`,
		`<link rel="icon" type="image/png" sizes="16x16" href="/public/favicon-16.png" />`,
		`<link rel="apple-touch-icon" sizes="180x180" href="/public/apple-touch-icon.png" />`,
		`<link rel="manifest" href="/public/site.webmanifest" crossorigin="use-credentials" />`,
		markerEnd,
	}
	return strings.Join(lines, "\n")
}

// rewriteMetaImage updates a property/name meta tag's content in place.
func rewriteMetaImage(html, key, value string) string {
	return metaPattern.ReplaceAllStringFunc(html, func(m string) string {
		parts := metaPattern.FindStringSubmatch(m)
		if len(parts) != 6 || !strings.EqualFold(parts[2], key) {
			return m
		}
		return parts[1] + parts[2] + parts[3] + value + parts[5]
	})
}
