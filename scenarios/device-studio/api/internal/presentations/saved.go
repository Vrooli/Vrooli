package presentations

import (
	"fmt"
	"image"

	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

const (
	SemanticVersionV2 = "device-studio-presentation-v2"
	SemanticVersionV3 = "device-studio-presentation-v3"
)

// ConfigFromSaved maps persisted protobuf choices to the pinned D6a raster
// profile. Defaults are explicit here so later workers do not consult UI or
// current compiler defaults. Plain app-only presentations use the same
// uniform contain/matte adapter; plain already-composed captures keep their
// legacy exact-raster path.
func ConfigFromSaved(target image.Point, p *shotsv1.ScreenPresentation) (Config, error) {
	return ConfigFromSavedVersion(target, p, SemanticVersionV2)
}

// ConfigFromSavedVersion maps a canonical typed snapshot to the exact profile
// semantics selected by its immutable presentation version.
func ConfigFromSavedVersion(target image.Point, p *shotsv1.ScreenPresentation, version string) (Config, error) {
	if p == nil || target.X <= 0 || target.Y <= 0 {
		return Config{}, ErrInvalidConfig
	}
	if p.GetKind() == shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN && version != SemanticVersionV2 {
		return Config{}, ErrInvalidConfig
	}
	if (p.GetKind() == shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM && version != SemanticVersionV2 && version != SemanticVersionV3) ||
		(p.GetKind() == shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP && version != SemanticVersionV2) {
		return Config{}, ErrInvalidConfig
	}
	if err := CanonicalizeSavedVersion(p, version); err != nil {
		return Config{}, err
	}
	for _, value := range []*shotsv1.PresentationRGB{p.GetMatte()} {
		if !validSavedRGB(value) {
			return Config{}, ErrInvalidConfig
		}
	}
	c := Config{Target: target, Matte: savedRGB(p.GetMatte())}
	switch p.GetKind() {
	case shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN:
		if p.GetCaptureArea() != shotsv1.CaptureArea_CAPTURE_AREA_APP_ONLY || p.GetPhoneSystemChrome() != nil || p.GetBrowserDesktopChrome() != nil ||
			p.GetFitMode() != shotsv1.ScreenFitMode_SCREEN_FIT_MODE_UNSPECIFIED && p.GetFitMode() != shotsv1.ScreenFitMode_SCREEN_FIT_MODE_CONTAIN {
			return Config{}, ErrInvalidConfig
		}
	case shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM:
		v := p.GetPhoneSystemChrome()
		var status, home, batteryLabel, cellular, wifi *bool
		if v != nil && (!validSavedRGB(v.GetTopFill()) || !validSavedRGB(v.GetBottomFill()) || !validSavedRGB(v.GetStatusInk()) || !validSavedRGB(v.GetHomeInk())) {
			return Config{}, ErrInvalidConfig
		}
		if v != nil {
			status, home, batteryLabel = v.ShowStatus, v.ShowHomeIndicator, v.ShowBatteryPercent
			cellular, wifi = v.ShowCellularIndicator, v.ShowWifiIndicator
			if v.GetBatteryPercent() > 100 || !validAppearance(v.GetAppearance(), version == SemanticVersionV3) {
				return Config{}, ErrInvalidConfig
			}
		}
		phone := PhoneConfig{ShowStatus: boolDefault(status, true), ShowHome: boolDefault(home, true),
			Clock: defaultText(v.GetClockText(), "9:41"), Carrier: v.GetCarrierText(), BatteryPercent: uint8(v.GetBatteryPercent()),
			ShowBatteryPercent: boolDefault(batteryLabel, false), Cellular: boolDefault(cellular, true),
			WiFi: boolDefault(wifi, true), Appearance: savedAppearance(v.GetAppearance()),
			TopFill: savedRGB(v.GetTopFill()), BottomFill: savedRGB(v.GetBottomFill()),
			StatusInk: savedRGB(v.GetStatusInk()), HomeInk: savedRGB(v.GetHomeInk())}
		if version == SemanticVersionV2 {
			phone.Carrier = defaultText(v.GetCarrierText(), "Meridian")
		} else {
			phone.ProfileVersion = PhoneProfileVersionAuto
		}
		c.Phone = &phone
	case shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP:
		v := p.GetBrowserDesktopChrome()
		var controls, toolbar, menu, dock *bool
		if v != nil && (!validSavedRGB(v.GetDesktopColor()) || !validSavedRGB(v.GetWindowColor()) || !validSavedRGB(v.GetChromeColor())) {
			return Config{}, ErrInvalidConfig
		}
		if v != nil && !validAppearance(v.GetAppearance(), false) {
			return Config{}, ErrInvalidConfig
		}
		if v != nil {
			controls, toolbar, menu, dock = v.ShowWindowControls, v.ShowToolbar, v.ShowMenuBar, v.ShowDock
		}
		browser := BrowserConfig{Title: defaultText(v.GetWindowTitle(), "Meridian"), Address: defaultText(v.GetAddressText(), "example.com"), Appearance: savedAppearance(v.GetAppearance()),
			ShowWindowControls: boolDefault(controls, true), ShowToolbar: boolDefault(toolbar, true),
			ShowMenuBar: boolDefault(menu, false), ShowDock: boolDefault(dock, false),
			DesktopColor: savedRGB(v.GetDesktopColor()), WindowColor: savedRGB(v.GetWindowColor()), ChromeColor: savedRGB(v.GetChromeColor())}
		if v == nil {
			browser.Title, browser.Address = "Meridian", "example.com"
		}
		if rect := v.GetWindow(); rect != nil {
			browser.Window = BasisRect{X: int(rect.GetX()), Y: int(rect.GetY()), Width: int(rect.GetWidth()), Height: int(rect.GetHeight())}
		}
		c.Browser = &browser
	default:
		return Config{}, fmt.Errorf("%w: saved presentation kind has no D6a profile", ErrInvalidConfig)
	}
	return c, nil
}

// CanonicalizeSaved writes the versioned simulated-profile defaults back into
// the typed revision so later reads never infer defaults from current code.
func CanonicalizeSaved(p *shotsv1.ScreenPresentation) error {
	return CanonicalizeSavedVersion(p, SemanticVersionV2)
}

// CanonicalizeSavedVersion persists defaults for the selected immutable
// presentation semantics. V2 remains the compatibility default for callers
// that have not stored a semantic version.
func CanonicalizeSavedVersion(p *shotsv1.ScreenPresentation, version string) error {
	if p == nil || !validSavedRGB(p.GetMatte()) {
		return ErrInvalidConfig
	}
	if version != SemanticVersionV2 && version != SemanticVersionV3 {
		return ErrInvalidConfig
	}
	if p.GetKind() == shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP && version != SemanticVersionV2 {
		return ErrInvalidConfig
	}
	switch p.GetKind() {
	case shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN:
		if version != SemanticVersionV2 || p.GetCaptureArea() != shotsv1.CaptureArea_CAPTURE_AREA_APP_ONLY || p.GetPhoneSystemChrome() != nil || p.GetBrowserDesktopChrome() != nil ||
			p.GetFitMode() != shotsv1.ScreenFitMode_SCREEN_FIT_MODE_UNSPECIFIED && p.GetFitMode() != shotsv1.ScreenFitMode_SCREEN_FIT_MODE_CONTAIN {
			return ErrInvalidConfig
		}
	case shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM:
		v := p.GetPhoneSystemChrome()
		if v == nil {
			v = &shotsv1.PhoneSystemChrome{}
			p.PhoneSystemChrome = v
		}
		allowAuto := version == SemanticVersionV3
		if !validSavedRGB(v.GetTopFill()) || !validSavedRGB(v.GetBottomFill()) || !validSavedRGB(v.GetStatusInk()) || !validSavedRGB(v.GetHomeInk()) || !validAppearance(v.GetAppearance(), allowAuto) || v.GetBatteryPercent() > 100 {
			return ErrInvalidConfig
		}
		v.ClockText = defaultText(v.ClockText, "9:41")
		if version == SemanticVersionV2 {
			v.CarrierText = defaultText(v.CarrierText, "Meridian")
		}
		if v.BatteryPercent == nil {
			v.BatteryPercent = uint32Pointer(84)
		}
		v.ShowStatus = boolPointerDefault(v.ShowStatus, true)
		v.ShowHomeIndicator = boolPointerDefault(v.ShowHomeIndicator, true)
		v.ShowCellularIndicator = boolPointerDefault(v.ShowCellularIndicator, true)
		v.ShowWifiIndicator = boolPointerDefault(v.ShowWifiIndicator, true)
		v.ShowBatteryPercent = boolPointerDefault(v.ShowBatteryPercent, false)
		if !validText(v.ClockText, 8) || !validClock(v.ClockText) || !validText(v.CarrierText, 32) {
			return ErrInvalidConfig
		}
		if v.Appearance == shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_UNSPECIFIED {
			if version == SemanticVersionV3 {
				v.Appearance = shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_AUTO
			} else {
				v.Appearance = shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_LIGHT
			}
		}
	case shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP:
		v := p.GetBrowserDesktopChrome()
		if v == nil {
			v = &shotsv1.BrowserDesktopChrome{}
			p.BrowserDesktopChrome = v
		}
		if !validSavedRGB(v.GetDesktopColor()) || !validSavedRGB(v.GetWindowColor()) || !validSavedRGB(v.GetChromeColor()) || !validAppearance(v.GetAppearance(), false) {
			return ErrInvalidConfig
		}
		v.WindowTitle = defaultText(v.WindowTitle, "Meridian")
		v.AddressText = defaultText(v.AddressText, "example.com")
		v.ShowToolbar = boolPointerDefault(v.ShowToolbar, true)
		v.ShowWindowControls = boolPointerDefault(v.ShowWindowControls, true)
		v.ShowMenuBar = boolPointerDefault(v.ShowMenuBar, false)
		v.ShowDock = boolPointerDefault(v.ShowDock, false)
		if v.GetShowWindowControls() && !v.GetShowToolbar() {
			return ErrInvalidConfig
		}
		if !validText(v.WindowTitle, 128) || !validText(v.AddressText, 2048) {
			return ErrInvalidConfig
		}
		if v.Appearance == shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_UNSPECIFIED {
			v.Appearance = shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_LIGHT
		}
		if v.Window == nil {
			v.Window = &shotsv1.PresentationRect{X: 800, Y: 800, Width: 8400, Height: 8000}
		}
		if v.Window.GetX() > uint32(BrowserRectBasis) || v.Window.GetY() > uint32(BrowserRectBasis) || v.Window.GetWidth() == 0 || v.Window.GetWidth() > uint32(BrowserRectBasis) || v.Window.GetHeight() == 0 || v.Window.GetHeight() > uint32(BrowserRectBasis) || uint64(v.Window.GetX())+uint64(v.Window.GetWidth()) > uint64(BrowserRectBasis) || uint64(v.Window.GetY())+uint64(v.Window.GetHeight()) > uint64(BrowserRectBasis) {
			return ErrInvalidConfig
		}
	default:
		return ErrInvalidConfig
	}
	return nil
}

// ValidateConfig uses the real decoded source for bounds, alpha/matte, edge
// sampling, profile geometry, and glyph checks. Eligibility must not be inferred
// from a synthetic probe image.
func ValidateConfig(source image.Image, config Config) error {
	layout, err := BuildLayout(source, config)
	if err != nil {
		return err
	}
	_ = layout
	return validateGlyphs(config)
}

func boolPointerDefault(value *bool, fallback bool) *bool {
	if value != nil {
		return value
	}
	return &fallback
}

func uint32Pointer(value uint32) *uint32 { return &value }

func boolDefault(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

func savedAppearance(v shotsv1.PresentationAppearance) Appearance {
	switch v {
	case shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_DARK:
		return AppearanceDark
	case shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_AUTO:
		return AppearanceAuto
	case shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_LIGHT:
		return AppearanceLight
	default:
		return AppearanceLight
	}
}

func validAppearance(v shotsv1.PresentationAppearance, allowAuto bool) bool {
	return v == shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_UNSPECIFIED || v == shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_LIGHT || v == shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_DARK || allowAuto && v == shotsv1.PresentationAppearance_PRESENTATION_APPEARANCE_AUTO
}

func defaultText(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func savedRGB(v *shotsv1.PresentationRGB) *RGB {
	if v == nil {
		return nil
	}
	return &RGB{R: uint8(v.GetRed()), G: uint8(v.GetGreen()), B: uint8(v.GetBlue())}
}

func validSavedRGB(v *shotsv1.PresentationRGB) bool {
	return v == nil || v.GetRed() <= 255 && v.GetGreen() <= 255 && v.GetBlue() <= 255
}
