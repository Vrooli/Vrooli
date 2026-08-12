// Package scenes renders finished procedural backdrops.
//
// It is deliberately separate from internal/scaffold, and the distinction is
// the point. A *scaffold* is conditioning input for a model: flat, blocky
// geometry whose only job is to tell a depth or edge preprocessor where the
// horizon, the focal mass and the copy-safe void sit. Crude is correct there.
// A *scene* is finished output that a treatment chain runs over and a visitor
// actually looks at, so it needs coherent noise, a light model, atmospheric
// depth and — critically — a full tonal range, because every ink-mapping
// treatment downstream distributes its inks across the tones this produces.
//
// Before this package existed the scaffold generators served both roles, so the
// procedural lane shipped conditioning geometry as its product.
//
// Every scene is a pure function of (preset, size, seed, params): no clocks, no
// global RNG, no I/O.
package scenes

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"sort"
)

// Presets are the scene generators this package can render.
//
// The first four are the legacy set: they depict things. The rest are field
// generators — most depict nothing, which is what they are good at, and between
// them they cover the abstract half of the taxonomy without the procedural lane
// ever having to pretend it can draw a building. `contour` and `nebula` are the
// exceptions: they depict a map and a sky respectively, and they claim those
// subjects rather than hiding under the abstract one.
//
// See docs/reference/taxonomy.md for which subject reaches which generator, and
// subjects.go for the rule that a style may only use a generator that depicts
// what the style says it depicts.
var Presets = []string{
	"horizon", "arcade", "terrain", "field",
	"flow", "voronoi", "reaction", "caustics",
	"mesh", "contour", "truchet", "attractor", "nebula",
}

// Request describes one scene render.
type Request struct {
	Preset     string
	Width      int
	Height     int
	Seed       int64
	ParamsJSON string
}

// Result carries the encoded scene and its content hash.
type Result struct {
	PNG    []byte
	SHA256 string
	Width  int
	Height int
}

// Render produces a finished procedural scene.
func Render(req Request) (Result, error) {
	if req.Width < 16 || req.Height < 16 {
		return Result{}, fmt.Errorf("scenes: width and height must be at least 16 (got %dx%d)", req.Width, req.Height)
	}
	if req.Width > 8192 || req.Height > 8192 {
		return Result{}, fmt.Errorf("scenes: width and height must be at most 8192 (got %dx%d)", req.Width, req.Height)
	}
	known := false
	for _, p := range Presets {
		if p == req.Preset {
			known = true
			break
		}
	}
	if !known {
		return Result{}, fmt.Errorf("scenes: unknown preset %q (known: %v)", req.Preset, Presets)
	}

	params := map[string]float64{}
	if req.ParamsJSON != "" {
		if err := json.Unmarshal([]byte(req.ParamsJSON), &params); err != nil {
			return Result{}, fmt.Errorf("scenes: invalid params_json: %w", err)
		}
	}
	p := paramSet{values: params}
	c := &canvas{img: image.NewNRGBA(image.Rect(0, 0, req.Width, req.Height)), w: req.Width, h: req.Height}

	switch req.Preset {
	case "horizon":
		drawHorizon(c, p, req.Seed)
	case "arcade":
		drawArcade(c, p, req.Seed)
	case "terrain":
		drawTerrain(c, p, req.Seed)
	case "field":
		drawField(c, p, req.Seed)
	case "flow":
		drawFlowField(c, p, req.Seed)
	case "voronoi":
		drawVoronoi(c, p, req.Seed)
	case "reaction":
		drawReactionDiffusion(c, p, req.Seed)
	case "caustics":
		drawCaustics(c, p, req.Seed)
	case "mesh":
		drawMeshGradient(c, p, req.Seed)
	case "contour":
		drawContour(c, p, req.Seed)
	case "truchet":
		drawTruchet(c, p, req.Seed)
	case "attractor":
		drawAttractor(c, p, req.Seed)
	case "nebula":
		drawNebula(c, p, req.Seed)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, c.img); err != nil {
		return Result{}, fmt.Errorf("scenes: encode PNG: %w", err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return Result{PNG: buf.Bytes(), SHA256: hex.EncodeToString(sum[:]), Width: req.Width, Height: req.Height}, nil
}

// paramSet reads optional scene parameters. Absence is distinguished from zero:
// scaffold.clamp treated 0 as "unset" and substituted a fallback, which made a
// legitimate edge value such as focal_x=0 unreachable.
type paramSet struct{ values map[string]float64 }

func (p paramSet) get(key string, min, max, fallback float64) float64 {
	v, ok := p.values[key]
	if !ok {
		return fallback
	}
	return math.Min(max, math.Max(min, v))
}

// ── canvas ───────────────────────────────────────────────────────────────

type canvas struct {
	img  *image.NRGBA
	w, h int
}

func clamp8(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v + 0.5)
}

func (c *canvas) set(x, y int, r, g, b float64) {
	if x < 0 || y < 0 || x >= c.w || y >= c.h {
		return
	}
	c.img.SetNRGBA(x, y, color.NRGBA{R: clamp8(r), G: clamp8(g), B: clamp8(b), A: 255})
}

func (c *canvas) at(x, y int) (float64, float64, float64) {
	if x < 0 || y < 0 || x >= c.w || y >= c.h {
		return 0, 0, 0
	}
	n := c.img.NRGBAAt(x, y)
	return float64(n.R), float64(n.G), float64(n.B)
}

// blend composites a colour over the existing pixel at the given coverage.
func (c *canvas) blend(x, y int, r, g, b, a float64) {
	if a <= 0 {
		return
	}
	if a > 1 {
		a = 1
	}
	or, og, ob := c.at(x, y)
	c.set(x, y, or+(r-or)*a, og+(g-og)*a, ob+(b-ob)*a)
}

// ── coherent noise ───────────────────────────────────────────────────────
//
// The scaffold generators drew their randomness from a per-pixel xorshift,
// which is white noise: adjacent pixels are uncorrelated, so it can only ever
// produce speckle. Terrain ridges, foliage masses and cloud structure all need
// *coherent* noise, where nearby samples are related.

func hash2(ix, iy int, seed int64) float64 {
	h := uint64(seed)*0x9e3779b97f4a7c15 ^ uint64(int64(ix))*0xbf58476d1ce4e5b9 ^ uint64(int64(iy))*0x94d049bb133111eb
	h ^= h >> 30
	h *= 0xbf58476d1ce4e5b9
	h ^= h >> 27
	h *= 0x94d049bb133111eb
	h ^= h >> 31
	return float64(h>>11) / float64(uint64(1)<<53)
}

func smoothstep(t float64) float64 { return t * t * (3 - 2*t) }

// valueNoise samples a smoothly interpolated 2-D lattice.
func valueNoise(x, y float64, seed int64) float64 {
	ix, iy := math.Floor(x), math.Floor(y)
	fx, fy := smoothstep(x-ix), smoothstep(y-iy)
	i, j := int(ix), int(iy)
	a := hash2(i, j, seed)
	b := hash2(i+1, j, seed)
	cc := hash2(i, j+1, seed)
	d := hash2(i+1, j+1, seed)
	return (a+(b-a)*fx)*(1-fy) + (cc+(d-cc)*fx)*fy
}

// fbm sums octaves of value noise.
func fbm(x, y float64, octaves int, seed int64) float64 {
	sum, amp, freq, norm := 0.0, 0.5, 1.0, 0.0
	for i := 0; i < octaves; i++ {
		sum += amp * valueNoise(x*freq, y*freq, seed+int64(i)*7919)
		norm += amp
		freq *= 2
		amp *= 0.5
	}
	return sum / norm
}

// ridged inverts and sharpens fbm into mountain-like crests.
func ridged(x, y float64, octaves int, seed int64) float64 {
	v := 1 - math.Abs(fbm(x, y, octaves, seed)*2-1)
	return v * v
}

// ── shared elements ──────────────────────────────────────────────────────

// sky paints a vertical gradient with a sun glow, brightening toward the
// horizon. The glow is what gives every downstream ink ramp a genuine white
// point to map to.
func (c *canvas) sky(horizonY, sunX, sunY, sunR float64, top, bottom [3]float64, warm [3]float64) {
	for y := 0; y < int(horizonY)+1 && y < c.h; y++ {
		t := float64(y) / math.Max(1, horizonY)
		t = math.Pow(t, 1.4)
		r := top[0] + (bottom[0]-top[0])*t
		g := top[1] + (bottom[1]-top[1])*t
		b := top[2] + (bottom[2]-top[2])*t
		for x := 0; x < c.w; x++ {
			d := math.Hypot(float64(x)-sunX, float64(y)-sunY)
			glow := math.Exp(-d / (sunR * 3.2))
			c.set(x, y, r+(warm[0]-r)*glow, g+(warm[1]-g)*glow, b+(warm[2]-b)*glow)
		}
	}
	// the disc itself — the scene's white point
	for y := int(sunY - sunR - 2); y <= int(sunY+sunR+2); y++ {
		for x := int(sunX - sunR - 2); x <= int(sunX+sunR+2); x++ {
			d := math.Hypot(float64(x)-sunX, float64(y)-sunY)
			if d <= sunR {
				c.set(x, y, 255, 252, 244)
			} else if d <= sunR+2 {
				c.blend(x, y, 255, 252, 244, 1-(d-sunR)/2)
			}
		}
	}
}

// ── scenes ───────────────────────────────────────────────────────────────

func drawHorizon(c *canvas, p paramSet, seed int64) {
	fw, fh := float64(c.w), float64(c.h)
	horizon := p.get("horizon", 0.2, 0.85, 0.58) * fh
	focal := p.get("focal_x", 0, 1, 0.74)
	sunX, sunY := focal*fw, horizon*0.36
	sunR := fh * 0.085

	c.sky(horizon, sunX, sunY, sunR, [3]float64{12, 28, 92}, [3]float64{176, 206, 232}, [3]float64{255, 214, 150})

	// sea, darkening with depth, with a sun column and coherent ripples
	for y := int(horizon); y < c.h; y++ {
		t := (float64(y) - horizon) / math.Max(1, fh-horizon)
		base := [3]float64{18 + 30*(1-t), 62 + 60*(1-t), 130 + 70*(1-t)}
		for x := 0; x < c.w; x++ {
			col := math.Exp(-math.Abs(float64(x)-sunX) / (fw * 0.06))
			rip := (fbm(float64(x)/(fw*0.02), float64(y)/(fh*0.006), 3, seed+11) - 0.5) * 34
			r := base[0]*(1-t*0.55) + rip + col*120*(1-t*0.7)
			g := base[1]*(1-t*0.55) + rip + col*95*(1-t*0.7)
			b := base[2]*(1-t*0.55) + rip + col*55*(1-t*0.7)
			c.set(x, y, r, g, b)
		}
	}

	// Distant headlands sit ON the horizon and rise upward. They must not fill
	// downward to the frame edge: that paints over the sea and the composition
	// loses its water entirely.
	for layer := 0; layer < 3; layer++ {
		lf := float64(layer)
		shade := [3]float64{96 - lf*26, 128 - lf*34, 122 - lf*30}
		amp := fh * (0.13 - lf*0.032)
		baseY := horizon - lf*fh*0.012
		for x := 0; x < c.w; x++ {
			n := fbm(float64(x)/(fw*0.24)+lf*13, lf*5, 4, seed+int64(layer)*331)
			top := baseY - n*amp - amp*0.3
			for y := int(top); y < int(baseY) && y < c.h; y++ {
				c.set(x, y, shade[0], shade[1], shade[2])
			}
		}
	}

	// Foreground bank along the bottom edge: the scene's tonal floor, which the
	// ink-mapping treatments need as their black point.
	bankTop := fh * 0.86
	for x := 0; x < c.w; x++ {
		n := fbm(float64(x)/(fw*0.16)+29, 3, 4, seed+77)
		top := bankTop - n*fh*0.06
		// a bright shoreline edge where the bank meets the water
		for y := int(top); y < int(top+fh*0.014) && y < c.h; y++ {
			c.set(x, y, 236, 224, 190)
		}
		for y := int(top + fh*0.014); y < c.h; y++ {
			g := fbm(float64(x)/(fw*0.01), float64(y)/(fh*0.01), 3, seed+53)
			c.set(x, y, 14+g*20, 26+g*26, 20+g*20)
		}
	}
}

func drawArcade(c *canvas, p paramSet, seed int64) {
	fw, fh := float64(c.w), float64(c.h)
	bays := int(p.get("bays", 1, 8, 3))
	focal := p.get("focal_x", 0, 1, 0.5)
	horizon := fh * 0.62
	sunX, sunY := focal*fw, horizon*0.55
	sunR := fh * 0.055

	// The view THROUGH the arches is painted first, across the whole frame.
	// The previous generator filled the arch openings with a colour darker than
	// the wall, so the arcade read as three dark slabs instead of openings.
	c.sky(horizon, sunX, sunY, sunR, [3]float64{92, 140, 190}, [3]float64{214, 222, 214}, [3]float64{255, 236, 198})
	for y := int(horizon); y < c.h; y++ {
		t := (float64(y) - horizon) / math.Max(1, fh-horizon)
		for x := 0; x < c.w; x++ {
			rip := (fbm(float64(x)/(fw*0.03), float64(y)/(fh*0.008), 3, seed+5) - 0.5) * 26
			c.set(x, y, 96-36*t+rip, 132-52*t+rip, 162-58*t+rip)
		}
	}
	// distant headland
	for x := 0; x < c.w; x++ {
		n := fbm(float64(x)/(fw*0.3), 0, 4, seed+91)
		top := horizon - n*fh*0.07 - fh*0.02
		for y := int(top); y < int(horizon)+1 && y < c.h; y++ {
			c.set(x, y, 140, 152, 154)
		}
	}

	// statue silhouette in the centre bay, lit from the sun side
	stX, stBase := fw*0.5, fh*0.80
	for y := int(stBase - fh*0.30); y < int(stBase); y++ {
		for x := int(stX - fw*0.05); x < int(stX+fw*0.05); x++ {
			dy := (float64(y) - (stBase - fh*0.30)) / (fh * 0.30)
			halfW := fw * 0.020 * (0.55 + 0.65*math.Sin(dy*math.Pi*0.9+0.35))
			if math.Abs(float64(x)-stX) < halfW {
				lit := 1 - math.Abs(float64(x)-stX)/halfW*0.45
				c.set(x, y, 224*lit+16, 218*lit+16, 202*lit+16)
			}
		}
	}
	for y := int(stBase); y < int(stBase+fh*0.09) && y < c.h; y++ {
		for x := int(stX - fw*0.035); x < int(stX+fw*0.035); x++ {
			c.set(x, y, 206, 198, 180)
		}
	}

	// the wall, with arch openings left untouched
	pad, gap := fw*0.07, fw*0.035
	span := (fw - pad*2 - gap*float64(bays-1)) / float64(bays)
	archTop, archBottom := fh*0.26, fh*0.90
	radius := span / 2
	inArch := func(x, y float64) bool {
		for i := 0; i < bays; i++ {
			ax := pad + float64(i)*(span+gap)
			if x < ax || x > ax+span || y > archBottom {
				continue
			}
			cy := archTop + radius
			if y >= cy {
				return true
			}
			if math.Hypot(x-(ax+radius), y-cy) <= radius {
				return true
			}
		}
		return false
	}
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			fx, fy := float64(x), float64(y)
			if inArch(fx, fy) {
				continue
			}
			// plaster with coherent grain, shaded away from the sun
			n := fbm(fx/(fw*0.04), fy/(fh*0.04), 4, seed+17)
			lit := 1 - math.Min(1, math.Abs(fx-sunX)/fw)*0.28
			v := (206 + n*34) * lit
			c.set(x, y, v, v*0.95, v*0.85)
		}
	}
	// ledge below the arches
	for y := int(archBottom); y < c.h; y++ {
		t := (float64(y) - archBottom) / math.Max(1, fh-archBottom)
		for x := 0; x < c.w; x++ {
			n := fbm(float64(x)/(fw*0.05), float64(y)/(fh*0.05), 3, seed+23)
			v := (188 + n*28) * (1 - t*0.42)
			c.set(x, y, v, v*0.95, v*0.86)
		}
	}

	// foliage canopy: coherent mass, not speckle
	for y := 0; y < int(fh*0.42); y++ {
		for x := 0; x < c.w; x++ {
			fx, fy := float64(x), float64(y)
			d := fbm(fx/(fw*0.06), fy/(fh*0.05), 4, seed+61)
			cover := d - (fy/(fh*0.42))*0.75
			if cover > 0.16 {
				shade := 0.55 + d*0.7
				c.blend(x, y, 26*shade, 56*shade, 34*shade, math.Min(1, (cover-0.16)*7))
			}
		}
	}
}

