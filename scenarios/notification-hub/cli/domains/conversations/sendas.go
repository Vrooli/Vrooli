package conversations

// NOTIFICATION_HUB_SEND_AS: how a host program that is not a scenario and not an
// agent run (the portfolio steward, started by a systemd timer) reaches the
// Hub's conversations commands with an identity.
//
// The Hub reads asks back only for a verified caller: an agent run, the owner,
// or a service principal. A host program has none of the first two, so it sends
// as a service principal, "<source>-notification-hub", whose credential the
// operator issues with `scenario-authenticator service-principals sync --apply`.
// With NOTIFICATION_HUB_SEND_AS=<source> the CLI exchanges that credential for a
// token at scenario-authenticator and presents it as the bearer on every
// conversations request (ask, get, list, wait, withdraw, and attachment
// uploads). The Hub then attributes what the process sends to <source> and shows
// it only the asks that source sent.
//
// Which identity a process uses, in order:
//
//  1. A run identity (VROOLI_AGENT_IDENTITY_TOKEN): an agent run is that run, as
//     the Hub treats it, and the switch is not used.
//  2. An explicit token (NOTIFICATION_HUB_API_TOKEN, VROOLI_API_TOKEN or the
//     saved config token) is the operator's override: the switch is not used.
//  3. Otherwise the service principal. There is no fallback from it: a
//     credential that is not issued, a credential store this process cannot
//     read, or a credential that fails to exchange is an error that says which
//     and names the principal. Falling back to an anonymous request would hide a
//     missing or revoked credential, and the Hub refuses an anonymous read.
//
// Without the switch nothing here is used and the CLI behaves as before.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/hubsender"
	"github.com/vrooli/api-core/serviceprincipal"
	"github.com/vrooli/api-core/servicetoken"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// SendAsEnv names the source a host program sends as.
const SendAsEnv = "NOTIFICATION_HUB_SEND_AS"

// sendAsSource matches a scenario-style source name.
var sendAsSource = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// sendAsSettings is everything the switch needs from the process, so a test can
// supply it without a Hub, a credential authority or environment variables.
type sendAsSettings struct {
	// source is the value of the switch, "" when it is not set.
	source string
	// initialBase is the base the generated clients were built with.
	initialBase string
	// apiBase is the Hub's API root as this invocation resolves it (it follows
	// --api-base, which is parsed after the clients are built).
	apiBase func() string
	// explicitToken reports an operator-given token that overrides the switch.
	explicitToken func() bool
	// runIdentity reports that the process carries an agent run identity.
	runIdentity func() bool
	// authorization is the "Bearer <token>" of the source's sender principal.
	authorization func(ctx context.Context) (string, error)
	timeout       time.Duration
}

// hubHTTPClient is the HTTP client of a conversations client: the one cli-core
// builds, or, under NOTIFICATION_HUB_SEND_AS, one that presents the source's
// sender token.
func hubHTTPClient(core *cliapp.ScenarioApp, inner connect.HTTPClient, base string, timeout time.Duration) connect.HTTPClient {
	source := strings.TrimSpace(os.Getenv(SendAsEnv))
	if source == "" || core == nil {
		return inner
	}
	return newSendAsClient(inner, sendAsSettings{
		source:      source,
		initialBase: base,
		apiBase:     core.APIRootBase,
		explicitToken: func() bool {
			return os.Getenv("NOTIFICATION_HUB_API_TOKEN") != "" || os.Getenv("VROOLI_API_TOKEN") != "" || strings.TrimSpace(core.Config.Token) != ""
		},
		runIdentity:   func() bool { return cliutil.DetectIdentity().IsIdentityPresent() },
		authorization: senderAuthorization(source, credentialauthority.ServiceCredential(hubsender.CredentialIdentity(source), hubsender.CredentialField)),
		timeout:       timeout,
	})
}

// credentialError is a failure to get the sender token, with what the credential
// source itself said. The token client reduces its cause to text, so the kind of
// failure (a credential nobody issued, or a store this process cannot read) is
// only known where the credential is read, and is kept here.
type credentialError struct {
	err   error
	cause error
}

func (e *credentialError) Error() string { return e.err.Error() }
func (e *credentialError) Unwrap() error { return e.err }

// senderAuthorization is the "Bearer <token>" source of source's sender
// principal, over a credential source. A failure carries what the credential
// source said, for explainCredentialFailure.
func senderAuthorization(source string, credential servicetoken.CredentialSource) func(context.Context) (string, error) {
	var mu sync.Mutex
	var last error
	tokens := hubsender.NewTokens(source, func(ctx context.Context) (string, error) {
		value, err := credential(ctx)
		mu.Lock()
		last = err
		mu.Unlock()
		return value, err
	})
	return func(ctx context.Context) (string, error) {
		authorization, err := tokens.Authorization(ctx)
		if err == nil {
			return authorization, nil
		}
		mu.Lock()
		defer mu.Unlock()
		return "", &credentialError{err: err, cause: last}
	}
}

func newSendAsClient(inner connect.HTTPClient, settings sendAsSettings) connect.HTTPClient {
	return &sendAsClient{
		inner:    inner,
		settings: settings,
		// A client of its own: the process-wide transport forwards a run identity,
		// which a service principal must never carry.
		http: &http.Client{Timeout: settings.timeout, Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}},
	}
}

