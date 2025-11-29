package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Template represents a landing page template with full metadata
type Template struct {
	ID                  string                 `json:"id"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	Version             string                 `json:"version"`
	Metadata            map[string]interface{} `json:"metadata"`
	Sections            map[string]interface{} `json:"sections"`
	MetricsHooks        []MetricHook           `json:"metrics_hooks"`
	CustomizationSchema map[string]interface{} `json:"customization_schema"`
	FrontendAesthetics  map[string]interface{} `json:"frontend_aesthetics,omitempty"`
	TechStack           map[string]interface{} `json:"tech_stack,omitempty"`
	GeneratedStructure  map[string]interface{} `json:"generated_structure,omitempty"`
}

// TemplateCatalog represents the template registry index
type TemplateCatalog struct {
	Version     string                      `json:"version"`
	Description string                      `json:"description"`
	Categories  map[string]CategoryInfo     `json:"categories"`
	Templates   map[string]TemplateRef      `json:"templates"`
}

// CategoryInfo describes a template category
type CategoryInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Templates   []string `json:"templates"`
}

// TemplateRef is a reference to a template in the catalog
type TemplateRef struct {
	Path         string   `json:"path"`
	Category     string   `json:"category"`
	Version      string   `json:"version"`
	SectionsUsed []string `json:"sections_used"`
}

// MetricHook defines an analytics event type
type MetricHook struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	EventType      string   `json:"event_type"`
	RequiredFields []string `json:"required_fields"`
}

// TemplateService handles template operations
type TemplateService struct {
	templatesDir  string
	cache         map[string]Template
	cacheMux      sync.RWMutex
	cacheExpiry   time.Time
	cacheDuration time.Duration
}

// GeneratedScenario summarizes a generated landing scenario for the factory UI.
type GeneratedScenario struct {
	ScenarioID      string `json:"scenario_id"`
	Name            string `json:"name"`
	TemplateID      string `json:"template_id,omitempty"`
	TemplateVersion string `json:"template_version,omitempty"`
	Path            string `json:"path"`
	Status          string `json:"status"`
	GeneratedAt     string `json:"generated_at,omitempty"`
}

// NewTemplateService creates a template service instance
func NewTemplateService() *TemplateService {
	// Check for test environment override
	if testDir := os.Getenv("TEMPLATES_DIR"); testDir != "" {
		return &TemplateService{
			templatesDir:  testDir,
			cache:         make(map[string]Template),
			cacheDuration: 5 * time.Minute, // Cache templates for 5 minutes
		}
	}

	// Determine templates directory relative to binary location
	execPath, err := os.Executable()
	if err != nil {
		execPath = "."
	}
	execDir := filepath.Dir(execPath)
	templatesDir := filepath.Join(execDir, "templates")

	// Fallback to current directory + templates if binary path templates don't exist
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		templatesDir = "templates"
	}

	return &TemplateService{
		templatesDir:  templatesDir,
		cache:         make(map[string]Template),
		cacheDuration: 5 * time.Minute, // Cache templates for 5 minutes
	}
}

// ListTemplates returns all available templates (with caching)
// Prefers catalog-based structure (catalog.json + catalog/{category}/*.json)
// Falls back to legacy flat structure ({id}.json) for backward compatibility
func (ts *TemplateService) ListTemplates() ([]Template, error) {
	// Check if cache is valid - skip if cache not initialized
	if ts.cache != nil {
		ts.cacheMux.RLock()
		cacheValid := time.Now().Before(ts.cacheExpiry) && len(ts.cache) > 0
		ts.cacheMux.RUnlock()

		// If cache is valid, return cached templates
		if cacheValid {
			ts.cacheMux.RLock()
			templates := make([]Template, 0, len(ts.cache))
			for _, tmpl := range ts.cache {
				templates = append(templates, tmpl)
			}
			ts.cacheMux.RUnlock()
			return templates, nil
		}
	}

	var templates []Template
	newCache := make(map[string]Template)

	// Try catalog-based structure first (screaming architecture)
	catalogPath := filepath.Join(ts.templatesDir, "catalog.json")
	if _, err := os.Stat(catalogPath); err == nil {
		catalog, err := ts.loadCatalog(catalogPath)
		if err == nil {
			for id, ref := range catalog.Templates {
				templatePath := filepath.Join(ts.templatesDir, ref.Path)
				template, err := ts.loadTemplate(templatePath)
				if err != nil {
					logStructured("template_load_error", map[string]interface{}{
						"id":    id,
						"path":  ref.Path,
						"error": err.Error(),
					})
					continue
				}
				templates = append(templates, template)
				newCache[template.ID] = template
			}
		}
	}

	// Fall back to legacy flat structure if no catalog templates found
	if len(templates) == 0 {
		entries, err := os.ReadDir(ts.templatesDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read templates directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			// Skip catalog.json and other non-template files
			if entry.Name() == "catalog.json" {
				continue
			}

			templatePath := filepath.Join(ts.templatesDir, entry.Name())
			template, err := ts.loadTemplate(templatePath)
			if err != nil {
				logStructured("template_load_error", map[string]interface{}{
					"file":  entry.Name(),
					"error": err.Error(),
				})
				continue
			}

			templates = append(templates, template)
			newCache[template.ID] = template
		}
	}

	// Update cache (write lock) - skip if cache not initialized
	if ts.cache != nil {
		ts.cacheMux.Lock()
		ts.cache = newCache
		ts.cacheExpiry = time.Now().Add(ts.cacheDuration)
		ts.cacheMux.Unlock()
	}

	return templates, nil
}

// loadCatalog reads and parses the template catalog index
func (ts *TemplateService) loadCatalog(path string) (*TemplateCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read catalog: %w", err)
	}

	var catalog TemplateCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse catalog JSON: %w", err)
	}

	return &catalog, nil
}

// GetTemplate returns a specific template by ID (with caching)
// Prefers catalog-based lookup, falls back to legacy flat structure
func (ts *TemplateService) GetTemplate(id string) (*Template, error) {
	// Check cache first (read lock) - skip if cache not initialized
	if ts.cache != nil {
		ts.cacheMux.RLock()
		if time.Now().Before(ts.cacheExpiry) {
			if cached, ok := ts.cache[id]; ok {
				ts.cacheMux.RUnlock()
				return &cached, nil
			}
		}
		ts.cacheMux.RUnlock()
	}

	// Try catalog-based lookup first (screaming architecture)
	var templatePath string
	catalogPath := filepath.Join(ts.templatesDir, "catalog.json")
	if _, err := os.Stat(catalogPath); err == nil {
		catalog, err := ts.loadCatalog(catalogPath)
		if err == nil {
			if ref, ok := catalog.Templates[id]; ok {
				templatePath = filepath.Join(ts.templatesDir, ref.Path)
			}
		}
	}

	// Fall back to legacy flat structure
	if templatePath == "" {
		templatePath = filepath.Join(ts.templatesDir, id+".json")
	}

	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("template not found: %s", id)
	}

	template, err := ts.loadTemplate(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	// Update cache (write lock) - skip if cache not initialized
	if ts.cache != nil {
		ts.cacheMux.Lock()
		ts.cache[id] = template
		if ts.cacheExpiry.IsZero() || time.Now().After(ts.cacheExpiry) {
			ts.cacheExpiry = time.Now().Add(ts.cacheDuration)
		}
		ts.cacheMux.Unlock()
	}

	return &template, nil
}

// loadTemplate reads and parses a template JSON file
func (ts *TemplateService) loadTemplate(path string) (Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Template{}, fmt.Errorf("failed to read file: %w", err)
	}

	var template Template
	if err := json.Unmarshal(data, &template); err != nil {
		return Template{}, fmt.Errorf("failed to parse template JSON: %w", err)
	}

	return template, nil
}

// GenerateScenario creates a new landing page scenario from a template
func (ts *TemplateService) GenerateScenario(templateID, name, slug string, options map[string]interface{}) (map[string]interface{}, error) {
	template, err := ts.GetTemplate(templateID)
	if err != nil {
		return nil, err
	}

	if name == "" || slug == "" {
		return nil, fmt.Errorf("name and slug are required")
	}

	outputDir, err := resolveGenerationPath(slug)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve generation path: %w", err)
	}

	if isDryRun(options) {
		return buildDryRunResponse(slug, name, template.ID, outputDir), nil
	}

	if err := ts.materializeScenario(outputDir, templateID, template, name, slug); err != nil {
		return nil, err
	}

	validation := validateGeneratedScenario(outputDir)
	return buildCreatedResponse(slug, name, template.ID, outputDir, validation), nil
}

func isDryRun(options map[string]interface{}) bool {
	if options == nil {
		return false
	}
	val, ok := options["dry_run"]
	if !ok {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true")
	default:
		return false
	}
}

func buildDryRunResponse(slug, name, templateID, outputDir string) map[string]interface{} {
	plannedPaths := []string{
		outputDir,
		filepath.Join(outputDir, "api"),
		filepath.Join(outputDir, "ui"),
		filepath.Join(outputDir, "requirements"),
		filepath.Join(outputDir, ".vrooli"),
		filepath.Join(outputDir, ".vrooli", "template.json"),
		filepath.Join(outputDir, "Makefile"),
		filepath.Join(outputDir, "PRD.md"),
		filepath.Join(outputDir, "README.md"),
	}

	return map[string]interface{}{
		"scenario_id": slug,
		"name":        name,
		"template":    templateID,
		"path":        outputDir,
		"status":      "dry_run",
		"plan": map[string]interface{}{
			"paths": plannedPaths,
		},
		"next_steps": []string{
			"Re-run without --dry-run to materialize the scenario",
			"Move the folder to /scenarios/<slug> when ready to run independently",
			"Start scenario: vrooli scenario start " + slug,
		},
	}
}

func buildCreatedResponse(slug, name, templateID, outputDir string, validation map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"scenario_id": slug,
		"name":        name,
		"template":    templateID,
		"path":        outputDir,
		"status":      "created",
		"validation":  validation,
		"next_steps": []string{
			"Review generated scenario files under generated/" + slug,
			"Move the folder to /scenarios/<slug> when ready to run independently",
			"Start scenario: vrooli scenario start " + slug,
			"Preview public landing: http://localhost:${UI_PORT}/ (after start)",
			"Preview admin portal: http://localhost:${UI_PORT}/admin (after start, login required)",
			"Customize content via CLI or admin portal in the generated scenario",
		},
	}
}

func (ts *TemplateService) materializeScenario(outputDir, templateID string, template *Template, name, slug string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := scaffoldScenario(outputDir); err != nil {
		return fmt.Errorf("failed to scaffold scenario: %w", err)
	}

	// Generate description for the scenario
	description := fmt.Sprintf("Generated landing page scenario from template '%s'", name)

	// Substitute template placeholders ({{SCENARIO_ID}}, {{SCENARIO_DISPLAY_NAME}}, {{SCENARIO_DESCRIPTION}})
	// across all text files. This follows the react-vite template pattern.
	if err := substituteTemplatePlaceholders(outputDir, slug, name, description); err != nil {
		return fmt.Errorf("failed to substitute template placeholders: %w", err)
	}

	if err := writeTemplateMetadata(outputDir, templateID, template); err != nil {
		return err
	}

	if err := writeTemplateProvenance(outputDir, template); err != nil {
		return fmt.Errorf("failed to stamp template provenance: %w", err)
	}

	if err := rewriteServiceConfig(outputDir, name, slug); err != nil {
		return fmt.Errorf("failed to rewrite service config: %w", err)
	}

	return nil
}

func writeTemplateMetadata(outputDir, templateID string, template *Template) error {
	templateOut := filepath.Join(outputDir, "api", "templates", templateID+".json")
	if err := os.MkdirAll(filepath.Dir(templateOut), 0o755); err != nil {
		return fmt.Errorf("failed to create template metadata directory: %w", err)
	}
	templateData, _ := json.MarshalIndent(template, "", "  ")
	if err := os.WriteFile(templateOut, templateData, 0o644); err != nil {
		return fmt.Errorf("failed to write template metadata: %w", err)
	}
	return nil
}

// resolveGenerationPath returns an absolute path for the generated scenario.
// Priority: GEN_OUTPUT_DIR env override → ../generated/<slug> relative to API binary.
func resolveGenerationPath(slug string) (string, error) {
	root, err := generationRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, slug), nil
}

// generationRoot resolves the base directory for generated scenarios.
func generationRoot() (string, error) {
	if override := strings.TrimSpace(os.Getenv("GEN_OUTPUT_DIR")); override != "" {
		return filepath.Abs(override)
	}

	scenarioRoot, err := resolveScenarioRoot()
	if err != nil {
		return "", err
	}
	defaultRoot := filepath.Join(scenarioRoot, "generated")
	return filepath.Abs(defaultRoot)
}

// resolveScenarioRoot finds the landing-manager scenario root directory
func resolveScenarioRoot() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)
	// Assume binary resides in /scenarios/landing-manager/api
	return filepath.Dir(execDir), nil
}

// scaffoldScenario copies the existing landing template assets into the output directory.
// It creates a self-contained scenario that can later be moved into /scenarios/<slug>.
func scaffoldScenario(outputDir string) error {
	// Allow explicit override for tooling runs outside lifecycle
	if override := strings.TrimSpace(os.Getenv("TEMPLATE_PAYLOAD_DIR")); override != "" {
		if err := copyTemplatePayload(override, outputDir); err != nil {
			return err
		}
		return nil
	}

	scenarioRoot, err := resolveScenarioRoot()
	if err != nil {
		return err
	}

	// Prefer dedicated template from repo root (scripts/scenarios/templates/landing-page-react-vite)
	repoRoot := filepath.Dir(filepath.Dir(scenarioRoot))
	templatePayload := filepath.Join(repoRoot, "scripts", "scenarios", "templates", "landing-page-react-vite")
	if _, err := os.Stat(templatePayload); err == nil {
		return copyTemplatePayload(templatePayload, outputDir)
	}

	// Fallback: copy from current scenario root (useful in tests with TEMPLATE_PAYLOAD_DIR)
	return copyTemplatePayload(scenarioRoot, outputDir)
}

// copyTemplatePayload copies template assets from the resolved template root into the output directory.
func copyTemplatePayload(scenarioRoot, outputDir string) error {
	// Source directories to copy
	sources := []struct {
		src string
		dst string
	}{
		{src: filepath.Join(scenarioRoot, "api"), dst: filepath.Join(outputDir, "api")},
		{src: filepath.Join(scenarioRoot, "ui"), dst: filepath.Join(outputDir, "ui")},
		{src: filepath.Join(scenarioRoot, "requirements"), dst: filepath.Join(outputDir, "requirements")},
		{src: filepath.Join(scenarioRoot, "initialization"), dst: filepath.Join(outputDir, "initialization")},
		{src: filepath.Join(scenarioRoot, ".vrooli"), dst: filepath.Join(outputDir, ".vrooli")},
		{src: filepath.Join(scenarioRoot, "Makefile"), dst: filepath.Join(outputDir, "Makefile")},
		{src: filepath.Join(scenarioRoot, "PRD.md"), dst: filepath.Join(outputDir, "PRD.md")},
	}

	for _, entry := range sources {
		if err := copyDir(entry.src, entry.dst); err != nil {
			return fmt.Errorf("failed to copy %s: %w", entry.src, err)
		}
	}

	// Remove factory-only UI surface from generated scenario
	factoryPage := filepath.Join(outputDir, "ui", "src", "pages", "FactoryHome.tsx")
	_ = os.Remove(factoryPage)

	// Adjust UI entrypoint for generated landing: default to landing experience
	appPath := filepath.Join(outputDir, "ui", "src", "App.tsx")
	if err := writeLandingApp(appPath); err != nil {
		return fmt.Errorf("failed to write landing App.tsx: %w", err)
	}

	// Fix workspace dependencies to point to absolute paths
	// This allows generated scenarios to run from generated/ folder via --path flag
	if err := fixWorkspaceDependencies(outputDir); err != nil {
		return fmt.Errorf("failed to fix workspace dependencies: %w", err)
	}

	// Write a minimal README for the generated scenario
	readme := `# Generated Landing Scenario

This scenario was generated by landing-manager. It contains the landing page (public) and admin portal.

## Run
make start

Public: http://localhost:<UI_PORT>/
Admin:  http://localhost:<UI_PORT>/admin

## Notes
- Generated inside landing-manager/generated/. Move this folder to /scenarios/<slug> to run independently.
- Update environment variables in .env or .vrooli/service.json as needed.
`
	if err := os.WriteFile(filepath.Join(outputDir, "README.md"), []byte(readme), 0o644); err != nil {
		return fmt.Errorf("failed to write generated README: %w", err)
	}

	return nil
}

// writeLandingApp rewrites App.tsx for generated scenarios to load the landing experience by default.
func writeLandingApp(path string) error {
	content := `import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './contexts/AuthContext';
import { VariantProvider } from './contexts/VariantContext';
import { ProtectedRoute } from './components/ProtectedRoute';
import { AdminLogin } from './pages/AdminLogin';
import { AdminHome } from './pages/AdminHome';
import { AdminAnalytics } from './pages/AdminAnalytics';
import { Customization } from './pages/Customization';
import { VariantEditor } from './pages/VariantEditor';
import { SectionEditor } from './pages/SectionEditor';
import { AgentCustomization } from './pages/AgentCustomization';
import PublicHome from './pages/PublicHome';

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <VariantProvider>
          <Routes>
            <Route path="/" element={<PublicHome />} />
            <Route path="/health" element={<PublicHome />} />

            <Route path="/admin/login" element={<AdminLogin />} />

            <Route
              path="/admin"
              element={
                <ProtectedRoute>
                  <AdminHome />
                </ProtectedRoute>
              }
            />

            <Route
              path="/admin/analytics"
              element={
                <ProtectedRoute>
                  <AdminAnalytics />
                </ProtectedRoute>
              }
            />
            <Route
              path="/admin/analytics/:variantSlug"
              element={
                <ProtectedRoute>
                  <AdminAnalytics />
                </ProtectedRoute>
              }
            />

            <Route
              path="/admin/customization"
              element={
                <ProtectedRoute>
                  <Customization />
                </ProtectedRoute>
              }
            />
            <Route
              path="/admin/customization/agent"
              element={
                <ProtectedRoute>
                  <AgentCustomization />
                </ProtectedRoute>
              }
            />
            <Route
              path="/admin/customization/variants/:slug"
              element={
                <ProtectedRoute>
                  <VariantEditor />
                </ProtectedRoute>
              }
            />
            <Route
              path="/admin/customization/variants/:variantSlug/sections/:sectionId"
              element={
                <ProtectedRoute>
                  <SectionEditor />
                </ProtectedRoute>
              }
            />

            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </VariantProvider>
      </AuthProvider>
    </BrowserRouter>
  );
}
`
	return os.WriteFile(path, []byte(content), 0o644)
}

// fixWorkspaceDependencies rewrites package.json to use absolute paths for workspace dependencies.
// This allows generated scenarios in staging (generated/) to reference shared packages.
// If package.json doesn't exist, this is a no-op (scaffolds may not have UI).
func fixWorkspaceDependencies(outputDir string) error {
	packagePath := filepath.Join(outputDir, "ui", "package.json")

	// Skip if package.json doesn't exist (e.g., API-only scenarios or minimal scaffolds)
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(packagePath)
	if err != nil {
		return fmt.Errorf("failed to read package.json: %w", err)
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return fmt.Errorf("failed to parse package.json: %w", err)
	}

	packagesDir, err := resolvePackagesDir()
	if err != nil {
		return err
	}

	// Fix dependencies section
	if deps, ok := pkg["dependencies"].(map[string]interface{}); ok {
		for name, value := range deps {
			if strValue, ok := value.(string); ok {
				// Replace relative file: references with absolute paths
				// Security note: The literal "../../../" below is NOT a vulnerability - it's detecting
				// workspace dependencies to validate them. resolvePackagePath() sanitizes all inputs
				// with filepath.Clean(), rejects ".." sequences, and validates against directory escape.
				if strings.HasPrefix(strValue, "file:../../../packages/") {
					packageName := strings.TrimPrefix(strValue, "file:../../../packages/")
					// Security: resolvePackagePath validates and prevents path traversal
					absolutePath, valid := resolvePackagePath(packageName, packagesDir)
					if !valid {
						continue // Skip invalid paths
					}
					deps[name] = "file:" + absolutePath
				}
			}
		}
	}

	// Write updated package.json
	updatedData, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated package.json: %w", err)
	}

	if err := os.WriteFile(packagePath, updatedData, 0o644); err != nil {
		return fmt.Errorf("failed to write updated package.json: %w", err)
	}

	return nil
}

// resolvePackagesDir finds the absolute path to the Vrooli packages directory
func resolvePackagesDir() (string, error) {
	scenarioRoot, err := resolveScenarioRoot()
	if err != nil {
		return "", err
	}
	vrooliRoot := getVrooliRootFromScenario(scenarioRoot)
	return filepath.Join(vrooliRoot, "packages"), nil
}

func getVrooliRootFromScenario(scenarioRoot string) string {
	return filepath.Dir(filepath.Dir(scenarioRoot)) // /path/to/Vrooli
}

// resolvePackagePath validates and resolves a package path securely
func resolvePackagePath(packageName, packagesDir string) (string, bool) {
	// Security: Clean the path to prevent traversal attacks
	cleanedName := filepath.Clean(packageName)

	// Reject paths containing traversal attempts
	if strings.Contains(cleanedName, "..") {
		return "", false
	}

	absolutePath := filepath.Join(packagesDir, cleanedName)

	// Verify the resolved path stays within the packages directory
	if !isPathWithinDirectory(absolutePath, packagesDir) {
		return "", false
	}

	return absolutePath, true
}

func isPathWithinDirectory(path, baseDir string) bool {
	cleanPath := filepath.Clean(path)
	cleanBase := filepath.Clean(baseDir)
	return strings.HasPrefix(cleanPath, cleanBase)
}

// rewriteServiceConfig removes factory-only steps from the generated scenario's service.json.
// Note: Most customization (names, binary names, paths) is handled by substituteTemplatePlaceholders.
// This function only removes factory-specific steps that shouldn't appear in generated scenarios.
func rewriteServiceConfig(outputDir, name, slug string) error {
	path := filepath.Join(outputDir, ".vrooli", "service.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse service.json: %w", err)
	}

	// Remove factory-only steps (e.g., install-cli) from generated lifecycle
	if lifecycle, ok := cfg["lifecycle"].(map[string]interface{}); ok {
		if setup, ok := lifecycle["setup"].(map[string]interface{}); ok {
			if steps, ok := setup["steps"].([]interface{}); ok {
				var filtered []interface{}
				for _, s := range steps {
					stepMap, ok := s.(map[string]interface{})
					if ok {
						if stepName, _ := stepMap["name"].(string); stepName == "install-cli" {
							continue
						}
					}
					filtered = append(filtered, s)
				}
				setup["steps"] = filtered
			}
		}
	}

	updated, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated service.json: %w", err)
	}

	return os.WriteFile(path, updated, 0o644)
}

// substituteTemplatePlaceholders walks through all text files in the output directory
// and replaces template placeholders ({{SCENARIO_ID}}, {{SCENARIO_DISPLAY_NAME}}, {{SCENARIO_DESCRIPTION}})
// with actual values. This follows the react-vite template pattern.
func substituteTemplatePlaceholders(outputDir, slug, displayName, description string) error {
	// File extensions to process (text files that might contain placeholders)
	textExtensions := map[string]bool{
		".go":   true,
		".json": true,
		".js":   true,
		".ts":   true,
		".tsx":  true,
		".jsx":  true,
		".md":   true,
		".html": true,
		".css":  true,
		".yaml": true,
		".yml":  true,
		".toml": true,
		".mod":  true,
		".sum":  true,
		".env":  true,
		".sh":   true,
		"":      true, // Makefile has no extension
	}

	// Directories to skip
	skipDirs := map[string]bool{
		"node_modules": true,
		"dist":         true,
		"coverage":     true,
		".git":         true,
	}

	return filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories in the skip list
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if this is a text file we should process
		ext := filepath.Ext(info.Name())
		if !textExtensions[ext] {
			// Special case for Makefile (no extension)
			if info.Name() != "Makefile" {
				return nil
			}
		}

		// Read the file
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		content := string(data)
		original := content

		// Replace placeholders
		content = strings.ReplaceAll(content, "{{SCENARIO_ID}}", slug)
		content = strings.ReplaceAll(content, "{{SCENARIO_DISPLAY_NAME}}", displayName)
		content = strings.ReplaceAll(content, "{{SCENARIO_DESCRIPTION}}", description)

		// Only write if content changed
		if content != original {
			if err := os.WriteFile(path, []byte(content), info.Mode()); err != nil {
				return fmt.Errorf("failed to write %s: %w", path, err)
			}
		}

		return nil
	})
}

// writeTemplateProvenance records template id/version and generation timestamp.
func writeTemplateProvenance(outputDir string, template *Template) error {
	provenance := map[string]interface{}{
		"template_id":      template.ID,
		"template_version": template.Version,
		"generated_at":     time.Now().UTC().Format(time.RFC3339),
	}
	outPath := filepath.Join(outputDir, ".vrooli", "template.json")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(provenance, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, data, 0o644)
}

// validateGeneratedScenario performs a lightweight sanity check on required assets.
func validateGeneratedScenario(outputDir string) map[string]interface{} {
	required := []string{"api", "ui", "requirements", ".vrooli", "Makefile", "PRD.md"}
	var missing []string

	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(outputDir, rel)); err != nil {
			missing = append(missing, rel)
		}
	}

	status := "passed"
	if len(missing) > 0 {
		status = "failed"
	}

	return map[string]interface{}{
		"status":        status,
		"missing":       missing,
		"checked_paths": required,
	}
}

// ListGeneratedScenarios reports any generated scenarios under the generation root for factory UI display.
func (ts *TemplateService) ListGeneratedScenarios() ([]GeneratedScenario, error) {
	root, err := generationRoot()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []GeneratedScenario{}, nil
		}
		return nil, fmt.Errorf("failed to read generated directory: %w", err)
	}

	// Initialize as empty slice instead of nil to ensure JSON marshals as [] not null
	scenarios := make([]GeneratedScenario, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		slug := entry.Name()
		scenarioPath := filepath.Join(root, slug)
		gs := GeneratedScenario{
			ScenarioID: slug,
			Name:       slug,
			Path:       scenarioPath,
			Status:     "present",
		}

		// Provenance: template id/version/timestamp
		if provData, err := os.ReadFile(filepath.Join(scenarioPath, ".vrooli", "template.json")); err == nil {
			var prov map[string]interface{}
			if err := json.Unmarshal(provData, &prov); err == nil {
				if v, ok := prov["template_id"].(string); ok {
					gs.TemplateID = v
				}
				if v, ok := prov["template_version"].(string); ok {
					gs.TemplateVersion = v
				}
				if v, ok := prov["generated_at"].(string); ok {
					gs.GeneratedAt = v
				}
			}
		}

		// Human-friendly name from service metadata
		if svcData, err := os.ReadFile(filepath.Join(scenarioPath, ".vrooli", "service.json")); err == nil {
			var svc map[string]interface{}
			if err := json.Unmarshal(svcData, &svc); err == nil {
				if service, ok := svc["service"].(map[string]interface{}); ok {
					if display, ok := service["displayName"].(string); ok && display != "" {
						gs.Name = display
					}
				}
			}
		}

		scenarios = append(scenarios, gs)
	}

	sort.Slice(scenarios, func(i, j int) bool {
		return scenarios[i].ScenarioID < scenarios[j].ScenarioID
	})

	return scenarios, nil
}

// copyDir copies a directory tree, skipping heavy build artifacts.
func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		// handle single-file sources (e.g., Makefile, PRD.md)
		return copyFile(src, dst, info.Mode())
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Skip unwanted directories
		skipDirs := map[string]bool{
			"node_modules": true,
			"dist":         true,
			"coverage":     true,
			".git":         true,
			"generated":    true,
		}
		if info.IsDir() && skipDirs[info.Name()] {
			return filepath.SkipDir
		}

		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		return copyFile(path, target, info.Mode())
	})
}

// copyFile copies a single file preserving mode.
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return nil
}

// GetPreviewLinks generates preview URLs for a generated scenario.
// Returns deep links to the public landing page and admin portal.
func (ts *TemplateService) GetPreviewLinks(scenarioID string) (map[string]interface{}, error) {
	root, err := generationRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve generation root: %w", err)
	}

	scenarioPath := filepath.Join(root, scenarioID)
	if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("generated scenario not found: %s", scenarioID)
	}

	// Get actual allocated UI_PORT from vrooli CLI (works for both staging and production scenarios)
	cmd := exec.Command("vrooli", "scenario", "port", scenarioID, "UI_PORT")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get UI_PORT for scenario %s (is it running?): %w", scenarioID, err)
	}

	uiPort := strings.TrimSpace(string(output))
	if uiPort == "" {
		return nil, fmt.Errorf("UI_PORT not allocated for scenario %s (scenario may not be running)", scenarioID)
	}

	// Get API_PORT as well for proxy support
	apiCmd := exec.Command("vrooli", "scenario", "port", scenarioID, "API_PORT")
	apiOutput, apiErr := apiCmd.CombinedOutput()
	apiPort := ""
	if apiErr == nil {
		apiPort = strings.TrimSpace(string(apiOutput))
	}

	baseURL := fmt.Sprintf("http://localhost:%s", uiPort)

	return map[string]interface{}{
		"scenario_id": scenarioID,
		"path":        scenarioPath,
		"base_url":    baseURL,
		"ui_port":     uiPort,
		"api_port":    apiPort,
		"links": map[string]string{
			"public":      baseURL + "/",
			"admin":       baseURL + "/admin",
			"admin_login": baseURL + "/admin/login",
			"health":      baseURL + "/health",
		},
		"instructions": []string{
			fmt.Sprintf("Start the scenario: vrooli scenario start %s", scenarioID),
			fmt.Sprintf("Public landing page: %s/", baseURL),
			fmt.Sprintf("Admin portal: %s/admin", baseURL),
			"Note: Scenario must be running for preview links to work",
		},
		"notes": "Scenario can be started from staging area (generated/) - no need to promote first",
	}, nil
}

// Persona represents an agent customization profile
type Persona struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Prompt      string   `json:"prompt"`
	UseCases    []string `json:"use_cases"`
	Keywords    []string `json:"keywords"`
}

// PersonaCatalog represents the personas catalog structure
type PersonaCatalog struct {
	Personas []Persona              `json:"personas"`
	Metadata map[string]interface{} `json:"_metadata"`
}

// GetPersonas retrieves all available agent personas from the catalog
func (ts *TemplateService) GetPersonas() ([]Persona, error) {
	catalogPath := filepath.Join(ts.templatesDir, "..", "personas", "catalog.json")

	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read persona catalog: %w", err)
	}

	var catalog PersonaCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse persona catalog: %w", err)
	}

	return catalog.Personas, nil
}

// GetPersona retrieves a specific persona by ID
func (ts *TemplateService) GetPersona(id string) (*Persona, error) {
	personas, err := ts.GetPersonas()
	if err != nil {
		return nil, err
	}

	for _, p := range personas {
		if p.ID == id {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("persona not found: %s", id)
}
