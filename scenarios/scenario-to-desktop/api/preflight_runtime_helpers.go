package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

func readFileWithRetry(path string, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	return nil, lastErr
}

func readPortFileWithRetry(path string, timeout time.Duration) (int, error) {
	data, err := readFileWithRetry(path, timeout)
	if err != nil {
		return 0, err
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}
	return port, nil
}

func waitForRuntimeHealth(client *http.Client, baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest(http.MethodGet, baseURL+"/healthz", nil)
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("runtime control API not responding within %s", timeout)
}

func fetchJSON(client *http.Client, baseURL, token, path, method string, payload interface{}, out interface{}, allow map[int]bool) (int, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return 0, err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, baseURL+path, body)
	if err != nil {
		return 0, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
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

func fetchText(client *http.Client, baseURL, token, path string) (string, int, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		return "", 0, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
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

func collectLogTails(client *http.Client, baseURL, token string, manifest *bundlemanifest.Manifest, request BundlePreflightRequest) []BundlePreflightLogTail {
	if request.LogTailLines <= 0 || manifest == nil {
		return nil
	}

	lines := request.LogTailLines
	if lines > 200 {
		lines = 200
	}

	serviceIDs := request.LogTailServices
	if len(serviceIDs) == 0 {
		for _, svc := range manifest.Services {
			if strings.TrimSpace(svc.LogDir) != "" {
				serviceIDs = append(serviceIDs, svc.ID)
			}
		}
	}

	if len(serviceIDs) == 0 {
		return nil
	}

	seen := map[string]bool{}
	var tails []BundlePreflightLogTail
	for _, serviceID := range serviceIDs {
		id := strings.TrimSpace(serviceID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true

		path := fmt.Sprintf("/logs/tail?serviceId=%s&lines=%d", url.QueryEscape(id), lines)
		content, _, err := fetchText(client, baseURL, token, path)
		tail := BundlePreflightLogTail{
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

func fetchReadyWithPolling(client *http.Client, baseURL, token string, request BundlePreflightRequest, timeout time.Duration, manifest *bundlemanifest.Manifest) (BundlePreflightReady, int, error) {
	var ready BundlePreflightReady
	if _, err := fetchJSON(client, baseURL, token, "/readyz", http.MethodGet, nil, &ready, nil); err != nil {
		return ready, 0, err
	}
	if !request.StartServices || request.StatusOnly {
		return ready, 0, nil
	}
	waitBudget := maxReadinessTimeout(manifest)
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
		if _, err := fetchJSON(client, baseURL, token, "/readyz", http.MethodGet, nil, &ready, nil); err != nil {
			return ready, int(time.Since(start).Seconds()), err
		}
	}
	return ready, int(time.Since(start).Seconds()), nil
}

func maxReadinessTimeout(manifest *bundlemanifest.Manifest) time.Duration {
	if manifest == nil {
		return 0
	}
	maxTimeout := time.Duration(0)
	for _, svc := range manifest.Services {
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
