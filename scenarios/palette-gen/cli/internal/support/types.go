package support

// PaletteResponse mirrors the palette-gen API shape returned by /generate.
type PaletteResponse struct {
	Success     bool                   `json:"success"`
	Palette     []string               `json:"palette,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Theme       string                 `json:"theme,omitempty"`
	Style       string                 `json:"style,omitempty"`
	Description string                 `json:"description,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Debug       map[string]interface{} `json:"debug,omitempty"`
}

// SuggestResponse mirrors /suggest; suggestions are variable-shape per AI/fallback output.
type SuggestResponse struct {
	Success     bool                     `json:"success"`
	Suggestions []map[string]interface{} `json:"suggestions"`
}

// ExportResponse mirrors /export.
type ExportResponse struct {
	Success bool   `json:"success"`
	Export  string `json:"export"`
}

// AccessibilityResponse mirrors /accessibility.
type AccessibilityResponse struct {
	Success        bool    `json:"success"`
	ContrastRatio  float64 `json:"contrast_ratio"`
	WCAGAA         bool    `json:"wcag_aa"`
	WCAGAAA        bool    `json:"wcag_aaa"`
	LargeTextAA    bool    `json:"large_text_aa"`
	LargeTextAAA   bool    `json:"large_text_aaa"`
	Recommendation string  `json:"recommendation,omitempty"`
}

// HarmonyResponse mirrors /harmony.
type HarmonyResponse struct {
	Success      bool                   `json:"success"`
	Analysis     map[string]interface{} `json:"analysis"`
	IsHarmonious bool                   `json:"is_harmonious"`
	Score        float64                `json:"score"`
}

// ColorblindResponse mirrors /colorblind.
type ColorblindResponse struct {
	Success   bool     `json:"success"`
	Simulated []string `json:"simulated"`
	Type      string   `json:"type"`
}

// HistoryResponse mirrors /history.
type HistoryResponse struct {
	Success bool              `json:"success"`
	History []PaletteResponse `json:"history"`
}
