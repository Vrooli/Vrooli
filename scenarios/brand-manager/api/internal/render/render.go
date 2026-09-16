// Package render composes a brand's vector mark plus a container style into the
// filter-free SVG variants every icon target derives from. It is deterministic
// and pure: no image-tools call, no clock, no randomness. Rasterization is a
// separate step (image-tools rasterize/icon_container).
package render

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"regexp"
	"strings"
)

// GlowLayer is one halo layer drawn under the accent paths.
type GlowLayer struct {
	Width   float64
	Opacity float64
}

// Style is a container style record.
type Style struct {
	Shape            string // rounded_square | square
	CornerRatio      float64
	BackgroundTop    string
	BackgroundBottom string
	MarkScale        float64
	MaskableScale    float64
	AccentColor      string
	Glow             []GlowLayer
}

// Variant selects the composition.
type Variant string

const (
	Rounded    Variant = "rounded"
	FullBleed  Variant = "full_bleed"
	Maskable   Variant = "maskable"
	SocialCard Variant = "social_card"
	Vector     Variant = "vector"
)

// Compose returns the SVG for one variant. markSVG is the vector mark; smallSVG
// (may be nil) is used when the target allows and needs a simplified mark.
// edge is the square edge in px; for social_card the output is 1200x630.
func Compose(markSVG []byte, smallSVG []byte, style Style, variant Variant, edge int, useSmall bool) ([]byte, error) {
	mark := markSVG
	if useSmall && len(smallSVG) > 0 {
		mark = smallSVG
	}
	if variant == Vector {
		if len(mark) == 0 {
			return nil, fmt.Errorf("render: no mark provided")
		}
		return mark, nil
	}
	inner, vbW, vbH, err := svgInner(mark)
	if err != nil {
		return nil, err
	}
	if edge <= 0 {
		edge = 512
	}

	width, height := edge, edge
	clipRadius := style.CornerRatio * float64(edge)
	opaque := variant == FullBleed || variant == Maskable
	if variant == SocialCard {
		width, height = 1200, 630
		clipRadius = 0
	}
	if variant == FullBleed || variant == Maskable {
		clipRadius = 0
	}

	// Fit the mark into the tile: MarkScale is the mark's longer side as a
	// fraction of the tile edge, so the scale is normalized by the mark's own
	// viewBox. (Using MarkScale directly drew a 2048-unit traced mark at 1761 px
	// on a 180 px icon, so every PNG showed a zoomed-in fragment.) A social card
	// sizes the mark to 0.8 of the card height.
	markFraction := style.MarkScale
	if markFraction <= 0 {
		markFraction = 0.86
	}
	longest := math.Max(float64(vbW), float64(vbH))
	if longest <= 0 {
		return nil, fmt.Errorf("render: mark has an empty viewBox")
	}
	scale := markFraction * float64(edge) / longest
	if variant == SocialCard {
		scale = 0.8 * float64(height) / longest
	}
	if variant == Maskable {
		s := style.MaskableScale
		if s <= 0 {
			s = 0.40
		}
		// The mark's circumscribed circle must fit radius s*edge: scale so the
		// mark's diagonal fits 2*s*edge.
		diag := math.Hypot(float64(vbW), float64(vbH))
		if diag > 0 {
			scale = (2 * s * float64(edge)) / diag
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, width, height, width, height))
	if style.BackgroundTop != "" {
		b.WriteString(fmt.Sprintf(`<defs><linearGradient id="bg" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="%s"/><stop offset="1" stop-color="%s"/></linearGradient></defs>`, style.BackgroundTop, style.BackgroundBottom))
	} else {
		style.BackgroundTop = "#000000"
	}
	rx := ""
	if clipRadius > 0 {
		rx = fmt.Sprintf(` rx="%.2f"`, clipRadius)
	}
	b.WriteString(fmt.Sprintf(`<rect x="0" y="0" width="%d" height="%d"%s fill="url(#bg)"/>`, width, height, rx))

	// Place the mark centred. For social_card the mark is centred at 0.8 of the
	// height (the plan's "centred at 0.8 of height").
	markW := float64(vbW) * scale
	markH := float64(vbH) * scale
	tx := (float64(width) - markW) / 2
	ty := (float64(height) - markH) / 2

	// Place the mark with one matrix() rather than a transform list: image-tools'
	// pure-Go rasterizer silently drops a group whose transform is
	// "translate(...) scale(...)", which rendered every PNG target as an empty
	// tile (web-console's apple-touch-icon, 2026-09-15).
	place := fmt.Sprintf(`matrix(%.6f 0 0 %.6f %.3f %.3f)`, scale, scale, tx, ty)

	// Layered glow strokes under each accent path.
	if style.AccentColor != "" && len(style.Glow) > 0 {
		accent := accentPaths(inner, style.AccentColor)
		if len(accent) > 0 {
			b.WriteString(fmt.Sprintf(`<g transform="%s" fill="none" stroke="%s" stroke-linejoin="round" stroke-linecap="round">`, place, style.AccentColor))
			for _, layer := range style.Glow {
				for _, d := range accent {
					b.WriteString(fmt.Sprintf(`<path d="%s" stroke-width="%.3f" stroke-opacity="%.3f"/>`, d, layer.Width, layer.Opacity))
				}
			}
			b.WriteString(`</g>`)
		}
	}

	b.WriteString(fmt.Sprintf(`<g transform="%s">%s</g>`, place, inner))
	b.WriteString(`</svg>`)
	out := []byte(b.String())
	if !opaque && variant != FullBleed {
		// rounded variant keeps transparency outside the tile
	}
	return out, nil
}

