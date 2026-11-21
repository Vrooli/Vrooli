package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type remoteProfileClient struct {
	resolveBase         func() string
	resolveServiceToken func() string
	profileTag          string
	http                *http.Client
}

type remoteProfileList struct {
	Profiles []struct {
		ID  int64  `json:"id"`
		Tag string `json:"tag"`
	} `json:"profiles"`
}

type remoteProfileProxyRequest struct {
	Method string            `json:"method"`
	Path   string            `json:"path"`
	Query  map[string]string `json:"query,omitempty"`
	Body   json.RawMessage   `json:"body,omitempty"`
}

func (c *remoteProfileClient) Name() string { return "lpbs-remote-profile" }

func (c *remoteProfileClient) Fetch(ctx context.Context, readPath string) (json.RawMessage, error) {
	base := strings.TrimRight(strings.TrimSpace(c.resolveBase()), "/")
	token := strings.TrimSpace(c.resolveServiceToken())
	if base == "" || token == "" || strings.TrimSpace(c.profileTag) == "" {
		return nil, ErrNotAvailable
	}
	profileID, err := c.findProfile(ctx, base, token)
	if err != nil {
		return nil, err
	}
	proxyPath, query, body, err := remoteProfileRequest(readPath)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(remoteProfileProxyRequest{Method: http.MethodPost, Path: proxyPath, Query: query, Body: body})
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/admin/remote-profiles/%d/proxy", base, profileID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lpbs remote profile: %w: %v", ErrNotAvailable, err)
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("lpbs remote profile http %d: %s", resp.StatusCode, truncate(string(result), 200))
	}
	return json.RawMessage(result), nil
}

func (c *remoteProfileClient) findProfile(ctx context.Context, base, token string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/admin/remote-profiles", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("lpbs remote profile lookup: %w: %v", ErrNotAvailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("lpbs remote profile lookup http %d", resp.StatusCode)
	}
	var profiles remoteProfileList
	if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
		return 0, err
	}
	for _, profile := range profiles.Profiles {
		if profile.Tag == c.profileTag {
			return profile.ID, nil
		}
	}
	return 0, fmt.Errorf("lpbs remote profile %q not configured", c.profileTag)
}

func remoteProfileRequest(readPath string) (string, map[string]string, json.RawMessage, error) {
	u, err := url.Parse(readPath)
	if err != nil {
		return "", nil, nil, err
	}
	query := map[string]string{}
	for key, values := range u.Query() {
		if len(values) > 0 && values[0] != "" {
			query[key] = values[0]
		}
	}
	if u.Path == "/api/v1/admin/dashboard/revenue" {
		return "/landing_page_business_suite.v1.AdminRevenueService/GetRevenueSummary", query, json.RawMessage(`{}`), nil
	}
	if !strings.Contains(u.Path, ".v1.") {
		return "", nil, nil, fmt.Errorf("lpbs production relay does not permit %q", readPath)
	}
	body := map[string]any{}
	for key, value := range query {
		if integer, parseErr := strconv.Atoi(value); parseErr == nil {
			body[key] = integer
		} else {
			body[key] = value
		}
	}
	encoded, err := json.Marshal(body)
	return u.Path, nil, encoded, err
}
