package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	issuespkg "app-issue-tracker-api/internal/issues"
	"app-issue-tracker-api/internal/server/metadata"
	"app-issue-tracker-api/internal/storage"
)

func seedIssue(t *testing.T, store storage.IssueStorage, issue *issuespkg.Issue, folder string) string {
	t.Helper()
	if issue.Metadata.Extra == nil {
		issue.Metadata.Extra = map[string]string{}
	}
	issue.Metadata.Tags = append(issue.Metadata.Tags, "test")
	path, err := store.SaveIssue(issue, folder)
	if err != nil {
		t.Fatalf("save issue: %v", err)
	}
	return path
}

func TestIssueContentServiceResolveAttachment(t *testing.T) {
	root := t.TempDir()
	issuesDir := filepath.Join(root, "issues")
	store := storage.NewFileIssueStore(issuesDir)

	issue := &issuespkg.Issue{
		ID: "ISSUE-1",
		Attachments: []issuespkg.Attachment{
			{Name: "Log", Path: "artifacts/log.txt"},
		},
	}
	issueDir := seedIssue(t, store, issue, "open")

	artifactPath := filepath.Join(issueDir, issuespkg.ArtifactsDirName)
	if err := os.MkdirAll(artifactPath, 0o755); err != nil {
		t.Fatalf("mkdir artifacts: %v", err)
	}
	fullAttachment := filepath.Join(artifactPath, "log.txt")
	if err := os.WriteFile(fullAttachment, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write attachment: %v", err)
	}

	srv := &Server{
		config: &Config{ScenarioRoot: root, VrooliRoot: root, IssuesDir: issuesDir, WebsocketAllowedOrigins: []string{"*"}},
		store:  store,
	}
	svc := NewIssueContentService(srv)
	resource, err := svc.ResolveAttachment("ISSUE-1", "artifacts/log.txt")
	if err != nil {
		t.Fatalf("ResolveAttachment failed: %v", err)
	}
	defer resource.File.Close()

	if !strings.HasPrefix(resource.ContentType, "text/plain") {
		t.Fatalf("expected text/plain content type, got %s", resource.ContentType)
	}
	if resource.Meta == nil || resource.Meta.Name != "Log" {
		t.Fatalf("expected attachment metadata to be populated")
	}
}

func TestIssueContentServiceAgentConversation(t *testing.T) {
	root := t.TempDir()
	issuesDir := filepath.Join(root, "issues")
	store := storage.NewFileIssueStore(issuesDir)

	issue := &issuespkg.Issue{ID: "ISSUE-2"}
	issue.Investigation.AgentID = "agent-42"
	seedIssue(t, store, issue, "open")
	issueDir := store.IssueDir("open", "ISSUE-2")

	transcriptRel := filepath.Join("tmp", "agents", "transcript.log")
	lastRel := filepath.Join("tmp", "agents", "last.txt")

	issue.Metadata.Extra[metadata.AgentTranscriptPathKey] = transcriptRel
	issue.Metadata.Extra[metadata.AgentLastMessagePathKey] = lastRel

	if err := store.WriteIssueMetadata(issueDir, issue); err != nil {
		t.Fatalf("rewrite metadata: %v", err)
	}

	transcriptPath := filepath.Join(root, transcriptRel)
	if err := os.MkdirAll(filepath.Dir(transcriptPath), 0o755); err != nil {
		t.Fatalf("mkdir transcript dir: %v", err)
	}
	entries := []string{
		`{"sandbox":{"cwd":"/workspace"}}`,
		`{"prompt":"Investigate"}`,
		`{"id":"1","timestamp":"2024-01-01T00:00:00Z","msg":{"type":"agent_message","role":"assistant","message":"Done"}}`,
	}
	if err := os.WriteFile(transcriptPath, []byte(strings.Join(entries, "\n")), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	if err := os.Chtimes(transcriptPath, time.Now(), time.Now()); err != nil {
		t.Fatalf("chtimes transcript: %v", err)
	}

	lastMessagePath := filepath.Join(root, lastRel)
	if err := os.WriteFile(lastMessagePath, []byte("Latest message"), 0o644); err != nil {
		t.Fatalf("write last message: %v", err)
	}

	srv := &Server{
		config: &Config{ScenarioRoot: root, VrooliRoot: root, IssuesDir: issuesDir, WebsocketAllowedOrigins: []string{"*"}},
		store:  store,
	}
	svc := NewIssueContentService(srv)
	payload, err := svc.AgentConversation("ISSUE-2", 10)
	if err != nil {
		t.Fatalf("AgentConversation failed: %v", err)
	}

	if !payload.Available {
		t.Fatalf("expected conversation to be available")
	}
	if payload.Provider != "agent-42" {
		t.Fatalf("expected provider agent-42, got %s", payload.Provider)
	}
	if payload.LastMessage != "Latest message" {
		t.Fatalf("unexpected last message: %s", payload.LastMessage)
	}
	if len(payload.Entries) == 0 {
		t.Fatalf("expected transcript entries to be parsed")
	}
	if payload.TranscriptTimestamp == "" {
		t.Fatalf("expected transcript timestamp to be set")
	}
}
