package scenes

import "math"

// drawAttractor plots the trajectory of a chaotic map and shades by how often
// it passes through each point.
//
// The map is Peter de Jong's:
//
//	x' = sin(a·y) − cos(b·x)
//	y' = sin(c·x) − cos(d·y)
//
// Four numbers and a starting point produce filamentary structure with cusps,
// folds and caustic-bright edges that no noise function generates — the density
// is genuinely singular along the fold lines, which is why the highlights are
// thin and sharp while the body of the picture stays soft.
//
// Nothing here is random. The trajectory is fully determined by the four
// coefficients, and the seed only chooses them, so the same seed is the same
// picture forever — the provenance property the whole pipeline rests on.
//
// The coefficients are jittered around a known-good set rather than drawn
// freely. Most of the parameter space is degenerate: it collapses to a point, a
// closed loop, or a full-frame smear, none of which is a backdrop. A generator
// that produces a usable picture for one seed in five is not a generator; it is
// a lottery, and the catalog would have to pin seeds to hide it.
func drawAttractor(c *canvas, p paramSet, seed int64) {
	fw, fh := float64(c.w), float64(c.h)
	// variation moves the coefficients within the well-behaved neighbourhood.
	// It is separate from the seed so a style can hold a look and still vary
	// its render, which is what the variation grid needs.
	variation := p.get("variation", 0, 1, 1)
	// density scales the iteration count relative to the frame's area.
	density := p.get("density", 0.25, 3, 1)
	// zoom frames the attractor. Below 1 leaves margin around it; above 1 crops
	// into the structure.
	//
	// The default crops. Framed whole, the attractor is an object sitting in
	// the middle of an empty field — which is a poster, not a backdrop: an
	// ambient image should run off its own edges so nothing in it competes with
	// the copy for the role of subject.
	zoom := p.get("zoom", 0.5, 2.6, 1.55)
	// focal_x and focal_y place the attractor's centre in the frame, so the
	// dense part can sit away from where the copy goes.
	focalX := p.get("focal_x", 0, 1, 0.62)
	focalY := p.get("focal_y", 0, 1, 0.44)
	palette := int(p.get("palette", 0, 1, 0))

	// The de Jong attractor lives in [-2,2]²; these coefficients are a stable
	// starting point and the jitter stays inside the range that keeps the
	// structure open.
	jitter := func(k int, spread float64) float64 {
		return (hash2(k, 17, seed) - 0.5) * 2 * spread * variation
	}
	a := 1.40 + jitter(1, 0.22)
	b := -2.30 + jitter(2, 0.22)
	cc := 2.40 + jitter(3, 0.22)
	d := -2.10 + jitter(4, 0.22)

	const gridShort = 640
	gw, gh := gridShort, gridShort
	if fw > fh {
		gw = int(float64(gridShort) * fw / fh)
	} else {
		gh = int(float64(gridShort) * fh / fw)
	}
	gwf, ghf := float64(gw), float64(gh)
	short := math.Min(gwf, ghf)

	accum := make([]float64, gw*gh)
	// Iterations scale with the grid's area so a wide frame is not a fainter
	// picture than a square one, and so the mean density is the same at every
	// delivery size.
	iterations := int(float64(gw*gh) * 6 * density)

	// The map's own range is [-2,2] in both axes; it is mapped to the short
	// edge so a wide frame shows more space around the attractor rather than
	// stretching it.
	scale := short / 4.4 * zoom
	offX, offY := focalX*gwf, focalY*ghf

	x, y := 0.1, 0.1
	// The first iterations are on the transient approach to the attractor and
	// would leave a visible thread from the starting point to the structure.
	for i := 0; i < 1000; i++ {
		x, y = math.Sin(a*y)-math.Cos(b*x), math.Sin(cc*x)-math.Cos(d*y)
	}
	for i := 0; i < iterations; i++ {
		x, y = math.Sin(a*y)-math.Cos(b*x), math.Sin(cc*x)-math.Cos(d*y)
		px := offX + x*scale
		py := offY + y*scale
		ix, iy := int(px), int(py)
		if ix < 0 || iy < 0 || ix >= gw || iy >= gh {
			continue
		}
		accum[iy*gw+ix]++
	}

	// A one-pixel-wide filament is speckle by the coherence measure and, more
	// importantly, samples at random under a fine screen. The blur is a
	// fraction of the grid, so filaments have the same body at every size.
	accum = blurBuffer(accum, gw, gh, int(math.Max(1, short/260)))

	// The fold lines are genuinely singular, so the maximum is orders of
	// magnitude above the body of the distribution. Normalising against it
	// would leave the picture black with a few bright threads.
	//
	// The percentile is 0.98 rather than 0.995 because the attractor occupies
	// only part of the frame: the empty space counts in the distribution, so a
	// percentile chosen for a full-frame generator lands far out on the tail
	// here and the folds never reach the top of the ramp. Measured at 0.995 the
	// 98th-percentile luminance of the finished scene was 0.571 — no highlight,
	// and a screen over it comes out uniformly grey.
	norm := percentile(accum, 0.98)
	if norm <= 0 {
		norm = 1
	}

	ground, body, filament, core := attractorPalette(palette)

	for py := 0; py < c.h; py++ {
		for px := 0; px < c.w; px++ {
			gx := float64(px) / fw * float64(gw-1)
			gy := float64(py) / fh * float64(gh-1)
			v := clamp01(sampleBilinear(accum, gw, gh, gx, gy) / norm)
			// Two gammas rather than one: the first opens the faint outer haze
			// so the frame is not empty, the second keeps the fold lines from
			// blowing out into a solid mass.
			haze := math.Pow(v, 0.38)
			bright := math.Pow(v, 1.6)

			col := mixRGB(ground, body, haze)
			col = mixRGB(col, filament, math.Pow(v, 0.75))
			col = mixRGB(col, core, bright)
			c.set(px, py, col[0], col[1], col[2])
		}
	}
}

// attractorPalette returns the empty ground, the haze, the filament ink and the
// core highlight, darkest first.
func attractorPalette(index int) (ground, body, filament, core [3]float64) {
	if index == 1 {
		// Iron: a graphite drawing on warm paper, inverted so the dense parts
		// are dark. Reads as a scientific plate rather than as a light show.
		return [3]float64{246, 242, 232}, [3]float64{176, 168, 150},
			[3]float64{74, 70, 66}, [3]float64{12, 12, 16}
	}
	// Phosphor: the long-exposure register — cold empty space, a green-cyan
	// haze, and a white-hot core along the folds.
	return [3]float64{6, 9, 16}, [3]float64{22, 68, 82},
		[3]float64{88, 196, 186}, [3]float64{244, 252, 248}
}
