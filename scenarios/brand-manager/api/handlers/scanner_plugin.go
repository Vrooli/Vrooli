// Package handlers - extensible scanner plugin system for inline validation.
// [REQ:BM-REQ-SCAN-PLUGINS] [REQ:BM-REQ-SCAN-EXTEND]
package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"brand-manager/domain"

	"github.com/gorilla/mux"
)

// ScannerPlugin defines the interface for file scanners that detect brand markers.
// Implement this interface to add support for new file types beyond CSS and JSON.
// [REQ:BM-REQ-SCAN-PLUGINS]
type ScannerPlugin interface {
	// Name returns a unique identifier for this plugin (e.g. "css", "json", "yaml").
	Name() string
	// Extensions returns the file extensions this plugin handles (e.g. [".css", ".scss"]).
	Extensions() []string
	// ScanFile scans a file for brand markers and returns results.
	ScanFile(path, relPath string) []domain.ScanResult
}

// ScannerRegistry manages registered scanner plugins. [REQ:BM-REQ-SCAN-PLUGINS]
type ScannerRegistry struct {
	plugins   []ScannerPlugin
	extLookup map[string]ScannerPlugin
}

// NewScannerRegistry creates a registry with the default CSS and JSON plugins.
func NewScannerRegistry() *ScannerRegistry {
	r := &ScannerRegistry{
		extLookup: make(map[string]ScannerPlugin),
	}
	r.Register(&CSSPlugin{})
	r.Register(&JSONPlugin{})
	return r
}

// Register adds a plugin to the registry. [REQ:BM-REQ-SCAN-PLUGINS]
func (r *ScannerRegistry) Register(p ScannerPlugin) {
	r.plugins = append(r.plugins, p)
	for _, ext := range p.Extensions() {
		r.extLookup[ext] = p
	}
}

// PluginForExt returns the plugin registered for the given file extension, or nil.
func (r *ScannerRegistry) PluginForExt(ext string) ScannerPlugin {
	return r.extLookup[ext]
}

// Plugins returns all registered plugins.
func (r *ScannerRegistry) Plugins() []ScannerPlugin {
	return r.plugins
}

// ListPlugins returns metadata about registered plugins.
func (r *ScannerRegistry) ListPlugins() []PluginInfo {
	info := make([]PluginInfo, len(r.plugins))
	for i, p := range r.plugins {
		info[i] = PluginInfo{
			Name:       p.Name(),
			Extensions: p.Extensions(),
		}
	}
	return info
}

// PluginInfo describes a registered scanner plugin.
type PluginInfo struct {
	Name       string   `json:"name"`
	Extensions []string `json:"extensions"`
}

// --- Built-in plugins ---

// CSSPlugin scans CSS/SCSS/LESS files for brand-manager comment markers.
// [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-EXTEND]
type CSSPlugin struct{}

func (p *CSSPlugin) Name() string         { return "css" }
func (p *CSSPlugin) Extensions() []string { return []string{".css", ".scss", ".less"} }
func (p *CSSPlugin) ScanFile(path, relPath string) []domain.ScanResult {
	return scanFileForCSS(path, relPath)
}

// JSONPlugin scans JSON files for _brand keys.
// [REQ:BM-REQ-SCAN-JSON] [REQ:BM-REQ-SCAN-EXTEND]
type JSONPlugin struct{}

func (p *JSONPlugin) Name() string         { return "json" }
func (p *JSONPlugin) Extensions() []string { return []string{".json"} }
func (p *JSONPlugin) ScanFile(path, relPath string) []domain.ScanResult {
	return scanFileForJSON(path, relPath)
}

// YAMLPlugin scans YAML files for brand-manager keys. [REQ:BM-REQ-SCAN-EXTEND]
type YAMLPlugin struct{}

var yamlBrandKeyRe = regexp.MustCompile(`^\s*(_brand\S*):\s*(.+)`)

func (p *YAMLPlugin) Name() string         { return "yaml" }
func (p *YAMLPlugin) Extensions() []string { return []string{".yaml", ".yml"} }
func (p *YAMLPlugin) ScanFile(path, relPath string) []domain.ScanResult {
	return scanFileWithRegex(path, relPath, "yaml", yamlBrandKeyRe)
}

