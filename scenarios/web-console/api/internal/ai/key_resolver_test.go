package ai

import (
	"context"
	"testing"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

type resolverCredentialClient struct {
	credentialclient.Client
	key string
}

func (c resolverCredentialClient) Resolve(context.Context, string, string) (string, error) {
	return c.key, nil
}

func TestCredentialKeyResolverAuthorityWinsOverEnvironment(t *testing.T) {
	t.Setenv("OPENROUTER"+"_API_KEY", "environment-key")
	key, source, err := NewCredentialKeyResolver(resolverCredentialClient{key: "authority-key"}).Resolve(context.Background())
	if err != nil || key != "authority-key" || source != KeySourceCredentialAuthority {
		t.Fatalf("Resolve() = %q, %q, %v", key, source, err)
	}
}

func TestCredentialKeyResolverUsesEnvironmentOnlyAsFallback(t *testing.T) {
	t.Setenv("OPENROUTER"+"_API_KEY", "environment-key")
	key, source, err := NewCredentialKeyResolver(nil).Resolve(context.Background())
	if err != nil || key != "environment-key" || source != KeySourceEnvironment {
		t.Fatalf("Resolve() = %q, %q, %v", key, source, err)
	}
}
