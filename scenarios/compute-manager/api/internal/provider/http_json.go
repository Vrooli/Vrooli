package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// DoJSONRequest sends an authenticated JSON request to a provider API.
func DoJSONRequest(ctx context.Context, token func(context.Context) (string, error), client *http.Client, baseURL, defaultBaseURL, name, method, path string, payload, out any) error {
	if token == nil {
		return fmt.Errorf("%w: %s credential source unavailable", ErrProviderUnavailable, name)
	}
	value, err := token(ctx)
	if err != nil {
		return fmt.Errorf("%w: resolve %s credential: %v", ErrProviderUnavailable, name, err)
	}
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+value)
	req.Header.Set("Content-Type", "application/json")
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := fmt.Sprintf("%s returned %s: %s", name, resp.Status, string(data))
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			return fmt.Errorf("%w: %s", ErrProviderUnavailable, message)
		}
		return fmt.Errorf("%s", message)
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode %s response: %w", name, err)
		}
	}
	return nil
}
