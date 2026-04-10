package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

type campaignAutoOptions struct {
	location string
	tag      string
	pattern  string
	name     string
}

func (o campaignAutoOptions) enabled() bool {
	return strings.TrimSpace(o.location) != "" || strings.TrimSpace(o.tag) != "" || strings.TrimSpace(o.pattern) != "" || strings.TrimSpace(o.name) != ""
}

func (o campaignAutoOptions) validate() error {
	if strings.TrimSpace(o.location) == "" || strings.TrimSpace(o.tag) == "" {
		return fmt.Errorf("both --location and --tag are required for auto-creation")
	}
	return nil
}

func (a *App) apiPath(v1Path string) string {
	v1Path = strings.TrimSpace(v1Path)
	if v1Path == "" {
		return ""
	}
	if !strings.HasPrefix(v1Path, "/") {
		v1Path = "/" + v1Path
	}
	base := strings.TrimRight(strings.TrimSpace(a.core.HTTPClient.BaseURL()), "/")
	if strings.HasSuffix(base, "/api/v1") {
		return v1Path
	}
	return "/api/v1" + v1Path
}

func errMissingFlagValue(flag string) error {
	return fmt.Errorf("missing value for %s", flag)
}

func (a *App) resolveCampaignID(opts campaignAutoOptions, jsonOutput bool) (string, error) {
	if opts.enabled() {
		if err := opts.validate(); err != nil {
			return "", err
		}
		campaign, created, err := a.findOrCreateCampaign(opts)
		if err != nil {
			return "", err
		}
		if created && !jsonOutput {
			fmt.Fprintf(os.Stderr, "Auto-created campaign: %s (ID: %s)\n", campaign.Name, campaign.ID)
		}
		return campaign.ID, nil
	}

	if strings.TrimSpace(a.campaignID) != "" {
		return strings.TrimSpace(a.campaignID), nil
	}
	if env := strings.TrimSpace(os.Getenv("VISITED_TRACKER_CAMPAIGN_ID")); env != "" {
		a.campaignID = env
		return env, nil
	}

	body, err := a.core.APIClient.Get(a.apiPath("/campaigns"), nil)
	if err != nil {
		return "", err
	}

	var response campaignListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("parse campaigns response: %w", err)
	}
	if len(response.Campaigns) == 0 {
		return "", errors.New("no campaigns available; create one with 'visited-tracker campaigns create' or pass --campaign-id")
	}
	id := strings.TrimSpace(response.Campaigns[0].ID)
	if id == "" {
		return "", errors.New("first campaign has no ID")
	}
	a.setCampaignID(id)
	return id, nil
}

func (a *App) findOrCreateCampaign(opts campaignAutoOptions) (campaign, bool, error) {
	pattern := strings.TrimSpace(opts.pattern)
	if pattern == "" {
		pattern = "**/*"
	}

	payload := map[string]interface{}{
		"location": opts.location,
		"tag":      opts.tag,
		"patterns": []string{pattern},
	}
	if strings.TrimSpace(opts.name) != "" {
		payload["name"] = opts.name
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/campaigns/find-or-create"), nil, payload)
	if err != nil {
		return campaign{}, false, err
	}

	var response findOrCreateResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return campaign{}, false, fmt.Errorf("parse find-or-create response: %w", err)
	}
	if response.Campaign.ID == "" {
		return campaign{}, false, errors.New("find-or-create response missing campaign id")
	}
	a.setCampaignID(response.Campaign.ID)
	return response.Campaign, response.Created, nil
}

func parseJSONInput(value string) (map[string]interface{}, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	if strings.HasPrefix(value, "@") {
		path := strings.TrimSpace(strings.TrimPrefix(value, "@"))
		if path == "" {
			return nil, errors.New("metadata file path is empty")
		}
		contents, err := cliutil.ReadFileString(path)
		if err != nil {
			return nil, err
		}
		value = contents
	}

	var out map[string]interface{}
	if err := json.Unmarshal([]byte(value), &out); err != nil {
		return nil, fmt.Errorf("invalid JSON metadata: %w", err)
	}
	return out, nil
}

func buildQuery(params map[string]string) url.Values {
	values := url.Values{}
	for key, value := range params {
		if strings.TrimSpace(value) == "" {
			continue
		}
		values.Set(key, value)
	}
	return values
}

func joinPatterns(patterns []string) string {
	if len(patterns) == 0 {
		return ""
	}
	cleaned := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		cleaned = append(cleaned, pattern)
	}
	return strings.Join(cleaned, ",")
}

func normalizePathList(paths []string) []string {
	result := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		result = append(result, path)
	}
	return result
}

func ensureFilePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("file path is required")
	}
	return filepath.Clean(path), nil
}
