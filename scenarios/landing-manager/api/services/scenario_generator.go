// Package services contains the business logic for the landing-manager API.
package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"landing-manager/util"
)

// ============================================================================
// Generation Mode Decisions
// ============================================================================
// Generation can operate in two modes: dry-run (planning) or actual (materialization).
// The decision is made early in the GenerateScenario flow based on options.

// GenerationMode represents how scenario generation should behave
type GenerationMode string

const (
	// ModeDryRun plans generation but doesn't create files
	ModeDryRun GenerationMode = "dry_run"
	// ModeCreate actually creates the scenario files
	ModeCreate GenerationMode = "create"
)

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

// ScenarioGenerator handles scenario generation from templates
type ScenarioGenerator struct {
	registry *TemplateRegistry
}

// NewScenarioGenerator creates a new scenario generator
func NewScenarioGenerator(registry *TemplateRegistry) *ScenarioGenerator {
	return &ScenarioGenerator{registry: registry}
}

// GenerateScenario creates a new landing page scenario from a template.
// Decision Flow:
//  1. Validate template exists
//  2. Validate required inputs (name, slug)
//  3. Determine generation mode (dry-run vs create)
//  4. If dry-run: return planned paths without creating files
//  5. If create: materialize scenario and validate output
func (sg *ScenarioGenerator) GenerateScenario(templateID, name, slug string, options map[string]interface{}) (map[string]interface{}, error) {
	template, err := sg.registry.GetTemplate(templateID)
	if err != nil {
		return nil, err
	}

	if name == "" || slug == "" {
		return nil, fmt.Errorf("name and slug are required")
	}

	outputDir, err := util.ResolveGenerationPath(slug)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve generation path: %w", err)
	}

	// Decision: Determine generation mode based on dry_run option
	mode := determineGenerationMode(options)
	if mode == ModeDryRun {
		return buildDryRunResponse(slug, name, template.ID, outputDir), nil
	}

	// Decision: Mode is ModeCreate - actually materialize the scenario
	if err := sg.materializeScenario(outputDir, templateID, template, name, slug); err != nil {
		return nil, err
	}

	validation := validateGeneratedScenario(outputDir)
	return buildCreatedResponse(slug, name, template.ID, outputDir, validation), nil
}

// determineGenerationMode decides whether to do a dry-run or actual generation.
// Decision: Returns ModeDryRun if options["dry_run"] is true (bool) or "true" (string).
func determineGenerationMode(options map[string]interface{}) GenerationMode {
	if isDryRun(options) {
		return ModeDryRun
	}
	return ModeCreate
}

