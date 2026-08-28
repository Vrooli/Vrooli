package capabilities

import (
	"context"
	"net/http"
	"os/exec"

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

// TranscodeChecker verifies the in-process audio conversion dependency. The
// runner is injectable so registry tests remain hermetic and can model a host
// where ffmpeg is absent.
type TranscodeChecker struct {
	LookPath func(string) (string, error)
}

func (c *TranscodeChecker) Check(context.Context) (Status, string) {
	lookPath := c.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	if _, err := lookPath("ffmpeg"); err != nil {
		return StatusUnavailable, "ffmpeg is not installed (install ffmpeg to enable audio transcoding)"
	}
	return StatusAvailable, "ffmpeg is available for audio transcoding"
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
