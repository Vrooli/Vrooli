package ops

import (
	"image"
	"math"

	"github.com/disintegration/imaging"
)

// detectSkew estimates the document skew angle (degrees) using the projection-
// profile variance method: for each candidate angle it rotates a downscaled,
// binarized copy, sums dark pixels per row, and scores the variance of that
// row-sum profile. Text aligned to rows produces sharp peaks (high variance);
// the angle with maximum variance is the skew. Returns 0 when no candidate
// beats the unrotated baseline meaningfully.
func detectSkew(img image.Image) float64 {
	// Downscale for speed; deskew accuracy doesn't need full resolution.
	small := imaging.Resize(img, 600, 0, imaging.Box)
	if small.Bounds().Dx() == 0 {
		small = imaging.Clone(img)
	}
	gray := imaging.Grayscale(small)

	const (
		limit = 15.0 // search +/-15 degrees
		step  = 0.5
	)
	baseline := projectionVariance(gray)
	bestAngle := 0.0
	bestScore := baseline
	for a := -limit; a <= limit+1e-9; a += step {
		if math.Abs(a) < 1e-9 {
			continue
		}
		rotated := imaging.Rotate(gray, a, image.White)
		score := projectionVariance(rotated)
		if score > bestScore {
			bestScore = score
			bestAngle = a
		}
	}
	// Require a clear improvement over the unrotated baseline to avoid
	// "correcting" images that aren't skewed.
	if bestScore < baseline*1.05 {
		return 0
	}
	return bestAngle
}

// projectionVariance returns the variance of per-row dark-pixel counts for a
// grayscale image. Pixels darker than mid-gray count as ink.
func projectionVariance(gray *image.NRGBA) float64 {
	b := gray.Bounds()
	h := b.Dy()
	w := b.Dx()
	if h == 0 || w == 0 {
		return 0
	}
	rows := make([]float64, h)
	for y := 0; y < h; y++ {
		count := 0
		base := y * gray.Stride
		for x := 0; x < w; x++ {
			if gray.Pix[base+x*4] < 128 { // R channel == luminance for grayscale
				count++
			}
		}
		rows[y] = float64(count)
	}
	var sum, sumSq float64
	for _, r := range rows {
		sum += r
		sumSq += r * r
	}
	n := float64(h)
	mean := sum / n
	return sumSq/n - mean*mean
}
