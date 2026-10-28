package resolver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type AdGuardHomeClient struct {
	baseURL   string
	creds     Credentials
	transport http.RoundTripper
	timeout   time.Duration
}

func NewAdGuardHomeClient(baseURL string, creds Credentials, transport http.RoundTripper, timeout time.Duration) *AdGuardHomeClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if transport == nil {
		transport = &http.Transport{Proxy: http.ProxyFromEnvironment}
	}
	return &AdGuardHomeClient{baseURL: strings.TrimRight(baseURL, "/"), creds: creds, transport: transport, timeout: timeout}
}

func (c *AdGuardHomeClient) Do(ctx context.Context, method, endpoint string, body any) (*http.Response, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	var reader io.Reader
	if body != nil {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
		reader = &buf
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.creds.Username != "" || c.creds.Password != "" {
		req.SetBasicAuth(c.creds.Username, c.creds.Password)
	}
	return c.transport.RoundTrip(req)
}

func GetAdGuardJSON[T any](ctx context.Context, client *AdGuardHomeClient, endpoint string) (T, int, error) {
	var zero T
	resp, err := client.Do(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return zero, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return zero, resp.StatusCode, nil
	}
	if err := json.NewDecoder(resp.Body).Decode(&zero); err != nil {
		return zero, resp.StatusCode, fmt.Errorf("decode %s: %w", endpoint, err)
	}
	return zero, resp.StatusCode, nil
}
