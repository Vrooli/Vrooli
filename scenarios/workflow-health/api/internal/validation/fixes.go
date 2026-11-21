package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"workflow-health/internal/workflows"

	"github.com/vrooli/maturity-go/autofix"
)

type FixRegistry struct {
	registry *autofix.Registry
}

func NewFixRegistry() *FixRegistry {
	return &FixRegistry{registry: autofix.NewRegistry(
		autofix.Fixer{RuleID: CodeRegistryMissing, Preview: previewRegistryFix, CanFix: canFixRegistry},
		autofix.Fixer{RuleID: CodeRegistryStale, Preview: previewRegistryFix, CanFix: canFixRegistry},
		autofix.Fixer{RuleID: CodeMetadataIncomplete, Preview: previewMetadataFix, CanFix: canFixWorkflowFile},
		autofix.Fixer{RuleID: CodeExecutionModeInvalid, Preview: previewExecutionModeFix, CanFix: canFixWorkflowFile},
		autofix.Fixer{RuleID: CodeObserverContentUnsafe, Preview: previewObserverContentFix, CanFix: canFixWorkflowFile},
		autofix.Fixer{RuleID: CodeMutatingSafety, Preview: previewMutatingSafetyFix, CanFix: canFixWorkflowFile},
		autofix.Fixer{RuleID: CodeSeedMissing, Preview: previewSeedDependencyFix, CanFix: canFixWorkflowFile},
		autofix.Fixer{RuleID: CodeResetLegacy, Preview: previewResetFix, CanFix: canFixWorkflowFile},
	)}
}

func (r *FixRegistry) Preview(root string, ruleIDs []string) ([]autofix.Candidate, error) {
	return r.registry.Preview(root, ruleIDs)
}

func (r *FixRegistry) Apply(root string, ruleIDs []string) ([]autofix.Candidate, error) {
	if len(ruleIDs) == 0 {
		ruleIDs = []string{CodeRegistryMissing, CodeRegistryStale, CodeMetadataIncomplete, CodeExecutionModeInvalid, CodeObserverContentUnsafe, CodeMutatingSafety, CodeSeedMissing, CodeResetLegacy}
	}
	var applied []autofix.Candidate
	for _, ruleID := range ruleIDs {
		candidates, err := r.registry.Apply(root, []string{ruleID})
		if err != nil {
			return applied, err
		}
		applied = append(applied, candidates...)
	}
	return applied, nil
}

func (r *FixRegistry) CanFix(root, ruleID, findingPath string) bool {
	return r.registry.CanFix(root, ruleID, findingPath)
}

func previewRegistryFix(root string) ([]autofix.Candidate, error) {
	catalog, err := workflowsScan(root)
	if err != nil {
		return nil, err
	}
	registryPath := filepath.Join(root, "bas", "registry.json")
	beforeBytes, _ := os.ReadFile(registryPath)
	before := string(beforeBytes)
	after, err := buildRegistryJSON(catalog)
	if err != nil {
		return nil, err
	}
	if before == after {
		return nil, nil
	}
	ruleID := CodeRegistryStale
	if before == "" {
		ruleID = CodeRegistryMissing
	}
	return []autofix.Candidate{{
		RuleID:      ruleID,
		FilePath:    registryPath,
		Description: "Rebuild bas/registry.json from cataloged validation cases.",
		Before:      before,
		After:       after,
	}}, nil
}

func workflowsScan(root string) (catalogView, error) {
	catalog, err := workflows.ScanScenario(root)
	if err != nil {
		return catalogView{}, err
	}
	view := catalogView{Scenario: catalog.Scenario}
	if isRFC3339(catalog.Registry.GeneratedAt) {
		view.GeneratedAt = catalog.Registry.GeneratedAt
	}
	for _, c := range catalog.Cases {
		if c.ParseError != "" {
			continue
		}
		view.Cases = append(view.Cases, registryCase{
			File:         c.Path,
			Description:  c.Description,
			Order:        c.Order,
			Requirements: requirementIDs(c.Requirements),
			Fixtures:     fixturePaths(c.Dependencies),
			Reset:        c.Reset,
			Mutating:     c.Safety.Mutating,
		})
	}
	sort.Slice(view.Cases, func(i, j int) bool { return view.Cases[i].File < view.Cases[j].File })
	return view, nil
}

