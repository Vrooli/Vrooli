package conversations_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

// A scenario that sends as itself (a "<source>-notification-hub" principal) is a
// verified sender, so it can carry a whole decision packet: it stores the files,
// asks with them attached, and closes its own ask. An identityless host process
// still cannot store a file, and a sender cannot touch another source's ask.

const explainerPage = "<!doctype html><title>Compare</title><h1>Two options</h1>"

func (f *readerFixture) putAs(t *testing.T, who, content string) string {
	t.Helper()
	hash := hashOf([]byte(content))
	_, err := f.client.PutAttachmentBlob(context.Background(), reading(who, &v1.PutAttachmentBlobRequest{Sha256: hash, MediaType: "text/html", Content: []byte(content)}))
	require.NoError(t, err)
	return hash
}

func TestAServiceSenderCarriesAFullPacketWithAnExplainer(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	const sender = "bearer:snd-desk"

	stat, err := f.client.StatAttachmentBlobs(ctx, reading(sender, &v1.StatAttachmentBlobsRequest{Sha256: []string{hashOf([]byte(explainerPage))}}))
	require.NoError(t, err)
	require.Len(t, stat.Msg.GetMissing(), 1, "the Hub does not hold the page yet")
	hash := f.putAs(t, sender, explainerPage)

	response, err := f.client.Ask(ctx, reading(sender, &v1.AskRequest{
		Question: "Which option?", AllowedAnswers: []string{"a", "b"}, SensitivityLabel: "private", IdempotencyKey: "desk-packet",
		Deadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		Attachments: []*v1.AskAttachment{
			{Name: "compare.html", Label: "Compare the options", Kind: v1.AskAttachment_KIND_FILE, Role: v1.AskAttachment_ROLE_EXPLAINER, Sha256: hash},
		},
	}))
	require.NoError(t, err)
	for _, warning := range response.Msg.GetWarnings() {
		require.NotEqual(t, "dropped", warning.GetSeverity(), "a verified sender's packet is kept: %v", warning)
	}

	got, err := f.client.GetAsk(ctx, reading("owner", &v1.GetAskRequest{AskId: response.Msg.GetAskId()}))
	require.NoError(t, err)
	require.Equal(t, "research-desk", got.Msg.GetAsk().GetSource())
	require.Len(t, got.Msg.GetAsk().GetAttachments(), 1)
	require.Equal(t, v1.AskAttachment_ROLE_EXPLAINER, got.Msg.GetAsk().GetAttachments()[0].GetRole())

	// The sender reads back what it sent, packet included.
	back, err := f.client.GetAsk(ctx, reading(sender, &v1.GetAskRequest{AskId: response.Msg.GetAskId()}))
	require.NoError(t, err)
	require.Equal(t, response.Msg.GetAskId(), back.Msg.GetAsk().GetId())
}

func TestStoringAFileNeedsAVerifiedSender(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	put := &v1.PutAttachmentBlobRequest{Sha256: hashOf([]byte(explainerPage)), Content: []byte(explainerPage)}

	for who, code := range map[string]connect.Code{
		"":                           connect.CodeUnauthenticated,
		"bearer:nothing":             connect.CodeUnauthenticated,
		"bearer:snd-no-scope":        connect.CodePermissionDenied,
		"bearer:snd-wrong-aud":       connect.CodePermissionDenied,
		"bearer:svc-desk":            connect.CodePermissionDenied,
		"bearer:snd-desk":            0,
		"owner":                      0,
		"run:valid-r1+c=conv-1":      0,
		"run:not-a-verified-run-tok": connect.CodeUnauthenticated,
	} {
		_, err := f.client.PutAttachmentBlob(ctx, reading(who, put))
		if code == 0 {
			require.NoError(t, err, who)
			continue
		}
		require.Equal(t, code, connect.CodeOf(err), "%q: %v", who, err)
	}
}

func TestAServiceSenderWithdrawsItsOwnAskOnly(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	mine := f.ask(t, "bearer:snd-desk", "mine", func(m *v1.AskRequest) { m.SubjectKey = "desk/mine" })
	other := f.ask(t, "bearer:snd-treasury", "other")
	event := f.eventAsk(t, "research-desk", "event")

	for name, id := range map[string]string{"another sender's ask": other, "an event ask of its own source": event} {
		_, err := f.client.Withdraw(ctx, reading("bearer:snd-desk", &v1.WithdrawRequest{AskId: id, Reason: "No longer needed."}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), name)
	}
	_, err := f.client.Withdraw(ctx, reading("", &v1.WithdrawRequest{AskId: mine, Reason: "No longer needed."}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "no identity, no withdrawal")

	closed, err := f.client.Withdraw(ctx, reading("bearer:snd-desk", &v1.WithdrawRequest{AskId: mine, Reason: "No longer needed."}))
	require.NoError(t, err)
	require.Equal(t, "withdrawn", closed.Msg.GetState())

	again := f.ask(t, "bearer:snd-desk", "again", func(m *v1.AskRequest) { m.SubjectKey = "desk/again" })
	byKey, err := f.client.Withdraw(ctx, reading("bearer:snd-desk", &v1.WithdrawRequest{SubjectKey: "desk/again", Reason: "Answered elsewhere."}))
	require.NoError(t, err)
	require.Equal(t, again, byKey.Msg.GetAskId(), "a sender closes its ask by subject key too")
	require.Equal(t, "withdrawn", byKey.Msg.GetState())
}
