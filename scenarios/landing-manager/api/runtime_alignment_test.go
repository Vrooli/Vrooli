package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// [REQ:TMPL-RUNTIME-ALIGNMENT]
func TestTODOImplementationPlanReferenced(t *testing.T) {
	targets := []struct {
		name string
		path string
	}{
		{
			name: "Landing Manager README",
			path: filepath.Join("..", "README.md"),
		},
		{
			name: "Landing Manager PRD",
			path: filepath.Join("..", "PRD.md"),
		},
		{
			name: "Landing Manager CLI help text",
			path: filepath.Join("..", "cli", "landing-manager"),
		},
	}

	const requiredSubstring = "scripts/scenarios/templates/landing-page-react-vite/PRD.md"

	for _, target := range targets {
		t.Run(target.name, func(t *testing.T) {
			data, err := os.ReadFile(target.path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", target.path, err)
			}

			if !strings.Contains(string(data), requiredSubstring) {
				t.Fatalf("%s (%s) does not mention %s", target.name, target.path, requiredSubstring)
			}
		})
	}
}
