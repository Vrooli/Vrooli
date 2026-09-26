package analysis

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"math"
	"sort"
	"strconv"
	"strings"

	internalops "image-tools/internal/ops"

	exif "github.com/dsoprea/go-exif/v3"
)

// Probe returns structured info about an image. It is pure-Go: no model, no GPU,
// no external program — the always-headless analysis op. It reuses the ops codec
// layer for decoding and the go-exif reader for metadata presence.
func Probe(src []byte) (ProbeResult, error) {
	img, meta, err := internalops.Decode(src)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("analysis: decode: %w", err)
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	res := ProbeResult{
		Width:          w,
		Height:         h,
		Format:         meta.Format,
		ColorModel:     colorModelName(img),
		HasAlpha:       hasAlpha(img),
		FrameCount:     frameCount(src, meta.Format),
		Megapixels:     round2(float64(w) * float64(h) / 1e6),
		SizeBytes:      int64(len(src)),
		DominantColors: dominantColors(img, 5),
	}
	res.HasEXIF, res.HasGPS, res.Orientation = exifFacts(src)
	return res, nil
}

func colorModelName(img image.Image) string {
	switch img.(type) {
	case *image.RGBA, *image.NRGBA:
		return "rgba"
	case *image.RGBA64, *image.NRGBA64:
		return "rgba64"
	case *image.Gray:
		return "gray"
	case *image.Gray16:
		return "gray16"
	case *image.YCbCr:
		return "ycbcr"
	case *image.CMYK:
		return "cmyk"
	case *image.Paletted:
		return "paletted"
	default:
		return "unknown"
	}
}

func hasAlpha(img image.Image) bool {
	switch t := img.(type) {
	case *image.RGBA, *image.NRGBA, *image.RGBA64, *image.NRGBA64:
		return true
	case *image.Paletted:
		for _, c := range t.Palette {
			if _, _, _, a := c.RGBA(); a < 0xffff {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// frameCount reports the number of frames; >1 only for animated GIF. WebP/AVIF
// animation frame-counting is left to a later refinement (decode-only support).
func frameCount(src []byte, format string) int {
	if format != internalops.FormatGIF {
		return 1
	}
	g, err := gif.DecodeAll(bytes.NewReader(src))
	if err != nil || len(g.Image) == 0 {
		return 1
	}
	return len(g.Image)
}

// exifFacts reports EXIF presence, GPS presence, and the orientation tag (0 when
// absent) using the same reader the metadata op uses.
func exifFacts(src []byte) (hasEXIF, hasGPS bool, orientation int) {
	raw, err := exif.SearchAndExtractExif(src)
	if err != nil {
		return false, false, 0
	}
	hasEXIF = true
	tags, _, err := exif.GetFlatExifData(raw, nil)
	if err != nil {
		return hasEXIF, false, 0
	}
	for _, t := range tags {
		if strings.Contains(strings.ToLower(t.IfdPath), "gps") {
			hasGPS = true
		}
		if t.TagName == "Orientation" {
			if n, err := strconv.Atoi(strings.TrimSpace(t.Formatted)); err == nil {
				orientation = n
			}
		}
	}
	return hasEXIF, hasGPS, orientation
}

// dominantColors extracts up to n palette swatches by coarse RGB bucketing over
// a sampled grid (cheap, deterministic — good enough for a probe summary).
func dominantColors(img image.Image, n int) []DominantColor {
	const bucket = 32 // 8 levels per channel
	counts := make(map[uint32]int)
	total := 0
	b := img.Bounds()
	stepX := stride(b.Dx())
	stepY := stride(b.Dy())
	for y := b.Min.Y; y < b.Max.Y; y += stepY {
		for x := b.Min.X; x < b.Max.X; x += stepX {
			r, g, bl, a := img.At(x, y).RGBA()
			if a < 0x8000 {
				continue // skip mostly-transparent pixels
			}
			r8, g8, b8 := uint32(r>>8), uint32(g>>8), uint32(bl>>8)
			key := (r8/bucket)<<16 | (g8/bucket)<<8 | (b8 / bucket)
			counts[key]++
			total++
		}
	}
	if total == 0 {
		return nil
	}
	type bc struct {
		key   uint32
		count int
	}
	list := make([]bc, 0, len(counts))
	for k, c := range counts {
		list = append(list, bc{k, c})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].count != list[j].count {
			return list[i].count > list[j].count
		}
		return list[i].key < list[j].key
	})
	if len(list) > n {
		list = list[:n]
	}
	out := make([]DominantColor, 0, len(list))
	for _, e := range list {
		r := ((e.key>>16)&0xff)*bucket + bucket/2
		g := ((e.key>>8)&0xff)*bucket + bucket/2
		bl := (e.key&0xff)*bucket + bucket/2
		out = append(out, DominantColor{
			Hex:      fmt.Sprintf("#%02x%02x%02x", clamp8(r), clamp8(g), clamp8(bl)),
			Fraction: round2(float64(e.count) / float64(total)),
		})
	}
	return out
}

func stride(dim int) int {
	const target = 64 // sample ~64 points per axis
	if dim <= target {
		return 1
	}
	return dim / target
}

func clamp8(v uint32) uint32 {
	if v > 255 {
		return 255
	}
	return v
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }
