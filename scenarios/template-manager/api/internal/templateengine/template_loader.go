package templateengine

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

func loadTemplates(root string) ([]templatecontracts.TemplateInfo, error) {
	baseDir := config.TemplateBaseDir(root)
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}
	templates := make([]templatecontracts.TemplateInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		info, err := loadTemplate(root, name)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				templates = append(templates, templatecontracts.TemplateInfo{Name: name, Path: filepath.Join(baseDir, name), Missing: true})
				continue
			}
			return nil, err
		}
		templates = append(templates, info)
	}
	sort.Slice(templates, func(i, j int) bool { return templates[i].Name < templates[j].Name })
	return templates, nil
}

func loadTemplate(root, name string) (templatecontracts.TemplateInfo, error) {
	templateDir := filepath.Join(config.TemplateBaseDir(root), name)
	info, err := os.Stat(templateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return templatecontracts.TemplateInfo{}, fmt.Errorf("%w: %s", ErrTemplateNotFound, name)
		}
		return templatecontracts.TemplateInfo{}, err
	}
	if !info.IsDir() {
		return templatecontracts.TemplateInfo{}, fmt.Errorf("template path is not a directory: %s", templateDir)
	}
	data, err := os.ReadFile(filepath.Join(templateDir, "template.json"))
	if err != nil {
		return templatecontracts.TemplateInfo{}, err
	}
	var manifest templatecontracts.TemplateManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return templatecontracts.TemplateInfo{}, err
	}
	if manifest.Name == "" {
		manifest.Name = name
	}
	if manifest.RequiredVars == nil {
		manifest.RequiredVars = map[string]templatecontracts.TemplateVar{}
	}
	if manifest.OptionalVars == nil {
		manifest.OptionalVars = map[string]templatecontracts.TemplateVar{}
	}
	if manifest.Docs == nil {
		manifest.Docs = map[string]string{}
	}
	return templatecontracts.TemplateInfo{Name: name, Path: templateDir, Manifest: manifest}, nil
}

func buildTemplateValues(root, destination, templateName string, manifest templatecontracts.TemplateManifest, baseValues map[string]string) (map[string]string, error) {
	currentDate := time.Now().UTC().Format("2006-01-02")
	randomToken, err := randomTemplateToken()
	if err != nil {
		return nil, err
	}
	values := copyStringMap(baseValues)
	values["CURRENT_DATE"] = currentDate
	values["RANDOM_TOKEN"] = randomToken
	if err := populateTemplatePathValues(root, destination, values); err != nil {
		return nil, fmt.Errorf("resolve template path placeholders for %s: %w", templateName, err)
	}
	optionalKeys := make([]string, 0, len(manifest.OptionalVars))
	for key := range manifest.OptionalVars {
		optionalKeys = append(optionalKeys, key)
	}
	sort.Strings(optionalKeys)
	for _, key := range optionalKeys {
		if strings.TrimSpace(values[key]) == "" {
			values[key] = renderTemplateString(manifest.OptionalVars[key].Default, values)
		}
	}
	// Derive snake_case identifiers from kebab-case scenario IDs so proto
	// package directives (which forbid hyphens), Go package aliases, and
	// Python module names get a valid identifier without each template
	// having to re-implement the conversion.
	if id, ok := values["SCENARIO_ID"]; ok && strings.TrimSpace(id) != "" {
		values["SCENARIO_ID_SNAKE"] = strings.ReplaceAll(id, "-", "_")
	}
	return values, nil
}

func populateTemplatePathValues(root, destination string, values map[string]string) error {
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return err
	}
	repoRoot := filepath.Clean(root)
	packagesDir, err := contract.TopLevelDir(root, "packages")
	if err != nil {
		return err
	}
	for key, dir := range map[string]string{
		"API":     filepath.Join(destination, "api"),
		"CLI":     filepath.Join(destination, "cli"),
		"RUNTIME": filepath.Join(destination, "runtime"),
	} {
		repoRel, err := filepath.Rel(dir, repoRoot)
		if err != nil {
			return err
		}
		packagesRel, err := filepath.Rel(dir, packagesDir)
		if err != nil {
			return err
		}
		values["REPO_ROOT_REL_FROM_"+key] = filepath.ToSlash(repoRel)
		values["PACKAGES_REL_FROM_"+key] = filepath.ToSlash(packagesRel)
	}
	return nil
}

func templateValidationSeedValues(info templatecontracts.TemplateInfo) map[string]string {
	values := map[string]string{}
	requiredKeys := make([]string, 0, len(info.Manifest.RequiredVars))
	for key := range info.Manifest.RequiredVars {
		requiredKeys = append(requiredKeys, key)
	}
	sort.Strings(requiredKeys)
	for _, key := range requiredKeys {
		switch key {
		case "SCENARIO_ID":
			values[key] = "template-validation-" + info.Name
		case "SCENARIO_DISPLAY_NAME":
			values[key] = coalesce(info.Manifest.DisplayName, info.Name+" Validation")
		case "SCENARIO_DESCRIPTION":
			values[key] = coalesce(info.Manifest.Description, "Validation scenario generated from "+info.Name)
		default:
			if fallback := strings.TrimSpace(info.Manifest.RequiredVars[key].Default); fallback != "" {
				values[key] = fallback
			} else {
				values[key] = strings.ToLower(strings.ReplaceAll(key, "_", "-"))
			}
		}
	}
	return values
}

func templateValidationSeedValuesForScenarioID(info templatecontracts.TemplateInfo, scenarioID string) map[string]string {
	values := templateValidationSeedValues(info)
	if strings.TrimSpace(scenarioID) != "" {
		values["SCENARIO_ID"] = scenarioID
	}
	return values
}
