// Package vector draws backdrops as SVG.
//
// It is a third generator family beside the raster scene generators, and it
// exists because of a measurement rather than a preference. Seven of the eight
// styles this catalog's 2026-08-13 verdict pass rejected fail the same way: a
// screen — halftone, stipple, engraving, line — applied to a raster source that
// contains three to five flat tonal zones. The screen has no tone to modulate,
// so it returns its own texture and the subject disappears. `cyanotype-arcade`
// delivers a field of dots with no arch in it.
//
// Screening a picture is the wrong way to get line work. A screen is a
// *reproduction* process: it turns a continuous-tone photograph into dots so a
// press can print it. What the reference material actually is — an engraved
// colonnade, a contour plate, a ringed moon over a hairline horizon — is line
// work *drawn as lines*. Drawing it as lines is what this package does, and it
// is why these generators produce a picture where the raster lane produced a
// texture.
//
// Three properties every generator here holds, and the raster lane cannot:
//
//   - Resolution independence. A stroke declared as a fraction of the frame is
//     the same stroke at 240px and at 2732px. A raster screen at a fixed ruling
//     aliases at one size and disappears at the other.
//   - Exact separation. Each depth plane is its own SVG group, so a plate model
//     gets exact alpha rather than an estimate from a depth model.
//   - Honest ink. `$brand.*` slots resolve at render time, so one generator is
//     one art direction across every brand rather than a colour baked into
//     pixels.
//
// Every generator is a pure function of (preset, size, seed, params): no
// clocks, no global RNG, no I/O. Rasterization is not this package's job and
// never will be — `image-tools` owns every pixel operation in this system, and
// this package hands it SVG.
package vector

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

// Presets are the vector generators this package can draw.
var Presets = []string{"colonnade", "contour-relief", "radiant-orb", "halftone-horizon"}

// Request describes one vector render.
type Request struct {
	Preset     string
	Width      int
	Height     int
	Seed       int64
	ParamsJSON string
	// Inks resolves the "$brand.*" slots the generator references. A slot the
	// generator names and this map does not cover is an error, never a literal
	// "$brand.primary" written into the SVG — that fail-open is what once put
	// the slot string on the wire and made ten of sixteen styles unrenderable.
	Inks map[string]string
	// Quiet is space the composition must leave for copy. Nil reserves nothing.
	Quiet *QuietZone
}

// QuietZone is space a generator reserves for copy, in fractions of the frame.
//
// The vector lane needs its own because the two mechanisms that serve the
// raster lane cannot reach it. `scenes.QuietZone` belongs to the raster
// generators, and the knockout (D-020) belongs to the treatment chain — and the
// vector styles that need this declare NO treatments at all, so there is no
// operation for a reserve to attach to. A picture with nothing applied to it
// can only be composed.
type QuietZone struct {
	X, Y, Width, Height float64
	// TowardLight selects which of the two reserves a printer would cut.
	//
	// Dark copy needs light ground, and the plate carries no ink there: a
	// knockout. Light copy needs dark ground, and the plate carries full ink: a
	// solid. They are the same decision in opposite directions, and getting it
	// backwards reserves the space by making the copy harder to read rather
	// than easier — which is what a black scrim under dark type did when this
	// problem was first attacked with a treatment.
	TowardLight bool
	// Feather is how far the reserve eases into the picture, as a fraction of
	// the SHORT edge so its softness is the same shape on a hero and on a
	// phone. Zero takes a default: a hard rectangle reads as a box laid over
	// the artwork, and a reader sees the box before they read the headline.
	Feather float64
	// Travel is how far the plate carrying this reserve slides against the
	// frame in motion, as a fraction of the frame HEIGHT.
	//
	// The copy does not move — it is laid out in the page — but the plate the
	// reserve is cut into does. Cut to the copy's own rectangle, the reserve
	// slides out from under the copy on the first scroll and the plates behind
	// come up into it. That is not hypothetical: it is what the parallax gate
	// caught here, on a style that measured 8.15 at rest and 1.00 half a screen
	// later.
	//
	// So the reserve is a volume rather than a rectangle: it extends DOWNWARD
	// by the plate's whole travel, because plates translate upward as the page
	// scrolls and the material that would arrive over the copy is the material
	// below it. Zero is correct for a plate that does not move.
	Travel float64
}

