package treasuryadmin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/stretchr/testify/require"
	authv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"

	"treasury/handlers/treasuryadmin"
	"treasury/internal/operatorauth"
	"treasury/internal/wallet"
)

type walletStub struct {
	calls    int
	operator string
}

func (s *walletStub) value(operator string) wallet.Wallet {
	s.calls++
	s.operator = operator
	return wallet.Wallet{ID: "wallet-1", BookID: "book-1", OperatorIdentity: operator, Network: "eip155:84532", Address: "0x1111111111111111111111111111111111111111", CreatedAt: time.Now()}
}

func (s *walletStub) Create(_ context.Context, in wallet.CreateInput) (wallet.Wallet, error) {
	return s.value(in.OperatorIdentity), nil
}

func (s *walletStub) Get(_ context.Context, _, _, operator string) (wallet.Wallet, error) {
	return s.value(operator), nil
}

func (s *walletStub) Prove(_ context.Context, _, _, operator, _ string) (wallet.Proof, error) {
	return wallet.Proof{Wallet: s.value(operator), Verified: true}, nil
}

func (s *walletStub) QualifyRecovery(_ context.Context, _, _, operator string) (wallet.Wallet, error) {
	return s.value(operator), nil
}

func (s *walletStub) Recover(_ context.Context, _, _, operator string) (wallet.Wallet, error) {
	return s.value(operator), nil
}

func TestResearchWalletRPCsPreserveOperatorRealmAndAttribution(t *testing.T) {
	authorizer, err := operatorauth.NewStaticToken("test-operator-secret")
	require.NoError(t, err)
	stub := &walletStub{}
	_, transport := authconnect.NewTreasuryAdminHandler(treasuryadmin.NewConnectHandler(authorizer, stub))
	server := httptest.NewServer(transport)
	defer server.Close()
	client := authconnect.NewTreasuryAdminClient(server.Client(), server.URL)
	methods := map[string]func(http.Header) error{
		"RecoverResearchWallet": func(h http.Header) error {
			r := connect.NewRequest(&authv1.ResearchWalletRequest{Id: "wallet-1", Network: "eip155:84532"})
			r.Header().Set(operatorauth.HeaderOperatorToken, h.Get(operatorauth.HeaderOperatorToken))
			r.Header().Set(operatorauth.HeaderAgentToken, h.Get(operatorauth.HeaderAgentToken))
			_, err := client.RecoverResearchWallet(context.Background(), r)
			return err
		},
		"CreateResearchWallet": func(h http.Header) error {
			r := connect.NewRequest(&authv1.CreateResearchWalletRequest{Id: "wallet-1", BookId: "book-1", Network: "eip155:84532"})
			r.Header().Set(operatorauth.HeaderOperatorToken, h.Get(operatorauth.HeaderOperatorToken))
			r.Header().Set(operatorauth.HeaderAgentToken, h.Get(operatorauth.HeaderAgentToken))
			_, err := client.CreateResearchWallet(context.Background(), r)
			return err
		},
		"GetResearchWallet": func(h http.Header) error {
			r := connect.NewRequest(&authv1.ResearchWalletRequest{Id: "wallet-1", Network: "eip155:84532"})
			r.Header().Set(operatorauth.HeaderOperatorToken, h.Get(operatorauth.HeaderOperatorToken))
			r.Header().Set(operatorauth.HeaderAgentToken, h.Get(operatorauth.HeaderAgentToken))
			_, err := client.GetResearchWallet(context.Background(), r)
			return err
		},
		"ProveResearchWallet": func(h http.Header) error {
			r := connect.NewRequest(&authv1.ProveResearchWalletRequest{Id: "wallet-1", Network: "eip155:84532", Nonce: "fixed_test_nonce_20261027"})
			r.Header().Set(operatorauth.HeaderOperatorToken, h.Get(operatorauth.HeaderOperatorToken))
			r.Header().Set(operatorauth.HeaderAgentToken, h.Get(operatorauth.HeaderAgentToken))
			_, err := client.ProveResearchWallet(context.Background(), r)
			return err
		},
		"QualifyResearchWalletRecovery": func(h http.Header) error {
			r := connect.NewRequest(&authv1.ResearchWalletRequest{Id: "wallet-1", Network: "eip155:84532"})
			r.Header().Set(operatorauth.HeaderOperatorToken, h.Get(operatorauth.HeaderOperatorToken))
			r.Header().Set(operatorauth.HeaderAgentToken, h.Get(operatorauth.HeaderAgentToken))
			_, err := client.QualifyResearchWalletRecovery(context.Background(), r)
			return err
		},
	}
	for name, call := range methods {
		t.Run(name, func(t *testing.T) {
			for _, h := range []http.Header{{}, {operatorauth.HeaderAgentToken: []string{"agent"}}, {operatorauth.HeaderAgentToken: []string{"agent"}, operatorauth.HeaderOperatorToken: []string{"test-operator-secret"}}, {operatorauth.HeaderOperatorToken: []string{"wrong"}}} {
				before := stub.calls
				require.Error(t, call(h))
				require.Equal(t, before, stub.calls)
			}
			h := http.Header{}
			h.Set(operatorauth.HeaderOperatorToken, "test-operator-secret")
			require.NoError(t, call(h))
			require.Equal(t, "local-operator", stub.operator)
		})
	}
}
