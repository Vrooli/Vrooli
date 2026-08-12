package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strings"
)

// SheetCell is one axis pair in a contact-sheet sweep. Labels are rendered in
// the sheet itself so an exported image remains useful outside the workbench.
type SheetCell struct {
	RowLabel, ColumnLabel string
	PNG                   []byte
}

func ContactSheet(cells []SheetCell, columns int) ([]byte, error) {
	if len(cells) == 0 || columns <= 0 {
		return nil, fmt.Errorf("render: contact sheet requires cells and a positive column count")
	}
	const cellW, cellH, labelH, rowW = 180, 120, 22, 92
	rows := (len(cells) + columns - 1) / columns
	out := image.NewRGBA(image.Rect(0, 0, rowW+columns*cellW, labelH+rows*cellH))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: color.RGBA{R: 18, G: 23, B: 34, A: 255}}, image.Point{}, draw.Src)
	for i, cell := range cells {
		col, row := i%columns, i/columns
		x, y := rowW+col*cellW, labelH+row*cellH
		drawText(out, x+4, 4, cell.ColumnLabel, color.RGBA{R: 235, G: 225, B: 193, A: 255})
		drawText(out, 4, y+4, cell.RowLabel, color.RGBA{R: 235, G: 225, B: 193, A: 255})
		img, err := png.Decode(bytes.NewReader(cell.PNG))
		if err != nil {
			return nil, fmt.Errorf("render: decode contact-sheet cell %d: %w", i, err)
		}
		drawScaled(out, image.Rect(x+2, y+labelH, x+cellW-2, y+cellH-2), img)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type PlacementPreview struct {
	Placement, Viewport string
	PNG                 []byte
	Ratio               float64
	Passes              bool
}

// PreviewPlacements makes the desktop/mobile result matrix explicit. The
// callback is deliberately injected so render remains independent of the
// legibility service and can be used in deterministic unit tests.
func PreviewPlacements(candidate []byte, placements []string, verdict func([]byte, string) (float64, bool)) ([]PlacementPreview, error) {
	if len(candidate) == 0 {
		return nil, fmt.Errorf("render: candidate image is required")
	}
	img, _, err := image.Decode(bytes.NewReader(candidate))
	if err != nil {
		return nil, fmt.Errorf("render: decode candidate: %w", err)
	}
	var out []PlacementPreview
	for _, placement := range placements {
		for _, viewport := range []string{"desktop", "mobile"} {
			w, h := 1280, 720
			if viewport == "mobile" {
				w, h = 390, 844
			}
			preview := composePlacement(img, placement, w, h)
			var b bytes.Buffer
			if err := png.Encode(&b, preview); err != nil {
				return nil, err
			}
			ratio, passes := 0.0, false
			if verdict != nil {
				ratio, passes = verdict(b.Bytes(), placement)
			}
			out = append(out, PlacementPreview{Placement: placement, Viewport: viewport, PNG: b.Bytes(), Ratio: ratio, Passes: passes})
		}
	}
	return out, nil
}

// composePlacement renders the candidate inside a mock page layout.
//
// This used to ignore the placement entirely and simply resize the candidate to
// the viewport, so full_bleed and split_panel produced byte-identical previews
// and the artifact showed no placement at all. A placement preview exists to
// answer one question — does this image work behind this copy, in this
// arrangement — so it has to draw the copy and the arrangement.
func composePlacement(src image.Image, placement string, w, h int) *image.RGBA {
	page := image.NewRGBA(image.Rect(0, 0, w, h))
	paper := color.RGBA{247, 247, 245, 255}
	fill(page, page.Bounds(), paper)

	mobile := h > w
	var imageRect, copyRect image.Rectangle
	scrim := "none"

	switch placement {
	case "split_panel":
		if mobile {
			imageRect = image.Rect(0, 0, w, h*45/100)
			copyRect = image.Rect(w*7/100, h*54/100, w*93/100, h*88/100)
		} else {
			imageRect = image.Rect(w/2, 0, w, h)
			copyRect = image.Rect(w*7/100, h*28/100, w*44/100, h*74/100)
		}
	case "framed_inset":
		if mobile {
			imageRect = image.Rect(w*5/100, h*6/100, w*95/100, h*46/100)
			copyRect = image.Rect(w*7/100, h*54/100, w*93/100, h*86/100)
		} else {
			imageRect = image.Rect(w*6/100, h*7/100, w*94/100, h*62/100)
			copyRect = image.Rect(w*6/100, h*70/100, w*62/100, h*92/100)
		}
	case "corner_bleed":
		if mobile {
			imageRect = image.Rect(w*30/100, 0, w, h*40/100)
			copyRect = image.Rect(w*7/100, h*50/100, w*93/100, h*84/100)
		} else {
			imageRect = image.Rect(w*46/100, 0, w, h*78/100)
			copyRect = image.Rect(w*6/100, h*30/100, w*42/100, h*74/100)
		}
	default: // full_bleed
		imageRect = page.Bounds()
		scrim = "left"
		if mobile {
			copyRect = image.Rect(w*8/100, h*52/100, w*92/100, h*86/100)
			scrim = "bottom"
		} else {
			copyRect = image.Rect(w*6/100, h*26/100, w*52/100, h*76/100)
		}
	}

	drawScaled(page, imageRect, src)
	if scrim != "none" {
		applyScrim(page, imageRect, scrim)
	}
	drawCopyBlock(page, copyRect, DefaultCopy(), "")
	return page
}

func fill(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			dst.SetRGBA(x, y, c)
		}
	}
}

