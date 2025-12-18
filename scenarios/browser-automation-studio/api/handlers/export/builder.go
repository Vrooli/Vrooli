package export

import (
	basexport "github.com/vrooli/browser-automation-studio/services/export"
)

// BuildThemeFromPreset constructs an ExportTheme by applying preset configurations
// to the baseline theme from the movie spec.
//
// This is a thin wrapper that delegates to services/export.BuildThemeFromPreset.
func BuildThemeFromPreset(baseline *basexport.ReplayMovieSpec, preset *ThemePreset) *basexport.ExportTheme {
	return basexport.BuildThemeFromPreset(baseline, preset)
}

// BuildCursorSpec constructs an ExportCursorSpec by applying preset configurations
// and defaults to the existing cursor spec.
//
// This is a thin wrapper that delegates to services/export.BuildCursorSpec.
func BuildCursorSpec(existing basexport.ExportCursorSpec, preset *CursorPreset) basexport.ExportCursorSpec {
	return basexport.BuildCursorSpec(existing, preset)
}