func drawTerrain(c *canvas, p paramSet, seed int64) {
	fw, fh := float64(c.w), float64(c.h)
	horizon := p.get("horizon", 0.2, 0.85, 0.42) * fh
	focal := p.get("focal_x", 0, 1, 0.26)

	// night sky with a moon as the white point
	moonX, moonY, moonR := focal*fw, horizon*0.42, fh*0.075
	c.sky(horizon, moonX, moonY, moonR, [3]float64{8, 16, 30}, [3]float64{132, 168, 176}, [3]float64{236, 246, 240})

	// five ridge layers, lightest at the back (atmospheric haze)
	const layers = 5
	for layer := 0; layer < layers; layer++ {
		lf := float64(layer) / float64(layers-1)
		baseY := horizon + lf*fh*0.34
		amp := fh * (0.26 - lf*0.15)
		v := 214 - lf*196
		for x := 0; x < c.w; x++ {
			r := ridged(float64(x)/(fw*0.22)+float64(layer)*17, float64(layer)*3, 5, seed+int64(layer)*613)
			top := baseY - r*amp
			for y := int(top); y < c.h; y++ {
				// a little vertical falloff so each mass has form
				dy := math.Min(1, (float64(y)-top)/(fh*0.4))
				c.set(x, y, (v+14)*(1-dy*0.22), (v+26)*(1-dy*0.22), (v+18)*(1-dy*0.22))
			}
		}
	}

	// foreground canopy: dark, near-black, giving the tonal floor
	for x := 0; x < c.w; x++ {
		n := fbm(float64(x)/(fw*0.05), 7, 4, seed+911)
		top := fh*0.82 - n*fh*0.10
		for y := int(top); y < c.h; y++ {
			g := fbm(float64(x)/(fw*0.01), float64(y)/(fh*0.01), 3, seed+53)
			c.set(x, y, 6+g*14, 12+g*20, 9+g*14)
		}
	}
}

