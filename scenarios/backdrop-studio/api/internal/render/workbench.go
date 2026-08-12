package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
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
			preview := resizeNearest(img, w, h)
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

func resizeNearest(src image.Image, w, h int) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sx := src.Bounds().Min.X + x*src.Bounds().Dx()/w
			sy := src.Bounds().Min.Y + y*src.Bounds().Dy()/h
			out.Set(x, y, src.At(sx, sy))
		}
	}
	return out
}

func drawScaled(dst draw.Image, target image.Rectangle, src image.Image) {
	for y := target.Min.Y; y < target.Max.Y; y++ {
		for x := target.Min.X; x < target.Max.X; x++ {
			sx := src.Bounds().Min.X + (x-target.Min.X)*src.Bounds().Dx()/target.Dx()
			sy := src.Bounds().Min.Y + (y-target.Min.Y)*src.Bounds().Dy()/target.Dy()
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
