// Package presentations builds deterministic, source-preserving device display
// rasters. It owns simulated presentation layers, not photographic projection.
package presentations

import (
	"context"
	"errors"
	"image"
	"image/color"
)

const (
	PhoneProfileID        = "generic-tall-phone-system"
	PhoneProfileVersion   = "2"
	BrowserProfileID      = "generic-browser-desktop"
	BrowserProfileVersion = "2"
	FillPolicyID          = "fixed-edge-mode-5bit-v1"
	RasterizerID          = "device-studio-bilinear-gofont-v1"
	BrowserRectBasis      = 10000
	maxRasterSide         = 8192
	maxRasterPixels       = 32_000_000
	maxSourcePixels       = 24_000_000
)

var (
	ErrInvalidConfig    = errors.New("presentations: invalid configuration")
	ErrUnsupportedGlyph = errors.New("presentations: configured text contains a glyph unsupported by the pinned font")
)

type Appearance string

const (
	AppearanceLight Appearance = "light"
	AppearanceDark  Appearance = "dark"
)

type RGB struct{ R, G, B uint8 }

// Config selects exactly one simulated presentation kind and the target
// display raster supplied by the compiled scene. Layout geometry remains
// profile-owned; Browser.Window is the only normalized placement control.
type Config struct {
	Target  image.Point
	Matte   *RGB
	Phone   *PhoneConfig
	Browser *BrowserConfig
}

type PhoneConfig struct {
	ShowStatus         bool
	ShowHome           bool
	Clock              string
	Carrier            string
	BatteryPercent     uint8
	ShowBatteryPercent bool
	Cellular           bool
	WiFi               bool
	Appearance         Appearance
	TopFill            *RGB
	BottomFill         *RGB
}

type BasisRect struct{ X, Y, Width, Height int }
type BrowserConfig struct {
	Window             BasisRect
	Title              string
	Address            string
	Appearance         Appearance
	ShowToolbar        bool
	ShowWindowControls bool
	ShowMenuBar        bool
	ShowDock           bool
	DesktopColor       *RGB
	WindowColor        *RGB
	ChromeColor        *RGB
}

type Transform struct {
	Scale      float64
	TranslateX float64
	TranslateY float64
}

type Layout struct {
	ProfileID          string
	ProfileVersion     string
	DisplaySize        image.Point
	SourceBounds       image.Rectangle
	AppViewport        image.Rectangle
	FittedApp          image.Rectangle
	SourceToDisplay    Transform
	Letterbox          []image.Rectangle
	ResolvedTopFill    RGB
	ResolvedBottomFill RGB
	FillPolicy         string
	Rasterizer         string
	ConfigSHA256       string
	FontSHA256         string
}

type Result struct {
	Raster *image.NRGBA
	Layout Layout
}

// Compose produces a display-space raster. Screen pixels are contained into
// the profile-owned viewport using one uniform scale and source pixel-edge
// coordinates. Context cancellation is checked before work and per output row.
func Compose(ctx context.Context, app image.Image, config Config) (Result, error) {
	if ctx == nil {
		return Result{}, ErrInvalidConfig
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	layout, err := buildLayout(ctx, app, config)
	if err != nil {
		return Result{}, err
	}
	if err = validateGlyphs(config); err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	return rasterize(ctx, app, config, layout)
}

// BuildLayout validates the profile and computes all source-to-display mapping
// without allocating the target raster.
func BuildLayout(app image.Image, config Config) (Layout, error) {
	return buildLayout(context.Background(), app, config)
}

func opaqueMatte(c *RGB) color.NRGBA {
	if c == nil {
		return color.NRGBA{R: 24, G: 28, B: 38, A: 255}
	}
	return color.NRGBA{R: c.R, G: c.G, B: c.B, A: 255}
}
