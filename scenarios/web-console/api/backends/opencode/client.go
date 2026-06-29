package opencode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client is the narrow surface the OpenCode watcher depends on. Tests substitute
// a fake; production uses HTTPClient against a loopback `opencode serve`.
type Client interface {
	ListSessions(ctx context.Context) ([]Session, error)
	SessionMessages(ctx context.Context, sessionID string) ([]MessageWithParts, error)
	// Events opens the SSE stream and calls onEvent for each decoded event until
	// ctx is cancelled or the stream terminates, returning the terminating
	// error (nil on a clean EOF).
	Events(ctx context.Context, onEvent func(Event)) error
}

// HTTPClient talks to an `opencode serve` instance over loopback HTTP.
type HTTPClient struct {
	BaseURL string
	HTTP    *http.Client
}

// NewHTTPClient builds a client with sensible per-call timeouts. The event
// stream uses its own long-lived request (no client timeout) so SSE is not
// killed mid-stream.
func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *HTTPClient) getJSON(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("opencode %s: status %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *HTTPClient) ListSessions(ctx context.Context) ([]Session, error) {
	var sessions []Session
	if err := c.getJSON(ctx, "/session", &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (c *HTTPClient) SessionMessages(ctx context.Context, sessionID string) ([]MessageWithParts, error) {
	var messages []MessageWithParts
	if err := c.getJSON(ctx, "/session/"+sessionID+"/message", &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// Events streams the SSE endpoint. SSE frames are `data: <json>` lines
// separated by blank lines; we decode each data line into an Event. Malformed
// frames are skipped so one bad line never tears down the stream.
func (c *HTTPClient) Events(ctx context.Context, onEvent func(Event)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/event", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	// No timeout on the streaming client: the stream is long-lived and bounded
	// only by ctx cancellation.
	streamClient := &http.Client{}
	resp, err := streamClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("opencode /event: status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		var ev Event
		if err := json.Unmarshal([]byte(payload), &ev); err != nil {
			continue
		}
		onEvent(ev)
	}
	return scanner.Err()
}
