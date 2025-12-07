package export

import (
	"testing"

	exportservices "github.com/vrooli/browser-automation-studio/services/export"
)

func TestBuildThemeFromPreset(t *testing.T) {
	t.Run("nil baseline", func(t *testing.T) {
		preset := &ThemePreset{ChromeTheme: "aurora"}
		result := BuildThemeFromPreset(nil, preset)
		if result != nil {
			t.Error("expected nil for nil baseline")
		}
	})

	t.Run("nil preset", func(t *testing.T) {
		baseline := &exportservices.ReplayMovieSpec{
			Theme: exportservices.ExportTheme{},
		}
		result := BuildThemeFromPreset(baseline, nil)
		if result != nil {
			t.Error("expected nil for nil preset")
		}
	})

	t.Run("aurora chrome theme", func(t *testing.T) {
		baseline := &exportservices.ReplayMovieSpec{
			Theme: exportservices.ExportTheme{
				BrowserChrome: exportservices.ExportBrowserChrome{},
			},
			Execution: exportservices.ExportExecutionMetadata{
				WorkflowName: "Test Workflow",
			},
		}
		preset := &ThemePreset{ChromeTheme: "aurora"}
		result := BuildThemeFromPreset(baseline, preset)

		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if !result.BrowserChrome.Visible {
			t.Error("aurora theme should be visible")
		}
		if result.BrowserChrome.Variant != "aurora" {
			t.Errorf("expected variant aurora, got %s", result.BrowserChrome.Variant)
		}
		if result.BrowserChrome.AccentColor != "#38BDF8" {
			t.Errorf("expected accent color #38BDF8, got %s", result.BrowserChrome.AccentColor)
		}
	})

	t.Run("minimal chrome theme", func(t *testing.T) {
		baseline := &exportservices.ReplayMovieSpec{
			Theme: exportservices.ExportTheme{
				BrowserChrome: exportservices.ExportBrowserChrome{},
			},
			Execution: exportservices.ExportExecutionMetadata{},
		}
		preset := &ThemePreset{ChromeTheme: "minimal"}
		result := BuildThemeFromPreset(baseline, preset)

		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.BrowserChrome.Visible {
			t.Error("minimal theme should not be visible")
		}
		if result.BrowserChrome.ShowAddress {
			t.Error("minimal theme should not show address")
		}
	})

	t.Run("background theme applied", func(t *testing.T) {
		baseline := &exportservices.ReplayMovieSpec{
			Theme: exportservices.ExportTheme{},
		}
		preset := &ThemePreset{BackgroundTheme: "midnight"}
		result := BuildThemeFromPreset(baseline, preset)

		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if len(result.BackgroundGradient) == 0 {
			t.Error("expected gradient to be set")
		}
		if result.SurfaceColor == "" {
			t.Error("expected surface color to be set")
		}
	})

	t.Run("workflow name used as title", func(t *testing.T) {
		baseline := &exportservices.ReplayMovieSpec{
			Theme: exportservices.ExportTheme{
				BrowserChrome: exportservices.ExportBrowserChrome{},
			},
			Execution: exportservices.ExportExecutionMetadata{
				WorkflowName: "My Workflow",
			},
		}
		preset := &ThemePreset{}
		result := BuildThemeFromPreset(baseline, preset)

		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.BrowserChrome.Title != "My Workflow" {
			t.Errorf("expected title 'My Workflow', got %s", result.BrowserChrome.Title)
		}
	})

	t.Run("default title when no workflow name", func(t *testing.T) {
		baseline := &exportservices.ReplayMovieSpec{
			Theme: exportservices.ExportTheme{
				BrowserChrome: exportservices.ExportBrowserChrome{},
			},
			Execution: exportservices.ExportExecutionMetadata{},
		}
		preset := &ThemePreset{}
		result := BuildThemeFromPreset(baseline, preset)

		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.BrowserChrome.Title != "Vrooli Ascension" {
			t.Errorf("expected default title, got %s", result.BrowserChrome.Title)
		}
	})

	t.Run("defaults applied when missing", func(t *testing.T) {
		baseline := &exportservices.ReplayMovieSpec{
			Theme: exportservices.ExportTheme{},
		}
		preset := &ThemePreset{}
		result := BuildThemeFromPreset(baseline, preset)

		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.AccentColor == "" {
			t.Error("expected default accent color")
		}
		if result.AmbientGlow == "" {
			t.Error("expected default ambient glow")
		}
		if result.SurfaceColor == "" {
			t.Error("expected default surface color")
		}
		if len(result.BackgroundGradient) == 0 {
			t.Error("expected default background gradient")
		}
	})

	t.Run("whitespace in theme names trimmed", func(t *testing.T) {
		baseline := &exportservices.ReplayMovieSpec{
			Theme: exportservices.ExportTheme{},
		}
		preset := &ThemePreset{
			ChromeTheme:     " aurora ",
			BackgroundTheme: " midnight ",
		}
		result := BuildThemeFromPreset(baseline, preset)

		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.BrowserChrome.Variant != "aurora" {
			t.Error("whitespace should be trimmed from chrome theme")
		}
	})

	t.Run("unknown preset names ignored", func(t *testing.T) {
		baseline := &exportservices.ReplayMovieSpec{
			Theme: exportservices.ExportTheme{},
		}
		preset := &ThemePreset{
			ChromeTheme:     "unknown",
			BackgroundTheme: "unknown",
		}
		result := BuildThemeFromPreset(baseline, preset)

		if result == nil {
			t.Fatal("expected non-nil result")
		}
		// Should still apply defaults
		if result.AccentColor == "" {
			t.Error("defaults should still be applied for unknown presets")
		}
	})
}

