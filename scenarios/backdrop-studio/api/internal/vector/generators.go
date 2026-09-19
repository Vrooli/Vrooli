package vector

import (
	"fmt"
	"math"
	"strings"
)

// The four generators.
//
// Each is aimed at a specific failure in the 2026-08-13 verdict pass, and the
// aim is stated on the generator rather than left to a reader to infer:
//
//	colonnade         repairs cyanotype-arcade and engraved-colonnade
//	contour-relief    repairs swiss-contour and relief-plate
//	halftone-horizon  gives the horizon family a screened source drawn as a screen
//	radiant-orb       is the new art direction the resemblance clusters need
//
// The shared craft rule, which is what separates these from the raster lane:
// tone is carried by *line density*, never by a flat fill. A wall shaded with
// hatching that tightens where the light falls away has tone at every scale; a
// wall filled with one grey and then screened has tone at no scale, and that is
// the whole reason the screened raster styles read as texture with no picture.

// ── engraved colonnade ───────────────────────────────────────────────────

// drawColonnade draws an arcade the way an engraver would cut it: the wall is
// hatched, not filled, and the hatching follows the arch. The raster arcade
// generator draws the same subject as flat cut-out shapes — a plaster-coloured
// rectangle with holes in it — which is why a screen over it returns dots in
// the shape of rectangles.
func drawColonnade(c *canvas) {
	bays := int(c.params.get("bays", 1, 6, 3))
	horizon := c.params.get("horizon", 0.3, 0.8, 0.60) * c.h
	focal := c.params.get("focal_x", 0, 1, 0.5)
	c.inkWobble("cut", c.h*0.0016, 0.04, c.seed)

	// Plane 0 — the view through the arches: sky, sea, and a far headland.
	c.plane("distance", 0)
	c.write(`<rect width="%s" height="%s" fill="%s"/>`, f(c.w), f(c.h), InkPaper)

	// Sea: horizontal rules whose spacing opens toward the viewer and whose
	// weight lifts toward the horizon. Perspective in an engraving is carried by
	// the interval between lines, so this is the depth cue, not a gradient.
	seaLines := int(math.Max(18, c.h/26))
	var sea strings.Builder
	for i := 0; i < seaLines; i++ {
		t := float64(i) / float64(seaLines-1)
		y := horizon + math.Pow(t, 1.7)*(c.h-horizon)
		weight := c.h * (0.0007 + 0.0022*t)
		// The sun column: lines break where the light lies on the water.
		gap := c.w * 0.055 * (1 - t*0.45)
		cx := focal * c.w
		fmt.Fprintf(&sea, `<path d="M0 %s H%s" stroke="%s" stroke-width="%s" fill="none" stroke-linecap="round"/>`+"\n",
			f(y), f(cx-gap), InkInk, f(weight))
		fmt.Fprintf(&sea, `<path d="M%s %s H%s" stroke="%s" stroke-width="%s" fill="none" stroke-linecap="round"/>`+"\n",
			f(cx+gap), f(y), f(c.w), InkInk, f(weight))
	}
	c.write(`<g filter="url(#cut)">%s</g>`, sea.String())

	// The sun, drawn as a ring rather than a disc: an engraving states a light
	// source by leaving the paper alone and outlining it.
	sunY := horizon * 0.42
	sunR := c.h * 0.075
	c.write(`<circle cx="%s" cy="%s" r="%s" fill="none" stroke="%s" stroke-width="%s"/>`,
		f(focal*c.w), f(sunY), f(sunR), InkAccent, f(c.h*0.0026))
	for i := 0; i < 22; i++ {
		a := float64(i) / 22 * 2 * math.Pi
		inner := sunR * 1.22
		outer := sunR * (1.42 + 0.5*hash2(i, 3, c.seed))
		c.write(`<path d="M%s %s L%s %s" stroke="%s" stroke-width="%s" stroke-linecap="round"/>`,
			f(focal*c.w+math.Cos(a)*inner), f(sunY+math.Sin(a)*inner),
			f(focal*c.w+math.Cos(a)*outer), f(sunY+math.Sin(a)*outer),
			InkAccent, f(c.h*0.0016))
	}

	// Plane 1 — the far headland, hatched at a slant so it separates from the
	// sea's horizontals by direction as well as by tone.
	c.plane("headland", 1)
	c.write(`<path d="%s" fill="%s" fill-opacity="0.10" stroke="%s" stroke-width="%s"/>`,
		ridgePath(c, horizon, 64, 0.075, 0.012, c.seed+91), InkInk, InkInk, f(c.h*0.0016))

	// Plane 2 — the arcade. The wall is hatched; the arch openings are left as
	// paper. Hatch direction turns away from the light so the wall has a form.
	c.plane("arcade", 2)
	pad, gap := c.w*0.06, c.w*0.032
	span := (c.w - pad*2 - gap*float64(bays-1)) / float64(bays)
	archTop, archBottom := c.h*0.24, c.h*0.88
	radius := span / 2

	// The wall is one path: an outer rectangle with an opening cut out of it per
	// bay, resolved by the even-odd rule. Building the geometry once and using
	// it both as the visible shape and as the hatching's clip is what keeps the
	// hatching exactly inside the masonry — a second, separately-built clip
	// would drift from the wall the moment either changed.
	//
	// The masonry runs to the bottom of the frame, and only the openings stop
	// at the ledge. Ending the wall itself at archBottom left the bottom twelve
	// percent of the picture showing the sea plane straight through the
	// building — horizontal water lines crossing the full width beneath the
	// arcade, which reads as a glitch rather than as a view.
	var wall strings.Builder
	fmt.Fprintf(&wall, "M0 0 H%s V%s H0 Z", f(c.w), f(c.h))
	for i := 0; i < bays; i++ {
		ax := pad + float64(i)*(span+gap)
		cy := archTop + radius
		fmt.Fprintf(&wall, " M%s %s V%s A%s %s 0 0 1 %s %s V%s Z",
			f(ax), f(archBottom), f(cy), f(radius), f(radius), f(ax+span), f(cy), f(archBottom))
	}
	wallPath := wall.String()
	c.def(`<clipPath id="wall" clipPathUnits="userSpaceOnUse"><path d="%s" clip-rule="evenodd"/></clipPath>`, wallPath)
	c.write(`<path d="%s" fill="%s" fill-rule="evenodd"/>`, wallPath, InkPaper)

	// The hatching, clipped to the wall. The burin pitch is tight enough that
	// the masonry reads as a tone rather than as a set of visible stripes, and
	// the second, steeper pass only runs where the light falls away — which is
	// what turns a flat hatched panel into a wall with a form.
	var hatch strings.Builder
	pitch := c.h * 0.0072
	for pass := 0; pass < 2; pass++ {
		angle := -62.0
		if pass == 1 {
			angle = -22.0
		}
		slope := math.Tan(angle * math.Pi / 180)
		extent := c.w + math.Abs(slope)*c.h
		for x := -math.Abs(slope) * c.h; x < extent; x += pitch {
			shade := 1 - math.Min(1, math.Abs(x-focal*c.w)/(c.w*0.62))
			if pass == 1 && shade > 0.42 {
				continue // the lit side takes one pass only
			}
			// Weight rides the light too, so the shadowed masonry is not merely
			// hatched twice but cut deeper.
			weight := c.h * (0.0011 + 0.0013*(1-shade))
			fmt.Fprintf(&hatch, `<path d="M%s 0 L%s %s" stroke="%s" stroke-width="%s"/>`+"\n",
				f(x), f(x+slope*c.h), f(c.h), InkInk, f(weight))
		}
	}
	c.write(`<g clip-path="url(#wall)" filter="url(#cut)">%s</g>`, hatch.String())

	// The ledge the arches stand on: a heavier rule plus a band of horizontal
	// cuts, which stops the masonry reading as one undifferentiated panel from
	// the arch springing to the bottom edge.
	c.write(`<path d="M0 %s H%s" stroke="%s" stroke-width="%s"/>`, f(archBottom), f(c.w), InkInk, f(c.h*0.0034))
	var ledge strings.Builder
	for y := archBottom + c.h*0.012; y < c.h; y += c.h * 0.014 {
		t := (y - archBottom) / math.Max(1, c.h-archBottom)
		fmt.Fprintf(&ledge, `<path d="M0 %s H%s" stroke="%s" stroke-width="%s" opacity="%s"/>`+"\n",
			f(y), f(c.w), InkInk, f(c.h*0.0012), f(0.25+0.5*t))
	}
	c.write(`<g filter="url(#cut)">%s</g>`, ledge.String())

	// The statue in the centre bay, drawn as contour hatching over a silhouette
	// so it reads as carved rather than as a cut-out.
	stX, stBase := c.w*0.5, c.h*0.78
	stH := c.h * 0.30
	var body strings.Builder
	for i := 0; i <= 40; i++ {
		t := float64(i) / 40
		y := stBase - stH + t*stH
		halfW := c.w * 0.019 * (0.55 + 0.62*math.Sin(t*math.Pi*0.92+0.32))
		if i == 0 {
			fmt.Fprintf(&body, "M%s %s", f(stX-halfW), f(y))
		} else {
			fmt.Fprintf(&body, " L%s %s", f(stX-halfW), f(y))
		}
	}
	for i := 40; i >= 0; i-- {
		t := float64(i) / 40
		y := stBase - stH + t*stH
		halfW := c.w * 0.019 * (0.55 + 0.62*math.Sin(t*math.Pi*0.92+0.32))
		fmt.Fprintf(&body, " L%s %s", f(stX+halfW), f(y))
	}
	body.WriteString(" Z")
	c.def(`<clipPath id="statue"><path d="%s"/></clipPath>`, body.String())
	c.write(`<path d="%s" fill="%s"/>`, body.String(), InkPaper)
	var carve strings.Builder
	for y := stBase - stH; y < stBase; y += c.h * 0.0075 {
		t := (y - (stBase - stH)) / stH
		// The shadow side is hatched; the lit side is left as paper.
		fmt.Fprintf(&carve, `<path d="M%s %s H%s" stroke="%s" stroke-width="%s"/>`+"\n",
			f(stX+c.w*0.002), f(y), f(stX+c.w*0.02*(0.5+0.5*math.Sin(t*3))), InkInk, f(c.h*0.0012))
	}
	c.write(`<g clip-path="url(#statue)" filter="url(#cut)">%s</g>`, carve.String())
	c.write(`<path d="%s" fill="none" stroke="%s" stroke-width="%s"/>`, body.String(), InkInk, f(c.h*0.0018))
	c.write(`<rect x="%s" y="%s" width="%s" height="%s" fill="none" stroke="%s" stroke-width="%s"/>`,
		f(stX-c.w*0.033), f(stBase), f(c.w*0.066), f(c.h*0.075), InkInk, f(c.h*0.0018))

	// Plane 3 — the canopy: stippled foliage across the top edge, densest at the
	// corners so the centre stays open for a headline.
	c.plane("canopy", 3)
	var leaves strings.Builder
	// Leaves are drawn in clusters rather than as independent points. Scattered
	// single dots read as confetti; a canopy reads as a canopy because foliage
	// arrives in masses with gaps between them.
	const clusters = 90
	for k := 0; k < clusters; k++ {
		cx := hash2(k, 1, c.seed+61) * c.w
		edge := math.Min(cx, c.w-cx) / (c.w * 0.5)
		reach := c.h * (0.34 - 0.20*edge)
		cy := hash2(k, 2, c.seed+61) * reach * 0.75
		spread := c.h * (0.018 + 0.032*hash2(k, 5, c.seed+61))
		for i := 0; i < 26; i++ {
			a := hash2(k*31+i, 7, c.seed+61) * 2 * math.Pi
			d := spread * math.Sqrt(hash2(k*31+i, 11, c.seed+61))
			x, y := cx+math.Cos(a)*d*1.6, cy+math.Sin(a)*d
			if y < 0 || y > reach {
				continue
			}
			density := 1 - y/reach
			r := c.h * (0.0018 + 0.0038*density)
			fmt.Fprintf(&leaves, `<circle cx="%s" cy="%s" r="%s" fill="%s"/>`+"\n", f(x), f(y), f(r), InkInk)
		}
	}
	c.write(`<g filter="url(#cut)">%s</g>`, leaves.String())
}

