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

	"deployment-manager/shared"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// BundleSecret represents bundle_secrets from secrets-manager.
type BundleSecret struct {
	ID          string                 `json:"id"`
	LogicalID   string                 `json:"logical_id"`
	Field       string                 `json:"field"`
	Class       string                 `json:"class"`
	Required    bool                   `json:"required"`
	Description string                 `json:"description"`
	Format      string                 `json:"format"`
	Target      Target                 `json:"target"`
	Prompt      *Prompt                `json:"prompt,omitempty"`
	Generator   map[string]interface{} `json:"generator,omitempty"`
}

// Target defines how a secret is injected.
type Target struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// Prompt defines user prompt information for a secret.
type Prompt struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// Client is a client for the secrets-manager service.
type Client struct {
	serviceToken string
}

// NewClient creates a new secrets-manager client.
func NewClient() *Client {
	return &Client{serviceToken: strings.TrimSpace(os.Getenv("SECRETS_MANAGER_DEPLOYMENT_TOKEN"))}
}

// FetchBundleSecrets retrieves bundle secrets from the secrets-manager service.
func (c *Client) FetchBundleSecrets(ctx context.Context, scenario, tier string) ([]BundleSecret, error) {
	base, err := shared.GetConfigResolver().ResolveSecretsManagerURL()
	if err != nil {
		return nil, err
	}

	target, err := url.Parse(fmt.Sprintf("%s/api/v1/deployment/secrets/%s", base, url.PathEscape(scenario)))
	if err != nil {
		return nil, fmt.Errorf("build secrets-manager url: %w", err)
	}
	q := target.Query()
	q.Set("tier", tier)
	q.Set("include_optional", "true")
	target.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, err
	}
	if token := c.token(ctx); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	res, err := shared.GetHTTPClient(ctx).Do(req)
	if err != nil {
		return nil, fmt.Errorf("request secrets-manager: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<16))
		return nil, fmt.Errorf("secrets-manager returned %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed struct {
		BundleSecrets []BundleSecret `json:"bundle_secrets"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode secrets-manager response: %w", err)
	}
	return parsed.BundleSecrets, nil
}

func (c *Client) token(ctx context.Context) string {
	if c.serviceToken != "" {
		return c.serviceToken
	}
	identity, err := credentialauthority.ParseIdentity("vrooli/secrets-manager/deployment")
	if err != nil {
		return ""
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return ""
	}
	token, err := authority.Require(identity, "service-token")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(token)
}
