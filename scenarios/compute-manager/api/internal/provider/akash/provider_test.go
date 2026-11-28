package akash

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"compute-manager/internal/provider"
	akashclient "compute-manager/internal/provider/akash/client"
	"github.com/vrooli/vrooli/packages/provider-effect-gate"
)

type fixtureClient struct {
	createCalls, releaseCalls int
	request                   LeaseRequest
	receipt                   LeaseReceipt
}

type typedFixtureClient struct {
	createCalls int
	getCalls    int
	lastRef     akashclient.LeaseReference
	lease       akashclient.Lease
	leases      []akashclient.Lease
}

func (c *typedFixtureClient) CreateLease(_ context.Context, request akashclient.CreateRequest) (akashclient.Lease, error) {
	c.createCalls++
	c.lease = akashclient.Lease{Reference: akashclient.LeaseReference{Network: akashclient.NetworkSandbox2, ChainID: "sandbox-2-fixture", IntentID: request.IntentID, ID: request.LeaseID}, State: "active", Rate: request.ExpectedRate}
	return c.lease, nil
}
func (c *typedFixtureClient) GetLease(_ context.Context, ref akashclient.LeaseReference) (akashclient.Lease, error) {
	c.getCalls++
	c.lastRef = ref
	c.lease.Reference = ref
	return c.lease, nil
}
func (c *typedFixtureClient) ListLeases(context.Context) ([]akashclient.Lease, error) {
	return c.leases, nil
}
func (c *typedFixtureClient) ReleaseLease(context.Context, akashclient.LeaseReference) error {
	return nil
}

func walletConfigured(context.Context) (bool, error) { return true, nil }

func (c *fixtureClient) CreateLease(_ context.Context, request LeaseRequest) (LeaseReceipt, error) {
	c.createCalls++
	c.request = request
	return c.receipt, nil
}
func (c *fixtureClient) GetLease(context.Context, string) (LeaseReceipt, error) {
	return c.receipt, nil
}
func (c *fixtureClient) ListLeases(context.Context) ([]LeaseReceipt, error) {
	return []LeaseReceipt{c.receipt}, nil
}
func (c *fixtureClient) ReleaseLease(context.Context, string) error { c.releaseCalls++; return nil }

func TestCreateRequiresExplicitLiveModeAndPersistsStableTargetReceipt(t *testing.T) {
	client := &fixtureClient{receipt: LeaseReceipt{DeploymentID: "fixture-deployment", LeaseID: "fixture-lease", Provider: "akash1fixture", Endpoint: "https://fixture.invalid", IntentID: "intent-1", CreatedAt: time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)}}
	config := Config{Network: "testnet", ChainID: "fixture-sandbox", Endpoint: "https://rpc.fixture.invalid", ProviderAddress: "akash1fixture", WalletCredentialRef: "credential-authority://compute-manager/akash-testnet-wallet"}
	closed, err := effectgate.New(effectgate.Config{})
	if err != nil {
		t.Fatal(err)
	}
	adapter := &Adapter{Config: config, Gate: closed, Client: client}
	spec := provider.Spec{Region: "sandbox", Size: "fixture-small", Image: "fixture-image", Tags: map[string]string{"vrooli-intent-id": "intent-1"}}
	if _, err := adapter.Create(context.Background(), spec); err == nil || !strings.Contains(err.Error(), "effect mode is \"stub\"") {
		t.Fatalf("stub create error=%v", err)
	}
	if client.createCalls != 0 {
		t.Fatal("stub gate called Akash client")
	}

	live, err := effectgate.New(effectgate.Config{Modes: map[string]effectgate.Mode{Name: effectgate.Live}})
	if err != nil {
		t.Fatal(err)
	}
	adapter.Gate = live
	adapter.WalletConfigured = walletConfigured
	created, err := adapter.Create(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if client.createCalls != 1 || client.request.IntentID != "intent-1" || client.request.WalletCredentialRef != config.WalletCredentialRef {
		t.Fatalf("client calls=%d request=%+v", client.createCalls, client.request)
	}
	want := "akash/fixture-sandbox/fixture-deployment/fixture-lease"
	if created.ID != want || created.Tags["akash-provider"] != "akash1fixture" || created.Tags["akash-lease-id"] != "fixture-lease" {
		t.Fatalf("target receipt=%+v, want reference %q", created, want)
	}
}

