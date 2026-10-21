// Package presentations builds deterministic, source-preserving device display
// rasters. It owns simulated presentation layers, not photographic projection.
package presentations

import (
	"context"
	"errors"
	"image"
	"image/color"
	"reflect"
)

const (
	PhoneProfileID             = "generic-tall-phone-system"
	PhoneProfileVersion        = "2"
	PhoneProfileVersionAuto    = "3"
	BrowserProfileID           = "generic-browser-desktop"
	BrowserProfileVersion      = "2"
	SimulatedCompositorProfile = "go-still-presentation-v2"
	FillPolicyID               = "fixed-edge-mode-5bit-v1"
	RasterizerID               = "device-studio-bilinear-gofont-v1"
	BrowserRectBasis           = 10000
	maxRasterSide              = 8192
	maxRasterPixels            = 32_000_000
	maxSourcePixels            = 24_000_000
)

var (
	ErrInvalidConfig      = errors.New("presentations: invalid configuration")
	ErrUnsupportedGlyph   = errors.New("presentations: configured text contains a glyph unsupported by the pinned font")
	ErrMatteRequired      = errors.New("presentations: transparent app requires an explicit matte")
	ErrCarrierDoesNotFit  = errors.New("presentations: carrier text does not fit the phone profile's safe status area")
	ErrVideoFillsRequired = errors.New("presentations: video layout requires explicit source-independent fill colors")
)

type Appearance string

const (
	AppearanceLight Appearance = "light"
	AppearanceDark  Appearance = "dark"
	AppearanceAuto  Appearance = "auto"
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
	ProfileVersion     string `json:"profile_version,omitempty"`
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
	StatusInk          *RGB `json:"status_ink,omitempty"`
	HomeInk            *RGB `json:"home_ink,omitempty"`
}

type (
	BasisRect     struct{ X, Y, Width, Height int }
	BrowserConfig struct {
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
)

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
	ResolvedStatusInk  *RGB `json:",omitempty"`
	ResolvedHomeInk    *RGB `json:",omitempty"`
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

// ComposeActiveViewport produces the direct screen-content raster used at the
// camera handoff. It keeps the existing fitted-app transform and letterbox
// base, while excluding simulated device or browser chrome from the viewport.
func ComposeActiveViewport(ctx context.Context, app image.Image, config Config, expected Layout) (*image.NRGBA, error) {
	if ctx == nil || app == nil {
		return nil, ErrInvalidConfig
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	layout, err := buildLayout(ctx, app, config)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(layout, expected) {
		return nil, errors.New("active viewport layout differs from the composed presentation")
	}
	if err := validateGlyphs(config); err != nil {
		return nil, err
	}
	viewport := layout.AppViewport
	local := layout
	local.AppViewport = image.Rect(0, 0, viewport.Dx(), viewport.Dy())
	local.FittedApp = layout.FittedApp.Sub(viewport.Min)
	local.SourceToDisplay.TranslateX -= float64(viewport.Min.X)
	local.SourceToDisplay.TranslateY -= float64(viewport.Min.Y)
	out := image.NewNRGBA(local.AppViewport)
	base := opaqueMatte(config.Matte)
	if config.Browser != nil {
		base = opaque(browserWindowFill(config.Browser))
	}
	if err := fill(ctx, out, out.Bounds(), base); err != nil {
		return nil, err
	}
	if err := compositeContain(ctx, out, app, local, opaqueMatte(config.Matte)); err != nil {
		return nil, err
	}
	return out, nil
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
