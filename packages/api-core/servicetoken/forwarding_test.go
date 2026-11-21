package servicetoken

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// insideAgentRequest runs call the way a scenario handler runs when an agent run
// called it: inside an inbound request provenance.Middleware verified as that run.
func insideAgentRequest(t *testing.T, call func(ctx context.Context)) {
	t.Helper()
	inbound := provenance.Middleware(provenance.VerifierFunc(func(string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: "run-a"}}, nil
	}))(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if provenance.ForwardedIdentityToken(r.Context()) == "" {
			t.Error("fixture: the inbound request should carry a verified agent identity")
		}
		call(r.Context())
	}))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "run-a-credential")
	inbound.ServeHTTP(httptest.NewRecorder(), req)
}

// receiver records, per path, the agent identity and Authorization each request carried.
type receiver struct {
	mu     sync.Mutex
	agent  map[string]string
	bearer map[string]string
}

func (r *receiver) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		r.agent[req.URL.Path] = req.Header.Get(cliutil.HeaderAgentIdentityToken) + req.Header.Get(cliutil.HeaderAgentOperationToken)
		r.bearer[req.URL.Path] = req.Header.Get("Authorization")
		r.mu.Unlock()
		next.ServeHTTP(w, req)
	})
}

func newReceiver(t *testing.T, next http.Handler) (*receiver, string) {
	t.Helper()
	r := &receiver{agent: map[string]string{}, bearer: map[string]string{}}
	server := httptest.NewServer(r.wrap(next))
	t.Cleanup(server.Close)
	return r, server.URL
}

// The clients use http.DefaultTransport, which forwards a verified caller's
// identity; the control call in each test shows that it would have been.
func TestAServiceCallNeverForwardsTheIdentityOfTheRunWhoseRequestLedToIt(t *testing.T) {
	seen, base := newReceiver(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	tokens := newFixture().tokens()
	insideAgentRequest(t, func(ctx context.Context) {
		control, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/control", nil)
		if response, err := http.DefaultClient.Do(control); err == nil {
			_ = response.Body.Close()
		}
		c := connect.NewClient[accountsv1.ExchangeServiceCredentialRequest, accountsv1.ExchangeServiceCredentialResponse](
			&http.Client{}, base+"/x.Service/Method", connect.WithProtoJSON(), connect.WithInterceptors(tokens.Interceptor()))
		if _, err := c.CallUnary(ctx, connect.NewRequest(&accountsv1.ExchangeServiceCredentialRequest{})); err != nil {
			t.Errorf("call: %v", err)
		}
	})
	if seen.agent["/control"] == "" {
		t.Fatal("fixture: a plain call inside the agent's request should forward its identity")
	}
	if got := seen.agent["/x.Service/Method"]; got != "" {
		t.Fatalf("the service call forwarded the agent identity %q", got)
	}
	if got := seen.bearer["/x.Service/Method"]; got != "Bearer tok-1" {
		t.Fatalf("the service call must still present its own token, got %q", got)
	}
}

func TestTheCredentialExchangeNeverForwardsTheIdentityOfTheRunWhoseRequestLedToIt(t *testing.T) {
	stub := &accountsStub{response: &accountsv1.ExchangeServiceCredentialResponse{AccessToken: "tok", AccessTokenExpiresAt: timestamppb.Now()}}
	mux := http.NewServeMux()
	path, handler := accountsconnect.NewAccountsServiceHandler(stub)
	mux.Handle(path, handler)
	mux.Handle("/control", http.NotFoundHandler())
	seen, base := newReceiver(t, mux)
	exchange := AuthenticatorExchanger(&http.Client{}, func(context.Context) (string, error) { return base, nil })
	insideAgentRequest(t, func(ctx context.Context) {
		control, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/control", nil)
		if response, err := http.DefaultClient.Do(control); err == nil {
			_ = response.Body.Close()
		}
		if _, err := exchange(ctx, "cred"); err != nil {
			t.Errorf("exchange: %v", err)
		}
	})
	if seen.agent["/control"] == "" {
		t.Fatal("fixture: a plain call inside the agent's request should forward its identity")
	}
	exchanged := accountsconnect.AccountsServiceExchangeServiceCredentialProcedure
	if _, ok := seen.agent[exchanged]; !ok || stub.credential != "cred" {
		t.Fatal("the exchange did not reach scenario-authenticator")
	}
	if got := seen.agent[exchanged]; got != "" {
		t.Fatalf("the exchange forwarded the agent identity %q", got)
	}
}

// Many scenario clients set the header themselves (tokens.Authorization) on a
// request sent through the default transport. The token marks itself when it is
// exchanged, so those requests carry no caller identity either.
func TestAServiceTokenSetByHandNeverCarriesTheIdentityOfTheRunWhoseRequestLedToIt(t *testing.T) {
	seen, base := newReceiver(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	tokens := newFixture().tokens()
	insideAgentRequest(t, func(ctx context.Context) {
		control, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/control", nil)
		if response, err := http.DefaultClient.Do(control); err == nil {
			_ = response.Body.Close()
		}
		authorization, err := tokens.Authorization(ctx)
		if err != nil {
			t.Errorf("authorization: %v", err)
			return
		}
		call, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/by-hand", nil)
		call.Header.Set("Authorization", authorization)
		if response, err := http.DefaultClient.Do(call); err == nil {
			_ = response.Body.Close()
		}
	})
	if seen.agent["/control"] == "" {
		t.Fatal("fixture: a plain call inside the agent's request should forward its identity")
	}
	if got := seen.agent["/by-hand"]; got != "" || seen.bearer["/by-hand"] != "Bearer tok-1" {
		t.Fatalf("the hand-built service call carried agent %q with bearer %q", got, seen.bearer["/by-hand"])
	}
}
