package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const imageToolsHeavyJobTelemetryPath = "/api/v1/jobs/telemetry/heavy"

// HTTPHeavyJobSource reads the image-tools owner telemetry endpoint. The
// bounded request timeout keeps this optional rule from stretching the alert
// evaluator cadence; the endpoint is read-only and report-only.
type HTTPHeavyJobSource struct {
	endpoint string
	client   *http.Client
}

func NewHTTPHeavyJobSource(baseURL string, client *http.Client) (*HTTPHeavyJobSource, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("image-tools heavy-job source: invalid base URL %q", baseURL)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + imageToolsHeavyJobTelemetryPath
	parsed.RawQuery = ""
	parsed.Fragment = ""
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	return &HTTPHeavyJobSource{endpoint: parsed.String(), client: client}, nil
}

func (s *HTTPHeavyJobSource) GetHeavyJobs(ctx context.Context) (HeavyJobSnapshot, error) {
	if s == nil || s.client == nil || s.endpoint == "" {
		return HeavyJobSnapshot{}, fmt.Errorf("image-tools heavy-job source is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint, nil)
	if err != nil {
		return HeavyJobSnapshot{}, fmt.Errorf("build image-tools heavy-job request: %w", err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return HeavyJobSnapshot{}, fmt.Errorf("read image-tools heavy-job telemetry: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
		return HeavyJobSnapshot{}, fmt.Errorf("image-tools heavy-job telemetry returned HTTP %d", resp.StatusCode)
	}
	var wire HeavyJobSnapshot
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	if err := decoder.Decode(&wire); err != nil {
		return HeavyJobSnapshot{}, fmt.Errorf("decode image-tools heavy-job telemetry: %w", err)
	}
	return wire, nil
}
