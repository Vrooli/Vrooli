package backlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromptLoader_Build(t *testing.T) {
	loader := NewPromptLoader("test-root")

	item := BacklogItem{
		Name:           "test-item",
		Title:          "Test Item Title",
		Description:    "A test description",
		Status:         StatusBacklog,
		Priority:       5,
		Tags:           []string{"tag1", "tag2"},
		Kind:           KindIdea,
		ResearchTarget: "some-target",
	}

	template := `Item: {{ITEM_NAME}}
Title: {{ITEM_TITLE}}
Description: {{ITEM_DESCRIPTION}}
Kind: {{ITEM_KIND}}
Status: {{ITEM_STATUS}}
Priority: {{ITEM_PRIORITY}}
Tags: {{ITEM_TAGS}}
Folder: {{ITEM_FOLDER}}
Research: {{RESEARCH_TARGET}}`

	result := loader.Build(template, item, "/path/to/folder")

	expected := `Item: test-item
Title: Test Item Title
Description: A test description
Kind: idea
Status: backlog
Priority: 5
Tags: tag1, tag2
Folder: /path/to/folder
Research: some-target`

	if result != expected {
		t.Errorf("Build() result mismatch.\nGot:\n%s\n\nExpected:\n%s", result, expected)
	}
}

func TestPromptLoader_Build_EmptyFields(t *testing.T) {
	loader := NewPromptLoader("test-root")

	item := BacklogItem{
		Name:        "empty-item",
		Title:       "Empty Item",
		Description: "",
		Status:      StatusBacklog,
		Priority:    0,
		Tags:        nil,
		Kind:        KindFix,
	}

	template := "Title: {{ITEM_TITLE}}, Tags: {{ITEM_TAGS}}, Research: {{RESEARCH_TARGET}}"
	result := loader.Build(template, item, "/folder")

	if !strings.Contains(result, "Title: Empty Item") {
		t.Errorf("Expected title to be substituted, got: %s", result)
	}
	if !strings.Contains(result, "Tags: ") {
		t.Errorf("Expected empty tags, got: %s", result)
	}
	if !strings.Contains(result, "Research: ") {
		t.Errorf("Expected empty research target, got: %s", result)
	}
}

func TestResearchPromptName(t *testing.T) {
	tests := []struct {
		mode     ResearchMode
		kind     BacklogKind
		expected string
	}{
		{ResearchModeClarify, KindIdea, "clarify"},
		{ResearchModeSuggest, KindIdea, "suggest"},
		{ResearchModeEnhance, KindIdea, "enhance"},
		{ResearchModeResearch, KindIdea, "deep-research-idea"},
		{ResearchModeResearch, KindFix, "deep-research-fix"},
		{ResearchModeResearch, KindExecute, "deep-research-other"},
		{ResearchModeResearch, KindResearch, "deep-research-other"},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode)+"_"+string(tt.kind), func(t *testing.T) {
			result := ResearchPromptName(tt.mode, tt.kind)
			if result != tt.expected {
				t.Errorf("ResearchPromptName(%s, %s) = %s, want %s",
					tt.mode, tt.kind, result, tt.expected)
			}
		})
	}
}

func TestProcessingPromptName(t *testing.T) {
	tests := []struct {
		kind     BacklogKind
		expected string
	}{
		{KindIdea, "process-idea"},
		{KindFix, "process-fix"},
		{KindExecute, "process-execute"},
		{KindResearch, "process-execute"},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			result := ProcessingPromptName(tt.kind)
			if result != tt.expected {
				t.Errorf("ProcessingPromptName(%s) = %s, want %s",
					tt.kind, result, tt.expected)
			}
		})
	}
}

func TestPromptLoader_LoadFromFile(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	promptDir := filepath.Join(tmpDir, "prompts", "workflow")
	if err := os.MkdirAll(promptDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a test prompt file
	testContent := "# Test Prompt\n\nThis is {{ITEM_TITLE}} content."
	testFile := filepath.Join(promptDir, "test-prompt.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	loader := NewPromptLoader(tmpDir)
	content, err := loader.Load(PromptCategoryWorkflow, "test-prompt")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if content != testContent {
		t.Errorf("Load() content mismatch.\nGot: %s\nExpected: %s", content, testContent)
	}
}

func TestPromptLoader_LoadFallback(t *testing.T) {
	// Use a non-existent directory to force fallback
	loader := NewPromptLoader("/non-existent-path")

	// Test that fallback works for known prompts
	content, err := loader.Load(PromptCategoryWorkflow, "clarify")
	if err != nil {
		t.Fatalf("Load() with fallback error = %v", err)
	}

	if !strings.Contains(content, "clarify") && !strings.Contains(content, "Clarify") {
		t.Errorf("Fallback content should contain 'clarify', got: %s", content)
	}
}

func TestPromptLoader_LoadNotFound(t *testing.T) {
	loader := NewPromptLoader("/non-existent-path")

	_, err := loader.Load(PromptCategoryWorkflow, "non-existent-prompt")
	if err == nil {
		t.Error("Load() expected error for non-existent prompt, got nil")
	}
}

func TestPromptLoader_Caching(t *testing.T) {
	tmpDir := t.TempDir()
	promptDir := filepath.Join(tmpDir, "prompts", "workflow")
	if err := os.MkdirAll(promptDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	testFile := filepath.Join(promptDir, "cached-prompt.md")
	if err := os.WriteFile(testFile, []byte("Version 1"), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	loader := NewPromptLoader(tmpDir)

	// First load
	content1, err := loader.Load(PromptCategoryWorkflow, "cached-prompt")
	if err != nil {
		t.Fatalf("First Load() error = %v", err)
	}

	// Second load should return cached content
	content2, err := loader.Load(PromptCategoryWorkflow, "cached-prompt")
	if err != nil {
		t.Fatalf("Second Load() error = %v", err)
	}

	if content1 != content2 {
		t.Error("Cached content should match first load")
	}
}

func TestPromptLoader_AllFallbacksExist(t *testing.T) {
	// Verify all expected fallbacks exist
	expectedFallbacks := []struct {
		category PromptCategory
		name     string
	}{
		{PromptCategoryWorkflow, "clarify"},
		{PromptCategoryWorkflow, "suggest"},
		{PromptCategoryWorkflow, "enhance"},
		{PromptCategoryResearch, "deep-research-idea"},
		{PromptCategoryResearch, "deep-research-fix"},
		{PromptCategoryResearch, "deep-research-other"},
		{PromptCategoryProcessing, "process-idea"},
		{PromptCategoryProcessing, "process-fix"},
		{PromptCategoryProcessing, "process-execute"},
	}

	loader := NewPromptLoader("/non-existent")

	for _, fb := range expectedFallbacks {
		t.Run(string(fb.category)+"/"+fb.name, func(t *testing.T) {
			content, err := loader.Load(fb.category, fb.name)
			if err != nil {
				t.Errorf("Fallback should exist for %s/%s: %v", fb.category, fb.name, err)
			}
			if content == "" {
				t.Errorf("Fallback content should not be empty for %s/%s", fb.category, fb.name)
			}
		})
	}
}
