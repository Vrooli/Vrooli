package componenttests

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var renderedStoryResult = regexp.MustCompile(`(?s)<pre[^>]*id="rcl-story-result"[^>]*>(.*?)</pre>`)

// ChromeHarnessExecutor drives the exact harness route used by the preview
// iframe. Chrome's dumped DOM includes the result element the harness writes
// after React commits and story assertions settle.
type ChromeHarnessExecutor struct {
	BaseURL    string
	ChromePath string
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
	return ChromeHarnessExecutor{BaseURL: baseURL, ChromePath: path}
}

func (e ChromeHarnessExecutor) ExecuteStory(ctx context.Context, libraryID, version, storyID string) (StoryExecution, error) {
	baseURL := strings.TrimRight(e.BaseURL, "/")
	if baseURL == "" {
		return StoryExecution{}, fmt.Errorf("preview API base URL is required")
	}
	u, err := url.Parse(baseURL + "/preview/" + url.PathEscape(libraryID) + "/harness.html")
	if err != nil {
		return StoryExecution{}, fmt.Errorf("build harness URL: %w", err)
	}
	q := u.Query()
	q.Set("version", version)
	q.Set("story", storyID)
	q.Set("runner", "1")
	u.RawQuery = q.Encode()
	started := time.Now()
	cmd := exec.CommandContext(ctx, e.ChromePath, "--headless=new", "--no-sandbox", "--disable-gpu", "--disable-dev-shm-usage", "--run-all-compositor-stages-before-draw", "--virtual-time-budget=5000", "--dump-dom", u.String())
	out, err := cmd.Output()
	if err != nil {
		return StoryExecution{}, fmt.Errorf("headless harness: %w", err)
	}
	execution, err := decodeRenderedStoryResult(out)
	if err != nil {
		return StoryExecution{}, err
	}
	execution.Duration = time.Since(started)
	return execution, nil
}

func decodeRenderedStoryResult(out []byte) (StoryExecution, error) {
	match := renderedStoryResult.FindSubmatch(out)
	if len(match) != 2 {
		return StoryExecution{}, fmt.Errorf("harness completed without a story result")
	}
	var result struct {
		Passed   bool `json:"passed"`
		Failures []struct {
			Message string `json:"message"`
		} `json:"failures"`
	}
	if err := json.Unmarshal([]byte(html.UnescapeString(string(match[1]))), &result); err != nil {
		return StoryExecution{}, fmt.Errorf("decode harness story result: %w", err)
	}
	failures := make([]string, 0, len(result.Failures))
	for _, failure := range result.Failures {
		if message := strings.TrimSpace(failure.Message); message != "" {
			failures = append(failures, message)
		}
	}
	return StoryExecution{Passed: result.Passed, Failures: failures}, nil
}
