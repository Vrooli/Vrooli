package scenes

import "math"

// drawCaustics renders the light pattern cast through a disturbed water
// surface: bright filaments where the surface focuses light, wide dim regions
// where it spreads.
//
// The physics is done rather than faked. A height field is built from layered
// noise, its gradient gives the surface normal, and the refracted ray's
// displacement is accumulated into a light buffer — so caustic lines appear
// exactly where neighbouring rays converge, which is what gives them their
// characteristic property: they are *thin and very bright*, with sharp cusps
// where they cross. A "caustics" texture painted with sinusoids has none of
// that; it looks like a plaid.
//
// The convergence also produces the widest tonal range of any generator here.
// Focused lines run far brighter than the ambient, so the histogram has a long
// bright tail and a broad dim body, and an ink-mapping treatment downstream has
// the whole ramp available instead of a narrow band.
func drawCaustics(c *canvas, p paramSet, seed int64) {
	fw, fh := float64(c.w), float64(c.h)
	focalX := p.get("focal_x", 0, 1, 0.5)
	focalY := p.get("focal_y", 0, 1, 0.5)
	// depth is the distance from the surface to the floor: shallow water gives
	// tight, sharp caustics, deep water gives broad soft ones.
	depth := p.get("depth", 0.2, 4, 1.15)
	// chop is the surface roughness — how disturbed the water is.
	chop := p.get("chop", 0.2, 3, 1.0)

	short := math.Min(fw, fh)
	// The height field's wavelength is a fraction of the frame, so the same
	// declared style produces the same water at every delivery size.
	waveScale := 1.0 / (short * 0.18 / chop)

	height := func(x, y float64) float64 {
		// Two octave sets at different scales and orientations: one long swell,
		// one finer ripple riding on it.
		swell := fbm(x*waveScale, y*waveScale, 4, seed+41)
		ripple := fbm(x*waveScale*2.7+31.5, y*waveScale*2.7-17.25, 3, seed+42)
		return swell*0.72 + ripple*0.28
	}

	light := make([]float64, c.w*c.h)
	// wavelength is the dominant surface feature size in pixels. Everything
	// below is expressed against it, so the caustics are the same fraction of
	// the frame at every delivery size.
	wavelength := 1 / waveScale
	// The finite-difference step is a fraction of the wavelength rather than a
	// fixed pixel count: at a fixed step, a large frame samples the same wave
	// far more finely and the gradient magnitude changes with resolution.
	eps := math.Max(1, wavelength*0.02)
	// Ray displacement per unit surface slope. A height field of unit amplitude
	// and this wavelength has slopes on the order of 1/wavelength, so scaling
	// by wavelength squared makes the displacement a stable multiple of the
	// wavelength itself — the regime where neighbouring rays actually converge
	// into lines. Scaling by the frame size instead (the first attempt here)
	// produced displacements of hundreds of pixels, which scatters rays at
	// random and renders as speckle, not as caustics.
	refract := depth * 0.16 * wavelength * wavelength

	splat := func(tx, ty, w float64) {
		// Bilinear splatting, not integer binning. Rounding each ray to a whole
		// pixel is itself a quantisation, and it puts energy at exactly the
		// single-pixel frequency the screens downstream work at — the result
		// reads as film grain over a wash rather than as focused light.
		x0, y0 := int(math.Floor(tx)), int(math.Floor(ty))
		fx, fy := tx-float64(x0), ty-float64(y0)
		for _, q := range [4]struct {
			dx, dy int
			w      float64
		}{
			{0, 0, (1 - fx) * (1 - fy)},
			{1, 0, fx * (1 - fy)},
			{0, 1, (1 - fx) * fy},
			{1, 1, fx * fy},
		} {
			x, y := x0+q.dx, y0+q.dy
			if x < 0 || y < 0 || x >= c.w || y >= c.h {
				continue
			}
			light[y*c.w+x] += w * q.w
		}
	}

	// Supersample: several rays per pixel, so convergence is measured from a
	// continuous wavefront rather than from a single sample per cell.
	const sub = 2
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			for sy := 0; sy < sub; sy++ {
				for sx := 0; sx < sub; sx++ {
					fx := float64(x) + (float64(sx)+0.5)/sub
					fy := float64(y) + (float64(sy)+0.5)/sub
					hx := (height(fx+eps, fy) - height(fx-eps, fy)) / (2 * eps)
					hy := (height(fx, fy+eps) - height(fx, fy-eps)) / (2 * eps)
					// Snell's law linearised for small angles: the refracted ray
					// lands displaced from directly below by an amount
					// proportional to the surface slope and the water depth.
					splat(fx-hx*refract, fy-hy*refract, 1.0/(sub*sub))
				}
			}
		}
	}

	// One ray per pixel makes the light buffer noisy at the single-pixel scale.
	// A short blur is not a cosmetic smoothing here: it stands in for the
	// finite width of a real light source, without which the caustics alias
	// into speckle at exactly the frequency the screens downstream work at.
	// The blur radius is a fraction of the short edge, not a pixel count. A
	// fixed radius is proportionally huge on a small surface and invisible on a
	// large one, which showed up as the frame's mean luminance drifting with
	// its size — the caustics were a different picture at every geometry.
	light = blurBuffer(light, c.w, c.h, int(math.Max(1, short/280)))

	norm := percentile(light, 0.995)
	if norm <= 0 {
		norm = 1
	}

	floorDeep := [3]float64{3, 9, 18}
	floorLit := [3]float64{30, 92, 118}
	caustic := [3]float64{146, 220, 230}
	crest := [3]float64{252, 253, 246}

	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			v := light[y*c.w+x] / norm
			if v > 1.6 {
				v = 1.6
			}

			// A slow ambient gradient across the floor, so the frame has an
			// overall light direction and does not read as a flat tile.
			//
			// It is the load-bearing structure in this scene, not decoration.
			// A screen laid over caustics has only two things to work with: the
			// caustic filaments, which are finer than any usable screen cell,
			// and this gradient. Measured on the finished renders, a caustic
			// field whose low-frequency composition was weak scored 0.372 on
			// subject survival under a posterize-and-halftone chain — there was
			// no composition for the screen to preserve, so the screen became
			// the picture. Widening and deepening the gradient is what gives a
			// screening treatment a subject.
			d := math.Hypot((float64(x)-focalX*fw)/(fw*0.78), (float64(y)-focalY*fh)/(fh*0.78))
			ambient := math.Max(0, 1-d*0.88)
			ambient = math.Pow(ambient, 1.15)

			// A broad shaft mask: some of the frame is lit water and some is
			// calm shadow. Without it the caustics are equally busy everywhere
			// — technically correct, compositionally inert, and rejected by the
			// perceptual gate's frequency-modulation bar for exactly the right
			// reason. Real water under real light is not uniformly bright.
			shaft := fbm(float64(x)/(short*1.15)+5.5, float64(y)/(short*1.15)-2.25, 2, seed+43)
			shaft = clamp01((shaft - 0.28) / 0.52)
			lit := clamp01(0.42 + 0.58*shaft*(0.35+0.65*ambient))

			// The floor runs the full range of the ramp rather than the top
			// two thirds of it: an unlit corner has to be genuinely dark, or
			// the difference between lit and unlit water survives neither a
			// quantiser nor the eye.
			col := mixRGB(floorDeep, floorLit, ambient*0.96+0.02)
			col = mixRGB(col, caustic, clamp01(math.Pow(v, 0.85)*0.95)*lit)
			// The cusps — where several rays land in one place — get the
			// highlight. This is the whole reason for accumulating rather than
			// evaluating a pattern function.
			// The cusps keep their full brightness wherever they land: the
			// shaft mask decides where light *falls*, not how bright a focused
			// caustic is once it does. Attenuating them too made the frame
			// uniformly dim, which trades one gate failure for another.
			col = mixRGB(col, crest, clamp01(math.Pow(math.Max(0, v-0.26)/0.62, 1.0)))
			c.set(x, y, col[0], col[1], col[2])
		}
	}
}
