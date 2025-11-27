package export

import (
	"testing"

	"github.com/vrooli/browser-automation-studio/services/ai"
	"github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/services/logutil"
	"github.com/vrooli/browser-automation-studio/services/recording"
	"github.com/vrooli/browser-automation-studio/services/replay"
	"github.com/vrooli/browser-automation-studio/services/workflow"
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
	spec := &export.ReplayMovieSpec{}

	// Should not panic with nil overrides
	Apply(spec, nil)
}

func TestApply_ThemePreset(t *testing.T) {
	spec := &export.ReplayMovieSpec{
		Theme: services.ExportTheme{
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
	spec := &export.ReplayMovieSpec{
		Theme: services.ExportTheme{
			AccentColor: "#000000",
		},
	}

	overrides := &Overrides{
		Theme: &services.ExportTheme{
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
	spec := &export.ReplayMovieSpec{}

	overrides := &Overrides{
		ThemePreset: &ThemePreset{
			ChromeTheme:     "modern",
			BackgroundTheme: "gradient-blue",
		},
		Theme: &services.ExportTheme{
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
	spec := &export.ReplayMovieSpec{
		Cursor: services.ExportCursorSpec{
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
	spec := &export.ReplayMovieSpec{
		Cursor: services.ExportCursorSpec{
			InitialPos: "center",
		},
	}

	overrides := &Overrides{
		Cursor: &services.ExportCursorSpec{
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

func TestApplyDecorOverrides_ThemePresetNames(t *testing.T) {
	spec := &export.ReplayMovieSpec{}

	overrides := &Overrides{
		ThemePreset: &ThemePreset{
			ChromeTheme:     "modern",
			BackgroundTheme: "gradient-blue",
		},
	}

	applyDecorOverrides(spec, overrides)

	if spec.Decor.ChromeTheme != "modern" {
		t.Errorf("expected chrome theme modern, got %s", spec.Decor.ChromeTheme)
	}
	if spec.Decor.BackgroundTheme != "gradient-blue" {
		t.Errorf("expected background theme gradient-blue, got %s", spec.Decor.BackgroundTheme)
	}
}

func TestApplyDecorOverrides_CursorPresetNames(t *testing.T) {
	spec := &export.ReplayMovieSpec{}

	overrides := &Overrides{
		CursorPreset: &CursorPreset{
			Theme:           "pointer",
			InitialPosition: "center",
			ClickAnimation:  "ripple",
			Scale:           1.5,
		},
	}

	applyDecorOverrides(spec, overrides)

	if spec.Decor.CursorTheme != "pointer" {
		t.Errorf("expected cursor theme pointer, got %s", spec.Decor.CursorTheme)
	}
	if spec.Decor.CursorInitial != "center" {
		t.Errorf("expected cursor initial center, got %s", spec.Decor.CursorInitial)
	}
	if spec.Decor.CursorClickAnimation != "ripple" {
		t.Errorf("expected click animation ripple, got %s", spec.Decor.CursorClickAnimation)
	}
	if spec.Decor.CursorScale != 1.5 {
		t.Errorf("expected cursor scale 1.5, got %f", spec.Decor.CursorScale)
	}
}

func TestApplyDecorOverrides_CursorScaleClamping(t *testing.T) {
	spec := &export.ReplayMovieSpec{}

	overrides := &Overrides{
		CursorPreset: &CursorPreset{
			Scale: 5.0, // Above max of 3.0
		},
	}

	applyDecorOverrides(spec, overrides)

	if spec.Decor.CursorScale != 3.0 {
		t.Errorf("expected cursor scale to be clamped to 3.0, got %f", spec.Decor.CursorScale)
	}
}

func TestApplyDecorOverrides_ExplicitCursorOverride(t *testing.T) {
	spec := &export.ReplayMovieSpec{}

	overrides := &Overrides{
		Cursor: &services.ExportCursorSpec{
			InitialPos: "top-right",
			ClickAnim:  "bounce",
			Scale:      1.8,
		},
	}

	applyDecorOverrides(spec, overrides)

	if spec.Decor.CursorInitial != "top-right" {
		t.Errorf("expected cursor initial top-right, got %s", spec.Decor.CursorInitial)
	}
	if spec.Decor.CursorClickAnimation != "bounce" {
		t.Errorf("expected click animation bounce, got %s", spec.Decor.CursorClickAnimation)
	}
	if spec.Decor.CursorScale != 1.8 {
		t.Errorf("expected cursor scale 1.8, got %f", spec.Decor.CursorScale)
	}
}

func TestSyncCursorFields_InitialPosition(t *testing.T) {
	spec := &export.ReplayMovieSpec{
		Cursor: services.ExportCursorSpec{
			InitialPos: "bottom-left",
		},
		Decor: services.ExportDecor{
			CursorInitial: "",
		},
	}

	syncCursorFields(spec)

	// Decor should be synced from Cursor
	if spec.Decor.CursorInitial != "bottom-left" {
		t.Errorf("expected decor cursor initial to sync from cursor, got %s", spec.Decor.CursorInitial)
	}
}

func TestSyncCursorFields_InitialPositionReverse(t *testing.T) {
	spec := &export.ReplayMovieSpec{
		Cursor: services.ExportCursorSpec{
			InitialPos: "",
		},
		Decor: services.ExportDecor{
			CursorInitial: "top-center",
		},
	}

	syncCursorFields(spec)

	// Cursor should be synced from Decor
	if spec.Cursor.InitialPos != "top-center" {
		t.Errorf("expected cursor initial pos to sync from decor, got %s", spec.Cursor.InitialPos)
	}
}

func TestSyncCursorFields_ClickAnimation(t *testing.T) {
	spec := &export.ReplayMovieSpec{
		Cursor: services.ExportCursorSpec{
			ClickAnim: "pulse",
		},
		Decor: services.ExportDecor{
			CursorClickAnimation: "",
		},
	}

	syncCursorFields(spec)

	if spec.Decor.CursorClickAnimation != "pulse" {
		t.Errorf("expected decor click animation to sync from cursor, got %s", spec.Decor.CursorClickAnimation)
	}
}

func TestSyncCursorFields_ScaleClamping(t *testing.T) {
	spec := &export.ReplayMovieSpec{
		Cursor: services.ExportCursorSpec{
			Scale: 4.0, // Above max
		},
	}

	syncCursorFields(spec)

	// Scale should be clamped
	if spec.Cursor.Scale != 3.0 {
		t.Errorf("expected cursor scale to be clamped to 3.0, got %f", spec.Cursor.Scale)
	}
	if spec.Decor.CursorScale != 3.0 {
		t.Errorf("expected decor cursor scale to be clamped to 3.0, got %f", spec.Decor.CursorScale)
	}
}

func TestSyncCursorFields_DefaultScale(t *testing.T) {
	spec := &export.ReplayMovieSpec{
		Decor: services.ExportDecor{
			CursorScale: 0,
		},
		Cursor: services.ExportCursorSpec{
			Scale: 0,
		},
	}

	syncCursorFields(spec)

	// Default scale should be 1.0
	if spec.Decor.CursorScale != 1.0 {
		t.Errorf("expected default decor cursor scale 1.0, got %f", spec.Decor.CursorScale)
	}
}

func TestSyncCursorFields_CursorMotionSync(t *testing.T) {
	spec := &export.ReplayMovieSpec{
		Cursor: services.ExportCursorSpec{
			InitialPos: "center",
			ClickAnim:  "ripple",
			Scale:      1.5,
		},
		Decor: services.ExportDecor{
			CursorInitial:        "center",
			CursorClickAnimation: "ripple",
			CursorScale:          1.5,
		},
		CursorMotion: services.ExportCursorMotion{
			InitialPosition: "",
			ClickAnimation:  "",
			CursorScale:     0,
		},
	}

	syncCursorFields(spec)

	// CursorMotion should be synced from Decor/Cursor
	if spec.CursorMotion.InitialPosition != "center" {
		t.Errorf("expected cursor motion initial position center, got %s", spec.CursorMotion.InitialPosition)
	}
	if spec.CursorMotion.ClickAnimation != "ripple" {
		t.Errorf("expected cursor motion click animation ripple, got %s", spec.CursorMotion.ClickAnimation)
	}
	if spec.CursorMotion.CursorScale != 1.5 {
		t.Errorf("expected cursor motion scale 1.5, got %f", spec.CursorMotion.CursorScale)
	}
}

func TestSyncCursorFields_DefaultInitialPosition(t *testing.T) {
	spec := &export.ReplayMovieSpec{
		CursorMotion: services.ExportCursorMotion{
			InitialPosition: "",
		},
	}

	syncCursorFields(spec)

	// Default initial position should be "center"
	if spec.CursorMotion.InitialPosition != "center" {
		t.Errorf("expected default initial position center, got %s", spec.CursorMotion.InitialPosition)
	}
}

func TestSyncCursorFields_NilSpec(t *testing.T) {
	// Should not panic with nil spec
	syncCursorFields(nil)
}
