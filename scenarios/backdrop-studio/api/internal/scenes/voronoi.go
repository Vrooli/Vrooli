package scenes

import "math"

// drawVoronoi renders a jittered-grid Voronoi field shaded by the F2−F1
// distance — the gap between the nearest and second-nearest site.
//
// F2−F1 is the choice that makes this a picture rather than a diagram. Shading
// cells by F1 alone gives each cell a radial gradient and a hard seam at every
// border: flat-ish plates with edges, which a halftone renders as a mosaic of
// unrelated blobs. F2−F1 is near zero exactly on the borders and rises smoothly
// into cell interiors, so the field is continuous everywhere and the structure
// reads as cracked glaze or dry earth — a surface with depth, not a partition.
//
// The grid is jittered rather than random-scattered because a scatter leaves
// large empty regions and tight clusters at any seed, and a backdrop needs its
// texture to be even enough that the composition, not the accident of the
// scatter, decides where the eye goes.
func drawVoronoi(c *canvas, p paramSet, seed int64) {
	fw, fh := float64(c.w), float64(c.h)
	focalX := p.get("focal_x", 0, 1, 0.5)
	focalY := p.get("focal_y", 0, 1, 0.45)
	// cells is the count across the short edge, so the cell size is a fraction
	// of the frame and the same style reads the same at every surface.
	cells := p.get("cells", 4, 64, 14)
	// jitter 0 is a regular lattice, 1 is fully random within each grid square.
	jitter := p.get("jitter", 0, 1, 0.85)
	// warp bends the sampling grid through noise so cell rows are not visibly
	// aligned to the axes.
	warp := p.get("warp", 0, 1, 0.45)

	short := math.Min(fw, fh)
	cell := short / cells
	site := func(gx, gy int) (float64, float64) {
		jx := (hash2(gx, gy*7+1, seed) - 0.5) * jitter
		jy := (hash2(gx*13+5, gy, seed) - 0.5) * jitter
		return (float64(gx) + 0.5 + jx) * cell, (float64(gy) + 0.5 + jy) * cell
	}

	// The palette spans nearly the whole ramp on purpose. An earlier set ran
	// from L* 0.32 to 0.67 and filled under half the histogram — legible as a
	// picture, useless as a screening source, because every ink a treatment
	// maps would land in the same third of the range.
	deep := [3]float64{6, 10, 18}
	mid := [3]float64{54, 100, 126}
	glaze := [3]float64{182, 212, 204}
	rim := [3]float64{252, 249, 240}

	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			fx, fy := float64(x), float64(y)
			if warp > 0 {
				w := cell * warp * 0.9
				fx += (fbm(fx/(short*0.30), fy/(short*0.30), 3, seed+21) - 0.5) * w
				fy += (fbm(fx/(short*0.30)+9.5, fy/(short*0.30)-4.25, 3, seed+22) - 0.5) * w
			}
			gx, gy := int(math.Floor(fx/cell)), int(math.Floor(fy/cell))

			f1, f2 := math.Inf(1), math.Inf(1)
			var nearestID float64
			for oy := -1; oy <= 1; oy++ {
				for ox := -1; ox <= 1; ox++ {
					sx, sy := site(gx+ox, gy+oy)
					d := math.Hypot(fx-sx, fy-sy)
					if d < f1 {
						f2 = f1
						f1 = d
						nearestID = hash2(gx+ox, gy+oy, seed+99)
					} else if d < f2 {
						f2 = d
					}
				}
			}

			// Border proximity, 0 on a seam and 1 deep inside a cell.
			edge := math.Min(1, (f2-f1)/(cell*0.85))
			edge = smoothstep(edge)

			// Depth: cells further from the focal point sit back, which turns a
			// flat tiling into a lit surface.
			d := math.Hypot((fx-focalX*fw)/(fw*0.62), (fy-focalY*fh)/(fh*0.62))
			depth := math.Max(0, 1-d)
			depth *= depth

			// Per-cell tonal variation, so neighbouring cells differ in value
			// and the field keeps a full histogram instead of one plateau.
			shade := 0.08 + 0.92*nearestID

			body := mixRGB(deep, mid, shade*0.70+depth*0.30)
			body = mixRGB(body, glaze, edge*0.88*(0.25+0.75*shade))
			// The seams take the highlight: a thin bright line where cells meet
			// is what makes the surface read as glaze rather than as paint.
			body = mixRGB(body, rim, math.Pow(1-edge, 2.2)*(0.55+0.45*depth))
			// A cell interior must be able to reach the bottom of the ramp, or
			// the darkest ink a treatment maps has nowhere to land. The deepest
			// cells go darker than the palette's floor by their own shade.
			body = mixRGB(body, deep, math.Pow(edge, 1.6)*(1-shade)*0.75)
			c.set(x, y, body[0], body[1], body[2])
		}
	}
}
