package androidjourney

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// HTTPWebViewAttacher consumes device-control's lease-scoped WebView verb.
// It deliberately returns only the forwarded CDP endpoint to the BAS client.
type HTTPWebViewAttacher struct {
	BaseURL string
	Actor   string
	Client  *http.Client
}

func (c HTTPWebViewAttacher) AttachWebView(ctx context.Context, lease Lease, packageName string) (WebViewAttachment, error) {
	if strings.TrimSpace(c.BaseURL) == "" {
		return WebViewAttachment{}, fmt.Errorf("device-control URL is not configured")
	}
	body, err := json.Marshal(map[string]string{"actor": c.Actor, "lease_token": lease.Token, "package": packageName})
	if err != nil {
		return WebViewAttachment{}, err
	}
	if strings.TrimSpace(lease.DeviceID) == "" {
		return WebViewAttachment{}, fmt.Errorf("device-control WebView attach requires the leased device identity")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/api/v1/devices/"+urlPathSegment(lease.DeviceID)+"/webview/attach", bytes.NewReader(body))
	if err != nil {
		return WebViewAttachment{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(req)
	if err != nil {
		return WebViewAttachment{}, fmt.Errorf("device-control WebView attach request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 16<<10))
		return WebViewAttachment{}, fmt.Errorf("device-control WebView attach returned %s: %s", response.Status, strings.TrimSpace(string(message)))
	}
	var result struct {
		Endpoint struct {
			CDPEndpoint string `json:"cdp_endpoint"`
			RendererID  string `json:"renderer_id"`
		} `json:"endpoint"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return WebViewAttachment{}, fmt.Errorf("decode device-control WebView endpoint: %w", err)
	}
	if strings.TrimSpace(result.Endpoint.CDPEndpoint) == "" {
		return WebViewAttachment{}, fmt.Errorf("device-control WebView response omitted cdp_endpoint")
	}
	if strings.TrimSpace(result.Endpoint.RendererID) == "" {
		return WebViewAttachment{}, fmt.Errorf("device-control WebView response omitted renderer_id")
	}
	return WebViewAttachment{CDPEndpoint: result.Endpoint.CDPEndpoint, RendererID: result.Endpoint.RendererID}, nil
}

func urlPathSegment(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "/", "%2F")
	return value
}
