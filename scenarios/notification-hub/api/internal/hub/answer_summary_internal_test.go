package hub

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func summaryAsk() AskRecord {
	return AskRecord{
		State: AskAnswered, Answer: "approve", AnswerLabel: "Approve with changes",
		Questions: []AskQuestion{
			{ID: "layout", Type: QuestionSingleChoice, Prompt: "Layout?", Choices: []AskQuestionChoice{{Key: "cards", Label: "Card stack"}, {Key: "table", Label: "Table"}}},
			{ID: "extras", Type: QuestionMultiChoice, Prompt: "Which extras do you want?", Choices: []AskQuestionChoice{{Key: "dark", Label: "Dark mode"}, {Key: "export", Label: "Export"}}},
			{ID: "plan", Type: QuestionSectionReview, Prompt: "Plan sections", Sections: []AskQuestionSection{{Key: "storage", Title: "Storage"}, {Key: "retention", Title: "Retention"}, {Key: "billing", Title: "Billing"}}},
			{ID: "why", Type: QuestionText, Prompt: "Anything else?"},
		},
	}
}

func TestAnswerSummaryMatchesTheDocumentedExample(t *testing.T) {
	ask := summaryAsk()
	ask.Note = "Keep the retention period short."
	ask.Responses = []AskQuestionResponse{
		{QuestionID: "layout", Selected: []string{"cards"}},
		{QuestionID: "plan", Sections: []AskSectionVerdict{
			{SectionKey: "storage", Verdict: "approve"},
			{SectionKey: "retention", Verdict: "change", Note: "keep 30 days, not 90"},
			{SectionKey: "billing", Verdict: "drop"},
		}},
	}
	ask.AttachmentNotes = []AskAttachmentNote{{Attachment: "character.glb", Note: "Hair is too glossy."}}
	require.Equal(t, `Answer: Approve with changes (approve)
Note: Keep the retention period short.
Layout: Card stack (cards)
Plan sections:
  - Retention: change — keep 30 days, not 90
  - Billing: drop
  - Other sections: approve
Notes on files:
  - character.glb: Hair is too glossy.`, AnswerSummary(ask))
}

func TestAnswerSummaryRendersEveryShape(t *testing.T) {
	ask := summaryAsk()
	ask.Answer, ask.AnswerLabel = "other", "Other"
	ask.Note = "Neither.\nShip a prototype first."
	ask.Responses = []AskQuestionResponse{
		{QuestionID: "layout", Selected: []string{"table"}, Text: "dense is better"},
		{QuestionID: "extras", Selected: []string{"dark", "export"}},
		{QuestionID: "why", Text: "line one\nline two"},
		{QuestionID: "removed-later", Selected: []string{"x"}},
	}
	require.Equal(t, `Answer: Other (other)
Note: Neither.
  Ship a prototype first.
Layout: Table (table) — dense is better
Which extras do you want: Dark mode (dark), Export (export)
Anything else: line one
  line two
removed-later: x`, AnswerSummary(ask))
}

func TestAnswerSummaryToleratesLegacyAndLongInput(t *testing.T) {
	// An ask without a form summarizes as its option alone.
	require.Equal(t, "Answer: approve", AnswerSummary(AskRecord{State: AskAnswered, Answer: "approve", AnswerLabel: "approve"}))
	// A key with no recorded label, and a section or choice the form no longer has.
	ask := summaryAsk()
	ask.Answer, ask.AnswerLabel = "approve", ""
	ask.Responses = []AskQuestionResponse{
		{QuestionID: "layout", Selected: []string{"grid"}},
		{QuestionID: "plan", Sections: []AskSectionVerdict{{SectionKey: "legal", Verdict: "drop"}}},
	}
	require.Equal(t, "Answer: approve\nLayout: grid\nPlan sections:\n  - legal: drop\n  - Other sections: approve", AnswerSummary(ask))
	// A long prompt is shortened to a line label.
	long := AskRecord{
		State: AskAnswered, Answer: "a",
		Questions: []AskQuestion{{ID: "q", Type: QuestionText, Prompt: strings.Repeat("word ", 40) + "?"}},
		Responses: []AskQuestionResponse{{QuestionID: "q", Text: "yes"}},
	}
	lines := strings.Split(AnswerSummary(long), "\n")
	require.Len(t, lines, 2)
	require.Equal(t, strings.Repeat("word ", 15)+"word…: yes", lines[1])
}

func TestAnswerSummaryIsEmptyUnlessAnOperatorAnswered(t *testing.T) {
	for _, state := range []string{AskPending, AskEscalated, AskDefaulted, AskExpired, AskWithdrawn, AskDismissed} {
		ask := summaryAsk()
		ask.State, ask.Answer = state, "approve"
		require.Empty(t, AnswerSummary(ask), state)
	}
}

func TestAnswerSummaryOfASectionReviewFollowsTheUnansweredVerdict(t *testing.T) {
	review := func(unanswered string, verdicts ...AskSectionVerdict) AskRecord {
		ask := summaryAsk()
		ask.Questions = ask.Questions[2:3]
		ask.Questions[0].UnansweredVerdict = unanswered
		ask.Answer, ask.AnswerLabel = "approve", ""
		ask.Responses = []AskQuestionResponse{{QuestionID: "plan", Sections: verdicts}}
		return ask
	}
	// The sections that match the default (and carry no note) fold into one line.
	require.Equal(t, "Answer: approve\nPlan sections:\n  - Billing: approve — only if capped\n  - Other sections: drop",
		AnswerSummary(review("drop",
			AskSectionVerdict{SectionKey: "storage", Verdict: "drop"},
			AskSectionVerdict{SectionKey: "retention", Verdict: "drop"},
			AskSectionVerdict{SectionKey: "billing", Verdict: "approve", Note: "only if capped"})))
	// With every section listed there is no "Other sections" line.
	require.Equal(t, "Answer: approve\nPlan sections:\n  - Storage: change\n  - Retention: drop\n  - Billing: change",
		AnswerSummary(review("approve",
			AskSectionVerdict{SectionKey: "storage", Verdict: "change"},
			AskSectionVerdict{SectionKey: "retention", Verdict: "drop"},
			AskSectionVerdict{SectionKey: "billing", Verdict: "change"})))
	// A note on a section that matches the default is still shown.
	require.Contains(t, AnswerSummary(review("", AskSectionVerdict{SectionKey: "storage", Verdict: "approve", Note: "nice"})), "  - Storage: approve — nice\n  - Other sections: approve")
}

func TestAnswerSummaryListsAttachmentNotesWithTheirFolderPath(t *testing.T) {
	ask := AskRecord{State: AskAnswered, Answer: "approve", AttachmentNotes: []AskAttachmentNote{
		{Attachment: "character.glb", Note: "Hair is too glossy."},
		{Attachment: "shots", Path: "day-1/intro.png", Note: "Crop the top.\nKeep the logo."},
		{Attachment: "shots", Note: "Too many; cut half."},
	}}
	require.Equal(t, "Answer: approve\nNotes on files:\n  - character.glb: Hair is too glossy.\n  - shots/day-1/intro.png: Crop the top.\n    Keep the logo.\n  - shots: Too many; cut half.", AnswerSummary(ask))
}
