package compose

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"

	"backdrop-studio/internal/catalog"
)

type DeviceArrangement string
type Region struct {
	X, Y, Width, Height float64
	Kind                string
}

const (
	DeviceCenter DeviceArrangement = "device_center"
	CaptionAbove DeviceArrangement = "caption_above_device"
	CaptionBelow DeviceArrangement = "caption_below_device"
	CaptionOnly  DeviceArrangement = "caption_only"
)

// ComposeDeviceFrame places a supplied application screenshot into a stable
// device footprint. Capture remains owned by the mobile scenario; this seam
// accepts bytes and returns a composed PNG plus the reserved occlusion region.
func ComposeDeviceFrame(backdropPNG, screenshotPNG []byte, arrangement DeviceArrangement, caption string) ([]byte, Region, error) {
	backdrop, _, err := image.Decode(bytes.NewReader(backdropPNG))
	if err != nil {
		return nil, Region{}, fmt.Errorf("compose: decode backdrop: %w", err)
	}
	screenshot, _, err := image.Decode(bytes.NewReader(screenshotPNG))
	if err != nil {
		return nil, Region{}, fmt.Errorf("compose: decode screenshot: %w", err)
	}
	if arrangement != DeviceCenter && arrangement != CaptionAbove && arrangement != CaptionBelow && arrangement != CaptionOnly {
		return nil, Region{}, fmt.Errorf("compose: unsupported device arrangement %q", arrangement)
	}
	out := image.NewRGBA(backdrop.Bounds())
	draw.Draw(out, out.Bounds(), backdrop, image.Point{}, draw.Src)
	region := Region{X: .28, Y: .16, Width: .44, Height: .68, Kind: "occlusion"}
	if arrangement == CaptionOnly {
		region = Region{X: .08, Y: .18, Width: .84, Height: .24, Kind: "overlay"}
		drawCaption(out, caption, normalizedRect(out.Bounds(), region))
	} else {
		if arrangement == CaptionAbove {
			region = Region{X: .28, Y: .27, Width: .44, Height: .57, Kind: "occlusion"}
			drawCaption(out, caption, normalizedRect(out.Bounds(), Region{X: .08, Y: .07, Width: .84, Height: .13, Kind: "overlay"}))
		} else if arrangement == CaptionBelow {
			region = Region{X: .28, Y: .12, Width: .44, Height: .57, Kind: "occlusion"}
			drawCaption(out, caption, normalizedRect(out.Bounds(), Region{X: .08, Y: .78, Width: .84, Height: .13, Kind: "overlay"}))
		}
		deviceRect := normalizedRect(out.Bounds(), region)
		draw.Draw(out, inflate(deviceRect, 8), &image.Uniform{C: color.RGBA{A: 220}}, image.Point{}, draw.Over)
		drawScaled(out, deviceRect, screenshot)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, Region{}, err
	}
	return buf.Bytes(), region, nil
}

func normalizedRect(bounds image.Rectangle, region Region) image.Rectangle {
	return image.Rect(
		bounds.Min.X+int(region.X*float64(bounds.Dx())),
		bounds.Min.Y+int(region.Y*float64(bounds.Dy())),
		bounds.Min.X+int((region.X+region.Width)*float64(bounds.Dx())),
		bounds.Min.Y+int((region.Y+region.Height)*float64(bounds.Dy())),
	)
}

func inflate(rect image.Rectangle, pixels int) image.Rectangle {
	return image.Rect(rect.Min.X-pixels, rect.Min.Y-pixels, rect.Max.X+pixels, rect.Max.Y+pixels)
}

func drawScaled(dst draw.Image, target image.Rectangle, src image.Image) {
	if target.Empty() {
		return
	}
	for y := target.Min.Y; y < target.Max.Y; y++ {
		for x := target.Min.X; x < target.Max.X; x++ {
			sx := src.Bounds().Min.X + (x-target.Min.X)*src.Bounds().Dx()/target.Dx()
			sy := src.Bounds().Min.Y + (y-target.Min.Y)*src.Bounds().Dy()/target.Dy()
			dst.Set(x, y, src.At(sx, sy))
		}
	}
}

// drawCaption keeps the composition deterministic without coupling this domain
// to a font renderer. The real caption remains consumer-owned; these neutral
// bars reserve and preview its footprint in the store mockup.
func drawCaption(dst draw.Image, caption string, rect image.Rectangle) {
	if rect.Empty() || strings.TrimSpace(caption) == "" {
		return
	}
	draw.Draw(dst, rect, &image.Uniform{C: color.RGBA{A: 190}}, image.Point{}, draw.Over)
	line := rect.Inset(max(4, rect.Dy()/5))
	width := min(line.Dx(), max(8, len([]rune(caption))*3))
	draw.Draw(dst, image.Rect(line.Min.X, line.Min.Y, line.Min.X+width, line.Min.Y+max(2, line.Dy()/3)), &image.Uniform{C: color.RGBA{R: 245, G: 245, B: 245, A: 220}}, image.Point{}, draw.Over)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type Brief struct {
	BrandID, Placement, Prompt string
	Seed                       int64
}
type BrandPalette interface {
	ResolveToken(name string) (string, bool)
}
type MapPalette map[string]string

func (m MapPalette) ResolveToken(name string) (string, bool) { v, ok := m[name]; return v, ok }

type Operation struct{ Name, ParamsJSON string }
type Plan struct {
	StyleID, Strategy     string
	Operations            []Operation
	ResolvedSlots         map[string]string
	ExpectedExecutionPath string
	Executable            bool
}

func Resolve(style catalog.Style, brief Brief, palette BrandPalette, adapter string, adapterCommercialUse bool) (Plan, error) {
	if style.ID == "" || style.Strategy == "" {
		return Plan{}, fmt.Errorf("compose: style and strategy are required")
	}
	if brief.Placement != "" && !contains(style.Placements, brief.Placement) {
		return Plan{}, fmt.Errorf("compose: placement %q is not permitted by style %q", brief.Placement, style.ID)
	}
	if adapter != "" && !adapterCommercialUse {
		return Plan{}, fmt.Errorf("compose: adapter %q forbids commercial use", adapter)
	}
	plan := Plan{StyleID: style.ID, Strategy: style.Strategy, ResolvedSlots: map[string]string{}, Executable: true}
	for _, treatment := range style.Treatments {
		if strings.HasPrefix(treatment, "$brand.") {
			if palette == nil {
				return Plan{}, fmt.Errorf("compose: unresolved palette slot %s", treatment)
			}
			value, ok := palette.ResolveToken(treatment)
			if !ok || strings.TrimSpace(value) == "" {
				return Plan{}, fmt.Errorf("compose: unresolved palette slot %s", treatment)
			}
			plan.ResolvedSlots[treatment] = value
			continue
		}
		plan.Operations = append(plan.Operations, Operation{Name: treatment})
	}
	if len(plan.Operations) == 0 {
		return Plan{}, fmt.Errorf("compose: treatment chain is empty")
	}
	plan.ExpectedExecutionPath = "procedural"
	if style.Strategy == "guided" {
		plan.ExpectedExecutionPath = "scaffold → image-tools inference → treatment"
	}
	if style.Strategy == "synthesized" {
		plan.ExpectedExecutionPath = "image-tools inference → treatment"
	}
	return plan, nil
}
func contains(xs []string, value string) bool {
	for _, x := range xs {
		if x == value {
			return true
		}
	}
	return false
}
