package export

import (
	basexport "github.com/vrooli/browser-automation-studio/services/export"
)

// Request represents the JSON payload for execution export endpoints.
// This is HTTP-layer only; it wraps the service-layer types for JSON binding.
type Request struct {
	Format    string                     `json:"format,omitempty"`
	FileName  string                     `json:"file_name,omitempty"`
	OutputDir string                     `json:"output_dir,omitempty"`
	Overrides *Overrides                 `json:"overrides,omitempty"`
	MovieSpec *basexport.ReplayMovieSpec `json:"movie_spec,omitempty"`
}

// Type aliases for backward compatibility - these delegate to services/export.
type (
	// Overrides allows clients to customize export themes and cursor configuration.
	Overrides = basexport.Overrides

	// ThemePreset specifies which chrome and background preset themes to apply.
	ThemePreset = basexport.ThemePreset

	// CursorPreset specifies which cursor preset theme to apply and additional options.
	CursorPreset = basexport.CursorPreset
)
