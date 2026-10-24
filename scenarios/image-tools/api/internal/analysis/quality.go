package analysis

import (
	"fmt"
	"image"
	"math"

	internalops "image-tools/internal/ops"
)

// blurThreshold is the Laplacian-variance cutoff below which an image is judged
// blurry. The variance-of-Laplacian is the standard no-reference sharpness
// proxy; ~100 is a widely-used boundary for normalized 8-bit luma.
const blurThreshold = 100.0

// QualityAssess computes no-reference image-quality heuristics: sharpness
// (variance of the Laplacian), brightness (mean luma), and contrast (luma
// standard deviation), plus a derived 0..1 overall score and human labels. Pure
// -Go (no model, no GPU) — the always-runnable analysis floor.
func QualityAssess(src []byte) (QualityResult, error) {
	img, _, err := internalops.Decode(src)
	if err != nil {
		return QualityResult{}, fmt.Errorf("analysis: decode: %w", err)
	}
	luma := lumaPlane(img)
	if luma.w == 0 || luma.h == 0 {
		return QualityResult{}, fmt.Errorf("analysis: empty image")
	}

	mean, std := meanStd(luma.px)
	sharp := laplacianVariance(luma)

	res := QualityResult{
		Sharpness:  round2(sharp),
		Blurry:     sharp < blurThreshold,
		Brightness: round2(mean),
		Contrast:   round2(std),
	}
	res.Exposure, res.Notes = exposureLabel(mean, std)
	if res.Blurry {
		res.Notes = append(res.Notes, "image appears soft / out of focus")
	}
	res.OverallScore = round2(qualityScore(sharp, mean, std))
	return res, nil
}

// grayPlane is a dense 8-bit luma plane.
type grayPlane struct {
	px   []float64
	w, h int
}

// lumaPlane extracts the Rec.601 luma of img into a dense float plane.
func lumaPlane(img image.Image) grayPlane {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	px := make([]float64, w*h)
	i := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			px[i] = 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(bl>>8)
			i++
		}
	}
	return grayPlane{px: px, w: w, h: h}
}

func meanStd(px []float64) (mean, std float64) {
	if len(px) == 0 {
		return 0, 0
	}
	var sum, sumSq float64
	for _, v := range px {
		sum += v
		sumSq += v * v
	}
	n := float64(len(px))
	mean = sum / n
	variance := sumSq/n - mean*mean
	if variance < 0 {
		variance = 0
	}
	return mean, math.Sqrt(variance)
}

// laplacianVariance convolves the luma plane with the 4-neighbour Laplacian and
// returns the variance of the response — the standard sharpness proxy.
func laplacianVariance(g grayPlane) float64 {
	if g.w < 3 || g.h < 3 {
		return 0
	}
	resp := make([]float64, 0, (g.w-2)*(g.h-2))
	at := func(x, y int) float64 { return g.px[y*g.w+x] }
	for y := 1; y < g.h-1; y++ {
		for x := 1; x < g.w-1; x++ {
			lap := at(x-1, y) + at(x+1, y) + at(x, y-1) + at(x, y+1) - 4*at(x, y)
			resp = append(resp, lap)
		}
	}
	_, std := meanStd(resp)
	return std * std
}

// exposureLabel maps brightness/contrast to a human exposure verdict + notes.
func exposureLabel(mean, std float64) (string, []string) {
	var notes []string
	label := "well-exposed"
	switch {
	case mean < 60:
		label = "underexposed"
		notes = append(notes, "image is dark")
	case mean > 200:
		label = "overexposed"
		notes = append(notes, "image is very bright / possibly clipped")
	}
	if std < 25 {
		notes = append(notes, "low contrast")
	}
	return label, notes
}

// qualityScore folds sharpness, exposure, and contrast into a 0..1 estimate.
func qualityScore(sharp, mean, std float64) float64 {
	// Sharpness component: saturates at ~400 variance.
	sharpComp := math.Min(sharp/400.0, 1.0)
	// Exposure component: 1.0 at mid-gray (128), tapering to 0 at the extremes.
	expComp := 1.0 - math.Abs(mean-128.0)/128.0
	if expComp < 0 {
		expComp = 0
	}
	// Contrast component: saturates at std 60.
	contrastComp := math.Min(std/60.0, 1.0)
	score := 0.5*sharpComp + 0.3*expComp + 0.2*contrastComp
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}
