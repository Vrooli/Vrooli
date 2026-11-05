package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// ResolveScenarioFeatureURLDefault resolves a feature for this call. Desktop
// runtimes select compatible full peers before their own installed components.
// A failed desktop feature lookup never falls back to bare scenario discovery.
func ResolveScenarioFeatureURLDefault(ctx context.Context, scenario, feature string) (string, error) {
	return DefaultResolver().ResolveScenarioFeatureURLDefault(ctx, scenario, feature)
}

func (r *Resolver) ResolveScenarioFeatureURLDefault(ctx context.Context, scenario, feature string) (string, error) {
	scenario, feature = strings.TrimSpace(scenario), strings.TrimSpace(feature)
	if scenario == "" || feature == "" {
		return "", fmt.Errorf("scenario and feature are required")
	}
	base := strings.TrimSpace(os.Getenv("VROOLI_RUNTIME_URL"))
	if base == "" {
		return r.ResolveScenarioURLDefault(ctx, scenario)
	}
	u, err := url.Parse(base)
	if err != nil || !loopbackHTTPURL(u) {
		return "", fmt.Errorf("invalid desktop runtime URL")
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/features"
	query := u.Query()
	query.Set("scenario", scenario)
	query.Set("feature", feature)
	u.RawQuery = query.Encode()
	token, err := os.ReadFile(os.Getenv("VROOLI_AUTH_LOCAL_TOKEN_FILE"))
	if err != nil || strings.TrimSpace(string(token)) == "" {
		return "", fmt.Errorf("desktop runtime credential unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(token)))
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("desktop feature lookup failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("desktop feature %s/%s unavailable (status %d)", scenario, feature, resp.StatusCode)
	}
	var selected struct {
		URL         string `json:"url"`
		Source      string `json:"source"`
		ComponentID string `json:"component_id"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16*1024)).Decode(&selected); err != nil {
		return "", fmt.Errorf("invalid desktop feature response")
	}
	endpoint, err := url.Parse(selected.URL)
	if err != nil || !loopbackHTTPURL(endpoint) {
		return "", fmt.Errorf("invalid desktop feature endpoint")
	}
	return strings.TrimRight(selected.URL, "/"), nil
}

func loopbackHTTPURL(u *url.URL) bool {
	if u == nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Port() == "" {
		return false
	}
	if u.Hostname() == "localhost" {
		return true
	}
	ip := net.ParseIP(u.Hostname())
	return ip != nil && ip.IsLoopback()
}