// applyScrim washes the image so overlaid copy has contrast to sit on. It is
// the same device the legibility gate measures, so the preview shows what the
// gate scores rather than an untreated image.
func applyScrim(dst *image.RGBA, r image.Rectangle, direction string) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			var t float64
			switch direction {
			case "bottom":
				t = float64(y-r.Min.Y) / math.Max(1, float64(r.Dy()-1))
			default: // left
				t = 1 - float64(x-r.Min.X)/math.Max(1, float64(r.Dx()-1))*1.7
			}
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}
			a := 0.82 * t
			c := dst.RGBAAt(x, y)
			dst.SetRGBA(x, y, color.RGBA{
				R: uint8(float64(c.R)*(1-a) + 8*a),
				G: uint8(float64(c.G)*(1-a) + 10*a),
				B: uint8(float64(c.B)*(1-a) + 16*a),
				A: 255,
			})
		}
	}
}

// CopyDeck is the text a placement preview renders. It is real copy rather than
// placeholder bars because the preview exists to answer exactly one question —
// can someone read this headline against this image — and a black rectangle
// cannot answer it. An operator judging a backdrop from bars is judging the
// bars.
type CopyDeck struct {
	Kicker, Headline, Subhead, CTA string
}

// DefaultCopy is representative landing-page copy: a real headline length, a
// real subhead length, a real call to action. The lengths matter more than the
// words, because they are what decides whether the copy mass collides with the
// focal subject.
func DefaultCopy() CopyDeck {
	return CopyDeck{
		Kicker:   "AMBIENT IMAGERY",
		Headline: "Ship what others are still planning",
		Subhead:  "Art-directed backdrops that carry mood and craft signal while your copy stays legible on top.",
		CTA:      "START BUILDING",
	}
}