// ── contour relief ───────────────────────────────────────────────────────

// peakLayout is where the land rises, in normalised frame coordinates.
//
// The composition is deliberately weighted right and low. A backdrop carries
// overlay copy in its upper-left third — that is where every seeded style puts
// its reserved region — so relief there competes with the headline for the
// reader's attention and the gate measures it as texture in the quiet zone.
// Keeping the summits out of that quadrant leaves a genuinely open ground for
// type without needing a scrim over the drawing.
type peak struct{ x, y, r, a float64 }

var peakLayout = []peak{
	{0.66, 0.56, 0.24, 1.00},
	{0.86, 0.32, 0.15, 0.74},
	{0.44, 0.80, 0.18, 0.66},
}

// referenceAspect is the frame the composition above is art-directed against.
const referenceAspect = 1.7

// reliefPeaks returns the terrain for a frame of a given aspect.
//
// A wider plate shows MORE land, not the same land stretched. That is how a map
// behaves, and it is also the only way this style survives its own gate at
// every surface it is offered on. The first version sampled peaks in normalised
// frame coordinates with the aspect correction hardcoded to 1.7, which was
// right for the 1440x900 the render matrix uses and wrong everywhere else: at
// `web.hero` (2:1) the same three hills covered proportionally less of the
// frame and frequency modulation fell to 0.028 against a 0.030 floor, and at
// `web.section-band` (3.4:1) tonal occupancy collapsed to 0.277 against 0.40.
// Three of eighteen seeded surfaces refused the style, and the matrix could not
// see it because it renders one geometry.
//
// Peaks are round in pixel space — a hill is round — so the count scales with
// the frame's area in short-edge units rather than the radii scaling with the
// width. Extra summits are placed on a golden-ratio sequence, which distributes
// them without clustering and without a random number generator, and they are
// kept out of the upper-left copy zone for the same reason the base three are.
func reliefPeaks(aspect float64) []peak {
	peaks := append([]peak(nil), peakLayout...)
	// The base composition's density, held constant as the frame grows.
	density := float64(len(peakLayout)) / referenceAspect
	want := int(math.Round(density * aspect))
	// The golden angle in one dimension: successive terms of the fractional
	// part of n*φ are maximally spread, so each new summit lands in the widest
	// remaining gap.
	const phi = 0.6180339887498949
	for n := 1; len(peaks) < want; n++ {
		x := math.Mod(0.5+float64(n)*phi, 1)
		// A second, faster orbit for the vertical so summits do not fall on a
		// line, biased low to stay clear of the overlay band.
		y := 0.42 + 0.5*math.Mod(float64(n)*phi*3, 1)
		if x < 0.46 && y < 0.44 {
			// The copy zone. Push it right rather than dropping it, so the
			// count stays a function of area alone.
			x += 0.46
			if x > 1 {
				x--
			}
		}
		peaks = append(peaks, peak{
			x: x, y: y,
			r: 0.13 + 0.09*math.Mod(float64(n)*phi*5, 1),
			a: 0.58 + 0.34*math.Mod(float64(n)*phi*7, 1),
		})
	}
	return peaks
}

