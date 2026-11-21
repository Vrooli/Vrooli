package diff

import (
	"image"
	"math"
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

func round2(v float64) float64 { return math.Round(v*100) / 100 }
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }
