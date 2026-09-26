// Package selection is image-tools' smart-select → classify → contextual-edit
// engine (IMG-P1-013). It turns a click point / box / "auto subject" prompt into
// a pixel mask that snaps to a region's silhouette, classifies the selected
// region with a deterministic heuristic, and resolves the class into a menu of
// contextual edits that compile to the existing AI op request shapes.
//
// The default segmenter is a built-in, pure-Go region-grow: it runs on ANY host
// with zero provisioning (the headless-completeness tenet, the same always-
// runnable floor naturalize establishes). A model-backed SAM path is the
// higher-quality upgrade, gated on a backend + weights like the other AI ops;
// when it is unavailable the service falls back to the built-in and warns.
//
// Everything here is pure Go over decoded pixels, so the segmentation and
// classification are deterministic and unit-testable without any external
// program or model weights.
package selection

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"

	"github.com/disintegration/imaging"
)

// maxWorkDim caps the longest edge the region-grow runs at. Large inputs are
// downscaled to this for the (BFS-bounded) segmentation, then the mask is
// resized back to the source dimensions. Keeps a click responsive and the work
// deterministic regardless of input megapixels.
const maxWorkDim = 1024

// Mode selects how the prompt is turned into a mask.
type Mode int

const (
	// ModePoint grows a region outward from seed point(s).
	ModePoint Mode = iota
	// ModeBox segments the dominant foreground object inside a box.
	ModeBox
	// ModeAuto extracts the foreground subject (border treated as background).
	ModeAuto
)

// Point is a normalized image coordinate (0..1).
type Point struct {
	X, Y     float64
	Negative bool
}

// Box is a normalized rectangle (0..1); X,Y = top-left.
type Box struct {
	X, Y, W, H float64
}

// Params are the parsed segmentation inputs.
type Params struct {
	Mode          Mode
	Points        []Point
	Box           *Box
	Tolerance     float64
	ModelOverride string
}

// defaultTolerance is the region-grow colour threshold (fraction of the max RGB
// distance ~441) used when the caller leaves Tolerance unset. Moderate: snaps
// to a coherent region without bleeding across soft gradients.
const defaultTolerance = 0.14

// effectiveTolerance clamps the knob into [0.02, 0.6] and defaults unset to
// defaultTolerance.
func (p Params) effectiveTolerance() float64 {
	t := p.Tolerance
	if t <= 0 {
		t = defaultTolerance
	}
	if t < 0.02 {
		t = 0.02
	}
	if t > 0.6 {
		t = 0.6
	}
	return t
}

// segMask is the working segmentation result over the (possibly downscaled)
// working image: a boolean per-pixel selection grid plus its dimensions.
type segMask struct {
	w, h int
	sel  []bool
}

func (m *segMask) at(x, y int) bool     { return m.sel[y*m.w+x] }
func (m *segMask) set(x, y int, v bool) { m.sel[y*m.w+x] = v }

// computed is the working-resolution segmentation: the downscaled NRGBA image,
// the boolean mask over it, the normalized bounding box, and the area fraction.
// It is the shared core so segmentation, classification, and mask encoding run
// off ONE region-grow pass.
type computed struct {
	work *image.NRGBA
	mask *segMask
	box  Box
	area float64
	// srcW, srcH are the source dimensions the encoded mask is resized back to.
	srcW, srcH int
}

