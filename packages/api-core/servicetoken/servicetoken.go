// Package servicetoken is the one client a scenario uses to act as an
// operator-issued scenario-authenticator service principal.
//
// The operator issues the principal's credential into the credential authority
// (an agent run cannot: issuing needs a verified human). The scenario reads the
// credential, exchanges it at scenario-authenticator for a short-lived access
// token, caches the token until shortly before it expires, and presents it as a
// bearer. The credential is read again on every exchange, so rotation and
// revocation take effect without a restart. A token whose identity is not
// exactly the one the scenario expects is never presented, so a mis-issued or
// swapped credential cannot speak for the scenario.
//
// The cache is in memory, per Tokens, and that is deliberate. A long-running
// scenario exchanges about once per token lifetime. A short-lived CLI process
// exchanges once per request it makes; sharing a bearer between processes would
// need a file of live bearers under the runtime state directory (mode 0600,
// keyed by principal, audience and a hash of the credential, dropped on 401),
// and that file would add a secret at rest, a cross-process race and a
// revocation lag to save a call that scenario-authenticator budgets per
// principal (accounts.ExchangeLimits), so one caller's calls cannot starve
// another's. A caller that exchanges too often is told so: ErrRateLimited.
//
// askreader builds on it for the Hub's ask reads; a scenario that calls another
// scenario as itself (Web Console, say) uses it directly.
package servicetoken

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/serviceprincipal"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
)

const (
	// DefaultRefreshMargin is how long before expiry a cached token is
	// exchanged again, unless Config.RefreshMargin says otherwise.
	DefaultRefreshMargin = 30 * time.Second
	exchangeTimeout      = 10 * time.Second
)

var (
	// ErrNotIssued means no credential could be read for the principal: the
	// credential authority holds none (the operator issues it), or it could not
	// be read, in which case the reason follows.
	ErrNotIssued = errors.New("the service credential is not issued")
	// ErrExchangeFailed means scenario-authenticator did not exchange the
	// credential: it is unreachable, the principal is not configured, or the
	// credential was rotated or revoked.
	ErrExchangeFailed = errors.New("scenario-authenticator did not exchange the service credential")
	// ErrRateLimited means scenario-authenticator refused the exchange because
	// this principal has used its budget, not because the credential is wrong.
	// Such a failure is also an ErrExchangeFailed. errors.As to *RateLimitedError
	// says when to retry.
	ErrRateLimited = errors.New("scenario-authenticator rate limited the service credential exchange")
	// ErrNoUsableToken means the exchange returned no token, or one that is
	// already (or about to be) expired.
	ErrNoUsableToken = errors.New("scenario-authenticator returned no usable service token")
	// ErrIdentityMismatch means the exchanged token is not exactly the expected
	// identity, so it is never presented.
	ErrIdentityMismatch = errors.New("scenario-authenticator returned an unexpected service identity")
	// ErrNotConfigured means the client has no credential source or exchanger.
	ErrNotConfigured = errors.New("the service identity is not configured")
)

// RateLimitedError is an exchange that scenario-authenticator refused because
// the principal's budget is spent. An Exchanger returns it (AuthenticatorExchanger
// does) for Tokens to report as ErrRateLimited.
type RateLimitedError struct {
	// RetryAfter is how long to wait; zero when the service did not say.
	RetryAfter time.Duration
}

func (e *RateLimitedError) Error() string {
	if e.RetryAfter <= 0 {
		return "rate limited, retry later"
	}
	return fmt.Sprintf("rate limited, retry after %ds", int(e.RetryAfter.Round(time.Second).Seconds()))
}

// Is makes a RateLimitedError an ErrRateLimited.
func (e *RateLimitedError) Is(target error) bool { return target == ErrRateLimited }

// rateLimitedFailure is a rate-limited exchange worded by the caller's Describe
// (or the default), that is still an ErrExchangeFailed and an ErrRateLimited.
type rateLimitedFailure struct {
	text    string
	failure error
	limited *RateLimitedError
}

