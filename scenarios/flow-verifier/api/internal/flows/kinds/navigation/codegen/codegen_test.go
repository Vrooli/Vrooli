package codegen_test

import (
	"strings"
	"testing"

	"flow-verifier/internal/flows/kind"
	"flow-verifier/internal/flows/kinds/navigation"
	"flow-verifier/internal/flows/kinds/navigation/codegen"
	"flow-verifier/internal/flows/schemas"
)

func TestTypeScriptFullExample(t *testing.T) {
	k, _ := kind.Get(navigation.Name)
	spec, err := k.Load(schemas.NavigationFullExample, "schemas/examples/navigation-full.json")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	g := spec.(*navigation.Spec).Graph()
	src, err := codegen.TypeScript(g)
	if err != nil {
		t.Fatalf("TypeScript: %v", err)
	}
	got := string(src)

	mustContain := []string{
		"DO NOT EDIT",
		"export const ROUTES = {",
		"home: \"/\",",
		// Parameterised route: /tasks/:id → camelCase taskDetail accessor.
		// Path-param identifier is emitted verbatim (no lowercasing).
		"taskDetail: (id: string): string => `/tasks/${id}`,",
		"login: \"/login\",",
		"adminUsers: \"/admin/users\",",
		"settingsDisplay: \"/settings/display\",",
		// ROUTE_PATTERNS exposes the raw spec path for React Router declarations.
		"export const ROUTE_PATTERNS = {",
		"taskDetail: \"/tasks/:id\",",
		"export const PUBLIC_ROUTES",
		"export const AUTH_REQUIRED_ROUTES",
		"export const ROUTE_IDS",
		// AUTH_REQUIRED_ROUTES should contain home and tasks_index (camelCased).
		"\"home\"",
		"\"tasksIndex\"",
	}
	for _, s := range mustContain {
		if !strings.Contains(got, s) {
			t.Errorf("output missing %q\n---\n%s", s, got)
		}
	}
}
