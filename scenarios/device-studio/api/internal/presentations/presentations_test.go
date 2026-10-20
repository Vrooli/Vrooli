package presentations

import (
	"context"
	"errors"
	"image"
	"image/color"
	"sync/atomic"
	"testing"
)

func phoneConfig(w, h int) Config {
	return Config{Target: image.Pt(w, h), Phone: &PhoneConfig{ShowStatus: true, ShowHome: true, Appearance: AppearanceDark, Cellular: true, WiFi: true}}
}

func TestPhoneRasterKeepsAppOutOfStatusAndHomeAndPreservesAspect(t *testing.T) {
	src := image.NewNRGBA(image.Rect(7, 11, 11, 19))
	for y := 11; y < 19; y++ {
		for x := 7; x < 11; x++ {
			src.SetNRGBA(x, y, color.NRGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}
	for x := 7; x < 11; x++ {
		src.SetNRGBA(x, 11, color.NRGBA{R: 255, A: 255})
		src.SetNRGBA(x, 18, color.NRGBA{B: 255, A: 255})
	}
	result, err := Compose(context.Background(), src, phoneConfig(430, 880))
	if err != nil {
		t.Fatal(err)
	}
	l := result.Layout
	if l.AppViewport.Min.Y <= 0 || l.AppViewport.Max.Y >= 880 || l.FittedApp.Dx()*src.Bounds().Dy() != l.FittedApp.Dy()*src.Bounds().Dx() {
		t.Fatalf("invalid viewport/aspect: %#v", l)
	}
	if got := result.Raster.NRGBAAt(200, 10); got.R == 0 && got.G == 255 {
		t.Fatalf("app green pixel leaked into status region: %#v", got)
	}
	if got := result.Raster.NRGBAAt(200, 879); got.R == 0 && got.G == 255 {
		t.Fatalf("app green pixel leaked into home region: %#v", got)
	}
	if got := result.Raster.NRGBAAt(l.FittedApp.Min.X, l.FittedApp.Min.Y); got.R != 255 || got.G != 0 {
		t.Fatalf("source bounds minimum was not mapped: %#v", got)
	}
	if len(l.Letterbox) != 2 {
		t.Fatalf("expected deterministic side letterboxes for unequal app/display aspect: %v", l.Letterbox)
	}
}

func TestContainMappingUsesPixelEdgesAndExplicitLetterboxes(t *testing.T) {
	src := image.NewNRGBA(image.Rect(13, 17, 17, 19))
	for y := 17; y < 19; y++ {
		for x := 13; x < 17; x++ {
			src.SetNRGBA(x, y, color.NRGBA{R: uint8((x - 13) * 50), G: uint8((y - 17) * 100), A: 255})
		}
	}
	c := phoneConfig(430, 880)
	l, err := BuildLayout(src, c)
	if err != nil {
		t.Fatal(err)
	}
	if l.SourceBounds.Min != image.Pt(13, 17) || l.FittedApp.Dy() > l.AppViewport.Dy() || l.FittedApp.Dx() > l.AppViewport.Dx() {
		t.Fatalf("bad edge mapping/layout: %#v", l)
	}
	for _, r := range l.Letterbox {
		if r.Empty() || r.Min.X < l.AppViewport.Min.X || r.Min.Y < l.AppViewport.Min.Y || r.Max.X > l.AppViewport.Max.X || r.Max.Y > l.AppViewport.Max.Y {
			t.Fatalf("letterbox outside viewport: %v", r)
		}
	}
	if l.SourceToDisplay.Scale <= 0 || l.SourceToDisplay.TranslateX != float64(l.FittedApp.Min.X)-13*l.SourceToDisplay.Scale || l.SourceToDisplay.TranslateY != float64(l.FittedApp.Min.Y)-17*l.SourceToDisplay.Scale {
		t.Fatalf("transform does not use source pixel-edge origin: %#v", l.SourceToDisplay)
	}
}

func TestBrowserWindowAndLayoutRejectInvalidOrUnsafeGeometry(t *testing.T) {
	src := opaqueTestImage(image.Rect(0, 0, 10, 10))
	c := Config{Target: image.Pt(1440, 900), Browser: &BrowserConfig{ShowToolbar: true, Window: BasisRect{X: 1000, Y: 1000, Width: 8000, Height: 8000}}}
	l, err := BuildLayout(src, c)
	if err != nil {
		t.Fatal(err)
	}
	if l.ProfileID != BrowserProfileID || l.AppViewport.Empty() || l.AppViewport.Min.Y <= scaled(c.Browser.Window.Y, 900, BrowserRectBasis) {
		t.Fatalf("browser toolbar not excluded from app viewport: %#v", l)
	}
	c.Browser.Window = BasisRect{X: 9000, Y: 0, Width: 2000, Height: 10000}
	if _, err = BuildLayout(src, c); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("out-of-bounds window accepted: %v", err)
	}
	c = phoneConfig(430, 880)
	c.Browser = &BrowserConfig{}
	if _, err = BuildLayout(src, c); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("ambiguous profile accepted: %v", err)
	}
	if _, err = BuildLayout(src, phoneConfig(80, 100)); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("unusable phone raster accepted: %v", err)
	}
}