func (r *rateLimitedFailure) Error() string   { return r.text }
func (r *rateLimitedFailure) Unwrap() []error { return []error{r.failure, r.limited} }

// Kind returns err as a failure of the given kind: it keeps err's text and its
// chain, and errors.Is(result, kind) holds. Config.Describe uses it to give a
// failure a scenario's own wording without losing the kind a caller tests.
func Kind(kind, err error) error { return &kinded{kind: kind, err: err} }

type kinded struct{ kind, err error }

func (k *kinded) Error() string        { return k.err.Error() }
func (k *kinded) Unwrap() error        { return k.err }
func (k *kinded) Is(target error) bool { return target == k.kind }

// CredentialSource reads the principal's credential from the credential
// authority, on every exchange. An error, or an empty value, means not issued.
type CredentialSource func(ctx context.Context) (string, error)

// Exchanged is what scenario-authenticator returns for a service credential.
// AccessToken stays in memory and is never logged.
type Exchanged struct {
	AccessToken string
	ExpiresAt   time.Time
	Subject     string
	Audience    string
	Scopes      []string
}

// Exchanger trades a credential for an access token.
type Exchanger func(ctx context.Context, credential string) (Exchanged, error)

// Identity is the exact identity a token must carry to be presented: its
// subject, its audience, and its complete scope set (order is irrelevant).
type Identity struct {
	Subject  string
	Audience string
	Scopes   []string
}

func (i Identity) matches(e Exchanged) bool {
	want, got := slices.Clone(i.Scopes), slices.Clone(e.Scopes)
	slices.Sort(want)
	slices.Sort(got)
	return i.Subject != "" && e.Subject == i.Subject && e.Audience == i.Audience && slices.Equal(want, got)
}

// ExpectIdentity is the Identity for Config.Expect.
func ExpectIdentity(subject, audience string, scopes ...string) *Identity {
	return &Identity{Subject: subject, Audience: audience, Scopes: scopes}
}

// Config describes one principal.
type Config struct {
	// Name labels the principal in the default error texts, for example
	// "research-desk ask-reader".
	Name string
	// Credential reads the credential from the credential authority.
	Credential CredentialSource
	// Expect is the identity the exchanged token must carry. Nil presents
	// whatever scenario-authenticator returns, for a principal whose peer
	// verifies the token itself.
	Expect *Identity
	// IssueHint says how the credential is issued; the default not-issued text
	// ends with it. Empty means "the operator runs " + serviceprincipal.IssueCommand.
	IssueHint string
	// RefreshMargin is how long before expiry a cached token is exchanged
	// again. Zero means DefaultRefreshMargin.
	RefreshMargin time.Duration
	// MinRemaining is how much life a fresh token must have to be used at all;
	// a token with less is ErrNoUsableToken. Zero accepts any unexpired token.
	MinRemaining time.Duration
	// Describe words a failure. It receives the kind (ErrNotIssued,
	// ErrExchangeFailed, ErrNoUsableToken, ErrIdentityMismatch or
	// ErrNotConfigured) and the underlying cause, which is nil when there is
	// none, and returns the error to report; wrap it with Kind so the kind stays
	// testable. Nil uses the default text, which names Config.Name and
	// Config.IssueHint and carries the cause.
	Describe func(kind, cause error) error
}

// Tokens supplies the bearer token of one service principal. It is safe for
// concurrent use.
type Tokens struct {
	config   Config
	exchange Exchanger
	now      func() time.Time

	mu      sync.Mutex
	token   string
	expires time.Time
}

// Option customizes Tokens; the defaults are production's.
type Option func(*Tokens)

// WithExchanger replaces the exchange at the discovered scenario-authenticator.
func WithExchanger(exchange Exchanger) Option { return func(t *Tokens) { t.exchange = exchange } }

// WithClock replaces the clock.
func WithClock(now func() time.Time) Option { return func(t *Tokens) { t.now = now } }

