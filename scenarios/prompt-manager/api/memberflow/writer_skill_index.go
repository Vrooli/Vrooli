package memberflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// LoadWriterSkillProducers walks the prompt-manager skill registry under
// <repoRoot>/scenarios/prompt-manager/store/skills/packs/<pack>/<id>/skill.json
// and returns the union of `writes_to[]` declarations across every skill
// tagged "writer-skill".
//
// Writer-skill writes_to[] is the producer-side declaration for skill-written
// prefixes: classifier and generic skills must be portable across teams and
// may not reference topic prefixes, but writer skills carry a producer
// registration that the validator treats as a first-class declared write.
//
// The returned slice is sorted for determinism. An empty slice (no error) is
// returned when the registry is missing or contains no writer skills — the
// caller treats that as "no writer-skill producers known," not an error.
//
// repoRoot is the absolute path to the repository root (the parent of
// scenarios/, docs/, etc.). When repoRoot is empty, returns (nil, nil) so
// the lazy-load in Validate stays a silent no-op for tests.
func LoadWriterSkillProducers(repoRoot string) ([]string, error) {
	if repoRoot == "" {
		return nil, nil
	}
	packsDir := filepath.Join(repoRoot, "scenarios", "prompt-manager", "store", "skills", "packs")
	packs, err := os.ReadDir(packsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("memberflow: read skill packs %q: %w", packsDir, err)
	}

	seen := map[string]bool{}
	for _, p := range packs {
		if !p.IsDir() {
			continue
		}
		skills, err := os.ReadDir(filepath.Join(packsDir, p.Name()))
		if err != nil {
			continue
		}
		for _, s := range skills {
			if !s.IsDir() {
				continue
			}
			path := filepath.Join(packsDir, p.Name(), s.Name(), "skill.json")
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var raw struct {
				Tags     []string `json:"tags"`
				WritesTo []string `json:"writes_to"`
			}
			if err := json.Unmarshal(data, &raw); err != nil {
				// Malformed skill.json: silently skipped. The skill-side
				// validation suite is the right place for shape errors;
				// this loader is consumed by a separate cross-graph rule
				// and should be tolerant.
				continue
			}
			isWriter := false
			for _, tag := range raw.Tags {
				if tag == "writer-skill" {
					isWriter = true
					break
				}
			}
			if !isWriter {
				continue
			}
			for _, prefix := range raw.WritesTo {
				if prefix == "" {
					continue
				}
				seen[prefix] = true
			}
		}
	}

	out := make([]string, 0, len(seen))
	for prefix := range seen {
		out = append(out, prefix)
	}
	sort.Strings(out)
	return out, nil
}
