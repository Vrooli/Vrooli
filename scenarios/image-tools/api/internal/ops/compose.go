package ops

import (
	"fmt"
	"image"
	"image/color"

	"github.com/disintegration/imaging"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Overlay composites a watermark onto the base image. Exactly one of
// p.OverlayImage (an image watermark) or p.Text (a text watermark/annotation)
// must be set. Position anchors the overlay (default bottom-right for
// watermarks); Opacity blends an image overlay (default 1.0); Color/FontSize
// style text.
func Overlay(base image.Image, p *Params) (image.Image, error) {
	switch {
	case len(p.OverlayImage) > 0 && p.Text != "":
		return nil, fmt.Errorf("ops: overlay accepts either an image or text, not both")
	case len(p.OverlayImage) > 0:
		return overlayImage(base, p)
	case p.Text != "":
		return overlayText(base, p)
	default:
		return nil, fmt.Errorf("ops: overlay requires an overlay image or text")
	}
}

func overlayImage(base image.Image, p *Params) (image.Image, error) {
	wm, _, err := Decode(p.OverlayImage)
	if err != nil {
		return nil, fmt.Errorf("ops: decode overlay image: %w", err)
	}
	opacity := p.Opacity
	if opacity <= 0 {
		opacity = 1.0
	}
	bg := imaging.Clone(base)
	pos := overlayAnchorPoint(p.Position, bg.Bounds(), wm.Bounds(), "bottom-right")
	return imaging.Overlay(bg, wm, pos, opacity), nil
}

func overlayText(base image.Image, p *Params) (image.Image, error) {
	col, err := parseHexColor(p.Color)
	if err != nil {
		return nil, err
	}
	if p.Color == "" {
		col = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	}
	size := p.FontSize
	if size <= 0 {
		size = 24
	}
	face, err := newFontFace(size)
	if err != nil {
		return nil, err
	}
	defer face.Close()

	dst := imaging.Clone(base)
	// Measure so we can anchor the text box.
	d := &font.Drawer{Face: face}
	textWidth := d.MeasureString(p.Text).Ceil()
	metrics := face.Metrics()
	textHeight := (metrics.Ascent + metrics.Descent).Ceil()

	pt := overlayAnchorPoint(p.Position, dst.Bounds(), image.Rect(0, 0, textWidth, textHeight), "bottom-right")
	d.Dst = dst
	d.Src = image.NewUniform(col)
	// font.Drawer draws from the baseline; offset by the ascent.
	d.Dot = fixed.P(pt.X, pt.Y+metrics.Ascent.Ceil())
	d.DrawString(p.Text)
	return dst, nil
}

func newFontFace(size float64) (font.Face, error) {
	ttf, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, fmt.Errorf("ops: parse font: %w", err)
	}
	face, err := opentype.NewFace(ttf, &opentype.FaceOptions{Size: size, DPI: 72})
	if err != nil {
		return nil, fmt.Errorf("ops: build font face: %w", err)
	}
	return face, nil
}

// overlayAnchorPoint returns the top-left point to place an inner rect within an
// outer rect for the given position name, with a small margin. Falls back to
// fallback when position is empty.
func overlayAnchorPoint(position string, outer, inner image.Rectangle, fallback string) image.Point {
	const margin = 12
	name := position
	if name == "" {
		name = fallback
	}
	ow, oh := outer.Dx(), outer.Dy()
	iw, ih := inner.Dx(), inner.Dy()
	pt := anchoredPoint(anchorFor(name), ow, oh, iw, ih)
	// Apply margins for edge/corner anchors.
	a := anchorFor(name)
	switch a {
	case imaging.TopLeft, imaging.Left, imaging.BottomLeft:
		pt.X += margin
	case imaging.TopRight, imaging.Right, imaging.BottomRight:
		pt.X -= margin
	}
	switch a {
	case imaging.TopLeft, imaging.Top, imaging.TopRight:
		pt.Y += margin
	case imaging.BottomLeft, imaging.Bottom, imaging.BottomRight:
		pt.Y -= margin
	}
	return pt
}
