package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type scenarioBundleCaptureClient struct {
	baseURL    string
	httpClient *http.Client
}

type scenarioBundleResponse struct {
	Manifest struct {
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
			SizeBytes int64  `json:"size_bytes"`
			SHA256    string `json:"sha256"`
		} `json:"artifacts"`
	} `json:"manifest"`
}

func (c scenarioBundleCaptureClient) CaptureScenarioPixels(ctx context.Context, scenario, route string, width, height int) ([]byte, error) {
	body, err := json.Marshal(map[string]any{
		"scenario": scenario, "route": route, "width": width, "height": height,
		"theme": "light", "wait_selector": "body",
	})
	if err != nil {
		return nil, fmt.Errorf("encode scenario bundle request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.baseURL, "/")+"/api/v1/ux-capture/scenario", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create scenario bundle request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request scenario bundle: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("scenario bundle returned HTTP %d", resp.StatusCode)
	}
	var result scenarioBundleResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2*1024*1024)).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode scenario bundle manifest: %w", err)
	}
	m := result.Manifest
	if m.SchemaVersion != "ux-capture.v1" || !validCaptureBundleID(m.ID) || m.Source.Kind != "scenario" || m.Source.Scenario != scenario || m.Source.Route != route || m.Privacy.RedactionStatus != "complete" {
		return nil, fmt.Errorf("bundle source or privacy metadata mismatch")
	}
	if m.Lenses["L0"].Status != "present" {
		return nil, fmt.Errorf("pixel lens unavailable: %s", m.Lenses["L0"].Reason)
	}
	var path, expectedDigest string
	var expectedSize int64
	for _, artifact := range m.Artifacts {
		if artifact.Kind == "pixels" {
			path, expectedDigest, expectedSize = artifact.Path, artifact.SHA256, artifact.SizeBytes
			break
		}
	}
	if !validCaptureArtifactPath(path) || expectedSize <= 0 || expectedSize > 20*1024*1024 {
		return nil, fmt.Errorf("bundle has no bounded viewport pixel artifact")
	}
	artifactURL := strings.TrimRight(c.baseURL, "/") + "/api/v1/ux-capture/bundles/" + url.PathEscape(m.ID) + "/artifacts/" + escapeCaptureArtifactPath(path)
	artifactReq, err := http.NewRequestWithContext(ctx, http.MethodGet, artifactURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create artifact request: %w", err)
	}
	artifactResp, err := c.httpClient.Do(artifactReq)
	if err != nil {
		return nil, fmt.Errorf("fetch viewport pixels: %w", err)
	}
	defer artifactResp.Body.Close()
	if artifactResp.StatusCode != http.StatusOK || artifactResp.Header.Get("Content-Type") != "image/png" {
		return nil, fmt.Errorf("viewport pixel artifact returned unexpected HTTP status or media type")
	}
	pixels, err := io.ReadAll(io.LimitReader(artifactResp.Body, 20*1024*1024+1))
	if err != nil {
		return nil, fmt.Errorf("read viewport pixels: %w", err)
	}
	if int64(len(pixels)) != expectedSize || http.DetectContentType(pixels) != "image/png" {
		return nil, fmt.Errorf("viewport pixel artifact size or PNG signature mismatch")
	}
	digest := sha256.Sum256(pixels)
	if hex.EncodeToString(digest[:]) != strings.ToLower(expectedDigest) {
		return nil, fmt.Errorf("viewport pixel artifact digest mismatch")
	}
	return pixels, nil
}

func validCaptureBundleID(value string) bool {
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

func validCaptureArtifactPath(value string) bool {
	if !strings.HasPrefix(value, "pixels/") || strings.ContainsAny(value, "\\\x00") {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func escapeCaptureArtifactPath(value string) string {
	parts := strings.Split(value, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}
