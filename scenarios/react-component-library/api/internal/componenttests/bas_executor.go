package componenttests

// This file is the production browser boundary for component tests. BAS owns
// the browser session and evidence bundle; RCL only supplies the exact story
// URL and interprets the story contract result embedded in the isolated route.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	repocontract "github.com/vrooli/repo-contract-go"
	"react-component-library/internal/components"
)

var (
	basStoryResult      = regexp.MustCompile(`(?s)<pre[^>]*id="rcl-story-result"[^>]*>(.*?)</pre>`)
	captureScenarioSlug = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)
	captureBundleID     = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// BASCaptureExecutor delegates story execution to BAS's canonical UX capture bundle. It is
// intentionally small so the runner remains independently testable while
// production receives BAS's screenshot, DOM, accessibility, console, network,
// and performance artifacts from one browser session.
type BASCaptureExecutor struct {
	RCLUIBaseURL   string
	BASBaseURL     string
	HTTPClient     *http.Client
	PageSourceRoot string
	PageUIBaseURL  string
	discoverBAS    bool
}

// ReportFreshnessKey versions the canonical UX bundle evidence consumed by
// component reports.
func (BASCaptureExecutor) ReportFreshnessKey() string {
	return "ux-capture-componenttests=1;policy=4"
}

const (
	uxScenarioCapturePath = "/ux-capture/scenario"
	uxArtifactPathPrefix  = "/ux-capture/bundles/"
)

// BASProviderInspection is the read-only observation of the BAS endpoint
// selected by this executor. It intentionally contains no response body or
// raw error string, which could include URL credentials or query parameters.
type BASProviderInspection struct {
	BASBaseURL     string
	CapturePath    string
	ResolvedAt     time.Time
	HealthStatus   string
	FailureClass   string
	HTTPStatusCode int
	SelectionExact bool
}

func NewBASCaptureExecutor() BASCaptureExecutor {
	basBase := strings.TrimRight(strings.TrimSpace(os.Getenv("RCL_BAS_API_URL")), "/")
	if basBase == "" {
		basBase = strings.TrimRight(strings.TrimSpace(os.Getenv("BAS_API_URL")), "/")
	}
	discoverBAS := basBase == ""
	if basBase == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		basBase, _ = discovery.NewResolver(discovery.ResolverConfig{}).ResolveScenarioURL(ctx, "browser-automation-studio", "API_PORT")
	}
	rclUIBase := strings.TrimRight(strings.TrimSpace(os.Getenv("RCL_UI_BASE_URL")), "/")
	if rclUIBase == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		rclUIBase, _ = discovery.NewResolver(discovery.ResolverConfig{}).ResolveScenarioURL(ctx, "react-component-library", "UI_PORT")
	}
	return BASCaptureExecutor{RCLUIBaseURL: rclUIBase, BASBaseURL: basBase, HTTPClient: &http.Client{Timeout: 2 * time.Minute}, discoverBAS: discoverBAS}
}

// Probe confirms that the browser boundary is reachable before a sweep
// creates partial evidence. BAS owns this health endpoint; a capture failure
// is not a reason to manufacture a blocked row for every remaining version.
func (e BASCaptureExecutor) Probe(ctx context.Context) error {
	_, err := e.InspectProvider(ctx)
	return err
}

// InspectProvider resolves and health-checks the BAS endpoint used by
// canonical UX-bundle capture operations without starting a browser capture.
func (e BASCaptureExecutor) InspectProvider(ctx context.Context) (BASProviderInspection, error) {
	inspection := BASProviderInspection{
		CapturePath:  uxScenarioCapturePath,
		HealthStatus: "unavailable",
	}
	basBase, err := e.basBaseURL(ctx)
	if err != nil {
		inspection.FailureClass = "resolution_failed"
		return inspection, ExecutorUnavailableError{Err: err}
	}
	if strings.TrimSpace(e.BASBaseURL) == "" && !e.discoverBAS {
		inspection.FailureClass = "selection_missing"
		return inspection, ExecutorUnavailableError{Err: fmt.Errorf("BAS UX capture base URL is required")}
	}
	inspection.BASBaseURL, inspection.SelectionExact = sanitizedBASBaseURL(basBase)
	inspection.ResolvedAt = time.Now().UTC()
	client := e.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	healthURL, err := appendURLPath(basBase, "/health")
	if err != nil {
		inspection.FailureClass = "invalid_health_url"
		return inspection, ExecutorUnavailableError{Err: err}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		inspection.FailureClass = "invalid_health_url"
		return inspection, ExecutorUnavailableError{Err: err}
	}
	resp, err := client.Do(req)
	if err != nil {
		inspection.FailureClass = "health_request_failed"
		return inspection, ExecutorUnavailableError{Err: fmt.Errorf("BAS health probe: %w", err)}
	}
	defer resp.Body.Close()
	inspection.HTTPStatusCode = resp.StatusCode
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		inspection.HealthStatus = "unhealthy"
		inspection.FailureClass = "health_http_status"
		return inspection, ExecutorUnavailableError{Err: fmt.Errorf("BAS health probe returned HTTP %d", resp.StatusCode)}
	}
	inspection.HealthStatus = "healthy"
	return inspection, nil
}

