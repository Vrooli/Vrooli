package diff

import (
	"image"
	"math"

	"github.com/disintegration/imaging"
)

// pixelStats bundles the byte-level comparison metrics.
type pixelStats struct {
	changed         int64
	total           int64
	changedFraction float64
	mae             float64
	rmse            float64
	psnr            float64
}

// pixelMetrics computes the per-pixel comparison of two equally-sized RGBA
// images. A pixel counts as "changed" when any channel's normalized difference
// exceeds tolerance (0..1). MAE/RMSE are mean per-channel differences in 0..255;
// PSNR is derived from the mean squared error (capped at psnrIdentical when MSE
// is ~0). Alpha is included so transparency changes are caught.
func pixelMetrics(base, cmp *image.NRGBA, tolerance float64) pixelStats {
	b := base.Bounds()
	w, h := b.Dx(), b.Dy()
	total := int64(w) * int64(h)
	if total == 0 {
		return pixelStats{total: 0, psnr: psnrIdentical}
	}
	tolByte := tolerance * 255

	var changed int64
	var absSum float64
	var sqSum float64
	for y := 0; y < h; y++ {
		bi := base.PixOffset(base.Rect.Min.X, base.Rect.Min.Y+y)
		ci := cmp.PixOffset(cmp.Rect.Min.X, cmp.Rect.Min.Y+y)
		brow := base.Pix[bi : bi+w*4]
		crow := cmp.Pix[ci : ci+w*4]
		for x := 0; x < w*4; x += 4 {
			var maxDelta float64
			for ch := 0; ch < 4; ch++ {
				d := math.Abs(float64(brow[x+ch]) - float64(crow[x+ch]))
				absSum += d
				sqSum += d * d
				if d > maxDelta {
					maxDelta = d
				}
			}
			if maxDelta > tolByte {
				changed++
			}
		}
	}

	samples := float64(total) * 4
	mae := absSum / samples
	mse := sqSum / samples
	rmse := math.Sqrt(mse)
	psnr := psnrIdentical
	if mse > 1e-9 {
		psnr = 10 * math.Log10((255*255)/mse)
		if psnr > psnrIdentical {
			psnr = psnrIdentical
		}
	}

	return pixelStats{
		changed:         changed,
		total:           total,
		changedFraction: float64(changed) / float64(total),
		mae:             round2(mae),
		rmse:            round2(rmse),
		psnr:            round2(psnr),
	}
}

// perceptualHash computes a 64-bit DCT-based perceptual hash (pHash). The image
// is reduced to 32x32 grayscale, a 2D DCT is taken, the top-left 8x8
// low-frequency block (excluding the DC term's bias) is compared to its median,
// and each coefficient becomes one bit. Resolution-independent and robust to
// re-encode / minor noise.
func perceptualHash(img image.Image) uint64 {
	const n = 32
	const lo = 8
	gray := imaging.Grayscale(imaging.Resize(img, n, n, imaging.Lanczos))

	// Load luminance into a float matrix.
	vals := make([][]float64, n)
	for y := 0; y < n; y++ {
		vals[y] = make([]float64, n)
		for x := 0; x < n; x++ {
			r, _, _, _ := gray.At(gray.Bounds().Min.X+x, gray.Bounds().Min.Y+y).RGBA()
			vals[y][x] = float64(r >> 8)
		}
	}

	dct := dct2D(vals, n)

	// Collect the low-frequency block, excluding the DC term (0,0).
	coeffs := make([]float64, 0, lo*lo-1)
	for y := 0; y < lo; y++ {
		for x := 0; x < lo; x++ {
			if x == 0 && y == 0 {
				continue
			}
			coeffs = append(coeffs, dct[y][x])
		}
	}
	med := medianOf(coeffs)

	var hash uint64
	bit := 0
	for y := 0; y < lo; y++ {
		for x := 0; x < lo; x++ {
			if x == 0 && y == 0 {
				continue
			}
			if dct[y][x] > med {
				hash |= 1 << uint(bit)
			}
			bit++
		}
	}
	return hash
}

