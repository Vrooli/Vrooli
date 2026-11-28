package hub_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	hub "notification-hub/internal/hub"
)

// An event producer cannot upload a file, so its page arrives inline. It must
// reach the operator as the ask's explainer, and a re-send with a new page must
// replace the old page rather than stack a second one.
func TestAnEventProducersInlinePageBecomesTheAsksExplainer(t *testing.T) {
	env := newAttachmentEnv(t)
	page := `<!doctype html><html><head><meta name="viewport" content="width=device-width, initial-scale=1"></head><body><h1>Delete caches?</h1></body></html>`
	spec := subjectSpec(env.now, "cache-cleanup")
	spec.ExplainerHTML = page
	first, err := env.service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err)

	list := env.attachments(t, first.Ask.ID)
	require.Len(t, list, 1)
	require.Equal(t, hub.InlineExplainerName, list[0].Name)
	require.Equal(t, hub.AttachmentRoleExplainer, list[0].Role)
	stored, err := env.service.ReadAttachmentBlob(context.Background(), list[0].SHA256, 1<<20)
	require.NoError(t, err)
	require.Equal(t, page, string(stored))

	revised := subjectSpec(env.now, "cache-cleanup")
	revised.ExplainerHTML = page + "<!-- revised -->"
	revised.IdempotencyKey = "event-2" // a changed page is a new event
	second, err := env.service.AskDecision(context.Background(), revised, "")
	require.NoError(t, err)
	list = env.attachments(t, second.Ask.ID)
	require.Len(t, list, 1, "a re-sent page replaces the old one by name")
	require.Equal(t, hub.AttachmentRoleExplainer, list[0].Role)
	require.NotEqual(t, sha(page), list[0].SHA256)
}