// New builds the token client. Without WithExchanger it exchanges at the
// scenario-authenticator that discovery resolves on each exchange.
func New(config Config, options ...Option) *Tokens {
	t := &Tokens{config: config, now: time.Now}
	for _, option := range options {
		option(t)
	}
	if t.exchange == nil {
		t.exchange = AuthenticatorExchanger(&http.Client{Timeout: exchangeTimeout}, func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "scenario-authenticator")
		})
	}
	return t
}

func (t *Tokens) refreshMargin() time.Duration {
	if t.config.RefreshMargin > 0 {
		return t.config.RefreshMargin
	}
	return DefaultRefreshMargin
}

// fail words a failure of the given kind.
func (t *Tokens) fail(kind, cause error) error {
	if t.config.Describe != nil {
		return t.config.Describe(kind, cause)
	}
	name := "service credential"
	if t.config.Name != "" {
		name = t.config.Name + " credential"
	}
	switch kind {
	case ErrNotIssued:
		hint := t.config.IssueHint
		if hint == "" {
			hint = "the operator runs " + serviceprincipal.IssueCommand
		}
		text := fmt.Sprintf("%s: %s; %s", name, kind, hint)
		if cause != nil {
			text += " (" + cause.Error() + ")"
		}
		return Kind(kind, errors.New(text))
	case ErrExchangeFailed:
		return Kind(kind, fmt.Errorf("%s: %w (%v)", name, kind, cause))
	}
	return fmt.Errorf("%s: %w", name, kind)
}

// failRateLimited words a rate-limited exchange. A caller's Describe words it
// as an exchange failure, as before, and the failure is also an ErrRateLimited;
// the default text says "rate limited", never that the credential was refused.
func (t *Tokens) failRateLimited(limited *RateLimitedError) error {
	failure := t.fail(ErrExchangeFailed, limited)
	text := failure.Error()
	if t.config.Describe == nil {
		name := "service credential"
		if t.config.Name != "" {
			name = t.config.Name + " credential"
		}
		text = fmt.Sprintf("%s: %s", name, limited)
	}
	return &rateLimitedFailure{text: text, failure: failure, limited: limited}
}

// Token returns a current access token. Its errors are of kind ErrNotIssued,
// ErrExchangeFailed, ErrNoUsableToken, ErrIdentityMismatch or ErrNotConfigured
// (test with errors.Is) and never carry the credential.
func (t *Tokens) Token(ctx context.Context) (string, error) {
	if t == nil {
		return "", ErrNotConfigured
	}
	if t.config.Credential == nil || t.exchange == nil || (t.config.Expect != nil && t.config.Expect.Subject == "") {
		return "", t.fail(ErrNotConfigured, nil)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	margin := t.refreshMargin()
	if t.token != "" && t.now().Add(margin).Before(t.expires) {
		return t.token, nil
	}
	t.token, t.expires = "", time.Time{}
	credential, err := t.config.Credential(ctx)
	credential = strings.TrimSpace(credential)
	if err != nil || credential == "" {
		return "", t.fail(ErrNotIssued, err)
	}
	exchanged, err := t.exchange(ctx, credential)
	var limited *RateLimitedError
	if errors.As(err, &limited) {
		return "", t.failRateLimited(limited)
	}
	if err != nil {
		return "", t.fail(ErrExchangeFailed, errors.New(strings.ReplaceAll(err.Error(), credential, "[credential]")))
	}
	if strings.TrimSpace(exchanged.AccessToken) == "" || !t.now().Add(t.config.MinRemaining).Before(exchanged.ExpiresAt) {
		return "", t.fail(ErrNoUsableToken, nil)
	}
	if t.config.Expect != nil && !t.config.Expect.matches(exchanged) {
		return "", t.fail(ErrIdentityMismatch, nil)
	}
	t.token, t.expires = exchanged.AccessToken, exchanged.ExpiresAt
	// However a caller presents it, a request carrying this token is the service
	// itself, so the forwarding transport never adds a caller's agent identity.
	provenance.MarkServiceBearer(t.token, t.expires)
	return t.token, nil
}

// Authorization is the Authorization header value: "Bearer <token>".
func (t *Tokens) Authorization(ctx context.Context) (string, error) {
	token, err := t.Token(ctx)
	if err != nil {
		return "", err
	}
	return "Bearer " + token, nil
}

// Interceptor sets the Authorization header of every call it wraps. A call
// whose token cannot be had fails before it is sent, with the reason.
func (t *Tokens) Interceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// The call is the service's own, under its token. When it runs inside
			// a request an agent run made, that run's identity must not ride along:
			// a receiver that resolves an agent before a service token would answer
			// as if the run had asked, and one that intersects them refuses both.
			ctx = provenance.WithoutCallerIdentity(ctx)
			authorization, err := t.Authorization(ctx)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, err)
			}
			req.Header().Set("Authorization", authorization)
			return next(ctx, req)
		}
	}
}