// drawContourRelief draws a survey plate: nested contour rings around real
// peaks, with index contours drawn heavier every fifth line.
//
// This is what `swiss-contour` and `relief-plate` were supposed to be. Both
// currently deliver a screen with no map under it, because a raster contour
// field drawn at one pixel and then screened at a coarser ruling loses every
// line it had. Here the line IS the picture, so there is nothing to lose.
func drawContourRelief(c *canvas) {
	bands := int(c.params.get("bands", 8, 40, 24))
	relief := c.params.get("relief", 0.2, 1, 0.72)
	c.inkWobble("pen", c.h*0.0011, 0.05, c.seed+5)

	c.plane("paper", 0)
	c.write(`<rect width="%s" height="%s" fill="%s"/>`, f(c.w), f(c.h), InkPaper)

	// Height field: a few peaks plus noise, sampled on a grid and traced by
	// marching squares. Tracing real iso-lines rather than drawing decorative
	// concentric blobs is what makes the plate read as a survey.
	const gx, gy = 160, 100
	// Distance is measured in short-edge units, so a hill is round in pixels at
	// any frame shape. The correction used to be the literal 1.7, which drew
	// correctly circular hills on a 1.7:1 frame and progressively flatter
	// ellipses on everything else.
	aspect := c.w / c.h
	peaks := reliefPeaks(aspect)
	height := make([][]float64, gy+1)
	for j := 0; j <= gy; j++ {
		height[j] = make([]float64, gx+1)
		for i := 0; i <= gx; i++ {
			u := float64(i) / gx
			v := float64(j) / gy
			// The noise is the *ground*, not the subject. At its original
			// amplitude it put contour lines across every square inch of the
			// frame at roughly equal density, and the perceptual gate refused
			// the style for exactly that: frequency modulation 0.022 against a
			// 0.030 floor, which is the gate's way of saying "uniformly busy is
			// noise, not a drawing". A survey plate is mostly quiet ground with
			// relief where the land actually rises, so the noise is damped and
			// the peaks carry the composition.
			// The noise is sampled over a domain that grows with the frame,
			// so its cells stay square and a wide plate shows more ground
			// rather than a horizontally smeared version of the same ground.
			h := fbm(u*2*aspect, v*3.4, 5, c.seed+13) * 0.11
			for _, p := range peaks {
				d := math.Hypot((u-p.x)*aspect, v-p.y) / p.r
				h += p.a * math.Exp(-d*d*1.6)
			}
			height[j][i] = h * relief
		}
	}

	c.plane("contours", 1)
	var lines strings.Builder
	// The lowest contour sits above the noise ceiling, so flat ground draws no
	// line at all and every band belongs to a peak.
	//
	// This is what a survey plate actually looks like — a plain carries no
	// contours — and it is also what the perceptual gate is asking for. With
	// the first band below the noise, the plate crossed four levels across its
	// whole flat ground and the frame came out uniformly busy: frequency
	// modulation 0.022, then 0.027, against a 0.030 floor. Uniform busy-ness is
	// texture, and texture edge to edge is the same defect the screened raster
	// styles have. Concentrating the bands on the relief is the repair at
	// cause; loosening the floor would only have hidden it.
	lo, hi := 0.16*relief, 1.15*relief
	for b := 0; b < bands; b++ {
		level := lo + (hi-lo)*float64(b)/float64(bands-1)
		index := b%5 == 0
		weight := c.h * 0.0011
		if index {
			weight = c.h * 0.0026
		}
		var segments strings.Builder
		for j := 0; j < gy; j++ {
			for i := 0; i < gx; i++ {
				x0, y0 := float64(i)/gx*c.w, float64(j)/gy*c.h
				x1, y1 := float64(i+1)/gx*c.w, float64(j+1)/gy*c.h
				a, bb := height[j][i], height[j][i+1]
				cc, d := height[j+1][i+1], height[j+1][i]
				appendIsoSegment(&segments, level, x0, y0, x1, y1, a, bb, cc, d)
			}
		}
		if segments.Len() == 0 {
			continue
		}
		fmt.Fprintf(&lines, `<g stroke="%s" stroke-width="%s" fill="none" stroke-linecap="round">%s</g>`+"\n",
			InkInk, f(weight), segments.String())
	}
	c.write(`<g filter="url(#pen)">%s</g>`, lines.String())

	// Plane 2 — the survey furniture: a graticule and spot heights, which is
	// what makes it a plate rather than a pattern.
	c.plane("survey", 2)
	for i := 1; i < 8; i++ {
		x := float64(i) / 8 * c.w
		c.write(`<path d="M%s 0 V%s" stroke="%s" stroke-width="%s" stroke-dasharray="%s %s" opacity="0.35"/>`,
			f(x), f(c.h), InkAccent, f(c.h*0.0009), f(c.h*0.006), f(c.h*0.012))
	}
	for j := 1; j < 5; j++ {
		y := float64(j) / 5 * c.h
		c.write(`<path d="M0 %s H%s" stroke="%s" stroke-width="%s" stroke-dasharray="%s %s" opacity="0.35"/>`,
			f(y), f(c.w), InkAccent, f(c.h*0.0009), f(c.h*0.006), f(c.h*0.012))
	}
	for _, p := range peaks {
		px, py := p.x*c.w, p.y*c.h
		s := c.h * 0.008
		c.write(`<path d="M%s %s L%s %s M%s %s L%s %s" stroke="%s" stroke-width="%s"/>`,
			f(px-s), f(py), f(px+s), f(py), f(px), f(py-s), f(px), f(py+s), InkAccent, f(c.h*0.0022))
	}
}

// ridgePath builds a closed silhouette rising from a baseline, as path data.
//
// It returns the `d` attribute's contents rather than a whole element because
// the two callers use it differently — one strokes and fills it, the other
// fills it at partial opacity — and a helper that returned markup would force
// its callers to string-edit the element it produced.
func ridgePath(c *canvas, baseline float64, steps int, amplitude, lift float64, seed int64) string {
	var out strings.Builder
	fmt.Fprintf(&out, "M0 %s", f(baseline))
	for i := 0; i <= steps; i++ {
		x := float64(i) / float64(steps) * c.w
		n := fbm(float64(i)/9+3, 1, 4, seed)
		fmt.Fprintf(&out, " L%s %s", f(x), f(baseline-n*c.h*amplitude-c.h*lift))
	}
	fmt.Fprintf(&out, " L%s %s Z", f(c.w), f(baseline))
	return out.String()
}

// appendIsoSegment emits the marching-squares segment for one cell, if the
// contour level crosses it. Corners are (a,b,c,d) clockwise from top-left.
func appendIsoSegment(out *strings.Builder, level, x0, y0, x1, y1, a, b, cc, d float64) {
	type pt struct{ x, y float64 }
	var crossings []pt
	edge := func(va, vb float64, ax, ay, bx, by float64) {
		if (va < level) == (vb < level) {
			return
		}
		t := (level - va) / (vb - va)
		crossings = append(crossings, pt{ax + (bx-ax)*t, ay + (by-ay)*t})
	}
	edge(a, b, x0, y0, x1, y0)
	edge(b, cc, x1, y0, x1, y1)
	edge(cc, d, x1, y1, x0, y1)
	edge(d, a, x0, y1, x0, y0)
	for i := 0; i+1 < len(crossings); i += 2 {
		fmt.Fprintf(out, `<path d="M%s %s L%s %s"/>`,
			f(crossings[i].x), f(crossings[i].y), f(crossings[i+1].x), f(crossings[i+1].y))
	}
}

