// Package handlers - discovery scanner for existing branding state.
// [REQ:BM-REQ-DISC-SCAN] [REQ:BM-REQ-DISC-IMPORT] [REQ:BM-REQ-DISC-LPBS]
package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"brand-manager/apierr"
	"brand-manager/domain"

	"github.com/gorilla/mux"
)

// DiscoveryResult reports what branding state was found in a scenario.
type DiscoveryResult struct {
	Scenario    string            `json:"scenario"`
	Sources     []DiscoverySource `json:"sources"`
	DraftBrand  *domain.Brand     `json:"draft_brand,omitempty"`
	Confidence  float64           `json:"confidence"` // 0.0-1.0 overall confidence
	Suggestions []string          `json:"suggestions,omitempty"`
}

// DiscoverySource records where branding data was found.
type DiscoverySource struct {
	File       string  `json:"file"`
	Type       string  `json:"type"` // "service_json", "branding_json", "theme_css", "manifest", "asset"
	Confidence float64 `json:"confidence"`
	Fields     int     `json:"fields"` // number of branding fields found
}

// DiscoverScenario handles GET /api/v1/discover/{scenario}. [REQ:BM-REQ-DISC-SCAN]
func (h *Handlers) DiscoverScenario(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario"]

	scenarioDir, done := h.resolveScenarioDir(w, scenario)
	if done {
		return
	}

	result := DiscoveryResult{Scenario: scenario}
	draft := &domain.Brand{Name: scenario}

	h.runDiscovery(scenarioDir, &result, draft)

	// Calculate overall confidence
	if len(result.Sources) > 0 {
		var total float64
		for _, s := range result.Sources {
			total += s.Confidence
		}
		result.Confidence = total / float64(len(result.Sources))
		result.DraftBrand = draft
	}

	// Add suggestions for missing data
	if draft.Colors == nil {
		result.Suggestions = append(result.Suggestions, "No color system found. Consider defining brand colors.")
	}
	if draft.Typography == nil {
		result.Suggestions = append(result.Suggestions, "No typography found. Consider defining brand fonts.")
	}
	if draft.Identity == nil || draft.Identity.LogoPath == "" {
		result.Suggestions = append(result.Suggestions, "No logo found. Consider uploading a brand logo.")
	}

	writeJSON(w, http.StatusOK, result)
}

// ImportDiscovery handles POST /api/v1/discover/{scenario}/import. [REQ:BM-REQ-DISC-IMPORT]
// Creates a new brand from discovered state.
func (h *Handlers) ImportDiscovery(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario"]

	scenarioDir, done := h.resolveScenarioDir(w, scenario)
	if done {
		return
	}

	draft := &domain.Brand{Name: scenario + " (discovered)"}
	result := &DiscoveryResult{Scenario: scenario}
	h.runDiscovery(scenarioDir, result, draft)

	if len(result.Sources) == 0 {
		apierr.Write(w, apierr.Validation("no branding state found to import"))
		return
	}

	draft.ID = h.newID()

	if isDryRun(r) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"dry_run":    true,
			"draft":      draft,
			"sources":    result.Sources,
			"confidence": result.Confidence,
		})
		return
	}

	if err := h.brands.Create(r.Context(), draft); err != nil {
		apierr.Write(w, apierr.Internal("create brand", err))
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"brand":      draft,
		"sources":    result.Sources,
		"confidence": result.Confidence,
	})
}

// runDiscovery scans all branding sources in priority order.
func (h *Handlers) runDiscovery(scenarioDir string, result *DiscoveryResult, draft *domain.Brand) {
	h.discoverServiceJSON(scenarioDir, result, draft)
	h.discoverBrandingJSON(scenarioDir, result, draft)
	h.discoverManifest(scenarioDir, result, draft)
	h.discoverThemeCSS(scenarioDir, result, draft)
	h.discoverAssets(scenarioDir, result, draft)
}

// ensureIdentity returns the brand's Identity, creating it if nil.
func ensureIdentity(draft *domain.Brand) *domain.Identity {
	if draft.Identity == nil {
		draft.Identity = &domain.Identity{}
	}
	return draft.Identity
}

// ensureColors returns the brand's Colors, creating it if nil.
func ensureColors(draft *domain.Brand) *domain.Colors {
	if draft.Colors == nil {
		draft.Colors = &domain.Colors{}
	}
	return draft.Colors
}

