package scenes

import "math"

// drawFlowField advects a dense population of particles through a curl-noise
// velocity field and accumulates where they went.
//
// The result is the silk-ribbon look: thousands of near-parallel filaments that
// bunch and separate, dense where trajectories converge and open where they
// diverge. It is non-representational by construction — it depicts nothing, it
// is a record of motion — which is what makes it honest for the `flow` subject
// rather than a stand-in for a picture the procedural lane cannot draw.
//
// Tonal depth comes from accumulation rather than from a palette ramp: a pixel
// crossed by forty filaments is genuinely forty times brighter than one crossed
// by one, so the histogram is continuous end to end and a dither or halftone
// downstream has real gradient to modulate. A generator that fills regions with
// flat colour gives a quantising treatment nothing to work with, which is the
// defect this whole family of generators exists to avoid.
func drawFlowField(c *canvas, p paramSet, seed int64) {
	fw, fh := float64(c.w), float64(c.h)
	focalX := p.get("focal_x", 0, 1, 0.42)
	focalY := p.get("focal_y", 0, 1, 0.48)
	// turbulence sets how tightly the field curls: low is laminar and calm,
	// high is a knot. The default sits where filaments still read as strokes.
	turbulence := p.get("turbulence", 0.2, 3, 1.0)
	// density scales the particle count with the canvas so a large surface is
	// not a sparser picture than a small one.
	density := p.get("density", 0.2, 3, 1.0)

	// The simulation runs on a canonical grid and the result is scaled to the
	// canvas, for the same reason the reaction-diffusion generator does it: a
	// filament deposited at canvas resolution is one pixel wide whatever the
	// canvas is, so it is 0.3% of a small frame and 0.1% of a large one, and
	// the picture is not the same picture at two sizes. On a fixed grid every
	// feature is a fixed fraction of the frame by construction.
	const gridShort = 640
	gw, gh := gridShort, gridShort
	if fw > fh {
		gw = int(float64(gridShort) * fw / fh)
	} else {
		gh = int(float64(gridShort) * fh / fw)
	}
	gwf, ghf := float64(gw), float64(gh)
	accum := make([]float64, gw*gh)
	tint := make([]float64, gw*gh)

	short := math.Min(gwf, ghf)
	particles := int(float64(gw*gh) * 0.012 * density)
	// Both the step length and the trajectory length are fractions of the short
	// edge. With a fixed pixel step and a fixed step count, a particle crossed
	// a small frame entirely and barely marked a large one — the frame's mean
	// luminance then drifted with its size, which is the same
	// resolution-dependence the treatment layer spent a phase removing.
	stepLen := math.Max(1, short/220)
	steps := int(short * 0.62 / stepLen)
	scale := 1.0 / (short * 0.55 / turbulence)

	for i := 0; i < particles; i++ {
		// Start positions are biased toward the focal point so the composition
		// has a centre of mass instead of an even wash.
		x := (hash2(i, 11, seed)*1.35-0.17)*gwf*0.85 + focalX*gwf*0.15
		y := (hash2(i, 12, seed)*1.35-0.17)*ghf*0.85 + focalY*ghf*0.15
		// Each particle carries a constant hue offset so the field bands into
		// related colours rather than averaging to a single muddy tone.
		hue := hash2(i, 13, seed)
		life := 0.55 + 0.45*hash2(i, 14, seed)

		for s := 0; s < steps; s++ {
			// Curl of a scalar noise field: divergence-free, so filaments
			// neither pile into points nor evaporate.
			eps := stepLen * 1.25
			n1 := fbm((x+eps)*scale, y*scale, 4, seed+3)
			n2 := fbm((x-eps)*scale, y*scale, 4, seed+3)
			n3 := fbm(x*scale, (y+eps)*scale, 4, seed+3)
			n4 := fbm(x*scale, (y-eps)*scale, 4, seed+3)
			vx := (n3 - n4) / (2 * eps)
			vy := -(n1 - n2) / (2 * eps)
			mag := math.Hypot(vx, vy)
			if mag < 1e-9 {
				break
			}
			x += vx / mag * stepLen
			y += vy / mag * stepLen
			if x < 0 || y < 0 || x >= gwf || y >= ghf {
				break
			}
			// Fade along the trajectory so filaments have ends rather than
			// running edge to edge like wallpaper.
			w := life * (1 - float64(s)/float64(steps)*0.65)
			idx := int(y)*gw + int(x)
			accum[idx] += w
			tint[idx] += w * hue
		}
	}

	// A one-pixel filament is a one-pixel feature, and a picture built from
	// them is speckle by the coherence measure the scene tests apply — and, more
	// to the point, a fine screen downstream samples it at random. A blur of a
	// fraction of the short edge gives each filament a body without dissolving
	// the weave. It is relative, not a pixel count, so filaments are the same
	// fraction of the frame at every delivery size.
	blur := int(math.Max(1, short/150))
	accum = blurBuffer(accum, gw, gh, blur)
	tint = blurBuffer(tint, gw, gh, blur)

	// Normalise against a high percentile rather than the maximum: a handful of
	// crossing points are far denser than the field and would otherwise
	// compress everything else into the bottom of the range.
	norm := percentile(accum, 0.99)
	if norm <= 0 {
		norm = 1
	}

	ground := [3]float64{10, 12, 24}
	cool := [3]float64{58, 132, 214}
	warm := [3]float64{236, 152, 92}
	high := [3]float64{248, 244, 232}

	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			gx := float64(x) / fw * float64(gw-1)
			gy := float64(y) / fh * float64(gh-1)
			density := sampleBilinear(accum, gw, gh, gx, gy)
			v := density / norm
			if v > 1 {
				v = 1
			}
			// A gentle gamma keeps the mid-tones open. Without it the field is
			// mostly ground with a few blown-out filaments, which reads as
			// noise on black rather than as woven light.
			v = math.Pow(v, 0.62)

			h := 0.5
			if density > 0 {
				h = sampleBilinear(tint, gw, gh, gx, gy) / density
			}
			base := mixRGB(cool, warm, h)

			// A soft radial lift at the focal point gives the frame somewhere
			// to look and guarantees a real highlight for ink mapping.
			d := math.Hypot((float64(x)-focalX*fw)/(fw*0.55), (float64(y)-focalY*fh)/(fh*0.55))
			glow := math.Max(0, 1-d)
			glow *= glow

			col := mixRGB(ground, base, v)
			// The brightest filaments must actually reach the top of the ramp:
			// a scene whose 98th percentile sits at 0.76 gives an ink-mapping
			// treatment no paper to map into, and every screen over it comes
			// out muddy.
			col = mixRGB(col, high, math.Min(1, math.Pow(v, 1.5)*1.35+glow*0.18))
			c.set(x, y, col[0], col[1], col[2])
		}
	}
}
