//go:build testing
// +build testing

package server

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSaveAndLoadIssue tests issue storage and retrieval
func TestSaveAndLoadIssue(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	issue := createTestIssue("issue-storage-test", "Storage Test", "bug", "high", "storage-app")

	// Test save
	issueDir, err := env.Server.saveIssue(issue, "open")
	if err != nil {
		t.Fatalf("Failed to save issue: %v", err)
	}

	if issueDir == "" {
		t.Fatal("Expected issue directory path")
	}

	// Test load
	loaded, err := env.Server.loadIssueFromDir(issueDir)
	if err != nil {
		t.Fatalf("Failed to load issue: %v", err)
	}

	if loaded.ID != issue.ID {
		t.Errorf("Expected ID %s, got %s", issue.ID, loaded.ID)
	}
	if loaded.Title != issue.Title {
		t.Errorf("Expected title %s, got %s", issue.Title, loaded.Title)
	}
}

// TestFindIssueDirectory tests issue directory lookup
func TestFindIssueDirectory(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	issue := createTestIssue("issue-find-test", "Find Test", "bug", "medium", "find-app")
	env.Server.saveIssue(issue, "open")

	// Test find
	issueDir, folder, err := env.Server.findIssueDirectory("issue-find-test")
	if err != nil {
		t.Fatalf("Failed to find issue: %v", err)
	}

	if folder != "open" {
		t.Errorf("Expected folder 'open', got %s", folder)
	}

	if issueDir == "" {
		t.Error("Expected issue directory path")
	}

	// Test find non-existent
	_, _, err = env.Server.findIssueDirectory("non-existent-issue")
	if err == nil {
		t.Error("Expected error for non-existent issue")
	}
}

// TestMoveIssueBetweenFolders tests moving issues between status folders
func TestMoveIssueBetweenFolders(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	issue := createTestIssue("issue-move-test", "Move Test", "bug", "low", "move-app")
	env.Server.saveIssue(issue, "open")

	// Move to active
	err := env.Server.moveIssue("issue-move-test", "active")
	if err != nil {
		t.Fatalf("Failed to move issue: %v", err)
	}

	// Verify it's in active folder
	_, folder, err := env.Server.findIssueDirectory("issue-move-test")
	if err != nil {
		t.Fatalf("Failed to find moved issue: %v", err)
	}

	if folder != "active" {
		t.Errorf("Expected folder 'active', got %s", folder)
	}

	// Verify it's no longer in open folder
	openPath := filepath.Join(env.IssuesDir, "open", "issue-move-test")
	if _, err := os.Stat(openPath); !os.IsNotExist(err) {
		t.Error("Issue should not exist in open folder")
	}
}

// TestGetAllIssues tests retrieving all issues with filters
func TestGetAllIssues(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create test issues
	for i, status := range []string{"open", "active", "completed"} {
		issue := createTestIssue(
			fmt.Sprintf("issue-getall-%d", i),
			fmt.Sprintf("GetAll Test %d", i),
			"bug",
			"medium",
			"getall-app",
		)
		env.Server.saveIssue(issue, status)
	}

	// Get all issues
	allIssues, err := env.Server.getAllIssues("", "", "", "", 0)
	if err != nil {
		t.Fatalf("Failed to get all issues: %v", err)
	}

	if len(allIssues) < 3 {
		t.Errorf("Expected at least 3 issues, got %d", len(allIssues))
	}

	// Filter by status
	activeIssues, err := env.Server.getAllIssues("active", "", "", "", 0)
	if err != nil {
		t.Fatalf("Failed to get active issues: %v", err)
	}

	for _, issue := range activeIssues {
		if issue.Status != "active" {
			t.Errorf("Expected status 'active', got %s", issue.Status)
		}
	}

	// Filter by priority
	priorityIssues, err := env.Server.getAllIssues("", "medium", "", "", 0)
	if err != nil {
		t.Fatalf("Failed to get issues by priority: %v", err)
	}
	for _, issue := range priorityIssues {
		if issue.Priority != "medium" {
			t.Errorf("Expected priority 'medium', got %s", issue.Priority)
		}
	}

	// Filter by app_id
	appIssues, err := env.Server.getAllIssues("", "", "", "getall-app", 0)
	if err != nil {
		t.Fatalf("Failed to get app issues: %v", err)
	}

	for _, issue := range appIssues {
		if issue.AppID != "getall-app" {
			t.Errorf("Expected app_id 'getall-app', got %s", issue.AppID)
		}
	}
}

