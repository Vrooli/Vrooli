package export

import (
	basexport "github.com/vrooli/browser-automation-studio/services/export"
)

// Request represents the JSON payload for execution export endpoints.
type Request struct {
	Format    string                       `json:"format,omitempty"`
	FileName  string                       `json:"file_name,omitempty"`
	OutputDir string                       `json:"output_dir,omitempty"`
	Overrides *Overrides                   `json:"overrides,omitempty"`
	MovieSpec *basexport.ReplayMovieSpec `json:"movie_spec,omitempty"`
}

// Overrides allows clients to customize export themes and cursor configuration.
type Overrides struct {
	Theme        *basexport.ExportTheme      `json:"theme,omitempty"`
	Cursor       *basexport.ExportCursorSpec `json:"cursor,omitempty"`
	ThemePreset  *ThemePreset                `json:"theme_preset,omitempty"`
	CursorPreset *CursorPreset               `json:"cursor_preset,omitempty"`
}

// ThemePreset specifies which chrome and background preset themes to apply.
type ThemePreset struct {
	ChromeTheme     string `json:"chrome_theme,omitempty"`
	BackgroundTheme string `json:"background_theme,omitempty"`
}

// CursorPreset specifies which cursor preset theme to apply and additional options.
type CursorPreset struct {
	Theme           string  `json:"theme,omitempty"`
	InitialPosition string  `json:"initial_position,omitempty"`
	Scale           float64 `json:"scale,omitempty"`
	ClickAnimation  string  `json:"click_animation,omitempty"`
}
