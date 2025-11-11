package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services"
)

type executionExportRequest struct {
	Format    string                    `json:"format,omitempty"`
	FileName  string                    `json:"file_name,omitempty"`
	Overrides *executionExportOverrides `json:"overrides,omitempty"`
	MovieSpec *services.ReplayMovieSpec `json:"movie_spec,omitempty"`
}

type executionExportOverrides struct {
	Theme        *services.ExportTheme      `json:"theme,omitempty"`
	Cursor       *services.ExportCursorSpec `json:"cursor,omitempty"`
	ThemePreset  *themePresetOverride       `json:"theme_preset,omitempty"`
	CursorPreset *cursorPresetOverride      `json:"cursor_preset,omitempty"`
}

type themePresetOverride struct {
	ChromeTheme     string `json:"chrome_theme,omitempty"`
	BackgroundTheme string `json:"background_theme,omitempty"`
}

type cursorPresetOverride struct {
	Theme           string  `json:"theme,omitempty"`
	InitialPosition string  `json:"initial_position,omitempty"`
	Scale           float64 `json:"scale,omitempty"`
	ClickAnimation  string  `json:"click_animation,omitempty"`
}

type chromeThemePreset struct {
	Visible     bool
	Variant     string
	ShowAddress bool
	AccentColor string
	AmbientGlow string
}

type backgroundThemePreset struct {
	Gradient    []string
	Pattern     string
	Surface     string
	AmbientGlow string
}

type cursorThemePreset struct {
	Style        string
	AccentColor  string
	TrailEnabled bool
	TrailOpacity float64
	TrailFadeMs  int
	TrailWeight  float64
	ClickEnabled bool
	ClickOpacity float64
}

var chromeThemePresets = map[string]chromeThemePreset{
	"aurora": {
		Visible:     true,
		Variant:     "aurora",
		ShowAddress: true,
		AccentColor: "#38BDF8",
		AmbientGlow: "rgba(56,189,248,0.28)",
	},
	"chromium": {
		Visible:     true,
		Variant:     "chromium",
		ShowAddress: true,
		AccentColor: "#0EA5E9",
		AmbientGlow: "rgba(59,130,246,0.22)",
	},
	"midnight": {
		Visible:     true,
		Variant:     "midnight",
		ShowAddress: true,
		AccentColor: "#A855F7",
		AmbientGlow: "rgba(168,85,247,0.26)",
	},
	"minimal": {
		Visible:     false,
		Variant:     "minimal",
		ShowAddress: false,
		AccentColor: "#94A3B8",
		AmbientGlow: "rgba(148,163,184,0.18)",
	},
}

var backgroundThemePresets = map[string]backgroundThemePreset{
	"aurora": {
		Gradient:    []string{"#0F172A", "#1D4ED8", "#38BDF8"},
		Pattern:     "aurora",
		Surface:     "rgba(15,23,42,0.78)",
		AmbientGlow: "rgba(56,189,248,0.24)",
	},
	"sunset": {
		Gradient:    []string{"#F472B6", "#FBBF24", "#43112D"},
		Pattern:     "sunset",
		Surface:     "rgba(43,12,30,0.85)",
		AmbientGlow: "rgba(244,114,182,0.24)",
	},
	"ocean": {
		Gradient:    []string{"#0EA5E9", "#1D4ED8", "#020617"},
		Pattern:     "ocean",
		Surface:     "rgba(8,31,60,0.82)",
		AmbientGlow: "rgba(14,116,144,0.26)",
	},
	"nebula": {
		Gradient:    []string{"#A855F7", "#6366F1", "#111827"},
		Pattern:     "nebula",
		Surface:     "rgba(49,16,80,0.82)",
		AmbientGlow: "rgba(147,51,234,0.26)",
	},
	"grid": {
		Gradient:    []string{"#0F172A", "#0B1120"},
		Pattern:     "grid",
		Surface:     "rgba(8,29,57,0.9)",
		AmbientGlow: "rgba(14,165,233,0.18)",
	},
	"charcoal": {
		Gradient:    []string{"#0F172A", "#020617"},
		Pattern:     "charcoal",
		Surface:     "rgba(15,23,42,0.92)",
		AmbientGlow: "rgba(148,163,184,0.18)",
	},
	"steel": {
		Gradient:    []string{"#1F2937", "#0B1120"},
		Pattern:     "steel",
		Surface:     "rgba(30,41,59,0.9)",
		AmbientGlow: "rgba(148,163,184,0.22)",
	},
	"emerald": {
		Gradient:    []string{"#10B981", "#064E3B", "#022C22"},
		Pattern:     "emerald",
		Surface:     "rgba(2,44,34,0.88)",
		AmbientGlow: "rgba(16,185,129,0.22)",
	},
	"none": {
		Gradient:    []string{"#0F172A"},
		Pattern:     "none",
		Surface:     "rgba(15,23,42,0.85)",
		AmbientGlow: "rgba(59,130,246,0.18)",
	},
	"geoPrism": {
		Gradient:    []string{"#0F172A", "#1E293B", "#0EA5E9"},
		Pattern:     "geo-prism",
		Surface:     "rgba(9,22,39,0.9)",
		AmbientGlow: "rgba(14,165,233,0.24)",
	},
	"geoOrbit": {
		Gradient:    []string{"#0B1120", "#1E293B", "#F59E0B"},
		Pattern:     "geo-orbit",
		Surface:     "rgba(10,18,34,0.92)",
		AmbientGlow: "rgba(245,158,11,0.22)",
	},
	"geoMosaic": {
		Gradient:    []string{"#0B1526", "#1E3A8A", "#38BDF8"},
		Pattern:     "geo-mosaic",
		Surface:     "rgba(11,21,38,0.92)",
		AmbientGlow: "rgba(56,189,248,0.24)",
	},
}

