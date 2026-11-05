package presentations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
	"sync/atomic"
	"testing"
)

func phoneConfig(w, h int) Config {
	return Config{Target: image.Pt(w, h), Phone: &PhoneConfig{ShowStatus: true, ShowHome: true, Appearance: AppearanceDark, Cellular: true, WiFi: true}}
}

func TestBuildVideoViewportIgnoresUnresolvedAndExplicitFills(t *testing.T) {
	auto := Config{Target: image.Pt(430, 880), Phone: &PhoneConfig{ProfileVersion: PhoneProfileVersionAuto, ShowStatus: true, ShowHome: true, Appearance: AppearanceAuto}}
	before := *auto.Phone
	viewport, err := BuildVideoViewport(image.Rect(0, 0, 430, 880), auto)
	if err != nil {
		t.Fatal(err)
	}
	if auto.Phone.TopFill != nil || auto.Phone.BottomFill != nil || auto.Phone.Appearance != before.Appearance {
		t.Fatal("geometry-only derivation mutated the unresolved caller config")
	}
	explicit := auto
	phone := *auto.Phone
	phone.TopFill, phone.BottomFill = &RGB{R: 255}, &RGB{B: 255}
	explicit.Phone = &phone
	other, err := BuildVideoViewport(image.Rect(0, 0, 430, 880), explicit)
	if err != nil {
		t.Fatal(err)
	}
	if viewport != other {
		t.Fatalf("source-independent fills changed viewport: %v vs %v", viewport, other)
	}
}

