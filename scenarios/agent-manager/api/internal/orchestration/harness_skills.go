// Responsibility: read where each harness discovers native skills, and what
// relocates that directory, from the harness's resource manifest.
package orchestration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"agent-manager/internal/domain"
)

// HarnessSkillStorage is a harness's native skill discovery directory as its
// resource manifest declares it (resources/<harness>/resource.json,
// storage.entries.skills). The resource owns this contract; Agent Manager only
// relocates the directory into a run-private root.
type HarnessSkillStorage struct {
	// RelocationKey is the variable that moves the directory, or "" when no
	// variable does (the harness reads a pinned path).
	RelocationKey string
	// RelativePath is the skills directory beneath the root RelocationKey
	// names, e.g. "skills" for CODEX_HOME or "opencode/skills" for
	// XDG_CONFIG_HOME.
	RelativePath string
	// DefaultRoot is the root RelocationKey replaces when it is unset.
	DefaultRoot string
	// Rationale is the manifest's explanation, quoted in refusals.
	Rationale string
}

// Relocatable reports whether a run-private root can hold the skills.
func (s HarnessSkillStorage) Relocatable() bool { return s.RelocationKey != "" && s.RelativePath != "" }

// SharedBaseDirectory reports whether the relocation key is an XDG base
// directory: it moves every XDG-aware program in the run's process tree, not
// only the harness, so the private root must keep the operator's other
// configuration visible.
func (s HarnessSkillStorage) SharedBaseDirectory() bool { return xdgDefaults[s.RelocationKey] != "" }

// xdgDefaults are the XDG Base Directory specification defaults, relative to
// the user's home. They belong to the specification, not to any harness.
var xdgDefaults = map[string]string{
	"XDG_CONFIG_HOME": ".config",
	"XDG_DATA_HOME":   filepath.Join(".local", "share"),
	"XDG_STATE_HOME":  filepath.Join(".local", "state"),
	"XDG_CACHE_HOME":  ".cache",
}

var (
	harnessSkillsOnce sync.Once
	harnessSkills     map[domain.RunnerType]HarnessSkillStorage
	harnessSkillsErr  error
	// harnessResourcesDir resolves the resource manifests; tests replace it.
	harnessResourcesDir = defaultHarnessResourcesDir
)

// defaultHarnessResourcesDir is $PROJECT_ROOT/resources, else the nearest
// ancestor of the working directory that holds a resources/ directory.
func defaultHarnessResourcesDir() string {
	if root := strings.TrimSpace(os.Getenv("PROJECT_ROOT")); root != "" {
		return filepath.Join(root, "resources")
	}
	dir, err := os.Getwd()
	if err != nil {
		return "resources"
	}
	for {
		candidate := filepath.Join(dir, "resources")
		if matches, _ := filepath.Glob(filepath.Join(candidate, "*", "resource.json")); len(matches) > 0 {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "resources"
		}
		dir = parent
	}
}

// harnessSkillStorage returns the runner's declared skill storage. A runner
// whose manifest declares no skills entry returns ok=false.
func harnessSkillStorage(runnerType domain.RunnerType) (HarnessSkillStorage, bool, error) {
	harnessSkillsOnce.Do(func() {
		harnessSkills, harnessSkillsErr = loadHarnessSkillStorage(harnessResourcesDir())
	})
	if harnessSkillsErr != nil {
		return HarnessSkillStorage{}, false, harnessSkillsErr
	}
	storage, ok := harnessSkills[runnerType]
	return storage, ok, nil
}

// loadHarnessSkillStorage reads every resource manifest under dir that
// declares storage.entries.skills, keyed by the resource name (the runner
// type). The relocation root is the XDG default for a base-directory key, and
// otherwise the manifest's own home entry.
func loadHarnessSkillStorage(dir string) (map[domain.RunnerType]HarnessSkillStorage, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read harness resource manifests: %w", err)
	}
	type storageEntry struct {
		Path       string `json:"path"`
		Rationale  string `json:"rationale"`
		Relocation *struct {
			Key string `json:"key"`
		} `json:"relocation"`
	}
	out := map[domain.RunnerType]HarnessSkillStorage{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name(), "resource.json")
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var manifest struct {
			Name    string `json:"name"`
			Storage struct {
				Entries map[string]json.RawMessage `json:"entries"`
			} `json:"storage"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		rawSkills, ok := manifest.Storage.Entries["skills"]
		if !ok {
			continue
		}
		var skills storageEntry
		if err := json.Unmarshal(rawSkills, &skills); err != nil || strings.TrimSpace(skills.Path) == "" {
			return nil, fmt.Errorf("%s: storage.entries.skills needs a path: %v", path, err)
		}
		name := strings.TrimSpace(manifest.Name)
		if name == "" {
			name = entry.Name()
		}
		storage := HarnessSkillStorage{Rationale: skills.Rationale}
		if skills.Relocation != nil && strings.TrimSpace(skills.Relocation.Key) != "" {
			storage.RelocationKey = strings.TrimSpace(skills.Relocation.Key)
			root := ""
			if xdg := xdgDefaults[storage.RelocationKey]; xdg != "" {
				root = "$USER_HOME/" + filepath.ToSlash(xdg)
			} else if rawHome, ok := manifest.Storage.Entries["home"]; ok {
				var home storageEntry
				if json.Unmarshal(rawHome, &home) == nil {
					root = home.Path
				}
			}
			relative := strings.TrimPrefix(skills.Path, strings.TrimSuffix(root, "/")+"/")
			if root == "" || relative == skills.Path || strings.HasPrefix(relative, "..") {
				return nil, fmt.Errorf("%s: skills path %s is not beneath the root %s relocates", path, skills.Path, storage.RelocationKey)
			}
			storage.DefaultRoot, storage.RelativePath = root, filepath.FromSlash(relative)
		}
		out[domain.RunnerType(name)] = storage
	}
	return out, nil
}

// RunnerSkillsDir is the skills directory a run of runnerType discovers given
// its prepared environment. It refuses a runner whose harness cannot be
// pointed at a run-private directory, so a skill pack fails loudly instead of
// silently missing the run.
func RunnerSkillsDir(runnerType domain.RunnerType, env map[string]string) (string, error) {
	storage, ok, err := harnessSkillStorage(runnerType)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("skill scoping is unsupported for %s: its resource manifest declares no skills directory", runnerType)
	}
	if !storage.Relocatable() {
		return "", fmt.Errorf("skill scoping is unsupported for %s: no variable relocates its skills directory (%s)", runnerType, storage.Rationale)
	}
	root := strings.TrimSpace(env[storage.RelocationKey])
	if root == "" {
		return "", fmt.Errorf("skill scoping for %s needs a run-private %s, which was not prepared", runnerType, storage.RelocationKey)
	}
	return filepath.Join(root, storage.RelativePath), nil
}
