package capabilities

import (
	"context"
	"net/http"

	"audio-tools/internal/httpc"
)

// ResourceChecker is the generic HTTP-probe Checker used for resources whose
// liveness can be determined by a single GET against a known URL. It is the
// base predicate that more specialised checkers (Whisper, Kokoro, Ollama,
// OpenRouter, ScenarioChecker) extend.
type ResourceChecker struct {
	URL  string
	Doer httpc.Doer
}

func (c *ResourceChecker) Check(ctx context.Context) (Status, string) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.URL, nil)
	if err != nil {
		return StatusUnavailable, "failed to create request: " + err.Error()
	}

	resp, err := c.Doer.Do(req)
	if err != nil {
		return StatusUnavailable, "resource is not responding"
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusTemporaryRedirect {
		return StatusAvailable, "resource is healthy"
	}

	return StatusUnavailable, "resource returned unexpected status"
}
