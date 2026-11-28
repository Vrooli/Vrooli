package ops

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"

	"github.com/disintegration/imaging"
)

const printPrepMaxPixels = 100_000_000

// printPrepRun resamples to the exact physical box at the requested resolution,
// then tags the PNG with that resolution. 1-bit mode means binary black ink on
// transparent alpha; PNG's storage channel depth remains an 8-bit RGBA carrier.
func printPrepRun(in RunInput) (RunResult, error) {
	p := in.Params
	if p.WidthIn <= 0 || p.HeightIn <= 0 {
		return RunResult{}, fmt.Errorf("print_prep: width_in and height_in must be greater than zero")
	}
	dpi := p.DPI
	if dpi == 0 {
		dpi = 300
	}
	if dpi < 72 || dpi > 1200 {
		return RunResult{}, fmt.Errorf("print_prep: dpi must be between 72 and 1200")
	}
	width := int(math.Round(p.WidthIn * float64(dpi)))
	height := int(math.Round(p.HeightIn * float64(dpi)))
	if width < 1 || height < 1 || int64(width)*int64(height) > printPrepMaxPixels {
		return RunResult{}, fmt.Errorf("print_prep: output box %dx%d exceeds the %d-pixel safety limit", width, height, printPrepMaxPixels)
	}
	mode := strings.ToLower(strings.TrimSpace(p.InkMode))
	if mode == "" {
		mode = "1bit"
	}
	if mode != "1bit" && mode != "colour" && mode != "color" {
		return RunResult{}, fmt.Errorf("print_prep: ink_mode must be 1bit or colour")
	}
	threshold := p.InkThreshold
	if threshold == 0 {
		threshold = 0.5
	}
	if threshold < 0 || threshold > 1 {
		return RunResult{}, fmt.Errorf("print_prep: threshold must be in 0..1")
	}
	// Trim to the requested aspect ratio before resampling so print preparation
	// never stretches artwork to fit a page box.
	source := in.Img
	sb := source.Bounds()
	wantAspect := float64(width) / float64(height)
	sourceAspect := float64(sb.Dx()) / float64(sb.Dy())
	if sourceAspect > wantAspect {
		cropWidth := int(math.Round(float64(sb.Dy()) * wantAspect))
		if cropWidth < 1 {
			cropWidth = 1
		}
		source = imaging.CropAnchor(source, cropWidth, sb.Dy(), anchorFor(p.Gravity))
	} else if sourceAspect < wantAspect {
		cropHeight := int(math.Round(float64(sb.Dx()) / wantAspect))
		if cropHeight < 1 {
			cropHeight = 1
		}
		source = imaging.CropAnchor(source, sb.Dx(), cropHeight, anchorFor(p.Gravity))
	}
	resized := imaging.Resize(source, width, height, imaging.Lanczos)
	out := image.NewNRGBA(image.Rect(0, 0, width, height))
	ink := 0
	for y := 0; y < height; y++ {
		if err := in.Context.Err(); err != nil {
			return RunResult{}, err
		}
		for x := 0; x < width; x++ {
			c := color.NRGBAModel.Convert(resized.At(x, y)).(color.NRGBA)
			if mode == "1bit" {
				luma := (0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)) / 255
				if c.A != 0 && luma <= threshold {
					out.SetNRGBA(x, y, color.NRGBA{R: 0, G: 0, B: 0, A: c.A})
					ink++
				} else {
					out.SetNRGBA(x, y, color.NRGBA{})
				}
			} else {
				out.SetNRGBA(x, y, c)
				if c.A != 0 {
					ink++
				}
			}
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, out); err != nil {
		return RunResult{}, fmt.Errorf("print_prep: encode PNG: %w", err)
	}
	data, err := withPNGResolution(encoded.Bytes(), dpi)
	if err != nil {
		return RunResult{}, err
	}
	resolved := map[string]float64{"dpi": float64(dpi), "width_px": float64(width), "height_px": float64(height)}
	if mode == "1bit" {
		resolved["ink_coverage"] = float64(ink) / float64(width*height)
	}
	return RunResult{
		Bytes: data, Format: FormatPNG, Mime: MIMEFor(FormatPNG), Width: width, Height: height,
		ResolvedParams: resolved,
	}, nil
}

// withPNGResolution inserts the standard PNG pHYs chunk after IHDR. PNG stores
// pixels per metre; conversion is rounded once so equivalent runs are stable.
func withPNGResolution(src []byte, dpi int) ([]byte, error) {
	if len(src) < 33 || string(src[:8]) != "\x89PNG\r\n\x1a\n" || string(src[12:16]) != "IHDR" {
		return nil, fmt.Errorf("print_prep: encoder returned an invalid PNG")
	}
	ppm := uint32(math.Round(float64(dpi) / 0.0254))
	payload := make([]byte, 9)
	binary.BigEndian.PutUint32(payload[0:4], ppm)
	binary.BigEndian.PutUint32(payload[4:8], ppm)
	payload[8] = 1 // metre units
	chunk := make([]byte, 4+4+len(payload)+4)
	binary.BigEndian.PutUint32(chunk[:4], uint32(len(payload)))
	copy(chunk[4:8], "pHYs")
	copy(chunk[8:17], payload)
	binary.BigEndian.PutUint32(chunk[17:21], crc32.ChecksumIEEE(chunk[4:17]))
	out := make([]byte, 0, len(src)+len(chunk))
	out = append(out, src[:33]...)
	out = append(out, chunk...)
	out = append(out, src[33:]...)
	return out, nil
}