// ── radiant orb ──────────────────────────────────────────────────────────

// drawRadiantOrb is the poster language: a colossal disc textured with
// concentric rings, a radial burst behind it, a hairline horizon and a single
// small figure for scale.
//
// It exists because the resemblance report found three clusters and the repair
// for a cluster is a genuinely different source, not a recolour. Nothing in the
// raster lane draws this — it is line work at every scale, and a raster
// generator would have to draw it at one size and then watch a screen destroy
// it, which is the defect this whole package answers.
func drawRadiantOrb(c *canvas) {
	orbR := c.params.get("orb_radius", 0.15, 0.6, 0.34) * math.Min(c.w, c.h) * 1.6
	orbX := c.params.get("focal_x", 0, 1, 0.52) * c.w
	orbY := c.params.get("focal_y", 0, 1, 0.44) * c.h
	groundY := c.params.get("ground", 0.6, 0.98, 0.86) * c.h
	c.inkWobble("scratch", c.h*0.0022, 0.03, c.seed+21)

	// Plane 0 — the ground of the poster and its star field.
	c.plane("void", 0)
	c.write(`<rect width="%s" height="%s" fill="%s"/>`, f(c.w), f(c.h), InkInk)
	var stars strings.Builder
	for i := 0; i < 260; i++ {
		x := hash2(i, 7, c.seed+3) * c.w
		y := hash2(i, 11, c.seed+3) * groundY
		if math.Hypot(x-orbX, y-orbY) < orbR*1.04 {
			continue
		}
		s := c.h * (0.0012 + 0.0026*hash2(i, 13, c.seed+3))
		// Four-point stars, the way an engraver marks one.
		fmt.Fprintf(&stars, `<path d="M%s %s L%s %s M%s %s L%s %s" stroke="%s" stroke-width="%s" stroke-linecap="round" opacity="%s"/>`+"\n",
			f(x-s), f(y), f(x+s), f(y), f(x), f(y-s), f(x), f(y+s),
			InkPaper, f(c.h*0.0011), f(0.35+0.6*hash2(i, 17, c.seed+3)))
	}
	c.write("%s", stars.String())

	// Plane 1 — the burst: rays of varying length and weight radiating from the
	// disc, which is what gives the composition its energy.
	c.plane("burst", 1)
	var rays strings.Builder
	const rayCount = 132
	for i := 0; i < rayCount; i++ {
		a := float64(i) / rayCount * 2 * math.Pi
		jitter := hash2(i, 23, c.seed+9)
		inner := orbR * 1.02
		outer := orbR * (1.10 + 1.35*jitter*jitter)
		fmt.Fprintf(&rays, `<path d="M%s %s L%s %s" stroke="%s" stroke-width="%s" stroke-linecap="round" opacity="%s"/>`+"\n",
			f(orbX+math.Cos(a)*inner), f(orbY+math.Sin(a)*inner),
			f(orbX+math.Cos(a)*outer), f(orbY+math.Sin(a)*outer),
			InkPaper, f(c.h*(0.0008+0.0018*jitter)), f(0.20+0.55*jitter))
	}
	c.write(`<g filter="url(#scratch)">%s</g>`, rays.String())

	// Plane 2 — the disc. Filled with paper, then textured with concentric
	// rings whose radius wanders, which is how the reference plates draw a moon.
	c.plane("orb", 2)
	c.write(`<circle cx="%s" cy="%s" r="%s" fill="%s"/>`, f(orbX), f(orbY), f(orbR), InkPaper)
	c.def(`<clipPath id="orb"><circle cx="%s" cy="%s" r="%s"/></clipPath>`, f(orbX), f(orbY), f(orbR))
	var rings strings.Builder
	for i := 0; i < 46; i++ {
		t := float64(i) / 46
		r := orbR * (0.06 + 0.96*t)
		// Ring centres drift, so the rings read as drawn by hand rather than
		// struck with a compass.
		dx := (hash2(i, 31, c.seed+15) - 0.5) * orbR * 0.16
		dy := (hash2(i, 37, c.seed+15) - 0.5) * orbR * 0.16
		fmt.Fprintf(&rings, `<circle cx="%s" cy="%s" r="%s" fill="none" stroke="%s" stroke-width="%s" opacity="%s"/>`+"\n",
			f(orbX+dx), f(orbY+dy), f(r), InkInk, f(c.h*0.0011), f(0.20+0.45*hash2(i, 41, c.seed+15)))
	}
	// A few craters: rings tight enough to read as depressions.
	for i := 0; i < 5; i++ {
		a := hash2(i, 43, c.seed+19) * 2 * math.Pi
		d := (0.18 + 0.62*hash2(i, 47, c.seed+19)) * orbR
		cx, cy := orbX+math.Cos(a)*d, orbY+math.Sin(a)*d
		cr := orbR * (0.04 + 0.09*hash2(i, 53, c.seed+19))
		for k := 0; k < 5; k++ {
			fmt.Fprintf(&rings, `<circle cx="%s" cy="%s" r="%s" fill="none" stroke="%s" stroke-width="%s" opacity="0.55"/>`+"\n",
				f(cx), f(cy), f(cr*(0.25+0.19*float64(k))), InkInk, f(c.h*0.0012))
		}
	}
	c.write(`<g clip-path="url(#orb)" filter="url(#scratch)">%s</g>`, rings.String())

	// Plane 3 — the ground line and the figure. The figure is small on purpose:
	// scale is the device that makes the disc read as colossal.
	c.plane("ground", 3)
	c.write(`<path d="M0 %s H%s" stroke="%s" stroke-width="%s"/>`, f(groundY), f(c.w), InkPaper, f(c.h*0.0022))
	var tickmarks strings.Builder
	for x := 0.0; x < c.w; x += c.w * 0.012 {
		h := c.h * (0.004 + 0.008*hash2(int(x), 59, c.seed+23))
		fmt.Fprintf(&tickmarks, `<path d="M%s %s V%s" stroke="%s" stroke-width="%s" opacity="0.5"/>`+"\n",
			f(x), f(groundY), f(groundY+h), InkPaper, f(c.h*0.0009))
	}
	c.write("%s", tickmarks.String())
	figX := c.params.get("figure_x", 0, 1, 0.30) * c.w
	figH := c.h * 0.055
	c.write(`<g fill="%s">
<circle cx="%s" cy="%s" r="%s"/>
<path d="M%s %s L%s %s L%s %s Z"/>
</g>`,
		InkPaper,
		f(figX), f(groundY-figH*0.86), f(figH*0.15),
		f(figX-figH*0.20), f(groundY), f(figX+figH*0.20), f(groundY), f(figX), f(groundY-figH*0.72))
}

