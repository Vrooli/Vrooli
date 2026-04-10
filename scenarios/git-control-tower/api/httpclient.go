package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// BaseClient provides shared HTTP helpers for service clients.
type BaseClient struct {
	httpClient  *http.Client
	serviceName string
	resolver    *discovery.Resolver
}

// NewBaseClient creates a BaseClient for the named service.
func NewBaseClient(serviceName string, timeout time.Duration) BaseClient {
	return BaseClient{
		httpClient:  &http.Client{Timeout: timeout},
		serviceName: serviceName,
	}
}

func (b *BaseClient) resolveBaseURL(ctx context.Context) (string, error) {
	if b.resolver != nil {
		return b.resolver.ResolveScenarioURLDefault(ctx, b.serviceName)
	}
	return discovery.ResolveScenarioURLDefault(ctx, b.serviceName)
}

func (b *BaseClient) doGet(ctx context.Context, path string, result interface{}) error {
	baseURL, err := b.resolveBaseURL(ctx)
	if err != nil {
		return fmt.Errorf("resolve %s url: %w", b.serviceName, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s request failed: %w", b.serviceName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return b.parseError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (b *BaseClient) doJSON(ctx context.Context, path string, body, result interface{}) error {
	baseURL, err := b.resolveBaseURL(ctx)
	if err != nil {
		return fmt.Errorf("resolve %s url: %w", b.serviceName, err)
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s request failed: %w", b.serviceName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return b.parseError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (b *BaseClient) doJSONAccept(ctx context.Context, path string, body, result interface{}, acceptStatuses ...int) error {
	baseURL, err := b.resolveBaseURL(ctx)
	if err != nil {
		return fmt.Errorf("resolve %s url: %w", b.serviceName, err)
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s request failed: %w", b.serviceName, err)
	}
	defer resp.Body.Close()

	accepted := false
	for _, s := range acceptStatuses {
		if resp.StatusCode == s {
			accepted = true
			break
		}
	}
	if !accepted {
		return b.parseError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (b *BaseClient) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.Error != "" {
			return fmt.Errorf("%s error: %s", b.serviceName, errResp.Error)
		}
		if errResp.Message != "" {
			return fmt.Errorf("%s error: %s", b.serviceName, errResp.Message)
		}
	}
	return fmt.Errorf("%s error: status %d, body: %s", b.serviceName, resp.StatusCode, string(body))
}
