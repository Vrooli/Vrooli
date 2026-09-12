package sttengine

import (
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Consumer is one scenario that declares a resource as a dependency. It backs
// the engine-switch informed prompt: before audio-tools stops using an engine,
// the operator is shown the OTHER scenarios still relying on that engine's
// backing resource so a shared resource is never stopped blindly.
type Consumer struct {
	// Scenario is the scenario directory name (its stable id).
	Scenario string
	// DisplayName is the service.json display/name field, falling back to the
	// directory name when absent.
	DisplayName string
	// Required reports whether the scenario hard-requires the resource
	// (dependencies.resources.<name>.required == true).
	Required bool
}

// ScanResourceConsumers walks a scenarios tree for service.json manifests that
// declare `resource` under dependencies.resources, returning every scenario
// except `exclude` (the current scenario). fsys is rooted at the scenarios
// directory: os.DirFS(scenariosDir) in production, fstest.MapFS in tests. A
// missing/garbled manifest is skipped, not fatal — the scan is advisory.
//
// seam: the fs.FS parameter is the filesystem seam for the cross-scenario
// consumer read (SEAMS.md row "sttengine.ScanResourceConsumers"). Production
// passes os.DirFS; tests pass an in-memory fstest.MapFS.
func ScanResourceConsumers(fsys fs.FS, resource, exclude string) ([]Consumer, error) {
	if fsys == nil || resource == "" {
		return nil, nil
	}
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	var out []Consumer
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		scenario := e.Name()
		if scenario == exclude {
			continue
		}
		manifestPath := path.Join(scenario, ".vrooli", "service.json")
		raw, err := fs.ReadFile(fsys, manifestPath)
		if err != nil {
			continue // not a scenario / no manifest
		}
		var doc struct {
			Name         string `json:"name"`
			DisplayName  string `json:"display_name"`
			Dependencies struct {
				Resources map[string]struct {
					Required bool `json:"required"`
				} `json:"resources"`
			} `json:"dependencies"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			continue
		}
		dep, ok := doc.Dependencies.Resources[resource]
		if !ok {
			continue
		}
		display := strings.TrimSpace(doc.DisplayName)
		if display == "" {
			display = strings.TrimSpace(doc.Name)
		}
		if display == "" {
			display = scenario
		}
		out = append(out, Consumer{Scenario: scenario, DisplayName: display, Required: dep.Required})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Scenario < out[j].Scenario })
	return out, nil
}

// ScenariosFS resolves the scenarios directory and returns a filesystem rooted
// at it. Resolution order: VROOLI_SCENARIOS_DIR; else the parent of
// VROOLI_SCENARIO_DIR (the current scenario's dir); else "". ok is false when
// no scenarios directory can be located — the caller treats the consumer list
// as unknown (the informed prompt still works, just without the list).
func ScenariosFS() (fsys fs.FS, root string, ok bool) {
	if raw, configured := os.LookupEnv("VROOLI_SCENARIOS_DIR"); configured && strings.TrimSpace(raw) != "" {
		dir := strings.TrimSpace(raw)
		if isDir(dir) {
			return os.DirFS(dir), dir, true
		}
	}
	if raw, configured := os.LookupEnv("VROOLI_SCENARIO_DIR"); configured && strings.TrimSpace(raw) != "" {
		scenarioDir := strings.TrimSpace(raw)
		parent := filepath.Dir(filepath.Clean(scenarioDir))
		if isDir(parent) {
			return os.DirFS(parent), parent, true
		}
	}
	return nil, "", false
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