func TestCreateRequiresPersistedIntentAndTestnetConfiguration(t *testing.T) {
	live, err := effectgate.New(effectgate.Config{Modes: map[string]effectgate.Mode{Name: effectgate.Live}})
	if err != nil {
		t.Fatal(err)
	}
	client := &fixtureClient{}
	base := Config{Network: "testnet", ChainID: "fixture-sandbox", Endpoint: "https://rpc.fixture.invalid", ProviderAddress: "akash1fixture", WalletCredentialRef: "credential-authority://compute-manager/akash-wallet"}
	adapter := &Adapter{Config: base, Gate: live, Client: client, WalletConfigured: walletConfigured}
	if _, err := adapter.Create(context.Background(), provider.Spec{}); err == nil || !strings.Contains(err.Error(), "persisted Compute Manager intent") {
		t.Fatalf("missing intent error=%v", err)
	}
	adapter.Config.Network = "mainnet"
	if _, err := adapter.Create(context.Background(), provider.Spec{Tags: map[string]string{"vrooli-intent-id": "intent"}}); err == nil || !strings.Contains(err.Error(), "testnet") {
		t.Fatalf("mainnet config error=%v", err)
	}
	if client.createCalls != 0 {
		t.Fatal("invalid configuration called Akash client")
	}
}

func TestCreateRefusesWhenWalletMetadataIsMissingOrUnavailable(t *testing.T) {
	live, err := effectgate.New(effectgate.Config{Modes: map[string]effectgate.Mode{Name: effectgate.Live}})
	if err != nil {
		t.Fatal(err)
	}
	client := &fixtureClient{}
	base := Config{Network: "testnet", ChainID: "fixture-sandbox", Endpoint: "https://rpc.fixture.invalid", ProviderAddress: "akash1fixture", WalletCredentialRef: TestnetWalletCredentialRef}
	spec := provider.Spec{Region: "sandbox", Size: "offer", Tags: map[string]string{"vrooli-intent-id": "intent-1"}}
	for name, status := range map[string]func(context.Context) (bool, error){
		"callback missing": nil,
		"not configured":   func(context.Context) (bool, error) { return false, nil },
		"status unknown":   func(context.Context) (bool, error) { return false, errors.New("authority unavailable") },
	} {
		t.Run(name, func(t *testing.T) {
			adapter := &Adapter{Config: base, Gate: live, Client: client, WalletConfigured: status}
			if _, err := adapter.Create(context.Background(), spec); err == nil {
				t.Fatal("create proceeded without positive Credential Authority metadata")
			}
			if client.createCalls != 0 {
				t.Fatalf("missing wallet metadata reached provider client %d times", client.createCalls)
			}
		})
	}
}

