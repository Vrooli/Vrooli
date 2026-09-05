package main

import (
	_ "embed"
	"encoding/json"
	"testing"
)

//go:embed manifest.json
var onboardingCLIManifest []byte

type manifestGroup struct {
	Name     string            `json:"name"`
	Commands []manifestCommand `json:"commands"`
	Groups   []manifestGroup   `json:"groups"`
}

type manifestCommand struct {
	Name       string          `json:"name"`
	Binding    json.RawMessage `json:"binding"`
	Governance struct {
		Effect string `json:"effect"`
	} `json:"governance"`
}

func TestCLIManifestDeclaresGovernanceForEveryCommand(t *testing.T) {
	var manifest struct {
		Name   string          `json:"name"`
		Groups []manifestGroup `json:"groups"`
	}
	if err := json.Unmarshal(onboardingCLIManifest, &manifest); err != nil {
		t.Fatalf("manifest JSON: %v", err)
	}
	if manifest.Name != appName {
		t.Fatalf("manifest name = %q, want %q", manifest.Name, appName)
	}
	var visit func([]manifestGroup, string)
	visit = func(groups []manifestGroup, prefix string) {
		for _, group := range groups {
			path := group.Name
			if prefix != "" {
				path = prefix + " " + path
			}
			for _, command := range group.Commands {
				if len(command.Binding) == 0 {
					t.Errorf("%s %s has no binding", path, command.Name)
				}
				if command.Governance.Effect == "" {
					t.Errorf("%s %s has no governance effect", path, command.Name)
				}
			}
			visit(group.Groups, path)
		}
	}
	visit(manifest.Groups, "")
}
