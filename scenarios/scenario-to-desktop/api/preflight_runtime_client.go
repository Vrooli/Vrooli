package main

import (
	"net/http"
	"time"

	runtimeapi "scenario-to-desktop-runtime/api"
	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

type PreflightRuntimeClient interface {
	Status() (*BundlePreflightRuntime, error)
	Validate() (*runtimeapi.BundleValidationResult, error)
	Secrets() ([]BundlePreflightSecret, error)
	ApplySecrets(secrets map[string]string) error
	Ready(request BundlePreflightRequest, timeout time.Duration) (BundlePreflightReady, int, error)
	Ports() (map[string]map[string]int, error)
	Telemetry() (*BundlePreflightTelemetry, error)
	LogTails(request BundlePreflightRequest) []BundlePreflightLogTail
}

type runtimeClient struct {
	httpClient *http.Client
	baseURL    string
	token      string
	manifest   *bundlemanifest.Manifest
}

func newRuntimeClient(baseURL, token string, manifest *bundlemanifest.Manifest) *runtimeClient {
	return &runtimeClient{
		httpClient: &http.Client{Timeout: 2 * time.Second},
		baseURL:    baseURL,
		token:      token,
		manifest:   manifest,
	}
}

func (c *runtimeClient) Status() (*BundlePreflightRuntime, error) {
	var status BundlePreflightRuntime
	if _, err := fetchJSON(c.httpClient, c.baseURL, c.token, "/status", http.MethodGet, nil, &status, nil); err != nil {
		return nil, err
	}
	return &status, nil
}

func (c *runtimeClient) Validate() (*runtimeapi.BundleValidationResult, error) {
	var validation runtimeapi.BundleValidationResult
	allowStatus := map[int]bool{http.StatusUnprocessableEntity: true}
	if _, err := fetchJSON(c.httpClient, c.baseURL, c.token, "/validate", http.MethodGet, nil, &validation, allowStatus); err != nil {
		return nil, err
	}
	return &validation, nil
}

func (c *runtimeClient) Secrets() ([]BundlePreflightSecret, error) {
	var resp struct {
		Secrets []BundlePreflightSecret `json:"secrets"`
	}
	if _, err := fetchJSON(c.httpClient, c.baseURL, c.token, "/secrets", http.MethodGet, nil, &resp, nil); err != nil {
		return nil, err
	}
	return resp.Secrets, nil
}

func (c *runtimeClient) ApplySecrets(secrets map[string]string) error {
	filtered := map[string]string{}
	for key, value := range secrets {
		if value == "" {
			continue
		}
		filtered[key] = value
	}
	if len(filtered) == 0 {
		return nil
	}
	payload := map[string]map[string]string{"secrets": filtered}
	_, err := fetchJSON(c.httpClient, c.baseURL, c.token, "/secrets", http.MethodPost, payload, nil, nil)
	return err
}

func (c *runtimeClient) Ready(request BundlePreflightRequest, timeout time.Duration) (BundlePreflightReady, int, error) {
	return fetchReadyWithPolling(c.httpClient, c.baseURL, c.token, request, timeout, c.manifest)
}

func (c *runtimeClient) Ports() (map[string]map[string]int, error) {
	var resp struct {
		Services map[string]map[string]int `json:"services"`
	}
	if _, err := fetchJSON(c.httpClient, c.baseURL, c.token, "/ports", http.MethodGet, nil, &resp, nil); err != nil {
		return nil, err
	}
	return resp.Services, nil
}

func (c *runtimeClient) Telemetry() (*BundlePreflightTelemetry, error) {
	var telemetry BundlePreflightTelemetry
	if _, err := fetchJSON(c.httpClient, c.baseURL, c.token, "/telemetry", http.MethodGet, nil, &telemetry, nil); err != nil {
		return nil, err
	}
	return &telemetry, nil
}

func (c *runtimeClient) LogTails(request BundlePreflightRequest) []BundlePreflightLogTail {
	if c.manifest == nil {
		return nil
	}
	return collectLogTails(c.httpClient, c.baseURL, c.token, c.manifest, request)
}