func TestBuildCursorSpec(t *testing.T) {
	t.Run("nil preset returns spec with defaults", func(t *testing.T) {
		existing := exportservices.ExportCursorSpec{}
		result := BuildCursorSpec(existing, nil)
		// Should return spec with defaults applied
		if result.Scale <= 0 {
			t.Error("expected default scale to be set")
		}
		if result.InitialPos == "" {
			t.Error("expected default initial position")
		}
	})

	t.Run("aura cursor theme", func(t *testing.T) {
		existing := exportservices.ExportCursorSpec{}
		preset := &CursorPreset{Theme: "aura"}
		result := BuildCursorSpec(existing, preset)

		if result.Style != "halo" {
			t.Errorf("expected style halo, got %s", result.Style)
		}
		if result.AccentColor == "" {
			t.Error("expected accent color to be set")
		}
	})

	t.Run("scale applied and clamped", func(t *testing.T) {
		existing := exportservices.ExportCursorSpec{}

		// Test normal scale
		preset := &CursorPreset{Theme: "aura", Scale: 1.5}
		result := BuildCursorSpec(existing, preset)
		if result.Scale != 1.5 {
			t.Errorf("expected scale 1.5, got %f", result.Scale)
		}

		// Test scale too low
		preset = &CursorPreset{Theme: "aura", Scale: 0.1}
		result = BuildCursorSpec(existing, preset)
		if result.Scale != CursorScaleMin {
			t.Errorf("expected scale clamped to %f, got %f", CursorScaleMin, result.Scale)
		}

		// Test scale too high
		preset = &CursorPreset{Theme: "aura", Scale: 10.0}
		result = BuildCursorSpec(existing, preset)
		if result.Scale != CursorScaleMax {
			t.Errorf("expected scale clamped to %f, got %f", CursorScaleMax, result.Scale)
		}
	})

	t.Run("initial position applied", func(t *testing.T) {
		existing := exportservices.ExportCursorSpec{}
		preset := &CursorPreset{
			Theme:           "aura",
			InitialPosition: "center",
		}
		result := BuildCursorSpec(existing, preset)

		if result.InitialPos != "center" {
			t.Errorf("expected initial position center, got %s", result.InitialPos)
		}
	})

	t.Run("click animation applied", func(t *testing.T) {
		existing := exportservices.ExportCursorSpec{}
		preset := &CursorPreset{
			Theme:          "aura",
			ClickAnimation: "ripple",
		}
		result := BuildCursorSpec(existing, preset)

		if result.ClickAnim != "ripple" {
			t.Errorf("expected click animation ripple, got %s", result.ClickAnim)
		}
	})

	t.Run("defaults applied when missing", func(t *testing.T) {
		existing := exportservices.ExportCursorSpec{}
		preset := &CursorPreset{Theme: "aura"}
		result := BuildCursorSpec(existing, preset)

		if result.Scale == 0 {
			t.Error("expected default scale to be set")
		}
	})

	t.Run("unknown theme name ignored", func(t *testing.T) {
		existing := exportservices.ExportCursorSpec{}
		preset := &CursorPreset{Theme: "unknown"}
		result := BuildCursorSpec(existing, preset)

		// Should still apply defaults
		if result.Scale == 0 {
			t.Error("defaults should still be applied for unknown themes")
		}
	})
}