func TestPhoneRasterKeepsAppOutOfStatusAndHomeAndPreservesAspect(t *testing.T) { // [REQ:DS-SCREEN-CHROME]
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

func TestContainMappingUsesPixelEdgesAndExplicitLetterboxes(t *testing.T) { // [REQ:DS-SCREEN-FIT]
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
	left, right := l.FittedApp.Min.X-l.AppViewport.Min.X, l.AppViewport.Max.X-l.FittedApp.Max.X
	top, bottom := l.FittedApp.Min.Y-l.AppViewport.Min.Y, l.AppViewport.Max.Y-l.FittedApp.Max.Y
	if abs(left-right) > 1 || abs(top-bottom) > 1 {
		t.Fatalf("contain fit is not centered within one raster pixel: horizontal=%d/%d vertical=%d/%d", left, right, top, bottom)
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

func TestPlainAppOnlyUsesUniformFullTargetContainAndMatte(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 8, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 8; x++ {
			src.SetNRGBA(x, y, color.NRGBA{R: 240, G: 30, B: 20, A: 255})
		}
	}
	matte := &RGB{R: 4, G: 9, B: 14}
	config := Config{Target: image.Pt(4, 8), Matte: matte}
	layout, err := BuildLayout(src, config)
	if err != nil {
		t.Fatal(err)
	}
	if layout.ProfileID != PlainProfileID || layout.ProfileVersion != PlainProfileVersion || layout.AppViewport != image.Rect(0, 0, 4, 8) || layout.FittedApp != image.Rect(0, 3, 4, 5) {
		t.Fatalf("plain profile did not contain the app across the full target: %+v", layout)
	}
	if layout.SourceToDisplay.Scale != 0.5 || layout.Letterbox[0] != image.Rect(0, 0, 4, 3) {
		t.Fatalf("plain fit stretched pixels or lost its matte geometry: %+v", layout)
	}
	result, err := Compose(context.Background(), src, config)
	if err != nil {
		t.Fatal(err)
	}
	for _, point := range []image.Point{image.Pt(0, 0), image.Pt(3, 7)} {
		if got := result.Raster.NRGBAAt(point.X, point.Y); got != (color.NRGBA{R: 4, G: 9, B: 14, A: 255}) {
			t.Fatalf("letterbox pixel %v = %#v, want the explicit matte", point, got)
		}
	}
	if got := result.Raster.NRGBAAt(1, 3); got != (color.NRGBA{R: 240, G: 30, B: 20, A: 255}) {
		t.Fatalf("contained source pixel = %#v, want source color without simulated chrome", got)
	}
	viewport, err := BuildVideoViewport(image.Rect(0, 0, 8, 4), config)
	if err != nil || viewport != image.Rect(0, 0, 4, 8) {
		t.Fatalf("plain recording viewport = %v, %v", viewport, err)
	}
}

func TestBrowserWindowAndLayoutRejectInvalidOrUnsafeGeometry(t *testing.T) { // [REQ:DS-SCREEN-CHROME]
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
	for _, size := range []image.Point{image.Pt(1000, 400), image.Pt(300, 1000)} {
		if _, err := BuildLayout(src, phoneConfig(size.X, size.Y)); !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("unsupported phone aspect %v accepted: %v", size, err)
		}
	}
}

func TestBrowserControlsCannotBeSavedWithoutToolbar(t *testing.T) {
	c := Config{Target: image.Pt(1440, 900), Browser: &BrowserConfig{ShowWindowControls: true, ShowToolbar: false}}
	source := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	source.SetNRGBA(0, 0, color.NRGBA{A: 255})
	if err := ValidateConfig(source, c); err == nil {
		t.Fatal("window controls without a toolbar were accepted")
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
	if _, err := BuildLayout(transparent, phoneConfig(430, 880)); !errors.Is(err, ErrMatteRequired) {
		t.Fatalf("transparent input without matte should expose ErrMatteRequired, got %v", err)
	}
	c := phoneConfig(430, 880)
	c.Matte = &RGB{R: 20, G: 30, B: 40}
	if _, err := BuildLayout(transparent, c); err != nil {
		t.Fatalf("explicit matte rejected: %v", err)
	}
}

func TestPhoneAppearanceDoesNotChangeSampledDefaultFills(t *testing.T) {
	src := image.NewNRGBA(image.Rect(5, 8, 13, 20))
	for y := src.Bounds().Min.Y; y < src.Bounds().Max.Y; y++ {
		for x := src.Bounds().Min.X; x < src.Bounds().Max.X; x++ {
			src.SetNRGBA(x, y, color.NRGBA{R: uint8(40 + x), G: uint8(70 + y), B: 120, A: 255})
		}
	}
	dark := phoneConfig(430, 880)
	light := dark
	lightPhone := *dark.Phone
	lightPhone.Appearance = AppearanceLight
	light.Phone = &lightPhone
	darkLayout, err := BuildLayout(src, dark)
	if err != nil {
		t.Fatal(err)
	}
	lightLayout, err := BuildLayout(src, light)
	if err != nil {
		t.Fatal(err)
	}
	if darkLayout.ResolvedTopFill != lightLayout.ResolvedTopFill || darkLayout.ResolvedBottomFill != lightLayout.ResolvedBottomFill {
		t.Fatalf("appearance changed default edge fills: dark=%#v/%#v light=%#v/%#v", darkLayout.ResolvedTopFill, darkLayout.ResolvedBottomFill, lightLayout.ResolvedTopFill, lightLayout.ResolvedBottomFill)
	}
	override := RGB{R: 246, G: 247, B: 250}
	light.Phone.TopFill = &override
	light.Phone.BottomFill = &override
	overridden, err := BuildLayout(src, light)
	if err != nil {
		t.Fatal(err)
	}
	if overridden.ResolvedTopFill != override || overridden.ResolvedBottomFill != override {
		t.Fatalf("explicit fill overrides were ignored: %#v/%#v", overridden.ResolvedTopFill, overridden.ResolvedBottomFill)
	}
}

func TestPhoneV3AutoResolvesStatusAndHomeInkIndependently(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 8, 64))
	for y := 0; y < 64; y++ {
		fill := color.NRGBA{R: 45, G: 50, B: 62, A: 255}
		if y >= 62 {
			fill = color.NRGBA{R: 238, G: 240, B: 244, A: 255}
		}
		for x := 0; x < 8; x++ {
			src.SetNRGBA(x, y, fill)
		}
	}
	c := Config{Target: image.Pt(430, 880), Phone: &PhoneConfig{
		ProfileVersion: PhoneProfileVersionAuto, Appearance: AppearanceAuto,
		ShowStatus: true, ShowHome: true, Clock: "9:41", Cellular: true, WiFi: true,
	}}
	layout, err := BuildLayout(src, c)
	if err != nil {
		t.Fatal(err)
	}
	darkFillLightInk := RGB{R: 236, G: 240, B: 247}
	lightFillDarkInk := RGB{R: 30, G: 34, B: 42}
	if layout.ProfileVersion != PhoneProfileVersionAuto || layout.ResolvedStatusInk == nil || *layout.ResolvedStatusInk != darkFillLightInk || layout.ResolvedHomeInk == nil || *layout.ResolvedHomeInk != lightFillDarkInk {
		t.Fatalf("AUTO did not independently resolve contrast against sampled fills: %+v", layout)
	}
	result, err := Compose(context.Background(), src, c)
	if err != nil {
		t.Fatal(err)
	}
	if countColor(result.Raster, image.Rect(0, 0, 430, 74), colorFromRGB(darkFillLightInk)) == 0 || countColor(result.Raster, image.Rect(0, 846, 430, 880), colorFromRGB(lightFillDarkInk)) == 0 {
		t.Fatal("AUTO selected ink was not drawn into both the status and home regions")
	}
}

