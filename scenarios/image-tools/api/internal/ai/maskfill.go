package ai

import (
	"image"
	"image/color"
	"math"
)

// MaskFillParams controls the deterministic membrane fill.
type MaskFillParams struct {
	// GrowPx dilates the mask before filling so a hand-drawn mask that clips the
	// object's edge still covers it.
	GrowPx int
	// HaloThreshold grows the mask over connected pixels whose colour differs
	// from the local background estimate by more than this many levels, which
	// absorbs a glow halo the mask missed.
	HaloThreshold int
	// MaxIterations bounds the Laplace relaxation.
	MaxIterations int
}

// DefaultMaskFillParams returns the documented defaults.
func DefaultMaskFillParams() MaskFillParams {
	return MaskFillParams{GrowPx: 6, HaloThreshold: 25, MaxIterations: 4000}
}

// MaskFill reconstructs the pixels under mask by solving Laplace's equation
// inside the mask with the surrounding pixels fixed (a membrane fill). It is
// exact on flat and smooth backgrounds, which is the logo case, and is always
// available because it is pure Go with no host dependency.
func MaskFill(src, maskImg image.Image, p MaskFillParams) *image.NRGBA {
	if p.GrowPx < 0 {
		p.GrowPx = 0
	}
	if p.MaxIterations <= 0 {
		p.MaxIterations = 4000
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	px := make([]color.NRGBA, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			px[y*w+x] = color.NRGBAModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
		}
	}
	mb := maskImg.Bounds()
	mw, mh := mb.Dx(), mb.Dy()
	mask := make([]bool, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Sample the mask scaled to the source geometry so the two need not
			// share an exact size.
			mx := x * mw / w
			my := y * mh / h
			r, g, bl, a := maskImg.At(mb.Min.X+mx, mb.Min.Y+my).RGBA()
			mask[y*w+x] = a > 0 && (int(r>>8)+int(g>>8)+int(bl>>8))/3 > 127
		}
	}
	if p.GrowPx > 0 {
		mask = dilateMask(mask, w, h, p.GrowPx)
	}
	if p.HaloThreshold > 0 {
		mask = growHalo(px, mask, w, h, p.HaloThreshold, 4)
	}
	solveLaplace(px, mask, w, h, p.MaxIterations)

	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	copy(out.Pix, flattenPix(px, w, h))
	return out
}

func flattenPix(px []color.NRGBA, w, h int) []byte {
	pix := make([]byte, 0, w*h*4)
	for _, c := range px {
		pix = append(pix, c.R, c.G, c.B, c.A)
	}
	return pix
}

// dilateMask grows mask by radius using a separable box dilation.
func dilateMask(mask []bool, w, h, radius int) []bool {
	row := make([]bool, len(mask))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ok := false
			for dx := -radius; dx <= radius && !ok; dx++ {
				xx := x + dx
				if xx >= 0 && xx < w && mask[y*w+xx] {
					ok = true
				}
			}
			row[y*w+x] = ok
		}
	}
	out := make([]bool, len(mask))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ok := false
			for dy := -radius; dy <= radius && !ok; dy++ {
				yy := y + dy
				if yy >= 0 && yy < h && row[yy*w+x] {
					ok = true
				}
			}
			out[y*w+x] = ok
		}
	}
	return out
}