var cursorThemePresets = map[string]cursorThemePreset{
	"disabled": {
		Style:        "hidden",
		AccentColor:  "#38BDF8",
		TrailEnabled: false,
		ClickEnabled: false,
	},
	"white": {
		Style:        "halo",
		AccentColor:  "#FFFFFF",
		TrailEnabled: true,
		TrailOpacity: 0.55,
		TrailFadeMs:  700,
		TrailWeight:  0.18,
		ClickEnabled: true,
		ClickOpacity: 0.65,
	},
	"black": {
		Style:        "halo",
		AccentColor:  "#0F172A",
		TrailEnabled: true,
		TrailOpacity: 0.5,
		TrailFadeMs:  700,
		TrailWeight:  0.18,
		ClickEnabled: true,
		ClickOpacity: 0.6,
	},
	"aura": {
		Style:        "halo",
		AccentColor:  "#38BDF8",
		TrailEnabled: true,
		TrailOpacity: 0.6,
		TrailFadeMs:  800,
		TrailWeight:  0.22,
		ClickEnabled: true,
		ClickOpacity: 0.7,
	},
	"arrowLight": {
		Style:        "arrow-light",
		AccentColor:  "#FFFFFF",
		TrailEnabled: false,
		ClickEnabled: true,
		ClickOpacity: 0.55,
	},
	"arrowDark": {
		Style:        "arrow-dark",
		AccentColor:  "#1E293B",
		TrailEnabled: false,
		ClickEnabled: true,
		ClickOpacity: 0.55,
	},
	"arrowNeon": {
		Style:        "arrow-neon",
		AccentColor:  "#38BDF8",
		TrailEnabled: true,
		TrailOpacity: 0.4,
		TrailFadeMs:  600,
		TrailWeight:  0.14,
		ClickEnabled: true,
		ClickOpacity: 0.6,
	},
	"handNeutral": {
		Style:        "hand-neutral",
		AccentColor:  "#F1F5F9",
		TrailEnabled: false,
		ClickEnabled: true,
		ClickOpacity: 0.55,
	},
	"handAura": {
		Style:        "hand-aura",
		AccentColor:  "#38BDF8",
		TrailEnabled: true,
		TrailOpacity: 0.45,
		TrailFadeMs:  720,
		TrailWeight:  0.18,
		ClickEnabled: true,
		ClickOpacity: 0.65,
	},
}

func applyExportOverrides(pkg *services.ReplayMovieSpec, overrides *executionExportOverrides) {
	if pkg == nil || overrides == nil {
		return
	}
	if overrides.ThemePreset != nil {
		if theme := buildThemeFromPreset(pkg, overrides.ThemePreset); theme != nil {
			pkg.Theme = *theme
		}
	}
	if overrides.Theme != nil {
		pkg.Theme = *overrides.Theme
	}
	if overrides.CursorPreset != nil {
		pkg.Cursor = applyCursorPreset(pkg.Cursor, overrides.CursorPreset)
	}
	if overrides.Cursor != nil {
		pkg.Cursor = *overrides.Cursor
	}

	applyDecorOverrides(pkg, overrides)
	syncCursorDecor(pkg)
}