// HTMLPlugin scans HTML files for brand-manager data attributes. [REQ:BM-REQ-SCAN-EXTEND]
type HTMLPlugin struct{}

var htmlBrandAttrRe = regexp.MustCompile(`data-brand-(\w+)="([^"]*)"`)

func (p *HTMLPlugin) Name() string         { return "html" }
func (p *HTMLPlugin) Extensions() []string { return []string{".html", ".htm"} }
func (p *HTMLPlugin) ScanFile(path, relPath string) []domain.ScanResult {
	return scanFileWithRegex(path, relPath, "html", htmlBrandAttrRe)
}

// --- Plugin-based scan handler ---

// ScanScenarioWithPlugins handles GET /api/v1/scan-ext/{scenario}. [REQ:BM-REQ-SCAN-PLUGINS] [REQ:BM-REQ-SCAN-EXTEND]
// Uses the plugin registry to support extensible file scanning.
func (h *Handlers) ScanScenarioWithPlugins(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario"]
	registry := h.scannerRegistry()

	scenarioDir, done := h.resolveScenarioDir(w, scenario)
	if done {
		return
	}

	report := domain.ScanReport{Scenario: scenario}

	walkScenarioDir(scenarioDir, func(path, relPath, ext string) {
		plugin := registry.PluginForExt(ext)
		if plugin == nil {
			return
		}

		results := plugin.ScanFile(path, relPath)
		report.Results = append(report.Results, results...)

		for _, r := range results {
			switch r.Type {
			case "css":
				report.CSSMarkers++
			case "json":
				report.JSONKeys++
			default:
				report.Total++
			}
		}
	})

	report.Total += report.CSSMarkers + report.JSONKeys
	writeJSON(w, http.StatusOK, report)
}

// ListScanPlugins handles GET /api/v1/scan/plugins. [REQ:BM-REQ-SCAN-PLUGINS]
func (h *Handlers) ListScanPlugins(w http.ResponseWriter, r *http.Request) {
	registry := h.scannerRegistry()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"plugins": registry.ListPlugins(),
	})
}

// scannerRegistry returns the handler's scanner registry, creating a default one if needed.
func (h *Handlers) scannerRegistry() *ScannerRegistry {
	if h.scanReg != nil {
		return h.scanReg
	}
	reg := NewScannerRegistry()
	reg.Register(&YAMLPlugin{})
	reg.Register(&HTMLPlugin{})
	return reg
}

// SetScannerRegistry sets a custom scanner registry for testing. [REQ:BM-REQ-SCAN-PLUGINS]
func (h *Handlers) SetScannerRegistry(reg *ScannerRegistry) {
	h.scanReg = reg
}

// --- Scan/Preview endpoint for UI apply preview ---

// ScanPreviewRequest specifies what to preview. [REQ:BM-REQ-UI-APPLY]
type ScanPreviewRequest struct {
	ScenarioName string   `json:"scenario_name"`
	Elements     []string `json:"elements,omitempty"`
}

// ApplyPreview handles POST /api/v1/brands/{id}/apply/preview. [REQ:BM-REQ-APPLY-PARTIAL]
// Returns what would change without actually writing files.
func (h *Handlers) ApplyPreview(w http.ResponseWriter, r *http.Request) {
	// Delegate to ApplyBrand with dry-run header set
	r.Header.Set("X-Dry-Run", "true")
	h.ApplyBrand(w, r)
}

// ThemePreview handles POST /api/v1/brands/{id}/theme-preview. [REQ:BM-REQ-UI-THEME]
// Returns CSS custom properties for the brand that can be injected for live preview.
type ThemePreviewResponse struct {
	BrandID string            `json:"brand_id"`
	CSS     string            `json:"css"`
	Tokens  map[string]string `json:"tokens"`
	Mode    string            `json:"mode"` // "light" or "dark"
}

