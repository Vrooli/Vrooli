package dbdetect

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// SourceCollector greps .go source files for known string tokens. It emits
// one observation per distinct matched token, with Count summed across files
// and Locations listing the scenario-relative paths.
//
// The set of tokens to look for is supplied by the profile, not hard-coded
// here. The collector returns observations for every distinct substring it
// found across the scenario; profile matchers filter them down.
type SourceCollector struct{}

func (SourceCollector) Name() string { return "source" }

// candidateTokens is the union of every token the resolver might ask about.
// Populated lazily from the active profiles via SetSourceTokens.
//
// Keeping it package-global is the pragmatic choice — collectors are
// instantiated per-resolver, but token discovery has to happen up-front so
// the source walk only reads each file once.
var registeredSourceTokens []string

// SetSourceTokens registers the union of tokens that the source collector
// should look for in this run. NewResolver calls this with the union derived
// from the active profiles.
func SetSourceTokens(tokens []string) {
	seen := map[string]bool{}
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	registeredSourceTokens = out
}

func (SourceCollector) Collect(_ context.Context, in ScenarioInputs) ([]Observation, error) {
	if in.Filesystem == nil || len(registeredSourceTokens) == 0 {
		return nil, nil
	}
	hits := map[string]*Observation{}
	err := in.Filesystem.Walk(in.ScenarioDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "coverage", "dist", "data", ".next", "vendor":
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := in.Filesystem.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("source read %s: %w", path, readErr)
		}
		content := string(data)
		rel, relErr := filepath.Rel(in.ScenarioDir, path)
		if relErr != nil {
			rel = path
		}
		for _, token := range registeredSourceTokens {
			if strings.Contains(content, token) {
				obs, ok := hits[token]
				if !ok {
					obs = &Observation{Collector: "source", Value: token}
					hits[token] = obs
				}
				obs.Count++
				obs.Locations = append(obs.Locations, rel)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]Observation, 0, len(hits))
	for _, k := range sortStringsCopy(mapKeys(hits)) {
		out = append(out, *hits[k])
	}
	return out, nil
}

func mapKeys(m map[string]*Observation) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
