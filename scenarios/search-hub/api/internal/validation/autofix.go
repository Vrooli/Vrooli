package validation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/maturity-go/autofix"
)

const defaultSearchVersion = "1.0.0"

// PreviewFixes returns conservative mechanical descriptor edits for a scenario.
// It does not invent endpoints, ownership, or eval cases.
func (s *Service) PreviewFixes(scenario, path string, ruleIDs []string) (string, []autofix.Candidate, error) {
	scenario, scenarioPath, err := s.resolveTarget(scenario, path)
	if err != nil {
		return "", nil, err
	}
	candidates, _, err := previewDescriptorFixes(scenarioPath, ruleIDs)
	return scenario, candidates, err
}

// ApplyFixes writes the same edits reported by PreviewFixes. Re-running after a
// successful apply is a no-op because the preview path re-checks descriptor state.
func (s *Service) ApplyFixes(scenario, path string, ruleIDs []string) (string, []autofix.Candidate, error) {
	scenario, scenarioPath, err := s.resolveTarget(scenario, path)
	if err != nil {
		return "", nil, err
	}
	candidates, final, err := previewDescriptorFixes(scenarioPath, ruleIDs)
	if err != nil || len(candidates) == 0 {
		return scenario, candidates, err
	}
	target := descriptorPath(scenarioPath)
	if err := os.WriteFile(target, []byte(final), 0o644); err != nil {
		return "", nil, fmt.Errorf("write %s: %w", target, err)
	}
	for i := range candidates {
		candidates[i].Applied = true
	}
	return scenario, candidates, nil
}

func previewDescriptorFixes(scenarioPath string, ruleIDs []string) ([]autofix.Candidate, string, error) {
	target := descriptorPath(scenarioPath)
	raw, err := os.ReadFile(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("read %s: %w", target, err)
	}
	var doc map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&doc); err != nil {
		return nil, "", nil
	}

	before := normalizeJSON(raw)
	current := before
	candidates := []autofix.Candidate{}
	addCandidate := func(ruleID, description string, changed bool) error {
		if !changed || !wantsFixRule(ruleIDs, ruleID) {
			return nil
		}
		after, err := marshalSearchJSON(doc)
		if err != nil {
			return err
		}
		if after == current {
			return nil
		}
		candidates = append(candidates, autofix.Candidate{
			RuleID:      ruleID,
			FilePath:    target,
			Description: description,
			Before:      current,
			After:       after,
		})
		current = after
		return nil
	}

	if wantsFixRule(ruleIDs, CodeConfigInvalid) {
		changed := normalizeVersion(doc)
		if err := addCandidate(CodeConfigInvalid, "Set search descriptor version to 1.0.0.", changed); err != nil {
			return nil, "", err
		}
	}
	if wantsFixRule(ruleIDs, CodeEvalCorpusInvalid) {
		changed := normalizeEvalCorpus(doc)
		if err := addCandidate(CodeEvalCorpusInvalid, "Fill deterministic eval corpus metadata: default suite_id prefixes and missing tests.description.", changed); err != nil {
			return nil, "", err
		}
	}
	if wantsFixRule(ruleIDs, CodeTuningBudgetInvalid) {
		changed := normalizeTuningBudget(doc)
		if err := addCandidate(CodeTuningBudgetInvalid, "Clamp rerank_shortlist to Search Hub's advisory budget ceiling.", changed); err != nil {
			return nil, "", err
		}
	}
	return candidates, current, nil
}

func descriptorPath(scenarioPath string) string {
	return filepath.Join(scenarioPath, ".vrooli", "search.json")
}

func normalizeVersion(doc map[string]any) bool {
	version, _ := doc["version"].(string)
	version = strings.TrimSpace(version)
	if version != "" && semverPattern.MatchString(version) {
		return false
	}
	doc["version"] = defaultSearchVersion
	return true
}

func normalizeEvalCorpus(doc map[string]any) bool {
	changed := false
	for _, provider := range providerObjects(doc) {
		tests, ok := provider["tests"].(map[string]any)
		if !ok {
			continue
		}
		cases, ok := tests["cases"].([]any)
		if !ok || len(cases) == 0 {
			continue
		}
		if strings.TrimSpace(stringValue(tests["description"])) == "" {
			tests["description"] = "Reviewed search eval corpus."
			changed = true
		}
		providerID := strings.TrimSpace(stringValue(provider["provider_id"]))
		suiteID := strings.TrimSpace(stringValue(tests["suite_id"]))
		if providerID != "" && (suiteID == "" || !strings.HasPrefix(suiteID, providerID)) {
			tests["suite_id"] = providerID + ".primary"
			changed = true
		}
	}
	return changed
}

func normalizeTuningBudget(doc map[string]any) bool {
	changed := false
	for _, provider := range providerObjects(doc) {
		tuning, ok := provider["tuning"].(map[string]any)
		if !ok {
			continue
		}
		if !boolValue(tuning["rerank_enabled"]) {
			continue
		}
		if intValue(tuning["rerank_shortlist"]) > 250 {
			tuning["rerank_shortlist"] = float64(250)
			changed = true
		}
	}
	return changed
}

func providerObjects(doc map[string]any) []map[string]any {
	rawProviders, ok := doc["providers"].([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(rawProviders))
	for _, rawProvider := range rawProviders {
		provider, ok := rawProvider.(map[string]any)
		if ok {
			out = append(out, provider)
		}
	}
	return out
}

func marshalSearchJSON(doc map[string]any) (string, error) {
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw) + "\n", nil
}

func normalizeJSON(raw []byte) string {
	var doc any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&doc); err != nil {
		return string(raw)
	}
	normalized, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(normalized) + "\n"
}

func wantsFixRule(ruleIDs []string, ruleID string) bool {
	if len(ruleIDs) == 0 {
		return true
	}
	for _, id := range ruleIDs {
		if strings.EqualFold(strings.TrimSpace(id), ruleID) {
			return true
		}
	}
	return false
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func boolValue(v any) bool {
	b, _ := v.(bool)
	return b
}

func intValue(v any) int {
	switch n := v.(type) {
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}
