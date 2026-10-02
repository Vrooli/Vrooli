package export

import (
	"encoding/json"

	basexport "github.com/vrooli/browser-automation-studio/services/export"
	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
	"google.golang.org/protobuf/encoding/protojson"
)

// Request represents the JSON payload for execution export endpoints.
// This is HTTP-layer only; it wraps the service-layer types for JSON binding.
type Request struct {
	Format       string                `json:"format,omitempty"`
	FileName     string                `json:"file_name,omitempty"`
	OutputDir    string                `json:"output_dir,omitempty"`
	RenderSource string                `json:"render_source,omitempty"`
	Overrides    *Overrides            `json:"overrides,omitempty"`
	MovieSpec    *exportsv1.ReplaySpec `json:"movie_spec,omitempty"`
}

// UnmarshalJSON validates movie_spec against the generated API contract.
func (request *Request) UnmarshalJSON(data []byte) error {
	type requestFields struct {
		Format       string          `json:"format,omitempty"`
		FileName     string          `json:"file_name,omitempty"`
		OutputDir    string          `json:"output_dir,omitempty"`
		RenderSource string          `json:"render_source,omitempty"`
		Overrides    *Overrides      `json:"overrides,omitempty"`
		MovieSpec    json.RawMessage `json:"movie_spec,omitempty"`
	}
	var fields requestFields
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	request.Format = fields.Format
	request.FileName = fields.FileName
	request.OutputDir = fields.OutputDir
	request.RenderSource = fields.RenderSource
	request.Overrides = fields.Overrides
	request.MovieSpec = nil
	if len(fields.MovieSpec) == 0 || string(fields.MovieSpec) == "null" {
		return nil
	}

	var wireSpec exportsv1.ReplaySpec
	if err := protojson.Unmarshal(fields.MovieSpec, &wireSpec); err != nil {
		return err
	}
	request.MovieSpec = &wireSpec
	return nil
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