func TestPhoneV3CanPinIndependentSampleResolvedInks(t *testing.T) {
	app := image.NewNRGBA(image.Rect(0, 0, 430, 880))
	for i := range app.Pix {
		app.Pix[i] = 255
	}
	status := RGB{R: 245, G: 245, B: 245}
	home := RGB{R: 24, G: 28, B: 38}
	cfg := Config{Target: image.Pt(430, 880), Phone: &PhoneConfig{ProfileVersion: PhoneProfileVersionAuto, Appearance: AppearanceAuto, Clock: "9:41", ShowStatus: true, ShowHome: true, StatusInk: &status, HomeInk: &home}}
	result, err := Compose(context.Background(), app, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.Layout.ResolvedStatusInk == nil || *result.Layout.ResolvedStatusInk != status || result.Layout.ResolvedHomeInk == nil || *result.Layout.ResolvedHomeInk != home {
		t.Fatalf("pinned independent inks were not preserved: %#v", result.Layout)
	}
	statusHits := countColor(result.Raster, image.Rect(0, 0, 430, 74), colorFromRGB(status))
	homeHits := countColor(result.Raster, image.Rect(0, 846, 430, 880), colorFromRGB(home))
	if statusHits == 0 || homeHits == 0 {
		t.Fatalf("pinned status/home ink not rasterized: %d/%d", statusHits, homeHits)
	}
}

func TestBuildLayoutForVideoRasterUsesVerifiedBoundsAndRequiresExplicitFills(t *testing.T) {
	top := RGB{R: 12, G: 34, B: 56}
	bottom := RGB{R: 78, G: 90, B: 123}
	config := Config{
		Target: image.Pt(430, 880),
		Phone: &PhoneConfig{
			ProfileVersion: PhoneProfileVersionAuto,
			Appearance:     AppearanceAuto,
			TopFill:        &top,
			BottomFill:     &bottom,
		},
	}
	bounds := image.Rect(13, -7, 443, 873)
	layout, err := BuildLayoutForVideoRaster(bounds, config)
	if err != nil {
		t.Fatal(err)
	}
	if layout.SourceBounds != bounds || layout.ResolvedTopFill != top || layout.ResolvedBottomFill != bottom || layout.FittedApp.Empty() {
		t.Fatalf("video layout lost verified bounds or explicit fills: %#v", layout)
	}
	config.Phone.TopFill = nil
	if _, err := BuildLayoutForVideoRaster(bounds, config); !errors.Is(err, ErrVideoFillsRequired) {
		t.Fatalf("missing explicit source-independent fill error = %v", err)
	}
}

func TestComposeActiveViewportDropsBrowserBorderButKeepsFittedSourceEdges(t *testing.T) {
	app := image.NewNRGBA(image.Rect(0, 0, 320, 180))
	for y := 0; y < app.Bounds().Dy(); y++ {
		for x := 0; x < app.Bounds().Dx(); x++ {
			c := color.NRGBA{R: 20, G: 40, B: 70, A: 255}
			if x < 10 {
				c = color.NRGBA{R: 250, G: 245, B: 235, A: 255}
			} else if x >= 310 {
				c = color.NRGBA{R: 235, G: 24, B: 30, A: 255}
			}
			app.SetNRGBA(x, y, c)
		}
	}
	windowFill := RGB{R: 23, G: 28, B: 38}
	config := Config{Target: image.Pt(1440, 900), Browser: &BrowserConfig{
		Window: BasisRect{X: 800, Y: 800, Width: 8400, Height: 8000},
		Title:  "Browser", Address: "example.test", Appearance: AppearanceDark,
		ShowToolbar: true, ShowWindowControls: true, ShowMenuBar: true, ShowDock: true,
		DesktopColor: &RGB{R: 120, G: 20, B: 30}, WindowColor: &windowFill,
	}}
	full, err := Compose(context.Background(), app, config)
	if err != nil {
		t.Fatal(err)
	}
	clean, err := ComposeActiveViewport(context.Background(), app, config, full.Layout)
	if err != nil {
		t.Fatal(err)
	}
	if clean.Bounds() != image.Rect(0, 0, full.Layout.AppViewport.Dx(), full.Layout.AppViewport.Dy()) {
		t.Fatalf("clean viewport bounds = %v, want zero-origin viewport size", clean.Bounds())
	}
	border := color.NRGBA{R: 132, G: 145, B: 164, A: 255}
	fullBorderPixels, cleanBorderPixels := 0, 0
	for y := 0; y < clean.Bounds().Dy(); y++ {
		for x := 0; x < clean.Bounds().Dx(); x++ {
			if got := full.Raster.NRGBAAt(full.Layout.AppViewport.Min.X+x, full.Layout.AppViewport.Min.Y+y); got == border {
				fullBorderPixels++
			}
			if clean.NRGBAAt(x, y) == border {
				cleanBorderPixels++
			}
		}
	}
	if fullBorderPixels == 0 || cleanBorderPixels != 0 {
		t.Fatalf("browser stroke pixels in active viewport: full=%d clean=%d, want full > 0 and clean 0", fullBorderPixels, cleanBorderPixels)
	}
	letterboxPoint := image.Pt(0, clean.Bounds().Dy()/2)
	if got := clean.NRGBAAt(letterboxPoint.X, letterboxPoint.Y); got != (color.NRGBA{R: windowFill.R, G: windowFill.G, B: windowFill.B, A: 255}) {
		t.Fatalf("clean letterbox pixel = %#v, want flat window fill", got)
	}
	leftEdgeX := full.Layout.FittedApp.Min.X - full.Layout.AppViewport.Min.X + 10
	rightEdgeX := full.Layout.FittedApp.Max.X - full.Layout.AppViewport.Min.X - 10
	edgeY := clean.Bounds().Dy() / 2
	left := clean.NRGBAAt(leftEdgeX, edgeY)
	right := clean.NRGBAAt(rightEdgeX, edgeY)
	if left.R < 220 || left.G < 210 || left.B < 190 || right.R < 210 || right.G > 60 || right.B > 60 {
		t.Fatalf("source edge marks were not preserved in clean viewport: left=%#v right=%#v", left, right)
	}
	changedLayout := full.Layout
	changedLayout.AppViewport.Min.X++
	if _, err := ComposeActiveViewport(context.Background(), app, config, changedLayout); err == nil {
		t.Fatal("clean viewport accepted a layout different from the recomputed presentation mapping")
	}
}

func TestComposeActiveViewportPhoneMatchesExistingViewportCrop(t *testing.T) {
	app := image.NewNRGBA(image.Rect(7, 11, 47, 91))
	for y := app.Bounds().Min.Y; y < app.Bounds().Max.Y; y++ {
		for x := app.Bounds().Min.X; x < app.Bounds().Max.X; x++ {
			app.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 3), G: uint8(y * 2), B: 140, A: 255})
		}
	}
	matte := RGB{R: 19, G: 27, B: 41}
	config := phoneConfig(430, 880)
	config.Matte = &matte
	full, err := Compose(context.Background(), app, config)
	if err != nil {
		t.Fatal(err)
	}
	clean, err := ComposeActiveViewport(context.Background(), app, config, full.Layout)
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < clean.Bounds().Dy(); y++ {
		for x := 0; x < clean.Bounds().Dx(); x++ {
			want := full.Raster.NRGBAAt(full.Layout.AppViewport.Min.X+x, full.Layout.AppViewport.Min.Y+y)
			if got := clean.NRGBAAt(x, y); got != want {
				t.Fatalf("phone clean viewport changed crop at (%d,%d): got %#v want %#v", x, y, got, want)
			}
		}
	}
}

