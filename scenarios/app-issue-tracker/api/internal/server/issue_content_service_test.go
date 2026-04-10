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

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	seedIssue(t, store, issue, "open")
	issueDir := store.IssueDir("open", "ISSUE-2")

	issue.Metadata.Extra[metadata.AgentRunIDKey] = "run-2"
	issue.Metadata.Extra[metadata.AgentRunnerTypeKey] = "claude-code"

	if err := store.WriteIssueMetadata(issueDir, issue); err != nil {
		t.Fatalf("rewrite metadata: %v", err)
	}

	agentMock := newTestAgentManager()
	agentMock.run = &domainpb.Run{
		Id:        "run-2",
		Status:    domainpb.RunStatus_RUN_STATUS_COMPLETE,
		SessionId: "session-2",
		EndedAt:   timestamppb.New(time.Now().UTC()),
		Summary: &domainpb.RunSummary{
			Description: "Done",
		},
	}
	agentMock.events = []*domainpb.RunEvent{
		{
			Id:        "evt-2",
			EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE,
			Data: &domainpb.RunEvent_Message{
				Message: &domainpb.MessageEventData{
					Role:    "assistant",
					Content: "Done",
				},
			},
		},
	}

	srv := &Server{
		config:       &Config{ScenarioRoot: root, VrooliRoot: root, IssuesDir: issuesDir, WebsocketAllowedOrigins: []string{"*"}},
		store:        store,
		agentManager: agentMock,
	}
	svc := NewIssueContentService(srv)
	payload, err := svc.AgentConversation("ISSUE-2", 10)
	if err != nil {
		t.Fatalf("AgentConversation failed: %v", err)
	}

	if !payload.Available {
		t.Fatalf("expected conversation to be available")
	}
	if len(payload.Entries) == 0 {
		t.Fatalf("expected transcript entries to be parsed")
	}
	if payload.TranscriptTimestamp == "" {
		t.Fatalf("expected transcript timestamp to be set")
	}
}
