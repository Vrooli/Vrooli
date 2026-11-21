package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type BrowserAutomationClient struct{ BaseClient }

func NewBrowserAutomationClient(timeout time.Duration) *BrowserAutomationClient {
	return &BrowserAutomationClient{BaseClient: NewBaseClient("browser-automation-studio", timeout)}
}

type uxBundleManifest struct {
	SchemaVersion string `json:"schema_version"`
	ID            string `json:"id"`
	Source        struct {
		Kind     string `json:"kind"`
		Scenario string `json:"scenario"`
		Route    string `json:"route"`
	} `json:"source"`
	Privacy struct {
		RedactionStatus string `json:"redaction_status"`
	} `json:"privacy"`
	Lenses map[string]struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	} `json:"lenses"`
	Artifacts []struct {
		Kind      string `json:"kind"`
		Path      string `json:"path"`
		SHA256    string `json:"sha256"`
		SizeBytes int64  `json:"size_bytes"`
	} `json:"artifacts"`
}

type uxCaptureResponse struct {
	Manifest             uxBundleManifest `json:"manifest"`
	FullPagePixelsStatus struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	} `json:"full_page_pixels_status"`
}

// CaptureScenarioBundle asks BAS to capture a running scenario and retrieves
// only the full-page pixel artifact declared by its verified manifest.
func (c *BrowserAutomationClient) CaptureScenarioBundle(ctx context.Context, scenario, route string, preset CapturePreset, waitSelector string) ([]byte, error) {
	var response uxCaptureResponse
	err := c.doJSONAccept(ctx, "/api/v1/ux-capture/scenario", map[string]any{
		"scenario": scenario, "route": route, "width": preset.Width, "height": preset.Height,
		"theme": preset.Theme, "wait_selector": waitSelector, "wait_hidden_selector": ".animate-spin", "full_page_pixels": true,
	}, &response, 201)
	if err != nil {
		return nil, fmt.Errorf("capture UX bundle: %w", err)
	}
	m := response.Manifest
	if m.SchemaVersion != "ux-capture.v1" || !validBundleUUID(m.ID) || m.Source.Kind != "scenario" || m.Source.Scenario != scenario || m.Source.Route != redactCaptureRoute(route) || m.Privacy.RedactionStatus != "complete" {
		return nil, fmt.Errorf("UX bundle source or privacy metadata did not match the request")
	}
	if lens := m.Lenses["L0"]; lens.Status != "present" {
		return nil, fmt.Errorf("pixel lens is %s: %s", lens.Status, lens.Reason)
	}
	var artifact *struct {
		Kind      string `json:"kind"`
		Path      string `json:"path"`
		SHA256    string `json:"sha256"`
		SizeBytes int64  `json:"size_bytes"`
	}
	for i := range m.Artifacts {
		if m.Artifacts[i].Kind == "full-page-pixels" {
			artifact = &m.Artifacts[i]
			break
		}
	}
	if artifact == nil || !safeFullPageArtifactPath(artifact.Path) || artifact.SizeBytes <= 0 || artifact.SizeBytes > 20*1024*1024 {
		status := response.FullPagePixelsStatus
		if status.Status == "absent" && status.Reason != "" {
			return nil, fmt.Errorf("full-page pixels unavailable: %s", status.Reason)
		}
		return nil, fmt.Errorf("bundle has no bounded full-page pixel artifact")
	}
	path := "/api/v1/ux-capture/bundles/" + url.PathEscape(m.ID) + "/artifacts/" + escapeArtifactPath(artifact.Path)
	content, contentType, err := c.doRaw(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("fetch full-page pixels: %w", err)
	}
	if contentType != "image/png" || int64(len(content)) != artifact.SizeBytes {
		return nil, fmt.Errorf("full-page pixel artifact metadata mismatch")
	}
	digest := sha256.Sum256(content)
	if hex.EncodeToString(digest[:]) != strings.ToLower(artifact.SHA256) {
		return nil, fmt.Errorf("full-page pixel artifact digest mismatch")
	}
	return content, nil
}

func safeFullPageArtifactPath(value string) bool {
	if !strings.HasPrefix(value, "pixels/full-page-") || !strings.HasSuffix(value, ".png") || strings.ContainsAny(value, "\\\x00") {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." || path.Clean(value) != value {
			return false
		}
	}
	return true
}

func redactCaptureRoute(route string) string {
	parsed, err := url.ParseRequestURI(route)
	if err != nil || parsed.RawQuery == "" {
		return route
	}
	query := parsed.Query()
	for key := range query {
		query.Set(key, "[REDACTED]")
	}
	parsed.RawQuery = query.Encode()
	return parsed.Path + "?" + parsed.RawQuery
}

func validBundleUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, r := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
			continue
		}
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func escapeArtifactPath(path string) string {
	parts := strings.Split(path, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

func (c *BrowserAutomationClient) doRaw(ctx context.Context, path string) ([]byte, string, error) {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("resolve BAS url: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return nil, "", fmt.Errorf("create BAS artifact request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("BAS artifact request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", c.parseError(resp)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 20*1024*1024+1))
	if err != nil {
		return nil, "", fmt.Errorf("read BAS artifact: %w", err)
	}
	if len(data) > 20*1024*1024 {
		return nil, "", fmt.Errorf("BAS artifact exceeds 20 MiB limit")
	}
	return data, resp.Header.Get("Content-Type"), nil
}
