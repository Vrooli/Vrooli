package scenes

import "math"

// drawMeshGradient renders a soft multi-point colour field, smeared along a
// slowly turning axis.
//
// This is the generator behind the contemporary hero backdrop: a few colour
// stops bleeding into each other with no edge anywhere, brightest where the
// headline is not. It is the only generator here whose output is meant to ship
// *untreated* — a screen over it would add the mechanical texture the look
// exists to avoid — which is why the `procedural` strategy exists at all.
//
// Two decisions carry the look:
//
// **Inverse-distance weighting, not a radial blend.** Weighting every stop by
// 1/dᵖ and normalising gives a field that is smooth everywhere and has no
// visible boundary between one stop's territory and the next. A nearest-stop or
// two-stop blend leaves seams that a large flat area shows immediately.
//
// **The smear is what makes it look lit rather than tie-dyed.** Sampling the
// field along a short segment whose angle turns slowly across the frame pulls
// the stops into blades of colour. Without it the same palette reads as
// overlapping blobs; with it, as light raking across a surface.
//
// The field is computed on a canonical grid and bilinearly upsampled. That is
// free here in a way it is not for the other generators: the picture is smooth
// by construction, so the upsample loses nothing, and it makes the output the
// same picture at every delivery size rather than one that gains detail with
// pixels.
func drawMeshGradient(c *canvas, p paramSet, seed int64) {
	fw, fh := float64(c.w), float64(c.h)
	focalX := p.get("focal_x", 0, 1, 0.72)
	focalY := p.get("focal_y", 0, 1, 0.28)
	// stops is the number of colour control points. Few and large reads as a
	// wash; many and small reads as marbling.
	stops := int(p.get("stops", 3, 14, 7))
	// smear is the sampling segment's length as a fraction of the short edge.
	// Zero is a plain mesh gradient; the default is where stops elongate into
	// blades but still read as one field.
	smear := p.get("smear", 0, 1.2, 0.42)
	// angle is the smear axis in degrees, measured clockwise from due east.
	angle := p.get("angle", 0, 360, 34) * math.Pi / 180
	// spread is how far the smear axis turns across the frame, in turns. A
	// small value fans the blades; a large one curls them.
	spread := p.get("spread", 0, 1, 0.16)
	// softness is the inverse-distance exponent. Low is a broad wash where
	// every stop reaches everywhere; high is tight pools with narrow bleeds.
	softness := p.get("softness", 1, 5, 2.4)
	// warp bends the sampling grid through noise, so stops are lobed rather
	// than circular and the field stops looking like an interpolation.
	warp := p.get("warp", 0, 1, 0.35)
	// blades is the amplitude of a lightness modulation that varies only
	// *across* the smear axis.
	//
	// This is the parameter that makes the generator produce the look rather
	// than an approximation of it. Smearing a field of soft blobs just blurs
	// them further — the first version of this scene was a brown cloud, because
	// averaging along a line removes structure and adds none. Modulation
	// perpendicular to that line survives the average exactly, so it comes out
	// as long parallel blades of light with the palette's colour in them, which
	// is what a raking light through a slatted surface actually does and what
	// every reference image of this genre shows.
	blades := p.get("blades", 0, 1, 0.55)
	// blade_scale is the spacing of those blades as a fraction of the short
	// edge. Small values give a fine rake, large ones a few broad shafts.
	bladeScale := p.get("blade_scale", 0.02, 1, 0.13)
	palette := meshPalette(int(p.get("palette", 0, 2, 0)))

	const gridShort = 512
	gw, gh := gridShort, gridShort
	if fw > fh {
		gw = int(float64(gridShort) * fw / fh)
	} else {
		gh = int(float64(gridShort) * fh / fw)
	}
	gwf, ghf := float64(gw), float64(gh)
	short := math.Min(gwf, ghf)

	// Stop placement. Two are pinned and the rest are scattered, because the
	// tonal extremes cannot be left to the seed: a scene whose darkest value is
	// a midtone gives every ink-mapping treatment downstream nowhere to put its
	// shadow, and one with no highlight comes out muddy under any screen. The
	// anchors put the palette's floor opposite the focal point and its ceiling
	// on it, which is also the composition the placement layer wants — quiet
	// where the copy sits, bright where it does not.
	type stop struct {
		x, y float64
		// core is the radius of the stop's flat centre, squared and in grid
		// units. A large core makes the stop a broad region; a small one makes
		// it a point the field spikes toward. Anchors get a large core because
		// a tonal anchor with a small one is a stain rather than a shadow — the
		// first version put a hard black smudge in the lower left of every
		// frame, which reads as a defect however correct the arithmetic is.
		core float64
		col  [3]float64
	}
	broad := short * short * 0.10
	tight := short * short * 0.012
	placed := make([]stop, 0, stops)
	placed = append(placed,
		stop{x: (1 - focalX) * gwf, y: (1 - focalY) * ghf, core: broad, col: palette[0]},
		stop{x: focalX * gwf, y: focalY * ghf, core: broad * 0.5, col: palette[len(palette)-1]},
	)
	for i := len(placed); i < stops; i++ {
		placed = append(placed, stop{
			x:    hash2(i, 3, seed) * gwf,
			y:    hash2(i, 5, seed) * ghf,
			core: tight,
			col:  palette[1+(i-2)%(len(palette)-2)],
		})
	}

	// sample returns the mesh colour at one grid position, warped, with the
	// blade modulation applied across the given smear axis.
	sample := func(x, y, theta float64) [3]float64 {
		if warp > 0 {
			amp := short * warp * 0.16
			nx := fbm(x/(short*0.55), y/(short*0.55), 3, seed+41)
			ny := fbm(x/(short*0.55)+7.3, y/(short*0.55)-2.9, 3, seed+42)
			x += (nx - 0.5) * amp
			y += (ny - 0.5) * amp
		}
		var acc [3]float64
		var wsum float64
		for _, s := range placed {
			dx, dy := x-s.x, y-s.y
			w := math.Pow(dx*dx+dy*dy+s.core, -softness/2)
			acc[0] += s.col[0] * w
			acc[1] += s.col[1] * w
			acc[2] += s.col[2] * w
			wsum += w
		}
		if wsum == 0 {
			return palette[0]
		}
		col := [3]float64{acc[0] / wsum, acc[1] / wsum, acc[2] / wsum}
		if blades <= 0 {
			return col
		}
		// The coordinate across the smear axis. Because it depends only on the
		// perpendicular projection, averaging along the axis leaves it intact.
		perp := -math.Sin(theta)*x + math.Cos(theta)*y
		n := fbm(perp/(short*bladeScale), 3.71, 3, seed+83)
		// Multiplicative, so a blade brightens and dims the colour that is
		// already there rather than painting a grey stripe over it.
		gain := 1 + blades*(n-0.5)*1.9
		return [3]float64{col[0] * gain, col[1] * gain, col[2] * gain}
	}

	// smearSamples is odd so the segment is centred on the pixel; more samples
	// than this stop changing the result once the segment is this short.
	const smearSamples = 9
	grid := make([][3]float64, gw*gh)
	for y := 0; y < gh; y++ {
		for x := 0; x < gw; x++ {
			fx, fy := float64(x), float64(y)
			// The axis turns across the frame, which is what fans the blades.
			theta := angle + (fx/gwf+fy/ghf-1)*spread*2*math.Pi
			if smear <= 0 {
				grid[y*gw+x] = sample(fx, fy, theta)
				continue
			}
			dx, dy := math.Cos(theta), math.Sin(theta)
			half := smear * short * 0.5
			var acc [3]float64
			for k := 0; k < smearSamples; k++ {
				t := (float64(k)/(smearSamples-1) - 0.5) * 2 * half
				col := sample(fx+dx*t, fy+dy*t, theta)
				acc[0] += col[0]
				acc[1] += col[1]
				acc[2] += col[2]
			}
			grid[y*gw+x] = [3]float64{acc[0] / smearSamples, acc[1] / smearSamples, acc[2] / smearSamples}
		}
	}

	// Channels are blurred separately at a radius proportional to the grid, to
	// remove the faint banding the finite smear sampling leaves behind.
	channels := make([][]float64, 3)
	for ch := range channels {
		buf := make([]float64, gw*gh)
		for i, col := range grid {
			buf[i] = col[ch]
		}
		channels[ch] = blurBuffer(buf, gw, gh, int(math.Max(1, short/220)))
	}
	expandRange(channels, 0.03, 0.99)

	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			gx := float64(x) / fw * float64(gw-1)
			gy := float64(y) / fh * float64(gh-1)
			col := [3]float64{
				sampleBilinear(channels[0], gw, gh, gx, gy),
				sampleBilinear(channels[1], gw, gh, gx, gy),
				sampleBilinear(channels[2], gw, gh, gx, gy),
			}
			// A wide, shallow vignette. Its job is to settle the frame's edge,
			// not to supply the black point — `expandRange` above already
			// guarantees the histogram reaches both ends, and stacking a deep
			// vignette on top of it darkened two thirds of the picture.
			d := math.Hypot((float64(x)/fw-0.5)*1.25, (float64(y)/fh-0.5)*1.25)
			shade := 1 - 0.28*smoothstep(clamp01((d-0.42)/0.58))
			col = [3]float64{col[0] * shade, col[1] * shade, col[2] * shade}
			c.set(x, y, col[0], col[1], col[2])
		}
	}
}

