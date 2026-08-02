package eventbus

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ReadRuntimeState reads the bounded receipt-runtime health headers exposed by
// AutomaticRuntime. Every api-core-served scenario receives this surface;
// callers do not need scenario-specific instrumentation to inspect it.
func ReadRuntimeState(ctx context.Context, baseURL string, client *http.Client) (RuntimeState, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/health", nil)
	if err != nil {
		return RuntimeState{}, err
	}
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return RuntimeState{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return RuntimeState{}, fmt.Errorf("receipt runtime health: %s", resp.Status)
	}
	return RuntimeStateFromHeaders(resp.Header)
}

// RuntimeStateFromHeaders decodes the public, bounded receipt-runtime health
// contract. A missing or malformed contract is an error rather than a guess.
func RuntimeStateFromHeaders(headers http.Header) (RuntimeState, error) {
	state := strings.TrimSpace(headers.Get(RuntimeStateHeader))
	if state == "" {
		return RuntimeState{}, fmt.Errorf("receipt runtime health headers are absent")
	}
	armed, err := strconv.ParseBool(headers.Get(RuntimeArmedHeader))
	if err != nil {
		return RuntimeState{}, fmt.Errorf("parse receipt runtime armed state: %w", err)
	}
	count, err := strconv.Atoi(headers.Get(RuntimePolicyCountHeader))
	if err != nil || count < 0 {
		return RuntimeState{}, fmt.Errorf("parse receipt runtime policy count")
	}
	var refreshed time.Time
	if raw := headers.Get(RuntimeLastRefreshHeader); raw != "" {
		refreshed, err = time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return RuntimeState{}, fmt.Errorf("parse receipt runtime refresh time: %w", err)
		}
	}
	return RuntimeState{State: state, Armed: armed, PolicyCount: count, LastRefresh: refreshed}, nil
}
