package manifest

import (
	"encoding/json"
	"net/url"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access for manifest operations.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new manifest client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// Validate sends a manifest to the API for validation.
func (c *Client) Validate(manifest map[string]interface{}) ([]byte, ValidateResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/manifest/validate", nil, manifest)
	if err != nil {
		return nil, ValidateResponse{}, err
	}
	var resp ValidateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, ValidateResponse{}, err
	}
	return body, resp, nil
}

// Schema fetches the canonical schema document for cloud manifests.
func (c *Client) Schema() ([]byte, SchemaResponse, error) {
	body, err := c.api.Get("/api/v1/manifest/schema", nil)
	if err != nil {
		return nil, SchemaResponse{}, err
	}
	var resp SchemaResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, SchemaResponse{}, err
	}
	return body, resp, nil
}

// Init creates a starter manifest from optional input fields.
func (c *Client) Init(req InitRequest) ([]byte, InitResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/manifest/init", nil, req)
	if err != nil {
		return nil, InitResponse{}, err
	}
	var resp InitResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, InitResponse{}, err
	}
	return body, resp, nil
}

// Template fetches the built-in manifest template.
func (c *Client) Template(variant string) ([]byte, TemplateResponse, error) {
	query := url.Values{}
	if variant != "" {
		query.Set("variant", variant)
	}
	body, err := c.api.Get("/api/v1/manifest/template", query)
	if err != nil {
		return nil, TemplateResponse{}, err
	}
	var resp TemplateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, TemplateResponse{}, err
	}
	return body, resp, nil
}

// Doctor analyzes and normalizes a manifest, returning issues and suggested fixes.
func (c *Client) Doctor(manifest map[string]interface{}) ([]byte, DoctorResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/manifest/doctor", nil, DoctorRequest{Manifest: manifest})
	if err != nil {
		return nil, DoctorResponse{}, err
	}
	var resp DoctorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, DoctorResponse{}, err
	}
	return body, resp, nil
}

// Fix analyzes and returns the auto-fixed normalized manifest.
func (c *Client) Fix(manifest map[string]interface{}) ([]byte, DoctorResponse, error) {
	body, err := c.api.Request("POST", "/api/v1/manifest/fix", nil, DoctorRequest{Manifest: manifest})
	if err != nil {
		return nil, DoctorResponse{}, err
	}
	var resp DoctorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, DoctorResponse{}, err
	}
	return body, resp, nil
}
