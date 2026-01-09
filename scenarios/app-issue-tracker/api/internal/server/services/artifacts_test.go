package services

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	issuespkg "app-issue-tracker-api/internal/issues"
)

func TestPersistArtifactsDecodesBase64(t *testing.T) {
	tempDir := t.TempDir()
	am := NewArtifactManager()

	payload := issuespkg.ArtifactPayload{
		Name:        "Diagnostic Logs",
		Category:    "logs",
		Content:     base64.StdEncoding.EncodeToString([]byte("hello world")),
		Encoding:    "base64",
		ContentType: "text/plain",
	}

	attachments, err := am.persistArtifacts(tempDir, []issuespkg.ArtifactPayload{payload})
	if err != nil {
		t.Fatalf("persistArtifacts returned error: %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}

	artifactPath := filepath.Join(tempDir, attachments[0].Path)
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("unable to read artifact: %v", err)
	}
	if string(data) != "hello world" {
		t.Fatalf("expected decoded content 'hello world', got %q", string(data))
	}
}

func TestPersistArtifactsGeneratesUniqueFilenames(t *testing.T) {
	tempDir := t.TempDir()
	am := NewArtifactManager()

	payloads := []issuespkg.ArtifactPayload{
		{
			Name:        "duplicate.txt",
			Category:    "logs",
			Content:     "first",
			Encoding:    "plain",
			ContentType: "text/plain",
		},
		{
			Name:        "duplicate.txt",
			Category:    "logs",
			Content:     "second",
			Encoding:    "plain",
			ContentType: "text/plain",
		},
	}

	attachments, err := am.persistArtifacts(tempDir, payloads)
	if err != nil {
		t.Fatalf("persistArtifacts returned error: %v", err)
	}
	if len(attachments) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(attachments))
	}
	if attachments[0].Path == attachments[1].Path {
		t.Fatalf("expected unique paths, both were %q", attachments[0].Path)
	}

	for _, attachment := range attachments {
		if _, err := os.Stat(filepath.Join(tempDir, attachment.Path)); err != nil {
			t.Fatalf("expected artifact on disk for %q: %v", attachment.Path, err)
		}
	}
}

func TestSanitizeFileComponent(t *testing.T) {
	cases := map[string]string{
		"  ..//dangerous name  ": "-dangerous-name",
		"Prompt##Report":         "prompt-report",
		"UPPER.txt":              "upper.txt",
	}

	for input, want := range cases {
		if got := sanitizeFileComponent(input); got != want {
			t.Fatalf("sanitizeFileComponent(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestResolveAttachmentPathRejectsTraversal(t *testing.T) {
	issueDir := t.TempDir()
	if _, err := resolveAttachmentPath(issueDir, "../"); err == nil {
		t.Fatal("expected resolveAttachmentPath to reject traversal path")
	}
}

func TestResolveAttachmentPathAllowsNestedArtifacts(t *testing.T) {
	issueDir := t.TempDir()
	full, err := resolveAttachmentPath(issueDir, "artifacts/logs/output.txt")
	if err != nil {
		t.Fatalf("expected nested path to resolve, got error: %v", err)
	}

	if !strings.HasSuffix(full, filepath.Join("artifacts", "logs", "output.txt")) {
		t.Fatalf("resolved path %q did not end with artifacts/logs/output.txt", full)
	}
}
