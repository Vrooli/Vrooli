package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const fixtureChainID = "sandbox-2-fixture-chain"

type fixtureSigner struct {
	t       *testing.T
	intents []TransactionIntent
	after   func(TransactionIntent)
	err     error
}

func (s *fixtureSigner) SignAndBroadcast(_ context.Context, intent TransactionIntent) (TransactionReceipt, error) {
	s.t.Helper()
	s.intents = append(s.intents, intent)
	if s.after != nil {
		s.after(intent)
	}
	if s.err != nil {
		return TransactionReceipt{}, s.err
	}
	return TransactionReceipt{Hash: "FIXTURE-TX-HASH", Code: 0}, nil
}

type fixtureNode struct {
	t          *testing.T
	chainID    string
	leases     []map[string]any
	requests   int
	wrongQuery bool
}

func (n *fixtureNode) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	n.t.Helper()
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/status":
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"node_info": map[string]any{"network": n.chainID}}})
	case marketLeaseList:
		n.requests++
		if r.URL.Query().Get("filters.owner") != "akash1cmfixture" {
			n.wrongQuery = true
			http.Error(w, "owner filter missing", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"leases": n.leases, "pagination": map[string]any{"next_key": ""}})
	default:
		http.NotFound(w, r)
	}
}

func fixtureLease(state, amount, denom string) map[string]any {
	return map[string]any{"lease": map[string]any{
		"lease_id":   map[string]any{"owner": "akash1cmfixture", "dseq": "12345", "gseq": 1, "oseq": 1, "provider": "akash1providerfixture"},
		"state":      state,
		"price":      map[string]any{"amount": amount, "denom": denom},
		"created_at": "2026-11-24T01:30:00Z",
	}}
}

func newFixtureClient(t *testing.T, node *fixtureNode, signer Signer) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(node)
	config := Config{
		Network: NetworkSandbox2, ChainID: fixtureChainID, OwnerAddress: "akash1cmfixture",
		RESTEndpoint: server.URL, RPCEndpoint: server.URL,
		CredentialRef: "credential-authority://compute-manager/akash-testnet-wallet",
	}
	client, err := New(config, server.Client(), signer)
	require.NoError(t, err)
	t.Cleanup(server.Close)
	return client, server
}

func fixtureRequest() CreateRequest {
	return CreateRequest{IntentID: "intent-fixture-01", LeaseID: LeaseID{Owner: "akash1cmfixture", DSeq: "12345", GSeq: 1, OSeq: 1, Provider: "akash1providerfixture"}, ExpectedRate: Rate{Amount: "3500.125", Denom: "uact"}}
}

