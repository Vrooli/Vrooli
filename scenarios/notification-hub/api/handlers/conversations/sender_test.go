package conversations_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/hubsender"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	"notification-hub/internal/hub"
)

func TestPortfolioStewardReadsOnlyTheAttestedAskAttachment(t *testing.T) {
	f := newReaderFixture(t)
	ctx := context.Background()
	content := []byte("<html><body>stored roadmap page</body></html>")
	hash := hashOf(content)
	_, err := f.client.PutAttachmentBlob(ctx, connect.NewRequest(&v1.PutAttachmentBlobRequest{Sha256: hash, Content: content}))
	require.NoError(t, err)
	key := "portfolio-steward:roadmap:2026-11-16"
	askID := f.ask(t, "run:valid-run-1+s=alice", key, func(req *v1.AskRequest) {
		req.SubjectKey = key
		req.Attachments = []*v1.AskAttachment{{Name: "decision.html", Kind: v1.AskAttachment_KIND_FILE, Sha256: hash}}
	})
	read := func(token string) *http.Response {
		req, requestErr := http.NewRequest(http.MethodGet, f.url+"/api/v1/asks/"+url.PathEscape(askID)+"/attachments/decision.html", nil)
		require.NoError(t, requestErr)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, requestErr := http.DefaultClient.Do(req)
		require.NoError(t, requestErr)
		return resp
	}
	before := read("snd-steward")
	require.Equal(t, http.StatusNotFound, before.StatusCode)
	_ = before.Body.Close()

	attestPortfolioAsk(t, f, "run:valid-run-1+s=alice", askID)
	response := read("snd-steward")
	body, readErr := io.ReadAll(response.Body)
	_ = response.Body.Close()
	require.NoError(t, readErr)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, string(content), string(body))
	require.Equal(t, `"sha256-`+hash+`"`, response.Header.Get("ETag"))

	foreign := read("snd-treasury")
	require.Equal(t, http.StatusNotFound, foreign.StatusCode, "another service cannot use the exact portfolio attestation")
	_ = foreign.Body.Close()
}

// A scenario that sends as itself holds a "<source>-notification-hub" service
// principal. The Hub reads the source off the principal's name: no request field
// names one, so what a sender raises is always attributed to its own scenario.

func senderPrincipal(name string) apiidentity.Principal {
	return apiidentity.Principal{
		Kind: apiidentity.ActorService, Subject: name, Verified: true, Source: apiidentity.SourceScenarioAuthenticator,
		Audience: hubsender.Audience, Scopes: []string{hubsender.Scope},
	}
}

func askSource(t *testing.T, f *readerFixture, askID string) string {
	t.Helper()
	got, err := f.client.GetAsk(context.Background(), reading("owner", &v1.GetAskRequest{AskId: askID}))
	require.NoError(t, err)
	return got.Msg.GetAsk().GetSource()
}

func TestAServiceSenderRaisesAsksAsItsOwnScenario(t *testing.T) {
	f := newReaderFixture(t)
	treasury := f.ask(t, "bearer:snd-treasury", "treasury-approve-budget")
	desk := f.ask(t, "bearer:snd-desk", "desk-approve-budget")

	require.NotEqual(t, treasury, desk)
	require.Equal(t, "treasury", askSource(t, f, treasury))
	require.Equal(t, "research-desk", askSource(t, f, desk))
}

