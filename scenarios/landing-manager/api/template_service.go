package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	templatesDir string
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
		return &TemplateService{templatesDir: testDir}
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
		templatesDir: templatesDir,
	}
}

// ListTemplates returns all available templates
func (ts *TemplateService) ListTemplates() ([]Template, error) {
	entries, err := os.ReadDir(ts.templatesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read templates directory: %w", err)
	}

	var templates []Template
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		templatePath := filepath.Join(ts.templatesDir, entry.Name())
		template, err := ts.loadTemplate(templatePath)
		if err != nil {
			// Log error but continue processing other templates
			logStructured("template_load_error", map[string]interface{}{
				"file":  entry.Name(),
				"error": err.Error(),
			})
			continue
		}

		templates = append(templates, template)
	}

	return templates, nil
}

// GetTemplate returns a specific template by ID
func (ts *TemplateService) GetTemplate(id string) (*Template, error) {
	templatePath := filepath.Join(ts.templatesDir, id+".json")

	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("template not found: %s", id)
	}

	template, err := ts.loadTemplate(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
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

	// Validate inputs
	if name == "" || slug == "" {
		return nil, fmt.Errorf("name and slug are required")
	}

	dryRun := false
	if options != nil {
		if val, ok := options["dry_run"]; ok {
			switch v := val.(type) {
			case bool:
				dryRun = v
			case string:
				dryRun = strings.EqualFold(v, "true")
			}
		}
	}

	// Resolve output directory (default: ../generated/<slug> relative to scenario root)
	outputDir, err := resolveGenerationPath(slug)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve generation path: %w", err)
	}

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

	if dryRun {
		return map[string]interface{}{
			"scenario_id": slug,
			"name":        name,
			"template":    template.ID,
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
		}, nil
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Copy template assets into the generated scenario directory
	if err := scaffoldScenario(outputDir); err != nil {
		return nil, fmt.Errorf("failed to scaffold scenario: %w", err)
	}

	// Write template metadata for the generated scenario
	templateOut := filepath.Join(outputDir, "api", "templates", templateID+".json")
	if err := os.MkdirAll(filepath.Dir(templateOut), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create template metadata directory: %w", err)
	}
	templateData, _ := json.MarshalIndent(template, "", "  ")
	if err := os.WriteFile(templateOut, templateData, 0o644); err != nil {
		return nil, fmt.Errorf("failed to write template metadata: %w", err)
	}

	if err := writeTemplateProvenance(outputDir, template); err != nil {
		return nil, fmt.Errorf("failed to stamp template provenance: %w", err)
	}

	if err := rewriteServiceConfig(outputDir, name, slug); err != nil {
		return nil, fmt.Errorf("failed to rewrite service config: %w", err)
	}

	validation := validateGeneratedScenario(outputDir)

	result := map[string]interface{}{
		"scenario_id": slug,
		"name":        name,
		"template":    template.ID,
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

	return result, nil
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

	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)
	// Assume binary resides in /scenarios/landing-manager/api
	scenarioRoot := filepath.Dir(execDir)
	defaultRoot := filepath.Join(scenarioRoot, "generated")
	return filepath.Abs(defaultRoot)
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

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)
	scenarioRoot := filepath.Dir(execDir)

	// Prefer dedicated template payload from repo root (scripts/scenarios/templates/saas-landing-page/payload)
	repoRoot := filepath.Dir(filepath.Dir(scenarioRoot))
	templatePayload := filepath.Join(repoRoot, "scripts", "scenarios", "templates", "saas-landing-page", "payload")
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

// rewriteServiceConfig adjusts service metadata for the generated scenario.
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

	service, ok := cfg["service"].(map[string]interface{})
	if ok {
		service["name"] = slug
		service["displayName"] = name
		service["description"] = fmt.Sprintf("Generated landing page scenario from template '%s'", name)
	}

	if repo, ok := service["repository"].(map[string]interface{}); ok {
		repo["directory"] = fmt.Sprintf("/scenarios/%s", slug)
	}

	// Remove factory-only steps (e.g., install-cli) from generated lifecycle
	if lifecycle, ok := cfg["lifecycle"].(map[string]interface{}); ok {
		if setup, ok := lifecycle["setup"].(map[string]interface{}); ok {
			if steps, ok := setup["steps"].([]interface{}); ok {
				var filtered []interface{}
				for _, s := range steps {
					stepMap, ok := s.(map[string]interface{})
					if ok {
						if name, _ := stepMap["name"].(string); name == "install-cli" {
							continue
						}
					}
					filtered = append(filtered, s)
				}
				setup["steps"] = filtered
			}
		}
		if develop, ok := lifecycle["develop"].(map[string]interface{}); ok {
			if steps, ok := develop["steps"].([]interface{}); ok {
				for i, s := range steps {
					if stepMap, ok := s.(map[string]interface{}); ok {
						if name, _ := stepMap["name"].(string); name == "start-api" {
							stepMap["run"] = "cd api && exec ./landing-manager-api"
							steps[i] = stepMap
						}
					}
				}
				develop["steps"] = steps
			}
		}
	}

	updated, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated service.json: %w", err)
	}

	return os.WriteFile(path, updated, 0o644)
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

	var scenarios []GeneratedScenario
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