// Result carries the SVG and its content hash.
type Result struct {
	SVG    []byte
	SHA256 string
	Width  int
	Height int
	// Planes names the depth groups the generator drew, back to front. The
	// plate model consumes these; a flat consumer ignores them and rasterizes
	// the whole document.
	Planes []string
	// PlaneDocuments carries one complete, independently renderable SVG per
	// depth group, in the same order as Planes.
	//
	// Each is the whole document with every other plane's marks removed — the
	// shared `<defs>` stay, because a plane's filter reference has to resolve —
	// so rasterizing one yields exactly that layer over transparency. This is
	// the property a depth model cannot give: the generator KNOWS which marks
	// belong to which layer, so the matte is exact rather than estimated, and a
	// hairline that a segmentation model would smear across two layers stays
	// wholly in its own.
	PlaneDocuments [][]byte
}

// Ink slots a vector generator may reference. They are the same names
// brand-manager emits, so the catalog, the raster lane and this package share
// one vocabulary.
const (
	InkPaper  = "$brand.background"
	InkInk    = "$brand.primary"
	InkAccent = "$brand.accent"
)

// UnresolvedInkError reports a slot the generator drew with and the palette did
// not cover.
type UnresolvedInkError struct {
	Preset string
	Slot   string
}

func (e *UnresolvedInkError) Error() string {
	return fmt.Sprintf(
		"vector: generator %q draws with ink slot %q and the effective palette resolves it to nothing; "+
			"bind a brand that defines it or give the style an ink default for it",
		e.Preset, e.Slot)
}

// Render draws one vector backdrop.
func Render(req Request) (Result, error) {
	if req.Width < 16 || req.Height < 16 {
		return Result{}, fmt.Errorf("vector: width and height must be at least 16 (got %dx%d)", req.Width, req.Height)
	}
	if req.Width > 8192 || req.Height > 8192 {
		return Result{}, fmt.Errorf("vector: width and height must be at most 8192 (got %dx%d)", req.Width, req.Height)
	}
	known := false
	for _, p := range Presets {
		if p == req.Preset {
			known = true
			break
		}
	}
	if !known {
		return Result{}, fmt.Errorf("vector: unknown preset %q (known: %v)", req.Preset, Presets)
	}

	values := map[string]float64{}
	if strings.TrimSpace(req.ParamsJSON) != "" {
		if err := json.Unmarshal([]byte(req.ParamsJSON), &values); err != nil {
			return Result{}, fmt.Errorf("vector: invalid params_json: %w", err)
		}
	}

	c := newCanvas(req.Width, req.Height, req.Seed, paramSet{values: values})
	switch req.Preset {
	case "colonnade":
		drawColonnade(c)
	case "contour-relief":
		drawContourRelief(c)
	case "radiant-orb":
		drawRadiantOrb(c)
	case "halftone-horizon":
		drawHalftoneHorizon(c)
	}
	if req.Quiet != nil {
		c.reserve(*req.Quiet)
	}

	document := c.document()
	resolved, err := ResolveInks(document, req.Preset, req.Inks)
	if err != nil {
		return Result{}, err
	}
	planes, err := c.planeDocuments(req.Preset, req.Inks)
	if err != nil {
		return Result{}, err
	}
	sum := sha256.Sum256(resolved)
	return Result{
		SVG:            resolved,
		SHA256:         hex.EncodeToString(sum[:]),
		Width:          req.Width,
		Height:         req.Height,
		Planes:         c.planeNames(),
		PlaneDocuments: planes,
	}, nil
}

// ResolveInks substitutes every "$brand.*" slot in an SVG document.
//
// It fails closed on an unresolved slot for the same reason the raster
// parameter merge does: writing the literal slot name into the document
// produces bytes that parse, rasterize, and draw the wrong colour, which is a
// defect nothing downstream can see.
func ResolveInks(document []byte, preset string, inks map[string]string) ([]byte, error) {
	text := string(document)
	// Longest slot first, so "$brand.background" is never partly matched by a
	// shorter slot that happens to be its prefix.
	slots := make([]string, 0, len(knownInkSlots))
	slots = append(slots, knownInkSlots...)
	sort.Slice(slots, func(i, j int) bool { return len(slots[i]) > len(slots[j]) })
	for _, slot := range slots {
		if !strings.Contains(text, slot) {
			continue
		}
		value := strings.TrimSpace(inks[slot])
		if value == "" {
			return nil, &UnresolvedInkError{Preset: preset, Slot: slot}
		}
		text = strings.ReplaceAll(text, slot, value)
	}
	if idx := strings.Index(text, "$brand."); idx >= 0 {
		end := idx + len("$brand.")
		for end < len(text) && (text[end] == '.' || text[end] >= 'a' && text[end] <= 'z') {
			end++
		}
		return nil, &UnresolvedInkError{Preset: preset, Slot: text[idx:end]}
	}
	return []byte(text), nil
}

