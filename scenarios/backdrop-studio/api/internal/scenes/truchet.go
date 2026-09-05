package scenes

import "math"

// drawTruchet renders a Truchet tiling: a grid in which every cell carries the
// same pair of quarter-arcs at one of two rotations, so the arcs join across
// cell borders into continuous curves that wander the whole frame.
//
// It is the cheapest way to get structure that reads as *authored* rather than
// as noise. Nothing here is random except one bit per cell, and yet no two
// seeds produce a similar picture, because the loops and the long runs are
// emergent — a property of how the bits happened to line up, not of anything a
// parameter says.
//
// Two scales are drawn, the finer at a lower weight. A single-scale tiling is
// a pattern; the second scale gives the eye somewhere to go up close without
// destroying the read at a distance, which is exactly what a backdrop needs.
// It also fills the histogram: one weight of line on one ground is two tones
// and a screen over it has nothing to modulate.
func drawTruchet(c *canvas, p paramSet, seed int64) {
	fw, fh := float64(c.w), float64(c.h)
	focalX := p.get("focal_x", 0, 1, 0.5)
	focalY := p.get("focal_y", 0, 1, 0.46)
	// cells is the count across the short edge, so tile size is a fraction of
	// the frame and the tiling reads the same at every delivery size.
	//
	// The default is low, and deliberately so. Arc edges are the only
	// high-contrast structure in the picture, and their count scales with the
	// grid: measured across the range, mean neighbour delta runs 0.023 at five
	// cells and 0.056 at nine, so a fine grid stops being a scene and becomes
	// texture. Six tiles across the short edge is also the bolder composition,
	// which is what an ambient backdrop wants.
	cells := p.get("cells", 3, 40, 6)
	// line_width is the arc width as a fraction of the tile.
	lineWidth := p.get("line_width", 0.04, 0.5, 0.19)
	// detail is the weight of the half-scale layer. At 0 this is a plain
	// single-scale tiling.
	detail := p.get("detail", 0, 1, 0.45)
	// drift bends the whole tiling through low-frequency noise, so the grid is
	// not axis-aligned everywhere and the frame does not read as wallpaper.
	drift := p.get("drift", 0, 1, 0.28)
	palette := int(p.get("palette", 0, 1, 0))

	short := math.Min(fw, fh)
	ground, deep, ink, glint := truchetPalette(palette)

	// The highlight's light source, pointing up and to the left. It is a fixed
	// world direction rather than each arc's own outward side, and that
	// distinction is visible: the outward side flips wherever two tiles join,
	// because the arcs there curve around opposite corners, so a highlight
	// keyed to it jumped across the stroke at every junction and left the
	// tiling covered in small rectangular notches. A single light direction is
	// both the correct model of a lit object and the artifact-free one.
	lightX, lightY := -0.6, -0.8

	// coverage returns how much ink one layer puts at a position and how much
	// of that catches the light, both in [0,1].
	//
	// The arcs are centred on two opposite corners of the tile with a radius of
	// half the tile, which is the condition that makes them meet the cell edges
	// exactly at their midpoints — and therefore join whatever the neighbour's
	// rotation is.
	//
	// The lit fraction is returned separately because the highlight belongs on
	// one flank rather than on the whole arc. Painting the glint across the
	// full arc width erased the ink wherever the frame was lit: the arcs simply
	// stopped existing over the bright half of the picture, and the scene's
	// darkest second percentile rose to 0.32 with no ink left to hold the
	// shadow.
	coverage := func(x, y, cell float64, layerSeed int64) (ink, lit float64) {
		gx, gy := math.Floor(x/cell), math.Floor(y/cell)
		u, v := x-gx*cell, y-gy*cell
		r := cell * 0.5
		var cx1, cy1, cx2, cy2 float64
		if hash2(int(gx), int(gy), layerSeed) < 0.5 {
			cx1, cy1, cx2, cy2 = 0, 0, cell, cell
		} else {
			cx1, cy1, cx2, cy2 = cell, 0, 0, cell
		}
		// Signed: positive outside the arc's radius, negative inside.
		s1 := math.Hypot(u-cx1, v-cy1) - r
		s2 := math.Hypot(u-cx2, v-cy2) - r
		s, cx, cy := s1, cx1, cy1
		if math.Abs(s2) < math.Abs(s1) {
			s, cx, cy = s2, cx2, cy2
		}
		half := cell * lineWidth * 0.5
		// One pixel of feather. Hard-edged arcs alias, and a dither downstream
		// turns the aliasing into a crawling stipple along every curve.
		ink = 1 - smoothstep(clamp01((math.Abs(s)-half)/1.0))
		if ink <= 0 || half <= 0 {
			return ink, 0
		}
		// The surface normal of a half-round moulding: it points away from the
		// arc's centre line, tilted by how far across the stroke the pixel is.
		nx, ny := u-cx, v-cy
		if d := math.Hypot(nx, ny); d > 1e-9 {
			nx, ny = nx/d, ny/d
		}
		tilt := clamp01(math.Abs(s)/half) * math.Copysign(1, s)
		// The fractional power broadens the highlight across the lit flank
		// instead of confining it to the few pixels where the normal points
		// straight at the light. With a linear falloff the scene's brightest
		// second percentile sat at 0.778: a highlight too narrow to be a white
		// point, so every ink ramp mapped its paper into a tone that was
		// nowhere in the picture.
		lit = math.Pow(clamp01((nx*lightX+ny*lightY)*tilt), 0.55) * ink
		return ink, lit
	}

	cell := short / cells
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			fx, fy := float64(x), float64(y)
			if drift > 0 {
				// Both offsets come from the *original* position. Feeding the
				// already-displaced x into the y offset couples the two axes,
				// which shears the whole tiling in one diagonal direction
				// instead of wandering it.
				amp := cell * drift * 0.55
				u, v := fx/(short*0.5), fy/(short*0.5)
				fx += (fbm(u, v, 3, seed+61) - 0.5) * amp
				fy += (fbm(u+4.7, v+1.3, 3, seed+62) - 0.5) * amp
			}

			// Depth: a broad falloff from the focal point, so the tiling sits
			// in a lit space instead of on a flat card. It is also what gives
			// the ground its own tonal range for a treatment to work with.
			// The falloff is broad and linear. Squaring it, or tightening the
			// radius, confines the lit ground to a few percent of the frame:
			// the second version of this scene had its 98th-percentile
			// luminance at 0.73 because everything outside a small disc was
			// already halfway to the shadow tone.
			d := math.Hypot((fx/fw-focalX)*0.85, (fy/fh-focalY)*0.85)
			lit := math.Max(0, 1-d)
			base := mixRGB(deep, ground, 0.34+0.66*lit)

			coarse, litArc := coverage(fx, fy, cell, seed)
			fine := 0.0
			if detail > 0 {
				f, _ := coverage(fx+cell*0.5, fy+cell*0.5, cell*0.5, seed+3001)
				fine = f * detail
			}

			col := mixRGB(base, ink, fine*0.75)
			col = mixRGB(col, ink, coarse)
			// A highlight along the outer flank of the coarse arcs. Flat arcs
			// read as a diagram; a glint reads as a drawn object and,
			// incidentally, supplies the white point every ink ramp downstream
			// needs.
			//
			// The exponent on `lit` is what decides whether that white point
			// exists at all. At 1.5 the glint was confined to a few percent of
			// the frame around the focal point and the 98th percentile
			// luminance never rose past 0.67 — a scene with no paper for a
			// treatment to map into. A near-linear falloff spreads it over the
			// whole lit half, which is also what a raking light actually does.
			col = mixRGB(col, glint, litArc*(0.45+0.55*lit))
			c.set(x, y, col[0], col[1], col[2])
		}
	}
}

// truchetPalette returns the lit ground, the shadowed ground, the arc ink and
// its highlight.
func truchetPalette(index int) (ground, deep, ink, glint [3]float64) {
	if index == 1 {
		// Blueprint: pale ink on a saturated ground, the drafting register.
		return [3]float64{34, 62, 148}, [3]float64{8, 16, 52},
			[3]float64{206, 224, 250}, [3]float64{252, 253, 255}
	}
	// Terrazzo: pale stone ground with a dark inlay and a polished highlight.
	return [3]float64{242, 235, 223}, [3]float64{56, 48, 42},
		[3]float64{28, 32, 40}, [3]float64{252, 248, 240}
}
