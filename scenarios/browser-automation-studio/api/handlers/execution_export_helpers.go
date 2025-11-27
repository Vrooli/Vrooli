package handlers

import (
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/handlers/export"
	"github.com/vrooli/browser-automation-studio/services/ai"
	"github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/services/logutil"
	"github.com/vrooli/browser-automation-studio/services/recording"
	"github.com/vrooli/browser-automation-studio/services/replay"
	"github.com/vrooli/browser-automation-studio/services/workflow"
)

// executionExportRequest represents the JSON payload for execution export endpoints.
// It wraps the export.Request type for backwards compatibility with existing handler code.
type executionExportRequest = export.Request

// executionExportOverrides allows clients to customize export themes and cursor configuration.
// It wraps the export.Overrides type for backwards compatibility with existing handler code.
type executionExportOverrides = export.Overrides

// themePresetOverride specifies which chrome and background preset themes to apply.
// It wraps the export.ThemePreset type for backwards compatibility with existing handler code.
type themePresetOverride = export.ThemePreset

// cursorPresetOverride specifies which cursor preset theme to apply and additional options.
// It wraps the export.CursorPreset type for backwards compatibility with existing handler code.
type cursorPresetOverride = export.CursorPreset

var errMovieSpecUnavailable = export.ErrMovieSpecUnavailable

// applyExportOverrides applies client-provided overrides to a movie spec.
func applyExportOverrides(spec *export.ReplayMovieSpec, overrides *executionExportOverrides) {
	export.Apply(spec, overrides)
}

// buildExportSpec constructs a validated ReplayMovieSpec for export by merging client-provided
// and server-generated specs, validating execution ID matching, and filling in defaults.
func buildExportSpec(baseline, incoming *export.ReplayMovieSpec, executionID uuid.UUID) (*export.ReplayMovieSpec, error) {
	return export.BuildSpec(baseline, incoming, executionID)
}

// cloneMovieSpec creates a deep copy of a ReplayMovieSpec.
func cloneMovieSpec(spec *export.ReplayMovieSpec) (*export.ReplayMovieSpec, error) {
	return export.Clone(spec)
}