// growHalo expands the mask over connected pixels that differ from a local
// background estimate by more than threshold levels. The estimate is the mean of
// the known (non-mask) pixels in a small window, computed with integral images.
func growHalo(px []color.NRGBA, mask []bool, w, h, threshold, passes int) []bool {
	const radius = 8
	// Integral images of known colour sums and counts.
	iw := w + 1
	sumR := make([]float64, iw*(h+1))
	sumG := make([]float64, iw*(h+1))
	sumB := make([]float64, iw*(h+1))
	cnt := make([]float64, iw*(h+1))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y+1)*iw + (x + 1)
			up := y*iw + (x + 1)
			left := (y+1)*iw + x
			diag := y*iw + x
			var cr, cg, cb, cc float64
			if !mask[y*w+x] {
				c := px[y*w+x]
				cr, cg, cb, cc = float64(c.R), float64(c.G), float64(c.B), 1
			}
			sumR[i] = sumR[up] + sumR[left] - sumR[diag] + cr
			sumG[i] = sumG[up] + sumG[left] - sumG[diag] + cg
			sumB[i] = sumB[up] + sumB[left] - sumB[diag] + cb
			cnt[i] = cnt[up] + cnt[left] - cnt[diag] + cc
		}
	}
	window := func(x, y int) (r, g, bl float64, n float64) {
		x0, y0 := maxInt(0, x-radius), maxInt(0, y-radius)
		x1, y1 := minInt(w, x+radius+1), minInt(h, y+radius+1)
		r = sumR[y1*iw+x1] - sumR[y0*iw+x1] - sumR[y1*iw+x0] + sumR[y0*iw+x0]
		g = sumG[y1*iw+x1] - sumG[y0*iw+x1] - sumG[y1*iw+x0] + sumG[y0*iw+x0]
		bl = sumB[y1*iw+x1] - sumB[y0*iw+x1] - sumB[y1*iw+x0] + sumB[y0*iw+x0]
		n = cnt[y1*iw+x1] - cnt[y0*iw+x1] - cnt[y1*iw+x0] + cnt[y0*iw+x0]
		return
	}
	for pass := 0; pass < passes; pass++ {
		grown := make([]bool, len(mask))
		copy(grown, mask)
		changed := false
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				i := y*w + x
				if mask[i] || !hasMaskNeighbor(mask, w, h, x, y) {
					continue
				}
				r, g, bl, n := window(x, y)
				if n == 0 {
					continue
				}
				c := px[i]
				br, bg, bb := r/n, g/n, bl/n
				d := math.Max(math.Abs(float64(c.R)-br), math.Max(math.Abs(float64(c.G)-bg), math.Abs(float64(c.B)-bb)))
				if d > float64(threshold) {
					grown[i] = true
					changed = true
				}
			}
		}
		mask = grown
		if !changed {
			break
		}
	}
	return mask
}

func hasMaskNeighbor(mask []bool, w, h, x, y int) bool {
	if x > 0 && mask[y*w+x-1] {
		return true
	}
	if x < w-1 && mask[y*w+x+1] {
		return true
	}
	if y > 0 && mask[(y-1)*w+x] {
		return true
	}
	if y < h-1 && mask[(y+1)*w+x] {
		return true
	}
	return false
}

// solveLaplace relaxes the masked pixels to the average of their neighbours with
// successive over-relaxation until the largest single-step change is below half
// a level or the iteration budget is spent.
func solveLaplace(px []color.NRGBA, mask []bool, w, h, maxIter int) {
	const omega = 1.9
	// Initialize masked pixels to the mean of the boundary so the first
	// iterations start near the answer.
	var sr, sg, sb, n float64
	for i, m := range mask {
		if !m && hasMaskNeighbor(mask, w, h, i%w, i/w) {
			c := px[i]
			sr += float64(c.R)
			sg += float64(c.G)
			sb += float64(c.B)
			n++
		}
	}
	if n > 0 {
		fill := color.NRGBA{uint8(sr / n), uint8(sg / n), uint8(sb / n), 255}
		for i, m := range mask {
			if m {
				px[i] = fill
			}
		}
	}
	idx := make([]int, 0, len(mask))
	for i, m := range mask {
		if m {
			idx = append(idx, i)
		}
	}
	if len(idx) == 0 {
		return
	}
	for iter := 0; iter < maxIter; iter++ {
		maxDelta := 0.0
		for _, i := range idx {
			x, y := i%w, i/w
			var sumR, sumG, sumB, count float64
			for _, nb := range [4]int{i - 1, i + 1, i - w, i + w} {
				switch nb {
				case i - 1:
					if x == 0 {
						continue
					}
				case i + 1:
					if x == w-1 {
						continue
					}
				case i - w:
					if y == 0 {
						continue
					}
				case i + w:
					if y == h-1 {
						continue
					}
				}
				c := px[nb]
				sumR += float64(c.R)
				sumG += float64(c.G)
				sumB += float64(c.B)
				count++
			}
			if count == 0 {
				continue
			}
			cur := px[i]
			nr := (1-omega)*float64(cur.R) + omega*sumR/count
			ng := (1-omega)*float64(cur.G) + omega*sumG/count
			nb := (1-omega)*float64(cur.B) + omega*sumB/count
			if d := math.Abs(nr - float64(cur.R)); d > maxDelta {
				maxDelta = d
			}
			if d := math.Abs(ng - float64(cur.G)); d > maxDelta {
				maxDelta = d
			}
			if d := math.Abs(nb - float64(cur.B)); d > maxDelta {
				maxDelta = d
			}
			px[i].R = clampByteF(nr)
			px[i].G = clampByteF(ng)
			px[i].B = clampByteF(nb)
			px[i].A = 255
		}
		if maxDelta < 0.5 {
			break
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
