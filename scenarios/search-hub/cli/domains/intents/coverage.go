package intents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// scenarioLeaves counts what a scenario exposes to agent discovery.
type scenarioLeaves struct {
	Scenario string `json:"scenario"`
	Skills   int    `json:"skills"`
	Programs int    `json:"programs"`
	Commands int    `json:"commands"`
	// Intents is the number of authored intent cases naming the scenario.
	Intents int `json:"intents"`
}

// Total is the number of discoverable leaves.
func (s scenarioLeaves) Total() int { return s.Skills + s.Programs + s.Commands }

// inventoryLeaves enumerates every scenario under root with its leaf counts,
// reading only authored files (skills/*/SKILL.md, program-runtime programs, and
// cli/manifest.json commands), so the report needs no running scenario.
func inventoryLeaves(root string) ([]scenarioLeaves, error) {
	dirs, err := filepath.Glob(filepath.Join(root, "scenarios", "*"))
	if err != nil {
		return nil, err
	}
	var out []scenarioLeaves
	for _, dir := range dirs {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			continue
		}
		entry := scenarioLeaves{Scenario: filepath.Base(dir)}
		skills, _ := filepath.Glob(filepath.Join(dir, "skills", "*", "SKILL.md"))
		entry.Skills = len(skills)
		programs, _ := filepath.Glob(filepath.Join(dir, ".vrooli", "program-runtime", "*.json"))
		entry.Programs = len(programs)
		entry.Commands = countManifestCommands(filepath.Join(dir, "cli", "manifest.json"))
		if entry.Total() > 0 {
			out = append(out, entry)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Scenario < out[j].Scenario })
	return out, nil
}

// countManifestCommands counts the commands declared in a CLI manifest, in
// either of its group layouts; an unreadable manifest counts as zero.
func countManifestCommands(path string) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	type group struct {
		Commands []json.RawMessage `json:"commands"`
	}
	var doc struct {
		Groups   []group           `json:"groups"`
		Commands []json.RawMessage `json:"commands"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return 0
	}
	n := len(doc.Commands)
	for _, g := range doc.Groups {
		n += len(g.Commands)
	}
	return n
}

// attachIntents fills Intents from the corpus.
func attachIntents(leaves []scenarioLeaves, cases []loadedCase) {
	counts := make(map[string]int)
	for _, c := range cases {
		counts[c.Scenario]++
	}
	for i := range leaves {
		leaves[i].Intents = counts[leaves[i].Scenario]
	}
}

// uncovered returns scenarios that expose leaves but declare no intent. Skills
// and programs are what an agent is steered to first, so they rank ahead of raw
// command counts: the largest authored guidance surface with no measured intent
// is the likeliest undiscoverable capability.
func uncovered(leaves []scenarioLeaves) []scenarioLeaves {
	var out []scenarioLeaves
	for _, l := range leaves {
		if l.Intents == 0 {
			out = append(out, l)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Total() > out[j].Total() })
	return out
}
