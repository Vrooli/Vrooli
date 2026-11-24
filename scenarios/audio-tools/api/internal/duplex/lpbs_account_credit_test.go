package duplex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/consumeridentity"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/lpbsidentity"
	"github.com/vrooli/api-core/servicetoken"
)

type testAccountCredential struct {
	value string
	err   error
	id    string
	field string
}

func (c *testAccountCredential) Resolve(_ context.Context, id, field string) (string, error) {
	c.id, c.field = id, field
	return c.value, c.err
}

type testAccountToken string

func (t testAccountToken) Token(context.Context) (string, error) { return string(t), nil }

func TestAccountCreditServiceTokenExactRoleAndMissingCredential(t *testing.T) {
	role := lpbsidentity.AccountCreditLifecycle
	now := time.Now()
	credential := &testAccountCredential{value: "secret-credential"}
	tokens, err := NewAccountCreditServiceTokens(credential, "http://authenticator.test", nil,
		servicetoken.WithClock(func() time.Time { return now }),
		servicetoken.WithExchanger(func(_ context.Context, got string) (servicetoken.Exchanged, error) {
			if got != "secret-credential" {
				t.Fatalf("credential=%q", got)
			}
			return servicetoken.Exchanged{AccessToken: "short-lived", ExpiresAt: now.Add(time.Hour), Subject: role.Principal("audio-tools"), Audience: role.Audience, Scopes: []string{role.Scope}}, nil
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := tokens.Token(context.Background()); err != nil || got != "short-lived" {
		t.Fatalf("Token=%q,%v", got, err)
	}
	if credential.id != role.CredentialIdentity("audio-tools") || credential.field != "credential" {
		t.Fatalf("credential lookup=%q/%q", credential.id, credential.field)
	}

	wrong, err := NewAccountCreditServiceTokens(credential, "http://authenticator.test", nil,
		servicetoken.WithClock(func() time.Time { return now }),
		servicetoken.WithExchanger(func(context.Context, string) (servicetoken.Exchanged, error) {
			return servicetoken.Exchanged{AccessToken: "wrong", ExpiresAt: now.Add(time.Hour), Subject: "audio-tools-lpbs-ai-inference", Audience: role.Audience, Scopes: []string{"lpbs:ai-inference.call"}}, nil
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wrong.Token(context.Background()); err == nil {
		t.Fatal("wrong-role token accepted")
	}
	missing, err := NewAccountCreditServiceTokens(&testAccountCredential{err: errors.New("not issued")}, "http://authenticator.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := missing.Token(context.Background()); err == nil {
		t.Fatal("missing credential produced token")
	}
}

func verifiedUserContext(t *testing.T) context.Context {
	t.Helper()
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "human-1", Source: identity.SourceConsumerAccessToken, Verified: true})
	ctx, err := WithVerifiedHumanBearer(ctx, "Bearer human-access")
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func TestWithVerifiedHumanBearerRejectsWrongIdentityAndMalformedHeader(t *testing.T) {
	for name, principal := range map[string]identity.Principal{
		"unverified":   {Kind: identity.ActorHuman, Source: identity.SourceConsumerAccessToken},
		"service":      {Kind: identity.ActorService, Subject: "service-1", Source: identity.SourceConsumerAccessToken, Verified: true},
		"agent":        {Kind: identity.ActorAgent, Subject: "agent-1", Source: identity.SourceConsumerAccessToken, Verified: true},
		"wrong source": {Kind: identity.ActorHuman, Subject: "human-1", Source: identity.SourceCloudflareAccess, Verified: true},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := identity.WithPrincipal(context.Background(), principal)
			if _, err := WithVerifiedHumanBearer(ctx, "Bearer human-access"); !errors.Is(err, ErrHumanBearerRequired) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	for _, header := range []string{"", "Cookie=not-a-bearer", "Bearer two words"} {
		if _, err := WithVerifiedHumanBearer(identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "human-1", Source: identity.SourceConsumerAccessToken, Verified: true}), header); !errors.Is(err, ErrHumanBearerRequired) {
			t.Fatalf("header %q accepted: %v", header, err)
		}
	}
}

func TestWithVerifiedHumanRequestCarriesAuthenticatedBearerOrCookie(t *testing.T) {
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "human-1", Source: identity.SourceConsumerAccessToken, Verified: true}
	for _, tc := range []struct {
		name  string
		build func() *http.Request
	}{
		{name: "bearer", build: func() *http.Request {
			req := httptest.NewRequest(http.MethodPost, "/duplex", nil)
			req.Header.Set("Authorization", "Bearer lpbs-token")
			return req
		}},
		{name: "browser cookie", build: func() *http.Request {
			req := httptest.NewRequest(http.MethodPost, "/duplex", nil)
			req.AddCookie(&http.Cookie{Name: "__Host-access_token", Value: "lpbs-cookie-token"})
			return req
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.build()
			ctx, err := WithVerifiedHumanRequest(identity.WithPrincipal(context.Background(), principal), req)
			if err != nil {
				t.Fatal(err)
			}
			got, ok := humanBearer(ctx)
			if !ok {
				t.Fatal("verified user token missing from request context")
			}
			want := consumeridentity.RequestAccessToken(req)
			if got != want {
				t.Fatalf("forwarded token does not match authenticated request: got %q want %q", got, want)
			}
		})
	}
	_, err := WithVerifiedHumanRequest(identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "human-1", Source: identity.SourceScenarioAuthenticator, Verified: true}), httptest.NewRequest(http.MethodPost, "/duplex", nil))
	if !errors.Is(err, ErrHumanBearerRequired) {
		t.Fatalf("scenario identity accepted for LPBS billing: %v", err)
	}
}

func TestAccountCreditClientKeepsAuthSeparateAndDoesNotFollowRedirect(t *testing.T) {
	ctx := verifiedUserContext(t)
	var destinationHit bool
	destination := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { destinationHit = true }))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer human-access" {
			t.Errorf("Authorization=%q", r.Header.Get("Authorization"))
		}
		if r.Header.Get(lpbsidentity.AccountCreditLifecycleAuthorizationHeader) != "Bearer service-access" {
			t.Errorf("service header=%q", r.Header.Get(lpbsidentity.AccountCreditLifecycleAuthorizationHeader))
		}
		if r.URL.Path != accountCreditUSDV1ReservePath {
			t.Errorf("path=%q", r.URL.Path)
		}
		http.Redirect(w, r, destination.URL+"/capture", http.StatusFound)
	}))
	defer source.Close()
	client, err := NewAccountCreditClient(source.URL, testAccountToken("service-access"), nil)
	if err != nil {
		t.Fatal(err)
	}
	var response any
	if err := client.call(ctx, http.MethodPost, accountCreditUSDV1ReservePath, map[string]int64{"budget_micros": 1}, &response); err == nil {
		t.Fatal("redirect status accepted")
	}
	if destinationHit {
		t.Fatal("sensitive auth headers followed redirect")
	}
}

