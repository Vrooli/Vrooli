package componenttests

// This file is the production browser boundary for component tests. BAS owns
// the browser session and evidence bundle; RCL only supplies the exact story
// URL and interprets the story contract result embedded in the isolated route.

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	repocontract "github.com/vrooli/repo-contract-go"
	"react-component-library/internal/components"
)

var basStoryResult = regexp.MustCompile(`(?s)<pre[^>]*id="rcl-story-result"[^>]*>(.*?)</pre>`)

// BASCaptureExecutor delegates story execution to BAS CaptureService. It is
// intentionally small so the runner remains independently testable while
// production receives BAS's screenshot, DOM, accessibility, console, network,
// and performance artifacts from one browser session.
type BASCaptureExecutor struct {
	RCLBaseURL     string
	BASBaseURL     string
	HTTPClient     *http.Client
	PageSourceRoot string
	PageUIBaseURL  string
	discoverBAS    bool
}

// ReportFreshnessKey versions the BAS evidence policy that affects durable
// component reports. Policy 2 also requires preview to hydrate its exact
// pinned dependency closure before capture, so pre-hydration failures cannot
// be reused as evidence for the repaired preview path.
func (BASCaptureExecutor) ReportFreshnessKey() string {
	return "bas-story-sheet-dsf=" + strconv.FormatFloat(basStorySheetDeviceScaleFactor, 'f', -1, 64) + ";policy=2"
}

const basCaptureServicePath = "/browser_automation_studio.v1.capture.CaptureService/Capture"

// Story-sheet screenshots contain several rendered stories in one image, so
// reduce only their raster scale to stay below BAS's fixed PNG payload cap.
// Keeping the CSS viewport unchanged preserves the grouped sheet's layout;
// individual story captures retain the standard review scale.
const basStorySheetDeviceScaleFactor = 1.0

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
	rclBase := strings.TrimRight(strings.TrimSpace(os.Getenv("RCL_API_BASE_URL")), "/")
	if rclBase == "" {
		if port := strings.TrimSpace(os.Getenv("API_PORT")); port != "" {
			rclBase = "http://127.0.0.1:" + port
		}
	}
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
	return BASCaptureExecutor{RCLBaseURL: rclBase, BASBaseURL: basBase, HTTPClient: &http.Client{Timeout: 2 * time.Minute}, discoverBAS: discoverBAS}
}

// Probe confirms that the browser boundary is reachable before a sweep
// creates partial evidence. BAS owns this health endpoint; a capture failure
// is not a reason to manufacture a blocked row for every remaining version.
func (e BASCaptureExecutor) Probe(ctx context.Context) error {
	_, err := e.InspectProvider(ctx)
	return err
}

// InspectProvider resolves and health-checks the same BAS endpoint used by
// capture operations, but never invokes CaptureService.
func (e BASCaptureExecutor) InspectProvider(ctx context.Context) (BASProviderInspection, error) {
	inspection := BASProviderInspection{
		CapturePath:  basCaptureServicePath,
		HealthStatus: "unavailable",
	}
	basBase, err := e.basBaseURL(ctx)
	if err != nil {
		inspection.FailureClass = "resolution_failed"
		return inspection, ExecutorUnavailableError{Err: err}
	}
	if strings.TrimSpace(e.BASBaseURL) == "" && !e.discoverBAS {
		inspection.FailureClass = "selection_missing"
		return inspection, ExecutorUnavailableError{Err: fmt.Errorf("BAS CaptureService base URL is required")}
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
		return "", fmt.Errorf("BAS CaptureService discovery: %w", err)
	}
	return base, nil
}

type basCaptureRequest struct {
	URL                 string        `json:"url"`
	Captures            []string      `json:"captures"`
	Dimensions          basDimensions `json:"dimensions"`
	WaitFor             basWaitFor    `json:"waitFor"`
	InlineDOM           bool          `json:"inlineDom"`
	InlineAccessibility bool          `json:"inlineAccessibility"`
	InlineComputedStyle bool          `json:"inlineComputedStyle"`
	ScreenshotSelector  string        `json:"screenshotSelector,omitempty"`
	Label               string        `json:"label"`
}

