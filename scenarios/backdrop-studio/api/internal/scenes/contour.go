package scenes

import "math"

// drawContour renders topographic isolines over a height field.
//
// It depicts `cartographic` and nothing else. Before this generator existed the
// catalog's one cartographic style rendered the terrain scene under a line
// screen — a picture of mountains wearing stripes, which is not a map. A
// contour map is a different object: the lines are level sets of a height
// field, so they never cross, they close on themselves, and they crowd exactly
// where the ground is steep. That last property is the whole reason the look
// carries information, and it is what a screen laid over a picture cannot fake.
//
// The interesting part is the line width. A naive test — "draw ink where the
// fractional part of height×bands is small" — gives lines that are hairline on
// a cliff and a broad band on a plain, because the same height interval covers
// a different number of pixels. Dividing the distance-to-level by the local
// gradient magnitude converts height distance into screen distance, so every
// line is the same width wherever it falls. It also antialiases for free: the
// ratio is a continuous measure of how far the pixel is from the line, in
// pixels, which is precisely what a coverage value should be.
func drawContour(c *canvas, p paramSet, seed int64) {
	fw, fh := float64(c.w), float64(c.h)
	focalX := p.get("focal_x", 0, 1, 0.62)
	focalY := p.get("focal_y", 0, 1, 0.44)
	// bands is the number of contour intervals across the full height range.
	bands := p.get("bands", 4, 48, 14)
	// relief blends smooth fbm toward ridged noise: 0 is rolling ground, 1 is
	// a range with sharp watersheds and tightly packed lines along the crests.
	relief := p.get("relief", 0, 1, 0.55)
	// line_weight is the ink width in pixels of the short edge per 1000, so a
	// map drawn at 390px and at 2796px has lines of the same visual weight.
	lineWeight := p.get("line_weight", 0.4, 6, 2.4)
	// index_every marks every Nth line as an index contour, drawn heavier —
	// the convention that makes a real map readable at a glance.
	indexEvery := math.Max(2, math.Round(p.get("index_every", 2, 12, 5)))
	// ink_strength is how far a line travels from the fill it sits on.
	//
	// It is deliberately low, and that is art direction rather than timidity.
	// This style's role is `ambient`: the image is the stage and attention
	// passes through it to the copy. A survey sheet's iron-gall ink on white
	// paper is a contrast of roughly 0.85, which behind a headline is not a
	// backdrop but a competing drawing. The elevation fill carries the tonal
	// range instead — it is continuous, so it costs nothing in edge energy.
	inkStrength := p.get("ink_strength", 0.05, 1, 0.34)
	palette := int(p.get("palette", 0, 1, 0))

	short := math.Min(fw, fh)
	// The noise scale is a fraction of the short edge, so the map shows the
	// same amount of country at every delivery size.
	scale := 1 / (short * 0.42)

	// The height field is built once into a buffer rather than sampled five
	// times per pixel, and then stretched to fill its own range.
	//
	// The stretch is what makes the interval mean anything. Composed of fBm
	// plus a dome, the raw field occupies roughly the middle 60% of [0,1] and
	// where it lands inside that depends on the seed — so a fixed ramp put the
	// valley floors at luminance 0.24 and the summits at 0.78, and the map had
	// neither a shadow nor a highlight for a treatment to map ink into. After
	// the stretch, `bands` is the number of intervals across the relief that is
	// actually there, which is what a contour interval is on a real sheet.
	hbuf := make([]float64, c.w*c.h)
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			fx, fy := float64(x), float64(y)
			u, v := fx*scale, fy*scale
			smooth := fbm(u, v, 5, seed+11)
			sharp := ridged(u*1.15, v*1.15, 5, seed+12)
			h := smooth*(1-relief) + sharp*relief
			// A broad dome centred on the focal point. It gives the map a
			// summit rather than an even scatter of hills, so the composition
			// has somewhere to look.
			dx := (fx/fw - focalX) * 1.6
			dy := (fy/fh - focalY) * 1.6
			dome := math.Max(0, 1-math.Hypot(dx, dy))
			hbuf[y*c.w+x] = h*0.68 + dome*dome*0.42
		}
	}
	lo, hi := percentile(hbuf, 0.005), percentile(hbuf, 0.995)
	if hi-lo < 1e-6 {
		hi = lo + 1
	}
	for i, v := range hbuf {
		hbuf[i] = clamp01((v - lo) / (hi - lo))
	}
	heightAt := func(x, y int) float64 {
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}
		if x >= c.w {
			x = c.w - 1
		}
		if y >= c.h {
			y = c.h - 1
		}
		return hbuf[y*c.w+x]
	}

	low, mid, high, ink := contourPalette(palette)

	// A line thinner than a pixel cannot be drawn: it rasterises to scattered
	// fragments, and the renderer compensates by never quite turning the line
	// off, so the map gets *more* ink the smaller it is drawn. Measured before
	// this clamp, mean luminance moved 0.08 between a 320px and a 960px render
	// of the same seed — the same class of defect the treatment layer's
	// halftone cell floor exists to prevent, in a different generator.
	//
	// The clamp holds the line at one pixel and reduces the interval count in
	// proportion, so the total ink coverage stays what the style asked for and
	// the map simply shows fewer contours where it has no room for more.
	const minContourInkPx = 1.0
	weight := lineWeight * short / 1000
	if weight < minContourInkPx {
		bands = math.Max(3, bands*weight/minContourInkPx)
		weight = minContourInkPx
	}

	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			h := heightAt(x, y)

			// The elevation fill is continuous, not quantised into layers.
			//
			// Quantising it was the first version and it looked like a map, but
			// it put a hard step at every band boundary on top of the line
			// already drawn there — doubling the edge energy for no information,
			// since the line marks the same level the step did. It is also what
			// pushed the mean neighbour delta to 0.109 and made this generator
			// read as line art rather than as a scene. Continuous fill costs
			// nothing in edges and carries the whole tonal range on its own.
			base := mixRGB(low, mid, smoothstep(clamp01(h/0.62)))
			base = mixRGB(base, high, smoothstep(clamp01((h-0.60)/0.40)))

			// Screen-space distance to the nearest contour level. The gradient
			// is a central difference over one pixel, which is the unit the
			// line width is expressed in.
			t := h * bands
			gx := (heightAt(x+1, y) - heightAt(x-1, y)) * bands / 2
			gy := (heightAt(x, y+1) - heightAt(x, y-1)) * bands / 2
			grad := math.Hypot(gx, gy)
			if grad < 1e-6 {
				// Dead flat: no line can be placed, and dividing by the
				// gradient here would paint the whole plateau with ink.
				c.set(x, y, base[0], base[1], base[2])
				continue
			}
			level := math.Round(t)
			distPx := math.Abs(t-level) / grad

			w := weight
			strength := inkStrength
			if math.Mod(math.Abs(level), indexEvery) < 0.5 {
				// The index contour is the one a reader counts from. It is
				// heavier and darker, which is the whole convention.
				w *= 2.1
				strength *= 1.5
			}
			// One pixel of feather on each side. Without it the lines alias,
			// and a halftone or dither downstream turns that aliasing into the
			// crawling texture this scenario spent a phase removing.
			cover := 1 - smoothstep(clamp01((distPx-w*0.5)/1.0))

			col := mixRGB(base, ink, cover*math.Min(1, strength))
			c.set(x, y, col[0], col[1], col[2])
		}
	}
}

// contourPalette returns the three stops of the elevation fill, lowest first,
// and the contour ink.
//
// The fill has to reach both ends of the tonal range on its own, because it is
// the only continuous quantity in the picture: the lines are deliberately
// low-contrast, so if the fill sits in the midtones the whole frame does, and
// every ink-mapping treatment downstream has a third of the ramp to work with.
func contourPalette(index int) (low, mid, high, ink [3]float64) {
	if index == 1 {
		// Night chart: the deep-water-to-ice ramp, and a pale ink. The register
		// infrastructure and observability products use.
		return [3]float64{8, 14, 22}, [3]float64{28, 92, 108},
			[3]float64{226, 246, 244}, [3]float64{196, 232, 228}
	}
	// Survey sheet: shadowed valley floor through sand to snow, drawn in an
	// iron-gall ink.
	return [3]float64{26, 30, 34}, [3]float64{178, 166, 138},
		[3]float64{250, 248, 242}, [3]float64{34, 38, 44}
}
