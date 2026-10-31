package selection

import (
	"image"
	"math"
)

// Region classes the built-in heuristic can label. These name the contextual
// edit menus (menu.go) and the registry's segment/detection model ops upgrade
// their accuracy — but never their vocabulary (the menu is keyed on these).
const (
	ClassPerson     = "person"
	ClassSky        = "sky"
	ClassFoliage    = "foliage"
	ClassBackground = "background"
	ClassObject     = "object"
)

// regionStats are the per-region aggregates the classifier reasons over.
type regionStats struct {
	count          int
	meanR, meanG   float64
	meanB          float64
	meanLuma       float64 // 0..1
	meanSat        float64 // 0..1
	skinFraction   float64 // share of region pixels in a skin-tone gamut
	centroidY      float64 // 0 (top) .. 1 (bottom)
	areaFraction   float64
	touchesBorders int // count of image borders the bbox touches (0..4)
}

// Classify labels the selected region with a coarse class + a heuristic
// confidence in (0,1]. It NEVER performs face/identity recognition (a hard
// non-goal): "person" means "skin-tone-dominant region", nothing more.
//
// A deterministic function of (work image, mask): position, colour, saturation,
// skin-tone share, area, and border contact decide the class. The seeded
// detection model (yolox-tiny) is the accuracy upgrade documented in the proto;
// this floor always runs.
func Classify(work *image.NRGBA, mask *segMask, area float64) (class string, confidence float64) {
	st := computeRegionStats(work, mask, area)
	if st.count == 0 {
		return ClassObject, 0.3
	}

	// Ordered rules; the first match wins. Confidence reflects rule strength.
	// 1. Background: large + touches most of the frame.
	if st.areaFraction > 0.5 && st.touchesBorders >= 3 {
		return ClassBackground, clamp01(0.55 + 0.4*(st.areaFraction-0.5)/0.5)
	}
	// 2. Sky: upper region, bright, low saturation or distinctly blue.
	if st.centroidY < 0.45 && st.meanLuma > 0.5 && (st.meanSat < 0.35 || isBluish(st.meanR, st.meanG, st.meanB)) {
		conf := 0.5 + 0.3*(0.45-st.centroidY)/0.45
		if isBluish(st.meanR, st.meanG, st.meanB) {
			conf += 0.15
		}
		return ClassSky, clamp01(conf)
	}
	// 3. Person: skin-tone-dominant region.
	if st.skinFraction > 0.4 {
		return ClassPerson, clamp01(0.45 + 0.5*(st.skinFraction-0.4)/0.6)
	}
	// 4. Foliage: green-dominant.
	if isGreenish(st.meanR, st.meanG, st.meanB) && st.meanSat > 0.18 {
		return ClassFoliage, clamp01(0.45 + 0.3*st.meanSat)
	}
	// 5. Default: a generic object.
	return ClassObject, 0.4
}

func computeRegionStats(work *image.NRGBA, mask *segMask, area float64) regionStats {
	b := work.Bounds()
	w, h := b.Dx(), b.Dy()
	var st regionStats
	var sumR, sumG, sumB, sumLuma, sumSat float64
	var skin int
	var sumY float64
	minX, minY, maxX, maxY := w, h, -1, -1

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !mask.at(x, y) {
				continue
			}
			c := work.NRGBAAt(x, y)
			r, g, bl := float64(c.R), float64(c.G), float64(c.B)
			st.count++
			sumR += r
			sumG += g
			sumB += bl
			sumLuma += (0.299*r + 0.587*g + 0.114*bl) / 255.0
			sumSat += saturation(r, g, bl)
			sumY += float64(y)
			if isSkin(c.R, c.G, c.B) {
				skin++
			}
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if st.count == 0 {
		return st
	}
	n := float64(st.count)
	st.meanR, st.meanG, st.meanB = sumR/n, sumG/n, sumB/n
	st.meanLuma = sumLuma / n
	st.meanSat = sumSat / n
	st.skinFraction = float64(skin) / n
	st.centroidY = (sumY / n) / float64(h)
	st.areaFraction = area
	if minX == 0 {
		st.touchesBorders++
	}
	if minY == 0 {
		st.touchesBorders++
	}
	if maxX == w-1 {
		st.touchesBorders++
	}
	if maxY == h-1 {
		st.touchesBorders++
	}
	return st
}

// saturation is the HSV S of an RGB triple (0..1).
func saturation(r, g, b float64) float64 {
	mx := math.Max(r, math.Max(g, b))
	mn := math.Min(r, math.Min(g, b))
	if mx <= 0 {
		return 0
	}
	return (mx - mn) / mx
}

// isSkin is a coarse RGB skin-tone gate (the well-known Kovac rule, daylight
// branch). It detects skin TONES, not identities — never face recognition.
func isSkin(r8, g8, b8 uint8) bool {
	r, g, b := int(r8), int(g8), int(b8)
	mx := maxInt(r, maxInt(g, b))
	mn := minInt(r, minInt(g, b))
	return r > 95 && g > 40 && b > 20 &&
		(mx-mn) > 15 &&
		abs(r-g) > 15 && r > g && r > b
}

func isBluish(r, g, b float64) bool { return b > r+12 && b > g+6 }
func isGreenish(r, g, b float64) bool {
	return g > r+10 && g > b+10
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
