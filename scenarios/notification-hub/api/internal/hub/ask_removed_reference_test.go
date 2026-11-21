package hub_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	hub "notification-hub/internal/hub"
)

// Removing or replacing a file that something else in the packet points at
// never fails the revision, and it never leaves a dead reference unreported:
// the packet check runs on the merged packet and names what now points at
// nothing.

const explainerNamingChart = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Compare</title></head><body><h1>Compare</h1><img src="chart.png" alt="Chart"></body></html>`

func TestRevision_RemovingAFileTheExplainerShowsIsReported(t *testing.T) {
	env := newAttachmentEnv(t)
	first := env.ask(t, "evt-0", env.file(t, "chart.png", "png bytes"),
		hub.AskAttachment{Name: "explainer.html", Kind: hub.AttachmentFile, Role: hub.AttachmentRoleExplainer, MediaType: "text/html", SHA256: env.put(t, explainerNamingChart)})
	require.NotContains(t, warningCodes(first.Warnings), "explainer_unknown_attachment", "the explainer's image is attached")
	env.advance(t, time.Minute)

	removed := env.revise(t, "evt-1", func(spec *hub.AskSpec) { spec.RemoveAttachments = []string{"chart.png"} })
	require.Equal(t, hub.AskUpdated, removed.Outcome, "the revision is applied; the dead reference is reported, not refused")
	warning := findWarning(t, removed.Warnings, "explainer_unknown_attachment")
	require.Contains(t, warning.Message, "chart.png")
	require.False(t, warning.CarriedOver, "it is news: this revision caused it")
}

func TestRevision_RemovingAFileAChoiceShowsIsReportedAndTheChoiceStays(t *testing.T) {
	env := newAttachmentEnv(t)
	spec := subjectSpec(env.now, "evt-0")
	spec.Attachments = []hub.AskAttachment{env.file(t, "cards.png", "cards"), env.file(t, "table.png", "table")}
	spec.Questions = []hub.AskQuestion{{ID: "layout", Type: "variant", Prompt: "Which layout?", Choices: []hub.AskQuestionChoice{
		{Key: "cards", Label: "Cards", Attachment: "cards.png"}, {Key: "table", Label: "Table", Attachment: "table.png"},
	}}}
	_, err := env.service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err)
	env.advance(t, time.Minute)

	removed := env.revise(t, "evt-1", func(spec *hub.AskSpec) { spec.RemoveAttachments = []string{"cards.png"} })
	require.Equal(t, hub.AskUpdated, removed.Outcome)
	warning := findWarning(t, removed.Warnings, hub.WarnQuestionAttachmentUnknown)
	require.Contains(t, warning.Message, "cards.png")
	require.Equal(t, []string{"layout"}, questionIDs(removed.Ask.Questions), "the question stays")
	require.Len(t, removed.Ask.Questions[0].Choices, 2, "the choice is kept; attaching the file again makes its image show again")
	require.Equal(t, "cards.png", removed.Ask.Questions[0].Choices[0].Attachment)
}
