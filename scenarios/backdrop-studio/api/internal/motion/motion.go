// Package motion turns a plate stack into a CSS transform descriptor.
//
// A landing page gets depth from a handful of images and a stylesheet: no video
// decode, no animated GIF, no runtime library. That is a deliberate boundary
// rather than a shortcut. `image-tools` records motion content as a non-goal, a
// video costs a decode for something a transform does for free, and a
// descriptor degrades to a flat image when a viewer asks for reduced motion —
// which a video does not.
//
// The revisit trigger is written down: the first ambient loop CSS genuinely
// cannot express is the signal to build a rich-media capability, not to add an
// encoder here.
//
// What this package does NOT do is decide art direction. It emits what the
// plates declare, and refuses to emit anything for a stack that declares no
// depth — a set of plates all moving together is a flat image wearing a
// manifest, and shipping one would tell a consumer there is parallax to render
// when there is not.
package motion

import (
	"fmt"
	"sort"
	"strings"
)

// Ambient loops a plate may declare.
const (
	// AmbientDrift translates the plate slowly along one axis and back: cloud,
	// smoke, a slow tide.
	AmbientDrift = "drift"
	// AmbientSway rotates it a fraction of a degree: foliage, a hanging sign.
	AmbientSway = "sway"
	// AmbientBreathe scales it imperceptibly: light, a horizon haze.
	AmbientBreathe = "breathe"
)

// Profile is how one plate moves.
type Profile struct {
	// Parallax is how far the plate travels against the scroll, as a fraction
	// of the viewport's travel. 0 is pinned to the page; 1 moves with it.
	Parallax float64
	// Ambient is an optional continuous loop; empty means scroll-only.
	Ambient string
	// AmbientSeconds is one period. Long, because an ambient motion a reader
	// can time is a distraction rather than an atmosphere.
	AmbientSeconds float64
	// AmbientAmplitude is the loop's travel as a fraction of the short edge.
	AmbientAmplitude float64
}

// Layer is one plate in the manifest.
type Layer struct {
	Name  string
	Depth int
	Blend string
	// File is the plate's filename within the delivery set.
	File    string
	Opacity float64
	Motion  Profile
}