// ── halftone horizon ─────────────────────────────────────────────────────

// drawHalftoneHorizon draws a horizon as a halftone: the dots are the drawing,
// their radius set from the tone at each cell.
//
// The distinction from the raster lane matters and is easy to miss. There, a
// horizon is rendered as pixels and image-tools then screens it — two steps,
// and the second one can only remove what the first produced. Here the tone
// function is evaluated directly at each screen cell, so the dot grid IS the
// image and no information is lost between them. That is why this holds its
// composition at 240px and at 2732px while the raster version does not.
func drawHalftoneHorizon(c *canvas) {
	horizon := c.params.get("horizon", 0.25, 0.8, 0.56) * c.h
	focal := c.params.get("focal_x", 0, 1, 0.68)
	// Cells across the frame. Resolution-independent by construction: the grid
	// is a count, so the screen is the same coarseness at every delivery size.
	cells := int(c.params.get("cells", 40, 220, 132))
	pitch := c.w / float64(cells)
	sunX, sunY := focal*c.w, horizon*0.40
	sunR := c.h * 0.10
	// A press does not lay a screen down perfectly. The dots wander by a
	// fraction of their own pitch, which is the difference between a halftone
	// that reads as printed and one that reads as computed — and it is the
	// quality the reference material has that a mathematically exact grid does
	// not. The displacement is small enough to preserve the tone and large
	// enough to break the grid's regularity.
	c.inkWobble("press", pitch*0.30, 0.16, c.seed+31)

	c.plane("paper", 0)
	c.write(`<rect width="%s" height="%s" fill="%s"/>`, f(c.w), f(c.h), InkPaper)

	// tone returns 0 (paper) to 1 (solid ink) at a point.
	tone := func(x, y float64) float64 {
		switch {
		case y < horizon:
			// Sky: darkens upward, opens around the sun.
			base := 0.62 * math.Pow(1-y/horizon, 1.25)
			glow := math.Exp(-math.Hypot(x-sunX, y-sunY) / (sunR * 1.5))
			if math.Hypot(x-sunX, y-sunY) < sunR {
				return 0
			}
			return math.Max(0, base-glow*0.85)
		default:
			// Sea: deepens with distance from the horizon, with a light column
			// under the sun and a coherent ripple.
			t := (y - horizon) / math.Max(1, c.h-horizon)
			base := 0.30 + 0.55*t
			column := math.Exp(-math.Abs(x-sunX)/(c.w*0.05)) * (1 - t*0.6)
			ripple := (fbm(x/(c.w*0.03), y/(c.h*0.012), 3, c.seed+11) - 0.5) * 0.22
			return math.Max(0, math.Min(1, base-column*0.8+ripple))
		}
	}

	// Plane 1 — the sky screen.
	c.plane("sky", 1)
	c.write(`<g fill="%s" filter="url(#press)">%s</g>`, InkInk, halftoneCells(c, pitch, 0, horizon, tone))

	// Plane 2 — the headland, a solid mass at the horizon that separates the
	// two screens by silhouette rather than by density alone.
	c.plane("headland", 2)
	c.write(`<path d="%s" fill="%s" fill-opacity="0.72"/>`,
		ridgePath(c, horizon, 72, 0.06, 0.008, c.seed+77), InkInk)

	// Plane 3 — the sea screen and the sun's ring.
	c.plane("sea", 3)
	c.write(`<g fill="%s" filter="url(#press)">%s</g>`, InkInk, halftoneCells(c, pitch, horizon, c.h, tone))
	c.write(`<circle cx="%s" cy="%s" r="%s" fill="none" stroke="%s" stroke-width="%s"/>`,
		f(sunX), f(sunY), f(sunR), InkAccent, f(c.h*0.0030))
}

// halftoneCells evaluates the tone function on a square grid and emits one dot
// per cell, sized from the tone. A cell whose tone rounds to nothing emits no
// element at all, which keeps the document proportional to the ink rather than
// to the grid.
func halftoneCells(c *canvas, pitch, top, bottom float64, tone func(x, y float64) float64) string {
	var out strings.Builder
	rows := int(math.Ceil((bottom - top) / pitch))
	cols := int(math.Ceil(c.w / pitch))
	maxR := pitch * 0.72
	for j := 0; j < rows; j++ {
		y := top + (float64(j)+0.5)*pitch
		for i := 0; i < cols; i++ {
			// Alternate rows offset by half a cell: a 45-degree screen is what
			// a printer uses because a square grid reads as a grid.
			x := (float64(i)+0.5)*pitch + float64(j%2)*pitch*0.5
			if x > c.w+pitch {
				continue
			}
			v := tone(x, y)
			if v <= 0.02 {
				continue
			}
			r := maxR * math.Sqrt(math.Min(1, v))
			if r < pitch*0.06 {
				continue
			}
			fmt.Fprintf(&out, `<circle cx="%s" cy="%s" r="%s"/>`, f(x), f(y), f(r))
		}
	}
	return out.String()
}