type sendAsClient struct {
	inner    connect.HTTPClient
	settings sendAsSettings
	http     *http.Client
}

func (c *sendAsClient) Do(req *http.Request) (*http.Response, error) {
	s := c.settings
	if s.runIdentity() || s.explicitToken() {
		return c.inner.Do(req)
	}
	if !sendAsSource.MatchString(s.source) {
		return nil, fmt.Errorf("%s=%q is not a source name: use the scenario-style name the principal %s-notification-hub is declared for, such as portfolio-steward", SendAsEnv, s.source, "<source>")
	}
	authorization, err := s.authorization(req.Context())
	if err != nil {
		return nil, fmt.Errorf("%s=%s: cannot send as %s: %s (%w)", SendAsEnv, s.source, hubsender.Principal(s.source), explainCredentialFailure(err), err)
	}
	if err := c.address(req); err != nil {
		return nil, err
	}
	req.Header.Del(cliutil.HeaderAgentIdentityToken)
	req.Header.Del(cliutil.HeaderAgentOperationToken)
	req.Header.Set("Authorization", authorization)
	return c.http.Do(req)
}

// address points the request at the Hub this invocation resolves, replacing the
// base the generated client was built with. The base is only known for certain
// after the command line is parsed, which is after the clients exist.
func (c *sendAsClient) address(req *http.Request) error {
	current := strings.TrimRight(strings.TrimSpace(c.settings.apiBase()), "/")
	if current == "" {
		return errors.New("the notification-hub API address is unknown: start the scenario, or pass --api-base")
	}
	target, err := url.Parse(current)
	if err != nil || target.Scheme == "" || target.Host == "" {
		return fmt.Errorf("the notification-hub API address %q is not a URL", current)
	}
	prefix := ""
	if initial, err := url.Parse(c.settings.initialBase); err == nil {
		prefix = strings.TrimRight(initial.Path, "/")
	}
	rel := strings.TrimPrefix(req.URL.Path, prefix)
	req.URL.Scheme, req.URL.Host, req.URL.User, req.Host = target.Scheme, target.Host, nil, ""
	req.URL.Path = strings.TrimRight(target.Path, "/") + "/" + strings.TrimLeft(rel, "/")
	req.URL.RawPath = ""
	return nil
}

// explainCredentialFailure says which kind of failure a sender credential had,
// because the fix differs: an operator issues a missing credential, but a
// credential the process cannot read is a problem with the process's access to
// the host credential store (a systemd user session, for one, may have no
// access to the keyring). Both are the same servicetoken kind, so the cause
// underneath tells them apart.
func explainCredentialFailure(err error) string {
	var failure *credentialError
	cause := error(nil)
	if errors.As(err, &failure) {
		cause = failure.cause
	}
	switch {
	case repoRootMissing(cause):
		return "this process could not find the Vrooli repository root, which the credential store needs to choose its backend, so whether the credential is issued is unknown; set VROOLI_ROOT to the repository path (a timer or systemd unit that starts in $HOME does not inherit it)"
	case errors.Is(cause, credentialauthority.ErrProviderUnavailable), errors.Is(cause, credentialauthority.ErrProviderAbsent):
		return "this process cannot read the host credential store, so whether the credential is issued is unknown; check that the process runs as the account that owns the store and has access to it (a systemd user session may not reach the keyring)"
	case errors.Is(cause, credentialauthority.ErrNotFound), errors.Is(cause, credentialauthority.ErrUnconfigured):
		return "the credential is not issued; the operator runs " + serviceprincipal.IssueCommand
	case errors.Is(err, servicetoken.ErrNotIssued):
		return "no credential could be read; the operator runs " + serviceprincipal.IssueCommand
	case errors.Is(err, servicetoken.ErrRateLimited):
		return rateLimitedText(err)
	case errors.Is(err, servicetoken.ErrExchangeFailed):
		return "scenario-authenticator did not accept the credential; it may have been rotated or revoked, or the service is down"
	}
	return "the sender token could not be obtained"
}

// repoRootMissingText is how the credential store words a failure to find the
// repository root while choosing its backend. The store reports that failure as
// an absent provider and keeps its cause as text only, so the text is the one
// handle there is; tests pin it to the real message.
const repoRootMissingText = "repo root not found"

// repoRootMissing reports that the credential store failed because this process
// could not locate the repository, which is fixed with VROOLI_ROOT and not with
// the keyring.
func repoRootMissing(cause error) bool {
	return cause != nil && strings.Contains(cause.Error(), repoRootMissingText)
}

// rateLimitedText says that scenario-authenticator is limiting this principal
// and when to retry. The credential is fine: nothing is wrong with it, and
// calling it "not accepted" would send the operator to rotate a good one.
func rateLimitedText(err error) string {
	var limited *servicetoken.RateLimitedError
	if errors.As(err, &limited) && limited.RetryAfter > 0 {
		return fmt.Sprintf("scenario-authenticator is rate limiting this principal's exchanges, retry after %ds; the credential is not at fault", int(limited.RetryAfter.Seconds()))
	}
	return "scenario-authenticator is rate limiting this principal's exchanges, retry shortly; the credential is not at fault"
}
