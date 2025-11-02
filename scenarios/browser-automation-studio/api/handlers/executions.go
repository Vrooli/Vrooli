package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services"
)

type executionExportRequest struct {
	Format    string                    `json:"format,omitempty"`
	FileName  string                    `json:"file_name,omitempty"`
	Overrides *executionExportOverrides `json:"overrides,omitempty"`
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

func applyExportOverrides(pkg *services.ReplayExportPackage, overrides *executionExportOverrides) {
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
}

const defaultAccentColor = "#38BDF8"

func buildThemeFromPreset(pkg *services.ReplayExportPackage, preset *themePresetOverride) *services.ExportTheme {
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
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
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

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	preview, svcErr := h.workflowService.DescribeExecutionExport(ctx, executionID)
	if svcErr != nil {
		if errors.Is(svcErr, database.ErrNotFound) {
			h.respondError(w, ErrExecutionNotFound.WithDetails(map[string]string{"execution_id": executionID.String()}))
			return
		}
		h.log.WithError(svcErr).WithField("execution_id", executionID).Error("Failed to describe execution export")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "describe_export"}))
		return
	}

	if format == "json" {
		applyExportOverrides(preview.Package, body.Overrides)
		h.respondSuccess(w, http.StatusOK, preview)
		return
	}

	if preview.Package == nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"error": "export package unavailable"}))
		return
	}

	if body.Overrides != nil {
		applyExportOverrides(preview.Package, body.Overrides)
	}

	media, renderErr := h.replayRenderer.Render(ctx, preview.Package, services.RenderFormat(format), body.FileName)
	if renderErr != nil {
		h.log.WithError(renderErr).WithField("execution_id", executionID).Error("Failed to render replay export")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "render_export"}))
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

	executions, err := h.workflowService.ListExecutions(ctx, workflowID, 100, 0)
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