func TestPhoneV3CarrierUsesMeasuredSafeSecondLineAndRejectsOverflow(t *testing.T) {
	for _, target := range []image.Point{image.Pt(430, 880), image.Pt(1206, 2622), image.Pt(360, 1000)} {
		src := opaqueTestImage(image.Rect(0, 0, 16, 24))
		c := Config{Target: target, Phone: &PhoneConfig{
			ProfileVersion: PhoneProfileVersionAuto, Appearance: AppearanceAuto,
			ShowStatus: true, Carrier: "Meridian",
		}}
		result, err := Compose(context.Background(), src, c)
		if err != nil {
			t.Fatalf("profile-owned carrier safe area rejected %v: %v", target, err)
		}
		withoutCarrier := c
		phoneWithoutCarrier := *c.Phone
		phoneWithoutCarrier.Carrier = ""
		withoutCarrier.Phone = &phoneWithoutCarrier
		baseline, err := Compose(context.Background(), src, withoutCarrier)
		if err != nil {
			t.Fatal(err)
		}
		sx, sy := float64(target.X)/phoneRefWidth, float64(target.Y)/phoneRefHeight
		left, right := int(math.Round(24*sx)), int(math.Round(142*sx))
		top, bottom := int(math.Round(52*sy)), int(math.Round(72*sy))
		found := 0
		for y := top; y < bottom; y++ {
			for x := 0; x < target.X; x++ {
				if result.Raster.NRGBAAt(x, y) != baseline.Raster.NRGBAAt(x, y) {
					found++
					if x < left || x >= right {
						t.Fatalf("carrier ink escaped safe horizontal span at %v: (%d,%d)", target, x, y)
					}
				}
			}
		}
		if found == 0 {
			t.Fatalf("carrier was not rasterized in safe second line for %v", target)
		}
		c.Phone.Carrier = strings.Repeat("W", 32)
		if _, err := BuildLayout(src, c); !errors.Is(err, ErrCarrierDoesNotFit) {
			t.Fatalf("overlong carrier was silently clipped for %v: %v", target, err)
		}
	}
}

