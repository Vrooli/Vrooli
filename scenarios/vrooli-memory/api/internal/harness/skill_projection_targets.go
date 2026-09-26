package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SkillProjectionTarget is the data contract shared by coding-agent
// resources and all projectors. A new harness is declared by adding a
// resource.json entry; the loader does not contain a runtime switch.
type SkillProjectionTarget struct {
	Runtime      string
	PathTemplate string
	Environment  string
	ProjectScope bool
}

type resourceProjectionDocument struct {
	Name    string `json:"name"`
	Storage struct {
		Entries map[string]json.RawMessage `json:"entries"`
	} `json:"storage"`
	MemoryTargets []struct {
		Runtime string `json:"runtime"`
		Path    string `json:"path"`
		Section string `json:"section"`
		ByteCap int    `json:"byte_cap"`
		LineCap int    `json:"line_cap"`
	} `json:"vrooli_memory_targets"`
}

// LoadSkillProjectionTargets discovers every resource declaring a skills
// storage entry. It is intentionally directory-driven and accepts fixture
// resources, which keeps the six-harness conformance test code-free.
func LoadSkillProjectionTargets(resourcesDir string) ([]SkillProjectionTarget, error) {
	entries, err := os.ReadDir(resourcesDir)
	if err != nil {
		return nil, fmt.Errorf("read resources directory: %w", err)
	}
	var targets []SkillProjectionTarget
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(resourcesDir, entry.Name(), "resource.json")
		data, readErr := os.ReadFile(path)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", path, readErr)
		}
		var doc resourceProjectionDocument
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		rawSkill, ok := doc.Storage.Entries["skills"]
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
		if err := json.Unmarshal(rawSkill, &skill); err != nil || strings.TrimSpace(skill.Path) == "" {
			continue
		}
		runtime := doc.Name
		if runtime == "" {
			runtime = entry.Name()
		}
		targets = append(targets, SkillProjectionTarget{Runtime: runtime, PathTemplate: skill.Path, Environment: skill.Projection.Environment, ProjectScope: skill.Projection.ProjectScope})
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Runtime < targets[j].Runtime })
	return targets, nil
}

// ResolveSkillProjectionPath expands only the portable home token. Other
// environment placeholders remain caller-controlled and are not guessed.
func ResolveSkillProjectionPath(template string, home string) string {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Clean(strings.Replace(template, "$USER_HOME", home, 1))
}

func LoadMemoryProjectionTargets(resourcesDir string, home string, workspace string) (map[string]projectionTarget, error) {
	entries, err := os.ReadDir(resourcesDir)
	if err != nil {
		return nil, fmt.Errorf("read resources directory: %w", err)
	}
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	if workspace == "" {
		workspace, _ = os.Getwd()
	}
	workspace, _ = filepath.Abs(workspace)
	workspaceKey := "-" + strings.Trim(strings.NewReplacer(string(filepath.Separator), "-", ":", "-", "_", "-").Replace(workspace), "-")
	targets := make(map[string]projectionTarget)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(resourcesDir, entry.Name(), "resource.json")
		data, readErr := os.ReadFile(path)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", path, readErr)
		}
		var doc resourceProjectionDocument
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		for _, target := range doc.MemoryTargets {
			if target.Runtime == "" || target.Path == "" {
				continue
			}
			resolved := strings.Replace(target.Path, "$USER_HOME", home, 1)
			resolved = strings.Replace(resolved, "$WORKSPACE_KEY", workspaceKey, 1)
			byteCap, lineCap := target.ByteCap, target.LineCap
			if byteCap <= 0 {
				byteCap = 32768
			}
			if lineCap <= 0 {
				lineCap = 200
			}
			targets[target.Runtime] = projectionTarget{Runtime: target.Runtime, Path: filepath.Clean(resolved), Section: target.Section, Cap: byteCap, LineCap: lineCap}
		}
	}
	return targets, nil
}

func discoverResourcesDir() string {
	if configured := strings.TrimSpace(os.Getenv("VROOLI_RESOURCES_DIR")); configured != "" {
		return configured
	}
	working, err := os.Getwd()
	if err != nil {
		return ""
	}
	for current := filepath.Clean(working); ; current = filepath.Dir(current) {
		candidate := filepath.Join(current, "resources")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
	}
}