type catalogView struct {
	Scenario    string
	GeneratedAt string
	Cases       []registryCase
}

type registryCase struct {
	File         string   `json:"file"`
	Description  string   `json:"description,omitempty"`
	Order        string   `json:"order,omitempty"`
	Requirements []string `json:"requirements,omitempty"`
	Fixtures     []string `json:"fixtures"`
	Reset        string   `json:"reset,omitempty"`
	Mutating     bool     `json:"-"`
}

func buildRegistryJSON(view catalogView) (string, error) {
	generatedAt := view.GeneratedAt
	if generatedAt == "" {
		generatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	executionMode := "observer"
	for _, c := range view.Cases {
		if c.Mutating {
			executionMode = "mutating"
			break
		}
	}
	payload := map[string]any{
		"scenario":     view.Scenario,
		"generated_at": generatedAt,
		"metadata": map[string]any{
			"execution_mode": executionMode,
		},
		"playbooks": view.Cases,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

func isRFC3339(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	_, err := time.Parse(time.RFC3339, value)
	return err == nil
}

func requirementIDs(links []workflows.RequirementLink) []string {
	ids := make([]string, 0, len(links))
	for _, link := range links {
		ids = append(ids, link.ID)
	}
	sort.Strings(ids)
	return ids
}

func fixturePaths(edges []workflows.DependencyEdge) []string {
	fixtures := make([]string, 0, len(edges))
	for _, edge := range edges {
		if edge.Kind != "fixture" || strings.TrimSpace(edge.ToPath) == "" {
			continue
		}
		fixtures = append(fixtures, strings.TrimSpace(edge.ToPath))
	}
	sort.Strings(fixtures)
	return fixtures
}

func previewMetadataFix(root string) ([]autofix.Candidate, error) {
	return previewWorkflowJSONEdits(root, CodeMetadataIncomplete, "Fill missing workflow metadata stubs.", func(path string, doc map[string]any) bool {
		changed := false
		if getString(doc, "metadata", "name") == "" {
			setNestedString(doc, titleFromPath(path), "metadata", "name")
			changed = true
		}
		if getString(doc, "metadata", "description") == "" {
			setNestedString(doc, fmt.Sprintf("Workflow asset %s.", strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))), "metadata", "description")
			changed = true
		}
		return changed
	})
}

func previewExecutionModeFix(root string) ([]autofix.Candidate, error) {
	return previewWorkflowJSONEdits(root, CodeExecutionModeInvalid, "Normalize execution_mode to the canonical authored form.", func(_ string, doc map[string]any) bool {
		mode := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(firstNonEmpty(getString(doc, "metadata", "execution_mode"), getString(doc, "execution_mode")))), "execution_mode_")
		if mode != "observer" && mode != "mutating" && mode != "destructive" {
			mode = "observer"
		}
		metadataMode := getString(doc, "metadata", "execution_mode")
		rootMode := getString(doc, "execution_mode")
		if metadataMode == mode && rootMode == "" {
			return false
		}
		setNestedString(doc, mode, "metadata", "execution_mode")
		delete(doc, "execution_mode")
		return true
	})
}

func previewObserverContentFix(root string) ([]autofix.Candidate, error) {
	return previewWorkflowJSONEdits(root, CodeObserverContentUnsafe, "Relabel observer workflow with mutating actions to mutating.", func(_ string, doc map[string]any) bool {
		if strings.ToLower(strings.TrimSpace(getString(doc, "metadata", "execution_mode"))) != "observer" {
			return false
		}
		setNestedString(doc, "mutating", "metadata", "execution_mode")
		return true
	})
}

