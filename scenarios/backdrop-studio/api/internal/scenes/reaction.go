package scenes

import "math"

// drawReactionDiffusion runs a Gray-Scott reaction-diffusion simulation and
// shades the settled chemical concentration.
//
// Two chemicals diffuse at different rates while one converts the other; the
// competition between diffusion and reaction produces coral, maze and mitosis
// patterns that no noise function generates and no designer draws by hand. It
// is the most genuinely emergent generator in the set, and unlike the others
// its structure has a *scale of its own* — the pattern's characteristic spot
// size falls out of the feed and kill rates, not out of a parameter anyone
// tuned to a frame size.
//
// The simulation runs on a coarse grid and is resampled up. That is not a
// shortcut: Gray-Scott is a physical model with a real length scale, so running
// it at delivery resolution would produce a pattern whose features are the same
// number of *pixels* at every surface, which is exactly the resolution
// dependence the rest of this system spent a phase removing. Simulating at a
// fixed grid and scaling the result keeps the pattern a fixed fraction of the
// picture.
func drawReactionDiffusion(c *canvas, p paramSet, seed int64) {
	fw, fh := float64(c.w), float64(c.h)
	focalX := p.get("focal_x", 0, 1, 0.5)
	focalY := p.get("focal_y", 0, 1, 0.45)
	// feed and kill select the pattern regime. The defaults sit in the coral
	// band, which keeps a connected structure with open ground between it —
	// the regime with the widest tonal spread.
	feed := p.get("feed", 0.01, 0.09, 0.0545)
	kill := p.get("kill", 0.03, 0.07, 0.062)
	steps := int(p.get("steps", 200, 6000, 2600))

	// A fixed simulation grid, aspect-matched so the pattern is not stretched.
	const gridShort = 180
	gw, gh := gridShort, gridShort
	if fw > fh {
		gw = int(float64(gridShort) * fw / fh)
	} else {
		gh = int(float64(gridShort) * fh / fw)
	}

	a := make([]float64, gw*gh)
	b := make([]float64, gw*gh)
	for i := range a {
		a[i] = 1
	}
	// Seeding: a handful of coherent patches rather than uniform noise. Uniform
	// noise settles into an even all-over texture with no composition; patches
	// grow into distinct colonies and leave quiet ground between them.
	for s := 0; s < 14; s++ {
		cx := hash2(s, 31, seed) * float64(gw)
		cy := hash2(s, 32, seed) * float64(gh)
		r := 3 + hash2(s, 33, seed)*7
		for y := int(cy - r); y <= int(cy+r); y++ {
			for x := int(cx - r); x <= int(cx+r); x++ {
				if x < 0 || y < 0 || x >= gw || y >= gh {
					continue
				}
				if math.Hypot(float64(x)-cx, float64(y)-cy) <= r {
					b[y*gw+x] = 1
				}
			}
		}
	}

	const dA, dB, dt = 1.0, 0.5, 1.0
	na := make([]float64, gw*gh)
	nb := make([]float64, gw*gh)
	// Toroidal wrap keeps the pattern uniform to the frame edge; a clamped
	// boundary grows a visible rind around the border.
	at := func(buf []float64, x, y int) float64 {
		return buf[((y+gh)%gh)*gw+(x+gw)%gw]
	}
	for step := 0; step < steps; step++ {
		for y := 0; y < gh; y++ {
			for x := 0; x < gw; x++ {
				i := y*gw + x
				// The canonical Gray-Scott stencil: orthogonal neighbours at
				// 0.2, diagonals at 0.05, centre at -1. The weights sum to zero
				// and the positive weights sum to one, which is what makes dA
				// and dB mean what the literature says they mean. An unweighted
				// nine-point sum is six times this, and at six times the
				// intended diffusion the reaction cannot hold a pattern: it
				// smooths to a uniform field and the frame renders blank.
				lapA := 0.2*(at(a, x-1, y)+at(a, x+1, y)+at(a, x, y-1)+at(a, x, y+1)) +
					0.05*(at(a, x-1, y-1)+at(a, x+1, y-1)+at(a, x-1, y+1)+at(a, x+1, y+1)) - a[i]
				lapB := 0.2*(at(b, x-1, y)+at(b, x+1, y)+at(b, x, y-1)+at(b, x, y+1)) +
					0.05*(at(b, x-1, y-1)+at(b, x+1, y-1)+at(b, x-1, y+1)+at(b, x+1, y+1)) - b[i]
				abb := a[i] * b[i] * b[i]
				na[i] = clamp01(a[i] + (dA*lapA-abb+feed*(1-a[i]))*dt)
				nb[i] = clamp01(b[i] + (dB*lapB+abb-(kill+feed)*b[i])*dt)
			}
		}
		copy(a, na)
		copy(b, nb)
	}

	deep := [3]float64{6, 8, 14}
	body := [3]float64{72, 96, 150}
	bloom := [3]float64{214, 156, 122}
	high := [3]float64{252, 248, 238}

	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			// Bilinear sampling: nearest-neighbour would give the pattern hard
			// stair-stepped edges, which a fine screen turns into aliasing.
			v := sampleBilinear(b, gw, gh, float64(x)/fw*float64(gw-1), float64(y)/fh*float64(gh-1))
			// Gray-Scott's B concentration is mostly near zero with structure in
			// a narrow band; stretching it is what turns a technically-correct
			// simulation into an image with a full tonal range.
			v = clamp01((v - 0.05) / 0.45)
			v = math.Pow(v, 0.78)

			d := math.Hypot((float64(x)-focalX*fw)/(fw*0.60), (float64(y)-focalY*fh)/(fh*0.60))
			depth := math.Max(0, 1-d)
			depth *= depth

			col := mixRGB(deep, body, v)
			col = mixRGB(col, bloom, math.Min(1, v*1.35)*0.55)
			col = mixRGB(col, high, math.Min(1, math.Pow(v, 1.6)*1.25+depth*0.12))
			c.set(x, y, col[0], col[1], col[2])
		}
	}
}
