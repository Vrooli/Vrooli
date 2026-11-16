package uxcapture

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/cli-core/cliapp"
)

const maxHTMLBytes = 2 * 1024 * 1024

// CommandGroups exposes the canonical BAS capture flow from the operator's
// existing Git Control Tower command surface. It forwards only the selected
// source and bounded options; local file paths never leave this process.
func CommandGroups() []cliapp.CommandGroup {
	return []cliapp.CommandGroup{{
		Title: "UX capture",
		Commands: []cliapp.Command{{
			Name:        "ux-capture",
			Description: "Capture a running scenario, public page, or local HTML file into a versioned UX bundle",
			Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
				{Name: "scenario", Description: "Name of a running local scenario"},
				{Name: "path", Description: "Scenario path; defaults to /"},
				{Name: "step", Repeated: true, Description: "State step such as click:Branches; repeat to apply more than one"},
				{Name: "wait-selector", Description: "Scenario CSS selector that must become visible before capture; scenario captures only"},
				{Name: "url", Description: "Public HTTP(S) page URL"},
				{Name: "file", Description: "Local .html/.htm file, including an exported claude.ai artifact"},
				{Name: "width", Description: "Viewport width, 320–2560; default 1280"},
				{Name: "height", Description: "Viewport height, 240–1600; default 720"},
				{Name: "theme", Description: "Color scheme: light, dark, or no-preference; default light"},
				{Name: "strict-tokens", Bool: true, Description: "Run the CSS custom-property sentinel diagnostic in the same scenario capture"},
				{Name: "evidence-profile", Description: "Optional scenario evidence profile: componenttests"},
				{Name: "region-selector", Description: "CSS region for the componenttests crop and style sample; scenario captures only"},
				{Name: "artifact-kind", Description: "Download one declared artifact from BAS through its artifact owner"},
				{Name: "artifact-output", Description: "Destination file for --artifact-kind; both flags are required together"},
			}},
			RunCtx: run,
		}},
	}}
}

