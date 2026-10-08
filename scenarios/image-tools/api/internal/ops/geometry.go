package ops

import (
	"fmt"
	"image"

	"github.com/disintegration/imaging"
)

// Resize scales an image. Mode is taken from p.Fit:
//   - "fit"     contain within WxH preserving aspect (the default when one of
//     W/H is 0, imaging derives the missing side);
//   - "fill"    cover WxH then center/anchor-crop (no distortion);
//   - "stretch" exact WxH, ignoring aspect.
func Resize(img image.Image, p *Params) (image.Image, error) {
	if p.Width <= 0 && p.Height <= 0 {
		return nil, fmt.Errorf("ops: resize requires width and/or height")
	}
	mode := p.Fit
	if mode == "" {
		if p.Width == 0 || p.Height == 0 {
			mode = "fit"
		} else {
			mode = "stretch"
		}
	}
	switch mode {
	case "fit":
		return imaging.Fit(img, dimOr(p.Width, maxInt), dimOr(p.Height, maxInt), resampleFilter()), nil
	case "fill":
		if p.Width <= 0 || p.Height <= 0 {
			return nil, fmt.Errorf("ops: resize fill requires both width and height")
		}
		return imaging.Fill(img, p.Width, p.Height, anchorFor(p.Gravity), resampleFilter()), nil
	case "stretch":
		return imaging.Resize(img, p.Width, p.Height, resampleFilter()), nil
	default:
		return nil, fmt.Errorf("ops: unknown resize fit %q (want fit|fill|stretch)", mode)
	}
}

// Crop cuts a rectangle from the image. With explicit W/H and zero X/Y it does
// a gravity-anchored crop; otherwise it crops the X,Y,W,H rectangle.
func Crop(img image.Image, p *Params) (image.Image, error) {
	if p.Width <= 0 || p.Height <= 0 {
		return nil, fmt.Errorf("ops: crop requires positive width and height")
	}
	b := img.Bounds()
	if p.X == 0 && p.Y == 0 && p.Gravity != "" {
		return imaging.CropAnchor(img, p.Width, p.Height, anchorFor(p.Gravity)), nil
	}
	rect := image.Rect(p.X, p.Y, p.X+p.Width, p.Y+p.Height).Add(b.Min)
	if !rect.In(b) {
		return nil, fmt.Errorf("ops: crop rect %v outside image bounds %v", rect, b)
	}
	return imaging.Crop(img, rect), nil
}

// Rotate rotates by p.Angle degrees counter-clockwise. Exact multiples of 90
// use the lossless fast paths; arbitrary angles fill exposed corners with the
// (optional) background color and expand the canvas.
func Rotate(img image.Image, p *Params) (image.Image, error) {
	bg, err := parseHexColor(p.Background)
	if err != nil {
		return nil, err
	}
	switch normAngle(p.Angle) {
	case 0:
		return imaging.Clone(img), nil
	case 90:
		return imaging.Rotate90(img), nil
	case 180:
		return imaging.Rotate180(img), nil
	case 270:
		return imaging.Rotate270(img), nil
	default:
		return imaging.Rotate(img, p.Angle, bg), nil
	}
}

// Flip mirrors the image along p.Axis ("horizontal" | "vertical").
func Flip(img image.Image, p *Params) (image.Image, error) {
	switch p.Axis {
	case "horizontal", "h", "":
		return imaging.FlipH(img), nil
	case "vertical", "v":
		return imaging.FlipV(img), nil
	default:
		return nil, fmt.Errorf("ops: unknown flip axis %q (want horizontal|vertical)", p.Axis)
	}
}

// Deskew auto-detects a small rotation that maximizes horizontal projection
// variance (the standard text-deskew heuristic) and straightens the image. The
// search is bounded to +/-15 degrees; outside that range a document is usually
// intentionally rotated, not skewed.
func Deskew(img image.Image, p *Params) (image.Image, error) {
	angle := detectSkew(img)
	if angle == 0 {
		return imaging.Clone(img), nil
	}
	bg, err := parseHexColor(p.Background)
	if err != nil {
		return nil, err
	}
	// imaging.Rotate is counter-clockwise; detectSkew returns the angle the
	// content is rotated by, so rotate by -angle to correct it.
	return imaging.Rotate(img, -angle, bg), nil
}

// Thumbnail produces a WxH thumbnail by scaling to fill then center-cropping.
func Thumbnail(img image.Image, p *Params) (image.Image, error) {
	if p.Width <= 0 || p.Height <= 0 {
		return nil, fmt.Errorf("ops: thumbnail requires positive width and height")
	}
	return imaging.Thumbnail(img, p.Width, p.Height, resampleFilter()), nil
}

// Canvas places the image on a WxH background of p.Background, anchored by
// p.Gravity (default center). Used for padding/extending to a target size
// without distortion. If W/H are 0 the source dimensions are used.
func Canvas(img image.Image, p *Params) (image.Image, error) {
	b := img.Bounds()
	w := dimOr(p.Width, b.Dx())
	h := dimOr(p.Height, b.Dy())
	bg, err := parseHexColor(p.Background)
	if err != nil {
		return nil, err
	}
	canvas := imaging.New(w, h, bg)
	pos := anchoredPoint(anchorFor(p.Gravity), w, h, b.Dx(), b.Dy())
	return imaging.Paste(canvas, img, pos), nil
}

// --- helpers ---

const maxInt = int(^uint(0) >> 1)

func dimOr(v, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return v
}

// normAngle returns the angle if it is an exact multiple of 90 within
// [0,360), else -1 (signaling the arbitrary-angle path).
func normAngle(a float64) int {
	if a != float64(int(a)) {
		return -1
	}
	n := int(a) % 360
	if n < 0 {
		n += 360
	}
	switch n {
	case 0, 90, 180, 270:
		return n
	default:
		return -1
	}
}

// anchoredPoint computes the top-left paste point for an inner WxH box of size
// iw,ih on an outer canvas of size ow,oh given an anchor.
func anchoredPoint(a imaging.Anchor, ow, oh, iw, ih int) image.Point {
	x, y := (ow-iw)/2, (oh-ih)/2 // center default
	switch a {
	case imaging.TopLeft:
		x, y = 0, 0
	case imaging.Top:
		x, y = (ow-iw)/2, 0
	case imaging.TopRight:
		x, y = ow-iw, 0
	case imaging.Left:
		x, y = 0, (oh-ih)/2
	case imaging.Right:
		x, y = ow-iw, (oh-ih)/2
	case imaging.BottomLeft:
		x, y = 0, oh-ih
	case imaging.Bottom:
		x, y = (ow-iw)/2, oh-ih
	case imaging.BottomRight:
		x, y = ow-iw, oh-ih
	}
	return image.Pt(x, y)
}
