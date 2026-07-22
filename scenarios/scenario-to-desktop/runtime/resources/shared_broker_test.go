package resources

import (
	"context"
	"testing"
	"time"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

type testSharedBrokerClient struct {
	resource string
	grant    SharedBrokerGrant
}

func (c *testSharedBrokerClient) GrantSharedService(_ context.Context, resource string, _ time.Duration) (SharedBrokerGrant, error) {
	c.resource = resource
	return c.grant, nil
}

func TestBrokerSharedServiceResolverUsesScopedBinding(t *testing.T) {
	client := &testSharedBrokerClient{grant: SharedBrokerGrant{
		Endpoint: "http://127.0.0.1:8200", Credential: "vault-app-token", ExpiresAt: time.Now().Add(time.Minute),
	}}
	resolver := BrokerSharedServiceResolver{SharedReuseConsented: true, Resources: map[string]BrokerSharedServiceConfig{
		"vault": {Client: client, Environment: map[string]string{
			"VAULT_ADDR": "${RESOURCE_ENDPOINT}", "VAULT_TOKEN": "${RESOURCE_CREDENTIAL}",
		}},
	}}
	binding, err := resolver.ResolveSharedService(context.Background(), Item{Resource: "vault", Service: &Service{ProviderPolicy: resourcedeployment.ProviderPolicy{
		DefaultMode:                resourcedeployment.ProviderManagedPrivate,
		AllowedModes:               []resourcedeployment.ProviderMode{resourcedeployment.ProviderManagedPrivate, resourcedeployment.ProviderManagedShared},
		SharedReuseRequiresConsent: true,
		ExternalManagement:         "forbidden",
	}}})
	if err != nil {
		t.Fatalf("ResolveSharedService: %v", err)
	}
	if client.resource != "vault" || binding.Endpoint != "http://127.0.0.1:8200" || binding.Environment["VAULT_TOKEN"] != "vault-app-token" {
		t.Fatalf("binding = %#v, resource = %q", binding, client.resource)
	}
}

func TestBrokerSharedServiceResolverRejectsPolicyWithoutSharedMode(t *testing.T) {
	resolver := BrokerSharedServiceResolver{}
	_, err := resolver.ResolveSharedService(context.Background(), Item{Resource: "vault", Service: &Service{ProviderPolicy: resourcedeployment.ProviderPolicy{
		DefaultMode:        resourcedeployment.ProviderManagedPrivate,
		AllowedModes:       []resourcedeployment.ProviderMode{resourcedeployment.ProviderManagedPrivate},
		ExternalManagement: "forbidden",
	}}})
	if err == nil {
		t.Fatal("expected managed-shared policy rejection")
	}
}

func TestBrokerSharedServiceResolverRequiresExplicitConsent(t *testing.T) {
	client := &testSharedBrokerClient{grant: SharedBrokerGrant{Endpoint: "http://127.0.0.1:8200", Credential: "scoped", ExpiresAt: time.Now().Add(time.Minute)}}
	resolver := BrokerSharedServiceResolver{Resources: map[string]BrokerSharedServiceConfig{"vault": {Client: client}}}
	_, err := resolver.ResolveSharedService(context.Background(), Item{Resource: "vault", Service: &Service{ProviderPolicy: resourcedeployment.ProviderPolicy{
		DefaultMode:                resourcedeployment.ProviderManagedPrivate,
		AllowedModes:               []resourcedeployment.ProviderMode{resourcedeployment.ProviderManagedPrivate, resourcedeployment.ProviderManagedShared},
		SharedReuseRequiresConsent: true,
		ExternalManagement:         "forbidden",
	}}})
	if err == nil {
		t.Fatal("expected explicit-consent rejection")
	}
	if client.resource != "" {
		t.Fatalf("broker client was used without consent for resource %q", client.resource)
	}
}
