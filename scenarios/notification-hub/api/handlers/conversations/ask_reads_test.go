package conversations_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/hubsender"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	conversationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"

	conversationsH "notification-hub/handlers/conversations"
	"notification-hub/internal/hub"
	"notification-hub/internal/identity"
	"notification-hub/internal/modules"
	"notification-hub/internal/testutil/hubtest"
)

// Reading an ask needs an identity: the owner, the run that sent the ask, or
// the service that raised it as an event. These tests pin who sees what.

const autohealEventType = "autoheal.remediation.decision_requested.v1"

// serviceTokens stands in for scenario-authenticator's service principals.
type serviceTokens map[string]apiidentity.Principal

func (t serviceTokens) Verify(_ context.Context, token string) (apiidentity.Principal, error) {
	principal, ok := t[token]
	if !ok {
		return apiidentity.Principal{}, apiidentity.NewFailure(apiidentity.FailureInvalid, apiidentity.SourceScenarioAuthenticator)
	}
	return principal, nil
}

func servicePrincipal(name string) apiidentity.Principal {
	return apiidentity.Principal{
		Kind: apiidentity.ActorService, Subject: name, Verified: true, Source: apiidentity.SourceScenarioAuthenticator,
		Audience: askreader.Audience, Scopes: []string{askreader.Scope},
	}
}

type readerFixture struct {
	client  conversationsconnect.ConversationsServiceClient
	service *hub.Service
	// url is the address of the in-process Hub, for a caller that is not a Connect client.
	url string
}

// newReaderFixture serves the conversations module the way production does:
// verified agent runs through the provenance middleware ("valid-<run>" tokens),
// service readers through a token verifier, and the owner "alice" through the
// legacy subject header the Hub's tests trust.
func newReaderFixture(t *testing.T) *readerFixture {
	t.Helper()
	return newReaderFixtureBehind(t, nil)
}

