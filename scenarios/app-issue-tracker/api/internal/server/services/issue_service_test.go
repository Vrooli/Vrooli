package services

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	issuespkg "app-issue-tracker-api/internal/issues"
	"app-issue-tracker-api/internal/server/metadata"
	"app-issue-tracker-api/internal/storage"
)

func TestTransitionIssueStatusRollback(t *testing.T) {
	root := t.TempDir()
	store := storage.NewFileIssueStore(root)

	now := time.Now().UTC().Format(time.RFC3339)
	issue := &issuespkg.Issue{
		ID:    "ISSUE-1",
		Title: "Sample",
		Metadata: struct {
			CreatedAt  string            `yaml:"created_at" json:"created_at"`
			UpdatedAt  string            `yaml:"updated_at" json:"updated_at"`
			ResolvedAt string            `yaml:"resolved_at,omitempty" json:"resolved_at,omitempty"`
			Tags       []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
			Labels     map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
			Watchers   []string          `yaml:"watchers,omitempty" json:"watchers,omitempty"`
			Extra      map[string]string `yaml:"extra,omitempty" json:"extra,omitempty"`
		}{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	if _, err := store.SaveIssue(issue, "open"); err != nil {
		t.Fatalf("failed to seed issue: %v", err)
	}

	service := NewIssueService(store, NewArtifactManager(), nil, func() time.Time { return time.Now() })

	issueDir := filepath.Join(root, "open", "ISSUE-1")
	targetDir := filepath.Join(root, "completed", "ISSUE-1")

	dir, folder, change, rollback, err := service.transitionIssueStatus("ISSUE-1", issue, issueDir, "open", "completed")
	if err != nil {
		t.Fatalf("transition failed: %v", err)
	}
	if dir != targetDir {
		t.Fatalf("expected target directory %s, got %s", targetDir, dir)
	}
	if folder != "completed" {
		t.Fatalf("expected folder completed, got %s", folder)
	}
	if change == nil || change.To != "completed" || change.From != "open" {
		t.Fatalf("unexpected status change %+v", change)
	}

	if _, statErr := os.Stat(targetDir); statErr != nil {
		t.Fatalf("expected target directory to exist after transition: %v", statErr)
	}

	if err := rollback(); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	if _, statErr := os.Stat(issueDir); statErr != nil {
		t.Fatalf("expected original directory after rollback: %v", statErr)
	}
	if _, statErr := os.Stat(targetDir); !os.IsNotExist(statErr) {
		t.Fatalf("expected target directory removed after rollback")
	}

	// second rollback should be a no-op
	if err := rollback(); err != nil {
		t.Fatalf("second rollback should be no-op: %v", err)
	}
}

func TestIssueServiceCreateIssueStoresArtifacts(t *testing.T) {
	root := t.TempDir()
	store := storage.NewFileIssueStore(root)
	fixed := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	svc := NewIssueService(store, NewArtifactManager(), nil, func() time.Time { return fixed })
	req := &issuespkg.CreateIssueRequest{
		Title:       "Sample Issue",
		Description: "Example description",
		AppID:       "demo-app",
		Artifacts: []issuespkg.ArtifactPayload{{
			Name:        "diagnostics.txt",
			Category:    "logs",
			Content:     "log content",
			Encoding:    "plain",
			ContentType: "text/plain",
		}},
	}

	issue, storagePath, err := svc.CreateIssue(req)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	if storagePath == "" {
		t.Fatalf("expected storage path to be returned")
	}
	if len(issue.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(issue.Attachments))
	}

	artifactPath := filepath.Join(root, storagePath, issue.Attachments[0].Path)
	if _, err := os.Stat(artifactPath); err != nil {
		t.Fatalf("expected artifact at %s: %v", artifactPath, err)
	}
	if issue.Metadata.CreatedAt == "" || issue.Metadata.UpdatedAt == "" {
		t.Fatalf("expected metadata timestamps to be set")
	}
}

func TestIssueServiceUpdateIssueTransitionsStatus(t *testing.T) {
	root := t.TempDir()
	store := storage.NewFileIssueStore(root)
	fixed := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	svc := NewIssueService(store, NewArtifactManager(), nil, func() time.Time { return fixed })
	req := &issuespkg.CreateIssueRequest{
		Title:       "Needs Update",
		Description: "Initial",
	}

	issue, _, err := svc.CreateIssue(req)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	updateStatus := "completed"
	updateReq := &issuespkg.UpdateIssueRequest{Status: &updateStatus}
	issue, change, err := svc.UpdateIssue(issue.ID, updateReq)
	if err != nil {
		t.Fatalf("UpdateIssue failed: %v", err)
	}
	if change == nil || change.From != "open" || change.To != "completed" {
		t.Fatalf("unexpected status change: %+v", change)
	}
	if issue.Metadata.ResolvedAt == "" {
		t.Fatalf("expected resolved timestamp to be set on completion")
	}

	artifactReq := &issuespkg.UpdateIssueRequest{
		Artifacts: []issuespkg.ArtifactPayload{{
			Name:        "report.md",
			Category:    "report",
			Content:     "# report",
			Encoding:    "plain",
			ContentType: "text/markdown",
		}},
	}

	issue, _, err = svc.UpdateIssue(issue.ID, artifactReq)
	if err != nil {
		t.Fatalf("UpdateIssue failed when appending artifact: %v", err)
	}
	if len(issue.Attachments) == 0 {
		t.Fatalf("expected attachments to include new artifact")
	}
	artifactPath := filepath.Join(root, "completed", issue.ID, issue.Attachments[len(issue.Attachments)-1].Path)
	if _, err := os.Stat(artifactPath); err != nil {
		t.Fatalf("expected artifact persisted at %s: %v", artifactPath, err)
	}
}

func TestResetInvestigationForReopenClearsResolvedTimestamp(t *testing.T) {
	now := time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC).Format(time.RFC3339)
	tissue := &issuespkg.Issue{}
	tissue.ID = "issue-reset"
	tissue.Metadata.ResolvedAt = now
	tissue.Metadata.Extra = map[string]string{
		metadata.AgentLastErrorKey: "boom",
		"custom":                   "value",
	}
	tissue.Investigation.AgentID = "agent-1"
	tissue.Fix.Applied = true

	ResetInvestigationForReopen(tissue)

	if tissue.Metadata.ResolvedAt != "" {
		t.Fatalf("expected resolved timestamp to be cleared, got %q", tissue.Metadata.ResolvedAt)
	}
	if tissue.Investigation.AgentID != "" {
		t.Fatalf("expected investigation data reset")
	}
	if tissue.Fix.Applied {
		t.Fatalf("expected fix state reset")
	}
	if _, ok := tissue.Metadata.Extra[metadata.AgentLastErrorKey]; ok {
		t.Fatalf("expected agent error metadata removed")
	}
	if tissue.Metadata.Extra["custom"] != "value" {
		t.Fatalf("expected custom metadata preserved, got %v", tissue.Metadata.Extra)
	}
}