func applyDecorOverrides(pkg *services.ReplayMovieSpec, overrides *executionExportOverrides) {
	if pkg == nil || overrides == nil {
		return
	}

	if preset := overrides.ThemePreset; preset != nil {
		if chrome := strings.TrimSpace(preset.ChromeTheme); chrome != "" {
			pkg.Decor.ChromeTheme = chrome
		}
		if background := strings.TrimSpace(preset.BackgroundTheme); background != "" {
			pkg.Decor.BackgroundTheme = background
		}
	}

	if cursorPreset := overrides.CursorPreset; cursorPreset != nil {
		if theme := strings.TrimSpace(cursorPreset.Theme); theme != "" {
			pkg.Decor.CursorTheme = theme
		}
		if initial := strings.TrimSpace(cursorPreset.InitialPosition); initial != "" {
			pkg.Decor.CursorInitial = initial
		}
		if anim := strings.TrimSpace(cursorPreset.ClickAnimation); anim != "" {
			pkg.Decor.CursorClickAnimation = anim
		}
		if cursorPreset.Scale > 0 {
			pkg.Decor.CursorScale = clampCursorScale(cursorPreset.Scale)
		}
	}

	if cursorOverride := overrides.Cursor; cursorOverride != nil {
		if strings.TrimSpace(cursorOverride.InitialPos) != "" {
			pkg.Decor.CursorInitial = cursorOverride.InitialPos
		}
		if strings.TrimSpace(cursorOverride.ClickAnim) != "" {
			pkg.Decor.CursorClickAnimation = cursorOverride.ClickAnim
		}
		if cursorOverride.Scale > 0 {
			pkg.Decor.CursorScale = clampCursorScale(cursorOverride.Scale)
		}
	}
}

func syncCursorDecor(pkg *services.ReplayMovieSpec) {
	if pkg == nil {
		return
	}

	if strings.TrimSpace(pkg.Decor.CursorInitial) == "" {
		pkg.Decor.CursorInitial = strings.TrimSpace(pkg.Cursor.InitialPos)
	}
	if strings.TrimSpace(pkg.Decor.CursorClickAnimation) == "" {
		pkg.Decor.CursorClickAnimation = strings.TrimSpace(pkg.Cursor.ClickAnim)
	}

	if strings.TrimSpace(pkg.Cursor.InitialPos) == "" {
		pkg.Cursor.InitialPos = pkg.Decor.CursorInitial
	}
	if strings.TrimSpace(pkg.Cursor.ClickAnim) == "" {
		pkg.Cursor.ClickAnim = pkg.Decor.CursorClickAnimation
	}

	if pkg.Decor.CursorScale <= 0 {
		pkg.Decor.CursorScale = pkg.Cursor.Scale
	}
	if pkg.Decor.CursorScale <= 0 {
		pkg.Decor.CursorScale = 1
	}

	if pkg.Decor.CursorScale > 0 {
		clamped := clampCursorScale(pkg.Decor.CursorScale)
		pkg.Decor.CursorScale = clamped
		if pkg.Cursor.Scale <= 0 {
			pkg.Cursor.Scale = clamped
		}
	}

	if pkg.Cursor.Scale > 0 {
		pkg.Cursor.Scale = clampCursorScale(pkg.Cursor.Scale)
	}

	if strings.TrimSpace(pkg.CursorMotion.InitialPosition) == "" {
		pkg.CursorMotion.InitialPosition = pkg.Decor.CursorInitial
	}
	if strings.TrimSpace(pkg.CursorMotion.ClickAnimation) == "" {
		pkg.CursorMotion.ClickAnimation = pkg.Decor.CursorClickAnimation
	}
	if pkg.Decor.CursorScale > 0 {
		pkg.CursorMotion.CursorScale = clampCursorScale(pkg.Decor.CursorScale)
	} else if pkg.Cursor.Scale > 0 {
		pkg.CursorMotion.CursorScale = clampCursorScale(pkg.Cursor.Scale)
	}

	if pkg.CursorMotion.InitialPosition == "" {
		pkg.CursorMotion.InitialPosition = "center"
	}
	if pkg.CursorMotion.ClickAnimation == "" {
		pkg.CursorMotion.ClickAnimation = pkg.Cursor.ClickAnim
	}
	if pkg.CursorMotion.CursorScale <= 0 {
		pkg.CursorMotion.CursorScale = clampCursorScale(pkg.Cursor.Scale)
	}

	if pkg.Decor.CursorScale <= 0 {
		pkg.Decor.CursorScale = pkg.CursorMotion.CursorScale
	}
}

const defaultAccentColor = "#38BDF8"

