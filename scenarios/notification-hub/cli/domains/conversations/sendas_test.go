package conversations

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/serviceprincipal"
	"github.com/vrooli/api-core/servicetoken"
	"github.com/vrooli/cli-core/cliutil"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
)

// sendAsHub is a Hub that records the headers of every request it receives.
type sendAsHub struct {
	stub *hubStub
	mu   sync.Mutex
	seen []http.Header
}

func (h *sendAsHub) requests() []http.Header {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]http.Header(nil), h.seen...)
}

// recordingInner stands for the client cli-core would have built, the one used
// when the switch is not in play. It records that it was used.
type recordingInner struct {
	mu    sync.Mutex
	calls int
}

func (r *recordingInner) Do(*http.Request) (*http.Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	return nil, errors.New("the default client was used")
}

func (r *recordingInner) used() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

// sendAsFixture is a conversations service whose client is a sendAsClient aimed
// at a recording Hub. change adjusts the process inputs a test cares about.
func sendAsFixture(t *testing.T, change func(*sendAsSettings)) (*service, *sendAsHub, *recordingInner) {
	t.Helper()
	hub := &sendAsHub{stub: &hubStub{}}
	mux := http.NewServeMux()
	path, handler := connectv1.NewConversationsServiceHandler(hub.stub)
	mux.Handle(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.mu.Lock()
		hub.seen = append(hub.seen, r.Header.Clone())
		hub.mu.Unlock()
		handler.ServeHTTP(w, r)
	}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	settings := sendAsSettings{
		source:        "portfolio-steward",
		initialBase:   "http://stale.invalid:1/prefix",
		apiBase:       func() string { return server.URL },
		explicitToken: func() bool { return false },
		runIdentity:   func() bool { return false },
		authorization: func(context.Context) (string, error) { return "Bearer sender-token", nil },
		timeout:       5 * time.Second,
	}
	if change != nil {
		change(&settings)
	}
	inner := &recordingInner{}
	client := connectv1.NewConversationsServiceClient(newSendAsClient(inner, settings), settings.initialBase)
	return &service{client: client, waitClient: client, uploadClient: client, now: time.Now}, hub, inner
}

func TestUnderTheSwitchEveryConversationsCallPresentsTheSenderTokenAndNoRunIdentity(t *testing.T) {
	svc, hub, inner := sendAsFixture(t, nil)
	for name, run := range map[string]func() error{
		"ask": func() error { _, _, err := runCommand(t, svc, "ask", minimalAsk()...); return err },
		"list": func() error {
			_, _, err := runCommand(t, svc, "list")
			return err
		},
		"get": func() error {
			_, _, err := runCommand(t, svc, "get", "--subject-key", "k")
			return err
		},
		"withdraw": func() error {
			_, _, err := runCommand(t, svc, "withdraw", "--subject-key", "k", "--reason", "done")
			return err
		},
	} {
		before := len(hub.requests())
		_ = run() // the stub answers some of these with not-found; only the request matters
		after := hub.requests()
		require.Greater(t, len(after), before, "%s reached the Hub", name)
		last := after[len(after)-1]
		require.Equal(t, "Bearer sender-token", last.Get("Authorization"), name)
		require.Empty(t, last.Get(cliutil.HeaderAgentIdentityToken), "%s presents no run identity", name)
		require.Empty(t, last.Get(cliutil.HeaderAgentOperationToken), name)
	}
	require.Zero(t, inner.used(), "the default client is not used under the switch")
}

func TestTheSwitchSendsToTheHubThisInvocationResolvesNotTheOneTheClientWasBuiltFor(t *testing.T) {
	svc, hub, _ := sendAsFixture(t, nil) // built for http://stale.invalid:1/prefix, resolves to the test Hub
	_, _, err := runCommand(t, svc, "list")
	require.NoError(t, err)
	require.Len(t, hub.requests(), 1)
}

