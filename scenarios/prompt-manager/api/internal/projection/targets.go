package projection

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Target is a resource-declared native skill directory. The resource owns
// the path contract; prompt-manager only expands the portable home token and
// writes generated skill content into it.
type Target struct {
	Runtime      string
	PathTemplate string
	Path         string
	Environment  string
	ProjectScope bool
}

type resourceDocument struct {
	Name    string `json:"name"`
	Storage struct {
		Entries map[string]json.RawMessage `json:"entries"`
	} `json:"storage"`
}

// LoadTargets discovers every resource with a skills storage entry. A new
// harness is therefore a resource-data change, not a switch in this package.
func LoadTargets(resourcesDir, home string) ([]Target, error) {
	if strings.TrimSpace(resourcesDir) == "" {
		return nil, fmt.Errorf("resource directory is required")
	}
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve user home: %w", err)
		}
	}
	entries, err := os.ReadDir(resourcesDir)
	if err != nil {
		return nil, fmt.Errorf("read resources directory: %w", err)
	}
	seenRuntime := map[string]string{}
	seenPath := map[string]string{}
	var targets []Target
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(resourcesDir, entry.Name(), "resource.json")
		data, readErr := os.ReadFile(manifestPath)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", manifestPath, readErr)
		}
		var document resourceDocument
		if err := json.Unmarshal(data, &document); err != nil {
			return nil, fmt.Errorf("parse %s: %w", manifestPath, err)
		}
		raw, ok := document.Storage.Entries["skills"]
		if !ok {
			continue
		}
		var skill struct {
			Path       string `json:"path"`
			Projection struct {
				Environment  string `json:"environment"`
				ProjectScope bool   `json:"project_scope"`
			} `json:"projection"`
		}
		if err := json.Unmarshal(raw, &skill); err != nil {
			return nil, fmt.Errorf("parse %s storage.entries.skills: %w", manifestPath, err)
		}
		if strings.TrimSpace(skill.Path) == "" {
			return nil, fmt.Errorf("%s storage.entries.skills.path is required", manifestPath)
		}
		runtime := strings.TrimSpace(document.Name)
		if runtime == "" {
			runtime = entry.Name()
		}
		path := filepath.Clean(strings.Replace(skill.Path, "$USER_HOME", home, 1))
		if previous, exists := seenRuntime[runtime]; exists {
			return nil, fmt.Errorf("duplicate skill projection runtime %q in %s and %s", runtime, previous, manifestPath)
		}
		if previous, exists := seenPath[path]; exists {
			return nil, fmt.Errorf("duplicate skill projection path %q in %s and %s", path, previous, manifestPath)
		}
		seenRuntime[runtime] = manifestPath
		seenPath[path] = manifestPath
		targets = append(targets, Target{Runtime: runtime, PathTemplate: skill.Path, Path: path, Environment: skill.Projection.Environment, ProjectScope: skill.Projection.ProjectScope})
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Runtime < targets[j].Runtime })
	return targets, nil
}

// ProjectTargets applies one bounded pack to all declared targets. A failed
// target is returned with its runtime so callers can report a degraded target
// without hiding successful projections.
func ProjectTargets(sourceRoot string, targets []Target, pack BasePack) ([]Result, error) {
	results := make([]Result, 0, len(targets))
	for _, target := range targets {
		result, err := Project(sourceRoot, target.Path, pack)
		if err != nil {
			return results, fmt.Errorf("project %s: %w", target.Runtime, err)
		}
		results = append(results, result)
	}
	return results, nil
}