// discoverServiceJSON reads .vrooli/service.json for branding hints. [REQ:BM-REQ-DISC-SCAN]
func (h *Handlers) discoverServiceJSON(scenarioDir string, result *DiscoveryResult, draft *domain.Brand) {
	path := filepath.Join(scenarioDir, ".vrooli", "service.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var svc map[string]interface{}
	if err := json.Unmarshal(data, &svc); err != nil {
		return
	}

	fields := 0
	if name, ok := svc["name"].(string); ok && name != "" {
		ensureIdentity(draft).DisplayName = name
		fields++
	}
	if desc, ok := svc["description"].(string); ok && desc != "" {
		draft.Description = desc
		fields++
	}
	if tags, ok := svc["tags"].([]interface{}); ok && len(tags) > 0 {
		fields++
	}

	if fields > 0 {
		result.Sources = append(result.Sources, DiscoverySource{
			File:       ".vrooli/service.json",
			Type:       "service_json",
			Confidence: 0.5,
			Fields:     fields,
		})
	}
}

// discoverBrandingJSON reads .vrooli/branding.json (LPBS format). [REQ:BM-REQ-DISC-LPBS]
func (h *Handlers) discoverBrandingJSON(scenarioDir string, result *DiscoveryResult, draft *domain.Brand) {
	path := filepath.Join(scenarioDir, ".vrooli", "branding.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var branding map[string]interface{}
	if err := json.Unmarshal(data, &branding); err != nil {
		return
	}

	fields := 0

	if siteName, ok := branding["site_name"].(string); ok && siteName != "" {
		ensureIdentity(draft).DisplayName = siteName
		fields++
	}
	if tagline, ok := branding["tagline"].(string); ok && tagline != "" {
		ensureIdentity(draft).Tagline = tagline
		fields++
	}

	// Parse theme colors from LPBS format
	if theme, ok := branding["theme"].(map[string]interface{}); ok {
		colors := &domain.Colors{}
		colorFields := 0
		for key, val := range theme {
			if s, ok := val.(string); ok && s != "" {
				switch key {
				case "primary", "primary_color":
					colors.Primary = s
					colorFields++
				case "secondary", "secondary_color":
					colors.Secondary = s
					colorFields++
				case "accent", "accent_color":
					colors.Accent = s
					colorFields++
				case "background", "background_color":
					colors.Background = s
					colorFields++
				case "text", "text_color":
					colors.Text = s
					colorFields++
				}
			}
		}
		if colorFields > 0 {
			draft.Colors = colors
			fields += colorFields
		}
	}

	// Parse logo URLs
	if logoURL, ok := branding["logo_url"].(string); ok && logoURL != "" {
		ensureIdentity(draft).LogoPath = logoURL
		fields++
	}

	if fields > 0 {
		confidence := float64(fields) / 8.0 // 8 possible fields
		if confidence > 1.0 {
			confidence = 1.0
		}
		result.Sources = append(result.Sources, DiscoverySource{
			File:       ".vrooli/branding.json",
			Type:       "branding_json",
			Confidence: confidence,
			Fields:     fields,
		})
	}
}

// discoverManifest reads ui/public/manifest.json for PWA branding. [REQ:BM-REQ-DISC-SCAN]
func (h *Handlers) discoverManifest(scenarioDir string, result *DiscoveryResult, draft *domain.Brand) {
	path := filepath.Join(scenarioDir, "ui", "public", "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return
	}

	fields := 0
	if name, ok := manifest["name"].(string); ok && name != "" {
		id := ensureIdentity(draft)
		if id.DisplayName == "" {
			id.DisplayName = name
		}
		fields++
	}
	if desc, ok := manifest["description"].(string); ok && desc != "" {
		if draft.Description == "" {
			draft.Description = desc
		}
		fields++
	}
	if bg, ok := manifest["background_color"].(string); ok && bg != "" {
		c := ensureColors(draft)
		if c.Background == "" {
			c.Background = bg
		}
		fields++
	}
	if tc, ok := manifest["theme_color"].(string); ok && tc != "" {
		c := ensureColors(draft)
		if c.Primary == "" {
			c.Primary = tc
		}
		fields++
	}

	if fields > 0 {
		result.Sources = append(result.Sources, DiscoverySource{
			File:       "ui/public/manifest.json",
			Type:       "manifest",
			Confidence: 0.6,
			Fields:     fields,
		})
	}
}

// discoverThemeCSS scans for CSS custom properties with brand-related names. [REQ:BM-REQ-DISC-SCAN]
func (h *Handlers) discoverThemeCSS(scenarioDir string, result *DiscoveryResult, draft *domain.Brand) {
	// Check common theme file locations
	candidates := []string{
		filepath.Join("ui", "src", "styles", "theme.css"),
		filepath.Join("ui", "src", "styles", "brand.css"),
		filepath.Join("ui", "src", "index.css"),
	}

	for _, relPath := range candidates {
		fullPath := filepath.Join(scenarioDir, relPath)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		content := string(data)
		fields := 0

		// Look for color-related CSS custom properties
		colorProps := map[string]*string{
			"--brand-primary":    nil,
			"--brand-secondary":  nil,
			"--brand-accent":     nil,
			"--brand-background": nil,
			"--brand-text":       nil,
			"--color-primary":    nil,
			"--primary":          nil,
		}

		for prop := range colorProps {
			if strings.Contains(content, prop) {
				fields++
			}
		}

		if fields > 0 {
			result.Sources = append(result.Sources, DiscoverySource{
				File:       relPath,
				Type:       "theme_css",
				Confidence: 0.7,
				Fields:     fields,
			})
		}
	}
}

// discoverAssets scans for common brand asset files. [REQ:BM-REQ-DISC-SCAN]
func (h *Handlers) discoverAssets(scenarioDir string, result *DiscoveryResult, draft *domain.Brand) {
	publicDir := filepath.Join(scenarioDir, "ui", "public")

	assetPatterns := map[string]string{
		"favicon": "favicon",
		"logo":    "logo",
	}

	for assetType, prefix := range assetPatterns {
		entries, err := os.ReadDir(publicDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name := strings.ToLower(entry.Name())
			if strings.HasPrefix(name, prefix) {
				relPath := filepath.Join("ui", "public", entry.Name())
				result.Sources = append(result.Sources, DiscoverySource{
					File:       relPath,
					Type:       "asset",
					Confidence: 0.8,
					Fields:     1,
				})

				id := ensureIdentity(draft)
				switch assetType {
				case "favicon":
					if id.FaviconPath == "" {
						id.FaviconPath = relPath
					}
				case "logo":
					if id.LogoPath == "" {
						id.LogoPath = relPath
					}
				}
				break // only first match per type
			}
		}
	}
}
