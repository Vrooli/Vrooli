// Package autofix hosts experience-manager's deterministic remediation
// registry on the shared maturity-go substrate.
package autofix

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"experience-manager/internal/basrefs"
	"experience-manager/internal/spec"

	"github.com/vrooli/maturity-go/autofix"
)

const (
	// RuleCaseScaffold derives BAS observer smoke stubs from active page specs.
	RuleCaseScaffold = "experience-fix.case_scaffold"
	// RuleIndexNormalization keeps experience/index.json aligned with page files.
	RuleIndexNormalization = "experience-fix.index_normalization"
	// RuleBindingDriftRepair fills missing element bindings with deterministic
	// data-testid suggestions based on the element id.
	RuleBindingDriftRepair = "experience-fix.binding_drift_repair"
	// RuleFindingDocStubs creates missing finding docs for the frozen vocabulary.
	RuleFindingDocStubs = "experience-fix.finding_doc_stub"
)

// NewRegistry returns the scenario's fixer registry.
func NewRegistry() *autofix.Registry {
	return autofix.NewRegistry(
		caseScaffoldFixer(),
		indexNormalizationFixer(),
		bindingDriftFixer(),
		findingDocStubsFixer(),
	)
}

// ApplySequential applies rules one at a time and re-previews between writes so
// independent rules that touch experience/index.json or page docs cannot clobber
// each other's snapshots.
func ApplySequential(reg *autofix.Registry, root string, ruleIDs []string) ([]autofix.Candidate, error) {
	rules := ruleIDs
	if len(rules) == 0 {
		rules = RuleIDs()
	}
	var out []autofix.Candidate
	for _, rule := range rules {
		applied, err := reg.Apply(root, []string{rule})
		if err != nil {
			return out, err
		}
		out = append(out, applied...)
	}
	return out, nil
}

// RuleIDs lists deterministic fixers in apply order.
func RuleIDs() []string {
	return []string{
		RuleIndexNormalization,
		RuleBindingDriftRepair,
		RuleFindingDocStubs,
		RuleCaseScaffold,
	}
}

func caseScaffoldFixer() autofix.Fixer {
	preview := func(root string) ([]autofix.Candidate, error) {
		report, err := spec.ParseScenario(root)
		if err != nil {
			return nil, nil
		}
		existing := caseRefs(root)
		statuses := pageStatuses(report.Spec.Index.Pages)
		var candidates []autofix.Candidate
		for _, pageID := range sortedPageIDs(report.Spec.Pages) {
			if statuses[pageID] != "active" || existing[pageID] {
				continue
			}
			page := report.Spec.Pages[pageID]
			rel := filepath.ToSlash(filepath.Join("bas", "cases", "experience-spec", pageID+".json"))
			path := filepath.Join(root, filepath.FromSlash(rel))
			before := readIfExists(path)
			afterBytes, err := json.MarshalIndent(caseStub(report.Scenario, page), "", "  ")
			if err != nil {
				return nil, err
			}
			after := string(afterBytes) + "\n"
			if before == after {
				continue
			}
			candidates = append(candidates, autofix.Candidate{
				RuleID:      RuleCaseScaffold,
				FilePath:    path,
				Description: fmt.Sprintf("Scaffold BAS observer case for active experience page %q", pageID),
				Before:      before,
				After:       after,
			})
		}
		regCandidate, err := registryCandidate(root, report.Scenario, candidates)
		if err != nil {
			return nil, err
		}
		if regCandidate != nil {
			candidates = append(candidates, *regCandidate)
		}
		sortCandidates(candidates)
		return candidates, nil
	}
	return autofix.Fixer{RuleID: RuleCaseScaffold, Preview: preview, CanFix: func(root, _ string) bool { return previewable(preview, root) }}
}

func indexNormalizationFixer() autofix.Fixer {
	preview := func(root string) ([]autofix.Candidate, error) {
		report, err := spec.ParseScenario(root)
		if err != nil && report.Spec == nil {
			return nil, nil
		}
		indexPath := filepath.Join(root, "experience", "index.json")
		before, err := os.ReadFile(indexPath)
		if err != nil {
			return nil, nil
		}
		var index spec.IndexDocument
		if err := json.Unmarshal(before, &index); err != nil {
			return nil, nil
		}
		known := map[string]bool{}
		for _, ref := range index.Pages {
			known[ref.ID] = true
		}
		for _, pageID := range sortedPageIDs(report.Spec.Pages) {
			if known[pageID] {
				continue
			}
			page := report.Spec.Pages[pageID]
			index.Pages = append(index.Pages, spec.DocumentRef{
				ID:     pageID,
				Path:   "pages/" + pageID + ".json",
				Title:  page.Page.Title,
				Status: "draft",
			})
		}
		sort.Slice(index.Pages, func(i, j int) bool { return index.Pages[i].ID < index.Pages[j].ID })
		sort.Slice(index.Journeys, func(i, j int) bool { return index.Journeys[i].ID < index.Journeys[j].ID })
		afterBytes, err := json.MarshalIndent(index, "", "  ")
		if err != nil {
			return nil, err
		}
		after := string(afterBytes) + "\n"
		if after == string(before) {
			return nil, nil
		}
		return []autofix.Candidate{{
			RuleID:      RuleIndexNormalization,
			FilePath:    indexPath,
			Description: "Normalize experience/index.json ordering and add unlisted page documents",
			Before:      string(before),
			After:       after,
		}}, nil
	}
	return autofix.Fixer{RuleID: RuleIndexNormalization, Preview: preview, CanFix: func(root, _ string) bool { return previewable(preview, root) }}
}

