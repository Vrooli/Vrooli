package export

import (
	"strings"

	"github.com/vrooli/browser-automation-studio/services"
)

// Apply applies client-provided overrides to a movie spec, applying presets first,
// then explicit overrides, and finally synchronizing cursor-related fields across
// the spec's Cursor, Decor, and CursorMotion structures.
func Apply(spec *services.ReplayMovieSpec, overrides *Overrides) {
	if spec == nil || overrides == nil {
		return
	}

	// Apply theme preset, then explicit theme override
	if overrides.ThemePreset != nil {
		if theme := BuildThemeFromPreset(spec, overrides.ThemePreset); theme != nil {
			spec.Theme = *theme
		}
	}
	if overrides.Theme != nil {
		spec.Theme = *overrides.Theme
	}

	// Apply cursor preset, then explicit cursor override
	if overrides.CursorPreset != nil {
		spec.Cursor = BuildCursorSpec(spec.Cursor, overrides.CursorPreset)
	}
	if overrides.Cursor != nil {
		spec.Cursor = *overrides.Cursor
	}

	// Apply decor overrides and synchronize cursor fields
	applyDecorOverrides(spec, overrides)
	syncCursorFields(spec)
}

// applyDecorOverrides applies theme and cursor preset names to the Decor field.
// The Decor field stores the original preset names for provenance tracking.
func applyDecorOverrides(spec *services.ReplayMovieSpec, overrides *Overrides) {
	if spec == nil || overrides == nil {
		return
	}

	// Record theme preset names
	if preset := overrides.ThemePreset; preset != nil {
		if chrome := strings.TrimSpace(preset.ChromeTheme); chrome != "" {
			spec.Decor.ChromeTheme = chrome
		}
		if background := strings.TrimSpace(preset.BackgroundTheme); background != "" {
			spec.Decor.BackgroundTheme = background
		}
	}

	// Record cursor preset name and overrides
	if cursorPreset := overrides.CursorPreset; cursorPreset != nil {
		if theme := strings.TrimSpace(cursorPreset.Theme); theme != "" {
			spec.Decor.CursorTheme = theme
		}
		if initial := strings.TrimSpace(cursorPreset.InitialPosition); initial != "" {
			spec.Decor.CursorInitial = initial
		}
		if anim := strings.TrimSpace(cursorPreset.ClickAnimation); anim != "" {
			spec.Decor.CursorClickAnimation = anim
		}
		if cursorPreset.Scale > 0 {
			spec.Decor.CursorScale = ClampCursorScale(cursorPreset.Scale)
		}
	}

	// Apply explicit cursor override values to Decor
	if cursorOverride := overrides.Cursor; cursorOverride != nil {
		if strings.TrimSpace(cursorOverride.InitialPos) != "" {
			spec.Decor.CursorInitial = cursorOverride.InitialPos
		}
		if strings.TrimSpace(cursorOverride.ClickAnim) != "" {
			spec.Decor.CursorClickAnimation = cursorOverride.ClickAnim
		}
		if cursorOverride.Scale > 0 {
			spec.Decor.CursorScale = ClampCursorScale(cursorOverride.Scale)
		}
	}
}

// syncCursorFields synchronizes cursor-related values across Cursor, Decor, and CursorMotion
// to ensure consistency throughout the movie spec.
func syncCursorFields(spec *services.ReplayMovieSpec) {
	if spec == nil {
		return
	}

	// Sync initial position
	if strings.TrimSpace(spec.Decor.CursorInitial) == "" {
		spec.Decor.CursorInitial = strings.TrimSpace(spec.Cursor.InitialPos)
	}
	if strings.TrimSpace(spec.Cursor.InitialPos) == "" {
		spec.Cursor.InitialPos = spec.Decor.CursorInitial
	}

	// Sync click animation
	if strings.TrimSpace(spec.Decor.CursorClickAnimation) == "" {
		spec.Decor.CursorClickAnimation = strings.TrimSpace(spec.Cursor.ClickAnim)
	}
	if strings.TrimSpace(spec.Cursor.ClickAnim) == "" {
		spec.Cursor.ClickAnim = spec.Decor.CursorClickAnimation
	}

	// Sync cursor scale
	if spec.Decor.CursorScale <= 0 {
		spec.Decor.CursorScale = spec.Cursor.Scale
	}
	if spec.Decor.CursorScale <= 0 {
		spec.Decor.CursorScale = 1
	}
	if spec.Decor.CursorScale > 0 {
		clamped := ClampCursorScale(spec.Decor.CursorScale)
		spec.Decor.CursorScale = clamped
		if spec.Cursor.Scale <= 0 {
			spec.Cursor.Scale = clamped
		}
	}
	if spec.Cursor.Scale > 0 {
		spec.Cursor.Scale = ClampCursorScale(spec.Cursor.Scale)
	}

	// Sync to CursorMotion
	if strings.TrimSpace(spec.CursorMotion.InitialPosition) == "" {
		spec.CursorMotion.InitialPosition = spec.Decor.CursorInitial
	}
	if strings.TrimSpace(spec.CursorMotion.ClickAnimation) == "" {
		spec.CursorMotion.ClickAnimation = spec.Decor.CursorClickAnimation
	}
	if spec.Decor.CursorScale > 0 {
		spec.CursorMotion.CursorScale = ClampCursorScale(spec.Decor.CursorScale)
	} else if spec.Cursor.Scale > 0 {
		spec.CursorMotion.CursorScale = ClampCursorScale(spec.Cursor.Scale)
	}

	// Apply final defaults to CursorMotion
	if spec.CursorMotion.InitialPosition == "" {
		spec.CursorMotion.InitialPosition = "center"
	}
	if spec.CursorMotion.ClickAnimation == "" {
		spec.CursorMotion.ClickAnimation = spec.Cursor.ClickAnim
	}
	if spec.CursorMotion.CursorScale <= 0 {
		spec.CursorMotion.CursorScale = ClampCursorScale(spec.Cursor.Scale)
	}

	// Ensure Decor.CursorScale is set from CursorMotion if still missing
	if spec.Decor.CursorScale <= 0 {
		spec.Decor.CursorScale = spec.CursorMotion.CursorScale
	}
}
