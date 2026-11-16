package identity

import (
	"connectrpc.com/connect"
	"context"
	"errors"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
	"net/http/httptest"
	"testing"
)

type mfaResolver string

func (r mfaResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return string(r), nil
}

type mfaAuthenticator struct {
	accountsconnect.UnimplementedAccountsServiceHandler
	t *testing.T
}

func (a mfaAuthenticator) Login(_ context.Context, req *connect.Request[accountsv1.LoginRequest]) (*connect.Response[accountsv1.LoginResponse], error) {
	if req.Msg.GetMfaChallenge() == "" {
		return connect.NewResponse(&accountsv1.LoginResponse{MfaRequired: true, MfaChallenge: "challenge"}), nil
	}
	if req.Msg.GetMfaChallenge() != "challenge" || req.Msg.GetRecoveryCode() != "recovery" || req.Msg.GetPassword() != "password" {
		a.t.Fatal("MFA fields were not relayed")
	}
	return connect.NewResponse(&accountsv1.LoginResponse{Tokens: &accountsv1.TokenPair{AccessToken: "access"}}), nil
}
func TestForwarderRelaysMFAChallengeAndRecoveryCode(t *testing.T) {
	_, handler := accountsconnect.NewAccountsServiceHandler(mfaAuthenticator{t: t})
	server := httptest.NewServer(handler)
	defer server.Close()
	f := Forwarder{Resolver: mfaResolver(server.URL)}
	first, err := f.Login(context.Background(), "owner@example.test", "password", "", "", "")
	if err != nil || !first.MFARequired || first.MFAChallenge != "challenge" || first.AccessToken != "" {
		t.Fatalf("challenge: %+v %v", first, err)
	}
	result, err := f.Login(context.Background(), "owner@example.test", "password", first.MFAChallenge, "", "recovery")
	if err != nil || result.AccessToken != "access" || result.MFARequired {
		t.Fatalf("completed: %+v %v", result, err)
	}
}

func TestForwarderPreservesRateLimitAndUnavailable(t *testing.T) {
	if !errors.Is(mapAuthError(connect.NewError(connect.CodeResourceExhausted, errors.New("limited"))), ErrRateLimited) {
		t.Fatal("rate limit collapsed")
	}
	if !errors.Is(mapAuthError(connect.NewError(connect.CodeUnavailable, errors.New("offline"))), ErrAuthUnavailable) {
		t.Fatal("outage collapsed")
	}
}