func TestAServiceSenderReadsBackOnlyTheAsksItSent(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	mine := f.ask(t, "bearer:snd-treasury", "mine")
	theirs := f.ask(t, "bearer:snd-desk", "theirs")
	anonymous := f.ask(t, "", "shell")

	got, err := f.client.GetAsk(ctx, reading("bearer:snd-treasury", &v1.GetAskRequest{AskId: mine}))
	require.NoError(t, err)
	require.Equal(t, mine, got.Msg.GetAsk().GetId())
	list, err := f.client.ListAsks(ctx, reading("bearer:snd-treasury", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.Equal(t, []string{mine}, ids(list.Msg.GetAsks()))
	for _, other := range []string{theirs, anonymous} {
		_, err = f.client.GetAsk(ctx, reading("bearer:snd-treasury", &v1.GetAskRequest{AskId: other}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "a sender never reads another source's ask")
	}
}

func TestPortfolioStewardSenderReadsOnlyItsExactLegacySubjectKey(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	key := "portfolio-steward:roadmap:2026-11-16"
	legacy := f.ask(t, "run:valid-run-1+s=alice", "legacy", func(r *v1.AskRequest) { r.SubjectKey = key })
	foreign := f.ask(t, "run:valid-run-1", "foreign", func(r *v1.AskRequest) { r.SubjectKey = "treasury:approval:1" })

	_, err := f.client.GetAsk(ctx, reading("bearer:snd-steward", &v1.GetAskRequest{AskId: legacy}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "a prefix and caller-selected ID do not replace a stored owner/service association")
	_, err = f.client.GetAsk(ctx, reading("bearer:snd-steward", &v1.GetAskRequest{AskId: "arbitrary-portfolio-steward:roadmap:2026-11-16"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "an arbitrary requested ID cannot attest itself")

	// The current verified run carries the same owner identity as the ask and
	// can perform the narrow one-ask migration through the owner authorization path.
	attestPortfolioAsk(t, f, "run:valid-run-1+s=alice", legacy)
	got, err := f.client.GetAsk(ctx, reading("bearer:snd-steward", &v1.GetAskRequest{AskId: legacy}))
	require.NoError(t, err)
	require.Equal(t, legacy, got.Msg.GetAsk().GetId())
	byKey, err := f.client.GetAsk(ctx, reading("bearer:snd-steward", &v1.GetAskRequest{SubjectKey: key}))
	require.NoError(t, err)
	require.Equal(t, legacy, byKey.Msg.GetAsk().GetId())
	_, err = f.client.GetAsk(ctx, reading("bearer:snd-steward", &v1.GetAskRequest{AskId: foreign}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "exact-source recovery never exposes another producer's direct ask")

	event := f.eventAsk(t, "portfolio-steward", "event-lookalike")
	_, err = f.client.GetAsk(ctx, reading("bearer:snd-steward", &v1.GetAskRequest{AskId: event}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "direct-source attestation never exposes event-sourced asks")
	require.Equal(t, http.StatusNotFound, attestPortfolioAskStatus(t, f, "owner", event), "an event ask cannot be migrated as a direct ask")
}

func attestPortfolioAsk(t *testing.T, f *readerFixture, who, askID string) {
	t.Helper()
	status := attestPortfolioAskStatus(t, f, who, askID)
	require.True(t, status == http.StatusOK || status == http.StatusCreated, "unexpected attestation status %d", status)
}

func attestPortfolioAskStatus(t *testing.T, f *readerFixture, who, askID string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, f.url+"/api/v1/conversations/asks/"+url.PathEscape(askID)+"/portfolio-continuity", nil)
	require.NoError(t, err)
	switch {
	case who == "owner":
		req.Header.Set("X-Vrooli-Identity-Subject", "alice")
	case len(who) >= len("run:") && who[:len("run:")] == "run:":
		req.Header.Set(cliutil.HeaderAgentIdentityToken, who[len("run:"):])
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestPortfolioContinuityMigrationRequiresVerifiedSameOwnerAndPendingDirectAsk(t *testing.T) {
	f := newReaderFixture(t)
	key := "portfolio-steward:daily:2026-11-16"
	askID := f.ask(t, "run:valid-run-1+s=alice", "continuity-target", func(r *v1.AskRequest) { r.SubjectKey = key })
	require.Equal(t, http.StatusUnauthorized, attestPortfolioAskStatus(t, f, "", askID))
	require.Equal(t, http.StatusNotFound, attestPortfolioAskStatus(t, f, "run:valid-run-1+s=bob", askID), "another owner cannot attest the recipient's ask")
	_, err := f.client.GetAsk(context.Background(), reading("bearer:snd-steward", &v1.GetAskRequest{AskId: askID}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "refused attestations must leave the service unable to read")
	_, err = f.client.Ask(context.Background(), reading("bearer:snd-steward", &v1.AskRequest{
		Question: "A replacement question?", AllowedAnswers: []string{"yes", "no"},
		SensitivityLabel: "private", IdempotencyKey: "continuity-before-attestation", SubjectKey: key,
	}))
	require.Error(t, err, "a service without the exact stored association cannot revise the ask")
	_, err = f.client.Answer(context.Background(), reading("bearer:snd-steward", &v1.AnswerRequest{AskId: askID, Answer: "yes"}))
	require.Error(t, err, "the producer service cannot submit an operator answer")

	attestPortfolioAsk(t, f, "owner", askID)
	// A forged association for a different recipient is not proof. Removing it
	// also revokes service access immediately; restoring the exact owner proof
	// through the supported handler restores access.
	_, err = f.primary.ExecContext(context.Background(), `DELETE FROM ask_source_attestations WHERE ask_id=? AND source='portfolio-steward'`, askID)
	require.NoError(t, err)
	_, err = f.client.GetAsk(context.Background(), reading("bearer:snd-steward", &v1.GetAskRequest{AskId: askID}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "a deleted association revokes service access")
	_, err = f.primary.ExecContext(context.Background(), `INSERT INTO ask_source_attestations(ask_id,source,recipient,attested_at) VALUES(?,?,?,?)`, askID, "portfolio-steward", "mallory", time.Now().UTC().Format(time.RFC3339Nano))
	require.NoError(t, err)
	_, err = f.client.GetAsk(context.Background(), reading("bearer:snd-steward", &v1.GetAskRequest{AskId: askID}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "a forged association for another recipient grants no access")
	attestPortfolioAsk(t, f, "owner", askID)
	_, err = f.client.Answer(context.Background(), reading("owner", &v1.AnswerRequest{AskId: askID, Answer: "yes"}))
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, attestPortfolioAskStatus(t, f, "run:valid-run-1+s=alice", askID),
		"an answered ask cannot acquire a new service association")
	stored, err := f.client.GetAsk(context.Background(), reading("bearer:snd-steward", &v1.GetAskRequest{AskId: askID}))
	require.NoError(t, err, "the proven producer must be able to read its exact answer")
	require.Equal(t, "answered", stored.Msg.GetAsk().GetState())
	require.Equal(t, "yes", stored.Msg.GetAsk().GetAnswer())

	_, err = f.client.Ask(context.Background(), reading("bearer:snd-steward", &v1.AskRequest{
		Question: "A replacement question?", AllowedAnswers: []string{"yes", "no"},
		SensitivityLabel: "private", IdempotencyKey: "continuity-after-answer", SubjectKey: key,
	}))
	require.Error(t, err, "a settled ask must not be revised")
	answerReadback, err := f.client.GetAsk(context.Background(), reading("bearer:snd-steward", &v1.GetAskRequest{SubjectKey: key}))
	require.NoError(t, err)
	require.Equal(t, askID, answerReadback.Msg.GetAsk().GetId(), "a settled revision attempt must not create a replacement ask")
	require.Equal(t, "answered", answerReadback.Msg.GetAsk().GetState())

	// A same-prefix subject from a different verified service remains outside
	// the steward's attested audience even when its ask belongs to this owner.
	other := f.ask(t, "bearer:snd-treasury", "same-prefix-from-other-service", func(r *v1.AskRequest) {
		r.SubjectKey = "portfolio-steward:daily:2026-11-17"
	})
	require.Equal(t, http.StatusNotFound, attestPortfolioAskStatus(t, f, "owner", other),
		"the migration endpoint must not attest a same-prefix ask from another producer")
	_, err = f.client.GetAsk(context.Background(), reading("bearer:snd-steward", &v1.GetAskRequest{AskId: other}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestPortfolioContinuityUsesOnlyExistingCreatorOrConversationLineage(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	key := "portfolio-steward:roadmap:2026-11-16"
	conversationAsk := f.ask(t, "run:valid-origin+s=alice+c=steward-conversation", "lineage-conversation", func(r *v1.AskRequest) {
		r.SubjectKey = key
	})

	// The run has no owner subject. The verified conversation itself already
	// grants access to this exact ask, so it can establish the one-ask handoff.
	_, err := f.client.GetAsk(ctx, reading("run:valid-followup+s=+c=steward-conversation", &v1.GetAskRequest{AskId: conversationAsk}))
	require.NoError(t, err)
	attestPortfolioAsk(t, f, "run:valid-followup+s=+c=steward-conversation", conversationAsk)
	got, err := f.client.GetAsk(ctx, reading("bearer:snd-steward", &v1.GetAskRequest{AskId: conversationAsk}))
	require.NoError(t, err)
	require.Equal(t, conversationAsk, got.Msg.GetAsk().GetId())
	require.Equal(t, "pending", got.Msg.GetAsk().GetState())
	storedAsk, err := f.service.GetAsk(ctx, conversationAsk)
	require.NoError(t, err)
	var associatedRecipient string
	require.NoError(t, f.primary.QueryRowContext(ctx, `SELECT recipient FROM ask_source_attestations WHERE ask_id=? AND source='portfolio-steward'`, conversationAsk).Scan(&associatedRecipient))
	require.Equal(t, storedAsk.Recipient, associatedRecipient, "the durable association is bound to the ask's stored recipient")

	// An unrelated ownerless run cannot use a matching subject-key prefix or
	// a caller-selected ID to create the same association.
	unrelated := f.ask(t, "run:valid-origin+s=alice", "lineage-unrelated", func(r *v1.AskRequest) {
		r.SubjectKey = "portfolio-steward:daily:2026-11-16"
	})
	for _, run := range []string{"run:valid-outsider+s=", "run:valid-outsider+s=alice"} {
		require.Equal(t, http.StatusNotFound, attestPortfolioAskStatus(t, f, run, unrelated),
			"creator/conversation access is required; subject-key capability alone is insufficient")
	}
	_, err = f.client.GetAsk(ctx, reading("bearer:snd-steward", &v1.GetAskRequest{AskId: unrelated}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "a refused handoff leaves no service access")

	// The creator identity is itself an accepted canonical lineage, even when
	// the current run has no owner subject.
	creatorAsk := f.ask(t, "run:valid-creator+s=alice", "lineage-creator", func(r *v1.AskRequest) {
		r.SubjectKey = "portfolio-steward:daily:2026-11-17"
	})
	attestPortfolioAsk(t, f, "run:valid-creator+s=", creatorAsk)

	// A same-prefix direct ask from another verified service is not in the run
	// audience and remains outside the steward association.
	foreignSource := f.ask(t, "bearer:snd-treasury", "lineage-foreign-source", func(r *v1.AskRequest) {
		r.SubjectKey = "portfolio-steward:daily:2026-11-18"
	})
	require.Equal(t, http.StatusNotFound, attestPortfolioAskStatus(t, f, "run:valid-outsider+s=", foreignSource))
	_, err = f.client.GetAsk(ctx, reading("bearer:snd-steward", &v1.GetAskRequest{AskId: foreignSource}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestPortfolioContinuityAndAnswerCommitPreservePendingStateOrdering(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	askID := f.ask(t, "run:valid-origin+s=alice+c=answer-race", "lineage-answer-race", func(r *v1.AskRequest) {
		r.SubjectKey = "portfolio-steward:daily:2026-11-17"
	})

	answerReady := make(chan struct{})
	continueAnswer := make(chan struct{})
	var releaseOnce sync.Once
	releaseAnswer := func() { releaseOnce.Do(func() { close(continueAnswer) }) }
	defer releaseAnswer()
	f.service.SetBeforeAnswerCommitForTest(func() {
		close(answerReady)
		<-continueAnswer
	})
	defer f.service.SetBeforeAnswerCommitForTest(nil)
	answerDone := make(chan error, 1)
	go func() {
		_, err := f.service.AnswerAsk(ctx, hub.AnswerInput{AskID: askID, Answer: "yes", Actor: "alice"})
		answerDone <- err
	}()
	<-answerReady

	// The exact conversation handoff commits while the answer is paused after
	// validation. Once the real answer commits, the service can read it but has
	// no path to revise the now-settled ask.
	attestPortfolioAsk(t, f, "run:valid-followup+s=+c=answer-race", askID)
	releaseAnswer()
	require.NoError(t, <-answerDone)

	stored, err := f.client.GetAsk(ctx, reading("bearer:snd-steward", &v1.GetAskRequest{AskId: askID}))
	require.NoError(t, err)
	require.Equal(t, "answered", stored.Msg.GetAsk().GetState())
	require.Equal(t, "yes", stored.Msg.GetAsk().GetAnswer())
	_, err = f.client.Ask(ctx, reading("bearer:snd-steward", &v1.AskRequest{
		Question: "A replacement question?", AllowedAnswers: []string{"yes", "no"},
		SensitivityLabel: "private", IdempotencyKey: "continuity-after-raced-answer", SubjectKey: "portfolio-steward:daily:2026-11-17",
	}))
	require.Error(t, err, "the association cannot make the answered subject revisable")
}

// A sender token is not an ask-reader token and the reverse: each role has its own
// exclusive audience, so neither can be replayed as the other.
func TestSenderAndAskReaderTokensAreNotInterchangeable(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	event := f.eventAsk(t, "treasury", "e1")

	// A sender does not read the event asks of its source; that is the ask-reader's.
	_, err := f.client.GetAsk(ctx, reading("bearer:snd-treasury", &v1.GetAskRequest{AskId: event}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	// An ask-reader token cannot send.
	_, err = f.client.Ask(ctx, reading("bearer:svc-autoheal", &v1.AskRequest{
		Question: "Q?", AllowedAnswers: []string{"yes"}, SensitivityLabel: "private", IdempotencyKey: "k",
	}))
	require.Error(t, err)
	require.NotEqual(t, connect.CodeOf(err), connect.CodeUnknown)
}

// A presented credential that is not a valid sender is refused, never downgraded
// to the identityless bucket that accepts a caller with no credential at all.
func TestAnInvalidSenderCredentialIsRefusedNotDowngradedToAnonymous(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	cases := map[string]connect.Code{
		"unknown bearer":             connect.CodeUnauthenticated,
		"token of another audience":  connect.CodePermissionDenied,
		"sender without its scope":   connect.CodePermissionDenied,
		"principal not named sender": connect.CodePermissionDenied,
		"human token":                connect.CodePermissionDenied,
		"unverified principal":       connect.CodePermissionDenied,
	}
	who := map[string]string{
		"unknown bearer": "bearer:nothing", "token of another audience": "bearer:snd-wrong-aud",
		"sender without its scope": "bearer:snd-no-scope", "principal not named sender": "bearer:snd-other-name",
		"human token": "bearer:snd-human", "unverified principal": "bearer:snd-unverified",
	}
	for name, code := range cases {
		_, err := f.client.Ask(ctx, reading(who[name], &v1.AskRequest{
			Question: "Q?", AllowedAnswers: []string{"yes"}, SensitivityLabel: "private", IdempotencyKey: name,
		}))
		require.Equal(t, code, connect.CodeOf(err), name)
	}
	list, err := f.client.ListAsks(ctx, reading("owner", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.Empty(t, list.Msg.GetAsks(), "a refused sender creates nothing, not even an anonymous ask")
}

func TestARefusedSenderIsToldWhatIsMissing(t *testing.T) {
	f := newReaderFixture(t)
	_, err := f.client.Ask(context.Background(), reading("bearer:snd-other-name", &v1.AskRequest{
		Question: "Q?", AllowedAnswers: []string{"yes"}, SensitivityLabel: "private", IdempotencyKey: "k",
	}))
	require.Error(t, err)
	require.Contains(t, err.Error(), `"treasury" is not named <source>-notification-hub`)
	_, err = f.client.Ask(context.Background(), reading("bearer:snd-no-scope", &v1.AskRequest{
		Question: "Q?", AllowedAnswers: []string{"yes"}, SensitivityLabel: "private", IdempotencyKey: "k2",
	}))
	require.Contains(t, err.Error(), `lacks scope "notification-hub:send"`)
}

func TestAnAnonymousAskIsUnchangedBySenderSupport(t *testing.T) {
	f := newReaderFixture(t)
	anonymous := f.ask(t, "", "from-a-shell")
	require.Equal(t, "alice", askSource(t, f, anonymous), "an identityless ask keeps the recipient as its source")
	run := f.ask(t, "run:valid-run-1", "from-a-run")
	require.Equal(t, "alice", askSource(t, f, run), "a run's ask keeps the recipient as its source")
}