var (
	svgInnerPattern = regexp.MustCompile(`(?is)<svg[^>]*>(.*)</svg>`)
	viewBoxPattern  = regexp.MustCompile(`(?i)viewBox\s*=\s*["']\s*[-\d.]+\s+[-\d.]+\s+([\d.]+)\s+([\d.]+)`)
	widthAttr       = regexp.MustCompile(`(?i)<svg[^>]*\bwidth\s*=\s*["']\s*([\d.]+)`)
	heightAttr      = regexp.MustCompile(`(?i)<svg[^>]*\bheight\s*=\s*["']\s*([\d.]+)`)
	pathPattern     = regexp.MustCompile(`(?is)<path\b[^>]*\bfill\s*=\s*"([^"]*)"[^>]*\bd\s*=\s*"([^"]+)"`)
	pathPatternAlt  = regexp.MustCompile(`(?is)<path\b[^>]*\bd\s*=\s*"([^"]+)"[^>]*\bfill\s*=\s*"([^"]*)"`)
)

func svgInner(mark []byte) (string, int, int, error) {
	m := svgInnerPattern.FindSubmatch(mark)
	if m == nil {
		return "", 0, 0, fmt.Errorf("render: mark is not an SVG")
	}
	inner := string(m[1])
	w, h := 0, 0
	if vb := viewBoxPattern.FindSubmatch(mark); vb != nil {
		fmt.Sscanf(string(vb[1]), "%d", &w)
		fmt.Sscanf(string(vb[2]), "%d", &h)
	}
	if w == 0 {
		if wm := widthAttr.FindSubmatch(mark); wm != nil {
			fmt.Sscanf(string(wm[1]), "%d", &w)
		}
	}
	if h == 0 {
		if hm := heightAttr.FindSubmatch(mark); hm != nil {
			fmt.Sscanf(string(hm[1]), "%d", &h)
		}
	}
	if w <= 0 || h <= 0 {
		return "", 0, 0, fmt.Errorf("render: mark has no usable viewBox/size")
	}
	// Sanity: the inner must be well-formed-ish XML.
	if err := xml.Unmarshal([]byte("<root>"+inner+"</root>"), new(any)); err != nil {
		// tolerate entities; only fail if clearly not XML
		if !strings.Contains(inner, "<") {
			return "", 0, 0, fmt.Errorf("render: mark inner content is empty")
		}
	}
	return inner, w, h, nil
}

// accentPaths returns the d attributes of paths whose fill matches the accent
// colour (case-insensitive).
func accentPaths(inner, accent string) []string {
	accent = strings.ToLower(strings.TrimSpace(accent))
	var out []string
	for _, m := range pathPattern.FindAllStringSubmatch(inner, -1) {
		if strings.ToLower(m[1]) == accent {
			out = append(out, m[2])
		}
	}
	for _, m := range pathPatternAlt.FindAllStringSubmatch(inner, -1) {
		if strings.ToLower(m[2]) == accent {
			out = append(out, m[1])
		}
	}
	return dedupe(out)
}

func dedupe(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// ContainsFilterFeatures reports whether an SVG uses a feature the pure-Go
// rasterizer silently drops. Rendered SVGs must return false.
func ContainsFilterFeatures(svg []byte) bool {
	for _, needle := range []string{"<filter", "<mask", "<pattern", "<style", "style="} {
		if bytes.Contains(svg, []byte(needle)) {
			return true
		}
	}
	return false
}