func appendURLPath(base, suffix string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		if err == nil {
			err = fmt.Errorf("URL requires scheme and host")
		}
		return "", fmt.Errorf("invalid BAS endpoint URL: %w", err)
	}
	joinedSuffix := "/" + strings.TrimLeft(suffix, "/")
	escapedBase := strings.TrimRight(parsed.EscapedPath(), "/")
	parsed.Path = strings.TrimRight(parsed.Path, "/") + joinedSuffix
	escapedPath := escapedBase + joinedSuffix
	if escapedPath == parsed.Path {
		parsed.RawPath = ""
	} else {
		parsed.RawPath = escapedPath
	}
	return parsed.String(), nil
}

// sanitizedBASBaseURL removes any credential-bearing or request-varying URL
// components before an endpoint is reported to a caller. A redacted URL is
// useful context but is explicitly not claimed to be an exact endpoint.
func sanitizedBASBaseURL(raw string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}
	exact := parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == ""
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	parsed.Fragment = ""
	parsed.RawFragment = ""
	return strings.TrimRight(parsed.String(), "/"), exact
}

// Resolve each operation through the shared bounded cache so a recovered or
// restarted dependency is not pinned to its constructor-time address.
func (e BASCaptureExecutor) basBaseURL(ctx context.Context) (string, error) {
	if !e.discoverBAS {
		return e.BASBaseURL, nil
	}
	base, err := discovery.NewResolver(discovery.ResolverConfig{}).ResolveScenarioURL(ctx, "browser-automation-studio", "API_PORT")
	if err != nil {
		return "", fmt.Errorf("BAS UX capture discovery: %w", err)
	}
	return base, nil
}

