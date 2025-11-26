package prompts

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ecosystem-manager/api/pkg/tasks"
)

func TestGeneratePromptSectionsFiltersByTypeAndOperation(t *testing.T) {
	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "sections.yaml"), `
name: test
type: unified
target: ecosystem
description: test config
base_sections:
  - core/common
  - resource-specific/alpha
  - scenario-specific/beta
  - generator-specific/only-gen
  - improver-specific/only-imp
operations:
  scenario-generator:
    name: scenario-generator
    type: generator
    target: scenarios
    description: scenario generator config
    additional_sections:
      - operations/shared
`)

	assembler, err := NewAssembler(tmp, tmp)
	if err != nil {
		t.Fatalf("NewAssembler() error = %v", err)
	}

	sections, err := assembler.GeneratePromptSections(tasks.TaskItem{
		ID:        "task-1",
		Type:      "scenario",
		Operation: "generator",
	})
	if err != nil {
		t.Fatalf("GeneratePromptSections() error = %v", err)
	}

	expected := []string{
		"core/common",
		"scenario-specific/beta",
		"generator-specific/only-gen",
		"operations/shared",
	}

	if !reflect.DeepEqual(sections, expected) {
		t.Fatalf("GeneratePromptSections() got %v, want %v", sections, expected)
	}
}

func TestResolveIncludesTracksCriticalAndOptionalFiles(t *testing.T) {
	tmp := t.TempDir()
	baseContent := "content"

	writeFile(t, filepath.Join(tmp, "core", "important.md"), baseContent)
	writeFile(t, filepath.Join(tmp, "extras", "nested.md"), "nested")
	writeFile(t, filepath.Join(tmp, "extras", "optional.md"), "optional {{INCLUDE: nested}}")

	assembler := &Assembler{PromptsDir: tmp}
	input := "{{INCLUDE: core/important}}\n{{INCLUDE: extras/optional}}"
	var includes []string

	resolved, err := assembler.resolveIncludes(input, tmp, 0, &includes)
	if err != nil {
		t.Fatalf("resolveIncludes() error = %v", err)
	}

	if !strings.Contains(resolved, baseContent) || !strings.Contains(resolved, "nested") {
		t.Fatalf("resolveIncludes() did not resolve content: %s", resolved)
	}

	expectedIncludes := []string{
		"core/important.md",
		"extras/optional.md",
		"extras/nested.md",
	}

	if !reflect.DeepEqual(includes, expectedIncludes) {
		t.Fatalf("resolveIncludes() captured %v, want %v", includes, expectedIncludes)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
