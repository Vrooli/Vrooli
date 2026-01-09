package export

import (
	"testing"

	exportservices "github.com/vrooli/browser-automation-studio/services/export"
)

func TestApply_NilSpec(t *testing.T) {
	overrides := &Overrides{
		ThemePreset: &ThemePreset{
			ChromeTheme:     "modern",
			BackgroundTheme: "gradient-blue",
		},
	}

	// Should not panic with nil spec
	Apply(nil, overrides)
}

func TestApply_NilOverrides(t *testing.T) {
	spec := &exportservices.ReplayMovieSpec{}

	// Should not panic with nil overrides
	Apply(spec, nil)
}

func TestApply_ThemePreset(t *testing.T) {
	spec := &exportservices.ReplayMovieSpec{
		Theme: exportservices.ExportTheme{
			AccentColor: "#000000",
		},
	}

	overrides := &Overrides{
		ThemePreset: &ThemePreset{
			ChromeTheme:     "modern",
			BackgroundTheme: "gradient-blue",
		},
	}

	Apply(spec, overrides)

	// Theme should be updated based on preset
	// We just verify the function runs without error
	// The actual theme building is tested in builder_test.go
}

func TestApply_ExplicitThemeOverride(t *testing.T) {
	spec := &exportservices.ReplayMovieSpec{
		Theme: exportservices.ExportTheme{
			AccentColor: "#000000",
		},
	}

	overrides := &Overrides{
		Theme: &exportservices.ExportTheme{
			AccentColor:  "#FF0000",
			SurfaceColor: "#FFFFFF",
		},
	}

	Apply(spec, overrides)

	if spec.Theme.AccentColor != "#FF0000" {
		t.Errorf("expected accent color #FF0000, got %s", spec.Theme.AccentColor)
	}
	if spec.Theme.SurfaceColor != "#FFFFFF" {
		t.Errorf("expected surface color #FFFFFF, got %s", spec.Theme.SurfaceColor)
	}
}

func TestApply_ThemePresetThenOverride(t *testing.T) {
	spec := &exportservices.ReplayMovieSpec{}

	overrides := &Overrides{
		ThemePreset: &ThemePreset{
			ChromeTheme:     "modern",
			BackgroundTheme: "gradient-blue",
		},
		Theme: &exportservices.ExportTheme{
			AccentColor: "#123456", // Explicit override should win
		},
	}

	Apply(spec, overrides)

	// Explicit theme override should take precedence
	if spec.Theme.AccentColor != "#123456" {
		t.Errorf("expected explicit theme override to win, got %s", spec.Theme.AccentColor)
	}
}

func TestApply_CursorPreset(t *testing.T) {
	spec := &exportservices.ReplayMovieSpec{
		Cursor: exportservices.ExportCursorSpec{
			InitialPos: "center",
			Scale:      1.0,
		},
	}

	overrides := &Overrides{
		CursorPreset: &CursorPreset{
			Theme:           "default",
			InitialPosition: "top-left",
			ClickAnimation:  "ripple",
			Scale:           1.5,
		},
	}

	Apply(spec, overrides)

	if spec.Cursor.InitialPos != "top-left" {
		t.Errorf("expected cursor initial position to be updated to top-left, got %s", spec.Cursor.InitialPos)
	}
	if spec.Cursor.ClickAnim != "ripple" {
		t.Errorf("expected cursor click animation to be updated to ripple, got %s", spec.Cursor.ClickAnim)
	}
}

func TestApply_ExplicitCursorOverride(t *testing.T) {
	spec := &exportservices.ReplayMovieSpec{
		Cursor: exportservices.ExportCursorSpec{
			InitialPos: "center",
		},
	}

	overrides := &Overrides{
		Cursor: &exportservices.ExportCursorSpec{
			InitialPos: "bottom-right",
			ClickAnim:  "pulse",
			Scale:      2.0,
		},
	}

	Apply(spec, overrides)

	if spec.Cursor.InitialPos != "bottom-right" {
		t.Errorf("expected cursor initial position bottom-right, got %s", spec.Cursor.InitialPos)
	}
	if spec.Cursor.ClickAnim != "pulse" {
		t.Errorf("expected cursor click animation pulse, got %s", spec.Cursor.ClickAnim)
	}
	if spec.Cursor.Scale != 2.0 {
		t.Errorf("expected cursor scale 2.0, got %f", spec.Cursor.Scale)
	}
}