// AuthenticatorExchanger exchanges at the scenario-authenticator whose base URL
// authenticatorURL returns for each call.
func AuthenticatorExchanger(httpClient connect.HTTPClient, authenticatorURL func(context.Context) (string, error)) Exchanger {
	return func(ctx context.Context, credential string) (Exchanged, error) {
		// The exchange is the service proving itself; no caller rides along.
		ctx = provenance.WithoutCallerIdentity(ctx)
		base, err := authenticatorURL(ctx)
		if err != nil {
			return Exchanged{}, err
		}
		client := accountsconnect.NewAccountsServiceClient(httpClient, strings.TrimRight(base, "/"))
		response, err := client.ExchangeServiceCredential(ctx, connect.NewRequest(&accountsv1.ExchangeServiceCredentialRequest{Credential: credential}))
		if err != nil {
			if connect.CodeOf(err) == connect.CodeResourceExhausted {
				return Exchanged{}, &RateLimitedError{RetryAfter: retryAfter(err)}
			}
			return Exchanged{}, err
		}
		// A response with no (or an invalid) expiry has no usable token: the zero
		// ExpiresAt makes Token report ErrNoUsableToken.
		var expires time.Time
		if at := response.Msg.GetAccessTokenExpiresAt(); at != nil && at.CheckValid() == nil {
			expires = at.AsTime()
		}
		return Exchanged{
			AccessToken: response.Msg.GetAccessToken(),
			ExpiresAt:   expires,
			Subject:     response.Msg.GetSubject(),
			Audience:    response.Msg.GetAudience(),
			Scopes:      append([]string(nil), response.Msg.GetScopes()...),
		}, nil
	}
}

// retryAfter reads the Retry-After seconds scenario-authenticator sent with a
// refusal; zero when there are none.
func retryAfter(err error) time.Duration {
	var failure *connect.Error
	if !errors.As(err, &failure) {
		return 0
	}
	seconds, convErr := strconv.Atoi(strings.TrimSpace(failure.Meta().Get("Retry-After")))
	if convErr != nil || seconds < 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

// DescribeWithCause is a Config.Describe whose texts name the scenario and carry
// the underlying cause: "<name> service credential unavailable: <cause>" when
// nothing could be read, "<name> service credential exchange: <cause>" when the
// exchange failed, and one fixed line for an exchange that returned nothing
// usable. For a scenario whose operators read these in logs.
func DescribeWithCause(name string) func(kind, cause error) error {
	return func(kind, cause error) error {
		switch kind {
		case ErrNotIssued:
			if cause == nil {
				cause = errors.New("no credential issued")
			}
			return Kind(kind, fmt.Errorf("%s service credential unavailable: %w", name, cause))
		case ErrExchangeFailed:
			return Kind(kind, fmt.Errorf("%s service credential exchange: %w", name, cause))
		case ErrNoUsableToken:
			return Kind(kind, fmt.Errorf("%s service credential exchange returned no usable token", name))
		case ErrIdentityMismatch:
			return Kind(kind, fmt.Errorf("%s service credential exchange returned an unexpected identity", name))
		}
		return Kind(kind, fmt.Errorf("%s service identity is unavailable", name))
	}
}