type uxScenarioCaptureRequest struct {
	Scenario        string `json:"scenario"`
	Route           string `json:"route"`
	WaitSelector    string `json:"wait_selector"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	EvidenceProfile string `json:"evidence_profile"`
	RegionSelector  string `json:"region_selector"`
}

type basDimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type uxCaptureArtifact struct {
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

type uxCaptureBundle struct {
	SchemaVersion string `json:"schema_version"`
	ID            string `json:"id"`
	Source        struct {
		Kind     string `json:"kind"`
		Scenario string `json:"scenario"`
		Route    string `json:"route"`
	} `json:"source"`
	Privacy struct {
		RedactionStatus string `json:"redaction_status"`
	} `json:"privacy"`
	Lenses map[string]struct {
		Status string `json:"status"`
	} `json:"lenses"`
	Artifacts []uxCaptureArtifact `json:"artifacts"`
}

type uxCaptureResponse struct {
	Manifest uxCaptureBundle `json:"manifest"`
}

type componentAccessibility struct {
	Schema   string           `json:"schema"`
	Snapshot string           `json:"snapshot"`
	Axe      *StoryAxeResults `json:"axe,omitempty"`
}

type componentRegionStyle struct {
	Schema        string `json:"schema"`
	ComputedStyle struct {
		Color       string `json:"color"`
		Background  string `json:"background_color"`
		BorderColor string `json:"border_color"`
		BorderStyle string `json:"border_style"`
		BorderWidth string `json:"border_width"`
		Visibility  string `json:"visibility"`
		Display     string `json:"display"`
		Opacity     string `json:"opacity"`
	} `json:"computed_style"`
}

// URLCapture is the structured result for a non-RCL page captured through the
// same BAS boundary. It is used by generated-scenario validation so that the
// provider has one browser implementation and no ad hoc Playwright runner.
type URLCapture struct {
	DOMHTML       string
	Accessibility string
	RegionStyle   string
	Artifacts     []Artifact
}

func (e BASCaptureExecutor) ExecuteStory(ctx context.Context, libraryID, version, storyID string) (StoryExecution, error) {
	return e.ExecuteStoryWithGeometry(ctx, libraryID, version, storyID, nil)
}

func (e BASCaptureExecutor) ExecuteStoryWithGeometry(ctx context.Context, libraryID, version, storyID string, geometry *components.StoryGeometry) (StoryExecution, error) {
	if strings.TrimSpace(e.RCLUIBaseURL) == "" {
		return StoryExecution{}, ExecutorUnavailableError{Err: fmt.Errorf("RCL UI base URL is required")}
	}
	if strings.TrimSpace(e.BASBaseURL) == "" && !e.discoverBAS {
		return StoryExecution{}, ExecutorUnavailableError{Err: fmt.Errorf("BAS UX capture base URL is required")}
	}
	storyURL, err := e.storyURL(libraryID, version, storyID)
	if err != nil {
		return StoryExecution{}, err
	}
	return e.executeScenarioCapture(ctx, storyURL, libraryID, version, storyID, storyViewport(geometry))
}

var pixelDimension = regexp.MustCompile(`^([1-9][0-9]{0,3})px$`)

func storyViewport(geometry *components.StoryGeometry) basDimensions {
	dimensions := basDimensions{Width: 1280, Height: 800}
	if geometry == nil {
		return dimensions
	}
	if match := pixelDimension.FindStringSubmatch(strings.TrimSpace(geometry.Width)); len(match) == 2 {
		if value, err := strconv.Atoi(match[1]); err == nil && value >= 240 && value <= 1920 {
			dimensions.Width = value
		}
	}
	if match := pixelDimension.FindStringSubmatch(strings.TrimSpace(geometry.Height)); len(match) == 2 {
		if value, err := strconv.Atoi(match[1]); err == nil && value >= 240 && value <= 2160 {
			dimensions.Height = value
		}
	}
	return dimensions
}

func (e BASCaptureExecutor) ExecuteStorySheet(ctx context.Context, libraryID, version string, storyIDs []string) (StoryExecution, error) {
	if strings.TrimSpace(e.RCLUIBaseURL) == "" {
		return StoryExecution{}, ExecutorUnavailableError{Err: fmt.Errorf("RCL UI base URL is required")}
	}
	if strings.TrimSpace(e.BASBaseURL) == "" && !e.discoverBAS {
		return StoryExecution{}, ExecutorUnavailableError{Err: fmt.Errorf("BAS UX capture base URL is required")}
	}
	if len(storyIDs) == 0 {
		return StoryExecution{}, fmt.Errorf("story sheet must contain at least one story")
	}
	base, err := e.storyURL(libraryID, version, "")
	if err != nil {
		return StoryExecution{}, err
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return StoryExecution{}, fmt.Errorf("parse story sheet URL: %w", err)
	}
	query := parsed.Query()
	query.Del("story")
	query.Set("stories", strings.Join(storyIDs, ","))
	parsed.RawQuery = query.Encode()
	return e.executeScenarioCapture(ctx, parsed.String(), libraryID, version, "review-sheet:"+strings.Join(storyIDs, ","), basDimensions{Width: 1280, Height: 800})
}

func (e BASCaptureExecutor) executeScenarioCapture(ctx context.Context, storyURL, libraryID, version, storyID string, dimensions basDimensions) (StoryExecution, error) {
	ready := `[data-preview-readiness-marker][data-preview-ready="true"][data-rcl-story-status="passed"]`
	regionSelector := `[data-preview-capture-boundary="component-sheet"]`
	if strings.HasPrefix(libraryID, components.PageStoryPrefix) {
		ready = `[data-preview-readiness-marker][data-preview-ready="true"]`
		regionSelector = `[data-preview-sheet]`
	}
	started := time.Now()
	capture, err := e.captureScenarioPage(ctx, storyURL, ready, regionSelector, dimensions)
	if err != nil {
		return StoryExecution{}, err
	}
	frozen, err := e.readBundleArtifact(ctx, capture, "frozen-page")
	if err != nil {
		return StoryExecution{}, err
	}
	result, err := decodeBASStoryResult(string(frozen))
	if err != nil {
		return StoryExecution{}, err
	}
	accessibilityBytes, err := e.readBundleArtifact(ctx, capture, "accessibility-snapshot")
	if err != nil {
		return StoryExecution{}, err
	}
	var accessibility componentAccessibility
	if err := json.Unmarshal(accessibilityBytes, &accessibility); err != nil || accessibility.Schema != "bas-accessibility-snapshot/v1" || strings.TrimSpace(accessibility.Snapshot) == "" {
		return StoryExecution{}, fmt.Errorf("BAS bundle accessibility snapshot is invalid")
	}
	accessibility.Axe = result.AxeResults
	accessibilityJSON, _ := json.Marshal(accessibility)
	result.AccessibilityJSON = string(accessibilityJSON)
	consoleBytes, err := e.readBundleArtifact(ctx, capture, "console-summary")
	if err != nil {
		return StoryExecution{}, err
	}
	var console struct {
		Schema       string `json:"schema"`
		ErrorCount   int    `json:"error_count"`
		WarningCount int    `json:"warning_count"`
	}
	if err := json.Unmarshal(consoleBytes, &console); err != nil || console.Schema != "bas-console-summary/v1" || console.ErrorCount < 0 || console.WarningCount < 0 {
		return StoryExecution{}, fmt.Errorf("BAS bundle console summary is invalid")
	}
	if console.ErrorCount > 0 {
		result.Console.ConsoleErrors = []string{fmt.Sprintf("%d console error(s)", console.ErrorCount)}
	}
	performanceBytes, err := e.readBundleArtifact(ctx, capture, "performance-metrics")
	if err != nil {
		return StoryExecution{}, err
	}
	var performance struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(performanceBytes, &performance); err != nil || performance.Schema != "bas-performance-metrics/v1" {
		return StoryExecution{}, fmt.Errorf("BAS bundle performance metrics are invalid")
	}
	styleBytes, err := e.readBundleArtifact(ctx, capture, "region-style-sample")
	if err != nil {
		return StoryExecution{}, err
	}
	if err := validateRegionStyleSample(styleBytes); err != nil {
		return StoryExecution{}, err
	}
	region := findUXArtifact(capture.Manifest, "region-pixels")
	if region == nil {
		return StoryExecution{}, fmt.Errorf("BAS component capture did not produce its story region image")
	}
	result.Artifacts, err = e.componentArtifacts(ctx, capture.Manifest, libraryID, version, storyID)
	if err != nil {
		return StoryExecution{}, err
	}
	result.Duration = time.Since(started)
	return result, nil
}

func (e BASCaptureExecutor) CaptureScenarioURL(ctx context.Context, scenario, route, readinessSelector, regionSelector string) (URLCapture, error) {
	if !captureScenarioSlug.MatchString(scenario) {
		return URLCapture{}, fmt.Errorf("scenario must be a valid slug")
	}
	parsed, err := url.ParseRequestURI(route)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/") {
		return URLCapture{}, fmt.Errorf("route must be an absolute local path")
	}
	dimensions := basDimensions{Width: 1280, Height: 800}
	if e.BASBaseURL == "" && !e.discoverBAS {
		return URLCapture{}, fmt.Errorf("BAS UX capture base URL is required")
	}
	capture, err := e.captureScenario(ctx, scenario, route, readinessSelector, regionSelector, dimensions)
	if err != nil {
		return URLCapture{}, err
	}
	a11y, err := e.readBundleArtifact(ctx, capture, "accessibility-snapshot")
	if err != nil {
		return URLCapture{}, err
	}
	style, err := e.readBundleArtifact(ctx, capture, "region-style-sample")
	if err != nil {
		return URLCapture{}, err
	}
	if err := validateRegionStyleSample(style); err != nil {
		return URLCapture{}, err
	}
	if _, err := e.readBundleArtifact(ctx, capture, "region-pixels"); err != nil {
		return URLCapture{}, err
	}
	return URLCapture{Accessibility: string(a11y), RegionStyle: string(style), Artifacts: []Artifact{}}, nil
}

func validateRegionStyleSample(raw []byte) error {
	var sample componentRegionStyle
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&sample); err != nil || sample.Schema != "bas-region-style-sample/v1" {
		return fmt.Errorf("BAS region style sample is invalid")
	}
	for _, value := range []string{sample.ComputedStyle.Color, sample.ComputedStyle.Background, sample.ComputedStyle.BorderColor, sample.ComputedStyle.BorderStyle, sample.ComputedStyle.BorderWidth, sample.ComputedStyle.Visibility, sample.ComputedStyle.Display, sample.ComputedStyle.Opacity} {
		if value == "" || len(value) > 128 {
			return fmt.Errorf("BAS region style sample contains an invalid property value")
		}
	}
	return nil
}

func (e BASCaptureExecutor) captureScenarioPage(ctx context.Context, targetURL, readinessSelector, regionSelector string, dimensions basDimensions) (uxCaptureResponse, error) {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return uxCaptureResponse{}, fmt.Errorf("parse RCL story URL: %w", err)
	}
	return e.captureScenario(ctx, "react-component-library", parsed.RequestURI(), readinessSelector, regionSelector, dimensions)
}

func (e BASCaptureExecutor) captureScenario(ctx context.Context, scenario, route, readinessSelector, regionSelector string, dimensions basDimensions) (uxCaptureResponse, error) {
	basBase, err := e.basBaseURL(ctx)
	if err != nil {
		return uxCaptureResponse{}, ExecutorUnavailableError{Err: err}
	}
	if strings.TrimSpace(route) == "" {
		return uxCaptureResponse{}, fmt.Errorf("BAS capture route is required")
	}
	if strings.TrimSpace(e.BASBaseURL) == "" && !e.discoverBAS {
		return uxCaptureResponse{}, fmt.Errorf("BAS UX capture base URL is required")
	}
	payload := uxScenarioCaptureRequest{
		Scenario: scenario, Route: route, WaitSelector: readinessSelector,
		Width: dimensions.Width, Height: dimensions.Height,
		EvidenceProfile: "componenttests", RegionSelector: regionSelector,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return uxCaptureResponse{}, fmt.Errorf("encode BAS UX capture request: %w", err)
	}
	captureURL, err := appendURLPath(basBase, uxScenarioCapturePath)
	if err != nil {
		return uxCaptureResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, captureURL, strings.NewReader(string(body)))
	if err != nil {
		return uxCaptureResponse{}, fmt.Errorf("create BAS UX capture request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client().Do(req)
	if err != nil {
		return uxCaptureResponse{}, ExecutorUnavailableError{Err: err}
	}
	defer resp.Body.Close()
	responseBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return uxCaptureResponse{}, fmt.Errorf("read BAS UX capture response: %w", readErr)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return uxCaptureResponse{}, fmt.Errorf("BAS UX capture returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	var capture uxCaptureResponse
	if err := json.Unmarshal(responseBody, &capture); err != nil {
		return uxCaptureResponse{}, fmt.Errorf("decode BAS UX capture response: %w", err)
	}
	if capture.Manifest.SchemaVersion != "ux-capture.v1" || capture.Manifest.ID == "" || capture.Manifest.Source.Kind != "scenario" || capture.Manifest.Source.Scenario != scenario || capture.Manifest.Privacy.RedactionStatus != "complete" {
		return uxCaptureResponse{}, fmt.Errorf("BAS returned a mismatched or incompletely redacted UX capture bundle")
	}
	return capture, nil
}

func (e BASCaptureExecutor) client() *http.Client {
	if e.HTTPClient != nil {
		return e.HTTPClient
	}
	return http.DefaultClient
}

func (e BASCaptureExecutor) storyURL(libraryID, version, storyID string) (string, error) {
	if strings.HasPrefix(libraryID, components.PageStoryPrefix) {
		if version != components.PageStoryVersion {
			return "", fmt.Errorf("page stories use version workspace")
		}
		root := e.PageSourceRoot
		if root == "" {
			repo, err := repocontract.ResolveRepoRoot()
			if err != nil {
				return "", err
			}
			root = filepath.Join(repo, "scenarios", "react-component-library")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		subject, err := components.LoadPageStory(ctx, root, libraryID)
		if err != nil {
			return "", err
		}
		uiBase := e.PageUIBaseURL
		if uiBase == "" {
			uiBase = e.RCLUIBaseURL
		}
		if uiBase == "" {
			uiBase, err = discovery.NewResolver(discovery.ResolverConfig{}).ResolveScenarioURL(ctx, "react-component-library", "UI_PORT")
			if err != nil {
				return "", err
			}
		}
		route := subject.Contract.Route
		for _, story := range subject.Contract.Stories {
			if story.ID == storyID && story.Route != "" {
				route = story.Route
			}
		}
		pageURL, err := url.Parse(strings.TrimRight(uiBase, "/") + route)
		if err != nil {
			return "", err
		}
		query := pageURL.Query()
		query.Set("__rcl_page_story", strings.TrimPrefix(libraryID, components.PageStoryPrefix))
		query.Set("__rcl_story", storyID)
		pageURL.RawQuery = query.Encode()
		return pageURL.String(), nil
	}
	uiBase := strings.TrimRight(e.RCLUIBaseURL, "/")
	if uiBase == "" {
		var err error
		uiBase, err = discovery.NewResolver(discovery.ResolverConfig{}).ResolveScenarioURL(context.Background(), "react-component-library", "UI_PORT")
		if err != nil {
			return "", fmt.Errorf("resolve RCL UI URL: %w", err)
		}
	}
	base, err := url.Parse(uiBase + "/preview/" + url.PathEscape(libraryID) + "/harness.html")
	if err != nil {
		return "", fmt.Errorf("build story URL: %w", err)
	}
	query := base.Query()
	query.Set("version", version)
	query.Set("story", storyID)
	query.Set("runner", "1")
	query.Set("motion", "reduce")
	base.RawQuery = query.Encode()
	return base.String(), nil
}

func decodeBASStoryResult(dom string) (StoryExecution, error) {
	match := basStoryResult.FindStringSubmatch(dom)
	if len(match) != 2 {
		return StoryExecution{}, fmt.Errorf("BAS capture completed without rcl-story-result")
	}
	var result struct {
		Passed      bool                `json:"passed"`
		Failures    []json.RawMessage   `json:"failures"`
		Performance PerformanceEvidence `json:"performance"`
		Axe         *StoryAxeResults    `json:"axe"`
	}
	if err := json.Unmarshal([]byte(html.UnescapeString(match[1])), &result); err != nil {
		return StoryExecution{}, fmt.Errorf("decode rcl-story-result: %w", err)
	}
	if err := validateStoryAxeResults(result.Axe); err != nil {
		return StoryExecution{}, err
	}
	failures := make([]string, 0, len(result.Failures))
	for _, raw := range result.Failures {
		var message string
		if json.Unmarshal(raw, &message) == nil {
			failures = append(failures, message)
			continue
		}
		var detail struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(raw, &detail) == nil && detail.Message != "" {
			failures = append(failures, detail.Message)
			continue
		}
		failures = append(failures, string(raw))
	}
	return StoryExecution{Passed: result.Passed, Failures: failures, Performance: result.Performance, AxeResults: result.Axe}, nil
}

func validateStoryAxeResults(result *StoryAxeResults) error {
	if result == nil {
		return nil
	}
	if result.Schema != "rcl-axe-results/v1" {
		return fmt.Errorf("decode rcl-story-result axe: unsupported schema %q", result.Schema)
	}
	switch result.Status {
	case "unavailable":
		if strings.TrimSpace(result.Reason) == "" {
			return fmt.Errorf("decode rcl-story-result axe: unavailable result requires a reason")
		}
	case "complete":
		if result.Engine != "axe-core" || strings.TrimSpace(result.Version) == "" || result.Violations == nil {
			return fmt.Errorf("decode rcl-story-result axe: complete result requires axe-core version and violations")
		}
	default:
		return fmt.Errorf("decode rcl-story-result axe: unsupported status %q", result.Status)
	}
	return nil
}

func findUXArtifact(bundle uxCaptureBundle, kind string) *uxCaptureArtifact {
	var found *uxCaptureArtifact
	for index := range bundle.Artifacts {
		if bundle.Artifacts[index].Kind == kind {
			if found != nil {
				return nil
			}
			found = &bundle.Artifacts[index]
		}
	}
	return found
}

func safeBundleArtifactPath(value string) bool {
	return value != "" && !strings.Contains(value, `\\`) && !strings.HasPrefix(value, "/") && path.Clean(value) == value && value != "." && value != ".." && !strings.HasPrefix(value, "../")
}

func (e BASCaptureExecutor) readBundleArtifact(ctx context.Context, capture uxCaptureResponse, kind string) ([]byte, error) {
	artifact := findUXArtifact(capture.Manifest, kind)
	if artifact == nil {
		return nil, fmt.Errorf("BAS UX bundle is missing %s evidence", kind)
	}
	base, err := e.basBaseURL(ctx)
	if err != nil {
		return nil, ExecutorUnavailableError{Err: err}
	}
	if !captureBundleID.MatchString(capture.Manifest.ID) || !safeBundleArtifactPath(artifact.Path) {
		return nil, fmt.Errorf("BAS UX bundle declared an unsafe artifact")
	}
	segments := strings.Split(artifact.Path, "/")
	escaped := make([]string, len(segments))
	for i, segment := range segments {
		if segment == "" {
			return nil, fmt.Errorf("BAS UX bundle declared an unsafe artifact")
		}
		escaped[i] = url.PathEscape(segment)
	}
	endpoint := uxArtifactPathPrefix + url.PathEscape(capture.Manifest.ID) + "/artifacts/" + strings.Join(escaped, "/")
	artifactURL, err := appendURLPath(base, endpoint)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, artifactURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create BAS UX artifact request: %w", err)
	}
	resp, err := e.client().Do(req)
	if err != nil {
		return nil, ExecutorUnavailableError{Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BAS UX artifact returned HTTP %d", resp.StatusCode)
	}
	const maxArtifactBytes = 5 * 1024 * 1024
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxArtifactBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read BAS UX artifact: %w", err)
	}
	if len(data) > maxArtifactBytes || int64(len(data)) != artifact.SizeBytes {
		return nil, fmt.Errorf("BAS UX artifact size does not match its manifest")
	}
	digest := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(digest[:]), artifact.SHA256) {
		return nil, fmt.Errorf("BAS UX artifact digest does not match its manifest")
	}
	return data, nil
}

func (e BASCaptureExecutor) componentArtifacts(ctx context.Context, bundle uxCaptureBundle, libraryID, version, storyID string) ([]Artifact, error) {
	wanted := []struct{ kind, label string }{
		{"region-pixels", "story-region"},
		{"accessibility-snapshot", "accessibility"},
		{"region-style-sample", "region-style"},
		{"console-summary", "console"},
		{"performance-metrics", "performance"},
		{"frozen-page", "frozen-page"},
	}
	result := make([]Artifact, 0, len(wanted))
	base, err := e.basBaseURL(ctx)
	if err != nil {
		return nil, ExecutorUnavailableError{Err: err}
	}
	for _, item := range wanted {
		artifact := findUXArtifact(bundle, item.kind)
		if artifact == nil {
			return nil, fmt.Errorf("BAS UX bundle is missing %s evidence", item.kind)
		}
		if !captureBundleID.MatchString(bundle.ID) || !safeBundleArtifactPath(artifact.Path) {
			return nil, fmt.Errorf("BAS UX bundle declared an unsafe artifact")
		}
		segments := strings.Split(artifact.Path, "/")
		for index := range segments {
			segments[index] = url.PathEscape(segments[index])
		}
		refPath := uxArtifactPathPrefix + url.PathEscape(bundle.ID) + "/artifacts/" + strings.Join(segments, "/")
		abs, err := appendURLPath(base, refPath)
		if err != nil {
			return nil, err
		}
		ref := browserVisibleBASArtifactPath(abs)
		result = append(result, Artifact{Kind: "bas-" + item.label, Label: fmt.Sprintf("%s:%s", storyID, item.label), StoryID: storyID, AssetLibraryID: libraryID, Version: version, Reference: ref})
	}
	return result, nil
}

// browserVisibleBASArtifactPath deliberately returns an RCL-origin route.
// BASBaseURL is server-to-server only; persisting it would make a public RCL
// browser connect to loopback and trigger the browser's local-network prompt.
func browserVisibleBASArtifactPath(viewURL string) string {
	const embeddedBASPrefix = "/embedded/browser-automation-studio"
	if strings.HasPrefix(viewURL, "/") {
		return embeddedBASPrefix + viewURL
	}
	if parsed, err := url.Parse(viewURL); err == nil && parsed.Path != "" {
		return embeddedBASPrefix + parsed.EscapedPath() + func() string {
			if parsed.RawQuery == "" {
				return ""
			}
			return "?" + parsed.RawQuery
		}()
	}
	return viewURL
}
