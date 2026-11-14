package support

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func ParseFlags(name string, args []string) (*flag.FlagSet, *bool, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return nil, nil, err
	}
	return fs, jsonOut, nil
}

type CampaignAutoOptions struct {
	Location string
	Tag      string
	Pattern  string
	Name     string
}

func (o CampaignAutoOptions) Enabled() bool {
	return strings.TrimSpace(o.Location) != "" || strings.TrimSpace(o.Tag) != "" || strings.TrimSpace(o.Pattern) != "" || strings.TrimSpace(o.Name) != ""
}

func (o CampaignAutoOptions) Validate() error {
	if strings.TrimSpace(o.Location) == "" || strings.TrimSpace(o.Tag) == "" {
		return fmt.Errorf("both --location and --tag are required for auto-creation")
	}
	return nil
}

type Resolver struct {
	Core       *cliapp.ScenarioApp
	CampaignID *string
}

func ErrMissingFlagValue(flag string) error {
	return fmt.Errorf("missing value for %s", flag)
}

func (r Resolver) ResolveCampaignID(opts CampaignAutoOptions, jsonOutput bool) (string, error) {
	if opts.Enabled() {
		if err := opts.Validate(); err != nil {
			return "", err
		}
		campaign, created, err := r.FindOrCreateCampaign(opts)
		if err != nil {
			return "", err
		}
		if created && !jsonOutput {
			fmt.Fprintf(os.Stderr, "Auto-created campaign: %s (ID: %s)\n", campaign.Name, campaign.ID)
		}
		return campaign.ID, nil
	}

	if r.CampaignID != nil && strings.TrimSpace(*r.CampaignID) != "" {
		return strings.TrimSpace(*r.CampaignID), nil
	}
	if env := strings.TrimSpace(os.Getenv("VISITED_TRACKER_CAMPAIGN_ID")); env != "" {
		r.SetCampaignID(env)
		return env, nil
	}

	body, err := r.Core.Get("/campaigns", nil)
	if err != nil {
		return "", err
	}

	var response CampaignListResponse
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
	r.SetCampaignID(id)
	return id, nil
}

func (r Resolver) FindOrCreateCampaign(opts CampaignAutoOptions) (Campaign, bool, error) {
	pattern := DefaultPattern(opts.Pattern)
	payload := map[string]interface{}{
		"location": strings.TrimSpace(opts.Location),
		"tag":      strings.TrimSpace(opts.Tag),
		"patterns": []string{pattern},
	}
	if strings.TrimSpace(opts.Name) != "" {
		payload["name"] = strings.TrimSpace(opts.Name)
	}

	body, err := r.Core.Request("POST", "/campaigns/find-or-create", nil, payload)
	if err != nil {
		return Campaign{}, false, err
	}

	var response FindOrCreateResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return Campaign{}, false, fmt.Errorf("parse find-or-create response: %w", err)
	}
	if response.Campaign.ID == "" {
		return Campaign{}, false, errors.New("find-or-create response missing campaign id")
	}
	r.SetCampaignID(response.Campaign.ID)
	return response.Campaign, response.Created, nil
}

func (r Resolver) SetCampaignID(id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	if r.CampaignID != nil {
		*r.CampaignID = id
	}
	_ = os.Setenv("VISITED_TRACKER_CAMPAIGN_ID", id)
}

func ParseJSONInput(value string) (map[string]interface{}, error) {
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

func BuildQuery(params map[string]string) url.Values {
	values := url.Values{}
	for key, value := range params {
		if strings.TrimSpace(value) == "" {
			continue
		}
		values.Set(key, value)
	}
	return values
}

func JoinPatterns(patterns []string) string {
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

func NormalizePathList(paths []string) []string {
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

func EnsureFilePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("file path is required")
	}
	return filepath.Clean(path), nil
}

func DefaultPattern(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "**/*"
	}
	return value
}

func ValueOrDash(value *string) string {
	if value == nil {
		return "-"
	}
	trim := strings.TrimSpace(*value)
	if trim == "" {
		return "-"
	}
	return trim
}

func BuildFileNotes(allFiles []string, globalNote string, perPaths []string, perNotes []string) map[string]string {
	result := make(map[string]string)
	globalNote = strings.TrimSpace(globalNote)
	if globalNote != "" {
		for _, file := range allFiles {
			result[file] = globalNote
		}
	}
	for i, path := range perPaths {
		if i >= len(perNotes) {
			break
		}
		note := strings.TrimSpace(perNotes[i])
		if strings.TrimSpace(path) == "" || note == "" {
			continue
		}
		result[strings.TrimSpace(path)] = note
	}
	return result
}

func JSONLines(body []byte) []string {
	var parsed interface{}
	if err := json.Unmarshal(body, &parsed); err == nil {
		pretty, _ := json.MarshalIndent(parsed, "", "  ")
		return strings.Split(strings.TrimSpace(string(pretty)), "\n")
	}
	return []string{strings.TrimSpace(string(body))}
}
