package servicetoken

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var expected = ExpectIdentity("research-desk", "scenario-authenticator:web-console", "web-console:read", "web-console:write")

// fixture is a Tokens over a scripted credential and exchanger.
type fixture struct {
	now        time.Time
	credential string
	credErr    error
	exchanges  atomic.Int32
	seen       []string
	result     Exchanged
	exchangeEr error
	mu         sync.Mutex
}

func newFixture() *fixture {
	f := &fixture{now: time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC), credential: "cred-1"}
	f.result = Exchanged{AccessToken: "tok-1", ExpiresAt: f.now.Add(10 * time.Minute), Subject: expected.Subject, Audience: expected.Audience, Scopes: []string{"web-console:write", "web-console:read"}}
	return f
}

func (f *fixture) tokens() *Tokens {
	return New(Config{Name: "research-desk web-console", Credential: func(context.Context) (string, error) { return f.credential, f.credErr }, Expect: expected, IssueHint: "the operator runs issue.sh"},
		WithClock(func() time.Time { return f.now }),
		WithExchanger(func(_ context.Context, credential string) (Exchanged, error) {
			f.exchanges.Add(1)
			f.mu.Lock()
			f.seen = append(f.seen, credential)
			f.mu.Unlock()
			return f.result, f.exchangeEr
		}))
}

func TestATokenIsCachedUntilJustBeforeItExpires(t *testing.T) {
	f := newFixture()
	tokens := f.tokens()
	for range 3 {
		got, err := tokens.Token(context.Background())
		if err != nil || got != "tok-1" {
			t.Fatalf("Token = %q, %v", got, err)
		}
	}
	if f.exchanges.Load() != 1 {
		t.Fatalf("exchanges = %d, want 1 while the token is current", f.exchanges.Load())
	}
	f.now = f.now.Add(10*time.Minute - DefaultRefreshMargin + time.Second)
	f.result.AccessToken, f.result.ExpiresAt = "tok-2", f.now.Add(10*time.Minute)
	got, err := tokens.Token(context.Background())
	if err != nil || got != "tok-2" || f.exchanges.Load() != 2 {
		t.Fatalf("after the refresh margin: %q, %v, exchanges %d", got, err, f.exchanges.Load())
	}
}

func TestTheCredentialIsReadAgainOnEveryExchangeSoRotationNeedsNoRestart(t *testing.T) {
	f := newFixture()
	tokens := f.tokens()
	if _, err := tokens.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(time.Hour)
	f.credential = "cred-2"
	f.result.ExpiresAt = f.now.Add(10 * time.Minute)
	if _, err := tokens.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Join(f.seen, ",") != "cred-1,cred-2" {
		t.Fatalf("credentials exchanged = %v, want the rotated one second", f.seen)
	}
}

func TestAMissingCredentialIsReportedAsNotIssuedAndSaysHowItIsIssued(t *testing.T) {
	for name, set := range map[string]func(*fixture){
		"read error": func(f *fixture) { f.credErr = errors.New("no such entry") },
		"empty":      func(f *fixture) { f.credential = "  " },
	} {
		f := newFixture()
		set(f)
		_, err := f.tokens().Token(context.Background())
		if !errors.Is(err, ErrNotIssued) || !strings.Contains(err.Error(), "the operator runs issue.sh") || !strings.Contains(err.Error(), "research-desk web-console") {
			t.Fatalf("%s: err = %v, want ErrNotIssued naming the principal and the issuing command", name, err)
		}
		if f.exchanges.Load() != 0 {
			t.Fatalf("%s: nothing may be exchanged without a credential", name)
		}
	}
}

func TestTheDefaultNotIssuedHintIsTheOneOperatorCommand(t *testing.T) {
	tokens := New(Config{Name: "x", Credential: func(context.Context) (string, error) { return "", nil }, Expect: expected})
	if _, err := tokens.Token(context.Background()); err == nil || !strings.Contains(err.Error(), "the operator runs scenario-authenticator service-principals sync --apply") {
		t.Fatalf("err = %v", err)
	}
}

func TestAFailedExchangeIsReportedWithoutTheCredentialAndNeverCached(t *testing.T) {
	f := newFixture()
	f.credential = "super-secret-credential"
	f.exchangeEr = errors.New("connection refused for super-secret-credential")
	tokens := f.tokens()
	_, err := tokens.Token(context.Background())
	if !errors.Is(err, ErrExchangeFailed) || strings.Contains(err.Error(), "super-secret") || !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("err = %v, want the reason without the credential", err)
	}
	f.exchangeEr = nil
	if got, err := tokens.Token(context.Background()); err != nil || got != "tok-1" {
		t.Fatalf("a later exchange must recover: %q, %v", got, err)
	}
}