// drawCopyBlock lays in a kicker, a headline, a subhead and a call to action as
// rendered type at the style's declared text colour.
//
// It used to draw solid bars. Bars carry copy *mass* honestly, which is why the
// choice was defensible, but they cannot show a stroke disappearing into a busy
// midtone or a serif dissolving into a halftone dot — and those are the failures
// a backdrop actually has.
func drawCopyBlock(dst *image.RGBA, r image.Rectangle, deck CopyDeck, declared string) {
	if r.Dx() < 40 || r.Dy() < 40 {
		return
	}
	// Choose ink from what is actually behind the copy, unless the style
	// declared one — a declared text colour is an art-direction decision and
	// the preview must show it, including when it is a bad decision.
	ink := inkForRegion(dst, r, declared)
	dark := color.RGBA{18, 20, 26, 255}
	lightInk := color.RGBA{252, 252, 250, 255}
	ctaInk := dark
	if luminanceOf(ink) < 0.5 {
		ctaInk = lightInk
	}

	unit := r.Dy() / 26
	if unit < 2 {
		unit = 2
	}
	y := r.Min.Y

	// Scales are relative to the copy column so the deck fits at every viewport.
	// The first version set the headline to a full unit, which on a desktop hero
	// rendered ~90px glyphs and truncated the headline after two words — the
	// preview then answered a question nobody asked.
	kickerScale := maxInt(1, unit/4)
	headlineScale := maxInt(2, unit*2/3)
	bodyScale := maxInt(1, unit/3)

	y = drawWrappedText(dst, image.Rect(r.Min.X, y, r.Max.X, r.Max.Y), deck.Kicker, kickerScale, ink)
	y += unit
	y = drawWrappedText(dst, image.Rect(r.Min.X, y, r.Max.X, r.Max.Y), deck.Headline, headlineScale, ink)
	y += unit
	y = drawWrappedText(dst, image.Rect(r.Min.X, y, r.Max.X, r.Max.Y), deck.Subhead, bodyScale, ink)
	y += unit * 2

	// Call to action: a filled button with knocked-out type.
	btnH := glyphHeight*bodyScale + bodyScale*6
	btnW := textWidth(deck.CTA, bodyScale) + bodyScale*12
	if btnW > r.Dx() {
		btnW = r.Dx()
	}
	if y+btnH <= r.Max.Y {
		fill(dst, image.Rect(r.Min.X, y, r.Min.X+btnW, y+btnH), ink)
		drawTextScaled(dst, r.Min.X+bodyScale*6, y+bodyScale*3, deck.CTA, bodyScale, ctaInk)
	}
}

// inkForRegion picks readable type colour. A declared colour wins; otherwise the
// preview measures what is behind the copy and picks the readable pole.
func inkForRegion(dst *image.RGBA, r image.Rectangle, declared string) color.RGBA {
	if parsed, ok := parseHexColor(declared); ok {
		return parsed
	}
	var sum float64
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y += 2 {
		for x := r.Min.X; x < r.Max.X; x += 2 {
			c := dst.RGBAAt(x, y)
			sum += (0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)) / 255
			n++
		}
	}
	if n > 0 && sum/float64(n) > 0.55 {
		return color.RGBA{18, 20, 26, 255}
	}
	return color.RGBA{252, 252, 250, 255}
}

func parseHexColor(value string) (color.RGBA, bool) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(value) != 6 {
		return color.RGBA{}, false
	}
	var rgb [3]uint8
	for i := 0; i < 3; i++ {
		var v int
		for j := 0; j < 2; j++ {
			c := value[i*2+j]
			var d int
			switch {
			case c >= '0' && c <= '9':
				d = int(c - '0')
			case c >= 'a' && c <= 'f':
				d = int(c-'a') + 10
			case c >= 'A' && c <= 'F':
				d = int(c-'A') + 10
			default:
				return color.RGBA{}, false
			}
			v = v*16 + d
		}
		rgb[i] = uint8(v)
	}
	return color.RGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: 255}, true
}

