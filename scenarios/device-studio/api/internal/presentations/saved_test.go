package presentations

import (
	"encoding/json"
	"image"
	"image/color"
	"testing"

	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

func TestSavedPhoneConfigurationUsesPresenceAwareDefaultsAndOverrides(t *testing.T) {
	cfg, err := ConfigFromSaved(image.Pt(1206, 2622), &shotsv1.ScreenPresentation{Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Phone == nil || !cfg.Phone.ShowStatus || !cfg.Phone.ShowHome || !cfg.Phone.Cellular || !cfg.Phone.WiFi || cfg.Phone.Clock != "9:41" || cfg.Phone.Carrier != "Meridian" {
		t.Fatalf("unexpected pinned defaults: %+v", cfg.Phone)
	}
	p := &shotsv1.ScreenPresentation{Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM}
	if err := CanonicalizeSaved(p); err != nil {
		t.Fatal(err)
	}
	if p.PhoneSystemChrome.GetBatteryPercent() != 84 || !p.PhoneSystemChrome.GetShowStatus() || !p.PhoneSystemChrome.GetShowHomeIndicator() || p.PhoneSystemChrome.GetAppearance() != shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_LIGHT {
		t.Fatalf("phone defaults were not persisted: %+v", p.PhoneSystemChrome)
	}
	zero := uint32(0)
	p = &shotsv1.ScreenPresentation{Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM, PhoneSystemChrome: &shotsv1.PhoneSystemChrome{BatteryPercent: &zero}}
	if err := CanonicalizeSaved(p); err != nil {
		t.Fatal(err)
	}
	if p.PhoneSystemChrome.GetBatteryPercent() != 0 {
		t.Fatalf("explicit zero battery percentage was replaced: %+v", p.PhoneSystemChrome)
	}
	noStatus := false
	cfg, err = ConfigFromSaved(image.Pt(1206, 2622), &shotsv1.ScreenPresentation{Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM, PhoneSystemChrome: &shotsv1.PhoneSystemChrome{ShowStatus: &noStatus, TopFill: &shotsv1.PresentationRGB{}}})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Phone.ShowStatus || cfg.Phone.TopFill == nil || *cfg.Phone.TopFill != (RGB{}) {
		t.Fatalf("explicit false/black override was lost: %+v", cfg.Phone)
	}
	_, err = ConfigFromSaved(image.Pt(1206, 2622), &shotsv1.ScreenPresentation{Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM, PhoneSystemChrome: &shotsv1.PhoneSystemChrome{TopFill: &shotsv1.PresentationRGB{Red: 256}}})
	if err == nil {
		t.Fatal("out-of-range RGB accepted")
	}
}

func TestSavedPhoneVersionDefaultsStayPinnedAndBrowserRejectsAuto(t *testing.T) {
	legacy := &shotsv1.ScreenPresentation{Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM}
	legacyConfig, err := ConfigFromSavedVersion(image.Pt(1206, 2622), legacy, SemanticVersionV2)
	if err != nil {
		t.Fatal(err)
	}
	if legacyConfig.Phone.Carrier != "Meridian" || legacyConfig.Phone.Appearance != AppearanceLight || legacyConfig.Phone.ProfileVersion != "" || legacyConfig.Phone.BatteryPercent != 84 {
		t.Fatalf("v2 compatibility defaults changed: %+v; saved=%+v", legacyConfig.Phone, legacy.PhoneSystemChrome)
	}

	v3 := &shotsv1.ScreenPresentation{Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM}
	v3Config, err := ConfigFromSavedVersion(image.Pt(1206, 2622), v3, SemanticVersionV3)
	if err != nil {
		t.Fatal(err)
	}
	if v3Config.Phone.Carrier != "" || v3Config.Phone.Appearance != AppearanceAuto || v3Config.Phone.ProfileVersion != "3" || v3Config.Phone.BatteryPercent != 84 {
		t.Fatalf("v3 defaults were not pinned: config=%+v saved=%+v", v3Config.Phone, v3.PhoneSystemChrome)
	}
	if v3Config.Phone.StatusInk != nil || v3Config.Phone.HomeInk != nil {
		t.Fatalf("absent v3 inks should remain automatic: %+v", v3Config.Phone)
	}

	for _, appearance := range []shotsv1.PresentationAppearance{
		shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_LIGHT,
		shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_DARK,
	} {
		p := &shotsv1.ScreenPresentation{Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM, PhoneSystemChrome: &shotsv1.PhoneSystemChrome{Appearance: appearance}}
		cfg, err := ConfigFromSavedVersion(image.Pt(1206, 2622), p, SemanticVersionV3)
		if err != nil || cfg.Phone.Appearance == AppearanceAuto {
			t.Fatalf("explicit appearance %s was not preserved: config=%+v error=%v", appearance, cfg.Phone, err)
		}
	}

	browser := &shotsv1.ScreenPresentation{Kind: shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP, BrowserDesktopChrome: &shotsv1.BrowserDesktopChrome{Appearance: shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_AUTO}}
	if _, err := ConfigFromSavedVersion(image.Pt(1440, 900), browser, SemanticVersionV2); err == nil {
		t.Fatal("browser AUTO appearance was accepted")
	}
}

func TestSavedPhoneV3ExplicitInksMapAndRejectOutOfRange(t *testing.T) {
	status := &shotsv1.PresentationRGB{Red: 236, Green: 240, Blue: 247}
	home := &shotsv1.PresentationRGB{Red: 12, Green: 18, Blue: 24}
	p := &shotsv1.ScreenPresentation{
		Kind:              shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM,
		PhoneSystemChrome: &shotsv1.PhoneSystemChrome{StatusInk: status, HomeInk: home},
	}
	config, err := ConfigFromSavedVersion(image.Pt(1206, 2622), p, SemanticVersionV3)
	if err != nil {
		t.Fatal(err)
	}
	if want := (RGB{R: 236, G: 240, B: 247}); config.Phone.StatusInk == nil || *config.Phone.StatusInk != want {
		t.Fatalf("explicit status ink was not mapped: got=%+v want=%+v", config.Phone.StatusInk, want)
	}
	if want := (RGB{R: 12, G: 18, B: 24}); config.Phone.HomeInk == nil || *config.Phone.HomeInk != want {
		t.Fatalf("explicit home ink was not mapped: got=%+v want=%+v", config.Phone.HomeInk, want)
	}

	for _, field := range []struct {
		name string
		set  func(*shotsv1.PhoneSystemChrome, *shotsv1.PresentationRGB)
	}{
		{name: "status_ink", set: func(c *shotsv1.PhoneSystemChrome, v *shotsv1.PresentationRGB) { c.StatusInk = v }},
		{name: "home_ink", set: func(c *shotsv1.PhoneSystemChrome, v *shotsv1.PresentationRGB) { c.HomeInk = v }},
	} {
		t.Run(field.name, func(t *testing.T) {
			bad := &shotsv1.ScreenPresentation{Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM, PhoneSystemChrome: &shotsv1.PhoneSystemChrome{}}
			field.set(bad.PhoneSystemChrome, &shotsv1.PresentationRGB{Red: 256})
			if err := CanonicalizeSavedVersion(bad, SemanticVersionV3); err != ErrInvalidConfig {
				t.Fatalf("out-of-range %s canonicalization error=%v, want %v", field.name, err, ErrInvalidConfig)
			}
			bad = &shotsv1.ScreenPresentation{Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM, PhoneSystemChrome: &shotsv1.PhoneSystemChrome{}}
			field.set(bad.PhoneSystemChrome, &shotsv1.PresentationRGB{Red: 256})
			if _, err := ConfigFromSavedVersion(image.Pt(1206, 2622), bad, SemanticVersionV3); err != ErrInvalidConfig {
				t.Fatalf("out-of-range %s config error=%v, want %v", field.name, err, ErrInvalidConfig)
			}
		})
	}
}

func TestStoredPhoneV2ConfigMatchesRetainedGoldenShape(t *testing.T) {
	appearance := shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_DARK
	phone := &shotsv1.ScreenPresentation{
		Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM,
		PhoneSystemChrome: &shotsv1.PhoneSystemChrome{
			CarrierText: "Meridian", ClockText: "9:41", BatteryPercent: uint32Pointer(84),
			ShowCellularIndicator: boolPointerDefault(nil, true), ShowWifiIndicator: boolPointerDefault(nil, true),
			ShowHomeIndicator: boolPointerDefault(nil, true), ShowStatus: boolPointerDefault(nil, true),
			ShowBatteryPercent: boolPointerDefault(nil, false), Appearance: appearance,
		},
	}
	config, err := ConfigFromSavedVersion(image.Pt(1206, 2622), phone, SemanticVersionV2)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	const golden = `{"Target":{"X":1206,"Y":2622},"Matte":null,"Phone":{"ShowStatus":true,"ShowHome":true,"Clock":"9:41","Carrier":"Meridian","BatteryPercent":84,"ShowBatteryPercent":false,"Cellular":true,"WiFi":true,"Appearance":"dark","TopFill":null,"BottomFill":null},"Browser":null}`
	if string(encoded) != golden {
		t.Fatalf("v2 config JSON changed from retained live closure shape:\n got %s\nwant %s", encoded, golden)
	}
}

func TestSavedBrowserConfigurationMapsNormalizedWindowAndExplicitHides(t *testing.T) {
	showControls := false
	cfg, err := ConfigFromSaved(image.Pt(2560, 1440), &shotsv1.ScreenPresentation{Kind: shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP, BrowserDesktopChrome: &shotsv1.BrowserDesktopChrome{
		ShowWindowControls: &showControls, Window: &shotsv1.PresentationRect{X: 500, Y: 600, Width: 9000, Height: 8000}, ChromeColor: &shotsv1.PresentationRGB{},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Browser == nil || cfg.Browser.ShowWindowControls || !cfg.Browser.ShowToolbar || cfg.Browser.Window != (BasisRect{X: 500, Y: 600, Width: 9000, Height: 8000}) || cfg.Browser.ChromeColor == nil || *cfg.Browser.ChromeColor != (RGB{}) {
		t.Fatalf("saved browser choices were not preserved: %+v", cfg.Browser)
	}
}

func TestSavedBrowserDefaultsArePinnedAndControlsRequireToolbar(t *testing.T) {
	p := &shotsv1.ScreenPresentation{Kind: shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP}
	config, err := ConfigFromSaved(image.Pt(1440, 900), p)
	if err != nil {
		t.Fatal(err)
	}
	chrome := p.BrowserDesktopChrome
	if chrome.GetWindowTitle() != "Meridian" || chrome.GetAddressText() != "example.com" || chrome.GetAppearance() != shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_LIGHT || !chrome.GetShowToolbar() || !chrome.GetShowWindowControls() {
		t.Fatalf("browser defaults were not persisted: %+v", chrome)
	}
	if chrome.GetWindow().GetX() != 800 || chrome.GetWindow().GetY() != 800 || chrome.GetWindow().GetWidth() != 8400 || chrome.GetWindow().GetHeight() != 8000 {
		t.Fatalf("default window placement was not persisted: %+v", chrome.GetWindow())
	}
	source := image.NewNRGBA(image.Rect(0, 0, 700, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 700; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 80, G: 90, B: 100, A: 255})
		}
	}
	if err := ValidateConfig(source, config); err != nil {
		t.Fatalf("canonical toolbar config should validate: %v", err)
	}
	config.Browser.ShowToolbar = false
	if err := ValidateConfig(source, config); err != ErrInvalidConfig {
		t.Fatalf("window controls without a toolbar error=%v, want %v", err, ErrInvalidConfig)
	}
}

func TestCanonicalizeRejectsBrowserControlsWithoutToolbar(t *testing.T) {
	controls := true
	toolbar := false
	p := &shotsv1.ScreenPresentation{
		Kind: shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP,
		BrowserDesktopChrome: &shotsv1.BrowserDesktopChrome{
			ShowWindowControls: &controls,
			ShowToolbar:        &toolbar,
		},
	}
	if err := CanonicalizeSaved(p); err != ErrInvalidConfig {
		t.Fatalf("controls without toolbar canonicalization error=%v, want %v", err, ErrInvalidConfig)
	}
}

func TestCanonicalizeRejectsUnknownAppearance(t *testing.T) {
	p := &shotsv1.ScreenPresentation{
		Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM,
		PhoneSystemChrome: &shotsv1.PhoneSystemChrome{
			Appearance: shotsv1.PresentationAppearance(77),
		},
	}
	if err := CanonicalizeSaved(p); err != ErrInvalidConfig {
		t.Fatalf("unknown appearance canonicalization error=%v, want %v", err, ErrInvalidConfig)
	}
}

func TestValidateConfigUsesRealSourceAlphaGeometryAndGlyphs(t *testing.T) {
	p := &shotsv1.ScreenPresentation{Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM}
	config, err := ConfigFromSaved(image.Pt(1206, 2622), p)
	if err != nil {
		t.Fatal(err)
	}
	transparent := image.NewNRGBA(image.Rect(0, 0, 400, 800))
	if err := ValidateConfig(transparent, config); err != ErrMatteRequired {
		t.Fatalf("transparent source without matte error=%v", err)
	}
	config.Matte = &RGB{R: 12, G: 24, B: 36}
	if err := ValidateConfig(transparent, config); err != nil {
		t.Fatalf("transparent source with matte: %v", err)
	}
	tooWide := image.NewNRGBA(image.Rect(0, 0, maxRasterSide+1, 1))
	if err := ValidateConfig(tooWide, config); err == nil {
		t.Fatal("source geometry beyond the actual source limit was accepted")
	}
	config.Phone.Carrier = "Meridian漢"
	opaque := image.NewNRGBA(image.Rect(0, 0, 400, 800))
	for y := 0; y < 800; y++ {
		for x := 0; x < 400; x++ {
			opaque.SetNRGBA(x, y, color.NRGBA{R: 40, G: 80, B: 120, A: 255})
		}
	}
	if err := ValidateConfig(opaque, config); err != ErrUnsupportedGlyph {
		t.Fatalf("unsupported configured glyph error=%v", err)
	}
}
