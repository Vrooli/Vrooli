package ai

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/credits"
)

// AIClientFactory creates AI clients for each request.
// This allows per-request configuration like BYOK keys while maintaining
// a shared chain configuration.
type AIClientFactory struct {
	chain *AIProviderChain
	log   *logrus.Logger
}

// AIClientFactoryOptions configures the factory.
type AIClientFactoryOptions struct {
	Chain  *AIProviderChain
	Logger *logrus.Logger
}

// NewAIClientFactory creates a new factory.
func NewAIClientFactory(opts AIClientFactoryOptions) *AIClientFactory {
	return &AIClientFactory{
		chain: opts.Chain,
		log:   opts.Logger,
	}
}

// CreateClient creates an AI client for a specific request context.
// The returned client handles provider selection and credit charging internally.
func (f *AIClientFactory) CreateClient(opts ClientOptions) AIClient {
	return &chainClient{
		chain:         f.chain,
		log:           f.log,
		userIdentity:  opts.UserIdentity,
		byokKey:       opts.BYOKKey,
		model:         opts.Model,
		operationType: opts.OperationType,
	}
}

// ClientOptions configures a per-request AI client.
type ClientOptions struct {
	// UserIdentity identifies the user for credit tracking.
	UserIdentity string

	// BYOKKey is the user's OpenRouter API key (optional).
	// If provided and valid, BYOK provider will be tried first.
	BYOKKey string

	// Model is the preferred AI model (optional).
	Model string

	// OperationType specifies which operation type to charge credits for.
	// If empty, defaults to OpAIWorkflowGenerate.
	OperationType credits.OperationType
}

// chainClient wraps the provider chain to implement AIClient.
type chainClient struct {
	chain         *AIProviderChain
	log           *logrus.Logger
	userIdentity  string
	byokKey       string
	model         string
	operationType credits.OperationType
}

// ExecutePrompt implements AIClient.
func (c *chainClient) ExecutePrompt(ctx context.Context, prompt string) (string, error) {
	result, err := c.chain.Execute(ctx, ProviderRequest{
		UserIdentity:  c.userIdentity,
		BYOKKey:       c.byokKey,
		Model:         c.model,
		Prompt:        prompt,
		OperationType: c.operationType,
	})
	if err != nil {
		return "", err
	}

	c.log.WithFields(logrus.Fields{
		"provider":        result.Provider,
		"model":           result.Model,
		"charged_credits": result.ChargedCredits,
	}).Debug("AI request completed via chain")

	return result.Response, nil
}

// Model implements AIClient.
func (c *chainClient) Model() string {
	if c.model != "" {
		return c.model
	}
	return byokDefaultModel
}

// Compile-time interface check
var _ AIClient = (*chainClient)(nil)

// ExecuteWithType creates a client and executes a prompt with a specific operation type.
// This is a convenience method for handlers that need to specify the operation type for credits.
func (f *AIClientFactory) ExecuteWithType(
	ctx context.Context,
	opts ClientOptions,
	prompt string,
	opType credits.OperationType,
) (*ProviderResult, error) {
	// Create provider request
	req := ProviderRequest{
		UserIdentity: opts.UserIdentity,
		BYOKKey:      opts.BYOKKey,
		Model:        opts.Model,
		Prompt:       prompt,
	}

	// Execute through chain (the chain handles credit charging for Vrooli provider)
	// For non-Vrooli providers, we need to handle credits based on opType
	result, err := f.chain.Execute(ctx, req)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetAvailableProviders returns which providers are currently available.
func (f *AIClientFactory) GetAvailableProviders(ctx context.Context) []ProviderType {
	return f.chain.GetAvailableProviders(ctx)
}

// IsAnyProviderAvailable checks if any AI provider is available.
func (f *AIClientFactory) IsAnyProviderAvailable(ctx context.Context) bool {
	return f.chain.IsAnyProviderAvailable(ctx)
}