// newReaderFixtureBehind is newReaderFixture with a middleware mounted in front
// of the handlers, the way the API mounts its owner authentication.
func newReaderFixtureBehind(t *testing.T, behind func(http.Handler) http.Handler) *readerFixture {
	t.Helper()
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	service := hubtest.New(t, database.NewFromPrimary(primary), nil, nil)
	service.SetDefaultRecipient("alice")
	service.SetAttachmentStore(hub.StaticBlobStore(blobstore.NewFilesystemBlobStore(t.TempDir())))
	router := mux.NewRouter()
	tokens := serviceTokens{
		"svc-autoheal":    servicePrincipal("vrooli-autoheal-ask-reader"),
		"svc-desk":        servicePrincipal("research-desk-ask-reader"),
		"svc-storage":     servicePrincipal("storage-manager-ask-reader"),
		"svc-wrong-aud":   {Kind: apiidentity.ActorService, Subject: "vrooli-autoheal-ask-reader", Verified: true, Audience: "scenario-authenticator:other", Scopes: []string{askreader.Scope}},
		"svc-no-scope":    {Kind: apiidentity.ActorService, Subject: "vrooli-autoheal-ask-reader", Verified: true, Audience: askreader.Audience},
		"svc-human":       {Kind: apiidentity.ActorHuman, Subject: "vrooli-autoheal-ask-reader", Verified: true, Audience: askreader.Audience, Scopes: []string{askreader.Scope}},
		"svc-unverified":  {Kind: apiidentity.ActorService, Subject: "vrooli-autoheal-ask-reader", Audience: askreader.Audience, Scopes: []string{askreader.Scope}},
		"svc-other-name":  {Kind: apiidentity.ActorService, Subject: "vrooli-autoheal", Verified: true, Audience: askreader.Audience, Scopes: []string{askreader.Scope}},
		"svc-bad-subject": {Kind: apiidentity.ActorService, Subject: "Not A Name", Verified: true, Audience: askreader.Audience, Scopes: []string{askreader.Scope}},
		// Sender principals (see sender_test.go). One verifier answers for both
		// roles here; each role's own audience, scope and name rule tell them apart.
		"snd-treasury":   senderPrincipal("treasury-notification-hub"),
		"snd-desk":       senderPrincipal("research-desk-notification-hub"),
		"snd-wrong-aud":  {Kind: apiidentity.ActorService, Subject: "treasury-notification-hub", Verified: true, Audience: askreader.Audience, Scopes: []string{hubsender.Scope}},
		"snd-no-scope":   {Kind: apiidentity.ActorService, Subject: "treasury-notification-hub", Verified: true, Audience: hubsender.Audience},
		"snd-other-name": {Kind: apiidentity.ActorService, Subject: "treasury", Verified: true, Audience: hubsender.Audience, Scopes: []string{hubsender.Scope}},
		"snd-human":      {Kind: apiidentity.ActorHuman, Subject: "treasury-notification-hub", Verified: true, Audience: hubsender.Audience, Scopes: []string{hubsender.Scope}},
		"snd-unverified": {Kind: apiidentity.ActorService, Subject: "treasury-notification-hub", Audience: hubsender.Audience, Scopes: []string{hubsender.Scope}},
	}
	conversationsH.ModuleWithIdentities(service, nil, identity.ServiceVerifiers{AskReaders: tokens, Senders: tokens}).Mount(router)
	runs := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		rest, ok := strings.CutPrefix(token, "valid-")
		if !ok {
			return &cliutil.VerifyResult{Valid: false}, nil
		}
		// "valid-<run>[+c=<conversation>][+s=<owner subject>]": the claims the
		// agent-manager token would carry. The owner defaults to "acting-agent".
		parts := strings.Split(rest, "+")
		claims := &cliutil.VerifiedClaims{RunID: parts[0], Subject: "acting-agent"}
		for _, part := range parts[1:] {
			switch {
			case strings.HasPrefix(part, "c="):
				claims.ConversationID = strings.TrimPrefix(part, "c=")
			case strings.HasPrefix(part, "s="):
				claims.Subject = strings.TrimPrefix(part, "s=")
			}
		}
		return &cliutil.VerifyResult{Valid: true, Claims: claims}, nil
	})
	var entry http.Handler = router
	if behind != nil {
		entry = behind(router)
	}
	server := httptest.NewServer(provenance.Middleware(runs)(entry))
	t.Cleanup(server.Close)
	return &readerFixture{client: conversationsconnect.NewConversationsServiceClient(server.Client(), server.URL), service: service, url: server.URL}
}

// reading stamps a request with one kind of caller: "" (none), "owner",
// "run:<token>", or "bearer:<token>".
func reading[T any](who string, msg *T) *connect.Request[T] {
	req := connect.NewRequest(msg)
	switch {
	case who == "owner":
		req.Header().Set("X-Vrooli-Identity-Subject", "alice")
	case strings.HasPrefix(who, "run:"):
		req.Header().Set(cliutil.HeaderAgentIdentityToken, strings.TrimPrefix(who, "run:"))
	case strings.HasPrefix(who, "bearer:"):
		req.Header().Set("Authorization", "Bearer "+strings.TrimPrefix(who, "bearer:"))
	}
	return req
}

func (f *readerFixture) ask(t *testing.T, who, key string, extra ...func(*v1.AskRequest)) string {
	t.Helper()
	msg := &v1.AskRequest{
		Question: "Continue with " + key + "?", AllowedAnswers: []string{"yes", "no"},
		Deadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339), SensitivityLabel: "private", IdempotencyKey: key,
	}
	for _, apply := range extra {
		apply(msg)
	}
	response, err := f.client.Ask(context.Background(), reading(who, msg))
	require.NoError(t, err)
	return response.Msg.GetAskId()
}

// eventAsk raises an event-sourced ask the way the integrations webhook does.
func (f *readerFixture) eventAsk(t *testing.T, source, key string) string {
	t.Helper()
	result, err := f.service.AskDecision(context.Background(), hub.AskSpec{
		Recipient: "alice", Source: source, SourceEventType: autohealEventType, Question: "Run the fix " + key + "?",
		Options:  []hub.AskOption{{Key: "approve", Label: "Approve"}, {Key: "reject", Label: "Reject"}},
		Deadline: time.Now().Add(time.Hour), DeadlineSpecified: true, SensitivityLabel: "private", IdempotencyKey: key,
	}, "")
	require.NoError(t, err)
	return result.Ask.ID
}