func TestEdgeFillTieBreakTransparencyAndExplicitMatte(t *testing.T) {
	src := image.NewNRGBA(image.Rect(4, 9, 6, 10))
	src.SetNRGBA(4, 9, color.NRGBA{R: 255, A: 255})
	src.SetNRGBA(5, 9, color.NRGBA{B: 255, A: 255})
	top, _, err := edgeModeFills(context.Background(), src, nil)
	if err != nil {
		t.Fatal(err)
	}
	if top != (RGB{B: 255}) {
		t.Fatalf("modal tie did not use deterministic low-bin tie break: %#v", top)
	}
	transparent := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	if _, err := BuildLayout(transparent, phoneConfig(430, 880)); err == nil {
		t.Fatal("transparent input without matte accepted")
	}
	c := phoneConfig(430, 880)
	c.Matte = &RGB{R: 20, G: 30, B: 40}
	if _, err := BuildLayout(transparent, c); err != nil {
		t.Fatalf("explicit matte rejected: %v", err)
	}
}

func TestConfigAndRasterAreDeterministicAndTextUsesPinnedFont(t *testing.T) {
	src := opaqueTestImage(image.Rect(0, 0, 20, 30))
	c := phoneConfig(430, 880)
	a, err := Compose(context.Background(), src, c)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Compose(context.Background(), src, c)
	if err != nil {
		t.Fatal(err)
	}
	if a.Layout.ConfigSHA256 != b.Layout.ConfigSHA256 || !equalRaster(a.Raster, b.Raster) {
		t.Fatal("same config and app raster produced different bytes")
	}
	c.Phone.Clock = "09:41"
	c.Phone.ShowBatteryPercent = true
	c.Phone.BatteryPercent = 84
	withText, err := Compose(context.Background(), src, c)
	if err != nil {
		t.Fatalf("pinned font text failed: %v", err)
	}
	if withText.Layout.FontSHA256 == "" || equalRaster(a.Raster, withText.Raster) {
		t.Fatal("text or pinned font identity was omitted")
	}
	c.Phone.Carrier = "界"
	if _, err = Compose(context.Background(), src, c); !errors.Is(err, ErrUnsupportedGlyph) {
		t.Fatalf("unsupported glyph did not fail closed: %v", err)
	}
	c = phoneConfig(430, 880)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = Compose(ctx, src, c); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation was ignored: %v", err)
	}
}

func TestBatteryFillReflectsConfiguredPercentAndAppearanceInk(t *testing.T) {
	zero := image.NewNRGBA(image.Rect(0, 0, 40, 20))
	full := image.NewNRGBA(image.Rect(0, 0, 40, 20))
	inkDark := statusInk(AppearanceDark)
	inkLight := statusInk(AppearanceLight)
	drawBattery(zero, 5, 10, inkDark, 1, 0)
	drawBattery(full, 5, 10, inkDark, 1, 100)
	if zero.NRGBAAt(10, 10) == full.NRGBAAt(10, 10) {
		t.Fatal("battery percentage did not affect charge fill")
	}
	if inkDark == inkLight || inkLight.R >= inkDark.R {
		t.Fatalf("status appearance did not select contrast-appropriate ink: dark=%#v light=%#v", inkDark, inkLight)
	}
}

