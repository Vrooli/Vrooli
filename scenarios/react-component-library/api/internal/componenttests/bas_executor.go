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
	"strings"
	"time"
)

var basStoryResult = regexp.MustCompile(`(?s)<pre[^>]*id="rcl-story-result"[^>]*>(.*?)</pre>`)

// BASCaptureExecutor delegates story execution to BAS CaptureService. It is
// intentionally small so the runner remains independently testable while
// production receives BAS's screenshot, DOM, accessibility, console, network,
// and performance artifacts from one browser session.
type BASCaptureExecutor struct {
	RCLBaseURL string
	BASBaseURL string
	HTTPClient *http.Client
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
	if basBase == "" {
		basBase = "http://127.0.0.1:17116"
	}
	return BASCaptureExecutor{RCLBaseURL: rclBase, BASBaseURL: basBase, HTTPClient: &http.Client{Timeout: 2 * time.Minute}}
}

type basCaptureRequest struct {
	URL                 string        `json:"url"`
	Captures            []string      `json:"captures"`
	Dimensions          basDimensions `json:"dimensions"`
	WaitFor             basWaitFor    `json:"waitFor"`
	InlineDOM           bool          `json:"inlineDom"`
	InlineAccessibility bool          `json:"inlineAccessibility"`
	InlineComputedStyle bool          `json:"inlineComputedStyle"`
	Label               string        `json:"label"`
}

type basDimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
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
}

func (e BASCaptureExecutor) ExecuteStory(ctx context.Context, libraryID, version, storyID string) (StoryExecution, error) {
	if strings.TrimSpace(e.RCLBaseURL) == "" {
		return StoryExecution{}, ExecutorUnavailableError{Err: fmt.Errorf("RCL preview API base URL is required")}
	}
	if strings.TrimSpace(e.BASBaseURL) == "" {
		return StoryExecution{}, ExecutorUnavailableError{Err: fmt.Errorf("BAS CaptureService base URL is required")}
	}
	storyURL, err := e.storyURL(libraryID, version, storyID)
	if err != nil {
		return StoryExecution{}, err
	}
	payload := basCaptureRequest{
		URL: storyURL,
		Captures: []string{
			"CAPTURE_TYPE_SCREENSHOT",
			"CAPTURE_TYPE_DOM",
			"CAPTURE_TYPE_CONSOLE_LOGS",
			"CAPTURE_TYPE_NETWORK",
			"CAPTURE_TYPE_PERFORMANCE",
			"CAPTURE_TYPE_ACCESSIBILITY",
		},
		Dimensions:          basDimensions{Width: 1280, Height: 800},
		WaitFor:             basWaitFor{Selector: `[data-experience-surface="component-harness"][data-experience-state="ready"]`},
		InlineDOM:           true,
		InlineAccessibility: true,
		InlineComputedStyle: true,
		Label:               fmt.Sprintf("rcl:%s@%s:%s", libraryID, version, storyID),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return StoryExecution{}, fmt.Errorf("encode BAS capture request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.BASBaseURL+"/browser_automation_studio.v1.capture.CaptureService/Capture", strings.NewReader(string(body)))
	if err != nil {
		return StoryExecution{}, fmt.Errorf("create BAS capture request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client().Do(req)
	if err != nil {
		return StoryExecution{}, ExecutorUnavailableError{Err: err}
	}
	defer resp.Body.Close()
	responseBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return StoryExecution{}, fmt.Errorf("read BAS capture response: %w", readErr)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return StoryExecution{}, fmt.Errorf("BAS CaptureService returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	var capture basCaptureResponse
	if err := json.Unmarshal(responseBody, &capture); err != nil {
		return StoryExecution{}, fmt.Errorf("decode BAS capture response: %w", err)
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
		reference := artifact.Reference
		if reference == "" {
			reference = artifact.Path
		}
		if reference == "" {
			continue
		}
		if _, ok := seen[reference]; ok {
			continue
		}
		seen[reference] = struct{}{}
		kind := strings.ToLower(strings.TrimPrefix(artifact.Type, "CAPTURE_TYPE_"))
		if viewURL := strings.TrimSpace(artifact.Metadata["view_url"]); viewURL != "" {
			reference = browserVisibleBASArtifactPath(viewURL)
		}
		result = append(result, Artifact{Kind: "bas-" + kind, Label: fmt.Sprintf("%s:%s", storyID, kind), AssetLibraryID: libraryID, Version: version, Reference: reference})
	}
	return result
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
