package scenes

import "math"

// drawField renders merging implicit surfaces — metaballs.
//
// It replaces a stack of twenty-two soft radial blobs alpha-blended over a dark
// ground. That version was the oldest generator here and the weakest, and the
// weakness was structural rather than a matter of tuning: alpha-blended
// gaussians have no surfaces. There is nowhere in the picture where one thing
// stops and another begins, so every treatment downstream had nothing to find.
// Under a duotone it came out a smudge; under an ASCII mosaic the composition
// scored 0.560 on subject survival — below the gate's floor — because there was
// no composition to survive.
//
// A metaball field is the same family of shape with the one property that was
// missing. Summing r²/d² kernels and reading the result against a threshold
// gives an implicit surface: blobs have edges, and when two approach they fuse
// through a visible neck rather than simply overlapping. That neck is the
// picture. It is also what a screen, a dither or a glyph mosaic can actually
// modulate, because it is a boundary rather than a gradient.
//
// The shading keeps three registers so the histogram stays continuous: the
// deep ground outside the surface, a lit body inside it, and a bright rim in
// the narrow band where the field crosses the threshold.
func drawField(c *canvas, p paramSet, seed int64) {
	fw, fh := float64(c.w), float64(c.h)
	focalX := p.get("focal_x", 0, 1, 0.5)
	focalY := p.get("focal_y", 0, 1, 0.46)
	// balls is the number of implicit sources. Few and large reads as a single
	// organism; many and small reads as foam.
	balls := int(p.get("balls", 3, 24, 11))
	// fuse raises the threshold, which pulls the surface in tight around each
	// source; lowering it lets distant sources merge into one mass.
	//
	// The default is well above 1 because the failure at 1 is not subtle: with
	// eleven sources the field clears the threshold almost everywhere, the
	// ground disappears, and the scene's darkest second percentile rises to
	// 0.349 — a picture with no shadow, which is the one thing an ink-mapping
	// treatment cannot work around. At 2.2 the surfaces are distinct and the
	// ground is visible between them at every source count in range.
	fuse := p.get("fuse", 0.4, 4, 2.2)
	// rim is the width of the bright band at the surface, as a fraction of the
	// threshold. It is what makes the shapes read as objects rather than as
	// regions of colour.
	rim := p.get("rim", 0, 1, 0.42)
	// turbulence warps the sampling position, so the surfaces are lobed and
	// organic instead of circular.
	turbulence := p.get("turbulence", 0, 1, 0.45)
	// grain is a fine tonal texture across the whole field. See its use below:
	// it is what stops a symbolic treatment rendering a large area as one
	// repeated character.
	grain := p.get("grain", 0, 1, 0.5)
	palette := int(p.get("palette", 0, 1, 0))

	short := math.Min(fw, fh)
	ground, body, surface, glow := fieldPalette(palette)

	type source struct {
		x, y, strength float64
	}
	sources := make([]source, 0, balls)
	for i := 0; i < balls; i++ {
		// Positions are pulled toward the focal point so the mass has a centre
		// and the frame has somewhere quiet for copy to sit.
		bx := hash2(i, 1, seed)*0.78 + focalX*0.22
		by := hash2(i, 2, seed)*0.78 + focalY*0.22
		// Radii are a fraction of the short edge, so the same seed is the same
		// picture at every delivery size.
		r := short * (0.10 + hash2(i, 3, seed)*0.20)
		sources = append(sources, source{x: bx * fw, y: by * fh, strength: r * r})
	}

	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			fx, fy := float64(x), float64(y)
			if turbulence > 0 {
				amp := short * turbulence * 0.10
				u, v := fx/(short*0.45), fy/(short*0.45)
				fx += (fbm(u, v, 3, seed+51) - 0.5) * amp
				fy += (fbm(u+3.9, v-6.1, 3, seed+52) - 0.5) * amp
			}

			// The implicit field: the classic inverse-square-distance sum. The
			// epsilon keeps a source's own centre finite rather than infinite,
			// which would otherwise blow the whole ramp on one pixel.
			var field float64
			for _, s := range sources {
				dx, dy := fx-s.x, fy-s.y
				field += s.strength / (dx*dx + dy*dy + short*short*0.0012)
			}

			// Distance from the surface, signed and normalised: negative
			// outside, positive inside, and roughly scale-free either way.
			depth := (field - fuse) / fuse

			var col [3]float64
			if depth > 0 {
				// Inside. The body brightens toward the core so a large mass is
				// not one flat plate — a quantising treatment needs gradient
				// inside the shape as much as it needs an edge around it.
				// The exponent keeps the body visible. The field grows fast near a
				// source, so a linear mix blows every interior to the core colour
				// and the picture becomes white shapes on black — no interior
				// gradient at all for a quantising treatment to step through.
				col = mixRGB(body, glow, smoothstep(clamp01(math.Pow(depth, 0.55)*0.42)))
			} else {
				// Outside, with a short falloff so the surface has a halo
				// rather than a hard cut against the ground.
				col = mixRGB(ground, body, smoothstep(clamp01(1+depth*3.2)))
			}
			// The rim: a bright band centred on the surface itself.
			if rim > 0 {
				band := 1 - smoothstep(clamp01(math.Abs(depth)/rim))
				col = mixRGB(col, surface, band*0.8)
			}
			// A fine tonal grain across the whole field.
			//
			// It is barely visible on its own and it is what makes the
			// generator usable under a symbolic treatment. A glyph mosaic
			// quantises tone into characters, so an area of constant tone
			// becomes an area of one repeated character — a wall of `@` with a
			// hard vertical edge where the next tone begins, which is precisely
			// the artifact the first audit found in `ascii-field` and blamed on
			// the treatment. The treatment was doing its job; the source had no
			// tone for it to differentiate. A quarter-step of grain gives every
			// large area a reason to break up.
			if grain > 0 {
				g := fbm(fx/(short*0.045), fy/(short*0.045), 3, seed+61) - 0.5
				scale := 1 + grain*g*0.34
				col = [3]float64{col[0] * scale, col[1] * scale, col[2] * scale}
			}
			c.set(x, y, col[0], col[1], col[2])
		}
	}
}

// fieldPalette returns the ground outside the surface, the lit body inside it,
// the surface rim, and the core glow — darkest first.
func fieldPalette(index int) (ground, body, surface, glow [3]float64) {
	if index == 1 {
		// Lab: a cool instrument register, pale bodies on near-black.
		return [3]float64{6, 10, 16}, [3]float64{34, 78, 104},
			[3]float64{206, 240, 246}, [3]float64{250, 252, 252}
	}
	// Lava lamp: warm bodies rising through a deep violet ground.
	return [3]float64{16, 10, 26}, [3]float64{132, 52, 96},
		[3]float64{252, 214, 168}, [3]float64{255, 246, 228}
}