var knownInkSlots = []string{InkPaper, InkInk, InkAccent}

// paramSet reads optional parameters. Absence is distinguished from zero, so a
// legitimate edge value such as focal_x=0 is reachable.
type paramSet struct{ values map[string]float64 }

func (p paramSet) get(key string, min, max, fallback float64) float64 {
	v, ok := p.values[key]
	if !ok {
		return fallback
	}
	return math.Min(max, math.Max(min, v))
}

// ── deterministic noise ──────────────────────────────────────────────────
//
// The same hash the raster scene generators use, so a seed means the same thing
// across both families and a style ported from one to the other keeps its
// character.

func hash2(ix, iy int, seed int64) float64 {
	h := uint64(seed)*0x9e3779b97f4a7c15 ^ uint64(int64(ix))*0xbf58476d1ce4e5b9 ^ uint64(int64(iy))*0x94d049bb133111eb
	h ^= h >> 30
	h *= 0xbf58476d1ce4e5b9
	h ^= h >> 27
	h *= 0x94d049bb133111eb
	h ^= h >> 31
	return float64(h>>11) / float64(uint64(1)<<53)
}

func smoothstep(t float64) float64 { return t * t * (3 - 2*t) }

func valueNoise(x, y float64, seed int64) float64 {
	ix, iy := math.Floor(x), math.Floor(y)
	fx, fy := smoothstep(x-ix), smoothstep(y-iy)
	i, j := int(ix), int(iy)
	a := hash2(i, j, seed)
	b := hash2(i+1, j, seed)
	cc := hash2(i, j+1, seed)
	d := hash2(i+1, j+1, seed)
	return (a+(b-a)*fx)*(1-fy) + (cc+(d-cc)*fx)*fy
}

func fbm(x, y float64, octaves int, seed int64) float64 {
	sum, amp, freq, norm := 0.0, 0.5, 1.0, 0.0
	for i := 0; i < octaves; i++ {
		sum += amp * valueNoise(x*freq, y*freq, seed+int64(i)*7919)
		norm += amp
		freq *= 2
		amp *= 0.5
	}
	return sum / norm
}

// ── canvas ───────────────────────────────────────────────────────────────

// plane is one depth group. Keeping them separate is what lets the plate model
// take exact alpha from a vector generator instead of estimating a matte from a
// flattened raster.
type plane struct {
	name  string
	depth int
	body  strings.Builder
}

type canvas struct {
	w, h   float64
	seed   int64
	params paramSet
	planes []*plane
	defs   strings.Builder
	active *plane
}

func newCanvas(width, height int, seed int64, params paramSet) *canvas {
	return &canvas{w: float64(width), h: float64(height), seed: seed, params: params}
}

// plane opens a depth group and makes it the target of subsequent draw calls.
// depth 0 is the furthest from the viewer.
func (c *canvas) plane(name string, depth int) {
	p := &plane{name: name, depth: depth}
	c.planes = append(c.planes, p)
	c.active = p
}

