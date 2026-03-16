package main

import (
	"testing"
)

func TestNormalizeApprovedChanges_WithRunID(t *testing.T) {
	preview := &workspaceSandboxCommitPreview{
		Files: []workspaceSandboxCommitPreviewFile{
			{
				RelativePath:      "src/api.go",
				Status:            "pending",
				SandboxID:         "sandbox-1",
				SandboxOwner:      "agent-a",
				ChangeType:        "modified",
				AgentManagerRunID: "run-123",
			},
			{
				RelativePath:      "src/handler.go",
				Status:            "already_committed",
				SandboxID:         "sandbox-2",
				SandboxOwner:      "agent-b",
				ChangeType:        "added",
				AgentManagerRunID: "",
			},
		},
		CommittableFiles: 1,
		SuggestedMessage: "test message",
	}

	result := normalizeApprovedChanges(preview)

	if !result.Available {
		t.Fatal("expected Available=true")
	}
	if result.CommittableFiles != 1 {
		t.Errorf("expected CommittableFiles=1, got %d", result.CommittableFiles)
	}
	if len(result.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(result.Files))
	}

	// First file should have run ID
	if result.Files[0].AgentManagerRunID != "run-123" {
		t.Errorf("expected AgentManagerRunID='run-123', got %q", result.Files[0].AgentManagerRunID)
	}
	if result.Files[0].RelativePath != "src/api.go" {
		t.Errorf("expected RelativePath='src/api.go', got %q", result.Files[0].RelativePath)
	}

	// Second file should have empty run ID
	if result.Files[1].AgentManagerRunID != "" {
		t.Errorf("expected empty AgentManagerRunID, got %q", result.Files[1].AgentManagerRunID)
	}
}

func TestNormalizeApprovedChanges_NilPreview(t *testing.T) {
	result := normalizeApprovedChanges(nil)
	if result.Available {
		t.Error("expected Available=false for nil preview")
	}
}

func TestNormalizeApprovedChanges_EmptyFiles(t *testing.T) {
	preview := &workspaceSandboxCommitPreview{
		Files:            []workspaceSandboxCommitPreviewFile{},
		CommittableFiles: 0,
	}

	result := normalizeApprovedChanges(preview)

	if !result.Available {
		t.Error("expected Available=true for empty but valid preview")
	}
	if len(result.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(result.Files))
	}
}