// Manifest is the delivery set's description of its own contents.
//
// It names every file a consumer receives, because a delivery set whose parts
// a consumer has to infer is a set they will get wrong. The composite is listed
// first and separately: it is what a consumer renders when it ignores
// everything else, which most will.
type Manifest struct {
	StyleID string `json:"style_id"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	// Composite is the flat picture, always present.
	Composite string `json:"composite"`
	// ReducedMotion is what a `prefers-reduced-motion` viewer gets. It is the
	// composite, named separately so the contract does not depend on a
	// consumer noticing they are the same file.
	ReducedMotion string `json:"reduced_motion"`
	// Layers are the plates, back to front. Empty when the stack declares no
	// depth, in which case there is no motion to describe.
	Layers []ManifestLayer `json:"layers,omitempty"`
	// CSS names the stylesheet, empty when no motion is emitted.
	CSS string `json:"css,omitempty"`
}

// ManifestLayer is one plate as the manifest describes it.
type ManifestLayer struct {
	Name             string  `json:"name"`
	File             string  `json:"file"`
	Depth            int     `json:"depth"`
	Blend            string  `json:"blend"`
	Opacity          float64 `json:"opacity"`
	Parallax         float64 `json:"parallax"`
	Ambient          string  `json:"ambient,omitempty"`
	AmbientSeconds   float64 `json:"ambient_seconds,omitempty"`
	AmbientAmplitude float64 `json:"ambient_amplitude,omitempty"`
}

// FlatStackError reports a stack whose plates all move together.
//
// It is an error rather than an empty descriptor because the two mean different
// things to a consumer: no manifest says "this is a picture", while a manifest
// with no depth says "render these layers with parallax" and then produces
// none. The second wastes a consumer's implementation on nothing.
type FlatStackError struct {
	StyleID  string
	Parallax float64
	Layers   int
}

func (e *FlatStackError) Error() string {
	return fmt.Sprintf(
		"motion: style %q declares %d plates all at parallax %g; that is a flat image and it needs no motion descriptor",
		e.StyleID, e.Layers, e.Parallax)
}

// Describe builds the manifest and the stylesheet for one delivery set.
//
// It returns the manifest even when it refuses the motion descriptor, because a
// consumer still needs to be told what files exist — the refusal is about
// motion, not about delivery.
func Describe(styleID string, width, height int, composite string, layers []Layer) (Manifest, string, error) {
	manifest := Manifest{
		StyleID:       styleID,
		Width:         width,
		Height:        height,
		Composite:     composite,
		ReducedMotion: composite,
	}
	if len(layers) < 2 {
		return manifest, "", nil
	}
	ordered := append([]Layer(nil), layers...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Depth < ordered[j].Depth })

	distinct := map[float64]struct{}{}
	for _, layer := range ordered {
		distinct[layer.Motion.Parallax] = struct{}{}
	}
	if len(distinct) < 2 {
		return manifest, "", &FlatStackError{StyleID: styleID, Parallax: ordered[0].Motion.Parallax, Layers: len(ordered)}
	}

	for _, layer := range ordered {
		manifest.Layers = append(manifest.Layers, ManifestLayer{
			Name: layer.Name, File: layer.File, Depth: layer.Depth, Blend: layer.Blend,
			Opacity: layer.Opacity, Parallax: layer.Motion.Parallax,
			Ambient: layer.Motion.Ambient, AmbientSeconds: layer.Motion.AmbientSeconds,
			AmbientAmplitude: layer.Motion.AmbientAmplitude,
		})
	}
	manifest.CSS = "motion.css"
	return manifest, CSS(styleID, manifest.Layers, composite), nil
}

// CSS emits the stylesheet.
//
// Two properties it holds deliberately. Every rule is scoped to a class derived
// from the style id, so a page carrying two backdrops does not have one
// stylesheet silently reposition the other's layers. And the whole motion block
// sits inside `@media (prefers-reduced-motion: no-preference)`, so reduced
// motion is the DEFAULT that requires no override — a consumer who forgets the
// media query still gets a still picture rather than an unwanted one.
func CSS(styleID string, layers []ManifestLayer, composite string) string {
	root := className(styleID)
	var b strings.Builder

	fmt.Fprintf(&b, "/* Backdrop Studio motion descriptor for %q.\n", styleID)
	b.WriteString(" *\n")
	b.WriteString(" * Plate images plus transforms — no video, no GIF, no runtime library.\n")
	b.WriteString(" * The flat composite is the ground of the stack and what a reduced-motion\n")
	b.WriteString(" * viewer sees; every layer above it is additive, so a consumer that loads\n")
	b.WriteString(" * only the composite gets a complete picture.\n")
	b.WriteString(" *\n")
	b.WriteString(" * Drive the parallax by setting --scroll on the root element to the page's\n")
	b.WriteString(" * scroll offset in pixels. With no --scroll the layers rest at zero, which\n")
	b.WriteString(" * is the composite's own composition.\n")
	b.WriteString(" */\n\n")

	fmt.Fprintf(&b, ".%s {\n", root)
	b.WriteString("  position: relative;\n  overflow: hidden;\n  isolation: isolate;\n")
	b.WriteString("  --scroll: 0px;\n")
	fmt.Fprintf(&b, "  background-image: url(\"%s\");\n", composite)
	b.WriteString("  background-size: cover;\n  background-position: center;\n}\n\n")

	fmt.Fprintf(&b, ".%s__layer {\n", root)
	b.WriteString("  position: absolute;\n  inset: 0;\n")
	b.WriteString("  background-size: cover;\n  background-position: center;\n")
	b.WriteString("  will-change: transform;\n}\n\n")

	// The layer rules. Every plate in the manifest gets exactly one, which the
	// emitter test asserts: a plate with no rule is a file a consumer downloads
	// and never draws.
	for _, layer := range layers {
		selector := fmt.Sprintf(".%s__layer--%s", root, className(layer.Name))
		fmt.Fprintf(&b, "%s {\n", selector)
		fmt.Fprintf(&b, "  background-image: url(\"%s\");\n", layer.File)
		fmt.Fprintf(&b, "  z-index: %d;\n", layer.Depth+1)
		if layer.Opacity > 0 && layer.Opacity < 1 {
			fmt.Fprintf(&b, "  opacity: %s;\n", trim(layer.Opacity))
		}
		if mode := cssBlend(layer.Blend); mode != "" {
			fmt.Fprintf(&b, "  mix-blend-mode: %s;\n", mode)
		}
		b.WriteString("}\n\n")
	}

	b.WriteString("@media (prefers-reduced-motion: no-preference) {\n")
	for _, layer := range layers {
		selector := fmt.Sprintf(".%s__layer--%s", root, className(layer.Name))
		fmt.Fprintf(&b, "  %s {\n", selector)
		// The parallax itself. A plate at factor f travels f times the scroll
		// in the opposite direction, which is what makes a far plate appear to
		// lag behind a near one.
		fmt.Fprintf(&b, "    transform: translate3d(0, calc(var(--scroll) * %s), 0);\n", trim(-layer.Parallax))
		if layer.Ambient != "" {
			fmt.Fprintf(&b, "    animation: %s-%s %ss ease-in-out infinite alternate;\n",
				root, className(layer.Name), trim(layer.AmbientSeconds))
		}
		b.WriteString("  }\n")
	}
	for _, layer := range layers {
		if layer.Ambient == "" {
			continue
		}
		b.WriteString("\n")
		fmt.Fprintf(&b, "  @keyframes %s-%s {\n", root, className(layer.Name))
		from, to := ambientFrames(layer)
		fmt.Fprintf(&b, "    from { %s }\n", from)
		fmt.Fprintf(&b, "    to   { %s }\n", to)
		b.WriteString("  }\n")
	}
	b.WriteString("}\n")
	return b.String()
}

// ambientFrames renders one loop's endpoints.
//
// Each keyframe restates the parallax translate, because a CSS `transform`
// declared in a keyframe REPLACES the one on the rule rather than composing
// with it — an animation that omitted it would snap every animated layer back
// to zero parallax the moment its loop started.
func ambientFrames(layer ManifestLayer) (string, string) {
	base := fmt.Sprintf("translate3d(0, calc(var(--scroll) * %s), 0)", trim(-layer.Parallax))
	amp := layer.AmbientAmplitude
	switch layer.Ambient {
	case AmbientSway:
		return fmt.Sprintf("transform: %s rotate(%sdeg);", base, trim(-amp*4)),
			fmt.Sprintf("transform: %s rotate(%sdeg);", base, trim(amp*4))
	case AmbientBreathe:
		return fmt.Sprintf("transform: %s scale(%s);", base, trim(1-amp*0.5)),
			fmt.Sprintf("transform: %s scale(%s);", base, trim(1+amp*0.5))
	default: // drift
		return fmt.Sprintf("transform: %s translateX(%s%%);", base, trim(-amp*100)),
			fmt.Sprintf("transform: %s translateX(%s%%);", base, trim(amp*100))
	}
}

// cssBlend maps a plate blend to its CSS keyword. `normal` emits nothing: it is
// the initial value, and writing it would add a line that changes nothing while
// implying it does.
func cssBlend(blend string) string {
	switch strings.ToLower(strings.TrimSpace(blend)) {
	case "multiply":
		return "multiply"
	case "screen":
		return "screen"
	default:
		return ""
	}
}

// className turns an id into a CSS-safe token.
func className(raw string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(raw)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == ' ' || r == '.':
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "backdrop"
	}
	return out
}

// trim formats a number without trailing zeroes, so the stylesheet reads like
// something a person wrote.
func trim(v float64) string {
	s := fmt.Sprintf("%.4f", v)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}
