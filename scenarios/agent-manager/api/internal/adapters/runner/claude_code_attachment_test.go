package runner

import (
	"testing"
)

func TestBuildPromptWithAttachments_NoAttachments(t *testing.T) {
	prompt := "Fix the bug in main.go"
	result := buildPromptWithAttachments(prompt, nil)
	if result != prompt {
		t.Errorf("expected original prompt %q, got %q", prompt, result)
	}

	// Also test with empty slice
	result = buildPromptWithAttachments(prompt, []Attachment{})
	if result != prompt {
		t.Errorf("expected original prompt %q with empty slice, got %q", prompt, result)
	}
}

func TestBuildPromptWithAttachments_SingleAttachment(t *testing.T) {
	prompt := "Describe this screenshot"
	attachments := []Attachment{
		{ID: "att-1", FileName: "screen.png", FilePath: "/tmp/uploads/screen.png"},
	}

	result := buildPromptWithAttachments(prompt, attachments)
	expected := "/tmp/uploads/screen.png\n\nDescribe this screenshot"
	if result != expected {
		t.Errorf("unexpected result:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestBuildPromptWithAttachments_MultipleAttachments(t *testing.T) {
	prompt := "Compare these images"
	attachments := []Attachment{
		{ID: "att-1", FileName: "before.png", FilePath: "/tmp/uploads/before.png"},
		{ID: "att-2", FileName: "after.png", FilePath: "/tmp/uploads/after.png"},
		{ID: "att-3", FileName: "diff.jpg", FilePath: "/data/diff.jpg"},
	}

	result := buildPromptWithAttachments(prompt, attachments)
	expected := "/tmp/uploads/before.png\n/tmp/uploads/after.png\n/data/diff.jpg\n\nCompare these images"
	if result != expected {
		t.Errorf("unexpected result:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestBuildPromptWithAttachments_EmptyPrompt(t *testing.T) {
	attachments := []Attachment{
		{ID: "att-1", FileName: "image.png", FilePath: "/tmp/image.png"},
	}

	result := buildPromptWithAttachments("", attachments)
	expected := "/tmp/image.png\n\n"
	if result != expected {
		t.Errorf("unexpected result:\ngot:  %q\nwant: %q", result, expected)
	}
}