// compute runs the built-in region-grow at the working resolution and returns
// the shared result. Pure function of (src, params).
func compute(src image.Image, p Params) (*computed, error) {
	full := imaging.Clone(src) // NRGBA at source resolution
	sb := full.Bounds()
	srcW, srcH := sb.Dx(), sb.Dy()
	if srcW == 0 || srcH == 0 {
		return nil, fmt.Errorf("selection: empty image")
	}

	// Downscale to the working resolution for the BFS.
	work := full
	if srcW > maxWorkDim || srcH > maxWorkDim {
		work = imaging.Fit(full, maxWorkDim, maxWorkDim, imaging.Box)
	}
	wb := work.Bounds()
	ww, wh := wb.Dx(), wb.Dy()

	var mask *segMask
	switch p.Mode {
	case ModeBox:
		if p.Box == nil {
			return nil, fmt.Errorf("selection: box mode requires a box")
		}
		mask = segmentBox(work, *p.Box, p.effectiveTolerance())
	case ModeAuto:
		mask = segmentAuto(work, p.effectiveTolerance())
	default: // ModePoint
		if len(p.Points) == 0 {
			return nil, fmt.Errorf("selection: point mode requires at least one point")
		}
		mask = segmentPoint(work, p.Points, p.effectiveTolerance())
	}

	keepLargestComponent(mask)
	fillHoles(mask)

	selCount, minX, minY, maxX, maxY := maskStats(mask)
	if selCount == 0 {
		// Nothing grew (e.g. an isolated pixel at a hard edge): select just the
		// seed pixel's cell so the caller always gets a non-empty, honest mask.
		mask = newMask(ww, wh)
		sx, sy := ww/2, wh/2
		if len(p.Points) > 0 {
			sx = clampInt(int(p.Points[0].X*float64(ww)), 0, ww-1)
			sy = clampInt(int(p.Points[0].Y*float64(wh)), 0, wh-1)
		}
		mask.set(sx, sy, true)
		minX, minY, maxX, maxY = sx, sy, sx, sy
		selCount = 1
	}

	return &computed{
		work: work,
		mask: mask,
		box: Box{
			X: float64(minX) / float64(ww),
			Y: float64(minY) / float64(wh),
			W: float64(maxX-minX+1) / float64(ww),
			H: float64(maxY-minY+1) / float64(wh),
		},
		area: float64(selCount) / float64(ww*wh),
		srcW: srcW,
		srcH: srcH,
	}, nil
}

// encodeMask renders the working mask to source-resolution gray PNG.
func (c *computed) encodeMask() ([]byte, error) {
	graySrc := imaging.Resize(maskToGray(c.mask), c.srcW, c.srcH, imaging.NearestNeighbor)
	return encodeGrayPNG(graySrc)
}

// Segment runs the built-in region-grow segmenter on src and returns the binary
// mask (PNG, white = selected) at the SOURCE resolution, plus the normalized
// bounding box and the selected area fraction. It is a pure function of
// (src, params).
func Segment(src image.Image, p Params) (maskPNG []byte, box Box, areaFraction float64, err error) {
	c, err := compute(src, p)
	if err != nil {
		return nil, Box{}, 0, err
	}
	maskPNG, err = c.encodeMask()
	if err != nil {
		return nil, Box{}, 0, err
	}
	return maskPNG, c.box, c.area, nil
}

func newMask(w, h int) *segMask { return &segMask{w: w, h: h, sel: make([]bool, w*h)} }

// segmentPoint grows a region from each positive seed, snapping to the colour/
// edge silhouette. tol is the fraction-of-max colour distance threshold.
func segmentPoint(img *image.NRGBA, points []Point, tol float64) *segMask {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	mask := newMask(w, h)
	tolAbs := tol * maxColorDist
	edgeTol := tolAbs * 0.7 // local smoothness: don't cross strong gradients

	var seeds [][2]int
	var negColors []color.NRGBA
	var seedSum [3]float64
	posCount := 0
	for _, pt := range points {
		px := clampInt(int(pt.X*float64(w)), 0, w-1)
		py := clampInt(int(pt.Y*float64(h)), 0, h-1)
		c := img.NRGBAAt(px, py)
		if pt.Negative {
			negColors = append(negColors, c)
			continue
		}
		seeds = append(seeds, [2]int{px, py})
		seedSum[0] += float64(c.R)
		seedSum[1] += float64(c.G)
		seedSum[2] += float64(c.B)
		posCount++
	}
	if posCount == 0 {
		return mask
	}
	seedColor := color.NRGBA{
		R: uint8(seedSum[0] / float64(posCount)),
		G: uint8(seedSum[1] / float64(posCount)),
		B: uint8(seedSum[2] / float64(posCount)),
	}

	queue := make([][2]int, 0, len(seeds))
	for _, s := range seeds {
		if !mask.at(s[0], s[1]) {
			mask.set(s[0], s[1], true)
			queue = append(queue, s)
		}
	}
	for len(queue) > 0 {
		p := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		px, py := p[0], p[1]
		cur := img.NRGBAAt(px, py)
		for _, d := range neighbors4 {
			nx, ny := px+d[0], py+d[1]
			if nx < 0 || ny < 0 || nx >= w || ny >= h || mask.at(nx, ny) {
				continue
			}
			nc := img.NRGBAAt(nx, ny)
			if colorDist(nc, seedColor) > tolAbs || colorDist(nc, cur) > edgeTol {
				continue
			}
			if nearAny(nc, negColors, tolAbs*0.5) {
				continue // soft exclusion around a negative seed
			}
			mask.set(nx, ny, true)
			queue = append(queue, [2]int{nx, ny})
		}
	}
	return mask
}

