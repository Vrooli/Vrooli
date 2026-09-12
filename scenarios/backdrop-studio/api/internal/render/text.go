package render

import (
	"image"
	"image/color"
	"strings"
)

// The preview's type is drawn from a compiled-in bitmap face rather than a
// font file, for the same reason the rest of this package is a pure function:
// a placement preview must render identically on every host, offline, with no
// font resolution to go wrong. The face is 5x7 with a one-pixel gap, blitted at
// an integer scale so strokes stay crisp instead of resampling to mush.
//
// It is deliberately not a typography specimen. It answers one question — does
// this copy read against this image — and real glyph shapes answer it where
// solid bars could not.
const (
	glyphWidth  = 5
	glyphHeight = 7
	glyphGap    = 1
)

// advance is the horizontal step for one character at a given scale.
func advance(scale int) int { return (glyphWidth + glyphGap) * scale }

func textWidth(text string, scale int) int {
	if text == "" {
		return 0
	}
	return len(text)*advance(scale) - glyphGap*scale
}

// drawTextScaled blits one line. Unknown runes advance without drawing, so a
// locale the face does not cover degrades to spacing rather than to garbage.
func drawTextScaled(dst *image.RGBA, x, y int, text string, scale int, c color.RGBA) {
	if scale < 1 {
		scale = 1
	}
	for _, r := range text {
		rows, ok := face[foldRune(r)]
		if !ok {
			x += advance(scale)
			continue
		}
		for gy, row := range rows {
			for gx, bit := range row {
				if bit != '1' {
					continue
				}
				for sy := 0; sy < scale; sy++ {
					for sx := 0; sx < scale; sx++ {
						px, py := x+gx*scale+sx, y+gy*scale+sy
						if px < dst.Bounds().Min.X || py < dst.Bounds().Min.Y || px >= dst.Bounds().Max.X || py >= dst.Bounds().Max.Y {
							continue
						}
						dst.SetRGBA(px, py, c)
					}
				}
			}
		}
		x += advance(scale)
	}
}

// drawWrappedText lays out a paragraph inside a rectangle and returns the y
// coordinate just below the last line it drew, so callers can stack blocks
// without tracking metrics themselves.
func drawWrappedText(dst *image.RGBA, r image.Rectangle, text string, scale int, c color.RGBA) int {
	if scale < 1 {
		scale = 1
	}
	lineHeight := glyphHeight*scale + scale*3
	y := r.Min.Y
	for _, line := range wrapText(text, scale, r.Dx()) {
		if y+glyphHeight*scale > r.Max.Y {
			break
		}
		drawTextScaled(dst, r.Min.X, y, line, scale, c)
		y += lineHeight
	}
	return y
}

// wrapText breaks on words, and only breaks inside a word when a single word is
// wider than the column.
func wrapText(text string, scale, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	perLine := width / advance(scale)
	if perLine < 1 {
		perLine = 1
	}
	var lines []string
	current := ""
	for _, word := range words {
		for len(word) > perLine {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			lines = append(lines, word[:perLine])
			word = word[perLine:]
		}
		candidate := word
		if current != "" {
			candidate = current + " " + word
		}
		if len(candidate) > perLine {
			lines = append(lines, current)
			current = word
			continue
		}
		current = candidate
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

// foldRune maps lowercase onto the uppercase face. The face is single-case
// because a 5x7 cell cannot carry a legible descender, and a preview that
// renders every word in caps still answers the legibility question honestly.
func foldRune(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 'a' + 'A'
	}
	return r
}

// face is the 5x7 bitmap. Rows are top to bottom, '1' is ink.
var face = map[rune][glyphHeight]string{
	'A':  {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'B':  {"11110", "10001", "10001", "11110", "10001", "10001", "11110"},
	'C':  {"01111", "10000", "10000", "10000", "10000", "10000", "01111"},
	'D':  {"11110", "10001", "10001", "10001", "10001", "10001", "11110"},
	'E':  {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'F':  {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'G':  {"01111", "10000", "10000", "10111", "10001", "10001", "01111"},
	'H':  {"10001", "10001", "10001", "11111", "10001", "10001", "10001"},
	'I':  {"11111", "00100", "00100", "00100", "00100", "00100", "11111"},
	'J':  {"00111", "00010", "00010", "00010", "00010", "10010", "01100"},
	'K':  {"10001", "10010", "10100", "11000", "10100", "10010", "10001"},
	'L':  {"10000", "10000", "10000", "10000", "10000", "10000", "11111"},
	'M':  {"10001", "11011", "10101", "10101", "10001", "10001", "10001"},
	'N':  {"10001", "11001", "10101", "10011", "10001", "10001", "10001"},
	'O':  {"01110", "10001", "10001", "10001", "10001", "10001", "01110"},
	'P':  {"11110", "10001", "10001", "11110", "10000", "10000", "10000"},
	'Q':  {"01110", "10001", "10001", "10001", "10101", "10010", "01101"},
	'R':  {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'S':  {"01111", "10000", "10000", "01110", "00001", "00001", "11110"},
	'T':  {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'U':  {"10001", "10001", "10001", "10001", "10001", "10001", "01110"},
	'V':  {"10001", "10001", "10001", "10001", "10001", "01010", "00100"},
	'W':  {"10001", "10001", "10001", "10101", "10101", "11011", "10001"},
	'X':  {"10001", "10001", "01010", "00100", "01010", "10001", "10001"},
	'Y':  {"10001", "10001", "01010", "00100", "00100", "00100", "00100"},
	'Z':  {"11111", "00001", "00010", "00100", "01000", "10000", "11111"},
	'0':  {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
	'1':  {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2':  {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3':  {"11110", "00001", "00001", "01110", "00001", "00001", "11110"},
	'4':  {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5':  {"11111", "10000", "10000", "11110", "00001", "00001", "11110"},
	'6':  {"01110", "10000", "10000", "11110", "10001", "10001", "01110"},
	'7':  {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8':  {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9':  {"01110", "10001", "10001", "01111", "00001", "00001", "01110"},
	'.':  {"00000", "00000", "00000", "00000", "00000", "01100", "01100"},
	',':  {"00000", "00000", "00000", "00000", "01100", "01100", "11000"},
	'!':  {"00100", "00100", "00100", "00100", "00100", "00000", "00100"},
	'?':  {"01110", "10001", "00001", "00110", "00100", "00000", "00100"},
	'\'': {"01100", "01100", "11000", "00000", "00000", "00000", "00000"},
	'-':  {"00000", "00000", "00000", "11111", "00000", "00000", "00000"},
	':':  {"00000", "01100", "01100", "00000", "01100", "01100", "00000"},
	'/':  {"00001", "00010", "00010", "00100", "01000", "01000", "10000"},
	'&':  {"01100", "10010", "10100", "01000", "10101", "10010", "01101"},
	' ':  {"00000", "00000", "00000", "00000", "00000", "00000", "00000"},
}