func TestATokenThatIsNotExactlyTheExpectedIdentityIsNeverUsed(t *testing.T) {
	cases := map[string]func(*Exchanged){
		"another subject":  func(e *Exchanged) { e.Subject = "agent-manager" },
		"another audience": func(e *Exchanged) { e.Audience = "scenario-authenticator:agent-manager" },
		"no scope":         func(e *Exchanged) { e.Scopes = nil },
		"missing scope":    func(e *Exchanged) { e.Scopes = []string{"web-console:read"} },
		"extra scope":      func(e *Exchanged) { e.Scopes = append(e.Scopes, "agent-manager:write") },
		"other scope":      func(e *Exchanged) { e.Scopes = []string{"web-console:read", "web-console:admin"} },
	}
	for name, mutate := range cases {
		f := newFixture()
		mutate(&f.result)
		got, err := f.tokens().Token(context.Background())
		if !errors.Is(err, ErrIdentityMismatch) || got != "" {
			t.Fatalf("%s: Token = %q, %v, want ErrIdentityMismatch", name, got, err)
		}
	}
}

func TestAnUnconfiguredTokenClientRefuses(t *testing.T) {
	var none *Tokens
	for name, tokens := range map[string]*Tokens{
		"nil":           none,
		"no subject":    New(Config{Credential: func(context.Context) (string, error) { return "c", nil }, Expect: &Identity{}}),
		"no credential": New(Config{Expect: expected}),
		"no exchanger":  {config: Config{Credential: func(context.Context) (string, error) { return "c", nil }}},
	} {
		if _, err := tokens.Token(context.Background()); err == nil {
			t.Fatalf("%s: want an error", name)
		}
	}
}

func TestConcurrentCallersShareOneExchange(t *testing.T) {
	f := newFixture()
	tokens := f.tokens()
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := tokens.Token(context.Background()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if f.exchanges.Load() != 1 {
		t.Fatalf("exchanges = %d, want 1", f.exchanges.Load())
	}
}

// echo records the Authorization header a call carried.
type echo struct{ got atomic.Value }

func (e *echo) client() (connect.HTTPClient, string) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e.got.Store(r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	return server.Client(), server.URL
}

func TestTheInterceptorPresentsTheTokenAndStopsACallThatHasNone(t *testing.T) {
	f := newFixture()
	e := &echo{}
	client, base := e.client()
	call := func(tokens *Tokens) error {
		c := connect.NewClient[accountsv1.ExchangeServiceCredentialRequest, accountsv1.ExchangeServiceCredentialResponse](
			client, base+"/x.Service/Method", connect.WithProtoJSON(), connect.WithInterceptors(tokens.Interceptor()))
		_, err := c.CallUnary(context.Background(), connect.NewRequest(&accountsv1.ExchangeServiceCredentialRequest{}))
		return err
	}
	if err := call(f.tokens()); err != nil {
		t.Fatal(err)
	}
	if got, _ := e.got.Load().(string); got != "Bearer tok-1" {
		t.Fatalf("Authorization = %q", got)
	}

	e.got.Store("")
	missing := newFixture()
	missing.credErr = errors.New("absent")
	err := call(missing.tokens())
	if connect.CodeOf(err) != connect.CodeUnauthenticated || !errors.Is(err, ErrNotIssued) {
		t.Fatalf("err = %v, want Unauthenticated wrapping ErrNotIssued", err)
	}
	if got, _ := e.got.Load().(string); got != "" {
		t.Fatalf("a call with no token must not be sent, but sent %q", got)
	}
}

type accountsStub struct {
	accountsconnect.UnimplementedAccountsServiceHandler
	credential string
	response   *accountsv1.ExchangeServiceCredentialResponse
	// refusal, when set, is returned instead of any response.
	refusal error
}

func (s *accountsStub) ExchangeServiceCredential(_ context.Context, req *connect.Request[accountsv1.ExchangeServiceCredentialRequest]) (*connect.Response[accountsv1.ExchangeServiceCredentialResponse], error) {
	s.credential = req.Msg.GetCredential()
	if s.refusal != nil {
		return nil, s.refusal
	}
	if s.response == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("credential rejected"))
	}
	return connect.NewResponse(s.response), nil
}