func drawField(c *canvas, p paramSet, seed int64) {
	fw, fh := float64(c.w), float64(c.h)
	focal := p.get("focal_x", 0, 1, 0.5)
	focalY := p.get("focal_y", 0, 1, 0.46)

	// A non-representational field: layered soft masses over a deep ground,
	// with a bright core so the tonal range still spans end to end.
	type blob struct {
		x, y, r float64
		col     [3]float64
	}
	palette := [][3]float64{
		{255, 108, 92},
		{255, 176, 74},
		{236, 96, 168},
		{122, 118, 255},
		{255, 232, 168},
		{96, 214, 200},
	}
	blobs := make([]blob, 0, 22)
	for i := 0; i < 22; i++ {
		fi := float64(i)
		bx := hash2(i, 1, seed)
		by := hash2(i, 2, seed)
		br := hash2(i, 3, seed)
		// pull the mass toward the focal point so the composition has a centre
		bx = bx*0.72 + focal*0.28
		by = by*0.72 + focalY*0.28
		blobs = append(blobs, blob{
			x: bx * fw, y: by * fh,
			r:   fh * (0.10 + br*0.38),
			col: palette[(i+int(fi))%len(palette)],
		})
	}
	sort.SliceStable(blobs, func(a, b int) bool { return blobs[a].r > blobs[b].r })

	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			c.set(x, y, 14, 10, 22)
		}
	}
	for _, b := range blobs {
		for y := int(b.y - b.r); y <= int(b.y+b.r); y++ {
			for x := int(b.x - b.r); x <= int(b.x+b.r); x++ {
				d := math.Hypot(float64(x)-b.x, float64(y)-b.y) / b.r
				if d >= 1 {
					continue
				}
				a := math.Pow(1-d, 2.2) * 0.72
				// coherent turbulence breaks the perfect circles
				a *= 0.65 + 0.7*fbm(float64(x)/(fw*0.05), float64(y)/(fh*0.05), 3, seed+7)
				c.blend(x, y, b.col[0], b.col[1], b.col[2], a)
			}
		}
	}
	// a bright core so the field has a white point for ink mapping
	cx, cy, cr := focal*fw, focalY*fh, fh*0.30
	for y := int(cy - cr); y <= int(cy+cr); y++ {
		for x := int(cx - cr); x <= int(cx+cr); x++ {
			d := math.Hypot(float64(x)-cx, float64(y)-cy) / cr
			if d < 1 {
				// Saturate to a solid plateau near the centre rather than falling
				// off from the very middle, so the field carries a real highlight
				// area for an ink ramp to map paper into, not a single bright pixel.
				c.blend(x, y, 255, 248, 236, math.Min(1, math.Pow(1-d, 1.5)*2.2))
			}
		}
	}
}
