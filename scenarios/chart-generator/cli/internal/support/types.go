package support

import "encoding/json"

// Style mirrors one entry returned by GET /api/v1/styles.
type Style struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
	IsDefault   bool   `json:"is_default,omitempty"`
}

// StyleListResponse is the shape returned by GET /api/v1/styles.
type StyleListResponse struct {
	Styles []Style `json:"styles"`
	Count  int     `json:"count"`
}

// Template mirrors one entry returned by GET /api/v1/templates.
type Template struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ChartType   string `json:"chart_type,omitempty"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	Industry    string `json:"industry,omitempty"`
}

// TemplateListResponse is the shape returned by GET /api/v1/templates.
type TemplateListResponse struct {
	Templates []Template `json:"templates"`
	Count     int        `json:"count"`
}

// ColorPalette is a single palette entry in GET /api/v1/styles/builder/palettes.
type ColorPalette struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Colors []string `json:"colors"`
}

// PaletteResponse is the shape returned by GET /api/v1/styles/builder/palettes.
type PaletteResponse struct {
	Palettes    []ColorPalette      `json:"palettes"`
	Recommended map[string][]string `json:"recommended,omitempty"`
}

// ChartGenerationResponse is the shape returned by POST /api/v1/charts/generate
// and its siblings. Metadata and Error are loosely typed because the API mixes
// stable and experimental fields in those maps.
type ChartGenerationResponse struct {
	Success  bool                   `json:"success"`
	ChartID  string                 `json:"chart_id,omitempty"`
	Files    map[string]string      `json:"files,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Error    *ChartErrorResponse    `json:"error,omitempty"`
}

// ChartErrorResponse is the nested error object on failed chart responses.
type ChartErrorResponse struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// DataTransformResponse is the shape returned by POST /api/v1/data/transform.
type DataTransformResponse struct {
	Success bool                     `json:"success"`
	Data    []map[string]interface{} `json:"data"`
	Count   int                      `json:"count"`
}

// DataAggregateResponse is the shape returned by POST /api/v1/data/aggregate.
type DataAggregateResponse struct {
	Success bool                     `json:"success"`
	Result  []map[string]interface{} `json:"result"`
	Data    []map[string]interface{} `json:"data,omitempty"`
	Method  string                   `json:"method,omitempty"`
	Field   string                   `json:"field,omitempty"`
	GroupBy string                   `json:"group_by,omitempty"`
}

// StyleBuilderPreviewResponse is the shape returned by
// POST /api/v1/styles/builder/preview.
type StyleBuilderPreviewResponse struct {
	Success    bool                   `json:"success"`
	Preview    string                 `json:"preview,omitempty"`
	PreviewURL string                 `json:"preview_url,omitempty"`
	Style      map[string]interface{} `json:"style,omitempty"`
}

// SavedStyle captures the response from POST /api/v1/styles/builder/save.
// Fields beyond the common ones are preserved as Extra for JSON passthrough.
type SavedStyle struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Colors     []string        `json:"colors,omitempty"`
	FontFamily string          `json:"font_family,omitempty"`
	FontSize   int             `json:"font_size,omitempty"`
	Background string          `json:"background,omitempty"`
	GridLines  bool            `json:"grid_lines,omitempty"`
	Animation  bool            `json:"animation,omitempty"`
	CreatedAt  string          `json:"created_at,omitempty"`
	Extra      json.RawMessage `json:"-"`
}
