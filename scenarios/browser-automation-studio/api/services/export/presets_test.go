package export

import (
	"testing"
)

func TestChromeThemePresets(t *testing.T) {
	expectedPresets := []string{"aurora", "chromium", "midnight", "minimal"}

	for _, name := range expectedPresets {
		t.Run(name, func(t *testing.T) {
			preset, ok := ChromeThemePresets[name]
			if !ok {
				t.Fatalf("expected preset %q to exist", name)
			}

			// Validate structure
			if preset.Variant == "" {
				t.Error("variant should not be empty")
			}
			if preset.AccentColor == "" {
				t.Error("accent color should not be empty")
			}
			if preset.AmbientGlow == "" {
				t.Error("ambient glow should not be empty")
			}

			// Special case: minimal theme should not be visible
			if name == "minimal" && preset.Visible {
				t.Error("minimal theme should not be visible")
			}
		})
	}
}

func TestBackgroundThemePresets(t *testing.T) {
	expectedPresets := []string{"aurora", "sunset", "ocean", "nebula", "grid", "charcoal", "steel", "emerald", "none"}

	for _, name := range expectedPresets {
		t.Run(name, func(t *testing.T) {
			preset, ok := BackgroundThemePresets[name]
			if !ok {
				t.Fatalf("expected preset %q to exist", name)
			}

			// Validate structure
			if len(preset.Gradient) == 0 {
				t.Error("gradient should not be empty")
			}
			if preset.Surface == "" {
				t.Error("surface should not be empty")
			}
		})
	}
}

func TestCursorThemePresets(t *testing.T) {
	expectedPresets := []string{"white", "dark", "aura", "arrowLight", "arrowDark", "arrowNeon", "handNeutral", "handAura"}

	for _, name := range expectedPresets {
		t.Run(name, func(t *testing.T) {
			preset, ok := CursorThemePresets[name]
			if !ok {
				t.Fatalf("expected preset %q to exist", name)
			}

			// Validate structure
			if preset.Style == "" {
				t.Error("style should not be empty")
			}
			if preset.AccentColor == "" {
				t.Error("accent color should not be empty")
			}

			// Validate trail opacity range
			if preset.TrailOpacity < 0 || preset.TrailOpacity > 1 {
				t.Errorf("trail opacity %f out of range [0,1]", preset.TrailOpacity)
			}

			// Validate click opacity range
			if preset.ClickOpacity < 0 || preset.ClickOpacity > 1 {
				t.Errorf("click opacity %f out of range [0,1]", preset.ClickOpacity)
			}

			// Validate trail weight is positive
			if preset.TrailWeight < 0 {
				t.Errorf("trail weight %f should be non-negative", preset.TrailWeight)
			}

			// Validate trail fade time is non-negative
			if preset.TrailFadeMs < 0 {
				t.Errorf("trail fade ms %d should be non-negative", preset.TrailFadeMs)
			}
		})
	}
}

func TestDefaultAccentColor(t *testing.T) {
	if DefaultAccentColor != "#38BDF8" {
		t.Errorf("DefaultAccentColor = %q, expected %q", DefaultAccentColor, "#38BDF8")
	}
}

func TestCursorScaleLimits(t *testing.T) {
	if CursorScaleMin != 0.5 {
		t.Errorf("CursorScaleMin = %f, expected 0.5", CursorScaleMin)
	}
	if CursorScaleMax != 3.0 {
		t.Errorf("CursorScaleMax = %f, expected 3.0", CursorScaleMax)
	}
	if CursorScaleMin >= CursorScaleMax {
		t.Error("CursorScaleMin should be less than CursorScaleMax")
	}
}

func TestChromeThemePresetConsistency(t *testing.T) {
	for name, preset := range ChromeThemePresets {
		t.Run(name, func(t *testing.T) {
			// Variant should match the preset name or be a valid alternative
			if preset.Visible && preset.Variant == "" {
				t.Error("visible chrome should have a variant")
			}

			// If not visible, address should not be shown
			if !preset.Visible && preset.ShowAddress {
				t.Error("invisible chrome should not show address")
			}
		})
	}
}

func TestBackgroundThemePresetGradients(t *testing.T) {
	for name, preset := range BackgroundThemePresets {
		t.Run(name, func(t *testing.T) {
			// All gradients should have at least one color
			if len(preset.Gradient) == 0 {
				t.Error("gradient should have at least one color")
			}

			// All gradient colors should start with #
			for i, color := range preset.Gradient {
				if len(color) == 0 || color[0] != '#' {
					t.Errorf("gradient[%d] = %q, expected hex color", i, color)
				}
			}
		})
	}
}

func TestCursorThemePresetAnimations(t *testing.T) {
	for name, preset := range CursorThemePresets {
		t.Run(name, func(t *testing.T) {
			// If trail is enabled, opacity and weight should be reasonable
			if preset.TrailEnabled {
				if preset.TrailOpacity == 0 {
					t.Error("enabled trail should have non-zero opacity")
				}
				if preset.TrailWeight == 0 {
					t.Error("enabled trail should have non-zero weight")
				}
			}

			// If click animation is enabled, opacity should be reasonable
			if preset.ClickEnabled {
				if preset.ClickOpacity == 0 {
					t.Error("enabled click animation should have non-zero opacity")
				}
			}
		})
	}
}
