package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type probeEndpointsRequest struct {
	ProxyURL  string `json:"proxy_url"`
	ServerURL string `json:"server_url"`
	APIURL    string `json:"api_url"`
	TimeoutMs int    `json:"timeout_ms"`
}

type probeEndpointResult struct {
	Status     string `json:"status"`
	StatusCode *int   `json:"status_code,omitempty"`
	Message    string `json:"message,omitempty"`
}

type probeEndpointsResponse struct {
	ProxyURL string              `json:"proxy_url,omitempty"`
	Server   probeEndpointResult `json:"server"`
	API      probeEndpointResult `json:"api"`
}

func (s *Server) probeEndpoints(ctx context.Context, request probeEndpointsRequest) (probeEndpointsResponse, error) {
	if request.ProxyURL != "" {
		normalized, err := normalizeProxyURL(request.ProxyURL)
		if err != nil {
			return probeEndpointsResponse{}, fmt.Errorf("invalid proxy_url: %w", err)
		}
		request.ProxyURL = normalized
		request.ServerURL = normalized
		if request.APIURL == "" {
			request.APIURL = proxyAPIURL(normalized)
		}
	}

	if request.ServerURL == "" && request.APIURL == "" {
		return probeEndpointsResponse{}, fmt.Errorf("provide at least a server_url or api_url to probe")
	}

	timeout := time.Duration(request.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	client := &http.Client{Timeout: timeout}
	probe := func(target string) probeEndpointResult {
		if target == "" {
			return probeEndpointResult{Status: "skipped", Message: "no URL provided"}
		}
		if _, err := url.ParseRequestURI(target); err != nil {
			return probeEndpointResult{Status: "error", Message: fmt.Sprintf("invalid URL: %v", err)}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return probeEndpointResult{Status: "error", Message: err.Error()}
		}

		resp, err := client.Do(req)
		if err != nil {
			return probeEndpointResult{Status: "error", Message: err.Error()}
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return probeEndpointResult{Status: "ok", StatusCode: &resp.StatusCode}
		}
		return probeEndpointResult{Status: "error", StatusCode: &resp.StatusCode, Message: fmt.Sprintf("server returned %d", resp.StatusCode)}
	}

	return probeEndpointsResponse{ProxyURL: request.ProxyURL, Server: probe(request.ServerURL), API: probe(request.APIURL)}, nil
}