func TestTheDefaultExchangeCallsScenarioAuthenticatorAndReadsTheIdentityBack(t *testing.T) {
	expiry := time.Date(2026, 11, 14, 11, 0, 0, 0, time.UTC)
	stub := &accountsStub{response: &accountsv1.ExchangeServiceCredentialResponse{
		AccessToken: "tok", AccessTokenExpiresAt: timestamppb.New(expiry),
		Subject: expected.Subject, Audience: expected.Audience, Scopes: expected.Scopes,
	}}
	mux := http.NewServeMux()
	mux.Handle(accountsconnect.NewAccountsServiceHandler(stub))
	server := httptest.NewServer(mux)
	defer server.Close()
	exchange := AuthenticatorExchanger(server.Client(), func(context.Context) (string, error) { return server.URL + "/", nil })

	got, err := exchange(context.Background(), "cred")
	if err != nil {
		t.Fatal(err)
	}
	if stub.credential != "cred" || got.AccessToken != "tok" || !got.ExpiresAt.Equal(expiry) || got.Subject != expected.Subject ||
		got.Audience != expected.Audience || len(got.Scopes) != 2 {
		t.Fatalf("exchanged %+v (credential sent %q)", got, stub.credential)
	}

	stub.response = nil
	if _, err := exchange(context.Background(), "cred"); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("a rejected credential must surface as the authenticator's refusal: %v", err)
	}
	stub.response = &accountsv1.ExchangeServiceCredentialResponse{AccessToken: "tok"}
	if got, err := exchange(context.Background(), "cred"); err != nil || !got.ExpiresAt.IsZero() {
		t.Fatalf("a response with no expiry carries a zero expiry, which Token refuses: %+v, %v", got, err)
	}
	down := AuthenticatorExchanger(server.Client(), func(context.Context) (string, error) { return "", errors.New("not running") })
	if _, err := down(context.Background(), "cred"); err == nil {
		t.Fatal("an unreachable authenticator must be an error")
	}
}

func TestATokenThatIsEmptyOrTooCloseToExpiryIsNotUsable(t *testing.T) {
	cases := map[string]func(*Exchanged){
		"empty token":      func(e *Exchanged) { e.AccessToken = " " },
		"already expired":  func(e *Exchanged) { e.ExpiresAt = e.ExpiresAt.Add(-time.Hour) },
		"no expiry":        func(e *Exchanged) { e.ExpiresAt = time.Time{} },
		"inside the floor": func(e *Exchanged) { e.ExpiresAt = e.ExpiresAt.Add(-10*time.Minute + time.Minute) },
	}
	for name, mutate := range cases {
		f := newFixture()
		mutate(&f.result)
		tokens := New(Config{Credential: func(context.Context) (string, error) { return "c", nil }, Expect: expected, MinRemaining: 2 * time.Minute},
			WithClock(func() time.Time { return f.now }),
			WithExchanger(func(context.Context, string) (Exchanged, error) { return f.result, nil }))
		if got, err := tokens.Token(context.Background()); !errors.Is(err, ErrNoUsableToken) || got != "" {
			t.Fatalf("%s: Token = %q, %v, want ErrNoUsableToken", name, got, err)
		}
	}
}

// A principal whose peer verifies the token itself has no expected identity, and
// a refresh margin of its own; any unexpired token is used.
func TestAPrincipalWithNoExpectedIdentityPresentsWhatTheAuthenticatorReturns(t *testing.T) {
	f := newFixture()
	f.result.Subject, f.result.Audience, f.result.Scopes = "anything", "any", nil
	var exchanges int
	tokens := New(Config{Credential: func(context.Context) (string, error) { return "c", nil }, RefreshMargin: 2 * time.Minute},
		WithClock(func() time.Time { return f.now }),
		WithExchanger(func(context.Context, string) (Exchanged, error) { exchanges++; return f.result, nil }))
	for range 2 {
		if got, err := tokens.Token(context.Background()); err != nil || got != "tok-1" {
			t.Fatalf("Token = %q, %v", got, err)
		}
	}
	f.now = f.now.Add(8*time.Minute + time.Second) // inside the 2 minute margin, not the default
	f.result.AccessToken, f.result.ExpiresAt = "tok-2", f.now.Add(10*time.Minute)
	if got, _ := tokens.Token(context.Background()); got != "tok-2" || exchanges != 2 {
		t.Fatalf("the configured margin must govern the refresh: %q, %d exchanges", got, exchanges)
	}
}

func TestDescribeGivesAFailureItsOwnWordingAndKeepsItsKind(t *testing.T) {
	describe := func(kind, cause error) error {
		text := "svc: " + kind.Error()
		if cause != nil {
			text += " / " + cause.Error()
		}
		return Kind(kind, errors.New("custom: "+text))
	}
	f := newFixture()
	f.credErr = errors.New("no such entry")
	tokens := New(Config{Credential: func(context.Context) (string, error) { return f.credential, f.credErr }, Expect: expected, Describe: describe},
		WithExchanger(func(context.Context, string) (Exchanged, error) { return f.result, f.exchangeEr }))
	_, err := tokens.Token(context.Background())
	if !errors.Is(err, ErrNotIssued) || !strings.HasPrefix(err.Error(), "custom: svc: ") || !strings.HasSuffix(err.Error(), "/ no such entry") {
		t.Fatalf("err = %v", err)
	}
	f.credErr, f.credential, f.exchangeEr = nil, "super-secret", errors.New("refused super-secret")
	_, err = tokens.Token(context.Background())
	if !errors.Is(err, ErrExchangeFailed) || strings.Contains(err.Error(), "super-secret") {
		t.Fatalf("a described exchange failure still never carries the credential: %v", err)
	}
	if _, err := New(Config{Describe: describe}).Token(context.Background()); !errors.Is(err, ErrNotConfigured) || !strings.HasPrefix(err.Error(), "custom: ") {
		t.Fatalf("err = %v", err)
	}
}

