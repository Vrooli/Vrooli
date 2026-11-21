// Package statevocab is the single runtime view of the canonical UX states.
// The authoritative document is capabilities/states.json; the fallback list
// keeps unit tests and isolated tooling deterministic when no repository root
// is available.
package statevocab

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type document struct {
	Canonical []canonicalState `json:"canonical"`
	Views     map[string]struct {
		States []string `json:"states"`
	} `json:"views"`
}

type canonicalState struct {
	ID      string   `json:"id"`
	Aliases []string `json:"aliases"`
}

var fallback = document{
	Canonical: []canonicalState{
		{ID: "default", Aliases: []string{"ready", "idle"}},
		{ID: "loading", Aliases: []string{"pending"}},
		{ID: "submitting"},
		{ID: "saving"},
		{ID: "syncing"},
		{ID: "refreshing"},
		{ID: "empty"},
		{ID: "partial"},
		{ID: "stale"},
		{ID: "success"},
		{ID: "validation-error"},
		{ID: "request-error", Aliases: []string{"error", "failure"}},
		{ID: "permission-denied"},
		{ID: "offline"},
		{ID: "retry", Aliases: []string{"retrying"}},
	},
	Views: map[string]struct {
		States []string `json:"states"`
	}{
		"region-lifecycle": {States: []string{"loading", "default", "empty", "partial", "request-error"}},
		"design-required":  {States: []string{"loading", "empty", "partial", "request-error", "retry", "stale"}},
	},
}

func Load(repoRoot string) document {
	path := filepath.Join(repoRoot, "scenarios", "experience-manager", "capabilities", "states.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	var out document
	if json.Unmarshal(data, &out) != nil || len(out.Canonical) == 0 {
		return fallback
	}
	return out
}

func Canonical(repoRoot string) []string {
	doc := Load(repoRoot)
	out := make([]string, 0, len(doc.Canonical))
	for _, state := range doc.Canonical {
		out = append(out, state.ID)
	}
	return out
}

func Normalize(repoRoot, value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	for _, state := range Load(repoRoot).Canonical {
		if value == state.ID {
			return state.ID
		}
		for _, alias := range state.Aliases {
			if value == alias {
				return state.ID
			}
		}
	}
	return ""
}

func View(repoRoot, name string) []string {
	doc := Load(repoRoot)
	states := append([]string(nil), doc.Views[name].States...)
	sort.Strings(states)
	return states
}

func RegionState(repoRoot, value string) bool {
	normalized := Normalize(repoRoot, value)
	if normalized == "" {
		return false
	}
	for _, state := range View(repoRoot, "region-lifecycle") {
		if normalized == state {
			return true
		}
	}
	return false
}