type lifecycleFixture struct {
	dispatch    bool
	dispatchErr error
	paths       []string
	userAuth    []string
	serviceAuth []string
}

func (f *lifecycleFixture) Do(r *http.Request) (*http.Response, error) {
	f.paths = append(f.paths, r.URL.Path)
	f.userAuth = append(f.userAuth, r.Header.Get("Authorization"))
	f.serviceAuth = append(f.serviceAuth, r.Header.Get(lpbsidentity.AccountCreditLifecycleAuthorizationHeader))
	if strings.HasSuffix(r.URL.Path, "/dispatch") && f.dispatchErr != nil {
		return nil, f.dispatchErr
	}
	status, body := http.StatusOK, "{}"
	switch {
	case r.URL.Path == accountCreditUSDV1ReservePath:
		data, _ := io.ReadAll(r.Body)
		var in struct {
			OperationID  string `json:"operation_id"`
			BudgetMicros int64  `json:"budget_micros"`
		}
		_ = json.Unmarshal(data, &in)
		body = fmt.Sprintf(`{"reservation_id":%q,"created":true,"phase":"reserved","contract_version":1,"budget_micros":%d,"reserved_amount":1,"cost_multiplier":10000}`, in.OperationID, in.BudgetMicros)
	case strings.HasSuffix(r.URL.Path, "/dispatch"):
		if f.dispatch {
			body = "{\"dispatch_authorized\":true,\"phase\":\"dispatched\"}"
		} else {
			body = "{\"dispatch_authorized\":false,\"phase\":\"already_dispatched_or_terminal\"}"
		}
	case strings.HasSuffix(r.URL.Path, "/settle") || strings.HasSuffix(r.URL.Path, "/settle-usd-v1") || strings.HasSuffix(r.URL.Path, "/release") || strings.HasSuffix(r.URL.Path, "/retain-uncertain"):
		operation := strings.TrimPrefix(r.URL.Path, accountCreditLifecycleBasePath+"/")
		id, _, _ := strings.Cut(operation, "/")
		phase := "settled"
		if strings.HasSuffix(r.URL.Path, "/release") {
			phase = "released"
		} else if strings.HasSuffix(r.URL.Path, "/retain-uncertain") {
			phase = "uncertain"
		}
		body = "{\"reservation_id\":\"" + id + "\",\"phase\":\"" + phase + "\"}"
	}
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
}

