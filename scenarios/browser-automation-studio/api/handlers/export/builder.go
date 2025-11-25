package export

import (
	"strings"

	"github.com/vrooli/browser-automation-studio/services"
)

// BuildThemeFromPreset constructs an ExportTheme by applying preset configurations
// to the baseline theme from the movie spec.
func BuildThemeFromPreset(baseline *services.ReplayMovieSpec, preset *ThemePreset) *services.ExportTheme {
	if baseline == nil || preset == nil {
		return nil
	}

	theme := baseline.Theme

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
		if name := strings.TrimSpace(baseline.Execution.WorkflowName); name != "" {
			theme.BrowserChrome.Title = name
		} else {
			theme.BrowserChrome.Title = "Browser Automation Studio"
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

	return &theme
}

// BuildCursorSpec constructs an ExportCursorSpec by applying preset configurations
// and defaults to the existing cursor spec.
func BuildCursorSpec(existing services.ExportCursorSpec, preset *CursorPreset) services.ExportCursorSpec {
	cursor := existing

	// Apply defaults to existing spec
	if cursor.Scale <= 0 {
		cursor.Scale = 1.0
	}
	if cursor.InitialPos == "" {
		cursor.InitialPos = "center"
	}
	if cursor.ClickAnim == "" {
		cursor.ClickAnim = "pulse"
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
					cursor.Trail.FadeMs = cfg.TrailFadeMs
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
			cursor.InitialPos = pos
		}
		if anim := strings.TrimSpace(preset.ClickAnimation); anim != "" {
			cursor.ClickAnim = anim
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
