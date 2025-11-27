package export

// NOTE: Request type remains in handlers/export/types.go (HTTP layer only)

// Overrides allows clients to customize export themes and cursor configuration.
type Overrides struct {
	Theme        *ExportTheme      `json:"theme,omitempty"`
	Cursor       *ExportCursorSpec `json:"cursor,omitempty"`
	ThemePreset  *ThemePreset      `json:"theme_preset,omitempty"`
	CursorPreset *CursorPreset     `json:"cursor_preset,omitempty"`
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
