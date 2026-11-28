package ops

import (
	"image"
	"image/color"
)

// ScreenshotCleanup trims the uniform bands a screenshot carries around its
// subject: the window chrome, the frame a picture was embedded in, and letterbox
// bars from a capture that did not match the content's aspect.
//
// It is deliberately conservative and deterministic. A band is trimmed only when
// it is a flat colour that continues the colour at that edge of the frame, and
// never beyond max_trim_percent of the dimension, because a photograph whose
// subject touches the edge can present a flat row (a clear sky, a white wall)
// that is content, not chrome. The trade is explicit: this removes what surrounds
// the picture, and leaves anything printed on top of it — a floating button, a
// burned-in toolbar — to a masked edit (the `inpaint` step a cleanup Look chains
// after this one).
//
// Parameters:
//   - Tolerance: per-channel match tolerance as a fraction of full scale
//     (default 0.04, i.e. ~10 of 255). Raise it for banded/gradient chrome.
//   - MaxTrimPercent: safety ceiling per side (default 0.25).
//   - PaddingPx: margin kept (or restored) around the detected content.
func ScreenshotCleanup(img image.Image, p *Params) (image.Image, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 3 || h < 3 {
		return img, nil
	}

	tolerance := p.Tolerance
	if tolerance <= 0 {
		tolerance = 0.04
	}
	tol := tolerance * 255
	if tol < 1 {
		tol = 1
	}
	limit := p.MaxTrimPercent
	if limit <= 0 || limit > 0.45 {
		limit = 0.25
	}
	maxX, maxY := int(float64(w)*limit), int(float64(h)*limit)

	pixel := func(x, y int) nrgba {
		c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
		return nrgba{float64(c.R), float64(c.G), float64(c.B)}
	}
	// Each edge is matched against its own corner, and each scan runs on the
	// rectangle the previous ones left: chrome is frequently asymmetric (a white
	// title bar above a dark rail), so a single frame colour taken from the four
	// original corners would match neither band.
	rowsMatch := func(r image.Rectangle, y int, ref nrgba) bool {
		for x := r.Min.X; x < r.Max.X; x++ {
			if !pixel(x, y).within(ref, tol) {
				return false
			}
		}
		return true
	}
	colsMatch := func(r image.Rectangle, x int, ref nrgba) bool {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			if !pixel(x, y).within(ref, tol) {
				return false
			}
		}
		return true
	}

	r := b
	top, bottom, left, right := 0, 0, 0, 0
	// The reference is read once per edge, before that edge is scanned, and after
	// the earlier edges have moved the frame. Re-reading it inside the loop would
	// make every row its own reference and trim a uniform image to the ceiling.
	topRef := pixel(r.Min.X, r.Min.Y)
	for top < maxY && rowsMatch(r, r.Min.Y, topRef) {
		r.Min.Y++
		top++
	}
	bottomRef := pixel(r.Max.X-1, r.Max.Y-1)
	for bottom < maxY && rowsMatch(r, r.Max.Y-1, bottomRef) {
		r.Max.Y--
		bottom++
	}
	leftRef := pixel(r.Min.X, r.Min.Y)
	for left < maxX && colsMatch(r, r.Min.X, leftRef) {
		r.Min.X++
		left++
	}
	rightRef := pixel(r.Max.X-1, r.Max.Y-1)
	for right < maxX && colsMatch(r, r.Max.X-1, rightRef) {
		r.Max.X--
		right++
	}

	// A band that consumed the ceiling is chrome by definition: the scan stopped
	// because of the safety limit, not because content began. Keeping the original
	// frame is the honest outcome — a caller who wants more should raise the limit.
	if top >= maxY || bottom >= maxY || left >= maxX || right >= maxX {
		return img, nil
	}

	pad := int(p.PaddingPx)
	x0 := maxOf(b.Min.X+left-pad, b.Min.X)
	y0 := maxOf(b.Min.Y+top-pad, b.Min.Y)
	x1 := minOf(b.Max.X-right+pad, b.Max.X)
	y1 := minOf(b.Max.Y-bottom+pad, b.Max.Y)
	if x0 <= b.Min.X && y0 <= b.Min.Y && x1 >= b.Max.X && y1 >= b.Max.Y {
		return img, nil // nothing was trimmed
	}
	return copyRect(img, image.Rect(x0, y0, x1, y1)), nil
}

// copyRect copies a sub-rectangle into a new image. Cropping is a copy rather
// than a sub-image view because every downstream operation and encoder expects a
// zero-origin image, and a view would keep the source's stride and origin.
func copyRect(img image.Image, r image.Rectangle) image.Image {
	out := image.NewNRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	for y := 0; y < r.Dy(); y++ {
		for x := 0; x < r.Dx(); x++ {
			out.SetNRGBA(x, y, color.NRGBAModel.Convert(img.At(r.Min.X+x, r.Min.Y+y)).(color.NRGBA))
		}
	}
	return out
}

// nrgba is a float64 RGB triple used for tolerance comparisons; the conversion
// keeps the arithmetic readable and avoids integer truncation drift when
// averaging two edge colours.
type nrgba struct{ r, g, b float64 }

func (c nrgba) avg(o nrgba) nrgba { return nrgba{(c.r + o.r) / 2, (c.g + o.g) / 2, (c.b + o.b) / 2} }

func (c nrgba) within(ref nrgba, tol float64) bool {
	return abs(c.r-ref.r) <= tol && abs(c.g-ref.g) <= tol && abs(c.b-ref.b) <= tol
}

func maxOf(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minOf(a, b int) int {
	if a < b {
		return a
	}
	return b
}