func TestPremultipliedBilinearSamplingAvoidsTransparentColorFringe(t *testing.T) {
	src := image.NewNRGBA(image.Rect(7, 11, 9, 12))
	src.SetNRGBA(7, 11, color.NRGBA{R: 255, A: 0})
	src.SetNRGBA(8, 11, color.NRGBA{B: 255, A: 255})
	got := sampleBilinearOver(src, src.Bounds(), 7.5, 11, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	want := color.NRGBA{R: 128, G: 128, B: 255, A: 255}
	if got != want {
		t.Fatalf("transparent source RGB leaked into bilinear edge: got %#v want %#v", got, want)
	}
}

func TestBrowserChromeScalesWithTheTargetRaster(t *testing.T) {
	src := opaqueTestImage(image.Rect(0, 0, 120, 80))
	config := func(w, h int) Config {
		return Config{Target: image.Pt(w, h), Browser: &BrowserConfig{
			Window:      BasisRect{X: 800, Y: 500, Width: 8400, Height: 8500},
			ShowToolbar: true, ShowMenuBar: true, ShowDock: true,
		}}
	}
	small, err := Compose(context.Background(), src, config(1440, 900))
	if err != nil {
		t.Fatal(err)
	}
	large, err := Compose(context.Background(), src, config(2560, 1600))
	if err != nil {
		t.Fatal(err)
	}
	smallToolbar := scaled(74, 900, browserRefHeight)
	largeToolbar := scaled(74, 1600, browserRefHeight)
	if smallToolbar != 74 || largeToolbar <= smallToolbar {
		t.Fatalf("toolbar dimensions do not scale with output: %d -> %d", smallToolbar, largeToolbar)
	}
	if small.Layout.AppViewport.Dx() == large.Layout.AppViewport.Dx() || large.Raster.Bounds().Size() != image.Pt(2560, 1600) {
		t.Fatalf("target geometry was not preserved across raster sizes: small=%v large=%v", small.Layout, large.Layout)
	}
	for _, result := range []Result{small, large} {
		for i := 3; i < len(result.Raster.Pix); i += 4 {
			if result.Raster.Pix[i] != 255 {
				t.Fatal("browser presentation exposed a nonopaque output pixel")
			}
		}
	}
}

func TestComposeStopsWhenContextCancelsDuringSourceSampling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	base := opaqueTestImage(image.Rect(0, 0, 32, 32))
	var reads atomic.Int64
	src := cancelingImage{Image: base, reads: &reads, cancelAt: 1090, cancel: cancel}
	_, err := Compose(ctx, src, phoneConfig(430, 880))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation during composition, got %v", err)
	}
	if got := reads.Load(); got <= 1088 {
		t.Fatalf("fixture did not reach raster sampling before cancellation: %d reads", got)
	}
}

func TestClockAcceptsTwelveHourStyleAndRejectsInvalidTimes(t *testing.T) {
	for _, v := range []string{"9:41", "09:41", "23:59", "0:00"} {
		if !validClock(v) {
			t.Errorf("valid clock rejected: %q", v)
		}
	}
	for _, v := range []string{"24:00", "9:60", "009:41", "9:4", "9-41"} {
		if validClock(v) {
			t.Errorf("invalid clock accepted: %q", v)
		}
	}
}

func TestBrowserColorsAllowExplicitBlackAndAlphaChromeIsComposited(t *testing.T) {
	src := opaqueTestImage(image.Rect(0, 0, 20, 10))
	black := RGB{}
	c := Config{Target: image.Pt(1440, 900), Browser: &BrowserConfig{
		Window:       BasisRect{X: 1000, Y: 800, Width: 8000, Height: 8000},
		DesktopColor: &black, ShowMenuBar: true, ShowDock: true,
	}}
	result, err := Compose(context.Background(), src, c)
	if err != nil {
		t.Fatal(err)
	}
	baseConfig := c
	baseConfig.Browser = &BrowserConfig{Window: c.Browser.Window, DesktopColor: &black}
	base, err := Compose(context.Background(), src, baseConfig)
	if err != nil {
		t.Fatal(err)
	}
	if got := base.Raster.NRGBAAt(2, 100); got.R != 1 || got.G != 1 || got.B != 0 || got.A != 255 {
		t.Fatalf("explicit black desktop color was treated as default: %#v", got)
	}
	got := result.Raster.NRGBAAt(10, 10)
	want := over(color.NRGBA{R: 28, G: 35, B: 48, A: 245}, base.Raster.NRGBAAt(10, 10))
	if got != want {
		t.Fatalf("menu alpha was not composited over desktop: got %#v want %#v", got, want)
	}
}

type cancelingImage struct {
	image.Image
	reads    *atomic.Int64
	cancelAt int64
	cancel   context.CancelFunc
}

func (i cancelingImage) At(x, y int) color.Color {
	if i.reads.Add(1) >= i.cancelAt {
		i.cancel()
	}
	return i.Image.At(x, y)
}

func equalRaster(a, b *image.NRGBA) bool {
	if a.Bounds() != b.Bounds() || len(a.Pix) != len(b.Pix) {
		return false
	}
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			return false
		}
	}
	return true
}

func opaqueTestImage(bounds image.Rectangle) *image.NRGBA {
	img := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8((x - bounds.Min.X) * 5), G: uint8((y - bounds.Min.Y) * 3), B: 80, A: 255})
		}
	}
	return img
}
