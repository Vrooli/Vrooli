package intelligence

import (
	"context"
	"fmt"
)

// getOpenRouterClient returns the injected client or builds one from the
// currently configured provider credential.
func (s *MeteredInferenceService) getOpenRouterClient(ctx context.Context) (OpenRouterClient, error) {
	if s.openRouterClient != nil {
		return s.openRouterClient, nil
	}

	apiKey, err := s.apiKeyService.Get(ctx, "openrouter")
	if err != nil {
		return nil, fmt.Errorf("get api key: %w", err)
	}
	if apiKey == "" {
		return nil, ErrNoAPIKeyConfigured
	}

	if s.clientFactory == nil {
		return nil, fmt.Errorf("create OpenRouter client: client factory is not configured")
	}
	return s.clientFactory(apiKey, s.log), nil
}

// HealthCheck verifies the metered inference provider can authenticate with its provider.
func (s *MeteredInferenceService) HealthCheck(ctx context.Context) error {
	client, err := s.getOpenRouterClient(ctx)
	if err != nil {
		return err
	}
	return client.VerifyAPIKey(ctx)
}

// UseOpenRouterClient installs a test or alternate provider client.
func (s *MeteredInferenceService) UseOpenRouterClient(client OpenRouterClient) {
	s.openRouterClient = client
}