func buildThemeFromPreset(pkg *services.ReplayMovieSpec, preset *themePresetOverride) *services.ExportTheme {
	if pkg == nil || preset == nil {
		return nil
	}

	theme := pkg.Theme

	if chromeID := strings.TrimSpace(preset.ChromeTheme); chromeID != "" {
		if chrome, ok := chromeThemePresets[chromeID]; ok {
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

	if backgroundID := strings.TrimSpace(preset.BackgroundTheme); backgroundID != "" {
		if background, ok := backgroundThemePresets[backgroundID]; ok {
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

	if theme.BrowserChrome.Title == "" {
		if name := strings.TrimSpace(pkg.Execution.WorkflowName); name != "" {
			theme.BrowserChrome.Title = name
		} else {
			theme.BrowserChrome.Title = "Browser Automation Studio"
		}
	}
	if theme.AccentColor == "" {
		theme.AccentColor = defaultAccentColor
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

func applyCursorPreset(existing services.ExportCursorSpec, preset *cursorPresetOverride) services.ExportCursorSpec {
	cursor := existing

	if cursor.Scale <= 0 {
		cursor.Scale = 1.0
	}
	if cursor.InitialPos == "" {
		cursor.InitialPos = "center"
	}
	if cursor.ClickAnim == "" {
		cursor.ClickAnim = "pulse"
	}

	if preset != nil {
		if themeKey := strings.TrimSpace(preset.Theme); themeKey != "" {
			if cfg, ok := cursorThemePresets[themeKey]; ok {
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
		if preset.Scale > 0 {
			cursor.Scale = clampCursorScale(preset.Scale)
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

	if cursor.AccentColor == "" {
		cursor.AccentColor = defaultAccentColor
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
	if strings.EqualFold(cursor.Style, "hidden") {
		cursor.Trail.Enabled = false
		cursor.ClickPulse.Enabled = false
	}

	return cursor
}

var errMovieSpecUnavailable = errors.New("movie spec unavailable")

func buildExportSpec(baseline, incoming *services.ReplayMovieSpec, executionID uuid.UUID) (*services.ReplayMovieSpec, error) {
	if incoming == nil && baseline == nil {
		return nil, errMovieSpecUnavailable
	}

	var (
		spec *services.ReplayMovieSpec
		err  error
	)

	if incoming != nil {
		spec, err = cloneMovieSpec(incoming)
		if err != nil {
			return nil, fmt.Errorf("invalid movie spec: %w", err)
		}
	} else {
		spec, err = cloneMovieSpec(baseline)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare movie spec: %w", err)
		}
	}

	if err := harmonizeMovieSpec(spec, baseline, executionID); err != nil {
		return nil, err
	}
	return spec, nil
}

func cloneMovieSpec(spec *services.ReplayMovieSpec) (*services.ReplayMovieSpec, error) {
	if spec == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	var cloned services.ReplayMovieSpec
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		return nil, err
	}
	return &cloned, nil
}

func harmonizeMovieSpec(spec, baseline *services.ReplayMovieSpec, executionID uuid.UUID) error {
	if spec == nil {
		return errMovieSpecUnavailable
	}

	if spec.Execution.ExecutionID == uuid.Nil {
		spec.Execution.ExecutionID = executionID
	} else if spec.Execution.ExecutionID != executionID {
		return fmt.Errorf("movie spec execution_id mismatch")
	}

	if baseline != nil {
		if baseline.Execution.WorkflowID != uuid.Nil {
			if spec.Execution.WorkflowID == uuid.Nil {
				spec.Execution.WorkflowID = baseline.Execution.WorkflowID
			} else if spec.Execution.WorkflowID != baseline.Execution.WorkflowID {
				return fmt.Errorf("movie spec workflow_id mismatch")
			}
		}
		if spec.Execution.WorkflowName == "" {
			spec.Execution.WorkflowName = baseline.Execution.WorkflowName
		}
		if spec.Execution.Status == "" {
			spec.Execution.Status = baseline.Execution.Status
		}
		if spec.Execution.Progress == 0 {
			spec.Execution.Progress = baseline.Execution.Progress
		}
		if spec.Execution.StartedAt.IsZero() && !baseline.Execution.StartedAt.IsZero() {
			spec.Execution.StartedAt = baseline.Execution.StartedAt
		}
		if spec.Execution.CompletedAt == nil && baseline.Execution.CompletedAt != nil {
			t := *baseline.Execution.CompletedAt
			spec.Execution.CompletedAt = &t
		}
	}

	if spec.Version == "" && baseline != nil {
		spec.Version = baseline.Version
	}
	if spec.Version == "" {
		spec.Version = "2025-11-07"
	}
	if spec.GeneratedAt.IsZero() {
		if baseline != nil && !baseline.GeneratedAt.IsZero() {
			spec.GeneratedAt = baseline.GeneratedAt
		} else {
			spec.GeneratedAt = time.Now().UTC()
		}
	}

	if len(spec.Frames) == 0 {
		if baseline != nil && len(baseline.Frames) > 0 {
			spec.Frames = baseline.Frames
		} else {
			return fmt.Errorf("movie spec missing frames")
		}
	}

	ensureTheme(spec, baseline)
	ensureDecor(spec, baseline)
	ensureCursor(spec, baseline)
	ensurePresentation(spec, baseline)
	ensureSummaryAndPlayback(spec, baseline)

	if len(spec.Assets) == 0 && baseline != nil {
		spec.Assets = baseline.Assets
	}

	if spec.Execution.TotalDuration <= 0 {
		spec.Execution.TotalDuration = spec.Summary.TotalDurationMs
	}

	return nil
}

func ensureTheme(spec, baseline *services.ReplayMovieSpec) {
	if baseline != nil {
		if len(spec.Theme.BackgroundGradient) == 0 {
			spec.Theme.BackgroundGradient = baseline.Theme.BackgroundGradient
		}
		if spec.Theme.BackgroundPattern == "" {
			spec.Theme.BackgroundPattern = baseline.Theme.BackgroundPattern
		}
		if spec.Theme.AccentColor == "" {
			spec.Theme.AccentColor = baseline.Theme.AccentColor
		}
		if spec.Theme.SurfaceColor == "" {
			spec.Theme.SurfaceColor = baseline.Theme.SurfaceColor
		}
		if spec.Theme.AmbientGlow == "" {
			spec.Theme.AmbientGlow = baseline.Theme.AmbientGlow
		}
		if spec.Theme.BrowserChrome.Variant == "" && baseline.Theme.BrowserChrome.Variant != "" {
			spec.Theme.BrowserChrome = baseline.Theme.BrowserChrome
		} else {
			if spec.Theme.BrowserChrome.AccentColor == "" {
				spec.Theme.BrowserChrome.AccentColor = spec.Theme.AccentColor
			}
		}
		return
	}

	if len(spec.Theme.BackgroundGradient) == 0 {
		spec.Theme.BackgroundGradient = []string{"#0f172a", "#020617", "#111827"}
	}
	if spec.Theme.BackgroundPattern == "" {
		spec.Theme.BackgroundPattern = "orbits"
	}
	if spec.Theme.AccentColor == "" {
		spec.Theme.AccentColor = "#38BDF8"
	}
	if spec.Theme.SurfaceColor == "" {
		spec.Theme.SurfaceColor = "rgba(15,23,42,0.72)"
	}
	if spec.Theme.AmbientGlow == "" {
		spec.Theme.AmbientGlow = "rgba(56,189,248,0.22)"
	}
	if spec.Theme.BrowserChrome.Variant == "" {
		spec.Theme.BrowserChrome = services.ExportBrowserChrome{
			Visible:     true,
			Variant:     "dark",
			Title:       spec.Execution.WorkflowName,
			ShowAddress: true,
			AccentColor: spec.Theme.AccentColor,
		}
	} else if spec.Theme.BrowserChrome.AccentColor == "" {
		spec.Theme.BrowserChrome.AccentColor = spec.Theme.AccentColor
	}
}

func ensureDecor(spec, baseline *services.ReplayMovieSpec) {
	if baseline != nil {
		if spec.Decor.ChromeTheme == "" {
			spec.Decor.ChromeTheme = baseline.Decor.ChromeTheme
		}
		if spec.Decor.BackgroundTheme == "" {
			spec.Decor.BackgroundTheme = baseline.Decor.BackgroundTheme
		}
		if spec.Decor.CursorTheme == "" {
			spec.Decor.CursorTheme = baseline.Decor.CursorTheme
		}
		if spec.Decor.CursorInitial == "" {
			spec.Decor.CursorInitial = baseline.Decor.CursorInitial
		}
		if spec.Decor.CursorClickAnimation == "" {
			spec.Decor.CursorClickAnimation = baseline.Decor.CursorClickAnimation
		}
		if spec.Decor.CursorScale == 0 {
			spec.Decor.CursorScale = baseline.Decor.CursorScale
		}
		return
	}

	if spec.Decor.ChromeTheme == "" {
		spec.Decor.ChromeTheme = "aurora"
	}
	if spec.Decor.BackgroundTheme == "" {
		spec.Decor.BackgroundTheme = "aurora"
	}
	if spec.Decor.CursorTheme == "" {
		spec.Decor.CursorTheme = "white"
	}
	if spec.Decor.CursorInitial == "" {
		spec.Decor.CursorInitial = "center"
	}
	if spec.Decor.CursorClickAnimation == "" {
		spec.Decor.CursorClickAnimation = "pulse"
	}
	if spec.Decor.CursorScale == 0 {
		spec.Decor.CursorScale = 1
	}
}

func ensureCursor(spec, baseline *services.ReplayMovieSpec) {
	if baseline != nil {
		if spec.Cursor.Style == "" {
			spec.Cursor = baseline.Cursor
		} else {
			if spec.Cursor.AccentColor == "" {
				spec.Cursor.AccentColor = baseline.Cursor.AccentColor
			}
			if spec.Cursor.InitialPos == "" {
				spec.Cursor.InitialPos = baseline.Cursor.InitialPos
			}
			if spec.Cursor.ClickAnim == "" {
				spec.Cursor.ClickAnim = baseline.Cursor.ClickAnim
			}
			if spec.Cursor.Trail.FadeMs == 0 {
				spec.Cursor.Trail.FadeMs = baseline.Cursor.Trail.FadeMs
			}
			if spec.Cursor.Trail.Weight == 0 {
				spec.Cursor.Trail.Weight = baseline.Cursor.Trail.Weight
			}
			if spec.Cursor.Trail.Opacity == 0 {
				spec.Cursor.Trail.Opacity = baseline.Cursor.Trail.Opacity
			}
			if spec.Cursor.ClickPulse.Radius == 0 {
				spec.Cursor.ClickPulse.Radius = baseline.Cursor.ClickPulse.Radius
			}
			if spec.Cursor.ClickPulse.DurationMs == 0 {
				spec.Cursor.ClickPulse.DurationMs = baseline.Cursor.ClickPulse.DurationMs
			}
			if spec.Cursor.ClickPulse.Opacity == 0 {
				spec.Cursor.ClickPulse.Opacity = baseline.Cursor.ClickPulse.Opacity
			}
			if !spec.Cursor.ClickPulse.Enabled && baseline.Cursor.ClickPulse.Enabled {
				spec.Cursor.ClickPulse.Enabled = true
			}
			if !spec.Cursor.Trail.Enabled && baseline.Cursor.Trail.Enabled {
				spec.Cursor.Trail.Enabled = true
			}
		}
	} else {
		if spec.Cursor.Style == "" {
			spec.Cursor.Style = "halo"
		}
		if spec.Cursor.AccentColor == "" {
			spec.Cursor.AccentColor = "#38BDF8"
		}
		if spec.Cursor.InitialPos == "" {
			spec.Cursor.InitialPos = "center"
		}
		if spec.Cursor.ClickAnim == "" {
			spec.Cursor.ClickAnim = "pulse"
		}
		if spec.Cursor.Trail.FadeMs == 0 {
			spec.Cursor.Trail.FadeMs = 650
		}
		if spec.Cursor.Trail.Weight == 0 {
			spec.Cursor.Trail.Weight = 0.16
		}
		if spec.Cursor.Trail.Opacity == 0 {
			spec.Cursor.Trail.Opacity = 0.55
		}
		if spec.Cursor.ClickPulse.Radius == 0 {
			spec.Cursor.ClickPulse.Radius = 42
		}
		if spec.Cursor.ClickPulse.DurationMs == 0 {
			spec.Cursor.ClickPulse.DurationMs = 420
		}
		if spec.Cursor.ClickPulse.Opacity == 0 {
			spec.Cursor.ClickPulse.Opacity = 0.65
		}
		if !spec.Cursor.ClickPulse.Enabled && !strings.EqualFold(spec.Cursor.Style, "hidden") {
			spec.Cursor.ClickPulse.Enabled = true
		}
		if !spec.Cursor.Trail.Enabled && !strings.EqualFold(spec.Cursor.Style, "hidden") {
			spec.Cursor.Trail.Enabled = true
		}
	}

	spec.Cursor.Scale = clampCursorScale(spec.Cursor.Scale)

	if spec.CursorMotion.SpeedProfile == "" && baseline != nil {
		spec.CursorMotion.SpeedProfile = baseline.CursorMotion.SpeedProfile
	}
	if spec.CursorMotion.SpeedProfile == "" {
		spec.CursorMotion.SpeedProfile = "easeInOut"
	}
	if spec.CursorMotion.PathStyle == "" && baseline != nil {
		spec.CursorMotion.PathStyle = baseline.CursorMotion.PathStyle
	}
	if spec.CursorMotion.PathStyle == "" {
		spec.CursorMotion.PathStyle = "linear"
	}
	if spec.CursorMotion.InitialPosition == "" {
		spec.CursorMotion.InitialPosition = spec.Cursor.InitialPos
	}
	if spec.CursorMotion.ClickAnimation == "" {
		spec.CursorMotion.ClickAnimation = spec.Cursor.ClickAnim
	}
	if spec.CursorMotion.CursorScale <= 0 {
		spec.CursorMotion.CursorScale = spec.Cursor.Scale
	}
}

func ensurePresentation(spec, baseline *services.ReplayMovieSpec) {
	if baseline != nil {
		if spec.Presentation.Canvas.Width == 0 {
			spec.Presentation.Canvas = baseline.Presentation.Canvas
		}
		if spec.Presentation.Viewport.Width == 0 {
			spec.Presentation.Viewport = baseline.Presentation.Viewport
		}
		if spec.Presentation.BrowserFrame.Width == 0 {
			spec.Presentation.BrowserFrame = baseline.Presentation.BrowserFrame
		}
		if spec.Presentation.DeviceScaleFactor == 0 {
			spec.Presentation.DeviceScaleFactor = baseline.Presentation.DeviceScaleFactor
		}
	} else {
		if spec.Presentation.Canvas.Width == 0 {
			spec.Presentation.Canvas.Width = 1920
		}
		if spec.Presentation.Canvas.Height == 0 {
			spec.Presentation.Canvas.Height = 1080
		}
		if spec.Presentation.Viewport.Width == 0 {
			spec.Presentation.Viewport.Width = spec.Presentation.Canvas.Width
		}
		if spec.Presentation.Viewport.Height == 0 {
			spec.Presentation.Viewport.Height = spec.Presentation.Canvas.Height
		}
		if spec.Presentation.BrowserFrame.Width == 0 {
			spec.Presentation.BrowserFrame = services.ExportFrameRect{
				X:      0,
				Y:      0,
				Width:  spec.Presentation.Canvas.Width,
				Height: spec.Presentation.Canvas.Height,
				Radius: 24,
			}
		}
	}

	if spec.Presentation.DeviceScaleFactor == 0 {
		spec.Presentation.DeviceScaleFactor = 1
	}
	if spec.Presentation.BrowserFrame.Radius == 0 {
		spec.Presentation.BrowserFrame.Radius = 24
	}
}

func ensureSummaryAndPlayback(spec, baseline *services.ReplayMovieSpec) {
	fallbackInterval := spec.Playback.FrameIntervalMs
	if fallbackInterval <= 0 && baseline != nil && baseline.Playback.FrameIntervalMs > 0 {
		fallbackInterval = baseline.Playback.FrameIntervalMs
	}
	if fallbackInterval <= 0 {
		fallbackInterval = 40
	}

	totalDuration, maxDuration := computeFrameDurations(spec.Frames, fallbackInterval)

	if spec.Summary.FrameCount <= 0 {
		spec.Summary.FrameCount = len(spec.Frames)
	}
	if spec.Summary.TotalDurationMs <= 0 {
		spec.Summary.TotalDurationMs = totalDuration
	}
	if spec.Summary.MaxFrameDurationMs <= 0 {
		spec.Summary.MaxFrameDurationMs = maxDuration
	}
	if spec.Summary.ScreenshotCount <= 0 {
		count := countFrameScreenshots(spec.Frames)
		if count > 0 {
			spec.Summary.ScreenshotCount = count
		} else if baseline != nil {
			spec.Summary.ScreenshotCount = baseline.Summary.ScreenshotCount
		}
	}

	if spec.Playback.FrameIntervalMs <= 0 {
		spec.Playback.FrameIntervalMs = fallbackInterval
	}
	if spec.Playback.DurationMs <= 0 {
		spec.Playback.DurationMs = spec.Summary.TotalDurationMs
	}
	if spec.Playback.TotalFrames <= 0 && spec.Playback.FrameIntervalMs > 0 && spec.Summary.TotalDurationMs > 0 {
		spec.Playback.TotalFrames = int(math.Ceil(float64(spec.Summary.TotalDurationMs) / float64(spec.Playback.FrameIntervalMs)))
	}
	if spec.Playback.TotalFrames <= 0 {
		spec.Playback.TotalFrames = spec.Summary.FrameCount
	}
	if spec.Playback.FPS <= 0 && spec.Playback.FrameIntervalMs > 0 {
		spec.Playback.FPS = int(math.Round(1000.0 / float64(spec.Playback.FrameIntervalMs)))
	}
	if spec.Playback.FPS <= 0 && baseline != nil && baseline.Playback.FPS > 0 {
		spec.Playback.FPS = baseline.Playback.FPS
	}
	if spec.Playback.FPS <= 0 {
		spec.Playback.FPS = 25
	}
}

func computeFrameDurations(frames []services.ExportFrame, fallback int) (int, int) {
	if fallback <= 0 {
		fallback = 40
	}
	total := 0
	maxDuration := 0
	for _, frame := range frames {
		duration := frame.DurationMs
		if duration <= 0 {
			duration = frame.HoldMs + frame.Enter.DurationMs + frame.Exit.DurationMs
		}
		if duration <= 0 {
			duration = fallback
		}
		total += duration
		if duration > maxDuration {
			maxDuration = duration
		}
	}
	return total, maxDuration
}

func countFrameScreenshots(frames []services.ExportFrame) int {
	count := 0
	for _, frame := range frames {
		if strings.TrimSpace(frame.ScreenshotAssetID) != "" {
			count++
		}
	}
	return count
}

func clampCursorScale(value float64) float64 {
	if value <= 0 {
		return 1.0
	}
	if value < 0.4 {
		return 0.4
	}
	if value > 2.4 {
		return 2.4
	}
	return value
}

// GetExecutionScreenshots handles GET /api/v1/executions/{id}/screenshots
func (h *Handler) GetExecutionScreenshots(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	executionID, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, ErrInvalidExecutionID)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	screenshots, err := h.workflowService.GetExecutionScreenshots(ctx, executionID)
	if err != nil {
		h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get screenshots")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_screenshots", "execution_id": executionID.String()}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"screenshots": screenshots,
	})
}

// GetExecutionTimeline handles GET /api/v1/executions/{id}/timeline
func (h *Handler) GetExecutionTimeline(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	executionID, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, ErrInvalidExecutionID)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	timeline, err := h.workflowService.GetExecutionTimeline(ctx, executionID)
	if err != nil {
		h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get execution timeline")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_timeline", "execution_id": executionID.String()}))
		return
	}

	h.respondSuccess(w, http.StatusOK, timeline)
}

// PostExecutionExport handles POST /api/v1/executions/{id}/export
func (h *Handler) PostExecutionExport(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	executionID, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, ErrInvalidExecutionID)
		return
	}

	var body executionExportRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := decodeJSONBodyAllowEmpty(w, r, &body); err != nil {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "invalid json payload"}))
			return
		}
	}

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if body.Format != "" {
		format = strings.ToLower(strings.TrimSpace(body.Format))
	}
	if format == "" {
		format = "json"
	}

	previewCtx, cancelPreview := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancelPreview()

	preview, svcErr := h.workflowService.DescribeExecutionExport(previewCtx, executionID)
	if svcErr != nil {
		if errors.Is(svcErr, database.ErrNotFound) {
			h.respondError(w, ErrExecutionNotFound.WithDetails(map[string]string{"execution_id": executionID.String()}))
			return
		}
		h.log.WithError(svcErr).WithField("execution_id", executionID).Error("Failed to describe execution export")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "describe_export"}))
		return
	}

	spec, specErr := buildExportSpec(preview.Package, body.MovieSpec, executionID)
	if specErr != nil {
		if errors.Is(specErr, errMovieSpecUnavailable) {
			if format == "json" {
				applyExportOverrides(preview.Package, body.Overrides)
				h.respondSuccess(w, http.StatusOK, preview)
			} else {
				h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"error": "export package unavailable"}))
			}
			return
		}
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": specErr.Error()}))
		return
	}

	preview.Package = spec

	if format == "json" {
		applyExportOverrides(spec, body.Overrides)
		h.respondSuccess(w, http.StatusOK, preview)
		return
	}

	if body.Overrides != nil {
		applyExportOverrides(spec, body.Overrides)
	}

	renderTimeout := services.EstimateReplayRenderTimeout(spec)
	renderCtx, cancelRender := context.WithTimeout(r.Context(), renderTimeout)
	defer cancelRender()

	media, renderErr := h.replayRenderer.Render(renderCtx, spec, services.RenderFormat(format), body.FileName)
	if renderErr != nil {
		errMsg := strings.TrimSpace(renderErr.Error())
		if len(errMsg) > 0 && len(errMsg) > 512 {
			errMsg = errMsg[:512]
		}
		fields := logrus.Fields{"execution_id": executionID}
		if errMsg != "" {
			fields["renderer_error"] = errMsg
		}
		h.log.WithError(renderErr).WithFields(fields).Error("Failed to render replay export")
		details := map[string]string{"operation": "render_export"}
		if errMsg != "" {
			details["error"] = errMsg
		}
		h.respondError(w, ErrInternalServer.WithDetails(details))
		return
	}
	defer media.Cleanup()

	file, err := os.Open(media.Path)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "open_export"}))
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "stat_export"}))
		return
	}

	w.Header().Set("Content-Type", media.ContentType)
	if info.Size() > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	}
	if strings.TrimSpace(media.Filename) != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", media.Filename))
	}

	http.ServeContent(w, r, media.Filename, info.ModTime(), file)
}