func TestAnExplicitTokenOverridesTheSwitch(t *testing.T) {
	svc, hub, inner := sendAsFixture(t, func(s *sendAsSettings) {
		s.explicitToken = func() bool { return true }
		s.authorization = func(context.Context) (string, error) { return "", errors.New("must not be asked") }
	})
	_, _, err := runCommand(t, svc, "list")
	require.Error(t, err, "the default client in this fixture always fails")
	require.Equal(t, 1, inner.used(), "the operator's token goes through the ordinary client")
	require.Empty(t, hub.requests())
}

func TestARunIdentityWinsOverTheSwitchSoAnAgentRunIsStillThatRun(t *testing.T) {
	svc, hub, inner := sendAsFixture(t, func(s *sendAsSettings) {
		s.runIdentity = func() bool { return true }
		s.authorization = func(context.Context) (string, error) { return "", errors.New("must not be asked") }
	})
	_, _, err := runCommand(t, svc, "list")
	require.Error(t, err, "the default client in this fixture always fails")
	require.Equal(t, 1, inner.used(), "a run identity goes through the ordinary client, which presents it")
	require.Empty(t, hub.requests())
}

func TestACredentialThatIsNotIssuedIsALoudErrorNamingThePrincipalAndNeverAnAnonymousRequest(t *testing.T) {
	svc, hub, inner := sendAsFixture(t, func(s *sendAsSettings) {
		s.authorization = func(context.Context) (string, error) {
			return "", fmt.Errorf("%w: the operator runs %s", servicetoken.ErrNotIssued, serviceprincipal.IssueCommand)
		}
	})
	for _, command := range []string{"ask", "list", "get", "withdraw"} {
		args := map[string][]string{
			"ask": minimalAsk(), "list": nil, "get": {"--subject-key", "k"}, "withdraw": {"--subject-key", "k", "--reason", "x"},
		}[command]
		_, _, err := runCommand(t, svc, command, args...)
		require.Error(t, err, command)
		for _, want := range []string{SendAsEnv + "=portfolio-steward", "portfolio-steward-notification-hub", serviceprincipal.IssueCommand} {
			require.ErrorContains(t, err, want, command)
		}
	}
	require.Empty(t, hub.requests(), "an unissued credential never degrades to an anonymous request")
	require.Zero(t, inner.used(), "nor to the ordinary client")
}

func TestAStoreThisProcessCannotReadIsToldApartFromACredentialThatIsNotIssued(t *testing.T) {
	// The causes come from the credential authority and survive servicetoken's
	// own wrapping, so the real token client tells the two apart.
	for name, tc := range map[string]struct {
		cause error
		want  string
		not   string
	}{
		"missing":      {credentialauthority.ErrNotFound, "the credential is not issued; the operator runs", "cannot read the host credential store"},
		"unconfigured": {credentialauthority.ErrUnconfigured, "the credential is not issued; the operator runs", "cannot read the host credential store"},
		"unavailable":  {credentialauthority.ErrProviderUnavailable, "cannot read the host credential store", "the credential is not issued"},
		"absent":       {credentialauthority.ErrProviderAbsent, "cannot read the host credential store", "the credential is not issued"},
	} {
		authorization := senderAuthorization("portfolio-steward", func(context.Context) (string, error) { return "", tc.cause })
		svc, hub, _ := sendAsFixture(t, func(s *sendAsSettings) { s.authorization = authorization })
		_, _, err := runCommand(t, svc, "list")
		require.Error(t, err, name)
		require.ErrorContains(t, err, tc.want, name)
		require.NotContains(t, err.Error(), tc.not, name)
		require.ErrorContains(t, err, "portfolio-steward-notification-hub", name)
		require.Empty(t, hub.requests(), name)
	}
}

