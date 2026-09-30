package ops

import (
	"image"
	"image/color"
	"os"
	"strings"
	"testing"
)

func TestVectorizeSquareWithHole(t *testing.T) {
	w, h := 64, 64
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{255, 255, 255, 255})
		}
	}
	for y := 16; y < 48; y++ {
		for x := 16; x < 48; x++ {
			img.Set(x, y, color.NRGBA{0, 0, 0, 255})
		}
	}
	for y := 28; y < 36; y++ {
		for x := 28; x < 36; x++ {
			img.Set(x, y, color.NRGBA{255, 255, 255, 255})
		}
	}
	out, err := Vectorize(RunInput{Img: img, Params: &Params{DropBackgroundLayers: true, Colors: 2, MinAreaPx: 20}})
	if err != nil {
		t.Fatalf("Vectorize: %v", err)
	}
	if out.Format != FormatSVG || out.Mime != "image/svg+xml" {
		t.Fatalf("expected svg output, got %s/%s", out.Format, out.Mime)
	}
	svg := string(out.Bytes)
	if !strings.Contains(svg, "<path ") {
		t.Fatalf("expected a path, got %s", svg)
	}
	// The pure-Go rasterizer ignores fill-rule=evenodd, so the tracer must emit
	// nonzero-compatible winding instead.
	if strings.Contains(svg, "fill-rule") {
		t.Fatalf("paths must not rely on fill-rule, got %s", svg)
	}
	if got := strings.Count(svg, "M"); got < 2 {
		t.Fatalf("expected the square outer boundary and its hole, got %d subpath(s): %s", got, svg)
	}
}

func TestVectorizeAquilaParity(t *testing.T) {
	ref, err := os.ReadFile("testdata/aquila-clean-512.png")
	if err != nil {
		t.Skipf("parity fixture missing: %v", err)
	}
	refImg, _, err := Decode(ref)
	if err != nil {
		t.Fatalf("decode reference: %v", err)
	}
	b := refImg.Bounds()
	// The approved raster is the rounded tile on a near-white page. The mark is
	// compared only inside the tile ("the reference region"): the page margin is
	// not part of the mark and would otherwise dominate the metric.
	var sr, sg, sb int64
	var n int64
	minX, minY, maxX, maxY := b.Dx(), b.Dy(), -1, -1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(refImg.At(x, y)).(color.NRGBA)
			if maxChannel(c) < 128 {
				sr += int64(c.R)
				sg += int64(c.G)
				sb += int64(c.B)
				n++
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	if n == 0 {
		n = 1
	}
	avg := rgb{uint8(sr / n), uint8(sg / n), uint8(sb / n)}
	_ = minX
	_ = minY
	_ = maxX
	_ = maxY

	out, err := Vectorize(RunInput{Img: refImg, Params: &Params{
		KeepColors:                 []string{"#ffffff", "#22d3ee"},
		ClipToLargestRoundedRegion: true,
		InsetPx:                    30,
		TolerancePx:                0.8,
	}})
	if err != nil {
		t.Fatalf("Vectorize: %v", err)
	}
	t.Logf("vector svg bytes=%d", len(out.Bytes))

	flat, err := flattenOnto(mustRasterize(t, out.Bytes, 512), hexRGB(avg))
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}

	// Compare only inside the rounded tile region (the mark's ground), not the
	// page margin or the corners outside the tile, which are not part of the mark.
	px := make([]rgb, b.Dx()*b.Dy())
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := color.NRGBAModel.Convert(refImg.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			px[y*b.Dx()+x] = rgb{c.R, c.G, c.B}
		}
	}
	_, labels, _ := quantize(px, 0, []string{"#ffffff", "#22d3ee"})
	regionMask := clipRegionMask(labels, nil, []int{0, 1}, nil, b.Dx(), b.Dy())
	region := regionFromHull(regionMask, b.Dx(), b.Dy(), 30)

	mismatch := mismatchFractionMask(flat, refImg, region, b.Dx(), b.Dy(), 96)
	white := countSubpaths(string(out.Bytes), "#ffffff")
	cyan := countSubpaths(string(out.Bytes), "#22d3ee")
	t.Logf("parity mismatch fraction=%.4f contours: white=%d cyan=%d", mismatch, white, cyan)
	if white < 1 {
		t.Fatalf("expected at least one white contour, got %d", white)
	}
	if cyan < 6 {
		t.Fatalf("expected at least six accent contours, got %d", cyan)
	}
	if mismatch > 0.03 {
		t.Fatalf("parity mismatch %.4f exceeds 0.03", mismatch)
	}
}

func mustRasterize(t *testing.T, svg []byte, size int) image.Image {
	t.Helper()
	img, err := rasterizeSVG(svg, size, size)
	if err != nil {
		t.Fatalf("rasterize output svg: %v", err)
	}
	return img
}

func mismatchFraction(a, b image.Image, tol int) float64 {
	ab, bb := a.Bounds(), b.Bounds()
	w := minInt(ab.Dx(), bb.Dx())
	h := minInt(ab.Dy(), bb.Dy())
	diff := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ca := color.NRGBAModel.Convert(a.At(ab.Min.X+x, ab.Min.Y+y)).(color.NRGBA)
			cb := color.NRGBAModel.Convert(b.At(bb.Min.X+x, bb.Min.Y+y)).(color.NRGBA)
			if maxChannelDiff(ca, cb) > tol {
				diff++
			}
		}
	}
	return float64(diff) / float64(w*h)
}

func maxChannelDiff(a, b color.NRGBA) int {
	d := absInt(int(a.R) - int(b.R))
	if v := absInt(int(a.G) - int(b.G)); v > d {
		d = v
	}
	if v := absInt(int(a.B) - int(b.B)); v > d {
		d = v
	}
	return d
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func maxChannel(c color.NRGBA) int {
	m := int(c.R)
	if int(c.G) > m {
		m = int(c.G)
	}
	if int(c.B) > m {
		m = int(c.B)
	}
	return m
}

// countSubpaths returns the number of contour subpaths in the path whose fill
// matches fill.
func countSubpaths(svg string, fill string) int {
	idx := strings.Index(svg, `fill="`+fill+`"`)
	if idx < 0 {
		return 0
	}
	start := strings.Index(svg[idx:], `d="`)
	if start < 0 {
		return 0
	}
	rest := svg[idx+start+3:]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return 0
	}
	return strings.Count(rest[:end], "M")
}

func mismatchFractionIn(a, b image.Image, region image.Rectangle, tol int) float64 {
	ab := a.Bounds()
	diff, total := 0, 0
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			if x < ab.Min.X || x >= ab.Max.X || y < ab.Min.Y || y >= ab.Max.Y {
				continue
			}
			total++
			ca := color.NRGBAModel.Convert(a.At(x, y)).(color.NRGBA)
			cb := color.NRGBAModel.Convert(b.At(x, y)).(color.NRGBA)
			if maxChannelDiff(ca, cb) > tol {
				diff++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(diff) / float64(total)
}

func mismatchFractionMask(a, b image.Image, mask []bool, w, h, tol int) float64 {
	ab := a.Bounds()
	diff, total := 0, 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !mask[y*w+x] {
				continue
			}
			total++
			ca := color.NRGBAModel.Convert(a.At(ab.Min.X+x, ab.Min.Y+y)).(color.NRGBA)
			cb := color.NRGBAModel.Convert(b.At(x, y)).(color.NRGBA)
			if maxChannelDiff(ca, cb) > tol {
				diff++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(diff) / float64(total)
}