// segmentAuto extracts the foreground subject: it grows the BACKGROUND inward
// from every border pixel (colours similar to the frame), then the selection is
// everything the background did not reach.
func segmentAuto(img *image.NRGBA, tol float64) *segMask {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	bg := newMask(w, h)
	tolAbs := tol * maxColorDist
	edgeTol := tolAbs * 0.9

	queue := make([][2]int, 0, 2*(w+h))
	pushBorder := func(x, y int) {
		if !bg.at(x, y) {
			bg.set(x, y, true)
			queue = append(queue, [2]int{x, y})
		}
	}
	for x := 0; x < w; x++ {
		pushBorder(x, 0)
		pushBorder(x, h-1)
	}
	for y := 0; y < h; y++ {
		pushBorder(0, y)
		pushBorder(w-1, y)
	}
	for len(queue) > 0 {
		p := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		px, py := p[0], p[1]
		cur := img.NRGBAAt(px, py)
		for _, d := range neighbors4 {
			nx, ny := px+d[0], py+d[1]
			if nx < 0 || ny < 0 || nx >= w || ny >= h || bg.at(nx, ny) {
				continue
			}
			if colorDist(img.NRGBAAt(nx, ny), cur) > edgeTol {
				continue
			}
			bg.set(nx, ny, true)
			queue = append(queue, [2]int{nx, ny})
		}
	}
	// Foreground = NOT background.
	fg := newMask(w, h)
	for i := range fg.sel {
		fg.sel[i] = !bg.sel[i]
	}
	return fg
}

// segmentBox segments the dominant foreground inside the box: the box-border
// ring is sampled as the background colour model, and interior pixels far from
// it become the selection.
func segmentBox(img *image.NRGBA, box Box, tol float64) *segMask {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	mask := newMask(w, h)
	x0 := clampInt(int(box.X*float64(w)), 0, w-1)
	y0 := clampInt(int(box.Y*float64(h)), 0, h-1)
	x1 := clampInt(int((box.X+box.W)*float64(w)), 0, w-1)
	y1 := clampInt(int((box.Y+box.H)*float64(h)), 0, h-1)
	if x1 <= x0 || y1 <= y0 {
		return mask
	}
	// Sample the inner border ring (10% frame) as background.
	frameX := maxInt(1, (x1-x0)/10)
	frameY := maxInt(1, (y1-y0)/10)
	var bgSamples []color.NRGBA
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if x < x0+frameX || x > x1-frameX || y < y0+frameY || y > y1-frameY {
				bgSamples = append(bgSamples, img.NRGBAAt(x, y))
			}
		}
	}
	tolAbs := tol * maxColorDist
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if !nearAny(img.NRGBAAt(x, y), bgSamples, tolAbs) {
				mask.set(x, y, true)
			}
		}
	}
	return mask
}

// ---- pixel + mask helpers -------------------------------------------------

var neighbors4 = [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

// maxColorDist is the maximum euclidean RGB distance (sqrt(3*255^2)).
const maxColorDist = 441.6729559300637

func colorDist(a, b color.NRGBA) float64 {
	dr := float64(a.R) - float64(b.R)
	dg := float64(a.G) - float64(b.G)
	db := float64(a.B) - float64(b.B)
	return math.Sqrt(dr*dr + dg*dg + db*db)
}

func nearAny(c color.NRGBA, set []color.NRGBA, tol float64) bool {
	for _, s := range set {
		if colorDist(c, s) <= tol {
			return true
		}
	}
	return false
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// maskStats returns the selected-pixel count and the inclusive bounding box.
func maskStats(m *segMask) (count, minX, minY, maxX, maxY int) {
	minX, minY = m.w, m.h
	maxX, maxY = -1, -1
	for y := 0; y < m.h; y++ {
		for x := 0; x < m.w; x++ {
			if !m.at(x, y) {
				continue
			}
			count++
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
	return count, minX, minY, maxX, maxY
}

// keepLargestComponent reduces the mask to its single largest 4-connected
// component (drops speckle from a noisy grow). A no-op for empty masks.
func keepLargestComponent(m *segMask) {
	labels := make([]int, len(m.sel))
	best, bestLabel := 0, 0
	label := 0
	stack := make([][2]int, 0, 64)
	for y := 0; y < m.h; y++ {
		for x := 0; x < m.w; x++ {
			if !m.at(x, y) || labels[y*m.w+x] != 0 {
				continue
			}
			label++
			size := 0
			stack = stack[:0]
			stack = append(stack, [2]int{x, y})
			labels[y*m.w+x] = label
			for len(stack) > 0 {
				p := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				size++
				for _, d := range neighbors4 {
					nx, ny := p[0]+d[0], p[1]+d[1]
					if nx < 0 || ny < 0 || nx >= m.w || ny >= m.h {
						continue
					}
					idx := ny*m.w + nx
					if m.sel[idx] && labels[idx] == 0 {
						labels[idx] = label
						stack = append(stack, [2]int{nx, ny})
					}
				}
			}
			if size > best {
				best, bestLabel = size, label
			}
		}
	}
	if bestLabel == 0 {
		return
	}
	for i := range m.sel {
		m.sel[i] = labels[i] == bestLabel
	}
}

// fillHoles fills background pockets fully enclosed by the selection: flood the
// "outside" from the image border across unselected pixels; any unselected pixel
// not reached is an interior hole and becomes selected.
func fillHoles(m *segMask) {
	outside := make([]bool, len(m.sel))
	stack := make([][2]int, 0, 2*(m.w+m.h))
	pushOut := func(x, y int) {
		idx := y*m.w + x
		if !m.sel[idx] && !outside[idx] {
			outside[idx] = true
			stack = append(stack, [2]int{x, y})
		}
	}
	for x := 0; x < m.w; x++ {
		pushOut(x, 0)
		pushOut(x, m.h-1)
	}
	for y := 0; y < m.h; y++ {
		pushOut(0, y)
		pushOut(m.w-1, y)
	}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, d := range neighbors4 {
			nx, ny := p[0]+d[0], p[1]+d[1]
			if nx < 0 || ny < 0 || nx >= m.w || ny >= m.h {
				continue
			}
			idx := ny*m.w + nx
			if !m.sel[idx] && !outside[idx] {
				outside[idx] = true
				stack = append(stack, [2]int{nx, ny})
			}
		}
	}
	for i := range m.sel {
		if !m.sel[i] && !outside[i] {
			m.sel[i] = true
		}
	}
}

// maskToGray renders a boolean mask to a Gray image (255 selected, 0 not).
func maskToGray(m *segMask) *image.Gray {
	g := image.NewGray(image.Rect(0, 0, m.w, m.h))
	for i, v := range m.sel {
		if v {
			g.Pix[i] = 255
		}
	}
	return g
}

func encodeGrayPNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("selection: encode mask: %w", err)
	}
	return buf.Bytes(), nil
}

// DecodeInput decodes a segmentation input image honoring EXIF orientation.
func DecodeInput(data []byte) (image.Image, error) {
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("selection: decode input: %w", err)
	}
	return img, nil
}
