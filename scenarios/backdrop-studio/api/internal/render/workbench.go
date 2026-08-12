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
	drawCopyBlock(page, copyRect)
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

// drawCopyBlock lays in a kicker, a two-line headline, two subhead lines and a
// call to action as solid bars. Real glyphs are unnecessary — the preview is
// judged on whether the copy mass is readable against the image behind it, and
// bars carry that mass honestly while staying locale-free.
func drawCopyBlock(dst *image.RGBA, r image.Rectangle) {
	if r.Dx() < 8 || r.Dy() < 8 {
		return
	}
	// Choose ink from what is actually behind the copy, so the preview shows a
	// legible pairing rather than assuming one.
	var sum float64
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y += 2 {
		for x := r.Min.X; x < r.Max.X; x += 2 {
			c := dst.RGBAAt(x, y)
			sum += (0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)) / 255
			n++
		}
	}
	dark := color.RGBA{18, 20, 26, 255}
	lightInk := color.RGBA{252, 252, 250, 255}
	ink, cta, ctaInk := lightInk, lightInk, dark
	if n > 0 && sum/float64(n) > 0.55 {
		ink, cta, ctaInk = dark, dark, lightInk
	}

	unit := r.Dy() / 22
	if unit < 2 {
		unit = 2
	}
	y := r.Min.Y
	bar := func(wFrac float64, height int, c color.RGBA) {
		bw := int(float64(r.Dx()) * wFrac)
		if bw > r.Dx() {
			bw = r.Dx()
		}
		if y+height > r.Max.Y {
			return
		}
		fill(dst, image.Rect(r.Min.X, y, r.Min.X+bw, y+height), c)
		y += height
	}
	gap := func(nUnits int) { y += unit * nUnits }

	bar(0.24, unit, ink) // kicker
	gap(2)
	bar(0.92, unit*3, ink) // headline line 1
	gap(1)
	bar(0.68, unit*3, ink) // headline line 2
	gap(2)
	bar(0.80, unit, ink) // subhead line 1
	gap(1)
	bar(0.62, unit, ink) // subhead line 2
	gap(2)
	// call to action
	btnW, btnH := int(float64(r.Dx())*0.30), unit*4
	if y+btnH <= r.Max.Y {
		fill(dst, image.Rect(r.Min.X, y, r.Min.X+btnW, y+btnH), cta)
		inset := btnH / 3
		if btnW > inset*4 {
			fill(dst, image.Rect(r.Min.X+inset, y+inset, r.Min.X+btnW-inset*3, y+btnH-inset), ctaInk)
		}
	}
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

	for y := target.Min.Y; y < target.Max.Y; y++ {
		fy := offY + float64(y-target.Min.Y)/scale
		sy := sb.Min.Y + int(fy)
		if sy < sb.Min.Y {
			sy = sb.Min.Y
		}
		if sy >= sb.Max.Y {
			sy = sb.Max.Y - 1
		}
		for x := target.Min.X; x < target.Max.X; x++ {
			fx := offX + float64(x-target.Min.X)/scale
			sx := sb.Min.X + int(fx)
			if sx < sb.Min.X {
				sx = sb.Min.X
			}
			if sx >= sb.Max.X {
				sx = sb.Max.X - 1
			}
			dst.Set(x, y, src.At(sx, sy))
		}
	}
}

var glyphs = map[rune][7]string{
	'A': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"}, 'B': {"11110", "10001", "10001", "11110", "10001", "10001", "11110"}, 'C': {"01111", "10000", "10000", "10000", "10000", "10000", "01111"}, 'D': {"11110", "10001", "10001", "10001", "10001", "10001", "11110"}, 'E': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"}, 'F': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"}, 'G': {"01111", "10000", "10000", "10111", "10001", "10001", "01111"}, 'H': {"10001", "10001", "10001", "11111", "10001", "10001", "10001"}, 'I': {"11111", "00100", "00100", "00100", "00100", "00100", "11111"}, 'J': {"00111", "00010", "00010", "00010", "00010", "10010", "01100"}, 'K': {"10001", "10010", "10100", "11000", "10100", "10010", "10001"}, 'L': {"10000", "10000", "10000", "10000", "10000", "10000", "11111"}, 'M': {"10001", "11011", "10101", "10101", "10001", "10001", "10001"}, 'N': {"10001", "11001", "10101", "10011", "10001", "10001", "10001"}, 'O': {"01110", "10001", "10001", "10001", "10001", "10001", "01110"}, 'P': {"11110", "10001", "10001", "11110", "10000", "10000", "10000"}, 'Q': {"01110", "10001", "10001", "10001", "10101", "10010", "01101"}, 'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"}, 'S': {"01111", "10000", "10000", "01110", "00001", "00001", "11110"}, 'T': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"}, 'U': {"10001", "10001", "10001", "10001", "10001", "10001", "01110"}, 'V': {"10001", "10001", "10001", "10001", "10001", "01010", "00100"}, 'W': {"10001", "10001", "10001", "10101", "10101", "11011", "10001"}, 'X': {"10001", "10001", "01010", "00100", "01010", "10001", "10001"}, 'Y': {"10001", "10001", "01010", "00100", "00100", "00100", "00100"}, 'Z': {"11111", "00001", "00010", "00100", "01000", "10000", "11111"}, '0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"}, '1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"}, '2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"}, '3': {"11110", "00001", "00001", "01110", "00001", "00001", "11110"}, '4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"}, '5': {"11111", "10000", "10000", "11110", "00001", "00001", "11110"}, '6': {"01110", "10000", "10000", "11110", "10001", "10001", "01110"}, '7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"}, '8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"}, '9': {"01110", "10001", "10001", "01111", "00001", "00001", "01110"}, '-': {"00000", "00000", "00000", "11111", "00000", "00000", "00000"}, ' ': {"00000", "00000", "00000", "00000", "00000", "00000", "00000"},
}

func drawText(dst draw.Image, x, y int, text string, c color.Color) {
	for _, r := range strings.ToUpper(text) {
		g, ok := glyphs[r]
		if !ok {
			x += 6
			continue
		}
		for yy, row := range g {
			for xx, v := range row {
				if v == '1' {
					dst.Set(x+xx, y+yy, c)
				}
			}
		}
		x += 6
	}
}