func bindingDriftFixer() autofix.Fixer {
	preview := func(root string) ([]autofix.Candidate, error) {
		report, err := spec.ParseScenario(root)
		if err != nil && report.Spec == nil {
			return nil, nil
		}
		refs := pageRefByID(report.Spec.Index.Pages)
		var candidates []autofix.Candidate
		for _, pageID := range sortedPageIDs(report.Spec.Pages) {
			page := report.Spec.Pages[pageID]
			changed := false
			if page.Bindings.Elements == nil {
				page.Bindings.Elements = map[string]spec.Binding{}
			}
			for _, element := range page.Elements {
				binding := page.Bindings.Elements[element.ID]
				if binding.TestID == "" && binding.Selector == "" {
					page.Bindings.Elements[element.ID] = spec.Binding{
						TestID: element.ID,
						Note:   "Generated by experience-manager deterministic binding drift repair; align UI data-testid before treating as verified.",
					}
					changed = true
				}
			}
			if !changed {
				continue
			}
			relPath := refs[pageID].Path
			if relPath == "" {
				relPath = "pages/" + pageID + ".json"
			}
			path := filepath.Join(root, "experience", filepath.FromSlash(relPath))
			before, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			afterBytes, err := json.MarshalIndent(page, "", "  ")
			if err != nil {
				return nil, err
			}
			candidates = append(candidates, autofix.Candidate{
				RuleID:      RuleBindingDriftRepair,
				FilePath:    path,
				Description: fmt.Sprintf("Add deterministic placeholder bindings for unbound elements on page %q", pageID),
				Before:      string(before),
				After:       string(afterBytes) + "\n",
			})
		}
		sortCandidates(candidates)
		return candidates, nil
	}
	return autofix.Fixer{RuleID: RuleBindingDriftRepair, Preview: preview, CanFix: func(root, _ string) bool { return previewable(preview, root) }}
}

func findingDocStubsFixer() autofix.Fixer {
	preview := func(root string) ([]autofix.Candidate, error) {
		var candidates []autofix.Candidate
		for _, code := range spec.AllFindingCodes {
			path := filepath.Join(root, "docs", "findings", code+".md")
			if _, err := os.Stat(path); err == nil {
				continue
			}
			after := fmt.Sprintf("---\ndocType: finding-%s\ncode: %s\n---\n\n# %s\n\nTODO: Document detection semantics, remediation guidance, and examples.\n", strings.ReplaceAll(code, ".", "-"), code, code)
			candidates = append(candidates, autofix.Candidate{
				RuleID:      RuleFindingDocStubs,
				FilePath:    path,
				Description: "Create missing finding documentation stub for " + code,
				After:       after,
			})
		}
		return candidates, nil
	}
	return autofix.Fixer{RuleID: RuleFindingDocStubs, Preview: preview, CanFix: func(root, _ string) bool { return previewable(preview, root) }}
}

func caseStub(scenario string, page spec.PageDocument) map[string]any {
	selector := "@selector/" + page.Page.ID
	if first := firstMachineBinding(page); first != "" {
		selector = "[data-testid=\"" + first + "\"]"
	}
	nodes := []map[string]any{{
		"id": "navigate-" + page.Page.ID,
		"action": map[string]any{
			"type": "ACTION_TYPE_NAVIGATE",
			"navigate": map[string]any{
				"destination_type": "NAVIGATE_DESTINATION_TYPE_SCENARIO",
				"scenario":         scenario,
				"scenario_path":    firstRoute(page.Page.Routes),
				"wait_until":       "NAVIGATE_WAIT_EVENT_NETWORKIDLE",
				"timeout_ms":       30000,
			},
			"metadata": map[string]any{"label": "Open " + page.Page.Title},
		},
	}, {
		"id": "wait-" + page.Page.ID + "-primary-binding",
		"action": map[string]any{
			"type": "ACTION_TYPE_WAIT",
			"wait": map[string]any{
				"selector":   selector,
				"state":      "WAIT_STATE_VISIBLE",
				"timeout_ms": 15000,
			},
			"metadata": map[string]any{"label": "Primary binding visible"},
		},
	}}
	return map[string]any{
		"metadata": map[string]any{
			"name":        "experience-" + page.Page.ID + "-spec-smoke",
			"description": "Generated observer smoke for active experience page " + page.Page.ID + ".",
			"requirement": firstOr(page.Page.PRDRefs, "OT-P0-005"),
			"version":     "1",
			"labels": map[string]string{
				"reset":         "none",
				"surface":       "experience-" + page.Page.ID,
				"spec_entry_id": page.Page.ID,
			},
			"execution_mode": "observer",
		},
		"settings": map[string]int{"viewport_width": 1280, "viewport_height": 800},
		"nodes":    nodes,
	}
}

