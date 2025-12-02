package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// resolveAnalyzerPort determines the port for the scenario-dependency-analyzer service.
func resolveAnalyzerPort() (string, error) {
	if port := strings.TrimSpace(os.Getenv("SCENARIO_DEPENDENCY_ANALYZER_API_PORT")); port != "" {
		return port, nil
	}

	cmd := exec.Command("vrooli", "scenario", "port", "scenario-dependency-analyzer", "API_PORT")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("SCENARIO_DEPENDENCY_ANALYZER_API_PORT not set and dynamic lookup failed: %w", err)
	}
	port := strings.TrimSpace(string(output))
	if port == "" {
		return "", fmt.Errorf("SCENARIO_DEPENDENCY_ANALYZER_API_PORT not set and dynamic lookup returned empty output")
	}
	return port, nil
}

// fetchSkeletonBundle retrieves the analyzer-emitted desktop bundle skeleton for a scenario.
func (s *Server) fetchSkeletonBundle(ctx context.Context, scenario string) (*desktopBundleManifest, error) {
	port, err := resolveAnalyzerPort()
	if err != nil {
		return nil, err
	}

	target, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%s/api/v1/scenarios/%s/bundle/manifest", port, url.PathEscape(scenario)))
	if err != nil {
		return nil, fmt.Errorf("build analyzer url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create analyzer request: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call analyzer: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<15))
		return nil, fmt.Errorf("analyzer returned %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed struct {
		Manifest json.RawMessage `json:"manifest"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode analyzer response: %w", err)
	}
	if len(parsed.Manifest) == 0 {
		return nil, fmt.Errorf("analyzer response missing manifest")
	}

	// Prefer skeleton field, fall back to manifest directly if already in final form.
	var shape struct {
		Skeleton json.RawMessage `json:"skeleton"`
	}
	_ = json.Unmarshal(parsed.Manifest, &shape)

	var manifestBytes []byte
	if len(shape.Skeleton) > 0 {
		manifestBytes = shape.Skeleton
	} else {
		manifestBytes = parsed.Manifest
	}
	if len(manifestBytes) == 0 {
		return nil, fmt.Errorf("analyzer manifest missing skeleton")
	}

	if err := validateDesktopBundleManifestBytes(manifestBytes); err != nil {
		return nil, fmt.Errorf("analyzer manifest failed validation: %w", err)
	}

	var manifest desktopBundleManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	return &manifest, nil
}

// fetchBundleSecrets retrieves bundle secrets from the secrets-manager service.
func (s *Server) fetchBundleSecrets(ctx context.Context, scenario, tier string) ([]secretsManagerBundleSecret, error) {
	base := os.Getenv("SECRETS_MANAGER_URL")
	if base == "" {
		base = os.Getenv("SECRETS_MANAGER_API_URL")
	}
	if base == "" {
		if port := os.Getenv("SECRETS_MANAGER_API_PORT"); port != "" {
			base = fmt.Sprintf("http://127.0.0.1:%s", port)
		}
	}
	if base == "" {
		return nil, fmt.Errorf("SECRETS_MANAGER_URL or SECRETS_MANAGER_API_URL must be set")
	}

	base = strings.TrimSuffix(base, "/")
	target, err := url.Parse(fmt.Sprintf("%s/api/v1/deployment/secrets/%s", base, url.PathEscape(scenario)))
	if err != nil {
		return nil, fmt.Errorf("build secrets-manager url: %w", err)
	}
	q := target.Query()
	q.Set("tier", tier)
	q.Set("include_optional", "true")
	target.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request secrets-manager: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<16))
		return nil, fmt.Errorf("secrets-manager returned %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed struct {
		BundleSecrets []secretsManagerBundleSecret `json:"bundle_secrets"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode secrets-manager response: %w", err)
	}
	return parsed.BundleSecrets, nil
}