func previewMutatingSafetyFix(root string) ([]autofix.Candidate, error) {
	return previewWorkflowJSONEdits(root, CodeMutatingSafety, "Declare confirmation and routed isolation for mutating workflows.", func(_ string, doc map[string]any) bool {
		mode := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(firstNonEmpty(getString(doc, "metadata", "execution_mode"), getString(doc, "execution_mode")))), "execution_mode_")
		if mode != "mutating" && mode != "destructive" {
			return false
		}
		changed := false
		if getString(doc, "metadata", "labels", "requires_confirmation") != "true" {
			setNestedString(doc, "true", "metadata", "labels", "requires_confirmation")
			changed = true
		}
		if getString(doc, "metadata", "labels", "routed_isolation") != "true" {
			setNestedString(doc, "true", "metadata", "labels", "routed_isolation")
			changed = true
		}
		return changed
	})
}

func previewSeedDependencyFix(root string) ([]autofix.Candidate, error) {
	return previewWorkflowJSONEdits(root, CodeSeedMissing, "Declare the scenario seed for full-reset mutating workflows.", func(_ string, doc map[string]any) bool {
		mode := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(firstNonEmpty(getString(doc, "metadata", "execution_mode"), getString(doc, "execution_mode")))), "execution_mode_")
		reset := strings.ToLower(strings.TrimSpace(firstNonEmpty(getString(doc, "metadata", "labels", "reset"), getString(doc, "metadata", "reset"))))
		if (mode != "mutating" && mode != "destructive") || reset != "full" || getString(doc, "metadata", "labels", "seed") != "" {
			return false
		}
		setNestedString(doc, "bas/seeds/seed.go", "metadata", "labels", "seed")
		return true
	})
}

func previewResetFix(root string) ([]autofix.Candidate, error) {
	return previewWorkflowJSONEdits(root, CodeResetLegacy, "Normalize legacy reset=database to reset=full.", func(_ string, doc map[string]any) bool {
		changed := false
		if strings.EqualFold(getString(doc, "metadata", "reset"), "database") {
			setNestedString(doc, "full", "metadata", "reset")
			changed = true
		}
		if strings.EqualFold(getString(doc, "metadata", "labels", "reset"), "database") {
			setNestedString(doc, "full", "metadata", "labels", "reset")
			changed = true
		}
		return changed
	})
}

func previewWorkflowJSONEdits(root, ruleID, description string, edit func(path string, doc map[string]any) bool) ([]autofix.Candidate, error) {
	files, err := workflowJSONFiles(root)
	if err != nil {
		return nil, err
	}
	var out []autofix.Candidate
	for _, path := range files {
		beforeBytes, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var doc map[string]any
		if err := json.Unmarshal(beforeBytes, &doc); err != nil {
			continue
		}
		rel, _ := filepath.Rel(root, path)
		if !edit(filepath.ToSlash(rel), doc) {
			continue
		}
		afterBytes, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return nil, err
		}
		after := string(afterBytes) + "\n"
		before := string(beforeBytes)
		if before == after {
			continue
		}
		out = append(out, autofix.Candidate{
			RuleID:      ruleID,
			FilePath:    path,
			Description: description,
			Before:      before,
			After:       after,
		})
	}
	return out, nil
}

func workflowJSONFiles(root string) ([]string, error) {
	var files []string
	for _, dir := range []string{"cases", "flows", "actions"} {
		base := filepath.Join(root, "bas", dir)
		err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if strings.HasPrefix(entry.Name(), "__") {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(entry.Name()) == ".json" {
				files = append(files, path)
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

func titleFromPath(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	parts := strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' })
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func canFixRegistry(root, _ string) bool {
	_, err := workflows.ScanScenario(root)
	return err == nil
}

func canFixWorkflowFile(root, findingPath string) bool {
	if strings.TrimSpace(findingPath) == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(findingPath)))
	return err == nil && !info.IsDir()
}
