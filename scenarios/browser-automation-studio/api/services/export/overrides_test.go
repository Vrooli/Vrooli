package export

import (
	"testing"
)

func TestApplyDecorOverrides_ThemePresetNames(t *testing.T) {
	spec := &ReplayMovieSpec{}

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
	spec := &ReplayMovieSpec{}

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
	spec := &ReplayMovieSpec{}

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
	spec := &ReplayMovieSpec{}

	overrides := &Overrides{
		Cursor: &ExportCursorSpec{
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
	spec := &ReplayMovieSpec{
		Cursor: ExportCursorSpec{
			InitialPos: "bottom-left",
		},
		Decor: ExportDecor{
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
	spec := &ReplayMovieSpec{
		Cursor: ExportCursorSpec{
			InitialPos: "",
		},
		Decor: ExportDecor{
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
	spec := &ReplayMovieSpec{
		Cursor: ExportCursorSpec{
			ClickAnim: "pulse",
		},
		Decor: ExportDecor{
			CursorClickAnimation: "",
		},
	}

	syncCursorFields(spec)

	if spec.Decor.CursorClickAnimation != "pulse" {
		t.Errorf("expected decor click animation to sync from cursor, got %s", spec.Decor.CursorClickAnimation)
	}
}

func TestSyncCursorFields_ScaleClamping(t *testing.T) {
	spec := &ReplayMovieSpec{
		Cursor: ExportCursorSpec{
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
	spec := &ReplayMovieSpec{
		Decor: ExportDecor{
			CursorScale: 0,
		},
		Cursor: ExportCursorSpec{
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
	spec := &ReplayMovieSpec{
		Cursor: ExportCursorSpec{
			InitialPos: "center",
			ClickAnim:  "ripple",
			Scale:      1.5,
		},
		Decor: ExportDecor{
			CursorInitial:        "center",
			CursorClickAnimation: "ripple",
			CursorScale:          1.5,
		},
		CursorMotion: ExportCursorMotion{
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
	spec := &ReplayMovieSpec{
		CursorMotion: ExportCursorMotion{
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