// reserve lays the plate's knockout or solid over the frontmost plane.
//
// Emitted as SVG rather than composed mark by mark, because that is what the
// mechanism physically is: a reserve is a property of the PLATE — an area that
// carries no ink, or full ink — not a decision each mark makes for itself. It
// also has to hold for every generator, including ones whose marks are a single
// shape the size of the picture, where "do not draw here" has no useful meaning.
//
// This is not the scrim that was tried and abandoned. A scrim is translucent:
// it shades the picture, the picture still shows through, and worst-pixel
// contrast barely moves — measured across a catalog sitting near 1.0
// everywhere, it repaired nothing. A reserve is opaque. What the copy sits on
// is decided entirely by the reserve, which is the whole point of cutting one.
//
// Placed on the frontmost plane and nowhere else, so the flat document and the
// composited stack stay identical: the reserve appears exactly once in each. A
// feathered edge repeated on every plane would accumulate its own alpha and the
// two would drift apart at the soft edge, which is precisely where such a
// difference is least likely to be noticed.
func (c *canvas) reserve(zone QuietZone) {
	if zone.Width <= 0 || zone.Height <= 0 || len(c.planes) == 0 {
		return
	}
	feather := zone.Feather
	if feather <= 0 {
		feather = 0.09
	}
	sigma := feather * math.Min(c.w, c.h) * 0.5
	// The mask rectangle is grown by two sigma before it is blurred, so the
	// area the style actually declared stays fully covered and the softening
	// happens outside it. Blurring the declared rectangle itself would erode
	// its edges inward, and the copy would sit partly on the falloff — legible
	// in the middle and not at its corners, which is the same as not legible.
	grow := sigma * 2
	x, y := zone.X*c.w-grow, zone.Y*c.h-grow
	w, h := zone.Width*c.w+grow*2, zone.Height*c.h+grow*2
	// The travel extends the reserve downward only. Growing both ways would
	// spend twice the picture to buy nothing: a plate that scrolls upward never
	// brings the material above it into view.
	h += math.Max(0, zone.Travel) * c.h
	ink := InkPaper
	if !zone.TowardLight {
		ink = InkInk
	}
	c.def(`<filter id="reserve" x="-30%%" y="-30%%" width="160%%" height="160%%"><feGaussianBlur stdDeviation="%s"/></filter>`, f(sigma))
	c.def(`<mask id="reserve-mask"><rect x="%s" y="%s" width="%s" height="%s" fill="#ffffff" filter="url(#reserve)"/></mask>`,
		f(x), f(y), f(w), f(h))

	// Cut into EVERY plane, not only the frontmost.
	//
	// The frontmost alone is enough at rest and fails the moment the page
	// scrolls. Plates travel at different rates, so wherever the front plate's
	// reserve has slid off the copy, whatever plate is behind it is showing —
	// and the plate behind is usually the ground, which does not move at all
	// and is a full sheet of opaque paper. Reserving only the front plate
	// therefore buys exactly one frame of legibility, the one nobody scrolls.
	//
	// Repeating it costs nothing in the flat picture, because the reserves are
	// identical and opaque and the frontmost is the one seen. It also keeps the
	// flat document and the composited stack equal: source-over is associative,
	// so N stacked reserves inside one document and N plane rasters each
	// carrying one composite to the same pixels.
	for _, p := range c.planes {
		fmt.Fprintf(&p.body, `<g mask="url(#reserve-mask)"><rect width="%s" height="%s" fill="%s"/></g>`+"\n",
			f(c.w), f(c.h), ink)
	}
}

func (c *canvas) planeNames() []string {
	ordered := append([]*plane(nil), c.planes...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].depth < ordered[j].depth })
	out := make([]string, 0, len(ordered))
	for _, p := range ordered {
		out = append(out, p.name)
	}
	return out
}

func (c *canvas) write(format string, args ...any) {
	if c.active == nil {
		c.plane("main", 0)
	}
	fmt.Fprintf(&c.active.body, format, args...)
	c.active.body.WriteByte('\n')
}

func (c *canvas) def(format string, args ...any) {
	fmt.Fprintf(&c.defs, format, args...)
	c.defs.WriteByte('\n')
}

// shortest formats a coordinate compactly and, critically, identically for the
// same input on every run. %g with an explicit precision is deterministic; the
// default float formatting is not stable enough to golden against.
func f(v float64) string { return fmt.Sprintf("%.3f", v) }

func (c *canvas) document() []byte {
	ordered := append([]*plane(nil), c.planes...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].depth < ordered[j].depth })

	var out strings.Builder
	fmt.Fprintf(&out, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %s %s" preserveAspectRatio="none">`+"\n",
		int(c.w), int(c.h), f(c.w), f(c.h))
	if c.defs.Len() > 0 {
		out.WriteString("<defs>\n")
		out.WriteString(c.defs.String())
		out.WriteString("</defs>\n")
	}
	for _, p := range ordered {
		// data-depth is the contract the plate model reads: one plate per group
		// that declares a depth, back to front.
		fmt.Fprintf(&out, `<g id="plane-%s" data-plane="%s" data-depth="%d">`+"\n", p.name, p.name, p.depth)
		out.WriteString(p.body.String())
		out.WriteString("</g>\n")
	}
	out.WriteString("</svg>\n")
	return []byte(out.String())
}

