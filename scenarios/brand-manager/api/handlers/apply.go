// Package handlers - programmatic brand application to scenario files.
// [REQ:BM-REQ-APPLY-CSS] [REQ:BM-REQ-APPLY-JSON] [REQ:BM-REQ-APPLY-ASSETS] [REQ:BM-REQ-APPLY-PARTIAL]
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"brand-manager/apierr"
	"brand-manager/domain"

	"github.com/gorilla/mux"
)

// ApplyRequest specifies what to apply and to which scenario.
type ApplyRequest struct {
	ScenarioName string   `json:"scenario_name"`
	Elements     []string `json:"elements,omitempty"` // empty = all
}

// ApplyResult reports what was applied.
type ApplyResult struct {
	Scenario     string        `json:"scenario"`
	BrandID      string        `json:"brand_id"`
	BrandVersion int           `json:"brand_version"`
	Applied      []ApplyAction `json:"applied"`
	Skipped      []SkipReason  `json:"skipped,omitempty"`
	DryRun       bool          `json:"dry_run,omitempty"`
}

// ApplyAction records a single application action.
type ApplyAction struct {
	Type    string `json:"type"`    // "css", "json", "asset"
	File    string `json:"file"`    // relative path within scenario
	Element string `json:"element"` // which brand element was applied
}

// SkipReason records why an element was skipped.
type SkipReason struct {
	Element string `json:"element"`
	Reason  string `json:"reason"`
}

// allApplyElements lists all supported application elements.
var allApplyElements = []string{"colors", "typography", "identity", "favicon", "logo"}

// ApplyBrand handles POST /api/v1/brands/{id}/apply. [REQ:BM-REQ-APPLY-CSS] [REQ:BM-REQ-APPLY-JSON] [REQ:BM-REQ-APPLY-ASSETS] [REQ:BM-REQ-APPLY-PARTIAL]
func (h *Handlers) ApplyBrand(w http.ResponseWriter, r *http.Request) {
	brandID := mux.Vars(r)["id"]

	brand, done := getOrNotFound(w, func() (*domain.Brand, error) {
		return h.brands.GetByID(r.Context(), brandID)
	}, "brand")
	if done {
		return
	}

	var req ApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.Write(w, apierr.Validation("invalid request body"))
		return
	}
	if req.ScenarioName == "" {
		apierr.Write(w, apierr.Validation("scenario_name is required"))
		return
	}

	scenarioDir, done := h.resolveScenarioDir(w, req.ScenarioName)
	if done {
		return
	}

	// Determine which elements to apply [REQ:BM-REQ-APPLY-PARTIAL]
	elements := req.Elements
	if len(elements) == 0 {
		elements = allApplyElements
	}

	dryRun := isDryRun(r)
	result := ApplyResult{
		Scenario:     req.ScenarioName,
		BrandID:      brand.ID,
		BrandVersion: brand.Version,
		DryRun:       dryRun,
	}

	for _, elem := range elements {
		actions, skip := h.applyElement(brand, scenarioDir, elem, dryRun)
		result.Applied = append(result.Applied, actions...)
		if skip != nil {
			result.Skipped = append(result.Skipped, *skip)
		}
	}

	// Create assignment record if not dry run and something was applied
	if !dryRun && len(result.Applied) > 0 {
		assignment := &domain.Assignment{
			ID:           h.newID(),
			BrandID:      brand.ID,
			ScenarioName: req.ScenarioName,
			BrandVersion: brand.Version,
			Elements:     elements,
		}
		if err := h.assignments.Create(r.Context(), assignment); err != nil {
			apierr.Write(w, apierr.Internal("create assignment", err))
			return
		}
	}

	writeJSON(w, http.StatusOK, result)
}

