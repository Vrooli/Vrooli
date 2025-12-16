package main

import (
	"path/filepath"
	"testing"
)

func TestExternalRules_StackGovernorRegistered(t *testing.T) {
	if !isExternalRule("GO_CLI_WORKSPACE_INDEPENDENCE") {
		t.Fatalf("expected GO_CLI_WORKSPACE_INDEPENDENCE to be registered as an external rule")
	}
	if !isExternalRule("REACT_VITE_UI_INSTALLS_DEPENDENCIES") {
		t.Fatalf("expected REACT_VITE_UI_INSTALLS_DEPENDENCIES to be registered as an external rule")
	}
}

func TestPathWithinDir(t *testing.T) {
	base := filepath.Join(string(filepath.Separator), "repo", "scenarios", "git-control-tower")

	if !pathWithinDir(base, base) {
		t.Fatalf("expected base to be within itself")
	}
	if !pathWithinDir(filepath.Join(base, ".vrooli", "service.json"), base) {
		t.Fatalf("expected child path to be within base")
	}
	if pathWithinDir(filepath.Join(string(filepath.Separator), "repo", "scenarios", "other", ".vrooli", "service.json"), base) {
		t.Fatalf("expected sibling scenario to not be within base")
	}
	if pathWithinDir(filepath.Join(base, "..", "other"), base) {
		t.Fatalf("expected .. traversal to not be within base")
	}
}