func TestAccountCreditDispatchResponseLossForcesRetainAndBlocksRelease(t *testing.T) {
	fixture := &lifecycleFixture{dispatchErr: errors.New("response lost")}
	client, err := NewAccountCreditClient("http://lpbs.test", testAccountToken("service-access"), fixture)
	if err != nil {
		t.Fatal(err)
	}
	reservation, err := client.Reserve(verifiedUserContext(t), 10, "model", "provider")
	if err != nil {
		t.Fatal(err)
	}
	ctx := verifiedUserContext(t)
	if _, err := reservation.BeginDispatch(ctx); err == nil {
		t.Fatal("lost dispatch response was treated as authorized")
	}
	if err := reservation.ReleaseBeforeDispatch(ctx); err == nil {
		t.Fatal("ambiguous dispatch hold was released")
	}
	if err := reservation.RetainUncertain(ctx); err != nil {
		t.Fatalf("could not retain ambiguous dispatch: %v", err)
	}
	if got := fixture.paths[len(fixture.paths)-1]; !strings.HasSuffix(got, "/retain-uncertain") {
		t.Fatalf("last lifecycle operation=%q", got)
	}
}

type fixedAccountCreditAdmission struct{ reservation Reservation }

func (a fixedAccountCreditAdmission) Authorize(context.Context, Request) error { return nil }
func (a fixedAccountCreditAdmission) Reserve(context.Context, Request) (Reservation, error) {
	return a.reservation, nil
}

func TestCoordinatorCloseUsesReservationHumanBearerForSettleAndRetain(t *testing.T) {
	for _, tc := range []struct {
		name      string
		confirmed bool
		terminal  string
	}{
		{name: "settle confirmed usage", confirmed: true, terminal: "/settle-usd-v1"},
		{name: "retain unconfirmed usage", confirmed: false, terminal: "/retain-uncertain"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := &lifecycleFixture{dispatch: true}
			client, err := NewAccountCreditClient("http://lpbs.test", testAccountToken("service-access"), fixture)
			if err != nil {
				t.Fatal(err)
			}
			reservation, err := client.Reserve(verifiedUserContext(t), 100, "gpt-live-1.1", "openai-gpt-live")
			if err != nil {
				t.Fatal(err)
			}
			provider := finalUsageProvider{cost: 25, confirmed: tc.confirmed}
			coordinator, err := NewCoordinator(provider, fixedAccountCreditAdmission{reservation: reservation})
			if err != nil {
				t.Fatal(err)
			}
			handle, err := coordinator.Open(verifiedUserContext(t), Request{})
			if err != nil {
				t.Fatal(err)
			}
			if err := handle.Close("test"); err != nil {
				t.Fatal(err)
			}
			if len(fixture.paths) != 3 || !strings.HasSuffix(fixture.paths[1], "/dispatch") || !strings.HasSuffix(fixture.paths[2], tc.terminal) {
				t.Fatalf("lifecycle calls=%v", fixture.paths)
			}
			for i := range fixture.paths {
				if fixture.userAuth[i] != "Bearer human-access" || fixture.serviceAuth[i] != "Bearer service-access" {
					t.Fatalf("request %d auth user=%q service=%q", i, fixture.userAuth[i], fixture.serviceAuth[i])
				}
			}
		})
	}
}