// applyElement applies a single brand element to the scenario directory.
func (h *Handlers) applyElement(brand *domain.Brand, scenarioDir, element string, dryRun bool) ([]ApplyAction, *SkipReason) {
	switch element {
	case "colors":
		return h.applyCSS(brand, scenarioDir, dryRun)
	case "typography":
		return h.applyTypographyCSS(brand, scenarioDir, dryRun)
	case "identity":
		return h.applyJSON(brand, scenarioDir, dryRun)
	case "favicon":
		return h.applyAsset(brand, scenarioDir, "favicon", dryRun)
	case "logo":
		return h.applyAsset(brand, scenarioDir, "logo", dryRun)
	default:
		return nil, &SkipReason{Element: element, Reason: "unknown element"}
	}
}

// applyCSS writes CSS custom properties with brand-manager comment markers. [REQ:BM-REQ-APPLY-CSS]
func (h *Handlers) applyCSS(brand *domain.Brand, scenarioDir string, dryRun bool) ([]ApplyAction, *SkipReason) {
	if brand.Colors == nil {
		return nil, &SkipReason{Element: "colors", Reason: "no colors defined"}
	}

	cssContent := generateColorCSS(brand.Colors)
	relPath := filepath.Join("ui", "src", "styles", "brand.css")
	fullPath := filepath.Join(scenarioDir, relPath)

	if dryRun {
		return []ApplyAction{{Type: "css", File: relPath, Element: "colors"}}, nil
	}

	if err := writeFileAtomic(fullPath, []byte(cssContent)); err != nil {
		return nil, &SkipReason{Element: "colors", Reason: "write failed: " + err.Error()}
	}

	return []ApplyAction{{Type: "css", File: relPath, Element: "colors"}}, nil
}

// applyTypographyCSS writes typography CSS custom properties. [REQ:BM-REQ-APPLY-CSS]
func (h *Handlers) applyTypographyCSS(brand *domain.Brand, scenarioDir string, dryRun bool) ([]ApplyAction, *SkipReason) {
	if brand.Typography == nil {
		return nil, &SkipReason{Element: "typography", Reason: "no typography defined"}
	}

	cssContent := generateTypographyCSS(brand.Typography)
	relPath := filepath.Join("ui", "src", "styles", "brand.css")
	fullPath := filepath.Join(scenarioDir, relPath)

	if dryRun {
		return []ApplyAction{{Type: "css", File: relPath, Element: "typography"}}, nil
	}

	// Append to existing brand.css or create new
	existing, _ := os.ReadFile(fullPath)
	combined := string(existing) + "\n" + cssContent

	if err := writeFileAtomic(fullPath, []byte(combined)); err != nil {
		return nil, &SkipReason{Element: "typography", Reason: "write failed: " + err.Error()}
	}

	return []ApplyAction{{Type: "css", File: relPath, Element: "typography"}}, nil
}

// applyJSON updates manifest.json with _brand keys. [REQ:BM-REQ-APPLY-JSON]
func (h *Handlers) applyJSON(brand *domain.Brand, scenarioDir string, dryRun bool) ([]ApplyAction, *SkipReason) {
	if brand.Identity == nil {
		return nil, &SkipReason{Element: "identity", Reason: "no identity defined"}
	}

	relPath := filepath.Join("ui", "public", "manifest.json")
	fullPath := filepath.Join(scenarioDir, relPath)

	// Read existing manifest or start fresh
	manifest := make(map[string]interface{})
	if data, err := os.ReadFile(fullPath); err == nil {
		json.Unmarshal(data, &manifest)
	}

	// Apply _brand keys for tracking
	if brand.Identity.DisplayName != "" {
		manifest["name"] = brand.Identity.DisplayName
		manifest["short_name"] = brand.Identity.DisplayName
		manifest["_brand_display_name"] = brand.Identity.DisplayName
	}
	if brand.Identity.Tagline != "" {
		manifest["description"] = brand.Identity.Tagline
		manifest["_brand_tagline"] = brand.Identity.Tagline
	}
	manifest["_brand_id"] = brand.ID
	manifest["_brand_version"] = brand.Version

	if dryRun {
		return []ApplyAction{{Type: "json", File: relPath, Element: "identity"}}, nil
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, &SkipReason{Element: "identity", Reason: "marshal failed: " + err.Error()}
	}

	if err := writeFileAtomic(fullPath, data); err != nil {
		return nil, &SkipReason{Element: "identity", Reason: "write failed: " + err.Error()}
	}

	return []ApplyAction{{Type: "json", File: relPath, Element: "identity"}}, nil
}