type basDimensions struct {
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	DeviceScaleFactor float64 `json:"deviceScaleFactor,omitempty"`
}

type basWaitFor struct {
	Selector string `json:"selector"`
}

type basCaptureResponse struct {
	ExecutionID   string          `json:"executionId"`
	DurationMS    json.RawMessage `json:"durationMs"`
	DOMHTML       string          `json:"domHtml"`
	Accessibility string          `json:"accessibilityJson"`
	Artifacts     []basArtifact   `json:"artifacts"`
	Readiness     basReadiness    `json:"readiness"`
}

type basReadiness struct {
	SelectedStrategy string `json:"selectedStrategy"`
	Outcome          string `json:"outcome"`
}

type basArtifact struct {
	Type      string            `json:"type"`
	Path      string            `json:"path"`
	Reference string            `json:"reference"`
	SizeBytes json.RawMessage   `json:"sizeBytes"`
	Metadata  map[string]string `json:"metadata"`
	Primary   bool              `json:"primary"`
}

// URLCapture is the structured result for a non-RCL page captured through the
// same BAS boundary. It is used by generated-scenario validation so that the
// provider has one browser implementation and no ad hoc Playwright runner.
type URLCapture struct {
	DOMHTML       string
	Accessibility string
	Artifacts     []basArtifact
}

func (e BASCaptureExecutor) ExecuteStory(ctx context.Context, libraryID, version, storyID string) (StoryExecution, error) {
	if strings.TrimSpace(e.RCLBaseURL) == "" {
		return StoryExecution{}, ExecutorUnavailableError{Err: fmt.Errorf("RCL preview API base URL is required")}
	}
	if strings.TrimSpace(e.BASBaseURL) == "" && !e.discoverBAS {
		return StoryExecution{}, ExecutorUnavailableError{Err: fmt.Errorf("BAS CaptureService base URL is required")}
	}
	storyURL, err := e.storyURL(libraryID, version, storyID)
	if err != nil {
		return StoryExecution{}, err
	}
	return e.executeCapture(ctx, storyURL, fmt.Sprintf("rcl:%s@%s:%s", libraryID, version, storyID), libraryID, version, storyID)
}