func TestIssueServiceDeleteIssueRemovesDirectory(t *testing.T) {
	root := t.TempDir()
	store := storage.NewFileIssueStore(root)
	svc := NewIssueService(store, NewArtifactManager(), nil, time.Now)
	req := &issuespkg.CreateIssueRequest{Title: "Delete me", Description: "to be removed"}

	issue, storagePath, err := svc.CreateIssue(req)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	issueDir := filepath.Join(root, storagePath)
	if _, err := os.Stat(issueDir); err != nil {
		t.Fatalf("expected issue directory to exist: %v", err)
	}

	if err := svc.DeleteIssue(issue.ID); err != nil {
		t.Fatalf("DeleteIssue failed: %v", err)
	}
	if _, err := os.Stat(issueDir); !os.IsNotExist(err) {
		t.Fatalf("expected issue directory to be removed")
	}
}

type stubProcessor struct {
	running map[string]bool
}

func (p *stubProcessor) IsRunning(issueID string) bool {
	return p.running[issueID]
}

func TestIssueServiceDeleteIssueBlocksRunningProcess(t *testing.T) {
	root := t.TempDir()
	store := storage.NewFileIssueStore(root)
	processor := &stubProcessor{running: map[string]bool{"issue-running": true}}
	svc := NewIssueService(store, NewArtifactManager(), processor, time.Now)

	req := &issuespkg.CreateIssueRequest{Title: "Running", Description: "Agent active"}
	issue, storagePath, err := svc.CreateIssue(req)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	issueDir := filepath.Join(root, storagePath)
	if _, err := os.Stat(issueDir); err != nil {
		t.Fatalf("expected issue directory to exist: %v", err)
	}

	processor.running[issue.ID] = true

	if err := svc.DeleteIssue(issue.ID); !errors.Is(err, ErrIssueRunning) {
		t.Fatalf("expected ErrIssueRunning, got %v", err)
	}

	if _, err := os.Stat(issueDir); err != nil {
		t.Fatalf("issue directory should remain when deletion is blocked: %v", err)
	}
}

func TestIssueServiceListIssuesRejectsInvalidStatus(t *testing.T) {
	root := t.TempDir()
	store := storage.NewFileIssueStore(root)
	svc := NewIssueService(store, NewArtifactManager(), nil, time.Now)

	if _, err := svc.ListIssues("not-real", "", "", "", 10); !errors.Is(err, ErrInvalidStatusFilter) {
		t.Fatalf("expected ErrInvalidStatusFilter, got %v", err)
	}
}

func TestIssueServiceListIssuesFiltersCaseInsensitive(t *testing.T) {
	root := t.TempDir()
	store := storage.NewFileIssueStore(root)
	svc := NewIssueService(store, NewArtifactManager(), nil, time.Now)

	now := time.Now().UTC().Format(time.RFC3339)
	issue := &issuespkg.Issue{
		ID:       "ISSUE-CASE",
		Title:    "Case sensitivity",
		Priority: "High",
		Type:     "Bug",
		AppID:    "Example-App",
		Status:   "open",
		Metadata: struct {
			CreatedAt  string            `yaml:"created_at" json:"created_at"`
			UpdatedAt  string            `yaml:"updated_at" json:"updated_at"`
			ResolvedAt string            `yaml:"resolved_at,omitempty" json:"resolved_at,omitempty"`
			Tags       []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
			Labels     map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
			Watchers   []string          `yaml:"watchers,omitempty" json:"watchers,omitempty"`
			Extra      map[string]string `yaml:"extra,omitempty" json:"extra,omitempty"`
		}{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	if _, err := store.SaveIssue(issue, "open"); err != nil {
		t.Fatalf("failed to seed issue: %v", err)
	}

	results, err := svc.ListIssues("", "HIGH", "BUG", "EXAMPLE-app", 10)
	if err != nil {
		t.Fatalf("ListIssues failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 filtered result, got %d", len(results))
	}
}
