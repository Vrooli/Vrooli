package export

import (
	"testing"

	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
)

func TestApplyUsesGeneratedReplaySpecForPresetsAndOverrides(t *testing.T) {
	spec := &exportsv1.ReplaySpec{
		Execution: &exportsv1.ReplayExecutionMetadata{WorkflowName: "Checkout"},
		Theme:     &exportsv1.ReplayTheme{AccentColor: "#111111", BrowserChrome: &exportsv1.ReplayBrowserChrome{}},
		Cursor:    &exportsv1.ReplayCursor{Trail: &exportsv1.ReplayCursorTrail{}, ClickPulse: &exportsv1.ReplayClickPulse{}},
		Decor:     &exportsv1.ReplayDecor{}, CursorMotion: &exportsv1.ReplayCursorMotion{},
	}
	Apply(spec, &Overrides{
		ThemePreset:  &ThemePreset{ChromeTheme: "midnight", BackgroundTheme: "sunset"},
		CursorPreset: &CursorPreset{Theme: "aura", InitialPosition: "top-left", Scale: 1.8, ClickAnimation: "ripple"},
	})
	if spec.GetTheme().GetAccentColor() != "#A855F7" || spec.GetTheme().GetBrowserChrome().GetVariant() != "midnight" {
		t.Fatalf("generated theme preset/override was not applied: %v", spec.GetTheme())
	}
	if spec.GetTheme().GetBackgroundPattern() != "sunset" || len(spec.GetTheme().GetBackgroundGradient()) == 0 {
		t.Fatalf("generated background preset was not applied: %v", spec.GetTheme())
	}
	if spec.GetCursor().GetStyle() != "halo" || spec.GetCursor().GetInitialPosition() != "top-left" || spec.GetCursor().GetClickAnimation() != "ripple" || spec.GetCursor().GetScale() != 1.8 {
		t.Fatalf("generated cursor preset was not applied: %v", spec.GetCursor())
	}
	if spec.GetCursorMotion().GetInitialPosition() != "top-left" || spec.GetDecor().GetCursorTheme() != "aura" {
		t.Fatalf("generated cursor fields were not synchronized: decor=%v motion=%v", spec.GetDecor(), spec.GetCursorMotion())
	}
	Apply(spec, &Overrides{Theme: &exportsv1.ReplayTheme{AccentColor: "#ff0000"}})
	if spec.GetTheme().GetAccentColor() != "#ff0000" {
		t.Fatalf("explicit generated theme override did not take precedence: %v", spec.GetTheme())
	}
}

func TestApplyDecorOverrides_ThemePresetNames(t *testing.T) {
	spec := &exportsv1.ReplaySpec{}

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
	spec := &exportsv1.ReplaySpec{}

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
	if spec.Decor.CursorInitialPosition != "center" {
		t.Errorf("expected cursor initial center, got %s", spec.Decor.CursorInitialPosition)
	}
	if spec.Decor.CursorClickAnimation != "ripple" {
		t.Errorf("expected click animation ripple, got %s", spec.Decor.CursorClickAnimation)
	}
	if spec.Decor.CursorScale != 1.5 {
		t.Errorf("expected cursor scale 1.5, got %f", spec.Decor.CursorScale)
	}
}

func TestApplyDecorOverrides_CursorScaleClamping(t *testing.T) {
	spec := &exportsv1.ReplaySpec{}

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
	spec := &exportsv1.ReplaySpec{}

	overrides := &Overrides{
		Cursor: &exportsv1.ReplayCursor{
			InitialPosition: "top-right",
			ClickAnimation:  "bounce",
			Scale:           1.8,
		},
	}

	applyDecorOverrides(spec, overrides)

	if spec.Decor.CursorInitialPosition != "top-right" {
		t.Errorf("expected cursor initial top-right, got %s", spec.Decor.CursorInitialPosition)
	}
	if spec.Decor.CursorClickAnimation != "bounce" {
		t.Errorf("expected click animation bounce, got %s", spec.Decor.CursorClickAnimation)
	}
	if spec.Decor.CursorScale != 1.8 {
		t.Errorf("expected cursor scale 1.8, got %f", spec.Decor.CursorScale)
	}
}

func TestSyncCursorFields_InitialPosition(t *testing.T) {
	spec := &exportsv1.ReplaySpec{
		Cursor: &exportsv1.ReplayCursor{
			InitialPosition: "bottom-left",
		},
		Decor: &exportsv1.ReplayDecor{
			CursorInitialPosition: "",
		},
	}

	syncCursorFields(spec)

	// Decor should be synced from Cursor
	if spec.Decor.CursorInitialPosition != "bottom-left" {
		t.Errorf("expected decor cursor initial to sync from cursor, got %s", spec.Decor.CursorInitialPosition)
	}
}

func TestSyncCursorFields_InitialPositionReverse(t *testing.T) {
	spec := &exportsv1.ReplaySpec{
		Cursor: &exportsv1.ReplayCursor{
			InitialPosition: "",
		},
		Decor: &exportsv1.ReplayDecor{
			CursorInitialPosition: "top-center",
		},
	}

	syncCursorFields(spec)

	// Cursor should be synced from Decor
	if spec.Cursor.InitialPosition != "top-center" {
		t.Errorf("expected cursor initial pos to sync from decor, got %s", spec.Cursor.InitialPosition)
	}
}

func TestSyncCursorFields_ClickAnimation(t *testing.T) {
	spec := &exportsv1.ReplaySpec{
		Cursor: &exportsv1.ReplayCursor{
			ClickAnimation: "pulse",
		},
		Decor: &exportsv1.ReplayDecor{
			CursorClickAnimation: "",
		},
	}

	syncCursorFields(spec)

	if spec.Decor.CursorClickAnimation != "pulse" {
		t.Errorf("expected decor click animation to sync from cursor, got %s", spec.Decor.CursorClickAnimation)
	}
}

func TestSyncCursorFields_ScaleClamping(t *testing.T) {
	spec := &exportsv1.ReplaySpec{
		Cursor: &exportsv1.ReplayCursor{
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
	spec := &exportsv1.ReplaySpec{
		Decor: &exportsv1.ReplayDecor{
			CursorScale: 0,
		},
		Cursor: &exportsv1.ReplayCursor{
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
	spec := &exportsv1.ReplaySpec{
		Cursor: &exportsv1.ReplayCursor{
			InitialPosition: "center",
			ClickAnimation:  "ripple",
			Scale:           1.5,
		},
		Decor: &exportsv1.ReplayDecor{
			CursorInitialPosition: "center",
			CursorClickAnimation:  "ripple",
			CursorScale:           1.5,
		},
		CursorMotion: &exportsv1.ReplayCursorMotion{
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
	spec := &exportsv1.ReplaySpec{
		CursorMotion: &exportsv1.ReplayCursorMotion{
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
