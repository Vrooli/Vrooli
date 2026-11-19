package conversations

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	"google.golang.org/protobuf/types/known/structpb"
)

func richAsk() *v1.Ask {
	return &v1.Ask{
		Id: "ask-1", State: "answered", Question: "Ship it?", Answer: "approve", SubjectKey: "release", RevisionCount: 2,
		Deadline: "2026-11-14T00:00:00Z", Recommended: "approve",
		Options: []*v1.AskOption{{Key: "approve", Label: "Approve"}, {Key: "reject", Label: "Reject"}},
		Attachments: []*v1.AskAttachment{
			{Name: "why.html", Kind: v1.AskAttachment_KIND_FILE, Role: v1.AskAttachment_ROLE_EXPLAINER, MediaType: "text/html", SizeBytes: 2048, PreviewKind: "html", SourceStatus: "fresh", SourcePath: "/work/why.html"},
			{
				Name: "shots", Label: "Screens", Kind: v1.AskAttachment_KIND_FOLDER, Truncated: true, SourceStatus: "changed", SourcePath: "/work/shots",
				Entries: []*v1.AskAttachmentEntry{{Path: "a.png", SizeBytes: 100}, {Path: "b.png", SizeBytes: 200}},
			},
		},
		Questions: []*v1.AskQuestion{
			{
				Id: "layout", Type: "single_choice", Prompt: "Layout?", Required: true,
				Choices:              []*v1.AskQuestionChoice{{Key: "cards", Label: "Cards"}, {Key: "table", Label: "Table", Attachment: "shots/a.png"}},
				Recommended:          &v1.AskQuestionResponse{QuestionId: "layout", Selected: []string{"cards"}},
				RecommendationReason: "Phone friendly.",
			},
			{Id: "plan", Type: "section_review", Prompt: "Review.", UnansweredVerdict: "drop", Sections: []*v1.AskQuestionSection{{Key: "storage", Title: "Storage"}}},
		},
		Responses: []*v1.AskQuestionResponse{
			{QuestionId: "layout", Selected: []string{"cards"}},
			{QuestionId: "plan", Sections: []*v1.AskSectionVerdict{{SectionKey: "storage", Verdict: "change", Note: "shorter"}}, Text: "ok"},
		},
		AttachmentNotes: []*v1.AskAttachmentNote{{Attachment: "why.html", Note: "Too dense."}, {Attachment: "shots", Path: "a.png", Note: "Crop tighter."}},
		AnswerSummary:   "Answer: Approve (approve)\nLayout: Cards (cards)",
		ContextRequests: []*v1.AskContextRequest{
			{Question: "Which region?", RequestedAt: "t1", AddressedAt: "t2", AddressedRevision: 1},
			{Question: "And the cost?", RequestedAt: "t3"},
		},
	}
}

func TestGetShowsEveryPartOfTheDecisionPacket(t *testing.T) {
	stub := &hubStub{getAsk: richAsk()}
	svc := newTestService(t, stub)
	stdout, _, err := runCommand(t, svc, "get", "ask-1")
	require.NoError(t, err)
	for _, want := range []string{
		"ask-1 [answered] Ship it? -> approve",
		"Subject key: release (revised 2 times)",
		"approve = Approve (recommended)",
		"why.html (explainer, file, text/html, 2.0 KiB, preview: html) - source fresh: /work/why.html",
		"shots (folder, 2 files, truncated, 300 B) Screens - source changed: /work/shots",
		"layout [single_choice] Layout? (required)",
		"table = Table [shots/a.png]",
		"recommended: cards - Phone friendly.",
		"plan [section_review] Review. (sections left alone: drop)",
		"storage: Storage",
		"layout: cards",
		`plan: storage change (shorter); "ok"`,
		"Attachment notes:",
		"why.html: Too dense.",
		"shots/a.png: Crop tighter.",
		"Answer summary:",
		"Layout: Cards (cards)",
		`"Which region?" (addressed in revision 1, asked t1)`,
		`"And the cost?" (unaddressed, asked t3)`,
	} {
		require.Contains(t, stdout, want)
	}
}

func TestGetOmitsPacketSectionsThatAreEmpty(t *testing.T) {
	stub := &hubStub{getAsk: &v1.Ask{Id: "a", State: "pending", Question: "Q?", Options: []*v1.AskOption{{Key: "x", Label: "X"}}}}
	svc := newTestService(t, stub)
	stdout, _, err := runCommand(t, svc, "get", "a")
	require.NoError(t, err)
	for _, heading := range []string{"Attachments:", "Questions:", "Responses:", "Attachment notes:", "Answer summary:", "Context requests:"} {
		require.NotContains(t, stdout, heading)
	}
	require.Contains(t, stdout, "Options:")
}

func TestGetJSONUsesActualProtoAnswerFields(t *testing.T) {
	changes, err := structpb.NewStruct(map[string]any{"pace": "use_by_wednesday", "steer": "more_self"})
	require.NoError(t, err)
	stub := &hubStub{getAsk: &v1.Ask{
		Id: "ask-1", State: "answered", SubjectKey: "portfolio-steward:daily:2026-11-16", Kind: "status",
		Answer: "send", AnswerLabel: "Send update", AnsweredAt: "2026-11-16T17:00:00Z",
		AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "personal_local",
		Note: "go faster", AnswerChanges: changes,
		AnswerImages: []*v1.AnswerImage{{Name: "plan.png", Sha256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", MediaType: "image/png", SizeBytes: 128}},
	}}
	stdout, _, err := runCommand(t, newTestService(t, stub), "get", "ask-1", "--json")
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &payload), "GetAsk --json is pure proto JSON: %q", stdout)
	ask, ok := payload["ask"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "send", ask["answer"], "the wire answer is a scalar")
	require.Equal(t, "Send update", ask["answer_label"])
	require.Equal(t, map[string]any{"pace": "use_by_wednesday", "steer": "more_self"}, ask["answer_changes"])
	require.NotContains(t, ask, "changes")
	require.Equal(t, "go faster", ask["note"])
	images, ok := ask["answer_images"].([]any)
	require.True(t, ok)
	require.Equal(t, map[string]any{"name": "plan.png", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "media_type": "image/png", "size_bytes": "128"}, images[0])
	require.NotContains(t, ask, "images")
}

func TestListSummariesCountPacketParts(t *testing.T) {
	line := askSummary(&v1.Ask{Id: "a", State: "pending", Question: "Q?", AttachmentCount: 2, QuestionCount: 1, OpenContextRequestCount: 1})
	require.True(t, strings.HasSuffix(line, "{2 attachments, 1 question, 1 open context request}"), line)
	require.Equal(t, "a [pending] Q?", askSummary(&v1.Ask{Id: "a", State: "pending", Question: "Q?"}))
}
