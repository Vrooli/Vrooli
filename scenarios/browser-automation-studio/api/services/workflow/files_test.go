package files

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

func TestSanitizeWorkflowSlug(t *testing.T) {
	cases := map[string]string{
		"Simple Name":          "simple-name",
		"   Leading  spaces  ": "leading-spaces",
		"Symbols@#!":           "symbols",
		"":                     "workflow",
	}

	for input, expected := range cases {
		if slug := sanitizeWorkflowSlug(input); slug != expected {
			t.Fatalf("sanitizeWorkflowSlug(%q) = %q, expected %q", input, slug, expected)
		}
	}
}

func TestNormalizeFolderPath(t *testing.T) {
	cases := map[string]string{
		"":           "/",
		" ":          "/",
		"demo":       "/demo",
		"demo/tests": "/demo/tests",
		"/demo/":     "/demo",
		`demo\\a`:    "/demo/a",
	}

	for input, expected := range cases {
		if out := normalizeFolderPath(input); out != expected {
			t.Fatalf("normalizeFolderPath(%q) = %q, expected %q", input, out, expected)
		}
	}
}

func TestDesiredWorkflowFilePath(t *testing.T) {
	project := &database.Project{FolderPath: "/tmp/projects/demo"}
	workflow := &database.Workflow{
		ID:         uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Name:       "Login Flow",
		FolderPath: "/auth",
	}

	abs, rel := (&WorkflowService{}).desiredWorkflowFilePath(project, workflow)
	if rel == "" {
		t.Fatalf("expected relative path to be populated")
	}
	if !strings.Contains(rel, "login-flow--11111111.workflow.json") {
		t.Fatalf("unexpected relative path %q", rel)
	}
	expectedAbsSuffix := filepath.Join("tmp", "projects", "demo", "workflows", filepath.FromSlash(rel))
	if !strings.HasSuffix(abs, expectedAbsSuffix) {
		t.Fatalf("absolute path %q did not end with %q", abs, expectedAbsSuffix)
	}
}
