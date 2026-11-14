package treasuryadmin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/stretchr/testify/require"
	authv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	"treasury/internal/baseusdc"

	"treasury/handlers/treasuryadmin"
	"treasury/internal/operatorauth"
	"treasury/internal/wallet"
)

type walletStub struct {
	calls    int
	operator string
	network  string
}

func (s *walletStub) value(operator string) wallet.Wallet {
	s.calls++
	s.operator = operator
	network := s.network
	if network == "" {
		network = "eip155:84532"
	}
	return wallet.Wallet{ID: "wallet-1", BookID: "book-1", OperatorIdentity: operator, Network: network, Address: "0x1111111111111111111111111111111111111111", CustodyReference: "vrooli/treasury/wallets/fixture/reference", CreatedAt: time.Now()}
}

type receiptServiceStub struct {
	calls    int
	wallet   wallet.Wallet
	operator string
	txHash   string
	priorID  string
	check    baseusdc.Check
	checks   []baseusdc.Check
}

func (s *receiptServiceStub) Recheck(_ context.Context, operator string, w wallet.Wallet, txHash, priorID string) (baseusdc.Check, error) {
	s.calls++
	s.wallet, s.operator, s.txHash, s.priorID = w, operator, txHash, priorID
	return s.check, nil
}

func (s *receiptServiceStub) Verify(_ context.Context, operator string, w wallet.Wallet, txHash string) (baseusdc.Check, error) {
	s.calls++
	s.wallet, s.operator, s.txHash = w, operator, txHash
	return s.check, nil
}

func (s *receiptServiceStub) List(_ context.Context, operator, walletID, txHash string, _ int) ([]baseusdc.Check, error) {
	s.operator, s.txHash = operator, txHash
	return s.checks, nil
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

func TestVerifyBaseUSDCReceiptIsOperatorAuthenticatedAndReadOnly(t *testing.T) {
	authorizer, err := operatorauth.NewStaticToken("test-operator-secret")
	require.NoError(t, err)
	wallets := &walletStub{network: baseusdc.Network}
	verifier := &receiptServiceStub{check: baseusdc.Check{ID: "receipt-check-1", OperatorIdentity: "local-operator", WalletID: "wallet-1", Network: baseusdc.Network, TransactionHash: "0x" + strings.Repeat("a", 64), ProviderID: "fixture-base-rpc", Outcome: "verified", CheckedAt: time.Date(2026, 11, 2, 18, 1, 0, 0, time.UTC), Transfers: []baseusdc.Observation{{
		EconomicKey: "eip155:8453/erc20:" + baseusdc.USDCAddress + "/log/7",
		WalletID:    "wallet-1", Network: baseusdc.Network, AssetID: baseusdc.TransferAsset,
		Decimals: 6, AtomicAmount: "5901522", TransactionHash: "0x" + strings.Repeat("a", 64),
		LogIndex: 7, BlockNumber: 100, BlockHash: "0x" + strings.Repeat("b", 64),
		BlockTime: time.Date(2026, 11, 2, 18, 0, 0, 0, time.UTC), Finality: "finalized",
		ObservedAt: time.Date(2026, 11, 2, 18, 1, 0, 0, time.UTC),
	}}}}
	_, transport := authconnect.NewTreasuryAdminHandler(treasuryadmin.NewConnectHandler(authorizer, wallets, verifier))
	server := httptest.NewServer(transport)
	defer server.Close()
	client := authconnect.NewTreasuryAdminClient(server.Client(), server.URL)

	request := connect.NewRequest(&authv1.VerifyBaseUSDCReceiptRequest{WalletId: "wallet-1", TransactionHash: "0x" + strings.Repeat("a", 64)})
	request.Header().Set(operatorauth.HeaderAgentToken, "agent")
	_, err = client.VerifyBaseUSDCReceipt(context.Background(), request)
	require.Error(t, err)
	require.Zero(t, wallets.calls)
	require.Zero(t, verifier.calls)

	request = connect.NewRequest(&authv1.VerifyBaseUSDCReceiptRequest{WalletId: "wallet-1", TransactionHash: "0x" + strings.Repeat("a", 64)})
	request.Header().Set(operatorauth.HeaderOperatorToken, "test-operator-secret")
	response, err := client.VerifyBaseUSDCReceipt(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, 1, wallets.calls)
	require.Equal(t, 1, verifier.calls)
	require.Equal(t, "local-operator", wallets.operator)
	require.Equal(t, "local-operator", verifier.operator)
	require.Equal(t, "wallet-1", verifier.wallet.ID)
	require.Equal(t, "0x"+strings.Repeat("a", 64), verifier.txHash)
	require.Equal(t, "receipt-check-1", response.Msg.CheckId)
	require.Equal(t, "verified", response.Msg.CheckOutcome)
	require.Len(t, response.Msg.Transfers, 1)
	require.Equal(t, "5901522", response.Msg.Transfers[0].AtomicAmount)
	require.Equal(t, "finalized", response.Msg.Transfers[0].Finality)
}

func TestListBaseUSDCReceiptChecksRequiresOperatorAndReturnsBoundHistory(t *testing.T) {
	authorizer, err := operatorauth.NewStaticToken("test-operator-secret")
	require.NoError(t, err)
	wallets := &walletStub{network: baseusdc.Network}
	hash := "0x" + strings.Repeat("a", 64)
	checks := &receiptServiceStub{checks: []baseusdc.Check{{
		ID: "receipt-check-1", OperatorIdentity: "local-operator", WalletID: "wallet-1", Network: baseusdc.Network,
		TransactionHash: hash, ProviderID: "fixture-base-rpc", Outcome: "verified", CheckedAt: time.Date(2026, 11, 2, 18, 1, 0, 0, time.UTC),
		Transfers: []baseusdc.Observation{{AtomicAmount: "5901522", Finality: "finalized"}},
	}}}
	_, transport := authconnect.NewTreasuryAdminHandler(treasuryadmin.NewConnectHandler(authorizer, wallets, checks))
	server := httptest.NewServer(transport)
	defer server.Close()
	client := authconnect.NewTreasuryAdminClient(server.Client(), server.URL)
	request := connect.NewRequest(&authv1.ListBaseUSDCReceiptChecksRequest{WalletId: "wallet-1", TransactionHash: hash})
	request.Header().Set(operatorauth.HeaderAgentToken, "agent")
	_, err = client.ListBaseUSDCReceiptChecks(context.Background(), request)
	require.Error(t, err)
	require.Zero(t, wallets.calls)

	request = connect.NewRequest(&authv1.ListBaseUSDCReceiptChecksRequest{WalletId: "wallet-1", TransactionHash: hash})
	request.Header().Set(operatorauth.HeaderOperatorToken, "test-operator-secret")
	response, err := client.ListBaseUSDCReceiptChecks(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, "local-operator", checks.operator)
	require.Equal(t, hash, checks.txHash)
	require.Len(t, response.Msg.Checks, 1)
	require.Equal(t, "receipt-check-1", response.Msg.Checks[0].Id)
	require.Equal(t, "5901522", response.Msg.Checks[0].Transfers[0].AtomicAmount)
}
