package export

import "github.com/vrooli/browser-automation-studio/services"

// Request represents the JSON payload for execution export endpoints.
type Request struct {
	Format    string                    `json:"format,omitempty"`
	FileName  string                    `json:"file_name,omitempty"`
	OutputDir string                    `json:"output_dir,omitempty"`
	Overrides *Overrides                `json:"overrides,omitempty"`
	MovieSpec *services.ReplayMovieSpec `json:"movie_spec,omitempty"`
}

// Overrides allows clients to customize export themes and cursor configuration.
type Overrides struct {
	Theme        *services.ExportTheme      `json:"theme,omitempty"`
	Cursor       *services.ExportCursorSpec `json:"cursor,omitempty"`
	ThemePreset  *ThemePreset               `json:"theme_preset,omitempty"`
	CursorPreset *CursorPreset              `json:"cursor_preset,omitempty"`
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
