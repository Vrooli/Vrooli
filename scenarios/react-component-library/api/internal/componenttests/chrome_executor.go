package componenttests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var renderedStoryResult = regexp.MustCompile(`(?s)<pre[^>]*id="rcl-story-result"[^>]*>(.*?)</pre>`)

// ChromeHarnessExecutor drives the exact harness route through Playwright and
// decodes the typed result emitted by the story page. The helper uses the same
// browser-backed route as the browser preview, so catalog validation exercises
// the page contract rather than a separate renderer.
type ChromeHarnessExecutor struct {
	BaseURL    string
	ChromePath string
	NodePath   string
	RunnerPath string
}

func NewChromeHarnessExecutor() ChromeHarnessExecutor {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("RCL_API_BASE_URL")), "/")
	if baseURL == "" {
		if port := strings.TrimSpace(os.Getenv("API_PORT")); port != "" {
			baseURL = "http://127.0.0.1:" + port
		}
	}
	path := strings.TrimSpace(os.Getenv("RCL_CHROME_BIN"))
	if path == "" {
		path = "google-chrome"
	}
	nodePath := strings.TrimSpace(os.Getenv("RCL_NODE_BIN"))
	if nodePath == "" {
		nodePath = "node"
	}
	return ChromeHarnessExecutor{
		BaseURL:    baseURL,
		ChromePath: path,
		NodePath:   nodePath,
		RunnerPath: componentHarnessRunnerPath(),
	}
}

func componentHarnessRunnerPath() string {
	if configured := strings.TrimSpace(os.Getenv("RCL_COMPONENT_HARNESS_RUNNER")); configured != "" {
		return configured
	}
	for _, candidate := range []string{
		filepath.Join("..", "ui", "scripts", "component-harness.mjs"),
		filepath.Join("scenarios", "react-component-library", "ui", "scripts", "component-harness.mjs"),
		filepath.Join("ui", "scripts", "component-harness.mjs"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join("..", "ui", "scripts", "component-harness.mjs")
}

func (e ChromeHarnessExecutor) ExecuteStory(ctx context.Context, libraryID, version, storyID string) (StoryExecution, error) {
	baseURL := strings.TrimRight(e.BaseURL, "/")
	if baseURL == "" {
		return StoryExecution{}, fmt.Errorf("preview API base URL is required")
	}
	if _, err := exec.LookPath(e.NodePath); err != nil {
		return StoryExecution{}, ExecutorUnavailableError{Err: fmt.Errorf("locate Node executable %q: %w", e.NodePath, err)}
	}
	if _, err := os.Stat(e.RunnerPath); err != nil {
		return StoryExecution{}, ExecutorUnavailableError{Err: fmt.Errorf("locate component harness runner %q: %w", e.RunnerPath, err)}
	}
	chromePath, err := exec.LookPath(e.ChromePath)
	if err != nil {
		return StoryExecution{}, ExecutorUnavailableError{Err: fmt.Errorf("locate Chrome executable %q: %w", e.ChromePath, err)}
	}
	u, err := url.Parse(baseURL + "/preview/" + url.PathEscape(libraryID) + "/harness.html")
	if err != nil {
		return StoryExecution{}, fmt.Errorf("build harness URL: %w", err)
	}
	q := u.Query()
	q.Set("version", version)
	q.Set("story", storyID)
	q.Set("runner", "1")
	if strings.Contains(strings.ToLower(storyID), "failure") {
		q.Set("fixtureShape", "failure")
	}
	u.RawQuery = q.Encode()
	started := time.Now()
	cmd := exec.CommandContext(ctx, e.NodePath, e.RunnerPath, u.String())
	cmd.Env = environWithChrome(os.Environ(), chromePath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return StoryExecution{}, fmt.Errorf("playwright harness: %w: %s", err, detail)
		}
		return StoryExecution{}, fmt.Errorf("playwright harness: %w", err)
	}
	execution, err := decodeStoryResultJSON(out)
	if err != nil {
		return StoryExecution{}, err
	}
	execution.Duration = time.Since(started)
	return execution, nil
}

func environWithChrome(environ []string, chromePath string) []string {
	filtered := make([]string, 0, len(environ)+1)
	for _, entry := range environ {
		if strings.HasPrefix(entry, "RCL_CHROME_BIN=") {
			continue
		}
		filtered = append(filtered, entry)
	}
	return append(filtered, "RCL_CHROME_BIN="+chromePath)
}

func decodeRenderedStoryResult(out []byte) (StoryExecution, error) {
	match := renderedStoryResult.FindSubmatch(out)
	if len(match) != 2 {
		return StoryExecution{}, fmt.Errorf("harness completed without a story result")
	}
	return decodeStoryResultJSON([]byte(html.UnescapeString(string(match[1]))))
}

func decodeStoryResultJSON(out []byte) (StoryExecution, error) {
	var result struct {
		Passed   bool `json:"passed"`
		Failures []struct {
			Message string `json:"message"`
		} `json:"failures"`
		Console     ConsoleEvidence     `json:"console"`
		Performance PerformanceEvidence `json:"performance"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &result); err != nil {
		return StoryExecution{}, fmt.Errorf("decode harness story result: %w", err)
	}
	failures := make([]string, 0, len(result.Failures))
	for _, failure := range result.Failures {
		if message := strings.TrimSpace(failure.Message); message != "" {
			failures = append(failures, message)
		}
	}
	return StoryExecution{Passed: result.Passed, Failures: failures, Console: result.Console, Performance: result.Performance}, nil
}