// GetExecution handles GET /api/v1/executions/{id}
func (h *Handler) GetExecution(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, ErrInvalidExecutionID)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	execution, err := h.workflowService.GetExecution(ctx, id)
	if err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to get execution")
		h.respondError(w, ErrExecutionNotFound.WithDetails(map[string]string{"execution_id": id.String()}))
		return
	}

	h.respondSuccess(w, http.StatusOK, execution)
}

// ListExecutions handles GET /api/v1/executions
func (h *Handler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	workflowIDStr := r.URL.Query().Get("workflow_id")
	var workflowID *uuid.UUID
	if workflowIDStr != "" {
		if id, err := uuid.Parse(workflowIDStr); err == nil {
			workflowID = &id
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	limit, offset := parsePaginationParams(r, defaultPageLimit, maxPageLimit)

	executions, err := h.workflowService.ListExecutions(ctx, workflowID, limit, offset)
	if err != nil {
		h.log.WithError(err).Error("Failed to list executions")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_executions"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"executions": executions,
	})
}

// StopExecution handles POST /api/v1/executions/{id}/stop
func (h *Handler) StopExecution(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, ErrInvalidExecutionID)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	if err := h.workflowService.StopExecution(ctx, id); err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to stop execution")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "stop_execution", "execution_id": id.String()}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"status": "stopped",
	})
}
