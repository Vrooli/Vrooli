package monetization

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"

	"github.com/vrooli/api-core/consumeridentity"
	"github.com/vrooli/vrooli/packages/credentialclient-go"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
)

type memoryIssuerCredentialClient struct{ values map[string]string }

func (m *memoryIssuerCredentialClient) Resolve(_ context.Context, identity, field string) (string, error) {
	return m.values[identity+"/"+field], nil
}

func (m *memoryIssuerCredentialClient) Provision(_ context.Context, request credentialclient.ProvisionRequest) (credentialclient.ProvisionResponse, error) {
	m.values[request.Identity+"/"+request.Field] = request.Value
	return credentialclient.ProvisionResponse{Identity: request.Identity, Field: request.Field}, nil
}

func (m *memoryIssuerCredentialClient) Status(_ context.Context, identity, field string) (credentialclient.CredentialStatus, error) {
	configured := m.values[identity+"/"+field] != ""
	return credentialclient.CredentialStatus{Identity: identity, Field: field, Configured: configured, ProviderState: "available"}, nil
}

func TestCredentialIssuerKeyStoreSeparatesIssuerTrustFromMutableLeaseFiles(t *testing.T) {
	client := &memoryIssuerCredentialClient{values: make(map[string]string)}
	store := NewCredentialIssuerKeyStore(client)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keys := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: "issuer", Key: &key.PublicKey})
	jwks, err := keys.JWKS()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(context.Background(), "https://lpbs.test"); !errors.Is(err, entitlementclient.ErrLeaseTrustUnavailable) {
		t.Fatalf("unseeded trust load = %v, want trust unavailable", err)
	}
	if err := store.Save(context.Background(), "https://LPBS.test/", jwks); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load(context.Background(), "https://lpbs.test")
	if err != nil || string(got) != string(jwks) {
		t.Fatalf("load normalized issuer keys = %q, %v", got, err)
	}
	if err := store.Save(context.Background(), "http://lpbs.test", jwks); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(context.Background(), "http://lpbs.test"); err != nil {
		t.Fatalf("load distinct scheme key set = %v", err)
	}
	if len(client.values) != 1 {
		t.Fatalf("issuer keysets used %d credential records, want one declared record", len(client.values))
	}
}
