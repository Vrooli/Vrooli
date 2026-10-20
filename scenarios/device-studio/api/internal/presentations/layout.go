package presentations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	phoneRefWidth, phoneRefHeight     = 430, 880
	browserRefWidth, browserRefHeight = 1440, 900
)

func buildLayout(ctx context.Context, app image.Image, c Config) (Layout, error) {
	if app == nil {
		return Layout{}, ErrInvalidConfig
	}
	if (c.Phone == nil) == (c.Browser == nil) {
		return Layout{}, ErrInvalidConfig
	}
	bounds := app.Bounds()
	w64, h64 := int64(bounds.Max.X)-int64(bounds.Min.X), int64(bounds.Max.Y)-int64(bounds.Min.Y)
	if bounds.Empty() || w64 <= 0 || h64 <= 0 || w64 > maxRasterSide || h64 > maxRasterSide || w64*h64 > maxSourcePixels {
		return Layout{}, ErrInvalidConfig
	}
	if c.Target.X <= 0 || c.Target.Y <= 0 || c.Target.X > maxRasterSide || c.Target.Y > maxRasterSide || int64(c.Target.X)*int64(c.Target.Y) > maxRasterPixels {
		return Layout{}, ErrInvalidConfig
	}
	if err := validateAlphaAndMatte(ctx, app, c.Matte); err != nil {
		return Layout{}, err
	}
	top, bottom, err := resolveFills(ctx, app, c)
	if err != nil {
		return Layout{}, err
	}
	var profile, version string
	var viewport image.Rectangle
	if c.Phone != nil {
		if c.Phone.Appearance != "" && c.Phone.Appearance != AppearanceLight && c.Phone.Appearance != AppearanceDark {
			return Layout{}, ErrInvalidConfig
		}
		if c.Phone.BatteryPercent > 100 || len(c.Phone.Clock) > 8 || len(c.Phone.Carrier) > 32 {
			return Layout{}, ErrInvalidConfig
		}
		if !validText(c.Phone.Carrier, 32) {
			return Layout{}, ErrInvalidConfig
		}
		if c.Phone.Clock != "" && !validClock(c.Phone.Clock) {
			return Layout{}, ErrInvalidConfig
		}
		profile, version = PhoneProfileID, PhoneProfileVersion
		sy := float64(c.Target.Y) / phoneRefHeight
		topInset := int(math.Round(74 * sy))
		bottomInset := int(math.Round(34 * sy))
		viewport = image.Rect(0, topInset, c.Target.X, c.Target.Y-bottomInset)
		if viewport.Dx() < 240 || viewport.Dy() < 300 {
			return Layout{}, ErrInvalidConfig
		}
	} else {
		b := c.Browser
		if !validText(b.Title, 128) || !validText(b.Address, 2048) {
			return Layout{}, ErrInvalidConfig
		}
		if b.Appearance != "" && b.Appearance != AppearanceLight && b.Appearance != AppearanceDark {
			return Layout{}, ErrInvalidConfig
		}
		profile, version = BrowserProfileID, BrowserProfileVersion
		window, err := browserWindow(c.Target, b.Window)
		if err != nil {
			return Layout{}, err
		}
		menuH, dockH := 0, 0
		if b.ShowMenuBar {
			menuH = scaled(32, c.Target.Y, browserRefHeight)
		}
		if b.ShowDock {
			dockH = scaled(78, c.Target.Y, browserRefHeight)
		}
		if window.Min.Y < menuH || window.Max.Y > c.Target.Y-dockH {
			return Layout{}, ErrInvalidConfig
		}
		bar := 0
		if b.ShowToolbar {
			bar = scaled(74, c.Target.Y, browserRefHeight)
		}
		viewport = image.Rect(window.Min.X, window.Min.Y+bar, window.Max.X, window.Max.Y)
		if viewport.Dx() < 240 || viewport.Dy() < 160 {
			return Layout{}, ErrInvalidConfig
		}
	}
	fitted, scale := fitContain(bounds, viewport)
	if fitted.Empty() || scale <= 0 {
		return Layout{}, ErrInvalidConfig
	}
	letterbox := rectanglesOutside(viewport, fitted)
	configBytes, err := json.Marshal(struct {
		Profile    string `json:"profile"`
		Version    string `json:"version"`
		Config     Config `json:"config"`
		FillPolicy string `json:"fill_policy"`
		Rasterizer string `json:"rasterizer"`
		FontSHA256 string `json:"font_sha256"`
	}{profile, version, c, FillPolicyID, RasterizerID, fontDigest()})
	if err != nil {
		return Layout{}, err
	}
	digest := sha256.Sum256(configBytes)
	return Layout{ProfileID: profile, ProfileVersion: version, DisplaySize: c.Target, SourceBounds: bounds, AppViewport: viewport, FittedApp: fitted,
		SourceToDisplay: Transform{Scale: scale, TranslateX: float64(fitted.Min.X) - float64(bounds.Min.X)*scale, TranslateY: float64(fitted.Min.Y) - float64(bounds.Min.Y)*scale},
		Letterbox:       letterbox, ResolvedTopFill: top, ResolvedBottomFill: bottom, FillPolicy: FillPolicyID, Rasterizer: RasterizerID, ConfigSHA256: hex.EncodeToString(digest[:]), FontSHA256: fontDigest()}, nil
}

