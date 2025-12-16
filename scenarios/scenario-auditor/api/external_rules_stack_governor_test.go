package main

import "testing"

func TestExternalRules_StackGovernorRegistered(t *testing.T) {
	if !isExternalRule("GO_CLI_WORKSPACE_INDEPENDENCE") {
		t.Fatalf("expected GO_CLI_WORKSPACE_INDEPENDENCE to be registered as an external rule")
	}
	if !isExternalRule("REACT_VITE_UI_INSTALLS_DEPENDENCIES") {
		t.Fatalf("expected REACT_VITE_UI_INSTALLS_DEPENDENCIES to be registered as an external rule")
	}
}