func TestCreateLeaseBroadcastsTypedIntentThenReadsExactRateAndIdentity(t *testing.T) {
	node := &fixtureNode{t: t, chainID: fixtureChainID}
	signer := &fixtureSigner{t: t, after: func(intent TransactionIntent) {
		require.Equal(t, OperationCreateLease, intent.Operation)
		require.Equal(t, "intent-fixture-01", intent.IntentID)
		require.Equal(t, fixtureRequest().LeaseID, intent.LeaseID)
		require.Equal(t, "credential-authority://compute-manager/akash-testnet-wallet", intent.CredentialRef)
		node.leases = []map[string]any{fixtureLease("active", "3500.125", "uact")}
	}}
	client, _ := newFixtureClient(t, node, signer)

	lease, err := client.CreateLease(context.Background(), fixtureRequest())
	require.NoError(t, err)
	require.Equal(t, fixtureRequest().LeaseID, lease.Reference.ID)
	require.Equal(t, fixtureRequest().IntentID, lease.Reference.IntentID)
	require.Equal(t, NetworkSandbox2, lease.Reference.Network)
	require.Equal(t, fixtureChainID, lease.Reference.ChainID)
	require.Equal(t, "active", lease.State)
	require.Equal(t, Rate{Amount: "3500.125", Denom: "uact"}, lease.Rate)
	require.Equal(t, "2026-11-24T01:30:00Z", lease.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	require.Len(t, signer.intents, 1)
	require.NotZero(t, node.requests)
	require.False(t, node.wrongQuery)
}

func TestCreateLeaseReadbackIsIdempotentAndAvoidsRepeatBroadcast(t *testing.T) {
	node := &fixtureNode{t: t, chainID: fixtureChainID, leases: []map[string]any{fixtureLease("active", "3500.125", "uact")}}
	signer := &fixtureSigner{t: t}
	client, _ := newFixtureClient(t, node, signer)

	lease, err := client.CreateLease(context.Background(), fixtureRequest())
	require.NoError(t, err)
	require.Equal(t, "active", lease.State)
	require.Empty(t, signer.intents)
}

func TestCreateLeaseReconcilesLostBroadcastResponseWithoutSecondTransaction(t *testing.T) {
	node := &fixtureNode{t: t, chainID: fixtureChainID}
	signer := &fixtureSigner{t: t, err: errors.New("fixture broadcast response lost"), after: func(TransactionIntent) {
		node.leases = []map[string]any{fixtureLease("active", "3500.125", "uact")}
	}}
	client, _ := newFixtureClient(t, node, signer)

	lease, err := client.CreateLease(context.Background(), fixtureRequest())
	require.NoError(t, err)
	require.Equal(t, "active", lease.State)
	require.Len(t, signer.intents, 1, "readback reconciles a lost response without a duplicate transaction")
}

func TestAmbiguousOrMismatchedReadbackFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name   string
		leases []map[string]any
	}{
		{name: "duplicate exact lease records", leases: []map[string]any{fixtureLease("active", "3500.125", "uact"), fixtureLease("active", "3500.125", "uact")}},
		{name: "different expected price", leases: []map[string]any{fixtureLease("active", "3500.126", "uact")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			node := &fixtureNode{t: t, chainID: fixtureChainID, leases: tc.leases}
			signer := &fixtureSigner{t: t}
			client, _ := newFixtureClient(t, node, signer)
			_, err := client.CreateLease(context.Background(), fixtureRequest())
			require.Error(t, err)
			require.Empty(t, signer.intents)
		})
	}
}

func TestListGetAndReleaseRequireExactComputeManagerReference(t *testing.T) {
	node := &fixtureNode{t: t, chainID: fixtureChainID, leases: []map[string]any{fixtureLease("active", "3500.125", "uact")}}
	signer := &fixtureSigner{t: t, after: func(intent TransactionIntent) {
		if intent.Operation == OperationCloseLease {
			node.leases = []map[string]any{fixtureLease("closed", "3500.125", "uact")}
		}
	}}
	client, _ := newFixtureClient(t, node, signer)

	leases, err := client.ListLeases(context.Background())
	require.NoError(t, err)
	require.Len(t, leases, 1)
	require.Equal(t, "3500.125", leases[0].Rate.Amount)
	ref := LeaseReference{Network: NetworkSandbox2, ChainID: fixtureChainID, IntentID: "intent-fixture-01", ID: fixtureRequest().LeaseID}
	lease, err := client.GetLease(context.Background(), ref)
	require.NoError(t, err)
	require.Equal(t, "active", lease.State)
	require.NoError(t, client.ReleaseLease(context.Background(), ref))
	require.Len(t, signer.intents, 1)
	require.Equal(t, OperationCloseLease, signer.intents[0].Operation)
	require.Equal(t, "closed", node.leases[0]["lease"].(map[string]any)["state"])
	require.NoError(t, client.ReleaseLease(context.Background(), ref), "release readback is idempotent")
	require.Len(t, signer.intents, 1)
}

func TestReleaseRefusesUnownedOrWrongNetworkLeaseBeforeSigner(t *testing.T) {
	node := &fixtureNode{t: t, chainID: fixtureChainID, leases: []map[string]any{fixtureLease("active", "3500.125", "uact")}}
	signer := &fixtureSigner{t: t}
	client, _ := newFixtureClient(t, node, signer)
	base := LeaseReference{Network: NetworkSandbox2, ChainID: fixtureChainID, IntentID: "intent-fixture-01", ID: fixtureRequest().LeaseID}
	wrongOwner := base
	wrongOwner.ID.Owner = "akash1other"
	missingIntent := base
	missingIntent.IntentID = ""
	wrongChain := base
	wrongChain.ChainID = "akashnet-2"
	for _, ref := range []LeaseReference{wrongOwner, missingIntent, wrongChain} {
		require.Error(t, client.ReleaseLease(context.Background(), ref))
	}
	require.Empty(t, signer.intents)
}