func browserWindow(target image.Point, b BasisRect) (image.Rectangle, error) {
	if b == (BasisRect{}) {
		b = BasisRect{X: 800, Y: 800, Width: 8400, Height: 8000}
	}
	if b.X < 0 || b.Y < 0 || b.X > BrowserRectBasis || b.Y > BrowserRectBasis || b.Width <= 0 || b.Height <= 0 || b.Width > BrowserRectBasis || b.Height > BrowserRectBasis || b.X+b.Width > BrowserRectBasis || b.Y+b.Height > BrowserRectBasis {
		return image.Rectangle{}, ErrInvalidConfig
	}
	min := image.Pt(scaled(b.X, target.X, BrowserRectBasis), scaled(b.Y, target.Y, BrowserRectBasis))
	max := image.Pt(scaled(b.X+b.Width, target.X, BrowserRectBasis), scaled(b.Y+b.Height, target.Y, BrowserRectBasis))
	if max.X-min.X < 320 || max.Y-min.Y < 220 {
		return image.Rectangle{}, ErrInvalidConfig
	}
	return image.Rectangle{Min: min, Max: max}, nil
}

func validText(value string, maxBytes int) bool {
	if len(value) > maxBytes || !utf8.ValidString(value) {
		return false
	}
	return strings.IndexFunc(value, unicode.IsControl) < 0
}

func scaled(value, extent, basis int) int {
	return int(math.Round(float64(value) * float64(extent) / float64(basis)))
}

func fitContain(src, viewport image.Rectangle) (image.Rectangle, float64) {
	scale := math.Min(float64(viewport.Dx())/float64(src.Dx()), float64(viewport.Dy())/float64(src.Dy()))
	w, h := int(math.Round(float64(src.Dx())*scale)), int(math.Round(float64(src.Dy())*scale))
	if w < 1 || h < 1 || w > viewport.Dx() || h > viewport.Dy() {
		return image.Rectangle{}, 0
	}
	x := viewport.Min.X + (viewport.Dx()-w)/2
	y := viewport.Min.Y + (viewport.Dy()-h)/2
	return image.Rect(x, y, x+w, y+h), scale
}

func rectanglesOutside(outer, inner image.Rectangle) []image.Rectangle {
	var out []image.Rectangle
	if inner.Min.Y > outer.Min.Y {
		out = append(out, image.Rect(outer.Min.X, outer.Min.Y, outer.Max.X, inner.Min.Y))
	}
	if inner.Max.Y < outer.Max.Y {
		out = append(out, image.Rect(outer.Min.X, inner.Max.Y, outer.Max.X, outer.Max.Y))
	}
	if inner.Min.X > outer.Min.X {
		out = append(out, image.Rect(outer.Min.X, inner.Min.Y, inner.Min.X, inner.Max.Y))
	}
	if inner.Max.X < outer.Max.X {
		out = append(out, image.Rect(inner.Max.X, inner.Min.Y, outer.Max.X, inner.Max.Y))
	}
	return out
}

