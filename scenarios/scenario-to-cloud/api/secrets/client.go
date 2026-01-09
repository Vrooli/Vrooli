package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/vrooli/api-core/discovery"

	"scenario-to-cloud/domain"
)

const (
	// DefaultDeploymentTier is the tier used for cloud deployments.
	DefaultDeploymentTier = "tier-4-saas"
)

// Fetcher defines the interface for fetching deployment secrets.
// This seam enables testing without requiring a live secrets-manager service.
type Fetcher interface {
	FetchBundleSecrets(ctx context.Context, scenario, tier string, resources []string) (*ManagerResponse, error)
	HealthCheck(ctx context.Context) error
}

// Client interfaces with the secrets-manager service to fetch deployment secrets.
// It uses dynamic service discovery to resolve the secrets-manager URL.
// Client implements Fetcher.
type Client struct {
	httpClient *http.Client
}

// Ensure Client implements Fetcher at compile time.
var _ Fetcher = (*Client)(nil)

// ManagerSecret represents a secret as returned by secrets-manager API.
type ManagerSecret struct {
	ID                string                 `json:"id"`
	ResourceName      string                 `json:"resource_name"`
	SecretKey         string                 `json:"secret_key"`
	SecretType        string                 `json:"secret_type"` // env_var, file, password, api_key
	Required          bool                   `json:"required"`
	Classification    string                 `json:"classification"` // infrastructure, integration, user_defined
	Description       string                 `json:"description"`
	ValidationPattern string                 `json:"validation_pattern"`
	OwnerTeam         string                 `json:"owner_team"`
	OwnerContact      string                 `json:"owner_contact"`
	HandlingStrategy  string                 `json:"handling_strategy"` // unspecified, generate, prompt, fetch, strip
	RequiresUserInput bool                   `json:"requires_user_input"`
	TierStrategies    map[string]string      `json:"tier_strategies"`
	GeneratorTemplate map[string]interface{} `json:"generator_template,omitempty"` // Generator config from secrets-manager
}

// ManagerResponse matches the actual response from secrets-manager deployment API.
type ManagerResponse struct {
	Scenario    string          `json:"scenario"`
	Tier        string          `json:"tier"`
	GeneratedAt time.Time       `json:"generated_at"`
	Resources   []string        `json:"resources"`
	Secrets     []ManagerSecret `json:"secrets"` // Note: field is "secrets", not "bundle_secrets"
	Summary     struct {
		TotalSecrets       int            `json:"total_secrets"`
		StrategizedSecrets int            `json:"strategized_secrets"`
		RequiresAction     int            `json:"requires_action"`
		StrategyBreakdown  map[string]int `json:"strategy_breakdown"`
	} `json:"summary"`
	// Transformed secrets - populated by TransformSecrets()
	BundleSecrets []domain.BundleSecretPlan `json:"-"`
}

// NewClient creates a new secrets-manager client.
// The URL is resolved dynamically on each request using service discovery.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// resolveBaseURL resolves the secrets-manager URL using discovery.
// Priority: 1) SECRETS_MANAGER_URL env var, 2) vrooli CLI discovery
func (c *Client) resolveBaseURL(ctx context.Context) (string, error) {
	// Check for explicit environment override
	if envURL := strings.TrimSpace(os.Getenv("SECRETS_MANAGER_URL")); envURL != "" {
		return strings.TrimSuffix(envURL, "/"), nil
	}

	// Use service discovery
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "secrets-manager")
	if err != nil {
		return "", fmt.Errorf("resolve secrets-manager URL: %w", err)
	}

	return strings.TrimSuffix(baseURL, "/"), nil
}

