// Package streaming provides utilities for streaming data from the API.
package streaming

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Event represents a Server-Sent Event.
type Event struct {
	Type string          `json:"type"` // Event type (progress, error, complete, etc.)
	Data json.RawMessage `json:"data"` // Raw JSON data payload
}

// ProgressEvent represents a deployment progress update.
type ProgressEvent struct {
	Step       string  `json:"step"`              // Current step name
	Percent    float64 `json:"percent"`           // Overall progress percentage (0-100)
	Message    string  `json:"message,omitempty"` // Human-readable status message
	Phase      string  `json:"phase,omitempty"`   // Current deployment phase
	IsComplete bool    `json:"is_complete"`       // Whether deployment is complete
	Success    *bool   `json:"success,omitempty"` // Success status (only set when complete)
	Error      string  `json:"error,omitempty"`   // Error message (only set on failure)
}

// ProgressHandler is called for each progress event received.
type ProgressHandler func(event ProgressEvent) error

// StreamOptions configures the SSE stream.
type StreamOptions struct {
	BaseURL    string            // API base URL
	Path       string            // Endpoint path (e.g., /api/v1/deployments/{id}/progress)
	Headers    map[string]string // Additional headers (e.g., Authorization)
	Timeout    time.Duration     // Connection timeout (0 = no timeout)
	RetryCount int               // Number of retries on connection failure (0 = no retry)
	RetryDelay time.Duration     // Delay between retries
}

// StreamProgress opens an SSE connection and calls the handler for each progress event.
// Returns nil on successful completion, or an error if streaming fails.
func StreamProgress(ctx context.Context, opts StreamOptions, handler ProgressHandler) error {
	url := strings.TrimSuffix(opts.BaseURL, "/") + opts.Path

	// Set defaults
	if opts.Timeout == 0 {
		opts.Timeout = 10 * time.Minute // Long timeout for deployment operations
	}
	if opts.RetryCount == 0 {
		opts.RetryCount = 3
	}
	if opts.RetryDelay == 0 {
		opts.RetryDelay = 2 * time.Second
	}

	var lastErr error
	for attempt := 0; attempt <= opts.RetryCount; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(opts.RetryDelay):
			}
		}

		err := streamOnce(ctx, url, opts, handler)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Don't retry on completed deployment (even if it failed)
		if isCompletionError(err) {
			return err
		}
	}

	return fmt.Errorf("SSE streaming failed after %d attempts: %w", opts.RetryCount+1, lastErr)
}

func streamOnce(ctx context.Context, url string, opts StreamOptions, handler ProgressHandler) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// SSE requires Accept header
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// Add custom headers (e.g., Authorization)
	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{
		Timeout: opts.Timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("SSE connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("SSE endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		return fmt.Errorf("unexpected content type: %s (expected text/event-stream)", contentType)
	}

	return parseSSEStream(resp.Body, handler)
}

func parseSSEStream(reader io.Reader, handler ProgressHandler) error {
	scanner := bufio.NewScanner(reader)

	var eventType string
	var dataLines []string

	for scanner.Scan() {
		line := scanner.Text()

		// Empty line = end of event
		if line == "" {
			if len(dataLines) > 0 {
				data := strings.Join(dataLines, "\n")
				if err := processEvent(eventType, data, handler); err != nil {
					return err
				}
			}
			eventType = ""
			dataLines = nil
			continue
		}

		// Parse SSE field
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimPrefix(line, "data:"))
		} else if strings.HasPrefix(line, ":") {
			// Comment line, ignore
		} else if strings.HasPrefix(line, "id:") {
			// Event ID, ignore for now
		} else if strings.HasPrefix(line, "retry:") {
			// Retry interval, ignore for now
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("SSE stream error: %w", err)
	}

	return nil
}

func processEvent(eventType, data string, handler ProgressHandler) error {
	// Handle different event types
	switch eventType {
	case "progress", "":
		var progress ProgressEvent
		if err := json.Unmarshal([]byte(data), &progress); err != nil {
			// Try to continue even if one event is malformed
			return nil
		}
		if err := handler(progress); err != nil {
			return err
		}
		// Check if deployment is complete
		if progress.IsComplete {
			if progress.Success != nil && !*progress.Success {
				return &completionError{message: progress.Error, success: false}
			}
			return &completionError{success: true}
		}

	case "error":
		var errData struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal([]byte(data), &errData); err != nil {
			return fmt.Errorf("stream error: %s", data)
		}
		msg := errData.Error
		if msg == "" {
			msg = errData.Message
		}
		return fmt.Errorf("stream error: %s", msg)

	case "complete":
		var complete struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &complete); err != nil {
			return &completionError{success: true}
		}
		if !complete.Success {
			msg := complete.Error
			if msg == "" {
				msg = complete.Message
			}
			return &completionError{message: msg, success: false}
		}
		return &completionError{success: true}

	case "ping", "keepalive":
		// Ignore ping events
	}

	return nil
}

// completionError indicates the stream completed (successfully or not).
type completionError struct {
	message string
	success bool
}

func (e *completionError) Error() string {
	if e.success {
		return "deployment completed successfully"
	}
	if e.message != "" {
		return fmt.Sprintf("deployment failed: %s", e.message)
	}
	return "deployment failed"
}

func (e *completionError) Success() bool {
	return e.success
}

func isCompletionError(err error) bool {
	_, ok := err.(*completionError)
	return ok
}

// IsSuccess checks if the error indicates successful completion.
func IsSuccess(err error) bool {
	if ce, ok := err.(*completionError); ok {
		return ce.success
	}
	return false
}