func run(rc cliapp.RunContext) error {
	urlValue := strings.TrimSpace(rc.Flag("url"))
	fileValue := strings.TrimSpace(rc.Flag("file"))
	scenario := strings.TrimSpace(rc.Flag("scenario"))
	provided := 0
	for _, value := range []string{urlValue, fileValue, scenario} {
		if value != "" {
			provided++
		}
	}
	if provided != 1 {
		return fmt.Errorf("provide exactly one of --url, --file, or --scenario")
	}
	width, err := dimension(rc.Flag("width"), 320, 2560)
	if err != nil {
		return fmt.Errorf("width: %w", err)
	}
	height, err := dimension(rc.Flag("height"), 240, 1600)
	if err != nil {
		return fmt.Errorf("height: %w", err)
	}
	theme := strings.TrimSpace(rc.Flag("theme"))
	if theme == "" {
		theme = "light"
	}
	if theme != "light" && theme != "dark" && theme != "no-preference" {
		return fmt.Errorf("theme must be light, dark, or no-preference")
	}
	strictTokens := rc.BoolFlag("strict-tokens")
	if strictTokens && scenario == "" {
		return fmt.Errorf("--strict-tokens is available for --scenario captures only")
	}
	evidenceProfile := strings.TrimSpace(rc.Flag("evidence-profile"))
	if evidenceProfile != "" && evidenceProfile != "componenttests" {
		return fmt.Errorf("--evidence-profile must be componenttests")
	}
	if evidenceProfile != "" && scenario == "" {
		return fmt.Errorf("--evidence-profile is available for --scenario captures only")
	}
	regionSelector, err := validateRegionSelector(scenario, evidenceProfile, rc.Flag("region-selector"))
	if err != nil {
		return err
	}
	waitSelector, err := validateWaitSelector(scenario, rc.Flag("wait-selector"))
	if err != nil {
		return err
	}
	artifactKind := strings.TrimSpace(rc.Flag("artifact-kind"))
	artifactOutput := strings.TrimSpace(rc.Flag("artifact-output"))
	if (artifactKind == "") != (artifactOutput == "") {
		return fmt.Errorf("--artifact-kind and --artifact-output must be used together")
	}

	endpoint := "/api/v1/ux-capture/public"
	request := map[string]any{"width": width, "height": height, "theme": theme}
	switch {
	case fileValue != "":
		extension := strings.ToLower(filepath.Ext(fileValue))
		if extension != ".html" && extension != ".htm" {
			return fmt.Errorf("--file must name an .html or .htm file")
		}
		file, openErr := os.Open(fileValue)
		if openErr != nil {
			return openErr
		}
		defer file.Close()
		content, readErr := io.ReadAll(io.LimitReader(file, maxHTMLBytes+1))
		if readErr != nil {
			return readErr
		}
		if len(content) == 0 || len(content) > maxHTMLBytes {
			return fmt.Errorf("HTML file must contain between 1 byte and 2 MiB")
		}
		endpoint = "/api/v1/ux-capture/file"
		request["filename"], request["html"] = filepath.Base(fileValue), string(content)
	case scenario != "":
		endpoint = "/api/v1/ux-capture/scenario"
		request = buildScenarioRequest(scenario, strings.TrimSpace(rc.Flag("path")), rc.FlagValues("step"), width, height, theme, waitSelector, strictTokens, evidenceProfile, regionSelector)
	default:
		request["url"] = urlValue
	}

	baseURL, err := discovery.ResolveScenarioURLDefault(context.Background(), "browser-automation-studio")
	if err != nil {
		return fmt.Errorf("browser-automation-studio is unavailable: %w", err)
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+endpoint, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("BAS UX capture request failed: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 8*1024*1024))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("BAS UX capture failed (HTTP %d): %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	if artifactKind != "" {
		if err := downloadDeclaredArtifact(ctx, baseURL, body, artifactKind, artifactOutput); err != nil {
			return err
		}
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, body, "", "  "); err != nil {
		return fmt.Errorf("BAS returned invalid JSON: %w", err)
	}
	_, err = fmt.Fprintln(rc.Stdout(), pretty.String())
	return err
}

type captureArtifactManifest struct {
	SchemaVersion string `json:"schema_version"`
	ID            string `json:"id"`
	Privacy       struct {
		RedactionStatus string `json:"redaction_status"`
	} `json:"privacy"`
	Artifacts []struct {
		Kind   string `json:"kind"`
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size_bytes"`
	} `json:"artifacts"`
}

func downloadDeclaredArtifact(ctx context.Context, baseURL string, response []byte, kind, destination string) error {
	var wrapper struct {
		Manifest captureArtifactManifest `json:"manifest"`
	}
	if err := json.Unmarshal(response, &wrapper); err != nil {
		return fmt.Errorf("decode capture artifact manifest: %w", err)
	}
	m := wrapper.Manifest
	if m.SchemaVersion != "ux-capture.v1" || !canonicalBundleID(m.ID) || m.Privacy.RedactionStatus != "complete" {
		return fmt.Errorf("BAS returned an invalid or unredacted artifact manifest")
	}
	var selected *struct {
		Kind   string `json:"kind"`
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size_bytes"`
	}
	for i := range m.Artifacts {
		if m.Artifacts[i].Kind == kind {
			if selected != nil {
				return fmt.Errorf("bundle contains multiple %s artifacts", kind)
			}
			selected = &m.Artifacts[i]
		}
	}
	if selected == nil || !safeArtifactPath(selected.Path) || selected.Size < 0 || selected.Size > 32*1024*1024 {
		return fmt.Errorf("bundle has no bounded declared %s artifact", kind)
	}
	parts := strings.Split(selected.Path, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	artifactURL := strings.TrimRight(baseURL, "/") + "/api/v1/ux-capture/bundles/" + url.PathEscape(m.ID) + "/artifacts/" + strings.Join(parts, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, artifactURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch BAS-owned %s artifact: %w", kind, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("BAS artifact owner returned HTTP %d", resp.StatusCode)
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024*1024+1))
	if err != nil {
		return err
	}
	if int64(len(content)) != selected.Size || len(content) > 32*1024*1024 {
		return fmt.Errorf("artifact size does not match manifest")
	}
	digest := sha256.Sum256(content)
	if hex.EncodeToString(digest[:]) != strings.ToLower(selected.SHA256) {
		return fmt.Errorf("artifact digest does not match manifest")
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(destination), ".bas-artifact-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if _, err := temp.Write(content); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, destination)
}

func canonicalBundleID(value string) bool {
	if len(value) != 36 || strings.ToLower(value) != value {
		return false
	}
	for i, r := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
		} else if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func safeArtifactPath(value string) bool {
	if value == "" || strings.ContainsAny(value, "\\\x00") || strings.HasPrefix(value, "/") {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func validateWaitSelector(scenario, value string) (string, error) {
	selector := strings.TrimSpace(value)
	if len(selector) > 512 {
		return "", fmt.Errorf("--wait-selector must be at most 512 characters")
	}
	if selector != "" && scenario == "" {
		return "", fmt.Errorf("--wait-selector is available for --scenario captures only")
	}
	return selector, nil
}

func validateRegionSelector(scenario, profile, value string) (string, error) {
	selector := strings.TrimSpace(value)
	if len(selector) > 256 || strings.HasPrefix(selector, "//") {
		return "", fmt.Errorf("--region-selector must be a CSS selector up to 256 characters")
	}
	if selector != "" && (scenario == "" || profile != "componenttests") {
		return "", fmt.Errorf("--region-selector requires --scenario and --evidence-profile componenttests")
	}
	return selector, nil
}

func buildScenarioRequest(scenario, route string, steps []string, width, height int, theme, waitSelector string, strictTokens bool, evidenceProfile, regionSelector string) map[string]any {
	request := map[string]any{
		"scenario": scenario,
		"route":    route,
		"steps":    steps,
		"width":    width,
		"height":   height,
		"theme":    theme,
	}
	if waitSelector != "" {
		request["wait_selector"] = waitSelector
	}
	if strictTokens {
		request["strict_tokens"] = true
	}
	if evidenceProfile != "" {
		request["evidence_profile"] = evidenceProfile
	}
	if regionSelector != "" {
		request["region_selector"] = regionSelector
	}
	return request
}

func dimension(value string, minimum, maximum int) (int, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minimum || parsed > maximum {
		return 0, fmt.Errorf("must be an integer between %d and %d", minimum, maximum)
	}
	return parsed, nil
}
