package operatingmode

import (
	"path/filepath"
)

type PromptCatalogEntry struct {
	CatalogID   string
	Title       string
	SkillID     string
	Mode        string
	Phase       string
	Trigger     string
	Purpose     string
	OutputPaths []string
}

func PromptCatalogEntries() []PromptCatalogEntry {
	entries := []PromptCatalogEntry{}
	for _, mode := range Modes() {
		def, err := DefinitionFor(mode)
		if err != nil || !def.RunsModeRounds() {
			continue
		}
		for _, phase := range orderedPhases(def) {
			phaseDef := def.PhaseGraph.Phases[phase]
			if phaseDef.Delegated() {
				// A delegated phase has no prompt of its own — the sub-mode's
				// phases carry the catalog entries.
				continue
			}
			entries = append(entries, promptCatalogEntryForPhase(def, phaseDef))
		}
	}
	return entries
}

func ExpectedPromptCatalogEntry(mode, phase string) (PromptCatalogEntry, bool) {
	def, err := DefinitionFor(Mode(mode))
	if err != nil || !def.RunsModeRounds() {
		return PromptCatalogEntry{}, false
	}
	phaseDef, ok := def.PhaseGraph.Phases[Phase(phase)]
	if !ok || phaseDef.Delegated() {
		return PromptCatalogEntry{}, false
	}
	return promptCatalogEntryForPhase(def, phaseDef), true
}

func promptCatalogEntryForPhase(def Definition, phaseDef PhaseDefinition) PromptCatalogEntry {
	return PromptCatalogEntry{
		CatalogID:   phaseDef.CatalogID,
		Title:       phaseDef.PromptCatalog.Title,
		SkillID:     phaseDef.SkillID,
		Mode:        string(def.Mode),
		Phase:       string(phaseDef.Phase),
		Trigger:     phaseDef.PromptCatalog.Trigger,
		Purpose:     phaseDef.PromptCatalog.Purpose,
		OutputPaths: promptCatalogOutputPaths(def, phaseDef),
	}
}

func promptCatalogOutputPaths(def Definition, phaseDef PhaseDefinition) []string {
	paths := make([]string, 0, len(phaseDef.OutputArtifacts)+1)
	for _, artifact := range phaseDef.OutputArtifacts {
		paths = append(paths, filepath.ToSlash(filepath.Clean(artifact.Path)))
	}
	paths = append(paths, filepath.ToSlash(filepath.Join(def.Artifact.RoundRoot, "round-NNN.json")))
	return paths
}
