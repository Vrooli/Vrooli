package preflight

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	runtimeapi "scenario-to-desktop-runtime/api"
	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

// HTTPRuntimeClient is the default HTTP-based implementation of RuntimeClient.
type HTTPRuntimeClient struct {
	httpClient *http.Client
	baseURL    string
	token      string
	manifest   *bundlemanifest.Manifest
}

// HTTPRuntimeClientOption configures an HTTPRuntimeClient.
type HTTPRuntimeClientOption func(*HTTPRuntimeClient)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) HTTPRuntimeClientOption {
	return func(c *HTTPRuntimeClient) {
		c.httpClient = client
	}
}

// NewHTTPRuntimeClient creates a new HTTP runtime client.
func NewHTTPRuntimeClient(baseURL, token string, manifest *bundlemanifest.Manifest, opts ...HTTPRuntimeClientOption) *HTTPRuntimeClient {
	c := &HTTPRuntimeClient{
		httpClient: &http.Client{Timeout: 2 * time.Second},
		baseURL:    baseURL,
		token:      token,
		manifest:   manifest,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Status returns runtime instance metadata.
func (c *HTTPRuntimeClient) Status() (*Runtime, error) {
	var status Runtime
	if _, err := c.fetchJSON("/status", http.MethodGet, nil, &status, nil); err != nil {
		return nil, err
	}
	return &status, nil
}

// Validate validates the bundle configuration.
func (c *HTTPRuntimeClient) Validate() (*runtimeapi.BundleValidationResult, error) {
	var validation runtimeapi.BundleValidationResult
	allowStatus := map[int]bool{http.StatusUnprocessableEntity: true}
	if _, err := c.fetchJSON("/validate", http.MethodGet, nil, &validation, allowStatus); err != nil {
		return nil, err
	}
	return &validation, nil
}

// Secrets retrieves the list of secrets from the runtime.
func (c *HTTPRuntimeClient) Secrets() ([]Secret, error) {
	var resp struct {
		Secrets []Secret `json:"secrets"`
	}
	if _, err := c.fetchJSON("/secrets", http.MethodGet, nil, &resp, nil); err != nil {
		return nil, err
	}
	return resp.Secrets, nil
}

// ApplySecrets applies secrets to the runtime.
func (c *HTTPRuntimeClient) ApplySecrets(secrets map[string]string) error {
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
	_, err := c.fetchJSON("/secrets", http.MethodPost, payload, nil, nil)
	return err
}

// Ready checks if the runtime is ready, with polling support.
func (c *HTTPRuntimeClient) Ready(request Request, timeout time.Duration) (Ready, int, error) {
	return c.fetchReadyWithPolling(request, timeout)
}

// Ports retrieves the port mappings from the runtime.
func (c *HTTPRuntimeClient) Ports() (map[string]map[string]int, error) {
	var resp struct {
		Services map[string]map[string]int `json:"services"`
	}
	if _, err := c.fetchJSON("/ports", http.MethodGet, nil, &resp, nil); err != nil {
		return nil, err
	}
	return resp.Services, nil
}

// Telemetry retrieves telemetry configuration.
func (c *HTTPRuntimeClient) Telemetry() (*Telemetry, error) {
	var telemetry Telemetry
	if _, err := c.fetchJSON("/telemetry", http.MethodGet, nil, &telemetry, nil); err != nil {
		return nil, err
	}
	return &telemetry, nil
}

// LogTails retrieves log tails for services.
func (c *HTTPRuntimeClient) LogTails(request Request) []LogTail {
	if c.manifest == nil {
		return nil
	}
	return c.collectLogTails(request)
}

// fetchJSON performs an HTTP request and decodes the JSON response.
func (c *HTTPRuntimeClient) fetchJSON(path, method string, payload interface{}, out interface{}, allow map[int]bool) (int, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return 0, err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return 0, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if allow != nil && allow[resp.StatusCode] {
			if out != nil {
				if decodeErr := json.NewDecoder(resp.Body).Decode(out); decodeErr != nil {
					return resp.StatusCode, decodeErr
				}
			}
			return resp.StatusCode, nil
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

// fetchText performs an HTTP GET request and returns the response as text.
func (c *HTTPRuntimeClient) fetchText(path string) (string, int, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return "", 0, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, err
	}
	bodyText := strings.TrimSpace(string(bodyBytes))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if bodyText == "" {
			bodyText = resp.Status
		}
		return bodyText, resp.StatusCode, fmt.Errorf("status %d: %s", resp.StatusCode, bodyText)
	}

	return bodyText, resp.StatusCode, nil
}

// collectLogTails retrieves log tails for the requested services.
func (c *HTTPRuntimeClient) collectLogTails(request Request) []LogTail {
	if request.LogTailLines <= 0 || c.manifest == nil {
		return nil
	}

	lines := request.LogTailLines
	if lines > 200 {
		lines = 200
	}

	serviceIDs := request.LogTailServices
	if len(serviceIDs) == 0 {
		for _, svc := range c.manifest.Services {
			if strings.TrimSpace(svc.LogDir) != "" {
				serviceIDs = append(serviceIDs, svc.ID)
			}
		}
	}

	if len(serviceIDs) == 0 {
		return nil
	}

	seen := map[string]bool{}
	var tails []LogTail
	for _, serviceID := range serviceIDs {
		id := strings.TrimSpace(serviceID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true

		path := fmt.Sprintf("/logs/tail?serviceId=%s&lines=%d", url.QueryEscape(id), lines)
		content, _, err := c.fetchText(path)
		tail := LogTail{
			ServiceID: id,
			Lines:     lines,
		}
		if err != nil {
			tail.Error = err.Error()
		} else {
			tail.Content = content
		}
		if tail.Content == "" && tail.Error == "" {
			continue
		}
		tails = append(tails, tail)
	}

	return tails
}

// fetchReadyWithPolling polls the readiness endpoint until ready or timeout.
func (c *HTTPRuntimeClient) fetchReadyWithPolling(request Request, timeout time.Duration) (Ready, int, error) {
	var ready Ready
	if _, err := c.fetchJSON("/readyz", http.MethodGet, nil, &ready, nil); err != nil {
		return ready, 0, err
	}
	if !request.StartServices || request.StatusOnly {
		return ready, 0, nil
	}
	waitBudget := c.maxReadinessTimeout()
	if waitBudget <= 0 {
		waitBudget = 30 * time.Second
	}
	if timeout > 0 && waitBudget > timeout {
		waitBudget = timeout
	}
	if waitBudget < 2*time.Second {
		waitBudget = 2 * time.Second
	}
	start := time.Now()
	deadline := start.Add(waitBudget)
	for time.Now().Before(deadline) {
		if ready.Ready {
			break
		}
		time.Sleep(1 * time.Second)
		if _, err := c.fetchJSON("/readyz", http.MethodGet, nil, &ready, nil); err != nil {
			return ready, int(time.Since(start).Seconds()), err
		}
	}
	return ready, int(time.Since(start).Seconds()), nil
}

// maxReadinessTimeout calculates the maximum readiness timeout from the manifest.
func (c *HTTPRuntimeClient) maxReadinessTimeout() time.Duration {
	if c.manifest == nil {
		return 0
	}
	maxTimeout := time.Duration(0)
	for _, svc := range c.manifest.Services {
		timeout := time.Duration(svc.Readiness.TimeoutMs) * time.Millisecond
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		if timeout > maxTimeout {
			maxTimeout = timeout
		}
	}
	return maxTimeout
}