func registryCandidate(root, scenario string, caseCandidates []autofix.Candidate) (*autofix.Candidate, error) {
	if len(caseCandidates) == 0 {
		return nil, nil
	}
	path := filepath.Join(root, "bas", "registry.json")
	before := readIfExists(path)
	reg := map[string]any{
		"scenario": scenario,
		"metadata": map[string]any{"execution_mode": "observer"},
	}
	if before != "" {
		_ = json.Unmarshal([]byte(before), &reg)
	}
	playbooks, _ := reg["playbooks"].([]any)
	seen := map[string]bool{}
	for _, item := range playbooks {
		if obj, ok := item.(map[string]any); ok {
			if file, ok := obj["file"].(string); ok {
				seen[file] = true
			}
		}
	}
	for _, c := range caseCandidates {
		rel, err := filepath.Rel(root, c.FilePath)
		if err != nil {
			rel = c.FilePath
		}
		file := filepath.ToSlash(rel)
		if seen[file] {
			continue
		}
		playbooks = append(playbooks, map[string]any{
			"file":         file,
			"description":  c.Description,
			"requirements": []string{},
			"fixtures":     []string{},
			"reset":        "none",
		})
	}
	sort.Slice(playbooks, func(i, j int) bool {
		return playbookFile(playbooks[i]) < playbookFile(playbooks[j])
	})
	reg["playbooks"] = playbooks
	afterBytes, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return nil, err
	}
	after := string(afterBytes) + "\n"
	if before == after {
		return nil, nil
	}
	return &autofix.Candidate{
		RuleID:      RuleCaseScaffold,
		FilePath:    path,
		Description: "Register generated experience BAS case stubs",
		Before:      before,
		After:       after,
	}, nil
}

func caseRefs(root string) map[string]bool {
	out := map[string]bool{}
	for _, path := range basrefs.CaseFiles(root) {
		var doc struct {
			Metadata struct {
				Labels map[string]string `json:"labels"`
			} `json:"metadata"`
		}
		data, err := os.ReadFile(path)
		if err == nil && json.Unmarshal(data, &doc) == nil {
			if id := strings.TrimSpace(doc.Metadata.Labels["spec_entry_id"]); id != "" {
				out[id] = true
			}
		}
	}
	return out
}

func pageStatuses(refs []spec.DocumentRef) map[string]string {
	out := map[string]string{}
	for _, ref := range refs {
		out[ref.ID] = ref.Status
	}
	return out
}

func pageRefByID(refs []spec.DocumentRef) map[string]spec.DocumentRef {
	out := map[string]spec.DocumentRef{}
	for _, ref := range refs {
		out[ref.ID] = ref
	}
	return out
}

func sortedPageIDs(pages map[string]spec.PageDocument) []string {
	out := make([]string, 0, len(pages))
	for id := range pages {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func firstMachineBinding(page spec.PageDocument) string {
	for _, claim := range page.Claims {
		if claim.Tier != "machine" {
			continue
		}
		for _, elementID := range claim.Elements {
			if binding := page.Bindings.Elements[elementID]; binding.TestID != "" {
				return binding.TestID
			}
		}
	}
	for _, elementID := range sortedBindingIDs(page.Bindings.Elements) {
		if binding := page.Bindings.Elements[elementID]; binding.TestID != "" {
			return binding.TestID
		}
	}
	return ""
}

func sortedBindingIDs(bindings map[string]spec.Binding) []string {
	out := make([]string, 0, len(bindings))
	for id := range bindings {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func firstRoute(routes []string) string {
	if len(routes) == 0 {
		return "/"
	}
	return routes[0]
}

func firstOr(values []string, fallback string) string {
	if len(values) == 0 || values[0] == "" {
		return fallback
	}
	return values[0]
}

func readIfExists(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func previewable(preview func(string) ([]autofix.Candidate, error), root string) bool {
	candidates, err := preview(root)
	return err == nil && len(candidates) > 0
}

func sortCandidates(candidates []autofix.Candidate) {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].FilePath != candidates[j].FilePath {
			return candidates[i].FilePath < candidates[j].FilePath
		}
		return candidates[i].RuleID < candidates[j].RuleID
	})
}

func playbookFile(value any) string {
	obj, _ := value.(map[string]any)
	file, _ := obj["file"].(string)
	return file
}
