package secrets

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access for secrets operations.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new secrets client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// LegacyGet retrieves legacy scenario secret plans.
func (c *Client) LegacyGet(scenarioID string, reveal bool) ([]byte, LegacyGetResponse, error) {
	query := url.Values{}
	if reveal {
		query.Set("reveal", "true")
	}
	body, err := c.api.Get(fmt.Sprintf("/api/v1/secrets/%s", scenarioID), query)
	if err != nil {
		return nil, LegacyGetResponse{}, err
	}
	var resp LegacyGetResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, LegacyGetResponse{}, err
	}
	return body, resp, nil
}

// GetLocal reads one local secret from workspace/scenario scope.
func (c *Client) GetLocal(scope, key, scenarioID string, reveal bool) ([]byte, LocalSecretGetResponse, error) {
	query := url.Values{}
	if strings.TrimSpace(scenarioID) != "" {
		query.Set("scenario_id", strings.TrimSpace(scenarioID))
	}
	if reveal {
		query.Set("reveal", "true")
	}
	body, err := c.api.Get(fmt.Sprintf("/api/v1/local-secrets/%s/%s", scope, key), query)
	if err != nil {
		return nil, LocalSecretGetResponse{}, err
	}
	var resp LocalSecretGetResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, LocalSecretGetResponse{}, err
	}
	return body, resp, nil
}

// SetLocal writes one local secret at workspace/scenario scope.
func (c *Client) SetLocal(scope, key, scenarioID string, req LocalSecretSetRequest) ([]byte, LocalSecretSetResponse, error) {
	query := url.Values{}
	if strings.TrimSpace(scenarioID) != "" {
		query.Set("scenario_id", strings.TrimSpace(scenarioID))
	}
	body, err := c.api.Request("PUT", fmt.Sprintf("/api/v1/local-secrets/%s/%s", scope, key), query, req)
	if err != nil {
		return nil, LocalSecretSetResponse{}, err
	}
	var resp LocalSecretSetResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, LocalSecretSetResponse{}, err
	}
	return body, resp, nil
}

// DeleteLocal removes one local secret at workspace/scenario scope.
func (c *Client) DeleteLocal(scope, key, scenarioID string) ([]byte, SecretOperationResponse, error) {
	query := url.Values{}
	if strings.TrimSpace(scenarioID) != "" {
		query.Set("scenario_id", strings.TrimSpace(scenarioID))
	}
	body, err := c.api.Request("DELETE", fmt.Sprintf("/api/v1/local-secrets/%s/%s", scope, key), query, nil)
	if err != nil {
		return nil, SecretOperationResponse{}, err
	}
	var resp SecretOperationResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, SecretOperationResponse{}, err
	}
	return body, resp, nil
}

// GetDeploymentSecret fetches one deployment secret.
func (c *Client) GetDeploymentSecret(deploymentID, key string, reveal bool) ([]byte, DeploymentSecretGetResponse, error) {
	query := url.Values{}
	if reveal {
		query.Set("reveal", "true")
	}
	body, err := c.api.Get(fmt.Sprintf("/api/v1/deployments/%s/secrets/%s", deploymentID, key), query)
	if err != nil {
		return nil, DeploymentSecretGetResponse{}, err
	}
	var resp DeploymentSecretGetResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, DeploymentSecretGetResponse{}, err
	}
	return body, resp, nil
}

// CreateDeploymentSecret creates a deployment secret.
func (c *Client) CreateDeploymentSecret(deploymentID string, req DeploymentSecretCreateRequest) ([]byte, SecretOperationResponse, error) {
	body, err := c.api.Request("POST", fmt.Sprintf("/api/v1/deployments/%s/secrets", deploymentID), nil, req)
	if err != nil {
		return nil, SecretOperationResponse{}, err
	}
	var resp SecretOperationResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, SecretOperationResponse{}, err
	}
	return body, resp, nil
}

// UpdateDeploymentSecret updates a deployment secret.
func (c *Client) UpdateDeploymentSecret(deploymentID, key string, req DeploymentSecretUpdateRequest) ([]byte, SecretOperationResponse, error) {
	body, err := c.api.Request("PUT", fmt.Sprintf("/api/v1/deployments/%s/secrets/%s", deploymentID, key), nil, req)
	if err != nil {
		return nil, SecretOperationResponse{}, err
	}
	var resp SecretOperationResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, SecretOperationResponse{}, err
	}
	return body, resp, nil
}

// DeleteDeploymentSecret deletes a deployment secret.
func (c *Client) DeleteDeploymentSecret(deploymentID, key string, restart bool) ([]byte, SecretOperationResponse, error) {
	req := DeploymentSecretDeleteRequest{
		Confirmation:    "DELETE",
		RestartScenario: restart,
	}
	body, err := c.api.Request("DELETE", fmt.Sprintf("/api/v1/deployments/%s/secrets/%s", deploymentID, key), nil, req)
	if err != nil {
		return nil, SecretOperationResponse{}, err
	}
	var resp SecretOperationResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, SecretOperationResponse{}, err
	}
	return body, resp, nil
}
