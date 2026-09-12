package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	skillReferencePattern = regexp.MustCompile(`(?i)prompt-manager\s+skill\s+read\s+([a-z][a-z0-9]*(?:-[a-z0-9]+)*)`)
	commandPattern        = regexp.MustCompile(`\b(vrooli|prompt-manager|swarm-manager|test-genie|scenario-dependency-analyzer|git-control-tower)(?:\s+([a-z][a-z0-9-]*))?`)
)

// Requirements is the portable dependency projection kept in frontmatter.
// The values are identifiers, not filesystem paths, so relocation does not
// invalidate a skill's declared requirements.
type Requirements struct {
	Scenarios []string `json:"scenarios,omitempty"`
	Commands  []string `json:"commands,omitempty"`
}

// InferRequirements derives the explicit skill and command references from a
// body. It is intentionally conservative: unresolved prose is reported to the
// caller rather than guessed into a distribution gate.
func InferRequirements(body string, knownSkills map[string]bool) (Requirements, []string) {
	scenarios := map[string]bool{}
	commands := map[string]bool{}
	unresolved := map[string]bool{}
	for _, match := range skillReferencePattern.FindAllStringSubmatch(body, -1) {
		id := strings.ToLower(match[1])
		if knownSkills != nil && !knownSkills[id] {
			unresolved["skill:"+id] = true
			continue
		}
		commands["prompt-manager skill read"] = true
		scenarios["prompt-manager"] = true
	}
	for _, match := range commandPattern.FindAllStringSubmatch(body, -1) {
		binary := strings.ToLower(match[1])
		subcommand := strings.ToLower(match[2])
		if subcommand == "" {
			commands[binary] = true
		} else {
			commands[binary+" "+subcommand] = true
		}
		scenarios[binary] = true
	}
	return Requirements{Scenarios: sortedKeys(scenarios), Commands: sortedKeys(commands)}, sortedKeys(unresolved)
}

// SyncCorpusRequirements updates the live frontmatter and compatibility
// sidecars, and returns unresolved identifiers instead of silently dropping
// them. Existing metadata outside requires is preserved byte-for-byte.
func SyncCorpusRequirements(root string) (map[string][]string, error) {
	paths := make([]string, 0)
	known := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != "SKILL.md" {
			return nil
		}
		paths = append(paths, path)
		known[filepath.Base(filepath.Dir(path))] = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	report := make(map[string][]string)
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		reqs, unresolved := InferRequirements(StripFrontmatter(string(content)), known)
		if len(unresolved) > 0 {
			report[path] = unresolved
		}
		updated, err := replaceFrontmatterRequirements(string(content), reqs)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			return nil, err
		}
		if err := syncSidecar(filepath.Join(filepath.Dir(path), "skill.json"), reqs); err != nil {
			return nil, fmt.Errorf("%s sidecar: %w", path, err)
		}
	}
	return report, nil
}

func replaceFrontmatterRequirements(content string, reqs Requirements) (string, error) {
	lines := strings.SplitAfter(content, "\n")
	start, end := -1, -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "requires:" && strings.HasPrefix(line, "  ") {
			start = i
			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(lines[j], "  ") && !strings.HasPrefix(lines[j], "    ") {
					end = j
					break
				}
			}
			if end < 0 {
				end = i + 3
			}
			break
		}
	}
	if start < 0 || end <= start {
		return content, fmt.Errorf("frontmatter requires block not found")
	}
	block := []string{
		"  requires:\n",
		"    scenarios: [" + quoteList(reqs.Scenarios) + "]\n",
		"    commands: [" + quoteList(reqs.Commands) + "]\n",
	}
	result := append([]string{}, lines[:start]...)
	result = append(result, block...)
	result = append(result, lines[end:]...)
	return strings.Join(result, ""), nil
}

func syncSidecar(path string, reqs Requirements) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	requires, _ := object["requires"].(map[string]any)
	if requires == nil {
		requires = map[string]any{}
	}
	requires["scenarios"] = reqs.Scenarios
	requires["commands"] = reqs.Commands
	object["requires"] = requires
	encoded, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(encoded, '\n'), 0o644)
}

func quoteList(values []string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = strconv.Quote(value)
	}
	return strings.Join(quoted, ", ")
}

func sortedKeys(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
