package orchestration

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/identity"

	"agent-manager/internal/domain"
)

func principalContext(kind identity.ActorKind, subject string, verified bool) context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{Kind: kind, Subject: subject, Verified: verified})
}

func TestNewUserMessageRecordsTheVerifiedSender(t *testing.T) {
	attachment := []domain.MessageAttachmentInfo{{ID: "a1", FileName: "shot.png"}}
	cases := []struct {
		name         string
		ctx          context.Context
		attachments  []domain.MessageAttachmentInfo
		sender, id   string
		wantAttached int
	}{
		{"a person", principalContext(identity.ActorHuman, "operator", true), nil, domain.MessageSenderPerson, "operator", 0},
		{"a person with an attachment", principalContext(identity.ActorHuman, "operator", true), attachment, domain.MessageSenderPerson, "operator", 1},
		{"an agent", principalContext(identity.ActorAgent, "run-7", true), nil, domain.MessageSenderAgent, "run-7", 0},
		{"an agent with an attachment", principalContext(identity.ActorAgent, "run-7", true), attachment, domain.MessageSenderAgent, "run-7", 1},
		{"a service is neither", principalContext(identity.ActorService, "scheduler", true), nil, "", "", 0},
		{"an unverified principal is nobody", principalContext(identity.ActorHuman, "operator", false), nil, "", "", 0},
		{"no principal (a wake or recovery)", context.Background(), nil, "", "", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			event := newUserMessage(tc.ctx, uuid.New(), "please continue", tc.attachments)
			data, ok := event.Data.(*domain.MessageEventData)
			if !ok || data.Role != "user" || data.Content != "please continue" {
				t.Fatalf("event = %+v", event.Data)
			}
			if data.Sender != tc.sender || data.SenderID != tc.id {
				t.Fatalf("sender = %q/%q, want %q/%q", data.Sender, data.SenderID, tc.sender, tc.id)
			}
			if len(data.Attachments) != tc.wantAttached {
				t.Fatalf("attachments = %d, want %d", len(data.Attachments), tc.wantAttached)
			}
		})
	}
}

// Every place Agent Manager accepts a user message goes through newUserMessage:
// the interactive continuation, the initial message of a created run, and the
// plain and attachment-carrying continuations. A site that built the event
// directly would silently record no sender, so the sites are pinned here.
func TestEveryUserMessageIsBuiltWithTheSender(t *testing.T) {
	direct := regexp.MustCompile(`domain\.NewMessageEvent(WithAttachments)?\(\s*[\w.]+,\s*"user"`)
	via := regexp.MustCompile(`newUserMessage\(ctx,`)
	want := map[string]int{
		"interactive_run_control.go": 1,
		"run_creation.go":            1,
		"run_continuation.go":        2,
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]int{}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") || file == "message_sender.go" {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if direct.Match(body) {
			t.Fatalf("%s builds a user message without the sender; use newUserMessage", file)
		}
		if count := len(via.FindAll(body, -1)); count > 0 {
			found[file] = count
		}
	}
	for file, count := range want {
		if found[file] != count {
			t.Fatalf("%s builds %d user message(s) with newUserMessage, want %d", file, found[file], count)
		}
	}
	for file := range found {
		if _, ok := want[file]; !ok {
			t.Fatalf("%s builds a user message the test does not know; add it to the sites", file)
		}
	}
}