func TestDescribeWithCauseNamesTheScenarioAndCarriesTheCause(t *testing.T) {
	describe := DescribeWithCause("prompt-manager")
	for _, c := range []struct {
		kind  error
		cause error
		want  string
	}{
		{ErrNotIssued, errors.New("no entry"), "prompt-manager service credential unavailable: no entry"},
		{ErrNotIssued, nil, "prompt-manager service credential unavailable: no credential issued"},
		{ErrExchangeFailed, errors.New("refused"), "prompt-manager service credential exchange: refused"},
		{ErrNoUsableToken, nil, "prompt-manager service credential exchange returned no usable token"},
		{ErrNotConfigured, nil, "prompt-manager service identity is unavailable"},
	} {
		err := describe(c.kind, c.cause)
		if err.Error() != c.want || !errors.Is(err, c.kind) {
			t.Errorf("describe(%v) = %q (Is kind %v), want %q", c.kind, err, errors.Is(err, c.kind), c.want)
		}
	}
}

// scenario-authenticator budgets exchanges per principal. A spent budget is told
// apart from a refused credential on the wire, and says when to retry.
func TestARateLimitedExchangeIsToldApartFromARefusedCredential(t *testing.T) {
	refusal := connect.NewError(connect.CodeResourceExhausted, errors.New("service credential exchange rate limited; retry after 42s"))
	refusal.Meta().Set("Retry-After", "42")
	stub := &accountsStub{refusal: refusal}
	mux := http.NewServeMux()
	mux.Handle(accountsconnect.NewAccountsServiceHandler(stub))
	server := httptest.NewServer(mux)
	defer server.Close()
	exchange := AuthenticatorExchanger(server.Client(), func(context.Context) (string, error) { return server.URL, nil })

	_, err := exchange(context.Background(), "cred")
	var limited *RateLimitedError
	if !errors.As(err, &limited) || limited.RetryAfter != 42*time.Second {
		t.Fatalf("exchange error = %v, want a RateLimitedError that says to retry after 42s", err)
	}

	stub.refusal = connect.NewError(connect.CodeResourceExhausted, errors.New("rate limited"))
	if _, err := exchange(context.Background(), "cred"); !errors.As(err, &limited) || limited.RetryAfter != 0 {
		t.Fatalf("a refusal with no Retry-After = %v, want a RateLimitedError with no wait", err)
	}

	stub.refusal = connect.NewError(connect.CodeUnauthenticated, errors.New("service credential rejected"))
	if _, err := exchange(context.Background(), "cred"); errors.As(err, &limited) || connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("a rejected credential = %v, must stay the authenticator's refusal", err)
	}
}

func TestARateLimitedTokenIsReportedAsRateLimitedAndAsAnExchangeFailure(t *testing.T) {
	f := newFixture()
	f.credential = "super-secret-credential"
	f.exchangeEr = &RateLimitedError{RetryAfter: 42 * time.Second}
	_, err := f.tokens().Token(context.Background())
	var limited *RateLimitedError
	if !errors.Is(err, ErrRateLimited) || !errors.Is(err, ErrExchangeFailed) || !errors.As(err, &limited) || limited.RetryAfter != 42*time.Second {
		t.Fatalf("err = %v, want ErrRateLimited and ErrExchangeFailed carrying the wait", err)
	}
	want := "research-desk web-console credential: rate limited, retry after 42s"
	if err.Error() != want {
		t.Fatalf("err = %q, want %q; the text must not call the credential refused", err, want)
	}

	// A caller's own wording still words it, and the kind survives.
	described := New(Config{
		Credential: func(context.Context) (string, error) { return "cred", nil },
		Describe:   DescribeWithCause("research-desk"),
	}, WithExchanger(func(context.Context, string) (Exchanged, error) { return Exchanged{}, f.exchangeEr }))
	_, err = described.Token(context.Background())
	if !errors.Is(err, ErrRateLimited) || !errors.Is(err, ErrExchangeFailed) || !strings.Contains(err.Error(), "rate limited, retry after 42s") {
		t.Fatalf("described err = %v", err)
	}
}