func ids(asks []*v1.Ask) []string {
	out := make([]string, 0, len(asks))
	for _, ask := range asks {
		out = append(out, ask.GetId())
	}
	return out
}

func waitFor(askID string) *v1.WaitRequest {
	return &v1.WaitRequest{AskId: askID, Deadline: time.Now().Add(time.Second).UTC().Format(time.RFC3339)}
}

func TestAskReadsRefuseACallerWithNoIdentity(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	askID := f.ask(t, "run:valid-run-1", "mine")

	_, err := f.client.GetAsk(ctx, reading("", &v1.GetAskRequest{AskId: askID}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = f.client.ListAsks(ctx, reading("", &v1.ListAsksRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "an unidentified caller must not enumerate ask ids")
	_, err = f.client.Wait(ctx, reading("", waitFor(askID)))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestAskReadsRefuseCredentialsThatDoNotVerify(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	askID := f.eventAsk(t, "vrooli-autoheal", "e1")
	for name, who := range map[string]string{
		"forged run token":         "run:forged",
		"unknown bearer":           "bearer:nothing",
		"service wrong audience":   "bearer:svc-wrong-aud",
		"service without scope":    "bearer:svc-no-scope",
		"human on service route":   "bearer:svc-human",
		"unverified principal":     "bearer:svc-unverified",
		"service name not a name":  "bearer:svc-bad-subject",
		"principal not ask-reader": "bearer:svc-other-name",
	} {
		_, err := f.client.GetAsk(ctx, reading(who, &v1.GetAskRequest{AskId: askID}))
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), name)
	}
}

func TestACreatorRunReadsListsAndWaitsOnItsOwnAsks(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	mine := f.ask(t, "run:valid-run-1", "mine")

	got, err := f.client.GetAsk(ctx, reading("run:valid-run-1", &v1.GetAskRequest{AskId: mine}))
	require.NoError(t, err)
	require.Equal(t, "Continue with mine?", got.Msg.GetAsk().GetQuestion())

	list, err := f.client.ListAsks(ctx, reading("run:valid-run-1", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.Equal(t, []string{mine}, ids(list.Msg.GetAsks()))

	waited, err := f.client.Wait(ctx, reading("run:valid-run-1", waitFor(mine)))
	require.NoError(t, err)
	require.Equal(t, mine, waited.Msg.GetAskId())
	require.Equal(t, "pending", waited.Msg.GetState())
}

func TestAnotherRunCannotReadListOrWaitOnAnAsk(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	mine := f.ask(t, "run:valid-run-1", "mine")
	theirs := f.ask(t, "run:valid-run-2", "theirs")

	_, err := f.client.GetAsk(ctx, reading("run:valid-run-2", &v1.GetAskRequest{AskId: mine}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "another run's ask is reported exactly like a missing one")
	_, err = f.client.GetAsk(ctx, reading("run:valid-run-2", &v1.GetAskRequest{AskId: "no-such-ask"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	_, err = f.client.Wait(ctx, reading("run:valid-run-2", waitFor(mine)))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	list, err := f.client.ListAsks(ctx, reading("run:valid-run-2", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.Equal(t, []string{theirs}, ids(list.Msg.GetAsks()), "a list holds only the caller's own asks")
	searched, err := f.client.ListAsks(ctx, reading("run:valid-run-2", &v1.ListAsksRequest{Query: "mine"}))
	require.NoError(t, err)
	require.Empty(t, searched.Msg.GetAsks(), "a search cannot reach other callers' asks")
}

func TestAnIdentitylessAskIsReadableOnlyByTheOwner(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	anonymous := f.ask(t, "", "from-a-shell")

	_, err := f.client.GetAsk(ctx, reading("run:valid-run-1", &v1.GetAskRequest{AskId: anonymous}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "an ask with no creating identity has no run that may read it")
	list, err := f.client.ListAsks(ctx, reading("run:valid-run-1", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.Empty(t, list.Msg.GetAsks())

	got, err := f.client.GetAsk(ctx, reading("owner", &v1.GetAskRequest{AskId: anonymous}))
	require.NoError(t, err)
	require.Equal(t, anonymous, got.Msg.GetAsk().GetId())
}

func TestTheOwnerReadsEveryAskOfTheRecipient(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	first := f.ask(t, "run:valid-run-1", "one")
	second := f.ask(t, "run:valid-run-2", "two")
	anonymous := f.ask(t, "", "three")
	event := f.eventAsk(t, "vrooli-autoheal", "e1")

	list, err := f.client.ListAsks(ctx, reading("owner", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.ElementsMatch(t, []string{first, second, anonymous, event}, ids(list.Msg.GetAsks()))
}

func TestAServiceReadsOnlyTheEventAsksItRaised(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	autoheal := f.eventAsk(t, "vrooli-autoheal", "e1")
	desk := f.eventAsk(t, "research-desk", "e2")
	direct := f.ask(t, "run:valid-run-1", "direct")
	// A direct ask whose source text equals the service name must stay out of
	// reach: only event-sourced asks are read back by their source.
	lookalike, err := f.service.AskDecision(ctx, hub.AskSpec{
		Recipient: "alice", Source: "vrooli-autoheal", Question: "Direct lookalike?",
		Options:  []hub.AskOption{{Key: "yes", Label: "Yes"}, {Key: "no", Label: "No"}},
		Deadline: time.Now().Add(time.Hour), DeadlineSpecified: true, SensitivityLabel: "private", IdempotencyKey: "lookalike",
	}, "")
	require.NoError(t, err)

	got, err := f.client.GetAsk(ctx, reading("bearer:svc-autoheal", &v1.GetAskRequest{AskId: autoheal}))
	require.NoError(t, err)
	require.Equal(t, autoheal, got.Msg.GetAsk().GetId())

	for name, askID := range map[string]string{"another service's ask": desk, "a run's ask": direct, "a direct ask with its name as source": lookalike.Ask.ID} {
		_, err := f.client.GetAsk(ctx, reading("bearer:svc-autoheal", &v1.GetAskRequest{AskId: askID}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), name)
		_, err = f.client.Wait(ctx, reading("bearer:svc-autoheal", waitFor(askID)))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), name)
	}

	list, err := f.client.ListAsks(ctx, reading("bearer:svc-autoheal", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.Equal(t, []string{autoheal}, ids(list.Msg.GetAsks()))
	deskList, err := f.client.ListAsks(ctx, reading("bearer:svc-desk", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.Equal(t, []string{desk}, ids(deskList.Msg.GetAsks()))

	waited, err := f.client.Wait(ctx, reading("bearer:svc-autoheal", waitFor(autoheal)))
	require.NoError(t, err)
	require.Equal(t, "pending", waited.Msg.GetState())
}

func TestAServiceReaderCannotAnswerOrDismiss(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	askID := f.eventAsk(t, "vrooli-autoheal", "e1")
	_, err := f.client.Answer(ctx, reading("bearer:svc-autoheal", &v1.AnswerRequest{AskId: askID, Answer: "approve"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = f.client.Dismiss(ctx, reading("bearer:svc-autoheal", &v1.DismissRequest{AskId: askID}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = f.client.Answer(ctx, reading("run:valid-run-1", &v1.AnswerRequest{AskId: askID, Answer: "approve"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestARunPresentingAnAgentTokenIsNeverTheOwnerEvenWithTheOwnerHeader(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	other := f.ask(t, "run:valid-run-2", "theirs")
	req := reading("owner", &v1.GetAskRequest{AskId: other})
	req.Header().Set(cliutil.HeaderAgentIdentityToken, "valid-run-1")
	_, err := f.client.GetAsk(ctx, req)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "an agent identity selects the run reader; it cannot borrow the owner")
}

func TestOnlyTheOwnerGetsTheSourceDriftCheck(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	host, err := os.Hostname()
	require.NoError(t, err)
	content := []byte("the plan")
	source := filepath.Join(t.TempDir(), "plan.md")
	require.NoError(t, os.WriteFile(source, content, 0o600))
	hash := hashOf(content)
	_, err = f.client.PutAttachmentBlob(ctx, reading("run:valid-run-1", &v1.PutAttachmentBlobRequest{Sha256: hash, Content: content}))
	require.NoError(t, err)
	askID := f.ask(t, "run:valid-run-1", "drift", func(r *v1.AskRequest) {
		r.Attachments = []*v1.AskAttachment{{Name: "plan.md", Kind: v1.AskAttachment_KIND_FILE, Sha256: hash, SourcePath: source, SourceHost: host}}
	})

	owner, err := f.client.GetAsk(ctx, reading("owner", &v1.GetAskRequest{AskId: askID}))
	require.NoError(t, err)
	require.Equal(t, "fresh", owner.Msg.GetAsk().GetAttachments()[0].GetSourceStatus(), "the owner learns whether the source still matches")

	// The creator knows its own path already, but must not be able to turn the
	// Hub into a file-existence and content oracle.
	require.NoError(t, os.WriteFile(source, []byte("changed"), 0o600))
	creator, err := f.client.GetAsk(ctx, reading("run:valid-run-1", &v1.GetAskRequest{AskId: askID}))
	require.NoError(t, err)
	require.Equal(t, "unknown", creator.Msg.GetAsk().GetAttachments()[0].GetSourceStatus())
	require.Empty(t, creator.Msg.GetAsk().GetAttachments()[0].GetSourceCheckedAt())
	changed, err := f.client.GetAsk(ctx, reading("owner", &v1.GetAskRequest{AskId: askID}))
	require.NoError(t, err)
	require.Equal(t, "changed", changed.Msg.GetAsk().GetAttachments()[0].GetSourceStatus())
}

func TestARunThatReusesTheSubjectKeyOfAnotherOwnersAskIsToldItCannotReadIt(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	subject := func(r *v1.AskRequest) { r.SubjectKey = "disk-pressure" }
	original := f.ask(t, "run:valid-run-1", "first", subject)
	revise := func(who, idempotency string) *v1.AskResponse {
		t.Helper()
		response, err := f.client.Ask(ctx, reading(who, &v1.AskRequest{
			Question: "Continue with first?", AllowedAnswers: []string{"yes", "no"}, SubjectKey: "disk-pressure",
			Deadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339), SensitivityLabel: "private", IdempotencyKey: idempotency,
		}))
		require.NoError(t, err)
		require.Equal(t, original, response.Msg.GetAskId(), "the subject key keeps one open ask")
		return response.Msg
	}
	restricted := func(response *v1.AskResponse) bool {
		for _, w := range response.GetWarnings() {
			if w.GetCode() == "ask_read_restricted" {
				return true
			}
		}
		return false
	}

	require.False(t, restricted(revise("run:valid-run-1", "second")), "the creator is not warned about its own ask")
	require.False(t, restricted(revise("run:valid-run-2", "third")),
		"an unrelated run of the same owner holds the key, so it can read what it revised and is not warned")
	require.True(t, restricted(revise("run:valid-run-3+s=another-owner", "fourth")),
		"a run of another owner revising the ask cannot read it back, and is told so")
	_, err := f.client.GetAsk(ctx, reading("run:valid-run-3+s=another-owner", &v1.GetAskRequest{AskId: original}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	require.True(t, restricted(revise("run:valid-run-4+s=", "fifth")), "a run with no verified owner holds no key")
}

func TestARecoveredRunReadsListsAndWaitsOnItsPredecessorsAsks(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	// A recovered run has a new run id and inherits the conversation of the run
	// it replaces; so does a child run, which also has its own id.
	predecessor := f.ask(t, "run:valid-run-1+c=thread-1", "before")
	for name, reader := range map[string]string{
		"recovered run": "run:valid-run-2+c=thread-1",
		"child run":     "run:valid-run-3+c=thread-1",
		"re-attached":   "run:valid-run-4+c=thread-1",
	} {
		got, err := f.client.GetAsk(ctx, reading(reader, &v1.GetAskRequest{AskId: predecessor}))
		require.NoError(t, err, name)
		require.Equal(t, predecessor, got.Msg.GetAsk().GetId(), name)
		list, err := f.client.ListAsks(ctx, reading(reader, &v1.ListAsksRequest{}))
		require.NoError(t, err, name)
		require.Equal(t, []string{predecessor}, ids(list.Msg.GetAsks()), name)
		waited, err := f.client.Wait(ctx, reading(reader, waitFor(predecessor)))
		require.NoError(t, err, name)
		require.Equal(t, "pending", waited.Msg.GetState(), name)
	}
}

func TestListAsksShowsOnlyTheCallersConversationLineage(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	first := f.ask(t, "run:valid-run-1+c=thread-1", "one")
	second := f.ask(t, "run:valid-run-2+c=thread-1", "two")
	elsewhere := f.ask(t, "run:valid-run-3+c=thread-2", "three")
	noConversation := f.ask(t, "run:valid-run-4", "four")
	f.ask(t, "", "five")

	list, err := f.client.ListAsks(ctx, reading("run:valid-run-5+c=thread-1", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.ElementsMatch(t, []string{first, second}, ids(list.Msg.GetAsks()), "the lineage, and no ask of another conversation or run")
	other, err := f.client.ListAsks(ctx, reading("run:valid-run-3+c=thread-2", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.Equal(t, []string{elsewhere}, ids(other.Msg.GetAsks()))
	// A token with no conversation claim falls back to the run id.
	alone, err := f.client.ListAsks(ctx, reading("run:valid-run-4", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.Equal(t, []string{noConversation}, ids(alone.Msg.GetAsks()))
	other, err = f.client.ListAsks(ctx, reading("run:valid-run-6", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.Empty(t, other.Msg.GetAsks(), "a run with no conversation sees nothing of the others")
}

func TestARunOutsideTheLineageCannotReadAnAskByIdEvenWithTheSameOwner(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	askID := f.ask(t, "run:valid-run-1+c=thread-1", "mine", func(r *v1.AskRequest) { r.SubjectKey = "plan-review" })
	for name, reader := range map[string]string{
		"another conversation, same owner":                        "run:valid-run-2+c=thread-2",
		"no conversation, same owner":                             "run:valid-run-3",
		"another owner, same conversation id is not a capability": "run:valid-run-4+c=thread-2+s=another-owner",
	} {
		_, err := f.client.GetAsk(ctx, reading(reader, &v1.GetAskRequest{AskId: askID}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), name)
		_, err = f.client.Wait(ctx, reading(reader, waitFor(askID)))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), name)
	}
}

func withdrawal(askID, key string) *v1.WithdrawRequest {
	return &v1.WithdrawRequest{AskId: askID, SubjectKey: key, Reason: "The plan changed."}
}

func TestWithdrawFollowsTheReadRule(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	keyed := func(r *v1.AskRequest) { r.SubjectKey = "plan-review" }
	askID := f.ask(t, "run:valid-run-1+c=thread-1", "mine", keyed)

	// By id: only the conversation lineage, never an unrelated run.
	_, err := f.client.Withdraw(ctx, reading("run:valid-run-2+c=thread-2", withdrawal(askID, "")))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "an unrelated run may not withdraw by id")
	// By subject key: a run of another owner holds no capability.
	_, err = f.client.Withdraw(ctx, reading("run:valid-run-3+c=thread-3+s=another-owner", withdrawal("", "plan-review")))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "a different owner is denied")
	// An unverified caller is refused outright.
	_, err = f.client.Withdraw(ctx, reading("", withdrawal("", "plan-review")))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	stillOpen, err := f.client.GetAsk(ctx, reading("owner", &v1.GetAskRequest{AskId: askID}))
	require.NoError(t, err)
	require.Equal(t, "pending", stillOpen.Msg.GetAsk().GetState(), "no refused withdrawal closed the ask")

	// By subject key: an unrelated run of the same owner closes it.
	closed, err := f.client.Withdraw(ctx, reading("run:valid-run-4+c=thread-4", withdrawal("", "plan-review")))
	require.NoError(t, err)
	require.Equal(t, askID, closed.Msg.GetAskId())
	require.Equal(t, "withdrawn", closed.Msg.GetState())

	// By id: the lineage closes its own ask.
	next := f.ask(t, "run:valid-run-1+c=thread-1", "next", keyed)
	closed, err = f.client.Withdraw(ctx, reading("run:valid-run-5+c=thread-1", withdrawal(next, "")))
	require.NoError(t, err)
	require.Equal(t, "withdrawn", closed.Msg.GetState())
}

func TestTheOwnerWithdrawsAnyDirectAsk(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	askID := f.ask(t, "run:valid-run-1+c=thread-1", "mine")
	closed, err := f.client.Withdraw(ctx, reading("owner", withdrawal(askID, "")))
	require.NoError(t, err)
	require.Equal(t, "withdrawn", closed.Msg.GetState())
}

func TestAskReaderVerifierAcceptsOnlyServiceTokensOfItsExclusiveAudience(t *testing.T) {
	var verifier authn.TokenVerifier = askreader.NewVerifier()
	_, err := verifier.Verify(context.Background(), "not-a-jwt")
	require.Error(t, err, "a malformed token is never an identity")
}

func keyed(key string) func(*v1.AskRequest) {
	return func(r *v1.AskRequest) { r.SubjectKey = key }
}

func waitForKey(key string) *v1.WaitRequest {
	return &v1.WaitRequest{SubjectKey: key, Deadline: time.Now().Add(time.Second).UTC().Format(time.RFC3339)}
}

// A restarted agent has a new run in another conversation. Re-deriving the
// subject key is its way back to the ask.
func TestAnUnrelatedRunOfTheSameOwnerGetsAndWaitsOnAnAskByItsSubjectKey(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	askID := f.ask(t, "run:valid-run-1+c=thread-1", "plan", keyed("plan-review"))

	got, err := f.client.GetAsk(ctx, reading("run:valid-run-2+c=thread-2", &v1.GetAskRequest{SubjectKey: "plan-review"}))
	require.NoError(t, err)
	require.Equal(t, askID, got.Msg.GetAsk().GetId())
	waited, err := f.client.Wait(ctx, reading("run:valid-run-2+c=thread-2", waitForKey("plan-review")))
	require.NoError(t, err)
	require.Equal(t, askID, waited.Msg.GetAskId())
	require.Equal(t, "pending", waited.Msg.GetState())

	// The key reaches one ask. It never lists, and it does not open the id route.
	list, err := f.client.ListAsks(ctx, reading("run:valid-run-2+c=thread-2", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.Empty(t, list.Msg.GetAsks())
	_, err = f.client.GetAsk(ctx, reading("run:valid-run-2+c=thread-2", &v1.GetAskRequest{AskId: askID}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	// The owner answers; the same key now reads the answered ask.
	_, err = f.client.Answer(ctx, reading("owner", &v1.AnswerRequest{AskId: askID, Answer: "yes"}))
	require.NoError(t, err)
	waited, err = f.client.Wait(ctx, reading("run:valid-run-2+c=thread-2", waitForKey("plan-review")))
	require.NoError(t, err)
	require.Equal(t, "answered", waited.Msg.GetState())
	require.Equal(t, "yes", waited.Msg.GetAnswer())
}

func TestASubjectKeyOpensNothingToAnotherOwnerAnAnonymousCallerOrARunWithNoOwner(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	f.ask(t, "run:valid-run-1+c=thread-1", "plan", keyed("plan-review"))
	f.ask(t, "", "from-a-shell", keyed("shell-ask"))

	for name, reader := range map[string]string{
		"a different owner":    "run:valid-run-2+c=thread-2+s=another-owner",
		"a run with no owner":  "run:valid-run-3+c=thread-3+s=",
		"a service reader":     "bearer:svc-autoheal",
		"a key nobody sent":    "run:valid-run-4+c=thread-4",
		"an ask with no owner": "run:valid-run-4+c=thread-4",
	} {
		key := "plan-review"
		switch name {
		case "a key nobody sent":
			key = "no-such-key"
		case "an ask with no owner":
			key = "shell-ask"
		}
		_, err := f.client.GetAsk(ctx, reading(reader, &v1.GetAskRequest{SubjectKey: key}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), name)
		_, err = f.client.Wait(ctx, reading(reader, waitForKey(key)))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), name+" (wait)")
	}

	_, err := f.client.GetAsk(ctx, reading("", &v1.GetAskRequest{SubjectKey: "plan-review"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "an anonymous caller is refused")
	_, err = f.client.Wait(ctx, reading("", waitForKey("plan-review")))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestARunNamesExactlyOneOfAnAskIdAndASubjectKey(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	askID := f.ask(t, "run:valid-run-1", "plan", keyed("plan-review"))
	deadline := time.Now().Add(time.Second).UTC().Format(time.RFC3339)
	for name, call := range map[string]func() error{
		"get both": func() error {
			_, err := f.client.GetAsk(ctx, reading("run:valid-run-1", &v1.GetAskRequest{AskId: askID, SubjectKey: "plan-review"}))
			return err
		},
		"get neither": func() error {
			_, err := f.client.GetAsk(ctx, reading("run:valid-run-1", &v1.GetAskRequest{}))
			return err
		},
		"get bad key": func() error {
			_, err := f.client.GetAsk(ctx, reading("run:valid-run-1", &v1.GetAskRequest{SubjectKey: "has spaces"}))
			return err
		},
		"wait both": func() error {
			_, err := f.client.Wait(ctx, reading("run:valid-run-1", &v1.WaitRequest{AskId: askID, SubjectKey: "plan-review", Deadline: deadline}))
			return err
		},
		"wait neither": func() error {
			_, err := f.client.Wait(ctx, reading("run:valid-run-1", &v1.WaitRequest{Deadline: deadline}))
			return err
		},
	} {
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(call()), name)
	}
}

// A producer that changes its answer menu replaces the ask. A wait by key is a
// wait on the subject, so it follows the replacement and returns the answer
// given to the successor.
func TestAWaitBySubjectKeyFollowsAReplacementOfTheAsk(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	original := f.ask(t, "run:valid-run-1+c=thread-1", "first", keyed("plan-review"))

	type outcome struct {
		resp *connect.Response[v1.WaitResponse]
		err  error
	}
	done := make(chan outcome, 1)
	go func() {
		req := reading("run:valid-run-2+c=thread-2", &v1.WaitRequest{SubjectKey: "plan-review", Deadline: time.Now().Add(20 * time.Second).UTC().Format(time.RFC3339)})
		resp, err := f.client.Wait(ctx, req)
		done <- outcome{resp, err}
	}()
	time.Sleep(200 * time.Millisecond)

	response, err := f.client.Ask(ctx, reading("run:valid-run-1+c=thread-1", &v1.AskRequest{
		Question: "Continue with a new menu?", AllowedAnswers: []string{"go", "stop"}, SubjectKey: "plan-review",
		Deadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339), SensitivityLabel: "private", IdempotencyKey: "second",
	}))
	require.NoError(t, err)
	successor := response.Msg.GetAskId()
	require.NotEqual(t, original, successor, "a changed answer menu replaces the ask")
	_, err = f.client.Answer(ctx, reading("owner", &v1.AnswerRequest{AskId: successor, Answer: "go"}))
	require.NoError(t, err)

	select {
	case got := <-done:
		require.NoError(t, got.err)
		require.Equal(t, successor, got.resp.Msg.GetAskId(), "the wait ended on the ask the key now names")
		require.Equal(t, "answered", got.resp.Msg.GetState())
		require.Equal(t, "go", got.resp.Msg.GetAnswer())
	case <-time.After(15 * time.Second):
		t.Fatal("the wait by subject key did not follow the replaced ask")
	}
}
