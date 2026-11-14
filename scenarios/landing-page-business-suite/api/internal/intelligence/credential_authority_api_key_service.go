package intelligence

import (
	"context"
	"fmt"
	"strings"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

const (
	sharedOpenRouterIdentity = "vrooli/openrouter"
	sharedOpenRouterField    = "api-key"
)

// CredentialAuthorityAPIKeyService resolves the provider credential owned by
// the OpenRouter resource. LPBS remains responsible for entitlement and credit
// accounting, but it does not maintain a second provider-key store for the
// metered path.
type CredentialAuthorityAPIKeyService struct {
	Client credentialclient.Client
}

func (s CredentialAuthorityAPIKeyService) Get(ctx context.Context, provider string) (string, error) {
	if strings.TrimSpace(provider) != "openrouter" {
		return "", fmt.Errorf("provider %q is not owned by the shared OpenRouter authority", provider)
	}
	if s.Client == nil {
		return "", fmt.Errorf("shared OpenRouter credential client is not configured")
	}
	value, err := s.Client.Resolve(ctx, sharedOpenRouterIdentity, sharedOpenRouterField)
	if err != nil {
		return "", fmt.Errorf("resolve shared OpenRouter credential: %w", err)
	}
	if strings.TrimSpace(value) == "" {
		return "", ErrNoAPIKeyConfigured
	}
	return strings.TrimSpace(value), nil
}

var _ APIKeyServicer = CredentialAuthorityAPIKeyService{}