// planeDocuments renders one standalone document per depth group.
//
// The envelope and the `<defs>` are repeated into every plane rather than
// factored out, because a plane has to be independently renderable: a filter
// reference that resolved only in the combined document would silently draw
// nothing when the plane was rasterized alone, and `oksvg`-class rasterizers
// drop an unresolved filter without reporting it. Repeating a few hundred bytes
// of defs is the cheap half of that trade.
func (c *canvas) planeDocuments(preset string, inks map[string]string) ([][]byte, error) {
	ordered := append([]*plane(nil), c.planes...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].depth < ordered[j].depth })
	out := make([][]byte, 0, len(ordered))
	for _, p := range ordered {
		var doc strings.Builder
		fmt.Fprintf(&doc, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %s %s" preserveAspectRatio="none">`+"\n",
			int(c.w), int(c.h), f(c.w), f(c.h))
		if c.defs.Len() > 0 {
			doc.WriteString("<defs>\n")
			doc.WriteString(c.defs.String())
			doc.WriteString("</defs>\n")
		}
		fmt.Fprintf(&doc, `<g id="plane-%s" data-plane="%s" data-depth="%d">`+"\n", p.name, p.name, p.depth)
		doc.WriteString(p.body.String())
		doc.WriteString("</g>\n</svg>\n")
		resolved, err := ResolveInks([]byte(doc.String()), preset, inks)
		if err != nil {
			return nil, err
		}
		out = append(out, resolved)
	}
	return out, nil
}

// inkWobble declares the hand-drawn filter every generator here uses.
//
// A perfectly geometric stroke reads as a diagram. The reference material this
// catalog is calibrated against reads as a *plate* — something a person cut or
// drew — and the difference is almost entirely that its lines are not exactly
// straight. A turbulence displacement at a small scale is the cheapest honest
// version of that, and it is deterministic because the turbulence carries its
// own seed.
//
// This is also the feature that makes the high-fidelity rasterizer load-bearing
// rather than a preference: oksvg drops `<filter>` silently, so on the old
// rasterizer every generator here would draw a clean geometric diagram and
// nothing would report that the character had been removed.
func (c *canvas) inkWobble(id string, scale float64, frequency float64, seed int64) {
	c.def(`<filter id="%s" x="-10%%" y="-10%%" width="120%%" height="120%%" filterUnits="objectBoundingBox">
  <feTurbulence type="fractalNoise" baseFrequency="%s" numOctaves="3" seed="%d" result="noise"/>
  <feDisplacementMap in="SourceGraphic" in2="noise" scale="%s" xChannelSelector="R" yChannelSelector="G"/>
</filter>`, id, f(frequency), seed%1000, f(scale))
}

// ── the authored lane ────────────────────────────────────────────────────

// AuthoredRequest renders a model-authored generator's body inside this
// package's document envelope.
//
// The envelope is not the author's to write. A generator that emitted its own
// `<svg>` element could set a viewBox that disagrees with the requested size,
// skip the depth planes the plate model reads, or declare an `<image>` in a
// place validation did not look. Handing this package a body and keeping the
// envelope here means every authored generator gets the same document shape and
// the same fail-closed ink resolution as a hand-written one, and cannot opt out
// of either.
type AuthoredRequest struct {
	// ID names the generator, for error messages and the plane id.
	ID string
	// Body is the rendered template output: marks only, no document element.
	Body string
	// Width and Height are the frame.
	Width, Height int
	// Inks resolves the "$brand.*" slots, exactly as for a built-in preset.
	Inks map[string]string
}

// RenderAuthored wraps an authored body and resolves its inks.
func RenderAuthored(req AuthoredRequest) (Result, error) {
	if req.Width < 16 || req.Height < 16 {
		return Result{}, fmt.Errorf("vector: width and height must be at least 16 (got %dx%d)", req.Width, req.Height)
	}
	if req.Width > 8192 || req.Height > 8192 {
		return Result{}, fmt.Errorf("vector: width and height must be at most 8192 (got %dx%d)", req.Width, req.Height)
	}
	c := newCanvas(req.Width, req.Height, 0, paramSet{values: map[string]float64{}})
	// One plane. An authored generator does not declare depth: separating a
	// composition into plates is a judgement about what sits behind what, and a
	// template that got it wrong would hand the plate model a stack that looks
	// right flattened and falls apart in parallax. The hand-written generators
	// declare their planes because their authors know the composition.
	c.plane("authored", 0)
	c.write("%s", req.Body)
	resolved, err := ResolveInks(c.document(), req.ID, req.Inks)
	if err != nil {
		return Result{}, err
	}
	sum := sha256.Sum256(resolved)
	return Result{
		SVG:    resolved,
		SHA256: hex.EncodeToString(sum[:]),
		Width:  req.Width,
		Height: req.Height,
		Planes: c.planeNames(),
	}, nil
}

// Noise and Fbm expose this package's deterministic fields to the authoring
// lane, so an authored generator's idea of a seed is the same as a hand-written
// one's rather than a second, subtly different noise.
func Noise(x, y float64, seed int64) float64 { return valueNoise(x, y, seed) }

func Fbm(x, y float64, octaves int, seed int64) float64 { return fbm(x, y, octaves, seed) }