func (h *Handlers) ThemePreview(w http.ResponseWriter, r *http.Request) {
	brandID := mux.Vars(r)["id"]

	brand, err := h.brands.GetByID(r.Context(), brandID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "brand not found"})
		return
	}

	mode := r.URL.Query().Get("mode")
	if mode != "dark" {
		mode = "light"
	}

	tokens := make(map[string]string)
	var cssBuilder strings.Builder
	cssBuilder.WriteString(":root {\n")

	if brand.Colors != nil {
		for _, p := range colorPairs(brand.Colors) {
			if p.value != "" {
				val := p.value
				if mode == "dark" {
					val = invertForDarkMode(p.name, p.value)
				}
				tokens[p.name] = val
				cssBuilder.WriteString("  --brand-" + p.name + ": " + val + ";\n")
			}
		}
	}

	if brand.Typography != nil {
		for _, p := range typographyPairs(brand.Typography) {
			if p.value != "" {
				tokens[p.name] = p.value
				cssBuilder.WriteString("  --brand-" + p.name + ": " + p.value + ";\n")
			}
		}
	}

	cssBuilder.WriteString("}\n")

	resp := ThemePreviewResponse{
		BrandID: brand.ID,
		CSS:     cssBuilder.String(),
		Tokens:  tokens,
		Mode:    mode,
	}

	writeJSON(w, http.StatusOK, resp)
}

// invertForDarkMode provides a simple dark-mode approximation for preview.
// Background/surface become dark, text becomes light. Other colors stay the same.
func invertForDarkMode(name, value string) string {
	switch name {
	case "background":
		return "#1a1a2e"
	case "surface":
		return "#16213e"
	case "text":
		return "#eaeaea"
	default:
		return value
	}
}

// --- Register extended routes ---

// RegisterExtendedRoutes adds the plugin-based and preview routes.
// [REQ:BM-REQ-SCAN-PLUGINS] [REQ:BM-REQ-SCAN-EXTEND] [REQ:BM-REQ-UI-APPLY] [REQ:BM-REQ-UI-THEME]
func (h *Handlers) RegisterExtendedRoutes(r *mux.Router) {
	api := r.PathPrefix("/api/v1").Subrouter()

	// Literal routes MUST be registered before wildcards to avoid conflicts.
	// Generate options (must come before /brands/{id} wildcard)
	api.HandleFunc("/brands/generate/options", h.GenerateOptions).Methods("GET")

	// Plugin list (must come before /scan/{scenario} wildcard)
	api.HandleFunc("/scanner/plugins", h.ListScanPlugins).Methods("GET")

	// Plugin-based scanner
	api.HandleFunc("/scan-ext/{scenario}", h.ScanScenarioWithPlugins).Methods("GET")

	// Apply preview (dry-run alias)
	api.HandleFunc("/brands/{id}/apply/preview", h.ApplyPreview).Methods("POST")

	// Theme preview
	api.HandleFunc("/brands/{id}/theme-preview", h.ThemePreview).Methods("GET")
}

// GenerateOptions handles GET /api/v1/brands/generate/options. [REQ:BM-REQ-UI-GENERATE]
// Returns available generation providers and their capabilities.
func (h *Handlers) GenerateOptions(w http.ResponseWriter, r *http.Request) {
	options := map[string]interface{}{
		"providers": []map[string]interface{}{
			{
				"id":           "manual",
				"name":         "Manual Entry",
				"description":  "Create brand elements by hand",
				"available":    true,
				"capabilities": []string{"colors", "typography", "identity", "voice"},
			},
			{
				"id":           "ollama",
				"name":         "Ollama (Local AI)",
				"description":  "Generate brand elements using local Ollama models",
				"available":    false,
				"capabilities": []string{"colors", "typography", "voice"},
				"requires":     "Ollama resource must be running",
			},
			{
				"id":           "openrouter",
				"name":         "OpenRouter (Cloud AI)",
				"description":  "Generate brand elements using cloud AI models",
				"available":    false,
				"capabilities": []string{"colors", "typography", "identity", "voice", "images"},
				"requires":     "OPENROUTER_API_KEY environment variable",
			},
		},
		"elements": []string{"colors", "typography", "identity", "voice", "logo", "favicon"},
	}
	data, _ := json.Marshal(options)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