// ListGeneratedScenarios reports any generated scenarios under the generation root
func (sg *ScenarioGenerator) ListGeneratedScenarios() ([]GeneratedScenario, error) {
	root, err := util.GenerationRoot()
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

// isDryRun checks if the dry_run option is set in generation options.
// Decision: Accepts both boolean true and string "true" (case-insensitive).
// This flexibility supports both programmatic (bool) and CLI (string) callers.
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

func (sg *ScenarioGenerator) materializeScenario(outputDir, templateID string, template *Template, name, slug string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := scaffoldScenario(outputDir); err != nil {
		return fmt.Errorf("failed to scaffold scenario: %w", err)
	}

	description := fmt.Sprintf("Generated landing page scenario from template '%s'", name)

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

// ============================================================================
// Template Source Decisions
// ============================================================================
// scaffoldScenario determines where to copy the template payload from.
// Decision Priority:
//  1. TEMPLATE_PAYLOAD_DIR env var (explicit override for testing/CI)
//  2. Standard template location: scripts/scenarios/templates/landing-page-react-vite
//  3. Fallback: Copy from landing-manager's own directory (legacy)
func scaffoldScenario(outputDir string) error {
	// Decision 1: Check for explicit override first (testing/CI use case)
	if override := strings.TrimSpace(os.Getenv("TEMPLATE_PAYLOAD_DIR")); override != "" {
		if err := copyTemplatePayload(override, outputDir); err != nil {
			return err
		}
		return nil
	}

	scenarioRoot, err := util.ResolveScenarioRoot()
	if err != nil {
		return err
	}

	// Decision 2: Use standard template location (preferred)
	repoRoot := filepath.Dir(filepath.Dir(scenarioRoot))
	templatePayload := filepath.Join(repoRoot, "scripts", "scenarios", "templates", "landing-page-react-vite")
	if _, err := os.Stat(templatePayload); err == nil {
		return copyTemplatePayload(templatePayload, outputDir)
	}

	// Decision 3: Fallback to scenario root (legacy behavior)
	return copyTemplatePayload(scenarioRoot, outputDir)
}

func copyTemplatePayload(scenarioRoot, outputDir string) error {
	sources := []struct {
		src string
		dst string
	}{
		{src: filepath.Join(scenarioRoot, "api"), dst: filepath.Join(outputDir, "api")},
		{src: filepath.Join(scenarioRoot, "ui"), dst: filepath.Join(outputDir, "ui")},
		{src: filepath.Join(scenarioRoot, "requirements"), dst: filepath.Join(outputDir, "requirements")},
		{src: filepath.Join(scenarioRoot, "initialization"), dst: filepath.Join(outputDir, "initialization")},
		{src: filepath.Join(scenarioRoot, ".vrooli"), dst: filepath.Join(outputDir, ".vrooli")},
		{src: filepath.Join(scenarioRoot, "docs"), dst: filepath.Join(outputDir, "docs")},
		{src: filepath.Join(scenarioRoot, "Makefile"), dst: filepath.Join(outputDir, "Makefile")},
		{src: filepath.Join(scenarioRoot, "PRD.md"), dst: filepath.Join(outputDir, "PRD.md")},
	}

	for _, entry := range sources {
		if err := util.CopyDir(entry.src, entry.dst); err != nil {
			return fmt.Errorf("failed to copy %s: %w", entry.src, err)
		}
	}

	factoryPage := filepath.Join(outputDir, "ui", "src", "pages", "FactoryHome.tsx")
	_ = os.Remove(factoryPage)

	// Template's App.tsx is used directly - it already has all routes configured correctly
	// (including /admin/downloads, /admin/branding, etc.)

	if err := fixWorkspaceDependencies(outputDir); err != nil {
		return fmt.Errorf("failed to fix workspace dependencies: %w", err)
	}

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

func fixWorkspaceDependencies(outputDir string) error {
	packagePath := filepath.Join(outputDir, "ui", "package.json")

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

	packagesDir, err := util.ResolvePackagesDir()
	if err != nil {
		return err
	}

	if deps, ok := pkg["dependencies"].(map[string]interface{}); ok {
		for name, value := range deps {
			if strValue, ok := value.(string); ok {
				if strings.HasPrefix(strValue, "file:../../../packages/") {
					packageName := strings.TrimPrefix(strValue, "file:../../../packages/")
					absolutePath, valid := util.ResolvePackagePath(packageName, packagesDir)
					if !valid {
						continue
					}
					deps[name] = "file:" + absolutePath
				}
			}
		}
	}

	updatedData, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated package.json: %w", err)
	}

	if err := os.WriteFile(packagePath, updatedData, 0o644); err != nil {
		return fmt.Errorf("failed to write updated package.json: %w", err)
	}

	return nil
}

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

func substituteTemplatePlaceholders(outputDir, slug, displayName, description string) error {
	textExtensions := map[string]bool{
		".go": true, ".json": true, ".js": true, ".ts": true, ".tsx": true,
		".jsx": true, ".md": true, ".html": true, ".css": true, ".yaml": true,
		".yml": true, ".toml": true, ".mod": true, ".sum": true, ".env": true,
		".sh": true, "": true,
	}

	skipDirs := map[string]bool{
		"node_modules": true, "dist": true, "coverage": true, ".git": true,
	}

	return filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(info.Name())
		if !textExtensions[ext] {
			if info.Name() != "Makefile" {
				return nil
			}
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		content := string(data)
		original := content

		content = strings.ReplaceAll(content, "{{SCENARIO_ID}}", slug)
		content = strings.ReplaceAll(content, "{{SCENARIO_DISPLAY_NAME}}", displayName)
		content = strings.ReplaceAll(content, "{{SCENARIO_DESCRIPTION}}", description)

		if content != original {
			if err := os.WriteFile(path, []byte(content), info.Mode()); err != nil {
				return fmt.Errorf("failed to write %s: %w", path, err)
			}
		}

		return nil
	})
}

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