// meshPalette returns one art-directed ramp, ordered darkest to lightest.
//
// Ordering is load-bearing rather than tidy: the placement code pins entry 0 to
// the quiet corner and the last entry to the focal point, so a ramp written out
// of order would put the highlight where the copy goes.
//
// A float parameter selecting a named palette is not elegant, but scene
// parameters are a `map[string]float64` on the wire and a style needs some way
// to choose. The alternative — one preset per palette — would make three
// generators out of one and lie about how much of the space is covered.
func meshPalette(index int) [][3]float64 {
	switch index {
	case 1:
		// Aurora: the dark-ground hero. Near-black navy through teal and
		// violet to a cold white.
		return [][3]float64{
			{8, 12, 26}, {18, 74, 92}, {36, 122, 138},
			{92, 78, 168}, {186, 138, 204}, {236, 240, 250},
		}
	case 2:
		// Solar: plum through magenta and orange to gold. The warmest of the
		// three and the one that survives a heavy posterize.
		return [][3]float64{
			{26, 8, 34}, {112, 26, 86}, {198, 54, 84},
			{236, 118, 52}, {246, 186, 88}, {252, 244, 220},
		}
	default:
		// Ember: scorched umber through burnt orange to a paper cream. The
		// warm-on-cream register of the reference set.
		return [][3]float64{
			{22, 12, 8}, {124, 44, 16}, {206, 88, 28},
			{238, 146, 62}, {246, 208, 156}, {250, 244, 234},
		}
	}
}
