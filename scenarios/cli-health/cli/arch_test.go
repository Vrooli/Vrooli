package main

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// TestManifestDeclaresArchitectureOnEveryCommand guards the reference-adopter
// contract: cli-health's own CLI declares a valid renderer-separated primitive
// on every manifest command, so its command_architecture maturity capability
// stays clean at L4. A new command added without an architecture.primitive
// fails here (and would earn arch.primitive_undeclared maturity debt from the
// classifier).
func TestManifestDeclaresArchitectureOnEveryCommand(t *testing.T) {
	m, err := cliapp.ParseManifest(manifestBytes)
	if err != nil {
		t.Fatalf("parse embedded manifest: %v", err)
	}
	for _, g := range m.Groups {
		for _, c := range g.Commands {
			arch := c.Architecture.CommandArchitecture()
			if arch.Primitive == "" {
				t.Errorf("command %s/%s declares no architecture.primitive", g.Name, c.Name)
				continue
			}
			if !arch.Primitive.Valid() {
				t.Errorf("command %s/%s declares unknown primitive %q", g.Name, c.Name, arch.Primitive)
			}
		}
	}
}