func TestPhoneV2ConfigAndLayoutJSONOmitV3InkIdentity(t *testing.T) {
	src := opaqueTestImage(image.Rect(3, 7, 13, 29))
	c := phoneConfig(430, 880)
	layout, err := BuildLayout(src, c)
	if err != nil {
		t.Fatal(err)
	}
	if layout.ProfileVersion != PhoneProfileVersion || layout.ResolvedStatusInk != nil || layout.ResolvedHomeInk != nil {
		t.Fatalf("legacy v2 layout acquired v3 state: %+v", layout)
	}
	layoutJSON, err := json.Marshal(layout)
	if err != nil {
		t.Fatal(err)
	}
	configJSON, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{layoutJSON, configJSON} {
		if bytes.Contains(data, []byte("ResolvedStatusInk")) || bytes.Contains(data, []byte("ResolvedHomeInk")) || bytes.Contains(data, []byte("profile_version")) {
			t.Fatalf("v2 JSON contains v3-only identity: %s", data)
		}
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
	var firstPNG, secondPNG bytes.Buffer
	if err := png.Encode(&firstPNG, a.Raster); err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(&secondPNG, b.Raster); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstPNG.Bytes(), secondPNG.Bytes()) {
		t.Fatal("same config and source did not produce byte-identical PNG encoding")
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
	inkDark := colorFromRGB(fixedAppearanceInk(AppearanceDark))
	inkLight := colorFromRGB(fixedAppearanceInk(AppearanceLight))
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

func countColor(img *image.NRGBA, r image.Rectangle, want color.NRGBA) int {
	r = r.Intersect(img.Bounds())
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if img.NRGBAAt(x, y) == want {
				n++
			}
		}
	}
	return n
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

type genericRaster struct{ image.Image }

func TestBilinearConcreteRasterRetainsPremultipliedGenericPixels(t *testing.T) {
	bounds := image.Rect(7, 11, 16, 20)
	rasters := []interface {
		image.Image
		Set(int, int, color.Color)
	}{image.NewRGBA(bounds), image.NewNRGBA(bounds), image.NewRGBA64(bounds), image.NewNRGBA64(bounds), image.NewGray(bounds), image.NewGray16(bounds), image.NewAlpha(bounds), image.NewAlpha16(bounds)}
	for _, raster := range rasters {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				raster.Set(x, y, color.NRGBA64{R: uint16(x * 3901), G: uint16(y * 2309), B: uint16((x + y) * 1297), A: uint16((x*17 + y*31) * 59)})
			}
		}
		for _, point := range [][2]float64{{3, 12.1}, {7, 11}, {8.25, 14.75}, {15.9, 18.1}, {20, 25}} {
			matte := color.NRGBA{R: 17, G: 89, B: 131, A: 255}
			got := sampleBilinearOver(raster, bounds, point[0], point[1], matte)
			want := sampleBilinearOver(genericRaster{raster}, bounds, point[0], point[1], matte)
			if got != want {
				t.Fatalf("%T at%v: got%v want%v", raster, point, got, want)
			}
		}
	}
}