func TestRecoverIntentUsesOnlyPersistedExactBidTuple(t *testing.T) {
	ref := akashclient.LeaseReference{Network: akashclient.NetworkSandbox2, ChainID: "sandbox-2-fixture", IntentID: "intent-record-1", ID: akashclient.LeaseID{Owner: "akash1manager", DSeq: "42", GSeq: 2, OSeq: 1, Provider: "akash1bidprovider"}}
	rate := akashclient.Rate{Amount: "0.025", Denom: "uakt"}
	client := &typedFixtureClient{lease: akashclient.Lease{Reference: ref, State: "active", Rate: rate, CreatedAt: time.Date(2026, 11, 24, 15, 0, 0, 0, time.UTC)}}
	adapter := &Adapter{Config: Config{Network: akashclient.NetworkSandbox2, ChainID: ref.ChainID, Endpoint: "https://rest.fixture.invalid", ProviderAddress: ref.ID.Owner, WalletCredentialRef: TestnetWalletCredentialRef}, SandboxClient: client}
	request := provider.IntentRecoveryRequest{IntentID: ref.IntentID, IdempotencyKey: "akash-recovery-key-001", Provider: Name, Facts: map[string]string{"dseq": "42", "gseq": "2", "oseq": "1", "provider": ref.ID.Provider, "rate_amount": rate.Amount, "rate_denom": rate.Denom}}
	instance, found, err := adapter.RecoverIntent(context.Background(), request)
	if err != nil || !found {
		t.Fatalf("exact intent recovery instance=%+v found=%v err=%v", instance, found, err)
	}
	if client.getCalls != 1 || client.lastRef != ref || instance.ID != TargetReferenceForLease(ref) || instance.Tags["vrooli-intent-id"] != ref.IntentID {
		t.Fatalf("recovery did not use/persist exact tuple: calls=%d ref=%+v instance=%+v", client.getCalls, client.lastRef, instance)
	}
	if client.createCalls != 0 || len(client.leases) != 0 {
		t.Fatalf("recovery created or listed leases: %+v", client)
	}

	missing := request
	missing.Facts = map[string]string{"dseq": "42", "gseq": "2", "oseq": "1", "provider": ref.ID.Provider}
	if _, found, err := adapter.RecoverIntent(context.Background(), missing); err == nil || found || client.getCalls != 1 {
		t.Fatalf("recovery without persisted rate was not refused before query: found=%v err=%v calls=%d", found, err, client.getCalls)
	}
	client.lease.Rate = akashclient.Rate{Amount: "0.03", Denom: "uakt"}
	if _, found, err := adapter.RecoverIntent(context.Background(), request); err == nil || found || client.getCalls != 2 {
		t.Fatalf("mismatched lease price was not refused: found=%v err=%v calls=%d", found, err, client.getCalls)
	}
}

func TestDestroyRequiresExplicitLiveModeAndAkashReference(t *testing.T) {
	client := &fixtureClient{}
	adapter := &Adapter{Config: Config{Network: "testnet", ChainID: "fixture-sandbox", Endpoint: "https://rpc.fixture.invalid", ProviderAddress: "akash1fixture", WalletCredentialRef: "credential-authority://compute-manager/akash-wallet"}, Client: client}
	if err := adapter.Destroy(context.Background(), "akash/chain/deploy/lease"); err == nil {
		t.Fatal("missing gate allowed release")
	}
	live, _ := effectgate.New(effectgate.Config{Modes: map[string]effectgate.Mode{Name: effectgate.Live}})
	adapter.Gate = live
	adapter.WalletConfigured = walletConfigured
	if err := adapter.Destroy(context.Background(), "hetzner/id"); err == nil {
		t.Fatal("non-Akash reference accepted")
	}
	if err := adapter.Destroy(context.Background(), "akash/other-chain/deploy/lease"); err == nil {
		t.Fatal("reference from another chain accepted")
	}
	if client.releaseCalls != 0 {
		t.Fatal("invalid release reached client")
	}
	if err := adapter.Destroy(context.Background(), "akash/fixture-sandbox/deploy/lease"); err != nil {
		t.Fatal(err)
	}
	if client.releaseCalls != 1 {
		t.Fatalf("release calls=%d", client.releaseCalls)
	}
}

