package templateengine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/config"
)

// relocationInheritanceRepo writes a base template whose proto/ is relocated
// into packages/proto and a child that inherits the base through baseTemplate.
func relocationInheritanceRepo(t *testing.T, childManifest string, childFiles map[string]string) string {
	t.Helper()
	root := t.TempDir()
	templates := filepath.Join(root, "templates", "scenarios")
	t.Setenv(config.TemplateBaseDirEnvVar, templates)
	files := map[string]string{
		"base/template.json":        `{"name":"base","relocations":[{"description":"base protos","from":"proto/","to":"packages/proto/schemas/{{SCENARIO_ID}}/"}]}`,
		"base/proto/v1/notes.proto": "syntax = \"proto3\";\n",
		"base/cli/main.go":          "package main\n",
		"base/api/main.go":          "package main\n",
		"child/template.json":       childManifest,
	}
	for name, body := range childFiles {
		files[name] = body
	}
	for name, body := range files {
		path := filepath.Join(templates, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestChildTemplateInheritsBaseRelocationForInheritedPaths(t *testing.T) {
	root := relocationInheritanceRepo(t, `{"name":"child","baseTemplate":"base","baseCopyExcludes":["api"]}`, nil)
	info, err := loadTemplate(root, "child")
	if err != nil {
		t.Fatal(err)
	}
	baseDir := filepath.Join(root, "templates", "scenarios", "base")
	if len(info.Manifest.Relocations) != 1 || info.Manifest.Relocations[0].SourceDir != baseDir {
		t.Fatalf("relocations = %+v, want the base proto relocation sourced from the base", info.Manifest.Relocations)
	}
	resolved, err := resolveRelocations(root, info, map[string]string{"SCENARIO_ID": "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 1 || resolved[0].SourceDir != baseDir || resolved[0].To != filepath.Join(root, "packages", "proto", "schemas", "demo") {
		t.Fatalf("resolved = %+v", resolved)
	}
	if err := copyRelocationTree(filepath.Join(resolved[0].SourceDir, resolved[0].From), resolved[0].To, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(resolved[0].To, "v1", "notes.proto")); err != nil {
		t.Fatalf("inherited relocation must deliver the base protos: %v", err)
	}

	// The scenario copy still omits the relocation source and the excluded path.
	destination := filepath.Join(root, "generated")
	if err := copyTemplate(info.Path, destination, map[string]string{"SCENARIO_ID": "demo"}, info.Manifest); err != nil {
		t.Fatal(err)
	}
	for _, absent := range []string{"proto", "api"} {
		if _, err := os.Stat(filepath.Join(destination, absent)); !os.IsNotExist(err) {
			t.Fatalf("%s must not be copied into the scenario: %v", absent, err)
		}
	}
	if _, err := os.Stat(filepath.Join(destination, "cli", "main.go")); err != nil {
		t.Fatalf("inherited base files must be copied: %v", err)
	}
}

func TestChildTemplateRelocationOverridesBaseRelocation(t *testing.T) {
	root := relocationInheritanceRepo(t,
		`{"name":"child","baseTemplate":"base","relocations":[{"description":"child protos","from":"proto","to":"packages/proto/schemas/{{SCENARIO_ID}}-child/"}]}`,
		map[string]string{"child/proto/v1/child.proto": "syntax = \"proto3\";\n"})
	info, err := loadTemplate(root, "child")
	if err != nil {
		t.Fatal(err)
	}
	childDir := filepath.Join(root, "templates", "scenarios", "child")
	if len(info.Manifest.Relocations) != 1 || info.Manifest.Relocations[0].SourceDir != childDir || info.Manifest.Relocations[0].Description != "child protos" {
		t.Fatalf("relocations = %+v, want only the child's override", info.Manifest.Relocations)
	}
	resolved, err := resolveRelocations(root, info, map[string]string{"SCENARIO_ID": "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 1 || resolved[0].SourceDir != childDir || resolved[0].To != filepath.Join(root, "packages", "proto", "schemas", "demo-child") {
		t.Fatalf("resolved = %+v", resolved)
	}
}

func TestChildTemplateExcludingBasePathDropsItsRelocation(t *testing.T) {
	root := relocationInheritanceRepo(t, `{"name":"child","baseTemplate":"base","baseCopyExcludes":["proto"]}`, nil)
	info, err := loadTemplate(root, "child")
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Manifest.Relocations) != 0 {
		t.Fatalf("relocations = %+v, want none for an excluded base path", info.Manifest.Relocations)
	}
}

func TestTemplateInheritanceCycleRefuses(t *testing.T) {
	root := relocationInheritanceRepo(t, `{"name":"child","baseTemplate":"child"}`, nil)
	if _, err := loadTemplate(root, "child"); err == nil {
		t.Fatal("a template that inherits itself must refuse")
	}
}
