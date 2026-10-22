package memberflow

import (
	"strings"
	"testing"
)

func TestLoadOperatingModelDocumentsIgnoresRetainedSourceArchives(t *testing.T) {
	repoRoot := t.TempDir()
	model := `<!-- prompt-manager-graph:
id: scenario-qa-operating-model
scope: team
team: scenario-qa
mode: contract
-->
` + "```mermaid" + `
flowchart LR
  OP["external:operator"]
  A["member:quality-auditor"]
  OP --> A
` + "```" + `
## Mission
Current canonical model.
`
	activeInternalModel := strings.Replace(model, "scenario-qa-operating-model", "current-internal-operating-model", 1)
	activeInternalModel = strings.Replace(activeInternalModel, "team: scenario-qa", "team: current-internal", 1)
	writeRepoFile(t, repoRoot, "docs/scenario-qa/operating/OPERATING_MODEL.md", model)
	writeRepoFile(t, repoRoot, "docs/internal/current-owner/OPERATING_MODEL.md", activeInternalModel)
	writeRepoFile(t, repoRoot, "docs/internal/delivery-reliability-research/sources/A136logs/publication-before/docs/scenario-qa/operating/OPERATING_MODEL.md", model)

	models, err := LoadOperatingModelDocuments(repoRoot)
	if err != nil {
		t.Fatalf("LoadOperatingModelDocuments: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("discovered models=%d, want the canonical and active internal models only: %+v", len(models), models)
	}
	byID := map[string]OperatingModelDocument{}
	for _, model := range models {
		byID[model.ID] = model
	}
	if got := byID["scenario-qa-operating-model"].Source.Path; got != "docs/scenario-qa/operating/OPERATING_MODEL.md" {
		t.Fatalf("canonical model source=%q, want canonical path", got)
	}
	if got := byID["current-internal-operating-model"].Source.Path; got != "docs/internal/current-owner/OPERATING_MODEL.md" {
		t.Fatalf("active internal model source=%q, want active internal path", got)
	}

	blocks, err := LoadOperatingGraphBlocks(repoRoot)
	if err != nil {
		t.Fatalf("LoadOperatingGraphBlocks: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("discovered graph blocks=%d, want both active models", len(blocks))
	}
	for _, block := range blocks {
		if block.Source.Path != "docs/scenario-qa/operating/OPERATING_MODEL.md" &&
			block.Source.Path != "docs/internal/current-owner/OPERATING_MODEL.md" {
			t.Fatalf("unexpected graph source %q; archived sources must not be live", block.Source.Path)
		}
	}
}