func TestAccountCreditReservationDispatchFenceAndUncertainState(t *testing.T) {
	fixture := &lifecycleFixture{dispatch: true}
	client, err := NewAccountCreditClient("http://lpbs.test", testAccountToken("service-access"), fixture)
	if err != nil {
		t.Fatal(err)
	}
	reservation, err := client.Reserve(verifiedUserContext(t), 100, "gpt-live-1.1", "openai-gpt-live")
	if err != nil {
		t.Fatal(err)
	}
	ctx := verifiedUserContext(t)
	if ok, err := reservation.BeginDispatch(ctx); err != nil || !ok {
		t.Fatalf("dispatch=%v,%v", ok, err)
	}
	usage := ProviderUsage{ProviderReceiptID: "live-session-1", Model: "gpt-live-1.1", Provider: "openai-gpt-live", ActiveSeconds: 2, PausedSeconds: 1, CostMicros: 25}
	wrongIdentity := usage
	wrongIdentity.Model = "different-model"
	if err := reservation.SettleProviderUsage(ctx, wrongIdentity); err == nil {
		t.Fatal("usage attributed to a different model was accepted")
	}
	wrongIdentity = usage
	wrongIdentity.Provider = "different-provider"
	if err := reservation.SettleProviderUsage(ctx, wrongIdentity); err == nil {
		t.Fatal("usage attributed to a different provider was accepted")
	}
	if len(fixture.paths) != 2 {
		t.Fatalf("invalid usage reached settlement endpoint: %v", fixture.paths)
	}
	overBudget := usage
	overBudget.CostMicros = 101
	if err := reservation.SettleProviderUsage(ctx, overBudget); err == nil {
		t.Fatal("measured cost above the USD reservation was accepted")
	}
	if len(fixture.paths) != 2 {
		t.Fatalf("over-budget usage reached LPBS: %v", fixture.paths)
	}
	if err := reservation.SettleProviderUsage(ctx, usage); err != nil {
		t.Fatal(err)
	}
	if err := reservation.ReleaseBeforeDispatch(ctx); err == nil {
		t.Fatal("released a dispatched hold")
	}
	if len(fixture.paths) != 3 || !strings.HasSuffix(fixture.paths[1], "/dispatch") || !strings.HasSuffix(fixture.paths[2], "/settle-usd-v1") {
		t.Fatalf("calls=%v", fixture.paths)
	}
}

func TestAccountCreditReservationRejectsMissingRequestContext(t *testing.T) {
	r := &AccountCreditReservation{userBearer: "user-token"}
	if _, err := r.ownerContext(nil); err == nil {
		t.Fatal("ownerContext(nil) unexpectedly created a detached context")
	}
	r.journal = nilContextJournal{}
	if err := r.journalTransition(nil, "uncertain", ProviderUsage{}); err == nil {
		t.Fatal("journalTransition(nil) unexpectedly accepted a detached context")
	}
}

type nilContextJournal struct{}

func (nilContextJournal) Create(context.Context, JournalOperation) error { return nil }
func (nilContextJournal) Transition(context.Context, string, string, string, ProviderUsage) error {
	return nil
}
func (nilContextJournal) Recoverable(context.Context, string) ([]JournalOperation, error) {
	return nil, nil
}

func TestAccountCreditReservationCanReleaseOnlyBeforeDispatch(t *testing.T) {
	fixture := &lifecycleFixture{}
	client, err := NewAccountCreditClient("http://lpbs.test", testAccountToken("service-access"), fixture)
	if err != nil {
		t.Fatal(err)
	}
	reservation, err := client.Reserve(verifiedUserContext(t), 10, "model", "provider")
	if err != nil {
		t.Fatal(err)
	}
	ctx := verifiedUserContext(t)
	if err := reservation.ReleaseBeforeDispatch(ctx); err != nil {
		t.Fatal(err)
	}
	if ok, err := reservation.BeginDispatch(ctx); err != nil || ok {
		t.Fatalf("dispatch after release=%v,%v", ok, err)
	}
	if len(fixture.paths) != 2 || !strings.HasSuffix(fixture.paths[1], "/release") {
		t.Fatalf("calls=%v", fixture.paths)
	}
}

func TestValidateSettlementAmountsBoundsSignedLPBSValues(t *testing.T) {
	maxInt64 := uint64(^uint64(0) >> 1)
	reservation := &AccountCreditReservation{budgetMicros: int64(maxInt64)}
	if err := reservation.validateSettlementAmounts(ProviderUsage{
		CostMicros: maxInt64, ActiveSeconds: maxInt64, PausedSeconds: maxInt64,
	}); err != nil {
		t.Fatalf("maximum signed values should fit LPBS settlement: %v", err)
	}

	cases := []struct {
		name  string
		usage ProviderUsage
	}{
		{name: "cost overflow", usage: ProviderUsage{CostMicros: maxInt64 + 1}},
		{name: "active time overflow", usage: ProviderUsage{ActiveSeconds: maxInt64 + 1}},
		{name: "paused time overflow", usage: ProviderUsage{PausedSeconds: maxInt64 + 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := reservation.validateSettlementAmounts(tc.usage); err == nil {
				t.Fatalf("out-of-range settlement value was accepted: %+v", tc.usage)
			}
		})
	}
}