// applyAsset copies a brand asset (favicon/logo) to the scenario's public directory. [REQ:BM-REQ-APPLY-ASSETS]
func (h *Handlers) applyAsset(brand *domain.Brand, scenarioDir, assetType string, dryRun bool) ([]ApplyAction, *SkipReason) {
	if brand.Identity == nil {
		return nil, &SkipReason{Element: assetType, Reason: "no identity defined"}
	}

	var sourcePath string
	switch assetType {
	case "favicon":
		sourcePath = brand.Identity.FaviconPath
	case "logo":
		sourcePath = brand.Identity.LogoPath
	}

	if sourcePath == "" {
		return nil, &SkipReason{Element: assetType, Reason: "no " + assetType + " path defined"}
	}

	// Resolve against asset base path if relative
	if !filepath.IsAbs(sourcePath) {
		sourcePath = filepath.Join(h.cfg.AssetBasePath, brand.ID, sourcePath)
	}

	relPath := filepath.Join("ui", "public", filepath.Base(sourcePath))

	if dryRun {
		return []ApplyAction{{Type: "asset", File: relPath, Element: assetType}}, nil
	}

	destPath := filepath.Join(scenarioDir, relPath)
	if err := copyFile(sourcePath, destPath); err != nil {
		return nil, &SkipReason{Element: assetType, Reason: "copy failed: " + err.Error()}
	}

	return []ApplyAction{{Type: "asset", File: relPath, Element: assetType}}, nil
}

// nameValue pairs a CSS custom property name with its value.
type nameValue struct{ name, value string }

// colorPairs returns the canonical color name→value pairs for a brand's color system.
func colorPairs(c *domain.Colors) []nameValue {
	return []nameValue{
		{"primary", c.Primary},
		{"secondary", c.Secondary},
		{"accent", c.Accent},
		{"background", c.Background},
		{"surface", c.Surface},
		{"text", c.Text},
		{"error", c.Error},
	}
}

// typographyPairs returns the canonical typography name→value pairs.
func typographyPairs(t *domain.Typography) []nameValue {
	return []nameValue{
		{"heading-font", t.HeadingFont},
		{"body-font", t.BodyFont},
		{"mono-font", t.MonoFont},
		{"base-font-size", t.BaseFontSize},
	}
}

// generateCSSBlock builds a :root CSS block from name/value pairs with brand-manager markers.
func generateCSSBlock(section string, pairs []nameValue) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("/* brand-manager:%s - Auto-generated brand %s */\n", section, section))
	b.WriteString(":root {\n")
	for _, p := range pairs {
		if p.value != "" {
			b.WriteString(fmt.Sprintf("  --brand-%s: %s; /* brand-manager:%s */\n", p.name, p.value, p.name))
		}
	}
	b.WriteString("}\n")
	return b.String()
}

// generateColorCSS produces CSS custom properties from brand colors.
func generateColorCSS(colors *domain.Colors) string {
	return generateCSSBlock("colors", colorPairs(colors))
}

// generateTypographyCSS produces CSS custom properties from brand typography.
func generateTypographyCSS(typo *domain.Typography) string {
	return generateCSSBlock("typography", typographyPairs(typo))
}

// writeFileAtomic writes data to a file atomically (temp + rename).
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}

	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// copyFile copies src to dst atomically.
func copyFile(src, dst string) error {
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	out.Close()
	if err != nil {
		os.Remove(tmp)
		return err
	}

	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