// HealthCheck verifies that secrets-manager is reachable and healthy.
func (c *Client) HealthCheck(ctx context.Context) error {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return fmt.Errorf("secrets-manager health check failed: %w", err)
	}

	endpoint := fmt.Sprintf("%s/health", baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("secrets-manager health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("secrets-manager health check returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

// FetchBundleSecrets retrieves the secrets manifest for a scenario.
// This calls the secrets-manager deployment API to get tier-specific secret strategies.
func (c *Client) FetchBundleSecrets(ctx context.Context, scenario, tier string, resources []string) (*ManagerResponse, error) {
	if tier == "" {
		tier = DefaultDeploymentTier
	}

	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("secrets-manager request failed: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/v1/deployment/secrets/%s", baseURL, url.PathEscape(scenario))

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse secrets URL: %w", err)
	}

	q := u.Query()
	q.Set("tier", tier)
	if len(resources) > 0 {
		q.Set("resources", strings.Join(resources, ","))
	}
	q.Set("include_optional", "false")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create secrets request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("secrets-manager request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("secrets-manager returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result ManagerResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode secrets response: %w", err)
	}

	// Transform secrets from secrets-manager format to domain.BundleSecretPlan format
	result.BundleSecrets = transformSecrets(result.Secrets, tier)

	return &result, nil
}

// transformSecrets converts ManagerSecret slice to domain.BundleSecretPlan slice.
// It applies tier-specific strategy resolution and maps the schema.
func transformSecrets(secrets []ManagerSecret, tier string) []domain.BundleSecretPlan {
	if len(secrets) == 0 {
		return nil
	}

	var plans []domain.BundleSecretPlan
	for _, s := range secrets {
		// Determine the effective strategy for this tier
		strategy := s.HandlingStrategy
		if tierStrategy, ok := s.TierStrategies[tier]; ok {
			strategy = tierStrategy
		}

		// Skip secrets that should be stripped for this tier
		if strategy == "strip" {
			continue
		}

		// Map handling_strategy/classification to class
		class := determineSecretClass(s, strategy)

		// Map secret_type to target type
		targetType := "env"
		if s.SecretType == "file" {
			targetType = "file"
		}

		plan := domain.BundleSecretPlan{
			ID:          s.ID,
			Class:       class,
			Required:    s.Required,
			Description: s.Description,
			Format:      s.ValidationPattern,
			Target: domain.BundleSecretTarget{
				Type: targetType,
				Name: s.SecretKey,
			},
		}

		// Add prompt metadata for user_prompt secrets
		if class == "user_prompt" {
			plan.Prompt = &domain.SecretPromptMetadata{
				Label:       formatSecretLabel(s.SecretKey, s.ResourceName),
				Description: s.Description,
			}
		}

		// Add generator config for per_install_generated secrets
		if class == "per_install_generated" {
			// Use generator_template from secrets-manager if available
			if len(s.GeneratorTemplate) > 0 {
				plan.Generator = s.GeneratorTemplate
			} else {
				// Fallback to inferred generator config
				plan.Generator = map[string]interface{}{
					"type":   inferGeneratorType(s.SecretType, s.SecretKey),
					"length": 32,
				}
			}
		}

		plans = append(plans, plan)
	}

	return plans
}

// determineSecretClass maps handling_strategy and classification to a domain.BundleSecretPlan class.
func determineSecretClass(s ManagerSecret, strategy string) string {
	// Priority: explicit strategy > requires_user_input > classification inference

	switch strategy {
	case "generate":
		return "per_install_generated"
	case "prompt":
		return "user_prompt"
	case "fetch":
		return "remote_fetch"
	}

	// If requires user input, it's a user_prompt secret
	if s.RequiresUserInput {
		return "user_prompt"
	}

	// Infer from classification and secret type
	switch s.Classification {
	case "infrastructure":
		// Infrastructure secrets are typically auto-generated
		// But config values like HOST/DB are set by the system
		if isConfigSecret(s.SecretKey) {
			return "infrastructure"
		}
		if isPasswordSecret(s.SecretType, s.SecretKey) {
			return "per_install_generated"
		}
		return "infrastructure"
	case "integration":
		// Integration secrets (API keys) typically require user input
		return "user_prompt"
	case "user_defined":
		return "user_prompt"
	}

	// Default to infrastructure for unclassified secrets
	return "infrastructure"
}

// isConfigSecret checks if a secret is a configuration value (not a credential).
func isConfigSecret(key string) bool {
	configSuffixes := []string{"_HOST", "_PORT", "_DB", "_DATABASE", "_NAME", "_URL", "_ENDPOINT"}
	upperKey := strings.ToUpper(key)
	for _, suffix := range configSuffixes {
		if strings.HasSuffix(upperKey, suffix) {
			return true
		}
	}
	return false
}

// isPasswordSecret checks if a secret is a password/credential that should be generated.
func isPasswordSecret(secretType, key string) bool {
	if secretType == "password" || secretType == "api_key" {
		return true
	}
	passwordPatterns := []string{"_PASSWORD", "_SECRET", "_KEY", "_TOKEN"}
	upperKey := strings.ToUpper(key)
	for _, pattern := range passwordPatterns {
		if strings.Contains(upperKey, pattern) {
			return true
		}
	}
	return false
}

// formatSecretLabel creates a human-readable label from secret key and resource.
func formatSecretLabel(key, resource string) string {
	// Convert POSTGRES_PASSWORD -> "PostgreSQL Password"
	label := strings.ReplaceAll(key, "_", " ")
	label = titleCaseWords(strings.ToLower(label))
	if resource != "" && !strings.Contains(strings.ToLower(label), strings.ToLower(resource)) {
		return resource + " " + label
	}
	return label
}

func titleCaseWords(input string) string {
	words := strings.Fields(input)
	for i, word := range words {
		runes := []rune(word)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

// inferGeneratorType determines what kind of generator to use for a secret.
func inferGeneratorType(secretType, key string) string {
	if secretType == "password" || strings.Contains(strings.ToUpper(key), "PASSWORD") {
		return "password"
	}
	if secretType == "api_key" || strings.Contains(strings.ToUpper(key), "KEY") {
		return "api_key"
	}
	if strings.Contains(strings.ToUpper(key), "TOKEN") {
		return "token"
	}
	return "random"
}

// BuildManifestSecrets converts a ManagerResponse to domain.ManifestSecrets for the domain.CloudManifest.
func BuildManifestSecrets(resp *ManagerResponse) *domain.ManifestSecrets {
	if resp == nil || len(resp.BundleSecrets) == 0 {
		return nil
	}

	// Count secrets by class
	perInstallGenerated := 0
	userPrompt := 0
	remoteFetch := 0
	infrastructure := 0

	for _, secret := range resp.BundleSecrets {
		switch secret.Class {
		case "per_install_generated":
			perInstallGenerated++
		case "user_prompt":
			userPrompt++
		case "remote_fetch":
			remoteFetch++
		case "infrastructure":
			infrastructure++
		}
	}

	return &domain.ManifestSecrets{
		BundleSecrets: resp.BundleSecrets,
		Summary: domain.SecretsSummary{
			TotalSecrets:        len(resp.BundleSecrets),
			PerInstallGenerated: perInstallGenerated,
			UserPrompt:          userPrompt,
			RemoteFetch:         remoteFetch,
			Infrastructure:      infrastructure,
		},
	}
}
