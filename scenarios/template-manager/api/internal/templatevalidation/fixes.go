package templatevalidation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/maturity-go/autofix"
	repocontract "github.com/vrooli/repo-contract-go"
)

type FixRegistry struct {
	repoRoot string
	registry *autofix.Registry
}

func NewFixRegistry(repoRoot string) *FixRegistry {
	r := &FixRegistry{repoRoot: repoRoot}
	r.registry = autofix.NewRegistry(autofix.Fixer{
		RuleID:  CodeProvenanceMissing,
		Preview: r.previewProvenanceMissing,
		CanFix:  serviceManifestCanFix,
	})
	return r
}

func (r *FixRegistry) Preview(root string, ruleIDs []string) ([]autofix.Candidate, error) {
	return r.registry.Preview(root, ruleIDs)
}

func (r *FixRegistry) Apply(root string, ruleIDs []string) ([]autofix.Candidate, error) {
	return r.registry.Apply(root, ruleIDs)
}

func (r *FixRegistry) previewProvenanceMissing(root string) ([]autofix.Candidate, error) {
	servicePath := filepath.Join(root, ".vrooli", "service.json")
	beforeBytes, err := os.ReadFile(servicePath)
	if err != nil {
		return nil, nil
	}
	before := string(beforeBytes)
	var doc map[string]any
	if err := json.Unmarshal(beforeBytes, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", servicePath, err)
	}
	generation, _ := doc["generation"].(map[string]any)
	templateID, templateVersion := generationTemplate(generation)
	if templateID != "" && templateVersion != "" {
		return nil, nil
	}
	repoRoot := strings.TrimSpace(r.repoRoot)
	if repoRoot == "" {
		resolved, err := repocontract.ResolveRepoRoot()
		if err != nil {
			return nil, err
		}
		repoRoot = resolved
	}
	id, version, err := LatestDefaultTemplate(repoRoot)
	if err != nil {
		return nil, err
	}
	if generation == nil {
		generation = map[string]any{}
		doc["generation"] = generation
	}
	generation["template"] = map[string]any{"id": id, "version": version}
	generation["generated_at"] = time.Now().UTC().Format(time.RFC3339)
	generation["adopted"] = true
	afterBytes, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	after := string(afterBytes) + "\n"
	if before == after {
		return nil, nil
	}
	return []autofix.Candidate{{
		RuleID:      CodeProvenanceMissing,
		FilePath:    servicePath,
		Description: "Stamp adopted generation provenance for the latest default scenario template.",
		Before:      before,
		After:       after,
	}}, nil
}

func generationTemplate(generation map[string]any) (string, string) {
	if generation == nil {
		return "", ""
	}
	template, _ := generation["template"].(map[string]any)
	id, _ := template["id"].(string)
	version, _ := template["version"].(string)
	return strings.TrimSpace(id), strings.TrimSpace(version)
}

func serviceManifestCanFix(root, _ string) bool {
	_, err := os.Stat(filepath.Join(root, ".vrooli", "service.json"))
	return err == nil
}