func (e BASCaptureExecutor) ExecuteStorySheet(ctx context.Context, libraryID, version string, storyIDs []string) (StoryExecution, error) {
	if strings.TrimSpace(e.RCLBaseURL) == "" {
		return StoryExecution{}, ExecutorUnavailableError{Err: fmt.Errorf("RCL preview API base URL is required")}
	}
	if strings.TrimSpace(e.BASBaseURL) == "" && !e.discoverBAS {
		return StoryExecution{}, ExecutorUnavailableError{Err: fmt.Errorf("BAS CaptureService base URL is required")}
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
	return e.executeCaptureAtDimensions(ctx, parsed.String(), fmt.Sprintf("rcl:%s@%s:sheet:%s", libraryID, version, strings.Join(storyIDs, ",")), libraryID, version, "review-sheet:"+strings.Join(storyIDs, ","), basDimensions{Width: 1280, Height: 800, DeviceScaleFactor: basStorySheetDeviceScaleFactor})
}

func (e BASCaptureExecutor) executeCapture(ctx context.Context, storyURL, label, libraryID, version, storyID string) (StoryExecution, error) {
	return e.executeCaptureAtDimensions(ctx, storyURL, label, libraryID, version, storyID, basDimensions{Width: 1280, Height: 800})
}

func (e BASCaptureExecutor) executeCaptureAtDimensions(ctx context.Context, storyURL, label, libraryID, version, storyID string, dimensions basDimensions) (StoryExecution, error) {
	ready := `[data-preview-readiness-marker][data-preview-ready="true"][data-rcl-story-status="passed"]`
	screenshotSelector := `[data-preview-capture-boundary="component-sheet"]`
	if strings.HasPrefix(libraryID, components.PageStoryPrefix) {
		ready = `[data-preview-readiness-marker][data-preview-ready="true"]`
		// Page stories mark the app-owned root with data-preview-sheet; they do
		// not participate in the component harness capture-boundary contract.
		screenshotSelector = `[data-preview-sheet]`
	}
	capture, err := e.captureURLAtDimensions(ctx, storyURL, label, ready, screenshotSelector, dimensions)
	if err != nil {
		return StoryExecution{}, err
	}
	result, err := decodeBASStoryResult(capture.DOMHTML)
	if err != nil {
		return StoryExecution{}, err
	}
	result.Duration = time.Duration(parseBASInt(capture.DurationMS)) * time.Millisecond
	result.AccessibilityJSON = capture.Accessibility
	result.Artifacts = basArtifacts(libraryID, version, storyID, e.BASBaseURL, capture.Artifacts)
	for _, artifact := range capture.Artifacts {
		if strings.Contains(strings.ToLower(artifact.Type), "console") && artifact.Path != "" {
			result.Console = readConsoleArtifact(artifact.Path)
		}
	}
	return result, nil
}

// CaptureURL captures an arbitrary page through BAS. Callers should provide a
// semantic readiness selector and a bounded screenshot selector; BAS owns the
// browser lifecycle and returns the durable structured evidence bundle.
func (e BASCaptureExecutor) CaptureURL(ctx context.Context, pageURL, label, readinessSelector, screenshotSelector string) (URLCapture, error) {
	capture, err := e.captureURL(ctx, pageURL, label, readinessSelector, screenshotSelector)
	if err != nil {
		return URLCapture{}, err
	}
	return URLCapture{DOMHTML: capture.DOMHTML, Accessibility: capture.Accessibility, Artifacts: capture.Artifacts}, nil
}

func (e BASCaptureExecutor) captureURL(ctx context.Context, pageURL, label, readinessSelector, screenshotSelector string) (basCaptureResponse, error) {
	return e.captureURLAtDimensions(ctx, pageURL, label, readinessSelector, screenshotSelector, basDimensions{Width: 1280, Height: 800})
}

func (e BASCaptureExecutor) captureURLAtDimensions(ctx context.Context, pageURL, label, readinessSelector, screenshotSelector string, dimensions basDimensions) (basCaptureResponse, error) {
	basBase, err := e.basBaseURL(ctx)
	if err != nil {
		return basCaptureResponse{}, ExecutorUnavailableError{Err: err}
	}
	if strings.TrimSpace(pageURL) == "" {
		return basCaptureResponse{}, fmt.Errorf("BAS capture URL is required")
	}
	if strings.TrimSpace(e.BASBaseURL) == "" && !e.discoverBAS {
		return basCaptureResponse{}, fmt.Errorf("BAS CaptureService base URL is required")
	}
	payload := basCaptureRequest{
		URL: pageURL,
		Captures: []string{
			"CAPTURE_TYPE_SCREENSHOT",
			"CAPTURE_TYPE_DOM",
			"CAPTURE_TYPE_CONSOLE_LOGS",
			"CAPTURE_TYPE_NETWORK",
			"CAPTURE_TYPE_PERFORMANCE",
			"CAPTURE_TYPE_ACCESSIBILITY",
		},
		Dimensions:          dimensions,
		WaitFor:             basWaitFor{Selector: readinessSelector},
		InlineDOM:           true,
		InlineAccessibility: true,
		InlineComputedStyle: true,
		ScreenshotSelector:  screenshotSelector,
		Label:               label,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return basCaptureResponse{}, fmt.Errorf("encode BAS capture request: %w", err)
	}
	captureURL, err := appendURLPath(basBase, basCaptureServicePath)
	if err != nil {
		return basCaptureResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, captureURL, strings.NewReader(string(body)))
	if err != nil {
		return basCaptureResponse{}, fmt.Errorf("create BAS capture request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client().Do(req)
	if err != nil {
		return basCaptureResponse{}, ExecutorUnavailableError{Err: err}
	}
	defer resp.Body.Close()
	responseBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return basCaptureResponse{}, fmt.Errorf("read BAS capture response: %w", readErr)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return basCaptureResponse{}, fmt.Errorf("BAS CaptureService returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	var capture basCaptureResponse
	if err := json.Unmarshal(responseBody, &capture); err != nil {
		return basCaptureResponse{}, fmt.Errorf("decode BAS capture response: %w", err)
	}
	return capture, nil
}

func parseBASInt(raw json.RawMessage) int64 {
	if len(raw) == 0 || string(raw) == "null" {
		return 0
	}
	var number int64
	if json.Unmarshal(raw, &number) == nil {
		return number
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		var parsed int64
		_, _ = fmt.Sscanf(text, "%d", &parsed)
		return parsed
	}
	return 0
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
	base, err := url.Parse(e.RCLBaseURL + "/preview/" + url.PathEscape(libraryID) + "/harness.html")
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
	}
	if err := json.Unmarshal([]byte(html.UnescapeString(match[1])), &result); err != nil {
		return StoryExecution{}, fmt.Errorf("decode rcl-story-result: %w", err)
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
	return StoryExecution{Passed: result.Passed, Failures: failures, Performance: result.Performance}, nil
}

func basArtifacts(libraryID, version, storyID, basBaseURL string, artifacts []basArtifact) []Artifact {
	result := make([]Artifact, 0, len(artifacts))
	seen := make(map[string]struct{}, len(artifacts))
	for _, artifact := range artifacts {
		// BAS may return intermediate screenshots from navigation and readiness
		// steps. The final authoritative frame is marked primary. Selecting the
		// first frame made dynamically mounted stories (especially overlays)
		// appear blank even though the browser captured the correct final frame.
		if strings.EqualFold(artifact.Type, "CAPTURE_TYPE_SCREENSHOT") && !artifact.Primary && hasPrimaryScreenshot(artifacts) {
			continue
		}
		kind := strings.ToLower(strings.TrimPrefix(artifact.Type, "CAPTURE_TYPE_"))
		reference := artifact.Reference
		if reference == "" {
			reference = artifact.Path
		}
		if reference == "" {
			continue
		}
		artifactKind := "bas-" + kind
		if strings.HasPrefix(storyID, "review-sheet:") && kind == "screenshot" {
			artifactKind = "bas-story-sheet"
		}
		if viewURL := strings.TrimSpace(artifact.Metadata["view_url"]); viewURL != "" {
			reference = browserVisibleBASArtifactPath(viewURL)
		}
		if _, ok := seen[reference]; ok {
			continue
		}
		seen[reference] = struct{}{}
		result = append(result, Artifact{Kind: artifactKind, Label: fmt.Sprintf("%s:%s", storyID, kind), StoryID: storyID, AssetLibraryID: libraryID, Version: version, Reference: reference})
	}
	return result
}

func hasPrimaryScreenshot(artifacts []basArtifact) bool {
	for _, artifact := range artifacts {
		if strings.EqualFold(artifact.Type, "CAPTURE_TYPE_SCREENSHOT") && artifact.Primary {
			return true
		}
	}
	return false
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

func readConsoleArtifact(path string) ConsoleEvidence {
	evidence := ConsoleEvidence{}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return evidence
	}
	var entries []struct {
		Level   string `json:"level"`
		Message string `json:"message"`
	}
	if json.Unmarshal(data, &entries) != nil {
		return evidence
	}
	for _, entry := range entries {
		if strings.EqualFold(entry.Level, "error") {
			evidence.ConsoleErrors = append(evidence.ConsoleErrors, entry.Message)
		}
	}
	return evidence
}