// A process that cannot find the repository root (a timer or systemd unit that
// starts in $HOME) gets the credential store's absent-provider answer, whose
// keyring wording sends the operator to the wrong place. The message names the
// real cause and fix. The cause below is the store's real text for it.
func TestAMissingRepoRootIsNamedNotBlamedOnTheKeyring(t *testing.T) {
	cause := fmt.Errorf("resolve credential x: %w", realProviderAbsent("credential provider is not implemented on this host: operating-system secure storage is not implemented on this host: resolve credential backend selection path: repo-contract not_found: repo root not found (/home/someone/.vrooli/bin)"))
	authorization := senderAuthorization("portfolio-steward", func(context.Context) (string, error) { return "", cause })
	svc, hub, _ := sendAsFixture(t, func(s *sendAsSettings) { s.authorization = authorization })
	_, _, err := runCommand(t, svc, "list")
	require.ErrorContains(t, err, "could not find the Vrooli repository root")
	require.ErrorContains(t, err, "VROOLI_ROOT")
	require.NotContains(t, err.Error(), "keyring")
	require.Empty(t, hub.requests())
}

// realProviderAbsent is an absent-provider error carrying text as its cause,
// as the credential store builds it.
func realProviderAbsent(text string) error {
	return fmt.Errorf("%w: %s", credentialauthority.ErrProviderAbsent, text)
}

func TestACredentialThatFailsToExchangeIsARealErrorToo(t *testing.T) {
	svc, hub, _ := sendAsFixture(t, func(s *sendAsSettings) {
		s.authorization = func(context.Context) (string, error) {
			return "", fmt.Errorf("%w: credential revoked", servicetoken.ErrExchangeFailed)
		}
	})
	_, _, err := runCommand(t, svc, "list")
	require.ErrorContains(t, err, "credential revoked")
	require.ErrorContains(t, err, "portfolio-steward-notification-hub")
	require.Empty(t, hub.requests())
}

// A principal that scenario-authenticator is rate limiting has a good credential.
// The error says so and when to retry; it must not send the operator to rotate it.
func TestARateLimitedExchangeSaysRateLimitedNotCredentialRefused(t *testing.T) {
	authorization := func(context.Context) (string, error) {
		return "", fmt.Errorf("portfolio-steward credential: %w", &servicetoken.RateLimitedError{RetryAfter: 42 * time.Second})
	}
	svc, hub, _ := sendAsFixture(t, func(s *sendAsSettings) { s.authorization = authorization })
	_, _, err := runCommand(t, svc, "list")
	require.ErrorContains(t, err, "rate limiting this principal's exchanges, retry after 42s")
	require.ErrorContains(t, err, "portfolio-steward-notification-hub")
	for _, wrong := range []string{"did not accept", "rotated", "revoked", "not issued"} {
		require.NotContains(t, err.Error(), wrong)
	}
	require.Empty(t, hub.requests())
}

func TestTheSwitchNeedsASourceNameNotAPrincipalOrAPath(t *testing.T) {
	for _, source := range []string{"Portfolio Steward", "portfolio-steward-notification-hub/x", "../x", "-x", "x-", ""} {
		svc, hub, _ := sendAsFixture(t, func(s *sendAsSettings) { s.source = source })
		_, _, err := runCommand(t, svc, "list")
		require.ErrorContains(t, err, "is not a source name", "%q", source)
		require.Empty(t, hub.requests())
	}
}

func TestWithoutTheSwitchTheOrdinaryClientIsUsedUnchanged(t *testing.T) {
	t.Setenv(SendAsEnv, "")
	inner := &recordingInner{}
	require.Same(t, connect.HTTPClient(inner), hubHTTPClient(nil, inner, "http://hub", time.Second))
}

func TestARefusedSenderTokenSaysWhichPrincipalAndHowToFixIt(t *testing.T) {
	t.Setenv(SendAsEnv, "portfolio-steward")
	t.Setenv(cliutil.EnvIdentityToken, "")
	t.Setenv("NOTIFICATION_HUB_API_TOKEN", "")
	stub := &hubStub{readErr: connect.NewError(connect.CodeUnauthenticated, errors.New("no identity"))}
	_, _, err := runCommand(t, newTestService(t, stub), "list")
	require.ErrorContains(t, err, "sender token of portfolio-steward-notification-hub")
	require.ErrorContains(t, err, "service-principals sync --apply")
	require.NotContains(t, err.Error(), "presented no verified identity", "the generic shell advice does not fit a host program")
}