func luminanceOf(c color.RGBA) float64 {
	return (0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)) / 255
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func drawScaled(dst draw.Image, target image.Rectangle, src image.Image) {
	sb := src.Bounds()
	if sb.Dx() <= 0 || sb.Dy() <= 0 || target.Dx() <= 0 || target.Dy() <= 0 {
		return
	}
	// Scale so the source covers the target on both axes, then centre it.
	scale := math.Max(float64(target.Dx())/float64(sb.Dx()), float64(target.Dy())/float64(sb.Dy()))
	srcW := float64(target.Dx()) / scale
	srcH := float64(target.Dy()) / scale
	offX := (float64(sb.Dx()) - srcW) / 2
	offY := (float64(sb.Dy()) - srcH) / 2

	// The filter radius follows the scale: when several source pixels collapse
	// into one destination pixel, the kernel has to be wide enough to see all
	// of them, or the ones it skips are the aliasing.
	radius := 1.0
	if scale < 1 {
		radius = 1 / scale
	}

	for y := target.Min.Y; y < target.Max.Y; y++ {
		fy := offY + (float64(y-target.Min.Y)+0.5)/scale
		for x := target.Min.X; x < target.Max.X; x++ {
			fx := offX + (float64(x-target.Min.X)+0.5)/scale
			dst.Set(x, y, sampleFiltered(src, fx, fy, radius, scale < 1))
		}
	}
}

// sampleFiltered evaluates one destination pixel. Weights are accumulated in
// premultiplied-alpha space so a transparent neighbour cannot bleed its colour
// into an opaque result.
func sampleFiltered(src image.Image, fx, fy, radius float64, downscale bool) color.NRGBA {
	sb := src.Bounds()
	minX := int(math.Floor(fx - radius))
	maxX := int(math.Ceil(fx + radius))
	minY := int(math.Floor(fy - radius))
	maxY := int(math.Ceil(fy + radius))

	var sumR, sumG, sumB, sumA, sumW float64
	for sy := minY; sy <= maxY; sy++ {
		wy := filterWeight((float64(sy)+0.5-fy)/radius, downscale)
		if wy == 0 {
			continue
		}
		cy := clampInt(sb.Min.Y+sy, sb.Min.Y, sb.Max.Y-1)
		for sx := minX; sx <= maxX; sx++ {
			wx := filterWeight((float64(sx)+0.5-fx)/radius, downscale)
			if wx == 0 {
				continue
			}
			cx := clampInt(sb.Min.X+sx, sb.Min.X, sb.Max.X-1)
			r, g, b, a := src.At(cx, cy).RGBA()
			w := wx * wy
			sumR += float64(r>>8) * w
			sumG += float64(g>>8) * w
			sumB += float64(b>>8) * w
			sumA += float64(a>>8) * w
			sumW += w
		}
	}
	if sumW == 0 {
		return color.NRGBA{}
	}
	return color.NRGBA{
		R: clampByte(sumR / sumW),
		G: clampByte(sumG / sumW),
		B: clampByte(sumB / sumW),
		A: clampByte(sumA / sumW),
	}
}

// filterWeight is Catmull-Rom for downscale and a triangle (bilinear) for
// upscale, both evaluated on a normalised distance.
func filterWeight(t float64, catmullRom bool) float64 {
	t = math.Abs(t)
	if !catmullRom {
		if t >= 1 {
			return 0
		}
		return 1 - t
	}
	// Catmull-Rom: B = 0, C = 0.5 in the Mitchell-Netravali family.
	switch {
	case t < 1:
		return 1.5*t*t*t - 2.5*t*t + 1
	case t < 2:
		return -0.5*t*t*t + 2.5*t*t - 4*t + 2
	default:
		return 0
	}
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

func clampByte(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v + 0.5)
}

// drawText labels a contact sheet at 1x from the same face the previews use, so
// there is one glyph table in this package rather than two that can drift.
func drawText(dst draw.Image, x, y int, text string, c color.Color) {
	for _, r := range text {
		rows, ok := face[foldRune(r)]
		if !ok {
			x += advance(1)
			continue
		}
		for gy, row := range rows {
			for gx, bit := range row {
				if bit == '1' {
					dst.Set(x+gx, y+gy, c)
				}
			}
		}
		x += advance(1)
	}
}