func TestConfigRejectsMainnetDefaultsSecretsAndNonLoopbackHTTP(t *testing.T) {
	valid := Config{Network: NetworkSandbox2, ChainID: fixtureChainID, OwnerAddress: "akash1cmfixture", RESTEndpoint: "https://rest.sandbox.invalid", RPCEndpoint: "https://rpc.sandbox.invalid", CredentialRef: "credential-authority://compute-manager/akash-wallet"}
	for name, mutate := range map[string]func(*Config){
		"mainnet network":                   func(c *Config) { c.Network = "mainnet" },
		"mainnet chain":                     func(c *Config) { c.ChainID = "akashnet-2" },
		"console managed endpoint":          func(c *Config) { c.RESTEndpoint = "https://console-api.akash.network" },
		"missing credential reference":      func(c *Config) { c.CredentialRef = "" },
		"raw token in credential reference": func(c *Config) { c.CredentialRef = "fixture-token-value" },
		"non-loopback HTTP":                 func(c *Config) { c.RESTEndpoint = "http://rpc.sandbox.invalid" },
		"missing endpoint":                  func(c *Config) { c.RPCEndpoint = "" },
	} {
		t.Run(name, func(t *testing.T) {
			config := valid
			mutate(&config)
			_, err := New(config, nil, nil)
			require.Error(t, err)
		})
	}
}

func TestNodeChainIDMismatchRefusesReadBeforeLeaseDecode(t *testing.T) {
	node := &fixtureNode{t: t, chainID: "akashnet-2", leases: []map[string]any{fixtureLease("active", "3500.125", "uact")}}
	client, _ := newFixtureClient(t, node, &fixtureSigner{t: t})
	_, err := client.ListLeases(context.Background())
	require.ErrorContains(t, err, "does not match configured sandbox chain")
	require.Zero(t, node.requests)
}

func TestMalformedLeaseFactsAreRefused(t *testing.T) {
	cases := []map[string]any{
		{"lease": map[string]any{"lease_id": map[string]any{"owner": "akash1cmfixture", "dseq": "12345", "gseq": 1, "oseq": 1, "provider": "akash1providerfixture"}, "state": "active", "price": map[string]any{"amount": "", "denom": "uact"}}},
		{"lease": map[string]any{"lease_id": map[string]any{"owner": "akash1cmfixture", "dseq": "12345", "gseq": 1, "oseq": 1, "provider": "akash1providerfixture"}, "state": "pending", "price": map[string]any{"amount": "1", "denom": "uact"}}},
		{"lease": map[string]any{"lease_id": map[string]any{"owner": "akash1cmfixture", "dseq": "12345", "gseq": 1, "oseq": 1}, "state": "active", "price": map[string]any{"amount": "1", "denom": "uact"}}},
	}
	for i, fixture := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			node := &fixtureNode{t: t, chainID: fixtureChainID, leases: []map[string]any{fixture}}
			client, _ := newFixtureClient(t, node, &fixtureSigner{t: t})
			_, err := client.ListLeases(context.Background())
			require.Error(t, err)
		})
	}
}

func TestReadDoesNotFollowRedirects(t *testing.T) {
	redirects := 0
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirects++
		http.Redirect(w, r, "/status", http.StatusTemporaryRedirect)
	}))
	defer redirectServer.Close()
	config := Config{Network: NetworkSandbox2, ChainID: fixtureChainID, OwnerAddress: "akash1cmfixture", RESTEndpoint: redirectServer.URL, RPCEndpoint: redirectServer.URL, CredentialRef: "credential-authority://compute-manager/akash-wallet"}
	client, err := New(config, redirectServer.Client(), nil)
	require.NoError(t, err)
	_, err = client.ListLeases(context.Background())
	require.Error(t, err)
	require.Equal(t, 1, redirects, "node endpoint redirects are not followed")
}

func TestRateUsesExactDecimalRepresentation(t *testing.T) {
	rate := Rate{Amount: "0.000000000000000001", Denom: "uact"}
	require.NoError(t, rate.Validate())
	require.True(t, strings.Contains(fmt.Sprintf("%#v", rate), rate.Amount))
	require.Error(t, (Rate{Amount: "1.5USD", Denom: "uact"}).Validate())
}