func validClock(v string) bool {
	if len(v) != 4 && len(v) != 5 {
		return false
	}
	colon := len(v) - 3
	if v[colon] != ':' || v[colon+1] < '0' || v[colon+1] > '5' || v[colon+2] < '0' || v[colon+2] > '9' {
		return false
	}
	hour := 0
	for _, ch := range v[:colon] {
		if ch < '0' || ch > '9' {
			return false
		}
		hour = hour*10 + int(ch-'0')
	}
	return hour <= 23
}

func validateAlphaAndMatte(ctx context.Context, img image.Image, matte *RGB) error {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a != 0xffff && matte == nil {
				return errors.New("presentations: transparent app requires an explicit matte")
			}
		}
	}
	return nil
}

func resolveFills(ctx context.Context, app image.Image, c Config) (RGB, RGB, error) {
	top, bottom, err := edgeModeFills(ctx, app, c.Matte)
	if err != nil {
		return RGB{}, RGB{}, err
	}
	if c.Phone != nil {
		if c.Phone.TopFill != nil {
			top = *c.Phone.TopFill
		} else if c.Phone.Appearance == AppearanceLight {
			top = RGB{R: 246, G: 247, B: 250}
		}
		if c.Phone.BottomFill != nil {
			bottom = *c.Phone.BottomFill
		} else if c.Phone.Appearance == AppearanceLight {
			bottom = RGB{R: 246, G: 247, B: 250}
		}
	}
	if c.Browser != nil {
		if c.Browser.DesktopColor != nil {
			top = *c.Browser.DesktopColor
		}
		bottom = top
	}
	return top, bottom, nil
}

func edgeModeFills(ctx context.Context, app image.Image, matte *RGB) (RGB, RGB, error) {
	b := app.Bounds()
	top := sampleEdge(ctx, app, b.Min.Y, b.Min.X, b.Dx(), b.Dy(), matte)
	if err := ctx.Err(); err != nil {
		return RGB{}, RGB{}, err
	}
	bottom := sampleEdge(ctx, app, b.Max.Y-1, b.Min.X, b.Dx(), b.Dy(), matte)
	if err := ctx.Err(); err != nil {
		return RGB{}, RGB{}, err
	}
	if len(top) == 0 || len(bottom) == 0 {
		return RGB{}, RGB{}, errors.New("presentations: no usable opaque edge samples")
	}
	return modal(top), modal(bottom), nil
}

func sampleEdge(ctx context.Context, img image.Image, y, x0, width, height int, matte *RGB) []RGB {
	strip := height / 32
	if strip < 1 {
		strip = 1
	}
	if strip > 16 {
		strip = 16
	}
	stride := (width + 511) / 512
	if stride < 1 {
		stride = 1
	}
	out := make([]RGB, 0, 512)
	for dy := 0; dy < strip; dy++ {
		if ctx.Err() != nil {
			return nil
		}
		yy := y + dy
		if y == img.Bounds().Max.Y-1 {
			yy = y - dy
		}
		for x := x0; x < x0+width; x += stride {
			r, g, b, a := img.At(x, yy).RGBA()
			if a != 0xffff {
				if matte == nil {
					continue
				}
				mr, mg, mb := uint64(matte.R)*257, uint64(matte.G)*257, uint64(matte.B)*257
				r = uint32(uint64(r) + mr*uint64(0xffff-a)/0xffff)
				g = uint32(uint64(g) + mg*uint64(0xffff-a)/0xffff)
				b = uint32(uint64(b) + mb*uint64(0xffff-a)/0xffff)
			}
			out = append(out, RGB{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)})
		}
	}
	return out
}

func modal(colors []RGB) RGB {
	counts := make(map[uint32]int, len(colors))
	sums := make(map[uint32][3]int, len(colors))
	for _, c := range colors {
		k := uint32(c.R>>3)<<10 | uint32(c.G>>3)<<5 | uint32(c.B>>3)
		counts[k]++
		s := sums[k]
		s[0] += int(c.R)
		s[1] += int(c.G)
		s[2] += int(c.B)
		sums[k] = s
	}
	var best uint32
	bestCount := -1
	for k, n := range counts {
		if n > bestCount || (n == bestCount && k < best) {
			best, bestCount = k, n
		}
	}
	s := sums[best]
	return RGB{uint8(s[0] / bestCount), uint8(s[1] / bestCount), uint8(s[2] / bestCount)}
}
