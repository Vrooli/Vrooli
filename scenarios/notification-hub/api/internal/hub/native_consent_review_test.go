package hub_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"notification-hub/internal/hub"
)

func TestNotification_NativeConsentReviewURL_NamesTheApprovalPageOnlyForAnAgentManagerRequest(t *testing.T) {
	id := uuid.NewString()
	page := "https://agent-manager.example.test/api/v1/authorizations/" + id + "/approve"
	request := hub.Notification{SourceScenario: "agent-manager", Category: "authorization-requested", DedupeKey: id, ContextURL: page}
	require.Equal(t, page, request.NativeConsentReviewURL())

	refused := map[string]func(*hub.Notification){
		"another source":          func(n *hub.Notification) { n.SourceScenario = "treasury" },
		"another category":        func(n *hub.Notification) { n.Category = "completed" },
		"dedupe key is not an id": func(n *hub.Notification) { n.DedupeKey = "grant-1" },
		"link names another id": func(n *hub.Notification) {
			n.ContextURL = "https://agent-manager.example.test/api/v1/authorizations/" + uuid.NewString() + "/approve"
		},
		"link is not the approval path": func(n *hub.Notification) { n.ContextURL = "https://agent-manager.example.test/runs/" + id },
		"link carries a query":          func(n *hub.Notification) { n.ContextURL = page + "?next=https://evil.example" },
		"link carries credentials": func(n *hub.Notification) {
			n.ContextURL = "https://owner:secret@agent-manager.example.test/api/v1/authorizations/" + id + "/approve"
		},
		"link is plain http off host": func(n *hub.Notification) {
			n.ContextURL = "http://agent-manager.example.test/api/v1/authorizations/" + id + "/approve"
		},
		"link is a Hub path": func(n *hub.Notification) { n.ContextURL = "/delivery/" + id },
		"link is empty":      func(n *hub.Notification) { n.ContextURL = "" },
	}
	for name, mutate := range refused {
		t.Run(name, func(t *testing.T) {
			n := request
			mutate(&n)
			require.Empty(t, n.NativeConsentReviewURL())
		})
	}
}

func TestGetForRecipient_CarriesTheIdentityThatNamesANativePermissionRequest(t *testing.T) {
	ctx := context.Background()
	s, _ := testService(t)
	id := uuid.NewString()
	page := "https://agent-manager.example.test/api/v1/authorizations/" + id + "/approve"
	sent, err := s.Send(ctx, hub.SendInput{RequestedBy: "alice", Title: "Permission", Body: "Exact scope", SensitivityLabel: "private", IdempotencyKey: id, DedupeKey: id, SourceScenario: "agent-manager", Category: "authorization-requested", ContextURL: page})
	require.NoError(t, err)

	read, _, err := s.GetForRecipient(ctx, sent.ID, "alice")
	require.NoError(t, err)
	require.Equal(t, page, read.NativeConsentReviewURL())

	plain, err := s.Send(ctx, hub.SendInput{RequestedBy: "alice", Title: "Report", Body: "Ready", SensitivityLabel: "private", IdempotencyKey: "plain-1", SourceScenario: "reports", ContextURL: page})
	require.NoError(t, err)
	readPlain, _, err := s.GetForRecipient(ctx, plain.ID, "alice")
	require.NoError(t, err)
	require.Empty(t, readPlain.NativeConsentReviewURL(), "a link of the right shape from another source is not a permission request")
}
