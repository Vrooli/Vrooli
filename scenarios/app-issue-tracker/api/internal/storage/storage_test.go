package storage

import (
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	issuespkg "app-issue-tracker-api/internal/issues"

	"gopkg.in/yaml.v3"
)

func writeIssueMetadata(t *testing.T, issueDir string, issue *issuespkg.Issue) {
	data, err := yaml.Marshal(issue)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if err := os.MkdirAll(issueDir, 0o755); err != nil {
		t.Fatalf("mkdir issue dir: %v", err)
	}
	metaPath := filepath.Join(issueDir, issuespkg.MetadataFilename)
	if err := os.WriteFile(metaPath, data, 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
}

func TestLoadIssueFromDirEnrichesArtifacts(t *testing.T) {
	root := t.TempDir()
	issueDir := filepath.Join(root, "open", "ISSUE-1")
	artifactsDir := filepath.Join(issueDir, issuespkg.ArtifactsDirName)
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		t.Fatalf("mkdir artifacts: %v", err)
	}

	original := &issuespkg.Issue{ID: "ISSUE-1"}
	writeIssueMetadata(t, issueDir, original)

	artifactPath := filepath.Join(artifactsDir, "investigation-report.md")
	if err := os.WriteFile(artifactPath, []byte("# report"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	store := NewFileIssueStore(root)
	loaded, err := store.LoadIssueFromDir(issueDir)
	if err != nil {
		t.Fatalf("LoadIssueFromDir error: %v", err)
	}

	if len(loaded.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(loaded.Attachments))
	}

	att := loaded.Attachments[0]
	if att.Path != "artifacts/investigation-report.md" {
		t.Fatalf("unexpected attachment path: %s", att.Path)
	}
	if !strings.HasPrefix(att.Type, "text/markdown") {
		t.Fatalf("expected markdown content type, got %s", att.Type)
	}
	if att.Category != "investigation_report" {
		t.Fatalf("expected investigation_report category, got %s", att.Category)
	}
	if att.Size == 0 {
		t.Fatalf("expected attachment size to be populated")
	}
}

func TestLoadIssueFromDirPropagatesWalkErrors(t *testing.T) {
	root := t.TempDir()
	issueDir := filepath.Join(root, "open", "ISSUE-2")
	artifactsDir := filepath.Join(issueDir, issuespkg.ArtifactsDirName)
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		t.Fatalf("mkdir artifacts: %v", err)
	}

	original := &issuespkg.Issue{ID: "ISSUE-2"}
	writeIssueMetadata(t, issueDir, original)

	baseOps := defaultFileOps()
	customOps := *baseOps
	customOps.WalkDir = func(root string, fn iofs.WalkDirFunc) error {
		return fmt.Errorf("walk failure")
	}

	store := NewFileIssueStoreWithOps(root, &customOps)
	_, err := store.LoadIssueFromDir(issueDir)
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "walk artifacts") {
		t.Fatalf("expected walk artifacts error, got %v", err)
	}
}
