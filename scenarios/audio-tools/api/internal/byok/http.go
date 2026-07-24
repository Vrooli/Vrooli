package byok

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"audio-tools/internal/clock"
	"audio-tools/internal/httpc"
)

// DoAudioRequest executes a unary BYOK audio request and returns its response
// bytes plus measured latency. Provider-specific request construction and
// result mapping stay with each adapter.
func DoAudioRequest(ctx context.Context, doer httpc.Doer, clk clock.Clock, provider string, request *http.Request) ([]byte, time.Duration, error) {
	if clk == nil {
		clk = clock.System{}
	}
	started := clk.Now()
	response, err := doer.Do(request.WithContext(ctx))
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", provider, err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		raw, _ := io.ReadAll(response.Body)
		return nil, 0, fmt.Errorf("%s: HTTP %d: %s", provider, response.StatusCode, truncate(string(raw), 256))
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, 0, err
	}
	return body, clk.Now().Sub(started), nil
}