// TestDeleteIssueStorage tests issue deletion
func TestDeleteIssueStorage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	issue := createTestIssue("issue-delete-storage", "Delete Test", "bug", "low", "delete-app")
	issueDir, err := env.Server.saveIssue(issue, "open")
	if err != nil {
		t.Fatalf("Failed to save issue: %v", err)
	}

	// Verify issue exists
	if _, err := os.Stat(issueDir); os.IsNotExist(err) {
		t.Fatal("Issue directory should exist")
	}

	// Delete issue
	err = os.RemoveAll(issueDir)
	if err != nil {
		t.Fatalf("Failed to delete issue directory: %v", err)
	}

	// Verify deletion
	if _, err := os.Stat(issueDir); !os.IsNotExist(err) {
		t.Error("Issue directory should not exist after deletion")
	}
}

// TestIssueMetadataTimestamps tests timestamp handling
func TestIssueMetadataTimestamps(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	issue := createTestIssue("issue-timestamp", "Timestamp Test", "bug", "medium", "timestamp-app")

	// Save issue
	issueDir, err := env.Server.saveIssue(issue, "open")
	if err != nil {
		t.Fatalf("Failed to save issue: %v", err)
	}

	// Load and check timestamps
	loaded, err := env.Server.loadIssueFromDir(issueDir)
	if err != nil {
		t.Fatalf("Failed to load issue: %v", err)
	}

	if loaded.Metadata.CreatedAt == "" {
		t.Error("Expected created_at timestamp")
	}

	if loaded.Metadata.UpdatedAt == "" {
		t.Error("Expected updated_at timestamp")
	}

	// Verify timestamp format (RFC3339)
	_, err = time.Parse(time.RFC3339, loaded.Metadata.CreatedAt)
	if err != nil {
		t.Errorf("Invalid created_at timestamp format: %v", err)
	}
}

func TestStoreIssueArtifactsReplace(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	issue := createTestIssue("artifact-replace", "Artifact Replace", "bug", "high", "artifact-app")
	issueDir, err := env.Server.saveIssue(tissue, "open")
	if err != nil {
		t.Fatalf("Failed to save issue: %v", err)
	}

	payloads := []ArtifactPayload{
		{
			Name:        "logs.txt",
			Content:     "log line",
			Encoding:    "plain",
			ContentType: "text/plain",
			Category:    "logs",
		},
		{
			Name:        "details.json",
			Content:     "{\"ok\":true}",
			Encoding:    "plain",
			ContentType: "application/json",
			Category:    "metadata",
		},
	}

	if err := env.Server.storeIssueArtifacts(tissue, tissueDir, payloads, true); err != nil {
		t.Fatalf("storeIssueArtifacts failed: %v", err)
	}

	if len(tissue.Attachments) != len(payloads) {
		t.Fatalf("expected %d attachments, got %d", len(payloads), len(tissue.Attachments))
	}

	for idx, att := range tissue.Attachments {
		path := filepath.Join(tissueDir, filepath.FromSlash(att.Path))
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("failed to read artifact %s: %v", att.Path, readErr)
		}
		if string(data) != payloads[idx].Content {
			t.Errorf("artifact content mismatch for %s", att.Name)
		}
	}
}

func TestStoreIssueArtifactsAppend(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	issue := createTestIssue("artifact-append", "Artifact Append", "bug", "high", "artifact-app")
	issueDir, err := env.Server.saveIssue(tissue, "open")
	if err != nil {
		t.Fatalf("Failed to save issue: %v", err)
	}

	initialPayload := []ArtifactPayload{{
		Name:        "initial.txt",
		Content:     "initial",
		Encoding:    "plain",
		ContentType: "text/plain",
	}}
	if err := env.Server.storeIssueArtifacts(tissue, tissueDir, initialPayload, true); err != nil {
		t.Fatalf("initial artifact store failed: %v", err)
	}

	appendPayload := []ArtifactPayload{{
		Name:        "extra.txt",
		Content:     "extra",
		Encoding:    "plain",
		ContentType: "text/plain",
		Category:    "additional",
	}}
	if err := env.Server.storeIssueArtifacts(tissue, tissueDir, appendPayload, false); err != nil {
		t.Fatalf("append artifact store failed: %v", err)
	}

	if len(tissue.Attachments) != 2 {
		t.Fatalf("expected 2 attachments after append, got %d", len(tissue.Attachments))
	}

	last := tissue.Attachments[len(tissue.Attachments)-1]
	if last.Name != "extra.txt" {
		t.Errorf("expected appended attachment name 'extra.txt', got %s", last.Name)
	}

	path := filepath.Join(tissueDir, filepath.FromSlash(last.Path))
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("failed to read appended artifact: %v", readErr)
	}
	if string(data) != "extra" {
		t.Errorf("expected appended artifact content 'extra', got %s", string(data))
	}
}
