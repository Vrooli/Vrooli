package main

import (
	"context"
	"fmt"

	"landing-page-business-suite-api/internal/envx"
)

// getOpenRouterClient returns the injected client or builds one from the
// currently configured provider credential.
func (s *AIGatewayService) getOpenRouterClient(ctx context.Context) (OpenRouterClient, error) {
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

	return NewOpenRouterClient(OpenRouterClientOptions{
		APIKey:  apiKey,
		BaseURL: envx.Get("OPENROUTER_BASE_URL"),
		Referer: envx.Get("OPENROUTER_REFERER"),
		Title:   envx.Get("OPENROUTER_TITLE"),
		Logger:  s.log,
	}), nil
}

// HealthCheck verifies the AI gateway can authenticate with its provider.
func (s *AIGatewayService) HealthCheck(ctx context.Context) error {
	client, err := s.getOpenRouterClient(ctx)
	if err != nil {
		return err
	}
	return client.VerifyAPIKey(ctx)
}

// UseOpenRouterClient installs a test or alternate provider client.
func (s *AIGatewayService) UseOpenRouterClient(client OpenRouterClient) {
	s.openRouterClient = client
}
