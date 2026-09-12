package scenes

import "math"

// drawNebula renders a deep-sky field: layered cloud emission, dust lanes that
// occlude it, and a scatter of stars.
//
// It depicts `atmospheric`, which had no generator before and so could only be
// answered by the model lane or, worse, by silently rendering something else.
// The subject is worth drawing procedurally because it is the one photographic
// register that is genuinely easier to synthesise than to shoot, and because
// its tonal shape — a very dark ground with small, extremely bright points —
// is the hardest input a screening treatment ever gets. A halftone that holds
// together over this holds together over anything.
//
// The structure comes from domain warping rather than from summing more
// octaves. Feeding a noise field's own output back in as a coordinate offset
// produces the sheared, wrung-out filaments real emission nebulae have; plain
// fBm at any octave count reads as fog. The dust lanes are a second, sparser
// field multiplied in, because in a real nebula the dark structure is
// foreground matter blocking the light, not an absence of emission — and that
// distinction is visible: multiplied dust has hard silhouetted edges where
// subtracted emission has soft ones.
func drawNebula(c *canvas, p paramSet, seed int64) {
	fw, fh := float64(c.w), float64(c.h)
	focalX := p.get("focal_x", 0, 1, 0.62)
	focalY := p.get("focal_y", 0, 1, 0.42)
	// turbulence is how hard the emission field is warped by itself.
	turbulence := p.get("turbulence", 0, 1.5, 0.72)
	// dust is how much foreground matter occludes the emission.
	dust := p.get("dust", 0, 1, 0.62)
	// stars is the star count per megapixel of the frame, so the sky has the
	// same density at every delivery size.
	starsPerMP := p.get("stars", 0, 4000, 900)
	// core_size is the emission core's radius as a fraction of the short edge.
	coreSize := p.get("core_size", 0.05, 0.9, 0.34)
	palette := int(p.get("palette", 0, 1, 0))

	short := math.Min(fw, fh)
	scale := 1 / (short * 0.62)
	space, cool, warm, hot := nebulaPalette(palette)

	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			fx, fy := float64(x), float64(y)
			u, v := fx*scale, fy*scale

			// Domain warp: the field displaces its own sample position.
			wx := fbm(u+1.7, v-3.1, 4, seed+71)
			wy := fbm(u-5.3, v+2.4, 4, seed+72)
			u += (wx - 0.5) * turbulence * 2.2
			v += (wy - 0.5) * turbulence * 2.2

			// A power curve on the emission is what separates filaments from
			// fog. Raw fBm is symmetric about its mean, so most of the frame
			// sits at a mid value and the structure reads as an even haze; the
			// curve pushes the bulk down and leaves the peaks, which is also
			// how emission actually falls off with density.
			emission := math.Pow(fbm(u, v, 5, seed+73), 1.7)

			// The core is where the cloud is being lit from. Without it the
			// field is an even fog with no composition and no white point.
			d := math.Hypot((fx/fw-focalX)*fw/short, (fy/fh-focalY)*fh/short) / coreSize
			core := math.Exp(-d * d * 1.35)
			// The core is modulated by the cloud it is lighting. A bare
			// Gaussian reads as a lens flare pasted onto a sky — the first
			// version of this scene was exactly that, a clean radial glow with
			// no relationship to the structure around it. Multiplying by the
			// emission field makes the lit region follow the material, so the
			// glow has filaments running through it.
			structure := math.Pow(fbm(u*0.8+3.3, v*0.8-1.9, 4, seed+75), 0.8)
			emission = clamp01(emission*1.45 + core*(0.28+0.72*structure)*0.9)

			// Foreground dust, multiplied in. Sparse octaves and a hard curve
			// give it the silhouetted edge that says "something is in front of
			// this" rather than "there is less of this here".
			occlusion := 1.0
			if dust > 0 {
				lane := fbm(u*0.62+11.5, v*0.62-7.25, 3, seed+74)
				occlusion = 1 - dust*math.Pow(clamp01((lane-0.42)/0.58), 1.6)
			}
			e := emission * occlusion

			col := mixRGB(space, cool, math.Pow(e, 0.85))
			col = mixRGB(col, warm, math.Pow(clamp01(e*1.12-0.24), 1.5))
			// The hot core is keyed to the unmodulated glow, tinted by the
			// structure rather than gated on it. Gating it — multiplying the
			// core into the highlight term as well as into the emission —
			// applied the same attenuation twice and left the scene's brightest
			// second percentile at 0.556, so the picture had no white point at
			// all despite obviously having a bright middle.
			col = mixRGB(col, hot, math.Pow(clamp01(core*(0.45+0.55*occlusion)), 0.8)*(0.55+0.45*structure))

			c.set(x, y, col[0], col[1], col[2])
		}
	}

	// Stars last, over the cloud. Each is a small disc with a soft edge rather
	// than a single pixel: a one-pixel star is a resolution-dependent feature —
	// invisible on a large surface, a hard speckle on a small one — and it
	// samples at random under any screen.
	count := int(starsPerMP * fw * fh / 1e6)
	radius := math.Max(0.7, short/900)
	for i := 0; i < count; i++ {
		sx := hash2(i, 101, seed) * fw
		sy := hash2(i, 103, seed) * fh
		// A steep magnitude distribution: mostly faint, a few bright. An even
		// one reads as dirt on the lens.
		mag := math.Pow(hash2(i, 107, seed), 3.2)
		r := radius * (0.6 + 1.9*mag)
		bright := 0.30 + 0.70*mag
		for oy := -int(r) - 1; oy <= int(r)+1; oy++ {
			for ox := -int(r) - 1; ox <= int(r)+1; ox++ {
				px, py := int(sx)+ox, int(sy)+oy
				dd := math.Hypot(float64(ox), float64(oy))
				if dd > r+1 {
					continue
				}
				a := bright * (1 - smoothstep(clamp01((dd-r*0.4)/(r*0.6+1))))
				c.blend(px, py, 252, 250, 244, a)
			}
		}
	}
}

// nebulaPalette returns empty space, the cool emission, the warm emission and
// the lit core, darkest first.
func nebulaPalette(index int) (space, cool, warm, hot [3]float64) {
	if index == 1 {
		// Ember cloud: a dust-lit region in reds and golds against near-black.
		return [3]float64{7, 5, 10}, [3]float64{86, 34, 48},
			[3]float64{234, 150, 96}, [3]float64{252, 240, 214}
	}
	// Cold emission: the classic teal-and-magenta plate.
	return [3]float64{5, 7, 14}, [3]float64{34, 96, 128},
		[3]float64{212, 138, 196}, [3]float64{246, 250, 252}
}
