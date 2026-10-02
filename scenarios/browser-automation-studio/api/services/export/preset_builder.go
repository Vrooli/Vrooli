package export

import (
	"strings"

	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
	"google.golang.org/protobuf/proto"
)

// BuildThemeFromPreset constructs an ExportTheme by applying preset configurations
// to the baseline theme from the movie spec.
func BuildThemeFromPreset(baseline *exportsv1.ReplaySpec, preset *ThemePreset) *exportsv1.ReplayTheme {
	if baseline == nil || preset == nil {
		return nil
	}

	theme := &exportsv1.ReplayTheme{}
	if baseline.GetTheme() != nil {
		theme = proto.Clone(baseline.Theme).(*exportsv1.ReplayTheme)
	}
	if theme.BrowserChrome == nil {
		theme.BrowserChrome = &exportsv1.ReplayBrowserChrome{}
	}

	// Apply chrome theme preset
	if chromeID := strings.TrimSpace(preset.ChromeTheme); chromeID != "" {
		if chrome, ok := ChromeThemePresets[chromeID]; ok {
			theme.BrowserChrome.Visible = chrome.Visible
			if chrome.Variant != "" {
				theme.BrowserChrome.Variant = chrome.Variant
			}
			theme.BrowserChrome.ShowAddress = chrome.ShowAddress
			if chrome.AccentColor != "" {
				theme.BrowserChrome.AccentColor = chrome.AccentColor
				theme.AccentColor = chrome.AccentColor
			}
			if chrome.AmbientGlow != "" {
				theme.AmbientGlow = chrome.AmbientGlow
			}
		}
	}

	// Apply background theme preset
	if backgroundID := strings.TrimSpace(preset.BackgroundTheme); backgroundID != "" {
		if background, ok := BackgroundThemePresets[backgroundID]; ok {
			if len(background.Gradient) > 0 {
				theme.BackgroundGradient = append([]string{}, background.Gradient...)
			}
			if background.Pattern != "" {
				theme.BackgroundPattern = background.Pattern
			}
			if background.Surface != "" {
				theme.SurfaceColor = background.Surface
			}
			if background.AmbientGlow != "" {
				theme.AmbientGlow = background.AmbientGlow
			}
		}
	}

	// Set sensible defaults for missing fields
	if theme.BrowserChrome.Title == "" {
		if name := strings.TrimSpace(baseline.GetExecution().GetWorkflowName()); name != "" {
			theme.BrowserChrome.Title = name
		} else {
			theme.BrowserChrome.Title = "Vrooli Ascension"
		}
	}
	if theme.AccentColor == "" {
		theme.AccentColor = DefaultAccentColor
	}
	if theme.BrowserChrome.AccentColor == "" {
		theme.BrowserChrome.AccentColor = theme.AccentColor
	}
	if theme.AmbientGlow == "" {
		theme.AmbientGlow = "rgba(56,189,248,0.22)"
	}
	if theme.SurfaceColor == "" {
		theme.SurfaceColor = "rgba(15,23,42,0.82)"
	}
	if len(theme.BackgroundGradient) == 0 {
		theme.BackgroundGradient = []string{"#0F172A"}
	}
	if theme.BrowserChrome.Variant == "" {
		theme.BrowserChrome.Variant = strings.TrimSpace(preset.ChromeTheme)
	}

	return theme
}

// BuildCursorSpec constructs an ExportCursorSpec by applying preset configurations
// and defaults to the existing cursor spec.
func BuildCursorSpec(existing *exportsv1.ReplayCursor, preset *CursorPreset) *exportsv1.ReplayCursor {
	cursor := &exportsv1.ReplayCursor{}
	if existing != nil {
		cursor = proto.Clone(existing).(*exportsv1.ReplayCursor)
	}
	if cursor.Trail == nil {
		cursor.Trail = &exportsv1.ReplayCursorTrail{}
	}
	if cursor.ClickPulse == nil {
		cursor.ClickPulse = &exportsv1.ReplayClickPulse{}
	}

	// Apply defaults to existing spec
	if cursor.Scale <= 0 {
		cursor.Scale = 1.0
	}
	if cursor.InitialPosition == "" {
		cursor.InitialPosition = "center"
	}
	if cursor.ClickAnimation == "" {
		cursor.ClickAnimation = "pulse"
	}

	// Apply preset if provided
	if preset != nil {
		if themeKey := strings.TrimSpace(preset.Theme); themeKey != "" {
			if cfg, ok := CursorThemePresets[themeKey]; ok {
				if cfg.Style != "" {
					cursor.Style = cfg.Style
				}
				if cfg.AccentColor != "" {
					cursor.AccentColor = cfg.AccentColor
				}
				cursor.Trail.Enabled = cfg.TrailEnabled
				if cfg.TrailOpacity > 0 {
					cursor.Trail.Opacity = cfg.TrailOpacity
				}
				if cfg.TrailFadeMs > 0 {
					cursor.Trail.FadeMs = int32(cfg.TrailFadeMs)
				}
				if cfg.TrailWeight > 0 {
					cursor.Trail.Weight = cfg.TrailWeight
				}
				cursor.ClickPulse.Enabled = cfg.ClickEnabled
				if cfg.ClickOpacity > 0 {
					cursor.ClickPulse.Opacity = cfg.ClickOpacity
				}
			}
		}

		// Apply override values from preset
		if preset.Scale > 0 {
			cursor.Scale = ClampCursorScale(preset.Scale)
		}
		if pos := strings.TrimSpace(preset.InitialPosition); pos != "" {
			cursor.InitialPosition = pos
		}
		if anim := strings.TrimSpace(preset.ClickAnimation); anim != "" {
			cursor.ClickAnimation = anim
			if strings.EqualFold(anim, "none") {
				cursor.ClickPulse.Enabled = false
			} else if !cursor.ClickPulse.Enabled {
				cursor.ClickPulse.Enabled = true
			}
		}
	}

	// Fill in remaining defaults
	if cursor.AccentColor == "" {
		cursor.AccentColor = DefaultAccentColor
	}
	if cursor.Trail.Enabled && cursor.Trail.Opacity <= 0 {
		cursor.Trail.Opacity = 0.55
	}
	if cursor.Trail.Enabled && cursor.Trail.Weight <= 0 {
		cursor.Trail.Weight = 0.16
	}
	if cursor.ClickPulse.Enabled && cursor.ClickPulse.Opacity <= 0 {
		cursor.ClickPulse.Opacity = 0.65
	}

	// Disable effects for hidden cursor style
	if strings.EqualFold(cursor.Style, "hidden") {
		cursor.Trail.Enabled = false
		cursor.ClickPulse.Enabled = false
	}

	return cursor
}
