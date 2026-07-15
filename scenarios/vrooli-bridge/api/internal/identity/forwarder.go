// Package identity is the integration boundary for owner sign-in and
// registration. Vrooli Bridge does not own identity — accounts, passwords, and
// JWT signing live in the scenario-authenticator scenario. This package
// forwards a same-origin Login/Register call (made by the control plane's own
// UI) to scenario-authenticator's typed Connect AccountsService, resolving its
// URL by name via api-core/discovery (no env var, no hardcoded port), and
// relays the issued owner JWT back. It owns no credential logic: it stores
// nothing, verifies nothing, mints nothing.
//
// The relayed owner JWT is what the browser then presents on the owner-gated
// fleet RPCs, which internal/auth verifies LOCALLY against the authenticator's
// published RS256 key — this package and internal/auth share the same resolver
// so both find the authenticator by name at call time.
package identity

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"

	"vrooli-bridge/internal/httpc"

	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
)

// DefaultAuthScenario is the scenario whose API issues owner JWTs.
const DefaultAuthScenario = "scenario-authenticator"

// URLResolver resolves a scenario's API base URL by slug. *discovery.Resolver
// satisfies it; tests substitute a static or failing resolver.
type URLResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error)
}

// Typed outcomes the transport edge maps to Connect codes.
var (
	// ErrInvalidCredentials — the authenticator rejected the email/password
	// (or the account is locked).
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrEmailTaken — registration hit an already-registered email.
	ErrEmailTaken = errors.New("email already registered")
	// ErrInvalidInput — malformed/weak input (carries the authenticator's message).
	ErrInvalidInput = errors.New("invalid input")
	// ErrAuthUnavailable — the authenticator could not be reached or misbehaved.
	ErrAuthUnavailable = errors.New("authenticator unavailable")
)

// Owner is the issued owner identity + token relayed back to the caller.
type Owner struct {
	Token  string
	Email  string
	UserID string
}

// Credentials is a sign-in request.
type Credentials struct {
	Email    string
	Password string
}

// Registration is an account-creation request.
type Registration struct {
	Email    string
	Password string
	Username string
}

// Forwarder calls scenario-authenticator's Connect AccountsService and relays
// the result.
type Forwarder struct {
	resolver URLResolver
	doer     httpc.Doer
	scenario string
}

// Config configures the Forwarder.
type Config struct {
	Resolver     URLResolver
	Doer         httpc.Doer
	AuthScenario string
}

// NewForwarder constructs a Forwarder, defaulting the HTTP client and scenario.
func NewForwarder(cfg Config) *Forwarder {
	doer := cfg.Doer
	if doer == nil {
		doer = newDefaultDoer()
	}
	scenario := strings.TrimSpace(cfg.AuthScenario)
	if scenario == "" {
		scenario = DefaultAuthScenario
	}
	return &Forwarder{resolver: cfg.Resolver, doer: doer, scenario: scenario}
}

// Login forwards credentials to scenario-authenticator and returns the owner.
func (f *Forwarder) Login(ctx context.Context, c Credentials) (Owner, error) {
	email := strings.TrimSpace(c.Email)
	if email == "" || c.Password == "" {
		return Owner{}, fmt.Errorf("%w: email and password are required", ErrInvalidInput)
	}
	client, err := f.client(ctx)
	if err != nil {
		return Owner{}, err
	}
	resp, err := client.Login(ctx, connect.NewRequest(&accountsv1.LoginRequest{
		Email: email, Password: c.Password,
	}))
	if err != nil {
		return Owner{}, mapConnectError(err)
	}
	return ownerFrom(resp.Msg.GetAccount(), resp.Msg.GetTokens())
}

// Register creates an account and returns the owner (auto-signed-in).
func (f *Forwarder) Register(ctx context.Context, r Registration) (Owner, error) {
	email := strings.TrimSpace(r.Email)
	if email == "" || r.Password == "" {
		return Owner{}, fmt.Errorf("%w: email and password are required", ErrInvalidInput)
	}
	client, err := f.client(ctx)
	if err != nil {
		return Owner{}, err
	}
	resp, err := client.Register(ctx, connect.NewRequest(&accountsv1.RegisterRequest{
		Email: email, Password: r.Password, Username: strings.TrimSpace(r.Username),
	}))
	if err != nil {
		return Owner{}, mapConnectError(err)
	}
	return ownerFrom(resp.Msg.GetAccount(), resp.Msg.GetTokens())
}

// client builds an AccountsService Connect client against the discovery-resolved
// authenticator base URL. The httpc.Doer satisfies connect.HTTPClient directly.
func (f *Forwarder) client(ctx context.Context) (accountsconnect.AccountsServiceClient, error) {
	base, err := f.baseURL(ctx)
	if err != nil {
		return nil, err
	}
	return accountsconnect.NewAccountsServiceClient(f.doer, base), nil
}

func (f *Forwarder) baseURL(ctx context.Context) (string, error) {
	if f.resolver == nil {
		return "", fmt.Errorf("%w: no authenticator resolver configured", ErrAuthUnavailable)
	}
	base, err := f.resolver.ResolveScenarioURLDefault(ctx, f.scenario)
	if err != nil {
		return "", fmt.Errorf("%w: resolve %s url: %v", ErrAuthUnavailable, f.scenario, err)
	}
	return strings.TrimRight(base, "/"), nil
}

// ownerFrom projects the Connect account + token pair onto an Owner, treating a
// missing access token as an unavailable authenticator (it never validly omits
// the token on success).
func ownerFrom(account *accountsv1.Account, tokens *accountsv1.TokenPair) (Owner, error) {
	if tokens == nil || tokens.GetAccessToken() == "" {
		return Owner{}, fmt.Errorf("%w: authenticator returned no token", ErrAuthUnavailable)
	}
	// The authenticator also issues a refresh token; the bridge deliberately
	// drops it — there is no refresh flow, so expiry means signing in again.
	o := Owner{Token: tokens.GetAccessToken()}
	if account != nil {
		o.Email = account.GetEmail()
		o.UserID = account.GetId()
	}
	return o, nil
}

// mapConnectError translates the authenticator's Connect codes into the control
// plane's typed outcomes. Unauthenticated and PermissionDenied (locked) both
// collapse to ErrInvalidCredentials so the control plane never leaks which one
// occurred.
func mapConnectError(err error) error {
	switch connect.CodeOf(err) {
	case connect.CodeAlreadyExists:
		return ErrEmailTaken
	case connect.CodeUnauthenticated, connect.CodePermissionDenied:
		return ErrInvalidCredentials
	case connect.CodeInvalidArgument:
		return fmt.Errorf("%w: %s", ErrInvalidInput, connectMessage(err))
	default:
		return fmt.Errorf("%w: %v", ErrAuthUnavailable, err)
	}
}

// connectMessage extracts the human message from a *connect.Error, falling back
// to the error string.
func connectMessage(err error) string {
	var ce *connect.Error
	if errors.As(err, &ce) {
		return ce.Message()
	}
	return err.Error()
}

func newDefaultDoer() httpc.Doer {
	return &http.Client{Timeout: 10 * time.Second}
}
