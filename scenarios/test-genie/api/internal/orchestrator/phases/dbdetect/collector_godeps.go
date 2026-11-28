package dbdetect

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

// GodepsCollector parses every go.mod under the scenario directory and emits
// one observation per direct or indirect require with the import-path value.
type GodepsCollector struct{}

func (GodepsCollector) Name() string { return "godeps" }

func (GodepsCollector) Collect(_ context.Context, in ScenarioInputs) ([]Observation, error) {
	if in.Filesystem == nil {
		return nil, nil
	}
	var modFiles []string
	walkErr := in.Filesystem.Walk(in.ScenarioDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "coverage", "dist", "data", ".next", "vendor":
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Base(path) == "go.mod" {
			modFiles = append(modFiles, path)
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("godeps walk: %w", walkErr)
	}

	seen := map[string]*Observation{}
	for _, p := range modFiles {
		data, err := in.Filesystem.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("godeps read %s: %w", p, err)
		}
		mf, err := modfile.Parse(p, data, nil)
		if err != nil {
			return nil, fmt.Errorf("godeps parse %s: %w", p, err)
		}
		for _, req := range mf.Require {
			path := strings.TrimSpace(req.Mod.Path)
			if path == "" {
				continue
			}
			obs, ok := seen[path]
			if !ok {
				obs = &Observation{Collector: "godeps", Value: path, Count: 0}
				seen[path] = obs
			}
			obs.Count++
			obs.Locations = append(obs.Locations, p)
		}
	}
	out := make([]Observation, 0, len(seen))
	for _, k := range sortedKeys(seen) {
		out = append(out, *seen[k])
	}
	return out, nil
}

func sortedKeys(m map[string]*Observation) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sortStrings(keys)
	return keys
}