func TestTypedSandboxReferencePreservesExactLeaseIdentityAndIntent(t *testing.T) {
	ref := akashclient.LeaseReference{Network: akashclient.NetworkSandbox2, ChainID: "sandbox-2-fixture", IntentID: "intent-42", ID: akashclient.LeaseID{Owner: "akash1owner", DSeq: "12345", GSeq: 2, OSeq: 3, Provider: "akash1provider"}}
	encoded := TargetReferenceForLease(ref)
	decoded, err := ParseTargetReference(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded != ref {
		t.Fatalf("round trip=%+v, want %+v", decoded, ref)
	}
	for _, invalid := range []string{"akash/chain/deployment/lease", "akashref:not-base64", TargetReferenceForLease(akashclient.LeaseReference{Network: akashclient.NetworkSandbox2, ChainID: ref.ChainID, ID: ref.ID})} {
		if _, err := ParseTargetReference(invalid); err == nil {
			t.Errorf("accepted incomplete reference %q", invalid)
		}
	}
}

func TestTypedSandboxCreateRefusesWithoutPersistedBidAndLeaseTuple(t *testing.T) {
	live, err := effectgate.New(effectgate.Config{Modes: map[string]effectgate.Mode{Name: effectgate.Live}})
	if err != nil {
		t.Fatal(err)
	}
	client := &typedFixtureClient{}
	adapter := &Adapter{Config: Config{Network: akashclient.NetworkSandbox2, ChainID: "sandbox-2-fixture", Endpoint: "https://rest.fixture.invalid", ProviderAddress: "akash1owner", WalletCredentialRef: TestnetWalletCredentialRef}, Gate: live, SandboxClient: client, WalletConfigured: walletConfigured}
	spec := provider.Spec{Tags: map[string]string{"vrooli-intent-id": "intent-1"}}
	if _, err := adapter.Create(context.Background(), spec); err == nil || !strings.Contains(err.Error(), "deployment/bid tuple") {
		t.Fatalf("missing persisted bid facts error=%v", err)
	}
	if client.createCalls != 0 {
		t.Fatal("incomplete request reached Akash client")
	}
}

func TestTypedSandboxListRefusesUnattributedOnChainLeases(t *testing.T) {
	client := &typedFixtureClient{leases: []akashclient.Lease{{Reference: akashclient.LeaseReference{Network: akashclient.NetworkSandbox2, ChainID: "sandbox-2-fixture", ID: akashclient.LeaseID{Owner: "akash1owner", DSeq: "123", GSeq: 1, OSeq: 1, Provider: "akash1provider"}}, State: "active", Rate: akashclient.Rate{Amount: "1", Denom: "uakt"}}}}
	adapter := &Adapter{Config: Config{Network: akashclient.NetworkSandbox2, ChainID: "sandbox-2-fixture", Endpoint: "https://rest.fixture.invalid", ProviderAddress: "akash1owner", WalletCredentialRef: TestnetWalletCredentialRef}, SandboxClient: client}
	if _, err := adapter.List(context.Background()); err == nil || !strings.Contains(err.Error(), "no persisted Compute Manager intent-to-bid mapping") {
		t.Fatalf("unattributed inventory error=%v", err)
	}
}

func TestTypedSandboxDescribeUsesExactPersistedReference(t *testing.T) {
	ref := akashclient.LeaseReference{Network: akashclient.NetworkSandbox2, ChainID: "sandbox-2-fixture", IntentID: "intent-9", ID: akashclient.LeaseID{Owner: "akash1owner", DSeq: "123", GSeq: 2, OSeq: 1, Provider: "akash1provider"}}
	client := &typedFixtureClient{lease: akashclient.Lease{Reference: ref, State: "active", Rate: akashclient.Rate{Amount: "23.5", Denom: "uakt"}}}
	adapter := &Adapter{Config: Config{Network: akashclient.NetworkSandbox2, ChainID: ref.ChainID, Endpoint: "https://rest.fixture.invalid", ProviderAddress: ref.ID.Owner, WalletCredentialRef: TestnetWalletCredentialRef}, SandboxClient: client}
	instance, err := adapter.Describe(context.Background(), TargetReferenceForLease(ref))
	if err != nil {
		t.Fatal(err)
	}
	if client.getCalls != 1 || client.lastRef != ref || instance.Tags["vrooli-intent-id"] != ref.IntentID || instance.Tags["akash-rate-denom"] != "uakt" {
		t.Fatalf("typed readback calls=%d ref=%+v instance=%+v", client.getCalls, client.lastRef, instance)
	}
	wrongOwner := ref
	wrongOwner.ID.Owner = "akash1other"
	if _, err := adapter.Describe(context.Background(), TargetReferenceForLease(wrongOwner)); err == nil {
		t.Fatal("reference for a different owner passed adapter validation")
	}
	if client.getCalls != 1 {
		t.Fatal("invalid owner reference reached Akash client")
	}
}