// dct2D returns the 2D type-II DCT of an n×n matrix (separable: rows then cols).
func dct2D(m [][]float64, n int) [][]float64 {
	// 1D DCT-II basis is precomputable per size; n=32 is small so direct is fine.
	cos := make([][]float64, n)
	for u := 0; u < n; u++ {
		cos[u] = make([]float64, n)
		for x := 0; x < n; x++ {
			cos[u][x] = math.Cos((2*float64(x) + 1) * float64(u) * math.Pi / (2 * float64(n)))
		}
	}
	// Rows.
	rows := make([][]float64, n)
	for y := 0; y < n; y++ {
		rows[y] = make([]float64, n)
		for u := 0; u < n; u++ {
			var sum float64
			for x := 0; x < n; x++ {
				sum += m[y][x] * cos[u][x]
			}
			rows[y][u] = sum
		}
	}
	// Columns.
	out := make([][]float64, n)
	for v := 0; v < n; v++ {
		out[v] = make([]float64, n)
	}
	for u := 0; u < n; u++ {
		for v := 0; v < n; v++ {
			var sum float64
			for y := 0; y < n; y++ {
				sum += rows[y][u] * cos[v][y]
			}
			out[v][u] = sum
		}
	}
	return out
}

// hammingDistance counts differing bits between two 64-bit hashes.
func hammingDistance(a, b uint64) int {
	return popcount(a ^ b)
}

func popcount(x uint64) int {
	count := 0
	for x != 0 {
		x &= x - 1
		count++
	}
	return count
}

// ssim returns a global structural-similarity estimate (0..1) over the two
// grayscale images. This is the single-window (whole-image) SSIM: it captures
// luminance, contrast, and covariance — a robust, cheap structural score that
// complements the per-pixel and perceptual metrics.
func ssim(base, cmp *image.NRGBA) float64 {
	b := base.Bounds()
	w, h := b.Dx(), b.Dy()
	n := float64(w * h)
	if n == 0 {
		return 1
	}

	var sumX, sumY, sumXX, sumYY, sumXY float64
	for y := 0; y < h; y++ {
		bi := base.PixOffset(base.Rect.Min.X, base.Rect.Min.Y+y)
		ci := cmp.PixOffset(cmp.Rect.Min.X, cmp.Rect.Min.Y+y)
		brow := base.Pix[bi : bi+w*4]
		crow := cmp.Pix[ci : ci+w*4]
		for x := 0; x < w*4; x += 4 {
			lx := luma(brow[x], brow[x+1], brow[x+2])
			ly := luma(crow[x], crow[x+1], crow[x+2])
			sumX += lx
			sumY += ly
			sumXX += lx * lx
			sumYY += ly * ly
			sumXY += lx * ly
		}
	}

	muX := sumX / n
	muY := sumY / n
	varX := sumXX/n - muX*muX
	varY := sumYY/n - muY*muY
	covXY := sumXY/n - muX*muY

	const l = 255.0
	c1 := (0.01 * l) * (0.01 * l)
	c2 := (0.03 * l) * (0.03 * l)
	num := (2*muX*muY + c1) * (2*covXY + c2)
	den := (muX*muX + muY*muY + c1) * (varX + varY + c2)
	if den == 0 {
		return 1
	}
	s := num / den
	return round4(clamp01(s))
}

func luma(r, g, b uint8) float64 {
	return 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
}

func medianOf(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := append([]float64(nil), xs...)
	// insertion sort is fine for the 63-element pHash block.
	for i := 1; i < len(cp); i++ {
		for j := i; j > 0 && cp[j-1] > cp[j]; j-- {
			cp[j-1], cp[j] = cp[j], cp[j-1]
		}
	}
	mid := len(cp) / 2
	if len(cp)%2 == 1 {
		return cp[mid]
	}
	return (cp[mid-1] + cp[mid]) / 2
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }
