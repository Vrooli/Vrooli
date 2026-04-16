package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"resource-gemini/cli/internal/auth"
	"resource-gemini/cli/internal/config"
	resourceenv "resource-gemini/cli/internal/env"
)

// HTTPClient is the narrow HTTP client contract used for safe Gemini probes.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Result summarizes a safe Gemini connectivity probe.
type Result struct {
	Status        string
	Message       string
	Endpoint      string
	HTTPStatus    int
	Authenticated bool
}

// Probe performs a provider-safe GET against the Gemini models endpoint. It
// does not mutate remote state or consume generate-content quota.
func Probe(ctx context.Context, client HTTPClient, runtime resourceenv.Runtime, creds auth.Credentials) (Result, error) {
	if client == nil {
		client = http.DefaultClient
	}

	endpoint, err := modelsURL(runtime.APIBaseURL, creds)
	if err != nil {
		return Result{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Result{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return Result{
			Status:        "unreachable",
			Message:       err.Error(),
			Endpoint:      endpoint,
			Authenticated: creds.Valid(),
		}, err
	}
	defer resp.Body.Close()

	result := Result{
		Endpoint:      endpoint,
		HTTPStatus:    resp.StatusCode,
		Authenticated: creds.Valid(),
	}
	switch resp.StatusCode {
	case http.StatusOK, http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden:
		result.Status = "reachable"
		result.Message = "Gemini API responded"
	default:
		result.Status = "degraded"
		result.Message = fmt.Sprintf("unexpected Gemini status: %d", resp.StatusCode)
	}
	return result, nil
}

// ListModels performs a safe model listing request and returns canonical model names.
func ListModels(ctx context.Context, client HTTPClient, runtime resourceenv.Runtime, creds auth.Credentials) ([]string, error) {
	if client == nil {
		client = http.DefaultClient
	}

	endpoint, err := modelsURL(runtime.APIBaseURL, creds)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list Gemini models: unexpected status %d", resp.StatusCode)
	}

	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(payload.Models))
	for _, model := range payload.Models {
		name := model.Name
		if len(name) > len("models/") && name[:len("models/")] == "models/" {
			name = name[len("models/"):]
		}
		if name != "" {
			models = append(models, name)
		}
	}
	return models, nil
}

func modelsURL(apiBase string, creds auth.Credentials) (string, error) {
	base := config.ModelsEndpoint(apiBase)
	if base == "" {
		return "", fmt.Errorf("Gemini models endpoint is required")
	}
	if !creds.Valid() {
		return base, nil
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("key", creds.APIKey)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
